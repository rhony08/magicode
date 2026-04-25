// Package dialog provides modal dialog components for the TUI.
// This file implements the command palette for executing commands.
package dialog

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// CommandPaletteDialog shows a searchable list of available commands
type CommandPaletteDialog struct {
	BaseDialog

	// Available commands
	commands []CommandItem

	// Filtered commands based on search
	filtered []CommandItem

	// Keybind registry for displaying shortcuts
	keybinds map[string]string
}

// CommandItem represents a command in the palette
type CommandItem struct {
	Name        string
	Description string
	Action      string
	Keybind     string
	Category    string
}

// NewCommandPaletteDialog creates a new command palette dialog
func NewCommandPaletteDialog(theme types.Theme, keybinds map[string]string) *CommandPaletteDialog {
	d := &CommandPaletteDialog{
		BaseDialog: BaseDialog{
			theme:    theme,
			title:    "Command Palette",
			action:   "Enter: Execute",
			search:   "",
			selected: 0,
		},
		keybinds: keybinds,
	}

	// Define available commands
	d.commands = []CommandItem{
		// Session commands
		{Name: "new_session", Description: "Create a new session", Action: "new_session", Keybind: "Ctrl+X n", Category: "Session"},
		{Name: "list_sessions", Description: "Show session list", Action: "session_list", Keybind: "Ctrl+X l", Category: "Session"},
		{Name: "export_session", Description: "Export current session", Action: "export_session", Keybind: "Ctrl+X x", Category: "Session"},
		{Name: "compact_session", Description: "Compact current session", Action: "compact_session", Keybind: "Ctrl+X c", Category: "Session"},
		{Name: "timeline", Description: "Show message timeline", Action: "timeline", Keybind: "Ctrl+X g", Category: "Session"},

		// Navigation commands
		{Name: "toggle_sidebar", Description: "Toggle sidebar visibility", Action: "toggle_sidebar", Keybind: "Ctrl+X b", Category: "Navigation"},

		// Model/Agent commands
		{Name: "model_list", Description: "Show model selection dialog", Action: "model_list", Keybind: "Ctrl+X m", Category: "Model"},
		{Name: "cycle_model", Description: "Cycle to next model", Action: "cycle_model", Keybind: "F2", Category: "Model"},
		{Name: "agent_list", Description: "Show agent selection dialog", Action: "agent_list", Keybind: "Ctrl+X a", Category: "Agent"},
		{Name: "cycle_agent", Description: "Cycle to next agent", Action: "cycle_agent", Keybind: "Tab", Category: "Agent"},
		{Name: "cycle_variant", Description: "Cycle to next variant", Action: "cycle_variant", Keybind: "Ctrl+T", Category: "Agent"},

		// Message commands
		{Name: "copy_last", Description: "Copy last message", Action: "copy_last", Keybind: "Ctrl+X y", Category: "Message"},
		{Name: "undo_message", Description: "Undo last message edit", Action: "undo_message", Keybind: "Ctrl+X u", Category: "Message"},
		{Name: "redo_message", Description: "Redo last message edit", Action: "redo_message", Keybind: "Ctrl+X r", Category: "Message"},

		// System commands
		{Name: "theme_list", Description: "Show theme selection dialog", Action: "theme_list", Keybind: "Ctrl+X t", Category: "System"},
		{Name: "status", Description: "Show system status", Action: "status", Keybind: "Ctrl+X s", Category: "System"},
		{Name: "help", Description: "Show help dialog", Action: "help", Keybind: "Ctrl+X h", Category: "System"},
		{Name: "quit", Description: "Exit application", Action: "quit", Keybind: "Ctrl+X q", Category: "System"},
	}

	// Apply custom keybinds if provided
	if keybinds != nil {
		for i, cmd := range d.commands {
			if customKey, ok := keybinds[cmd.Action]; ok {
				d.commands[i].Keybind = customKey
			}
		}
	}

	d.filtered = d.commands
	d.SetItemCount(len(d.filtered))

	return d
}

// ID returns the dialog type
func (d *CommandPaletteDialog) ID() types.DialogType {
	return types.DialogCommand
}

// Init initializes the dialog
func (d *CommandPaletteDialog) Init() tea.Cmd {
	return nil
}

// Update handles events
func (d *CommandPaletteDialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		handled, cmd := d.HandleKey(msg)
		if handled {
			// Re-filter after search change
			d.filterCommands()
			return d, cmd
		}

		if msg.String() == "enter" && len(d.filtered) > 0 {
			// Execute the selected command
			selected := d.filtered[d.Selected()]
			return d, tea.Batch(
				CloseCmd(),
				func() tea.Msg {
					return SelectMsg{
						Type:  types.DialogCommand,
						Index: d.Selected(),
						Data:  selected.Action,
					}
				},
			)
		}

	case tea.WindowSizeMsg:
		d.SetDimensions(msg.Width, msg.Height)
	}

	return d, nil
}

