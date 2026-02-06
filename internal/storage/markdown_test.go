// ABOUTME: Tests for MarkdownStore file-based storage backend.
// ABOUTME: Covers CRUD for notes, tags, attachments, search, prefix lookup, and edge cases.

package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/harper/memo/internal/models"
)

// newTestMarkdownStore creates a MarkdownStore in a temporary directory for testing.
func newTestMarkdownStore(t *testing.T) *MarkdownStore {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := NewMarkdownStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create test markdown store: %v", err)
	}
	return store
}

func TestNewMarkdownStore(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "memo-data")

	store, err := NewMarkdownStore(dataDir)
	if err != nil {
		t.Fatalf("NewMarkdownStore failed: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("NewMarkdownStore returned nil")
	}

	// Verify data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Fatal("Data directory was not created")
	}

	// Verify notes subdirectory exists
	notesDir := filepath.Join(dataDir, "notes")
	if _, err := os.Stat(notesDir); os.IsNotExist(err) {
		t.Fatal("Notes subdirectory was not created")
	}
}

func TestMarkdownNoteCRUD(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Create
	note := models.NewNote("Test Note", "This is the content.")
	tags := []string{"test", "example"}
	err := store.CreateNote(note, tags)
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	// Read by ID
	got, gotTags, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("GetNoteByID failed: %v", err)
	}
	if got.Title != note.Title {
		t.Errorf("expected title %q, got %q", note.Title, got.Title)
	}
	if got.Content != note.Content {
		t.Errorf("expected content %q, got %q", note.Content, got.Content)
	}
	if len(gotTags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(gotTags))
	}

	// Update
	note.Title = "Updated Title"
	note.Content = "Updated content."
	note.Touch()
	newTags := []string{"test", "updated"}
	err = store.UpdateNote(note, newTags)
	if err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}
	got, gotTags, _ = store.GetNoteByID(note.ID)
	if got.Title != "Updated Title" {
		t.Errorf("expected updated title, got %q", got.Title)
	}
	if got.Content != "Updated content." {
		t.Errorf("expected updated content, got %q", got.Content)
	}
	if !containsTag(gotTags, "updated") {
		t.Errorf("expected tag 'updated', got %v", gotTags)
	}

	// Delete
	err = store.DeleteNote(note.ID)
	if err != nil {
		t.Fatalf("DeleteNote failed: %v", err)
	}
	_, _, err = store.GetNoteByID(note.ID)
	if err == nil {
		t.Error("expected error getting deleted note")
	}
}

func TestMarkdownGetNoteByPrefix(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Prefix Test", "Content for prefix test.")
	_ = store.CreateNote(note, nil)

	// By prefix (first 8 chars)
	prefix := note.ID.String()[:8]
	got, _, err := store.GetNoteByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetNoteByPrefix failed: %v", err)
	}
	if got.ID != note.ID {
		t.Errorf("expected ID %s, got %s", note.ID, got.ID)
	}

	// Prefix too short
	_, _, err = store.GetNoteByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}

	// Not found
	_, _, err = store.GetNoteByPrefix("zzzzzzzzz")
	if !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("expected ErrNoteNotFound, got %v", err)
	}
}

func TestMarkdownListNotes(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Create several notes
	note1 := models.NewNote("First Note", "Content 1")
	time.Sleep(time.Millisecond)
	note2 := models.NewNote("Second Note", "Content 2")
	time.Sleep(time.Millisecond)
	note3 := models.NewNote("Third Note", "Content 3")

	_ = store.CreateNote(note1, []string{"alpha"})
	_ = store.CreateNote(note2, []string{"beta"})
	_ = store.CreateNote(note3, []string{"alpha", "gamma"})

	// List all
	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != 3 {
		t.Errorf("expected 3 notes, got %d", len(notes))
	}

	// Verify sorted by updated_at desc (newest first)
	if len(notes) >= 2 {
		for i := 0; i < len(notes)-1; i++ {
			if notes[i].UpdatedAt.Before(notes[i+1].UpdatedAt) {
				t.Errorf("notes not sorted by updated_at desc")
				break
			}
		}
	}

	// List with tag filter
	tag := "alpha"
	filtered, err := store.ListNotes(&NoteFilter{Tag: &tag})
	if err != nil {
		t.Fatalf("ListNotes with tag filter failed: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 notes with tag 'alpha', got %d", len(filtered))
	}

	// List with limit
	limited, err := store.ListNotes(&NoteFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListNotes with limit failed: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("expected 2 notes with limit, got %d", len(limited))
	}
}

