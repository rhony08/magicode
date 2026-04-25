// Package layout provides layout components for the TUI.
// Footer implements the status bar matching OpenCode's design.
package layout

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// FooterConfig holds configuration for the footer
type FooterConfig struct {
	ShowDirectory bool // Whether to show current directory
	ShowLSPMCP    bool // Whether to show LSP/MCP count
	ShowHint      bool // Whether to show status hint
}

// DefaultFooterConfig returns the default footer configuration
func DefaultFooterConfig() FooterConfig {
	return FooterConfig{
		ShowDirectory: true,
		ShowLSPMCP:    true,
		ShowHint:      true,
	}
}

// Footer is the footer status bar component
type Footer struct {
	config FooterConfig
	theme  types.Theme
	styles FooterStyles
	width  int
}

// FooterStyles holds lipgloss styles for footer
type FooterStyles struct {
	Container  lipgloss.Style
	Left       lipgloss.Style
	Right      lipgloss.Style
	Text       lipgloss.Style
	TextMuted  lipgloss.Style
	Indicator  lipgloss.Style
	StatusHint lipgloss.Style
	Separator  lipgloss.Style
}

// NewFooter creates a new footer component
func NewFooter(config FooterConfig, theme types.Theme) *Footer {
	styles := FooterStyles{
		Container: lipgloss.NewStyle().
			Background(theme.Background).
			Foreground(theme.Text).
			Padding(0, 1),
		Left: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1),
		Right: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1),
		Text: lipgloss.NewStyle().
			Foreground(theme.Text),
		TextMuted: lipgloss.NewStyle().
			Foreground(theme.TextMuted),
		Indicator: lipgloss.NewStyle().
			Foreground(theme.Info),
		StatusHint: lipgloss.NewStyle().
			Foreground(theme.Primary),
		Separator: lipgloss.NewStyle().
			Foreground(theme.TextMuted),
	}

	return &Footer{
		config: config,
		theme:  theme,
		styles: styles,
	}
}

// Init initializes the footer (tea.Model interface)
func (f *Footer) Init() tea.Cmd {
	return nil
}

// Update handles events (tea.Model interface)
func (f *Footer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Footer doesn't handle any events directly
	return f, nil
}

// View renders the footer (tea.Model interface)
func (f *Footer) View() string {
	return f.Render(nil)
}

// Render renders the footer with given state
func (f *Footer) Render(state *types.AppState) string {
	if f.width == 0 {
		return ""
	}

	// Split footer into left and right sections
	leftWidth := f.width / 2
	rightWidth := f.width / 2

	// ===========================================
	// Left Section: Directory path
	// ===========================================
	var leftContent string
	if f.config.ShowDirectory {
		dir := ""
		if state != nil && len(state.Sync.Sessions) > 0 {
			dir = state.Sync.Sessions[0].Directory
		}
		if dir == "" {
			dir = "~" // Default placeholder
		}

		// Truncate directory path if needed
		if len(dir) > leftWidth-4 {
			parts := strings.Split(dir, "/")
			if len(parts) > 3 {
				dir = ".../" + strings.Join(parts[len(parts)-2:], "/")
			}
			if len(dir) > leftWidth-4 {
				dir = dir[:leftWidth-7] + "..."
			}
		}

		leftContent = f.styles.Left.Render(dir)
	}

	// ===========================================
	// Right Section: LSP/MCP count + status hint
	// ===========================================
	var rightParts []string

	// LSP/MCP indicators
	if f.config.ShowLSPMCP && state != nil {
		lspCount := len(state.Sync.LSPServers)
		mcpCount := len(state.Sync.MCPServers)
		if lspCount > 0 || mcpCount > 0 {
			lspText := fmt.Sprintf("%d LSP", lspCount)
			mcpText := fmt.Sprintf("%d MCP", mcpCount)
			rightParts = append(rightParts, f.styles.TextMuted.Render(lspText))
			rightParts = append(rightParts, f.styles.Separator.Render("·"))
			rightParts = append(rightParts, f.styles.TextMuted.Render(mcpText))
		}
	}

	// Separator
	if len(rightParts) > 0 && f.config.ShowHint {
		rightParts = append(rightParts, f.styles.Separator.Render("·"))
	}

	// Status hint
	if f.config.ShowHint {
		rightParts = append(rightParts, f.styles.StatusHint.Render("/status"))
	}

	rightContent := strings.Join(rightParts, " ")

	// ===========================================
	// Combine left and right
	// ===========================================
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Align(lipgloss.Left)

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Align(lipgloss.Right)

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(leftContent),
		rightStyle.Render(rightContent),
	)

	// Apply container style
	return f.styles.Container.
		Width(f.width).
		Render(row)
}

