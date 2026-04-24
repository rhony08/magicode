package pty

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/opencode-ai/opencode-go/internal/bus"
)

// MockProcess is a mock PTY process for testing
type MockProcess struct {
	pid     int
	writes  []byte
	readBuf *bytes.Buffer
 exited  bool
	exit    *Exit
	mu      sync.Mutex
}

func NewMockProcess() *MockProcess {
	return &MockProcess{
		pid:     12345,
		readBuf: bytes.NewBuffer([]byte("Hello from terminal\n")),
	}
}

func (m *MockProcess) PID() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pid
}

func (m *MockProcess) Write(data []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.exited {
		return 0, fmt.Errorf("process exited")
	}
	m.writes = append(m.writes, data...)
	return len(data), nil
}

func (m *MockProcess) Read(data []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.exited {
		return 0, io.EOF
	}
	return m.readBuf.Read(data)
}

func (m *MockProcess) Resize(cols, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.exited {
		return fmt.Errorf("process exited")
	}
	return nil
}

func (m *MockProcess) Kill(signal string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exited = true
	m.exit = &Exit{Code: 137, Signal: signal}
	return nil
}

func (m *MockProcess) Wait(ctx context.Context) (*Exit, error) {
	// Simulate exit after 100ms
	select {
	case <-time.After(100 * time.Millisecond):
		m.mu.Lock()
		m.exited = true
		m.exit = &Exit{Code: 0}
		m.mu.Unlock()
		return m.exit, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *MockProcess) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exited = true
	m.readBuf = bytes.NewBuffer(nil)
	return nil
}

func (m *MockProcess) SetReadData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuf = bytes.NewBuffer(data)
}

func (m *MockProcess) GetWrites() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writes
}

var _ Process = (*MockProcess)(nil)

// MockSocket is a mock WebSocket for testing
type MockSocket struct {
	readyState int
	sendBuf    []byte
	sendError  error
	closeError error
}

func NewMockSocket() *MockSocket {
	return &MockSocket{
		readyState: WebSocketOpen,
	}
}

func (s *MockSocket) ReadyState() int {
	return s.readyState
}

func (s *MockSocket) Send(data []byte) error {
	if s.sendError != nil {
		return s.sendError
	}
	s.sendBuf = append(s.sendBuf, data...)
	return nil
}

func (s *MockSocket) Close(code int, reason string) error {
	s.readyState = WebSocketClosed
	return s.closeError
}

func (s *MockSocket) GetSentData() []byte {
	return s.sendBuf
}

var _ Socket = (*MockSocket)(nil)

// TestTypes tests PTY types
func TestTypes(t *testing.T) {
	// Test Status
	if StatusRunning != "running" {
		t.Error("StatusRunning should be 'running'")
	}
	if StatusExited != "exited" {
		t.Error("StatusExited should be 'exited'")
	}

	// Test EventType
	if EventCreated != "pty.created" {
		t.Error("EventCreated should be 'pty.created'")
	}
	if EventUpdated != "pty.updated" {
		t.Error("EventUpdated should be 'pty.updated'")
	}
	if EventExited != "pty.exited" {
		t.Error("EventExited should be 'pty.exited'")
	}
	if EventDeleted != "pty.deleted" {
		t.Error("EventDeleted should be 'pty.deleted'")
	}

	// Test Constants
	if BufferLimit != 2*1024*1024 {
		t.Errorf("BufferLimit should be 2MB, got %d", BufferLimit)
	}
	if BufferChunk != 64*1024 {
		t.Errorf("BufferChunk should be 64KB, got %d", BufferChunk)
	}
	if MaxSessions != 5 {
		t.Errorf("MaxSessions should be 5, got %d", MaxSessions)
	}
	if DefaultCols != 80 {
		t.Errorf("DefaultCols should be 80, got %d", DefaultCols)
	}
	if DefaultRows != 24 {
		t.Errorf("DefaultRows should be 24, got %d", DefaultRows)
	}
}

// TestMetaFrame tests metadata frame creation
func TestMetaFrame(t *testing.T) {
	// Test with various cursor positions
	tests := []int{0, 100, 1000, 10000}

	for _, cursor := range tests {
		frame := MetaFrame(cursor)
		if len(frame) < 2 {
			t.Errorf("MetaFrame should have at least 2 bytes for cursor %d", cursor)
		}
		if frame[0] != 0x00 {
			t.Error("MetaFrame should start with 0x00 marker")
		}
		// Should contain "cursor" in JSON
		if !bytes.Contains(frame[1:], []byte("cursor")) {
			t.Error("MetaFrame JSON should contain 'cursor' key")
		}
	}
}

