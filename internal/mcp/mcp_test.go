// ABOUTME: Tests for MCP server tools and handlers.
// ABOUTME: Validates note CRUD operations via MCP tool interface.

package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/harperreed/memo/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// uuidRegex matches a valid UUID format.
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// extractNoteID extracts and validates the note ID from a "Created note <uuid>" message.
func extractNoteID(t *testing.T, text string) string {
	t.Helper()
	noteID := strings.TrimPrefix(text, "Created note ")
	if !uuidRegex.MatchString(noteID) {
		t.Fatalf("expected valid UUID format, got %q", noteID)
	}
	return noteID
}

// extractAttachmentID extracts and validates the attachment ID from an "Added attachment <uuid> to note <uuid>" message.
func extractAttachmentID(t *testing.T, text string) string {
	t.Helper()
	// Expected format: "Added attachment <uuid> to note <uuid>"
	parts := strings.Split(text, " ")
	if len(parts) < 3 {
		t.Fatalf("unexpected attachment response format: %q", text)
	}
	attachmentID := parts[2]
	if !uuidRegex.MatchString(attachmentID) {
		t.Fatalf("expected valid UUID format for attachment ID, got %q", attachmentID)
	}
	return attachmentID
}

// newTestServer creates a new MCP server with a test store.
func newTestServer(t *testing.T) *Server {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := storage.NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return NewServer(store)
}

// makeCallToolRequest creates a CallToolRequest with the given arguments.
func makeCallToolRequest(args map[string]interface{}) *mcp.CallToolRequest {
	argBytes, err := json.Marshal(args)
	if err != nil {
		// This should never happen with simple test data
		panic(err)
	}
	return &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: argBytes,
		},
	}
}

func TestNewServer(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := storage.NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	server := NewServer(store)
	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.server == nil {
		t.Error("expected non-nil inner MCP server")
	}
	if server.store == nil {
		t.Error("expected non-nil store")
	}
}

func TestHandleAddNote(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
		wantText  string
	}{
		{
			name: "valid note",
			args: map[string]interface{}{
				"title":   "Test Note",
				"content": "This is test content",
			},
			wantError: false,
			wantText:  "Created note",
		},
		{
			name: "note with tags",
			args: map[string]interface{}{
				"title":   "Tagged Note",
				"content": "Content with tags",
				"tags":    []string{"work", "important"},
			},
			wantError: false,
			wantText:  "Created note",
		},
		{
			name: "empty content",
			args: map[string]interface{}{
				"title":   "Empty Note",
				"content": "",
			},
			wantError: true,
			wantText:  "cannot be empty",
		},
		{
			name: "whitespace only content",
			args: map[string]interface{}{
				"title":   "Whitespace Note",
				"content": "   \n\t  ",
			},
			wantError: true,
			wantText:  "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			req := makeCallToolRequest(tt.args)

			result, err := server.handleAddNote(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsError != tt.wantError {
				t.Errorf("IsError = %v, want %v", result.IsError, tt.wantError)
			}

			if len(result.Content) == 0 {
				t.Fatal("expected content in result")
			}

			textContent, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatal("expected TextContent")
			}

			if !strings.Contains(textContent.Text, tt.wantText) {
				t.Errorf("text = %q, want to contain %q", textContent.Text, tt.wantText)
			}
		})
	}
}

func TestHandleListNotes(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create some test notes
	for i := 0; i < 3; i++ {
		req := makeCallToolRequest(map[string]interface{}{
			"title":   "Note " + string(rune('A'+i)),
			"content": "Content " + string(rune('A'+i)),
		})
		_, err := server.handleAddNote(ctx, req)
		if err != nil {
			t.Fatalf("failed to create test note: %v", err)
		}
	}

	t.Run("list all notes", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{})
		result, err := server.handleListNotes(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Error("unexpected error in result")
		}

		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("expected TextContent")
		}

		// Should contain JSON array of notes
		if !strings.HasPrefix(strings.TrimSpace(textContent.Text), "[") {
			t.Error("expected JSON array in response")
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"limit": 2,
		})
		result, err := server.handleListNotes(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Error("unexpected error in result")
		}
	})
}

