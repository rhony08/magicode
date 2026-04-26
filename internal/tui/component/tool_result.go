// Package component provides reusable UI components for the TUI.
// This file implements tool result rendering for various tool types.
package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// ToolResultRenderer renders tool results with appropriate formatting
type ToolResultRenderer struct {
	theme  types.Theme
	styles types.ThemeStyles
}

// NewToolResultRenderer creates a new tool result renderer
func NewToolResultRenderer(theme types.Theme, styles types.ThemeStyles) *ToolResultRenderer {
	return &ToolResultRenderer{
		theme:  theme,
		styles: styles,
	}
}

// RenderResult renders a tool result based on the tool name and output
func (r *ToolResultRenderer) RenderResult(toolName string, toolInput string, output string, status string) string {
	switch toolName {
	case "bash":
		return r.renderBash(toolInput, output, status)
	case "read":
		return r.renderRead(toolInput, output, status)
	case "write":
		return r.renderWrite(toolInput, output, status)
	case "edit":
		return r.renderEdit(toolInput, output, status)
	case "glob":
		return r.renderGlob(toolInput, output, status)
	case "grep":
		return r.renderGrep(toolInput, output, status)
	case "todo":
		return r.renderTodo(toolInput, output, status)
	default:
		return r.renderGeneric(toolName, toolInput, output, status)
	}
}

