// Package component provides reusable UI components for the TUI.
// This file contains tests for the tool result renderer.
package component

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

func TestToolResultRenderer_RenderResult(t *testing.T) {
	theme := types.Theme{
		Primary:   lipgloss.Color("#7C3AED"),
		Secondary: lipgloss.Color("#2563EB"),
		Success:   lipgloss.Color("#10B981"),
		Error:     lipgloss.Color("#EF4444"),
		Warning:   lipgloss.Color("#F59E0B"),
		Info:      lipgloss.Color("#3B82F6"),
		Text:      lipgloss.Color("#E5E7EB"),
		TextMuted: lipgloss.Color("#9CA3AF"),
		Added:     lipgloss.Color("#22C55E"),
		Removed:   lipgloss.Color("#EF4444"),
		AddedBg:   lipgloss.Color("#14532D"),
		RemovedBg: lipgloss.Color("#7F1D1D"),
	}

	styles := types.ThemeStyles{
		Success: lipgloss.NewStyle().Foreground(theme.Success),
		Error:   lipgloss.NewStyle().Foreground(theme.Error),
	}

	renderer := NewToolResultRenderer(theme, styles)

	tests := []struct {
		name    string
		tool    string
		input   string
		output  string
		status  string
		wantErr bool
	}{
		{
			name:    "bash success",
			tool:    "bash",
			input:   "ls -la",
			output:  "file1.txt\nfile2.txt",
			status:  "success",
			wantErr: false,
		},
		{
			name:    "bash error",
			tool:    "bash",
			input:   "invalid_cmd",
			output:  "command not found",
			status:  "error",
			wantErr: false,
		},
		{
			name:    "read file",
			tool:    "read",
			input:   "/path/to/file.go",
			output:  "package main\n\nfunc main() {}",
			status:  "success",
			wantErr: false,
		},
		{
			name:    "write file",
			tool:    "write",
			input:   "/path/to/file.go",
			output:  "package main",
			status:  "success",
			wantErr: false,
		},
		{
			name:    "edit file",
			tool:    "edit",
			input:   "/path/to/file.go",
			output:  "-old line\n+new line",
			status:  "success",
			wantErr: false,
		},
		{
			name:    "glob",
			tool:    "glob",
			input:   "**/*.go",
			output:  "main.go\nutil.go\nfoo/bar.go",
			status:  "success",
			wantErr: false,
		},
		{
			name:    "grep",
			tool:    "grep",
			input:   "func main",
			output:  "main.go:10:func main() {}",
			status:  "success",
			wantErr: false,
		},
		{
			name:    "todo",
			tool:    "todo",
			input:   "Add feature X",
			output:  "Todo added",
			status:  "success",
			wantErr: false,
		},
		{
			name:    "unknown tool",
			tool:    "custom_tool",
			input:   "input",
			output:  "output",
			status:  "success",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.RenderResult(tt.tool, tt.input, tt.output, tt.status)

			// Check that result is not empty
			if result == "" {
				t.Errorf("RenderResult() returned empty string")
			}

			// Check that tool name appears in output
			if !strings.Contains(result, tt.tool) && tt.tool != "unknown tool" {
				t.Errorf("RenderResult() output missing tool name: %s", tt.tool)
			}

			// Check status indicator
			switch tt.status {
			case "success":
				if !strings.Contains(result, "Success") && !strings.Contains(result, "✓") {
					t.Errorf("RenderResult() success status not rendered")
				}
			case "error":
				if !strings.Contains(result, "Error") && !strings.Contains(result, "✗") {
					t.Errorf("RenderResult() error status not rendered")
				}
			}
		})
	}
}

func TestToolResultRenderer_renderBash(t *testing.T) {
	theme := types.Theme{
		Primary:   lipgloss.Color("#7C3AED"),
		Text:      lipgloss.Color("#E5E7EB"),
		TextMuted: lipgloss.Color("#9CA3AF"),
		Success:   lipgloss.Color("#10B981"),
	}
	styles := types.ThemeStyles{}
	renderer := NewToolResultRenderer(theme, styles)

	result := renderer.renderBash("ls -la", "file1.txt\nfile2.txt", "success")

	if !strings.Contains(result, "bash") {
		t.Errorf("renderBash() missing 'bash' header")
	}

	if !strings.Contains(result, "ls -la") {
		t.Errorf("renderBash() missing command")
	}

	if !strings.Contains(result, "file1.txt") {
		t.Errorf("renderBash() missing output")
	}
}