// TestSessionInfo tests session info
func TestSessionInfo(t *testing.T) {
	info := SessionInfo{
		ID:      "test-123",
		Title:   "Test Terminal",
		Command: "/bin/bash",
		Args:    []string{"-l"},
		CWD:     "/home/user",
		Status:  StatusRunning,
		PID:     12345,
	}

	if info.ID != "test-123" {
		t.Error("ID should be 'test-123'")
	}
	if info.Title != "Test Terminal" {
		t.Error("Title should be 'Test Terminal'")
	}
	if info.Command != "/bin/bash" {
		t.Error("Command should be '/bin/bash'")
	}
	if len(info.Args) != 1 || info.Args[0] != "-l" {
		t.Error("Args should be ['-l']")
	}
	if info.CWD != "/home/user" {
		t.Error("CWD should be '/home/user'")
	}
	if info.Status != StatusRunning {
		t.Error("Status should be 'running'")
	}
	if info.PID != 12345 {
		t.Error("PID should be 12345")
	}
}

// TestCreateInput tests create input validation
func TestCreateInput(t *testing.T) {
	input := CreateInput{
		Command: "/bin/bash",
		Args:    []string{"-l"},
		CWD:     "/home/user",
		Title:   "My Terminal",
		Env:     map[string]string{"TERM": "xterm-256color"},
		Cols:    120,
		Rows:    40,
	}

	if input.Command != "/bin/bash" {
		t.Error("Command should be '/bin/bash'")
	}
	if len(input.Args) != 1 {
		t.Error("Args should have 1 element")
	}
	if input.CWD != "/home/user" {
		t.Error("CWD should be '/home/user'")
	}
	if input.Title != "My Terminal" {
		t.Error("Title should be 'My Terminal'")
	}
	if len(input.Env) != 1 {
		t.Error("Env should have 1 entry")
	}
	if input.Cols != 120 {
		t.Error("Cols should be 120")
	}
	if input.Rows != 40 {
		t.Error("Rows should be 40")
	}
}

// TestUpdateInput tests update input
func TestUpdateInput(t *testing.T) {
	// Test with title
	input1 := UpdateInput{
		Title: "New Title",
	}
	if input1.Title != "New Title" {
		t.Error("Title should be 'New Title'")
	}

	// Test with size
	input2 := UpdateInput{
		Size: &Size{Cols: 100, Rows: 30},
	}
	if input2.Size == nil {
		t.Error("Size should not be nil")
	}
	if input2.Size.Cols != 100 {
		t.Error("Size.Cols should be 100")
	}
	if input2.Size.Rows != 30 {
		t.Error("Size.Rows should be 30")
	}
}

// TestEvent tests PTY events
func TestEvent(t *testing.T) {
	info := &SessionInfo{
		ID:     "test-123",
		Title:  "Test",
		Status: StatusRunning,
	}

	// Test Created event
	created := Event{
		Type: EventCreated,
		ID:   "test-123",
		Info: info,
	}
	if created.Type != EventCreated {
		t.Error("Type should be EventCreated")
	}
	if created.ID != "test-123" {
		t.Error("ID should be 'test-123'")
	}
	if created.Info == nil {
		t.Error("Info should not be nil")
	}

	// Test Exited event
	exited := Event{
		Type:     EventExited,
		ID:       "test-123",
		ExitCode: 0,
	}
	if exited.Type != EventExited {
		t.Error("Type should be EventExited")
	}
	if exited.ExitCode != 0 {
		t.Error("ExitCode should be 0")
	}

	// Test Deleted event
	deleted := Event{
		Type: EventDeleted,
		ID:   "test-123",
	}
	if deleted.Type != EventDeleted {
		t.Error("Type should be EventDeleted")
	}
}