// renderBash renders a bash command result
func (r *ToolResultRenderer) renderBash(command string, output string, status string) string {
	var lines []string

	// Command header
	cmdStyle := lipgloss.NewStyle().
		Foreground(r.theme.Primary).
		Bold(true)
	lines = append(lines, cmdStyle.Render("▶ bash"))

	// Command (truncated if too long)
	maxCmdLen := 100
	if len(command) > maxCmdLen {
		command = command[:maxCmdLen] + "..."
	}
	cmdTextStyle := lipgloss.NewStyle().
		Foreground(r.theme.Text).
		Background(r.theme.ElementBg).
		Padding(0, 1)
	lines = append(lines, cmdTextStyle.Render(command))

	// Output
	if output != "" {
		lines = append(lines, "")

		// Truncate long output
		maxLines := 50
		maxChars := 5000
		output = r.truncateOutput(output, maxLines, maxChars)

		outputStyle := lipgloss.NewStyle().
			Foreground(r.theme.Text)
		lines = append(lines, outputStyle.Render(output))
	}

	// Status indicator
	statusLine := r.renderStatus(status)
	if statusLine != "" {
		lines = append(lines, "")
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

// renderRead renders a read file result
func (r *ToolResultRenderer) renderRead(filePath string, output string, status string) string {
	var lines []string

	// File header
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Primary).
		Bold(true)
	lines = append(lines, headerStyle.Render("📄 read"))

	// File path
	pathStyle := lipgloss.NewStyle().
		Foreground(r.theme.TextMuted)
	lines = append(lines, pathStyle.Render(filePath))

	// Content
	if output != "" {
		lines = append(lines, "")

		// Check if it's a directory listing
		if strings.Contains(output, "entries)") {
			lines = append(lines, r.renderDirectoryListing(output))
		} else {
			// File content with line numbers
			lines = append(lines, r.renderFileContent(output, filePath))
		}
	}

	// Status
	statusLine := r.renderStatus(status)
	if statusLine != "" {
		lines = append(lines, "")
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

// renderWrite renders a write file result
func (r *ToolResultRenderer) renderWrite(filePath string, output string, status string) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Success).
		Bold(true)
	lines = append(lines, headerStyle.Render("✎ write"))

	// File path
	pathStyle := lipgloss.NewStyle().
		Foreground(r.theme.TextMuted)
	lines = append(lines, pathStyle.Render(filePath))

	// Content preview
	if output != "" {
		lines = append(lines, "")
		lines = append(lines, r.renderContentPreview(output, "new"))
	}

	// Status
	statusLine := r.renderStatus(status)
	if statusLine != "" {
		lines = append(lines, "")
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

// renderEdit renders an edit file result with diff
func (r *ToolResultRenderer) renderEdit(filePath string, output string, status string) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Warning).
		Bold(true)
	lines = append(lines, headerStyle.Render("✎ edit"))

	// File path
	pathStyle := lipgloss.NewStyle().
		Foreground(r.theme.TextMuted)
	lines = append(lines, pathStyle.Render(filePath))

	// Diff content - use the new diff renderer
	if output != "" {
		lines = append(lines, "")
		diffRenderer := NewDiffRenderer(r.theme, r.styles)
		diffRenderer.SetWidth(80) // Default width
		diffRenderer.SetWrapLines(true)
		// Use simple diff for backwards compatibility with existing tool output
		lines = append(lines, diffRenderer.RenderUnifiedSimple(output))
	}

	// Status
	statusLine := r.renderStatus(status)
	if statusLine != "" {
		lines = append(lines, "")
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

// renderGlob renders a glob result (file list)
func (r *ToolResultRenderer) renderGlob(pattern string, output string, status string) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Info).
		Bold(true)
	lines = append(lines, headerStyle.Render("📁 glob"))

	// Pattern
	patternStyle := lipgloss.NewStyle().
		Foreground(r.theme.TextMuted)
	lines = append(lines, patternStyle.Render(pattern))

	// File list
	if output != "" {
		lines = append(lines, "")

		files := strings.Split(output, "\n")
		fileCount := len(files)

		// Limit display to 20 files
		maxDisplay := 20
		displayFiles := files
		if len(files) > maxDisplay {
			displayFiles = files[:maxDisplay]
		}

		for _, file := range displayFiles {
			if file == "" {
				continue
			}
			fileStyle := lipgloss.NewStyle().
				Foreground(r.theme.Text)
			lines = append(lines, fileStyle.Render("  • "+file))
		}

		if len(files) > maxDisplay {
			moreStyle := lipgloss.NewStyle().
				Foreground(r.theme.TextMuted)
			lines = append(lines, moreStyle.Render(fmt.Sprintf("  ... and %d more files", len(files)-maxDisplay)))
		}

		lines = append(lines, "")
		countStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted)
		lines = append(lines, countStyle.Render(fmt.Sprintf("(%d files)", fileCount)))
	}

	// Status
	statusLine := r.renderStatus(status)
	if statusLine != "" {
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

// renderGrep renders a grep result (search matches)
func (r *ToolResultRenderer) renderGrep(pattern string, output string, status string) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Info).
		Bold(true)
	lines = append(lines, headerStyle.Render("🔍 grep"))

	// Pattern
	patternStyle := lipgloss.NewStyle().
		Foreground(r.theme.TextMuted)
	lines = append(lines, patternStyle.Render(pattern))

	// Matches
	if output != "" {
		lines = append(lines, "")

		matches := strings.Split(output, "\n")
		matchCount := 0

		// Limit display to 30 matches
		maxDisplay := 30
		for i, match := range matches {
			if match == "" {
				continue
			}
			if i >= maxDisplay {
				break
			}
			matchCount++

			// Highlight the matched pattern
			highlighted := r.highlightPattern(match, pattern)
			matchStyle := lipgloss.NewStyle().
				Foreground(r.theme.Text)
			lines = append(lines, matchStyle.Render("  "+highlighted))
		}

		if len(matches) > maxDisplay {
			moreStyle := lipgloss.NewStyle().
				Foreground(r.theme.TextMuted)
			lines = append(lines, moreStyle.Render(fmt.Sprintf("  ... and %d more matches", len(matches)-maxDisplay)))
		}

		lines = append(lines, "")
		countStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted)
		lines = append(lines, countStyle.Render(fmt.Sprintf("(%d matches)", len(matches))))
	}

	// Status
	statusLine := r.renderStatus(status)
	if statusLine != "" {
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

// renderTodo renders a todo result
func (r *ToolResultRenderer) renderTodo(input string, output string, status string) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Accent).
		Bold(true)
	lines = append(lines, headerStyle.Render("☑ todo"))

	// Action description
	if input != "" {
		actionStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted)
		lines = append(lines, actionStyle.Render(input))
	}

	// Output
	if output != "" {
		lines = append(lines, "")
		outputStyle := lipgloss.NewStyle().
			Foreground(r.theme.Text)
		lines = append(lines, outputStyle.Render(output))
	}

	// Status
	statusLine := r.renderStatus(status)
	if statusLine != "" {
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

// renderGeneric renders a generic tool result
func (r *ToolResultRenderer) renderGeneric(toolName string, toolInput string, output string, status string) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Primary).
		Bold(true)
	lines = append(lines, headerStyle.Render("▶ "+toolName))

	// Input
	if toolInput != "" {
		inputStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted)
		if len(toolInput) > 100 {
			toolInput = toolInput[:100] + "..."
		}
		lines = append(lines, inputStyle.Render(toolInput))
	}

	// Output
	if output != "" {
		lines = append(lines, "")
		output = r.truncateOutput(output, 30, 3000)
		outputStyle := lipgloss.NewStyle().
			Foreground(r.theme.Text)
		lines = append(lines, outputStyle.Render(output))
	}

	// Status
	statusLine := r.renderStatus(status)
	if statusLine != "" {
		lines = append(lines, "")
		lines = append(lines, statusLine)
	}

	return strings.Join(lines, "\n")
}

// renderStatus renders the status line
func (r *ToolResultRenderer) renderStatus(status string) string {
	switch status {
	case "success":
		return lipgloss.NewStyle().
			Foreground(r.theme.Success).
			Render("✓ Success")
	case "error":
		return lipgloss.NewStyle().
			Foreground(r.theme.Error).
			Render("✗ Error")
	case "pending":
		return lipgloss.NewStyle().
			Foreground(r.theme.Warning).
			Render("⏳ Pending")
	case "running":
		return lipgloss.NewStyle().
			Foreground(r.theme.Info).
			Render("▶ Running")
	default:
		return ""
	}
}

