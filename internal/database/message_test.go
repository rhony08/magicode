// Package database provides tests for message storage.
package database

import (
	"context"
	"fmt"
	"testing"
)

// TestMessageCreate tests message creation
func TestMessageCreate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a session first
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create message
	store := NewMessageStorage(db)
	ctx := context.Background()

	message, err := store.Create(ctx, Message{
		SessionID: session.ID,
		Data: MessageInfo{
			Role:    "user",
			Content: "Hello, world!",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Check auto-generated fields
	if message.ID == "" {
		t.Error("ID should be auto-generated")
	}
	if message.Timestamps.TimeCreated == 0 {
		t.Error("TimeCreated should be set")
	}

	// Verify can be retrieved
	retrieved, err := store.Get(ctx, message.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve message: %v", err)
	}
	if retrieved.Data.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", retrieved.Data.Role)
	}
	if retrieved.Data.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", retrieved.Data.Content)
	}
}

// TestMessageList tests message listing
func TestMessageList(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create session
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create multiple messages
	store := NewMessageStorage(db)
	ctx := context.Background()

	roles := []string{"user", "assistant", "user"}
	for i, role := range roles {
		_, err := store.Create(ctx, Message{
			SessionID: session.ID,
			Data: MessageInfo{
				Role:    role,
				Content: fmt.Sprintf("Message %d", i),
			},
		})
		if err != nil {
			t.Fatalf("Failed to create message %d: %v", i, err)
		}
	}

	// List messages
	messages, err := store.List(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Check order (chronological)
	if messages[0].Data.Role != "user" {
		t.Errorf("First message should be 'user', got '%s'", messages[0].Data.Role)
	}
	if messages[2].Data.Role != "user" {
		t.Errorf("Third message should be 'user', got '%s'", messages[2].Data.Role)
	}
}

// TestMessageListWithLimit tests message listing with limit
func TestMessageListWithLimit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create session
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create 5 messages
	store := NewMessageStorage(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := store.Create(ctx, Message{
			SessionID: session.ID,
			Data: MessageInfo{
				Role:    "user",
				Content: fmt.Sprintf("Message %d", i),
			},
		})
		if err != nil {
			t.Fatalf("Failed to create message %d: %v", i, err)
		}
	}

	// List with limit
	messages, err := store.ListWithLimit(ctx, session.ID, 3)
	if err != nil {
		t.Fatalf("Failed to list messages with limit: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Should be last 3 in chronological order
	if messages[0].Data.Content != "Message 2" {
		t.Errorf("First should be 'Message 2', got '%s'", messages[0].Data.Content)
	}
	if messages[2].Data.Content != "Message 4" {
		t.Errorf("Last should be 'Message 4', got '%s'", messages[2].Data.Content)
	}
}

// TestMessageUpdate tests message update
func TestMessageUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create session and message
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	store := NewMessageStorage(db)
	ctx := context.Background()

	message, err := store.Create(ctx, Message{
		SessionID: session.ID,
		Data: MessageInfo{
			Role:    "assistant",
			Content: "Original response",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Update message
	message.Data.Content = "Updated response"
	message.Data.Cost = 1000
	message.Data.Tokens = 500

	err = store.Update(ctx, *message)
	if err != nil {
		t.Fatalf("Failed to update message: %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, message.ID)
	if err != nil {
		t.Fatalf("Failed to get updated message: %v", err)
	}

	if retrieved.Data.Content != "Updated response" {
		t.Errorf("Expected 'Updated response', got '%s'", retrieved.Data.Content)
	}
	if retrieved.Data.Cost != 1000 {
		t.Errorf("Expected cost 1000, got %d", retrieved.Data.Cost)
	}
	if retrieved.Data.Tokens != 500 {
		t.Errorf("Expected tokens 500, got %d", retrieved.Data.Tokens)
	}
}

// TestMessageDelete tests message deletion
func TestMessageDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create session and message
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	store := NewMessageStorage(db)
	ctx := context.Background()

	message, err := store.Create(ctx, Message{
		SessionID: session.ID,
		Data: MessageInfo{
			Role:    "user",
			Content: "To delete",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Delete message
	err = store.Delete(ctx, message.ID)
	if err != nil {
		t.Fatalf("Failed to delete message: %v", err)
	}

	// Verify deletion
	_, err = store.Get(ctx, message.ID)
	if err == nil {
		t.Error("Message should not exist after deletion")
	}
}

// TestPartCreate tests part creation
func TestPartCreate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create session and message
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	messageStore := NewMessageStorage(db)
	message, err := messageStore.Create(context.Background(), Message{
		SessionID: session.ID,
		Data: MessageInfo{
			Role: "assistant",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Create part
	store := NewPartStorage(db)
	ctx := context.Background()

	part, err := store.Create(ctx, Part{
		MessageID: message.ID,
		SessionID: session.ID,
		Data: PartData{
			Type:   "text",
			Text:   "Hello from part!",
			Status: "success",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create part: %v", err)
	}

	// Check auto-generated fields
	if part.ID == "" {
		t.Error("ID should be auto-generated")
	}

	// Verify can be retrieved
	retrieved, err := store.Get(ctx, part.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve part: %v", err)
	}
	if retrieved.Data.Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", retrieved.Data.Type)
	}
	if retrieved.Data.Text != "Hello from part!" {
		t.Errorf("Expected text 'Hello from part!', got '%s'", retrieved.Data.Text)
	}
}

// TestPartListByMessage tests part listing by message
func TestPartListByMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create session and message
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	messageStore := NewMessageStorage(db)
	message, err := messageStore.Create(context.Background(), Message{
		SessionID: session.ID,
		Data: MessageInfo{
			Role: "assistant",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Create multiple parts
	store := NewPartStorage(db)
	ctx := context.Background()

	types := []string{"text", "tool_use", "tool_result"}
	for i, typ := range types {
		_, err := store.Create(ctx, Part{
			MessageID: message.ID,
			SessionID: session.ID,
			Data: PartData{
				Type:   typ,
				Text:   fmt.Sprintf("Part %d", i),
				Status: "success",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create part %d: %v", i, err)
		}
	}

	// List parts by message
	parts, err := store.ListByMessage(ctx, message.ID)
	if err != nil {
		t.Fatalf("Failed to list parts: %v", err)
	}

	if len(parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(parts))
	}

	// Check order
	if parts[0].Data.Type != "text" {
		t.Errorf("First part should be 'text', got '%s'", parts[0].Data.Type)
	}
}

// TestPartUpdate tests part update
func TestPartUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create session, message, and part
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	messageStore := NewMessageStorage(db)
	message, err := messageStore.Create(context.Background(), Message{
		SessionID: session.ID,
		Data: MessageInfo{
			Role: "assistant",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	store := NewPartStorage(db)
	ctx := context.Background()

	part, err := store.Create(ctx, Part{
		MessageID: message.ID,
		SessionID: session.ID,
		Data: PartData{
			Type:   "tool_use",
			ToolName: "bash",
			Status: "pending",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create part: %v", err)
	}

	// Update part
	part.Data.Status = "running"
	part.Data.ToolInput = map[string]any{"command": "ls -la"}

	err = store.Update(ctx, *part)
	if err != nil {
		t.Fatalf("Failed to update part: %v", err)
	}

	// Verify update
	retrieved, err := store.Get(ctx, part.ID)
	if err != nil {
		t.Fatalf("Failed to get updated part: %v", err)
	}

	if retrieved.Data.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", retrieved.Data.Status)
	}
	if retrieved.Data.ToolInput["command"] != "ls -la" {
		t.Errorf("Expected command 'ls -la', got '%v'", retrieved.Data.ToolInput["command"])
	}
}

// TestPartDelete tests part deletion
func TestPartDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create session, message, and part
	sessionStore := NewSessionStorage(db)
	session, err := sessionStore.Create(context.Background(), Session{
		ProjectID: "test-project",
		Directory: "/test/dir",
		Title:     "Test Session",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	messageStore := NewMessageStorage(db)
	message, err := messageStore.Create(context.Background(), Message{
		SessionID: session.ID,
		Data: MessageInfo{
			Role: "assistant",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	store := NewPartStorage(db)
	ctx := context.Background()

	part, err := store.Create(ctx, Part{
		MessageID: message.ID,
		SessionID: session.ID,
		Data: PartData{
			Type:   "text",
			Text:   "To delete",
			Status: "success",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create part: %v", err)
	}

	// Delete part
	err = store.Delete(ctx, part.ID)
	if err != nil {
		t.Fatalf("Failed to delete part: %v", err)
	}

	// Verify deletion
	_, err = store.Get(ctx, part.ID)
	if err == nil {
		t.Error("Part should not exist after deletion")
	}
}