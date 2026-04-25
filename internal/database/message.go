// Package database provides message storage operations.
package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// MessageStorage provides message CRUD operations
type MessageStorage struct {
	db *Database
}

// NewMessageStorage creates a new message storage
func NewMessageStorage(db *Database) *MessageStorage {
	return &MessageStorage{db: db}
}

// Create creates a new message
func (s *MessageStorage) Create(ctx context.Context, message Message) (*Message, error) {
	if message.ID == "" {
		message.ID = uuid.New().String()
	}
	message.Timestamps = NewTimestamps()

	// Serialize JSON field
	data, err := json.Marshal(message.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
	INSERT INTO message (id, session_id, time_created, time_updated, data)
	VALUES (?, ?, ?, ?, ?)`

	err = s.db.Exec(ctx, query,
		message.ID, message.SessionID,
		message.Timestamps.TimeCreated, message.Timestamps.TimeUpdated,
		string(data),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return &message, nil
}

// Get retrieves a message by ID
func (s *MessageStorage) Get(ctx context.Context, id string) (*Message, error) {
	query := `
	SELECT id, session_id, time_created, time_updated, data
	FROM message WHERE id = ?`

	row := s.db.QueryRow(ctx, query, id)
	var message Message
	var dataStr string

	err := row.Scan(
		&message.ID, &message.SessionID,
		&message.Timestamps.TimeCreated, &message.Timestamps.TimeUpdated,
		&dataStr,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("message not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Deserialize JSON field
	if err := json.Unmarshal([]byte(dataStr), &message.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &message, nil
}

// List retrieves messages by session ID
func (s *MessageStorage) List(ctx context.Context, sessionID string) ([]Message, error) {
	query := `
	SELECT id, session_id, time_created, time_updated, data
	FROM message WHERE session_id = ? ORDER BY time_created ASC`

	rows, err := s.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		var dataStr string

		err := rows.Scan(
			&message.ID, &message.SessionID,
			&message.Timestamps.TimeCreated, &message.Timestamps.TimeUpdated,
			&dataStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Deserialize JSON field
		if err := json.Unmarshal([]byte(dataStr), &message.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		messages = append(messages, message)
	}

	return messages, nil
}

// ListWithLimit retrieves messages by session ID with a limit
func (s *MessageStorage) ListWithLimit(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	query := `
	SELECT id, session_id, time_created, time_updated, data
	FROM message WHERE session_id = ? ORDER BY time_created DESC LIMIT ?`

	rows, err := s.db.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		var dataStr string

		err := rows.Scan(
			&message.ID, &message.SessionID,
			&message.Timestamps.TimeCreated, &message.Timestamps.TimeUpdated,
			&dataStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Deserialize JSON field
		if err := json.Unmarshal([]byte(dataStr), &message.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		messages = append(messages, message)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// Update updates an existing message
func (s *MessageStorage) Update(ctx context.Context, message Message) error {
	message.Timestamps.UpdateTimestamps()

	// Serialize JSON field
	data, err := json.Marshal(message.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
	UPDATE message SET time_updated = ?, data = ? WHERE id = ?`

	return s.db.Exec(ctx, query, message.Timestamps.TimeUpdated, string(data), message.ID)
}

// Delete deletes a message by ID
func (s *MessageStorage) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM message WHERE id = ?`
	return s.db.Exec(ctx, query, id)
}

// DeleteBySession deletes all messages for a session
func (s *MessageStorage) DeleteBySession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM message WHERE session_id = ?`
	return s.db.Exec(ctx, query, sessionID)
}

