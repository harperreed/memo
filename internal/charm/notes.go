// ABOUTME: Note operations using Charm KV storage
// ABOUTME: Uses type-prefixed keys (note:uuid) with denormalized tags

package charm

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

const (
	// NotePrefix is the key prefix for notes.
	NotePrefix = "note:"
)

var (
	ErrPrefixTooShort  = errors.New("prefix must be at least 6 characters")
	ErrAmbiguousPrefix = errors.New("prefix matches multiple notes")
	ErrNoteNotFound    = errors.New("note not found")
)

// NoteData represents a note stored in charm KV.
type NoteData struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags,omitempty"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

// ToModel converts NoteData to a models.Note.
func (n *NoteData) ToModel() (*models.Note, error) {
	id, err := uuid.Parse(n.ID)
	if err != nil {
		return nil, fmt.Errorf("parse note ID: %w", err)
	}
	return &models.Note{
		ID:        id,
		Title:     n.Title,
		Content:   n.Content,
		CreatedAt: time.Unix(n.CreatedAt, 0),
		UpdatedAt: time.Unix(n.UpdatedAt, 0),
	}, nil
}

// FromModel creates NoteData from a models.Note with tags.
func FromModel(note *models.Note, tags []string) *NoteData {
	return &NoteData{
		ID:        note.ID.String(),
		Title:     note.Title,
		Content:   note.Content,
		Tags:      tags,
		CreatedAt: note.CreatedAt.Unix(),
		UpdatedAt: note.UpdatedAt.Unix(),
	}
}

// noteKey returns the key for a note.
func noteKey(id uuid.UUID) []byte {
	return []byte(NotePrefix + id.String())
}

// CreateNote creates a new note.
func (c *Client) CreateNote(note *models.Note, tags []string) error {
	data := FromModel(note, tags)
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal note: %w", err)
	}
	return c.Set(noteKey(note.ID), encoded)
}

// GetNoteByID retrieves a note by its UUID.
func (c *Client) GetNoteByID(id uuid.UUID) (*models.Note, []string, error) {
	data, err := c.Get(noteKey(id))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, nil, ErrNoteNotFound
		}
		return nil, nil, err
	}

	var noteData NoteData
	if err := json.Unmarshal(data, &noteData); err != nil {
		return nil, nil, fmt.Errorf("unmarshal note: %w", err)
	}

	note, err := noteData.ToModel()
	if err != nil {
		return nil, nil, err
	}
	return note, noteData.Tags, nil
}

// GetNoteByPrefix finds a note by ID prefix (minimum 6 chars).
func (c *Client) GetNoteByPrefix(prefix string) (*models.Note, []string, error) {
	if len(prefix) < 6 {
		return nil, nil, ErrPrefixTooShort
	}

	var matches []*NoteData
	searchPrefix := []byte(NotePrefix + prefix)

	err := c.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(searchPrefix); it.ValidForPrefix(searchPrefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var nd NoteData
				if err := json.Unmarshal(val, &nd); err != nil {
					return err
				}
				matches = append(matches, &nd)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	if len(matches) == 0 {
		return nil, nil, ErrNoteNotFound
	}
	if len(matches) > 1 {
		return nil, nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(matches))
	}

	note, err := matches[0].ToModel()
	if err != nil {
		return nil, nil, err
	}
	return note, matches[0].Tags, nil
}

// NoteFilter defines criteria for filtering notes.
type NoteFilter struct {
	Tag    *string // Filter by tag name
	DirTag *string // Filter by dir: tag
	Global bool    // Only notes without dir: tags
	Limit  int     // Max results (0 = unlimited)
	Search string  // FTS search term (simple contains for now)
}

// ListNotes returns notes matching the filter, sorted by updated_at desc.
func (c *Client) ListNotes(filter *NoteFilter) ([]*models.Note, error) {
	var notes []*NoteData

	prefix := []byte(NotePrefix)
	err := c.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var nd NoteData
				if err := json.Unmarshal(val, &nd); err != nil {
					return err
				}

				// Apply filters
				if !matchesFilter(&nd, filter) {
					return nil
				}

				notes = append(notes, &nd)
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

	// Sort by updated_at descending
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].UpdatedAt > notes[j].UpdatedAt
	})

	// Apply limit
	limit := filter.Limit
	if limit > 0 && len(notes) > limit {
		notes = notes[:limit]
	}

	// Convert to models
	result := make([]*models.Note, 0, len(notes))
	for _, nd := range notes {
		note, err := nd.ToModel()
		if err != nil {
			continue // Skip invalid notes
		}
		result = append(result, note)
	}

	return result, nil
}

// matchesFilter checks if a note matches the filter criteria.
func matchesFilter(nd *NoteData, filter *NoteFilter) bool {
	if filter == nil {
		return true
	}

	// Tag filter
	if filter.Tag != nil {
		if !hasTag(nd.Tags, *filter.Tag) {
			return false
		}
	}

	// Dir tag filter
	if filter.DirTag != nil {
		dirTag := strings.ToLower("dir:" + *filter.DirTag)
		if !hasTag(nd.Tags, dirTag) {
			return false
		}
	}

	// Global filter (no dir: tags)
	if filter.Global {
		for _, tag := range nd.Tags {
			if strings.HasPrefix(strings.ToLower(tag), "dir:") {
				return false
			}
		}
	}

	// Search filter (simple contains for now)
	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		titleMatch := strings.Contains(strings.ToLower(nd.Title), searchLower)
		contentMatch := strings.Contains(strings.ToLower(nd.Content), searchLower)
		if !titleMatch && !contentMatch {
			return false
		}
	}

	return true
}

// hasTag checks if a tag exists in the list (case-insensitive).
func hasTag(tags []string, name string) bool {
	nameLower := strings.ToLower(name)
	for _, t := range tags {
		if strings.ToLower(t) == nameLower {
			return true
		}
	}
	return false
}

// UpdateNote updates an existing note.
func (c *Client) UpdateNote(note *models.Note, tags []string) error {
	// Check if note exists first
	_, _, err := c.GetNoteByID(note.ID)
	if err != nil {
		return err
	}

	data := FromModel(note, tags)
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal note: %w", err)
	}
	return c.Set(noteKey(note.ID), encoded)
}

// DeleteNote deletes a note and its attachments.
func (c *Client) DeleteNote(id uuid.UUID) error {
	// Delete attachments first (cascade)
	if err := c.deleteAttachmentsByNote(id); err != nil {
		return fmt.Errorf("delete attachments: %w", err)
	}

	// Delete the note
	if err := c.Delete(noteKey(id)); err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrNoteNotFound
		}
		return err
	}
	return nil
}

// GetNoteTags returns the tags for a note.
func (c *Client) GetNoteTags(id uuid.UUID) ([]string, error) {
	_, tags, err := c.GetNoteByID(id)
	return tags, err
}

// CountGlobalNotes returns count of notes without dir: tags.
func (c *Client) CountGlobalNotes() (int, error) {
	count := 0
	prefix := []byte(NotePrefix)

	err := c.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var nd NoteData
				if err := json.Unmarshal(val, &nd); err != nil {
					return err
				}

				// Check if has any dir: tag
				isGlobal := true
				for _, tag := range nd.Tags {
					if strings.HasPrefix(strings.ToLower(tag), "dir:") {
						isGlobal = false
						break
					}
				}
				if isGlobal {
					count++
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return count, err
}
