// Package component provides reusable UI components for the TUI.
// This file contains tests for clipboard operations.
package component

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewClipboard(t *testing.T) {
	c := NewClipboard()

	if c == nil {
		t.Fatal("NewClipboard() returned nil")
	}

	// Should have detected commands for current platform
	if c.GetPlatform() == "" {
		t.Error("GetPlatform() should return non-empty string")
	}
}

func TestClipboard_detectCommands(t *testing.T) {
	c := NewClipboard()
	c.detectCommands()

	// Platform should be set
	platform := c.GetPlatform()
	if platform != "linux" && platform != "darwin" && platform != "windows" {
		t.Errorf("Unexpected platform: %s", platform)
	}

	// Commands should be detected based on platform
	switch platform {
	case "darwin":
		if c.GetCopyCommand() != "pbcopy" {
			t.Errorf("macOS copy command = %s, want pbcopy", c.GetCopyCommand())
		}
	case "windows":
		if c.GetCopyCommand() != "clip" {
			t.Errorf("Windows copy command = %s, want clip", c.GetCopyCommand())
		}
	}
}

func TestClipboard_commandExists(t *testing.T) {
	c := NewClipboard()

	// "ls" should exist on Unix-like systems
	if c.GetPlatform() != "windows" {
		if !c.commandExists("ls") {
			t.Error("commandExists('ls') should return true")
		}
	}

	// Non-existent command should return false
	if c.commandExists("this_command_does_not_exist_12345") {
		t.Error("commandExists should return false for non-existent command")
	}
}

func TestClipboard_IsAvailable(t *testing.T) {
	c := NewClipboard()

	// Should return true or false, not panic
	_ = c.IsAvailable()
}

func TestMockClipboard(t *testing.T) {
	m := NewMockClipboard()

	if m == nil {
		t.Fatal("NewMockClipboard() returned nil")
	}

	// IsAvailable should return true
	if !m.IsAvailable() {
		t.Error("IsAvailable() should return true")
	}

	// Copy should work
	err := m.Copy("test content")
	if err != nil {
		t.Errorf("Copy() error: %v", err)
	}

	// Paste should return the content
	content, err := m.Paste()
	if err != nil {
		t.Errorf("Paste() error: %v", err)
	}

	if content != "test content" {
		t.Errorf("Paste() = %s, want 'test content'", content)
	}
}

func TestMockClipboard_CopyFromFile(t *testing.T) {
	m := NewMockClipboard()

	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "file content"

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Copy from file
	err = m.CopyFromFile(tmpFile)
	if err != nil {
		t.Errorf("CopyFromFile() error: %v", err)
	}

	// Paste should return the file content
	result, err := m.Paste()
	if err != nil {
		t.Errorf("Paste() error: %v", err)
	}

	if result != content {
		t.Errorf("Paste() = %s, want %s", result, content)
	}
}

func TestMockClipboard_CopyFromFile_NonExistent(t *testing.T) {
	m := NewMockClipboard()

	err := m.CopyFromFile("/non/existent/file.txt")
	if err == nil {
		t.Error("CopyFromFile() should return error for non-existent file")
	}
}

func TestMockClipboard_CopyCodeBlock(t *testing.T) {
	m := NewMockClipboard()

	// Copy code with language
	err := m.CopyCodeBlock("console.log('hello')", "javascript")
	if err != nil {
		t.Errorf("CopyCodeBlock() error: %v", err)
	}

	content, _ := m.Paste()
	if content == "" {
		t.Error("Paste() should return non-empty content")
	}

	// Should contain code fence
	if content[:3] != "```" {
		t.Error("Content should start with code fence")
	}

	// Should contain language
	if content[:13] != "```javascript" {
		t.Errorf("Content should contain language, got: %s", content[:20])
	}
}

func TestMockClipboard_CopyCodeBlock_NoLanguage(t *testing.T) {
	m := NewMockClipboard()

	// Copy code without language
	err := m.CopyCodeBlock("some code", "")
	if err != nil {
		t.Errorf("CopyCodeBlock() error: %v", err)
	}

	content, _ := m.Paste()
	if content[:3] != "```" {
		t.Error("Content should start with code fence")
	}
}

func TestMockClipboard_GetPlatform(t *testing.T) {
	m := NewMockClipboard()
	// Just ensure it doesn't panic
	_ = m.GetPlatform()
}

func TestMockClipboard_GetCopyCommand(t *testing.T) {
	m := NewMockClipboard()
	cmd := m.GetCopyCommand()
	// Mock doesn't set commands
	if cmd != "" {
		t.Errorf("GetCopyCommand() = %s, want empty string", cmd)
	}
}

func TestMockClipboard_GetPasteCommand(t *testing.T) {
	m := NewMockClipboard()
	cmd := m.GetPasteCommand()
	// Mock doesn't set commands
	if cmd != "" {
		t.Errorf("GetPasteCommand() = %s, want empty string", cmd)
	}
}
