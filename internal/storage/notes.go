// ABOUTME: Note CRUD operations for SQLite storage.
// ABOUTME: Handles note creation, retrieval, update, deletion with FTS5 search.

package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

var (
	// ErrNoteNotFound is returned when a note cannot be found.
	ErrNoteNotFound = errors.New("note not found")
	// ErrPrefixTooShort is returned when a note ID prefix is less than 6 characters.
	ErrPrefixTooShort = errors.New("prefix must be at least 6 characters")
	// ErrAmbiguousPrefix is returned when a prefix matches multiple notes.
	ErrAmbiguousPrefix = errors.New("prefix matches multiple notes")
)

// NoteFilter defines criteria for filtering notes.
type NoteFilter struct {
	Tag    *string // Filter by tag name
	DirTag *string // Filter by dir: tag
	Global bool    // Only notes without dir: tags
	Limit  int     // Max results (0 = unlimited)
	Search string  // FTS search term
}

// CreateNote creates a new note with the given tags.
func (s *Store) CreateNote(note *models.Note, tags []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // Rollback is a no-op if committed

	// Insert the note
	_, err = tx.Exec(`
		INSERT INTO notes (id, title, content, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, note.ID.String(), note.Title, note.Content, note.CreatedAt, note.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert note: %w", err)
	}

	// Add tags
	if err := s.setNoteTags(tx, note.ID.String(), tags); err != nil {
		return err
	}

	return tx.Commit()
}

// GetNoteByID retrieves a note by its UUID.
func (s *Store) GetNoteByID(id uuid.UUID) (*models.Note, []string, error) {
	note, err := s.getNoteByIDInternal(id.String())
	if err != nil {
		return nil, nil, err
	}

	tags, err := s.getNoteTags(id.String())
	if err != nil {
		return nil, nil, err
	}

	return note, tags, nil
}

// GetNoteByPrefix finds a note by ID prefix (minimum 6 chars).
func (s *Store) GetNoteByPrefix(prefix string) (*models.Note, []string, error) {
	if len(prefix) < 6 {
		return nil, nil, ErrPrefixTooShort
	}

	// Find matching notes
	rows, err := s.db.Query(`
		SELECT id, title, content, created_at, updated_at
		FROM notes
		WHERE id LIKE ?
	`, prefix+"%")
	if err != nil {
		return nil, nil, fmt.Errorf("query notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var notes []*models.Note
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			continue
		}
		notes = append(notes, note)
	}

	if len(notes) == 0 {
		return nil, nil, ErrNoteNotFound
	}
	if len(notes) > 1 {
		return nil, nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(notes))
	}

	tags, err := s.getNoteTags(notes[0].ID.String())
	if err != nil {
		return nil, nil, err
	}

	return notes[0], tags, nil
}

// UpdateNote updates an existing note's title, content, and tags.
func (s *Store) UpdateNote(note *models.Note, tags []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // Rollback is a no-op if committed

	// Check if note exists
	var exists bool
	err = tx.QueryRow(`SELECT 1 FROM notes WHERE id = ?`, note.ID.String()).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoteNotFound
	}
	if err != nil {
		return fmt.Errorf("check note exists: %w", err)
	}

	// Update the note
	_, err = tx.Exec(`
		UPDATE notes
		SET title = ?, content = ?, updated_at = ?
		WHERE id = ?
	`, note.Title, note.Content, note.UpdatedAt, note.ID.String())
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}

	// Update tags
	if err := s.setNoteTags(tx, note.ID.String(), tags); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteNote deletes a note and its attachments (via CASCADE).
func (s *Store) DeleteNote(id uuid.UUID) error {
	result, err := s.db.Exec(`DELETE FROM notes WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNoteNotFound
	}

	return nil
}

