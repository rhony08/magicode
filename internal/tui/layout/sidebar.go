// Package layout provides layout components for the TUI.
// Sidebar implements the session info panel matching OpenCode's design.
package layout

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// SidebarConfig holds configuration for the sidebar
type SidebarConfig struct {
	Width       int  // Sidebar width in columns (default: 344px ≈ 43 cols)
	ShowVersion bool // Whether to show version info
}

// DefaultSidebarConfig returns the default sidebar configuration
func DefaultSidebarConfig() SidebarConfig {
	return SidebarConfig{
		Width:       43, // 344px equivalent in terminal columns
		ShowVersion: true,
	}
}

// Sidebar is the sidebar panel component
type Sidebar struct {
	config SidebarConfig
	theme  types.Theme
	styles SidebarStyles
	width  int
	height int
}

// SidebarStyles holds lipgloss styles for sidebar
type SidebarStyles struct {
	Container lipgloss.Style
	Title     lipgloss.Style
	Text      lipgloss.Style
	TextMuted lipgloss.Style
	Status    lipgloss.Style
	Version   lipgloss.Style
	Border    lipgloss.Style
	Section   lipgloss.Style
	Indicator lipgloss.Style
}

// NewSidebar creates a new sidebar component
func NewSidebar(config SidebarConfig, theme types.Theme) *Sidebar {
	styles := SidebarStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Border).
			Background(theme.PanelBg).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),
		Text: lipgloss.NewStyle().
			Foreground(theme.Text),
		TextMuted: lipgloss.NewStyle().
			Foreground(theme.TextMuted),
		Status: lipgloss.NewStyle().
			Foreground(theme.Success),
		Version: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true),
		Border: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Border),
		Section: lipgloss.NewStyle().
			Foreground(theme.Text).
			Padding(0, 1),
		Indicator: lipgloss.NewStyle().
			Foreground(theme.Success),
	}

	return &Sidebar{
		config: config,
		theme:  theme,
		styles: styles,
		width:  config.Width,
	}
}

// Init initializes the sidebar (tea.Model interface)
func (s *Sidebar) Init() tea.Cmd {
	return nil
}

// Update handles events (tea.Model interface)
func (s *Sidebar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Sidebar doesn't handle any events directly
	return s, nil
}

// View renders the sidebar (tea.Model interface)
func (s *Sidebar) View() string {
	return s.Render(nil)
}

// Render renders the sidebar with given state
func (s *Sidebar) Render(state *types.AppState) string {
	var lines []string

	// Calculate content width (subtract padding and border)
	contentWidth := s.width - 6 // 2 padding on each side + 2 border

	// ===========================================
	// Section 1: Session Info
	// ===========================================
	lines = append(lines, s.styles.Title.Render("Session"))
	lines = append(lines, "")

	if state != nil && len(state.Sync.Sessions) > 0 {
		session := state.Sync.Sessions[0]
		// Truncate title if too long
		title := session.Title
		if len(title) > contentWidth {
			title = title[:contentWidth-3] + "..."
		}
		lines = append(lines, s.styles.Text.Render(title))

		// Session ID (shortened)
		idShort := session.ID
		if len(idShort) > 20 {
			idShort = idShort[:8] + "..." + idShort[len(idShort)-4:]
		}
		lines = append(lines, s.styles.TextMuted.Render(idShort))
	} else {
		lines = append(lines, s.styles.TextMuted.Render("No session"))
	}

	// ===========================================
	// Section 2: Workspace Status
	// ===========================================
	lines = append(lines, "")
	lines = append(lines, s.styles.Title.Render("Workspace"))
	lines = append(lines, "")

	// Status indicator (● for active)
	if state != nil && state.Sync.ProjectID != "" {
		lines = append(lines, s.styles.Indicator.Render("●")+
			s.styles.Text.Render(" Active"))
		lines = append(lines, s.styles.TextMuted.Render(state.Sync.ProjectName))
	} else {
		lines = append(lines, s.styles.TextMuted.Render("○ Idle"))
	}

	// ===========================================
	// Section 3: LSP/MCP Status
	// ===========================================
	if state != nil {
		lspCount := len(state.Sync.LSPServers)
		mcpCount := len(state.Sync.MCPServers)

		if lspCount > 0 || mcpCount > 0 {
			lines = append(lines, "")
			statusLine := fmt.Sprintf("%d LSP · %d MCP", lspCount, mcpCount)
			lines = append(lines, s.styles.TextMuted.Render(statusLine))
		}
	}

	// ===========================================
	// Section 4: Directory
	// ===========================================
	if state != nil {
		lines = append(lines, "")
		lines = append(lines, s.styles.Title.Render("Directory"))
		lines = append(lines, "")

		// Show shortened directory path
		dir := ""
		if len(state.Sync.Sessions) > 0 && state.Sync.Sessions[0].Directory != "" {
			dir = state.Sync.Sessions[0].Directory
		}

		if dir != "" {
			// Truncate long paths
			if len(dir) > contentWidth {
				// Show last part of path
				parts := strings.Split(dir, "/")
				if len(parts) > 3 {
					dir = ".../" + strings.Join(parts[len(parts)-2:], "/")
				}
				if len(dir) > contentWidth {
					dir = dir[:contentWidth-3] + "..."
				}
			}
			lines = append(lines, s.styles.TextMuted.Render(dir))
		}
	}

	// ===========================================
	// Section 5: Version (bottom)
	// ===========================================
	if s.config.ShowVersion {
		lines = append(lines, "")
		lines = append(lines, "")
		version := "MagiCode v1.0.0"
		lines = append(lines, s.styles.Version.Render(version))
	}

	content := strings.Join(lines, "\n")

	// Apply container style with dimensions
	return s.styles.Container.
		Width(s.width).
		Height(s.height).
		Render(content)
}