func TestMarkdownListNotesSearch(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	_ = store.CreateNote(models.NewNote("Go Programming", "Learn Go language"), nil)
	_ = store.CreateNote(models.NewNote("Python Basics", "Learn Python"), nil)
	_ = store.CreateNote(models.NewNote("Random Thoughts", "Something about Go and Python"), nil)

	// Search in title
	results, err := store.ListNotes(&NoteFilter{Search: "Go"})
	if err != nil {
		t.Fatalf("ListNotes with search failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 notes matching 'Go', got %d", len(results))
	}

	// Search in content (case-insensitive)
	results, err = store.ListNotes(&NoteFilter{Search: "python"})
	if err != nil {
		t.Fatalf("ListNotes with search 'python' failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 notes matching 'python', got %d", len(results))
	}

	// Search with no match
	results, err = store.ListNotes(&NoteFilter{Search: "javascript"})
	if err != nil {
		t.Fatalf("ListNotes with no match failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 notes matching 'javascript', got %d", len(results))
	}
}

func TestMarkdownListNotesDirTagFilter(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	_ = store.CreateNote(models.NewNote("Project Note", "In project dir"), []string{"dir:myproject"})
	_ = store.CreateNote(models.NewNote("Global Note", "No dir tag"), []string{"general"})
	_ = store.CreateNote(models.NewNote("Another Project", "Also in project dir"), []string{"dir:myproject", "work"})

	// Filter by dir tag
	dirTag := "myproject"
	results, err := store.ListNotes(&NoteFilter{DirTag: &dirTag})
	if err != nil {
		t.Fatalf("ListNotes with DirTag failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 notes with dir:myproject, got %d", len(results))
	}

	// Global filter (no dir: tags)
	results, err = store.ListNotes(&NoteFilter{Global: true})
	if err != nil {
		t.Fatalf("ListNotes with Global failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 global note, got %d", len(results))
	}
}