// TestMockProcess tests mock process
func TestMockProcess(t *testing.T) {
	proc := NewMockProcess()

	// Test PID
	if proc.PID() != 12345 {
		t.Error("PID should be 12345")
	}

	// Test Write
	n, err := proc.Write([]byte("test input\n"))
	if err != nil {
		t.Errorf("Write should not error: %v", err)
	}
	if n != 11 {
		t.Errorf("Write should return 11 bytes, got %d", n)
	}

	// Test Read
	buf := make([]byte, 100)
	n, err = proc.Read(buf)
	if err != nil {
		t.Errorf("Read should not error: %v", err)
	}
	if n != 20 {
		t.Errorf("Read should return 20 bytes, got %d", n)
	}

	// Test Resize
	err = proc.Resize(100, 50)
	if err != nil {
		t.Errorf("Resize should not error: %v", err)
	}

	// Test Kill
	err = proc.Kill("SIGTERM")
	if err != nil {
		t.Errorf("Kill should not error: %v", err)
	}

	// Test Close
	err = proc.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

// TestMockSocket tests mock socket
func TestMockSocket(t *testing.T) {
	socket := NewMockSocket()

	// Test ReadyState
	if socket.ReadyState() != WebSocketOpen {
		t.Error("ReadyState should be WebSocketOpen")
	}

	// Test Send
	err := socket.Send([]byte("test data"))
	if err != nil {
		t.Errorf("Send should not error: %v", err)
	}
	if len(socket.GetSentData()) != 9 {
		t.Errorf("Sent data should be 9 bytes, got %d", len(socket.GetSentData()))
	}

	// Test Close
	err = socket.Close(1000, "Normal closure")
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
	if socket.ReadyState() != WebSocketClosed {
		t.Error("ReadyState should be WebSocketClosed after close")
	}
}

// TestManagerBasics tests manager creation and basic operations
func TestManagerBasics(t *testing.T) {
	eventBus := bus.NewDefault()
	mgr := NewManager("/tmp/test", eventBus)

	if mgr.directory != "/tmp/test" {
		t.Error("Directory should be '/tmp/test'")
	}
	if mgr.maxSessions != MaxSessions {
		t.Errorf("MaxSessions should be %d", MaxSessions)
	}
	if len(mgr.sessions) != 0 {
		t.Error("Should have no sessions initially")
	}

	// Test List (empty)
	list := mgr.List()
	if len(list) != 0 {
		t.Error("List should be empty")
	}

	// Shutdown
	mgr.Shutdown()
}

// TestSession tests session operations
func TestSession(t *testing.T) {
	proc := NewMockProcess()
	session := &Session{
		Info: SessionInfo{
			ID:      "test-session",
			Title:   "Test Session",
			Command: "/bin/bash",
			Status:  StatusRunning,
			PID:     12345,
		},
		Process:     proc,
		Buffer:      make([]byte, 0, BufferChunk),
		Subscribers: make(map[string]*Subscriber),
		CreatedAt:   time.Now(),
	}

	// Test GetInfo
	info := session.GetInfo()
	if info.ID != "test-session" {
		t.Error("ID should be 'test-session'")
	}

	// Test IsRunning
	if !session.IsRunning() {
		t.Error("Session should be running")
	}

	// Test markExited
	session.markExited(&Exit{Code: 0})
	if session.Exited {
		t.Log("Session marked as exited correctly")
	}
	if session.Info.Status != StatusExited {
		t.Error("Status should be 'exited'")
	}
}

// TestSubscriber tests subscriber operations
func TestSubscriber(t *testing.T) {
	socket := NewMockSocket()
	sub := &Subscriber{
		ID:     "sub-123",
		Send:   socket.Send,
		Close:  func() error { return socket.Close(1000, "done") },
		Ready:  false,
		Cursor: 0,
	}

	if sub.ID != "sub-123" {
		t.Error("ID should be 'sub-123'")
	}

	// Test Send
	err := sub.Send([]byte("test"))
	if err != nil {
		t.Errorf("Send should not error: %v", err)
	}

	// Test Close
	err = sub.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

// TestBufferPooling tests buffer pool operations
func TestBufferPooling(t *testing.T) {
	eventBus := bus.NewDefault()
	mgr := NewManager("/tmp/test", eventBus)

	// Get buffer
	buf := mgr.getBuffer()
	if buf == nil {
		t.Error("Buffer should not be nil")
	}

	// Use buffer
	buf.Write([]byte("test data"))
	if buf.Len() != 9 {
		t.Errorf("Buffer should have 9 bytes, got %d", buf.Len())
	}

	// Return buffer
	mgr.putBuffer(buf)

	// Get again (should be reset)
	buf2 := mgr.getBuffer()
	if buf2.Len() != 0 {
		t.Errorf("Reused buffer should be empty, got %d bytes", buf2.Len())
	}

	mgr.putBuffer(buf2)
	mgr.Shutdown()
}

// TestSize tests size struct
func TestSize(t *testing.T) {
	size := Size{Cols: 100, Rows: 50}

	if size.Cols != 100 {
		t.Error("Cols should be 100")
	}
	if size.Rows != 50 {
		t.Error("Rows should be 50")
	}
}

// TestExit tests exit struct
func TestExit(t *testing.T) {
	// Normal exit
	exit1 := Exit{Code: 0}
	if exit1.Code != 0 {
		t.Error("Exit code should be 0")
	}

	// Error exit
	exit2 := Exit{Code: 1}
	if exit2.Code != 1 {
		t.Error("Exit code should be 1")
	}

	// Signal exit
	exit3 := Exit{Code: 137, Signal: "SIGKILL"}
	if exit3.Signal != "SIGKILL" {
		t.Error("Signal should be 'SIGKILL'")
	}
}

// TestWebSocketConstants tests WebSocket state constants
func TestWebSocketConstants(t *testing.T) {
	if WebSocketConnecting != 0 {
		t.Error("WebSocketConnecting should be 0")
	}
	if WebSocketOpen != 1 {
		t.Error("WebSocketOpen should be 1")
	}
	if WebSocketClosing != 2 {
		t.Error("WebSocketClosing should be 2")
	}
	if WebSocketClosed != 3 {
		t.Error("WebSocketClosed should be 3")
	}
}

// TestBroadcast tests session broadcast functionality
func TestBroadcast(t *testing.T) {
	proc := NewMockProcess()
	socket1 := NewMockSocket()
	socket2 := NewMockSocket()

	session := &Session{
		Info: SessionInfo{
			ID:     "test",
			Status: StatusRunning,
		},
		Process:     proc,
		Buffer:      make([]byte, 0),
		Subscribers: make(map[string]*Subscriber),
		CreatedAt:   time.Now(),
	}

	// Add subscribers
	session.SubMu.Lock()
	session.Subscribers["sub1"] = &Subscriber{
		ID:    "sub1",
		Send:  socket1.Send,
		Ready: true,
	}
	session.Subscribers["sub2"] = &Subscriber{
		ID:    "sub2",
		Send:  socket2.Send,
		Ready: true,
	}
	session.SubMu.Unlock()

	// Broadcast data
	session.broadcast([]byte("test broadcast\n"))

	// Check both sockets received data
	data1 := socket1.GetSentData()
	data2 := socket2.GetSentData()

	if len(data1) != 15 {
		t.Errorf("Socket1 should have 15 bytes, got %d", len(data1))
	}
	if len(data2) != 15 {
		t.Errorf("Socket2 should have 15 bytes, got %d", len(data2))
	}

	// Verify content
	if string(data1) != "test broadcast\n" {
		t.Errorf("Socket1 data incorrect: %s", data1)
	}
	if string(data2) != "test broadcast\n" {
		t.Errorf("Socket2 data incorrect: %s", data2)
	}
}

// TestGetShell tests shell detection
func TestGetShell(t *testing.T) {
	shell := GetShell()
	if shell == "" {
		t.Error("Shell should not be empty")
	}
	t.Logf("Detected shell: %s", shell)
}

// TestIsLoginShell tests login shell detection
func TestIsLoginShell(t *testing.T) {
	// Common shells
	if !IsLoginShell("/bin/bash") {
		t.Error("/bin/bash should be a login shell")
	}
	if !IsLoginShell("bash") {
		t.Error("bash should be a login shell")
	}
	if !IsLoginShell("/bin/zsh") {
		t.Error("/bin/zsh should be a login shell")
	}
	if !IsLoginShell("/bin/sh") {
		t.Error("/bin/sh should be a login shell")
	}

	// Non-login shells
	if IsLoginShell("/bin/fish") {
		t.Error("/bin/fish should not be a login shell (uses different login)")
	}
}

// TestGenerateID tests ID generation
func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	// IDs should be different
	if id1 == id2 {
		t.Error("IDs should be unique")
	}

	// ID should have correct prefix
	if !bytes.HasPrefix([]byte(id1), []byte("pty_")) {
		t.Error("ID should start with 'pty_'")
	}

	// ID should be reasonable length
	if len(id1) < 10 {
		t.Error("ID should be at least 10 characters")
	}
}

// TestItoa tests integer to string conversion
func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{1000, "1000"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		result := string(itoa(tt.input))
		if result != tt.expected {
			t.Errorf("itoa(%d) = '%s', expected '%s'", tt.input, result, tt.expected)
		}
	}
}