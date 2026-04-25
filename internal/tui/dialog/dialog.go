// Package dialog provides modal dialog components for the TUI.
// This file defines the Dialog interface and common types.
package dialog

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rhony08/magicode/internal/tui/types"
)

// Dialog interface that all dialogs must implement
type Dialog interface {
	// Init initializes the dialog (tea.Model interface)
	Init() tea.Cmd

	// Update handles events (tea.Model interface)
	Update(msg tea.Msg) (Dialog, tea.Cmd)

	// View renders the dialog (tea.Model interface)
	View() string

	// ID returns a unique identifier for the dialog type
	ID() types.DialogType

	// Close returns a command to close the dialog
	Close() tea.Cmd

	// SetDimensions updates dialog dimensions
	SetDimensions(width, height int)
}

// BaseDialog provides common functionality for all dialogs
type BaseDialog struct {
	// Theme for styling
	theme types.Theme

	// Dimensions
	width  int
	height int

	// Title and optional action
	title  string
	action string // Optional action button text

	// Search state
	search     string
	searchMode bool // True when search input is focused

	// Selection state
	selected int // Index of selected item
	items    int // Total number of items

	// Close callback
	closeCallback func()
}

// SetDimensions updates dialog dimensions
func (d *BaseDialog) SetDimensions(width, height int) {
	d.width = width
	d.height = height
}

// SetTheme updates dialog theme
func (d *BaseDialog) SetTheme(theme types.Theme) {
	d.theme = theme
}

// Selected returns the current selected index
func (d *BaseDialog) Selected() int {
	return d.selected
}

// SetSelected sets the selected index
func (d *BaseDialog) SetSelected(index int) {
	if index < 0 {
		d.selected = 0
	} else if index >= d.items {
		d.selected = d.items - 1
	} else {
		d.selected = index
	}
}

// SetItemCount sets the total number of selectable items
func (d *BaseDialog) SetItemCount(count int) {
	d.items = count
	if d.selected >= count && count > 0 {
		d.selected = count - 1
	}
}

// Search returns the current search text
func (d *BaseDialog) Search() string {
	return d.search
}

// SetSearch sets the search text
func (d *BaseDialog) SetSearch(text string) {
	d.search = text
	d.selected = 0 // Reset selection when search changes
}

// ClearSearch clears the search text
func (d *BaseDialog) ClearSearch() {
	d.search = ""
	d.selected = 0
}

// NavigateUp moves selection up by one
func (d *BaseDialog) NavigateUp() {
	if d.selected > 0 {
		d.selected--
	}
}

// NavigateDown moves selection down by one
func (d *BaseDialog) NavigateDown() {
	if d.selected < d.items-1 {
		d.selected++
	}
}

// NavigatePageUp moves selection up by a page
func (d *BaseDialog) NavigatePageUp(pageSize int) {
	d.selected -= pageSize
	if d.selected < 0 {
		d.selected = 0
	}
}

// NavigatePageDown moves selection down by a page
func (d *BaseDialog) NavigatePageDown(pageSize int) {
	d.selected += pageSize
	if d.selected >= d.items {
		d.selected = d.items - 1
	}
}

// HandleKey handles common key events
// Returns true if the key was handled, false otherwise
func (d *BaseDialog) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	switch {
	case msg.String() == "up" || msg.String() == "k":
		d.NavigateUp()
		return true, nil

	case msg.String() == "down" || msg.String() == "j":
		d.NavigateDown()
		return true, nil

	case msg.String() == "pgup":
		d.NavigatePageUp(10)
		return true, nil

	case msg.String() == "pgdown":
		d.NavigatePageDown(10)
		return true, nil

	case msg.String() == "home" || msg.String() == "g":
		d.selected = 0
		return true, nil

	case msg.String() == "end" || msg.String() == "G":
		d.selected = d.items - 1
		return true, nil

	case msg.String() == "esc" || msg.String() == "ctrl+c":
		return true, d.Close()

	case msg.String() == "enter":
		// Handled by specific dialog
		return false, nil

	default:
		// Check if it's a printable character for search
		if len(msg.String()) == 1 && msg.String() >= " " {
			d.search += msg.String()
			d.selected = 0
			return true, nil
		}

		// Backspace to delete search
		if msg.String() == "backspace" {
			if len(d.search) > 0 {
				d.search = d.search[:len(d.search)-1]
				d.selected = 0
			}
			return true, nil
		}

		return false, nil
	}
}

// CloseCmd returns a tea.Cmd that closes the dialog
func CloseCmd() tea.Cmd {
	return func() tea.Msg {
		return CloseMsg{}
	}
}

// CloseMsg is sent when a dialog should close
type CloseMsg struct{}

// SelectMsg is sent when an item is selected
type SelectMsg struct {
	Type  types.DialogType
	Index int
	Data  any
}

// Close returns the close command for BaseDialog
func (d *BaseDialog) Close() tea.Cmd {
	return CloseCmd()
}
