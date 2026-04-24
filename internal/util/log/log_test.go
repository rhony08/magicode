package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
		hasError bool
	}{
		{"DEBUG", DEBUG, false},
		{"debug", DEBUG, false},
		{"INFO", INFO, false},
		{"info", INFO, false},
		{"WARN", WARN, false},
		{"warn", WARN, false},
		{"WARNING", WARN, false},
		{"ERROR", ERROR, false},
		{"error", ERROR, false},
		{"UNKNOWN", INFO, true}, // Returns INFO with error
		{"", INFO, true},
	}

	for _, tt := range tests {
		level, err := ParseLevel(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseLevel(%s) expected error, got none", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseLevel(%s) unexpected error: %v", tt.input, err)
			}
		}
		if level != tt.expected {
			t.Errorf("ParseLevel(%s) expected %d, got %d", tt.input, tt.expected, level)
		}
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{Level(100), "UNKNOWN"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("Level(%d).String() expected %s, got %s", tt.level, tt.expected, result)
		}
	}
}

func TestInitDefault(t *testing.T) {
	InitDefault()

	if Default == nil {
		t.Fatal("Expected Default logger to be initialized")
	}

	if Default.level != INFO {
		t.Errorf("Expected default level=INFO, got %d", Default.level)
	}

	if Default.service != "default" {
		t.Errorf("Expected service=default, got %s", Default.service)
	}
}

func TestCreate(t *testing.T) {
	InitDefault()

	logger := Create(map[string]string{"service": "test"})
	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if logger.service != "test" {
		t.Errorf("Expected service=test, got %s", logger.service)
	}

	// Test cached logger
	logger2 := Create(map[string]string{"service": "test"})
	if logger != logger2 {
		t.Error("Expected cached logger to be returned")
	}
}

func TestLoggerTag(t *testing.T) {
	InitDefault()

	logger := Create(map[string]string{"service": "test"})
	tagged := logger.Tag("key", "value")

	if tagged.tags["key"] != "value" {
		t.Errorf("Expected tag key=value, got %s", tagged.tags["key"])
	}

	// Original logger should not be modified
	if logger.tags["key"] != "" {
		t.Error("Expected original logger tags to be empty")
	}
}

func TestLoggerLogging(t *testing.T) {
	// Create a buffer to capture output
	buf := &bytes.Buffer{}

	InitDefault()
	Default.SetWriter(buf)
	Default.print = true
	Default.level = DEBUG

	Default.Debug("debug message", map[string]interface{}{"key": "value"})
	Default.Info("info message", nil)
	Default.Warn("warn message", nil)
	Default.Error("error message", nil)

	output := buf.String()

	// Check that all messages are logged
	if !strings.Contains(output, "level=DEBUG") {
		t.Error("Expected DEBUG level in output")
	}
	if !strings.Contains(output, "level=INFO") {
		t.Error("Expected INFO level in output")
	}
	if !strings.Contains(output, "level=WARN") {
		t.Error("Expected WARN level in output")
	}
	if !strings.Contains(output, "level=ERROR") {
		t.Error("Expected ERROR level in output")
	}
}

func TestLoggerLevelFilter(t *testing.T) {
	buf := &bytes.Buffer{}

	InitDefault()
	Default.SetWriter(buf)
	Default.print = true
	Default.level = WARN // Only WARN and ERROR should be logged

	Default.Debug("debug message", nil)
	Default.Info("info message", nil)
	Default.Warn("warn message", nil)
	Default.Error("error message", nil)

	output := buf.String()

	// DEBUG and INFO should be filtered out
	if strings.Contains(output, "level=DEBUG") {
		t.Error("DEBUG should be filtered")
	}
	if strings.Contains(output, "level=INFO") {
		t.Error("INFO should be filtered")
	}

	// WARN and ERROR should be logged
	if !strings.Contains(output, "level=WARN") {
		t.Error("Expected WARN in output")
	}
	if !strings.Contains(output, "level=ERROR") {
		t.Error("Expected ERROR in output")
	}
}

func TestFormatEntry(t *testing.T) {
	entry := map[string]interface{}{
		"time":    "2024-01-01T00:00:00Z",
		"level":   "INFO",
		"service": "test",
		"message": "test message",
		"extra":   "value",
	}

	result := formatEntry(entry)

	// Check format
	if !strings.Contains(result, "time=") {
		t.Error("Expected time in formatted entry")
	}
	if !strings.Contains(result, "level=INFO") {
		t.Error("Expected level in formatted entry")
	}
	if !strings.Contains(result, "service=test") {
		t.Error("Expected service in formatted entry")
	}
	if !strings.Contains(result, "message=test message") {
		t.Error("Expected message in formatted entry")
	}
	if !strings.Contains(result, "extra=value") {
		t.Error("Expected extra field in formatted entry")
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init("DEBUG", true, tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if Default == nil {
		t.Fatal("Expected Default logger to be set")
	}

	if Default.level != DEBUG {
		t.Errorf("Expected level=DEBUG, got %d", Default.level)
	}

	// Check log file path
	logPath := File()
	if !strings.Contains(logPath, "magicode.log") {
		t.Errorf("Expected log path to contain magicode.log, got %s", logPath)
	}

	// Close the log file
	Close()
}

func TestLoggerClone(t *testing.T) {
	InitDefault()

	logger := Create(map[string]string{"service": "test"})
	clone := logger.Clone()

	if clone.service != logger.service {
		t.Errorf("Expected same service, got %s vs %s", clone.service, logger.service)
	}

	// Modify clone should not affect original
	clone.level = ERROR
	if logger.level == ERROR {
		t.Error("Modifying clone should not affect original")
	}
}