// Package layout provides layout components for the TUI.
package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// AutocompleteOption represents an autocomplete suggestion
type AutocompleteOption struct {
	Value       string
	Display     string
	Description string
	Icon        string
}

// Autocomplete provides tab completion for the prompt
type Autocomplete struct {
	visible  bool
	options  []AutocompleteOption
	selected int
	trigger  string // "@" or "/"
	query    string
	theme    types.Theme
}

// NewAutocomplete creates a new autocomplete component
func NewAutocomplete(theme types.Theme) *Autocomplete {
	return &Autocomplete{
		visible:  false,
		options:  []AutocompleteOption{},
		selected: 0,
		theme:    theme,
	}
}

// Show shows autocomplete with options
func (a *Autocomplete) Show(options []AutocompleteOption, trigger string, query string) {
	a.visible = true
	a.options = options
	a.selected = 0
	a.trigger = trigger
	a.query = query
}

// Hide hides autocomplete
func (a *Autocomplete) Hide() {
	a.visible = false
}

// IsVisible returns true if autocomplete is visible
func (a *Autocomplete) IsVisible() bool {
	return a.visible
}

// Next selects the next option
func (a *Autocomplete) Next() {
	if len(a.options) > 0 {
		a.selected = (a.selected + 1) % len(a.options)
	}
}

// Previous selects the previous option
func (a *Autocomplete) Previous() {
	if len(a.options) > 0 {
		a.selected = (a.selected - 1 + len(a.options)) % len(a.options)
	}
}

// Select returns the currently selected option
func (a *Autocomplete) Select() (AutocompleteOption, bool) {
	if !a.visible || len(a.options) == 0 {
		return AutocompleteOption{}, false
	}
	return a.options[a.selected], true
}

// GetSelectedIndex returns the currently selected index
func (a *Autocomplete) GetSelectedIndex() int {
	return a.selected
}

// SetOptions updates the autocomplete options
func (a *Autocomplete) SetOptions(options []AutocompleteOption) {
	a.options = options
	if a.selected >= len(options) {
		a.selected = 0
	}
}

// Render renders the autocomplete dropdown
func (a *Autocomplete) Render(width int) string {
	if !a.visible || len(a.options) == 0 {
		return ""
	}

	var lines []string
	maxVisible := 8
	if len(a.options) < maxVisible {
		maxVisible = len(a.options)
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(a.theme.Primary).
		Bold(true)
	lines = append(lines, titleStyle.Render("  Suggestions"))

	// Options
	for i := 0; i < maxVisible; i++ {
		option := a.options[i]

		// Determine styles based on selection
		var prefix, display, description string
		if i == a.selected {
			prefix = "> "
			display = lipgloss.NewStyle().
				Foreground(a.theme.Background).
				Background(a.theme.Primary).
				Bold(true).
				Render(option.Display)
			if option.Description != "" {
				description = lipgloss.NewStyle().
					Foreground(a.theme.Background).
					Background(a.theme.Primary).
					Render(" " + option.Description)
			}
		} else {
			prefix = "  "
			display = lipgloss.NewStyle().
				Foreground(a.theme.Text).
				Render(option.Display)
			if option.Description != "" {
				description = lipgloss.NewStyle().
					Foreground(a.theme.TextMuted).
					Render(" " + option.Description)
			}
		}

		// Icon
		icon := option.Icon
		if icon == "" {
			icon = "  "
		} else {
			icon = icon + " "
		}

		line := prefix + icon + display + description
		lines = append(lines, line)
	}

	// Show count if there are more options
	if len(a.options) > maxVisible {
		remaining := len(a.options) - maxVisible
		countStyle := lipgloss.NewStyle().
			Foreground(a.theme.TextMuted).
			Italic(true)
		lines = append(lines, countStyle.Render("  ... and "+string(rune('0'+remaining))+" more"))
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(a.theme.TextMuted)
	lines = append(lines, helpStyle.Render("  Tab: select · ↑↓: navigate · Esc: close"))

	content := strings.Join(lines, "\n")

	// Border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Border).
		Background(a.theme.PanelBg).
		Width(width - 4)

	return borderStyle.Render(content)
}

// Height returns the height needed for the component
func (a *Autocomplete) Height() int {
	if !a.visible {
		return 0
	}
	height := 3 // Title + help + border
	maxVisible := 8
	if len(a.options) < maxVisible {
		maxVisible = len(a.options)
	}
	height += maxVisible
	if len(a.options) > maxVisible {
		height++ // "and X more" line
	}
	return height
}

// GetTrigger returns the trigger character ("@" or "/")
func (a *Autocomplete) GetTrigger() string {
	return a.trigger
}

// GetQuery returns the current query string
func (a *Autocomplete) GetQuery() string {
	return a.query
}
