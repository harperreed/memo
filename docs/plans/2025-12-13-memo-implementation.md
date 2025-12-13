# memo Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a CLI notes tool with SQLite storage, FTS5 search, and MCP integration.

**Architecture:** Cobra CLI dispatching to internal packages. SQLite for persistence with FTS5 virtual table for search. MCP server exposes same functionality via tools/resources/prompts.

**Tech Stack:** Go 1.24+, Cobra, modernc.org/sqlite, glamour, fatih/color, google/uuid, go-sdk MCP

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/memo/main.go`
- Create: `Makefile`

**Step 1: Initialize Go module**

```bash
cd /Users/harper/Public/src/personal/notes
go mod init github.com/harper/memo
```

**Step 2: Add dependencies**

```bash
go get github.com/spf13/cobra@latest
go get modernc.org/sqlite@latest
go get github.com/google/uuid@latest
go get github.com/charmbracelet/glamour@latest
go get github.com/fatih/color@latest
go get github.com/modelcontextprotocol/go-sdk@latest
```

**Step 3: Create main.go**

```go
// ABOUTME: Entry point for memo CLI application.
// ABOUTME: Initializes and executes the root command.

package main

import (
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 4: Create Makefile**

```makefile
.PHONY: build test install clean

build:
	go build -o bin/memo ./cmd/memo

test:
	go test -v ./...

install:
	go install ./cmd/memo

clean:
	rm -rf bin/
```

**Step 5: Verify build scaffolding**

```bash
go mod tidy
```

**Step 6: Commit**

```bash
git add -A
git commit -m "feat: initialize project scaffolding"
```

---

## Task 2: Note Model

**Files:**
- Create: `internal/models/note.go`
- Create: `internal/models/note_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for Note model constructor and methods.
// ABOUTME: Validates UUID generation and timestamp handling.

package models

import (
	"testing"
	"time"
)

func TestNewNote(t *testing.T) {
	title := "Test Note"
	content := "This is test content"

	note := NewNote(title, content)

	if note.ID.String() == "" {
		t.Error("expected UUID to be generated")
	}
	if note.Title != title {
		t.Errorf("expected title %q, got %q", title, note.Title)
	}
	if note.Content != content {
		t.Errorf("expected content %q, got %q", content, note.Content)
	}
	if note.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if note.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestNoteTouch(t *testing.T) {
	note := NewNote("Test", "Content")
	originalUpdated := note.UpdatedAt

	time.Sleep(time.Millisecond)
	note.Touch()

	if !note.UpdatedAt.After(originalUpdated) {
		t.Error("expected UpdatedAt to be updated")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/models/... -v
```

Expected: FAIL - package does not exist

**Step 3: Write minimal implementation**

```go
// ABOUTME: Note model representing a markdown note with metadata.
// ABOUTME: Provides constructor and methods for note lifecycle.

package models

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID        uuid.UUID
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewNote(title, content string) *Note {
	now := time.Now()
	return &Note{
		ID:        uuid.New(),
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (n *Note) Touch() {
	n.UpdatedAt = time.Now()
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/models/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add Note model with constructor and Touch method"
```

---

## Task 3: Tag Model

**Files:**
- Create: `internal/models/tag.go`
- Create: `internal/models/tag_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for Tag model.
// ABOUTME: Validates tag creation and name normalization.

package models

import "testing"

func TestNewTag(t *testing.T) {
	tag := NewTag("TestTag")

	if tag.Name != "testtag" {
		t.Errorf("expected lowercase name 'testtag', got %q", tag.Name)
	}
}

func TestNewTagWithSpaces(t *testing.T) {
	tag := NewTag("  My Tag  ")

	if tag.Name != "my tag" {
		t.Errorf("expected trimmed lowercase 'my tag', got %q", tag.Name)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/models/... -v -run TestNewTag
```

Expected: FAIL - NewTag not defined

**Step 3: Write minimal implementation**

```go
// ABOUTME: Tag model for categorizing notes.
// ABOUTME: Normalizes tag names to lowercase with trimmed whitespace.

package models

import "strings"

type Tag struct {
	ID   int64
	Name string
}

func NewTag(name string) *Tag {
	return &Tag{
		Name: strings.ToLower(strings.TrimSpace(name)),
	}
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/models/... -v -run TestNewTag
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add Tag model with name normalization"
```

---

## Task 4: Attachment Model

**Files:**
- Create: `internal/models/attachment.go`
- Create: `internal/models/attachment_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for Attachment model.
// ABOUTME: Validates attachment creation with file metadata.

package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewAttachment(t *testing.T) {
	noteID := uuid.New()
	filename := "test.pdf"
	mimeType := "application/pdf"
	data := []byte("fake pdf content")

	att := NewAttachment(noteID, filename, mimeType, data)

	if att.ID.String() == "" {
		t.Error("expected UUID to be generated")
	}
	if att.NoteID != noteID {
		t.Errorf("expected NoteID %v, got %v", noteID, att.NoteID)
	}
	if att.Filename != filename {
		t.Errorf("expected filename %q, got %q", filename, att.Filename)
	}
	if att.MimeType != mimeType {
		t.Errorf("expected mimeType %q, got %q", mimeType, att.MimeType)
	}
	if string(att.Data) != string(data) {
		t.Error("expected data to match")
	}
	if att.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/models/... -v -run TestNewAttachment
```

Expected: FAIL - NewAttachment not defined

**Step 3: Write minimal implementation**

```go
// ABOUTME: Attachment model for binary files attached to notes.
// ABOUTME: Stores file content as blob with metadata.

package models

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID        uuid.UUID
	NoteID    uuid.UUID
	Filename  string
	MimeType  string
	Data      []byte
	CreatedAt time.Time
}

func NewAttachment(noteID uuid.UUID, filename, mimeType string, data []byte) *Attachment {
	return &Attachment{
		ID:        uuid.New(),
		NoteID:    noteID,
		Filename:  filename,
		MimeType:  mimeType,
		Data:      data,
		CreatedAt: time.Now(),
	}
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/models/... -v -run TestNewAttachment
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add Attachment model for binary file storage"
```

---

## Task 5: Database Connection and Migrations

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/db_test.go`

**Step 1: Write the failing test**

```go
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
	defer db.Close()

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
	defer db.Close()

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
	defer os.Setenv("XDG_DATA_HOME", original)

	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tmpDir)

	path := DefaultPath()
	expected := filepath.Join(tmpDir, "memo", "memo.db")

	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v
