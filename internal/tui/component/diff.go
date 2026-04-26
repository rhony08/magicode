// Package component provides reusable UI components for the TUI.
// This file implements diff rendering with unified and split (side-by-side) views.
package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
	"github.com/rhony08/magicode/internal/tui/util"
)

// DiffStyle determines how diffs are displayed
type DiffStyle string

const (
	DiffStyleUnified DiffStyle = "unified" // Stacked view (default for narrow terminals)
	DiffStyleSplit   DiffStyle = "split"   // Side-by-side view (for wide terminals)
	DiffStyleAuto    DiffStyle = "auto"    // Auto-select based on terminal width
)

// DiffRenderer renders diffs with unified or split views
type DiffRenderer struct {
	theme     types.Theme
	styles    types.ThemeStyles
	style     DiffStyle
	width     int
	wrapLines bool
}

// NewDiffRenderer creates a new diff renderer
func NewDiffRenderer(theme types.Theme, styles types.ThemeStyles) *DiffRenderer {
	return &DiffRenderer{
		theme:     theme,
		styles:    styles,
		style:     DiffStyleAuto,
		width:     80,
		wrapLines: true,
	}
}

// SetStyle sets the diff display style
func (d *DiffRenderer) SetStyle(style DiffStyle) {
	d.style = style
}

// SetWidth sets the available width for rendering
func (d *DiffRenderer) SetWidth(width int) {
	d.width = width
}

// SetWrapLines enables or disables line wrapping
func (d *DiffRenderer) SetWrapLines(wrap bool) {
	d.wrapLines = wrap
}

// Render renders a diff with the configured style
func (d *DiffRenderer) Render(diffContent string, filePath string) string {
	// Determine which style to use
	viewStyle := d.style
	if viewStyle == DiffStyleAuto {
		// Use split view for wide terminals (>120), unified for narrow
		if d.width > 120 {
			viewStyle = DiffStyleSplit
		} else {
			viewStyle = DiffStyleUnified
		}
	}

	// Parse the diff into hunks
	hunks := d.parseDiff(diffContent)

	if viewStyle == DiffStyleSplit {
		return d.renderSplit(hunks, filePath)
	}
	return d.renderUnified(hunks, filePath)
}

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type    string // "added", "removed", "context", "header"
	Content string
	OldLine int // Line number in old file (for removed/context)
	NewLine int // Line number in new file (for added/context)
}

// DiffHunk represents a diff hunk with header and lines
type DiffHunk struct {
	Header   string // e.g., "@@ -1,5 +1,7 @@"
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// parseDiff parses unified diff format into hunks
func (d *DiffRenderer) parseDiff(diffContent string) []DiffHunk {
	lines := strings.Split(diffContent, "\n")
	var hunks []DiffHunk
	var currentHunk *DiffHunk

	oldLine := 0
	newLine := 0

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse hunk header: @@ -oldStart,oldCount +newStart,newCount @@
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}
			currentHunk = &DiffHunk{
				Header: line,
				Lines:  []DiffLine{},
			}
			// Parse line numbers from header
			d.parseHunkHeader(line, currentHunk)
			oldLine = currentHunk.OldStart
			newLine = currentHunk.NewStart
			continue
		}

		if currentHunk == nil {
			continue
		}

		// Parse individual lines
		if strings.HasPrefix(line, "+") {
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Type:    "added",
				Content: line[1:], // Remove the + prefix
				NewLine: newLine,
			})
			newLine++
		} else if strings.HasPrefix(line, "-") {
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Type:    "removed",
				Content: line[1:], // Remove the - prefix
				OldLine: oldLine,
			})
			oldLine++
		} else if strings.HasPrefix(line, " ") {
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Type:    "context",
				Content: line[1:], // Remove the space prefix
				OldLine: oldLine,
				NewLine: newLine,
			})
			oldLine++
			newLine++
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// parseHunkHeader extracts line numbers from hunk header
func (d *DiffRenderer) parseHunkHeader(header string, hunk *DiffHunk) {
	// Format: @@ -oldStart,oldCount +newStart,newCount @@
	// Example: @@ -1,5 +1,7 @@
	parts := strings.Split(header, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "-") {
			// Old file info: -start,count or -start (count defaults to 1)
			info := part[1:]
			if strings.Contains(info, ",") {
				nums := strings.Split(info, ",")
				hunk.OldStart = d.parseLineNumber(nums[0])
				hunk.OldCount = d.parseLineNumber(nums[1])
			} else {
				hunk.OldStart = d.parseLineNumber(info)
				hunk.OldCount = 1
			}
		} else if strings.HasPrefix(part, "+") {
			// New file info: +start,count or +start
			info := part[1:]
			if strings.Contains(info, ",") {
				nums := strings.Split(info, ",")
				hunk.NewStart = d.parseLineNumber(nums[0])
				hunk.NewCount = d.parseLineNumber(nums[1])
			} else {
				hunk.NewStart = d.parseLineNumber(info)
				hunk.NewCount = 1
			}
		}
	}
}

