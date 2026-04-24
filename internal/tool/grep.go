// Package tool provides the Grep tool implementation.
package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxGrepResults = 100
)

// GrepTool searches file contents using regex
type GrepTool struct{}

// NewGrepTool creates a new Grep tool
func NewGrepTool() *GrepTool {
	return &GrepTool{}
}

// ID returns the tool ID
func (t *GrepTool) ID() ToolID {
	return ToolGrep
}

// Definition returns the tool definition
func (t *GrepTool) Definition() ToolDefinition {
	return ToolDefinition{
		ID:          "grep",
		Description: "Searches file contents using regular expressions. Supports full regex syntax. Returns file paths and line numbers with matches. Use this tool when you need to find files containing specific patterns. IMPORTANT: Avoid using Bash with grep, use this specialized Grep tool instead.",
		Parameters: map[string]ParameterSchema{
			"pattern": {
				Type:        "string",
				Description: "The regex pattern to search for",
				Required:    true,
			},
			"path": {
				Type:        "string",
				Description: "The directory to search in (defaults to current directory)",
				Required:    false,
			},
			"include": {
				Type:        "string",
				Description: "File pattern to include (e.g., '*.js', '*.ts')",
				Required:    false,
			},
			"output_mode": {
				Type:        "string",
				Description: "Output mode: 'files_with_matches' (default), 'content', or 'count'",
				Required:    false,
				Default:     "files_with_matches",
				Enum:        []string{"files_with_matches", "content", "count"},
			},
		},
	}
}

// Validate checks parameters
func (t *GrepTool) Validate(params map[string]interface{}) error {
	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return NewValidationError(ToolGrep, "pattern is required and must be a string")
	}

	// Validate regex pattern
	_, err := regexp.Compile(pattern)
	if err != nil {
		return NewValidationError(ToolGrep, fmt.Sprintf("invalid regex pattern: %v", err))
	}

	// Validate output_mode
	if mode, ok := params["output_mode"].(string); ok {
		validModes := []string{"files_with_matches", "content", "count"}
		if !contains(validModes, mode) {
			return NewValidationError(ToolGrep, fmt.Sprintf("invalid output_mode: %s (valid: %v)", mode, validModes))
		}
	}

	return nil
}

// Execute searches for the pattern
func (t *GrepTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
	pattern := params["pattern"].(string)

	// Get search path
	searchPath := toolCtx.WorkDir
	if p, ok := params["path"].(string); ok && p != "" {
		if filepath.IsAbs(p) {
			searchPath = p
		} else if toolCtx.WorkDir != "" {
			searchPath = filepath.Join(toolCtx.WorkDir, p)
		} else {
			searchPath = p
		}
	}

	// Get include pattern
	includePattern := ""
	if i, ok := params["include"].(string); ok {
		includePattern = i
	}

	// Get output mode
	outputMode := "files_with_matches"
	if m, ok := params["output_mode"].(string); ok {
		outputMode = m
	}

	// Compile regex
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, NewExecutionError(ToolGrep, err.Error())
	}

	// Search files
	results := grepSearch(re, searchPath, includePattern, outputMode, maxGrepResults)

	// Build output
	title := fmt.Sprintf("Grep: %s", truncatePattern(pattern, 30))
	output := formatGrepResults(results, outputMode)

	return &ToolResult{
		Title:  title,
		Output: output,
		Metadata: Metadata{
			RowsAffected: len(results),
		},
	}, nil
}

// grepResult represents a grep match
type grepResult struct {
	filePath string
	lineNum  int
	content  string
}

// grepSearch performs the search
func grepSearch(re *regexp.Regexp, rootPath, includePattern, outputMode string, maxResults int) []grepResult {
	results := []grepResult{}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Skip hidden directories and common exclude patterns
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check include pattern
		if includePattern != "" {
			matched, err := filepath.Match(includePattern, info.Name())
			if err != nil || !matched {
				return nil
			}
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Search for matches
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				results = append(results, grepResult{
					filePath: path,
					lineNum:  i + 1,
					content:  strings.TrimSpace(line),
				})

				// Limit results
				if outputMode != "count" && len(results) >= maxResults {
					return fmt.Errorf("max results reached") // stop walking
				}
			}
		}

		return nil
	})

	if err != nil && err.Error() != "max results reached" {
		// Log error but continue
	}

	return results
}

// formatGrepResults formats the results
func formatGrepResults(results []grepResult, outputMode string) string {
	if len(results) == 0 {
		return "No matches found"
	}

	switch outputMode {
	case "files_with_matches":
		files := make(map[string]bool)
		for _, r := range results {
			files[r.filePath] = true
		}
		var lines []string
		for f := range files {
			lines = append(lines, f)
		}
		return strings.Join(lines, "\n")

	case "content":
		var lines []string
		for _, r := range results {
			// Make path relative to display
			lines = append(lines, fmt.Sprintf("%s:%d: %s", r.filePath, r.lineNum, r.content))
		}
		return strings.Join(lines, "\n")

	case "count":
		counts := make(map[string]int)
		for _, r := range results {
			counts[r.filePath]++
		}
		var lines []string
		for f, c := range counts {
			lines = append(lines, fmt.Sprintf("%s: %d matches", f, c))
		}
		return strings.Join(lines, "\n")

	default:
		return fmt.Sprintf("%d matches found", len(results))
	}
}

// truncatePattern truncates a pattern for display
func truncatePattern(pattern string, maxLen int) string {
	if len(pattern) <= maxLen {
		return pattern
	}
	return pattern[:maxLen] + "..."
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}