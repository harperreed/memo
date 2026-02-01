// ABOUTME: SQLite database initialization and connection management.
// ABOUTME: Uses modernc.org/sqlite (pure Go) with FTS5 for full-text search.

package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store provides SQLite-based storage for memo notes, tags, and attachments.
type Store struct {
	db     *sql.DB
	dbPath string
}

// NewStore creates a new Store with the given database path.
// It initializes the schema if the database doesn't exist.
func NewStore(dbPath string) (*Store, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	// Open database with foreign keys enabled
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	store := &Store{
		db:     db,
		dbPath: dbPath,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// initSchema creates all required tables, indexes, and triggers.
func (s *Store) initSchema() error {
	schema := `
	-- Notes table with rowid for FTS5 content_rowid
	CREATE TABLE IF NOT EXISTS notes (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
		id TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	-- Tags table for tag normalization
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	-- Note-tag many-to-many relationship
	CREATE TABLE IF NOT EXISTS note_tags (
		note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (note_id, tag_id)
	);

	-- Attachments with binary blob storage
	CREATE TABLE IF NOT EXISTS attachments (
		id TEXT PRIMARY KEY,
		note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		filename TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		data BLOB NOT NULL,
		created_at DATETIME NOT NULL
	);

	-- Index for note ID lookups
	CREATE INDEX IF NOT EXISTS idx_notes_id ON notes(id);

	-- Index for attachment note_id lookups
	CREATE INDEX IF NOT EXISTS idx_attachments_note_id ON attachments(note_id);

	-- FTS5 virtual table for full-text search
	CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
		title, content, content=notes, content_rowid=rowid
	);
	`

	// Execute schema in a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // Rollback is a no-op if committed

	if _, err := tx.Exec(schema); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	// Create triggers for FTS sync - these need to be created separately
	// because CREATE TRIGGER doesn't support IF NOT EXISTS in older SQLite
	triggers := []struct {
		name string
		sql  string
	}{
		{
			name: "notes_ai",
			sql: `CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes BEGIN
				INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
			END`,
		},
		{
			name: "notes_ad",
			sql: `CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
				INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
			END`,
		},
		{
			name: "notes_au",
			sql: `CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
				INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
				INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
			END`,
		},
	}

	for _, trigger := range triggers {
		if _, err := tx.Exec(trigger.sql); err != nil {
			return fmt.Errorf("create trigger %s: %w", trigger.name, err)
		}
	}

	return tx.Commit()
}

// DefaultDataPath returns the default path for the memo database.
// Uses XDG_DATA_HOME or falls back to ~/.local/share/memo/memo.db.
func DefaultDataPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "memo", "memo.db")
}
