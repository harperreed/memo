# memo

A command-line notes tool that stores markdown notes with tags and attachments in SQLite.

## Features

- **Markdown-first**: Notes are stored as markdown with full formatting support
- **Tags**: Organize notes with multiple tags
- **Attachments**: Attach files to notes (stored as blobs in SQLite)
- **Full-text search**: FTS5-powered search across titles and content
- **Beautiful output**: Glamour-rendered markdown in the terminal
- **MCP Server**: Built-in Model Context Protocol server for AI assistant integration
- **Portable**: Single SQLite database file, XDG-compliant storage

## Installation

### Homebrew (macOS)

```bash
brew install harperreed/tap/memo
```

### From source

```bash
go install github.com/harperreed/memo/cmd/memo@latest
```

### Download binary

Download from [GitHub Releases](https://github.com/harperreed/memo/releases).

## Usage

### Add a note

```bash
# Open $EDITOR to write content
memo add "Meeting Notes"

# Inline content
memo add "Quick thought" --content "Remember to call mom"

# From file
memo add "Article Draft" --file draft.md

# With tags
memo add "Project Ideas" --content "..." --tags "work,brainstorm"
```

### List notes

```bash
# List recent notes
memo list

# Filter by tag
memo list --tag work

# Search
memo list --search "meeting"

# Limit results
memo list --limit 5
```

### View a note

```bash
# Use ID prefix (6+ characters)
memo show abc123
```

### Edit a note

```bash
memo edit abc123
```

### Delete a note

```bash
memo rm abc123

# Skip confirmation
memo rm abc123 --force
```

### Manage tags

```bash
# Add tag to note
memo tag add abc123 important

# Remove tag
memo tag rm abc123 important

# List all tags
memo tag list
```

### Attachments

```bash
# Attach a file
memo attach abc123 document.pdf

# List attachments
memo attach abc123 --list

# Extract attachment
memo attach get def456 --output ./downloads/
```

### Export/Import

```bash
# Export all notes to JSON
memo export --format json --output backup.json

# Export to markdown directory
memo export --format md --output ./notes/

# Import from JSON
memo import backup.json

# Import markdown files
memo import ./notes/
```

### MCP Server

Start the MCP server for AI assistant integration:

```bash
memo mcp
```

Add to your Claude desktop config (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "memo": {
      "command": "memo",
      "args": ["mcp"]
    }
  }
}
```

#### Available MCP Tools

| Tool | Description |
|------|-------------|
| `add_note` | Create a new note |
| `list_notes` | List notes with optional filtering |
| `get_note` | Get a note by ID |
| `update_note` | Update note title or content |
| `delete_note` | Delete a note |
| `search_notes` | Full-text search |
| `add_tag` | Add tag to note |
| `remove_tag` | Remove tag from note |
| `add_attachment` | Add attachment (base64) |
| `list_attachments` | List note attachments |
| `get_attachment` | Get attachment content |
| `export_note` | Export note as JSON or markdown |

## Storage

Notes are stored in a SQLite database at:
- **macOS/Linux**: `~/.local/share/memo/memo.db`
- **Custom**: Use `--db /path/to/memo.db`

## Building

```bash
# Build
make build

# Run tests
make test

# Run tests with race detector
make test-race

# Run linter
make lint

# Run all checks
make check
```

## License

MIT
