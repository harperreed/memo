// ABOUTME: Database operations for notes.
// ABOUTME: Provides CRUD and prefix-based lookup for notes.

package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

var ErrPrefixTooShort = errors.New("prefix must be at least 6 characters")
var ErrAmbiguousPrefix = errors.New("prefix matches multiple notes")
var ErrNoteNotFound = errors.New("note not found")

func CreateNote(db *sql.DB, note *models.Note) error {
	_, err := db.Exec(
		`INSERT INTO notes (id, title, content, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		note.ID.String(), note.Title, note.Content, note.CreatedAt, note.UpdatedAt,
	)
	return err
}

func GetNoteByID(db *sql.DB, id uuid.UUID) (*models.Note, error) {
	note := &models.Note{}
	var idStr string
	err := db.QueryRow(
		`SELECT id, title, content, created_at, updated_at FROM notes WHERE id = ?`,
		id.String(),
	).Scan(&idStr, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNoteNotFound
	}
	if err != nil {
		return nil, err
	}
	var parseErr error
	note.ID, parseErr = uuid.Parse(idStr)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid note ID in database: %w", parseErr)
	}
	return note, nil
}

func GetNoteByPrefix(db *sql.DB, prefix string) (*models.Note, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	rows, err := db.Query(
		`SELECT id, title, content, created_at, updated_at FROM notes WHERE id LIKE ?`,
		prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		var idStr string
		if err := rows.Scan(&idStr, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		var parseErr error
		note.ID, parseErr = uuid.Parse(idStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid note ID in database: %w", parseErr)
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(notes) == 0 {
		return nil, ErrNoteNotFound
	}
	if len(notes) > 1 {
		return nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(notes))
	}
	return notes[0], nil
}

func ListNotes(db *sql.DB, tag *string, limit int) ([]*models.Note, error) {
	var rows *sql.Rows
	var err error

	if tag != nil {
		rows, err = db.Query(
			`SELECT DISTINCT n.id, n.title, n.content, n.created_at, n.updated_at
			 FROM notes n
			 JOIN note_tags nt ON n.id = nt.note_id
			 JOIN tags t ON nt.tag_id = t.id
			 WHERE t.name = ?
			 ORDER BY n.updated_at DESC
			 LIMIT ?`,
			*tag, limit,
		)
	} else {
		rows, err = db.Query(
			`SELECT id, title, content, created_at, updated_at FROM notes
			 ORDER BY updated_at DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		var idStr string
		if err := rows.Scan(&idStr, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		var parseErr error
		note.ID, parseErr = uuid.Parse(idStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid note ID in database: %w", parseErr)
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notes, nil
}

func UpdateNote(db *sql.DB, note *models.Note) error {
	result, err := db.Exec(
		`UPDATE notes SET title = ?, content = ?, updated_at = ? WHERE id = ?`,
		note.Title, note.Content, time.Now(), note.ID.String(),
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNoteNotFound
	}
	return nil
}

func DeleteNote(db *sql.DB, id uuid.UUID) error {
	result, err := db.Exec(`DELETE FROM notes WHERE id = ?`, id.String())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNoteNotFound
	}
	return nil
}

// ListNotesByDirTag returns notes tagged with a specific directory.
func ListNotesByDirTag(db *sql.DB, dirPath string, limit int) ([]*models.Note, error) {
	dirTag := strings.ToLower("dir:" + dirPath)
	rows, err := db.Query(
		`SELECT DISTINCT n.id, n.title, n.content, n.created_at, n.updated_at
		 FROM notes n
		 JOIN note_tags nt ON n.id = nt.note_id
		 JOIN tags t ON nt.tag_id = t.id
		 WHERE t.name = ?
		 ORDER BY n.updated_at DESC
		 LIMIT ?`,
		dirTag, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		var idStr string
		if err := rows.Scan(&idStr, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		var parseErr error
		note.ID, parseErr = uuid.Parse(idStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid note ID in database: %w", parseErr)
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notes, nil
}

// ListGlobalNotes returns notes that don't have any dir: tag.
func ListGlobalNotes(db *sql.DB, limit int) ([]*models.Note, error) {
	rows, err := db.Query(
		`SELECT n.id, n.title, n.content, n.created_at, n.updated_at
		 FROM notes n
		 WHERE NOT EXISTS (
			 SELECT 1 FROM note_tags nt
			 JOIN tags t ON nt.tag_id = t.id
			 WHERE nt.note_id = n.id AND t.name LIKE 'dir:%'
		 )
		 ORDER BY n.updated_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		var idStr string
		if err := rows.Scan(&idStr, &note.Title, &note.Content, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		var parseErr error
		note.ID, parseErr = uuid.Parse(idStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid note ID in database: %w", parseErr)
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notes, nil
}

// CountGlobalNotes returns the total count of notes without dir: tags.
func CountGlobalNotes(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM notes n
		 WHERE NOT EXISTS (
			 SELECT 1 FROM note_tags nt
			 JOIN tags t ON nt.tag_id = t.id
			 WHERE nt.note_id = n.id AND t.name LIKE 'dir:%'
		 )`,
	).Scan(&count)
	return count, err
}
