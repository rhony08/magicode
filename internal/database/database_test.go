// Package database provides tests for database operations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDatabaseNew tests database initialization
func TestDatabaseNew(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create database
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(context.Background(), Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Check database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Check path
	if db.Path() != dbPath {
		t.Errorf("Expected path %s, got %s", dbPath, db.Path())
	}
}

// TestDatabaseMigrations tests that migrations run correctly
func TestDatabaseMigrations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(context.Background(), Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Check that tables exist
	ctx := context.Background()
	tables := []string{
		"project", "workspace", "session", "message", "part",
		"todo", "session_entry", "permission", "event_sequence",
		"event", "account", "account_state", "control_account",
		"session_share", "_migrations",
	}

	for _, table := range tables {
		var count int
		row := db.QueryRow(ctx,
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table)
		if err := row.Scan(&count); err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
			continue
		}
		if count != 1 {
			t.Errorf("Table %s not found", table)
		}
	}
}

// TestDatabaseExec tests query execution
func TestDatabaseExec(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(context.Background(), Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Insert and query (need to include required timestamps)
	err = db.Exec(ctx, "INSERT INTO project (id, worktree, name, time_created, time_updated) VALUES (?, ?, ?, ?, ?)",
		"test-id", "/test/path", "Test Project", time.Now().UnixMilli(), time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	var name string
	row := db.QueryRow(ctx, "SELECT name FROM project WHERE id = ?", "test-id")
	if err := row.Scan(&name); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if name != "Test Project" {
		t.Errorf("Expected 'Test Project', got '%s'", name)
	}
}

// TestDatabaseTransaction tests transaction handling
func TestDatabaseTransaction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(context.Background(), Config{Path: dbPath})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	now := time.Now().UnixMilli()

	// Test successful transaction
	err = db.InTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO project (id, worktree, name, time_created, time_updated) VALUES (?, ?, ?, ?, ?)",
			"tx-test-1", "/test/1", "Project 1", now, now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO project (id, worktree, name, time_created, time_updated) VALUES (?, ?, ?, ?, ?)",
			"tx-test-2", "/test/2", "Project 2", now, now)
		return err
	})
	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Check both inserted
	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM project WHERE id LIKE 'tx-test-%'")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 projects, got %d", count)
	}

	// Test failed transaction (should rollback)
	err = db.InTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO project (id, worktree, name, time_created, time_updated) VALUES (?, ?, ?, ?, ?)",
			"tx-test-3", "/test/3", "Project 3", now, now)
		if err != nil {
			return err
		}
		// Force error
		return fmt.Errorf("intentional error")
	})
	if err == nil {
		t.Error("Expected transaction to fail")
	}

	// Check rollback
	row = db.QueryRow(ctx, "SELECT COUNT(*) FROM project WHERE id = 'tx-test-3'")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 projects (rolled back), got %d", count)
	}
}

// TestTimestamps tests timestamp operations
func TestTimestamps(t *testing.T) {
	ts := NewTimestamps()

	if ts.TimeCreated == 0 {
		t.Error("TimeCreated should not be zero")
	}
	if ts.TimeUpdated == 0 {
		t.Error("TimeUpdated should not be zero")
	}
	if ts.TimeCreated != ts.TimeUpdated {
		t.Error("Initial timestamps should be equal")
	}

	// Update - wait a bit to ensure different millisecond
	time.Sleep(2 * time.Millisecond)
	oldUpdated := ts.TimeUpdated
	ts.UpdateTimestamps()
	if ts.TimeUpdated <= oldUpdated {
		t.Error("TimeUpdated should increase after UpdateTimestamps")
	}
	// After update, TimeCreated stays the same but TimeUpdated changes
	if ts.TimeCreated == ts.TimeUpdated {
		// This could still be equal if within same millisecond, which is OK
		// The key check is that TimeUpdated >= oldUpdated
	}
}

// TestSqlNullableString tests nullable string scanning
func TestSqlNullableString(t *testing.T) {
	var ns sqlNullableString

	// Test nil value
	if err := ns.Scan(nil); err != nil {
		t.Errorf("Failed to scan nil: %v", err)
	}
	if ns.Valid {
		t.Error("nil should set Valid to false")
	}

	// Test string value
	if err := ns.Scan("test string"); err != nil {
		t.Errorf("Failed to scan string: %v", err)
	}
	if !ns.Valid {
		t.Error("string should set Valid to true")
	}
	if ns.String != "test string" {
		t.Errorf("Expected 'test string', got '%s'", ns.String)
	}

	// Test byte slice
	if err := ns.Scan([]byte("byte string")); err != nil {
		t.Errorf("Failed to scan bytes: %v", err)
	}
	if !ns.Valid {
		t.Error("bytes should set Valid to true")
	}
	if ns.String != "byte string" {
		t.Errorf("Expected 'byte string', got '%s'", ns.String)
	}
}