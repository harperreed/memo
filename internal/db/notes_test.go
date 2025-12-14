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

func TestListNotesByDirTag(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Create notes - one with dir tag, one without
	dirNote := models.NewNote("Dir Note", "Content in directory")
	globalNote := models.NewNote("Global Note", "Content everywhere")
	if err := CreateNote(db, dirNote); err != nil {
		t.Fatalf("failed to create dir note: %v", err)
	}
	if err := CreateNote(db, globalNote); err != nil {
		t.Fatalf("failed to create global note: %v", err)
	}

	// Tag one note with a directory
	testDir := "/Users/harper/projects/memo"
	if err := AddTagToNote(db, dirNote.ID, "dir:"+testDir); err != nil {
		t.Fatalf("failed to add dir tag: %v", err)
	}

	// Verify the tag was added
	tags, err := GetNoteTags(db, dirNote.ID)
	if err != nil {
		t.Fatalf("failed to get note tags: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}

	// Query notes for that directory
	notes, err := ListNotesByDirTag(db, testDir, 10)
	if err != nil {
		t.Fatalf("failed to list notes by dir tag: %v", err)
	}

	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].ID != dirNote.ID {
		t.Errorf("expected dir note ID, got different note")
	}

	// Query notes for a different directory should return empty
	otherNotes, err := ListNotesByDirTag(db, "/other/path", 10)
	if err != nil {
		t.Fatalf("failed to list notes for other dir: %v", err)
	}
	if len(otherNotes) != 0 {
		t.Errorf("expected 0 notes for other dir, got %d", len(otherNotes))
	}
}

func TestListGlobalNotes(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Create notes - one with dir tag, two without
	dirNote := models.NewNote("Dir Note", "Content in directory")
	globalNote1 := models.NewNote("Global 1", "Content everywhere")
	globalNote2 := models.NewNote("Global 2", "More global content")
	CreateNote(db, dirNote)
	CreateNote(db, globalNote1)
	CreateNote(db, globalNote2)

	// Tag one note with a directory
	if err := AddTagToNote(db, dirNote.ID, "dir:/some/path"); err != nil {
		t.Fatalf("failed to add dir tag: %v", err)
	}

	// Add a regular tag to a global note (should still be global)
	if err := AddTagToNote(db, globalNote1.ID, "work"); err != nil {
		t.Fatalf("failed to add regular tag: %v", err)
	}

	// Query global notes
	notes, err := ListGlobalNotes(db, 10)
	if err != nil {
		t.Fatalf("failed to list global notes: %v", err)
	}

	if len(notes) != 2 {
		t.Errorf("expected 2 global notes, got %d", len(notes))
	}

	// Verify dir note is not in results
	for _, n := range notes {
		if n.ID == dirNote.ID {
			t.Error("dir-tagged note should not appear in global notes")
		}
	}
}

func TestCountGlobalNotes(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Start with zero notes
	count, err := CountGlobalNotes(db)
	if err != nil {
		t.Fatalf("failed to count global notes: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 global notes, got %d", count)
	}

	// Create 3 notes, tag 1 with directory
	for i := 0; i < 3; i++ {
		note := models.NewNote("Note", "Content")
		CreateNote(db, note)
		if i == 0 {
			AddTagToNote(db, note.ID, "dir:/some/dir")
		}
	}

	// Should count 2 global notes
	count, err = CountGlobalNotes(db)
	if err != nil {
		t.Fatalf("failed to count global notes: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 global notes, got %d", count)
	}
}
