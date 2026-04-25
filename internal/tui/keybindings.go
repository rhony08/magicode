// Package tui provides keybinding definitions.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
)

// Keybinding represents a keyboard shortcut
type Keybinding struct {
	Key  tea.KeyType
	Str  string
	Alt  bool
	Ctrl bool
}

// Match checks if a key message matches the keybinding
func (kb *Keybinding) Match(msg tea.KeyMsg) bool {
	// Check key type
	if msg.Type == kb.Key {
		return true
	}

	// Check string match (this handles ctrl+key, alt+key, etc.)
	msgStr := msg.String()
	if msgStr == kb.Str {
		return true
	}

	// For Ctrl keys, also check without the ctrl+ prefix
	if kb.Ctrl {
		baseKey := strings.TrimPrefix(kb.Str, "ctrl+")
		if baseKey != kb.Str {
			// The keybinding has ctrl+ prefix, check if message matches
			return strings.Contains(msgStr, baseKey) && strings.Contains(msgStr, "ctrl")
		}
	}

	return false
}

// Keybindings holds all keybindings
type Keybindings struct {
	Submit         Keybinding
	Quit           Keybinding
	Back           Keybinding
	Sessions       Keybinding
	Help           Keybinding
	NewSession     Keybinding
	Up             Keybinding
	Down           Keybinding
	PageUp         Keybinding
	PageDown       Keybinding
	HistoryUp      Keybinding
	HistoryDown    Keybinding
	Select         Keybinding
	Cancel         Keybinding
	Tab            Keybinding
	ShiftTab       Keybinding
	CommandPalette Keybinding
}

// DefaultKeybindings returns the default keybindings
func DefaultKeybindings() Keybindings {
	return Keybindings{
		Submit: Keybinding{
			Key: tea.KeyEnter,
			Str: "enter",
		},
		Quit: Keybinding{
			Key: tea.KeyCtrlC,
			Str: "q",
		},
		Back: Keybinding{
			Str: "esc",
		},
		Sessions: Keybinding{
			Str:  "ctrl+s",
			Ctrl: true,
		},
		Help: Keybinding{
			Str:  "ctrl+h",
			Ctrl: true,
		},
		NewSession: Keybinding{
			Str:  "ctrl+n",
			Ctrl: true,
		},
		Up: Keybinding{
			Key: tea.KeyUp,
			Str: "up",
		},
		Down: Keybinding{
			Key: tea.KeyDown,
			Str: "down",
		},
		PageUp: Keybinding{
			Key: tea.KeyPgUp,
			Str: "pgup",
		},
		PageDown: Keybinding{
			Key: tea.KeyPgDown,
			Str: "pgdown",
		},
		HistoryUp: Keybinding{
			Str:  "ctrl+up",
			Ctrl: true,
		},
		HistoryDown: Keybinding{
			Str:  "ctrl+down",
			Ctrl: true,
		},
		Select: Keybinding{
			Key: tea.KeyEnter,
			Str: "enter",
		},
		Cancel: Keybinding{
			Str: "esc",
		},
		Tab: Keybinding{
			Key: tea.KeyTab,
			Str: "tab",
		},
		ShiftTab: Keybinding{
			Key: tea.KeyShiftTab,
			Str: "shift+tab",
		},
		CommandPalette: Keybinding{
			Str:  "ctrl+p",
			Ctrl: true,
		},
	}
}

// KeyNames returns human-readable key names
func KeyNames(kb Keybindings) map[string]string {
	return map[string]string{
		"submit":         "Enter",
		"quit":           "Ctrl+C / q",
		"back":           "Esc",
		"sessions":       "Ctrl+S",
		"help":           "Ctrl+H",
		"newSession":     "Ctrl+N",
		"up":             "↑",
		"down":           "↓",
		"pageUp":         "PgUp",
		"pageDown":       "PgDn",
		"historyUp":      "Ctrl+↑",
		"historyDown":    "Ctrl+↓",
		"select":         "Enter",
		"cancel":         "Esc",
		"tab":            "Tab",
		"shiftTab":       "Shift+Tab",
		"commandPalette": "Ctrl+P",
	}
}

// FormatKeybinding formats a keybinding for display
func FormatKeybinding(kb Keybinding) string {
	var parts []string

	if kb.Ctrl {
		parts = append(parts, "Ctrl")
	}
	if kb.Alt {
		parts = append(parts, "Alt")
	}

	if kb.Str != "" && kb.Str != "enter" && kb.Str != "esc" && kb.Str != "tab" {
		// Custom string representation
		if len(parts) > 0 {
			return strings.Join(parts, "+") + "+" + kb.Str
		}
		return kb.Str
	}

	// Use key type name
	keyName := ""
	switch kb.Key {
	case tea.KeyEnter:
		keyName = "Enter"
	case tea.KeyEsc:
		keyName = "Esc"
	case tea.KeyTab:
		keyName = "Tab"
	case tea.KeyUp:
		keyName = "↑"
	case tea.KeyDown:
		keyName = "↓"
	case tea.KeyLeft:
		keyName = "←"
	case tea.KeyRight:
		keyName = "→"
	case tea.KeyPgUp:
		keyName = "PgUp"
	case tea.KeyPgDown:
		keyName = "PgDn"
	case tea.KeyHome:
		keyName = "Home"
	case tea.KeyEnd:
		keyName = "End"
	case tea.KeyCtrlC:
		keyName = "Ctrl+C"
	case tea.KeyCtrlD:
		keyName = "Ctrl+D"
	case tea.KeyCtrlL:
		keyName = "Ctrl+L"
	case tea.KeyCtrlU:
		keyName = "Ctrl+U"
	case tea.KeyCtrlK:
		keyName = "Ctrl+K"
	case tea.KeySpace:
		keyName = "Space"
	case tea.KeyBackspace:
		keyName = "Backspace"
	case tea.KeyDelete:
		keyName = "Delete"
	default:
		if kb.Str != "" {
			keyName = kb.Str
		} else {
			keyName = "Unknown"
		}
	}

	if len(parts) > 0 {
		return strings.Join(parts, "+") + "+" + keyName
	}
	return keyName
}
