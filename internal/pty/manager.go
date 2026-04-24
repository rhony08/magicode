package pty

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/opencode-ai/opencode-go/internal/bus"
)

// Manager manages PTY sessions with bounded limits and buffer pooling.
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex

	// Buffer pool for efficient memory reuse
	bufferPool sync.Pool

	// Max sessions limit
	maxSessions int

	// Bus for events
	bus *bus.Service

	// Event definitions
	eventCreated bus.Definition
	eventUpdated bus.Definition
	eventExited  bus.Definition
	eventDeleted bus.Definition

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Directory for default CWD
	directory string
}

// NewManager creates a new PTY manager.
func NewManager(directory string, eventBus *bus.Service) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		sessions:   make(map[string]*Session),
		maxSessions: MaxSessions,
		bus:        eventBus,
		ctx:        ctx,
		cancel:     cancel,
		directory:  directory,
		bufferPool: sync.Pool{
			New: func() interface{} {
				// Create 64KB buffer chunks
				return bytes.NewBuffer(make([]byte, 0, BufferChunk))
			},
		},
	}

	// Define PTY events
	if eventBus != nil {
		m.eventCreated = eventBus.Define(string(EventCreated), Event{})
		m.eventUpdated = eventBus.Define(string(EventUpdated), Event{})
		m.eventExited = eventBus.Define(string(EventExited), Event{})
		m.eventDeleted = eventBus.Define(string(EventDeleted), Event{})
	}

	return m
}

// List returns all active sessions.
func (m *Manager) List() []*SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		info := s.GetInfo()
		result = append(result, &info)
	}
	return result
}

// Get returns a specific session by ID.
func (m *Manager) Get(id string) (*SessionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	info := s.GetInfo()
	return &info, nil
}

// Create creates a new PTY session.
func (m *Manager) Create(input CreateInput) (*SessionInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check session limit
	if len(m.sessions) >= m.maxSessions {
		// Evict oldest session
		m.evictOldest()
	}

	// Generate session ID
	id := generateID()

	// Determine command and args
	command := input.Command
	if command == "" {
		command = GetShell()
	}

	args := input.Args
	if len(args) == 0 {
		args = []string{}
	}

	// Add login flag if needed
	if IsLoginShell(command) {
		args = append(args, "-l")
	}

	// Set working directory
	cwd := input.CWD
	if cwd == "" {
		cwd = m.directory
	}

	// Set environment
	env := make(map[string]string)
	for k, v := range input.Env {
		env[k] = v
	}

	// Create the process
	opts := &CreateInput{
		Command: command,
		Args:    args,
		CWD:     cwd,
		Env:     env,
		Cols:    input.Cols,
		Rows:    input.Rows,
	}

	proc, err := NewProcess(command, args, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create PTY process: %w", err)
	}

	// Create session
	session := &Session{
		Info: SessionInfo{
			ID:      id,
			Title:   input.Title,
			Command: command,
			Args:    args,
			CWD:     cwd,
			Status:  StatusRunning,
			PID:     proc.PID(),
		},
		Process:     proc,
		Buffer:      make([]byte, 0, BufferChunk),
		Subscribers: make(map[string]*Subscriber),
		CreatedAt:   time.Now(),
	}

	// Set default title if not provided
	if session.Info.Title == "" {
		session.Info.Title = fmt.Sprintf("Terminal %s", id[:4])
	}

	// Store session
	m.sessions[id] = session

	// Start output reader goroutine
	go m.readOutput(session)

	// Publish created event
	if m.bus != nil {
		m.bus.Publish(m.eventCreated, Event{
			Type: EventCreated,
			ID:   id,
			Info: &session.Info,
		})
	}

	return &session.Info, nil
}

// Update updates a session's properties.
func (m *Manager) Update(id string, input UpdateInput) (*SessionInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Update title
	if input.Title != "" {
		s.Info.Title = input.Title
	}

	// Resize terminal
	if input.Size != nil {
		err := s.Process.Resize(input.Size.Cols, input.Size.Rows)
		if err != nil {
			return nil, fmt.Errorf("failed to resize: %w", err)
		}
	}

	info := s.GetInfo()

	// Publish updated event
	if m.bus != nil {
		m.bus.Publish(m.eventUpdated, Event{
			Type: EventUpdated,
			ID:   id,
			Info: &info,
		})
	}

	return &info, nil
}

// Remove removes and terminates a session.
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	// Tear down session
	m.teardown(s)

	// Remove from map
	delete(m.sessions, id)

	// Publish deleted event
	if m.bus != nil {
		m.bus.Publish(m.eventDeleted, Event{
			Type: EventDeleted,
			ID:   id,
		})
	}

	return nil
}

// Resize resizes a session's terminal.
func (m *Manager) Resize(id string, cols, rows int) error {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	return s.Process.Resize(cols, rows)
}

// Write writes data to a session's PTY.
func (m *Manager) Write(id string, data []byte) error {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	_, err := s.Process.Write(data)
	return err
}

// Connect connects a WebSocket to a session.
func (m *Manager) Connect(id string, socket Socket, cursor int) (*Subscriber, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		socket.Close(1000, "Session not found")
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Generate subscriber ID
	subID := uuid.New().String()

	// Create subscriber
	sub := &Subscriber{
		ID:     subID,
		Send:   socket.Send,
		Close:  func() error { return socket.Close(1000, "Disconnected") },
		Ready:  false,
		Cursor: cursor,
	}

	// Add subscriber
	s.SubMu.Lock()
	s.Subscribers[subID] = sub
	s.SubMu.Unlock()

	// Send buffered data based on cursor
	m.sendBufferedData(s, sub, cursor)

	// Mark as ready
	sub.Ready = true

	return sub, nil
}

