package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/nowbox/nowbox/internal/manifest"
)

// WebSocketExec implements the Adapter interface using HTTP REST for
// create/destroy and WebSocket TTY for connect. Parameterized entirely
// by the host manifest.
type WebSocketExec struct {
	Host *manifest.HostManifest
}

func (a *WebSocketExec) Create(sessionName string, vars map[string]string) (string, error) {
	cfg := a.Host.Create

	reqURL := manifest.Expand(cfg.URL, vars)
	body := manifest.Expand(cfg.Body, vars)
	headers := manifest.ExpandMap(cfg.Headers, vars)

	req, err := http.NewRequest(cfg.Method, reqURL, bytes.NewBufferString(body))
	if err != nil {
		return "", fmt.Errorf("building create request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("create failed (%d): %s", resp.StatusCode, string(respBody))
	}

	// Extract instance ID using parse_id (simple .field extraction)
	instanceID, err := extractField(respBody, cfg.ParseID)
	if err != nil {
		return "", fmt.Errorf("extracting instance ID: %w", err)
	}

	return instanceID, nil
}

func (a *WebSocketExec) Connect(instanceID string, vars map[string]string) (Stream, error) {
	cfg := a.Host.Connect

	vars["INSTANCE_ID"] = instanceID

	wsURL := manifest.Expand(cfg.URL, vars)
	headers := manifest.ExpandMap(cfg.Headers, vars)

	// Build URL with query params
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, fmt.Errorf("parsing connect URL: %w", err)
	}
	q := u.Query()
	for k, v := range manifest.ExpandMap(cfg.Query, vars) {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// WebSocket dial
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
		return nil, fmt.Errorf("websocket connect failed: %w", err)
	}

	return newWSStream(conn), nil
}

func (a *WebSocketExec) Destroy(instanceID string, vars map[string]string) error {
	cfg := a.Host.Destroy

	vars["INSTANCE_ID"] = instanceID

	reqURL := manifest.Expand(cfg.URL, vars)
	headers := manifest.ExpandMap(cfg.Headers, vars)

	req, err := http.NewRequest(cfg.Method, reqURL, nil)
	if err != nil {
		return fmt.Errorf("building destroy request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("destroy request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("destroy failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// extractField does simple ".fieldname" extraction from JSON.
func extractField(data []byte, path string) (string, error) {
	if len(path) < 2 || path[0] != '.' {
		return "", fmt.Errorf("invalid parse path: %s", path)
	}
	field := path[1:]

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parsing JSON: %w", err)
	}

	val, ok := result[field]
	if !ok {
		return "", fmt.Errorf("field %q not found in response", field)
	}

	return fmt.Sprintf("%v", val), nil
}

// wsStream wraps a gorilla/websocket conn to implement the Stream interface.
// TTY mode: binary frames are raw terminal data both directions.
// Text frames are JSON control messages (handled separately).
type wsStream struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	reader io.Reader
}

func newWSStream(conn *websocket.Conn) *wsStream {
	return &wsStream{conn: conn}
}

func (s *wsStream) Read(p []byte) (int, error) {
	for {
		if s.reader != nil {
			n, err := s.reader.Read(p)
			if n > 0 {
				return n, nil
			}
			s.reader = nil
			if err != io.EOF {
				return 0, err
			}
		}

		msgType, data, err := s.conn.ReadMessage()
		if err != nil {
			return 0, err
		}

		switch msgType {
		case websocket.BinaryMessage:
			// TTY mode: binary = raw terminal data
			s.reader = bytes.NewReader(data)
		case websocket.TextMessage:
			// JSON control message — check for exit
			var msg map[string]interface{}
			if json.Unmarshal(data, &msg) == nil {
				if msg["type"] == "exit" {
					return 0, io.EOF
				}
			}
			// Other text messages (session_info, debug) — skip
		}
	}
}

func (s *wsStream) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *wsStream) Close() error {
	return s.conn.Close()
}

func (s *wsStream) Resize(cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := fmt.Sprintf(`{"type":"resize","cols":%d,"rows":%d}`, cols, rows)
	return s.conn.WriteMessage(websocket.TextMessage, []byte(msg))
}