func TestHandleGetNote(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Get Test Note",
		"content": "Content for get test",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	// Extract and validate UUID from "Created note <uuid>"
	noteID := extractNoteID(t, textContent.Text)

	t.Run("get by full ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		result, err := server.handleGetNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})

	t.Run("get by prefix", func(t *testing.T) {
		prefix := noteID[:6]
		req := makeCallToolRequest(map[string]interface{}{
			"id": prefix,
		})
		result, err := server.handleGetNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})

	t.Run("get non-existent note", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": "000000",
		})
		result, err := server.handleGetNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestHandleUpdateNote(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Update Test Note",
		"content": "Original content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("update title", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":    noteID,
			"title": "Updated Title",
		})
		result, err := server.handleUpdateNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})

	t.Run("update content", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":      noteID,
			"content": "Updated content",
		})
		result, err := server.handleUpdateNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})

	t.Run("update with empty content", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":      noteID,
			"content": "",
		})
		result, err := server.handleUpdateNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for empty content")
		}
	})

	t.Run("update non-existent note", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":    "000000",
			"title": "Should Fail",
		})
		result, err := server.handleUpdateNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestHandleDeleteNote(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Delete Test Note",
		"content": "Content to delete",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("delete by full ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		result, err := server.handleDeleteNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})

	t.Run("delete non-existent note", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": "000000",
		})
		result, err := server.handleDeleteNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestHandleSearchNotes(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create test notes with different content
	notes := []map[string]interface{}{
		{"title": "Apple Note", "content": "This is about apples and fruit"},
		{"title": "Banana Note", "content": "This is about bananas"},
		{"title": "Cherry Note", "content": "This is about cherries and apples"},
	}

	for _, note := range notes {
		req := makeCallToolRequest(note)
		_, err := server.handleAddNote(ctx, req)
		if err != nil {
			t.Fatalf("failed to create test note: %v", err)
		}
	}

	t.Run("search for existing term", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"query": "apples",
		})
		result, err := server.handleSearchNotes(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}

		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("expected TextContent")
		}

		// Should be a JSON array
		if !strings.HasPrefix(strings.TrimSpace(textContent.Text), "[") {
			t.Error("expected JSON array in response")
		}
	})

	t.Run("search with limit", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"query": "about",
			"limit": 2,
		})
		result, err := server.handleSearchNotes(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Error("unexpected error in result")
		}
	})
}

