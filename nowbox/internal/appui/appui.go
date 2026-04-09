//go:build cgo

package appui

import (
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/nowbox/nowbox/internal/adapter"
	"github.com/nowbox/nowbox/internal/token"
	webview "github.com/webview/webview_go"
)

//go:embed page.html
var pageHTML string

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
		return strings.HasPrefix(origin, "http://127.0.0.1") || strings.HasPrefix(origin, "http://localhost")
	},
}

// SessionInfo holds the data needed to save a .now file from the app UI.
type SessionInfo struct {
	HostName      string
	AgentName     string
	Vars          map[string]string
	InstanceID    string
	SetupCommands []string
}

// Serve opens a native window with the chat + TTY UI and blocks until it closes.
func Serve(stream adapter.Stream, sessionName string, hostAgent string, info *SessionInfo) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting app server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }

	wsPath := "/ws/" + sessionToken

	// Template the HTML
	html := strings.ReplaceAll(pageHTML, "{{TITLE}}", sessionName+" — "+hostAgent)
	html = strings.ReplaceAll(html, "{{SESSION}}", sessionName)
	html = strings.ReplaceAll(html, "{{HOST_AGENT}}", hostAgent)
	html = strings.ReplaceAll(html, "{{WS_PATH}}", wsPath)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Save handler
	savePath := "/save/" + sessionToken
	mux.HandleFunc(savePath, func(w http.ResponseWriter, r *http.Request) {
		if info == nil {
			http.Error(w, "no session info available", http.StatusInternalServerError)
			return
		}

		vars := make(map[string]string)
		for k, v := range info.Vars {
			if k == "SESSION_NAME" || k == "INSTANCE_ID" {
				continue
			}
			vars[k] = v
		}

		sealed, err := token.Seal(&token.Payload{
			Host:       info.HostName,
			Agent:      info.AgentName,
			Vars:       vars,
			InstanceID: info.InstanceID,
		})
		if err != nil {
			http.Error(w, "failed to encrypt session", http.StatusInternalServerError)
			return
		}

		script := fmt.Sprintf("#!/bin/sh\n# nowbox — %s + %s\nset -e\nexport NOWBOX_TOKEN=\"%s\"\ncurl -fsSL nowbox.lol | sh -s -- %s %s\n",
			info.HostName, info.AgentName, sealed, info.HostName, info.AgentName)

		filename := sessionName + ".now"
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.Write([]byte(script))
	})

	html = strings.ReplaceAll(html, "{{SAVE_PATH}}", savePath)

	// WebSocket proxy — window ↔ remote stream
	mux.HandleFunc(wsPath, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Remote → window
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

		// Window → remote
		setupSent := false
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				finish()
				return
			}
			if mt == websocket.TextMessage {
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
					if !setupSent && info != nil && len(info.SetupCommands) > 0 {
						setupSent = true
						for _, cmd := range info.SetupCommands {
							stream.Write([]byte(cmd + "\n"))
						}
					}
					continue
				}
			}
			if _, err := stream.Write(data); err != nil {
				finish()
				return
			}
		}
	})

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Fprintf(os.Stderr, "  app: %s\n", url)

	go http.Serve(listener, mux)
	defer listener.Close()

	// Open native window
	w := webview.New(false)
	defer w.Destroy()
	setAppIcon()

	w.SetTitle(sessionName + " — " + hostAgent)
	w.SetSize(960, 640, webview.HintNone)
	w.Navigate(url)

	// When WebSocket closes (sandbox dies), close the window
	go func() {
		<-done
		w.Terminate()
	}()

	w.Run() // blocks until window closed
	finish()

	return nil
}
