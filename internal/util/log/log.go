package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level represents log severity
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to a Level
func ParseLevel(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG, nil
	case "INFO":
		return INFO, nil
	case "WARN", "WARNING":
		return WARN, nil
	case "ERROR":
		return ERROR, nil
	default:
		return INFO, fmt.Errorf("unknown log level: %s", s)
	}
}

// Logger is a structured logger
type Logger struct {
	service string
	level   Level
	print   bool
	writer  io.Writer
	file    *os.File
	mu      sync.Mutex
	tags    map[string]string
}

var (
	// Default is the default logger
	Default *Logger

	// loggers cache by service name
	loggers = make(map[string]*Logger)
	mu      sync.RWMutex

	// logFilePath stores the log file path
	logFilePath string
)

// Init initializes the logging system
func Init(levelStr string, print bool, logDir string) error {
	level, err := ParseLevel(levelStr)
	if err != nil {
		level = INFO // default to INFO
	}

	// Determine log file path
	if logDir == "" {
		// Use temp directory if no log dir specified
		logDir = os.TempDir()
	}
	logFilePath = filepath.Join(logDir, "magicode.log")

	// Open log file
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Fall back to stderr only
		file = nil
	}

	Default = &Logger{
		service: "default",
		level:   level,
		print:   print,
		writer:  os.Stderr,
		file:    file,
		tags:    make(map[string]string),
	}

	return nil
}

// InitDefault initializes with default settings (stderr only)
func InitDefault() {
	Default = &Logger{
		service: "default",
		level:   INFO,
		print:   false,
		writer:  os.Stderr,
		file:    nil,
		tags:    make(map[string]string),
	}
}

// Create creates a new logger for a service
func Create(opts map[string]string) *Logger {
	if Default == nil {
		InitDefault()
	}

	service := opts["service"]
	if service == "" {
		service = "unknown"
	}

	mu.RLock()
	if l, ok := loggers[service]; ok {
		mu.RUnlock()
		return l
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	l := &Logger{
		service: service,
		level:   Default.level,
		print:   Default.print,
		writer:  Default.writer,
		file:    Default.file,
		tags:    make(map[string]string),
	}

	loggers[service] = l
	return l
}

// Clone creates a copy of the logger
func (l *Logger) Clone() *Logger {
	return &Logger{
		service: l.service,
		level:   l.level,
		print:   l.print,
		writer:  l.writer,
		file:    l.file,
		tags:    make(map[string]string),
	}
}

// Tag adds a tag to the logger
func (l *Logger) Tag(key, value string) *Logger {
	clone := l.Clone()
	clone.tags[key] = value
	return clone
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log writes a log entry
func (l *Logger) log(level Level, msg string, data map[string]interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := map[string]interface{}{
		"time":    time.Now().Format(time.RFC3339),
		"level":   level.String(),
		"service": l.service,
		"message": msg,
	}

	// Add tags
	for k, v := range l.tags {
		entry[k] = v
	}

	// Add data
	for k, v := range data {
		entry[k] = v
	}

	// Format log line
	line := formatEntry(entry)

	// Write to file
	if l.file != nil {
		l.file.WriteString(line + "\n")
	}

	// Print to stderr if enabled
	if l.print && l.writer != nil {
		fmt.Fprintln(l.writer, line)
	}
}

func formatEntry(entry map[string]interface{}) string {
	// Simple key=value format
	var parts []string
	parts = append(parts, fmt.Sprintf("time=%s", entry["time"]))
	parts = append(parts, fmt.Sprintf("level=%s", entry["level"]))
	parts = append(parts, fmt.Sprintf("service=%s", entry["service"]))
	parts = append(parts, fmt.Sprintf("message=%s", entry["message"]))
	
	for k, v := range entry {
		if k != "time" && k != "level" && k != "service" && k != "message" {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	return strings.Join(parts, " ")
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, data map[string]interface{}) {
	l.log(DEBUG, msg, data)
}

// Info logs an info message
func (l *Logger) Info(msg string, data map[string]interface{}) {
	l.log(INFO, msg, data)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, data map[string]interface{}) {
	l.log(WARN, msg, data)
}

// Error logs an error message
func (l *Logger) Error(msg string, data map[string]interface{}) {
	l.log(ERROR, msg, data)
}

// Close closes the log file
func Close() error {
	if Default != nil && Default.file != nil {
		return Default.file.Close()
	}
	return nil
}

// File returns the log file path
func File() string {
	return logFilePath
}

// SetWriter sets the output writer (for testing)
func (l *Logger) SetWriter(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = w
}

// Convenience functions that use Default logger with variadic key-value pairs
// These match the slog-style API: log.Debug("message", "key1", value1, "key2", value2)

// Debug logs a debug message using Default logger
func Debug(msg string, keysAndValues ...interface{}) {
	if Default == nil {
		InitDefault()
	}
	data := keysToMap(keysAndValues)
	Default.log(DEBUG, msg, data)
}

// Info logs an info message using Default logger
func Info(msg string, keysAndValues ...interface{}) {
	if Default == nil {
		InitDefault()
	}
	data := keysToMap(keysAndValues)
	Default.log(INFO, msg, data)
}

// Warn logs a warning message using Default logger
func Warn(msg string, keysAndValues ...interface{}) {
	if Default == nil {
		InitDefault()
	}
	data := keysToMap(keysAndValues)
	Default.log(WARN, msg, data)
}

// Error logs an error message using Default logger
func Error(msg string, keysAndValues ...interface{}) {
	if Default == nil {
		InitDefault()
	}
	data := keysToMap(keysAndValues)
	Default.log(ERROR, msg, data)
}

// keysToMap converts variadic key-value pairs to a map
func keysToMap(kv []interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	for i := 0; i < len(kv)-1; i += 2 {
		if key, ok := kv[i].(string); ok {
			data[key] = kv[i+1]
		}
	}
	return data
}