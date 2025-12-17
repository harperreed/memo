// ABOUTME: Tests for applying remote changes to local database
// ABOUTME: Verifies note and attachment change application, including edge cases

package sync

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyNoteChangeUpsert(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	noteID := uuid.New()
	createdAt := time.Now().UTC().Unix()
	updatedAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"title":      "test-note",
		"content":    "test content",
		"tags":       []string{"tag1", "tag2"},
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityNote,
		EntityID: noteID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyNoteChange(ctx, change)
	require.NoError(t, err)

	// Verify note was created
	var title, content string
	var dbCreatedAt, dbUpdatedAt time.Time
	err = appDB.QueryRowContext(ctx,
		`SELECT title, content, created_at, updated_at FROM notes WHERE id = ?`,
		noteID.String()).Scan(&title, &content, &dbCreatedAt, &dbUpdatedAt)
	require.NoError(t, err)
	assert.Equal(t, "test-note", title)
	assert.Equal(t, "test content", content)
	assert.Equal(t, time.Unix(createdAt, 0).Unix(), dbCreatedAt.Unix())
	assert.Equal(t, time.Unix(updatedAt, 0).Unix(), dbUpdatedAt.Unix())

	// Verify tags were created
	var tagCount int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM note_tags WHERE note_id = ?`,
		noteID.String()).Scan(&tagCount)
	require.NoError(t, err)
	assert.Equal(t, 2, tagCount)
}

func TestApplyNoteChangeUpdate(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	noteID := uuid.New()

	// Insert initial note
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "original-title", "original content", time.Now(), time.Now())
	require.NoError(t, err)

	// Apply update
	createdAt := time.Now().UTC().Unix()
	updatedAt := time.Now().UTC().Unix()
	payload := map[string]any{
		"title":      "updated-title",
		"content":    "updated content",
		"tags":       []string{"newtag"},
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityNote,
		EntityID: noteID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyNoteChange(ctx, change)
	require.NoError(t, err)

	// Verify note was updated
	var title, content string
	err = appDB.QueryRowContext(ctx,
		`SELECT title, content FROM notes WHERE id = ?`,
		noteID.String()).Scan(&title, &content)
	require.NoError(t, err)
	assert.Equal(t, "updated-title", title)
	assert.Equal(t, "updated content", content)
}

func TestApplyNoteChangeDelete(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	noteID := uuid.New()

	// Insert note
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	// Apply delete
	change := vault.Change{
		Entity:   EntityNote,
		EntityID: noteID.String(),
		Op:       vault.OpDelete,
		Deleted:  true,
	}

	err = syncer.applyNoteChange(ctx, change)
	require.NoError(t, err)

	// Verify note was deleted
	var count int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notes WHERE id = ?`,
		noteID.String()).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestApplyNoteChangeDeleteWithDeletedFlag(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	noteID := uuid.New()

	// Insert note
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	// Apply delete with Deleted flag
	change := vault.Change{
		Entity:   EntityNote,
		EntityID: noteID.String(),
		Op:       vault.OpUpsert,
		Deleted:  true,
	}

	err = syncer.applyNoteChange(ctx, change)
	require.NoError(t, err)

	// Verify note was deleted
	var count int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notes WHERE id = ?`,
		noteID.String()).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestApplyNoteChangeWithoutTags(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	noteID := uuid.New()
	createdAt := time.Now().UTC().Unix()
	updatedAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"title":      "no-tags-note",
		"content":    "content without tags",
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityNote,
		EntityID: noteID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyNoteChange(ctx, change)
	require.NoError(t, err)

	// Verify note was created without tags
	var title string
	err = appDB.QueryRowContext(ctx,
		`SELECT title FROM notes WHERE id = ?`,
		noteID.String()).Scan(&title)
	require.NoError(t, err)
	assert.Equal(t, "no-tags-note", title)

	// Verify no tags
	var tagCount int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM note_tags WHERE note_id = ?`,
		noteID.String()).Scan(&tagCount)
	require.NoError(t, err)
	assert.Equal(t, 0, tagCount)
}

