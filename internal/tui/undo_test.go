// Package tui provides undo/redo functionality for message history.
// This file contains tests for the undo/redo manager.
package tui

import (
	"testing"

	"github.com/rhony08/magicode/internal/tui/types"
)

func TestNewUndoRedoManager(t *testing.T) {
	m := NewUndoRedoManager()

	if m == nil {
		t.Fatal("NewUndoRedoManager() returned nil")
	}

	if m.maxSize != 50 {
		t.Errorf("maxSize = %d, want 50", m.maxSize)
	}

	if len(m.undoStack) != 0 {
		t.Errorf("undoStack length = %d, want 0", len(m.undoStack))
	}

	if len(m.redoStack) != 0 {
		t.Errorf("redoStack length = %d, want 0", len(m.redoStack))
	}
}

func TestUndoRedoManager_SetMaxSize(t *testing.T) {
	m := NewUndoRedoManager()
	m.SetMaxSize(100)

	if m.maxSize != 100 {
		t.Errorf("maxSize = %d, want 100", m.maxSize)
	}
}

func TestUndoRedoManager_PushUndo(t *testing.T) {
	m := NewUndoRedoManager()

	state1 := MessageState{MessageID: "1", Content: "Hello", Action: "add"}
	state2 := MessageState{MessageID: "2", Content: "World", Action: "add"}

	m.PushUndo(state1)
	if len(m.undoStack) != 1 {
		t.Errorf("undoStack length = %d, want 1", len(m.undoStack))
	}

	m.PushUndo(state2)
	if len(m.undoStack) != 2 {
		t.Errorf("undoStack length = %d, want 2", len(m.undoStack))
	}
}

func TestUndoRedoManager_PushUndoClearsRedo(t *testing.T) {
	m := NewUndoRedoManager()

	state1 := MessageState{MessageID: "1", Content: "Hello", Action: "add"}
	state2 := MessageState{MessageID: "2", Content: "World", Action: "add"}

	// Add two states
	m.PushUndo(state1)
	m.PushUndo(state2)

	// Undo one (moves to redo stack)
	m.Undo()

	if len(m.redoStack) != 1 {
		t.Errorf("redoStack length = %d, want 1", len(m.redoStack))
	}

	// Push new state - should clear redo stack
	state3 := MessageState{MessageID: "3", Content: "Test", Action: "add"}
	m.PushUndo(state3)

	if len(m.redoStack) != 0 {
		t.Errorf("redoStack should be cleared, got length %d", len(m.redoStack))
	}
}

func TestUndoRedoManager_Undo(t *testing.T) {
	m := NewUndoRedoManager()

	// Undo on empty stack should return false
	_, ok := m.Undo()
	if ok {
		t.Error("Undo() on empty stack should return false")
	}

	// Push a state and undo it
	state := MessageState{MessageID: "1", Content: "Hello", Action: "add"}
	m.PushUndo(state)

	result, ok := m.Undo()
	if !ok {
		t.Error("Undo() should return true")
	}

	if result.MessageID != "1" {
		t.Errorf("Undo() returned MessageID = %s, want 1", result.MessageID)
	}

	// State should be in redo stack
	if len(m.redoStack) != 1 {
		t.Errorf("redoStack length = %d, want 1", len(m.redoStack))
	}
}

func TestUndoRedoManager_Redo(t *testing.T) {
	m := NewUndoRedoManager()

	// Redo on empty stack should return false
	_, ok := m.Redo()
	if ok {
		t.Error("Redo() on empty stack should return false")
	}

	// Push, undo, then redo
	state := MessageState{MessageID: "1", Content: "Hello", Action: "add"}
	m.PushUndo(state)
	m.Undo()

	result, ok := m.Redo()
	if !ok {
		t.Error("Redo() should return true")
	}

	if result.MessageID != "1" {
		t.Errorf("Redo() returned MessageID = %s, want 1", result.MessageID)
	}

	// State should be back in undo stack
	if len(m.undoStack) != 1 {
		t.Errorf("undoStack length = %d, want 1", len(m.undoStack))
	}
}

func TestUndoRedoManager_CanUndo(t *testing.T) {
	m := NewUndoRedoManager()

	if m.CanUndo() {
		t.Error("CanUndo() should return false for empty stack")
	}

	m.PushUndo(MessageState{MessageID: "1", Content: "Hello", Action: "add"})

	if !m.CanUndo() {
		t.Error("CanUndo() should return true after PushUndo")
	}

	m.Undo()

	if m.CanUndo() {
		t.Error("CanUndo() should return false after Undo")
	}
}

func TestUndoRedoManager_CanRedo(t *testing.T) {
	m := NewUndoRedoManager()

	if m.CanRedo() {
		t.Error("CanRedo() should return false for empty stack")
	}

	m.PushUndo(MessageState{MessageID: "1", Content: "Hello", Action: "add"})
	m.Undo()

	if !m.CanRedo() {
		t.Error("CanRedo() should return true after Undo")
	}

	m.Redo()

	if m.CanRedo() {
		t.Error("CanRedo() should return false after Redo")
	}
}

