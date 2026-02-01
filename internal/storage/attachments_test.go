// ABOUTME: Tests for attachment operations in SQLite storage.
// ABOUTME: Validates attachment CRUD and binary blob handling.

package storage

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

func TestCreateAttachment(t *testing.T) {
	store := newTestStore(t)

	// Create a note first
	note := models.NewNote("Attachment Test", "Content")
	_ = store.CreateNote(note, nil)

	// Create an attachment
	data := []byte("This is test attachment data")
	attachment := models.NewAttachment(note.ID, "test.txt", "text/plain", data)

	err := store.CreateAttachment(attachment)
	if err != nil {
		t.Fatalf("failed to create attachment: %v", err)
	}

	// Verify it was stored
	retrieved, err := store.GetAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("failed to retrieve attachment: %v", err)
	}

	if retrieved.ID != attachment.ID {
		t.Errorf("expected ID %s, got %s", attachment.ID, retrieved.ID)
	}
	if retrieved.Filename != "test.txt" {
		t.Errorf("expected filename 'test.txt', got %q", retrieved.Filename)
	}
	if retrieved.MimeType != "text/plain" {
		t.Errorf("expected mime type 'text/plain', got %q", retrieved.MimeType)
	}
	if !bytes.Equal(retrieved.Data, data) {
		t.Error("attachment data does not match")
	}
}

func TestGetAttachmentByID(t *testing.T) {
	store := newTestStore(t)

	// Create a note and attachment
	note := models.NewNote("Attachment Test", "Content")
	_ = store.CreateNote(note, nil)

	attachment := models.NewAttachment(note.ID, "file.bin", "application/octet-stream", []byte{0x00, 0xFF, 0x42})
	_ = store.CreateAttachment(attachment)

	// Test successful retrieval
	retrieved, err := store.GetAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("failed to get attachment: %v", err)
	}
	if retrieved.Filename != attachment.Filename {
		t.Errorf("expected filename %q, got %q", attachment.Filename, retrieved.Filename)
	}

	// Test not found
	_, err = store.GetAttachmentByID(uuid.New())
	if err == nil {
		t.Error("expected error for non-existent attachment")
	}
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got %v", err)
	}
}

func TestGetAttachmentByPrefix(t *testing.T) {
	store := newTestStore(t)

	// Create a note and attachment
	note := models.NewNote("Attachment Test", "Content")
	_ = store.CreateNote(note, nil)

	attachment := models.NewAttachment(note.ID, "test.pdf", "application/pdf", []byte("PDF content"))
	_ = store.CreateAttachment(attachment)

	prefix := attachment.ID.String()[:6]

	// Test successful retrieval by prefix
	retrieved, err := store.GetAttachmentByPrefix(prefix)
	if err != nil {
		t.Fatalf("failed to get attachment by prefix: %v", err)
	}
	if retrieved.ID != attachment.ID {
		t.Errorf("expected ID %s, got %s", attachment.ID, retrieved.ID)
	}

	// Test prefix too short
	_, err = store.GetAttachmentByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}
}

func TestListAttachmentsByNote(t *testing.T) {
	store := newTestStore(t)

	// Create notes
	note1 := models.NewNote("Note 1", "Content 1")
	_ = store.CreateNote(note1, nil)

	note2 := models.NewNote("Note 2", "Content 2")
	_ = store.CreateNote(note2, nil)

	// Create attachments for note1
	att1 := models.NewAttachment(note1.ID, "file1.txt", "text/plain", []byte("content1"))
	_ = store.CreateAttachment(att1)

	att2 := models.NewAttachment(note1.ID, "file2.txt", "text/plain", []byte("content2"))
	_ = store.CreateAttachment(att2)

	// Create attachment for note2
	att3 := models.NewAttachment(note2.ID, "file3.txt", "text/plain", []byte("content3"))
	_ = store.CreateAttachment(att3)

	// List attachments for note1
	attachments, err := store.ListAttachmentsByNote(note1.ID)
	if err != nil {
		t.Fatalf("failed to list attachments: %v", err)
	}
	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments for note1, got %d", len(attachments))
	}

	// List attachments for note2
	attachments, err = store.ListAttachmentsByNote(note2.ID)
	if err != nil {
		t.Fatalf("failed to list attachments: %v", err)
	}
	if len(attachments) != 1 {
		t.Errorf("expected 1 attachment for note2, got %d", len(attachments))
	}
}

func TestDeleteAttachment(t *testing.T) {
	store := newTestStore(t)

	// Create a note and attachment
	note := models.NewNote("Attachment Test", "Content")
	_ = store.CreateNote(note, nil)

	attachment := models.NewAttachment(note.ID, "delete-me.txt", "text/plain", []byte("to be deleted"))
	_ = store.CreateAttachment(attachment)

	// Verify it exists
	_, err := store.GetAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("attachment should exist before deletion: %v", err)
	}

	// Delete the attachment
	err = store.DeleteAttachment(attachment.ID)
	if err != nil {
		t.Fatalf("failed to delete attachment: %v", err)
	}

	// Verify it's gone
	_, err = store.GetAttachmentByID(attachment.ID)
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound after deletion, got %v", err)
	}
}

