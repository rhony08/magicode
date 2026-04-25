// Package dialog provides modal dialog components for the TUI.
// This file implements the help dialog showing keyboard shortcuts.
package dialog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// KeybindCategory represents a category of keybindings
type KeybindCategory struct {
	Name  string
	Items []KeybindItem
}

// KeybindItem represents a single keybinding
type KeybindItem struct {
	Key         string
	Description string
}

// HelpDialog shows keyboard shortcuts organized by category
type HelpDialog struct {
	BaseDialog

	// Categories of keybindings
	categories []KeybindCategory

	// Current scroll offset
	scrollOffset int

	// Total lines for scrolling
	totalLines int
}

// NewHelpDialog creates a new help dialog
func NewHelpDialog(theme types.Theme) *HelpDialog {
	d := &HelpDialog{
		BaseDialog: BaseDialog{
			theme:    theme,
			title:    "Keyboard Shortcuts",
			search:   "",
			selected: 0,
		},
		scrollOffset: 0,
	}

	// Build keybind categories
	d.buildKeybinds()
	d.SetItemCount(d.totalLines)

	return d
}

// buildKeybinds builds the keybind categories
func (d *HelpDialog) buildKeybinds() {
	d.categories = []KeybindCategory{
		{
			Name: "App",
			Items: []KeybindItem{
				{Key: "Ctrl+C, Ctrl+D, q", Description: "Exit"},
				{Key: "Ctrl+Z", Description: "Suspend (Unix only)"},
				{Key: "Ctrl+X", Description: "Leader key prefix"},
			},
		},
		{
			Name: "Navigation",
			Items: []KeybindItem{
				{Key: "Ctrl+X l", Description: "Session list"},
				{Key: "Ctrl+X b", Description: "Toggle sidebar"},
				{Key: "Ctrl+X g", Description: "Message timeline"},
				{Key: "Ctrl+S", Description: "Sessions"},
			},
		},
		{
			Name: "Messages",
			Items: []KeybindItem{
				{Key: "↑, k", Description: "Scroll up"},
				{Key: "↓, j", Description: "Scroll down"},
				{Key: "PgUp, Ctrl+B", Description: "Page up"},
				{Key: "PgDn, Ctrl+F", Description: "Page down"},
				{Key: "Home, g", Description: "First message"},
				{Key: "End, G", Description: "Last message"},
			},
		},
		{
			Name: "Model/Agent",
			Items: []KeybindItem{
				{Key: "F2", Description: "Cycle model"},
				{Key: "Ctrl+X m", Description: "Model list"},
				{Key: "Ctrl+X a", Description: "Agent list"},
			},
		},
		{
			Name: "Input",
			Items: []KeybindItem{
				{Key: "Enter", Description: "Submit prompt"},
				{Key: "Shift+Enter", Description: "New line"},
				{Key: "Ctrl+C", Description: "Clear input"},
				{Key: "Ctrl+V", Description: "Paste"},
				{Key: "Ctrl+-", Description: "Undo input"},
				{Key: "Ctrl+.", Description: "Redo input"},
			},
		},
		{
			Name: "Session",
			Items: []KeybindItem{
				{Key: "Ctrl+X n", Description: "New session"},
				{Key: "Ctrl+C", Description: "Interrupt"},
				{Key: "Ctrl+X c", Description: "Compact session"},
				{Key: "Ctrl+X x", Description: "Export"},
			},
		},
		{
			Name: "System",
			Items: []KeybindItem{
				{Key: "Ctrl+P", Description: "Command palette"},
				{Key: "Ctrl+X t", Description: "Themes"},
				{Key: "Ctrl+X s", Description: "Status"},
				{Key: "Ctrl+H", Description: "Help"},
			},
		},
	}

	// Count total lines
	d.totalLines = 0
	for _, cat := range d.categories {
		d.totalLines += len(cat.Items) + 2 // Category header + spacer
	}
}

// ID returns the dialog type
func (d *HelpDialog) ID() types.DialogType {
	return types.DialogHelp
}

// Init initializes the dialog
func (d *HelpDialog) Init() tea.Cmd {
	return nil
}

// Update handles events
func (d *HelpDialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		handled, cmd := d.HandleKey(msg)
		if handled {
			// Scroll handling is different for help - scroll through content
			return d, cmd
		}

	case tea.WindowSizeMsg:
		d.SetDimensions(msg.Width, msg.Height)
	}

	return d, nil
}

// NavigateUp moves scroll up
func (d *HelpDialog) NavigateUp() {
	if d.scrollOffset > 0 {
		d.scrollOffset -= 2 // Scroll by ~2 lines
	}
}

// NavigateDown moves scroll down
func (d *HelpDialog) NavigateDown() {
	maxScroll := d.totalLines - 10 // Keep some content visible
	if d.scrollOffset < maxScroll {
		d.scrollOffset += 2
	}
}

// View renders the dialog
func (d *HelpDialog) View() string {
	width := min(d.width-4, 70)
	height := min(d.height-4, 30)

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(d.theme.Primary).
		Bold(true).
		Padding(0, 1)

	categoryStyle := lipgloss.NewStyle().
		Foreground(d.theme.Secondary).
		Bold(true).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(d.theme.Text).
		Width(20).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.theme.Border).
		Padding(1, 1)

	// Build content
	var lines []string

	// Title
	lines = append(lines, titleStyle.Render(d.title))
	lines = append(lines, "")

	// Build all content first for proper scrolling
	var allLines []string
	for _, cat := range d.categories {
		// Category header
		allLines = append(allLines, categoryStyle.Render(cat.Name))

		// Keybind items
		for _, item := range cat.Items {
			keyText := keyStyle.Render(item.Key)
			descText := descStyle.Render(item.Description)
			line := lipgloss.JoinHorizontal(lipgloss.Top, keyText, descText)
			allLines = append(allLines, line)
		}

		// Spacer
		allLines = append(allLines, "")
	}

	// Calculate visible range
	visibleHeight := height - 6
	start := d.scrollOffset
	end := min(start+visibleHeight, len(allLines))

	// Add visible lines
	for i := start; i < end; i++ {
		if i < len(allLines) {
			lines = append(lines, allLines[i])
		}
	}

	// Footer
	scrollInfo := ""
	if d.totalLines > visibleHeight {
		scrollInfo = fmt.Sprintf(" (%d-%d/%d)", start+1, end, len(allLines))
	}
	hints := fmt.Sprintf("↑/↓ Scroll  Esc Close%s", scrollInfo)
	lines = append(lines, "")
	lines = append(lines, descStyle.Render(hints))

	content := strings.Join(lines, "\n")
	return borderStyle.Width(width).Height(height).Render(content)
}
