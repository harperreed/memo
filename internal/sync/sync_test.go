// ABOUTME: Tests for vault sync integration
// ABOUTME: Verifies change queuing, syncing, and pending count tracking

package sync

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyncer(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test app database
	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	// Create seed and derive key
	seed, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	require.NotNil(t, syncer)
	defer func() { _ = syncer.Close() }()

	assert.Equal(t, cfg, syncer.config)
	assert.NotNil(t, syncer.store)
	assert.NotNil(t, syncer.client)
	assert.NotNil(t, syncer.keys)

	// Verify keys were derived correctly
	expectedKeys, err := vault.DeriveKeys(seed, "", vault.DefaultKDFParams())
	require.NoError(t, err)
	assert.Equal(t, expectedKeys.EncKey, syncer.keys.EncKey)
}

func TestNewSyncerNoDerivedKey(t *testing.T) {
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	cfg := &Config{
		Server:   "https://test.example.com",
		DeviceID: "test-device",
		VaultDB:  filepath.Join(tmpDir, "vault.db"),
	}

	_, err := NewSyncer(cfg, appDB)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "derived key not configured")
}

func TestNewSyncerInvalidDerivedKey(t *testing.T) {
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: "invalid-key-format",
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	_, err := NewSyncer(cfg, appDB)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid derived key")
}

