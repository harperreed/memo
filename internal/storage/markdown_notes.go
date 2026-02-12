// ABOUTME: Note CRUD operations for MarkdownStore backend.
// ABOUTME: Handles note creation, retrieval, update, deletion with case-insensitive search.

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/harperreed/mdstore"

	"github.com/harperreed/memo/internal/models"
)

// CreateNote creates a new note with the given tags.
func (s *MarkdownStore) CreateNote(note *models.Note, tags []string) error {
	return mdstore.WithLock(s.dataDir, func() error {
		// Normalize tags
		normalizedTags := normalizeTags(tags)

		// Generate filename
		filename := s.noteFileName(note.Title, note.ID)
		path := filepath.Join(s.notesDirPath(), filename)

		// Render note content
		content, err := renderNote(note, normalizedTags)
		if err != nil {
			return fmt.Errorf("render note: %w", err)
		}

		// Write file atomically
		if err := mdstore.AtomicWrite(path, []byte(content)); err != nil {
			return fmt.Errorf("write note file: %w", err)
		}

		// Register tags
		for _, tag := range normalizedTags {
			if err := s.ensureTagInRegistry(tag); err != nil {
				return fmt.Errorf("register tag %q: %w", tag, err)
			}
		}

		return nil
	})
}

// GetNoteByID retrieves a note by its UUID.
func (s *MarkdownStore) GetNoteByID(id uuid.UUID) (*models.Note, []string, error) {
	path, err := s.findNoteFile(id)
	if err != nil {
		return nil, nil, err
	}

	note, tags, err := noteFromFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read note file: %w", err)
	}

	return note, tags, nil
}

// GetNoteByPrefix finds a note by ID prefix (minimum 6 chars).
func (s *MarkdownStore) GetNoteByPrefix(prefix string) (*models.Note, []string, error) {
	if len(prefix) < 6 {
		return nil, nil, ErrPrefixTooShort
	}

	notes, tagMap, err := s.readAllNotes()
	if err != nil {
		return nil, nil, err
	}

	var matches []*models.Note
	for _, note := range notes {
		if strings.HasPrefix(note.ID.String(), prefix) {
			matches = append(matches, note)
		}
	}

	if len(matches) == 0 {
		return nil, nil, ErrNoteNotFound
	}
	if len(matches) > 1 {
		return nil, nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(matches))
	}

	return matches[0], tagMap[matches[0].ID], nil
}

// UpdateNote updates an existing note's title, content, and tags.
func (s *MarkdownStore) UpdateNote(note *models.Note, tags []string) error {
	return mdstore.WithLock(s.dataDir, func() error {
		// Find existing file
		oldPath, err := s.findNoteFile(note.ID)
		if err != nil {
			return err
		}

		// Normalize tags
		normalizedTags := normalizeTags(tags)

		// Generate new filename (may differ if title changed)
		filename := s.noteFileName(note.Title, note.ID)
		newPath := filepath.Join(s.notesDirPath(), filename)

		// Render note content
		content, err := renderNote(note, normalizedTags)
		if err != nil {
			return fmt.Errorf("render note: %w", err)
		}

		// Write file atomically
		if err := mdstore.AtomicWrite(newPath, []byte(content)); err != nil {
			return fmt.Errorf("write note file: %w", err)
		}

		// Remove old file if name changed
		if oldPath != newPath {
			_ = os.Remove(oldPath)
		}

		// Register tags
		for _, tag := range normalizedTags {
			if err := s.ensureTagInRegistry(tag); err != nil {
				return fmt.Errorf("register tag %q: %w", tag, err)
			}
		}

		return nil
	})
}

// DeleteNote deletes a note and its attachments.
func (s *MarkdownStore) DeleteNote(id uuid.UUID) error {
	return mdstore.WithLock(s.dataDir, func() error {
		path, err := s.findNoteFile(id)
		if err != nil {
			return err
		}

		// Delete the note file
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("delete note file: %w", err)
		}

		// Delete attachments directory for this note
		notePrefix := id.String()[:8]
		attDir := s.attachmentDirPath(notePrefix)
		if _, err := os.Stat(attDir); err == nil {
			if err := os.RemoveAll(attDir); err != nil {
				return fmt.Errorf("delete attachments directory: %w", err)
			}
		}

		return nil
	})
}

// ListNotes returns notes matching the filter, sorted by updated_at desc.
func (s *MarkdownStore) ListNotes(filter *NoteFilter) ([]*models.Note, error) {
	if filter == nil {
		filter = &NoteFilter{}
	}

	notes, tagMap, err := s.readAllNotes()
	if err != nil {
		return nil, err
	}

	// Apply filters
	var filtered []*models.Note
	for _, note := range notes {
		tags := tagMap[note.ID]
		if noteMatchesFilter(note, tags, filter) {
			filtered = append(filtered, note)
		}
	}

	// Sort by updated_at desc
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})

	// Apply limit
	if filter.Limit > 0 && len(filtered) > filter.Limit {
		filtered = filtered[:filter.Limit]
	}

	return filtered, nil
}

// noteMatchesFilter checks whether a single note passes all criteria in the given filter.
func noteMatchesFilter(note *models.Note, tags []string, filter *NoteFilter) bool {
	// Tag filter
	if filter.Tag != nil {
		targetTag := strings.ToLower(strings.TrimSpace(*filter.Tag))
		if !containsTag(tags, targetTag) {
			return false
		}
	}

	// DirTag filter
	if filter.DirTag != nil {
		dirTag := strings.ToLower("dir:" + *filter.DirTag)
		if !containsTag(tags, dirTag) {
			return false
		}
	}

	// Global filter (no dir: tags)
	if filter.Global {
		for _, t := range tags {
			if strings.HasPrefix(t, "dir:") {
				return false
			}
		}
	}

	// Search filter (case-insensitive string matching)
	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		titleLower := strings.ToLower(note.Title)
		contentLower := strings.ToLower(note.Content)
		if !strings.Contains(titleLower, searchLower) && !strings.Contains(contentLower, searchLower) {
			return false
		}
	}

	return true
}

// GetNoteTags returns the tag names for a note.
func (s *MarkdownStore) GetNoteTags(id uuid.UUID) ([]string, error) {
	path, err := s.findNoteFile(id)
	if err != nil {
		return nil, err
	}

	_, tags, err := noteFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("read note file: %w", err)
	}

	return tags, nil
}

// CountGlobalNotes returns the count of notes without dir: tags.
func (s *MarkdownStore) CountGlobalNotes() (int, error) {
	_, tagMap, err := s.readAllNotes()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, tags := range tagMap {
		hasDirTag := false
		for _, t := range tags {
			if strings.HasPrefix(t, "dir:") {
				hasDirTag = true
				break
			}
		}
		if !hasDirTag {
			count++
		}
	}

	return count, nil
}

// normalizeTags normalizes tag names to lowercase with trimmed whitespace,
// filtering out empty strings.
func normalizeTags(tags []string) []string {
	var result []string
	for _, t := range tags {
		normalized := strings.ToLower(strings.TrimSpace(t))
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}

// containsTag checks if a tag list contains a specific tag (case-insensitive).
func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.ToLower(t) == target {
			return true
		}
	}
	return false
}