func TestMarkdownGetNoteTags(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Tagged Note", "Content")
	_ = store.CreateNote(note, []string{"alpha", "beta", "gamma"})

	tags, err := store.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("GetNoteTags failed: %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}
}

func TestMarkdownCountGlobalNotes(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	_ = store.CreateNote(models.NewNote("Global1", "C"), nil)
	_ = store.CreateNote(models.NewNote("Global2", "C"), []string{"regular"})
	_ = store.CreateNote(models.NewNote("Scoped", "C"), []string{"dir:proj"})

	count, err := store.CountGlobalNotes()
	if err != nil {
		t.Fatalf("CountGlobalNotes failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 global notes, got %d", count)
	}
}

func TestMarkdownTagCRUD(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Tag Test", "Content")
	_ = store.CreateNote(note, []string{"initial"})

	// Add tag
	err := store.AddTagToNote(note.ID, "added")
	if err != nil {
		t.Fatalf("AddTagToNote failed: %v", err)
	}

	tags, _ := store.GetNoteTags(note.ID)
	if !containsTag(tags, "added") {
		t.Errorf("expected 'added' tag, got %v", tags)
	}
	if !containsTag(tags, "initial") {
		t.Errorf("expected 'initial' tag still present, got %v", tags)
	}

	// Add duplicate tag (should be no-op)
	err = store.AddTagToNote(note.ID, "added")
	if err != nil {
		t.Fatalf("AddTagToNote duplicate failed: %v", err)
	}
	tags, _ = store.GetNoteTags(note.ID)
	addedCount := 0
	for _, t := range tags {
		if t == "added" {
			addedCount++
		}
	}
	if addedCount != 1 {
		t.Errorf("expected exactly 1 'added' tag, got %d", addedCount)
	}

	// Remove tag
	err = store.RemoveTagFromNote(note.ID, "initial")
	if err != nil {
		t.Fatalf("RemoveTagFromNote failed: %v", err)
	}

	tags, _ = store.GetNoteTags(note.ID)
	if containsTag(tags, "initial") {
		t.Errorf("'initial' tag should have been removed, got %v", tags)
	}
}

func TestMarkdownListAllTags(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	_ = store.CreateNote(models.NewNote("Note1", "C"), []string{"alpha", "beta"})
	_ = store.CreateNote(models.NewNote("Note2", "C"), []string{"beta", "gamma"})
	_ = store.CreateNote(models.NewNote("Note3", "C"), []string{"alpha"})

	tags, err := store.ListAllTags()
	if err != nil {
		t.Fatalf("ListAllTags failed: %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("expected 3 unique tags, got %d", len(tags))
	}

	// Verify counts
	for _, tc := range tags {
		switch tc.Tag.Name {
		case "alpha":
			if tc.Count != 2 {
				t.Errorf("expected alpha count 2, got %d", tc.Count)
			}
		case "beta":
			if tc.Count != 2 {
				t.Errorf("expected beta count 2, got %d", tc.Count)
			}
		case "gamma":
			if tc.Count != 1 {
				t.Errorf("expected gamma count 1, got %d", tc.Count)
			}
		}
	}
}

func TestMarkdownAttachmentCRUD(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Attachment Test", "Content")
	_ = store.CreateNote(note, nil)

	// Create attachment
	att := models.NewAttachment(note.ID, "test.txt", "text/plain", []byte("test content"))
	err := store.CreateAttachment(att)
	if err != nil {
		t.Fatalf("CreateAttachment failed: %v", err)
	}

	// Read
	got, err := store.GetAttachmentByID(att.ID)
	if err != nil {
		t.Fatalf("GetAttachmentByID failed: %v", err)
	}
	if got.Filename != att.Filename {
		t.Errorf("expected filename %q, got %q", att.Filename, got.Filename)
	}
	if string(got.Data) != "test content" {
		t.Errorf("expected data %q, got %q", "test content", string(got.Data))
	}
	if got.MimeType != att.MimeType {
		t.Errorf("expected mime type %q, got %q", att.MimeType, got.MimeType)
	}

	// List
	attachments, err := store.ListAttachmentsByNote(note.ID)
	if err != nil {
		t.Fatalf("ListAttachmentsByNote failed: %v", err)
	}
	if len(attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(attachments))
	}

	// Delete
	err = store.DeleteAttachment(att.ID)
	if err != nil {
		t.Fatalf("DeleteAttachment failed: %v", err)
	}
	_, err = store.GetAttachmentByID(att.ID)
	if err == nil {
		t.Error("expected error getting deleted attachment")
	}
}

func TestMarkdownGetAttachmentByPrefix(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Prefix Att Test", "Content")
	_ = store.CreateNote(note, nil)

	att := models.NewAttachment(note.ID, "file.txt", "text/plain", []byte("data"))
	_ = store.CreateAttachment(att)

	// By prefix
	prefix := att.ID.String()[:8]
	got, err := store.GetAttachmentByPrefix(prefix)
	if err != nil {
		t.Fatalf("GetAttachmentByPrefix failed: %v", err)
	}
	if got.ID != att.ID {
		t.Errorf("expected ID %s, got %s", att.ID, got.ID)
	}

	// Prefix too short
	_, err = store.GetAttachmentByPrefix("abc")
	if !errors.Is(err, ErrPrefixTooShort) {
		t.Errorf("expected ErrPrefixTooShort, got %v", err)
	}

	// Not found
	_, err = store.GetAttachmentByPrefix("zzzzzz")
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got %v", err)
	}
}

func TestMarkdownAttachmentFilenameCollision(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Collision Test", "Content")
	_ = store.CreateNote(note, nil)

	// Create first attachment
	att1 := models.NewAttachment(note.ID, "report.txt", "text/plain", []byte("first report"))
	err := store.CreateAttachment(att1)
	if err != nil {
		t.Fatalf("CreateAttachment (first) failed: %v", err)
	}

	// Create second attachment with same filename
	att2 := models.NewAttachment(note.ID, "report.txt", "text/plain", []byte("second report"))
	err = store.CreateAttachment(att2)
	if err != nil {
		t.Fatalf("CreateAttachment (collision) failed: %v", err)
	}

	// Both should be retrievable
	got1, err := store.GetAttachmentByID(att1.ID)
	if err != nil {
		t.Fatalf("GetAttachmentByID (first) failed: %v", err)
	}
	if string(got1.Data) != "first report" {
		t.Errorf("expected first data, got %q", string(got1.Data))
	}

	got2, err := store.GetAttachmentByID(att2.ID)
	if err != nil {
		t.Fatalf("GetAttachmentByID (second) failed: %v", err)
	}
	if string(got2.Data) != "second report" {
		t.Errorf("expected second data, got %q", string(got2.Data))
	}
	// Original filename should be preserved for display
	if got2.Filename != "report.txt" {
		t.Errorf("expected filename 'report.txt', got %q", got2.Filename)
	}

	// Both should appear in list
	attachments, err := store.ListAttachmentsByNote(note.ID)
	if err != nil {
		t.Fatalf("ListAttachmentsByNote failed: %v", err)
	}
	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(attachments))
	}
}

