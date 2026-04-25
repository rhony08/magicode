// Package tui provides the leader key system for keyboard shortcuts.
// This file implements the Ctrl+X prefix handling with timeout, matching OpenCode's behavior.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// LeaderTimeout is the duration to wait for the next key after Ctrl+X
const LeaderTimeout = 2000 // 2 seconds (matching OpenCode's timeout)

// LeaderState represents the current leader key state
type LeaderState int

const (
	LeaderStateNone     LeaderState = 0 // No leader key pressed
	LeaderStateActive   LeaderState = 1 // Ctrl+X pressed, waiting for next key
	LeaderStateComplete LeaderState = 2 // Leader sequence complete
)

// LeaderKeyHandler manages the leader key (Ctrl+X) prefix handling
type LeaderKeyHandler struct {
	state       LeaderState
	buffer      string // Buffer for the second key
	timer       *time.Timer
	timeout     time.Duration
	keybindings LeaderKeyBindings
}

// LeaderKeyBindings maps leader key sequences to actions
type LeaderKeyBindings struct {
	// Session commands
	SessionList     LeaderAction
	SessionNew      LeaderAction
	SessionExport   LeaderAction
	SessionCompact  LeaderAction
	SessionTimeline LeaderAction

	// Navigation
	SidebarToggle LeaderAction

	// Model/Agent
	ModelList LeaderAction
	AgentList LeaderAction

	// Messages
	MessageCopy LeaderAction
	MessageUndo LeaderAction
	MessageRedo LeaderAction

	// System
	ThemeList  LeaderAction
	StatusView LeaderAction
	HelpDialog LeaderAction

	// Exit
	ExitApp LeaderAction
}

// LeaderAction represents an action triggered by a leader key sequence
type LeaderAction struct {
	Key         string
	Description string
	Category    string
}

// DefaultLeaderKeyBindings returns the default leader key bindings
func DefaultLeaderKeyBindings() LeaderKeyBindings {
	return LeaderKeyBindings{
		SessionList: LeaderAction{
			Key:         "l",
			Description: "Session list",
			Category:    "Session",
		},
		SessionNew: LeaderAction{
			Key:         "n",
			Description: "New session",
			Category:    "Session",
		},
		SessionExport: LeaderAction{
			Key:         "x",
			Description: "Export session",
			Category:    "Session",
		},
		SessionCompact: LeaderAction{
			Key:         "c",
			Description: "Compact session",
			Category:    "Session",
		},
		SessionTimeline: LeaderAction{
			Key:         "g",
			Description: "Timeline",
			Category:    "Session",
		},
		SidebarToggle: LeaderAction{
			Key:         "b",
			Description: "Toggle sidebar",
			Category:    "Navigation",
		},
		ModelList: LeaderAction{
			Key:         "m",
			Description: "Model list",
			Category:    "Model/Agent",
		},
		AgentList: LeaderAction{
			Key:         "a",
			Description: "Agent list",
			Category:    "Model/Agent",
		},
		MessageCopy: LeaderAction{
			Key:         "y",
			Description: "Copy last message",
			Category:    "Messages",
		},
		MessageUndo: LeaderAction{
			Key:         "u",
			Description: "Undo",
			Category:    "Messages",
		},
		MessageRedo: LeaderAction{
			Key:         "r",
			Description: "Redo",
			Category:    "Messages",
		},
		ThemeList: LeaderAction{
			Key:         "t",
			Description: "Theme list",
			Category:    "System",
		},
		StatusView: LeaderAction{
			Key:         "s",
			Description: "Status view",
			Category:    "System",
		},
		HelpDialog: LeaderAction{
			Key:         "h",
			Description: "Help dialog",
			Category:    "System",
		},
		ExitApp: LeaderAction{
			Key:         "q",
			Description: "Exit",
			Category:    "App",
		},
	}
}

// NewLeaderKeyHandler creates a new leader key handler
func NewLeaderKeyHandler() *LeaderKeyHandler {
	return &LeaderKeyHandler{
		state:       LeaderStateNone,
		buffer:      "",
		timeout:     LeaderTimeout * time.Millisecond,
		keybindings: DefaultLeaderKeyBindings(),
	}
}

// LeaderKeyMsg is sent when a leader key action is triggered
type LeaderKeyMsg struct {
	Action string
	Key    string
}

// LeaderTimeoutMsg is sent when the leader key timeout expires
type LeaderTimeoutMsg struct{}

// LeaderStatusMsg provides the current leader key status
type LeaderStatusMsg struct {
	Active  bool
	Buffer  string
	Pending bool
}

// HandleKey processes a key event and returns true if handled
// Returns (handled, cmd, msg) where msg is non-nil if an action was triggered
func (h *LeaderKeyHandler) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd, *LeaderKeyMsg) {
	keyStr := strings.ToLower(msg.String())

	switch h.state {
	case LeaderStateNone:
		// Check if this is the leader key (Ctrl+X)
		if keyStr == "ctrl+x" {
			h.activate()
			return true, h.startTimer(), nil
		}
		return false, nil, nil

	case LeaderStateActive:
		// Check for timeout or second key
		if h.isTimeout() {
			h.reset()
			return false, nil, nil
		}

		// Process the second key
		action := h.processSecondKey(keyStr)
		h.reset()

		if action != nil {
			return true, nil, action
		}
		// Unknown sequence, just consume it
		return true, nil, nil

	case LeaderStateComplete:
		// Should not happen, but reset just in case
		h.reset()
		return false, nil, nil
	}

	return false, nil, nil
}

