// ABOUTME: Attachment model for binary files attached to notes.
// ABOUTME: Stores file content as blob with metadata.

package models

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID        uuid.UUID
	NoteID    uuid.UUID
	Filename  string
	MimeType  string
	Data      []byte
	CreatedAt time.Time
}

func NewAttachment(noteID uuid.UUID, filename, mimeType string, data []byte) *Attachment {
	return &Attachment{
		ID:        uuid.New(),
		NoteID:    noteID,
		Filename:  filename,
		MimeType:  mimeType,
		Data:      data,
		CreatedAt: time.Now(),
	}
}
