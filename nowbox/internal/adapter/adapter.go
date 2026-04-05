package adapter

import "io"

// Stream is a bidirectional connection to a remote sandbox.
type Stream interface {
	io.ReadWriteCloser
	Resize(cols, rows int) error
}

// Adapter handles the protocol-specific work for a host provider.
type Adapter interface {
	// Create provisions a new sandbox. Returns the provider's instance ID.
	Create(sessionName string, vars map[string]string) (instanceID string, err error)

	// Connect opens a live stream to an existing sandbox.
	Connect(instanceID string, vars map[string]string) (Stream, error)

	// Destroy tears down a sandbox by instance ID.
	Destroy(instanceID string, vars map[string]string) error
}
