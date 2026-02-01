# Memo - Charm Removal Plan

## Charmbracelet Dependencies

**Direct:**
- `github.com/charmbracelet/charm` (replaced with 2389-research fork) - KV storage
- `github.com/charmbracelet/glamour v0.10.0` - Markdown terminal rendering

**Indirect (removed when direct deps go):**
- bubbles, bubbletea, colorprofile, keygen, lipgloss, log, x/ansi, x/cellbuf, x/term

## Files Importing Charm

| File | Packages | Purpose |
|------|----------|---------|
| `internal/charm/client.go` | `charm/client`, `charm/kv`, `charm/proto` | KV client wrapper |
| `internal/charm/notes.go` | `charm/kv` | Note CRUD |
| `internal/charm/tags.go` | `charm/kv` | Tag operations |
| `internal/charm/attachments.go` | `charm/kv` | Attachment storage |
| `internal/charm/config.go` | `charm/kv` | Stale threshold |
| `cmd/memo/sync.go` | `charm/kv` | Sync commands |
| `internal/ui/format.go` | `glamour` | Markdown rendering |

## Removal Strategy

### Phase 1: Replace glamour with goldmark

Use `github.com/yuin/goldmark` (already indirect dep) with custom ANSI terminal renderer using `fatih/color`.

Simple terminal rendering:
- Bold for headers
- Code block formatting
- Link display
- List formatting

### Phase 2: Replace Charm KV with SQLite

Use `modernc.org/sqlite` (pure Go). The original design doc already has a SQLite schema!

**Schema (from docs/plans/2025-12-13-memo-design.md):**

```sql
CREATE TABLE notes (
    rowid INTEGER PRIMARY KEY AUTOINCREMENT,  -- Required for FTS5 content_rowid
    id TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE note_tags (
    note_id TEXT REFERENCES notes(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);

CREATE TABLE attachments (
    id TEXT PRIMARY KEY,
    note_id TEXT REFERENCES notes(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    data BLOB NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX idx_notes_id ON notes(id);

-- FTS5 with external content
CREATE VIRTUAL TABLE notes_fts USING fts5(
    title, content, content=notes, content_rowid=rowid
);

-- Triggers to keep FTS in sync
CREATE TRIGGER notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
END;
CREATE TRIGGER notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
END;
CREATE TRIGGER notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
END;
```

### Phase 3: Add YAML Export Format

Extend existing export command:

```bash
memo export --format yaml -o backup.yaml
memo import backup.yaml
```

**YAML Format:**
```yaml
version: "1.0"
exported_at: "2026-01-31T10:00:00Z"
tool: "memo"

notes:
  - id: abc123...
    title: Note Title
    content: |
      Markdown content here
    tags: [tag1, tag2]
    attachments:
      - id: def456...
        filename: image.png
        mime_type: image/png
        data: <base64>
```

### Phase 4: Remove Sync

- Delete `cmd/memo/sync.go`
- Remove sync command from root.go
- Update banner to remove "with Charm"

## Files to Modify

### DELETE:
- `internal/charm/client.go`
- `internal/charm/notes.go`
- `internal/charm/tags.go`
- `internal/charm/attachments.go`
- `internal/charm/config.go`
- `internal/charm/wal_test.go`
- `cmd/memo/sync.go`

### CREATE:
- `internal/storage/db.go` - Store struct, Open/Close, migrations
- `internal/storage/notes.go` - Note CRUD
- `internal/storage/tags.go` - Tag operations with normalization
- `internal/storage/attachments.go` - Attachment CRUD with BLOB
- `internal/storage/search.go` - FTS5 search
- `internal/ui/markdown.go` - goldmark-based terminal renderer

### MODIFY:
- `go.mod` - Remove charm/glamour, add goldmark, yaml.v3
- `cmd/memo/root.go` - Initialize storage.Store instead of charm.Client
- `cmd/memo/add.go` - Use storage.Store
- `cmd/memo/list.go` - Use storage.Store
- `cmd/memo/show.go` - Use storage.Store
- `cmd/memo/edit.go` - Use storage.Store
- `cmd/memo/delete.go` - Use storage.Store
- `cmd/memo/export.go` - Add YAML format
- `cmd/memo/import.go` - Add YAML format
- `internal/ui/format.go` - Replace glamour with goldmark
- `internal/mcp/server.go` - Use storage.Store
- `internal/mcp/tools.go` - Use storage.Store

## Implementation Order

1. Create `internal/storage` package with SQLite implementation
2. Replace glamour with goldmark terminal renderer
3. Migrate command files from charm.Client to storage.Store
4. Update MCP server
5. Remove sync command
6. Add YAML export format
7. Delete `internal/charm/`
8. Update go.mod, run `go mod tidy`
9. Create migration tool: `memo migrate-from-charm`

## Data Path

New: `~/.local/share/memo/memo.db`
