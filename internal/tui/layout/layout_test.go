// Package layout provides tests for layout components.
package layout

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// defaultTestTheme creates a default theme for testing
func defaultTestTheme() types.Theme {
	return types.Theme{
		Name:         "Default",
		ID:           "default",
		IsDark:       true,
		Primary:      lipgloss.Color("#7C3AED"),
		Secondary:    lipgloss.Color("#2563EB"),
		Accent:       lipgloss.Color("#10B981"),
		Error:        lipgloss.Color("#EF4444"),
		Warning:      lipgloss.Color("#F59E0B"),
		Success:      lipgloss.Color("#10B981"),
		Info:         lipgloss.Color("#3B82F6"),
		Text:         lipgloss.Color("#E5E7EB"),
		TextMuted:    lipgloss.Color("#9CA3AF"),
		TextBold:     lipgloss.Color("#FFFFFF"),
		Background:   lipgloss.Color("#1F2937"),
		PanelBg:      lipgloss.Color("#374151"),
		ElementBg:    lipgloss.Color("#4B5563"),
		MenuBg:       lipgloss.Color("#111827"),
		Border:       lipgloss.Color("#374151"),
		BorderActive: lipgloss.Color("#7C3AED"),
		Spinner:      lipgloss.Color("#7C3AED"),
	}
}

// TestNewSidebar tests sidebar creation
func TestNewSidebar(t *testing.T) {
	config := DefaultSidebarConfig()
	theme := defaultTestTheme()

	sidebar := NewSidebar(config, theme)
	if sidebar == nil {
		t.Error("NewSidebar should return non-nil sidebar")
	}
	if sidebar.Width() != config.Width {
		t.Errorf("Sidebar width should be %d, got %d", config.Width, sidebar.Width())
	}
}

// TestSidebarRender tests sidebar rendering
func TestSidebarRender(t *testing.T) {
	config := DefaultSidebarConfig()
	theme := defaultTestTheme()
	sidebar := NewSidebar(config, theme)

	// Render with no state
	output := sidebar.Render(nil)
	if output == "" {
		t.Error("Sidebar should render non-empty output")
	}

	// Should contain version info
	if !strings.Contains(output, "MagiCode") {
		t.Error("Sidebar should contain version info")
	}

	// Should contain session label
	if !strings.Contains(output, "Session") {
		t.Error("Sidebar should contain Session label")
	}

	// Should contain workspace label
	if !strings.Contains(output, "Workspace") {
		t.Error("Sidebar should contain Workspace label")
	}
}

// TestSidebarRenderWithState tests sidebar rendering with state
func TestSidebarRenderWithState(t *testing.T) {
	config := DefaultSidebarConfig()
	theme := defaultTestTheme()
	sidebar := NewSidebar(config, theme)

	state := types.NewAppState()
	state.Sync.Sessions = []types.Session{
		{
			ID:        "test-session-123",
			Title:     "Test Session",
			Directory: "/home/user/project",
			CreatedAt: time.Now(),
			Active:    true,
		},
	}
	state.Sync.ProjectID = "project-1"
	state.Sync.ProjectName = "MyProject"
	state.Sync.LSPServers = []types.LSPServer{
		{Name: "gopls", Language: "go", Status: "running"},
	}
	state.Sync.MCPServers = []types.MCPServer{
		{Name: "mcp-1", Status: "running"},
	}

	output := sidebar.Render(&state)
	if output == "" {
		t.Error("Sidebar should render non-empty output with state")
	}

	// Should contain session title
	if !strings.Contains(output, "Test Session") {
		t.Error("Sidebar should contain session title")
	}

	// Should contain LSP count
	if !strings.Contains(output, "1 LSP") {
		t.Error("Sidebar should show LSP count")
	}

	// Should contain MCP count
	if !strings.Contains(output, "1 MCP") {
		t.Error("Sidebar should show MCP count")
	}
}

