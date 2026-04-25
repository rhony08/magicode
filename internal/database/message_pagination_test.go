// Package database provides tests for message pagination.
package database

import (
	"context"
	"testing"
	"time"
)

// TestMessagePagination tests cursor-based pagination
func TestMessagePagination(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	ctx := context.Background()
	db, err := New(ctx, Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create project first
	projectStorage := NewProjectStorage(db)
	project := Project{
		ID:       "test-project-pagination",
		Worktree: "/test/worktree",
		Name:     "Pagination Test Project",
	}
	_, err = projectStorage.Create(ctx, project)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create session
	sessionStorage := NewSessionStorage(db)
	session := Session{
		ID:        "test-session-pagination",
		ProjectID: project.ID,
		Directory: "/test/dir",
		Title:     "Pagination Test",
		Version:   "1.0",
		Slug:      "pagination-test",
	}
	_, err = sessionStorage.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create many messages (more than page size)
	messageStorage := NewMessageStorage(db)
	numMessages := 100 // More than InitialMessagePageSize (80)

	for i := 0; i < numMessages; i++ {
		msg := Message{
			SessionID: session.ID,
			Data: MessageInfo{
				Role:       "user",
				ProviderID: "test-provider",
				ModelID:    "test-model",
				Time:       map[string]interface{}{"created": time.Now().Add(time.Duration(i) * time.Millisecond).UnixMilli()},
			},
		}
		_, err := messageStorage.Create(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to create message %d: %v", i, err)
		}
	}

	// Test initial load (no cursor)
	pageSize := 10
	messages, cursor, complete, err := messageStorage.ListPaginated(ctx, session.ID, pageSize, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	if len(messages) != pageSize {
		t.Errorf("Expected %d messages, got %d", pageSize, len(messages))
	}

	if complete {
		t.Error("Expected complete=false since more messages exist")
	}

	if cursor == 0 {
		t.Error("Expected cursor to be set")
	}

	t.Logf("First page: %d messages, cursor: %d, complete: %v", len(messages), cursor, complete)

	// Test load more (with cursor)
	moreMessages, _, nextComplete, err := messageStorage.ListPaginated(ctx, session.ID, pageSize, cursor)
	if err != nil {
		t.Fatalf("Failed to load more messages: %v", err)
	}

	if len(moreMessages) == 0 {
		t.Error("Expected more messages to be loaded")
	}

	// Verify no duplicate messages
	firstPageIDs := make(map[string]bool)
	for _, m := range messages {
		firstPageIDs[m.ID] = true
	}

	for _, m := range moreMessages {
		if firstPageIDs[m.ID] {
			t.Errorf("Duplicate message found: %s", m.ID)
		}
	}

	t.Logf("Second page: %d messages, complete: %v", len(moreMessages), nextComplete)

	// Test count
	count, err := messageStorage.Count(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}

	if count != numMessages {
		t.Errorf("Expected count %d, got %d", numMessages, count)
	}
}

// TestMessagePaginationComplete tests pagination when all messages are loaded
func TestMessagePaginationComplete(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	ctx := context.Background()
	db, err := New(ctx, Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create project first
	projectStorage := NewProjectStorage(db)
	project := Project{
		ID:       "test-project-complete",
		Worktree: "/test/worktree",
		Name:     "Complete Test Project",
	}
	_, err = projectStorage.Create(ctx, project)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create session
	sessionStorage := NewSessionStorage(db)
	session := Session{
		ID:        "test-session-complete",
		ProjectID: project.ID,
		Directory: "/test/dir",
		Title:     "Complete Test",
		Version:   "1.0",
		Slug:      "complete-test",
	}
	_, err = sessionStorage.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create fewer messages than page size
	messageStorage := NewMessageStorage(db)
	numMessages := 5

	for i := 0; i < numMessages; i++ {
		msg := Message{
			SessionID: session.ID,
			Data: MessageInfo{
				Role: "user",
				Time: map[string]interface{}{"created": time.Now().Add(time.Duration(i) * time.Millisecond).UnixMilli()},
			},
		}
		_, err := messageStorage.Create(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to create message %d: %v", i, err)
		}
	}

	// Load with page size larger than message count
	pageSize := 10
	messages, _, complete, err := messageStorage.ListPaginated(ctx, session.ID, pageSize, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	if len(messages) != numMessages {
		t.Errorf("Expected %d messages, got %d", numMessages, len(messages))
	}

	if !complete {
		t.Error("Expected complete=true since all messages loaded")
	}

	// Test count
	count, err := messageStorage.Count(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}

	if count != numMessages {
		t.Errorf("Expected count %d, got %d", numMessages, count)
	}
}

// TestMessagePaginationEmpty tests pagination with no messages
func TestMessagePaginationEmpty(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	ctx := context.Background()
	db, err := New(ctx, Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create project first
	projectStorage := NewProjectStorage(db)
	project := Project{
		ID:       "test-project-empty",
		Worktree: "/test/worktree",
		Name:     "Empty Test Project",
	}
	_, err = projectStorage.Create(ctx, project)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create session without messages
	sessionStorage := NewSessionStorage(db)
	session := Session{
		ID:        "test-session-empty",
		ProjectID: project.ID,
		Directory: "/test/dir",
		Title:     "Empty Test",
		Version:   "1.0",
		Slug:      "empty-test",
	}
	_, err = sessionStorage.Create(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	messageStorage := NewMessageStorage(db)

	// Load messages
	messages, _, complete, err := messageStorage.ListPaginated(ctx, session.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(messages))
	}

	if !complete {
		t.Error("Expected complete=true for empty session")
	}

	// Test count
	count, err := messageStorage.Count(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}