// Disconnect disconnects a subscriber from a session.
func (m *Manager) Disconnect(id string, subID string) error {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	s.SubMu.Lock()
	delete(s.Subscribers, subID)
	s.SubMu.Unlock()

	return nil
}

// Shutdown shuts down all sessions.
func (m *Manager) Shutdown() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.sessions {
		m.teardown(s)
	}
	m.sessions = make(map[string]*Session)
}

// readOutput reads output from a session and broadcasts to subscribers.
func (m *Manager) readOutput(session *Session) {
	// Get buffer from pool
	buf := m.getBuffer()
	defer m.putBuffer(buf)

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
		}

		// Read from PTY
		n, err := session.Process.Read(buf.Bytes())
		if err != nil {
			if err == io.EOF {
				// Process exited
				session.markExited(nil)
				return
			}
			// Other error
			session.markExited(&Exit{Code: 1})
			return
		}

		if n == 0 {
			continue
		}

		data := buf.Bytes()[:n]

		// Update cursor
		session.mu.Lock()
		session.Cursor += n
		session.mu.Unlock()

		// Broadcast to subscribers
		session.broadcast(data)

		// Append to buffer with limit check
		session.BufferMu.Lock()
		session.Buffer = append(session.Buffer, data...)
		// Trim if exceeds limit
		if len(session.Buffer) > BufferLimit {
			excess := len(session.Buffer) - BufferLimit
			session.Buffer = session.Buffer[excess:]
			session.BufferPos += excess
		}
		session.BufferMu.Unlock()
	}
}

// sendBufferedData sends buffered data to a new subscriber.
func (m *Manager) sendBufferedData(session *Session, sub *Subscriber, cursor int) {
	session.BufferMu.Lock()
	defer session.BufferMu.Unlock()

	session.mu.RLock()
	totalCursor := session.Cursor
	session.mu.RUnlock()

	// Calculate from position
	startPos := session.BufferPos
	endPos := totalCursor

	from := cursor
	if cursor == -1 {
		from = endPos
	} else if cursor < 0 {
		from = 0
	}

	// Calculate offset in buffer
	offset := from - startPos
	if offset < 0 {
		offset = 0
	}
	if offset >= len(session.Buffer) {
		// No data to send
		return
	}

	data := session.Buffer[offset:]

	// Send data in chunks
	for i := 0; i < len(data); i += BufferChunk {
		end := i + BufferChunk
		if end > len(data) {
			end = len(data)
		}

		chunk := data[i:end]
		if err := sub.Send(chunk); err != nil {
			// Failed to send, remove subscriber
			session.SubMu.Lock()
			delete(session.Subscribers, sub.ID)
			session.SubMu.Unlock()
			return
		}
	}

	// Send meta frame with cursor position
	meta := MetaFrame(endPos)
	if err := sub.Send(meta); err != nil {
		session.SubMu.Lock()
		delete(session.Subscribers, sub.ID)
		session.SubMu.Unlock()
	}
}

// broadcast sends data to all subscribers.
func (s *Session) broadcast(data []byte) {
	s.SubMu.RLock()
	defer s.SubMu.RUnlock()

	for id, sub := range s.Subscribers {
		if !sub.Ready {
			continue
		}

		err := sub.Send(data)
		if err != nil {
			// Remove failed subscriber
			s.SubMu.RUnlock()
			s.SubMu.Lock()
			delete(s.Subscribers, id)
			s.SubMu.Unlock()
			s.SubMu.RLock()
		}
	}
}

// markExited marks a session as exited.
func (s *Session) markExited(exit *Exit) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Exited {
		return
	}

	s.Exited = true
	s.Exit = exit
	s.Info.Status = StatusExited

	if exit != nil {
		s.Info.PID = 0 // Process no longer running
	}

	// Close all subscribers
	s.SubMu.Lock()
	for _, sub := range s.Subscribers {
		if sub.Close != nil {
			sub.Close()
		}
	}
	s.Subscribers = make(map[string]*Subscriber)
	s.SubMu.Unlock()
}

// GetInfo returns the session info.
func (s *Session) GetInfo() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Info
}

// teardown tears down a session.
func (m *Manager) teardown(session *Session) {
	// Kill process
	if session.Process != nil {
		session.Process.Kill("SIGTERM")
		session.Process.Close()
	}

	// Close all subscribers
	session.SubMu.Lock()
	for _, sub := range session.Subscribers {
		if sub.Close != nil {
			sub.Close()
		}
	}
	session.SubMu.Unlock()
}

// evictOldest evicts the oldest session to make room for a new one.
func (m *Manager) evictOldest() {
	var oldestID string
	var oldestTime time.Time

	for id, s := range m.sessions {
		if oldestID == "" || s.CreatedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = s.CreatedAt
		}
	}

	if oldestID != "" {
		m.teardown(m.sessions[oldestID])
		delete(m.sessions, oldestID)

		// Publish deleted event
		if m.bus != nil {
			m.bus.Publish(m.eventDeleted, Event{
				Type: EventDeleted,
				ID:   oldestID,
			})
		}
	}
}

// getBuffer gets a buffer from the pool.
func (m *Manager) getBuffer() *bytes.Buffer {
	buf := m.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putBuffer returns a buffer to the pool.
func (m *Manager) putBuffer(buf *bytes.Buffer) {
	m.bufferPool.Put(buf)
}

// generateID generates a unique session ID.
func generateID() string {
	id := uuid.New().String()
	// Format: pty_<timestamp>_<random>
	return fmt.Sprintf("pty_%d_%s", time.Now().UnixNano(), id[:8])
}

// IsRunning checks if a session is still running.
func (s *Session) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.Exited
}