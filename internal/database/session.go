// Package database provides session storage operations.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionStorage provides session CRUD operations
type SessionStorage struct {
	db *Database
}

// NewSessionStorage creates a new session storage
func NewSessionStorage(db *Database) *SessionStorage {
	return &SessionStorage{db: db}
}

// Create creates a new session
func (s *SessionStorage) Create(ctx context.Context, session Session) (*Session, error) {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.Slug == "" {
		session.Slug = fmt.Sprintf("session-%s", session.ID[:8])
	}
	if session.Title == "" {
		session.Title = "New Session"
	}
	if session.Version == "" {
		session.Version = "1"
	}

	session.Timestamps = NewTimestamps()

	// Serialize JSON fields
	summaryDiffs, err := json.Marshal(session.SummaryDiffs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal summary_diffs: %w", err)
	}
	revert, err := json.Marshal(session.Revert)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal revert: %w", err)
	}
	permission, err := json.Marshal(session.Permission)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal permission: %w", err)
	}

	query := `
	INSERT INTO session (
		id, project_id, workspace_id, parent_id, slug, directory, title, version,
		share_url, summary_additions, summary_deletions, summary_files, summary_diffs,
		revert, permission, time_created, time_updated, time_compacting, time_archived
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	err = s.db.Exec(ctx, query,
		session.ID, session.ProjectID, session.WorkspaceID, session.ParentID,
		session.Slug, session.Directory, session.Title, session.Version,
		session.ShareURL, session.SummaryAdditions, session.SummaryDeletions,
		session.SummaryFiles, string(summaryDiffs), string(revert), string(permission),
		session.Timestamps.TimeCreated, session.Timestamps.TimeUpdated,
		session.TimeCompacting, session.TimeArchived,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &session, nil
}

// Get retrieves a session by ID
func (s *SessionStorage) Get(ctx context.Context, id string) (*Session, error) {
	query := `
	SELECT id, project_id, workspace_id, parent_id, slug, directory, title, version,
		share_url, summary_additions, summary_deletions, summary_files, summary_diffs,
		revert, permission, time_created, time_updated, time_compacting, time_archived
	FROM session WHERE id = ?`

	row := s.db.QueryRow(ctx, query, id)
	var session Session
	var summaryDiffs, revert, permission sqlNullableString
	var timeCompacting, timeArchived sql.NullInt64

	err := row.Scan(
		&session.ID, &session.ProjectID, &session.WorkspaceID, &session.ParentID,
		&session.Slug, &session.Directory, &session.Title, &session.Version,
		&session.ShareURL, &session.SummaryAdditions, &session.SummaryDeletions,
		&session.SummaryFiles, &summaryDiffs, &revert, &permission,
		&session.Timestamps.TimeCreated, &session.Timestamps.TimeUpdated,
		&timeCompacting, &timeArchived,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Handle nullable integer fields
	if timeCompacting.Valid {
		session.TimeCompacting = timeCompacting.Int64
	}
	if timeArchived.Valid {
		session.TimeArchived = timeArchived.Int64
	}

	// Deserialize JSON fields
	if summaryDiffs.Valid && summaryDiffs.String != "" {
		json.Unmarshal([]byte(summaryDiffs.String), &session.SummaryDiffs)
	}
	if revert.Valid && revert.String != "" {
		json.Unmarshal([]byte(revert.String), &session.Revert)
	}
	if permission.Valid && permission.String != "" {
		json.Unmarshal([]byte(permission.String), &session.Permission)
	}

	return &session, nil
}

// sqlNullableString helps scan nullable TEXT columns
type sqlNullableString struct {
	String string
	Valid  bool
}

// Scan implements sql.Scanner
func (ns *sqlNullableString) Scan(value any) error {
	if value == nil {
		ns.String = ""
		ns.Valid = false
		return nil
	}
	switch v := value.(type) {
	case string:
		ns.String = v
		ns.Valid = true
	case []byte:
		ns.String = string(v)
		ns.Valid = true
	}
	return nil
}

// List retrieves sessions by project ID
func (s *SessionStorage) List(ctx context.Context, projectID string) ([]Session, error) {
	query := `
	SELECT id, project_id, workspace_id, parent_id, slug, directory, title, version,
		share_url, summary_additions, summary_deletions, summary_files, summary_diffs,
		revert, permission, time_created, time_updated, time_compacting, time_archived
	FROM session WHERE project_id = ? ORDER BY time_created DESC`

	rows, err := s.db.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var summaryDiffs, revert, permission sqlNullableString
		var timeCompacting, timeArchived sql.NullInt64

		err := rows.Scan(
			&session.ID, &session.ProjectID, &session.WorkspaceID, &session.ParentID,
			&session.Slug, &session.Directory, &session.Title, &session.Version,
			&session.ShareURL, &session.SummaryAdditions, &session.SummaryDeletions,
			&session.SummaryFiles, &summaryDiffs, &revert, &permission,
			&session.Timestamps.TimeCreated, &session.Timestamps.TimeUpdated,
			&timeCompacting, &timeArchived,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Handle nullable integer fields
		if timeCompacting.Valid {
			session.TimeCompacting = timeCompacting.Int64
		}
		if timeArchived.Valid {
			session.TimeArchived = timeArchived.Int64
		}

		// Deserialize JSON fields
		if summaryDiffs.Valid && summaryDiffs.String != "" {
			json.Unmarshal([]byte(summaryDiffs.String), &session.SummaryDiffs)
		}
		if revert.Valid && revert.String != "" {
			json.Unmarshal([]byte(revert.String), &session.Revert)
		}
		if permission.Valid && permission.String != "" {
			json.Unmarshal([]byte(permission.String), &session.Permission)
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// ListByDirectory retrieves sessions by directory
func (s *SessionStorage) ListByDirectory(ctx context.Context, directory string) ([]Session, error) {
	query := `
	SELECT id, project_id, workspace_id, parent_id, slug, directory, title, version,
		share_url, summary_additions, summary_deletions, summary_files, summary_diffs,
		revert, permission, time_created, time_updated, time_compacting, time_archived
	FROM session WHERE directory = ? ORDER BY time_created DESC`

	rows, err := s.db.Query(ctx, query, directory)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions by directory: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var summaryDiffs, revert, permission sqlNullableString
		var timeCompacting, timeArchived sql.NullInt64

		err := rows.Scan(
			&session.ID, &session.ProjectID, &session.WorkspaceID, &session.ParentID,
			&session.Slug, &session.Directory, &session.Title, &session.Version,
			&session.ShareURL, &session.SummaryAdditions, &session.SummaryDeletions,
			&session.SummaryFiles, &summaryDiffs, &revert, &permission,
			&session.Timestamps.TimeCreated, &session.Timestamps.TimeUpdated,
			&timeCompacting, &timeArchived,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Handle nullable integer fields
		if timeCompacting.Valid {
			session.TimeCompacting = timeCompacting.Int64
		}
		if timeArchived.Valid {
			session.TimeArchived = timeArchived.Int64
		}

		// Deserialize JSON fields
		if summaryDiffs.Valid && summaryDiffs.String != "" {
			json.Unmarshal([]byte(summaryDiffs.String), &session.SummaryDiffs)
		}
		if revert.Valid && revert.String != "" {
			json.Unmarshal([]byte(revert.String), &session.Revert)
		}
		if permission.Valid && permission.String != "" {
			json.Unmarshal([]byte(permission.String), &session.Permission)
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Update updates an existing session
func (s *SessionStorage) Update(ctx context.Context, session Session) error {
	session.Timestamps.UpdateTimestamps()

	// Serialize JSON fields
	summaryDiffs, err := json.Marshal(session.SummaryDiffs)
	if err != nil {
		return fmt.Errorf("failed to marshal summary_diffs: %w", err)
	}
	revert, err := json.Marshal(session.Revert)
	if err != nil {
		return fmt.Errorf("failed to marshal revert: %w", err)
	}
	permission, err := json.Marshal(session.Permission)
	if err != nil {
		return fmt.Errorf("failed to marshal permission: %w", err)
	}

	query := `
	UPDATE session SET
		title = ?, share_url = ?, summary_additions = ?, summary_deletions = ?,
		summary_files = ?, summary_diffs = ?, revert = ?, permission = ?,
		time_updated = ?, time_compacting = ?, time_archived = ?
	WHERE id = ?`

	return s.db.Exec(ctx, query,
		session.Title, session.ShareURL, session.SummaryAdditions, session.SummaryDeletions,
		session.SummaryFiles, string(summaryDiffs), string(revert), string(permission),
		session.Timestamps.TimeUpdated, session.TimeCompacting, session.TimeArchived,
		session.ID,
	)
}

// Delete deletes a session by ID
func (s *SessionStorage) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM session WHERE id = ?`
	return s.db.Exec(ctx, query, id)
}

// Archive archives a session
func (s *SessionStorage) Archive(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()
	query := `UPDATE session SET time_archived = ?, time_updated = ? WHERE id = ?`
	return s.db.Exec(ctx, query, now, now, id)
}

// Unarchive unarchives a session
func (s *SessionStorage) Unarchive(ctx context.Context, id string) error {
	now := time.Now().UnixMilli()
	query := `UPDATE session SET time_archived = NULL, time_updated = ? WHERE id = ?`
	return s.db.Exec(ctx, query, now, id)
}

// Fork creates a new session from an existing one
func (s *SessionStorage) Fork(ctx context.Context, parentID string) (*Session, error) {
	parent, err := s.Get(ctx, parentID)
	if err != nil {
		return nil, err
	}

	fork := Session{
		ID:         uuid.New().String(),
		ProjectID:  parent.ProjectID,
		WorkspaceID: parent.WorkspaceID,
		ParentID:   parentID,
		Slug:       fmt.Sprintf("%s-fork", parent.Slug),
		Directory:  parent.Directory,
		Title:      fmt.Sprintf("Fork of %s", parent.Title),
		Version:    "1",
	}

	return s.Create(ctx, fork)
}