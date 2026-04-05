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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nowbox/nowbox/internal/adapter"
	"github.com/nowbox/nowbox/internal/manifest"
)

// Serve starts an MCP server that exposes a remote coding agent.
// The primary interface is the `message` tool — you're talking to an agent.
// File and exec tools provide access to the agent's environment.
func Serve(
	host *manifest.HostManifest,
	agentStream adapter.Stream,
	instanceID string,
	sessionName string,
	agentName string,
	vars map[string]string,
) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting mcp server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	srv := &mcpHandler{
		host:        host,
		agentStream: agentStream,
		instanceID:  instanceID,
		agentName:   agentName,
		vars:        vars,
	}

	// Start reading agent output in background
	go srv.readAgentOutput()

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", srv.handleMCP)

	mcpURL := fmt.Sprintf("http://localhost:%d/mcp", port)
	mcpName := "nowbox-" + sessionName

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  nowbox mcp: %s (%s)\n", mcpURL, agentName)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  claude code:   claude mcp add %s --url %s\n", mcpName, mcpURL)
	fmt.Fprintf(os.Stderr, "  cursor:        add %s to mcp settings\n", mcpURL)
	fmt.Fprintf(os.Stderr, "  codex:         codex mcp add %s --url %s\n", mcpName, mcpURL)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  ctrl-c to stop\n")

	go http.Serve(listener, mux)

	select {}
}

type mcpHandler struct {
	host        *manifest.HostManifest
	agentStream adapter.Stream
	instanceID  string
	agentName   string
	vars        map[string]string
	mu          sync.Mutex

	// Buffer of agent output — message tool reads from here
	outputMu  sync.Mutex
	outputBuf bytes.Buffer
}

// readAgentOutput continuously reads from the agent PTY stream
func (s *mcpHandler) readAgentOutput() {
	buf := make([]byte, 32*1024)
	for {
		n, err := s.agentStream.Read(buf)
		if n > 0 {
			s.outputMu.Lock()
			s.outputBuf.Write(buf[:n])
			s.outputMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// drainOutput returns and clears whatever the agent has output
func (s *mcpHandler) drainOutput() string {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	out := s.outputBuf.String()
	s.outputBuf.Reset()
	return stripANSI(out)
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
			"serverInfo": map[string]interface{}{
				"name":    fmt.Sprintf("nowbox/%s", s.agentName),
				"version": "0.1.0",
			},
		}

	case "tools/list":
		resp.Result = map[string]interface{}{
			"tools": s.toolList(),
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

func (s *mcpHandler) toolList() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "message",
			"description": fmt.Sprintf("Send a message to the %s agent running in the sandbox. This is the primary way to interact — give it tasks, ask questions, or have it write code. The agent responds asynchronously through the terminal.", s.agentName),
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]string{
						"type":        "string",
						"description": "Message to send to the agent",
					},
					"wait_seconds": map[string]interface{}{
						"type":        "integer",
						"description": "Seconds to wait for the agent's response (default: 10)",
					},
				},
				"required": []string{"text"},
			},
		},
		{
			"name":        "read_output",
			"description": fmt.Sprintf("Read the latest output from the %s agent. Use this to check what the agent has done since your last message.", s.agentName),
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "exec",
			"description": "Run a shell command in the agent's sandbox environment. Use this for tasks the agent doesn't need to handle — checking files, installing packages, running tests, etc.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]string{"type": "string", "description": "Shell command to run"},
				},
				"required": []string{"command"},
			},
		},
		{
			"name":        "read_file",
			"description": "Read a file from the agent's sandbox. Useful for reviewing code the agent has written or checking config files.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string", "description": "File path"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "write_file",
			"description": "Write a file to the agent's sandbox. Use this to provide code, config, or data for the agent to work with.",
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
			"description": "List files in a directory in the agent's sandbox.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string", "description": "Directory path (default: current dir)"},
				},
			},
		},
	}
}

func (s *mcpHandler) handleToolCall(params json.RawMessage) interface{} {
	var call struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return toolError("invalid params: " + err.Error())
	}

	switch call.Name {
	case "message":
		return s.handleMessage(call.Arguments)
	case "read_output":
		return s.handleReadOutput()
	case "exec":
		return s.handleExec(call.Arguments)
	case "read_file":
		return s.handleExec(map[string]interface{}{"command": fmt.Sprintf("cat '%v'", call.Arguments["path"])})
	case "write_file":
		content := fmt.Sprintf("%v", call.Arguments["content"])
		path := fmt.Sprintf("%v", call.Arguments["path"])
		return s.handleExec(map[string]interface{}{"command": fmt.Sprintf("cat > '%s' << 'NOWBOX_EOF'\n%s\nNOWBOX_EOF", path, content)})
	case "list_files":
		path := "."
		if p, ok := call.Arguments["path"]; ok && p != nil {
			path = fmt.Sprintf("%v", p)
		}
		return s.handleExec(map[string]interface{}{"command": fmt.Sprintf("ls -la '%s'", path)})
	default:
		return toolError("unknown tool: " + call.Name)
	}
}

func (s *mcpHandler) handleMessage(args map[string]interface{}) interface{} {
	text := fmt.Sprintf("%v", args["text"])
	waitSec := 10
	if w, ok := args["wait_seconds"]; ok {
		if wf, ok := w.(float64); ok {
			waitSec = int(wf)
		}
	}

	// Clear any pending output
	s.drainOutput()

	// Send the message to the agent via PTY
	s.agentStream.Write([]byte(text + "\n"))

	// Wait for the agent to respond
	time.Sleep(time.Duration(waitSec) * time.Second)

	// Collect the response
	output := s.drainOutput()
	if output == "" {
		output = "(agent is still working — use read_output to check later)"
	}

	return map[string]interface{}{
		"content": []map[string]string{{"type": "text", "text": output}},
	}
}

func (s *mcpHandler) handleReadOutput() interface{} {
	output := s.drainOutput()
	if output == "" {
		output = "(no new output from agent)"
	}
	return map[string]interface{}{
		"content": []map[string]string{{"type": "text", "text": output}},
	}
}

func (s *mcpHandler) handleExec(args map[string]interface{}) interface{} {
	cmd := fmt.Sprintf("%v", args["command"])
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

// execInSandbox opens a non-TTY WebSocket to run a command directly in the sandbox OS.
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
		streamID := data[0]
		payload := data[1:]
		switch streamID {
		case 1, 2:
			stdout.Write(payload)
		case 3:
			return stdout.String(), nil
		}
	}
	return stdout.String(), nil
}

// stripANSI removes ANSI escape sequences from terminal output
func stripANSI(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\033' {
			// Skip ESC sequence
			i++
			if i < len(s) && s[i] == '[' {
				i++
				for i < len(s) && !((s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
					i++
				}
				if i < len(s) {
					i++ // skip the final letter
				}
			} else if i < len(s) && s[i] == ']' {
				// OSC sequence — skip until BEL or ST
				i++
				for i < len(s) && s[i] != '\007' && s[i] != '\033' {
					i++
				}
				if i < len(s) && s[i] == '\007' {
					i++
				}
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
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