```

Expected: FAIL - package does not exist

**Step 3: Write minimal implementation**

```go
// ABOUTME: Database connection and schema management for memo.
// ABOUTME: Handles XDG paths, SQLite initialization, and migrations.

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS note_tags (
    note_id TEXT REFERENCES notes(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);

CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    note_id TEXT REFERENCES notes(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    data BLOB NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    title, content, content='notes', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, title, content) VALUES (NEW.rowid, NEW.title, NEW.content);
END;

CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES('delete', OLD.rowid, OLD.title, OLD.content);
END;

CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES('delete', OLD.rowid, OLD.title, OLD.content);
    INSERT INTO notes_fts(rowid, title, content) VALUES (NEW.rowid, NEW.title, NEW.content);
END;
`

func Open(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Enable foreign keys for this connection
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}

func DefaultPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "memo", "memo.db")
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/db/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add database connection with schema migrations and XDG support"
```

---

## Task 6: Note CRUD Operations

**Files:**
- Create: `internal/db/notes.go`
- Create: `internal/db/notes_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for note database operations.
// ABOUTME: Covers create, read, update, delete, and prefix matching.

package db

import (
	"path/filepath"
	"testing"

	"github.com/harper/memo/internal/models"
)

func TestCreateAndGetNote(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Test Title", "Test content")
	if err := CreateNote(db, note); err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	got, err := GetNoteByID(db, note.ID)
	if err != nil {
		t.Fatalf("failed to get note: %v", err)
	}

	if got.Title != note.Title {
		t.Errorf("expected title %q, got %q", note.Title, got.Title)
	}
	if got.Content != note.Content {
		t.Errorf("expected content %q, got %q", note.Content, got.Content)
	}
}

func TestGetNoteByPrefix(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Test", "Content")
	if err := CreateNote(db, note); err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	prefix := note.ID.String()[:8]
	got, err := GetNoteByPrefix(db, prefix)
	if err != nil {
		t.Fatalf("failed to get note by prefix: %v", err)
	}

	if got.ID != note.ID {
		t.Errorf("expected ID %v, got %v", note.ID, got.ID)
	}
}

func TestGetNoteByPrefixTooShort(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	_, err = GetNoteByPrefix(db, "abc")
	if err == nil {
		t.Error("expected error for short prefix")
	}
}

func TestListNotes(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note1 := models.NewNote("First", "Content 1")
	note2 := models.NewNote("Second", "Content 2")
	CreateNote(db, note1)
	CreateNote(db, note2)

	notes, err := ListNotes(db, nil, 20)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}

func TestUpdateNote(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Original", "Original content")
	CreateNote(db, note)

	note.Title = "Updated"
	note.Content = "Updated content"
	note.Touch()

	if err := UpdateNote(db, note); err != nil {
		t.Fatalf("failed to update note: %v", err)
	}

	got, _ := GetNoteByID(db, note.ID)
	if got.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %q", got.Title)
	}
}

func TestDeleteNote(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("ToDelete", "Content")
	CreateNote(db, note)

	if err := DeleteNote(db, note.ID); err != nil {
		t.Fatalf("failed to delete note: %v", err)
	}

	_, err = GetNoteByID(db, note.ID)
	if err == nil {
		t.Error("expected error getting deleted note")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v -run "Note"
```

Expected: FAIL - functions not defined

**Step 3: Write minimal implementation**

```go
// ABOUTME: Database operations for notes.
// ABOUTME: Provides CRUD and prefix-based lookup for notes.

package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

var ErrPrefixTooShort = errors.New("prefix must be at least 6 characters")
var ErrAmbiguousPrefix = errors.New("prefix matches multiple notes")
var ErrNoteNotFound = errors.New("note not found")

func CreateNote(db *sql.DB, note *models.Note) error {
	_, err := db.Exec(
		`INSERT INTO notes (id, title, content, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		note.ID.String(), note.Title, note.Content, note.CreatedAt, note.UpdatedAt,
	)
	return err
}

func GetNoteByID(db *sql.DB, id uuid.UUID) (*models.Note, error) {
	note := &models.Note{}
	var idStr string
	err := db.QueryRow(
		`SELECT id, title, content, created_at, updated_at FROM notes WHERE id = ?`,
		id.String(),
	).Scan(&idStr, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNoteNotFound
	}
	if err != nil {
		return nil, err
	}
	note.ID, _ = uuid.Parse(idStr)
	return note, nil
}

func GetNoteByPrefix(db *sql.DB, prefix string) (*models.Note, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	rows, err := db.Query(
		`SELECT id, title, content, created_at, updated_at FROM notes WHERE id LIKE ?`,
		prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		var idStr string
		if err := rows.Scan(&idStr, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		note.ID, _ = uuid.Parse(idStr)
		notes = append(notes, note)
	}

	if len(notes) == 0 {
		return nil, ErrNoteNotFound
	}
	if len(notes) > 1 {
		return nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(notes))
	}
	return notes[0], nil
}

