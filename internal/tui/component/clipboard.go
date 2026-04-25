// Package component provides reusable UI components for the TUI.
// This file implements clipboard operations for cross-platform support.
package component

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Clipboard provides cross-platform clipboard operations
type Clipboard struct {
	// Platform-specific copy command
	copyCmd []string

	// Platform-specific paste command
	pasteCmd []string
}

// NewClipboard creates a new clipboard instance
func NewClipboard() *Clipboard {
	c := &Clipboard{}
	c.detectCommands()
	return c
}

// detectCommands detects the appropriate clipboard commands for the platform
func (c *Clipboard) detectCommands() {
	switch runtime.GOOS {
	case "darwin":
		// macOS uses pbcopy/pbpaste
		c.copyCmd = []string{"pbcopy"}
		c.pasteCmd = []string{"pbpaste"}

	case "windows":
		// Windows uses clip for copy, PowerShell for paste
		c.copyCmd = []string{"clip"}
		c.pasteCmd = []string{"powershell", "-command", "Get-Clipboard"}

	default:
		// Linux - try various clipboard tools
		// Prefer xclip, then xsel, then wl-copy (Wayland)
		if c.commandExists("xclip") {
			c.copyCmd = []string{"xclip", "-selection", "clipboard", "-in"}
			c.pasteCmd = []string{"xclip", "-selection", "clipboard", "-out"}
		} else if c.commandExists("xsel") {
			c.copyCmd = []string{"xsel", "--clipboard", "--input"}
			c.pasteCmd = []string{"xsel", "--clipboard", "--output"}
		} else if c.commandExists("wl-copy") {
			c.copyCmd = []string{"wl-copy"}
			c.pasteCmd = []string{"wl-paste"}
		}
	}
}

// commandExists checks if a command exists in PATH
func (c *Clipboard) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// IsAvailable returns true if clipboard operations are available
func (c *Clipboard) IsAvailable() bool {
	return c.copyCmd != nil || c.pasteCmd != nil
}

// Copy copies text to the clipboard
func (c *Clipboard) Copy(text string) error {
	if c.copyCmd == nil {
		return fmt.Errorf("clipboard copy not available on this platform")
	}

	cmd := exec.Command(c.copyCmd[0], c.copyCmd[1:]...)
	cmd.Stdin = strings.NewReader(text)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	return nil
}

// Paste reads text from the clipboard
func (c *Clipboard) Paste() (string, error) {
	if c.pasteCmd == nil {
		return "", fmt.Errorf("clipboard paste not available on this platform")
	}

	cmd := exec.Command(c.pasteCmd[0], c.pasteCmd[1:]...)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to paste from clipboard: %w", err)
	}

	return string(output), nil
}

// CopyFromFile copies content from a file to the clipboard
func (c *Clipboard) CopyFromFile(filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	return c.Copy(string(content))
}

// CopyCodeBlock copies a code block with proper formatting
func (c *Clipboard) CopyCodeBlock(code, language string) error {
	// Add code fence if language is provided
	var content string
	if language != "" {
		content = fmt.Sprintf("```%s\n%s\n```", language, code)
	} else {
		content = fmt.Sprintf("```\n%s\n```", code)
	}

	return c.Copy(content)
}

// GetPlatform returns the current platform
func (m *MockClipboard) GetPlatform() string {
	return "mock"
}

// GetCopyCommand returns empty for mock
func (m *MockClipboard) GetCopyCommand() string {
	return ""
}

// GetPasteCommand returns empty for mock
func (m *MockClipboard) GetPasteCommand() string {
	return ""
}

// GetCopyCommand returns the detected copy command
func (c *Clipboard) GetCopyCommand() string {
	if c.copyCmd == nil {
		return ""
	}
	return strings.Join(c.copyCmd, " ")
}

// GetPasteCommand returns the detected paste command
func (c *Clipboard) GetPasteCommand() string {
	if c.pasteCmd == nil {
		return ""
	}
	return strings.Join(c.pasteCmd, " ")
}

// GetPlatform returns the current platform
func (c *Clipboard) GetPlatform() string {
	return runtime.GOOS
}

// MockClipboard is a mock clipboard for testing
type MockClipboard struct {
	Content string
}

// NewMockClipboard creates a new mock clipboard
func NewMockClipboard() *MockClipboard {
	return &MockClipboard{}
}

// IsAvailable returns true
func (m *MockClipboard) IsAvailable() bool {
	return true
}

// Copy stores text in the mock
func (m *MockClipboard) Copy(text string) error {
	m.Content = text
	return nil
}

// Paste returns the stored text
func (m *MockClipboard) Paste() (string, error) {
	return m.Content, nil
}

// CopyFromFile copies file content
func (m *MockClipboard) CopyFromFile(filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	m.Content = string(content)
	return nil
}

// CopyCodeBlock copies code with fences
func (m *MockClipboard) CopyCodeBlock(code, language string) error {
	if language != "" {
		m.Content = fmt.Sprintf("```%s\n%s\n```", language, code)
	} else {
		m.Content = fmt.Sprintf("```\n%s\n```", code)
	}
	return nil
}
