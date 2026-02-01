// ABOUTME: Tests for SQLite database initialization and management.
// ABOUTME: Validates schema creation, migrations, and connection handling.

package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStore(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "memo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestStoreTables(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Verify all expected tables exist
	tables := []string{"notes", "tags", "note_tags", "attachments", "notes_fts"}
	for _, table := range tables {
		var count int
		err := store.db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
		if err != nil {
			t.Fatalf("failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestStoreFTSTriggers(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// Verify FTS triggers exist
	triggers := []string{"notes_ai", "notes_ad", "notes_au"}
	for _, trigger := range triggers {
		var count int
		err := store.db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name=?`, trigger).Scan(&count)
		if err != nil {
			t.Fatalf("failed to check trigger %s: %v", trigger, err)
		}
		if count != 1 {
			t.Errorf("expected trigger %s to exist", trigger)
		}
	}
}

func TestDefaultDataPath(t *testing.T) {
	path := DefaultDataPath()

	// Should contain memo.db
	if filepath.Base(path) != "memo.db" {
		t.Errorf("expected path to end with memo.db, got %s", filepath.Base(path))
	}

	// Should be under .local/share/memo
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
}

func TestDefaultDataPathWithXDGDataHome(t *testing.T) {
	// Save original value
	original := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", original)

	// Set XDG_DATA_HOME to a custom path
	customPath := "/custom/data/home"
	os.Setenv("XDG_DATA_HOME", customPath)

	path := DefaultDataPath()

	// Should use XDG_DATA_HOME
	if !strings.HasPrefix(path, customPath) {
		t.Errorf("expected path to start with %s, got %s", customPath, path)
	}

	// Should still end with memo/memo.db
	expected := filepath.Join(customPath, "memo", "memo.db")
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

func TestStoreCloseNilDB(t *testing.T) {
	// Test closing a store with nil db
	store := &Store{db: nil}
	err := store.Close()
	if err != nil {
		t.Errorf("expected nil error closing store with nil db, got %v", err)
	}
}

// newTestStore creates a new in-memory store for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "memo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Register cleanup for temp dir
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Register cleanup for store
	t.Cleanup(func() {
		store.Close()
	})

	return store
}
