package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/nowbox/nowbox/internal/manifest"
)

// HTTPExec implements the Adapter interface using HTTP REST for
// create/connect/destroy. Used by providers like Vercel, Runloop,
// Daytona, Blaxel — any provider with REST exec endpoints.
// Parameterized entirely by the host manifest.
type HTTPExec struct {
	Host *manifest.HostManifest
}

func (a *HTTPExec) Create(sessionName string, vars map[string]string) (string, error) {
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

	instanceID, err := extractField(respBody, cfg.ParseID)
	if err != nil {
		return "", fmt.Errorf("extracting instance ID: %w", err)
	}

	return instanceID, nil
}

func (a *HTTPExec) Connect(instanceID string, vars map[string]string) (Stream, error) {
	cfg := a.Host.Connect
	vars["INSTANCE_ID"] = instanceID

	execURL := manifest.Expand(cfg.URL, vars)
	headers := manifest.ExpandMap(cfg.Headers, vars)

	return &httpStream{
		execURL: execURL,
		headers: headers,
		query:   cfg.Query,
		readBuf: bytes.NewBuffer(nil),
	}, nil
}

func (a *HTTPExec) Destroy(instanceID string, vars map[string]string) error {
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

// httpStream wraps HTTP exec endpoints as a Stream.
// Write sends commands as POST requests, Read returns the response.
type httpStream struct {
	execURL string
	headers map[string]string
	query   map[string]string
	readBuf *bytes.Buffer
	mu      sync.Mutex
}

func (s *httpStream) Read(p []byte) (int, error) {
	if s.readBuf.Len() > 0 {
		return s.readBuf.Read(p)
	}
	// Block until there's something to read — for HTTP exec this means
	// we wait for the next Write to trigger a command and fill the buffer
	return 0, io.EOF
}

func (s *httpStream) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := string(p)
	body, _ := json.Marshal(map[string]interface{}{
		"command": cmd,
	})

	req, err := http.NewRequest("POST", s.execURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	s.readBuf.Write(respBody)

	return len(p), nil
}

func (s *httpStream) Close() error {
	return nil
}

func (s *httpStream) Resize(cols, rows int) error {
	// HTTP exec doesn't support resize
	return nil
}