// ListPaginated retrieves messages with cursor-based pagination
// Returns the most recent messages first, along with pagination metadata
// cursor is the timestamp (in milliseconds) to load messages older than this
// If cursor is 0, loads the most recent messages
func (s *MessageStorage) ListPaginated(ctx context.Context, sessionID string, limit int, cursor int64) ([]Message, int64, bool, error) {
	var query string
	var args []interface{}

	if cursor == 0 {
		// Initial load: get most recent messages
		query = `
		SELECT id, session_id, time_created, time_updated, data
		FROM message WHERE session_id = ? 
		ORDER BY time_created DESC LIMIT ?`
		args = []interface{}{sessionID, limit}
	} else {
		// Load more: get messages older than cursor timestamp
		query = `
		SELECT id, session_id, time_created, time_updated, data
		FROM message WHERE session_id = ? AND time_created < ?
		ORDER BY time_created DESC LIMIT ?`
		args = []interface{}{sessionID, cursor, limit}
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		var dataStr string

		err := rows.Scan(
			&message.ID, &message.SessionID,
			&message.Timestamps.TimeCreated, &message.Timestamps.TimeUpdated,
			&dataStr,
		)
		if err != nil {
			return nil, 0, false, fmt.Errorf("failed to scan message: %w", err)
		}

		// Deserialize JSON field
		if err := json.Unmarshal([]byte(dataStr), &message.Data); err != nil {
			return nil, 0, false, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		messages = append(messages, message)
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	// Determine next cursor (timestamp of oldest message in this batch)
	var nextCursor int64
	var complete bool

	if len(messages) < limit {
		// Less than limit means no more messages
		complete = true
	} else if len(messages) > 0 {
		// Next cursor is the timestamp of the oldest message in this batch (first after reversal)
		nextCursor = messages[0].Timestamps.TimeCreated
		complete = false
	}

	return messages, nextCursor, complete, nil
}

// Count returns the total message count for a session
func (s *MessageStorage) Count(ctx context.Context, sessionID string) (int, error) {
	query := `SELECT COUNT(*) FROM message WHERE session_id = ?`
	row := s.db.QueryRow(ctx, query, sessionID)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// ===========================================
// Part Storage
// ===========================================

// PartStorage provides part CRUD operations
type PartStorage struct {
	db *Database
}

// NewPartStorage creates a new part storage
func NewPartStorage(db *Database) *PartStorage {
	return &PartStorage{db: db}
}

// Create creates a new part
func (s *PartStorage) Create(ctx context.Context, part Part) (*Part, error) {
	if part.ID == "" {
		part.ID = uuid.New().String()
	}
	part.Timestamps = NewTimestamps()

	// Serialize JSON field
	data, err := json.Marshal(part.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
	INSERT INTO part (id, message_id, session_id, time_created, time_updated, data)
	VALUES (?, ?, ?, ?, ?, ?)`

	err = s.db.Exec(ctx, query,
		part.ID, part.MessageID, part.SessionID,
		part.Timestamps.TimeCreated, part.Timestamps.TimeUpdated,
		string(data),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create part: %w", err)
	}

	return &part, nil
}

// Get retrieves a part by ID
func (s *PartStorage) Get(ctx context.Context, id string) (*Part, error) {
	query := `
	SELECT id, message_id, session_id, time_created, time_updated, data
	FROM part WHERE id = ?`

	row := s.db.QueryRow(ctx, query, id)
	var part Part
	var dataStr string

	err := row.Scan(
		&part.ID, &part.MessageID, &part.SessionID,
		&part.Timestamps.TimeCreated, &part.Timestamps.TimeUpdated,
		&dataStr,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("part not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get part: %w", err)
	}

	// Deserialize JSON field
	if err := json.Unmarshal([]byte(dataStr), &part.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &part, nil
}

// ListByMessage retrieves parts by message ID
func (s *PartStorage) ListByMessage(ctx context.Context, messageID string) ([]Part, error) {
	query := `
	SELECT id, message_id, session_id, time_created, time_updated, data
	FROM part WHERE message_id = ? ORDER BY id ASC`

	rows, err := s.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to list parts: %w", err)
	}
	defer rows.Close()

	var parts []Part
	for rows.Next() {
		var part Part
		var dataStr string

		err := rows.Scan(
			&part.ID, &part.MessageID, &part.SessionID,
			&part.Timestamps.TimeCreated, &part.Timestamps.TimeUpdated,
			&dataStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan part: %w", err)
		}

		// Deserialize JSON field
		if err := json.Unmarshal([]byte(dataStr), &part.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		parts = append(parts, part)
	}

	return parts, nil
}

// ListBySession retrieves parts by session ID
func (s *PartStorage) ListBySession(ctx context.Context, sessionID string) ([]Part, error) {
	query := `
	SELECT id, message_id, session_id, time_created, time_updated, data
	FROM part WHERE session_id = ? ORDER BY time_created ASC`

	rows, err := s.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list parts: %w", err)
	}
	defer rows.Close()

	var parts []Part
	for rows.Next() {
		var part Part
		var dataStr string

		err := rows.Scan(
			&part.ID, &part.MessageID, &part.SessionID,
			&part.Timestamps.TimeCreated, &part.Timestamps.TimeUpdated,
			&dataStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan part: %w", err)
		}

		// Deserialize JSON field
		if err := json.Unmarshal([]byte(dataStr), &part.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		parts = append(parts, part)
	}

	return parts, nil
}

// Update updates an existing part
func (s *PartStorage) Update(ctx context.Context, part Part) error {
	part.Timestamps.UpdateTimestamps()

	// Serialize JSON field
	data, err := json.Marshal(part.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	query := `
	UPDATE part SET time_updated = ?, data = ? WHERE id = ?`

	return s.db.Exec(ctx, query, part.Timestamps.TimeUpdated, string(data), part.ID)
}

// Delete deletes a part by ID
func (s *PartStorage) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM part WHERE id = ?`
	return s.db.Exec(ctx, query, id)
}

// DeleteByMessage deletes all parts for a message
func (s *PartStorage) DeleteByMessage(ctx context.Context, messageID string) error {
	query := `DELETE FROM part WHERE message_id = ?`
	return s.db.Exec(ctx, query, messageID)
}
