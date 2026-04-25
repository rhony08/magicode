// Package opencode provides compatibility with OpenCode v1.2 file-based storage
package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewFileStorage(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		wantPath string
	}{
		{
			name:     "custom path",
			basePath: "/custom/path",
			wantPath: "/custom/path",
		},
		{
			name:     "empty path",
			basePath: "",
			wantPath: "",
		},
		{
			name:     "path with spaces",
			basePath: "/path with spaces/opencode",
			wantPath: "/path with spaces/opencode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFileStorage(tt.basePath)
			if fs.basePath != tt.wantPath {
				t.Errorf("NewFileStorage() basePath = %v, want %v", fs.basePath, tt.wantPath)
			}
		})
	}
}

func TestDefaultFileStorage(t *testing.T) {
	fs := DefaultFileStorage()

	if fs == nil {
		t.Fatal("DefaultFileStorage() returned nil")
	}

	// Should have a non-empty base path (unless home dir can't be determined)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, basePath should be empty
		if fs.basePath != "" {
			t.Errorf("Expected empty basePath when home dir unavailable, got %s", fs.basePath)
		}
	} else {
		expectedPath := filepath.Join(homeDir, ".local", "share", "opencode")
		if fs.basePath != expectedPath {
			t.Errorf("DefaultFileStorage() basePath = %v, want %v", fs.basePath, expectedPath)
		}
	}
}

