// ABOUTME: Note model representing a markdown note with metadata.
// ABOUTME: Provides constructor and methods for note lifecycle.

package models

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID        uuid.UUID
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewNote(title, content string) *Note {
	now := time.Now()
	return &Note{
		ID:        uuid.New(),
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (n *Note) Touch() {
	n.UpdatedAt = time.Now()
}
