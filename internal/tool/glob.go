// Package tool provides the Glob tool implementation.
package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxGlobResults = 100
)

// GlobTool finds files by glob pattern
type GlobTool struct{}

// NewGlobTool creates a new Glob tool
func NewGlobTool() *GlobTool {
	return &GlobTool{}
}

// ID returns the tool ID
func (t *GlobTool) ID() ToolID {
	return ToolGlob
}

// Definition returns the tool definition
func (t *GlobTool) Definition() ToolDefinition {
	return ToolDefinition{
		ID:          "glob",
		Description: "Finds files by glob pattern. Returns matching file paths sorted by modification time. Use this tool when you need to find files by name patterns. Supports glob patterns like '**/*.js' or 'src/**/*.ts'. IMPORTANT: Use the Glob tool instead of Bash find command.",
		Parameters: map[string]ParameterSchema{
			"pattern": {
				Type:        "string",
				Description: "The glob pattern to match files against (e.g., '**/*.js', 'src/**/*.ts')",
				Required:    true,
			},
			"path": {
				Type:        "string",
				Description: "The directory to search in (defaults to current directory)",
				Required:    false,
			},
		},
	}
}

// Validate checks parameters
func (t *GlobTool) Validate(params map[string]interface{}) error {
	pattern, ok := params["pattern"].(string)
	if !ok || pattern == "" {
		return NewValidationError(ToolGlob, "pattern is required and must be a string")
	}

	return nil
}

// Execute finds files matching the pattern
func (t *GlobTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
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

	// Find matching files
	matches := globSearch(pattern, searchPath, maxGlobResults)

	// Sort by modification time (newest first)
	sortByModTime(matches)

	// Build output
	title := fmt.Sprintf("Glob: %s", pattern)
	output := formatGlobResults(matches, searchPath)

	return &ToolResult{
		Title:  title,
		Output: output,
		Metadata: Metadata{
			RowsAffected: len(matches),
		},
	}, nil
}

// globSearch performs the search
func globSearch(pattern, rootPath string, maxResults int) []string {
	matches := []string{}

	// Handle ** patterns
	fullPattern := filepath.Join(rootPath, pattern)

	// Use filepath.Glob for simple patterns
	if !strings.Contains(pattern, "**") {
		m, err := filepath.Glob(fullPattern)
		if err == nil {
			matches = m
		}
	} else {
		// For ** patterns, we need a custom implementation
		matches = globDoubleStar(rootPath, pattern, maxResults)
	}

	// Limit results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches
}

// globDoubleStar handles ** patterns by walking the directory
func globDoubleStar(rootPath, pattern string, maxResults int) []string {
	matches := []string{}

	// Split pattern to find the ** part
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		// Fallback to simple glob
		m, err := filepath.Glob(filepath.Join(rootPath, pattern))
		if err == nil {
			return m
		}
		return matches
	}

	before := parts[0]
	after := parts[1]

	// Walk all directories
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Check if this directory matches the before pattern
			if before != "" {
				relDir, err := filepath.Rel(rootPath, path)
				if err != nil {
					return nil
				}
				// Check prefix match
				if !strings.HasPrefix(relDir, strings.TrimSuffix(before, "/")) {
					return nil
				}
			}

			// Check files in this directory for the after pattern
			if after != "" {
				entries, err := os.ReadDir(path)
				if err != nil {
					return nil
				}
				for _, entry := range entries {
					name := entry.Name()
					matched, err := filepath.Match(strings.TrimPrefix(after, "/"), name)
					if err == nil && matched {
						fullPath := filepath.Join(path, name)
						matches = append(matches, fullPath)
						if len(matches) >= maxResults {
							return fmt.Errorf("max results")
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil && err.Error() != "max results" {
		// Ignore errors
	}

	return matches
}

// sortByModTime sorts files by modification time (newest first)
func sortByModTime(files []string) {
	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return files[i] < files[j] // alphabetical fallback
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})
}

// formatGlobResults formats the results
func formatGlobResults(matches []string, rootPath string) string {
	if len(matches) == 0 {
		return "No files found matching pattern"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d files:", len(matches)))

	for _, m := range matches {
		// Make path relative for display
		rel, err := filepath.Rel(rootPath, m)
		if err != nil {
			rel = m
		}

		// Get file info for size
		info, err := os.Stat(m)
		if err == nil {
			lines = append(lines, fmt.Sprintf("%s (%d bytes, modified: %s)",
				rel, info.Size(), info.ModTime().Format(time.RFC3339)))
		} else {
			lines = append(lines, rel)
		}
	}

	return strings.Join(lines, "\n")
}