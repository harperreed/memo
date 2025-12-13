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