func TestMarkdownDeleteNoteCascadesAttachments(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Cascade Test", "Content")
	_ = store.CreateNote(note, nil)

	att := models.NewAttachment(note.ID, "file.txt", "text/plain", []byte("data"))
	_ = store.CreateAttachment(att)

	// Verify attachment exists
	_, err := store.GetAttachmentByID(att.ID)
	if err != nil {
		t.Fatalf("attachment should exist before delete: %v", err)
	}

	// Delete note
	err = store.DeleteNote(note.ID)
	if err != nil {
		t.Fatalf("DeleteNote failed: %v", err)
	}

	// Attachment should be gone
	_, err = store.GetAttachmentByID(att.ID)
	if err == nil {
		t.Error("attachment should be deleted when note is deleted")
	}
}

func TestMarkdownDeleteNonexistent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	fakeID := uuid.New()

	err := store.DeleteNote(fakeID)
	if err == nil {
		t.Error("expected error deleting nonexistent note")
	}

	err = store.DeleteAttachment(fakeID)
	if err == nil {
		t.Error("expected error deleting nonexistent attachment")
	}
}

func TestMarkdownGetNonexistent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	fakeID := uuid.New()

	_, _, err := store.GetNoteByID(fakeID)
	if err == nil {
		t.Error("expected error for nonexistent note")
	}

	_, err = store.GetAttachmentByID(fakeID)
	if err == nil {
		t.Error("expected error for nonexistent attachment")
	}
}

func TestMarkdownNoteWithMultilineContent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	content := "# Heading\n\nSome paragraph.\n\n```go\nfunc main() {\n\tprintln(\"hello\")\n}\n```\n\n- list item 1\n- list item 2\n\n> Blockquote here"
	note := models.NewNote("Multiline Test", content)
	_ = store.CreateNote(note, nil)

	got, _, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("GetNoteByID failed: %v", err)
	}
	if got.Content != content {
		t.Errorf("multiline content mismatch:\nwant: %q\ngot:  %q", content, got.Content)
	}
}

func TestMarkdownNoteFileCreation(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("My Test Note", "Content here")
	_ = store.CreateNote(note, nil)

	// Verify file was created with slugified name
	expectedPath := filepath.Join(store.dataDir, "notes", "my-test-note.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected note file at %s", expectedPath)
	}
}

func TestMarkdownNoteFilenameCollision(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note1 := models.NewNote("Hello World!", "Content 1")
	note2 := models.NewNote("Hello World?", "Content 2")

	_ = store.CreateNote(note1, nil)
	_ = store.CreateNote(note2, nil)

	// Both should be retrievable
	got1, _, err := store.GetNoteByID(note1.ID)
	if err != nil {
		t.Fatalf("GetNoteByID (1) failed: %v", err)
	}
	if got1.Title != "Hello World!" {
		t.Errorf("note 1 title mismatch: want %q, got %q", "Hello World!", got1.Title)
	}

	got2, _, err := store.GetNoteByID(note2.ID)
	if err != nil {
		t.Fatalf("GetNoteByID (2) failed: %v", err)
	}
	if got2.Title != "Hello World?" {
		t.Errorf("note 2 title mismatch: want %q, got %q", "Hello World?", got2.Title)
	}
}

func TestMarkdownTagNormalization(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Tag Norm Test", "Content")
	_ = store.CreateNote(note, []string{"  UPPER  ", "MiXeD", ""})

	tags, _ := store.GetNoteTags(note.ID)

	if !containsTag(tags, "upper") {
		t.Errorf("expected normalized 'upper' tag, got %v", tags)
	}
	if !containsTag(tags, "mixed") {
		t.Errorf("expected normalized 'mixed' tag, got %v", tags)
	}
	// Empty tag should be filtered out
	if len(tags) != 2 {
		t.Errorf("expected 2 tags (empty filtered out), got %d: %v", len(tags), tags)
	}
}

