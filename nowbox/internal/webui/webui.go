package webui

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/nowbox/nowbox/internal/adapter"
)

//go:embed page.html
var pageHTML string

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Serve starts a local web server, opens the browser, and blocks until
// the session ends.
func Serve(stream adapter.Stream, sessionName string, hostAgent string) error {
	// Pick a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting web server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }

	// Serve the HTML page
	html := strings.ReplaceAll(pageHTML, "{{TITLE}}", sessionName+" — "+hostAgent)
	html = strings.ReplaceAll(html, "{{SESSION}}", sessionName)
	html = strings.ReplaceAll(html, "{{HOST_AGENT}}", hostAgent)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// WebSocket proxy — browser ↔ remote stream
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Remote → browser
		go func() {
			buf := make([]byte, 32*1024)
			for {
				n, err := stream.Read(buf)
				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						finish()
						return
					}
				}
				if err != nil {
					finish()
					return
				}
			}
		}()

		// Browser → remote
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				finish()
				return
			}
			if mt == websocket.TextMessage {
				// Check for resize
				var msg struct {
					Type string `json:"type"`
					Cols int    `json:"cols"`
					Rows int    `json:"rows"`
				}
				if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" {
					if err := stream.Resize(msg.Cols, msg.Rows); err != nil {
						finish()
						return
					}
					continue
				}
			}
			// Binary or non-resize text — send as input
			if _, err := stream.Write(data); err != nil {
				finish()
				return
			}
		}
	})

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Fprintf(os.Stderr, "  web: %s\n", url)

	go http.Serve(listener, mux)
	defer listener.Close()

	// Open browser
	openBrowser(url)

	// Block until session ends
	<-done
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}