// IsLeaderActive returns true if the leader key is active (waiting for second key)
func (h *LeaderKeyHandler) IsLeaderActive() bool {
	return h.state == LeaderStateActive && !h.isTimeout()
}

// GetStatus returns the current leader key status
func (h *LeaderKeyHandler) GetStatus() LeaderStatusMsg {
	return LeaderStatusMsg{
		Active:  h.state == LeaderStateActive,
		Buffer:  h.buffer,
		Pending: h.state == LeaderStateActive && !h.isTimeout(),
	}
}

// GetPendingKey returns the current buffer when leader is active
func (h *LeaderKeyHandler) GetPendingKey() string {
	if h.IsLeaderActive() {
		return h.buffer
	}
	return ""
}

// GetAllActions returns all available leader actions for help display
func (h *LeaderKeyHandler) GetAllActions() []LeaderAction {
	return []LeaderAction{
		h.keybindings.SessionList,
		h.keybindings.SessionNew,
		h.keybindings.SessionExport,
		h.keybindings.SessionCompact,
		h.keybindings.SessionTimeline,
		h.keybindings.SidebarToggle,
		h.keybindings.ModelList,
		h.keybindings.AgentList,
		h.keybindings.MessageCopy,
		h.keybindings.MessageUndo,
		h.keybindings.MessageRedo,
		h.keybindings.ThemeList,
		h.keybindings.StatusView,
		h.keybindings.HelpDialog,
		h.keybindings.ExitApp,
	}
}

// GetActionsByCategory returns actions grouped by category
func (h *LeaderKeyHandler) GetActionsByCategory() map[string][]LeaderAction {
	result := make(map[string][]LeaderAction)
	for _, action := range h.GetAllActions() {
		result[action.Category] = append(result[action.Category], action)
	}
	return result
}

// FormatLeaderKey formats a key as a leader key sequence for display
func FormatLeaderKey(key string) string {
	return fmt.Sprintf("Ctrl+X %s", strings.ToUpper(key))
}

// FormatLeaderKeyShort returns just the key after leader for compact display
func FormatLeaderKeyShort(key string) string {
	return strings.ToLower(key)
}

// internal methods

func (h *LeaderKeyHandler) activate() {
	h.state = LeaderStateActive
	h.buffer = ""
}

func (h *LeaderKeyHandler) reset() {
	h.state = LeaderStateNone
	h.buffer = ""
	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
}

func (h *LeaderKeyHandler) startTimer() tea.Cmd {
	return tea.Tick(h.timeout, func(t time.Time) tea.Msg {
		return LeaderTimeoutMsg{}
	})
}

func (h *LeaderKeyHandler) isTimeout() bool {
	// The timer check is handled by the tea.Tick message
	// This is a placeholder for potential future time-based checks
	return false
}

func (h *LeaderKeyHandler) processSecondKey(key string) *LeaderKeyMsg {
	switch key {
	case h.keybindings.SessionList.Key:
		return &LeaderKeyMsg{Action: "session_list", Key: key}
	case h.keybindings.SessionNew.Key:
		return &LeaderKeyMsg{Action: "session_new", Key: key}
	case h.keybindings.SessionExport.Key:
		return &LeaderKeyMsg{Action: "session_export", Key: key}
	case h.keybindings.SessionCompact.Key:
		return &LeaderKeyMsg{Action: "session_compact", Key: key}
	case h.keybindings.SessionTimeline.Key:
		return &LeaderKeyMsg{Action: "session_timeline", Key: key}
	case h.keybindings.SidebarToggle.Key:
		return &LeaderKeyMsg{Action: "sidebar_toggle", Key: key}
	case h.keybindings.ModelList.Key:
		return &LeaderKeyMsg{Action: "model_list", Key: key}
	case h.keybindings.AgentList.Key:
		return &LeaderKeyMsg{Action: "agent_list", Key: key}
	case h.keybindings.MessageCopy.Key:
		return &LeaderKeyMsg{Action: "message_copy", Key: key}
	case h.keybindings.MessageUndo.Key:
		return &LeaderKeyMsg{Action: "message_undo", Key: key}
	case h.keybindings.MessageRedo.Key:
		return &LeaderKeyMsg{Action: "message_redo", Key: key}
	case h.keybindings.ThemeList.Key:
		return &LeaderKeyMsg{Action: "theme_list", Key: key}
	case h.keybindings.StatusView.Key:
		return &LeaderKeyMsg{Action: "status_view", Key: key}
	case h.keybindings.HelpDialog.Key:
		return &LeaderKeyMsg{Action: "help_dialog", Key: key}
	case h.keybindings.ExitApp.Key:
		return &LeaderKeyMsg{Action: "exit_app", Key: key}
	default:
		return nil
	}
}

// LeaderTimeout returns the timeout duration
func (h *LeaderKeyHandler) LeaderTimeout() time.Duration {
	return h.timeout
}

// SetTimeout sets a custom timeout duration
func (h *LeaderKeyHandler) SetTimeout(timeout time.Duration) {
	h.timeout = timeout
}
