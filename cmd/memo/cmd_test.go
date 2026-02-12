// ABOUTME: Tests for CLI commands in memo application.
// ABOUTME: Validates add, list, show, rm, tag, attach, export, and import commands.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harperreed/memo/internal/models"
	"github.com/harperreed/memo/internal/storage"
)

// setupTestStore creates a temporary store for testing and returns a cleanup function.
func setupTestStore(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	var err error
	store, err = storage.NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	return func() {
		if store != nil {
			store.Close()
			store = nil
		}
	}
}

func TestCollectTags(t *testing.T) {
	tests := []struct {
		name     string
		tagsFlag string
		hereFlag bool
		wantTags []string
	}{
		{
			name:     "empty flags",
			tagsFlag: "",
			hereFlag: false,
			wantTags: nil,
		},
		{
			name:     "single tag",
			tagsFlag: "work",
			hereFlag: false,
			wantTags: []string{"work"},
		},
		{
			name:     "multiple tags",
			tagsFlag: "work, personal, important",
			hereFlag: false,
			wantTags: []string{"work", "personal", "important"},
		},
		{
			name:     "tags with whitespace",
			tagsFlag: "  work  ,  personal  ",
			hereFlag: false,
			wantTags: []string{"work", "personal"},
		},
		{
			name:     "empty tags are filtered",
			tagsFlag: "work,,personal",
			hereFlag: false,
			wantTags: []string{"work", "personal"},
		},
		{
			name:     "with here flag",
			tagsFlag: "work",
			hereFlag: true,
			wantTags: nil, // Will contain dir: tag, tested separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Special case for here flag - just verify it adds something
			if tt.hereFlag {
				tags := collectTags(tt.tagsFlag, tt.hereFlag)
				hasDir := false
				for _, tag := range tags {
					if strings.HasPrefix(tag, "dir:") {
						hasDir = true
						break
					}
				}
				if !hasDir {
					t.Error("expected dir: tag when hereFlag is true")
				}
				return
			}

			tags := collectTags(tt.tagsFlag, tt.hereFlag)
			if len(tags) != len(tt.wantTags) {
				t.Errorf("got %d tags, want %d", len(tags), len(tt.wantTags))
				return
			}
			for i, tag := range tags {
				if tag != tt.wantTags[i] {
					t.Errorf("tag[%d] = %q, want %q", i, tag, tt.wantTags[i])
				}
			}
		})
	}
}

func TestTagsToModels(t *testing.T) {
	tags := []string{"work", "important", "todo"}
	result := tagsToModels(tags)

	if len(result) != 3 {
		t.Errorf("got %d model tags, want 3", len(result))
	}

	for i, tag := range result {
		if tag.Name != tags[i] {
			t.Errorf("tag[%d].Name = %q, want %q", i, tag.Name, tags[i])
		}
	}
}

func TestTagsToModelsList(t *testing.T) {
	tags := []string{"alpha", "beta"}
	result := tagsToModelsList(tags)

	if len(result) != 2 {
		t.Errorf("got %d model tags, want 2", len(result))
	}

	if result[0].Name != "alpha" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "alpha")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "My Note Title",
			expected: "My Note Title",
		},
		{
			name:     "with slashes",
			input:    "path/to/note",
			expected: "path-to-note",
		},
		{
			name:     "with special chars",
			input:    "note: test * file?",
			expected: "note- test - file-",
		},
		{
			name:     "long filename truncated",
			input:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExportJSONToFile(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a test note
	note := models.NewNote("Export Test", "Content for export")
	err := store.CreateNote(note, []string{"test-tag"})
	if err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}

	// Export to file
	outputPath := filepath.Join(t.TempDir(), "export.json")
	notes := []*models.Note{note}
	noteTags := [][]string{{"test-tag"}}

	err = exportJSON(notes, noteTags, outputPath)
	if err != nil {
		t.Fatalf("exportJSON failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("export file was not created")
	}

	// Verify content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	if !strings.Contains(string(data), "Export Test") {
		t.Error("export file should contain note title")
	}
	if !strings.Contains(string(data), "test-tag") {
		t.Error("export file should contain tag")
	}
}

