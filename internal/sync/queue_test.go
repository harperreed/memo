// ABOUTME: Tests for sync queue helper functions
// ABOUTME: Verifies silent-fail behavior when sync is not configured

package sync

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
)

func TestTrySync_NoConfig(t *testing.T) {
	// When no config exists, TrySync should return nil (not error)
	ctx := context.Background()

	err := TrySync(ctx, nil)

	if err != nil {
		t.Errorf("TrySync should not error when config doesn't exist, got %v", err)
	}
}

func TestTryQueueNoteChange_NoConfig(t *testing.T) {
	// When no config exists, TryQueueNoteChange should return nil (not error)
	// This tests the silent-fail behavior for users without sync configured

	ctx := context.Background()
	noteID := uuid.New()

	err := TryQueueNoteChange(
		ctx,
		nil, // db connection not needed when config doesn't exist
		noteID,
		"Test Note",
		"Test content",
		[]string{"tag1", "tag2"},
		time.Now(),
		time.Now(),
		vault.OpUpsert,
	)

	if err != nil {
		t.Errorf("TryQueueNoteChange should not error when config doesn't exist, got %v", err)
	}
}

func TestTryQueueNoteChange_WithConfig(t *testing.T) {
	// Test queuing with valid config (but no server to avoid sync)
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	_, phrase, err := vault.NewSeedPhrase()
	if err != nil {
		t.Fatalf("NewSeedPhrase failed: %v", err)
	}

	cfg := &Config{
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	err = SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	ctx := context.Background()
	noteID := uuid.New()

	err = TryQueueNoteChange(
		ctx,
		appDB,
		noteID,
		"Test Note",
		"Test content",
		[]string{"tag1"},
		time.Now(),
		time.Now(),
		vault.OpUpsert,
	)

	if err != nil {
		t.Errorf("TryQueueNoteChange should not error with valid config, got %v", err)
	}
}

func TestTryQueueAttachmentChange_NoConfig(t *testing.T) {
	// When no config exists, TryQueueAttachmentChange should return nil

	ctx := context.Background()
	attID := uuid.New()
	noteID := uuid.New()

	err := TryQueueAttachmentChange(
		ctx,
		nil,
		attID,
		noteID,
		"test.txt",
		"text/plain",
		[]byte("test data"),
		time.Now(),
		vault.OpUpsert,
	)

	if err != nil {
		t.Errorf("TryQueueAttachmentChange should not error when config doesn't exist, got %v", err)
	}
}

func TestTryQueueNoteDelete_NoConfig(t *testing.T) {
	// When no config exists, TryQueueNoteDelete should return nil

	ctx := context.Background()
	noteID := uuid.New()
	attachmentIDs := []uuid.UUID{uuid.New(), uuid.New()}

	err := TryQueueNoteDelete(ctx, nil, noteID, attachmentIDs)

	if err != nil {
		t.Errorf("TryQueueNoteDelete should not error when config doesn't exist, got %v", err)
	}
}
