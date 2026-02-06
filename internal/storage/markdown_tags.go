// ABOUTME: Tag operations for MarkdownStore backend.
// ABOUTME: Handles tag listing, adding, and removing from notes via frontmatter and registry.

package storage

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/suite/mdstore"

	"github.com/harper/memo/internal/models"
)

// ListAllTags returns all unique tags with their usage counts.
func (s *MarkdownStore) ListAllTags() ([]*TagWithCount, error) {
	_, tagMap, err := s.readAllNotes()
	if err != nil {
		return nil, err
	}

	// Count tag usage across all notes
	counts := make(map[string]int)
	for _, tags := range tagMap {
		for _, t := range tags {
			counts[strings.ToLower(t)]++
		}
	}

	// Build result sorted by tag name
	var result []*TagWithCount
	for name, count := range counts {
		result = append(result, &TagWithCount{
			Tag:   models.NewTag(name),
			Count: count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Tag.Name < result[j].Tag.Name
	})

	return result, nil
}

// AddTagToNote adds a tag to a note.
func (s *MarkdownStore) AddTagToNote(noteID uuid.UUID, tagName string) error {
	normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
	if normalizedTag == "" {
		return nil
	}

	return mdstore.WithLock(s.dataDir, func() error {
		// Read current note
		path, err := s.findNoteFile(noteID)
		if err != nil {
			return err
		}

		note, tags, err := noteFromFile(path)
		if err != nil {
			return fmt.Errorf("read note file: %w", err)
		}

		// Check if tag already exists
		for _, t := range tags {
			if strings.ToLower(t) == normalizedTag {
				return nil
			}
		}

		// Add tag
		tags = append(tags, normalizedTag)

		// Re-render and write
		content, err := renderNote(note, tags)
		if err != nil {
			return fmt.Errorf("render note: %w", err)
		}

		if err := mdstore.AtomicWrite(path, []byte(content)); err != nil {
			return fmt.Errorf("write note file: %w", err)
		}

		// Register tag
		if err := s.ensureTagInRegistry(normalizedTag); err != nil {
			return fmt.Errorf("register tag %q: %w", normalizedTag, err)
		}

		return nil
	})
}

// RemoveTagFromNote removes a tag from a note.
func (s *MarkdownStore) RemoveTagFromNote(noteID uuid.UUID, tagName string) error {
	normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
	if normalizedTag == "" {
		return nil
	}

	return mdstore.WithLock(s.dataDir, func() error {
		// Read current note
		path, err := s.findNoteFile(noteID)
		if err != nil {
			return err
		}

		note, tags, err := noteFromFile(path)
		if err != nil {
			return fmt.Errorf("read note file: %w", err)
		}

		// Remove tag
		var remaining []string
		for _, t := range tags {
			if strings.ToLower(t) != normalizedTag {
				remaining = append(remaining, t)
			}
		}

		// Re-render and write
		content, err := renderNote(note, remaining)
		if err != nil {
			return fmt.Errorf("render note: %w", err)
		}

		if err := mdstore.AtomicWrite(path, []byte(content)); err != nil {
			return fmt.Errorf("write note file: %w", err)
		}

		return nil
	})
}