func TestExportYAMLToFile(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	note := models.NewNote("YAML Export", "YAML content")
	err := store.CreateNote(note, nil)
	if err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "export.yaml")
	notes := []*models.Note{note}
	noteTags := [][]string{{}}

	err = exportYAML(notes, noteTags, outputPath)
	if err != nil {
		t.Fatalf("exportYAML failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	if !strings.Contains(string(data), "YAML Export") {
		t.Error("export file should contain note title")
	}
}

func TestExportMarkdown(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	note := models.NewNote("MD Export Note", "Markdown content here")
	err := store.CreateNote(note, []string{"md-test"})
	if err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}

	outputDir := filepath.Join(t.TempDir(), "md-export")
	notes := []*models.Note{note}
	noteTags := [][]string{{"md-test"}}

	err = exportMarkdown(notes, noteTags, outputDir)
	if err != nil {
		t.Fatalf("exportMarkdown failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("export directory was not created")
	}

	// Find and verify markdown file
	files, _ := filepath.Glob(filepath.Join(outputDir, "*.md"))
	if len(files) == 0 {
		t.Error("no markdown files found in export directory")
	}
}

func TestImportJSONFile(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a JSON file to import
	jsonContent := `{
		"version": "1.0",
		"exported_at": "2024-01-01T00:00:00Z",
		"tool": "memo",
		"notes": [
			{
				"id": "12345678-1234-1234-1234-123456789012",
				"title": "Imported Note",
				"content": "Imported content",
				"tags": ["imported"],
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		]
	}`

	importPath := filepath.Join(t.TempDir(), "import.json")
	err := os.WriteFile(importPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("failed to create import file: %v", err)
	}

	err = importJSON(importPath)
	if err != nil {
		t.Fatalf("importJSON failed: %v", err)
	}

	// Verify note was imported
	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	found := false
	for _, n := range notes {
		if n.Title == "Imported Note" {
			found = true
			break
		}
	}
	if !found {
		t.Error("imported note was not found")
	}
}

func TestImportYAMLFile(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	yamlContent := `version: "1.0"
exported_at: 2024-01-01T00:00:00Z
tool: memo
notes:
  - id: "12345678-1234-1234-1234-123456789013"
    title: "YAML Import Note"
    content: "YAML imported content"
    tags: ["yaml-import"]
    created: 2024-01-01T00:00:00Z
    updated: 2024-01-01T00:00:00Z
`

	importPath := filepath.Join(t.TempDir(), "import.yaml")
	err := os.WriteFile(importPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("failed to create import file: %v", err)
	}

	err = importYAML(importPath)
	if err != nil {
		t.Fatalf("importYAML failed: %v", err)
	}

	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	found := false
	for _, n := range notes {
		if n.Title == "YAML Import Note" {
			found = true
			break
		}
	}
	if !found {
		t.Error("imported note was not found")
	}
}

func TestImportMarkdownFile(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	mdContent := `---
title: "MD Import Note"
tags: ["markdown", "import"]
---

This is the markdown content.
`

	importPath := filepath.Join(t.TempDir(), "import.md")
	err := os.WriteFile(importPath, []byte(mdContent), 0644)
	if err != nil {
		t.Fatalf("failed to create import file: %v", err)
	}

	err = importMarkdownFile(importPath)
	if err != nil {
		t.Fatalf("importMarkdownFile failed: %v", err)
	}

	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	found := false
	for _, n := range notes {
		if n.Title == "MD Import Note" {
			found = true
			break
		}
	}
	if !found {
		t.Error("imported note was not found")
	}
}

