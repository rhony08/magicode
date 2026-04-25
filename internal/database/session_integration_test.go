// Package database provides integration tests for session storage.
// These tests verify actual database operations with real SQLite database.
package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupIntegrationDB creates a test database using t.TempDir() for isolation
func setupIntegrationDB(t *testing.T) (*Database, string) {
	// Use t.TempDir() for automatic cleanup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	ctx := context.Background()
	db, err := New(ctx, Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to create database at %s: %v", dbPath, err)
	}

	return db, dbPath
}

// createTestProject creates a project required for session tests
func createTestProject(t *testing.T, db *Database, id, worktree, name string) *Project {
	ctx := context.Background()
	projectStore := NewProjectStorage(db)

	project, err := projectStore.Create(ctx, Project{
		ID:       id,
		Worktree: worktree,
		Name:     name,
	})
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	return project
}

// TestSessionDatabaseFileCreation verifies that the database file is actually created
func TestSessionDatabaseFileCreation(t *testing.T) {
	db, dbPath := setupIntegrationDB(t)
	defer db.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created at %s", dbPath)
	}

	// Verify we can write to it by creating a project
	project := createTestProject(t, db, "test-project", "/test/worktree", "Test Project")
	if project.ID != "test-project" {
		t.Errorf("Expected project ID 'test-project', got '%s'", project.ID)
	}

	// Verify the file is readable and has content
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Errorf("Failed to stat database file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Database file exists but is empty")
	}

	t.Logf("Database file created at %s, size: %d bytes", dbPath, info.Size())
}

// TestSessionCreationAndRetrieval tests the full flow of creating and retrieving sessions
func TestSessionCreationAndRetrieval(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()

	// Create a project first (sessions require a project)
	project := createTestProject(t, db, "test-project", "/test/worktree", "Test Project")

	ctx := context.Background()
	sessionStore := NewSessionStorage(db)

	// Create a session
	session, err := sessionStore.Create(ctx, Session{
		ProjectID: project.ID,
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.ID == "" {
		t.Fatal("Session ID was not generated")
	}

	t.Logf("Created session with ID: %s, ProjectID: %s", session.ID, session.ProjectID)

	// Retrieve the session
	retrieved, err := sessionStore.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, retrieved.ID)
	}

	if retrieved.ProjectID != project.ID {
		t.Errorf("Expected project ID %s, got %s", project.ID, retrieved.ProjectID)
	}

	if retrieved.Title != "Test Session" {
		t.Errorf("Expected title 'Test Session', got '%s'", retrieved.Title)
	}

	if retrieved.Directory != "/test/dir" {
		t.Errorf("Expected directory '/test/dir', got '%s'", retrieved.Directory)
	}

	t.Logf("Successfully retrieved session: %+v", retrieved)
}

