// Package database provides schema migrations.
package database

import (
	"context"
	"fmt"
)

// migrate runs all schema migrations
func (d *Database) migrate(ctx context.Context) error {
	// Create migrations table first
	if err := d.createMigrationsTable(ctx); err != nil {
		return err
	}

	// Run all migrations in order
	migrations := []migration{
		{"001_initial", d.migrateInitial},
		// Future migrations can be added here
	}

	for _, m := range migrations {
		if err := d.runMigration(ctx, m); err != nil {
			return err
		}
	}

	return nil
}

// migration represents a single migration
type migration struct {
	name string
	fn   func(ctx context.Context, db *Database) error
}

// createMigrationsTable creates the migrations tracking table
func (d *Database) createMigrationsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS _migrations (
		name TEXT PRIMARY KEY,
		applied_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now') * 1000)
	)`
	return d.Exec(ctx, query)
}

// runMigration runs a single migration if not already applied
func (d *Database) runMigration(ctx context.Context, m migration) error {
	// Check if migration already applied
	var count int
	row := d.QueryRow(ctx, "SELECT COUNT(*) FROM _migrations WHERE name = ?", m.name)
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count > 0 {
		return nil // Already applied
	}

	// Run migration
	if err := m.fn(ctx, d); err != nil {
		return fmt.Errorf("migration %s failed: %w", m.name, err)
	}

	// Record migration
	return d.Exec(ctx, "INSERT INTO _migrations (name) VALUES (?)", m.name)
}

// migrateInitial creates all initial tables
// This matches the TypeScript Drizzle schema
func (d *Database) migrateInitial(ctx context.Context, db *Database) error {
	// Project table
	projectSchema := `
	CREATE TABLE IF NOT EXISTS project (
		id TEXT PRIMARY KEY,
		worktree TEXT NOT NULL,
		vcs TEXT,
		name TEXT,
		icon_url TEXT,
		icon_color TEXT,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		time_initialized INTEGER,
		sandboxes TEXT NOT NULL DEFAULT '[]',
		commands TEXT
	)`

	// Workspace table
	workspaceSchema := `
	CREATE TABLE IF NOT EXISTS workspace (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '',
		branch TEXT,
		directory TEXT,
		extra TEXT,
		project_id TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE
	)`

	// Session table
	sessionSchema := `
	CREATE TABLE IF NOT EXISTS session (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE,
		workspace_id TEXT,
		parent_id TEXT,
		slug TEXT NOT NULL,
		directory TEXT NOT NULL,
		title TEXT NOT NULL,
		version TEXT NOT NULL,
		share_url TEXT,
		summary_additions INTEGER,
		summary_deletions INTEGER,
		summary_files INTEGER,
		summary_diffs TEXT,
		revert TEXT,
		permission TEXT,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		time_compacting INTEGER,
		time_archived INTEGER
	)`

	// Message table
	messageSchema := `
	CREATE TABLE IF NOT EXISTS message (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		data TEXT NOT NULL
	)`

	// Part table
	partSchema := `
	CREATE TABLE IF NOT EXISTS part (
		id TEXT PRIMARY KEY,
		message_id TEXT NOT NULL REFERENCES message(id) ON DELETE CASCADE,
		session_id TEXT NOT NULL,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		data TEXT NOT NULL
	)`

	// Todo table
	todoSchema := `
	CREATE TABLE IF NOT EXISTS todo (
		session_id TEXT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		status TEXT NOT NULL,
		priority TEXT NOT NULL,
		position INTEGER NOT NULL,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		PRIMARY KEY (session_id, position)
	)`

	// Session entry table
	sessionEntrySchema := `
	CREATE TABLE IF NOT EXISTS session_entry (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
		type TEXT NOT NULL,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		data TEXT NOT NULL
	)`

	// Permission table
	permissionSchema := `
	CREATE TABLE IF NOT EXISTS permission (
		project_id TEXT PRIMARY KEY REFERENCES project(id) ON DELETE CASCADE,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		data TEXT NOT NULL
	)`

	// Event sequence table (for sync)
	eventSequenceSchema := `
	CREATE TABLE IF NOT EXISTS event_sequence (
		aggregate_id TEXT PRIMARY KEY,
		seq INTEGER NOT NULL
	)`

	// Event table (for sync)
	eventSchema := `
	CREATE TABLE IF NOT EXISTS event (
		id TEXT PRIMARY KEY,
		aggregate_id TEXT NOT NULL REFERENCES event_sequence(aggregate_id) ON DELETE CASCADE,
		seq INTEGER NOT NULL,
		type TEXT NOT NULL,
		data TEXT NOT NULL
	)`

	// Account table
	accountSchema := `
	CREATE TABLE IF NOT EXISTS account (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		url TEXT NOT NULL,
		access_token TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		token_expiry INTEGER,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL
	)`

	// Account state table (single row for active account)
	accountStateSchema := `
	CREATE TABLE IF NOT EXISTS account_state (
		id INTEGER PRIMARY KEY,
		active_account_id TEXT REFERENCES account(id) ON DELETE SET NULL,
		active_org_id TEXT
	)`

	// Control account table (legacy)
	controlAccountSchema := `
	CREATE TABLE IF NOT EXISTS control_account (
		email TEXT NOT NULL,
		url TEXT NOT NULL,
		access_token TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		token_expiry INTEGER,
		active INTEGER NOT NULL DEFAULT 0,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		PRIMARY KEY (email, url)
	)`

	// Session share table
	sessionShareSchema := `
	CREATE TABLE IF NOT EXISTS session_share (
		session_id TEXT PRIMARY KEY REFERENCES session(id) ON DELETE CASCADE,
		id TEXT NOT NULL,
		secret TEXT NOT NULL,
		url TEXT NOT NULL,
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL
	)`

	// Create all tables
	schemas := []string{
		projectSchema,
		workspaceSchema,
		sessionSchema,
		messageSchema,
		partSchema,
		todoSchema,
		sessionEntrySchema,
		permissionSchema,
		eventSequenceSchema,
		eventSchema,
		accountSchema,
		accountStateSchema,
		controlAccountSchema,
		sessionShareSchema,
	}

	for _, schema := range schemas {
		if err := db.Exec(ctx, schema); err != nil {
			return err
		}
	}

	// Create indexes
	return d.createIndexes(ctx, db)
}

// createIndexes creates all indexes
func (d *Database) createIndexes(ctx context.Context, db *Database) error {
	indexes := []string{
		// Project indexes
		"CREATE INDEX IF NOT EXISTS project_worktree_idx ON project(worktree)",

		// Workspace indexes
		"CREATE INDEX IF NOT EXISTS workspace_project_idx ON workspace(project_id)",

		// Session indexes
		"CREATE INDEX IF NOT EXISTS session_project_idx ON session(project_id)",
		"CREATE INDEX IF NOT EXISTS session_workspace_idx ON session(workspace_id)",
		"CREATE INDEX IF NOT EXISTS session_parent_idx ON session(parent_id)",
		"CREATE INDEX IF NOT EXISTS session_directory_idx ON session(directory)",

		// Message indexes
		"CREATE INDEX IF NOT EXISTS message_session_time_created_id_idx ON message(session_id, time_created, id)",
		"CREATE INDEX IF NOT EXISTS message_session_idx ON message(session_id)",

		// Part indexes
		"CREATE INDEX IF NOT EXISTS part_message_id_id_idx ON part(message_id, id)",
		"CREATE INDEX IF NOT EXISTS part_session_idx ON part(session_id)",

		// Todo indexes
		"CREATE INDEX IF NOT EXISTS todo_session_idx ON todo(session_id)",

		// Session entry indexes
		"CREATE INDEX IF NOT EXISTS session_entry_session_idx ON session_entry(session_id)",
		"CREATE INDEX IF NOT EXISTS session_entry_session_type_idx ON session_entry(session_id, type)",
		"CREATE INDEX IF NOT EXISTS session_entry_time_created_idx ON session_entry(time_created)",

		// Event indexes
		"CREATE INDEX IF NOT EXISTS event_aggregate_idx ON event(aggregate_id)",
		"CREATE INDEX IF NOT EXISTS event_type_idx ON event(type)",

		// Account indexes
		"CREATE INDEX IF NOT EXISTS account_email_idx ON account(email)",
	}

	for _, idx := range indexes {
		if err := db.Exec(ctx, idx); err != nil {
			return err
		}
	}
	return nil
}