// parseLineNumber converts a string to int, returns 0 on error
func (d *DiffRenderer) parseLineNumber(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			break
		}
	}
	return result
}

// renderUnified renders a unified diff (stacked view)
func (d *DiffRenderer) renderUnified(hunks []DiffHunk, filePath string) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(d.theme.Warning).
		Bold(true)
	lines = append(lines, headerStyle.Render(fmt.Sprintf("← Edit %s", filePath)))

	// Calculate available width for content
	contentWidth := d.width - 10 // Reserve space for line numbers and prefix
	if contentWidth < 40 {
		contentWidth = 40
	}

	for _, hunk := range hunks {
		// Hunk header
		lines = append(lines, "") // Blank line before hunk
		hunkHeaderStyle := lipgloss.NewStyle().
			Foreground(d.theme.Info)
		lines = append(lines, hunkHeaderStyle.Render(hunk.Header))

		// Render each line in the hunk
		for _, dl := range hunk.Lines {
			lines = append(lines, d.renderUnifiedLine(dl, contentWidth))
		}
	}

	return strings.Join(lines, "\n")
}

// renderUnifiedLine renders a single line in unified diff
func (d *DiffRenderer) renderUnifiedLine(dl DiffLine, width int) string {
	// Determine style based on type
	var prefix string
	var lineStyle lipgloss.Style

	switch dl.Type {
	case "added":
		prefix = "+"
		lineStyle = lipgloss.NewStyle().
			Foreground(d.theme.Added).
			Background(d.theme.AddedBg)
	case "removed":
		prefix = "-"
		lineStyle = lipgloss.NewStyle().
			Foreground(d.theme.Removed).
			Background(d.theme.RemovedBg)
	case "context":
		prefix = " "
		lineStyle = lipgloss.NewStyle().
			Foreground(d.theme.TextMuted)
	default:
		prefix = " "
		lineStyle = lipgloss.NewStyle().
			Foreground(d.theme.Text)
	}

	// Format line number
	lineNum := ""
	if dl.Type == "added" && dl.NewLine > 0 {
		lineNum = fmt.Sprintf("%4d", dl.NewLine)
	} else if dl.Type == "removed" && dl.OldLine > 0 {
		lineNum = fmt.Sprintf("%4d", dl.OldLine)
	} else if dl.Type == "context" && dl.NewLine > 0 {
		lineNum = fmt.Sprintf("%4d", dl.NewLine)
	}

	// Line number style
	lineNumStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted).
		Width(4).
		Align(lipgloss.Right)

	// Wrap content if enabled
	content := dl.Content
	if d.wrapLines && util.StringWidth(content) > width {
		wrapped := util.WrapPreserveNewlines(content, width)
		content = strings.Join(wrapped, "\n")
	}

	// Build the line
	lineNumPart := lineNumStyle.Render(lineNum)
	prefixPart := lineStyle.Render(prefix)
	contentPart := lineStyle.Render(content)

	return lineNumPart + " " + prefixPart + contentPart
}

// renderSplit renders a split diff (side-by-side view)
func (d *DiffRenderer) renderSplit(hunks []DiffHunk, filePath string) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(d.theme.Warning).
		Bold(true)
	lines = append(lines, headerStyle.Render(fmt.Sprintf("← Edit %s", filePath)))

	// Calculate column widths for split view
	// Each column gets roughly half the width minus line number space
	halfWidth := (d.width - 20) / 2 // -20 for line numbers and separators
	if halfWidth < 30 {
		halfWidth = 30
	}

	for _, hunk := range hunks {
		// Hunk header
		lines = append(lines, "")
		hunkHeaderStyle := lipgloss.NewStyle().
			Foreground(d.theme.Info)
		lines = append(lines, hunkHeaderStyle.Render(hunk.Header))

		// Build left and right columns
		leftLines := []string{}
		rightLines := []string{}

		for _, dl := range hunk.Lines {
			switch dl.Type {
			case "removed":
				leftLines = append(leftLines, d.renderSplitLine(dl, "removed", halfWidth))
				rightLines = append(rightLines, d.renderEmptySplitLine(halfWidth))
			case "added":
				leftLines = append(leftLines, d.renderEmptySplitLine(halfWidth))
				rightLines = append(rightLines, d.renderSplitLine(dl, "added", halfWidth))
			case "context":
				leftLines = append(leftLines, d.renderSplitLine(dl, "context-left", halfWidth))
				rightLines = append(rightLines, d.renderSplitLine(dl, "context-right", halfWidth))
			}
		}

		// Join left and right columns side by side
		separatorStyle := lipgloss.NewStyle().
			Foreground(d.theme.Border)
		separator := separatorStyle.Render(" │ ")

		maxLines := max(len(leftLines), len(rightLines))
		for i := 0; i < maxLines; i++ {
			left := ""
			right := ""
			if i < len(leftLines) {
				left = leftLines[i]
			}
			if i < len(rightLines) {
				right = rightLines[i]
			}
			lines = append(lines, left+separator+right)
		}
	}

	return strings.Join(lines, "\n")
}