// renderDirectoryListing renders a directory listing
func (r *ToolResultRenderer) renderDirectoryListing(output string) string {
	var lines []string

	// Parse the entries
	// Format: entries are listed with / for directories
	entries := strings.Split(output, "\n")
	for _, entry := range entries {
		if entry == "" || strings.Contains(entry, "entries") {
			continue
		}

		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}

		// Check if it's a directory (ends with /)
		if strings.HasSuffix(trimmed, "/") {
			dirStyle := lipgloss.NewStyle().
				Foreground(r.theme.Secondary).
				Bold(true)
			lines = append(lines, dirStyle.Render("📁 "+strings.TrimSuffix(trimmed, "/")))
		} else {
			fileStyle := lipgloss.NewStyle().
				Foreground(r.theme.Text)
			lines = append(lines, fileStyle.Render("📄 "+trimmed))
		}
	}

	return strings.Join(lines, "\n")
}

// renderFileContent renders file content with optional line numbers
func (r *ToolResultRenderer) renderFileContent(content string, filePath string) string {
	lines := strings.Split(content, "\n")
	var result []string

	// Limit to 100 lines for display
	maxLines := 100
	displayLines := lines
	truncated := false
	if len(lines) > maxLines {
		displayLines = lines[:maxLines]
		truncated = true
	}

	// Add line numbers
	for i, line := range displayLines {
		lineNum := i + 1
		numStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted).
			Width(4).
			Align(lipgloss.Right)
		contentStyle := lipgloss.NewStyle().
			Foreground(r.theme.Text)

		num := numStyle.Render(fmt.Sprintf("%d", lineNum))
		content := contentStyle.Render(" " + line)
		result = append(result, num+content)
	}

	if truncated {
		moreStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted)
		result = append(result, moreStyle.Render(fmt.Sprintf("  ... (%d more lines)", len(lines)-maxLines)))
	}

	return strings.Join(result, "\n")
}

// renderContentPreview renders a content preview
func (r *ToolResultRenderer) renderContentPreview(content string, label string) string {
	// Limit preview
	maxLines := 20
	lines := strings.Split(content, "\n")

	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	var result []string
	for _, line := range lines {
		lineStyle := lipgloss.NewStyle().
			Foreground(r.theme.Text)
		result = append(result, lineStyle.Render(line))
	}

	if len(strings.Split(content, "\n")) > maxLines {
		moreStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted)
		result = append(result, moreStyle.Render("  ..."))
	}

	return strings.Join(result, "\n")
}

// renderDiff renders a diff view with added/removed lines
func (r *ToolResultRenderer) renderDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var result []string

	for _, line := range lines {
		if line == "" {
			result = append(result, "")
			continue
		}

		switch {
		case strings.HasPrefix(line, "+"):
			// Added line
			addedStyle := lipgloss.NewStyle().
				Foreground(r.theme.Added).
				Background(r.theme.AddedBg)
			result = append(result, addedStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			// Removed line
			removedStyle := lipgloss.NewStyle().
				Foreground(r.theme.Removed).
				Background(r.theme.RemovedBg)
			result = append(result, removedStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			// Diff header
			headerStyle := lipgloss.NewStyle().
				Foreground(r.theme.Info)
			result = append(result, headerStyle.Render(line))
		default:
			// Context line
			contextStyle := lipgloss.NewStyle().
				Foreground(r.theme.TextMuted)
			result = append(result, contextStyle.Render(line))
		}
	}

	return strings.Join(result, "\n")
}

// truncateOutput truncates output to reasonable limits
func (r *ToolResultRenderer) truncateOutput(output string, maxLines int, maxChars int) string {
	if len(output) > maxChars {
		output = output[:maxChars] + "\n\n[...output truncated...]"
	}

	lines := strings.Split(output, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		output = strings.Join(lines, "\n") + "\n\n[...output truncated...]"
	}

	return output
}

// highlightPattern highlights the search pattern in the text
func (r *ToolResultRenderer) highlightPattern(text string, pattern string) string {
	// Simple highlighting - just bold the pattern if found
	if pattern == "" {
		return text
	}

	// Escape special characters for simple matching
	lowerText := strings.ToLower(text)
	lowerPattern := strings.ToLower(pattern)

	idx := strings.Index(lowerText, lowerPattern)
	if idx == -1 {
		return text
	}

	before := text[:idx]
	match := text[idx : idx+len(pattern)]
	after := text[idx+len(pattern):]

	highlightStyle := lipgloss.NewStyle().
		Foreground(r.theme.Accent).
		Bold(true)

	return before + highlightStyle.Render(match) + after
}

// RenderToolCall renders a tool call (the request to execute a tool)
func (r *ToolResultRenderer) RenderToolCall(toolName string, toolInput string) string {
	var lines []string

	// Tool name
	nameStyle := lipgloss.NewStyle().
		Foreground(r.theme.Primary).
		Bold(true)
	lines = append(lines, nameStyle.Render(fmt.Sprintf("▶ %s", toolName)))

	// Input (truncated)
	if toolInput != "" {
		input := toolInput
		if len(input) > 150 {
			input = input[:150] + "..."
		}
		inputStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted)
		lines = append(lines, inputStyle.Render(input))
	}

	return strings.Join(lines, "\n")
}