func TestApplyAttachmentChangeUpsert(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create note first
	noteID := uuid.New()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	attID := uuid.New()
	createdAt := time.Now().UTC().Unix()
	data := []byte("test file data")

	payload := map[string]any{
		"note_id":    noteID.String(),
		"filename":   "test.txt",
		"mime_type":  "text/plain",
		"data":       "dGVzdCBmaWxlIGRhdGE=", // base64 of "test file data"
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityAttachment,
		EntityID: attID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyAttachmentChange(ctx, change)
	require.NoError(t, err)

	// Verify attachment was created
	var filename, mimeType string
	var dbData []byte
	var dbCreatedAt time.Time
	err = appDB.QueryRowContext(ctx,
		`SELECT filename, mime_type, data, created_at FROM attachments WHERE id = ?`,
		attID.String()).Scan(&filename, &mimeType, &dbData, &dbCreatedAt)
	require.NoError(t, err)
	assert.Equal(t, "test.txt", filename)
	assert.Equal(t, "text/plain", mimeType)
	assert.Equal(t, data, dbData)
	assert.Equal(t, time.Unix(createdAt, 0).Unix(), dbCreatedAt.Unix())
}

func TestApplyAttachmentChangeUpdate(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create note
	noteID := uuid.New()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	// Create initial attachment
	attID := uuid.New()
	_, err = appDB.ExecContext(ctx,
		`INSERT INTO attachments (id, note_id, filename, mime_type, data, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		attID.String(), noteID.String(), "old.txt", "text/plain", []byte("old data"), time.Now())
	require.NoError(t, err)

	// Apply update
	createdAt := time.Now().UTC().Unix()
	payload := map[string]any{
		"note_id":    noteID.String(),
		"filename":   "new.txt",
		"mime_type":  "text/plain",
		"data":       "bmV3IGRhdGE=", // base64 of "new data"
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityAttachment,
		EntityID: attID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyAttachmentChange(ctx, change)
	require.NoError(t, err)

	// Verify attachment was updated
	var filename string
	var data []byte
	err = appDB.QueryRowContext(ctx,
		`SELECT filename, data FROM attachments WHERE id = ?`,
		attID.String()).Scan(&filename, &data)
	require.NoError(t, err)
	assert.Equal(t, "new.txt", filename)
	assert.Equal(t, []byte("new data"), data)
}

func TestApplyAttachmentChangeDelete(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create note
	noteID := uuid.New()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	// Create attachment
	attID := uuid.New()
	_, err = appDB.ExecContext(ctx,
		`INSERT INTO attachments (id, note_id, filename, mime_type, data, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		attID.String(), noteID.String(), "test.txt", "text/plain", []byte("data"), time.Now())
	require.NoError(t, err)

	// Apply delete
	change := vault.Change{
		Entity:   EntityAttachment,
		EntityID: attID.String(),
		Op:       vault.OpDelete,
		Deleted:  true,
	}

	err = syncer.applyAttachmentChange(ctx, change)
	require.NoError(t, err)

	// Verify attachment was deleted
	var count int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM attachments WHERE id = ?`,
		attID.String()).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestApplyChangeUnknownEntity(t *testing.T) {
	ctx := context.Background()
	syncer, _, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	change := vault.Change{
		Entity:   "unknown-entity",
		EntityID: uuid.New().String(),
		Op:       vault.OpUpsert,
		Payload:  []byte("{}"),
	}

	// Should not error, just ignore
	err := syncer.applyChange(ctx, change)
	assert.NoError(t, err)
}

func TestApplyChangeNote(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	noteID := uuid.New()
	createdAt := time.Now().UTC().Unix()
	updatedAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"title":      "routed-note",
		"content":    "routed content",
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityNote,
		EntityID: noteID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyChange(ctx, change)
	require.NoError(t, err)

	// Verify note was created
	var title string
	err = appDB.QueryRowContext(ctx,
		`SELECT title FROM notes WHERE id = ?`,
		noteID.String()).Scan(&title)
	require.NoError(t, err)
	assert.Equal(t, "routed-note", title)
}

func TestApplyChangeAttachment(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create note
	noteID := uuid.New()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	attID := uuid.New()
	createdAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"note_id":    noteID.String(),
		"filename":   "routed.txt",
		"mime_type":  "text/plain",
		"data":       "cm91dGVkIGRhdGE=", // base64 of "routed data"
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityAttachment,
		EntityID: attID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyChange(ctx, change)
	require.NoError(t, err)

	// Verify attachment was created
	var filename string
	err = appDB.QueryRowContext(ctx,
		`SELECT filename FROM attachments WHERE id = ?`,
		attID.String()).Scan(&filename)
	require.NoError(t, err)
	assert.Equal(t, "routed.txt", filename)
}

func TestApplyNoteChangeInvalidPayload(t *testing.T) {
	ctx := context.Background()
	syncer, _, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	change := vault.Change{
		Entity:   EntityNote,
		EntityID: uuid.New().String(),
		Op:       vault.OpUpsert,
		Payload:  []byte("invalid json"),
	}

	err := syncer.applyNoteChange(ctx, change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestApplyAttachmentChangeInvalidPayload(t *testing.T) {
	ctx := context.Background()
	syncer, _, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	change := vault.Change{
		Entity:   EntityAttachment,
		EntityID: uuid.New().String(),
		Op:       vault.OpUpsert,
		Payload:  []byte("invalid json"),
	}

	err := syncer.applyAttachmentChange(ctx, change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestApplyNoteChangeInvalidID(t *testing.T) {
	ctx := context.Background()
	syncer, _, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	change := vault.Change{
		Entity:   EntityNote,
		EntityID: "not-a-uuid",
		Op:       vault.OpUpsert,
		Payload:  []byte("{}"),
	}

	err := syncer.applyNoteChange(ctx, change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid note ID")
}

func TestApplyAttachmentChangeInvalidID(t *testing.T) {
	ctx := context.Background()
	syncer, _, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	change := vault.Change{
		Entity:   EntityAttachment,
		EntityID: "not-a-uuid",
		Op:       vault.OpUpsert,
		Payload:  []byte("{}"),
	}

	err := syncer.applyAttachmentChange(ctx, change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid attachment ID")
}

func TestApplyNoteChangeReplaceTags(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	noteID := uuid.New()

	// Insert note with initial tags
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	// Add some initial tags via tag table
	_, err = appDB.ExecContext(ctx, `INSERT INTO tags (name) VALUES (?)`, "oldtag1")
	require.NoError(t, err)
	var tagID int
	err = appDB.QueryRowContext(ctx, `SELECT id FROM tags WHERE name = ?`, "oldtag1").Scan(&tagID)
	require.NoError(t, err)
	_, err = appDB.ExecContext(ctx, `INSERT INTO note_tags (note_id, tag_id) VALUES (?, ?)`, noteID.String(), tagID)
	require.NoError(t, err)

	// Apply update with new tags
	createdAt := time.Now().UTC().Unix()
	updatedAt := time.Now().UTC().Unix()
	payload := map[string]any{
		"title":      "updated-note",
		"content":    "updated content",
		"tags":       []string{"newtag1", "newtag2"},
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityNote,
		EntityID: noteID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyNoteChange(ctx, change)
	require.NoError(t, err)

	// Verify old tags were replaced with new ones
	var tagCount int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM note_tags WHERE note_id = ?`,
		noteID.String()).Scan(&tagCount)
	require.NoError(t, err)
	assert.Equal(t, 2, tagCount)

	// Verify no oldtag1 association
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM note_tags nt JOIN tags t ON nt.tag_id = t.id WHERE nt.note_id = ? AND t.name = ?`,
		noteID.String(), "oldtag1").Scan(&tagCount)
	require.NoError(t, err)
	assert.Equal(t, 0, tagCount)
}

func TestApplyMultipleAttachmentsForSameNote(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create note
	noteID := uuid.New()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	// Add multiple attachments for the same note
	attachments := []struct {
		filename string
		mimeType string
		data     string
	}{
		{"file1.txt", "text/plain", "ZmlsZTE="},
		{"file2.jpg", "image/jpeg", "ZmlsZTI="},
		{"file3.pdf", "application/pdf", "ZmlsZTM="},
	}

	for _, att := range attachments {
		attID := uuid.New()
		createdAt := time.Now().UTC().Unix()

		payload := map[string]any{
			"note_id":    noteID.String(),
			"filename":   att.filename,
			"mime_type":  att.mimeType,
			"data":       att.data,
			"created_at": createdAt,
		}

		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		change := vault.Change{
			Entity:   EntityAttachment,
			EntityID: attID.String(),
			Op:       vault.OpUpsert,
			Payload:  payloadBytes,
		}

		err = syncer.applyAttachmentChange(ctx, change)
		require.NoError(t, err)
	}

	// Verify all attachments were created
	var count int
	err = appDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM attachments WHERE note_id = ?`,
		noteID.String()).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(attachments), count)
}

