//go:build !cgo

package appui

import (
	"fmt"

	"github.com/nowbox/nowbox/internal/adapter"
)

// SessionInfo holds the data needed to save a .now file from the app UI.
type SessionInfo struct {
	HostName      string
	AgentName     string
	Vars          map[string]string
	InstanceID    string
	SetupCommands []string
}

// Serve is a stub for builds without CGo. The native app window requires CGo.
func Serve(stream adapter.Stream, sessionName string, hostAgent string, info *SessionInfo) error {
	return fmt.Errorf("app mode requires CGo (native webview); use 'browser' or 'cli' mode instead")
}