// SetWidth updates footer width
func (f *Footer) SetWidth(width int) {
	f.width = width
}

// Width returns the footer width
func (f *Footer) Width() int {
	return f.width
}

// ===========================================
// Status Bar (Alternative footer with processing state)
// ===========================================

// StatusBar is a status bar that shows processing state
type StatusBar struct {
	Footer
	processing bool
	spinner    string
}

// NewStatusBar creates a status bar with processing state
func NewStatusBar(config FooterConfig, theme types.Theme) *StatusBar {
	return &StatusBar{
		Footer: *NewFooter(config, theme),
	}
}

// SetProcessing sets the processing state
func (s *StatusBar) SetProcessing(processing bool, spinner string) {
	s.processing = processing
	s.spinner = spinner
}

// Render renders the status bar with processing state
func (s *StatusBar) Render(state *types.AppState) string {
	if s.width == 0 {
		return ""
	}

	// Show processing spinner if active
	if s.processing {
		status := s.spinner + " Processing..."
		return s.styles.Container.
			Width(s.width).
			Foreground(s.theme.Warning).
			Render(status)
	}

	// Show status text from state
	if state != nil && state.StatusText != "" {
		return s.styles.Container.
			Width(s.width).
			Render(state.StatusText)
	}

	// Default: show footer content
	return s.Footer.Render(state)
}

// ===========================================
// Keybind Hint Bar
// ===========================================

// KeybindHintBar shows keybinding hints at the bottom
type KeybindHintBar struct {
	theme  types.Theme
	styles KeybindHintStyles
	width  int
	hints  []KeybindHint
}

// KeybindHint represents a keybinding hint
type KeybindHint struct {
	Key  string
	Text string
}

// KeybindHintStyles holds styles for keybind hints
type KeybindHintStyles struct {
	Container lipgloss.Style
	Key       lipgloss.Style
	Text      lipgloss.Style
	Separator lipgloss.Style
}

// NewKeybindHintBar creates a keybind hint bar
func NewKeybindHintBar(theme types.Theme) *KeybindHintBar {
	styles := KeybindHintStyles{
		Container: lipgloss.NewStyle().
			Background(theme.Background).
			Foreground(theme.TextMuted).
			Padding(0, 1),
		Key: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),
		Text: lipgloss.NewStyle().
			Foreground(theme.TextMuted),
		Separator: lipgloss.NewStyle().
			Foreground(theme.Border),
	}

	// Default hints for prompt area
	defaultHints := []KeybindHint{
		{Key: "Enter", Text: "submit"},
		{Key: "Ctrl+P", Text: "palette"},
		{Key: "Tab", Text: "cycle"},
	}

	return &KeybindHintBar{
		theme:  theme,
		styles: styles,
		hints:  defaultHints,
	}
}

// SetHints sets the keybinding hints
func (k *KeybindHintBar) SetHints(hints []KeybindHint) {
	k.hints = hints
}

// SetWidth updates width
func (k *KeybindHintBar) SetWidth(width int) {
	k.width = width
}

// View renders the keybind hint bar
func (k *KeybindHintBar) View() string {
	if k.width == 0 || len(k.hints) == 0 {
		return ""
	}

	var parts []string
	for _, hint := range k.hints {
		part := k.styles.Key.Render(hint.Key) +
			k.styles.Text.Render(" "+hint.Text)
		parts = append(parts, part)
	}

	content := strings.Join(parts, k.styles.Separator.Render(" · "))

	return k.styles.Container.
		Width(k.width).
		Align(lipgloss.Right).
		Render(content)
}