func ListNotes(db *sql.DB, tag *string, limit int) ([]*models.Note, error) {
	var rows *sql.Rows
	var err error

	if tag != nil {
		rows, err = db.Query(
			`SELECT DISTINCT n.id, n.title, n.content, n.created_at, n.updated_at
			 FROM notes n
			 JOIN note_tags nt ON n.id = nt.note_id
			 JOIN tags t ON nt.tag_id = t.id
			 WHERE t.name = ?
			 ORDER BY n.updated_at DESC
			 LIMIT ?`,
			*tag, limit,
		)
	} else {
		rows, err = db.Query(
			`SELECT id, title, content, created_at, updated_at FROM notes
			 ORDER BY updated_at DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		var idStr string
		if err := rows.Scan(&idStr, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		note.ID, _ = uuid.Parse(idStr)
		notes = append(notes, note)
	}
	return notes, nil
}

func UpdateNote(db *sql.DB, note *models.Note) error {
	result, err := db.Exec(
		`UPDATE notes SET title = ?, content = ?, updated_at = ? WHERE id = ?`,
		note.Title, note.Content, time.Now(), note.ID.String(),
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNoteNotFound
	}
	return nil
}

func DeleteNote(db *sql.DB, id uuid.UUID) error {
	result, err := db.Exec(`DELETE FROM notes WHERE id = ?`, id.String())
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNoteNotFound
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/db/... -v -run "Note"
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add note CRUD operations with prefix matching"
```

---

## Task 7: Tag Operations

**Files:**
- Create: `internal/db/tags.go`
- Create: `internal/db/tags_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for tag database operations.
// ABOUTME: Covers tag creation, association, and listing.

package db

import (
	"path/filepath"
	"testing"

	"github.com/harper/memo/internal/models"
)

func TestGetOrCreateTag(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	tag1, err := GetOrCreateTag(db, "test")
	if err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	tag2, err := GetOrCreateTag(db, "test")
	if err != nil {
		t.Fatalf("failed to get existing tag: %v", err)
	}

	if tag1.ID != tag2.ID {
		t.Error("expected same tag ID for same name")
	}
}

func TestAddAndRemoveTagFromNote(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Test", "Content")
	CreateNote(db, note)

	if err := AddTagToNote(db, note.ID, "important"); err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}

	tags, err := GetNoteTags(db, note.ID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}

	if len(tags) != 1 || tags[0].Name != "important" {
		t.Errorf("expected [important], got %v", tags)
	}

	if err := RemoveTagFromNote(db, note.ID, "important"); err != nil {
		t.Fatalf("failed to remove tag: %v", err)
	}

	tags, _ = GetNoteTags(db, note.ID)
	if len(tags) != 0 {
		t.Errorf("expected no tags after removal, got %v", tags)
	}
}

func TestListAllTags(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note1 := models.NewNote("Note 1", "Content")
	note2 := models.NewNote("Note 2", "Content")
	CreateNote(db, note1)
	CreateNote(db, note2)

	AddTagToNote(db, note1.ID, "shared")
	AddTagToNote(db, note2.ID, "shared")
	AddTagToNote(db, note1.ID, "unique")

	tags, err := ListAllTags(db)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v -run "Tag"
```

Expected: FAIL - functions not defined

**Step 3: Write minimal implementation**

```go
// ABOUTME: Database operations for tags.
// ABOUTME: Handles tag creation, note associations, and listing.

package db

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

func GetOrCreateTag(db *sql.DB, name string) (*models.Tag, error) {
	tag := models.NewTag(name)

	// Try to get existing
	err := db.QueryRow(`SELECT id FROM tags WHERE name = ?`, tag.Name).Scan(&tag.ID)
	if err == nil {
		return tag, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new
	result, err := db.Exec(`INSERT INTO tags (name) VALUES (?)`, tag.Name)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	tag.ID = id
	return tag, nil
}

func AddTagToNote(db *sql.DB, noteID uuid.UUID, tagName string) error {
	tag, err := GetOrCreateTag(db, tagName)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)`,
		noteID.String(), tag.ID,
	)
	return err
}

func RemoveTagFromNote(db *sql.DB, noteID uuid.UUID, tagName string) error {
	tag := models.NewTag(tagName)
	_, err := db.Exec(
		`DELETE FROM note_tags WHERE note_id = ? AND tag_id = (SELECT id FROM tags WHERE name = ?)`,
		noteID.String(), tag.Name,
	)
	return err
}

func GetNoteTags(db *sql.DB, noteID uuid.UUID) ([]*models.Tag, error) {
	rows, err := db.Query(
		`SELECT t.id, t.name FROM tags t
		 JOIN note_tags nt ON t.id = nt.tag_id
		 WHERE nt.note_id = ?
		 ORDER BY t.name`,
		noteID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		tag := &models.Tag{}
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

type TagWithCount struct {
	Tag   *models.Tag
	Count int
}

func ListAllTags(db *sql.DB) ([]*TagWithCount, error) {
	rows, err := db.Query(
		`SELECT t.id, t.name, COUNT(nt.note_id) as count
		 FROM tags t
		 LEFT JOIN note_tags nt ON t.id = nt.tag_id
		 GROUP BY t.id
		 ORDER BY t.name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*TagWithCount
	for rows.Next() {
		tc := &TagWithCount{Tag: &models.Tag{}}
		if err := rows.Scan(&tc.Tag.ID, &tc.Tag.Name, &tc.Count); err != nil {
			return nil, err
		}
		tags = append(tags, tc)
	}
	return tags, nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/db/... -v -run "Tag"
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add tag operations with note associations"
```

---

## Task 8: Attachment Operations

**Files:**
- Create: `internal/db/attachments.go`
- Create: `internal/db/attachments_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for attachment database operations.
// ABOUTME: Covers attachment CRUD with blob storage.

package db

import (
	"path/filepath"
	"testing"

	"github.com/harper/memo/internal/models"
)

func TestCreateAndGetAttachment(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Test", "Content")
	CreateNote(db, note)

	data := []byte("test file content")
	att := models.NewAttachment(note.ID, "test.txt", "text/plain", data)

	if err := CreateAttachment(db, att); err != nil {
		t.Fatalf("failed to create attachment: %v", err)
	}

	got, err := GetAttachment(db, att.ID)
	if err != nil {
		t.Fatalf("failed to get attachment: %v", err)
	}

	if got.Filename != att.Filename {
		t.Errorf("expected filename %q, got %q", att.Filename, got.Filename)
	}
	if string(got.Data) != string(data) {
		t.Error("expected data to match")
	}
}

func TestListNoteAttachments(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Test", "Content")
	CreateNote(db, note)

	att1 := models.NewAttachment(note.ID, "file1.txt", "text/plain", []byte("content1"))
	att2 := models.NewAttachment(note.ID, "file2.txt", "text/plain", []byte("content2"))
	CreateAttachment(db, att1)
	CreateAttachment(db, att2)

	attachments, err := ListNoteAttachments(db, note.ID)
	if err != nil {
		t.Fatalf("failed to list attachments: %v", err)
	}

	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(attachments))
	}
}

func TestDeleteAttachment(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Test", "Content")
	CreateNote(db, note)

	att := models.NewAttachment(note.ID, "test.txt", "text/plain", []byte("content"))
	CreateAttachment(db, att)

	if err := DeleteAttachment(db, att.ID); err != nil {
		t.Fatalf("failed to delete attachment: %v", err)
	}

	_, err = GetAttachment(db, att.ID)
	if err == nil {
		t.Error("expected error getting deleted attachment")
	}
}

func TestCascadeDeleteAttachments(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Test", "Content")
	CreateNote(db, note)

	att := models.NewAttachment(note.ID, "test.txt", "text/plain", []byte("content"))
	CreateAttachment(db, att)

	// Delete note should cascade to attachments
	DeleteNote(db, note.ID)

	_, err = GetAttachment(db, att.ID)
	if err == nil {
		t.Error("expected attachment to be deleted with note")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v -run "Attachment"
```

Expected: FAIL - functions not defined

**Step 3: Write minimal implementation**

```go
// ABOUTME: Database operations for attachments.
// ABOUTME: Handles blob storage and retrieval for note attachments.

package db

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

var ErrAttachmentNotFound = errors.New("attachment not found")

func CreateAttachment(db *sql.DB, att *models.Attachment) error {
	_, err := db.Exec(
		`INSERT INTO attachments (id, note_id, filename, mime_type, data, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		att.ID.String(), att.NoteID.String(), att.Filename, att.MimeType, att.Data, att.CreatedAt,
	)
	return err
}

func GetAttachment(db *sql.DB, id uuid.UUID) (*models.Attachment, error) {
	att := &models.Attachment{}
	var idStr, noteIDStr string

	err := db.QueryRow(
		`SELECT id, note_id, filename, mime_type, data, created_at
		 FROM attachments WHERE id = ?`,
		id.String(),
	).Scan(&idStr, &noteIDStr, &att.Filename, &att.MimeType, &att.Data, &att.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}

	att.ID, _ = uuid.Parse(idStr)
	att.NoteID, _ = uuid.Parse(noteIDStr)
	return att, nil
}

func GetAttachmentByPrefix(db *sql.DB, prefix string) (*models.Attachment, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	att := &models.Attachment{}
	var idStr, noteIDStr string

	rows, err := db.Query(
		`SELECT id, note_id, filename, mime_type, data, created_at
		 FROM attachments WHERE id LIKE ?`,
		prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		count++
		if count > 1 {
			return nil, ErrAmbiguousPrefix
		}
		if err := rows.Scan(&idStr, &noteIDStr, &att.Filename, &att.MimeType, &att.Data, &att.CreatedAt); err != nil {
			return nil, err
		}
	}

	if count == 0 {
		return nil, ErrAttachmentNotFound
	}

	att.ID, _ = uuid.Parse(idStr)
	att.NoteID, _ = uuid.Parse(noteIDStr)
	return att, nil
}

type AttachmentMeta struct {
	ID        uuid.UUID
	NoteID    uuid.UUID
	Filename  string
	MimeType  string
	CreatedAt string
}

func ListNoteAttachments(db *sql.DB, noteID uuid.UUID) ([]*AttachmentMeta, error) {
	rows, err := db.Query(
		`SELECT id, note_id, filename, mime_type, created_at
		 FROM attachments WHERE note_id = ?
		 ORDER BY created_at DESC`,
		noteID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*AttachmentMeta
	for rows.Next() {
		att := &AttachmentMeta{}
		var idStr, noteIDStr string
		if err := rows.Scan(&idStr, &noteIDStr, &att.Filename, &att.MimeType, &att.CreatedAt); err != nil {
			return nil, err
		}
		att.ID, _ = uuid.Parse(idStr)
		att.NoteID, _ = uuid.Parse(noteIDStr)
		attachments = append(attachments, att)
	}
	return attachments, nil
}

func DeleteAttachment(db *sql.DB, id uuid.UUID) error {
	result, err := db.Exec(`DELETE FROM attachments WHERE id = ?`, id.String())
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAttachmentNotFound
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/db/... -v -run "Attachment"
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add attachment CRUD operations with blob storage"
```

---

## Task 9: FTS5 Search Operations

**Files:**
- Create: `internal/db/search.go`
- Create: `internal/db/search_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for FTS5 full-text search operations.
// ABOUTME: Validates search queries and result ranking.

package db

import (
	"path/filepath"
	"testing"

	"github.com/harper/memo/internal/models"
)

func TestSearchNotes(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note1 := models.NewNote("Go Programming", "Learn about goroutines and channels")
	note2 := models.NewNote("Python Basics", "Introduction to Python programming")
	note3 := models.NewNote("Cooking Recipes", "How to make pasta")
	CreateNote(db, note1)
	CreateNote(db, note2)
	CreateNote(db, note3)

	results, err := SearchNotes(db, "programming", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results for 'programming', got %d", len(results))
	}
}

func TestSearchNotesNoResults(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	note := models.NewNote("Test", "Content")
	CreateNote(db, note)

	results, err := SearchNotes(db, "nonexistent", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchNotesRanking(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Note with term in title should rank higher
	note1 := models.NewNote("Other Topic", "This mentions golang once")
	note2 := models.NewNote("Golang Guide", "Golang golang golang everywhere")
	CreateNote(db, note1)
	CreateNote(db, note2)

	results, err := SearchNotes(db, "golang", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Note with more matches should be first
	if results[0].ID != note2.ID {
		t.Error("expected note with more matches to rank first")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/db/... -v -run "Search"
```

Expected: FAIL - SearchNotes not defined

**Step 3: Write minimal implementation**

```go
// ABOUTME: FTS5 full-text search operations for notes.
// ABOUTME: Provides ranked search across note titles and content.

package db

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

type SearchResult struct {
	*models.Note
	Rank float64
}

func SearchNotes(db *sql.DB, query string, limit int) ([]*SearchResult, error) {
	rows, err := db.Query(
		`SELECT n.id, n.title, n.content, n.created_at, n.updated_at, rank
		 FROM notes_fts
		 JOIN notes n ON notes_fts.rowid = n.rowid
		 WHERE notes_fts MATCH ?
		 ORDER BY rank
		 LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		result := &SearchResult{Note: &models.Note{}}
		var idStr string
		if err := rows.Scan(&idStr, &result.Title, &result.Content, &result.CreatedAt, &result.UpdatedAt, &result.Rank); err != nil {
			return nil, err
		}
		result.ID, _ = uuid.Parse(idStr)
		results = append(results, result)
	}
	return results, nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/db/... -v -run "Search"
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add FTS5 full-text search with ranking"
```

---

## Task 10: UI Formatting with Glamour

**Files:**
- Create: `internal/ui/format.go`
- Create: `internal/ui/format_test.go`

**Step 1: Write the failing test**

```go
// ABOUTME: Tests for terminal UI formatting functions.
// ABOUTME: Validates note display and markdown rendering.

package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

func TestFormatNoteListItem(t *testing.T) {
	note := &models.Note{
		ID:        uuid.New(),
		Title:     "Test Note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tags := []*models.Tag{{Name: "important"}, {Name: "work"}}

	output := FormatNoteListItem(note, tags)

	if !strings.Contains(output, note.ID.String()[:6]) {
		t.Error("expected output to contain ID prefix")
	}
	if !strings.Contains(output, "Test Note") {
		t.Error("expected output to contain title")
	}
	if !strings.Contains(output, "important") {
		t.Error("expected output to contain tag")
	}
}

func TestFormatNoteContent(t *testing.T) {
	content := "# Hello\n\nThis is **bold** text."

	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFormatTagList(t *testing.T) {
	tags := []TagCount{
		{Name: "work", Count: 5},
		{Name: "personal", Count: 3},
	}

	output := FormatTagList(tags)

	if !strings.Contains(output, "work") {
		t.Error("expected output to contain 'work'")
	}
	if !strings.Contains(output, "5") {
		t.Error("expected output to contain count '5'")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/... -v
```

Expected: FAIL - package does not exist

**Step 3: Write minimal implementation**

```go
// ABOUTME: Terminal UI formatting for memo output.
// ABOUTME: Uses glamour for markdown and fatih/color for styling.

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/harper/memo/internal/models"
)

var (
	faint   = color.New(color.Faint).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
)

type TagCount struct {
	Name  string
	Count int
}

func FormatNoteListItem(note *models.Note, tags []*models.Tag) string {
	var sb strings.Builder

	// ID prefix and title
	idPrefix := note.ID.String()[:6]
	sb.WriteString(fmt.Sprintf("  %s  %s\n", faint(idPrefix), bold(note.Title)))

	// Tags line if present
	if len(tags) > 0 {
		var tagNames []string
		for _, t := range tags {
			tagNames = append(tagNames, t.Name)
		}
		sb.WriteString(fmt.Sprintf("         %s %s\n",
			faint("Tags:"),
			cyan(strings.Join(tagNames, ", "))))
	}

	// Date
	sb.WriteString(fmt.Sprintf("         %s %s\n",
		faint("Updated:"),
		faint(note.UpdatedAt.Format("2006-01-02 15:04"))))

	return sb.String()
}

func FormatNoteContent(content string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return content, nil // Fallback to raw content
	}

	out, err := renderer.Render(content)
	if err != nil {
		return content, nil
	}
	return out, nil
}

func FormatNoteHeader(note *models.Note, tags []*models.Tag) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", bold(note.Title)))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("ID:"), faint(note.ID.String())))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("Created:"), faint(note.CreatedAt.Format("2006-01-02 15:04"))))
	sb.WriteString(fmt.Sprintf("%s %s\n", faint("Updated:"), faint(note.UpdatedAt.Format("2006-01-02 15:04"))))

	if len(tags) > 0 {
		var tagNames []string
		for _, t := range tags {
			tagNames = append(tagNames, t.Name)
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", faint("Tags:"), cyan(strings.Join(tagNames, ", "))))
	}

	sb.WriteString(Separator())
	return sb.String()
}

func FormatTagList(tags []TagCount) string {
	var sb strings.Builder

	for _, t := range tags {
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			cyan(t.Name),
			faint(fmt.Sprintf("(%d)", t.Count))))
	}

	return sb.String()
}

func FormatAttachmentList(attachments []AttachmentInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n%s\n", bold("Attachments:")))
	for _, a := range attachments {
		sb.WriteString(fmt.Sprintf("  %s  %s %s\n",
			faint(a.ID[:6]),
			a.Filename,
			faint(fmt.Sprintf("[%s]", a.MimeType))))
	}

	return sb.String()
}

type AttachmentInfo struct {
	ID       string
	Filename string
	MimeType string
}

func Separator() string {
	return faint(strings.Repeat("─", 50)) + "\n"
}

func Success(msg string) string {
	return color.New(color.FgGreen).Sprint("✓ ") + msg
}

func Error(msg string) string {
	return color.New(color.FgRed).Sprint("✗ ") + msg
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/ui/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: add UI formatting with glamour markdown rendering"
```

---

## Task 11: Root Command and DB Initialization

**Files:**
- Create: `cmd/memo/root.go`
- Modify: `cmd/memo/main.go`

**Step 1: Create root command**

```go
// ABOUTME: Root command for memo CLI with database initialization.
// ABOUTME: Handles global flags and persistent pre/post run hooks.

package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/harper/memo/internal/db"
	"github.com/spf13/cobra"
)

var (
	dbPath string
	dbConn *sql.DB
)

var rootCmd = &cobra.Command{
	Use:   "memo",
	Short: "A CLI notes tool with markdown support",
	Long:  `memo is a command-line notes tool that stores markdown notes with tags and attachments in SQLite.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip DB init for version command
		if cmd.Name() == "version" {
			return nil
		}

		var err error
		if dbPath == "" {
			dbPath = db.DefaultPath()
		}
		dbConn, err = db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if dbConn != nil {
			return dbConn.Close()
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database path (default: ~/.local/share/memo/memo.db)")
}

func Execute() error {
	return rootCmd.Execute()
}
```

**Step 2: Update main.go**

```go
// ABOUTME: Entry point for memo CLI application.
// ABOUTME: Initializes and executes the root command.

package main

import (
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 3: Verify it compiles**

```bash
go build -o bin/memo ./cmd/memo
./bin/memo --help
```

Expected: Help output displayed

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: add root command with database initialization"
```

---

## Task 12: Add Command

**Files:**
- Create: `cmd/memo/add.go`

**Step 1: Write the add command**

```go
// ABOUTME: Add command for creating new notes.
// ABOUTME: Supports inline content, file input, or $EDITOR.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/models"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new note",
	Long:  `Create a new note with the given title. Content can be provided via --content, --file, or $EDITOR.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]

		tagsFlag, _ := cmd.Flags().GetString("tags")
		contentFlag, _ := cmd.Flags().GetString("content")
		fileFlag, _ := cmd.Flags().GetString("file")

		var content string
		var err error

		switch {
		case contentFlag != "":
			content = contentFlag
		case fileFlag != "":
			data, err := os.ReadFile(fileFlag)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			content = string(data)
		default:
			content, err = openEditor("")
			if err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}
		}

		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("note content cannot be empty")
		}

		note := models.NewNote(title, content)
		if err := db.CreateNote(dbConn, note); err != nil {
			return fmt.Errorf("failed to create note: %w", err)
		}

		// Add tags if provided
		if tagsFlag != "" {
			for _, tag := range strings.Split(tagsFlag, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					if err := db.AddTagToNote(dbConn, note.ID, tag); err != nil {
						return fmt.Errorf("failed to add tag %q: %w", tag, err)
					}
				}
			}
		}

		fmt.Println(ui.Success(fmt.Sprintf("Created note %s", note.ID.String()[:6])))
		return nil
	},
}

func init() {
	addCmd.Flags().StringP("tags", "t", "", "comma-separated tags")
	addCmd.Flags().StringP("content", "c", "", "note content (inline)")
	addCmd.Flags().StringP("file", "f", "", "read content from file")
	rootCmd.AddCommand(addCmd)
}

func openEditor(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	tmpFile, err := os.CreateTemp("", "memo-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if initial != "" {
		tmpFile.WriteString(initial)
	}
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return string(data), nil
}
```

**Step 2: Test manually**

```bash
go build -o bin/memo ./cmd/memo
echo "Test content" | ./bin/memo add "Test Note" --content "Test content"
```

Expected: "✓ Created note xxxxxx"

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: add 'add' command for creating notes"
```

---

## Task 13: List Command

**Files:**
- Create: `cmd/memo/list.go`

**Step 1: Write the list command**

```go
// ABOUTME: List command for displaying notes.
// ABOUTME: Supports filtering by tag and search queries.

package main

import (
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes",
	Long:  `List all notes, optionally filtered by tag or search query.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tagFlag, _ := cmd.Flags().GetString("tag")
		searchFlag, _ := cmd.Flags().GetString("search")
		limitFlag, _ := cmd.Flags().GetInt("limit")

		if searchFlag != "" {
			results, err := db.SearchNotes(dbConn, searchFlag, limitFlag)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No notes found.")
				return nil
			}

			for _, result := range results {
				tags, _ := db.GetNoteTags(dbConn, result.ID)
				fmt.Print(ui.FormatNoteListItem(result.Note, tags))
			}
			return nil
		}

		var tag *string
		if tagFlag != "" {
			tag = &tagFlag
		}

		notes, err := db.ListNotes(dbConn, tag, limitFlag)
		if err != nil {
			return fmt.Errorf("failed to list notes: %w", err)
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
			return nil
		}

		for _, note := range notes {
			tags, _ := db.GetNoteTags(dbConn, note.ID)
			fmt.Print(ui.FormatNoteListItem(note, tags))
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringP("tag", "t", "", "filter by tag")
	listCmd.Flags().StringP("search", "s", "", "search query")
	listCmd.Flags().IntP("limit", "n", 20, "number of results")
	rootCmd.AddCommand(listCmd)
}
```

**Step 2: Commit**

```bash
git add -A
git commit -m "feat: add 'list' command with tag and search filtering"
```

---

## Task 14: Show Command

**Files:**
- Create: `cmd/memo/show.go`

**Step 1: Write the show command**

```go
// ABOUTME: Show command for displaying a single note.
// ABOUTME: Renders markdown content with glamour.

package main

import (
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id-prefix>",
	Short: "Show a note",
	Long:  `Display a note's full content with rendered markdown.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		tags, _ := db.GetNoteTags(dbConn, note.ID)
		attachments, _ := db.ListNoteAttachments(dbConn, note.ID)

		// Print header
		fmt.Print(ui.FormatNoteHeader(note, tags))

		// Print content
		content, _ := ui.FormatNoteContent(note.Content)
		fmt.Print(content)

		// Print attachments if any
		if len(attachments) > 0 {
			var attInfos []ui.AttachmentInfo
			for _, a := range attachments {
				attInfos = append(attInfos, ui.AttachmentInfo{
					ID:       a.ID.String(),
					Filename: a.Filename,
					MimeType: a.MimeType,
				})
			}
			fmt.Print(ui.FormatAttachmentList(attInfos))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
```

**Step 2: Commit**

```bash
git add -A
git commit -m "feat: add 'show' command for displaying notes"
```

---

## Task 15: Edit Command

**Files:**
- Create: `cmd/memo/edit.go`

**Step 1: Write the edit command**

```go
// ABOUTME: Edit command for modifying existing notes.
// ABOUTME: Opens note content in $EDITOR for modification.

package main

import (
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id-prefix>",
	Short: "Edit a note",
	Long:  `Open a note in $EDITOR for editing.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		newContent, err := openEditor(note.Content)
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		if newContent == note.Content {
			fmt.Println("No changes made.")
			return nil
		}

		note.Content = newContent
		note.Touch()

		if err := db.UpdateNote(dbConn, note); err != nil {
			return fmt.Errorf("failed to update note: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Updated note %s", note.ID.String()[:6])))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
```

**Step 2: Commit**

```bash
git add -A
git commit -m "feat: add 'edit' command for modifying notes"
```

---

## Task 16: Remove Command

**Files:**
- Create: `cmd/memo/rm.go`

**Step 1: Write the rm command**

```go
// ABOUTME: Remove command for deleting notes.
// ABOUTME: Includes confirmation prompt before deletion.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <id-prefix>",
	Short: "Remove a note",
	Long:  `Delete a note and all its attachments.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		force, _ := cmd.Flags().GetBool("force")

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if !force {
			fmt.Printf("Delete note %q (%s)? [y/N] ", note.Title, note.ID.String()[:6])
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := db.DeleteNote(dbConn, note.ID); err != nil {
			return fmt.Errorf("failed to delete note: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Deleted note %s", note.ID.String()[:6])))
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolP("force", "f", false, "skip confirmation")
	rootCmd.AddCommand(rmCmd)
}
```

**Step 2: Commit**

```bash
git add -A
git commit -m "feat: add 'rm' command for deleting notes"
```

---

## Task 17: Tag Command

**Files:**
- Create: `cmd/memo/tag.go`

**Step 1: Write the tag command with subcommands**

```go
// ABOUTME: Tag command for managing note tags.
// ABOUTME: Provides add, rm, and list subcommands.

package main

import (
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
	Long:  `Add, remove, or list tags on notes.`,
}

var tagAddCmd = &cobra.Command{
	Use:   "add <id-prefix> <tag>",
	Short: "Add a tag to a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		tagName := args[1]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if err := db.AddTagToNote(dbConn, note.ID, tagName); err != nil {
			return fmt.Errorf("failed to add tag: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Added tag %q to note %s", tagName, note.ID.String()[:6])))
		return nil
	},
}

var tagRmCmd = &cobra.Command{
	Use:   "rm <id-prefix> <tag>",
	Short: "Remove a tag from a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		tagName := args[1]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		if err := db.RemoveTagFromNote(dbConn, note.ID, tagName); err != nil {
			return fmt.Errorf("failed to remove tag: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Removed tag %q from note %s", tagName, note.ID.String()[:6])))
		return nil
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		tags, err := db.ListAllTags(dbConn)
		if err != nil {
			return fmt.Errorf("failed to list tags: %w", err)
		}

		if len(tags) == 0 {
			fmt.Println("No tags found.")
			return nil
		}

		var tagCounts []ui.TagCount
		for _, t := range tags {
			tagCounts = append(tagCounts, ui.TagCount{
				Name:  t.Tag.Name,
				Count: t.Count,
			})
		}
		fmt.Print(ui.FormatTagList(tagCounts))
		return nil
	},
}

func init() {
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRmCmd)
	tagCmd.AddCommand(tagListCmd)
	rootCmd.AddCommand(tagCmd)
}
```

**Step 2: Commit**

```bash
git add -A
git commit -m "feat: add 'tag' command with add/rm/list subcommands"
```

---

## Task 18: Attach Command

**Files:**
- Create: `cmd/memo/attach.go`

**Step 1: Write the attach command**

```go
// ABOUTME: Attach command for managing note attachments.
// ABOUTME: Provides add and get subcommands for binary files.

package main

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/models"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <id-prefix> <file>",
	Short: "Add an attachment to a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		filePath := args[1]

		note, err := db.GetNoteByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get note: %w", err)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		filename := filepath.Base(filePath)
		mimeType := mime.TypeByExtension(filepath.Ext(filePath))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		att := models.NewAttachment(note.ID, filename, mimeType, data)
		if err := db.CreateAttachment(dbConn, att); err != nil {
			return fmt.Errorf("failed to create attachment: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Added attachment %s to note %s", att.ID.String()[:6], note.ID.String()[:6])))
		return nil
	},
}

var attachGetCmd = &cobra.Command{
	Use:   "get <attachment-id-prefix>",
	Short: "Extract an attachment to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := args[0]
		outputPath, _ := cmd.Flags().GetString("output")

		att, err := db.GetAttachmentByPrefix(dbConn, prefix)
		if err != nil {
			return fmt.Errorf("failed to get attachment: %w", err)
		}

		if outputPath == "" {
			outputPath = att.Filename
		}

		if outputPath == "-" {
			_, err = io.Copy(os.Stdout, bytes.NewReader(att.Data))
			return err
		}

		if err := os.WriteFile(outputPath, att.Data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Extracted %s to %s", att.Filename, outputPath)))
		return nil
	},
}

func init() {
	attachGetCmd.Flags().StringP("output", "o", "", "output path (default: original filename)")
	attachCmd.AddCommand(attachGetCmd)
	rootCmd.AddCommand(attachCmd)
}
```

Note: Add `"bytes"` to imports in attach.go.

**Step 2: Commit**

```bash
git add -A
git commit -m "feat: add 'attach' command for managing attachments"
```

---

## Task 19: Export Command

**Files:**
- Create: `cmd/memo/export.go`

**Step 1: Write the export command**

```go
// ABOUTME: Export command for backing up notes.
// ABOUTME: Supports JSON and markdown export formats.

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ExportNote struct {
	ID          string             `json:"id" yaml:"id"`
	Title       string             `json:"title" yaml:"title"`
	Content     string             `json:"content" yaml:"content"`
	Tags        []string           `json:"tags" yaml:"tags"`
	CreatedAt   time.Time          `json:"created_at" yaml:"created"`
	UpdatedAt   time.Time          `json:"updated_at" yaml:"updated"`
	Attachments []ExportAttachment `json:"attachments,omitempty" yaml:"-"`
}

type ExportAttachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // base64 encoded
}

type ExportData struct {
	ExportedAt time.Time    `json:"exported_at"`
	Version    string       `json:"version"`
	Notes      []ExportNote `json:"notes"`
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export notes",
	Long:  `Export notes to JSON or markdown format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		outputPath, _ := cmd.Flags().GetString("output")
		notePrefix, _ := cmd.Flags().GetString("note")

		var notes []*db.SearchResult
		var err error

		if notePrefix != "" {
			note, err := db.GetNoteByPrefix(dbConn, notePrefix)
			if err != nil {
				return fmt.Errorf("failed to get note: %w", err)
			}
			notes = append(notes, &db.SearchResult{Note: note})
		} else {
			allNotes, err := db.ListNotes(dbConn, nil, 10000)
			if err != nil {
				return fmt.Errorf("failed to list notes: %w", err)
			}
			for _, n := range allNotes {
				notes = append(notes, &db.SearchResult{Note: n})
			}
		}

		switch format {
		case "json":
			return exportJSON(notes, outputPath)
		case "md":
			return exportMarkdown(notes, outputPath)
		default:
			return fmt.Errorf("unknown format: %s", format)
		}
	},
}

func exportJSON(notes []*db.SearchResult, outputPath string) error {
	export := ExportData{
		ExportedAt: time.Now(),
		Version:    "1.0",
	}

	for _, n := range notes {
		tags, _ := db.GetNoteTags(dbConn, n.ID)
		attachments, _ := db.ListNoteAttachments(dbConn, n.ID)

		en := ExportNote{
			ID:        n.ID.String(),
			Title:     n.Title,
			Content:   n.Content,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
		}

		for _, t := range tags {
			en.Tags = append(en.Tags, t.Name)
		}

		for _, a := range attachments {
			att, _ := db.GetAttachment(dbConn, a.ID)
			if att != nil {
				en.Attachments = append(en.Attachments, ExportAttachment{
					ID:       att.ID.String(),
					Filename: att.Filename,
					MimeType: att.MimeType,
					Data:     base64.StdEncoding.EncodeToString(att.Data),
				})
			}
		}

		export.Notes = append(export.Notes, en)
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}

	if outputPath == "" || outputPath == "-" {
		fmt.Println(string(data))
		return nil
	}

	return os.WriteFile(outputPath, data, 0644)
}

func exportMarkdown(notes []*db.SearchResult, outputDir string) error {
	if outputDir == "" {
		outputDir = "export"
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	for _, n := range notes {
		tags, _ := db.GetNoteTags(dbConn, n.ID)
		attachments, _ := db.ListNoteAttachments(dbConn, n.ID)

		en := ExportNote{
			ID:        n.ID.String(),
			Title:     n.Title,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
		}
		for _, t := range tags {
			en.Tags = append(en.Tags, t.Name)
		}

		// Write markdown file with frontmatter
		var sb strings.Builder
		sb.WriteString("---\n")

		frontmatter, _ := yaml.Marshal(en)
		sb.Write(frontmatter)
		sb.WriteString("---\n\n")
		sb.WriteString(n.Content)

		filename := sanitizeFilename(n.Title) + ".md"
		filePath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
			return err
		}

		// Export attachments
		if len(attachments) > 0 {
			attDir := filepath.Join(outputDir, "attachments", n.ID.String()[:8])
			os.MkdirAll(attDir, 0755)

			for _, a := range attachments {
				att, _ := db.GetAttachment(dbConn, a.ID)
				if att != nil {
					attPath := filepath.Join(attDir, att.Filename)
					os.WriteFile(attPath, att.Data, 0644)
				}
			}
		}
	}

	fmt.Println(ui.Success(fmt.Sprintf("Exported %d notes to %s", len(notes), outputDir)))
	return nil
}

func sanitizeFilename(name string) string {
	// Replace unsafe characters
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "-", "\"", "-", "<", "-", ">", "-", "|", "-",
	)
	name = replacer.Replace(name)
	if len(name) > 100 {
		name = name[:100]
	}
	return name
}

func init() {
	exportCmd.Flags().StringP("format", "f", "json", "export format (json|md)")
	exportCmd.Flags().StringP("output", "o", "", "output path")
	exportCmd.Flags().StringP("note", "n", "", "single note ID to export")
	rootCmd.AddCommand(exportCmd)
}
```

Note: Add `"gopkg.in/yaml.v3"` to go.mod.

**Step 2: Commit**

```bash
go get gopkg.in/yaml.v3
git add -A
git commit -m "feat: add 'export' command for JSON and markdown export"
```

---

## Task 20: Import Command

**Files:**
- Create: `cmd/memo/import.go`

**Step 1: Write the import command**

```go
// ABOUTME: Import command for restoring notes from backup.
// ABOUTME: Supports JSON and markdown directory import.

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/models"
	"github.com/harper/memo/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var importCmd = &cobra.Command{
	Use:   "import <path>",
	Short: "Import notes",
	Long:  `Import notes from a JSON file or directory of markdown files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat path: %w", err)
		}

		if info.IsDir() {
			return importMarkdownDir(path)
		}

		if strings.HasSuffix(path, ".json") {
			return importJSON(path)
		}

		return importMarkdownFile(path)
	},
}

func importJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var export ExportData
	if err := json.Unmarshal(data, &export); err != nil {
		return err
	}

	count := 0
	for _, en := range export.Notes {
		note := models.NewNote(en.Title, en.Content)
		// Try to preserve original ID if valid
		if id, err := uuid.Parse(en.ID); err == nil {
			note.ID = id
		}
		note.CreatedAt = en.CreatedAt
		note.UpdatedAt = en.UpdatedAt

		if err := db.CreateNote(dbConn, note); err != nil {
			fmt.Printf("Warning: failed to import %q: %v\n", en.Title, err)
			continue
		}

		for _, tagName := range en.Tags {
			db.AddTagToNote(dbConn, note.ID, tagName)
		}

		for _, att := range en.Attachments {
			data, _ := base64.StdEncoding.DecodeString(att.Data)
			attachment := models.NewAttachment(note.ID, att.Filename, att.MimeType, data)
			if id, err := uuid.Parse(att.ID); err == nil {
				attachment.ID = id
			}
			db.CreateAttachment(dbConn, attachment)
		}

		count++
	}

	fmt.Println(ui.Success(fmt.Sprintf("Imported %d notes", count)))
	return nil
}

func importMarkdownDir(dir string) error {
	count := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		if err := importMarkdownFile(path); err != nil {
			fmt.Printf("Warning: failed to import %s: %v\n", path, err)
			return nil
		}
		count++
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println(ui.Success(fmt.Sprintf("Imported %d notes", count)))
	return nil
}

func importMarkdownFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	var title string
	var tags []string

	// Try to parse frontmatter
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content, "---\n", 3)
		if len(parts) >= 3 {
			var frontmatter struct {
				Title string   `yaml:"title"`
				Tags  []string `yaml:"tags"`
			}
			if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err == nil {
				title = frontmatter.Title
				tags = frontmatter.Tags
				content = parts[2]
			}
		}
	}

	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	note := models.NewNote(title, strings.TrimSpace(content))
	if err := db.CreateNote(dbConn, note); err != nil {
		return err
	}

	for _, tag := range tags {
		db.AddTagToNote(dbConn, note.ID, tag)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(importCmd)
}
```

**Step 2: Commit**

```bash
git add -A
git commit -m "feat: add 'import' command for JSON and markdown import"
```

---

## Task 21: MCP Server

**Files:**
- Create: `internal/mcp/server.go`
- Create: `internal/mcp/tools.go`
- Create: `internal/mcp/resources.go`
- Create: `internal/mcp/prompts.go`
- Create: `cmd/memo/mcp.go`

**Step 1: Create MCP server core**

```go
// ABOUTME: MCP server for memo integration with AI agents.
// ABOUTME: Provides tools, resources, and prompts for note management.