// ListNotes returns notes matching the filter, sorted by updated_at desc.
func (s *Store) ListNotes(filter *NoteFilter) ([]*models.Note, error) {
	if filter == nil {
		filter = &NoteFilter{}
	}

	// Use FTS search if search term is provided
	if filter.Search != "" {
		return s.searchNotes(filter)
	}

	// Build query based on filters
	query := `
		SELECT DISTINCT n.id, n.title, n.content, n.created_at, n.updated_at
		FROM notes n
	`
	var args []interface{}
	var conditions []string

	// Tag filter
	if filter.Tag != nil {
		query += `
			JOIN note_tags nt ON n.id = nt.note_id
			JOIN tags t ON nt.tag_id = t.id
		`
		conditions = append(conditions, "t.name = ?")
		args = append(args, strings.ToLower(strings.TrimSpace(*filter.Tag)))
	}

	// Dir tag filter
	if filter.DirTag != nil {
		query += `
			JOIN note_tags nt2 ON n.id = nt2.note_id
			JOIN tags t2 ON nt2.tag_id = t2.id
		`
		conditions = append(conditions, "t2.name = ?")
		args = append(args, strings.ToLower("dir:"+*filter.DirTag))
	}

	// Global filter (no dir: tags)
	if filter.Global {
		conditions = append(conditions, `
			NOT EXISTS (
				SELECT 1 FROM note_tags nt3
				JOIN tags t3 ON nt3.tag_id = t3.id
				WHERE nt3.note_id = n.id AND t3.name LIKE 'dir:%'
			)
		`)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY n.updated_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var notes []*models.Note
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			continue
		}
		notes = append(notes, note)
	}

	return notes, nil
}

// searchNotes performs FTS5 search on notes.
func (s *Store) searchNotes(filter *NoteFilter) ([]*models.Note, error) {
	query := `
		SELECT n.id, n.title, n.content, n.created_at, n.updated_at
		FROM notes n
		JOIN notes_fts fts ON n.rowid = fts.rowid
		WHERE notes_fts MATCH ?
		ORDER BY n.updated_at DESC
	`
	args := []interface{}{filter.Search}

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var notes []*models.Note
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			continue
		}
		notes = append(notes, note)
	}

	return notes, nil
}

// GetNoteTags returns the tag names for a note.
func (s *Store) GetNoteTags(id uuid.UUID) ([]string, error) {
	return s.getNoteTags(id.String())
}

// CountGlobalNotes returns the count of notes without dir: tags.
func (s *Store) CountGlobalNotes() (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM notes n
		WHERE NOT EXISTS (
			SELECT 1 FROM note_tags nt
			JOIN tags t ON nt.tag_id = t.id
			WHERE nt.note_id = n.id AND t.name LIKE 'dir:%'
		)
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count global notes: %w", err)
	}
	return count, nil
}

// Internal helpers

func (s *Store) getNoteByIDInternal(id string) (*models.Note, error) {
	row := s.db.QueryRow(`
		SELECT id, title, content, created_at, updated_at
		FROM notes
		WHERE id = ?
	`, id)

	note, err := scanNoteRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get note: %w", err)
	}

	return note, nil
}

func (s *Store) getNoteTags(noteID string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT t.name
		FROM tags t
		JOIN note_tags nt ON t.id = nt.tag_id
		WHERE nt.note_id = ?
		ORDER BY t.name
	`, noteID)
	if err != nil {
		return nil, fmt.Errorf("get note tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tags = append(tags, name)
	}

	return tags, nil
}

func (s *Store) setNoteTags(tx *sql.Tx, noteID string, tags []string) error {
	// Remove existing tags for this note
	_, err := tx.Exec(`DELETE FROM note_tags WHERE note_id = ?`, noteID)
	if err != nil {
		return fmt.Errorf("delete existing tags: %w", err)
	}

	// Add new tags
	for _, tagName := range tags {
		normalizedTag := strings.ToLower(strings.TrimSpace(tagName))
		if normalizedTag == "" {
			continue
		}

		// Get or create tag
		var tagID int64
		err = tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, normalizedTag).Scan(&tagID)
		if errors.Is(err, sql.ErrNoRows) {
			result, err := tx.Exec(`INSERT INTO tags (name) VALUES (?)`, normalizedTag)
			if err != nil {
				return fmt.Errorf("insert tag: %w", err)
			}
			tagID, err = result.LastInsertId()
			if err != nil {
				return fmt.Errorf("get tag id: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("get tag: %w", err)
		}

		// Link note to tag
		_, err = tx.Exec(`INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)`, noteID, tagID)
		if err != nil {
			return fmt.Errorf("link note to tag: %w", err)
		}
	}

	return nil
}

func scanNote(rows *sql.Rows) (*models.Note, error) {
	var idStr string
	var title, content string
	var createdAt, updatedAt time.Time

	if err := rows.Scan(&idStr, &title, &content, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse note ID: %w", err)
	}

	return &models.Note{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func scanNoteRow(row *sql.Row) (*models.Note, error) {
	var idStr string
	var title, content string
	var createdAt, updatedAt time.Time

	if err := row.Scan(&idStr, &title, &content, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse note ID: %w", err)
	}

	return &models.Note{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}
