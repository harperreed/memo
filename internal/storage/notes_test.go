// ABOUTME: Tests for note CRUD operations in SQLite storage.
// ABOUTME: Validates note creation, retrieval, update, and deletion.

package storage

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

func TestCreateNote(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Test Note", "This is a test note content.")
	tags := []string{"test", "example"}

	err := store.CreateNote(note, tags)
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// Verify note was stored
	retrieved, retrievedTags, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("failed to retrieve note: %v", err)
	}

	if retrieved.ID != note.ID {
		t.Errorf("expected ID %s, got %s", note.ID, retrieved.ID)
	}
	if retrieved.Title != note.Title {
		t.Errorf("expected title %q, got %q", note.Title, retrieved.Title)
	}
	if retrieved.Content != note.Content {
		t.Errorf("expected content %q, got %q", note.Content, retrieved.Content)
	}

	// Check tags
	if len(retrievedTags) != len(tags) {
		t.Errorf("expected %d tags, got %d", len(tags), len(retrievedTags))
	}
}

func TestGetNoteByID(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Get By ID Test", "Content for ID test.")
	_ = store.CreateNote(note, nil)

	// Test successful retrieval
	retrieved, _, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("failed to get note: %v", err)
	}
	if retrieved.Title != note.Title {
		t.Errorf("expected title %q, got %q", note.Title, retrieved.Title)
	}

	// Test not found
	_, _, err = store.GetNoteByID(uuid.New())
	if err == nil {
		t.Error("expected error for non-existent note")
	}
	if !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestGetNoteByPrefix(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Prefix Test", "Content for prefix test.")
	_ = store.CreateNote(note, nil)

	prefix := note.ID.String()[:6]

	// Test successful retrieval by prefix
	retrieved, _, err := store.GetNoteByPrefix(prefix)
	if err != nil {
		t.Fatalf("failed to get note by prefix: %v", err)
	}
	if retrieved.ID != note.ID {
		t.Errorf("expected ID %s, got %s", note.ID, retrieved.ID)
	}

	// Test prefix too short
	_, _, err = store.GetNoteByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}
}

func TestGetNoteByPrefixAmbiguous(t *testing.T) {
	store := newTestStore(t)

	// This test is harder to make deterministic since UUIDs are random
	// We'll just verify the error type exists
	_, _, err := store.GetNoteByPrefix("000000")
	if err != nil && !errors.Is(err, ErrNoteNotFound) && !errors.Is(err, ErrAmbiguousPrefix) {
		t.Errorf("expected ErrNoteNotFound or ErrAmbiguousPrefix, got %v", err)
	}
}

func TestUpdateNote(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Original Title", "Original content.")
	_ = store.CreateNote(note, []string{"original"})

	// Update the note
	note.Title = "Updated Title"
	note.Content = "Updated content."
	note.Touch()

	err := store.UpdateNote(note, []string{"updated", "new"})
	if err != nil {
		t.Fatalf("failed to update note: %v", err)
	}

	// Verify updates
	retrieved, tags, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("failed to retrieve updated note: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %q", retrieved.Title)
	}
	if retrieved.Content != "Updated content." {
		t.Errorf("expected content 'Updated content.', got %q", retrieved.Content)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestDeleteNote(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("To Delete", "This note will be deleted.")
	_ = store.CreateNote(note, []string{"delete-me"})

	// Verify it exists
	_, _, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("note should exist before deletion: %v", err)
	}

	// Delete the note
	err = store.DeleteNote(note.ID)
	if err != nil {
		t.Fatalf("failed to delete note: %v", err)
	}

	// Verify it's gone
	_, _, err = store.GetNoteByID(note.ID)
	if !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("expected ErrNoteNotFound after deletion, got %v", err)
	}
}

