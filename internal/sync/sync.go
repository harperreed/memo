// ABOUTME: Vault sync integration for memo
// ABOUTME: Handles change queuing, syncing, and applying remote changes

package sync

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"suitesync/vault"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/db"
)

const (
	// MemoAppID is the unique application identifier for memo sync.
	// This UUID isolates memo's data from other apps using the same vault infrastructure.
	MemoAppID = "c7a3f8d1-5e2b-4a9c-b6d4-e8f912345678"

	EntityNote       = "note"
	EntityAttachment = "attachment"
)

// Syncer manages vault sync for memo data.
type Syncer struct {
	config *Config
	store  *vault.Store
	keys   vault.Keys
	client *vault.Client
	appDB  *sql.DB
}

// NewSyncer creates a new syncer from config.
func NewSyncer(cfg *Config, appDB *sql.DB) (*Syncer, error) {
	if cfg.DerivedKey == "" {
		return nil, errors.New("derived key not configured - run 'memo sync login' first")
	}

	// DerivedKey is stored as hex-encoded seed
	seed, err := vault.ParseSeedPhrase(cfg.DerivedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid derived key: %w", err)
	}

	keys, err := vault.DeriveKeys(seed, "", vault.DefaultKDFParams())
	if err != nil {
		return nil, fmt.Errorf("derive keys: %w", err)
	}

	store, err := vault.OpenStore(cfg.VaultDB)
	if err != nil {
		return nil, fmt.Errorf("open vault store: %w", err)
	}

	client := vault.NewClient(vault.SyncConfig{
		AppID:     MemoAppID,
		BaseURL:   cfg.Server,
		DeviceID:  cfg.DeviceID,
		AuthToken: cfg.Token,
	})

	return &Syncer{
		config: cfg,
		store:  store,
		keys:   keys,
		client: client,
		appDB:  appDB,
	}, nil
}

// Close releases syncer resources.
func (s *Syncer) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// NotePayload represents a note's sync payload including tags.
type NotePayload struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags,omitempty"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

// QueueNoteChange queues a change for a note.
func (s *Syncer) QueueNoteChange(ctx context.Context, noteID uuid.UUID, title, content string, tags []string, createdAt, updatedAt time.Time, op vault.Op) error {
	var payload map[string]any
	if op != vault.OpDelete {
		payload = map[string]any{
			"title":      title,
			"content":    content,
			"tags":       tags,
			"created_at": createdAt.UTC().Unix(),
			"updated_at": updatedAt.UTC().Unix(),
		}
	}

	return s.queueChange(ctx, EntityNote, noteID.String(), op, payload)
}

