// Package database provides project storage operations.
package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ProjectStorage provides project CRUD operations
type ProjectStorage struct {
	db *Database
}

// NewProjectStorage creates a new project storage
func NewProjectStorage(db *Database) *ProjectStorage {
	return &ProjectStorage{db: db}
}

// Create creates a new project
func (s *ProjectStorage) Create(ctx context.Context, project Project) (*Project, error) {
	if project.ID == "" {
		project.ID = uuid.New().String()
	}
	if project.Sandboxes == nil {
		project.Sandboxes = []string{}
	}

	project.Timestamps = NewTimestamps()

	// Serialize JSON fields
	sandboxes, err := json.Marshal(project.Sandboxes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sandboxes: %w", err)
	}
	commands, err := json.Marshal(project.Commands)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal commands: %w", err)
	}

	query := `
	INSERT INTO project (
		id, worktree, vcs, name, icon_url, icon_color,
		time_created, time_updated, time_initialized, sandboxes, commands
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	err = s.db.Exec(ctx, query,
		project.ID, project.Worktree, project.VCS, project.Name,
		project.IconURL, project.IconColor,
		project.Timestamps.TimeCreated, project.Timestamps.TimeUpdated,
		project.TimeInitialized, string(sandboxes), string(commands),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return &project, nil
}

// Get retrieves a project by ID
func (s *ProjectStorage) Get(ctx context.Context, id string) (*Project, error) {
	query := `
	SELECT id, worktree, vcs, name, icon_url, icon_color,
		time_created, time_updated, time_initialized, sandboxes, commands
	FROM project WHERE id = ?`

	row := s.db.QueryRow(ctx, query, id)
	var project Project
	var sandboxesStr, commandsStr sqlNullableString

	err := row.Scan(
		&project.ID, &project.Worktree, &project.VCS, &project.Name,
		&project.IconURL, &project.IconColor,
		&project.Timestamps.TimeCreated, &project.Timestamps.TimeUpdated,
		&project.TimeInitialized, &sandboxesStr, &commandsStr,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("project not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Deserialize JSON fields
	if sandboxesStr.Valid {
		if err := json.Unmarshal([]byte(sandboxesStr.String), &project.Sandboxes); err != nil {
			project.Sandboxes = []string{}
		}
	}
	if commandsStr.Valid && commandsStr.String != "" {
		if err := json.Unmarshal([]byte(commandsStr.String), &project.Commands); err != nil {
			project.Commands = nil
		}
	}

	return &project, nil
}

// GetByWorktree retrieves a project by worktree path
func (s *ProjectStorage) GetByWorktree(ctx context.Context, worktree string) (*Project, error) {
	query := `
	SELECT id, worktree, vcs, name, icon_url, icon_color,
		time_created, time_updated, time_initialized, sandboxes, commands
	FROM project WHERE worktree = ?`

	row := s.db.QueryRow(ctx, query, worktree)
	var project Project
	var sandboxesStr, commandsStr sqlNullableString

	err := row.Scan(
		&project.ID, &project.Worktree, &project.VCS, &project.Name,
		&project.IconURL, &project.IconColor,
		&project.Timestamps.TimeCreated, &project.Timestamps.TimeUpdated,
		&project.TimeInitialized, &sandboxesStr, &commandsStr,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("project not found for worktree: %s", worktree)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Deserialize JSON fields
	if sandboxesStr.Valid {
		if err := json.Unmarshal([]byte(sandboxesStr.String), &project.Sandboxes); err != nil {
			project.Sandboxes = []string{}
		}
	}
	if commandsStr.Valid && commandsStr.String != "" {
		if err := json.Unmarshal([]byte(commandsStr.String), &project.Commands); err != nil {
			project.Commands = nil
		}
	}

	return &project, nil
}

// List retrieves all projects
func (s *ProjectStorage) List(ctx context.Context) ([]Project, error) {
	query := `
	SELECT id, worktree, vcs, name, icon_url, icon_color,
		time_created, time_updated, time_initialized, sandboxes, commands
	FROM project ORDER BY time_created DESC`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		var sandboxesStr, commandsStr sqlNullableString

		err := rows.Scan(
			&project.ID, &project.Worktree, &project.VCS, &project.Name,
			&project.IconURL, &project.IconColor,
			&project.Timestamps.TimeCreated, &project.Timestamps.TimeUpdated,
			&project.TimeInitialized, &sandboxesStr, &commandsStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		// Deserialize JSON fields
		if sandboxesStr.Valid {
			if err := json.Unmarshal([]byte(sandboxesStr.String), &project.Sandboxes); err != nil {
				project.Sandboxes = []string{}
			}
		}
		if commandsStr.Valid && commandsStr.String != "" {
			if err := json.Unmarshal([]byte(commandsStr.String), &project.Commands); err != nil {
				project.Commands = nil
			}
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// Update updates an existing project
func (s *ProjectStorage) Update(ctx context.Context, project Project) error {
	project.Timestamps.UpdateTimestamps()

	// Serialize JSON fields
	sandboxes, err := json.Marshal(project.Sandboxes)
	if err != nil {
		return fmt.Errorf("failed to marshal sandboxes: %w", err)
	}
	commands, err := json.Marshal(project.Commands)
	if err != nil {
		return fmt.Errorf("failed to marshal commands: %w", err)
	}

	query := `
	UPDATE project SET
		name = ?, icon_url = ?, icon_color = ?, vcs = ?,
		time_updated = ?, time_initialized = ?, sandboxes = ?, commands = ?
	WHERE id = ?`

	return s.db.Exec(ctx, query,
		project.Name, project.IconURL, project.IconColor, project.VCS,
		project.Timestamps.TimeUpdated, project.TimeInitialized,
		string(sandboxes), string(commands),
		project.ID,
	)
}

// Delete deletes a project by ID
func (s *ProjectStorage) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM project WHERE id = ?`
	return s.db.Exec(ctx, query, id)
}

// Initialize marks a project as initialized
func (s *ProjectStorage) Initialize(ctx context.Context, id string) error {
	query := `UPDATE project SET time_initialized = ?, time_updated = ? WHERE id = ?`
	now := NewTimestamps()
	return s.db.Exec(ctx, query, now.TimeCreated, now.TimeUpdated, id)
}