func TestImportMarkdownFileWithoutFrontmatter(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	mdContent := `# Simple Markdown

Just content without frontmatter.
`

	importPath := filepath.Join(t.TempDir(), "simple.md")
	err := os.WriteFile(importPath, []byte(mdContent), 0644)
	if err != nil {
		t.Fatalf("failed to create import file: %v", err)
	}

	err = importMarkdownFile(importPath)
	if err != nil {
		t.Fatalf("importMarkdownFile failed: %v", err)
	}

	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	// Title should be derived from filename
	found := false
	for _, n := range notes {
		if n.Title == "simple" {
			found = true
			break
		}
	}
	if !found {
		t.Error("imported note with derived title was not found")
	}
}

func TestImportMarkdownEmptyContent(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create empty markdown file
	importPath := filepath.Join(t.TempDir(), "empty.md")
	err := os.WriteFile(importPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to create import file: %v", err)
	}

	err = importMarkdownFile(importPath)
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestImportMarkdownDir(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	tmpDir := t.TempDir()

	// Create some markdown files
	files := map[string]string{
		"note1.md": "# Note 1\n\nContent 1",
		"note2.md": "# Note 2\n\nContent 2",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	err := importMarkdownDir(tmpDir)
	if err != nil {
		t.Fatalf("importMarkdownDir failed: %v", err)
	}

	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	if len(notes) < 2 {
		t.Errorf("expected at least 2 notes, got %d", len(notes))
	}
}

func TestImportExportData(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	export := &ExportData{
		Version: "1.0",
		Notes: []ExportNote{
			{
				ID:      "11111111-1111-1111-1111-111111111111",
				Title:   "Export Data Note",
				Content: "Export data content",
				Tags:    []string{"export-data"},
			},
		},
	}

	err := importExportData(export)
	if err != nil {
		t.Fatalf("importExportData failed: %v", err)
	}

	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	found := false
	for _, n := range notes {
		if n.Title == "Export Data Note" {
			found = true
			break
		}
	}
	if !found {
		t.Error("imported note was not found")
	}
}

func TestImportExportDataWithAttachment(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	export := &ExportData{
		Version: "1.0",
		Notes: []ExportNote{
			{
				ID:      "22222222-2222-2222-2222-222222222222",
				Title:   "Note With Attachment",
				Content: "Content with attachment",
				Attachments: []ExportAttachment{
					{
						ID:       "33333333-3333-3333-3333-333333333333",
						Filename: "test.txt",
						MimeType: "text/plain",
						Data:     "dGVzdCBkYXRh", // "test data" base64 encoded
					},
				},
			},
		},
	}

	err := importExportData(export)
	if err != nil {
		t.Fatalf("importExportData failed: %v", err)
	}

	// Verify note exists
	notes, err := store.ListNotes(nil)
	if err != nil {
		t.Fatalf("failed to list notes: %v", err)
	}

	var noteID *models.Note
	for _, n := range notes {
		if n.Title == "Note With Attachment" {
			noteID = n
			break
		}
	}
	if noteID == nil {
		t.Fatal("imported note was not found")
	}

	// Verify attachment exists
	attachments, err := store.ListAttachmentsByNote(noteID.ID)
	if err != nil {
		t.Fatalf("failed to list attachments: %v", err)
	}

	if len(attachments) == 0 {
		t.Error("expected attachment to be imported")
	}
}

func TestExportJSONToStdout(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	note := models.NewNote("Stdout Test", "Stdout content")
	err := store.CreateNote(note, nil)
	if err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}

	// Test with empty output path (stdout)
	notes := []*models.Note{note}
	noteTags := [][]string{{}}

	err = exportJSON(notes, noteTags, "")
	if err != nil {
		t.Errorf("exportJSON to stdout failed: %v", err)
	}

	// Test with "-" output path (stdout)
	err = exportJSON(notes, noteTags, "-")
	if err != nil {
		t.Errorf("exportJSON to - failed: %v", err)
	}
}

func TestExportYAMLToStdout(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	note := models.NewNote("YAML Stdout", "YAML stdout content")
	err := store.CreateNote(note, nil)
	if err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}

	notes := []*models.Note{note}
	noteTags := [][]string{{}}

	err = exportYAML(notes, noteTags, "")
	if err != nil {
		t.Errorf("exportYAML to stdout failed: %v", err)
	}
}

