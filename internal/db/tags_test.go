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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

	note := models.NewNote("Test", "Content")
	_ = CreateNote(db, note)

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
	defer func() { _ = db.Close() }()

	note1 := models.NewNote("Note 1", "Content")
	note2 := models.NewNote("Note 2", "Content")
	_ = CreateNote(db, note1)
	_ = CreateNote(db, note2)

	_ = AddTagToNote(db, note1.ID, "shared")
	_ = AddTagToNote(db, note2.ID, "shared")
	_ = AddTagToNote(db, note1.ID, "unique")

	tags, err := ListAllTags(db)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}