func TestToolResultRenderer_renderRead(t *testing.T) {
	theme := types.Theme{
		Primary:   lipgloss.Color("#7C3AED"),
		TextMuted: lipgloss.Color("#9CA3AF"),
	}
	styles := types.ThemeStyles{}
	renderer := NewToolResultRenderer(theme, styles)

	// Test file reading
	result := renderer.renderRead("/path/to/file.go", "package main", "success")

	if !strings.Contains(result, "read") {
		t.Errorf("renderRead() missing 'read' header")
	}

	if !strings.Contains(result, "/path/to/file.go") {
		t.Errorf("renderRead() missing file path")
	}
}

func TestToolResultRenderer_renderGlob(t *testing.T) {
	theme := types.Theme{
		Info:      lipgloss.Color("#3B82F6"),
		TextMuted: lipgloss.Color("#9CA3AF"),
		Text:      lipgloss.Color("#E5E7EB"),
	}
	styles := types.ThemeStyles{}
	renderer := NewToolResultRenderer(theme, styles)

	files := "main.go\nutil.go\ntest.go"
	result := renderer.renderGlob("**/*.go", files, "success")

	if !strings.Contains(result, "glob") {
		t.Errorf("renderGlob() missing 'glob' header")
	}

	if !strings.Contains(result, "**/*.go") {
		t.Errorf("renderGlob() missing pattern")
	}

	if !strings.Contains(result, "main.go") {
		t.Errorf("renderGlob() missing files")
	}

	if !strings.Contains(result, "(3 files)") {
		t.Errorf("renderGlob() missing file count")
	}
}

func TestToolResultRenderer_renderGrep(t *testing.T) {
	theme := types.Theme{
		Info:      lipgloss.Color("#3B82F6"),
		TextMuted: lipgloss.Color("#9CA3AF"),
		Text:      lipgloss.Color("#E5E7EB"),
		Accent:    lipgloss.Color("#10B981"),
	}
	styles := types.ThemeStyles{}
	renderer := NewToolResultRenderer(theme, styles)

	matches := "main.go:10:func main() {}"
	result := renderer.renderGrep("func main", matches, "success")

	if !strings.Contains(result, "grep") {
		t.Errorf("renderGrep() missing 'grep' header")
	}

	if !strings.Contains(result, "func main") {
		t.Errorf("renderGrep() missing pattern")
	}
}

func TestToolResultRenderer_renderDiff(t *testing.T) {
	theme := types.Theme{
		Added:     lipgloss.Color("#22C55E"),
		Removed:   lipgloss.Color("#EF4444"),
		AddedBg:   lipgloss.Color("#14532D"),
		RemovedBg: lipgloss.Color("#7F1D1D"),
		Info:      lipgloss.Color("#3B82F6"),
		TextMuted: lipgloss.Color("#9CA3AF"),
	}
	styles := types.ThemeStyles{}
	renderer := NewToolResultRenderer(theme, styles)

	diff := "-removed line\n+added line\n@@ context"
	result := renderer.renderDiff(diff)

	if result == "" {
		t.Error("renderDiff() returned empty string")
	}
}

func TestToolResultRenderer_truncateOutput(t *testing.T) {
	theme := types.Theme{}
	styles := types.ThemeStyles{}
	renderer := NewToolResultRenderer(theme, styles)

	// Test truncation by lines
	longOutput := strings.Repeat("line\n", 100)
	result := renderer.truncateOutput(longOutput, 10, 10000)

	if !strings.Contains(result, "truncated") {
		t.Errorf("truncateOutput() should indicate truncation")
	}

	// Test truncation by chars
	longOutput = strings.Repeat("a", 10000)
	result = renderer.truncateOutput(longOutput, 100, 1000)

	if !strings.Contains(result, "truncated") {
		t.Errorf("truncateOutput() should indicate truncation")
	}
}

func TestToolResultRenderer_highlightPattern(t *testing.T) {
	theme := types.Theme{
		Accent: lipgloss.Color("#10B981"),
	}
	styles := types.ThemeStyles{}
	renderer := NewToolResultRenderer(theme, styles)

	result := renderer.highlightPattern("hello world", "world")

	if result == "" {
		t.Error("highlightPattern() returned empty string")
	}

	// Test no match
	result = renderer.highlightPattern("hello world", "foo")
	if result != "hello world" {
		t.Errorf("highlightPattern() should return original text when no match")
	}
}

func TestToolResultRenderer_RenderToolCall(t *testing.T) {
	theme := types.Theme{
		Primary:   lipgloss.Color("#7C3AED"),
		TextMuted: lipgloss.Color("#9CA3AF"),
	}
	styles := types.ThemeStyles{}
	renderer := NewToolResultRenderer(theme, styles)

	result := renderer.RenderToolCall("bash", "ls -la")

	if !strings.Contains(result, "bash") {
		t.Errorf("RenderToolCall() missing tool name")
	}

	if !strings.Contains(result, "ls -la") {
		t.Errorf("RenderToolCall() missing input")
	}
}
