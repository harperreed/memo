// ABOUTME: Core MarkdownStore struct and helpers for file-based memo storage.
// ABOUTME: Provides constructor, slug generation, frontmatter parsing, and tag registry via mdstore library.

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/suite/mdstore"
	"gopkg.in/yaml.v3"

	"github.com/harper/memo/internal/models"
)

// MarkdownStore provides file-based storage for memo notes, tags, and attachments.
type MarkdownStore struct {
	dataDir string
}

// Compile-time check that MarkdownStore implements Storage.
var _ Storage = (*MarkdownStore)(nil)

// NewMarkdownStore creates a new markdown-backed store rooted at dataDir.
func NewMarkdownStore(dataDir string) (*MarkdownStore, error) {
	if err := mdstore.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	// Ensure notes subdirectory exists
	if err := mdstore.EnsureDir(filepath.Join(dataDir, "notes")); err != nil {
		return nil, fmt.Errorf("create notes directory: %w", err)
	}
	return &MarkdownStore{dataDir: dataDir}, nil
}

// Close releases resources. For MarkdownStore this is a no-op.
func (s *MarkdownStore) Close() error {
	return nil
}

// notesDirPath returns the path to the notes directory.
func (s *MarkdownStore) notesDirPath() string {
	return filepath.Join(s.dataDir, "notes")
}

// tagsFilePath returns the path to the _tags.yaml file.
func (s *MarkdownStore) tagsFilePath() string {
	return filepath.Join(s.dataDir, "_tags.yaml")
}

// attachmentBasePath returns the base path for attachments.
func (s *MarkdownStore) attachmentBasePath() string {
	return filepath.Join(s.dataDir, "_attachments")
}

// attachmentDirPath returns the directory path for attachments of a specific note.
func (s *MarkdownStore) attachmentDirPath(noteIDPrefix string) string {
	return filepath.Join(s.attachmentBasePath(), noteIDPrefix)
}

// noteFrontmatter holds the YAML frontmatter of a note file.
type noteFrontmatter struct {
	ID        string   `yaml:"id"`
	Title     string   `yaml:"title"`
	Tags      []string `yaml:"tags,omitempty"`
	CreatedAt string   `yaml:"created_at"`
	UpdatedAt string   `yaml:"updated_at"`
}

// noteFileName generates a unique filename for a note.
// Uses slugified title with UUID prefix suffix on collision.
func (s *MarkdownStore) noteFileName(title string, noteID uuid.UUID) string {
	slug := mdstore.Slugify(title)
	base := slug + ".md"
	notesDir := s.notesDirPath()

	path := filepath.Join(notesDir, base)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return base
	}

	// Check if this file belongs to the same note (e.g., during update)
	fm, err := readNoteFrontmatter(path)
	if err == nil && fm.ID == noteID.String() {
		return base
	}

	// Collision: add UUID prefix
	return slug + "-" + noteID.String()[:8] + ".md"
}

// readNoteFrontmatter reads just the frontmatter from a note file.
func readNoteFrontmatter(path string) (*noteFrontmatter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseNoteFrontmatter(string(data))
}

// parseNoteFrontmatter extracts the YAML frontmatter from note markdown content.
func parseNoteFrontmatter(content string) (*noteFrontmatter, error) {
	yamlStr, _ := mdstore.ParseFrontmatter(content)
	if yamlStr == "" {
		return nil, fmt.Errorf("no frontmatter found")
	}

	var fm noteFrontmatter
	if err := yaml.Unmarshal([]byte(yamlStr), &fm); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return &fm, nil
}