// TestSidebarSetDimensions tests setting dimensions
func TestSidebarSetDimensions(t *testing.T) {
	config := DefaultSidebarConfig()
	theme := defaultTestTheme()
	sidebar := NewSidebar(config, theme)

	sidebar.SetDimensions(50, 30)

	if sidebar.Width() != 50 {
		t.Errorf("Width should be 50, got %d", sidebar.Width())
	}
	if sidebar.Height() != 30 {
		t.Errorf("Height should be 30, got %d", sidebar.Height())
	}
}

// TestNewFooter tests footer creation
func TestNewFooter(t *testing.T) {
	config := DefaultFooterConfig()
	theme := defaultTestTheme()

	footer := NewFooter(config, theme)
	if footer == nil {
		t.Error("NewFooter should return non-nil footer")
	}
}

// TestFooterRender tests footer rendering
func TestFooterRender(t *testing.T) {
	config := DefaultFooterConfig()
	theme := defaultTestTheme()
	footer := NewFooter(config, theme)
	footer.SetWidth(80)

	output := footer.Render(nil)
	if output == "" {
		t.Error("Footer should render non-empty output")
	}

	// Should contain status hint
	if config.ShowHint && !strings.Contains(output, "/status") {
		t.Error("Footer should contain /status hint")
	}
}

// TestFooterRenderWithState tests footer rendering with state
func TestFooterRenderWithState(t *testing.T) {
	config := DefaultFooterConfig()
	theme := defaultTestTheme()
	footer := NewFooter(config, theme)
	footer.SetWidth(100)

	state := types.NewAppState()
	state.Sync.Sessions = []types.Session{
		{
			ID:        "test-session",
			Title:     "Test",
			Directory: "/home/user/project",
			Active:    true,
		},
	}
	state.Sync.LSPServers = []types.LSPServer{
		{Name: "gopls", Status: "running"},
		{Name: "ts-ls", Status: "running"},
	}
	state.Sync.MCPServers = []types.MCPServer{
		{Name: "mcp-1", Status: "running"},
	}

	output := footer.Render(&state)
	if output == "" {
		t.Error("Footer should render with state")
	}

	// Should show LSP count
	if !strings.Contains(output, "2 LSP") {
		t.Error("Footer should show 2 LSP")
	}

	// Should show MCP count
	if !strings.Contains(output, "1 MCP") {
		t.Error("Footer should show 1 MCP")
	}
}

// TestNewPrompt tests prompt creation
func TestNewPrompt(t *testing.T) {
	config := DefaultPromptConfig()
	theme := defaultTestTheme()

	prompt := NewPrompt(config, theme)
	if prompt == nil {
		t.Error("NewPrompt should return non-nil prompt")
	}
}

// TestPromptSetValue tests setting prompt value
func TestNewPromptWithMultiline(t *testing.T) {
	// Test multiline config
	config := PromptConfig{
		Placeholder:   "Type here...",
		CharLimit:     1000,
		Multiline:     true,
		ShowModelInfo: true,
	}
	theme := defaultTestTheme()

	prompt := NewPrompt(config, theme)
	if prompt == nil {
		t.Error("NewPrompt should return non-nil prompt with multiline")
	}

	// Test set value
	prompt.SetValue("Hello world")
	if prompt.GetValue() != "Hello world" {
		t.Errorf("GetValue should return 'Hello world', got '%s'", prompt.GetValue())
	}

	// Test clear
	prompt.Clear()
	if prompt.GetValue() != "" {
		t.Error("GetValue should return empty string after clear")
	}
}

// TestPromptSetModelInfo tests setting model info
func TestPromptSetModelInfo(t *testing.T) {
	config := DefaultPromptConfig()
	theme := defaultTestTheme()
	prompt := NewPrompt(config, theme)
	prompt.SetDimensions(80, 5)

	prompt.SetModelInfo("build", "claude-3", "extended")

	output := prompt.Render()
	if output == "" {
		t.Error("Prompt should render with model info")
	}

	// Should contain agent
	if !strings.Contains(output, "build") {
		t.Error("Prompt should show agent")
	}

	// Should contain model
	if !strings.Contains(output, "claude-3") {
		t.Error("Prompt should show model")
	}
}