func TestMarkdownClose(t *testing.T) {
	store := newTestMarkdownStore(t)
	err := store.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestMarkdownEmptyListNotes(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("ListNotes on empty store failed: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}
}

func TestMarkdownEmptyListAllTags(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	tags, err := store.ListAllTags()
	if err != nil {
		t.Fatalf("ListAllTags on empty store failed: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

func TestMarkdownEmptyListAttachments(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("No Attachments", "Content")
	_ = store.CreateNote(note, nil)

	attachments, err := store.ListAttachmentsByNote(note.ID)
	if err != nil {
		t.Fatalf("ListAttachmentsByNote on note with no attachments failed: %v", err)
	}
	if len(attachments) != 0 {
		t.Errorf("expected 0 attachments, got %d", len(attachments))
	}
}

func TestMarkdownNoteContentWithFrontmatterDelimiters(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Content that looks like frontmatter
	content := "Here is some content\n---\nThis looks like frontmatter\n---\nBut it's not"
	note := models.NewNote("Delimiter Test", content)
	_ = store.CreateNote(note, nil)

	got, _, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("GetNoteByID failed: %v", err)
	}
	if got.Content != content {
		t.Errorf("content with frontmatter delimiters was corrupted:\nwant: %q\ngot:  %q", content, got.Content)
	}
}

func TestMarkdownUpdateNoteChangesFilename(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Original Title", "Content")
	_ = store.CreateNote(note, nil)

	// Verify original file exists
	origPath := filepath.Join(store.notesDirPath(), "original-title.md")
	if _, err := os.Stat(origPath); os.IsNotExist(err) {
		t.Fatal("original file should exist")
	}

	// Update title
	note.Title = "New Title"
	note.Touch()
	_ = store.UpdateNote(note, nil)

	// Old file should be gone
	if _, err := os.Stat(origPath); !os.IsNotExist(err) {
		t.Error("old file should be removed after title change")
	}

	// New file should exist
	newPath := filepath.Join(store.notesDirPath(), "new-title.md")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("new file should exist after title change")
	}

	// Note should be retrievable
	got, _, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("GetNoteByID after rename failed: %v", err)
	}
	if got.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %q", got.Title)
	}
}

func TestMarkdownFrontmatterParsing(t *testing.T) {
	content := `---
id: 5681e681-3603-4dbf-b289-08ae47819163
title: Test Note
tags:
    - alpha
    - beta
created_at: "2026-02-02T02:39:08Z"
updated_at: "2026-02-02T02:39:08Z"
---

This is the note content.
`
	fm, err := parseNoteFrontmatter(content)
	if err != nil {
		t.Fatalf("parseNoteFrontmatter failed: %v", err)
	}
	if fm.ID != "5681e681-3603-4dbf-b289-08ae47819163" {
		t.Errorf("expected ID 5681e681-3603-4dbf-b289-08ae47819163, got %s", fm.ID)
	}
	if fm.Title != "Test Note" {
		t.Errorf("expected title 'Test Note', got %s", fm.Title)
	}
	if len(fm.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(fm.Tags))
	}
}

func TestMarkdownNoteWithEmptyContent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Empty Content", "")
	_ = store.CreateNote(note, nil)

	got, _, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("GetNoteByID failed: %v", err)
	}
	if got.Content != "" {
		t.Errorf("expected empty content, got %q", got.Content)
	}
}

func TestMarkdownAddEmptyTag(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Empty Tag", "Content")
	_ = store.CreateNote(note, nil)

	// Adding empty tag should be no-op
	err := store.AddTagToNote(note.ID, "")
	if err != nil {
		t.Fatalf("AddTagToNote with empty string failed: %v", err)
	}

	err = store.AddTagToNote(note.ID, "   ")
	if err != nil {
		t.Fatalf("AddTagToNote with whitespace failed: %v", err)
	}

	tags, _ := store.GetNoteTags(note.ID)
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after adding empty tags, got %d: %v", len(tags), tags)
	}
}

func TestMarkdownRemoveEmptyTag(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Remove Empty Tag", "Content")
	_ = store.CreateNote(note, nil)

	// Removing empty tag should be no-op
	err := store.RemoveTagFromNote(note.ID, "")
	if err != nil {
		t.Fatalf("RemoveTagFromNote with empty string failed: %v", err)
	}
}

// --- Concurrency Tests ---

