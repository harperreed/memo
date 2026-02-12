// ABOUTME: Storage interface for memo data backends.
// ABOUTME: Defines the contract that all storage implementations must satisfy.

package storage

import (
	"github.com/google/uuid"
	"github.com/harperreed/memo/internal/models"
)

// Storage defines the interface for memo data persistence.
// Implementations include SqliteStore and MarkdownStore.
type Storage interface {
	// Note CRUD
	CreateNote(note *models.Note, tags []string) error
	GetNoteByID(id uuid.UUID) (*models.Note, []string, error)
	GetNoteByPrefix(prefix string) (*models.Note, []string, error)
	UpdateNote(note *models.Note, tags []string) error
	DeleteNote(id uuid.UUID) error
	ListNotes(filter *NoteFilter) ([]*models.Note, error)
	GetNoteTags(id uuid.UUID) ([]string, error)
	CountGlobalNotes() (int, error)

	// Tag operations
	ListAllTags() ([]*TagWithCount, error)
	AddTagToNote(noteID uuid.UUID, tagName string) error
	RemoveTagFromNote(noteID uuid.UUID, tagName string) error

	// Attachment CRUD
	CreateAttachment(att *models.Attachment) error
	GetAttachmentByID(id uuid.UUID) (*models.Attachment, error)
	GetAttachmentByPrefix(prefix string) (*models.Attachment, error)
	ListAttachmentsByNote(noteID uuid.UUID) ([]*models.Attachment, error)
	DeleteAttachment(id uuid.UUID) error

	// Close releases resources held by the storage backend.
	Close() error
}
