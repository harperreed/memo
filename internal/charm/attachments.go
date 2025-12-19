// ABOUTME: Attachment operations using Charm KV storage
// ABOUTME: Uses type-prefixed keys (attachment:uuid) with base64 blob data

package charm

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

const (
	// AttachmentPrefix is the key prefix for attachments.
	AttachmentPrefix = "attachment:"
)

var (
	ErrAttachmentNotFound = errors.New("attachment not found")
)

// AttachmentData represents an attachment stored in charm KV.
type AttachmentData struct {
	ID        string `json:"id"`
	NoteID    string `json:"note_id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	Data      string `json:"data"` // base64-encoded
	CreatedAt int64  `json:"created_at"`
}

// ToModel converts AttachmentData to a models.Attachment.
func (a *AttachmentData) ToModel() (*models.Attachment, error) {
	id, err := uuid.Parse(a.ID)
	if err != nil {
		return nil, fmt.Errorf("parse attachment ID: %w", err)
	}
	noteID, err := uuid.Parse(a.NoteID)
	if err != nil {
		return nil, fmt.Errorf("parse note ID: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(a.Data)
	if err != nil {
		return nil, fmt.Errorf("decode attachment data: %w", err)
	}
	return &models.Attachment{
		ID:        id,
		NoteID:    noteID,
		Filename:  a.Filename,
		MimeType:  a.MimeType,
		Data:      data,
		CreatedAt: time.Unix(a.CreatedAt, 0),
	}, nil
}

// FromAttachmentModel creates AttachmentData from a models.Attachment.
func FromAttachmentModel(att *models.Attachment) *AttachmentData {
	return &AttachmentData{
		ID:        att.ID.String(),
		NoteID:    att.NoteID.String(),
		Filename:  att.Filename,
		MimeType:  att.MimeType,
		Data:      base64.StdEncoding.EncodeToString(att.Data),
		CreatedAt: att.CreatedAt.Unix(),
	}
}

// attachmentKey returns the key for an attachment.
func attachmentKey(id uuid.UUID) []byte {
	return []byte(AttachmentPrefix + id.String())
}

// CreateAttachment creates a new attachment.
func (c *Client) CreateAttachment(att *models.Attachment) error {
	data := FromAttachmentModel(att)
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal attachment: %w", err)
	}
	return c.Set(attachmentKey(att.ID), encoded)
}

// GetAttachmentByID retrieves an attachment by its UUID.
func (c *Client) GetAttachmentByID(id uuid.UUID) (*models.Attachment, error) {
	data, err := c.Get(attachmentKey(id))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrAttachmentNotFound
		}
		return nil, err
	}

	var attData AttachmentData
	if err := json.Unmarshal(data, &attData); err != nil {
		return nil, fmt.Errorf("unmarshal attachment: %w", err)
	}

	return attData.ToModel()
}

// GetAttachmentByPrefix finds an attachment by ID prefix (minimum 6 chars).
func (c *Client) GetAttachmentByPrefix(prefix string) (*models.Attachment, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	var matches []*AttachmentData
	searchPrefix := []byte(AttachmentPrefix + prefix)

	err := c.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(searchPrefix); it.ValidForPrefix(searchPrefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var ad AttachmentData
				if err := json.Unmarshal(val, &ad); err != nil {
					return err
				}
				matches = append(matches, &ad)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, ErrAttachmentNotFound
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(matches))
	}

	return matches[0].ToModel()
}

// ListAttachmentsByNote returns all attachments for a note.
func (c *Client) ListAttachmentsByNote(noteID uuid.UUID) ([]*models.Attachment, error) {
	var attachments []*models.Attachment
	prefix := []byte(AttachmentPrefix)
	noteIDStr := noteID.String()

	err := c.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var ad AttachmentData
				if err := json.Unmarshal(val, &ad); err != nil {
					return err
				}
				if ad.NoteID == noteIDStr {
					att, err := ad.ToModel()
					if err != nil {
						return err
					}
					attachments = append(attachments, att)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return attachments, err
}

// DeleteAttachment deletes an attachment by ID.
func (c *Client) DeleteAttachment(id uuid.UUID) error {
	if err := c.Delete(attachmentKey(id)); err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrAttachmentNotFound
		}
		return err
	}
	return nil
}

// deleteAttachmentsByNote deletes all attachments for a note (cascade delete).
func (c *Client) deleteAttachmentsByNote(noteID uuid.UUID) error {
	attachments, err := c.ListAttachmentsByNote(noteID)
	if err != nil {
		return err
	}

	for _, att := range attachments {
		if err := c.Delete(attachmentKey(att.ID)); err != nil {
			// Ignore not found errors during cascade
			if !errors.Is(err, badger.ErrKeyNotFound) {
				return err
			}
		}
	}
	return nil
}
