// Package tui provides the terminal user interface for MagiCode.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	// Colors
	colorPrimary    = lipgloss.Color("#7C3AED")
	colorSecondary  = lipgloss.Color("#2563EB")
	colorSuccess    = lipgloss.Color("#10B981")
	colorError      = lipgloss.Color("#EF4444")
	colorWarning    = lipgloss.Color("#F59E0B")
	colorInfo       = lipgloss.Color("#3B82F6")
	colorText       = lipgloss.Color("#E5E7EB")
	colorTextMuted  = lipgloss.Color("#9CA3AF")
	colorBackground = lipgloss.Color("#1F2937")
	colorBorder     = lipgloss.Color("#374151")

	// Base styles
	styleNormal = lipgloss.NewStyle()
	styleBold   = lipgloss.NewStyle().Bold(true)
	styleText   = lipgloss.NewStyle().Foreground(colorText)
	styleTextMuted = lipgloss.NewStyle().Foreground(colorTextMuted)

	// Component styles
	styleTitle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	stylePrompt = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleUserMessage = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1, 0, 2)

	styleAssistantMessage = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Padding(0, 1, 0, 2)

	styleSystemMessage = lipgloss.NewStyle().
			Foreground(colorTextMuted).
			Italic(true).
			Padding(0, 1)

	styleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess)

	styleStatus = lipgloss.NewStyle().
			Foreground(colorTextMuted).
			Background(colorBackground).
			Padding(0, 1)

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	styleSidebar = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	styleSessionItem = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	styleSessionActive = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorTextMuted).
			Padding(0, 1)

	// New styles for message parts
	styleToolUse = lipgloss.NewStyle().
			Foreground(colorInfo).
			Bold(true).
			Padding(0, 1)

	styleToolResult = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Padding(0, 1)

	styleThinking = lipgloss.NewStyle().
			Foreground(colorTextMuted).
			Italic(true).
			Padding(0, 1)

	styleTimestamp = lipgloss.NewStyle().
			Foreground(colorTextMuted).
			Padding(0, 1)
)

// Message types
type (
	// Msg is a generic message
	Msg tea.Msg

	// TickMsg is a timer tick
	TickMsg time.Time

	// StreamMsg is a streaming response chunk
	StreamMsg struct {
		Content string
		Done    bool
	}

	// ResponseMsg is a complete response
	ResponseMsg struct {
		Content string
		Error   error
	}

	// ToolCallMsg is a tool call event
	ToolCallMsg struct {
		Tool    string
		Input   string
		Result  string
		Status  string // "pending", "running", "success", "error"
	}

	// SessionMsg is a session event
	SessionMsg struct {
		ID      string
		Title   string
		Action  string // "create", "delete", "switch"
	}

	// ErrorMsg is an error message
	ErrorMsg struct {
		Error error
	}

	// ResizeMsg is a window resize event
	ResizeMsg struct {
		Width  int
		Height int
	}
)

// Role represents message role
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Message represents a chat message
type Message struct {
	ID        string
	Role      Role
	Content   string
	Timestamp time.Time
	ToolCall  *ToolCall
	// Additional fields for loaded messages
	Model     string
	Provider  string
	Parts     []Part  // For assistant messages with multiple parts
}

// Part represents a part of a message (for assistant responses)
type Part struct {
	ID         string
	Type       string  // text, tool_use, tool_result
	Text       string  // For text parts
	ToolID     string  // For tool parts
	ToolName   string  // For tool parts
	ToolInput  string  // JSON string of input
	ToolResult string  // For tool_result
	Status     string  // pending, running, success, error
}

// ToolCall represents a tool call
type ToolCall struct {
	Tool   string
	Input  string
	Result string
	Status string
}

// Session represents a session
type Session struct {
	ID        string
	Title     string
	Directory string
	CreatedAt time.Time
	Active    bool
}

// ViewState represents the current view
type ViewState string

const (
	ViewChat    ViewState = "chat"
	ViewSession ViewState = "session"
	ViewHelp    ViewState = "help"
)

// InputMode represents the input mode
type InputMode string

const (
	ModeNormal InputMode = "normal"
	ModeInput  InputMode = "input"
	ModeWait   InputMode = "wait"
)

// LoadMessagesRequest is a command to load messages from database
type LoadMessagesRequest struct {
	SessionID string
}

// LoadMessagesResult is the result of loading messages
type LoadMessagesResult struct {
	Messages []Message
	Error    error
}