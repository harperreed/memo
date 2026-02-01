// ABOUTME: Tag operations for SQLite storage.
// ABOUTME: Handles tag listing, adding, and removing from notes.

package storage

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

// TagWithCount represents a tag with its usage count.
type TagWithCount struct {
	Tag   *models.Tag
	Count int
}

// ListAllTags returns all unique tags with their usage counts.
func (s *Store) ListAllTags() ([]*TagWithCount, error) {
	rows, err := s.db.Query(`
		SELECT t.name, COUNT(nt.note_id) as count
		FROM tags t
		JOIN note_tags nt ON t.id = nt.tag_id
		GROUP BY t.id, t.name
		ORDER BY t.name
	`)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []*TagWithCount
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}
		result = append(result, &TagWithCount{
			Tag:   models.NewTag(name),
			Count: count,
		})
	}

	return result, nil
}

// AddTagToNote adds a tag to a note.
func (s *Store) AddTagToNote(noteID uuid.UUID, tagName string) error {
	normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
	if normalizedTag == "" {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // Rollback is a no-op if committed

	// Get or create tag
	var tagID int64
	err = tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, normalizedTag).Scan(&tagID)
	if err != nil {
		// Tag doesn't exist, create it
		result, err := tx.Exec(`INSERT INTO tags (name) VALUES (?)`, normalizedTag)
		if err != nil {
			return fmt.Errorf("insert tag: %w", err)
		}
		tagID, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get tag id: %w", err)
		}
	}

	// Link note to tag (ignore if already exists)
	_, err = tx.Exec(`INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)`, noteID.String(), tagID)
	if err != nil {
		return fmt.Errorf("link note to tag: %w", err)
	}

	return tx.Commit()
}

// RemoveTagFromNote removes a tag from a note.
func (s *Store) RemoveTagFromNote(noteID uuid.UUID, tagName string) error {
	normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
	if normalizedTag == "" {
		return nil
	}

	// Delete the note-tag link
	_, err := s.db.Exec(`
		DELETE FROM note_tags
		WHERE note_id = ? AND tag_id IN (
			SELECT id FROM tags WHERE name = ?
		)
	`, noteID.String(), normalizedTag)
	if err != nil {
		return fmt.Errorf("remove tag from note: %w", err)
	}

	return nil
}
