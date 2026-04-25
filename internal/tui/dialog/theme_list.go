// Package dialog provides modal dialog components for the TUI.
// This file implements the theme selection dialog.
package dialog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// ThemeListDialog shows a list of available themes
type ThemeListDialog struct {
	BaseDialog

	// Available themes
	themes []types.Theme

	// State reference
	state *types.AppState
}

// NewThemeListDialog creates a new theme list dialog
// The themes parameter should be obtained from the registry
func NewThemeListDialog(theme types.Theme, state *types.AppState, themes []types.Theme) *ThemeListDialog {
	d := &ThemeListDialog{
		BaseDialog: BaseDialog{
			theme:    theme,
			title:    "Select Theme",
			action:   "",
			search:   "",
			selected: 0,
		},
		state:  state,
		themes: themes,
	}

	if d.themes == nil {
		d.themes = []types.Theme{}
	}

	d.SetItemCount(len(d.themes))

	// Select current theme
	for i, t := range d.themes {
		if t.ID == state.KV.Theme {
			d.selected = i
			break
		}
	}

	return d
}

// ID returns the dialog type
func (d *ThemeListDialog) ID() types.DialogType {
	return types.DialogThemeList
}

// Init initializes the dialog
func (d *ThemeListDialog) Init() tea.Cmd {
	return nil
}

// Update handles events
func (d *ThemeListDialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		handled, cmd := d.HandleKey(msg)
		if handled {
			d.filterThemes()
			return d, cmd
		}

		if msg.String() == "enter" && len(d.themes) > 0 {
			selected := d.themes[d.Selected()]
			return d, tea.Batch(
				CloseCmd(),
				func() tea.Msg {
					return SelectMsg{
						Type:  types.DialogThemeList,
						Index: d.Selected(),
						Data:  selected.ID,
					}
				},
			)
		}

	case tea.WindowSizeMsg:
		d.SetDimensions(msg.Width, msg.Height)
	}

	return d, nil
}

// filterThemes filters the theme list based on search text
func (d *ThemeListDialog) filterThemes() {
	if d.search != "" {
		searchLower := strings.ToLower(d.search)
		filtered := []types.Theme{}
		for _, t := range d.themes {
			if strings.Contains(strings.ToLower(t.Name), searchLower) ||
				strings.Contains(strings.ToLower(t.ID), searchLower) {
				filtered = append(filtered, t)
			}
		}
		d.themes = filtered
	}

	d.SetItemCount(len(d.themes))
	if d.selected >= len(d.themes) {
		d.selected = 0
	}
}

// View renders the dialog
func (d *ThemeListDialog) View() string {
	width := min(d.width-4, 50)
	height := min(d.height-4, 18)

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
		Foreground(d.theme.Primary).
		Bold(true).
		Background(d.theme.PanelBg).
		Padding(0, 1)

	mutedStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted)

	currentStyle := lipgloss.NewStyle().
		Foreground(d.theme.Success).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.theme.Border).
		Padding(1, 1)

	// Build content
	var lines []string

	// Title
	lines = append(lines, titleStyle.Render(d.title))

	// Search
	searchText := "🔍 " + d.search
	if d.search == "" {
		searchText = "🔍 Search themes..."
	}
	lines = append(lines, searchStyle.Render(searchText))
	lines = append(lines, "")

	// Theme list
	visibleHeight := height - 6
	startIdx := d.selected
	if startIdx > len(d.themes)-visibleHeight {
		startIdx = max(0, len(d.themes)-visibleHeight)
	}

	for i := startIdx; i < len(d.themes) && i < startIdx+visibleHeight; i++ {
		t := d.themes[i]

		// Format theme line
		var line string
		if t.ID == d.state.KV.Theme {
			line = fmt.Sprintf("✓ %s", t.Name)
		} else {
			line = fmt.Sprintf("  %s", t.Name)
		}

		// Show preview color
		preview := lipgloss.NewStyle().
			Background(t.Primary).
			Render("   ")
		line = lipgloss.JoinHorizontal(lipgloss.Left, line, "  ", preview)

		if i == d.selected {
			lines = append(lines, selectedStyle.Render(line))
		} else if t.ID == d.state.KV.Theme {
			lines = append(lines, currentStyle.Render(line))
		} else {
			lines = append(lines, itemStyle.Render(line))
		}
	}

	// Empty state
	if len(d.themes) == 0 {
		emptyText := "No themes found"
		if d.search != "" {
			emptyText = fmt.Sprintf("No themes matching \"%s\"", d.search)
		}
		lines = append(lines, mutedStyle.Render(emptyText))
	}

	// Footer
	hints := "↑/↓ Navigate  Enter Select  Esc Close"
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render(hints))

	content := strings.Join(lines, "\n")
	return borderStyle.Width(width).Height(height).Render(content)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
