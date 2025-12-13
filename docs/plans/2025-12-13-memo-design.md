# memo - CLI Notes Tool Design

A CLI notes tool with MCP integration, inspired by toki's architecture.

## Overview

**memo** stores markdown notes with tags and attachments in a single SQLite database. It provides a CLI for CRUD operations, full-text search via FTS5, and an MCP server for AI agent integration.

- **Storage:** `~/.local/share/memo/memo.db` (respects `XDG_DATA_HOME`)
- **Output:** glamour for markdown rendering, fatih/color for status
- **Search:** SQLite FTS5 full-text search with ranking

## Data Model

```sql
CREATE TABLE notes (
    id TEXT PRIMARY KEY,           -- UUID
    title TEXT NOT NULL,
    content TEXT NOT NULL,         -- Markdown
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
    id TEXT PRIMARY KEY,           -- UUID
    note_id TEXT REFERENCES notes(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,        -- Original filename
    mime_type TEXT NOT NULL,
    data BLOB NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE VIRTUAL TABLE notes_fts USING fts5(
    title, content, content='notes', content_rowid='rowid'
);
```

## CLI Commands

```
memo add "Title" [flags]
  --tags, -t     Comma-separated tags
  --file, -f     Read content from file (otherwise opens $EDITOR)
  --content, -c  Inline content (for piping)

memo list [flags]
  --tag, -t      Filter by tag
  --search, -s   FTS5 search query
  --limit, -n    Number of results (default 20)

memo show <id-prefix>
  # Renders note with glamour, shows attachments list

memo edit <id-prefix>
  # Opens $EDITOR with current content, saves on exit

memo rm <id-prefix>
  # Deletes note and all attachments (with confirmation)

memo tag add <id-prefix> <tag>
memo tag rm <id-prefix> <tag>
memo tag list
  # Lists all tags with note counts

memo attach <id-prefix> <file>
  # Adds attachment to note

memo attach get <id-prefix> <attachment-id> [--output path]
  # Extracts attachment to file

memo export [flags]
  --format, -f   json|md (default: json)
  --output, -o   Output path (default: stdout for json, ./export/ for md)
  --note, -n     Single note ID (otherwise exports all)

memo import <file>
  # Imports from JSON or directory of markdown files

memo mcp
  # Starts MCP server on stdio
```

**UUID prefix matching:** 6+ character prefix required, error message lists matches on ambiguity.

**$EDITOR flow:** For `add` without `--file`/`--content` and for `edit`, opens temp file in `$EDITOR`, reads content on save/exit.

## MCP Server

### Tools (12)

| Tool | Description |
|------|-------------|
| `add_note` | Create note with title, content, optional tags |
| `list_notes` | List notes with optional tag/search filter |
| `get_note` | Get single note by ID (full content + tags + attachments list) |
| `update_note` | Update title, content |
| `delete_note` | Remove note and attachments |
| `search_notes` | FTS5 full-text search with ranking |
| `add_tag` | Add tag to note |
| `remove_tag` | Remove tag from note |
| `add_attachment` | Add attachment (base64 encoded) |
| `list_attachments` | List attachments for a note |
| `get_attachment` | Get attachment content (base64) |
| `export_note` | Export single note as JSON or markdown |

### Resources (4)

| URI | Description |
|-----|-------------|
| `memo://notes` | All notes (metadata only) |
| `memo://notes/recent` | Last 10 notes |
| `memo://tags` | All tags with counts |
| `memo://note/{id}` | Single note with full content |

### Prompts (3)

| Prompt | Purpose |
|--------|---------|
| `capture-notes` | Guide for capturing meeting notes, research, ideas |
| `daily-journal` | Template for daily journaling workflow |
| `organize-notes` | Review and tag/clean up existing notes |

## Project Structure

```
memo/
├── cmd/memo/
│   ├── main.go
│   ├── root.go
│   ├── add.go
│   ├── list.go
│   ├── show.go
│   ├── edit.go
│   ├── rm.go
│   ├── tag.go
│   ├── attach.go
│   ├── export.go
│   ├── import.go
│   └── mcp.go
├── internal/
│   ├── db/
│   │   ├── db.go
│   │   ├── notes.go
│   │   ├── tags.go
│   │   ├── attachments.go
│   │   └── search.go
│   ├── models/
│   │   ├── note.go
│   │   ├── tag.go
│   │   └── attachment.go
│   ├── mcp/
│   │   ├── server.go
│   │   ├── tools.go
│   │   ├── resources.go
│   │   └── prompts.go
│   └── ui/
│       └── format.go
├── test/
│   └── integration_test.go
├── docs/plans/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Tech Stack

- Go 1.24+
- github.com/spf13/cobra - CLI framework
- modernc.org/sqlite - Pure Go SQLite driver
- github.com/google/uuid - UUID generation
- github.com/charmbracelet/glamour - Markdown rendering
- github.com/fatih/color - Terminal colors
- github.com/modelcontextprotocol/go-sdk - MCP server

## Import/Export Formats

**JSON export:** Full fidelity backup including attachments as base64, exact timestamps.

**Markdown export:** One `.md` file per note with YAML frontmatter:
```yaml
---
id: abc123...
title: Note Title
tags: [tag1, tag2]
created: 2025-12-13T10:00:00Z
updated: 2025-12-13T10:00:00Z
---

Note content here...
```

Attachments exported to `attachments/<note-id>/` subdirectory.
