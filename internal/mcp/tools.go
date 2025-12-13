// ABOUTME: MCP tools for note CRUD operations.
// ABOUTME: Maps CLI functionality to MCP tool interface.

package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerTools() {
	// add_note
	s.server.AddTool(&mcp.Tool{
		Name:        "add_note",
		Description: "Create a new note with title and content",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"title": {"type": "string", "description": "Note title"},
				"content": {"type": "string", "description": "Note content (markdown)"},
				"tags": {"type": "array", "items": {"type": "string"}, "description": "Optional tags"}
			},
			"required": ["title", "content"]
		}`),
	}, s.handleAddNote)

	// list_notes
	s.server.AddTool(&mcp.Tool{
		Name:        "list_notes",
		Description: "List notes with optional filtering",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"tag": {"type": "string", "description": "Filter by tag"},
				"limit": {"type": "integer", "description": "Max results", "default": 20}
			}
		}`),
	}, s.handleListNotes)

	// get_note
	s.server.AddTool(&mcp.Tool{
		Name:        "get_note",
		Description: "Get a note by ID prefix",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Note ID or prefix (6+ chars)"}
			},
			"required": ["id"]
		}`),
	}, s.handleGetNote)

	// update_note
	s.server.AddTool(&mcp.Tool{
		Name:        "update_note",
		Description: "Update a note's title or content",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Note ID or prefix"},
				"title": {"type": "string", "description": "New title"},
				"content": {"type": "string", "description": "New content"}
			},
			"required": ["id"]
		}`),
	}, s.handleUpdateNote)

	// delete_note
	s.server.AddTool(&mcp.Tool{
		Name:        "delete_note",
		Description: "Delete a note",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Note ID or prefix"}
			},
			"required": ["id"]
		}`),
	}, s.handleDeleteNote)

	// search_notes
	s.server.AddTool(&mcp.Tool{
		Name:        "search_notes",
		Description: "Full-text search notes",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "Search query"},
				"limit": {"type": "integer", "description": "Max results", "default": 10}
			},
			"required": ["query"]
		}`),
	}, s.handleSearchNotes)

	// add_tag
	s.server.AddTool(&mcp.Tool{
		Name:        "add_tag",
		Description: "Add a tag to a note",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Note ID or prefix"},
				"tag": {"type": "string", "description": "Tag name"}
			},
			"required": ["id", "tag"]
		}`),
	}, s.handleAddTag)

	// remove_tag
	s.server.AddTool(&mcp.Tool{
		Name:        "remove_tag",
		Description: "Remove a tag from a note",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Note ID or prefix"},
				"tag": {"type": "string", "description": "Tag name"}
			},
			"required": ["id", "tag"]
		}`),
	}, s.handleRemoveTag)

	// add_attachment
	s.server.AddTool(&mcp.Tool{
		Name:        "add_attachment",
		Description: "Add an attachment to a note",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Note ID or prefix"},
				"filename": {"type": "string", "description": "Filename"},
				"mime_type": {"type": "string", "description": "MIME type"},
				"data": {"type": "string", "description": "Base64 encoded data"}
			},
			"required": ["id", "filename", "mime_type", "data"]
		}`),
	}, s.handleAddAttachment)

	// list_attachments
	s.server.AddTool(&mcp.Tool{
		Name:        "list_attachments",
		Description: "List attachments for a note",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Note ID or prefix"}
			},
			"required": ["id"]
		}`),
	}, s.handleListAttachments)

	// get_attachment
	s.server.AddTool(&mcp.Tool{
		Name:        "get_attachment",
		Description: "Get an attachment's content",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Attachment ID or prefix"}
			},
			"required": ["id"]
		}`),
	}, s.handleGetAttachment)

	// export_note
	s.server.AddTool(&mcp.Tool{
		Name:        "export_note",
		Description: "Export a note as JSON or markdown",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"id": {"type": "string", "description": "Note ID or prefix"},
				"format": {"type": "string", "description": "Format: json or md", "default": "json"}
			},
			"required": ["id"]
		}`),
	}, s.handleExportNote)
}

