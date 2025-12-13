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