func TestHandleAddTag(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Tag Test Note",
		"content": "Content for tag test",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("add tag successfully", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":  noteID,
			"tag": "important",
		})
		result, err := server.handleAddTag(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})

	t.Run("add tag to non-existent note", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":  "000000",
			"tag": "test",
		})
		result, err := server.handleAddTag(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestHandleRemoveTag(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with a tag
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Remove Tag Test",
		"content": "Content for remove tag test",
		"tags":    []string{"removeme"},
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("remove tag successfully", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":  noteID,
			"tag": "removeme",
		})
		result, err := server.handleRemoveTag(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})

	t.Run("remove tag from non-existent note", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":  "000000",
			"tag": "test",
		})
		result, err := server.handleRemoveTag(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestHandleAddAttachment(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Attachment Test",
		"content": "Content for attachment test",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("add attachment successfully", func(t *testing.T) {
		data := base64.StdEncoding.EncodeToString([]byte("test file content"))
		req := makeCallToolRequest(map[string]interface{}{
			"id":        noteID,
			"filename":  "test.txt",
			"mime_type": "text/plain",
			"data":      data,
		})
		result, err := server.handleAddAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})

	t.Run("add attachment with invalid base64", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":        noteID,
			"filename":  "test.txt",
			"mime_type": "text/plain",
			"data":      "not-valid-base64!!!",
		})
		result, err := server.handleAddAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("add attachment to non-existent note", func(t *testing.T) {
		data := base64.StdEncoding.EncodeToString([]byte("test"))
		req := makeCallToolRequest(map[string]interface{}{
			"id":        "000000",
			"filename":  "test.txt",
			"mime_type": "text/plain",
			"data":      data,
		})
		result, err := server.handleAddAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestHandleListAttachments(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with an attachment
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "List Attachments Test",
		"content": "Content for list attachments test",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	// Add an attachment
	data := base64.StdEncoding.EncodeToString([]byte("attachment content"))
	attachReq := makeCallToolRequest(map[string]interface{}{
		"id":        noteID,
		"filename":  "attached.txt",
		"mime_type": "text/plain",
		"data":      data,
	})
	_, _ = server.handleAddAttachment(ctx, attachReq)

	t.Run("list attachments successfully", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		result, err := server.handleListAttachments(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}

		resultText := result.Content[0].(*mcp.TextContent)
		if !strings.Contains(resultText.Text, "attached.txt") {
			t.Error("expected attachment filename in response")
		}
	})

	t.Run("list attachments for non-existent note", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": "000000",
		})
		result, err := server.handleListAttachments(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestHandleGetAttachment(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with an attachment
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Get Attachment Test",
		"content": "Content for get attachment test",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	// Add an attachment
	originalData := []byte("get attachment content")
	encodedData := base64.StdEncoding.EncodeToString(originalData)
	attachReq := makeCallToolRequest(map[string]interface{}{
		"id":        noteID,
		"filename":  "getme.txt",
		"mime_type": "text/plain",
		"data":      encodedData,
	})
	attachResult, _ := server.handleAddAttachment(ctx, attachReq)
	attachText := attachResult.Content[0].(*mcp.TextContent)
	// Extract attachment ID from "Added attachment <uuid> to note <uuid>"
	attachmentID := extractAttachmentID(t, attachText.Text)

	t.Run("get attachment by ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": attachmentID,
		})
		result, err := server.handleGetAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}

		resultText := result.Content[0].(*mcp.TextContent)
		if !strings.Contains(resultText.Text, encodedData) {
			t.Error("expected base64 encoded data in response")
		}
	})

	t.Run("get non-existent attachment", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": "000000",
		})
		result, err := server.handleGetAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent attachment")
		}
	})
}

func TestHandleExportNote(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with tags
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Export Test Note",
		"content": "Content for export test",
		"tags":    []string{"export", "test"},
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("export as JSON (default)", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		result, err := server.handleExportNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}

		resultText := result.Content[0].(*mcp.TextContent)
		// Should be valid JSON
		var export map[string]interface{}
		if err := json.Unmarshal([]byte(resultText.Text), &export); err != nil {
			t.Errorf("expected valid JSON, got error: %v", err)
		}
	})

	t.Run("export as markdown", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":     noteID,
			"format": "md",
		})
		result, err := server.handleExportNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}

		resultText := result.Content[0].(*mcp.TextContent)
		if !strings.HasPrefix(resultText.Text, "# Export Test Note") {
			t.Error("expected markdown heading in export")
		}
		if !strings.Contains(resultText.Text, "## Tags") {
			t.Error("expected tags section in markdown export")
		}
	})

	t.Run("export non-existent note", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": "000000",
		})
		result, err := server.handleExportNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.IsError {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestHandleReadResource(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := storage.NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	server := NewServer(store)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Resource Test Note",
		"content": "Content for resource test",
		"tags":    []string{"resource-test"},
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("read resource by ID", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{
				URI: "memo://note/" + noteID,
			},
		}
		result, err := server.handleReadResource(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Contents) == 0 {
			t.Fatal("expected contents in result")
		}

		content := result.Contents[0]
		if content.MIMEType != "text/markdown" {
			t.Errorf("expected text/markdown MIME type, got %s", content.MIMEType)
		}
		if !strings.Contains(content.Text, "Resource Test Note") {
			t.Error("expected note title in content")
		}
	})

	t.Run("read resource by prefix", func(t *testing.T) {
		prefix := noteID[:6]
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{
				URI: "memo://note/" + prefix,
			},
		}
		result, err := server.handleReadResource(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Contents) == 0 {
			t.Fatal("expected contents in result")
		}

		if !strings.Contains(result.Contents[0].Text, "Resource Test Note") {
			t.Error("expected note title in content")
		}
	})

	t.Run("read resource with invalid URI", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{
				URI: "invalid://uri",
			},
		}
		_, err := server.handleReadResource(ctx, req)
		if err == nil {
			t.Error("expected error for invalid URI")
		}
	})

	t.Run("read resource with non-existent note", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{
				URI: "memo://note/000000",
			},
		}
		_, err := server.handleReadResource(ctx, req)
		if err == nil {
			t.Error("expected error for non-existent note")
		}
	})
}

func TestPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := storage.NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	server := NewServer(store)
	ctx := context.Background()

	t.Run("meeting notes prompt", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "create-meeting-notes",
				Arguments: map[string]string{
					"meeting_title": "Team Standup",
				},
			},
		}
		result, err := server.getMeetingNotesPrompt(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Messages) == 0 {
			t.Fatal("expected messages in result")
		}

		content := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(content.Text, "Team Standup") {
			t.Error("expected meeting title in prompt")
		}
	})

	t.Run("meeting notes prompt without title", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name:      "create-meeting-notes",
				Arguments: map[string]string{},
			},
		}
		result, err := server.getMeetingNotesPrompt(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(content.Text, "Meeting") {
			t.Error("expected default meeting title in prompt")
		}
	})

	t.Run("daily journal prompt", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "create-daily-journal",
				Arguments: map[string]string{
					"date": "2024-01-15",
				},
			},
		}
		result, err := server.getDailyJournalPrompt(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(content.Text, "2024-01-15") {
			t.Error("expected date in prompt")
		}
	})

	t.Run("daily journal prompt without date", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name:      "create-daily-journal",
				Arguments: map[string]string{},
			},
		}
		result, err := server.getDailyJournalPrompt(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(content.Text, "today") {
			t.Error("expected default date in prompt")
		}
	})

	t.Run("summarize note prompt", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "summarize-note",
				Arguments: map[string]string{
					"note_id": "abc123",
				},
			},
		}
		result, err := server.getSummarizeNotePrompt(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(content.Text, "abc123") {
			t.Error("expected note_id in prompt")
		}
	})

	t.Run("summarize note prompt without note_id", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name:      "summarize-note",
				Arguments: map[string]string{},
			},
		}
		_, err := server.getSummarizeNotePrompt(ctx, req)
		if err == nil {
			t.Error("expected error for missing note_id")
		}
	})

	t.Run("organize notes prompt", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "organize-notes",
			},
		}
		result, err := server.getOrganizeNotesPrompt(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(content.Text, "organize") {
			t.Error("expected organize keyword in prompt")
		}
	})

	t.Run("project note prompt", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "create-project-note",
				Arguments: map[string]string{
					"project_name": "Memo App",
				},
			},
		}
		result, err := server.getProjectNotePrompt(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(content.Text, "Memo App") {
			t.Error("expected project name in prompt")
		}
	})

	t.Run("project note prompt without name", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name:      "create-project-note",
				Arguments: map[string]string{},
			},
		}
		result, err := server.getProjectNotePrompt(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(content.Text, "Project") {
			t.Error("expected default project name in prompt")
		}
	})
}

func TestListNotesWithTag(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create notes with and without tags
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Tagged Note",
		"content": "Has a tag",
		"tags":    []string{"special"},
	})
	_, _ = server.handleAddNote(ctx, createReq)

	createReq2 := makeCallToolRequest(map[string]interface{}{
		"title":   "Untagged Note",
		"content": "No special tag",
	})
	_, _ = server.handleAddNote(ctx, createReq2)

	t.Run("list notes with specific tag", func(t *testing.T) {
		tagName := "special"
		req := makeCallToolRequest(map[string]interface{}{
			"tag": tagName,
		})
		result, err := server.handleListNotes(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Error("unexpected error in result")
		}

		resultText := result.Content[0].(*mcp.TextContent)
		if !strings.Contains(resultText.Text, "Tagged Note") {
			t.Error("expected tagged note in results")
		}
	})
}

