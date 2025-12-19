// ABOUTME: Tag operations using denormalized data from notes
// ABOUTME: Tags are stored inline in notes, this aggregates them

package charm

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/dgraph-io/badger/v3"
	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

// TagWithCount represents a tag with its usage count.
type TagWithCount struct {
	Tag   *models.Tag
	Count int
}

// ListAllTags returns all unique tags with their usage counts.
func (c *Client) ListAllTags() ([]*TagWithCount, error) {
	tagCounts := make(map[string]int)
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
				for _, tag := range nd.Tags {
					tagCounts[strings.ToLower(tag)]++
				}
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

	// Convert to TagWithCount slice
	result := make([]*TagWithCount, 0, len(tagCounts))
	for name, count := range tagCounts {
		result = append(result, &TagWithCount{
			Tag:   models.NewTag(name),
			Count: count,
		})
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Tag.Name < result[j].Tag.Name
	})

	return result, nil
}

// AddTagToNote adds a tag to a note (updates the note's tags list).
func (c *Client) AddTagToNote(noteID uuid.UUID, tagName string) error {
	note, tags, err := c.GetNoteByID(noteID)
	if err != nil {
		return err
	}

	// Normalize tag name
	normalizedTag := strings.ToLower(strings.TrimSpace(tagName))

	// Check if already has tag
	for _, t := range tags {
		if strings.ToLower(t) == normalizedTag {
			return nil // Already has tag
		}
	}

	// Add tag and update
	tags = append(tags, normalizedTag)
	return c.UpdateNote(note, tags)
}

// RemoveTagFromNote removes a tag from a note.
func (c *Client) RemoveTagFromNote(noteID uuid.UUID, tagName string) error {
	note, tags, err := c.GetNoteByID(noteID)
	if err != nil {
		return err
	}

	// Normalize tag name
	normalizedTag := strings.ToLower(strings.TrimSpace(tagName))

	// Remove tag
	newTags := make([]string, 0, len(tags))
	for _, t := range tags {
		if strings.ToLower(t) != normalizedTag {
			newTags = append(newTags, t)
		}
	}

	return c.UpdateNote(note, newTags)
}
