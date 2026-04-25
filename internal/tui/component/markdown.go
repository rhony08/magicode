// Package component provides reusable UI components for the TUI.
// This file implements markdown rendering for the terminal.
package component

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// MarkdownRenderer renders markdown content for terminal display
type MarkdownRenderer struct {
	theme  types.Theme
	styles types.ThemeStyles
	width  int
}

// NewMarkdownRenderer creates a new markdown renderer
func NewMarkdownRenderer(theme types.Theme, styles types.ThemeStyles, width int) *MarkdownRenderer {
	if width <= 0 {
		width = 80
	}
	return &MarkdownRenderer{
		theme:  theme,
		styles: styles,
		width:  width,
	}
}

// SetWidth updates the render width
func (r *MarkdownRenderer) SetWidth(width int) {
	r.width = width
}

// Render converts markdown text to styled terminal output
func (r *MarkdownRenderer) Render(markdown string) string {
	if markdown == "" {
		return ""
	}

	// Split into blocks (paragraphs, code blocks, lists, etc.)
	blocks := r.splitBlocks(markdown)

	var rendered []string
	for _, block := range blocks {
		if block == "" {
			continue
		}
		rendered = append(rendered, r.renderBlock(block))
	}

	return strings.Join(rendered, "\n\n")
}

// splitBlocks splits markdown into blocks
func (r *MarkdownRenderer) splitBlocks(markdown string) []string {
	// Split by double newlines, but preserve code blocks
	var blocks []string
	var currentBlock strings.Builder
	inCodeBlock := false
	codeFence := ""

	lines := strings.Split(markdown, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code block start/end
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if !inCodeBlock {
				// End current block if any
				if currentBlock.Len() > 0 {
					blocks = append(blocks, strings.TrimSpace(currentBlock.String()))
					currentBlock.Reset()
				}
				inCodeBlock = true
				codeFence = trimmed[:3]
				currentBlock.WriteString(line)
				if i < len(lines)-1 {
					currentBlock.WriteString("\n")
				}
			} else if strings.HasPrefix(trimmed, codeFence) {
				// End code block
				currentBlock.WriteString(line)
				blocks = append(blocks, currentBlock.String())
				currentBlock.Reset()
				inCodeBlock = false
				codeFence = ""
			} else {
				currentBlock.WriteString(line)
				if i < len(lines)-1 {
					currentBlock.WriteString("\n")
				}
			}
			continue
		}

		if inCodeBlock {
			currentBlock.WriteString(line)
			if i < len(lines)-1 {
				currentBlock.WriteString("\n")
			}
		} else if trimmed == "" {
			// Empty line ends current block
			if currentBlock.Len() > 0 {
				blocks = append(blocks, strings.TrimSpace(currentBlock.String()))
				currentBlock.Reset()
			}
		} else {
			currentBlock.WriteString(line)
			if i < len(lines)-1 {
				currentBlock.WriteString("\n")
			}
		}
	}

	// Add remaining block
	if currentBlock.Len() > 0 {
		blocks = append(blocks, strings.TrimSpace(currentBlock.String()))
	}

	return blocks
}

// renderBlock renders a single markdown block
func (r *MarkdownRenderer) renderBlock(block string) string {
	trimmed := strings.TrimSpace(block)

	// Code block
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		return r.renderCodeBlock(block)
	}

	// Heading
	if match := regexp.MustCompile(`^(#{1,6})\s+(.+)$`).FindStringSubmatch(trimmed); match != nil {
		level := len(match[1])
		content := match[2]
		return r.renderHeading(content, level)
	}

	// Blockquote
	if strings.HasPrefix(trimmed, ">") {
		return r.renderBlockquote(block)
	}

	// Unordered list
	if regexp.MustCompile(`^[\s]*[-*+]\s+`).MatchString(trimmed) {
		return r.renderList(block, false)
	}

	// Ordered list
	if regexp.MustCompile(`^[\s]*\d+\.\s+`).MatchString(trimmed) {
		return r.renderList(block, true)
	}

	// Horizontal rule
	if regexp.MustCompile(`^(---+|===+|___+)$`).MatchString(trimmed) {
		return r.renderHorizontalRule()
	}

	// Table
	if r.isTable(block) {
		return r.renderTable(block)
	}

	// Regular paragraph
	return r.renderParagraph(block)
}

