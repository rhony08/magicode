// Package database provides KV (key-value) storage for persistent preferences
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// KV represents a key-value pair stored in the database
type KV struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	TimeCreated int64  `json:"time_created"`
	TimeUpdated int64  `json:"time_updated"`
}

// KVStorage provides KV storage operations
type KVStorage struct {
	db *Database
}

// NewKVStorage creates a new KV storage instance
func NewKVStorage(db *Database) *KVStorage {
	return &KVStorage{db: db}
}

// Get retrieves a value by key
func (s *KVStorage) Get(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM kv WHERE key = ?`
	row := s.db.QueryRow(ctx, query, key)

	var value string
	err := row.Scan(&value)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", fmt.Errorf("key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return value, nil
}

// GetOrDefault retrieves a value by key, returning default if not found
func (s *KVStorage) GetOrDefault(ctx context.Context, key string, defaultValue string) string {
	value, err := s.Get(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// Set stores a key-value pair
func (s *KVStorage) Set(ctx context.Context, key string, value string) error {
	now := getCurrentTimestamp()

	query := `
		INSERT INTO kv (key, value, time_created, time_updated) 
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			time_updated = excluded.time_updated
	`

	err := s.db.Exec(ctx, query, key, value, now, now)
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

// SetJSON stores a value as JSON
func (s *KVStorage) SetJSON(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}
	return s.Set(ctx, key, string(data))
}

// GetJSON retrieves and unmarshals a JSON value
func (s *KVStorage) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := s.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value for key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key-value pair
func (s *KVStorage) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM kv WHERE key = ?`
	err := s.db.Exec(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// List returns all keys with a given prefix
func (s *KVStorage) List(ctx context.Context, prefix string) ([]KV, error) {
	query := `SELECT key, value, time_created, time_updated FROM kv WHERE key LIKE ? ORDER BY key`
	rows, err := s.db.Query(ctx, query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to list keys with prefix %s: %w", prefix, err)
	}
	defer rows.Close()

	var result []KV
	for rows.Next() {
		var kv KV
		if err := rows.Scan(&kv.Key, &kv.Value, &kv.TimeCreated, &kv.TimeUpdated); err != nil {
			return nil, fmt.Errorf("failed to scan kv row: %w", err)
		}
		result = append(result, kv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating kv rows: %w", err)
	}

	return result, nil
}

// GetAll returns all KV pairs
func (s *KVStorage) GetAll(ctx context.Context) (map[string]string, error) {
	query := `SELECT key, value FROM kv ORDER BY key`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all kv: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan kv row: %w", err)
		}
		result[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating kv rows: %w", err)
	}

	return result, nil
}

// ThemeKVKeys defines the KV keys for theme-related preferences
const (
	KVKeyTheme          = "theme"
	KVKeyThemeDark      = "theme_dark"
	KVKeySidebarMode    = "sidebar_mode"
	KVKeySidebarWidth   = "sidebar_width"
	KVKeyShowTimestamps = "show_timestamps"
)

// GetTheme retrieves the stored theme preference
func (s *KVStorage) GetTheme(ctx context.Context) string {
	return s.GetOrDefault(ctx, KVKeyTheme, "default")
}

// SetTheme stores the theme preference
func (s *KVStorage) SetTheme(ctx context.Context, theme string) error {
	return s.Set(ctx, KVKeyTheme, theme)
}

// GetSidebarMode retrieves the sidebar mode preference
func (s *KVStorage) GetSidebarMode(ctx context.Context) string {
	return s.GetOrDefault(ctx, KVKeySidebarMode, "auto")
}

// SetSidebarMode stores the sidebar mode preference
func (s *KVStorage) SetSidebarMode(ctx context.Context, mode string) error {
	return s.Set(ctx, KVKeySidebarMode, mode)
}

// getCurrentTimestamp returns current time in milliseconds
func getCurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}