// SetDimensions updates sidebar dimensions
func (s *Sidebar) SetDimensions(width, height int) {
	s.width = width
	s.height = height
}

// Width returns the sidebar width
func (s *Sidebar) Width() int {
	return s.width
}

// Height returns the sidebar height
func (s *Sidebar) Height() int {
	return s.height
}

// ToggleWidth toggles sidebar visibility by returning 0 width
func (s *Sidebar) ToggleWidth(visible bool) {
	if visible {
		s.width = s.config.Width
	} else {
		s.width = 0
	}
}

// ===========================================
// Mobile Sidebar (Overlay for narrow terminals)
// ===========================================

// MobileSidebar is a mobile sidebar overlay for narrow terminals
type MobileSidebar struct {
	Sidebar
	visible bool
}

// NewMobileSidebar creates a mobile sidebar overlay
func NewMobileSidebar(theme types.Theme) *MobileSidebar {
	config := SidebarConfig{
		Width:       40, // Narrower for mobile
		ShowVersion: true,
	}
	return &MobileSidebar{
		Sidebar: *NewSidebar(config, theme),
		visible: false,
	}
}

// Show shows the mobile sidebar
func (m *MobileSidebar) Show() {
	m.visible = true
}

// Hide hides the mobile sidebar
func (m *MobileSidebar) Hide() {
	m.visible = false
}

// Toggle toggles mobile sidebar visibility
func (m *MobileSidebar) Toggle() {
	m.visible = !m.visible
}

// IsVisible returns whether mobile sidebar is visible
func (m *MobileSidebar) IsVisible() bool {
	return m.visible
}

// View renders the mobile sidebar overlay
func (m *MobileSidebar) View() string {
	if !m.visible {
		return ""
	}

	// Add overlay styling
	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary).
		Background(m.theme.Background).
		Padding(1, 2).
		Width(40).
		Height(20)

	var lines []string
	lines = append(lines, m.styles.Title.Render("Sidebar"))
	lines = append(lines, m.styles.TextMuted.Render("(Esc to close)"))
	lines = append(lines, "")
	lines = append(lines, m.styles.Text.Render("Session Info"))
	lines = append(lines, "")
	lines = append(lines, m.styles.Version.Render("MagiCode v1.0.0"))

	return overlayStyle.Render(strings.Join(lines, "\n"))
}

// Update handles mobile sidebar events
func (m *MobileSidebar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.Hide()
		}
	}
	return m, nil
}
