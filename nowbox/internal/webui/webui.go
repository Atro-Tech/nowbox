package webui

import (
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/mdns"
	"github.com/nowbox/nowbox/internal/adapter"
)

//go:embed page.html
var pageHTML string

// sessionToken is a per-session random token required on the WebSocket path.
// Prevents cross-origin WebSocket hijacking from malicious pages.
var sessionToken string

func init() {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	sessionToken = fmt.Sprintf("%x", b)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Only allow our own localhost origin
		return strings.HasPrefix(origin, "http://127.0.0.1") || strings.HasPrefix(origin, "http://localhost")
	},
}

// Serve starts a local web server, opens the browser, and blocks until
// the session ends.
func Serve(stream adapter.Stream, sessionName string, hostAgent string) error {
	// Pick a random port
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("starting web server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }

	wsPath := "/ws/" + sessionToken

	// Serve the HTML page
	html := strings.ReplaceAll(pageHTML, "{{TITLE}}", sessionName+" — "+hostAgent)
	html = strings.ReplaceAll(html, "{{SESSION}}", sessionName)
	html = strings.ReplaceAll(html, "{{HOST_AGENT}}", hostAgent)
	html = strings.ReplaceAll(html, "{{WS_PATH}}", wsPath)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// WebSocket proxy — browser ↔ remote stream
	mux.HandleFunc(wsPath, func(w http.ResponseWriter, r *http.Request) {
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

	// Register mDNS so the session is reachable at <session>.local
	mdnsServer, mdnsErr := registerMDNS(sessionName, port)
	if mdnsServer != nil {
		defer mdnsServer.Shutdown()
	}

	var url string
	if mdnsErr == nil {
		url = fmt.Sprintf("http://%s.local:%d", sessionName, port)
	} else {
		url = fmt.Sprintf("http://localhost:%d", port)
	}
	fmt.Fprintf(os.Stderr, "  web: %s\n", url)

	go http.Serve(listener, mux)
	defer listener.Close()

	openBrowser(url)

	<-done
	return nil
}

func registerMDNS(name string, port int) (*mdns.Server, error) {
	host := name + "."
	info := []string{"nowbox"}
	service, err := mdns.NewMDNSService(name, "_http._tcp", "", host, port, nil, info)
	if err != nil {
		return nil, err
	}
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, err
	}
	return server, nil
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
