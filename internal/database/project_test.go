// Package database provides tests for project storage.
package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// setupProjectTestDB creates a test database for project tests
func setupProjectTestDB(t *testing.T) (*Database, func()) {
	tmpDir, err := os.MkdirTemp("", "magicode-project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(context.Background(), Config{Path: dbPath})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

// TestProjectCreate tests project creation
func TestProjectCreate(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	store := NewProjectStorage(db)
	ctx := context.Background()

	project, err := store.Create(ctx, Project{
		Worktree: "/test/worktree",
		Name:     "Test Project",
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Check auto-generated fields
	if project.ID == "" {
		t.Error("ID should be auto-generated")
	}
	if project.Timestamps.TimeCreated == 0 {
		t.Error("TimeCreated should be set")
	}
	if project.Sandboxes == nil {
		t.Error("Sandboxes should be initialized as empty array")
	}

	// Verify can be retrieved
	retrieved, err := store.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve project: %v", err)
	}
	if retrieved.ID != project.ID {
		t.Errorf("Expected ID %s, got %s", project.ID, retrieved.ID)
	}
}

// TestProjectCreateWithCustomID tests project creation with custom ID
func TestProjectCreateWithCustomID(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	store := NewProjectStorage(db)
	ctx := context.Background()

	project, err := store.Create(ctx, Project{
		ID:       "custom-project-id",
		Worktree: "/test/worktree",
		Name:     "Custom Project",
		VCS:      "git",
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	if project.ID != "custom-project-id" {
		t.Errorf("Expected custom ID, got %s", project.ID)
	}
	if project.VCS != "git" {
		t.Errorf("Expected VCS 'git', got '%s'", project.VCS)
	}
}

// TestProjectGetByWorktree tests project retrieval by worktree
func TestProjectGetByWorktree(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	store := NewProjectStorage(db)
	ctx := context.Background()

	// Create project
	_, err := store.Create(ctx, Project{
		Worktree: "/unique/worktree/path",
		Name:     "Worktree Test",
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Get by worktree
	project, err := store.GetByWorktree(ctx, "/unique/worktree/path")
	if err != nil {
		t.Fatalf("Failed to get project by worktree: %v", err)
	}

	if project.Name != "Worktree Test" {
		t.Errorf("Expected name 'Worktree Test', got '%s'", project.Name)
	}
}

// TestProjectList tests project listing
func TestProjectList(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	store := NewProjectStorage(db)
	ctx := context.Background()

	// Create multiple projects
	for i := 0; i < 3; i++ {
		_, err := store.Create(ctx, Project{
			Worktree: fmt.Sprintf("/test/worktree%d", i),
			Name:     fmt.Sprintf("Project %d", i),
		})
		if err != nil {
			t.Fatalf("Failed to create project %d: %v", i, err)
		}
	}

	// List projects
	projects, err := store.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(projects))
	}
}

// TestProjectUpdate tests project update
func TestProjectUpdate(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	store := NewProjectStorage(db)
	ctx := context.Background()

	// Create project
	project, err := store.Create(ctx, Project{
		Worktree: "/test/worktree",
		Name:     "Original Name",
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Update project
	project.Name = "Updated Name"
	project.VCS = "git"
	project.Sandboxes = []string{"sandbox1", "sandbox2"}
	project.Commands = &Commands{Start: "npm start"}

	err = store.Update(ctx, *project)
	if err != nil {
		t.Fatalf("Failed to update project: %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to get updated project: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected 'Updated Name', got '%s'", retrieved.Name)
	}
	if retrieved.VCS != "git" {
		t.Errorf("Expected VCS 'git', got '%s'", retrieved.VCS)
	}
	if len(retrieved.Sandboxes) != 2 {
		t.Errorf("Expected 2 sandboxes, got %d", len(retrieved.Sandboxes))
	}
	if retrieved.Commands == nil {
		t.Error("Commands should not be nil")
	}
	if retrieved.Commands.Start != "npm start" {
		t.Errorf("Expected start command 'npm start', got '%s'", retrieved.Commands.Start)
	}
}

// TestProjectDelete tests project deletion
func TestProjectDelete(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	store := NewProjectStorage(db)
	ctx := context.Background()

	// Create project
	project, err := store.Create(ctx, Project{
		Worktree: "/test/worktree",
		Name:     "To Delete",
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Delete project
	err = store.Delete(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	// Verify deletion
	_, err = store.Get(ctx, project.ID)
	if err == nil {
		t.Error("Project should not exist after deletion")
	}
}

// TestProjectInitialize tests project initialization
func TestProjectInitialize(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	store := NewProjectStorage(db)
	ctx := context.Background()

	// Create project
	project, err := store.Create(ctx, Project{
		Worktree: "/test/worktree",
		Name:     "To Initialize",
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Check not initialized
	if project.TimeInitialized != 0 {
		t.Error("New project should not be initialized")
	}

	// Initialize project
	err = store.Initialize(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Verify initialization
	retrieved, err := store.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to get initialized project: %v", err)
	}

	if retrieved.TimeInitialized == 0 {
		t.Error("TimeInitialized should be set after Initialize")
	}
}

// TestProjectJSONFields tests JSON field handling
func TestProjectJSONFields(t *testing.T) {
	db, cleanup := setupProjectTestDB(t)
	defer cleanup()

	store := NewProjectStorage(db)
	ctx := context.Background()

	// Create project with JSON fields
	project, err := store.Create(ctx, Project{
		Worktree: "/test/worktree",
		Name:     "JSON Test",
		Sandboxes: []string{"dev", "staging", "prod"},
		Commands: &Commands{
			Start: "npm run dev",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Retrieve and check JSON fields
	retrieved, err := store.Get(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve project: %v", err)
	}

	if len(retrieved.Sandboxes) != 3 {
		t.Errorf("Expected 3 sandboxes, got %d", len(retrieved.Sandboxes))
	}
	if retrieved.Sandboxes[0] != "dev" {
		t.Errorf("Expected first sandbox 'dev', got '%s'", retrieved.Sandboxes[0])
	}
	if retrieved.Commands == nil {
		t.Error("Commands should not be nil")
	}
	if retrieved.Commands.Start != "npm run dev" {
		t.Errorf("Expected start command 'npm run dev', got '%s'", retrieved.Commands.Start)
	}
}