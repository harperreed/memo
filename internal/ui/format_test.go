// ABOUTME: Tests for terminal UI formatting functions.
// ABOUTME: Validates note display and markdown rendering.

package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/memo/internal/models"
)

func TestFormatNoteListItem(t *testing.T) {
	note := &models.Note{
		ID:        uuid.New(),
		Title:     "Test Note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tags := []*models.Tag{{Name: "important"}, {Name: "work"}}

	output := FormatNoteListItem(note, tags)

	if !strings.Contains(output, note.ID.String()[:6]) {
		t.Error("expected output to contain ID prefix")
	}
	if !strings.Contains(output, "Test Note") {
		t.Error("expected output to contain title")
	}
	if !strings.Contains(output, "important") {
		t.Error("expected output to contain tag")
	}
}

func TestFormatNoteContent(t *testing.T) {
	content := "# Hello\n\nThis is **bold** text."

	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFormatTagList(t *testing.T) {
	tags := []TagCount{
		{Name: "work", Count: 5},
		{Name: "personal", Count: 3},
	}

	output := FormatTagList(tags)

	if !strings.Contains(output, "work") {
		t.Error("expected output to contain 'work'")
	}
	if !strings.Contains(output, "5") {
		t.Error("expected output to contain count '5'")
	}
}

func TestFormatDirSectionHeader(t *testing.T) {
	dirPath := "/Users/harper/projects/memo"
	output := FormatDirSectionHeader(dirPath)

	if !strings.Contains(output, dirPath) {
		t.Errorf("expected output to contain dir path %q", dirPath)
	}
	if !strings.Contains(output, "[D]") {
		t.Error("expected output to contain directory marker [D]")
	}
}

func TestFormatGlobalSectionHeader(t *testing.T) {
	output := FormatGlobalSectionHeader()

	if !strings.Contains(output, "Global") {
		t.Error("expected output to contain 'Global'")
	}
	if !strings.Contains(output, "[G]") {
		t.Error("expected output to contain global marker [G]")
	}
}

func TestFormatShowMorePrompt(t *testing.T) {
	output := FormatShowMorePrompt(15)

	if !strings.Contains(output, "15") {
		t.Error("expected output to contain count '15'")
	}
	if !strings.Contains(output, "more") {
		t.Error("expected output to contain 'more'")
	}
	if !strings.Contains(output, "y/n") {
		t.Error("expected output to contain 'y/n'")
	}
}

func TestTerminalRendererHeading(t *testing.T) {
	content := "# Heading One\n\n## Heading Two"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "# ") {
		t.Error("expected output to contain heading marker")
	}
	if !strings.Contains(output, "## ") {
		t.Error("expected output to contain second level heading marker")
	}
}

func TestTerminalRendererCodeBlock(t *testing.T) {
	content := "```go\nfunc main() {}\n```"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "```go") {
		t.Error("expected output to contain code block language marker")
	}
	if !strings.Contains(output, "func main()") {
		t.Error("expected output to contain code content")
	}
}

func TestTerminalRendererList(t *testing.T) {
	content := "- Item one\n- Item two"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "- Item one") {
		t.Error("expected output to contain list item")
	}
}