// filterCommands filters the command list based on search text
func (d *CommandPaletteDialog) filterCommands() {
	if d.search == "" {
		d.filtered = d.commands
	} else {
		d.filtered = []CommandItem{}
		searchLower := strings.ToLower(d.search)

		for _, cmd := range d.commands {
			// Search in name, description, and category
			if strings.Contains(strings.ToLower(cmd.Name), searchLower) ||
				strings.Contains(strings.ToLower(cmd.Description), searchLower) ||
				strings.Contains(strings.ToLower(cmd.Category), searchLower) {
				d.filtered = append(d.filtered, cmd)
			}
		}
	}

	d.SetItemCount(len(d.filtered))
	if d.selected >= len(d.filtered) {
		d.selected = 0
	}
}

// View renders the dialog
func (d *CommandPaletteDialog) View() string {
	// Dialog dimensions
	width := min(d.width-4, 70)
	height := min(d.height-4, 25)

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(d.theme.Primary).
		Bold(true).
		Padding(0, 1)

	searchStyle := lipgloss.NewStyle().
		Foreground(d.theme.Text).
		Background(d.theme.PanelBg).
		Padding(0, 1).
		Width(width - 4)

	itemStyle := lipgloss.NewStyle().
		Foreground(d.theme.Text).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(d.theme.Background).
		Background(d.theme.Primary).
		Padding(0, 1).
		Bold(true)

	keybindStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted).
		Align(lipgloss.Right)

	categoryStyle := lipgloss.NewStyle().
		Foreground(d.theme.Secondary).
		Bold(true)

	mutedStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted).
		Padding(0, 1)

	// Build dialog content
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render(d.title))
	content.WriteString("\n\n")

	// Search box
	searchText := d.search
	if searchText == "" {
		searchText = "Type to search commands..."
		searchStyle = searchStyle.Foreground(d.theme.TextMuted)
	}
	content.WriteString(searchStyle.Render("▶ " + searchText))
	content.WriteString("\n\n")

	// Command list
	listHeight := height - 6 // Account for title, search, and hints
	startIdx := 0
	endIdx := len(d.filtered)

	// Scroll if list is too long
	if len(d.filtered) > listHeight {
		// Center selection
		startIdx = d.selected - listHeight/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + listHeight
		if endIdx > len(d.filtered) {
			endIdx = len(d.filtered)
			startIdx = endIdx - listHeight
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Show scroll indicators
	if startIdx > 0 {
		content.WriteString(mutedStyle.Render("  ▲ ..."))
		content.WriteString("\n")
	}

	// Current category for grouping
	currentCategory := ""

	for i := startIdx; i < endIdx; i++ {
		cmd := d.filtered[i]
		isSelected := i == d.selected

		// Add category header if changed
		if cmd.Category != currentCategory {
			currentCategory = cmd.Category
			content.WriteString("\n")
			content.WriteString(categoryStyle.Render("  " + cmd.Category))
			content.WriteString("\n")
		}

		// Format: "  Command Name         Keybind"
		nameWidth := width - 18 // Space for keybind display

		name := cmd.Description
		if len(name) > nameWidth-4 {
			name = name[:nameWidth-7] + "..."
		}

		keybind := cmd.Keybind
		if keybind == "" {
			keybind = "-"
		}

		// Build the line
		var line strings.Builder
		line.WriteString("  ")
		if isSelected {
			line.WriteString(selectedStyle.Render(name))
		} else {
			line.WriteString(itemStyle.Render(name))
		}

		// Pad and add keybind
		nameLen := lipgloss.Width(name)
		padding := nameWidth - nameLen
		if padding < 0 {
			padding = 0
		}
		line.WriteString(strings.Repeat(" ", padding))
		line.WriteString(keybindStyle.Render(keybind))

		content.WriteString(line.String())
		content.WriteString("\n")
	}

	if endIdx < len(d.filtered) {
		content.WriteString(mutedStyle.Render("  ▼ ..."))
		content.WriteString("\n")
	}

	// Fill empty space
	linesRendered := endIdx - startIdx
	if startIdx > 0 {
		linesRendered++
	}
	if endIdx < len(d.filtered) {
		linesRendered++
	}
	// Account for category headers
	categoryCount := 0
	if len(d.filtered) > 0 {
		seenCategories := make(map[string]bool)
		for i := startIdx; i < endIdx; i++ {
			if !seenCategories[d.filtered[i].Category] {
				seenCategories[d.filtered[i].Category] = true
				categoryCount++
			}
		}
	}
	for i := linesRendered + categoryCount; i < listHeight; i++ {
		content.WriteString("\n")
	}

	// Hints
	content.WriteString("\n")
	hintStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted).
		Padding(0, 1)
	content.WriteString(hintStyle.Render("Enter: Execute  Esc: Cancel"))

	// Dialog border
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.theme.Border).
		Padding(1, 2).
		Width(width)

	return dialogStyle.Render(content.String())
}
