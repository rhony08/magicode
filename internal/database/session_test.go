// Package database provides tests for session storage.
package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) (*Database, func()) {
	tmpDir, err := os.MkdirTemp("", "opencode-session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(context.Background(), Config{Path: dbPath})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create a project first (sessions need it)
	projectStore := NewProjectStorage(db)
	_, err = projectStore.Create(context.Background(), Project{
		ID:       "test-project",
		Worktree: "/test/worktree",
		Name:     "Test Project",
	})
	if err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create test project: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

// TestSessionCreate tests session creation
func TestSessionCreate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	session, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Check auto-generated fields
	if session.ID == "" {
		t.Error("ID should be auto-generated")
	}
	if session.Slug == "" {
		t.Error("Slug should be auto-generated")
	}
	if session.Title == "" {
		t.Error("Title should have default value")
	}
	if session.Timestamps.TimeCreated == 0 {
		t.Error("TimeCreated should be set")
	}

	// Verify can be retrieved
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}
	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, retrieved.ID)
	}
}

// TestSessionCreateWithCustomID tests session creation with custom ID
func TestSessionCreateWithCustomID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	session, err := store.Create(ctx, Session{
		ID:        "custom-session-id",
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Custom Title",
		Slug:      "custom-slug",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.ID != "custom-session-id" {
		t.Errorf("Expected custom ID, got %s", session.ID)
	}
	if session.Title != "Custom Title" {
		t.Errorf("Expected custom title, got %s", session.Title)
	}
	if session.Slug != "custom-slug" {
		t.Errorf("Expected custom slug, got %s", session.Slug)
	}
}

// TestSessionGet tests session retrieval
func TestSessionGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create session
	session, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get session
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.Title != "Test Session" {
		t.Errorf("Expected title 'Test Session', got '%s'", retrieved.Title)
	}
	if retrieved.Directory != "/test/dir" {
		t.Errorf("Expected directory '/test/dir', got '%s'", retrieved.Directory)
	}
}

// TestSessionGetNotFound tests session retrieval for non-existent ID
func TestSessionGetNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	_, err := store.Get(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
	if err.Error() != "session not found: non-existent-id" {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestSessionList tests session listing by project
func TestSessionList(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		_, err := store.Create(ctx, Session{
			ProjectID: "test-project",
			Directory: "/test/dir",
			Title:     fmt.Sprintf("Session %d", i),
		})
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
	}

	// List sessions
	sessions, err := store.List(ctx, "test-project")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
}

// TestSessionListByDirectory tests session listing by directory
func TestSessionListByDirectory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create sessions in different directories
	_, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir1",
		Title:     "Session 1",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	_, err = store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir2",
		Title:     "Session 2",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	_, err = store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir1",
		Title:     "Session 3",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// List by directory
	sessions, err := store.ListByDirectory(ctx, "/test/dir1")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions in dir1, got %d", len(sessions))
	}

	sessions, err = store.ListByDirectory(ctx, "/test/dir2")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session in dir2, got %d", len(sessions))
	}
}

// TestSessionUpdate tests session update
func TestSessionUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create session
	session, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Original Title",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Update session
	session.Title = "Updated Title"
	session.SummaryAdditions = 100
	session.SummaryDeletions = 50
	session.SummaryFiles = 5

	err = store.Update(ctx, *session)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Expected 'Updated Title', got '%s'", retrieved.Title)
	}
	if retrieved.SummaryAdditions != 100 {
		t.Errorf("Expected 100 additions, got %d", retrieved.SummaryAdditions)
	}
	if retrieved.Timestamps.TimeUpdated <= session.Timestamps.TimeCreated {
		t.Error("TimeUpdated should be greater than TimeCreated after update")
	}
}

// TestSessionDelete tests session deletion
func TestSessionDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create session
	session, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "To Delete",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Delete session
	err = store.Delete(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify deletion
	_, err = store.Get(ctx, session.ID)
	if err == nil {
		t.Error("Session should not exist after deletion")
	}
}

// TestSessionArchive tests session archiving
func TestSessionArchive(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create session
	session, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "To Archive",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Archive session
	err = store.Archive(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to archive session: %v", err)
	}

	// Verify archive
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get archived session: %v", err)
	}

	if retrieved.TimeArchived == 0 {
		t.Error("TimeArchived should be set")
	}
}

// TestSessionUnarchive tests session unarchiving
func TestSessionUnarchive(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create and archive session
	session, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "To Archive/Unarchive",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err = store.Archive(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to archive session: %v", err)
	}

	// Unarchive session
	err = store.Unarchive(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to unarchive session: %v", err)
	}

	// Verify unarchive
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get unarchived session: %v", err)
	}

	if retrieved.TimeArchived != 0 {
		t.Error("TimeArchived should be 0 after unarchive")
	}
}

// TestSessionFork tests session forking
func TestSessionFork(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create parent session
	parent, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Parent Session",
		Slug:      "parent",
	})
	if err != nil {
		t.Fatalf("Failed to create parent session: %v", err)
	}

	// Fork session
	fork, err := store.Fork(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Failed to fork session: %v", err)
	}

	// Verify fork
	if fork.ParentID != parent.ID {
		t.Errorf("Expected parent ID %s, got %s", parent.ID, fork.ParentID)
	}
	if fork.ProjectID != parent.ProjectID {
		t.Error("Fork should have same project ID")
	}
	if fork.Directory != parent.Directory {
		t.Error("Fork should have same directory")
	}
	if fork.ID == parent.ID {
		t.Error("Fork should have different ID")
	}
}

// TestSessionJSONFields tests JSON field handling
func TestSessionJSONFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStorage(db)
	ctx := context.Background()

	// Create session with JSON fields
	session, err := store.Create(ctx, Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "JSON Test",
		SummaryDiffs: []FileDiff{
			{Path: "file1.txt", Additions: 10, Deletions: 5},
			{Path: "file2.txt", Additions: 20, Deletions: 10},
		},
		Permission: &Ruleset{
			Allow: []string{"read", "write"},
			Deny:  []string{"delete"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Retrieve and check JSON fields
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if len(retrieved.SummaryDiffs) != 2 {
		t.Errorf("Expected 2 diffs, got %d", len(retrieved.SummaryDiffs))
	}
	if retrieved.SummaryDiffs[0].Path != "file1.txt" {
		t.Errorf("Expected path 'file1.txt', got '%s'", retrieved.SummaryDiffs[0].Path)
	}
	if retrieved.Permission == nil {
		t.Error("Permission should not be nil")
	}
	if len(retrieved.Permission.Allow) != 2 {
		t.Errorf("Expected 2 allow rules, got %d", len(retrieved.Permission.Allow))
	}
}