// Package opencode provides compatibility with OpenCode v1.2 file-based storage
package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rhony08/magicode/internal/util/log"
)

// FileSession represents a session stored in OpenCode v1.2 format
type FileSession struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Directory string `json:"directory"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	Model     string `json:"model,omitempty"`
	Provider  string `json:"provider,omitempty"`
}

// FileStorage reads OpenCode v1.2 file-based storage
type FileStorage struct {
	basePath string
}

// NewFileStorage creates a new file-based storage reader
func NewFileStorage(basePath string) *FileStorage {
	return &FileStorage{basePath: basePath}
}

// DefaultFileStorage creates a file storage using default OpenCode path
func DefaultFileStorage() *FileStorage {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &FileStorage{basePath: ""}
	}
	return NewFileStorage(filepath.Join(homeDir, ".local", "share", "opencode"))
}

// HasFileStorage checks if file-based storage exists
func (fs *FileStorage) HasFileStorage() bool {
	if fs.basePath == "" {
		return false
	}
	storageDir := filepath.Join(fs.basePath, "storage", "session")
	_, err := os.Stat(storageDir)
	return err == nil
}

// ListSessions reads all sessions from file-based storage
func (fs *FileStorage) ListSessions() ([]FileSession, error) {
	if fs.basePath == "" {
		return nil, fmt.Errorf("base path not set")
	}

	sessionDir := filepath.Join(fs.basePath, "storage", "session")

	// Check if directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("session directory does not exist: %s", sessionDir)
	}

	// Read all session directories
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []FileSession
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip special directories
		if entry.Name() == "global" {
			continue
		}

		// Read session files in this directory
		dirPath := filepath.Join(sessionDir, entry.Name())
		sessionFiles, err := os.ReadDir(dirPath)
		if err != nil {
			log.Warn("Failed to read session subdirectory", "dir", entry.Name(), "error", err.Error())
			continue
		}

		for _, file := range sessionFiles {
			if filepath.Ext(file.Name()) != ".json" {
				continue
			}

			filePath := filepath.Join(dirPath, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				log.Warn("Failed to read session file", "file", file.Name(), "error", err.Error())
				continue
			}

			var session FileSession
			if err := json.Unmarshal(data, &session); err != nil {
				log.Warn("Failed to parse session file", "file", file.Name(), "error", err.Error())
				continue
			}

			// Validate required fields
			if session.ID == "" {
				log.Warn("Session file missing ID", "file", file.Name())
				continue
			}

			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// CountSessions returns the number of sessions in file-based storage
func (fs *FileStorage) CountSessions() int {
	sessions, err := fs.ListSessions()
	if err != nil {
		return 0
	}
	return len(sessions)
}

// ToTUISession converts a FileSession to TUI Session format
func (fs *FileSession) ToTUISession() (ID string, Title string, Directory string, CreatedAt time.Time) {
	id := fs.ID
	if id == "" {
		id = fmt.Sprintf("file-session-%d", fs.CreatedAt)
	}

	title := fs.Title
	if title == "" {
		title = "Untitled Session"
	}

	directory := fs.Directory
	if directory == "" {
		directory = "/unknown"
	}

	createdAt := time.UnixMilli(fs.CreatedAt)
	if fs.CreatedAt == 0 {
		createdAt = time.Now()
	}

	return id, title, directory, createdAt
}

// GetStoragePath returns the path to the file-based storage
func (fs *FileStorage) GetStoragePath() string {
	return filepath.Join(fs.basePath, "storage", "session")
}