func TestMarkdownConcurrentNoteCreation(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	const numGoroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			note := models.NewNote(fmt.Sprintf("Note %d", idx), fmt.Sprintf("Content %d", idx))
			if err := store.CreateNote(note, []string{fmt.Sprintf("tag-%d", idx)}); err != nil {
				errs <- fmt.Errorf("goroutine %d: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	// Verify all notes are readable
	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != numGoroutines {
		t.Errorf("expected %d notes, got %d", numGoroutines, len(notes))
	}
}

func TestMarkdownConcurrentTagOperations(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Concurrent Tag Test", "Content")
	_ = store.CreateNote(note, nil)

	const numGoroutines = 15
	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			tagName := fmt.Sprintf("concurrent-tag-%d", idx)
			if err := store.AddTagToNote(note.ID, tagName); err != nil {
				errs <- fmt.Errorf("goroutine %d: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	tags, err := store.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("GetNoteTags failed: %v", err)
	}
	if len(tags) != numGoroutines {
		t.Errorf("expected %d tags, got %d", numGoroutines, len(tags))
	}
}

func TestMarkdownNoteWithSpecialCharactersInTitle(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	tests := []struct {
		title   string
		content string
	}{
		{"Title with spaces", "Content 1"},
		{"Title-with-hyphens", "Content 2"},
		{"UPPERCASE TITLE", "Content 3"},
		{"123-numbers-first", "Content 4"},
		{"Special! @chars# $here", "Content 5"},
	}

	for _, tt := range tests {
		note := models.NewNote(tt.title, tt.content)
		if err := store.CreateNote(note, nil); err != nil {
			t.Errorf("CreateNote(%q) failed: %v", tt.title, err)
			continue
		}

		got, _, err := store.GetNoteByID(note.ID)
		if err != nil {
			t.Errorf("GetNoteByID for %q failed: %v", tt.title, err)
			continue
		}
		if got.Title != tt.title {
			t.Errorf("title mismatch: want %q, got %q", tt.title, got.Title)
		}
	}

	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != len(tests) {
		t.Errorf("expected %d notes, got %d", len(tests), len(notes))
	}
}

func TestMarkdownBinaryAttachment(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Binary Attachment", "Content")
	_ = store.CreateNote(note, nil)

	// Binary data (PNG header-like)
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0xFF}
	att := models.NewAttachment(note.ID, "image.png", "image/png", binaryData)
	_ = store.CreateAttachment(att)

	got, err := store.GetAttachmentByID(att.ID)
	if err != nil {
		t.Fatalf("GetAttachmentByID failed: %v", err)
	}
	if string(got.Data) != string(binaryData) {
		t.Errorf("binary data mismatch: want %v, got %v", binaryData, got.Data)
	}
}

func TestMarkdownTagRegistry(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Create notes with tags
	_ = store.CreateNote(models.NewNote("N1", "C"), []string{"alpha", "beta"})
	_ = store.CreateNote(models.NewNote("N2", "C"), []string{"gamma"})

	// Verify _tags.yaml exists and contains tags
	tagsPath := store.tagsFilePath()
	if _, err := os.Stat(tagsPath); os.IsNotExist(err) {
		t.Fatal("_tags.yaml should exist after creating notes with tags")
	}

	entries, err := store.readTagRegistry()
	if err != nil {
		t.Fatalf("readTagRegistry failed: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 tag entries in registry, got %d", len(entries))
	}

	// Verify sorted
	for i := 0; i < len(entries)-1; i++ {
		if entries[i].Name > entries[i+1].Name {
			t.Errorf("tag registry not sorted: %q > %q", entries[i].Name, entries[i+1].Name)
		}
	}
}

func TestMarkdownNoteWithManyTags(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	var tags []string
	for i := 0; i < 20; i++ {
		tags = append(tags, fmt.Sprintf("tag-%02d", i))
	}

	note := models.NewNote("Many Tags", "Content")
	_ = store.CreateNote(note, tags)

	gotTags, err := store.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("GetNoteTags failed: %v", err)
	}
	if len(gotTags) != 20 {
		t.Errorf("expected 20 tags, got %d", len(gotTags))
	}
}

func TestMarkdownMalformedNoteFile(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Create a valid note first
	validNote := models.NewNote("Valid Note", "Valid content")
	_ = store.CreateNote(validNote, nil)

	// Write a malformed file to the notes directory
	malformedPath := filepath.Join(store.notesDirPath(), "malformed.md")
	if err := os.WriteFile(malformedPath, []byte("this has no frontmatter"), 0644); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}

	// ListNotes should skip malformed files and return valid ones
	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 valid note (skipping malformed), got %d", len(notes))
	}
}