func TestUndoRedoManager_Clear(t *testing.T) {
	m := NewUndoRedoManager()

	// Add some states
	m.PushUndo(MessageState{MessageID: "1", Content: "Hello", Action: "add"})
	m.PushUndo(MessageState{MessageID: "2", Content: "World", Action: "add"})
	m.Undo()

	// Clear
	m.Clear()

	if len(m.undoStack) != 0 {
		t.Errorf("undoStack length = %d, want 0 after Clear", len(m.undoStack))
	}

	if len(m.redoStack) != 0 {
		t.Errorf("redoStack length = %d, want 0 after Clear", len(m.redoStack))
	}
}

func TestUndoRedoManager_GetCounts(t *testing.T) {
	m := NewUndoRedoManager()

	if m.GetUndoCount() != 0 {
		t.Errorf("GetUndoCount() = %d, want 0", m.GetUndoCount())
	}

	if m.GetRedoCount() != 0 {
		t.Errorf("GetRedoCount() = %d, want 0", m.GetRedoCount())
	}

	m.PushUndo(MessageState{MessageID: "1", Content: "Hello", Action: "add"})
	m.Undo()

	if m.GetUndoCount() != 0 {
		t.Errorf("GetUndoCount() = %d, want 0", m.GetUndoCount())
	}

	if m.GetRedoCount() != 1 {
		t.Errorf("GetRedoCount() = %d, want 1", m.GetRedoCount())
	}
}

func TestUndoRedoManager_MaxSize(t *testing.T) {
	m := NewUndoRedoManager()
	m.SetMaxSize(3)

	// Push 5 states
	for i := 0; i < 5; i++ {
		m.PushUndo(MessageState{MessageID: string(rune('0' + i)), Content: "Test", Action: "add"})
	}

	// Should only keep last 3
	if len(m.undoStack) != 3 {
		t.Errorf("undoStack length = %d, want 3", len(m.undoStack))
	}

	// Should have states 2, 3, 4 (0, 1 were dropped)
	if m.undoStack[0].MessageID != "2" {
		t.Errorf("First item MessageID = %s, want 2", m.undoStack[0].MessageID)
	}
}

func TestNewMessageHistoryTracker(t *testing.T) {
	tracker := NewMessageHistoryTracker("session-1")

	if tracker == nil {
		t.Fatal("NewMessageHistoryTracker() returned nil")
	}

	if tracker.sessionID != "session-1" {
		t.Errorf("sessionID = %s, want session-1", tracker.sessionID)
	}

	if len(tracker.snapshots) != 0 {
		t.Errorf("snapshots length = %d, want 0", len(tracker.snapshots))
	}

	if tracker.undoRedo == nil {
		t.Error("undoRedo should not be nil")
	}
}

func TestMessageHistoryTracker_UpdateAndGetMessages(t *testing.T) {
	tracker := NewMessageHistoryTracker("session-1")

	messages := []types.Message{
		{ID: "1", Role: "user", Content: "Hello"},
		{ID: "2", Role: "assistant", Content: "Hi there"},
	}

	tracker.UpdateMessages(messages)

	retrieved := tracker.GetMessages()

	if len(retrieved) != 2 {
		t.Errorf("GetMessages() returned %d messages, want 2", len(retrieved))
	}

	if retrieved[0].ID != "1" {
		t.Errorf("First message ID = %s, want 1", retrieved[0].ID)
	}
}

func TestMessageHistoryTracker_AddSnapshot(t *testing.T) {
	tracker := NewMessageHistoryTracker("session-1")

	messages := []types.Message{
		{ID: "1", Role: "user", Content: "Hello"},
	}

	tracker.UpdateMessages(messages)

	snapshotID := tracker.AddSnapshot(12345)

	if snapshotID == "" {
		t.Error("AddSnapshot() returned empty ID")
	}

	// Should be able to restore
	restored, ok := tracker.RestoreSnapshot(snapshotID)
	if !ok {
		t.Error("RestoreSnapshot() should return true")
	}

	if len(restored) != 1 {
		t.Errorf("Restored %d messages, want 1", len(restored))
	}
}

func TestMessageHistoryTracker_Clear(t *testing.T) {
	tracker := NewMessageHistoryTracker("session-1")

	messages := []types.Message{
		{ID: "1", Role: "user", Content: "Hello"},
	}

	tracker.UpdateMessages(messages)
	tracker.AddSnapshot(12345)
	tracker.GetUndoRedoManager().PushUndo(MessageState{MessageID: "1", Action: "add"})

	tracker.Clear()

	if len(tracker.GetMessages()) != 0 {
		t.Error("Messages should be cleared")
	}

	if tracker.GetUndoRedoManager().CanUndo() {
		// Should have been cleared
		t.Error("Undo stack should be cleared")
	}
}

func TestMessageHistoryTracker_GetUndoRedoManager(t *testing.T) {
	tracker := NewMessageHistoryTracker("session-1")

	manager := tracker.GetUndoRedoManager()

	if manager == nil {
		t.Error("GetUndoRedoManager() returned nil")
	}
}

func TestMessageHistoryTracker_RestoreNonExistentSnapshot(t *testing.T) {
	tracker := NewMessageHistoryTracker("session-1")

	_, ok := tracker.RestoreSnapshot("non-existent")
	if ok {
		t.Error("RestoreSnapshot() should return false for non-existent snapshot")
	}
}
