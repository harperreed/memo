// ABOUTME: Tag operations using denormalized data from notes
// ABOUTME: Tags are stored inline in notes, this aggregates them

package charm

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

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

	// Get all keys and filter by prefix
	keys, err := c.kv.Keys()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if !bytes.HasPrefix(key, prefix) {
			continue
		}

		val, err := c.kv.Get(key)
		if err != nil {
			continue // Skip keys that can't be read
		}

		var nd NoteData
		if err := json.Unmarshal(val, &nd); err != nil {
			continue // Skip invalid data
		}

		for _, tag := range nd.Tags {
			tagCounts[strings.ToLower(tag)]++
		}
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
