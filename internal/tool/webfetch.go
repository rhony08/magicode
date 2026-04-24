// Package tool provides the WebFetch tool implementation.
package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	webFetchTimeout = 30 * time.Second
	maxWebResponseSize = 51200 // 50KB
)

// WebFetchTool fetches content from URLs
type WebFetchTool struct{}

// NewWebFetchTool creates a new WebFetch tool
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{}
}

// ID returns the tool ID
func (t *WebFetchTool) ID() ToolID {
	return ToolWebFetch
}

// Definition returns the tool definition
func (t *WebFetchTool) Definition() ToolDefinition {
	return ToolDefinition{
		ID:          "webfetch",
		Description: "Fetches content from a specified URL. Takes a URL and optional format as input. Fetches the URL content, converts to requested format (markdown by default). Use this tool when you need to retrieve and analyze web content. IMPORTANT: Prefer specialized tools over webfetch.",
		Parameters: map[string]ParameterSchema{
			"url": {
				Type:        "string",
				Description: "The URL to fetch content from",
				Required:    true,
			},
			"format": {
				Type:        "string",
				Description: "The format to return content in: 'text', 'markdown', or 'html'",
				Required:    false,
				Default:     "markdown",
				Enum:        []string{"text", "markdown", "html"},
			},
			"timeout": {
				Type:        "number",
				Description: "Optional timeout in seconds (max 120)",
				Required:    false,
				Default:     30,
			},
		},
	}
}

// Validate checks parameters
func (t *WebFetchTool) Validate(params map[string]interface{}) error {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return NewValidationError(ToolWebFetch, "url is required and must be a string")
	}

	// Validate URL format
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return NewValidationError(ToolWebFetch, "url must start with http:// or https://")
	}

	// Validate format
	if format, ok := params["format"].(string); ok {
		validFormats := []string{"text", "markdown", "html"}
		if !contains(validFormats, format) {
			return NewValidationError(ToolWebFetch, fmt.Sprintf("invalid format: %s (valid: %v)", format, validFormats))
		}
	}

	// Validate timeout
	if timeout, ok := params["timeout"]; ok {
		switch v := timeout.(type) {
		case int:
			if v < 1 || v > 120 {
				return NewValidationError(ToolWebFetch, "timeout must be between 1 and 120 seconds")
			}
		case float64:
			if v < 1 || v > 120 {
				return NewValidationError(ToolWebFetch, "timeout must be between 1 and 120 seconds")
			}
		}
	}

	return nil
}

// Execute fetches the URL
func (t *WebFetchTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
	url := params["url"].(string)

	// Get format
	format := "markdown"
	if f, ok := params["format"].(string); ok {
		format = f
	}

	// Get timeout
	timeout := webFetchTimeout
	if t, ok := params["timeout"]; ok {
		switch v := t.(type) {
		case int:
			timeout = time.Duration(v) * time.Second
		case float64:
			timeout = time.Duration(int(v)) * time.Second
		}
	}

	// Create client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, NewExecutionError(ToolWebFetch, err.Error())
	}

	// Set headers
	req.Header.Set("User-Agent", "OpenCode/1.0")
	req.Header.Set("Accept", "text/html, application/json, text/plain, */*")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, NewExecutionError(ToolWebFetch, fmt.Sprintf("failed to fetch URL: %v", err))
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, NewExecutionError(ToolWebFetch, fmt.Sprintf("HTTP error: %d %s", resp.StatusCode, resp.Status))
	}

	// Read content
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxWebResponseSize))
	if err != nil {
		return nil, NewExecutionError(ToolWebFetch, err.Error())
	}

	// Convert format
	content := convertFormat(string(body), format)

	// Build title
	title := fmt.Sprintf("Fetch: %s", truncateURL(url, 50))

	// Build output
	output := fmt.Sprintf("URL: %s", url)
	output += fmt.Sprintf("\nStatus: %d %s", resp.StatusCode, resp.Status)
	output += fmt.Sprintf("\nContent-Type: %s", resp.Header.Get("Content-Type"))
	output += fmt.Sprintf("\n\n%s", content)

	return &ToolResult{
		Title:  title,
		Output: output,
		Metadata: Metadata{
			RowsAffected: len(body),
		},
	}, nil
}

// convertFormat converts content to the requested format
func convertFormat(content, format string) string {
	switch format {
	case "html":
		return content
	case "markdown":
		// Simple HTML to markdown conversion
		return htmlToMarkdown(content)
	case "text":
		// Strip HTML tags for plain text
		return stripHTMLTags(content)
	default:
		return content
	}
}

// htmlToMarkdown does a simple HTML to markdown conversion
func htmlToMarkdown(html string) string {
	// Simple conversions
	result := html

	// Headers
	result = strings.ReplaceAll(result, "<h1>", "# ")
	result = strings.ReplaceAll(result, "</h1>", "\n")
	result = strings.ReplaceAll(result, "<h2>", "## ")
	result = strings.ReplaceAll(result, "</h2>", "\n")
	result = strings.ReplaceAll(result, "<h3>", "### ")
	result = strings.ReplaceAll(result, "</h3>", "\n")

	// Bold/Italic
	result = strings.ReplaceAll(result, "<strong>", "**")
	result = strings.ReplaceAll(result, "</strong>", "**")
	result = strings.ReplaceAll(result, "<b>", "**")
	result = strings.ReplaceAll(result, "</b>", "**")
	result = strings.ReplaceAll(result, "<em>", "*")
	result = strings.ReplaceAll(result, "</em>", "*")
	result = strings.ReplaceAll(result, "<i>", "*")
	result = strings.ReplaceAll(result, "</i>", "*")

	// Links
	// Simple pattern: <a href="url">text</a> -> [text](url)
	// This is a simplified version

	// Paragraphs
	result = strings.ReplaceAll(result, "<p>", "")
	result = strings.ReplaceAll(result, "</p>", "\n\n")

	// Line breaks
	result = strings.ReplaceAll(result, "<br>", "\n")
	result = strings.ReplaceAll(result, "<br/>", "\n")
	result = strings.ReplaceAll(result, "<br />", "\n")

	// Lists
	result = strings.ReplaceAll(result, "<li>", "- ")
	result = strings.ReplaceAll(result, "</li>", "\n")

	// Code
	result = strings.ReplaceAll(result, "<code>", "`")
	result = strings.ReplaceAll(result, "</code>", "`")
	result = strings.ReplaceAll(result, "<pre>", "\n```\n")
	result = strings.ReplaceAll(result, "</pre>", "\n```\n")

	// Clean up remaining tags
	result = stripHTMLTags(result)

	return result
}

// stripHTMLTags removes HTML tags
func stripHTMLTags(html string) string {
	// Simple tag stripping (not perfect but works for basic cases)
	result := html
	inTag := false
	var cleaned strings.Builder

	for _, char := range result {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			cleaned.WriteRune(char)
		}
	}

	return cleaned.String()
}

// truncateURL truncates a URL for display
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen] + "..."
}