// renderHeading renders a heading
func (r *MarkdownRenderer) renderHeading(content string, level int) string {
	var style lipgloss.Style

	switch level {
	case 1:
		style = lipgloss.NewStyle().
			Foreground(r.theme.Heading).
			Bold(true).
			Underline(true).
			MarginBottom(1)
	case 2:
		style = lipgloss.NewStyle().
			Foreground(r.theme.Heading).
			Bold(true).
			MarginBottom(1)
	case 3:
		style = lipgloss.NewStyle().
			Foreground(r.theme.Heading).
			Bold(true)
	default:
		style = lipgloss.NewStyle().
			Foreground(r.theme.Heading).
			Bold(true)
	}

	// Process inline formatting
	content = r.renderInline(content)

	return style.Render(content)
}

// renderParagraph renders a paragraph
func (r *MarkdownRenderer) renderParagraph(content string) string {
	// Process inline formatting
	content = r.renderInline(content)

	// Wrap text
	style := lipgloss.NewStyle().
		Width(r.width).
		Foreground(r.theme.Text)

	return style.Render(content)
}

// renderBlockquote renders a blockquote
func (r *MarkdownRenderer) renderBlockquote(content string) string {
	lines := strings.Split(content, "\n")
	var cleaned []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">") {
			cleaned = append(cleaned, strings.TrimSpace(trimmed[1:]))
		} else {
			cleaned = append(cleaned, trimmed)
		}
	}

	content = strings.Join(cleaned, " ")
	content = r.renderInline(content)

	style := lipgloss.NewStyle().
		Foreground(r.theme.BlockQuote).
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(r.theme.Border).
		PaddingLeft(1).
		Width(r.width - 2)

	return style.Render(content)
}

// renderCodeBlock renders a code block
func (r *MarkdownRenderer) renderCodeBlock(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return r.renderParagraph(content)
	}

	// Extract language from first line
	firstLine := strings.TrimSpace(lines[0])
	language := ""
	if strings.HasPrefix(firstLine, "```") {
		language = strings.TrimSpace(firstLine[3:])
	} else if strings.HasPrefix(firstLine, "~~~") {
		language = strings.TrimSpace(firstLine[3:])
	}

	// Get code content (skip first and last lines)
	codeLines := lines[1 : len(lines)-1]
	code := strings.Join(codeLines, "\n")

	// Apply syntax highlighting
	code = r.highlightCode(code, language)

	// Code block style
	codeStyle := lipgloss.NewStyle().
		Background(r.theme.CodeBg).
		Foreground(r.theme.Code).
		Padding(1).
		Width(r.width)

	result := codeStyle.Render(code)

	// Add language label if present
	if language != "" {
		labelStyle := lipgloss.NewStyle().
			Background(r.theme.PanelBg).
			Foreground(r.theme.TextMuted).
			Padding(0, 1).
			MarginBottom(0)
		label := labelStyle.Render(language)
		result = label + "\n" + result
	}

	return result
}

