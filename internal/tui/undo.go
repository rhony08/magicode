// Package tui provides undo/redo functionality for message history.
// This file implements an undo/redo stack for session messages.
package tui

import (
	"fmt"
	"sync"

	"github.com/rhony08/magicode/internal/tui/types"
)

// UndoRedoManager manages undo/redo state for messages
type UndoRedoManager struct {
	mu sync.RWMutex

	// Stack of message states for undo
	undoStack []MessageState

	// Stack of message states for redo
	redoStack []MessageState

	// Maximum stack size
	maxSize int
}

// MessageState represents a message snapshot for undo/redo
type MessageState struct {
	// Message ID for reference
	MessageID string

	// Original content
	Content string

	// Role (user, assistant)
	Role string

	// Index in the message array
	Index int

	// Action type (add, edit, delete)
	Action string
}

// NewUndoRedoManager creates a new undo/redo manager
func NewUndoRedoManager() *UndoRedoManager {
	return &UndoRedoManager{
		undoStack: make([]MessageState, 0),
		redoStack: make([]MessageState, 0),
		maxSize:   50,
	}
}

// SetMaxSize sets the maximum stack size
func (m *UndoRedoManager) SetMaxSize(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxSize = size
}

// PushUndo adds a state to the undo stack
func (m *UndoRedoManager) PushUndo(state MessageState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to undo stack
	m.undoStack = append(m.undoStack, state)

	// Clear redo stack when new action is performed
	m.redoStack = make([]MessageState, 0)

	// Trim if too large
	if len(m.undoStack) > m.maxSize {
		m.undoStack = m.undoStack[len(m.undoStack)-m.maxSize:]
	}
}

// CanUndo returns true if undo is available
func (m *UndoRedoManager) CanUndo() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.undoStack) > 0
}

// CanRedo returns true if redo is available
func (m *UndoRedoManager) CanRedo() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.redoStack) > 0
}

// Undo pops the last state from undo stack and moves it to redo stack
func (m *UndoRedoManager) Undo() (MessageState, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.undoStack) == 0 {
		return MessageState{}, false
	}

	// Pop from undo stack
	state := m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]

	// Push to redo stack
	m.redoStack = append(m.redoStack, state)

	return state, true
}

// Redo pops the last state from redo stack and moves it to undo stack
func (m *UndoRedoManager) Redo() (MessageState, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.redoStack) == 0 {
		return MessageState{}, false
	}

	// Pop from redo stack
	state := m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]

	// Push to undo stack
	m.undoStack = append(m.undoStack, state)

	return state, true
}

// GetUndoStack returns a copy of the undo stack (for display)
func (m *UndoRedoManager) GetUndoStack() []MessageState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stack := make([]MessageState, len(m.undoStack))
	copy(stack, m.undoStack)
	return stack
}

// GetRedoStack returns a copy of the redo stack (for display)
func (m *UndoRedoManager) GetRedoStack() []MessageState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stack := make([]MessageState, len(m.redoStack))
	copy(stack, m.redoStack)
	return stack
}

// Clear clears both stacks
func (m *UndoRedoManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.undoStack = make([]MessageState, 0)
	m.redoStack = make([]MessageState, 0)
}

// GetUndoCount returns the number of available undo actions
func (m *UndoRedoManager) GetUndoCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.undoStack)
}

// GetRedoCount returns the number of available redo actions
func (m *UndoRedoManager) GetRedoCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.redoStack)
}

// MessageHistoryTracker tracks message history for a session
type MessageHistoryTracker struct {
	mu sync.RWMutex

	// Session ID
	sessionID string

	// Message snapshots indexed by message ID
	snapshots map[string]MessageSnapshot

	// Current state of messages
	messages []types.Message

	// Undo/redo manager
	undoRedo *UndoRedoManager
}

// MessageSnapshot captures the state of messages at a point in time
type MessageSnapshot struct {
	Messages []types.Message
	Cursor   int64 // Timestamp cursor
}

// NewMessageHistoryTracker creates a new message history tracker
func NewMessageHistoryTracker(sessionID string) *MessageHistoryTracker {
	return &MessageHistoryTracker{
		sessionID: sessionID,
		snapshots: make(map[string]MessageSnapshot),
		messages:  make([]types.Message, 0),
		undoRedo:  NewUndoRedoManager(),
	}
}

// AddSnapshot captures the current message state
func (t *MessageHistoryTracker) AddSnapshot(cursor int64) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create snapshot ID based on timestamp
	snapshotID := fmt.Sprintf("%s_%d", t.sessionID, cursor)

	// Copy current messages
	messages := make([]types.Message, len(t.messages))
	copy(messages, t.messages)

	t.snapshots[snapshotID] = MessageSnapshot{
		Messages: messages,
		Cursor:   cursor,
	}

	return snapshotID
}

// RestoreSnapshot restores messages from a snapshot
func (t *MessageHistoryTracker) RestoreSnapshot(snapshotID string) ([]types.Message, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	snapshot, ok := t.snapshots[snapshotID]
	if !ok {
		return nil, false
	}

	// Restore messages
	t.messages = make([]types.Message, len(snapshot.Messages))
	copy(t.messages, snapshot.Messages)

	return t.messages, true
}

// UpdateMessages updates the current message list
func (t *MessageHistoryTracker) UpdateMessages(messages []types.Message) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.messages = make([]types.Message, len(messages))
	copy(t.messages, messages)
}

// GetMessages returns the current messages
func (t *MessageHistoryTracker) GetMessages() []types.Message {
	t.mu.RLock()
	defer t.mu.RUnlock()

	messages := make([]types.Message, len(t.messages))
	copy(messages, t.messages)
	return messages
}

// GetUndoRedoManager returns the undo/redo manager
func (t *MessageHistoryTracker) GetUndoRedoManager() *UndoRedoManager {
	return t.undoRedo
}

// Clear clears all history
func (t *MessageHistoryTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.snapshots = make(map[string]MessageSnapshot)
	t.messages = make([]types.Message, 0)
	t.undoRedo.Clear()
}
