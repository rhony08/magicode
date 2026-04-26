// Package component provides reusable UI components for the TUI.
package component

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// Test diff renderer with basic unified diff
func TestDiffRenderer_UnifiedDiff(t *testing.T) {
	theme := types.Theme{
		Added:     lipgloss.Color("#22C55E"),
		Removed:   lipgloss.Color("#EF4444"),
		AddedBg:   lipgloss.Color("#14532D"),
		RemovedBg: lipgloss.Color("#7F1D1D"),
		Info:      lipgloss.Color("#3B82F6"),
		TextMuted: lipgloss.Color("#9CA3AF"),
		Text:      lipgloss.Color("#E5E7EB"),
		Warning:   lipgloss.Color("#F59E0B"),
	}
	styles := types.ThemeStyles{}

	renderer := NewDiffRenderer(theme, styles)
	renderer.SetWidth(80)
	renderer.SetStyle(DiffStyleUnified)

	// Test basic diff
	diff := `@@ -1,5 +1,7 @@
 line1
-line2
+line2_modified
+new_line
 line3
 line4
 line5`

	result := renderer.Render(diff, "test.go")

	// Check that result contains the file path
	if !strings.Contains(result, "test.go") {
		t.Error("Expected result to contain file path")
	}

	// Check that result contains the hunk header
	if !strings.Contains(result, "@@") {
		t.Error("Expected result to contain hunk header")
	}

	// Result should be non-empty
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

// Test diff renderer with split style
func TestDiffRenderer_SplitDiff(t *testing.T) {
	theme := types.Theme{
		Added:     lipgloss.Color("#22C55E"),
		Removed:   lipgloss.Color("#EF4444"),
		AddedBg:   lipgloss.Color("#14532D"),
		RemovedBg: lipgloss.Color("#7F1D1D"),
		Info:      lipgloss.Color("#3B82F6"),
		TextMuted: lipgloss.Color("#9CA3AF"),
		Text:      lipgloss.Color("#E5E7EB"),
		Warning:   lipgloss.Color("#F59E0B"),
		Border:    lipgloss.Color("#374151"),
	}
	styles := types.ThemeStyles{}

	renderer := NewDiffRenderer(theme, styles)
	renderer.SetWidth(150) // Wide terminal for split view
	renderer.SetStyle(DiffStyleSplit)

	diff := `@@ -1,3 +1,4 @@
 old_line1
-old_line2
+new_line2
+new_line3
 old_line3`

	result := renderer.Render(diff, "split_test.go")

	// Result should be non-empty
	if result == "" {
		t.Error("Expected non-empty result for split diff")
	}

	// Split view should have separator
	if !strings.Contains(result, "│") {
		t.Error("Expected split diff to contain separator")
	}
}

// Test auto style selection
func TestDiffRenderer_AutoStyle(t *testing.T) {
	theme := types.Theme{
		Added:     lipgloss.Color("#22C55E"),
		Removed:   lipgloss.Color("#EF4444"),
		AddedBg:   lipgloss.Color("#14532D"),
		RemovedBg: lipgloss.Color("#7F1D1D"),
		Info:      lipgloss.Color("#3B82F6"),
		TextMuted: lipgloss.Color("#9CA3AF"),
		Text:      lipgloss.Color("#E5E7EB"),
		Warning:   lipgloss.Color("#F59E0B"),
		Border:    lipgloss.Color("#374151"),
	}
	styles := types.ThemeStyles{}

	// Test narrow terminal (should use unified)
	renderer := NewDiffRenderer(theme, styles)
	renderer.SetWidth(80) // Narrow
	renderer.SetStyle(DiffStyleAuto)

	diff := `@@ -1,2 +1,2 @@
 a
-b
+c`

	result := renderer.Render(diff, "auto_test.go")
	if !strings.Contains(result, "@@") {
		t.Error("Expected auto style (narrow) to use unified diff")
	}

	// Test wide terminal (should use split)
	renderer.SetWidth(150) // Wide
	result2 := renderer.Render(diff, "auto_test_wide.go")
	if !strings.Contains(result2, "│") {
		t.Error("Expected auto style (wide) to use split diff")
	}
}

// Test line wrapping in diff
func TestDiffRenderer_WrapLines(t *testing.T) {
	theme := types.Theme{
		Added:     lipgloss.Color("#22C55E"),
		Removed:   lipgloss.Color("#EF4444"),
		AddedBg:   lipgloss.Color("#14532D"),
		RemovedBg: lipgloss.Color("#7F1D1D"),
		Info:      lipgloss.Color("#3B82F6"),
		TextMuted: lipgloss.Color("#9CA3AF"),
		Text:      lipgloss.Color("#E5E7EB"),
		Warning:   lipgloss.Color("#F59E0B"),
	}
	styles := types.ThemeStyles{}

	renderer := NewDiffRenderer(theme, styles)
	renderer.SetWidth(40) // Narrow width to trigger wrapping
	renderer.SetWrapLines(true)

	// Very long line
	longLine := "this_is_a_very_long_line_that_should_be_wrapped_when_display_width_is_small"
	diff := `@@ -1,1 +1,1 @@
 ` + longLine

	result := renderer.Render(diff, "wrap_test.go")

	// Result should handle the long line (wrapped or truncated)
	if result == "" {
		t.Error("Expected non-empty result with wrapping")
	}
}

// Test simple diff rendering (backwards compatibility)
func TestDiffRenderer_SimpleDiff(t *testing.T) {
	theme := types.Theme{
		Added:     lipgloss.Color("#22C55E"),
		Removed:   lipgloss.Color("#EF4444"),
		AddedBg:   lipgloss.Color("#14532D"),
		RemovedBg: lipgloss.Color("#7F1D1D"),
		Info:      lipgloss.Color("#3B82F6"),
		TextMuted: lipgloss.Color("#9CA3AF"),
	}
	styles := types.ThemeStyles{}

	renderer := NewDiffRenderer(theme, styles)
	renderer.SetWidth(80)

	diff := `+added line
-removed line
 context line`

	result := renderer.RenderUnifiedSimple(diff)

	// Should contain all lines
	if !strings.Contains(result, "added line") {
		t.Error("Expected simple diff to contain added line")
	}
	if !strings.Contains(result, "removed line") {
		t.Error("Expected simple diff to contain removed line")
	}
}

// Test hunk parsing
func TestDiffRenderer_ParseHunk(t *testing.T) {
	theme := types.Theme{}
	styles := types.ThemeStyles{}

	renderer := NewDiffRenderer(theme, styles)

	// Parse hunk header
	hunks := renderer.parseDiff("@@ -1,5 +1,7 @@ function test()")

	if len(hunks) == 0 {
		t.Error("Expected at least one hunk")
		return
	}

	hunk := hunks[0]
	if hunk.OldStart != 1 {
		t.Errorf("Expected OldStart=1, got %d", hunk.OldStart)
	}
	if hunk.OldCount != 5 {
		t.Errorf("Expected OldCount=5, got %d", hunk.OldCount)
	}
	if hunk.NewStart != 1 {
		t.Errorf("Expected NewStart=1, got %d", hunk.NewStart)
	}
	if hunk.NewCount != 7 {
		t.Errorf("Expected NewCount=7, got %d", hunk.NewCount)
	}
}

// Test empty diff
func TestDiffRenderer_EmptyDiff(t *testing.T) {
	theme := types.Theme{}
	styles := types.ThemeStyles{}

	renderer := NewDiffRenderer(theme, styles)
	renderer.SetWidth(80)

	result := renderer.Render("", "empty.go")

	// Should still show file path
	if !strings.Contains(result, "empty.go") {
		t.Error("Expected empty diff to show file path")
	}
}