// Tool handlers.
func (s *Server) handleAddNote(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	// Validate content is not empty
	if strings.TrimSpace(params.Content) == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "note content cannot be empty"},
			},
			IsError: true,
		}, nil
	}

	note := models.NewNote(params.Title, params.Content)
	if err := db.CreateNote(s.db, note); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to create note: %v", err)},
			},
			IsError: true,
		}, nil
	}

	for _, tag := range params.Tags {
		if err := db.AddTagToNote(s.db, note.ID, tag); err != nil {
			// Log but don't fail - note was already created
			continue
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Created note %s", note.ID.String())},
		},
	}, nil
}

func (s *Server) handleListNotes(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Tag   *string `json:"tag"`
		Limit int     `json:"limit"`
	}
	params.Limit = 20 // default
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	notes, err := db.ListNotes(s.db, params.Tag, params.Limit)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to list notes: %v", err)},
			},
			IsError: true,
		}, nil
	}

	data, _ := json.MarshalIndent(notes, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (s *Server) handleGetNote(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	var note *models.Note
	var err error

	// Try parsing as UUID first
	if id, parseErr := uuid.Parse(params.ID); parseErr == nil {
		note, err = db.GetNoteByID(s.db, id)
	} else {
		// Try as prefix
		note, err = db.GetNoteByPrefix(s.db, params.ID)
	}

	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to get note: %v", err)},
			},
			IsError: true,
		}, nil
	}

	data, _ := json.MarshalIndent(note, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (s *Server) handleUpdateNote(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID      string  `json:"id"`
		Title   *string `json:"title"`
		Content *string `json:"content"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	// Get existing note
	var note *models.Note
	var err error
	if id, parseErr := uuid.Parse(params.ID); parseErr == nil {
		note, err = db.GetNoteByID(s.db, id)
	} else {
		note, err = db.GetNoteByPrefix(s.db, params.ID)
	}
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to find note: %v", err)},
			},
			IsError: true,
		}, nil
	}

	// Update fields
	if params.Title != nil {
		note.Title = *params.Title
	}
	if params.Content != nil {
		if strings.TrimSpace(*params.Content) == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "note content cannot be empty"},
				},
				IsError: true,
			}, nil
		}
		note.Content = *params.Content
	}
	note.UpdatedAt = time.Now()

	if err := db.UpdateNote(s.db, note); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to update note: %v", err)},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Updated note %s", note.ID.String())},
		},
	}, nil
}

func (s *Server) handleDeleteNote(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	var id uuid.UUID
	var err error
	if parsedID, parseErr := uuid.Parse(params.ID); parseErr == nil {
		id = parsedID
	} else {
		// Get by prefix first
		note, err := db.GetNoteByPrefix(s.db, params.ID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to find note: %v", err)},
				},
				IsError: true,
			}, nil
		}
		id = note.ID
	}

	if err = db.DeleteNote(s.db, id); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to delete note: %v", err)},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Deleted note %s", id.String())},
		},
	}, nil
}

func (s *Server) handleSearchNotes(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	params.Limit = 10 // default
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	notes, err := db.SearchNotes(s.db, params.Query, params.Limit)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to search notes: %v", err)},
			},
			IsError: true,
		}, nil
	}

	data, _ := json.MarshalIndent(notes, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (s *Server) handleAddTag(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID  string `json:"id"`
		Tag string `json:"tag"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	var id uuid.UUID
	if parsedID, parseErr := uuid.Parse(params.ID); parseErr == nil {
		id = parsedID
	} else {
		note, err := db.GetNoteByPrefix(s.db, params.ID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to find note: %v", err)},
				},
				IsError: true,
			}, nil
		}
		id = note.ID
	}

	if err := db.AddTagToNote(s.db, id, params.Tag); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to add tag: %v", err)},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Added tag '%s' to note %s", params.Tag, id.String())},
		},
	}, nil
}

