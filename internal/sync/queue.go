// ABOUTME: Helper functions for queuing sync changes from CLI commands
// ABOUTME: Provides silent-fail pattern for optional sync integration

package sync

import (
	"context"
	"database/sql"
	"time"

	"suitesync/vault"

	"github.com/google/uuid"
)

// TrySync attempts to sync with the server (push and pull).
// Returns nil if sync is not configured or completes successfully.
// This should be called on read operations to pull remote changes.
func TrySync(ctx context.Context, appDB *sql.DB) error {
	if appDB == nil {
		return nil
	}

	cfg, err := LoadConfig()
	if err != nil {
		return nil
	}

	if !cfg.IsConfigured() {
		return nil
	}

	syncer, err := NewSyncer(cfg, appDB)
	if err != nil {
		return nil
	}
	defer func() { _ = syncer.Close() }()

	return syncer.Sync(ctx)
}

// TryQueueNoteChange attempts to queue a note change for sync.
// Returns nil if sync is not configured or if the change is queued successfully.
// Only returns an error if there's an actual sync failure after configuration.
func TryQueueNoteChange(ctx context.Context, appDB *sql.DB, noteID uuid.UUID, title, content string, tags []string, createdAt, updatedAt time.Time, op vault.Op) error {
	cfg, err := LoadConfig()
	if err != nil {
		// No config or can't load - sync not set up, skip silently
		return nil
	}

	if cfg.DerivedKey == "" {
		// Sync not configured - skip silently
		return nil
	}

	syncer, err := NewSyncer(cfg, appDB)
	if err != nil {
		// Can't create syncer - skip silently (might be temp issue)
		return nil
	}
	defer func() { _ = syncer.Close() }()

	return syncer.QueueNoteChange(ctx, noteID, title, content, tags, createdAt, updatedAt, op)
}

// TryQueueAttachmentChange attempts to queue an attachment change for sync.
// Returns nil if sync is not configured or if the change is queued successfully.
func TryQueueAttachmentChange(ctx context.Context, appDB *sql.DB, attID, noteID uuid.UUID, filename, mimeType string, data []byte, createdAt time.Time, op vault.Op) error {
	cfg, err := LoadConfig()
	if err != nil {
		return nil
	}

	if cfg.DerivedKey == "" {
		return nil
	}

	syncer, err := NewSyncer(cfg, appDB)
	if err != nil {
		return nil
	}
	defer func() { _ = syncer.Close() }()

	return syncer.QueueAttachmentChange(ctx, attID, noteID, filename, mimeType, data, createdAt, op)
}

// TryQueueNoteDelete queues a delete for a note and all its attachments.
func TryQueueNoteDelete(ctx context.Context, appDB *sql.DB, noteID uuid.UUID, attachmentIDs []uuid.UUID) error {
	cfg, err := LoadConfig()
	if err != nil {
		return nil
	}

	if cfg.DerivedKey == "" {
		return nil
	}

	syncer, err := NewSyncer(cfg, appDB)
	if err != nil {
		return nil
	}
	defer func() { _ = syncer.Close() }()

	// Queue attachment deletes first
	for _, attID := range attachmentIDs {
		if err := syncer.QueueAttachmentChange(ctx, attID, noteID, "", "", nil, time.Time{}, vault.OpDelete); err != nil {
			return err
		}
	}

	// Queue note delete
	return syncer.QueueNoteChange(ctx, noteID, "", "", nil, time.Time{}, time.Time{}, vault.OpDelete)
}