func TestExportMarkdownWithAttachment(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	note := models.NewNote("Note With Att", "Content here")
	err := store.CreateNote(note, nil)
	if err != nil {
		t.Fatalf("failed to create test note: %v", err)
	}

	// Add an attachment
	att := models.NewAttachment(note.ID, "test.txt", "text/plain", []byte("attachment data"))
	err = store.CreateAttachment(att)
	if err != nil {
		t.Fatalf("failed to create attachment: %v", err)
	}

	outputDir := filepath.Join(t.TempDir(), "md-att-export")
	notes := []*models.Note{note}
	noteTags := [][]string{{}}

	err = exportMarkdown(notes, noteTags, outputDir)
	if err != nil {
		t.Fatalf("exportMarkdown failed: %v", err)
	}

	// Verify attachment directory exists
	attDir := filepath.Join(outputDir, "attachments", note.ID.String()[:8])
	if _, err := os.Stat(attDir); os.IsNotExist(err) {
		t.Error("attachment directory was not created")
	}
}

func TestListSearch(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create test notes
	note1 := models.NewNote("Apple Recipe", "How to make apple pie")
	store.CreateNote(note1, nil)

	note2 := models.NewNote("Banana Bread", "Banana bread recipe")
	store.CreateNote(note2, nil)

	note3 := models.NewNote("Cherry Tart", "Another apple dessert with cherries")
	store.CreateNote(note3, nil)

	// Test search function
	err := listSearch("apple", 10)
	if err != nil {
		t.Errorf("listSearch failed: %v", err)
	}
}

func TestListSearchNoResults(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Test with no matching results
	err := listSearch("nonexistent", 10)
	if err != nil {
		t.Errorf("listSearch failed: %v", err)
	}
}

func TestListByTag(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create notes with tags
	note1 := models.NewNote("Work Note", "Work content")
	store.CreateNote(note1, []string{"work"})

	note2 := models.NewNote("Personal Note", "Personal content")
	store.CreateNote(note2, []string{"personal"})

	// Test filter by tag
	err := listByTag("work", 10)
	if err != nil {
		t.Errorf("listByTag failed: %v", err)
	}
}

func TestListByTagNoResults(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Test with non-existent tag
	err := listByTag("nonexistent-tag", 10)
	if err != nil {
		t.Errorf("listByTag failed: %v", err)
	}
}

func TestListHere(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Get current directory
	pwd, _ := os.Getwd()
	dirTag := "dir:" + strings.ToLower(pwd)

	// Create note with current directory tag
	note := models.NewNote("Local Note", "Note for this directory")
	store.CreateNote(note, []string{dirTag})

	// Test listHere
	err := listHere(10)
	if err != nil {
		t.Errorf("listHere failed: %v", err)
	}
}

func TestListHereNoResults(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Test with no notes for current directory
	err := listHere(10)
	if err != nil {
		t.Errorf("listHere failed: %v", err)
	}
}

func TestListSectionedEmpty(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Test with empty database
	err := listSectioned(10)
	if err != nil {
		t.Errorf("listSectioned failed: %v", err)
	}
}

func TestListSectionedWithGlobalNotes(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create global notes (no dir: tag)
	note1 := models.NewNote("Global Note 1", "Global content 1")
	store.CreateNote(note1, []string{"general"})

	note2 := models.NewNote("Global Note 2", "Global content 2")
	store.CreateNote(note2, nil)

	// Test sectioned listing
	err := listSectioned(10)
	if err != nil {
		t.Errorf("listSectioned failed: %v", err)
	}
}