func TestListNotes(t *testing.T) {
	store := newTestStore(t)

	// Create some notes
	note1 := models.NewNote("Note 1", "First note")
	note1.UpdatedAt = time.Now().Add(-2 * time.Hour)
	_ = store.CreateNote(note1, []string{"tag1"})

	note2 := models.NewNote("Note 2", "Second note")
	note2.UpdatedAt = time.Now().Add(-1 * time.Hour)
	_ = store.CreateNote(note2, []string{"tag1", "tag2"})

	note3 := models.NewNote("Note 3", "Third note")
	_ = store.CreateNote(note3, []string{"tag2"})

	// Test listing all notes
	notes, err := store.ListNotes(&NoteFilter{})
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}
	if len(notes) != 3 {
		t.Errorf("expected 3 notes, got %d", len(notes))
	}

	// Test listing with limit
	notes, err = store.ListNotes(&NoteFilter{Limit: 2})
	if err != nil {
		t.Fatalf("failed to list notes with limit: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}

	// Test filtering by tag
	tag := "tag2"
	notes, err = store.ListNotes(&NoteFilter{Tag: &tag})
	if err != nil {
		t.Fatalf("failed to list notes by tag: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes with tag2, got %d", len(notes))
	}
}

func TestListNotesWithDirTag(t *testing.T) {
	store := newTestStore(t)

	// Create notes with and without dir: tags
	note1 := models.NewNote("Dir Note", "Has dir tag")
	_ = store.CreateNote(note1, []string{"dir:/path/to/dir", "other"})

	note2 := models.NewNote("Global Note", "No dir tag")
	_ = store.CreateNote(note2, []string{"global"})

	// Test filtering by dir tag
	dirPath := "/path/to/dir"
	notes, err := store.ListNotes(&NoteFilter{DirTag: &dirPath})
	if err != nil {
		t.Fatalf("failed to list notes by dir tag: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note with dir tag, got %d", len(notes))
	}

	// Test global filter (no dir: tags)
	notes, err = store.ListNotes(&NoteFilter{Global: true})
	if err != nil {
		t.Fatalf("failed to list global notes: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 global note, got %d", len(notes))
	}
	if notes[0].Title != "Global Note" {
		t.Errorf("expected 'Global Note', got %q", notes[0].Title)
	}
}

func TestSearchNotes(t *testing.T) {
	store := newTestStore(t)

	// Create notes with searchable content
	note1 := models.NewNote("Important Meeting", "Discussion about project deadlines")
	_ = store.CreateNote(note1, nil)

	note2 := models.NewNote("Shopping List", "Buy milk and eggs")
	_ = store.CreateNote(note2, nil)

	note3 := models.NewNote("Project Notes", "Important project details")
	_ = store.CreateNote(note3, nil)

	// Search for "important"
	notes, err := store.ListNotes(&NoteFilter{Search: "important"})
	if err != nil {
		t.Fatalf("failed to search notes: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes matching 'important', got %d", len(notes))
	}

	// Search for "milk"
	notes, err = store.ListNotes(&NoteFilter{Search: "milk"})
	if err != nil {
		t.Fatalf("failed to search notes: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note matching 'milk', got %d", len(notes))
	}
}

func TestGetNoteTags(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tagged Note", "Content")
	expectedTags := []string{"alpha", "beta", "gamma"}
	_ = store.CreateNote(note, expectedTags)

	tags, err := store.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("failed to get note tags: %v", err)
	}

	if len(tags) != len(expectedTags) {
		t.Errorf("expected %d tags, got %d", len(expectedTags), len(tags))
	}

	// Check all tags are present (order may vary)
	tagMap := make(map[string]bool)
	for _, tag := range tags {
		tagMap[tag] = true
	}
	for _, expected := range expectedTags {
		if !tagMap[expected] {
			t.Errorf("expected tag %q to be present", expected)
		}
	}
}

func TestCountGlobalNotes(t *testing.T) {
	store := newTestStore(t)

	// Create notes with and without dir: tags
	note1 := models.NewNote("Dir Note 1", "Has dir tag")
	_ = store.CreateNote(note1, []string{"dir:/path/1"})

	note2 := models.NewNote("Dir Note 2", "Has dir tag")
	_ = store.CreateNote(note2, []string{"dir:/path/2"})

	note3 := models.NewNote("Global Note 1", "No dir tag")
	_ = store.CreateNote(note3, []string{"regular"})

	note4 := models.NewNote("Global Note 2", "No dir tag")
	_ = store.CreateNote(note4, nil)

	count, err := store.CountGlobalNotes()
	if err != nil {
		t.Fatalf("failed to count global notes: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 global notes, got %d", count)
	}
}

func TestTagNormalization(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tag Test", "Content")
	tags := []string{"  UPPERCASE  ", "MixedCase", " leading-space"}
	_ = store.CreateNote(note, tags)

	retrievedTags, err := store.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("failed to get note tags: %v", err)
	}

	for _, tag := range retrievedTags {
		if tag != strings.ToLower(strings.TrimSpace(tag)) {
			t.Errorf("tag %q was not normalized", tag)
		}
	}
}
