// ABOUTME: Attachment operations for SQLite storage.
// ABOUTME: Handles attachment CRUD with binary blob data.

package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

var (
	// ErrAttachmentNotFound is returned when an attachment cannot be found.
	ErrAttachmentNotFound = errors.New("attachment not found")
)

// CreateAttachment creates a new attachment.
func (s *SqliteStore) CreateAttachment(att *models.Attachment) error {
	_, err := s.db.Exec(`
		INSERT INTO attachments (id, note_id, filename, mime_type, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, att.ID.String(), att.NoteID.String(), att.Filename, att.MimeType, att.Data, att.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert attachment: %w", err)
	}
	return nil
}

// GetAttachmentByID retrieves an attachment by its UUID.
func (s *SqliteStore) GetAttachmentByID(id uuid.UUID) (*models.Attachment, error) {
	row := s.db.QueryRow(`
		SELECT id, note_id, filename, mime_type, data, created_at
		FROM attachments
		WHERE id = ?
	`, id.String())

	att, err := scanAttachmentRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAttachmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get attachment: %w", err)
	}

	return att, nil
}

// GetAttachmentByPrefix finds an attachment by ID prefix (minimum 6 chars).
func (s *SqliteStore) GetAttachmentByPrefix(prefix string) (*models.Attachment, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	rows, err := s.db.Query(`
		SELECT id, note_id, filename, mime_type, data, created_at
		FROM attachments
		WHERE id LIKE ?
	`, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query attachments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var attachments []*models.Attachment
	for rows.Next() {
		att, err := scanAttachment(rows)
		if err != nil {
			continue
		}
		attachments = append(attachments, att)
	}

	if len(attachments) == 0 {
		return nil, ErrAttachmentNotFound
	}
	if len(attachments) > 1 {
		return nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(attachments))
	}

	return attachments[0], nil
}

// ListAttachmentsByNote returns all attachments for a note.
func (s *SqliteStore) ListAttachmentsByNote(noteID uuid.UUID) ([]*models.Attachment, error) {
	rows, err := s.db.Query(`
		SELECT id, note_id, filename, mime_type, data, created_at
		FROM attachments
		WHERE note_id = ?
		ORDER BY created_at
	`, noteID.String())
	if err != nil {
		return nil, fmt.Errorf("query attachments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var attachments []*models.Attachment
	for rows.Next() {
		att, err := scanAttachment(rows)
		if err != nil {
			continue
		}
		attachments = append(attachments, att)
	}

	return attachments, nil
}

// DeleteAttachment deletes an attachment by ID.
func (s *SqliteStore) DeleteAttachment(id uuid.UUID) error {
	result, err := s.db.Exec(`DELETE FROM attachments WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAttachmentNotFound
	}

	return nil
}

// Internal helpers

func scanAttachment(rows *sql.Rows) (*models.Attachment, error) {
	var idStr, noteIDStr string
	var filename, mimeType string
	var data []byte
	var createdAt time.Time

	if err := rows.Scan(&idStr, &noteIDStr, &filename, &mimeType, &data, &createdAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse attachment ID: %w", err)
	}

	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse note ID: %w", err)
	}

	return &models.Attachment{
		ID:        id,
		NoteID:    noteID,
		Filename:  filename,
		MimeType:  mimeType,
		Data:      data,
		CreatedAt: createdAt,
	}, nil
}

func scanAttachmentRow(row *sql.Row) (*models.Attachment, error) {
	var idStr, noteIDStr string
	var filename, mimeType string
	var data []byte
	var createdAt time.Time

	if err := row.Scan(&idStr, &noteIDStr, &filename, &mimeType, &data, &createdAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse attachment ID: %w", err)
	}

	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse note ID: %w", err)
	}

	return &models.Attachment{
		ID:        id,
		NoteID:    noteID,
		Filename:  filename,
		MimeType:  mimeType,
		Data:      data,
		CreatedAt: createdAt,
	}, nil
}
