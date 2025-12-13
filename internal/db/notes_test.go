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
