// ABOUTME: Database operations for attachments.
// ABOUTME: Handles blob storage and retrieval for note attachments.

package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

var ErrAttachmentNotFound = errors.New("attachment not found")

func CreateAttachment(db *sql.DB, att *models.Attachment) error {
	_, err := db.Exec(
		`INSERT INTO attachments (id, note_id, filename, mime_type, data, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		att.ID.String(), att.NoteID.String(), att.Filename, att.MimeType, att.Data, att.CreatedAt,
	)
	return err
}

func GetAttachment(db *sql.DB, id uuid.UUID) (*models.Attachment, error) {
	att := &models.Attachment{}
	var idStr, noteIDStr string

	err := db.QueryRow(
		`SELECT id, note_id, filename, mime_type, data, created_at
		 FROM attachments WHERE id = ?`,
		id.String(),
	).Scan(&idStr, &noteIDStr, &att.Filename, &att.MimeType, &att.Data, &att.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}

	var parseErr error
	att.ID, parseErr = uuid.Parse(idStr)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid attachment ID in database: %w", parseErr)
	}
	att.NoteID, parseErr = uuid.Parse(noteIDStr)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid note ID in database: %w", parseErr)
	}
	return att, nil
}

func GetAttachmentByPrefix(db *sql.DB, prefix string) (*models.Attachment, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	att := &models.Attachment{}
	var idStr, noteIDStr string

	rows, err := db.Query(
		`SELECT id, note_id, filename, mime_type, data, created_at
		 FROM attachments WHERE id LIKE ?`,
		prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var count int
	for rows.Next() {
		count++
		if count > 1 {
			return nil, ErrAmbiguousPrefix
		}
		if err := rows.Scan(&idStr, &noteIDStr, &att.Filename, &att.MimeType, &att.Data, &att.CreatedAt); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, ErrAttachmentNotFound
	}

	var parseErr error
	att.ID, parseErr = uuid.Parse(idStr)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid attachment ID in database: %w", parseErr)
	}
	att.NoteID, parseErr = uuid.Parse(noteIDStr)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid note ID in database: %w", parseErr)
	}
	return att, nil
}

type AttachmentMeta struct {
	ID        uuid.UUID
	NoteID    uuid.UUID
	Filename  string
	MimeType  string
	CreatedAt string
}

func ListNoteAttachments(db *sql.DB, noteID uuid.UUID) ([]*AttachmentMeta, error) {
	rows, err := db.Query(
		`SELECT id, note_id, filename, mime_type, created_at
		 FROM attachments WHERE note_id = ?
		 ORDER BY created_at DESC`,
		noteID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var attachments []*AttachmentMeta
	for rows.Next() {
		att := &AttachmentMeta{}
		var idStr, noteIDStr string
		if err := rows.Scan(&idStr, &noteIDStr, &att.Filename, &att.MimeType, &att.CreatedAt); err != nil {
			return nil, err
		}
		var parseErr error
		att.ID, parseErr = uuid.Parse(idStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid attachment ID in database: %w", parseErr)
		}
		att.NoteID, parseErr = uuid.Parse(noteIDStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid note ID in database: %w", parseErr)
		}
		attachments = append(attachments, att)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attachments, nil
}

func DeleteAttachment(db *sql.DB, id uuid.UUID) error {
	result, err := db.Exec(`DELETE FROM attachments WHERE id = ?`, id.String())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrAttachmentNotFound
	}
	return nil
}