package mcp

import (
	"database/sql"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/server"
)

type Server struct {
	mcp *server.MCPServer
	db  *sql.DB
}

func NewServer(db *sql.DB) *Server {
	s := &Server{db: db}

	s.mcp = server.NewMCPServer(
		"memo",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
		server.WithPromptCapabilities(true),
	)

	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}

func (s *Server) Serve() error {
	return server.ServeStdio(s.mcp)
}
```

**Step 2: Create MCP tools**

```go
// ABOUTME: MCP tools for note CRUD operations.
// ABOUTME: Maps CLI functionality to MCP tool interface.

package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerTools() {
	// add_note
	s.mcp.AddTool(mcp.Tool{
		Name:        "add_note",
		Description: "Create a new note with title and content",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"title":   map[string]string{"type": "string", "description": "Note title"},
				"content": map[string]string{"type": "string", "description": "Note content (markdown)"},
				"tags":    map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Optional tags"},
			},
			Required: []string{"title", "content"},
		},
	}, s.handleAddNote)

	// list_notes
	s.mcp.AddTool(mcp.Tool{
		Name:        "list_notes",
		Description: "List notes with optional filtering",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"tag":   map[string]string{"type": "string", "description": "Filter by tag"},
				"limit": map[string]interface{}{"type": "integer", "description": "Max results", "default": 20},
			},
		},
	}, s.handleListNotes)

	// get_note
	s.mcp.AddTool(mcp.Tool{
		Name:        "get_note",
		Description: "Get a note by ID prefix",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]string{"type": "string", "description": "Note ID or prefix (6+ chars)"},
			},
			Required: []string{"id"},
		},
	}, s.handleGetNote)

	// update_note
	s.mcp.AddTool(mcp.Tool{
		Name:        "update_note",
		Description: "Update a note's title or content",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id":      map[string]string{"type": "string", "description": "Note ID or prefix"},
				"title":   map[string]string{"type": "string", "description": "New title"},
				"content": map[string]string{"type": "string", "description": "New content"},
			},
			Required: []string{"id"},
		},
	}, s.handleUpdateNote)

	// delete_note
	s.mcp.AddTool(mcp.Tool{
		Name:        "delete_note",
		Description: "Delete a note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]string{"type": "string", "description": "Note ID or prefix"},
			},
			Required: []string{"id"},
		},
	}, s.handleDeleteNote)

	// search_notes
	s.mcp.AddTool(mcp.Tool{
		Name:        "search_notes",
		Description: "Full-text search notes",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]string{"type": "string", "description": "Search query"},
				"limit": map[string]interface{}{"type": "integer", "description": "Max results", "default": 10},
			},
			Required: []string{"query"},
		},
	}, s.handleSearchNotes)

	// add_tag
	s.mcp.AddTool(mcp.Tool{
		Name:        "add_tag",
		Description: "Add a tag to a note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id":  map[string]string{"type": "string", "description": "Note ID or prefix"},
				"tag": map[string]string{"type": "string", "description": "Tag name"},
			},
			Required: []string{"id", "tag"},
		},
	}, s.handleAddTag)

	// remove_tag
	s.mcp.AddTool(mcp.Tool{
		Name:        "remove_tag",
		Description: "Remove a tag from a note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id":  map[string]string{"type": "string", "description": "Note ID or prefix"},
				"tag": map[string]string{"type": "string", "description": "Tag name"},
			},
			Required: []string{"id", "tag"},
		},
	}, s.handleRemoveTag)

	// add_attachment
	s.mcp.AddTool(mcp.Tool{
		Name:        "add_attachment",
		Description: "Add an attachment to a note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id":        map[string]string{"type": "string", "description": "Note ID or prefix"},
				"filename":  map[string]string{"type": "string", "description": "Filename"},
				"mime_type": map[string]string{"type": "string", "description": "MIME type"},
				"data":      map[string]string{"type": "string", "description": "Base64 encoded data"},
			},
			Required: []string{"id", "filename", "mime_type", "data"},
		},
	}, s.handleAddAttachment)

	// list_attachments
	s.mcp.AddTool(mcp.Tool{
		Name:        "list_attachments",
		Description: "List attachments for a note",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]string{"type": "string", "description": "Note ID or prefix"},
			},
			Required: []string{"id"},
		},
	}, s.handleListAttachments)

	// get_attachment
	s.mcp.AddTool(mcp.Tool{
		Name:        "get_attachment",
		Description: "Get an attachment's content",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id": map[string]string{"type": "string", "description": "Attachment ID or prefix"},
			},
			Required: []string{"id"},
		},
	}, s.handleGetAttachment)

	// export_note
	s.mcp.AddTool(mcp.Tool{
		Name:        "export_note",
		Description: "Export a note as JSON or markdown",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"id":     map[string]string{"type": "string", "description": "Note ID or prefix"},
				"format": map[string]string{"type": "string", "description": "Format: json or md", "default": "json"},
			},
			Required: []string{"id"},
		},
	}, s.handleExportNote)
}

