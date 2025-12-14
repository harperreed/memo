// ABOUTME: Tests for database initialization and migrations.
// ABOUTME: Verifies schema creation and XDG path handling.

package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected database file to be created")
	}
}

func TestOpenRunsMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify tables exist
	tables := []string{"notes", "tags", "note_tags", "attachments", "notes_fts"}
	for _, table := range tables {
		var name string
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		if table == "notes_fts" {
			query = "SELECT name FROM sqlite_master WHERE type='table' AND name LIKE ?"
			table = "notes_fts%"
		}
		err := db.QueryRow(query, table).Scan(&name)
		if err != nil {
			t.Errorf("expected table %s to exist: %v", table, err)
		}
	}
}

func TestDefaultPath(t *testing.T) {
	// Save and restore XDG_DATA_HOME
	original := os.Getenv("XDG_DATA_HOME")
	defer func() { _ = os.Setenv("XDG_DATA_HOME", original) }()

	tmpDir := t.TempDir()
	_ = os.Setenv("XDG_DATA_HOME", tmpDir)

	path := DefaultPath()
	expected := filepath.Join(tmpDir, "memo", "memo.db")

	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}