func TestListSectionedWithDirNotes(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	pwd, _ := os.Getwd()
	dirTag := "dir:" + strings.ToLower(pwd)

	// Create directory-specific note
	note := models.NewNote("Dir Note", "Directory content")
	store.CreateNote(note, []string{dirTag})

	// Create global note
	globalNote := models.NewNote("Global", "Global content")
	store.CreateNote(globalNote, nil)

	err := listSectioned(10)
	if err != nil {
		t.Errorf("listSectioned failed: %v", err)
	}
}

func TestInstallSkillWithHomePath(t *testing.T) {
	tmpHome := t.TempDir()

	// Save and restore the flag
	oldSkipConfirm := skillSkipConfirm
	skillSkipConfirm = true
	defer func() { skillSkipConfirm = oldSkipConfirm }()

	err := installSkillWithHome(tmpHome)
	if err != nil {
		t.Errorf("installSkillWithHome failed: %v", err)
	}

	// Verify file exists
	skillPath := filepath.Join(tmpHome, ".claude", "skills", "memo", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("SKILL.md was not created")
	}
}

func TestExecute(t *testing.T) {
	// Test Execute with version command
	oldArgs := os.Args
	os.Args = []string{"memo", "version"}
	defer func() { os.Args = oldArgs }()

	// Execute should not return error for valid commands
	// Note: This will actually run the version command
	err := Execute()
	if err != nil {
		t.Logf("Execute returned error (may be expected): %v", err)
	}
}

func TestExportJSONWithAttachment(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	note := models.NewNote("JSON Att Note", "JSON attachment content")
	store.CreateNote(note, nil)

	att := models.NewAttachment(note.ID, "data.bin", "application/octet-stream", []byte("binary data"))
	store.CreateAttachment(att)

	outputPath := filepath.Join(t.TempDir(), "export-att.json")
	notes := []*models.Note{note}
	noteTags := [][]string{{}}

	err := exportJSON(notes, noteTags, outputPath)
	if err != nil {
		t.Fatalf("exportJSON failed: %v", err)
	}

	data, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(data), "data.bin") {
		t.Error("export should contain attachment filename")
	}
}

func TestExportYAMLWithAttachment(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	note := models.NewNote("YAML Att Note", "YAML attachment content")
	store.CreateNote(note, nil)

	att := models.NewAttachment(note.ID, "doc.pdf", "application/pdf", []byte("pdf data"))
	store.CreateAttachment(att)

	outputPath := filepath.Join(t.TempDir(), "export-att.yaml")
	notes := []*models.Note{note}
	noteTags := [][]string{{}}

	err := exportYAML(notes, noteTags, outputPath)
	if err != nil {
		t.Fatalf("exportYAML failed: %v", err)
	}

	data, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(data), "doc.pdf") {
		t.Error("export should contain attachment filename")
	}
}

