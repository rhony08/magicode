// Package dialog provides modal dialog components for the TUI.
// This file contains tests for the command palette dialog.
package dialog

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

func TestCommandPaletteDialog(t *testing.T) {
	theme := types.Theme{
		Primary:    lipgloss.Color("#7C3AED"),
		Secondary:  lipgloss.Color("#2563EB"),
		Text:       lipgloss.Color("#E5E7EB"),
		TextMuted:  lipgloss.Color("#9CA3AF"),
		Background: lipgloss.Color("#1F2937"),
		PanelBg:    lipgloss.Color("#374151"),
		Border:     lipgloss.Color("#374151"),
	}

	keybinds := map[string]string{
		"new_session": "Ctrl+X n",
	}

	d := NewCommandPaletteDialog(theme, keybinds)

	if d.ID() != types.DialogCommand {
		t.Errorf("ID() = %v, want DialogCommand", d.ID())
	}

	if d.title != "Command Palette" {
		t.Errorf("title = %v, want Command Palette", d.title)
	}

	// Should have commands
	if len(d.commands) == 0 {
		t.Error("commands should not be empty")
	}
}

func TestCommandPaletteDialogFilter(t *testing.T) {
	theme := types.Theme{
		Primary:    lipgloss.Color("#7C3AED"),
		Text:       lipgloss.Color("#E5E7EB"),
		TextMuted:  lipgloss.Color("#9CA3AF"),
		Background: lipgloss.Color("#1F2937"),
		PanelBg:    lipgloss.Color("#374151"),
		Border:     lipgloss.Color("#374151"),
	}

	d := NewCommandPaletteDialog(theme, nil)

	// Initial state - all commands visible
	if len(d.filtered) != len(d.commands) {
		t.Errorf("filtered count = %d, want %d", len(d.filtered), len(d.commands))
	}

	// Filter by search
	d.SetSearch("new")
	d.filterCommands()

	// Should have filtered results
	if len(d.filtered) == len(d.commands) {
		t.Error("filter should have reduced results")
	}

	// Should find "new_session" command
	found := false
	for _, cmd := range d.filtered {
		if cmd.Name == "new_session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtered should contain 'new_session'")
	}

	// Filter by category
	d.SetSearch("session")
	d.filterCommands()

	// Should find session-related commands
	found = false
	for _, cmd := range d.filtered {
		if cmd.Category == "Session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("filtered should contain Session category commands")
	}
}

func TestCommandPaletteDialogView(t *testing.T) {
	theme := types.Theme{
		Primary:    lipgloss.Color("#7C3AED"),
		Secondary:  lipgloss.Color("#2563EB"),
		Text:       lipgloss.Color("#E5E7EB"),
		TextMuted:  lipgloss.Color("#9CA3AF"),
		Background: lipgloss.Color("#1F2937"),
		PanelBg:    lipgloss.Color("#374151"),
		Border:     lipgloss.Color("#374151"),
	}

	d := NewCommandPaletteDialog(theme, nil)
	d.SetDimensions(100, 30)

	view := d.View()

	if view == "" {
		t.Error("View() should not return empty")
	}

	// Should contain title
	if !strings.Contains(view, "Command Palette") {
		t.Error("View() should contain title")
	}

	// Should contain commands
	foundCommand := false
	for _, cmd := range d.commands {
		if strings.Contains(view, cmd.Description) {
			foundCommand = true
			break
		}
	}
	if !foundCommand {
		t.Error("View() should contain at least one command")
	}
}

func TestCommandPaletteDialogUpdate(t *testing.T) {
	theme := types.Theme{
		Primary:    lipgloss.Color("#7C3AED"),
		Text:       lipgloss.Color("#E5E7EB"),
		TextMuted:  lipgloss.Color("#9CA3AF"),
		Background: lipgloss.Color("#1F2937"),
		PanelBg:    lipgloss.Color("#374151"),
		Border:     lipgloss.Color("#374151"),
	}

	d := NewCommandPaletteDialog(theme, nil)

	// Test resize
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	newD, _ := d.Update(msg)
	if newD == nil {
		t.Error("Update(WindowSizeMsg) should return non-nil dialog")
	}
}

func TestCommandPaletteDialogCommands(t *testing.T) {
	theme := types.Theme{
		Primary:    lipgloss.Color("#7C3AED"),
		Text:       lipgloss.Color("#E5E7EB"),
		TextMuted:  lipgloss.Color("#9CA3AF"),
		Background: lipgloss.Color("#1F2937"),
		PanelBg:    lipgloss.Color("#374151"),
		Border:     lipgloss.Color("#374151"),
	}

	d := NewCommandPaletteDialog(theme, nil)

	// Check that we have the expected command categories
	categories := make(map[string]bool)
	for _, cmd := range d.commands {
		categories[cmd.Category] = true
	}

	expectedCategories := []string{"Session", "Navigation", "Model", "Agent", "Message", "System"}
	for _, cat := range expectedCategories {
		if !categories[cat] {
			t.Errorf("Missing category: %s", cat)
		}
	}

	// Check for specific commands
	commandNames := make(map[string]bool)
	for _, cmd := range d.commands {
		commandNames[cmd.Name] = true
	}

	expectedCommands := []string{
		"new_session",
		"list_sessions",
		"toggle_sidebar",
		"model_list",
		"theme_list",
		"help",
		"quit",
	}

	for _, cmd := range expectedCommands {
		if !commandNames[cmd] {
			t.Errorf("Missing command: %s", cmd)
		}
	}
}