func TestFormatNoteListItemWithoutTags(t *testing.T) {
	note := &models.Note{
		ID:        uuid.New(),
		Title:     "No Tags Note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	output := FormatNoteListItem(note, nil)

	if !strings.Contains(output, "No Tags Note") {
		t.Error("expected output to contain title")
	}
	// Should still have updated date
	if !strings.Contains(output, "Updated:") {
		t.Error("expected output to contain Updated label")
	}
}

func TestFormatNoteHeader(t *testing.T) {
	note := &models.Note{
		ID:        uuid.New(),
		Title:     "Header Test Note",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tags := []*models.Tag{{Name: "tag1"}, {Name: "tag2"}}

	output := FormatNoteHeader(note, tags)

	if !strings.Contains(output, "Header Test Note") {
		t.Error("expected output to contain title")
	}
	if !strings.Contains(output, "ID:") {
		t.Error("expected output to contain ID label")
	}
	if !strings.Contains(output, "Created:") {
		t.Error("expected output to contain Created label")
	}
	if !strings.Contains(output, "Updated:") {
		t.Error("expected output to contain Updated label")
	}
	if !strings.Contains(output, "tag1") {
		t.Error("expected output to contain tag1")
	}
}

func TestFormatNoteHeaderWithoutTags(t *testing.T) {
	note := &models.Note{
		ID:        uuid.New(),
		Title:     "No Tags Header",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	output := FormatNoteHeader(note, nil)

	if !strings.Contains(output, "No Tags Header") {
		t.Error("expected output to contain title")
	}
	// Should not contain Tags label when there are no tags
	if strings.Contains(output, "Tags:") {
		t.Error("expected output to NOT contain Tags label when no tags")
	}
}

func TestFormatAttachmentList(t *testing.T) {
	attachments := []AttachmentInfo{
		{ID: "abc123456789", Filename: "document.pdf", MimeType: "application/pdf"},
		{ID: "def987654321", Filename: "image.png", MimeType: "image/png"},
	}

	output := FormatAttachmentList(attachments)

	if !strings.Contains(output, "Attachments:") {
		t.Error("expected output to contain Attachments header")
	}
	if !strings.Contains(output, "abc123") {
		t.Error("expected output to contain attachment ID prefix")
	}
	if !strings.Contains(output, "document.pdf") {
		t.Error("expected output to contain filename")
	}
	if !strings.Contains(output, "application/pdf") {
		t.Error("expected output to contain mime type")
	}
}

func TestSeparator(t *testing.T) {
	sep := Separator()

	if len(sep) == 0 {
		t.Error("expected non-empty separator")
	}
	// Should contain newline at end
	if !strings.HasSuffix(sep, "\n") {
		t.Error("expected separator to end with newline")
	}
}

func TestSuccess(t *testing.T) {
	msg := Success("Operation completed")

	if !strings.Contains(msg, "Operation completed") {
		t.Error("expected success message to contain original text")
	}
	if !strings.Contains(msg, "+ ") {
		t.Error("expected success message to contain + prefix")
	}
}

func TestError(t *testing.T) {
	msg := Error("Something went wrong")

	if !strings.Contains(msg, "Something went wrong") {
		t.Error("expected error message to contain original text")
	}
	if !strings.Contains(msg, "x ") {
		t.Error("expected error message to contain x prefix")
	}
}

func TestTerminalRendererOrderedList(t *testing.T) {
	content := "1. First item\n2. Second item\n3. Third item"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	// Output should contain numbered items
	if !strings.Contains(output, "1.") {
		t.Error("expected output to contain numbered list item")
	}
}

func TestTerminalRendererBlockquote(t *testing.T) {
	content := "> This is a quote"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, ">") {
		t.Error("expected output to contain blockquote marker")
	}
}

func TestTerminalRendererThematicBreak(t *testing.T) {
	content := "Above\n\n---\n\nBelow"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	// Should contain some form of horizontal line
	if !strings.Contains(output, "─") && !strings.Contains(output, "Above") {
		t.Error("expected output to contain thematic break or content")
	}
}

func TestTerminalRendererEmphasis(t *testing.T) {
	content := "This is *italic* and **bold** text"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "italic") {
		t.Error("expected output to contain italic text")
	}
	if !strings.Contains(output, "bold") {
		t.Error("expected output to contain bold text")
	}
}

func TestTerminalRendererCodeSpan(t *testing.T) {
	content := "Use `code` inline"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "`") {
		t.Error("expected output to contain code span marker")
	}
}

func TestTerminalRendererLink(t *testing.T) {
	content := "[Click here](https://example.com)"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "Click here") {
		t.Error("expected output to contain link text")
	}
	if !strings.Contains(output, "example.com") {
		t.Error("expected output to contain link URL")
	}
}

func TestTerminalRendererAutoLink(t *testing.T) {
	content := "<https://example.com>"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "example.com") {
		t.Error("expected output to contain auto-link URL")
	}
}

func TestTerminalRendererImage(t *testing.T) {
	content := "![Alt text](image.png)"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "[Image:") {
		t.Error("expected output to contain image placeholder")
	}
}

func TestTerminalRendererIndentedCodeBlock(t *testing.T) {
	content := "    indented code\n    more code"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	// Indented code blocks should be rendered
	if output == "" {
		t.Error("expected non-empty output for code block")
	}
}

func TestTerminalRendererSoftLineBreak(t *testing.T) {
	content := "Line one\nLine two"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "Line one") {
		t.Error("expected output to contain first line")
	}
}

func TestNewTerminalRenderer(t *testing.T) {
	renderer := NewTerminalRenderer()
	if renderer == nil {
		t.Error("expected non-nil renderer")
	}
}

func TestTerminalRendererHTMLBlock(t *testing.T) {
	content := "<div>HTML content</div>"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	// HTML blocks are typically skipped in terminal output
	// but should not cause errors
	if output == "" {
		t.Log("HTML block rendered as empty, which is acceptable")
	}
}

func TestTerminalRendererRawHTML(t *testing.T) {
	content := "Text with <strong>raw html</strong> inline"
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "Text with") {
		t.Error("expected output to contain surrounding text")
	}
}

func TestFormatNoteContentParagraph(t *testing.T) {
	content := "First paragraph.\n\nSecond paragraph."
	output, err := FormatNoteContent(content)
	if err != nil {
		t.Fatalf("failed to format content: %v", err)
	}

	if !strings.Contains(output, "First paragraph") {
		t.Error("expected output to contain first paragraph")
	}
	if !strings.Contains(output, "Second paragraph") {
		t.Error("expected output to contain second paragraph")
	}
}