func TestQueueNoteChange(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	noteID := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	// Queue note create
	err := syncer.QueueNoteChange(ctx, noteID, "Test Note", "Test content", []string{"tag1"}, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueNoteChangeWithTags(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	noteID := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()
	tags := []string{"work", "important", "todo"}

	// Queue note with multiple tags
	err := syncer.QueueNoteChange(ctx, noteID, "Tagged Note", "Content", tags, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueNoteChangeDelete(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	noteID := uuid.New()

	// Queue note delete
	err := syncer.QueueNoteChange(ctx, noteID, "", "", nil, time.Time{}, time.Time{}, vault.OpDelete)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueAttachmentChange(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	attID := uuid.New()
	noteID := uuid.New()
	createdAt := time.Now().UTC()
	data := []byte("test file data")

	// Queue attachment create
	err := syncer.QueueAttachmentChange(ctx, attID, noteID, "test.txt", "text/plain", data, createdAt, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueAttachmentChangeDelete(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	attID := uuid.New()
	noteID := uuid.New()

	// Queue attachment delete
	err := syncer.QueueAttachmentChange(ctx, attID, noteID, "", "", nil, time.Time{}, vault.OpDelete)
	require.NoError(t, err)

	// Verify change was queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPendingCount(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	// Initially zero
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Queue multiple changes
	noteID1 := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()
	err = syncer.QueueNoteChange(ctx, noteID1, "Note 1", "Content 1", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	noteID2 := uuid.New()
	err = syncer.QueueNoteChange(ctx, noteID2, "Note 2", "Content 2", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	// Verify count
	count, err = syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestPendingCountDoesNotConsumeItems(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	// Queue a change
	noteID := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()
	err := syncer.QueueNoteChange(ctx, noteID, "Test", "Content", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	// Call PendingCount multiple times
	count1, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count1)

	count2, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count2)

	count3, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count3)

	// All calls should return the same count - not consuming items
	assert.Equal(t, count1, count2)
	assert.Equal(t, count2, count3)
}

func TestMultipleChanges(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	// Create multiple notes
	for i := 0; i < 5; i++ {
		noteID := uuid.New()
		err := syncer.QueueNoteChange(ctx, noteID, "Note", "Content", nil, createdAt, updatedAt, vault.OpUpsert)
		require.NoError(t, err)
	}

	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestAutoSyncDisabled(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	// vaultSyncer.CanSync() returns false in test setup (no Server/Token/UserID)
	assert.False(t, syncer.vaultSyncer.CanSync())

	noteID := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()
	err := syncer.QueueNoteChange(ctx, noteID, "Test", "Content", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	// Change should be queued but not synced
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSyncNotConfigured(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	// Create syncer with missing server config
	cfg := &Config{
		Server:     "", // Empty server
		UserID:     "",
		Token:      "",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	defer func() { _ = syncer.Close() }()

	// Sync should fail with helpful error
	err = syncer.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync not configured")
}

func TestCanSync(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "fully configured",
			config: &Config{
				Server: "https://example.com",
				Token:  "token",
				UserID: "user",
			},
			expected: true,
		},
		{
			name: "missing server",
			config: &Config{
				Server: "",
				Token:  "token",
				UserID: "user",
			},
			expected: false,
		},
		{
			name: "missing token",
			config: &Config{
				Server: "https://example.com",
				Token:  "",
				UserID: "user",
			},
			expected: false,
		},
		{
			name: "missing user id",
			config: &Config{
				Server: "https://example.com",
				Token:  "token",
				UserID: "",
			},
			expected: false,
		},
		{
			name: "all missing",
			config: &Config{
				Server: "",
				Token:  "",
				UserID: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			appDB := setupTestDB(t, tmpDir)
			defer func() { _ = appDB.Close() }()

			_, phrase, err := vault.NewSeedPhrase()
			require.NoError(t, err)

			tt.config.DerivedKey = phrase
			tt.config.DeviceID = "test-device"
			tt.config.VaultDB = filepath.Join(tmpDir, "vault.db")

			syncer, err := NewSyncer(tt.config, appDB)
			require.NoError(t, err)
			defer func() { _ = syncer.Close() }()

			assert.Equal(t, tt.expected, syncer.vaultSyncer.CanSync())
		})
	}
}

func TestPendingChanges(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	// Queue some changes
	noteID1 := uuid.New()
	err := syncer.QueueNoteChange(ctx, noteID1, "Note 1", "Content 1", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	noteID2 := uuid.New()
	err = syncer.QueueNoteChange(ctx, noteID2, "Note 2", "Content 2", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	// Get pending changes
	changes, err := syncer.PendingChanges(ctx)
	require.NoError(t, err)
	require.Len(t, changes, 2)

	// Verify structure
	for _, change := range changes {
		assert.NotEmpty(t, change.ChangeID)
		assert.True(t, strings.HasSuffix(change.Entity, EntityNote), "entity should end with %s, got %s", EntityNote, change.Entity)
		assert.False(t, change.TS.IsZero())
	}
}

func TestLastSyncedSeq(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	// Initially should be "0"
	seq, err := syncer.LastSyncedSeq(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0", seq)
}

func TestCloseNilStore(t *testing.T) {
	syncer := &Syncer{
		store: nil,
	}

	err := syncer.Close()
	assert.NoError(t, err)
}

func TestQueueChangeEncryption(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	noteID := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()
	err := syncer.QueueNoteChange(ctx, noteID, "Encrypted Note", "Secret content", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	// Verify change was encrypted (indirectly by checking it was queued)
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestQueueMixedChanges(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	// Queue notes
	noteID1 := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()
	err := syncer.QueueNoteChange(ctx, noteID1, "Note", "Content", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	// Queue attachments
	attID := uuid.New()
	noteID2 := uuid.New()
	data := []byte("file data")
	err = syncer.QueueAttachmentChange(ctx, attID, noteID2, "file.txt", "text/plain", data, createdAt, vault.OpUpsert)
	require.NoError(t, err)

	// Verify both were queued
	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestQueueLargeAttachment(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	attID := uuid.New()
	noteID := uuid.New()
	createdAt := time.Now().UTC()

	// Create a large attachment (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := syncer.QueueAttachmentChange(ctx, attID, noteID, "large.bin", "application/octet-stream", largeData, createdAt, vault.OpUpsert)
	require.NoError(t, err)

	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestNewSyncerWithTokenRefresh(t *testing.T) {
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:       "https://test.example.com",
		UserID:       "test-user",
		Token:        "test-token",
		RefreshToken: "test-refresh",
		TokenExpires: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		DerivedKey:   phrase,
		DeviceID:     "test-device",
		VaultDB:      filepath.Join(tmpDir, "vault.db"),
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	require.NotNil(t, syncer)
	defer func() { _ = syncer.Close() }()

	assert.NotNil(t, syncer.client)
}

func TestNewSyncerWithInvalidTokenExpires(t *testing.T) {
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:       "https://test.example.com",
		UserID:       "test-user",
		Token:        "test-token",
		TokenExpires: "invalid-date-format",
		DerivedKey:   phrase,
		DeviceID:     "test-device",
		VaultDB:      filepath.Join(tmpDir, "vault.db"),
	}

	// Should still create syncer, just with zero time for expires
	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	require.NotNil(t, syncer)
	defer func() { _ = syncer.Close() }()

	assert.NotNil(t, syncer.client)
}

func TestQueueChangeWithoutSync(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	// Create syncer without server config (canSync returns false)
	cfg := &Config{
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	defer func() { _ = syncer.Close() }()

	noteID := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	// Queue should succeed and not attempt sync (no server)
	err = syncer.QueueNoteChange(ctx, noteID, "Test", "Content", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	count, err := syncer.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSyncWithEvents(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	appDB := setupTestDB(t, tmpDir)
	defer func() { _ = appDB.Close() }()

	_, phrase, err := vault.NewSeedPhrase()
	require.NoError(t, err)

	cfg := &Config{
		Server:     "",
		UserID:     "",
		Token:      "",
		DerivedKey: phrase,
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
	}

	syncer, err := NewSyncer(cfg, appDB)
	require.NoError(t, err)
	defer func() { _ = syncer.Close() }()

	// Sync with events should fail with proper error (no server configured)
	err = syncer.SyncWithEvents(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync not configured")
}

func TestPendingChangesMultipleEntities(t *testing.T) {
	ctx := context.Background()
	syncer := setupTestSyncer(t)
	defer func() { _ = syncer.Close() }()

	createdAt := time.Now().UTC()
	updatedAt := time.Now().UTC()

	// Queue different entity types
	noteID := uuid.New()
	err := syncer.QueueNoteChange(ctx, noteID, "Note", "Content", nil, createdAt, updatedAt, vault.OpUpsert)
	require.NoError(t, err)

	attID := uuid.New()
	data := []byte("data")
	err = syncer.QueueAttachmentChange(ctx, attID, noteID, "file.txt", "text/plain", data, createdAt, vault.OpUpsert)
	require.NoError(t, err)

	// Get pending changes
	changes, err := syncer.PendingChanges(ctx)
	require.NoError(t, err)
	require.Len(t, changes, 2)

	// Verify we have both entity types (entities now prefixed with AppID)
	hasNote := false
	hasAttachment := false
	for _, change := range changes {
		if strings.HasSuffix(change.Entity, EntityNote) {
			hasNote = true
		}
		if strings.HasSuffix(change.Entity, EntityAttachment) {
			hasAttachment = true
		}
	}
	assert.True(t, hasNote, "should have a note entity")
	assert.True(t, hasAttachment, "should have an attachment entity")
}