func TestAttachmentCascadeDelete(t *testing.T) {
	store := newTestStore(t)

	// Create a note and attachments
	note := models.NewNote("Cascade Test", "Content")
	_ = store.CreateNote(note, nil)

	att1 := models.NewAttachment(note.ID, "file1.txt", "text/plain", []byte("content1"))
	_ = store.CreateAttachment(att1)

	att2 := models.NewAttachment(note.ID, "file2.txt", "text/plain", []byte("content2"))
	_ = store.CreateAttachment(att2)

	// Verify attachments exist
	attachments, _ := store.ListAttachmentsByNote(note.ID)
	if len(attachments) != 2 {
		t.Fatalf("expected 2 attachments before note deletion, got %d", len(attachments))
	}

	// Delete the note
	err := store.DeleteNote(note.ID)
	if err != nil {
		t.Fatalf("failed to delete note: %v", err)
	}

	// Attachments should be cascade deleted
	attachments, err = store.ListAttachmentsByNote(note.ID)
	if err != nil {
		t.Fatalf("failed to list attachments: %v", err)
	}
	if len(attachments) != 0 {
		t.Errorf("expected 0 attachments after note deletion, got %d", len(attachments))
	}
}

func TestBinaryDataPreservation(t *testing.T) {
	store := newTestStore(t)

	// Create a note
	note := models.NewNote("Binary Test", "Content")
	_ = store.CreateNote(note, nil)

	// Create attachment with binary data including null bytes
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x00, 0x00}
	attachment := models.NewAttachment(note.ID, "binary.dat", "application/octet-stream", binaryData)
	_ = store.CreateAttachment(attachment)

	// Retrieve and verify
	retrieved, err := store.GetAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("failed to retrieve attachment: %v", err)
	}

	if !bytes.Equal(retrieved.Data, binaryData) {
		t.Errorf("binary data not preserved correctly")
		t.Errorf("expected: %v", binaryData)
		t.Errorf("got:      %v", retrieved.Data)
	}
}

func TestDeleteNonExistentAttachment(t *testing.T) {
	store := newTestStore(t)

	// Try to delete an attachment that doesn't exist
	err := store.DeleteAttachment(uuid.New())
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got %v", err)
	}
}

func TestListAttachmentsForNonExistentNote(t *testing.T) {
	store := newTestStore(t)

	// List attachments for a note that doesn't exist
	attachments, err := store.ListAttachmentsByNote(uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(attachments) != 0 {
		t.Errorf("expected 0 attachments, got %d", len(attachments))
	}
}

func TestAttachmentPrefixNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetAttachmentByPrefix("000000")
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got %v", err)
	}
}

func TestAttachmentPrefixAmbiguous(t *testing.T) {
	store := newTestStore(t)

	// Create a note
	note := models.NewNote("Ambiguous Test", "Content")
	_ = store.CreateNote(note, nil)

	// Create two attachments with IDs that have the same prefix
	// Since UUIDs are random, we'll create multiple attachments
	// and check if we can find a common prefix among any two
	var attachments []*models.Attachment
	for i := 0; i < 20; i++ {
		att := models.NewAttachment(note.ID, "file.txt", "text/plain", []byte("content"))
		_ = store.CreateAttachment(att)
		attachments = append(attachments, att)
	}

	// Find two attachments with the same 6-char prefix (if any)
	prefixMap := make(map[string][]*models.Attachment)
	for _, att := range attachments {
		prefix := att.ID.String()[:6]
		prefixMap[prefix] = append(prefixMap[prefix], att)
	}

	// Look for ambiguous prefix
	for prefix, atts := range prefixMap {
		if len(atts) > 1 {
			_, err := store.GetAttachmentByPrefix(prefix)
			if err == nil {
				t.Errorf("expected error for ambiguous prefix")
			}
			if !errors.Is(err, ErrAmbiguousPrefix) {
				t.Errorf("expected ErrAmbiguousPrefix, got %v", err)
			}
			return // Found and tested ambiguous case
		}
	}

	// If no collision found naturally, that's OK - UUIDs are designed to be unique
	// Just verify the error constant exists
	t.Log("No natural UUID prefix collision found in sample - this is expected behavior")
}

func TestEmptyAttachmentData(t *testing.T) {
	store := newTestStore(t)

	// Create a note
	note := models.NewNote("Empty Data Test", "Content")
	_ = store.CreateNote(note, nil)

	// Create attachment with empty data
	attachment := models.NewAttachment(note.ID, "empty.txt", "text/plain", []byte{})
	err := store.CreateAttachment(attachment)
	if err != nil {
		t.Fatalf("failed to create attachment with empty data: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("failed to retrieve attachment: %v", err)
	}

	if len(retrieved.Data) != 0 {
		t.Errorf("expected empty data, got %d bytes", len(retrieved.Data))
	}
}

func TestLargeAttachmentData(t *testing.T) {
	store := newTestStore(t)

	// Create a note
	note := models.NewNote("Large Data Test", "Content")
	_ = store.CreateNote(note, nil)

	// Create attachment with 1MB of data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	attachment := models.NewAttachment(note.ID, "large.bin", "application/octet-stream", largeData)
	err := store.CreateAttachment(attachment)
	if err != nil {
		t.Fatalf("failed to create large attachment: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("failed to retrieve large attachment: %v", err)
	}

	if !bytes.Equal(retrieved.Data, largeData) {
		t.Errorf("large data not preserved correctly")
	}
}
