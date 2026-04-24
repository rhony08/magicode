// Package tui provides tests for TUI package.
package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// TestNewApp tests app creation
func TestNewApp(t *testing.T) {
	app := NewApp(Config{
		Title: "Test",
	})

	if app == nil {
		t.Error("NewApp should return non-nil app")
	}
	if app.view != ViewChat {
		t.Errorf("Initial view should be ViewChat, got %s", app.view)
	}
	if app.mode != ModeInput {
		t.Errorf("Initial mode should be ModeInput, got %s", app.mode)
	}
	if app.status != "Ready" {
		t.Errorf("Initial status should be 'Ready', got '%s'", app.status)
	}
}

// TestNewAppWithSession tests app creation with session
func TestNewAppWithSession(t *testing.T) {
	session := Session{
		ID:    "test-session",
		Title: "Test Session",
	}

	app := NewApp(Config{
		Session: session,
	})

	if app.activeSession == nil {
		t.Error("App should have active session")
	}
	if app.activeSession.ID != "test-session" {
		t.Errorf("Session ID should be 'test-session', got '%s'", app.activeSession.ID)
	}
	if len(app.sessions) != 1 {
		t.Errorf("Should have 1 session, got %d", len(app.sessions))
	}
}

// TestViewState tests view state constants
func TestViewState(t *testing.T) {
	if ViewChat != "chat" {
		t.Errorf("ViewChat should be 'chat', got '%s'", ViewChat)
	}
	if ViewSession != "session" {
		t.Errorf("ViewSession should be 'session', got '%s'", ViewSession)
	}
	if ViewHelp != "help" {
		t.Errorf("ViewHelp should be 'help', got '%s'", ViewHelp)
	}
}

// TestInputMode tests input mode constants
func TestInputMode(t *testing.T) {
	if ModeNormal != "normal" {
		t.Errorf("ModeNormal should be 'normal', got '%s'", ModeNormal)
	}
	if ModeInput != "input" {
		t.Errorf("ModeInput should be 'input', got '%s'", ModeInput)
	}
	if ModeWait != "wait" {
		t.Errorf("ModeWait should be 'wait', got '%s'", ModeWait)
	}
}