func TestApplyAttachmentChangeInvalidNoteID(t *testing.T) {
	ctx := context.Background()
	syncer, _, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	attID := uuid.New()
	createdAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"note_id":    "not-a-uuid",
		"filename":   "test.txt",
		"mime_type":  "text/plain",
		"data":       "dGVzdA==",
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityAttachment,
		EntityID: attID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyAttachmentChange(ctx, change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid note ID")
}

func TestApplyAttachmentChangeInvalidBase64(t *testing.T) {
	ctx := context.Background()
	syncer, appDB, cleanup := setupTestSyncerWithDB(t)
	defer cleanup()

	// Create note
	noteID := uuid.New()
	_, err := appDB.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		noteID.String(), "test-note", "test content", time.Now(), time.Now())
	require.NoError(t, err)

	attID := uuid.New()
	createdAt := time.Now().UTC().Unix()

	payload := map[string]any{
		"note_id":    noteID.String(),
		"filename":   "test.txt",
		"mime_type":  "text/plain",
		"data":       "invalid!!!base64",
		"created_at": createdAt,
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	change := vault.Change{
		Entity:   EntityAttachment,
		EntityID: attID.String(),
		Op:       vault.OpUpsert,
		Payload:  payloadBytes,
	}

	err = syncer.applyAttachmentChange(ctx, change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode attachment data")
}