// noteFromFile reads a note file and converts it to a Note model with tags.
func noteFromFile(path string) (*models.Note, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	fm, err := parseNoteFrontmatter(string(data))
	if err != nil {
		return nil, nil, err
	}

	id, err := uuid.Parse(fm.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("parse note ID %q: %w", fm.ID, err)
	}

	createdAt, err := mdstore.ParseTime(fm.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("parse note created_at %q: %w", fm.CreatedAt, err)
	}

	updatedAt, err := mdstore.ParseTime(fm.UpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("parse note updated_at %q: %w", fm.UpdatedAt, err)
	}

	// Body is the content after frontmatter
	_, body := mdstore.ParseFrontmatter(string(data))

	// Trim only leading/trailing newlines, preserving internal whitespace
	body = strings.Trim(body, "\n\r")

	note := &models.Note{
		ID:        id,
		Title:     fm.Title,
		Content:   body,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return note, fm.Tags, nil
}

// renderNote renders a complete note file (frontmatter + content body).
func renderNote(note *models.Note, tags []string) (string, error) {
	// Sort tags for deterministic output
	sortedTags := make([]string, len(tags))
	copy(sortedTags, tags)
	sort.Strings(sortedTags)

	fm := noteFrontmatter{
		ID:        note.ID.String(),
		Title:     note.Title,
		Tags:      sortedTags,
		CreatedAt: mdstore.FormatTime(note.CreatedAt.UTC()),
		UpdatedAt: mdstore.FormatTime(note.UpdatedAt.UTC()),
	}

	body := "\n" + note.Content + "\n"

	content, err := mdstore.RenderFrontmatter(&fm, body)
	if err != nil {
		return "", fmt.Errorf("render note frontmatter: %w", err)
	}
	return content, nil
}

// findNoteFile finds the file path for a note by ID by scanning all note files.
func (s *MarkdownStore) findNoteFile(id uuid.UUID) (string, error) {
	notesDir := s.notesDirPath()
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoteNotFound
		}
		return "", fmt.Errorf("read notes directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		fp := filepath.Join(notesDir, entry.Name())
		fm, err := readNoteFrontmatter(fp)
		if err != nil {
			continue
		}
		if fm.ID == id.String() {
			return fp, nil
		}
	}
	return "", ErrNoteNotFound
}

// readAllNotes reads all note files and returns them with their tags.
func (s *MarkdownStore) readAllNotes() ([]*models.Note, map[uuid.UUID][]string, error) {
	notesDir := s.notesDirPath()
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read notes directory: %w", err)
	}

	var notes []*models.Note
	tagMap := make(map[uuid.UUID][]string)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		fp := filepath.Join(notesDir, entry.Name())
		note, tags, err := noteFromFile(fp)
		if err != nil {
			continue
		}
		notes = append(notes, note)
		tagMap[note.ID] = tags
	}

	return notes, tagMap, nil
}

// tagEntry represents a tag in the _tags.yaml file.
type tagEntry struct {
	Name string `yaml:"name"`
}

// readTagRegistry reads the _tags.yaml file.
func (s *MarkdownStore) readTagRegistry() ([]tagEntry, error) {
	var entries []tagEntry
	if err := mdstore.ReadYAML(s.tagsFilePath(), &entries); err != nil {
		return nil, fmt.Errorf("read tags file: %w", err)
	}
	return entries, nil
}

// writeTagRegistry writes the _tags.yaml file atomically.
func (s *MarkdownStore) writeTagRegistry(entries []tagEntry) error {
	return mdstore.WriteYAML(s.tagsFilePath(), entries)
}

// ensureTagInRegistry ensures a tag name exists in the registry.
func (s *MarkdownStore) ensureTagInRegistry(tagName string) error {
	entries, err := s.readTagRegistry()
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Name == tagName {
			return nil
		}
	}

	entries = append(entries, tagEntry{Name: tagName})
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return s.writeTagRegistry(entries)
}

// attachmentMeta holds metadata for an attachment stored alongside the file.
type attachmentMeta struct {
	ID               string `yaml:"id"`
	NoteID           string `yaml:"note_id"`
	Filename         string `yaml:"filename"`
	OriginalFilename string `yaml:"original_filename,omitempty"`
	MimeType         string `yaml:"mime_type"`
	CreatedAt        string `yaml:"created_at"`
}
