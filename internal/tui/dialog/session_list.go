// Package dialog provides modal dialog components for the TUI.
// This file implements the session list dialog.
package dialog

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// SessionListDialog shows a searchable list of sessions
type SessionListDialog struct {
	BaseDialog

	// Sessions to display
	sessions []types.Session

	// Filtered sessions based on search
	filtered []types.Session

	// State reference for selection
	state *types.AppState
}

// NewSessionListDialog creates a new session list dialog
func NewSessionListDialog(theme types.Theme, state *types.AppState) *SessionListDialog {
	d := &SessionListDialog{
		BaseDialog: BaseDialog{
			theme:    theme,
			title:    "Sessions",
			action:   "Ctrl+N New",
			search:   "",
			selected: 0,
		},
		state: state,
	}

	d.sessions = state.Sync.Sessions
	d.filtered = d.sessions
	d.SetItemCount(len(d.filtered))

	return d
}

// ID returns the dialog type
func (d *SessionListDialog) ID() types.DialogType {
	return types.DialogSessionList
}

// Init initializes the dialog
func (d *SessionListDialog) Init() tea.Cmd {
	return nil
}

// Update handles events
func (d *SessionListDialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		handled, cmd := d.HandleKey(msg)
		if handled {
			// Re-filter after search change
			d.filterSessions()
			return d, cmd
		}

		if msg.String() == "enter" && len(d.filtered) > 0 {
			// Select the session
			selected := d.filtered[d.Selected()]
			return d, tea.Batch(
				CloseCmd(),
				func() tea.Msg {
					return SelectMsg{
						Type:  types.DialogSessionList,
						Index: d.Selected(),
						Data:  selected,
					}
				},
			)
		}

		// Ctrl+N for new session
		if msg.String() == "ctrl+n" {
			return d, tea.Batch(
				CloseCmd(),
				func() tea.Msg {
					return SelectMsg{
						Type: types.DialogSessionList,
						Data: "new",
					}
				},
			)
		}

	case tea.WindowSizeMsg:
		d.SetDimensions(msg.Width, msg.Height)
	}

	return d, nil
}

// filterSessions filters the session list based on search text
func (d *SessionListDialog) filterSessions() {
	if d.search == "" {
		d.filtered = d.sessions
	} else {
		d.filtered = []types.Session{}
		searchLower := strings.ToLower(d.search)

		for _, s := range d.sessions {
			// Search in title and directory
			if strings.Contains(strings.ToLower(s.Title), searchLower) ||
				strings.Contains(strings.ToLower(s.Directory), searchLower) ||
				strings.Contains(strings.ToLower(s.ID), searchLower) {
				d.filtered = append(d.filtered, s)
			}
		}
	}

	d.SetItemCount(len(d.filtered))
	if d.selected >= len(d.filtered) {
		d.selected = 0
	}
}

// View renders the dialog
func (d *SessionListDialog) View() string {
	// Dialog dimensions
	width := min(d.width-4, 60)
	height := min(d.height-4, 20)

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

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.theme.Border).
		Padding(1, 1)

	// Build content
	var lines []string

	// Title line
	titleLine := titleStyle.Render(d.title)
	if d.action != "" {
		actionStyle := lipgloss.NewStyle().
			Foreground(d.theme.TextMuted).
			Padding(0, 1)
		titleLine = lipgloss.JoinHorizontal(lipgloss.Top,
			titleLine,
			lipgloss.NewStyle().Width(width-lipgloss.Width(titleLine)-lipgloss.Width(actionStyle.Render(d.action))).Render(""),
			actionStyle.Render(d.action),
		)
	}
	lines = append(lines, titleLine)

	// Search line
	searchText := "🔍 " + d.search
	if d.search == "" {
		searchText = "🔍 Search sessions..."
	}
	lines = append(lines, searchStyle.Render(searchText))
	lines = append(lines, "") // Spacer

	// Session list
	visibleHeight := height - 6 // Account for title, search, spacer, footer
	startIdx := d.selected
	if startIdx > len(d.filtered)-visibleHeight {
		startIdx = max(0, len(d.filtered)-visibleHeight)
	}

	for i := startIdx; i < len(d.filtered) && i < startIdx+visibleHeight; i++ {
		s := d.filtered[i]

		// Format session info
		timeStr := formatTime(s.CreatedAt)
		title := s.Title
		if title == "" {
			title = "Untitled"
		}

		// Truncate title if too long
		maxTitleLen := width - 20
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}

		line := fmt.Sprintf("%s  %s", title, mutedStyle.Render(timeStr))

		if i == d.selected {
			lines = append(lines, selectedStyle.Render(line))
		} else {
			lines = append(lines, itemStyle.Render(line))
		}

		// Show directory on second line for selected
		if i == d.selected && s.Directory != "" {
			dirText := mutedStyle.Render("   " + s.Directory)
			lines = append(lines, dirText)
		}
	}

	// Empty state
	if len(d.filtered) == 0 {
		emptyText := "No sessions found"
		if d.search != "" {
			emptyText = fmt.Sprintf("No sessions matching \"%s\"", d.search)
		}
		lines = append(lines, mutedStyle.Render(emptyText))
	}

	// Footer hints
	hints := "↑/↓ Navigate  Enter Select  Esc Close"
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render(hints))

	// Wrap in border
	content := strings.Join(lines, "\n")
	return borderStyle.Width(width).Height(height).Render(content)
}

// formatTime formats a timestamp for display
func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "Just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}

	return t.Format("Jan 2")
}
