// ABOUTME: Tests for tag operations in SQLite storage.
// ABOUTME: Validates tag listing, adding, and removing from notes.

package storage

import (
	"testing"

	"github.com/harper/memo/internal/models"
)

func TestListAllTags(t *testing.T) {
	store := newTestStore(t)

	// Create notes with various tags
	note1 := models.NewNote("Note 1", "Content 1")
	_ = store.CreateNote(note1, []string{"alpha", "beta"})

	note2 := models.NewNote("Note 2", "Content 2")
	_ = store.CreateNote(note2, []string{"beta", "gamma"})

	note3 := models.NewNote("Note 3", "Content 3")
	_ = store.CreateNote(note3, []string{"alpha", "gamma", "delta"})

	// List all tags with counts
	tags, err := store.ListAllTags()
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}

	if len(tags) != 4 {
		t.Errorf("expected 4 unique tags, got %d", len(tags))
	}

	// Build map for easier assertion
	tagCounts := make(map[string]int)
	for _, tc := range tags {
		tagCounts[tc.Tag.Name] = tc.Count
	}

	// Verify counts
	expectedCounts := map[string]int{
		"alpha": 2,
		"beta":  2,
		"gamma": 2,
		"delta": 1,
	}

	for name, expectedCount := range expectedCounts {
		if count, ok := tagCounts[name]; !ok {
			t.Errorf("expected tag %q to exist", name)
		} else if count != expectedCount {
			t.Errorf("expected tag %q to have count %d, got %d", name, expectedCount, count)
		}
	}
}

func TestAddTagToNote(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tag Test", "Content")
	_ = store.CreateNote(note, []string{"existing"})

	// Add a new tag
	err := store.AddTagToNote(note.ID, "new-tag")
	if err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}

	// Verify tag was added
	tags, err := store.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	// Check both tags exist
	found := make(map[string]bool)
	for _, tag := range tags {
		found[tag] = true
	}

	if !found["existing"] || !found["new-tag"] {
		t.Errorf("expected tags 'existing' and 'new-tag', got %v", tags)
	}
}

func TestAddTagToNoteIdempotent(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tag Test", "Content")
	_ = store.CreateNote(note, []string{"existing"})

	// Add same tag twice
	_ = store.AddTagToNote(note.ID, "new-tag")
	err := store.AddTagToNote(note.ID, "new-tag")
	if err != nil {
		t.Fatalf("adding same tag twice should not error: %v", err)
	}

	// Should still only have 2 tags
	tags, _ := store.GetNoteTags(note.ID)
	if len(tags) != 2 {
		t.Errorf("expected 2 tags after duplicate add, got %d", len(tags))
	}
}

func TestAddTagToNoteCaseInsensitive(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tag Test", "Content")
	_ = store.CreateNote(note, []string{"existing"})

	// Add tag with different case
	_ = store.AddTagToNote(note.ID, "UPPERCASE")
	_ = store.AddTagToNote(note.ID, "uppercase")

	// Should only have 2 tags (existing + one uppercase)
	tags, _ := store.GetNoteTags(note.ID)
	if len(tags) != 2 {
		t.Errorf("expected 2 tags (case should normalize), got %d: %v", len(tags), tags)
	}
}

func TestRemoveTagFromNote(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tag Test", "Content")
	_ = store.CreateNote(note, []string{"keep", "remove", "also-keep"})

	// Remove a tag
	err := store.RemoveTagFromNote(note.ID, "remove")
	if err != nil {
		t.Fatalf("failed to remove tag: %v", err)
	}

	// Verify tag was removed
	tags, err := store.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags after removal, got %d", len(tags))
	}

	// Check the correct tags remain
	found := make(map[string]bool)
	for _, tag := range tags {
		found[tag] = true
	}

	if !found["keep"] || !found["also-keep"] {
		t.Errorf("expected tags 'keep' and 'also-keep', got %v", tags)
	}
	if found["remove"] {
		t.Error("tag 'remove' should have been removed")
	}
}

func TestRemoveTagFromNoteNonExistent(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tag Test", "Content")
	_ = store.CreateNote(note, []string{"existing"})

	// Removing a non-existent tag should not error
	err := store.RemoveTagFromNote(note.ID, "non-existent")
	if err != nil {
		t.Errorf("removing non-existent tag should not error: %v", err)
	}

	// Tags should be unchanged
	tags, _ := store.GetNoteTags(note.ID)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag unchanged, got %d", len(tags))
	}
}

func TestAddEmptyTagToNote(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tag Test", "Content")
	_ = store.CreateNote(note, []string{"existing"})

	// Adding an empty tag should not error and not add anything
	err := store.AddTagToNote(note.ID, "")
	if err != nil {
		t.Errorf("adding empty tag should not error: %v", err)
	}

	// Adding a whitespace-only tag should not error and not add anything
	err = store.AddTagToNote(note.ID, "   ")
	if err != nil {
		t.Errorf("adding whitespace tag should not error: %v", err)
	}

	// Tags should be unchanged
	tags, _ := store.GetNoteTags(note.ID)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag unchanged, got %d", len(tags))
	}
}

func TestRemoveEmptyTagFromNote(t *testing.T) {
	store := newTestStore(t)

	note := models.NewNote("Tag Test", "Content")
	_ = store.CreateNote(note, []string{"existing"})

	// Removing an empty tag should not error
	err := store.RemoveTagFromNote(note.ID, "")
	if err != nil {
		t.Errorf("removing empty tag should not error: %v", err)
	}

	// Removing a whitespace-only tag should not error
	err = store.RemoveTagFromNote(note.ID, "   ")
	if err != nil {
		t.Errorf("removing whitespace tag should not error: %v", err)
	}

	// Tags should be unchanged
	tags, _ := store.GetNoteTags(note.ID)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag unchanged, got %d", len(tags))
	}
}

func TestListAllTagsEmpty(t *testing.T) {
	store := newTestStore(t)

	// No notes with tags
	tags, err := store.ListAllTags()
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}
