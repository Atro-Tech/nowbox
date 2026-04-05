package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/nowbox/nowbox/internal/manifest"
)

// Serve starts an MCP-compatible HTTP server that proxies tools to the sandbox.
func Serve(
	host *manifest.HostManifest,
	instanceID string,
	sessionName string,
	vars map[string]string,
) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting mcp server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	srv := &mcpHandler{
		host:       host,
		instanceID: instanceID,
		vars:       vars,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", srv.handleMCP)

	fmt.Fprintf(os.Stderr, "\n  mcp server running on port %d\n", port)
	fmt.Fprintf(os.Stderr, "\n  add to claude code mcp settings:\n\n")
	fmt.Fprintf(os.Stderr, "  {\n")
	fmt.Fprintf(os.Stderr, "    \"mcpServers\": {\n")
	fmt.Fprintf(os.Stderr, "      \"nowbox-%s\": {\n", sessionName)
	fmt.Fprintf(os.Stderr, "        \"url\": \"http://localhost:%d/mcp\"\n", port)
	fmt.Fprintf(os.Stderr, "      }\n")
	fmt.Fprintf(os.Stderr, "    }\n")
	fmt.Fprintf(os.Stderr, "  }\n\n")
	fmt.Fprintf(os.Stderr, "  ctrl-c to stop\n\n")

	go http.Serve(listener, mux)

	select {}
}

type mcpHandler struct {
	host       *manifest.HostManifest
	instanceID string
	vars       map[string]string
	mu         sync.Mutex
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *mcpHandler) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "event: endpoint\ndata: /mcp\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		<-r.Context().Done()
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	var req jsonRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}

	var resp jsonRPCResponse
	resp.JSONRPC = "2.0"
	resp.ID = req.ID

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "nowbox", "version": "0.1.0"},
		}

	case "tools/list":
		resp.Result = map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "exec",
					"description": "Execute a shell command in the sandbox and return output",
					"inputSchema": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{"command": map[string]string{"type": "string", "description": "Shell command to run"}},
						"required":   []string{"command"},
					},
				},
				{
					"name":        "read_file",
					"description": "Read the contents of a file in the sandbox",
					"inputSchema": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{"path": map[string]string{"type": "string", "description": "Absolute or relative file path"}},
						"required":   []string{"path"},
					},
				},
				{
					"name":        "write_file",
					"description": "Write content to a file in the sandbox (creates or overwrites)",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path":    map[string]string{"type": "string", "description": "File path"},
							"content": map[string]string{"type": "string", "description": "File content"},
						},
						"required": []string{"path", "content"},
					},
				},
				{
					"name":        "list_files",
					"description": "List files in a directory in the sandbox",
					"inputSchema": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{"path": map[string]string{"type": "string", "description": "Directory path (default: current dir)"}},
					},
				},
			},
		}

	case "tools/call":
		resp.Result = s.handleToolCall(req.Params)

	case "notifications/initialized":
		w.WriteHeader(200)
		return

	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *mcpHandler) handleToolCall(params json.RawMessage) interface{} {
	var call struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return toolError("invalid params: " + err.Error())
	}

	var cmd string
	switch call.Name {
	case "exec":
		cmd = fmt.Sprintf("%v", call.Arguments["command"])
	case "read_file":
		cmd = fmt.Sprintf("cat '%v'", call.Arguments["path"])
	case "write_file":
		content := fmt.Sprintf("%v", call.Arguments["content"])
		path := fmt.Sprintf("%v", call.Arguments["path"])
		cmd = fmt.Sprintf("cat > '%s' << 'NOWBOX_EOF'\n%s\nNOWBOX_EOF", path, content)
	case "list_files":
		path := "."
		if p, ok := call.Arguments["path"]; ok && p != nil {
			path = fmt.Sprintf("%v", p)
		}
		cmd = fmt.Sprintf("ls -la '%s'", path)
	default:
		return toolError("unknown tool: " + call.Name)
	}

	output, err := s.execInSandbox(cmd)
	if err != nil {
		return toolError(err.Error())
	}

	return map[string]interface{}{
		"content": []map[string]string{{"type": "text", "text": output}},
	}
}

func toolError(msg string) interface{} {
	return map[string]interface{}{
		"content": []map[string]string{{"type": "text", "text": msg}},
		"isError": true,
	}
}

// execInSandbox opens a non-TTY WebSocket to the sandbox, runs the command,
// and returns stdout+stderr.
func (s *mcpHandler) execInSandbox(cmd string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vars := make(map[string]string)
	for k, v := range s.vars {
		vars[k] = v
	}
	vars["INSTANCE_ID"] = s.instanceID

	wsURL := manifest.Expand(s.host.Connect.URL, vars)
	headers := manifest.ExpandMap(s.host.Connect.Headers, vars)

	u, err := url.Parse(wsURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("tty", "false")
	q.Set("stdin", "true")
	q.Set("cmd", "sh")
	q.Add("cmd", "-c")
	q.Add("cmd", cmd)
	q.Set("path", "sh")
	u.RawQuery = q.Encode()

	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
		return "", fmt.Errorf("exec connect: %w", err)
	}
	defer conn.Close()

	var stdout bytes.Buffer
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if len(data) == 0 {
			continue
		}

		// Non-TTY: first byte is stream ID
		streamID := data[0]
		payload := data[1:]
		switch streamID {
		case 1: // stdout
			stdout.Write(payload)
		case 2: // stderr
			stdout.Write(payload)
		case 3: // exit
			return stdout.String(), nil
		}
	}
	return stdout.String(), nil
}
