package util

import (
	"strings"
	"testing"
)

func TestWrap(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected []string
	}{
		{
			name:     "short text no wrap",
			text:     "hello world",
			width:    20,
			expected: []string{"hello world"},
		},
		{
			name:     "text needs wrapping",
			text:     "hello world this is a long text that needs to be wrapped",
			width:    20,
			expected: []string{"hello world this is", "a long text that", "needs to be wrapped"},
		},
		{
			name:     "long word needs breaking",
			text:     "supercalifragilisticexpialidocious",
			width:    10,
			expected: []string{"supercalif", "ragilistic", "expialidoc", "ious"},
		},
		{
			name:     "empty text",
			text:     "",
			width:    10,
			expected: []string{},
		},
		{
			name:     "single word",
			text:     "hello",
			width:    10,
			expected: []string{"hello"},
		},
		{
			name:     "width zero",
			text:     "hello",
			width:    0,
			expected: []string{"hello"},
		},
		{
			name:     "multiple spaces",
			text:     "hello    world",
			width:    20,
			expected: []string{"hello world"}, // Fields() collapses spaces
		},
		{
			name:     "text with newlines",
			text:     "line one\nline two",
			width:    20,
			expected: []string{"line one line two"}, // Newlines collapsed by Fields()
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Wrap(tt.text, tt.width)

			// Compare line by line
			if len(result) != len(tt.expected) {
				t.Errorf("Wrap() = %v, expected %v (line count mismatch)", result, tt.expected)
				return
			}

			for i := 0; i < len(result); i++ {
				if result[i] != tt.expected[i] {
					t.Errorf("Wrap() line %d = %q, expected %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestWrapPreserveIndent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected []string
	}{
		{
			name:  "code with indentation",
			text:  "    function hello() {\n        return 'world';\n    }",
			width: 30,
			expected: []string{
				"    function hello() {",
				"        return 'world';",
				"    }",
			},
		},
		{
			name:  "wrapped indented line",
			text:  "    this is a very long line that needs wrapping",
			width: 20,
			expected: []string{
				"    this is a very",
				"    long line that",
				"    needs wrapping",
			},
		},
		{
			name:  "mixed indentation",
			text:  "  line one\n    line two\n  line three",
			width: 20,
			expected: []string{
				"  line one",
				"    line two",
				"  line three",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapPreserveIndent(tt.text, tt.width)

			if len(result) != len(tt.expected) {
				t.Errorf("WrapPreserveIndent() = %v, expected %v", result, tt.expected)
				return
			}

			for i := 0; i < len(result); i++ {
				if result[i] != tt.expected[i] {
					t.Errorf("WrapPreserveIndent() line %d = %q, expected %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		suffix   string
		expected string
	}{
		{
			name:     "no truncation needed",
			text:     "hello",
			width:    10,
			suffix:   "...",
			expected: "hello",
		},
		{
			name:     "truncate with ellipsis",
			text:     "hello world",
			width:    8,
			suffix:   "...",
			expected: "hello...",
		},
		{
			name:     "width smaller than suffix",
			text:     "hello",
			width:    2,
			suffix:   "...",
			expected: "..",
		},
		{
			name:     "width equals suffix length",
			text:     "hello",
			width:    3,
			suffix:   "...",
			expected: "...",
		},
		{
			name:     "empty text",
			text:     "",
			width:    10,
			suffix:   "...",
			expected: "",
		},
		{
			name:     "zero width",
			text:     "hello",
			width:    0,
			suffix:   "...",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.text, tt.width, tt.suffix)
			if result != tt.expected {
				t.Errorf("Truncate() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "ascii only",
			text:     "hello",
			expected: 5,
		},
		{
			name:     "with spaces",
			text:     "hello world",
			expected: 11,
		},
		{
			name:     "with ansi codes",
			text:     "\x1b[31mred\x1b[0m",
			expected: 3, // ANSI codes have 0 width
		},
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "cjk character",
			text:     "中文",
			expected: 4, // CJK has width 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringWidth(tt.text)
			if result != tt.expected {
				t.Errorf("StringWidth(%q) = %d, expected %d", tt.text, result, tt.expected)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ansi",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "simple color",
			input:    "\x1b[31mred\x1b[0m",
			expected: "red",
		},
		{
			name:     "multiple codes",
			input:    "\x1b[1mbold\x1b[0m and \x1b[4munderline\x1b[0m",
			expected: "bold and underline",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestWrapPreserveNewlines(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected []string
	}{
		{
			name:  "preserve newlines",
			text:  "line one\nline two\nline three",
			width: 20,
			expected: []string{
				"line one",
				"line two",
				"line three",
			},
		},
		{
			name:  "wrap and preserve newlines",
			text:  "this is a very long first line\nsecond line",
			width: 15,
			expected: []string{
				"this is a very",
				"long first line",
				"second line",
			},
		},
		{
			name:  "multiple consecutive newlines",
			text:  "line one\n\nline three",
			width: 20,
			expected: []string{
				"line one",
				"", // Empty line preserved
				"line three",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapPreserveNewlines(tt.text, tt.width)

			if len(result) != len(tt.expected) {
				t.Errorf("WrapPreserveNewlines() = %v, expected %v", result, tt.expected)
				return
			}

			for i := 0; i < len(result); i++ {
				if result[i] != tt.expected[i] {
					t.Errorf("WrapPreserveNewlines() line %d = %q, expected %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func BenchmarkWrap(b *testing.B) {
	text := strings.Repeat("hello world this is a test ", 100)
	for i := 0; i < b.N; i++ {
		Wrap(text, 80)
	}
}

func BenchmarkStringWidth(b *testing.B) {
	text := strings.Repeat("hello", 100) + "\x1b[31mred\x1b[0m"
	for i := 0; i < b.N; i++ {
		StringWidth(text)
	}
}
