// Package database provides KV (key-value) storage for persistent preferences
package database

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"
)

func TestKVStorage(t *testing.T) {
	ctx := context.Background()

	// Create a temporary database
	db, err := New(ctx, Config{Path: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	storage := NewKVStorage(db)

	t.Run("Set and Get", func(t *testing.T) {
		// Set a value
		err := storage.Set(ctx, "test_key", "test_value")
		if err != nil {
			t.Errorf("Set() error = %v", err)
		}

		// Get the value
		value, err := storage.Get(ctx, "test_key")
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if value != "test_value" {
			t.Errorf("Get() = %v, want %v", value, "test_value")
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		_, err := storage.Get(ctx, "non_existent_key")
		if err == nil {
			t.Error("Get() expected error for non-existent key")
		}
	})

	t.Run("GetOrDefault", func(t *testing.T) {
		// Existing key
		value := storage.GetOrDefault(ctx, "test_key", "default")
		if value != "test_value" {
			t.Errorf("GetOrDefault() = %v, want %v", value, "test_value")
		}

		// Non-existent key
		value = storage.GetOrDefault(ctx, "non_existent", "default")
		if value != "default" {
			t.Errorf("GetOrDefault() = %v, want %v", value, "default")
		}
	})

	t.Run("Update existing key", func(t *testing.T) {
		// Set initial value
		err := storage.Set(ctx, "update_key", "initial")
		if err != nil {
			t.Errorf("Set() error = %v", err)
		}

		// Update value
		err = storage.Set(ctx, "update_key", "updated")
		if err != nil {
			t.Errorf("Set() error = %v", err)
		}

		// Verify update
		value, err := storage.Get(ctx, "update_key")
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if value != "updated" {
			t.Errorf("Get() = %v, want %v", value, "updated")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		// Set and delete
		err := storage.Set(ctx, "delete_key", "value")
		if err != nil {
			t.Errorf("Set() error = %v", err)
		}

		err = storage.Delete(ctx, "delete_key")
		if err != nil {
			t.Errorf("Delete() error = %v", err)
		}

		// Verify deletion
		_, err = storage.Get(ctx, "delete_key")
		if err == nil {
			t.Error("Get() expected error after delete")
		}
	})

	t.Run("List", func(t *testing.T) {
		// Set multiple keys with prefix
		storage.Set(ctx, "prefix_key1", "value1")
		storage.Set(ctx, "prefix_key2", "value2")
		storage.Set(ctx, "other_key", "value3")

		// List with prefix
		kvs, err := storage.List(ctx, "prefix_")
		if err != nil {
			t.Errorf("List() error = %v", err)
		}
		if len(kvs) != 2 {
			t.Errorf("List() returned %d items, want 2", len(kvs))
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		// Clear and set test data
		storage.Set(ctx, "all_key1", "value1")
		storage.Set(ctx, "all_key2", "value2")

		// Get all
		all, err := storage.GetAll(ctx)
		if err != nil {
			t.Errorf("GetAll() error = %v", err)
		}
		if len(all) < 2 {
			t.Errorf("GetAll() returned %d items, want at least 2", len(all))
		}
	})

	t.Run("SetJSON and GetJSON", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		// Set JSON value
		data := TestStruct{Name: "test", Value: 42}
		err := storage.SetJSON(ctx, "json_key", data)
		if err != nil {
			t.Errorf("SetJSON() error = %v", err)
		}

		// Get JSON value
		var result TestStruct
		err = storage.GetJSON(ctx, "json_key", &result)
		if err != nil {
			t.Errorf("GetJSON() error = %v", err)
		}
		if result.Name != "test" || result.Value != 42 {
			t.Errorf("GetJSON() = %+v, want {Name:test Value:42}", result)
		}
	})
}

func TestKVStorage_Theme(t *testing.T) {
	ctx := context.Background()

	// Create a temporary database
	db, err := New(ctx, Config{Path: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	storage := NewKVStorage(db)

	t.Run("GetTheme default", func(t *testing.T) {
		theme := storage.GetTheme(ctx)
		if theme != "default" {
			t.Errorf("GetTheme() = %v, want default", theme)
		}
	})

	t.Run("SetTheme and GetTheme", func(t *testing.T) {
		err := storage.SetTheme(ctx, "catppuccin")
		if err != nil {
			t.Errorf("SetTheme() error = %v", err)
		}

		theme := storage.GetTheme(ctx)
		if theme != "catppuccin" {
			t.Errorf("GetTheme() = %v, want catppuccin", theme)
		}
	})

	t.Run("SidebarMode", func(t *testing.T) {
		// Default
		mode := storage.GetSidebarMode(ctx)
		if mode != "auto" {
			t.Errorf("GetSidebarMode() = %v, want auto", mode)
		}

		// Set
		err := storage.SetSidebarMode(ctx, "show")
		if err != nil {
			t.Errorf("SetSidebarMode() error = %v", err)
		}

		// Verify
		mode = storage.GetSidebarMode(ctx)
		if mode != "show" {
			t.Errorf("GetSidebarMode() = %v, want show", mode)
		}
	})
}
