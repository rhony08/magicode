// Package database provides SQLite database operations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"

	"github.com/rhony08/magicode/internal/util/log"
)

// Database wraps the SQL database connection
type Database struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// Config holds database configuration
type Config struct {
	Path string // Database file path
}

// New creates a new database connection
func New(ctx context.Context, cfg Config) (*Database, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("database path is required")
	}

	log.Debug("Opening database", "path", cfg.Path)

	// Open database connection
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite works best with single connection
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // No limit for SQLite

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:   db,
		path: cfg.Path,
	}

	// Run migrations
	if err := database.migrate(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Info("Database initialized", "path", cfg.Path)
	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db != nil {
		log.Debug("Closing database", "path", d.path)
		return d.db.Close()
	}
	return nil
}

// Path returns the database file path
func (d *Database) Path() string {
	return d.path
}

// DB returns the underlying sql.DB for direct queries
// Use this for custom queries, but prefer the typed methods
func (d *Database) DB() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// Exec executes a query without returning rows
func (d *Database) Exec(ctx context.Context, query string, args ...any) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Error("Query execution failed", "query", query, "error", err)
		return fmt.Errorf("query execution failed: %w", err)
	}
	return nil
}

// Query executes a query that returns rows
func (d *Database) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error("Query failed", "query", query, "error", err)
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return rows, nil
}

// QueryRow executes a query that returns at most one row
func (d *Database) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a new transaction
func (d *Database) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// InTransaction executes a function within a transaction
func (d *Database) InTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Error("Transaction rollback failed", "error", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}