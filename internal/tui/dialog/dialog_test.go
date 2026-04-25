// Package dialog provides tests for dialog components.
package dialog

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// testTheme creates a test theme
func testTheme() types.Theme {
	return types.Theme{
		Name:       "Test",
		ID:         "test",
		IsDark:     true,
		Primary:    lipgloss.Color("#7C3AED"),
		Secondary:  lipgloss.Color("#2563EB"),
		Text:       lipgloss.Color("#E5E7EB"),
		TextMuted:  lipgloss.Color("#9CA3AF"),
		Background: lipgloss.Color("#1F2937"),
		PanelBg:    lipgloss.Color("#374151"),
		Border:     lipgloss.Color("#374151"),
		Success:    lipgloss.Color("#10B981"),
		Error:      lipgloss.Color("#EF4444"),
	}
}

// testAppState creates a test app state with sessions
func testAppState() types.AppState {
	state := types.NewAppState()
	state.Sync.Sessions = []types.Session{
		{
			ID:        "session-1",
			Title:     "Test Session 1",
			Directory: "/home/user/project1",
			CreatedAt: time.Now().Add(-1 * time.Hour),
			Active:    true,
		},
		{
			ID:        "session-2",
			Title:     "Test Session 2",
			Directory: "/home/user/project2",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			Active:    false,
		},
	}
	state.Sync.Providers = []types.Provider{
		{
			ID:        "provider-1",
			Name:      "Test Provider",
			Connected: true,
			Models: map[string]types.Model{
				"model-1": {
					ID:         "model-1",
					Name:       "Test Model 1",
					ProviderID: "provider-1",
				},
				"model-2": {
					ID:         "model-2",
					Name:       "Test Model 2",
					ProviderID: "provider-1",
				},
			},
		},
	}
	state.Local.CurrentModel = types.ModelKey{
		ProviderID: "provider-1",
		ModelID:    "model-1",
	}
	return state
}

// TestSessionListDialog tests the session list dialog
func TestSessionListDialog(t *testing.T) {
	theme := testTheme()
	state := testAppState()

	d := NewSessionListDialog(theme, &state)

	// Check ID
	if d.ID() != types.DialogSessionList {
		t.Errorf("Expected ID %s, got %s", types.DialogSessionList, d.ID())
	}

	// Check initial state
	if d.Search() != "" {
		t.Errorf("Expected empty search, got %s", d.Search())
	}

	if d.Selected() != 0 {
		t.Errorf("Expected selected=0, got %d", d.Selected())
	}

	// Test dimensions
	d.SetDimensions(80, 24)
	view := d.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Check title is in view
	if !containsSubstring(view, "Sessions") {
		t.Error("Expected 'Sessions' in view")
	}
}

// TestSessionListDialogNavigation tests navigation in session list
func TestSessionListDialogNavigation(t *testing.T) {
	theme := testTheme()
	state := testAppState()

	d := NewSessionListDialog(theme, &state)
	d.SetDimensions(80, 24)

	// Test navigation down
	d.NavigateDown()
	if d.Selected() != 1 {
		t.Errorf("Expected selected=1 after NavigateDown, got %d", d.Selected())
	}

	// Test navigation up
	d.NavigateDown() // Now at 2 (but only 2 items, so should be 1)
	d.NavigateUp()
	if d.Selected() != 0 {
		t.Errorf("Expected selected=0 after NavigateUp, got %d", d.Selected())
	}

	// Test boundary - can't go above 0
	d.NavigateUp()
	if d.Selected() != 0 {
		t.Errorf("Expected selected=0 at boundary, got %d", d.Selected())
	}

	// Test boundary - can't go below count
	d.NavigateDown()
	d.NavigateDown()
	d.NavigateDown()
	if d.Selected() != 1 {
		t.Errorf("Expected selected=1 at boundary, got %d", d.Selected())
	}
}

// TestSessionListDialogSearch tests search functionality
func TestSessionListDialogSearch(t *testing.T) {
	theme := testTheme()
	state := testAppState()

	d := NewSessionListDialog(theme, &state)
	d.SetDimensions(80, 24)

	// Set search
	d.SetSearch("Session 2")
	d.filterSessions()

	if len(d.filtered) != 1 {
		t.Errorf("Expected 1 filtered session, got %d", len(d.filtered))
	}

	if d.filtered[0].ID != "session-2" {
		t.Errorf("Expected session-2, got %s", d.filtered[0].ID)
	}

	// Clear search
	d.ClearSearch()
	d.filterSessions()

	if len(d.filtered) != 2 {
		t.Errorf("Expected 2 sessions after clear, got %d", len(d.filtered))
	}
}

