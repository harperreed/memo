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