func TestDeleteNoteByPrefix(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Delete By Prefix Test",
		"content": "Will be deleted by prefix",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)
	prefix := noteID[:6]

	t.Run("delete by prefix", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": prefix,
		})
		result, err := server.handleDeleteNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}

		// Verify note is gone
		getReq := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		getResult, _ := server.handleGetNote(ctx, getReq)
		if !getResult.IsError {
			t.Error("expected error when getting deleted note")
		}
	})
}

func TestExportNoteWithAttachment(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Export With Attachment",
		"content": "Has an attachment",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	// Add an attachment
	data := base64.StdEncoding.EncodeToString([]byte("attachment data"))
	attachReq := makeCallToolRequest(map[string]interface{}{
		"id":        noteID,
		"filename":  "export-test.txt",
		"mime_type": "text/plain",
		"data":      data,
	})
	_, _ = server.handleAddAttachment(ctx, attachReq)

	t.Run("export markdown with attachment", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":     noteID,
			"format": "md",
		})
		result, err := server.handleExportNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultText := result.Content[0].(*mcp.TextContent)
		if !strings.Contains(resultText.Text, "## Attachments") {
			t.Error("expected attachments section in markdown export")
		}
		if !strings.Contains(resultText.Text, "export-test.txt") {
			t.Error("expected attachment filename in markdown export")
		}
	})
}

func TestMain(m *testing.M) {
	// Run tests
	os.Exit(m.Run())
}

func TestHandleUpdateNoteByPrefix(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Prefix Update Test",
		"content": "Original content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)
	prefix := noteID[:6]

	t.Run("update by prefix", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":      prefix,
			"content": "Updated via prefix",
		})
		result, err := server.handleUpdateNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleAddTagByFullID(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Full ID Tag Test",
		"content": "Content for full ID tag test",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("add tag by full ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":  noteID,
			"tag": "fullid-tag",
		})
		result, err := server.handleAddTag(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleRemoveTagByFullID(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with tag
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Full ID Remove Tag Test",
		"content": "Content",
		"tags":    []string{"to-remove"},
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("remove tag by full ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":  noteID,
			"tag": "to-remove",
		})
		result, err := server.handleRemoveTag(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleAddAttachmentByFullID(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Full ID Attachment Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("add attachment by full ID", func(t *testing.T) {
		data := base64.StdEncoding.EncodeToString([]byte("test content"))
		req := makeCallToolRequest(map[string]interface{}{
			"id":        noteID,
			"filename":  "fullid.txt",
			"mime_type": "text/plain",
			"data":      data,
		})
		result, err := server.handleAddAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleListAttachmentsByFullID(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with attachment
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Full ID List Attachments Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	// Add attachment
	data := base64.StdEncoding.EncodeToString([]byte("content"))
	attachReq := makeCallToolRequest(map[string]interface{}{
		"id":        noteID,
		"filename":  "list.txt",
		"mime_type": "text/plain",
		"data":      data,
	})
	_, _ = server.handleAddAttachment(ctx, attachReq)

	t.Run("list attachments by full ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		result, err := server.handleListAttachments(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleGetAttachmentByFullID(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with attachment
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Full ID Get Attachment Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	// Add attachment
	data := base64.StdEncoding.EncodeToString([]byte("content"))
	attachReq := makeCallToolRequest(map[string]interface{}{
		"id":        noteID,
		"filename":  "get.txt",
		"mime_type": "text/plain",
		"data":      data,
	})
	attachResult, _ := server.handleAddAttachment(ctx, attachReq)
	attachText := attachResult.Content[0].(*mcp.TextContent)
	attachmentID := extractAttachmentID(t, attachText.Text)

	t.Run("get attachment by full ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": attachmentID,
		})
		result, err := server.handleGetAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleExportNoteByFullID(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Full ID Export Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("export by full ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		result, err := server.handleExportNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleDeleteNoteByFullID(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Full ID Delete Test",
		"content": "Will be deleted by full ID",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("delete by full ID", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		result, err := server.handleDeleteNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleUpdateNoteTitle(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Original Title",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	t.Run("update only title", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":    noteID,
			"title": "New Title",
		})
		result, err := server.handleUpdateNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}

		// Verify title was updated
		getReq := makeCallToolRequest(map[string]interface{}{
			"id": noteID,
		})
		getResult, _ := server.handleGetNote(ctx, getReq)
		getText := getResult.Content[0].(*mcp.TextContent)
		if !strings.Contains(getText.Text, "New Title") {
			t.Error("expected new title in note")
		}
	})
}

func TestListNotesWithDirTag(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a note with dir tag
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Dir Tagged Note",
		"content": "Has dir tag",
		"tags":    []string{"dir:/some/path"},
	})
	_, _ = server.handleAddNote(ctx, createReq)

	t.Run("list all notes", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{})
		result, err := server.handleListNotes(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Error("unexpected error in result")
		}

		resultText := result.Content[0].(*mcp.TextContent)
		if !strings.Contains(resultText.Text, "Dir Tagged Note") {
			t.Error("expected dir tagged note in results")
		}
	})
}