// renderList renders a list (ordered or unordered)
func (r *MarkdownRenderer) renderList(content string, ordered bool) string {
	lines := strings.Split(content, "\n")
	var items []string
	var currentItem strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for list item
		unorderedMatch := regexp.MustCompile(`^(\s*)[-*+]\s+(.+)$`).FindStringSubmatch(line)
		orderedMatch := regexp.MustCompile(`^(\s*)\d+\.\s+(.+)$`).FindStringSubmatch(line)

		if unorderedMatch != nil || orderedMatch != nil {
			// Save previous item
			if currentItem.Len() > 0 {
				items = append(items, currentItem.String())
				currentItem.Reset()
			}

			// Start new item
			var match []string
			if unorderedMatch != nil {
				match = unorderedMatch
			} else {
				match = orderedMatch
			}

			itemContent := match[2]
			currentItem.WriteString(itemContent)
		} else if currentItem.Len() > 0 {
			// Continue current item
			currentItem.WriteString(" ")
			currentItem.WriteString(trimmed)
		}
	}

	// Add last item
	if currentItem.Len() > 0 {
		items = append(items, currentItem.String())
	}

	// Render items
	var rendered []string
	for i, item := range items {
		item = r.renderInline(item)

		var prefix string
		if ordered {
			prefix = fmt.Sprintf("%d. ", i+1)
		} else {
			prefix = "• "
		}

		prefixStyle := lipgloss.NewStyle().
			Foreground(r.theme.Primary)

		itemStyle := lipgloss.NewStyle().
			Foreground(r.theme.Text).
			Width(r.width - 4)

		rendered = append(rendered, prefixStyle.Render(prefix)+itemStyle.Render(item))
	}

	return strings.Join(rendered, "\n")
}

// renderTable renders a markdown table
func (r *MarkdownRenderer) isTable(content string) bool {
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return false
	}
	// Check for separator line (contains |---|---|)
	return regexp.MustCompile(`\|[-:]+\|`).MatchString(lines[1])
}

// renderTable renders a table
func (r *MarkdownRenderer) renderTable(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return r.renderParagraph(content)
	}

	// Parse header
	headerLine := lines[0]
	headers := r.parseTableRow(headerLine)

	// Skip separator line
	var rows [][]string
	for i := 2; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			rows = append(rows, r.parseTableRow(lines[i]))
		}
	}

	// Render table
	var rendered []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Primary).
		Bold(true).
		Underline(true)
	headerCells := make([]string, len(headers))
	for i, h := range headers {
		headerCells[i] = headerStyle.Render(r.truncateString(strings.TrimSpace(h), 20))
	}
	rendered = append(rendered, strings.Join(headerCells, " | "))

	// Separator
	sepStyle := lipgloss.NewStyle().
		Foreground(r.theme.Border)
	sep := sepStyle.Render(strings.Repeat("-", r.width))
	rendered = append(rendered, sep)

	// Rows
	rowStyle := lipgloss.NewStyle().
		Foreground(r.theme.Text)
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = rowStyle.Render(r.truncateString(strings.TrimSpace(cell), 20))
		}
		rendered = append(rendered, strings.Join(cells, " | "))
	}

	return strings.Join(rendered, "\n")
}