func TestMarkdownAddTagToNonexistentNote(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	err := store.AddTagToNote(uuid.New(), "tag")
	if err == nil {
		t.Error("expected error adding tag to nonexistent note")
	}
}

func TestMarkdownRemoveTagFromNonexistentNote(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	err := store.RemoveTagFromNote(uuid.New(), "tag")
	if err == nil {
		t.Error("expected error removing tag from nonexistent note")
	}
}

func TestMarkdownSearchCaseInsensitive(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	_ = store.CreateNote(models.NewNote("UPPERCASE TITLE", "lowercase content"), nil)
	_ = store.CreateNote(models.NewNote("lowercase title", "UPPERCASE CONTENT"), nil)

	// Search should be case-insensitive
	results, err := store.ListNotes(&NoteFilter{Search: "uppercase"})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 case-insensitive matches, got %d", len(results))
	}
}

func TestMarkdownMultipleNotesMultipleAttachments(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note1 := models.NewNote("Note 1", "Content 1")
	note2 := models.NewNote("Note 2", "Content 2")
	_ = store.CreateNote(note1, nil)
	_ = store.CreateNote(note2, nil)

	att1 := models.NewAttachment(note1.ID, "file1.txt", "text/plain", []byte("data1"))
	att2 := models.NewAttachment(note1.ID, "file2.txt", "text/plain", []byte("data2"))
	att3 := models.NewAttachment(note2.ID, "file3.txt", "text/plain", []byte("data3"))
	_ = store.CreateAttachment(att1)
	_ = store.CreateAttachment(att2)
	_ = store.CreateAttachment(att3)

	// Note 1 should have 2 attachments
	atts1, _ := store.ListAttachmentsByNote(note1.ID)
	if len(atts1) != 2 {
		t.Errorf("expected 2 attachments for note 1, got %d", len(atts1))
	}

	// Note 2 should have 1 attachment
	atts2, _ := store.ListAttachmentsByNote(note2.ID)
	if len(atts2) != 1 {
		t.Errorf("expected 1 attachment for note 2, got %d", len(atts2))
	}
}

func TestMarkdownNoteWithContentContainingYAML(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Content that contains YAML-like text and frontmatter delimiters
	content := "Here is some YAML:\n```yaml\nkey: value\n---\nmore: stuff\n```\nEnd of note"
	note := models.NewNote("YAML in Content", content)
	_ = store.CreateNote(note, nil)

	got, _, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("GetNoteByID failed: %v", err)
	}
	if got.Content != content {
		t.Errorf("content with YAML was corrupted:\nwant: %q\ngot:  %q", content, got.Content)
	}
}

func TestMarkdownUpdateNotePreservesAttachments(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	note := models.NewNote("Attachment Preserve", "Content")
	_ = store.CreateNote(note, nil)

	att := models.NewAttachment(note.ID, "preserved.txt", "text/plain", []byte("keep me"))
	_ = store.CreateAttachment(att)

	// Update note
	note.Title = "Updated Title"
	note.Touch()
	_ = store.UpdateNote(note, nil)

	// Attachment should still exist
	got, err := store.GetAttachmentByID(att.ID)
	if err != nil {
		t.Fatalf("attachment should survive note update: %v", err)
	}
	if string(got.Data) != "keep me" {
		t.Errorf("attachment data changed after note update")
	}
}

func TestMarkdownLongContent(t *testing.T) {
	store := newTestMarkdownStore(t)
	defer store.Close()

	// Create a note with very long content.
	// Note: mdstore.ParseFrontmatter trims trailing whitespace from the body,
	// so content should not rely on trailing spaces being preserved.
	longContent := strings.TrimRight(strings.Repeat("This is a long line of content. ", 1000), " ")
	note := models.NewNote("Long Content", longContent)
	_ = store.CreateNote(note, nil)

	got, _, err := store.GetNoteByID(note.ID)
	if err != nil {
		t.Fatalf("GetNoteByID failed: %v", err)
	}
	if got.Content != longContent {
		t.Errorf("long content length mismatch: want %d, got %d", len(longContent), len(got.Content))
	}
}
