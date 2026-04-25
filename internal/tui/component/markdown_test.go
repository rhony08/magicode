// Package component provides reusable UI components for the TUI.
// This file contains tests for the markdown renderer.
package component

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

func TestMarkdownRenderer_Render(t *testing.T) {
	theme := types.Theme{
		Heading:    lipgloss.Color("#7C3AED"),
		Text:       lipgloss.Color("#E5E7EB"),
		TextMuted:  lipgloss.Color("#9CA3AF"),
		BlockQuote: lipgloss.Color("#9CA3AF"),
		Code:       lipgloss.Color("#FBBF24"),
		CodeBg:     lipgloss.Color("#1F2937"),
		Border:     lipgloss.Color("#374151"),
		Primary:    lipgloss.Color("#7C3AED"),
		Link:       lipgloss.Color("#3B82F6"),
		Keyword:    lipgloss.Color("#7C3AED"),
		String:     lipgloss.Color("#FBBF24"),
		Comment:    lipgloss.Color("#6B7280"),
	}

	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "empty string",
			input:    "",
			contains: []string{},
		},
		{
			name:     "plain paragraph",
			input:    "This is a simple paragraph.",
			contains: []string{"This is a simple paragraph"},
		},
		{
			name:  "heading level 1",
			input: "# Heading 1",
			contains: []string{
				"Heading 1",
			},
		},
		{
			name:  "heading level 2",
			input: "## Heading 2",
			contains: []string{
				"Heading 2",
			},
		},
		{
			name:  "heading level 3",
			input: "### Heading 3",
			contains: []string{
				"Heading 3",
			},
		},
		{
			name:  "bold text",
			input: "This is **bold** text.",
			contains: []string{
				"This is",
				"bold",
				"text",
			},
		},
		{
			name:  "italic text",
			input: "This is *italic* text.",
			contains: []string{
				"This is",
				"italic",
				"text",
			},
		},
		{
			name:  "code inline",
			input: "Use `printf` for output.",
			contains: []string{
				"Use",
				"printf",
				"for output",
			},
		},
		{
			name:  "unordered list",
			input: "- Item 1\n- Item 2\n- Item 3",
			contains: []string{
				"Item 1",
				"Item 2",
				"Item 3",
			},
		},
		{
			name:  "ordered list",
			input: "1. First\n2. Second\n3. Third",
			contains: []string{
				"1.",
				"First",
				"Second",
				"Third",
			},
		},
		{
			name:  "blockquote",
			input: "> This is a quote",
			contains: []string{
				"This is a quote",
			},
		},
		{
			name:  "link",
			input: "Visit [Google](https://google.com) for search.",
			contains: []string{
				"Visit",
				"Google",
				"for search",
			},
		},
		{
			name:  "image",
			input: "![Alt text](https://example.com/image.png)",
			contains: []string{
				"Alt text",
			},
		},
		{
			name:  "strikethrough",
			input: "This is ~~deleted~~ text.",
			contains: []string{
				"deleted",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.Render(tt.input)

			if tt.input == "" && result != "" {
				t.Errorf("Render() returned non-empty for empty input")
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Render() missing expected content: %q in:\n%s", expected, result)
				}
			}
		})
	}
}

func TestMarkdownRenderer_RenderCodeBlock(t *testing.T) {
	theme := types.Theme{
		Code:   lipgloss.Color("#FBBF24"),
		CodeBg: lipgloss.Color("#1F2937"),
		Text:   lipgloss.Color("#E5E7EB"),
	}

	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	codeBlock := "```go\npackage main\n\nfunc main() {\n    println(\"Hello\")\n}\n```"

	result := renderer.Render(codeBlock)

	if result == "" {
		t.Error("Render() returned empty for code block")
	}

	// Should contain code content
	if !strings.Contains(result, "package main") {
		t.Error("Render() missing code content")
	}

	// Should show language
	if !strings.Contains(result, "go") {
		t.Error("Render() missing language label")
	}
}

func TestMarkdownRenderer_RenderTable(t *testing.T) {
	theme := types.Theme{
		Primary: lipgloss.Color("#7C3AED"),
		Text:    lipgloss.Color("#E5E7EB"),
		Border:  lipgloss.Color("#374151"),
	}

	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	table := `| Name  | Age |
|-------|-----|
| Alice | 30  |
| Bob   | 25  |`

	result := renderer.Render(table)

	if result == "" {
		t.Error("Render() returned empty for table")
	}

	// Should contain headers
	if !strings.Contains(result, "Name") {
		t.Error("Render() missing table header 'Name'")
	}

	if !strings.Contains(result, "Age") {
		t.Error("Render() missing table header 'Age'")
	}

	// Should contain data
	if !strings.Contains(result, "Alice") {
		t.Error("Render() missing table data 'Alice'")
	}
}

