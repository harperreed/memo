// ABOUTME: Data migration between memo storage backends.
// ABOUTME: Copies notes, tags, and attachments from source to destination.

package storage

import (
	"fmt"
	"os"
)

// MigrateSummary holds counts of migrated entities.
type MigrateSummary struct {
	Notes       int
	Tags        int
	Attachments int
}

// MigrateData copies all data from src to dst storage.
// It iterates through notes, their tags, and their attachments,
// creating each entity in the destination. The destination should be empty
// before calling this function.
func MigrateData(src, dst Storage) (*MigrateSummary, error) {
	summary := &MigrateSummary{}

	// List all notes (no filter = get everything)
	notes, err := src.ListNotes(nil)
	if err != nil {
		return nil, fmt.Errorf("list source notes: %w", err)
	}

	for _, note := range notes {
		// Get tags for this note
		tags, err := src.GetNoteTags(note.ID)
		if err != nil {
			return nil, fmt.Errorf("get tags for note %q (%s): %w", note.Title, note.ID, err)
		}

		// Create note with tags in destination
		if err := dst.CreateNote(note, tags); err != nil {
			return nil, fmt.Errorf("create note %q (%s): %w", note.Title, note.ID, err)
		}
		summary.Notes++
		summary.Tags += len(tags)

		// Migrate attachments for this note
		attachments, err := src.ListAttachmentsByNote(note.ID)
		if err != nil {
			return nil, fmt.Errorf("list attachments for note %q (%s): %w", note.Title, note.ID, err)
		}

		for _, att := range attachments {
			if err := dst.CreateAttachment(att); err != nil {
				return nil, fmt.Errorf("create attachment %q for note %q: %w", att.Filename, note.Title, err)
			}
			summary.Attachments++
		}
	}

	return summary, nil
}

// IsDirNonEmpty checks whether a directory exists and contains any files or subdirectories.
// Returns false if the directory does not exist or is empty.
func IsDirNonEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read directory %q: %w", path, err)
	}
	return len(entries) > 0, nil
}
