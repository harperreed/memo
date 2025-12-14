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
	defer func() { _ = db.Close() }()

	note := models.NewNote("Test", "Content")
	_ = CreateNote(db, note)

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
	defer func() { _ = db.Close() }()

	note := models.NewNote("Test", "Content")
	_ = CreateNote(db, note)

	att1 := models.NewAttachment(note.ID, "file1.txt", "text/plain", []byte("content1"))
	att2 := models.NewAttachment(note.ID, "file2.txt", "text/plain", []byte("content2"))
	_ = CreateAttachment(db, att1)
	_ = CreateAttachment(db, att2)

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
	defer func() { _ = db.Close() }()

	note := models.NewNote("Test", "Content")
	_ = CreateNote(db, note)

	att := models.NewAttachment(note.ID, "test.txt", "text/plain", []byte("content"))
	_ = CreateAttachment(db, att)

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
	defer func() { _ = db.Close() }()

	note := models.NewNote("Test", "Content")
	_ = CreateNote(db, note)

	att := models.NewAttachment(note.ID, "test.txt", "text/plain", []byte("content"))
	_ = CreateAttachment(db, att)

	// Delete note should cascade to attachments
	_ = DeleteNote(db, note.ID)

	_, err = GetAttachment(db, att.ID)
	if err == nil {
		t.Error("expected attachment to be deleted with note")
	}
}

func TestGetAttachmentByPrefix(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	note := models.NewNote("Test", "Content")
	_ = CreateNote(db, note)

	att := models.NewAttachment(note.ID, "test.txt", "text/plain", []byte("content"))
	_ = CreateAttachment(db, att)

	prefix := att.ID.String()[:8]
	got, err := GetAttachmentByPrefix(db, prefix)
	if err != nil {
		t.Fatalf("failed to get attachment by prefix: %v", err)
	}

	if got.ID != att.ID {
		t.Errorf("expected ID %v, got %v", att.ID, got.ID)
	}
}

func TestGetAttachmentByPrefixTooShort(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, err = GetAttachmentByPrefix(db, "abc")
	if err == nil {
		t.Error("expected error for short prefix")
	}
}