// TestPromptProcessing tests processing state
func TestPromptProcessing(t *testing.T) {
	config := DefaultPromptConfig()
	theme := defaultTestTheme()
	prompt := NewPrompt(config, theme)
	prompt.SetDimensions(80, 5)

	// Set processing state
	prompt.SetProcessing(true, "⠋")

	if !prompt.IsProcessing() {
		t.Error("Prompt should be processing")
	}

	output := prompt.Render()
	// When processing, shows "Waiting for response..."
	if !strings.Contains(output, "Waiting") {
		t.Error("Prompt should show waiting indicator when processing")
	}

	// Clear processing
	prompt.SetProcessing(false, "")

	if prompt.IsProcessing() {
		t.Error("Prompt should not be processing after clear")
	}
}

// TestMobileSidebar tests mobile sidebar
func TestMobileSidebar(t *testing.T) {
	theme := defaultTestTheme()
	mobile := NewMobileSidebar(theme)

	if mobile.IsVisible() {
		t.Error("Mobile sidebar should start hidden")
	}

	// Test toggle
	mobile.Toggle()
	if !mobile.IsVisible() {
		t.Error("Mobile sidebar should be visible after toggle")
	}

	// Test hide
	mobile.Hide()
	if mobile.IsVisible() {
		t.Error("Mobile sidebar should be hidden after hide")
	}

	// Test show
	mobile.Show()
	if !mobile.IsVisible() {
		t.Error("Mobile sidebar should be visible after show")
	}

	// Test render
	output := mobile.View()
	if output == "" {
		t.Error("Mobile sidebar should render when visible")
	}
}

// TestKeybindHintBar tests keybind hint bar
func TestKeybindHintBar(t *testing.T) {
	theme := defaultTestTheme()
	hints := NewKeybindHintBar(theme)
	hints.SetWidth(80)

	output := hints.View()
	if output == "" {
		t.Error("KeybindHintBar should render")
	}

	// Should contain Enter
	if !strings.Contains(output, "Enter") {
		t.Error("KeybindHintBar should show Enter key")
	}

	// Test custom hints
	hints.SetHints([]KeybindHint{
		{Key: "Esc", Text: "cancel"},
		{Key: "Tab", Text: "switch"},
	})

	output = hints.View()
	if !strings.Contains(output, "Esc") {
		t.Error("KeybindHintBar should show custom Esc key")
	}
}

// TestPromptWithAttachments tests prompt attachments
func TestPromptWithAttachments(t *testing.T) {
	config := DefaultPromptConfig()
	theme := defaultTestTheme()
	prompt := NewPromptWithAttachments(config, theme)
	prompt.SetDimensions(80, 5)

	// Add attachment
	prompt.AddAttachment("/path/to/file.go")
	attachments := prompt.GetAttachments()
	if len(attachments) != 1 {
		t.Errorf("Should have 1 attachment, got %d", len(attachments))
	}

	// Render should show attachment
	output := prompt.Render()
	if !strings.Contains(output, "file.go") {
		t.Error("Prompt should show file attachment")
	}

	// Remove attachment
	prompt.RemoveAttachment(0)
	if len(prompt.GetAttachments()) != 0 {
		t.Error("Should have 0 attachments after remove")
	}

	// Clear attachments
	prompt.AddAttachment("file1.go")
	prompt.AddAttachment("file2.go")
	prompt.ClearAttachments()
	if len(prompt.GetAttachments()) != 0 {
		t.Error("Should have 0 attachments after clear")
	}
}

// TestStatusBar tests status bar
func TestStatusBar(t *testing.T) {
	config := DefaultFooterConfig()
	theme := defaultTestTheme()
	statusBar := NewStatusBar(config, theme)
	statusBar.SetWidth(80)

	// Test normal state
	state := types.NewAppState()
	state.StatusText = "Ready"

	output := statusBar.Render(&state)
	if !strings.Contains(output, "Ready") {
		t.Error("StatusBar should show status text")
	}

	// Test processing state
	statusBar.SetProcessing(true, "⠋")

	output = statusBar.Render(&state)
	if !strings.Contains(output, "Processing") {
		t.Error("StatusBar should show processing indicator")
	}
}