// TestRole tests role constants
func TestRole(t *testing.T) {
	if RoleUser != "user" {
		t.Errorf("RoleUser should be 'user', got '%s'", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant should be 'assistant', got '%s'", RoleAssistant)
	}
	if RoleSystem != "system" {
		t.Errorf("RoleSystem should be 'system', got '%s'", RoleSystem)
	}
	if RoleTool != "tool" {
		t.Errorf("RoleTool should be 'tool', got '%s'", RoleTool)
	}
}

// TestMessage tests message struct
func TestMessage(t *testing.T) {
	msg := Message{
		ID:        "msg-1",
		Role:      RoleUser,
		Content:   "Hello",
		Timestamp: time.Now(),
	}

	if msg.ID != "msg-1" {
		t.Errorf("Message.ID should be 'msg-1'")
	}
	if msg.Role != RoleUser {
		t.Errorf("Message.Role should be RoleUser")
	}
	if msg.Content != "Hello" {
		t.Errorf("Message.Content should be 'Hello'")
	}
}

// TestSession tests session struct
func TestSession(t *testing.T) {
	session := Session{
		ID:        "session-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		Active:    true,
	}

	if session.ID != "session-1" {
		t.Errorf("Session.ID should be 'session-1'")
	}
	if session.Title != "Test Session" {
		t.Errorf("Session.Title should be 'Test Session'")
	}
	if !session.Active {
		t.Error("Session.Active should be true")
	}
}

// TestToolCall tests tool call struct
func TestToolCall(t *testing.T) {
	tc := ToolCall{
		Tool:   "bash",
		Input:  "ls -la",
		Result: "file list",
		Status: "success",
	}

	if tc.Tool != "bash" {
		t.Errorf("ToolCall.Tool should be 'bash'")
	}
	if tc.Status != "success" {
		t.Errorf("ToolCall.Status should be 'success'")
	}
}

// TestDefaultKeybindings tests keybindings
func TestDefaultKeybindings(t *testing.T) {
	kb := DefaultKeybindings()

	if kb.Submit.Key != tea.KeyEnter {
		t.Errorf("Submit key should be Enter")
	}
	if kb.Quit.Key != tea.KeyCtrlC {
		t.Errorf("Quit key should be Ctrl+C")
	}
	if kb.Up.Key != tea.KeyUp {
		t.Errorf("Up key should be Up arrow")
	}
	if kb.Down.Key != tea.KeyDown {
		t.Errorf("Down key should be Down arrow")
	}
}

// TestKeyNames tests key names
func TestKeyNames(t *testing.T) {
	kb := DefaultKeybindings()
	names := KeyNames(kb)

	if names["submit"] != "Enter" {
		t.Errorf("submit key name should be 'Enter'")
	}
	if names["quit"] != "Ctrl+C / q" {
		t.Errorf("quit key name should be 'Ctrl+C / q'")
	}
	if names["up"] != "↑" {
		t.Errorf("up key name should be '↑'")
	}
}

// TestFormatKeybinding tests keybinding formatting
func TestFormatKeybinding(t *testing.T) {
	tests := []struct {
		kb       Keybinding
		expected string
	}{
		{Keybinding{Key: tea.KeyEnter}, "Enter"},
		{Keybinding{Key: tea.KeyUp}, "↑"},
		{Keybinding{Key: tea.KeyDown}, "↓"},
		{Keybinding{Key: tea.KeyTab}, "Tab"},
		{Keybinding{Key: tea.KeyEsc}, "Esc"},
	}

	for _, tt := range tests {
		result := FormatKeybinding(tt.kb)
		if result != tt.expected {
			t.Errorf("FormatKeybinding: expected '%s', got '%s'", tt.expected, result)
		}
	}
}

// TestAppSetTitle tests setting title
func TestAppSetTitle(t *testing.T) {
	app := NewApp(Config{})
	app.SetTitle("New Title")

	// Title is associated with active session
	// If no session, this is a no-op
}

// TestAppSetStatus tests setting status
func TestAppSetStatus(t *testing.T) {
	app := NewApp(Config{})
	app.SetStatus("Processing")

	if app.status != "Processing" {
		t.Errorf("Status should be 'Processing', got '%s'", app.status)
	}
}

// TestAppAddMessage tests adding messages
func TestAppAddMessage(t *testing.T) {
	app := NewApp(Config{})
	app.addMessage(Message{
		Role:    RoleUser,
		Content: "Hello",
	})

	if len(app.messages) != 1 {
		t.Errorf("Should have 1 message, got %d", len(app.messages))
	}
}

// TestAppSetMessages tests setting messages
func TestAppSetMessages(t *testing.T) {
	app := NewApp(Config{})

	messages := []Message{
		{ID: "1", Role: RoleUser, Content: "Hello"},
		{ID: "2", Role: RoleAssistant, Content: "Hi there"},
	}

	app.SetMessages(messages)

	if len(app.messages) != 2 {
		t.Errorf("Should have 2 messages, got %d", len(app.messages))
	}
}

// TestAppSetSessions tests setting sessions
func TestAppSetSessions(t *testing.T) {
	app := NewApp(Config{})

	sessions := []Session{
		{ID: "s1", Title: "Session 1"},
		{ID: "s2", Title: "Session 2"},
	}

	app.SetSessions(sessions)

	if len(app.sessions) != 2 {
		t.Errorf("Should have 2 sessions, got %d", len(app.sessions))
	}
}

// TestAppMethods tests accessor methods
func TestAppMethods(t *testing.T) {
	app := NewApp(Config{})
	app.width = 100
	app.height = 50
	app.processing = true

	if app.Width() != 100 {
		t.Errorf("Width should be 100, got %d", app.Width())
	}
	if app.Height() != 50 {
		t.Errorf("Height should be 50, got %d", app.Height())
	}
	if !app.Processing() {
		t.Error("Processing should be true")
	}
}

// TestStreamMsg tests stream message
func TestStreamMsg(t *testing.T) {
	msg := StreamMsg{
		Content: "Hello",
		Done:    false,
	}

	if msg.Content != "Hello" {
		t.Errorf("StreamMsg.Content should be 'Hello'")
	}
	if msg.Done {
		t.Error("StreamMsg.Done should be false")
	}
}

// TestResponseMsg tests response message
func TestResponseMsg(t *testing.T) {
	msg := ResponseMsg{
		Content: "Response text",
		Error:   nil,
	}

	if msg.Content != "Response text" {
		t.Errorf("ResponseMsg.Content should be 'Response text'")
	}
	if msg.Error != nil {
		t.Error("ResponseMsg.Error should be nil")
	}
}

// TestToolCallMsg tests tool call message
func TestToolCallMsg(t *testing.T) {
	msg := ToolCallMsg{
		Tool:   "bash",
		Input:  "ls",
		Result: "output",
		Status: "success",
	}

	if msg.Tool != "bash" {
		t.Errorf("ToolCallMsg.Tool should be 'bash'")
	}
}

// TestSessionMsg tests session message
func TestSessionMsg(t *testing.T) {
	msg := SessionMsg{
		ID:     "session-1",
		Title:  "Test",
		Action: "create",
	}

	if msg.ID != "session-1" {
		t.Errorf("SessionMsg.ID should be 'session-1'")
	}
	if msg.Action != "create" {
		t.Errorf("SessionMsg.Action should be 'create'")
	}
}

// TestErrorMsg tests error message
func TestErrorMsg(t *testing.T) {
	err := fmt.Errorf("test error")
	msg := ErrorMsg{Error: err}

	if msg.Error == nil {
		t.Error("ErrorMsg.Error should not be nil")
	}
}

// TestResizeMsg tests resize message
func TestResizeMsg(t *testing.T) {
	msg := ResizeMsg{
		Width:  100,
		Height: 50,
	}

	if msg.Width != 100 {
		t.Errorf("ResizeMsg.Width should be 100")
	}
	if msg.Height != 50 {
		t.Errorf("ResizeMsg.Height should be 50")
	}
}

// TestStyles tests style definitions
func TestStyles(t *testing.T) {
	// Test that styles are defined
	if styleNormal.GetForeground() == lipgloss.Color("") {
		// styleNormal has no foreground, that's fine
	}
	if styleTitle.GetForeground() != colorPrimary {
		t.Error("styleTitle should have primary color")
	}
	if styleBold.GetBold() != true {
		t.Error("styleBold should be bold")
	}
}