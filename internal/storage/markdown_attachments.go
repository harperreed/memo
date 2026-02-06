// ABOUTME: Attachment CRUD operations for MarkdownStore backend.
// ABOUTME: Stores attachment files and metadata in _attachments/<note-prefix>/ directories.

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/suite/mdstore"

	"github.com/harper/memo/internal/models"
)

// CreateAttachment stores a new attachment on disk.
func (s *MarkdownStore) CreateAttachment(att *models.Attachment) error {
	return mdstore.WithLock(s.dataDir, func() error {
		notePrefix := att.NoteID.String()[:8]
		attDir := s.attachmentDirPath(notePrefix)
		if err := mdstore.EnsureDir(attDir); err != nil {
			return fmt.Errorf("create attachment directory: %w", err)
		}

		// Resolve filename, handling collisions by prefixing with attachment ID
		storedFilename := att.Filename
		originalFilename := ""
		dataPath := filepath.Join(attDir, storedFilename)
		if _, err := os.Stat(dataPath); err == nil {
			// File already exists, make unique by prefixing with attachment ID
			originalFilename = att.Filename
			storedFilename = att.ID.String()[:8] + "-" + att.Filename
			dataPath = filepath.Join(attDir, storedFilename)
		}

		// Write attachment data file
		if err := mdstore.AtomicWrite(dataPath, att.Data); err != nil {
			return fmt.Errorf("write attachment data: %w", err)
		}

		// Write metadata file
		meta := attachmentMeta{
			ID:               att.ID.String(),
			NoteID:           att.NoteID.String(),
			Filename:         storedFilename,
			OriginalFilename: originalFilename,
			MimeType:         att.MimeType,
			CreatedAt:        mdstore.FormatTime(att.CreatedAt.UTC()),
		}
		metaPath := filepath.Join(attDir, storedFilename+".meta.yaml")
		if err := mdstore.WriteYAML(metaPath, &meta); err != nil {
			return fmt.Errorf("write attachment metadata: %w", err)
		}

		return nil
	})
}

// GetAttachmentByID retrieves an attachment by its UUID.
func (s *MarkdownStore) GetAttachmentByID(id uuid.UUID) (*models.Attachment, error) {
	attBase := s.attachmentBasePath()
	noteDirs, err := os.ReadDir(attBase)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrAttachmentNotFound
		}
		return nil, fmt.Errorf("read attachment base directory: %w", err)
	}

	for _, noteDir := range noteDirs {
		if !noteDir.IsDir() {
			continue
		}
		dirPath := filepath.Join(attBase, noteDir.Name())
		att, err := s.findAttachmentInDir(dirPath, id)
		if err == nil {
			return att, nil
		}
	}

	return nil, ErrAttachmentNotFound
}

// GetAttachmentByPrefix finds an attachment by ID prefix (minimum 6 chars).
func (s *MarkdownStore) GetAttachmentByPrefix(prefix string) (*models.Attachment, error) {
	if len(prefix) < 6 {
		return nil, ErrPrefixTooShort
	}

	attBase := s.attachmentBasePath()
	noteDirs, err := os.ReadDir(attBase)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrAttachmentNotFound
		}
		return nil, fmt.Errorf("read attachment base directory: %w", err)
	}

	var matches []*models.Attachment
	for _, noteDir := range noteDirs {
		if !noteDir.IsDir() {
			continue
		}
		dirPath := filepath.Join(attBase, noteDir.Name())
		metaFiles, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, mf := range metaFiles {
			if !strings.HasSuffix(mf.Name(), ".meta.yaml") {
				continue
			}
			metaPath := filepath.Join(dirPath, mf.Name())
			att, err := readAttachmentFromMeta(metaPath, dirPath)
			if err != nil {
				continue
			}
			if strings.HasPrefix(att.ID.String(), prefix) {
				matches = append(matches, att)
			}
		}
	}

	if len(matches) == 0 {
		return nil, ErrAttachmentNotFound
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("%w: %d matches", ErrAmbiguousPrefix, len(matches))
	}

	return matches[0], nil
}