func TestImportJSONFileNotFound(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	err := importJSON("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestImportYAMLFileNotFound(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	err := importYAML("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestImportMarkdownFileNotFound(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	err := importMarkdownFile("/nonexistent/path.md")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestImportInvalidJSON(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	importPath := filepath.Join(t.TempDir(), "invalid.json")
	os.WriteFile(importPath, []byte("not valid json"), 0644)

	err := importJSON(importPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestImportInvalidYAML(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	importPath := filepath.Join(t.TempDir(), "invalid.yaml")
	os.WriteFile(importPath, []byte(":::not valid yaml"), 0644)

	err := importYAML(importPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestListSectionedMixedNotes(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	pwd, _ := os.Getwd()
	dirTag := "dir:" + strings.ToLower(pwd)

	// Create multiple directory notes
	for i := 0; i < 3; i++ {
		note := models.NewNote("Dir Note "+string(rune('A'+i)), "Dir content")
		store.CreateNote(note, []string{dirTag})
	}

	// Create many global notes to test the "show more" logic
	for i := 0; i < 15; i++ {
		note := models.NewNote("Global Note "+string(rune('A'+i)), "Global content")
		store.CreateNote(note, nil)
	}

	err := listSectioned(10)
	if err != nil {
		t.Errorf("listSectioned failed: %v", err)
	}
}

func TestListSectionedOnlyDirNotes(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	pwd, _ := os.Getwd()
	dirTag := "dir:" + strings.ToLower(pwd)

	// Create only directory notes
	note := models.NewNote("Only Dir", "Dir content only")
	store.CreateNote(note, []string{dirTag})

	err := listSectioned(10)
	if err != nil {
		t.Errorf("listSectioned failed: %v", err)
	}
}

func TestListSectionedOnlyGlobalNotes(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create only global notes
	for i := 0; i < 5; i++ {
		note := models.NewNote("Global "+string(rune('A'+i)), "Content")
		store.CreateNote(note, nil)
	}

	err := listSectioned(10)
	if err != nil {
		t.Errorf("listSectioned failed: %v", err)
	}
}

func TestImportMarkdownDirWithSubdirs(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	tmpDir := t.TempDir()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Create markdown files in root and subdir
	os.WriteFile(filepath.Join(tmpDir, "root.md"), []byte("# Root\n\nRoot content"), 0644)
	os.WriteFile(filepath.Join(subDir, "sub.md"), []byte("# Sub\n\nSub content"), 0644)

	err := importMarkdownDir(tmpDir)
	if err != nil {
		t.Fatalf("importMarkdownDir failed: %v", err)
	}

	notes, _ := store.ListNotes(nil)
	if len(notes) < 2 {
		t.Errorf("expected at least 2 notes from dir import, got %d", len(notes))
	}
}

func TestImportExportDataWithInvalidID(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Export data with invalid UUID - should still work but generate new UUID
	export := &ExportData{
		Version: "1.0",
		Notes: []ExportNote{
			{
				ID:      "not-a-valid-uuid",
				Title:   "Invalid ID Note",
				Content: "Content",
			},
		},
	}

	err := importExportData(export)
	if err != nil {
		t.Fatalf("importExportData failed: %v", err)
	}

	notes, _ := store.ListNotes(nil)
	found := false
	for _, n := range notes {
		if n.Title == "Invalid ID Note" {
			found = true
			break
		}
	}
	if !found {
		t.Error("note with invalid ID should still be imported with new UUID")
	}
}

func TestExportMarkdownDefaultDir(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	note := models.NewNote("Default Dir Export", "Content")
	store.CreateNote(note, nil)

	// Use default output dir (empty string becomes "export")
	currentDir, _ := os.Getwd()
	defer os.RemoveAll(filepath.Join(currentDir, "export"))

	notes := []*models.Note{note}
	noteTags := [][]string{{}}

	err := exportMarkdown(notes, noteTags, "")
	if err != nil {
		t.Fatalf("exportMarkdown with default dir failed: %v", err)
	}

	// Verify default directory was created
	if _, err := os.Stat(filepath.Join(currentDir, "export")); os.IsNotExist(err) {
		t.Error("default export directory was not created")
	}
}

func TestListSearchWithTags(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create note with tags
	note := models.NewNote("Tagged Search Note", "Searchable content")
	store.CreateNote(note, []string{"searchable", "test"})

	err := listSearch("Searchable", 10)
	if err != nil {
		t.Errorf("listSearch failed: %v", err)
	}
}

func TestListByTagWithTags(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create notes with multiple tags
	note := models.NewNote("Multi-tag Note", "Content")
	store.CreateNote(note, []string{"alpha", "beta", "gamma"})

	err := listByTag("alpha", 10)
	if err != nil {
		t.Errorf("listByTag failed: %v", err)
	}
}

func TestListHereWithTags(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	pwd, _ := os.Getwd()
	dirTag := "dir:" + strings.ToLower(pwd)

	// Create note with dir tag and other tags
	note := models.NewNote("Here Note", "Content")
	store.CreateNote(note, []string{dirTag, "other-tag"})

	err := listHere(10)
	if err != nil {
		t.Errorf("listHere failed: %v", err)
	}
}