func (s *Server) handleRemoveTag(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID  string `json:"id"`
		Tag string `json:"tag"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	var id uuid.UUID
	if parsedID, parseErr := uuid.Parse(params.ID); parseErr == nil {
		id = parsedID
	} else {
		note, err := db.GetNoteByPrefix(s.db, params.ID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to find note: %v", err)},
				},
				IsError: true,
			}, nil
		}
		id = note.ID
	}

	if err := db.RemoveTagFromNote(s.db, id, params.Tag); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to remove tag: %v", err)},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Removed tag '%s' from note %s", params.Tag, id.String())},
		},
	}, nil
}

func (s *Server) handleAddAttachment(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID       string `json:"id"`
		Filename string `json:"filename"`
		MimeType string `json:"mime_type"`
		Data     string `json:"data"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	var noteID uuid.UUID
	if parsedID, parseErr := uuid.Parse(params.ID); parseErr == nil {
		noteID = parsedID
	} else {
		note, err := db.GetNoteByPrefix(s.db, params.ID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to find note: %v", err)},
				},
				IsError: true,
			}, nil
		}
		noteID = note.ID
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(params.Data)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("invalid base64 data: %v", err)},
			},
			IsError: true,
		}, nil
	}

	attachment := models.NewAttachment(noteID, params.Filename, params.MimeType, data)
	if err := db.CreateAttachment(s.db, attachment); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to create attachment: %v", err)},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Added attachment %s to note %s", attachment.ID.String(), noteID.String())},
		},
	}, nil
}

func (s *Server) handleListAttachments(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	var noteID uuid.UUID
	if parsedID, parseErr := uuid.Parse(params.ID); parseErr == nil {
		noteID = parsedID
	} else {
		note, err := db.GetNoteByPrefix(s.db, params.ID)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("failed to find note: %v", err)},
				},
				IsError: true,
			}, nil
		}
		noteID = note.ID
	}

	attachments, err := db.ListNoteAttachments(s.db, noteID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to list attachments: %v", err)},
			},
			IsError: true,
		}, nil
	}

	data, _ := json.MarshalIndent(attachments, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (s *Server) handleGetAttachment(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	var attachment *models.Attachment
	var err error
	if id, parseErr := uuid.Parse(params.ID); parseErr == nil {
		attachment, err = db.GetAttachment(s.db, id)
	} else {
		attachment, err = db.GetAttachmentByPrefix(s.db, params.ID)
	}

	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to get attachment: %v", err)},
			},
			IsError: true,
		}, nil
	}

	// Return with base64 encoded data
	result := map[string]interface{}{
		"id":       attachment.ID.String(),
		"note_id":  attachment.NoteID.String(),
		"filename": attachment.Filename,
		"mimetype": attachment.MimeType,
		"size":     len(attachment.Data),
		"data":     base64.StdEncoding.EncodeToString(attachment.Data),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

func (s *Server) handleExportNote(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		ID     string `json:"id"`
		Format string `json:"format"`
	}
	params.Format = "json" // default
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return nil, err
	}

	var note *models.Note
	var err error
	if id, parseErr := uuid.Parse(params.ID); parseErr == nil {
		note, err = db.GetNoteByID(s.db, id)
	} else {
		note, err = db.GetNoteByPrefix(s.db, params.ID)
	}
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("failed to get note: %v", err)},
			},
			IsError: true,
		}, nil
	}

	// Get tags and attachments
	tags, _ := db.GetNoteTags(s.db, note.ID)
	attachments, _ := db.ListNoteAttachments(s.db, note.ID)

	if params.Format == "md" {
		// Export as markdown
		result := fmt.Sprintf("# %s\n\n%s\n", note.Title, note.Content)
		if len(tags) > 0 {
			result += "\n## Tags\n"
			for _, tag := range tags {
				result += fmt.Sprintf("- %s\n", tag.Name)
			}
		}
		if len(attachments) > 0 {
			result += "\n## Attachments\n"
			for _, att := range attachments {
				result += fmt.Sprintf("- %s (%s)\n", att.Filename, att.MimeType)
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil
	}

	// Export as JSON
	export := map[string]interface{}{
		"note":        note,
		"tags":        tags,
		"attachments": attachments,
	}
	data, _ := json.MarshalIndent(export, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}