func TestHandleGetAttachmentByPrefix(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with attachment
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Prefix Get Attachment Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)

	// Add attachment
	data := base64.StdEncoding.EncodeToString([]byte("content"))
	attachReq := makeCallToolRequest(map[string]interface{}{
		"id":        noteID,
		"filename":  "prefix.txt",
		"mime_type": "text/plain",
		"data":      data,
	})
	attachResult, _ := server.handleAddAttachment(ctx, attachReq)
	attachText := attachResult.Content[0].(*mcp.TextContent)
	attachmentID := extractAttachmentID(t, attachText.Text)
	prefix := attachmentID[:6]

	t.Run("get attachment by prefix", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": prefix,
		})
		result, err := server.handleGetAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleAddAttachmentByPrefix(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Prefix Attachment Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)
	prefix := noteID[:6]

	t.Run("add attachment by prefix", func(t *testing.T) {
		data := base64.StdEncoding.EncodeToString([]byte("content"))
		req := makeCallToolRequest(map[string]interface{}{
			"id":        prefix,
			"filename":  "prefixed.txt",
			"mime_type": "text/plain",
			"data":      data,
		})
		result, err := server.handleAddAttachment(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleListAttachmentsByPrefix(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with attachment
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Prefix List Attachments Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)
	prefix := noteID[:6]

	// Add attachment
	data := base64.StdEncoding.EncodeToString([]byte("content"))
	attachReq := makeCallToolRequest(map[string]interface{}{
		"id":        noteID,
		"filename":  "list-by-prefix.txt",
		"mime_type": "text/plain",
		"data":      data,
	})
	_, _ = server.handleAddAttachment(ctx, attachReq)

	t.Run("list attachments by prefix", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": prefix,
		})
		result, err := server.handleListAttachments(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleExportNoteByPrefix(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Prefix Export Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)
	prefix := noteID[:6]

	t.Run("export by prefix", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id": prefix,
		})
		result, err := server.handleExportNote(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleAddTagByPrefix(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Prefix Add Tag Test",
		"content": "Content",
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)
	prefix := noteID[:6]

	t.Run("add tag by prefix", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":  prefix,
			"tag": "prefix-tag",
		})
		result, err := server.handleAddTag(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}

func TestHandleRemoveTagByPrefix(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()

	// Create a test note with tag
	createReq := makeCallToolRequest(map[string]interface{}{
		"title":   "Prefix Remove Tag Test",
		"content": "Content",
		"tags":    []string{"remove-by-prefix"},
	})
	createResult, _ := server.handleAddNote(ctx, createReq)
	textContent := createResult.Content[0].(*mcp.TextContent)
	noteID := extractNoteID(t, textContent.Text)
	prefix := noteID[:6]

	t.Run("remove tag by prefix", func(t *testing.T) {
		req := makeCallToolRequest(map[string]interface{}{
			"id":  prefix,
			"tag": "remove-by-prefix",
		})
		result, err := server.handleRemoveTag(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Errorf("unexpected error: %v", result.Content[0].(*mcp.TextContent).Text)
		}
	})
}
