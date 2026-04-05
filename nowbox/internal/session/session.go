package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nowbox/nowbox/internal/adapter"
	"github.com/nowbox/nowbox/internal/manifest"
	"github.com/nowbox/nowbox/internal/names"
)

// Session represents a live connection to a remote sandbox.
type Session struct {
	Name       string
	InstanceID string
	Host       *manifest.HostManifest
	Agent      *manifest.AgentManifest
	Adapter    adapter.Adapter
	Stream     adapter.Stream
	Vars       map[string]string
}

// recoveryFile is written during the session so orphans can be cleaned up.
type recoveryFile struct {
	SessionName string `json:"session_name"`
	InstanceID  string `json:"instance_id"`
	DestroyURL  string `json:"destroy_url"`
	Provider    string `json:"provider"`
}

var recoveryPath = filepath.Join(os.Getenv("HOME"), ".cache", "nowbox", "active-session.json")

// CheckOrphan checks if a previous session wasn't cleaned up.
// Returns the recovery info if found, nil otherwise.
func CheckOrphan() *recoveryFile {
	data, err := os.ReadFile(recoveryPath)
	if err != nil {
		return nil
	}
	var rf recoveryFile
	if json.Unmarshal(data, &rf) != nil {
		return nil
	}
	return &rf
}

// ClearRecovery deletes the recovery file.
func ClearRecovery() {
	os.Remove(recoveryPath)
}

// New creates a new session: provisions the sandbox and connects.
func New(host *manifest.HostManifest, agent *manifest.AgentManifest, vars map[string]string) (*Session, error) {
	sessionName := names.Generate()
	vars["SESSION_NAME"] = sessionName

	a, err := newAdapter(host)
	if err != nil {
		return nil, err
	}

	// Create sandbox
	fmt.Fprintf(os.Stderr, "  creating %s...\n", sessionName)
	instanceID, err := a.Create(sessionName, vars)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox: %w", err)
	}

	// Write recovery file immediately after create
	writeRecovery(sessionName, instanceID, host)

	// Connect
	stream, err := a.Connect(instanceID, vars)
	if err != nil {
		// Created but can't connect — destroy
		fmt.Fprintf(os.Stderr, "  connection failed, destroying %s...\n", sessionName)
		if destroyErr := a.Destroy(instanceID, vars); destroyErr == nil {
			ClearRecovery()
		} else {
			return nil, fmt.Errorf("connecting: %w (cleanup failed: %v)", err, destroyErr)
		}
		return nil, fmt.Errorf("connecting: %w", err)
	}

	return &Session{
		Name:       sessionName,
		InstanceID: instanceID,
		Host:       host,
		Agent:      agent,
		Adapter:    a,
		Stream:     stream,
		Vars:       vars,
	}, nil
}

// Destroy tears down the sandbox and cleans up.
func (s *Session) Destroy() error {
	if s.Stream != nil {
		s.Stream.Close()
	}

	err := s.Adapter.Destroy(s.InstanceID, s.Vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  nowbox: warning: could not destroy %s (%s)\n", s.Name, s.InstanceID)
		fmt.Fprintf(os.Stderr, "  nowbox: you may need to clean up manually at your provider's dashboard\n")
		return err
	}

	ClearRecovery()
	return nil
}

func DestroyOrphan(provider, instanceID string, vars map[string]string) error {
	host, err := manifest.LoadHost(provider)
	if err != nil {
		return err
	}

	a, err := newAdapter(host)
	if err != nil {
		return err
	}

	if err := a.Destroy(instanceID, vars); err != nil {
		return err
	}

	ClearRecovery()
	return nil
}

func writeRecovery(name, instanceID string, host *manifest.HostManifest) {
	rf := recoveryFile{
		SessionName: name,
		InstanceID:  instanceID,
		Provider:    host.Name,
		DestroyURL:  host.Destroy.URL,
	}

	data, err := json.Marshal(rf)
	if err != nil {
		return
	}

	dir := filepath.Dir(recoveryPath)
	os.MkdirAll(dir, 0700)
	os.WriteFile(recoveryPath, data, 0600)
}

func newAdapter(host *manifest.HostManifest) (adapter.Adapter, error) {
	// Check if the manifest has real endpoints
	if host.Create.URL == "" {
		return nil, fmt.Errorf("%s host is not yet supported (no API endpoints in manifest)", host.Name)
	}

	// Route to adapter based on connection type
	connectURL := host.Connect.URL
	if connectURL == "" {
		return nil, fmt.Errorf("%s host has no connect endpoint", host.Name)
	}

	// WebSocket connections
	if strings.HasPrefix(connectURL, "wss://") || strings.HasPrefix(connectURL, "ws://") {
		return &adapter.WebSocketExec{Host: host}, nil
	}

	// HTTP connections
	if strings.HasPrefix(connectURL, "https://") || strings.HasPrefix(connectURL, "http://") {
		return &adapter.HTTPExec{Host: host}, nil
	}

	return nil, fmt.Errorf("%s host has unsupported connect URL scheme: %s", host.Name, connectURL)
}
