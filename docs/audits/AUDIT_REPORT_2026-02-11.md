# Documentation Audit Report

Generated: 2026-02-11 | Commit: 70a0685

## Executive Summary

| Metric | Count |
|--------|-------|
| Documents scanned | 2 (README.md, SKILL.md) |
| Claims verified | ~30 |
| Verified TRUE | 18 (60%) |
| **Verified FALSE** | **10 (33%)** |
| Partially false / outdated | 2 (7%) |

---

## False Claims Requiring Fixes

### README.md

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| 3 | "stores markdown notes with tags and attachments in SQLite" | New users default to **markdown backend**, not SQLite. SQLite is only preserved for existing users. | Rewrite to mention both backends |
| 11 | "Glamour-rendered markdown in the terminal" | Uses **goldmark** with a custom `TerminalRenderer`, not glamour. Glamour is not a dependency. | Change "Glamour-rendered" to "Rendered markdown" or "Goldmark-rendered" |
| 27 | `go install github.com/harperreed/memo/cmd/memo@latest` | `go.mod` declares module as `github.com/harper/memo`. Install path won't resolve. | Change to `github.com/harper/memo/cmd/memo@latest` **or** fix `go.mod` to match |
| 31 | `https://github.com/harperreed/memo/releases` | Module path mismatch with `go.mod` (`github.com/harper/memo`). Repo URL may be correct but Go install path is wrong. | Reconcile module path and repo URL |
| 109 | `memo attach abc123 --list` | No `--list` flag exists on the `attach` command. Listing attachments is only available via MCP tool. | Remove this example or implement the `--list` flag |
| 173 | "Custom: Use `--db /path/to/memo.db`" | No `--db` flag exists anywhere in the codebase. Storage location is controlled by config file (`~/.config/memo/config.json` `data_dir` field). | Remove `--db` reference, document config file approach |
| 171 | Storage path: `~/.local/share/memo/memo.db` | Only true for SQLite backend. New users default to markdown backend where data is stored as files in `~/.local/share/memo/`. | Mention both backends and their data locations |

### SKILL.md (cmd/memo/skill/SKILL.md)

| Line | Claim | Reality | Fix |
|------|-------|---------|-----|
| 21 | MCP tool `mcp__notes__create_note` | Actual tool name is `add_note`, not `create_note` | Change to `mcp__notes__add_note` |
| 29 | MCP tool `mcp__notes__list_tags` | **Does not exist** as an MCP tool. The 12 registered tools do not include `list_tags`. | Remove from table or implement the tool |
| 57 | `memo add -m "Quick note"` | No `-m` flag exists. The correct flag is `--content` (no shorthand). | Change to `memo add "Quick note" --content "text"` |
| 60 | `memo search "query"` | No standalone `search` subcommand exists. Correct: `memo list --search "query"` or `memo list -s "query"` | Change to `memo list -s "query"` |
| 68 | Data location: `~/.local/share/memo/memo.db` (SQLite) | New users default to markdown backend, not SQLite | Mention both backends |

---

## Undocumented Features (Gap Detection)

### Features present in code but missing from both README and SKILL.md

| Feature | Location | Description |
|---------|----------|-------------|
| `--here` flag on `add` | `cmd/memo/add.go:131` | Tags note with current working directory |
| `--here` flag on `list` | `cmd/memo/list.go:229` | Shows only notes tagged with current directory |
| `migrate` command | `cmd/memo/migrate.go` | Migrate data between SQLite and markdown backends |
| `install-skill` command | `cmd/memo/skill.go` | Install Claude Code skill definition |
| `version` command | `cmd/memo/version.go` | Display version info |
| Config file | `internal/config/config.go` | `~/.config/memo/config.json` controls backend + data_dir |
| Markdown backend | `internal/storage/markdown*.go` | Full alternative to SQLite using mdstore |
| MCP Prompts (5) | `internal/mcp/prompts.go` | meeting-notes, journal, summarize, organize, project-planning |
| MCP Resources | `internal/mcp/resources.go` | `memo://note/{id}` resource template |
| Directory-aware listing | `cmd/memo/list.go` | Sectioned output showing pwd-relevant notes first |

### MCP tools present in code but missing from SKILL.md

| Tool | Purpose |
|------|---------|
| `add_attachment` | Add file attachment to a note (base64) |
| `list_attachments` | List attachments for a note |
| `get_attachment` | Get attachment content |
| `export_note` | Export note as JSON or markdown |

(These 4 tools ARE documented in README.md's MCP table, just not in SKILL.md)

---

## CI/Build Discrepancies

| Item | Documented/Configured | Reality | Impact |
|------|----------------------|---------|--------|
| CI Go versions | Matrix: 1.21, 1.22, 1.23 | `go.mod` requires Go 1.24.11 | CI may fail or be untested on the required Go version |
| CI lint Go version | 1.23 | `go.mod` requires 1.24.11 | Linting runs on wrong Go version |
| goreleaser homepage | `https://github.com/harperreed/memo` | `go.mod`: `github.com/harper/memo` | Module path mismatch |

---

## Pattern Summary

| Pattern | Count | Root Cause |
|---------|-------|------------|
| Stale SQLite-only references | 4 | Markdown backend added but docs not fully updated |
| Nonexistent CLI flags/commands | 3 | Features documented that were never implemented or were removed |
| Wrong MCP tool names | 2 | SKILL.md uses different naming convention than actual registration |
| Missing MCP tools in SKILL.md | 4 | SKILL.md not updated when tools were added |
| Module path mismatch | 3 | `go.mod` and README/goreleaser disagree on module path |
| CI version drift | 2 | `go.mod` bumped to 1.24.11 but CI not updated |

---

## Human Review Queue

- [ ] **Module path**: Decide whether the canonical path is `github.com/harper/memo` or `github.com/harperreed/memo` and reconcile `go.mod`, README, and goreleaser
- [ ] **Attachment --list**: Decide whether to implement `memo attach <id> --list` or remove from docs
- [ ] **list_tags MCP tool**: Decide whether to implement or remove from SKILL.md
- [ ] **CI Go versions**: Update CI matrix to include Go 1.24
- [ ] **--here feature**: Decide if directory-aware features should be documented in README
- [ ] **Markdown backend**: Document the dual-backend story in README