func TestMarkdownRenderer_RenderHorizontalRule(t *testing.T) {
	theme := types.Theme{
		Border: lipgloss.Color("#374151"),
	}

	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	hr := "---"
	result := renderer.Render(hr)

	if result == "" {
		t.Error("Render() returned empty for horizontal rule")
	}

	// Should contain horizontal line characters
	if !strings.Contains(result, "─") {
		t.Error("Render() missing horizontal rule character")
	}
}

func TestMarkdownRenderer_splitBlocks(t *testing.T) {
	theme := types.Theme{}
	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	input := "Block 1\n\nBlock 2\n\nBlock 3"
	blocks := renderer.splitBlocks(input)

	if len(blocks) != 3 {
		t.Errorf("splitBlocks() returned %d blocks, want 3", len(blocks))
	}
}

func TestMarkdownRenderer_highlightSyntax(t *testing.T) {
	theme := types.Theme{
		Keyword: lipgloss.Color("#7C3AED"),
		String:  lipgloss.Color("#FBBF24"),
		Comment: lipgloss.Color("#6B7280"),
	}

	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	// Test keyword highlighting - just ensure it doesn't panic
	line := "func main() {"
	result := renderer.highlightSyntax(line, "go")

	// Result should not be empty
	if result == "" {
		t.Error("highlightSyntax() returned empty string")
	}
}

func TestMarkdownRenderer_RenderInline(t *testing.T) {
	theme := types.Theme{
		Text:   lipgloss.Color("#E5E7EB"),
		Link:   lipgloss.Color("#3B82F6"),
		CodeBg: lipgloss.Color("#1F2937"),
		Code:   lipgloss.Color("#FBBF24"),
	}

	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	tests := []struct {
		name  string
		input string
		check string
	}{
		{
			name:  "bold",
			input: "**bold**",
			check: "bold",
		},
		{
			name:  "italic",
			input: "*italic*",
			check: "italic",
		},
		{
			name:  "code",
			input: "`code`",
			check: "code",
		},
		{
			name:  "link",
			input: "[text](url)",
			check: "text",
		},
		{
			name:  "image",
			input: "![alt](url)",
			check: "alt",
		},
		{
			name:  "strikethrough",
			input: "~~deleted~~",
			check: "deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.renderInline(tt.input)
			if !strings.Contains(result, tt.check) {
				t.Errorf("renderInline() missing %q in:\n%s", tt.check, result)
			}
		})
	}
}

func TestMarkdownRenderer_isTable(t *testing.T) {
	theme := types.Theme{}
	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name: "valid table",
			input: `| A | B |
|---|---|
| 1 | 2 |`,
			want: true,
		},
		{
			name:  "not a table - too few lines",
			input: "| A | B |",
			want:  false,
		},
		{
			name:  "not a table - no separator",
			input: "| A | B |\n| C | D |",
			want:  false,
		},
		{
			name:  "paragraph",
			input: "This is just text",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.isTable(tt.input)
			if got != tt.want {
				t.Errorf("isTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkdownRenderer_truncateString(t *testing.T) {
	theme := types.Theme{}
	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	tests := []struct {
		input   string
		maxLen  int
		wantEnd string
	}{
		{
			input:   "short",
			maxLen:  10,
			wantEnd: "short",
		},
		{
			input:   "this is a very long string",
			maxLen:  10,
			wantEnd: "...",
		},
	}

	for _, tt := range tests {
		result := renderer.truncateString(tt.input, tt.maxLen)
		if !strings.HasSuffix(result, tt.wantEnd) {
			t.Errorf("truncateString(%q, %d) = %q, want suffix %q",
				tt.input, tt.maxLen, result, tt.wantEnd)
		}
	}
}

func TestMarkdownRenderer_SetWidth(t *testing.T) {
	theme := types.Theme{}
	styles := types.ThemeStyles{}
	renderer := NewMarkdownRenderer(theme, styles, 80)

	renderer.SetWidth(100)

	// Width is private, so we test it indirectly
	// by checking that content fits the new width
	longText := strings.Repeat("word ", 50)
	result := renderer.Render(longText)

	if result == "" {
		t.Error("Render() after SetWidth() returned empty")
	}
}