// Tool handlers
func (s *Server) handleAddNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	note := models.NewNote(params.Title, params.Content)
	if err := db.CreateNote(s.db, note); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create note: %v", err)), nil
	}

	for _, tag := range params.Tags {
		db.AddTagToNote(s.db, note.ID, tag)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Created note %s", note.ID.String())), nil
}

// ... Additional handlers follow same pattern
```

**Step 3: Create MCP command**

```go
// ABOUTME: MCP command to start the MCP server.
// ABOUTME: Runs on stdio for integration with AI agents.

package main

import (
	"github.com/harper/memo/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server",
	Long:  `Start the Model Context Protocol server for AI agent integration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer(dbConn)
		return server.Serve()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
```

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: add MCP server with tools, resources, and prompts"
```

---

## Task 22: Integration Tests

**Files:**
- Create: `test/integration_test.go`

**Step 1: Write integration tests**

```go
// ABOUTME: Integration tests for memo CLI commands.
// ABOUTME: Tests full workflow from add to delete.

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var memoBin string

func TestMain(m *testing.M) {
	// Build memo binary
	cmd := exec.Command("go", "build", "-o", "bin/memo", "./cmd/memo")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	wd, _ := os.Getwd()
	memoBin = filepath.Join(wd, "..", "bin", "memo")

	os.Exit(m.Run())
}

func TestAddListShowDelete(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Add a note
	out, err := runMemo(dbPath, "add", "Test Note", "--content", "Test content here")
	if err != nil {
		t.Fatalf("add failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created note") {
		t.Errorf("expected 'Created note' in output: %s", out)
	}

	// List notes
	out, err = runMemo(dbPath, "list")
	if err != nil {
		t.Fatalf("list failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Test Note") {
		t.Errorf("expected 'Test Note' in list: %s", out)
	}

	// Extract ID prefix from list output
	lines := strings.Split(out, "\n")
	var idPrefix string
	for _, line := range lines {
		if strings.Contains(line, "Test Note") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				idPrefix = fields[0]
				break
			}
		}
	}

	if idPrefix == "" {
		t.Fatal("could not extract ID prefix")
	}

	// Show note
	out, err = runMemo(dbPath, "show", idPrefix)
	if err != nil {
		t.Fatalf("show failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Test content") {
		t.Errorf("expected 'Test content' in show: %s", out)
	}

	// Delete note
	out, err = runMemo(dbPath, "rm", idPrefix, "--force")
	if err != nil {
		t.Fatalf("rm failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Deleted") {
		t.Errorf("expected 'Deleted' in output: %s", out)
	}
}

func TestTagOperations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Add note with tags
	runMemo(dbPath, "add", "Tagged Note", "--content", "Content", "--tags", "work,urgent")

	// List by tag
	out, _ := runMemo(dbPath, "list", "--tag", "work")
	if !strings.Contains(out, "Tagged Note") {
		t.Errorf("expected note in tag filter: %s", out)
	}

	// Tag list
	out, _ = runMemo(dbPath, "tag", "list")
	if !strings.Contains(out, "work") {
		t.Errorf("expected 'work' tag in list: %s", out)
	}
}

func TestSearch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	runMemo(dbPath, "add", "Go Programming", "--content", "Learn about goroutines")
	runMemo(dbPath, "add", "Cooking", "--content", "How to make pasta")

	out, _ := runMemo(dbPath, "list", "--search", "goroutines")
	if !strings.Contains(out, "Go Programming") {
		t.Errorf("expected 'Go Programming' in search: %s", out)
	}
	if strings.Contains(out, "Cooking") {
		t.Errorf("did not expect 'Cooking' in search: %s", out)
	}
}

func runMemo(dbPath string, args ...string) (string, error) {
	allArgs := append([]string{"--db", dbPath}, args...)
	cmd := exec.Command(memoBin, allArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
```

**Step 2: Run tests**

```bash
go test ./test/... -v
```

Expected: PASS

**Step 3: Commit**

```bash
git add -A
git commit -m "test: add integration tests for CLI commands"
```

---

## Task 23: Final Verification

**Step 1: Run all tests**

```bash
make test
```

Expected: All tests pass

**Step 2: Build and manual verification**

```bash
make build
./bin/memo --help
./bin/memo add "First Note" --content "Hello world" --tags "test"
./bin/memo list
./bin/memo show <id>
./bin/memo tag list
./bin/memo export --format json
```

**Step 3: Final commit**

```bash
git add -A
git commit -m "chore: complete memo CLI implementation"
```