// AttachmentPayload represents an attachment's sync payload.
type AttachmentPayload struct {
	NoteID    string `json:"note_id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	Data      string `json:"data"` // base64-encoded
	CreatedAt int64  `json:"created_at"`
}

// QueueAttachmentChange queues a change for an attachment.
func (s *Syncer) QueueAttachmentChange(ctx context.Context, attID, noteID uuid.UUID, filename, mimeType string, data []byte, createdAt time.Time, op vault.Op) error {
	var payload map[string]any
	if op != vault.OpDelete {
		payload = map[string]any{
			"note_id":    noteID.String(),
			"filename":   filename,
			"mime_type":  mimeType,
			"data":       base64.StdEncoding.EncodeToString(data),
			"created_at": createdAt.UTC().Unix(),
		}
	}

	return s.queueChange(ctx, EntityAttachment, attID.String(), op, payload)
}

func (s *Syncer) queueChange(ctx context.Context, entity, entityID string, op vault.Op, payload map[string]any) error {
	change, err := vault.NewChange(entity, entityID, op, payload)
	if err != nil {
		return fmt.Errorf("create change: %w", err)
	}
	if op == vault.OpDelete {
		change.Deleted = true
	}

	plain, err := json.Marshal(change)
	if err != nil {
		return fmt.Errorf("marshal change: %w", err)
	}

	aad := change.AAD(s.config.UserID, s.config.DeviceID)
	env, err := vault.Encrypt(s.keys.EncKey, plain, aad)
	if err != nil {
		return fmt.Errorf("encrypt change: %w", err)
	}

	if err := s.store.EnqueueEncryptedChange(ctx, change, s.config.UserID, s.config.DeviceID, env); err != nil {
		return fmt.Errorf("enqueue change: %w", err)
	}

	// Auto-sync if enabled
	if s.config.AutoSync && s.canSync() {
		return s.Sync(ctx)
	}

	return nil
}

func (s *Syncer) canSync() bool {
	return s.config.Server != "" && s.config.Token != "" && s.config.UserID != ""
}

// Sync pushes local changes and pulls remote changes.
func (s *Syncer) Sync(ctx context.Context) error {
	return s.SyncWithEvents(ctx, nil)
}

// SyncWithEvents pushes local changes and pulls remote changes with progress callbacks.
func (s *Syncer) SyncWithEvents(ctx context.Context, events *vault.SyncEvents) error {
	if !s.canSync() {
		return errors.New("sync not configured - run 'memo sync login' first")
	}

	return vault.Sync(ctx, s.store, s.client, s.keys, s.config.UserID, s.applyChange, events)
}

// applyChange applies a remote change to the local database.
func (s *Syncer) applyChange(ctx context.Context, c vault.Change) error {
	switch c.Entity {
	case EntityNote:
		return s.applyNoteChange(ctx, c)
	case EntityAttachment:
		return s.applyAttachmentChange(ctx, c)
	default:
		// Ignore unknown entities
		return nil
	}
}

func (s *Syncer) applyNoteChange(ctx context.Context, c vault.Change) error {
	noteID, err := uuid.Parse(c.EntityID)
	if err != nil {
		return fmt.Errorf("invalid note ID: %w", err)
	}

	if c.Op == vault.OpDelete || c.Deleted {
		return db.DeleteNote(s.appDB, noteID)
	}

	var payload NotePayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal note payload: %w", err)
	}

	createdAt := time.Unix(payload.CreatedAt, 0)
	updatedAt := time.Unix(payload.UpdatedAt, 0)

	// Upsert note
	_, err = s.appDB.ExecContext(ctx, `
		INSERT INTO notes (id, title, content, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			content = excluded.content,
			updated_at = excluded.updated_at
	`, noteID.String(), payload.Title, payload.Content, createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("upsert note: %w", err)
	}

	// Sync tags: remove old, add new
	_, err = s.appDB.ExecContext(ctx, `DELETE FROM note_tags WHERE note_id = ?`, noteID.String())
	if err != nil {
		return fmt.Errorf("clear note tags: %w", err)
	}

	for _, tagName := range payload.Tags {
		if err := db.AddTagToNote(s.appDB, noteID, tagName); err != nil {
			return fmt.Errorf("add tag %s: %w", tagName, err)
		}
	}

	return nil
}

func (s *Syncer) applyAttachmentChange(ctx context.Context, c vault.Change) error {
	attID, err := uuid.Parse(c.EntityID)
	if err != nil {
		return fmt.Errorf("invalid attachment ID: %w", err)
	}

	if c.Op == vault.OpDelete || c.Deleted {
		return db.DeleteAttachment(s.appDB, attID)
	}

	var payload AttachmentPayload
	if err := json.Unmarshal(c.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal attachment payload: %w", err)
	}

	noteID, err := uuid.Parse(payload.NoteID)
	if err != nil {
		return fmt.Errorf("invalid note ID in attachment: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return fmt.Errorf("decode attachment data: %w", err)
	}

	createdAt := time.Unix(payload.CreatedAt, 0)

	// Upsert attachment
	_, err = s.appDB.ExecContext(ctx, `
		INSERT INTO attachments (id, note_id, filename, mime_type, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			filename = excluded.filename,
			mime_type = excluded.mime_type,
			data = excluded.data
	`, attID.String(), noteID.String(), payload.Filename, payload.MimeType, data, createdAt)

	return err
}

// PendingCount returns the number of changes waiting to be synced.
func (s *Syncer) PendingCount(ctx context.Context) (int, error) {
	batch, err := s.store.DequeueBatch(ctx, 1000)
	if err != nil {
		return 0, err
	}
	return len(batch), nil
}

// PendingItem represents a change waiting to be synced.
type PendingItem struct {
	ChangeID string
	Entity   string
	TS       time.Time
}

// PendingChanges returns details of changes waiting to be synced.
func (s *Syncer) PendingChanges(ctx context.Context) ([]PendingItem, error) {
	batch, err := s.store.DequeueBatch(ctx, 100)
	if err != nil {
		return nil, err
	}

	items := make([]PendingItem, len(batch))
	for i, b := range batch {
		items[i] = PendingItem{
			ChangeID: b.ChangeID,
			Entity:   b.Entity,
			TS:       time.Unix(b.TS, 0),
		}
	}
	return items, nil
}

// LastSyncedSeq returns the last pulled sequence number.
func (s *Syncer) LastSyncedSeq(ctx context.Context) (string, error) {
	return s.store.GetState(ctx, "last_pulled_seq", "0")
}