// parseTableRow parses a table row
func (r *MarkdownRenderer) parseTableRow(line string) []string {
	// Remove leading and trailing |
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// renderHorizontalRule renders a horizontal rule
func (r *MarkdownRenderer) renderHorizontalRule() string {
	return lipgloss.NewStyle().
		Foreground(r.theme.Border).
		Render(strings.Repeat("─", r.width))
}

// renderInline processes inline markdown formatting
func (r *MarkdownRenderer) renderInline(content string) string {
	// Bold: **text**
	boldDoubleRe := regexp.MustCompile(`\*\*(.+?)\*\*`)
	content = boldDoubleRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := boldDoubleRe.FindStringSubmatch(match)[1]
		style := lipgloss.NewStyle().Bold(true)
		return style.Render(inner)
	})

	// Bold: __text__
	boldUnderRe := regexp.MustCompile(`__(.+?)__`)
	content = boldUnderRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := boldUnderRe.FindStringSubmatch(match)[1]
		style := lipgloss.NewStyle().Bold(true)
		return style.Render(inner)
	})

	// Italic: *text*
	italicStarRe := regexp.MustCompile(`\*(.+?)\*`)
	content = italicStarRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := italicStarRe.FindStringSubmatch(match)[1]
		style := lipgloss.NewStyle().Italic(true)
		return style.Render(inner)
	})

	// Italic: _text_
	italicUnderRe := regexp.MustCompile(`_(.+?)_`)
	content = italicUnderRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := italicUnderRe.FindStringSubmatch(match)[1]
		style := lipgloss.NewStyle().Italic(true)
		return style.Render(inner)
	})

	// Strikethrough: ~~text~~
	strikeRe := regexp.MustCompile(`~~(.+?)~~`)
	content = strikeRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := strikeRe.FindStringSubmatch(match)[1]
		style := lipgloss.NewStyle().Strikethrough(true)
		return style.Render(inner)
	})

	// Inline code: `code`
	codeRe := regexp.MustCompile("`([^`]+)`")
	content = codeRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := codeRe.FindStringSubmatch(match)[1]
		style := lipgloss.NewStyle().
			Background(r.theme.CodeBg).
			Foreground(r.theme.Code)
		return style.Render(inner)
	})

	// Links: [text](url) or [text](url "title")
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)"]+)(?:\s+"[^"]*")?\)`)
	content = linkRe.ReplaceAllStringFunc(content, func(match string) string {
		matches := linkRe.FindStringSubmatch(match)
		text := matches[1]
		// url := matches[2] // URL not displayed in terminal
		style := lipgloss.NewStyle().
			Foreground(r.theme.Link).
			Underline(true)
		return style.Render(text)
	})

	// Images: ![alt](url) - show alt text
	imageRe := regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	content = imageRe.ReplaceAllStringFunc(content, func(match string) string {
		matches := imageRe.FindStringSubmatch(match)
		alt := matches[1]
		if alt == "" {
			alt = "[image]"
		}
		style := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted).
			Italic(true)
		return style.Render("[📷 " + alt + "]")
	})

	return content
}

// highlightCode applies basic syntax highlighting
func (r *MarkdownRenderer) highlightCode(code string, language string) string {
	// For now, apply simple highlighting based on common patterns
	// In a full implementation, you could use chroma or similar

	lines := strings.Split(code, "\n")
	var highlighted []string

	for _, line := range lines {
		// Comments (simple detection)
		if strings.TrimSpace(line) == "" {
			highlighted = append(highlighted, line)
			continue
		}

		// Basic syntax highlighting
		line = r.highlightSyntax(line, language)
		highlighted = append(highlighted, line)
	}

	return strings.Join(highlighted, "\n")
}

// highlightSyntax applies basic syntax highlighting to a line
func (r *MarkdownRenderer) highlightSyntax(line string, language string) string {
	// This is a simplified highlighter
	// Keywords
	keywords := []string{"func", "function", "class", "if", "else", "for", "while", "return", "import", "package", "const", "var", "let", "def", "async", "await"}
	for _, kw := range keywords {
		re := regexp.MustCompile(`\b` + kw + `\b`)
		line = re.ReplaceAllStringFunc(line, func(match string) string {
			style := lipgloss.NewStyle().Foreground(r.theme.Keyword)
			return style.Render(match)
		})
	}

	// Strings (simple double and single quotes)
	stringRe := regexp.MustCompile(`"([^"]*)"`)
	line = stringRe.ReplaceAllStringFunc(line, func(match string) string {
		style := lipgloss.NewStyle().Foreground(r.theme.String)
		return style.Render(match)
	})

	// Comments (single line)
	if idx := strings.Index(line, "//"); idx != -1 {
		before := line[:idx]
		comment := line[idx:]
		commentStyle := lipgloss.NewStyle().Foreground(r.theme.Comment)
		line = before + commentStyle.Render(comment)
	}

	return line
}

// truncateString truncates a string to a maximum length
func (r *MarkdownRenderer) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
