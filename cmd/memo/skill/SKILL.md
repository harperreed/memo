---
name: memo
description: Note-taking and knowledge capture - create, search, and manage notes with tags. Use when the user wants to save information, take notes, or recall saved content.
---

# memo - Notes & Knowledge

Markdown notes with tags, full-text search, and attachments.

## When to use memo

- User wants to save or remember something
- User asks "do I have notes about X?"
- User wants to jot down ideas or information
- User needs to find previously saved content

## Available MCP tools

| Tool | Purpose |
|------|---------|
| `mcp__notes__add_note` | Create a new note |
| `mcp__notes__list_notes` | List all notes |
| `mcp__notes__get_note` | Get note by ID |
| `mcp__notes__search_notes` | Full-text search |
| `mcp__notes__update_note` | Modify a note |
| `mcp__notes__delete_note` | Remove a note |
| `mcp__notes__add_tag` | Tag a note |
| `mcp__notes__remove_tag` | Untag a note |
| `mcp__notes__add_attachment` | Attach a file to a note |
| `mcp__notes__list_attachments` | List note attachments |
| `mcp__notes__get_attachment` | Get attachment content |
| `mcp__notes__export_note` | Export note as JSON or markdown |

## Common patterns

### Create a note
```
mcp__notes__add_note(title="Meeting Notes", content="# Discussion\n\n- Point 1\n- Point 2", tags=["meetings", "work"])
```

### Search notes
```
mcp__notes__search_notes(query="kubernetes deployment")
```

### List by tag
```
mcp__notes__list_notes(tag="work")
```

### Update content
```
mcp__notes__update_note(id="uuid", content="Updated content here")
```

## CLI commands (if MCP unavailable)

```bash
memo add "Note title"        # Opens editor
memo add "Quick note" --content "text"  # Inline content
memo list                    # All notes
memo list --tag work         # By tag
memo list -s "query"         # Full-text search
memo show <id>               # View note
memo edit <id>               # Edit note
memo export --format yaml    # Backup
```

## Data location

`~/.local/share/memo/` (markdown files by default, or `memo.db` for SQLite backend; respects XDG_DATA_HOME)