// TestSessionListDialogUpdate tests Update method
func TestSessionListDialogUpdate(t *testing.T) {
	theme := testTheme()
	state := testAppState()

	d := NewSessionListDialog(theme, &state)
	d.SetDimensions(80, 24)

	// Test key handling - up arrow
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	newD, cmd := d.Update(upMsg)
	if cmd != nil {
		t.Error("Expected nil cmd for up arrow")
	}
	// Since we start at 0, up shouldn't change selection
	_ = newD

	// Test key handling - down arrow
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	newD, cmd = d.Update(downMsg)
	if cmd != nil {
		t.Error("Expected nil cmd for down arrow")
	}
	if newD.(*SessionListDialog).Selected() != 1 {
		t.Error("Expected selected=1 after down arrow")
	}

	// Test escape key
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	newD, cmd = d.Update(escMsg)
	if cmd == nil {
		t.Error("Expected close cmd for escape")
	}
}

// TestModelListDialog tests the model list dialog
func TestModelListDialog(t *testing.T) {
	theme := testTheme()
	state := testAppState()

	d := NewModelListDialog(theme, &state)

	// Check ID
	if d.ID() != types.DialogModelList {
		t.Errorf("Expected ID %s, got %s", types.DialogModelList, d.ID())
	}

	// Test dimensions
	d.SetDimensions(80, 24)
	view := d.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Check title is in view
	if !containsSubstring(view, "Select Model") {
		t.Error("Expected 'Select Model' in view")
	}
}

// TestModelListDialogNavigation tests navigation in model list
func TestModelListDialogNavigation(t *testing.T) {
	theme := testTheme()
	state := testAppState()

	d := NewModelListDialog(theme, &state)
	d.SetDimensions(80, 24)

	// Test navigation - should have models
	if len(d.models) < 2 {
		t.Errorf("Expected at least 2 models, got %d", len(d.models))
	}

	// Test navigation down
	d.NavigateDown()
	// Selected should increase
}

// TestHelpDialog tests the help dialog
func TestHelpDialog(t *testing.T) {
	theme := testTheme()

	d := NewHelpDialog(theme)

	// Check ID
	if d.ID() != types.DialogHelp {
		t.Errorf("Expected ID %s, got %s", types.DialogHelp, d.ID())
	}

	// Test dimensions
	d.SetDimensions(80, 24)
	view := d.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}

	// Check title is in view
	if !containsSubstring(view, "Keyboard Shortcuts") {
		t.Error("Expected 'Keyboard Shortcuts' in view")
	}
}

// TestHelpDialogScroll tests scrolling in help dialog
func TestHelpDialogScroll(t *testing.T) {
	theme := testTheme()

	d := NewHelpDialog(theme)
	d.SetDimensions(80, 24)

	// Initial scroll offset
	if d.scrollOffset != 0 {
		t.Errorf("Expected scrollOffset=0, got %d", d.scrollOffset)
	}

	// Scroll down
	d.NavigateDown()
	if d.scrollOffset <= 0 {
		t.Error("Expected scrollOffset > 0 after NavigateDown")
	}

	// Scroll up
	d.NavigateUp()
	if d.scrollOffset != 0 {
		t.Errorf("Expected scrollOffset=0 after NavigateUp to top, got %d", d.scrollOffset)
	}
}

// TestBaseDialog tests base dialog functionality
func TestBaseDialog(t *testing.T) {
	theme := testTheme()

	bd := BaseDialog{
		theme:    theme,
		title:    "Test",
		selected: 0,
		items:    5,
	}

	// Test navigation
	bd.NavigateDown()
	if bd.Selected() != 1 {
		t.Errorf("Expected selected=1, got %d", bd.Selected())
	}

	// Test page navigation
	bd.NavigatePageDown(2)
	if bd.Selected() != 3 {
		t.Errorf("Expected selected=3 after page down, got %d", bd.Selected())
	}

	// Test boundary
	bd.NavigatePageDown(10) // Should hit boundary at 4
	if bd.Selected() != 4 {
		t.Errorf("Expected selected=4 at boundary, got %d", bd.Selected())
	}

	// Test search
	bd.SetSearch("test")
	if bd.Search() != "test" {
		t.Errorf("Expected search='test', got %s", bd.Search())
	}

	bd.ClearSearch()
	if bd.Search() != "" {
		t.Errorf("Expected empty search after clear, got %s", bd.Search())
	}
}

// TestCloseCmd tests the close command
func TestCloseCmd(t *testing.T) {
	cmd := CloseCmd()
	if cmd == nil {
		t.Error("Expected non-nil close command")
	}

	// Execute the command
	msg := cmd()
	if _, ok := msg.(CloseMsg); !ok {
		t.Errorf("Expected CloseMsg, got %T", msg)
	}
}

// containsSubstring checks if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

// findSubstring helper
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