// TestSessionListAll returns all sessions
func TestSessionListAll(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()

	// Create a project
	project := createTestProject(t, db, "test-project", "/test/worktree", "Test Project")

	ctx := context.Background()
	sessionStore := NewSessionStorage(db)

	// Initially should have no sessions
	sessions, err := sessionStore.ListAll(ctx)
	if err != nil {
		t.Fatalf("Failed to list all sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions initially, got %d", len(sessions))
	}

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		_, err := sessionStore.Create(ctx, Session{
			ProjectID: project.ID,
			Directory: "/test/dir",
			Title:     "Test Session",
		})
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// List all sessions
	sessions, err = sessionStore.ListAll(ctx)
	if err != nil {
		t.Fatalf("Failed to list all sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	// Verify sessions are ordered by time_created DESC (newest first)
	for i := 1; i < len(sessions); i++ {
		if sessions[i].Timestamps.TimeCreated > sessions[i-1].Timestamps.TimeCreated {
			t.Error("Sessions are not ordered by time_created DESC")
		}
	}

	t.Logf("ListAll returned %d sessions", len(sessions))
}

// TestSessionListByProject filters sessions by project ID
func TestSessionListByProject(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()

	// Create two projects
	project1 := createTestProject(t, db, "project-1", "/test/worktree1", "Project 1")
	project2 := createTestProject(t, db, "project-2", "/test/worktree2", "Project 2")

	ctx := context.Background()
	sessionStore := NewSessionStorage(db)

	// Create sessions in different projects
	for i := 0; i < 2; i++ {
		_, err := sessionStore.Create(ctx, Session{
			ProjectID: project1.ID,
			Directory: "/test/dir1",
			Title:     "Project 1 Session",
		})
		if err != nil {
			t.Fatalf("Failed to create session for project 1: %v", err)
		}
	}

	for i := 0; i < 3; i++ {
		_, err := sessionStore.Create(ctx, Session{
			ProjectID: project2.ID,
			Directory: "/test/dir2",
			Title:     "Project 2 Session",
		})
		if err != nil {
			t.Fatalf("Failed to create session for project 2: %v", err)
		}
	}

	// List sessions for project 1
	sessions1, err := sessionStore.List(ctx, project1.ID)
	if err != nil {
		t.Fatalf("Failed to list sessions for project 1: %v", err)
	}
	if len(sessions1) != 2 {
		t.Errorf("Expected 2 sessions for project 1, got %d", len(sessions1))
	}

	// List sessions for project 2
	sessions2, err := sessionStore.List(ctx, project2.ID)
	if err != nil {
		t.Fatalf("Failed to list sessions for project 2: %v", err)
	}
	if len(sessions2) != 3 {
		t.Errorf("Expected 3 sessions for project 2, got %d", len(sessions2))
	}

	t.Logf("Project 1 has %d sessions, Project 2 has %d sessions", len(sessions1), len(sessions2))
}

// TestSessionListByDirectoryIntegration filters sessions by directory
func TestSessionListByDirectoryIntegration(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()

	project := createTestProject(t, db, "test-project", "/test/worktree", "Test Project")

	ctx := context.Background()
	sessionStore := NewSessionStorage(db)

	// Create sessions in different directories
	dirs := []string{"/test/dir1", "/test/dir2", "/test/dir1"} // dir1 appears twice
	for i, dir := range dirs {
		_, err := sessionStore.Create(ctx, Session{
			ProjectID: project.ID,
			Directory: dir,
			Title:     "Session " + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// List by dir1
	sessions1, err := sessionStore.ListByDirectory(ctx, "/test/dir1")
	if err != nil {
		t.Fatalf("Failed to list sessions by directory: %v", err)
	}
	if len(sessions1) != 2 {
		t.Errorf("Expected 2 sessions in dir1, got %d", len(sessions1))
	}

	// List by dir2
	sessions2, err := sessionStore.ListByDirectory(ctx, "/test/dir2")
	if err != nil {
		t.Fatalf("Failed to list sessions by directory: %v", err)
	}
	if len(sessions2) != 1 {
		t.Errorf("Expected 1 session in dir2, got %d", len(sessions2))
	}

	// List by non-existent dir
	sessions3, err := sessionStore.ListByDirectory(ctx, "/nonexistent")
	if err != nil {
		t.Fatalf("Failed to list sessions by directory: %v", err)
	}
	if len(sessions3) != 0 {
		t.Errorf("Expected 0 sessions in nonexistent dir, got %d", len(sessions3))
	}

	t.Logf("Directory filter: dir1=%d, dir2=%d, nonexistent=%d", len(sessions1), len(sessions2), len(sessions3))
}

// TestSessionProjectIDRequired verifies behavior when creating session without project ID
// Note: SQLite does not enforce foreign keys by default, so this test documents
// that the application layer should validate project IDs
func TestSessionProjectIDRequired(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()

	ctx := context.Background()
	sessionStore := NewSessionStorage(db)

	// Try to create session without project ID
	// SQLite does NOT enforce foreign keys by default (requires PRAGMA foreign_keys = ON)
	// So this will succeed at the database level, but the application should validate
	session, err := sessionStore.Create(ctx, Session{
		ProjectID: "", // Empty project ID
		Directory: "/test/dir",
		Title:     "Orphan Session",
	})

	// This succeeds because SQLite doesn't enforce FK constraints by default
	if err != nil {
		t.Logf("Got error (FK constraints may be enabled): %v", err)
	} else {
		t.Logf("WARNING: Session created without valid ProjectID: %s", session.ID)
		t.Log("This indicates foreign key constraints are NOT enforced in the database")
		t.Log("The application layer should validate ProjectID before creating sessions")
	}
}

// TestSessionGetNonExistent verifies behavior when session doesn't exist
func TestSessionGetNonExistent(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()

	// Create a project (not strictly needed but keeps test realistic)
	createTestProject(t, db, "test-project", "/test/worktree", "Test Project")

	ctx := context.Background()
	sessionStore := NewSessionStorage(db)

	// Try to get non-existent session
	_, err := sessionStore.Get(ctx, "non-existent-session-id")
	if err == nil {
		t.Error("Expected error when getting non-existent session, but got none")
	} else {
		t.Logf("Got expected error for non-existent session: %v", err)
	}
}

// TestSessionPersistenceAcrossClose verifies sessions persist after closing and reopening database
func TestSessionPersistenceAcrossClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "persist_test.db")

	ctx := context.Background()

	// Create database and session
	{
		db, err := New(ctx, Config{Path: dbPath})
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		project := createTestProject(t, db, "test-project", "/test/worktree", "Test Project")
		sessionStore := NewSessionStorage(db)

		session, err := sessionStore.Create(ctx, Session{
			ProjectID: project.ID,
			Directory: "/test/dir",
			Title:     "Persistent Session",
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		t.Logf("Created session with ID: %s", session.ID)
		db.Close()
	}

	// Reopen database and verify session exists
	{
		db, err := New(ctx, Config{Path: dbPath})
		if err != nil {
			t.Fatalf("Failed to reopen database: %v", err)
		}
		defer db.Close()

		sessionStore := NewSessionStorage(db)

		// List all sessions
		sessions, err := sessionStore.ListAll(ctx)
		if err != nil {
			t.Fatalf("Failed to list sessions after reopen: %v", err)
		}

		if len(sessions) != 1 {
			t.Errorf("Expected 1 session after reopen, got %d", len(sessions))
		}

		if sessions[0].Title != "Persistent Session" {
			t.Errorf("Expected title 'Persistent Session', got '%s'", sessions[0].Title)
		}

		t.Logf("Successfully retrieved session after reopen: %s", sessions[0].ID)
	}
}

// TestSessionComplexWorkflow tests a realistic workflow
func TestSessionComplexWorkflow(t *testing.T) {
	db, dbPath := setupIntegrationDB(t)
	defer db.Close()

	ctx := context.Background()
	projectStore := NewProjectStorage(db)
	sessionStore := NewSessionStorage(db)

	// 1. Create a project
	project, err := projectStore.Create(ctx, Project{
		ID:       "my-project",
		Worktree: "/home/user/myproject",
		Name:     "My Project",
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	t.Logf("Created project: %s at %s", project.ID, project.Worktree)

	// 2. Create multiple sessions
	sessionTitles := []string{"Session A", "Session B", "Session C"}
	var sessionIDs []string
	for _, title := range sessionTitles {
		session, err := sessionStore.Create(ctx, Session{
			ProjectID: project.ID,
			Directory: "/home/user/myproject",
			Title:     title,
		})
		if err != nil {
			t.Fatalf("Failed to create session '%s': %v", title, err)
		}
		sessionIDs = append(sessionIDs, session.ID)
		t.Logf("Created session: %s (%s)", session.ID, title)
		time.Sleep(2 * time.Millisecond) // Ensure different timestamps
	}

	// 3. List all sessions for the project
	sessions, err := sessionStore.List(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
	t.Logf("Listed %d sessions for project", len(sessions))

	// 4. Get a specific session
	sessionB, err := sessionStore.Get(ctx, sessionIDs[1])
	if err != nil {
		t.Fatalf("Failed to get session B: %v", err)
	}
	if sessionB.Title != "Session B" {
		t.Errorf("Expected 'Session B', got '%s'", sessionB.Title)
	}
	t.Logf("Retrieved session: %s", sessionB.Title)

	// 5. Update a session
	sessionB.Title = "Updated Session B"
	sessionB.SummaryAdditions = 100
	sessionB.SummaryDeletions = 50
	err = sessionStore.Update(ctx, *sessionB)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	// Verify update
	updated, err := sessionStore.Get(ctx, sessionIDs[1])
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}
	if updated.Title != "Updated Session B" {
		t.Errorf("Expected updated title, got '%s'", updated.Title)
	}
	if updated.SummaryAdditions != 100 {
		t.Errorf("Expected 100 additions, got %d", updated.SummaryAdditions)
	}
	t.Logf("Updated session successfully")

	// 6. Archive a session
	err = sessionStore.Archive(ctx, sessionIDs[0])
	if err != nil {
		t.Fatalf("Failed to archive session: %v", err)
	}

	archived, err := sessionStore.Get(ctx, sessionIDs[0])
	if err != nil {
		t.Fatalf("Failed to get archived session: %v", err)
	}
	if archived.TimeArchived == 0 {
		t.Error("Expected TimeArchived to be set")
	}
	t.Logf("Archived session successfully")

	// 7. Verify database file exists and has content
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Failed to stat database: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Database file is empty")
	}
	t.Logf("Database file size: %d bytes", info.Size())

	// 8. Verify total count
	allSessions, err := sessionStore.ListAll(ctx)
	if err != nil {
		t.Fatalf("Failed to list all sessions: %v", err)
	}
	if len(allSessions) != 3 {
		t.Errorf("Expected 3 total sessions, got %d", len(allSessions))
	}

	t.Logf("Complex workflow completed successfully with %d sessions", len(allSessions))
}

// TestSessionWithJSONFields tests sessions with complex JSON fields
func TestSessionWithJSONFields(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()

	project := createTestProject(t, db, "test-project", "/test/worktree", "Test Project")

	ctx := context.Background()
	sessionStore := NewSessionStorage(db)

	// Create session with JSON fields
	session, err := sessionStore.Create(ctx, Session{
		ProjectID: project.ID,
		Directory: "/test/dir",
		Title:     "Session with JSON",
		SummaryDiffs: []FileDiff{
			{Path: "file1.go", Additions: 10, Deletions: 5},
			{Path: "file2.go", Additions: 20, Deletions: 10},
			{Path: "file3.go", Additions: 5, Deletions: 0},
		},
		Permission: &Ruleset{
			Allow: []string{"read", "write", "execute"},
			Deny:  []string{"delete"},
		},
		Revert: &RevertInfo{
			MessageID: "msg-123",
			PartID:    "part-456",
			Snapshot:  "snap-789",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create session with JSON fields: %v", err)
	}

	// Retrieve and verify
	retrieved, err := sessionStore.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	// Check SummaryDiffs
	if len(retrieved.SummaryDiffs) != 3 {
		t.Errorf("Expected 3 diffs, got %d", len(retrieved.SummaryDiffs))
	}
	if retrieved.SummaryDiffs[0].Path != "file1.go" {
		t.Errorf("Expected path 'file1.go', got '%s'", retrieved.SummaryDiffs[0].Path)
	}
	if retrieved.SummaryDiffs[0].Additions != 10 {
		t.Errorf("Expected 10 additions, got %d", retrieved.SummaryDiffs[0].Additions)
	}

	// Check Permission
	if retrieved.Permission == nil {
		t.Error("Permission should not be nil")
	} else {
		if len(retrieved.Permission.Allow) != 3 {
			t.Errorf("Expected 3 allow rules, got %d", len(retrieved.Permission.Allow))
		}
		if len(retrieved.Permission.Deny) != 1 {
			t.Errorf("Expected 1 deny rule, got %d", len(retrieved.Permission.Deny))
		}
	}

	// Check Revert
	if retrieved.Revert == nil {
		t.Error("Revert should not be nil")
	} else {
		if retrieved.Revert.MessageID != "msg-123" {
			t.Errorf("Expected MessageID 'msg-123', got '%s'", retrieved.Revert.MessageID)
		}
	}

	t.Logf("Session with JSON fields persisted and retrieved successfully")
}

// TestSessionForkIntegration verifies session forking works correctly
func TestSessionForkIntegration(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()

	project := createTestProject(t, db, "test-project", "/test/worktree", "Test Project")

	ctx := context.Background()
	sessionStore := NewSessionStorage(db)

	// Create parent session
	parent, err := sessionStore.Create(ctx, Session{
		ProjectID:   project.ID,
		Directory:   "/test/dir",
		Title:       "Parent Session",
		Slug:        "parent-slug",
		WorkspaceID: "workspace-123",
	})
	if err != nil {
		t.Fatalf("Failed to create parent session: %v", err)
	}

	// Fork the session
	fork, err := sessionStore.Fork(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Failed to fork session: %v", err)
	}

	// Verify fork properties
	if fork.ParentID != parent.ID {
		t.Errorf("Expected ParentID '%s', got '%s'", parent.ID, fork.ParentID)
	}
	if fork.ProjectID != parent.ProjectID {
		t.Errorf("Expected ProjectID '%s', got '%s'", parent.ProjectID, fork.ProjectID)
	}
	if fork.Directory != parent.Directory {
		t.Errorf("Expected Directory '%s', got '%s'", parent.Directory, fork.Directory)
	}
	if fork.WorkspaceID != parent.WorkspaceID {
		t.Errorf("Expected WorkspaceID '%s', got '%s'", parent.WorkspaceID, fork.WorkspaceID)
	}
	if fork.ID == parent.ID {
		t.Error("Fork should have different ID from parent")
	}
	if fork.Slug != "parent-slug-fork" {
		t.Errorf("Expected slug 'parent-slug-fork', got '%s'", fork.Slug)
	}

	// Verify fork can be retrieved
	retrieved, err := sessionStore.Get(ctx, fork.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve fork: %v", err)
	}
	if retrieved.ParentID != parent.ID {
		t.Errorf("Retrieved fork has wrong ParentID: %s", retrieved.ParentID)
	}

	t.Logf("Session forked successfully: %s -> %s", parent.ID, fork.ID)
}
