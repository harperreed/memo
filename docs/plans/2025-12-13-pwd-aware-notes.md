# PWD-Aware Notes

## Overview

Add directory-aware note filtering so notes can be associated with specific directories. When listing notes, show directory-specific notes first, then global notes.

## Commands

### Adding notes

- `memo add "Title"` - creates note (no directory tag)
- `memo add "Title" --here` - creates note tagged with `dir:<pwd>`

### Listing notes

- `memo list` - sectioned output:
  1. Notes tagged with current pwd (if any)
  2. Up to 10 global notes
  3. "Show more? (y/n)" prompt if more exist
- `memo list --here` - only notes tagged with current pwd
- `memo list --tag work` - existing behavior unchanged

### Tag format

Directory tags use the `dir:` prefix:
- `dir:/Users/harper/projects/memo`

Stored as regular tags in the existing tags table.

## Implementation

### Changes to `cmd/memo/add.go`

- Add `--here` bool flag
- If set, get `os.Getwd()` and add tag `dir:<pwd>`

### Changes to `cmd/memo/list.go`

- Add `--here` bool flag
- Default behavior (no flags):
  1. Query notes with `dir:<pwd>` tag
  2. Query notes WITHOUT any `dir:*` tag (global), limit 10
  3. Print pwd section with header if notes exist
  4. Print global section with header
  5. If more global notes exist, prompt "Show more? (y/n)"
- With `--here`: only query pwd-tagged notes

### Changes to `internal/db/notes.go`

- Add `ListNotesByDirTag(db, dirPath, limit)` - notes with specific dir tag
- Add `ListGlobalNotes(db, limit)` - notes without any `dir:*` tag
- Add `CountGlobalNotes(db)` - count for "show more" logic

### No schema changes

Uses existing tags table with `dir:` prefix convention.

## UI Output

```
📁 /Users/harper/projects/memo
  abc123  Meeting Notes              2h ago   #work
  def456  Project TODO               1d ago   #planning

🌐 Global
  ghi789  Recipe ideas               3d ago   #cooking
  jkl012  Book recommendations       1w ago   #reading
  ...

Show 8 more notes? (y/n)
```

### Formatting rules

- Section headers with emoji icons (📁 for dir, 🌐 for global)
- Reuse existing `FormatNoteListItem` for note rows
- Dim the "Show more?" prompt
- If pwd section is empty, skip it silently (just show global)
- If both empty, show "No notes found."
