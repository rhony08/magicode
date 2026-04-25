// Package dialog provides modal dialog components for the TUI.
// This file implements the theme list dialog for theme selection.
package dialog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// ThemeListDialog shows a list of available themes for selection
type ThemeListDialog struct {
	BaseDialog

	// Themes to display
	themes []ThemeItem

	// Filtered themes based on search
	filtered []ThemeItem

	// Currently selected theme ID
	currentTheme string
}

// ThemeItem represents a theme in the list
type ThemeItem struct {
	ID          string
	Name        string
	Description string
	IsDark      bool
}

// NewThemeListDialog creates a new theme list dialog
func NewThemeListDialog(theme types.Theme, state *types.AppState, availableThemes []types.Theme) *ThemeListDialog {
	d := &ThemeListDialog{
		BaseDialog: BaseDialog{
			theme:    theme,
			title:    "Select Theme",
			action:   "Enter Select",
			search:   "",
			selected: 0,
		},
		currentTheme: state.KV.Theme,
	}

	// Convert types.Theme to ThemeItem
	d.themes = make([]ThemeItem, 0, len(availableThemes))
	for _, t := range availableThemes {
		description := "Dark theme"
		if !t.IsDark {
			description = "Light theme"
		}
		d.themes = append(d.themes, ThemeItem{
			ID:          t.ID,
			Name:        t.Name,
			Description: description,
			IsDark:      t.IsDark,
		})
	}

	d.filtered = d.themes
	d.SetItemCount(len(d.filtered))

	// Set initial selection to current theme
	for i, t := range d.filtered {
		if t.ID == d.currentTheme {
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
			// Re-filter after search change
			d.filterThemes()
			return d, cmd
		}

		if msg.String() == "enter" && len(d.filtered) > 0 {
			// Select the theme
			selected := d.filtered[d.Selected()]
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

		// Number keys 1-8 for quick selection
		key := msg.String()
		if len(key) == 1 && key >= "1" && key <= "8" {
			index := int(key[0] - '1')
			if index < len(d.filtered) {
				d.selected = index
				selected := d.filtered[index]
				return d, tea.Batch(
					CloseCmd(),
					func() tea.Msg {
						return SelectMsg{
							Type:  types.DialogThemeList,
							Index: index,
							Data:  selected.ID,
						}
					},
				)
			}
		}

	case tea.WindowSizeMsg:
		d.SetDimensions(msg.Width, msg.Height)
	}

	return d, nil
}

// filterThemes filters the theme list based on search text
func (d *ThemeListDialog) filterThemes() {
	if d.search == "" {
		d.filtered = d.themes
	} else {
		d.filtered = []ThemeItem{}
		searchLower := strings.ToLower(d.search)

		for _, t := range d.themes {
			// Search in name and description
			if strings.Contains(strings.ToLower(t.Name), searchLower) ||
				strings.Contains(strings.ToLower(t.Description), searchLower) {
				d.filtered = append(d.filtered, t)
			}
		}
	}

	d.SetItemCount(len(d.filtered))
	if d.selected >= len(d.filtered) {
		d.selected = 0
	}
}

// View renders the dialog
func (d *ThemeListDialog) View() string {
	// Dialog dimensions
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
		Foreground(d.theme.Background).
		Background(d.theme.Primary).
		Padding(0, 1).
		Bold(true)

	currentStyle := lipgloss.NewStyle().
		Foreground(d.theme.Success).
		Padding(0, 1)

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
		searchText = "Type to search..."
		searchStyle = searchStyle.Foreground(d.theme.TextMuted)
	}
	content.WriteString(searchStyle.Render("/ " + searchText))
	content.WriteString("\n\n")

	// Theme list
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

	for i := startIdx; i < endIdx; i++ {
		theme := d.filtered[i]
		isSelected := i == d.selected
		isCurrent := theme.ID == d.currentTheme

		// Format: "1. Theme Name ●" or "   Theme Name"
		var line strings.Builder

		// Number hint
		if i < 8 {
			line.WriteString(fmt.Sprintf("%d. ", i+1))
		} else {
			line.WriteString("   ")
		}

		line.WriteString(theme.Name)

		// Add indicator for current theme
		if isCurrent {
			line.WriteString(" ●")
		}

		// Apply style
		if isSelected {
			content.WriteString(selectedStyle.Render(line.String()))
		} else if isCurrent {
			content.WriteString(currentStyle.Render(line.String()))
		} else {
			content.WriteString(itemStyle.Render(line.String()))
		}
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
	for i := linesRendered; i < listHeight; i++ {
		content.WriteString("\n")
	}

	// Hints
	content.WriteString("\n")
	hintStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted).
		Padding(0, 1)
	content.WriteString(hintStyle.Render("Enter: Select  1-8: Quick select  Esc: Cancel"))

	// Dialog border
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.theme.Border).
		Padding(1, 2).
		Width(width)

	return dialogStyle.Render(content.String())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