// ListAttachmentsByNote returns all attachments for a note.
func (s *MarkdownStore) ListAttachmentsByNote(noteID uuid.UUID) ([]*models.Attachment, error) {
	notePrefix := noteID.String()[:8]
	attDir := s.attachmentDirPath(notePrefix)

	dirEntries, err := os.ReadDir(attDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read attachment directory: %w", err)
	}

	var attachments []*models.Attachment
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".meta.yaml") {
			continue
		}

		metaPath := filepath.Join(attDir, de.Name())
		att, err := readAttachmentFromMeta(metaPath, attDir)
		if err != nil {
			continue
		}
		attachments = append(attachments, att)
	}

	return attachments, nil
}

// DeleteAttachment deletes an attachment from disk.
func (s *MarkdownStore) DeleteAttachment(id uuid.UUID) error {
	return mdstore.WithLock(s.dataDir, func() error {
		attBase := s.attachmentBasePath()
		noteDirs, err := os.ReadDir(attBase)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrAttachmentNotFound
			}
			return fmt.Errorf("read attachment base directory: %w", err)
		}

		for _, noteDir := range noteDirs {
			if !noteDir.IsDir() {
				continue
			}
			dirPath := filepath.Join(attBase, noteDir.Name())
			if err := s.deleteAttachmentInDir(dirPath, id); err == nil {
				return nil
			}
		}

		return ErrAttachmentNotFound
	})
}

// findAttachmentInDir searches a note's attachment directory for an attachment by ID.
func (s *MarkdownStore) findAttachmentInDir(dirPath string, id uuid.UUID) (*models.Attachment, error) {
	metaFiles, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, mf := range metaFiles {
		if !strings.HasSuffix(mf.Name(), ".meta.yaml") {
			continue
		}
		metaPath := filepath.Join(dirPath, mf.Name())
		att, err := readAttachmentFromMeta(metaPath, dirPath)
		if err != nil {
			continue
		}
		if att.ID == id {
			return att, nil
		}
	}

	return nil, fmt.Errorf("attachment not found: %s", id)
}

// deleteAttachmentInDir deletes a specific attachment from a directory.
func (s *MarkdownStore) deleteAttachmentInDir(dirPath string, id uuid.UUID) error {
	metaFiles, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, mf := range metaFiles {
		if !strings.HasSuffix(mf.Name(), ".meta.yaml") {
			continue
		}
		metaPath := filepath.Join(dirPath, mf.Name())
		meta, err := readAttachmentMetaFile(metaPath)
		if err != nil {
			continue
		}
		if meta.ID == id.String() {
			// Delete data file and meta file
			dataPath := filepath.Join(dirPath, meta.Filename)
			_ = os.Remove(dataPath)
			_ = os.Remove(metaPath)
			return nil
		}
	}

	return fmt.Errorf("attachment not found: %s", id)
}

// readAttachmentFromMeta reads an attachment from its metadata file.
func readAttachmentFromMeta(metaPath, dir string) (*models.Attachment, error) {
	meta, err := readAttachmentMetaFile(metaPath)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(meta.ID)
	if err != nil {
		return nil, fmt.Errorf("parse attachment ID %q: %w", meta.ID, err)
	}
	noteID, err := uuid.Parse(meta.NoteID)
	if err != nil {
		return nil, fmt.Errorf("parse attachment note ID %q: %w", meta.NoteID, err)
	}
	createdAt, err := mdstore.ParseTime(meta.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse attachment created_at %q: %w", meta.CreatedAt, err)
	}

	// Read data file
	dataPath := filepath.Join(dir, meta.Filename)
	data, err := os.ReadFile(dataPath)
	if err != nil {
		return nil, fmt.Errorf("read attachment data: %w", err)
	}

	// Use original filename if available (collision case), otherwise use stored filename
	displayFilename := meta.Filename
	if meta.OriginalFilename != "" {
		displayFilename = meta.OriginalFilename
	}

	return &models.Attachment{
		ID:        id,
		NoteID:    noteID,
		Filename:  displayFilename,
		MimeType:  meta.MimeType,
		Data:      data,
		CreatedAt: createdAt,
	}, nil
}

// readAttachmentMetaFile reads an attachment metadata YAML file.
func readAttachmentMetaFile(path string) (*attachmentMeta, error) {
	var meta attachmentMeta
	if err := mdstore.ReadYAML(path, &meta); err != nil {
		return nil, err
	}
	if meta.ID == "" {
		return nil, fmt.Errorf("attachment metadata not found or empty: %s", path)
	}
	return &meta, nil
}
