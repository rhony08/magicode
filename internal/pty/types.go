// Package pty provides pseudo-terminal management for MagiCode.
package pty

import (
	"context"
	"io"
	"sync"
	"time"
)

// Status represents the current state of a PTY session.
type Status string

const (
	StatusRunning Status = "running"
	StatusExited  Status = "exited"
)

// Exit represents the exit information of a PTY process.
type Exit struct {
	Code   int
	Signal string // Optional signal that caused exit
}

// SessionInfo contains information about a PTY session.
type SessionInfo struct {
	ID      string
	Title   string
	Command string
	Args    []string
	CWD     string
	Status  Status
	PID     int
}

// CreateInput contains options for creating a new PTY session.
type CreateInput struct {
	Command string            // Shell or command to run (defaults to user's shell)
	Args    []string          // Arguments for the command
	CWD     string            // Working directory (defaults to instance directory)
	Title   string            // Session title
	Env     map[string]string // Additional environment variables
	Cols    int               // Initial terminal width (defaults to 80)
	Rows    int               // Initial terminal height (defaults to 24)
}

// UpdateInput contains options for updating a PTY session.
type UpdateInput struct {
	Title string // New session title
	Size  *Size  // Terminal resize
}

// Size represents terminal dimensions.
type Size struct {
	Cols int
	Rows int
}

// EventType defines PTY event types.
type EventType string

const (
	EventCreated EventType = "pty.created"
	EventUpdated EventType = "pty.updated"
	EventExited  EventType = "pty.exited"
	EventDeleted EventType = "pty.deleted"
)

// Event represents a PTY event.
type Event struct {
	Type     EventType
	ID       string
	Info     *SessionInfo // For Created, Updated events
	ExitCode int          // For Exited event
}

// Process represents a running PTY process.
type Process interface {
	// PID returns the process ID.
	PID() int

	// Write writes data to the PTY.
	Write(data []byte) (int, error)

	// Read reads data from the PTY.
	Read(data []byte) (int, error)

	// Resize changes the terminal dimensions.
	Resize(cols, rows int) error

	// Kill terminates the process.
	Kill(signal string) error

	// Wait waits for the process to exit and returns the exit info.
	Wait(ctx context.Context) (*Exit, error)

	// Close closes the PTY file handles.
	Close() error
}

// Session represents an active PTY session with subscribers.
type Session struct {
	Info       SessionInfo
	Process    Process
	Buffer     []byte     // Output buffer (max 2MB)
	BufferMu   sync.Mutex // Protects buffer access
	BufferPos  int        // Position in buffer for cursor tracking
	Cursor     int        // Total bytes written (for cursor sync)
	CreatedAt  time.Time

	// Subscriber management
	Subscribers map[string]*Subscriber
	SubMu       sync.RWMutex

	// State management
	Exited bool
	Exit   *Exit
	mu     sync.RWMutex
}

// Subscriber represents a WebSocket connection subscribed to a session.
type Subscriber struct {
	ID       string
	Send     func(data []byte) error // Callback to send data to WebSocket
	Close    func() error            // Callback to close connection
	Ready    bool                    // Connection ready state
	Cursor   int                     // Subscriber's cursor position
}

// Constants for memory management.
const (
	// BufferLimit is the maximum size of the output buffer (2MB).
	BufferLimit = 2 * 1024 * 1024

	// BufferChunk is the chunk size for sending buffered data (64KB).
	BufferChunk = 64 * 1024

	// MaxSessions is the maximum number of concurrent PTY sessions.
	MaxSessions = 5

	// DefaultCols and DefaultRows are default terminal dimensions.
	DefaultCols = 80
	DefaultRows = 24
)

// Socket is an interface for WebSocket-like connections.
type Socket interface {
	ReadyState() int
	Send(data []byte) error
	Close(code int, reason string) error
}

// WebSocket ready state constants.
const (
	WebSocketConnecting = 0
	WebSocketOpen       = 1
	WebSocketClosing    = 2
	WebSocketClosed     = 3
)

// MetaFrame creates a metadata frame with cursor position.
// The first byte is 0x00 to indicate it's a control frame,
// followed by JSON-encoded cursor info.
func MetaFrame(cursor int) []byte {
	// Simple JSON format: {"cursor":123}
	data := []byte("{\"cursor\":")
	data = append(data, []byte(itoa(cursor))...)
	data = append(data, '}')
	
	// Prepend with 0x00 marker
	result := make([]byte, len(data)+1)
	result[0] = 0x00
	copy(result[1:], data)
	return result
}

// itoa converts int to string without importing strconv (for small numbers).
func itoa(n int) []byte {
	if n == 0 {
		return []byte{'0'}
	}
	
	var digits []byte
	negative := n < 0
	if negative {
		n = -n
	}
	
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	
	return digits
}

// Writer is an interface for writing output.
type Writer interface {
	io.Writer
}

// Reader is an interface for reading output.
type Reader interface {
	io.Reader
}