// renderSplitLine renders a line for split view
func (d *DiffRenderer) renderSplitLine(dl DiffLine, side string, width int) string {
	var lineStyle lipgloss.Style
	var lineNum int

	switch side {
	case "removed":
		lineStyle = lipgloss.NewStyle().
			Foreground(d.theme.Removed).
			Background(d.theme.RemovedBg)
		lineNum = dl.OldLine
	case "added":
		lineStyle = lipgloss.NewStyle().
			Foreground(d.theme.Added).
			Background(d.theme.AddedBg)
		lineNum = dl.NewLine
	case "context-left":
		lineStyle = lipgloss.NewStyle().
			Foreground(d.theme.TextMuted)
		lineNum = dl.OldLine
	case "context-right":
		lineStyle = lipgloss.NewStyle().
			Foreground(d.theme.TextMuted)
		lineNum = dl.NewLine
	}

	// Line number
	lineNumStr := fmt.Sprintf("%4d", lineNum)
	lineNumStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted).
		Width(4).
		Align(lipgloss.Right)
	lineNumPart := lineNumStyle.Render(lineNumStr)

	// Content (wrapped if needed)
	content := dl.Content
	if d.wrapLines && util.StringWidth(content) > width {
		wrapped := util.Wrap(content, width)
		content = strings.Join(wrapped, "")
	}

	// Truncate if too long (for split view we need to maintain column width)
	if util.StringWidth(content) > width {
		content = util.Truncate(content, width, "…")
	}

	contentPart := lineStyle.Render(content)

	// Pad to width
	padding := width - util.StringWidth(content)
	if padding > 0 {
		contentPart = contentPart + strings.Repeat(" ", padding)
	}

	return lineNumPart + " " + contentPart
}

// renderEmptySplitLine renders an empty line in split view
func (d *DiffRenderer) renderEmptySplitLine(width int) string {
	emptyNumStyle := lipgloss.NewStyle().
		Foreground(d.theme.TextMuted).
		Width(4).
		Align(lipgloss.Right)
	emptyNum := emptyNumStyle.Render("    ") // 4 spaces for empty line number

	emptyContent := strings.Repeat(" ", width+1)

	return emptyNum + emptyContent
}

// RenderUnifiedSimple renders a simple unified diff (for backwards compatibility)
// This is used by tool_result.go
func (d *DiffRenderer) RenderUnifiedSimple(diff string) string {
	return d.renderSimpleDiff(diff)
}

// renderSimpleDiff renders a basic unified diff (used by tool_result.go)
func (d *DiffRenderer) renderSimpleDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var result []string

	// Calculate available width
	contentWidth := d.width - 5 // Reserve space for +/- prefix
	if contentWidth < 40 {
		contentWidth = 40
	}

	for _, line := range lines {
		if line == "" {
			result = append(result, "")
			continue
		}

		switch {
		case strings.HasPrefix(line, "+"):
			// Added line
			content := line
			if d.wrapLines && util.StringWidth(content) > contentWidth {
				wrapped := util.WrapPreserveNewlines(content, contentWidth)
				content = strings.Join(wrapped, "\n")
			}
			addedStyle := lipgloss.NewStyle().
				Foreground(d.theme.Added).
				Background(d.theme.AddedBg)
			result = append(result, addedStyle.Render(content))
		case strings.HasPrefix(line, "-"):
			// Removed line
			content := line
			if d.wrapLines && util.StringWidth(content) > contentWidth {
				wrapped := util.WrapPreserveNewlines(content, contentWidth)
				content = strings.Join(wrapped, "\n")
			}
			removedStyle := lipgloss.NewStyle().
				Foreground(d.theme.Removed).
				Background(d.theme.RemovedBg)
			result = append(result, removedStyle.Render(content))
		case strings.HasPrefix(line, "@@"):
			// Diff header
			headerStyle := lipgloss.NewStyle().
				Foreground(d.theme.Info)
			result = append(result, headerStyle.Render(line))
		default:
			// Context line
			contextStyle := lipgloss.NewStyle().
				Foreground(d.theme.TextMuted)
			result = append(result, contextStyle.Render(line))
		}
	}

	return strings.Join(result, "\n")
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
