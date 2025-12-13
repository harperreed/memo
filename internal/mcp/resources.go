// ABOUTME: MCP resources for exposing notes as readable resources.
// ABOUTME: Allows AI agents to access note content via URI scheme.

package mcp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/db"
	"github.com/harper/memo/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources() {
	// We register a resource template for dynamic note access
	// The SDK will automatically handle listing based on the template
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "memo://note/{id}",
			Name:        "Note",
			Description: "Access individual notes by ID",
			MIMEType:    "text/markdown",
		},
		s.handleReadResource,
	)
}

func (s *Server) handleReadResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Parse URI: memo://note/{id}
	var noteIDStr string
	_, err := fmt.Sscanf(req.Params.URI, "memo://note/%s", &noteIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid resource URI: %s", req.Params.URI)
	}

	// Try to parse as UUID or prefix
	var note *models.Note
	if noteID, parseErr := uuid.Parse(noteIDStr); parseErr == nil {
		note, err = db.GetNoteByID(s.db, noteID)
	} else {
		note, err = db.GetNoteByPrefix(s.db, noteIDStr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	// Get tags
	tags, _ := db.GetNoteTags(s.db, note.ID)
	tagNames := make([]string, len(tags))
	for i, tag := range tags {
		tagNames[i] = tag.Name
	}

	// Format as markdown with frontmatter
	content := fmt.Sprintf("# %s\n\n", note.Title)
	if len(tagNames) > 0 {
		content += fmt.Sprintf("**Tags:** %v\n\n", tagNames)
	}
	content += note.Content

	// Return as text content
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/markdown",
				Text:     content,
			},
		},
	}, nil
}