func TestFileStorage_HasFileStorage(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create the storage/session directory
	storageDir := filepath.Join(tempDir, "storage", "session")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		t.Fatalf("Failed to create storage directory: %v", err)
	}

	tests := []struct {
		name     string
		basePath string
		setup    func()
		want     bool
	}{
		{
			name:     "empty base path",
			basePath: "",
			want:     false,
		},
		{
			name:     "storage exists",
			basePath: tempDir,
			want:     true,
		},
		{
			name:     "storage does not exist",
			basePath: filepath.Join(tempDir, "nonexistent"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFileStorage(tt.basePath)
			got := fs.HasFileStorage()
			if got != tt.want {
				t.Errorf("HasFileStorage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileStorage_ListSessions(t *testing.T) {
	// Create a temporary directory with mock session files
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, "storage", "session")

	// Create session directories
	sessionDir1 := filepath.Join(storageDir, "abc123")
	sessionDir2 := filepath.Join(storageDir, "def456")
	globalDir := filepath.Join(storageDir, "global") // Should be skipped

	for _, dir := range []string{sessionDir1, sessionDir2, globalDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create valid session files
	session1 := `{"id": "ses_abc123", "title": "Test Session 1", "directory": "/path/to/project1", "created_at": 1704067200000, "updated_at": 1704067200000, "model": "gpt-4", "provider": "openai"}`
	session2 := `{"id": "ses_def456", "title": "Test Session 2", "directory": "/path/to/project2", "created_at": 1704153600000, "updated_at": 1704153600000}`

	if err := os.WriteFile(filepath.Join(sessionDir1, "ses_abc123.json"), []byte(session1), 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir2, "ses_def456.json"), []byte(session2), 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	tests := []struct {
		name    string
		fs      *FileStorage
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty base path",
			fs:      NewFileStorage(""),
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "nonexistent directory",
			fs:      NewFileStorage(filepath.Join(tempDir, "nonexistent")),
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "valid storage with sessions",
			fs:      NewFileStorage(tempDir),
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fs.ListSessions()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListSessions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("ListSessions() returned %d sessions, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestFileStorage_ListSessions_InvalidFiles(t *testing.T) {
	// Create temp directory with various invalid files
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, "storage", "session", "testhash")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Valid session
	validSession := `{"id": "valid123", "title": "Valid Session", "directory": "/test", "created_at": 1704067200000}`
	if err := os.WriteFile(filepath.Join(storageDir, "valid.json"), []byte(validSession), 0644); err != nil {
		t.Fatalf("Failed to write valid session: %v", err)
	}

	// Invalid JSON
	invalidJSON := `{"id": "invalid", "title": "Broken"` // Missing closing brace
	if err := os.WriteFile(filepath.Join(storageDir, "invalid.json"), []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Missing ID field
	missingID := `{"title": "No ID", "directory": "/test", "created_at": 1704067200000}`
	if err := os.WriteFile(filepath.Join(storageDir, "noid.json"), []byte(missingID), 0644); err != nil {
		t.Fatalf("Failed to write missing ID session: %v", err)
	}

	// Non-JSON file (should be skipped)
	if err := os.WriteFile(filepath.Join(storageDir, "readme.txt"), []byte("Not a session"), 0644); err != nil {
		t.Fatalf("Failed to write text file: %v", err)
	}

	fs := NewFileStorage(tempDir)
	sessions, err := fs.ListSessions()
	if err != nil {
		t.Errorf("ListSessions() unexpected error: %v", err)
	}

	// Should only get the valid session
	if len(sessions) != 1 {
		t.Errorf("ListSessions() returned %d sessions, want 1", len(sessions))
	}

	if len(sessions) > 0 && sessions[0].ID != "valid123" {
		t.Errorf("ListSessions() returned session with ID %s, want valid123", sessions[0].ID)
	}
}

func TestFileStorage_CountSessions(t *testing.T) {
	// Create temp directory with sessions
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, "storage", "session", "testhash")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create 3 sessions
	for i := 0; i < 3; i++ {
		session := `{"id": "session` + string(rune('0'+i)) + `", "title": "Session ` + string(rune('0'+i)) + `", "created_at": 1704067200000}`
		filename := filepath.Join(storageDir, "session"+string(rune('0'+i))+".json")
		if err := os.WriteFile(filename, []byte(session), 0644); err != nil {
			t.Fatalf("Failed to write session: %v", err)
		}
	}

	tests := []struct {
		name string
		fs   *FileStorage
		want int
	}{
		{
			name: "empty base path",
			fs:   NewFileStorage(""),
			want: 0,
		},
		{
			name: "storage with 3 sessions",
			fs:   NewFileStorage(tempDir),
			want: 3,
		},
		{
			name: "nonexistent storage",
			fs:   NewFileStorage(filepath.Join(tempDir, "nonexistent")),
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fs.CountSessions()
			if got != tt.want {
				t.Errorf("CountSessions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileSession_ToTUISession(t *testing.T) {
	tests := []struct {
		name          string
		session       FileSession
		wantID        string
		wantTitle     string
		wantDirectory string
		wantCreated   int64
	}{
		{
			name: "complete session",
			session: FileSession{
				ID:        "ses_abc123",
				Title:     "Test Session",
				Directory: "/path/to/project",
				CreatedAt: 1704067200000, // 2024-01-01 00:00:00 UTC
			},
			wantID:        "ses_abc123",
			wantTitle:     "Test Session",
			wantDirectory: "/path/to/project",
			wantCreated:   1704067200000,
		},
		{
			name:          "empty session - defaults",
			session:       FileSession{},
			wantID:        "file-session-0",
			wantTitle:     "Untitled Session",
			wantDirectory: "/unknown",
			wantCreated:   0, // Will be checked separately since it's time.Now()
		},
		{
			name: "session with only ID",
			session: FileSession{
				ID: "only_id",
			},
			wantID:        "only_id",
			wantTitle:     "Untitled Session",
			wantDirectory: "/unknown",
			wantCreated:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotTitle, gotDirectory, gotCreated := tt.session.ToTUISession()

			if gotID != tt.wantID {
				t.Errorf("ToTUISession() ID = %v, want %v", gotID, tt.wantID)
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("ToTUISession() Title = %v, want %v", gotTitle, tt.wantTitle)
			}
			if gotDirectory != tt.wantDirectory {
				t.Errorf("ToTUISession() Directory = %v, want %v", gotDirectory, tt.wantDirectory)
			}

			if tt.wantCreated == 0 {
				// Check that CreatedAt is recent (within last minute)
				if time.Since(gotCreated) > time.Minute {
					t.Errorf("ToTUISession() CreatedAt should be recent, got %v", gotCreated)
				}
			} else {
				wantTime := time.UnixMilli(tt.wantCreated)
				if !gotCreated.Equal(wantTime) {
					t.Errorf("ToTUISession() CreatedAt = %v, want %v", gotCreated, wantTime)
				}
			}
		})
	}
}

func TestFileStorage_GetStoragePath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		want     string
	}{
		{
			name:     "custom path",
			basePath: "/custom/path",
			want:     filepath.Join("/custom/path", "storage", "session"),
		},
		{
			name:     "empty path",
			basePath: "",
			want:     filepath.Join("", "storage", "session"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewFileStorage(tt.basePath)
			got := fs.GetStoragePath()
			if got != tt.want {
				t.Errorf("GetStoragePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileSession_Fields(t *testing.T) {
	// Test that all fields are properly parsed
	sessionJSON := `{
		"id": "test-id",
		"title": "Test Title",
		"directory": "/test/dir",
		"created_at": 1704067200000,
		"updated_at": 1704153600000,
		"model": "gpt-4-turbo",
		"provider": "openai"
	}`

	var session FileSession
	if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}

	if session.ID != "test-id" {
		t.Errorf("ID = %v, want test-id", session.ID)
	}
	if session.Title != "Test Title" {
		t.Errorf("Title = %v, want Test Title", session.Title)
	}
	if session.Directory != "/test/dir" {
		t.Errorf("Directory = %v, want /test/dir", session.Directory)
	}
	if session.CreatedAt != 1704067200000 {
		t.Errorf("CreatedAt = %v, want 1704067200000", session.CreatedAt)
	}
	if session.UpdatedAt != 1704153600000 {
		t.Errorf("UpdatedAt = %v, want 1704153600000", session.UpdatedAt)
	}
	if session.Model != "gpt-4-turbo" {
		t.Errorf("Model = %v, want gpt-4-turbo", session.Model)
	}
	if session.Provider != "openai" {
		t.Errorf("Provider = %v, want openai", session.Provider)
	}
}

func TestFileSession_OptionalFields(t *testing.T) {
	// Test that optional fields (model, provider) can be empty
	sessionJSON := `{
		"id": "minimal-id",
		"title": "Minimal Session",
		"directory": "/test",
		"created_at": 1704067200000
	}`

	var session FileSession
	if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
		t.Fatalf("Failed to unmarshal minimal session: %v", err)
	}

	if session.ID != "minimal-id" {
		t.Errorf("ID = %v, want minimal-id", session.ID)
	}
	if session.Model != "" {
		t.Errorf("Model should be empty, got %v", session.Model)
	}
	if session.Provider != "" {
		t.Errorf("Provider should be empty, got %v", session.Provider)
	}
}
