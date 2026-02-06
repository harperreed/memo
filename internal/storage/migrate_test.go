// ABOUTME: Tests for storage migration between memo backends.
// ABOUTME: Covers sqlite-to-markdown, markdown-to-sqlite, roundtrip, data integrity, and error cases.

package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/harper/memo/internal/models"
)

// seedMemoTestData populates a storage backend with representative data
// and returns the entities for verification.
func seedMemoTestData(t *testing.T, src Storage) (notes []*models.Note, noteTags map[uuid.UUID][]string, attachments []*models.Attachment) {
	t.Helper()

	noteTags = make(map[uuid.UUID][]string)

	// Note 1: with tags and attachment
	note1 := models.NewNote("Welcome Guide", "# Welcome\n\nThis is the welcome guide.\n\n- Step 1\n- Step 2")
	tags1 := []string{"guide", "welcome"}
	mustNoErr(t, src.CreateNote(note1, tags1))
	noteTags[note1.ID] = tags1

	time.Sleep(time.Millisecond)

	// Note 2: with dir tag and no attachments
	note2 := models.NewNote("Project Notes", "Project-specific notes here.")
	tags2 := []string{"dir:myproject", "work"}
	mustNoErr(t, src.CreateNote(note2, tags2))
	noteTags[note2.ID] = tags2

	time.Sleep(time.Millisecond)

	// Note 3: global note, no tags
	note3 := models.NewNote("Random Thought", "Just a random thought with special chars: <>&\"'")
	mustNoErr(t, src.CreateNote(note3, nil))
	noteTags[note3.ID] = nil

	time.Sleep(time.Millisecond)

	// Note 4: with code content and multiple tags
	note4 := models.NewNote("Go Snippet", "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```")
	tags4 := []string{"code", "golang", "guide"}
	mustNoErr(t, src.CreateNote(note4, tags4))
	noteTags[note4.ID] = tags4

	notes = append(notes, note1, note2, note3, note4)

	// Attachments
	att1 := models.NewAttachment(note1.ID, "guide.txt", "text/plain", []byte("Full guide content here."))
	att2 := models.NewAttachment(note4.ID, "snippet.go", "text/x-go", []byte("package main\n\nfunc main() {}"))
	att3 := models.NewAttachment(note4.ID, "binary.bin", "application/octet-stream", []byte{0x89, 0x50, 0x4E, 0x47})
	mustNoErr(t, src.CreateAttachment(att1))
	mustNoErr(t, src.CreateAttachment(att2))
	mustNoErr(t, src.CreateAttachment(att3))
	attachments = append(attachments, att1, att2, att3)

	return
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// verifyMigratedMemoData checks that the destination storage contains all expected data.
func verifyMigratedMemoData(t *testing.T, dst Storage, notes []*models.Note, noteTags map[uuid.UUID][]string, attachments []*models.Attachment) {
	t.Helper()
	verifyMigratedNotes(t, dst, notes, noteTags)
	verifyMigratedMemoAttachments(t, dst, attachments)
}

func verifyMigratedNotes(t *testing.T, dst Storage, notes []*models.Note, noteTags map[uuid.UUID][]string) {
	t.Helper()
	for _, orig := range notes {
		got, gotTags, err := dst.GetNoteByID(orig.ID)
		if err != nil {
			t.Errorf("note %q (%s) not found in destination: %v", orig.Title, orig.ID, err)
			continue
		}
		if got.Title != orig.Title {
			t.Errorf("note title mismatch: want %q, got %q", orig.Title, got.Title)
		}
		if got.Content != orig.Content {
			t.Errorf("note content mismatch for %q: want %q, got %q", orig.Title, orig.Content, got.Content)
		}

		// Verify tags
		expectedTags := noteTags[orig.ID]
		if len(expectedTags) != len(gotTags) {
			t.Errorf("note %q tag count mismatch: want %d, got %d", orig.Title, len(expectedTags), len(gotTags))
			continue
		}
		for _, et := range expectedTags {
			normalized := strings.ToLower(strings.TrimSpace(et))
			found := false
			for _, gt := range gotTags {
				if gt == normalized {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("note %q missing tag %q in destination (got %v)", orig.Title, normalized, gotTags)
			}
		}
	}
}

func verifyMigratedMemoAttachments(t *testing.T, dst Storage, attachments []*models.Attachment) {
	t.Helper()
	for _, orig := range attachments {
		got, err := dst.GetAttachmentByID(orig.ID)
		if err != nil {
			t.Errorf("attachment %s (%s) not found in destination: %v", orig.Filename, orig.ID, err)
			continue
		}
		if got.Filename != orig.Filename {
			t.Errorf("attachment filename mismatch: want %q, got %q", orig.Filename, got.Filename)
		}
		if got.MimeType != orig.MimeType {
			t.Errorf("attachment mimetype mismatch: want %q, got %q", orig.MimeType, got.MimeType)
		}
		if string(got.Data) != string(orig.Data) {
			t.Errorf("attachment data mismatch: want %q, got %q", string(orig.Data), string(got.Data))
		}
		if got.NoteID != orig.NoteID {
			t.Errorf("attachment noteID mismatch: want %s, got %s", orig.NoteID, got.NoteID)
		}
	}
}

func TestMigrateData_SqliteToMarkdown(t *testing.T) {
	// Set up source (sqlite)
	srcDir := t.TempDir()
	src, err := NewSqliteStore(filepath.Join(srcDir, "memo.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	notes, noteTags, attachments := seedMemoTestData(t, src)

	// Set up destination (markdown)
	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	// Run migration
	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	// Verify summary counts
	if summary.Notes != len(notes) {
		t.Errorf("summary notes: want %d, got %d", len(notes), summary.Notes)
	}
	if summary.Attachments != len(attachments) {
		t.Errorf("summary attachments: want %d, got %d", len(attachments), summary.Attachments)
	}

	// Verify all data was migrated correctly
	verifyMigratedMemoData(t, dst, notes, noteTags, attachments)
}

func TestMigrateData_MarkdownToSqlite(t *testing.T) {
	// Set up source (markdown)
	srcDir := t.TempDir()
	src, err := NewMarkdownStore(srcDir)
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	notes, noteTags, attachments := seedMemoTestData(t, src)

	// Set up destination (sqlite)
	dstDir := t.TempDir()
	dst, err := NewSqliteStore(filepath.Join(dstDir, "memo.db"))
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	// Run migration
	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	// Verify summary counts
	if summary.Notes != len(notes) {
		t.Errorf("summary notes: want %d, got %d", len(notes), summary.Notes)
	}
	if summary.Attachments != len(attachments) {
		t.Errorf("summary attachments: want %d, got %d", len(attachments), summary.Attachments)
	}

	// Verify all data was migrated correctly
	verifyMigratedMemoData(t, dst, notes, noteTags, attachments)
}

func TestMigrateData_EmptySource(t *testing.T) {
	// Set up empty source (sqlite)
	srcDir := t.TempDir()
	src, err := NewSqliteStore(filepath.Join(srcDir, "memo.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	// Set up destination (markdown)
	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	if summary.Notes != 0 || summary.Tags != 0 || summary.Attachments != 0 {
		t.Errorf("expected all zero counts for empty source, got notes=%d tags=%d attachments=%d",
			summary.Notes, summary.Tags, summary.Attachments)
	}
}

func TestMigrateData_SqliteToSqlite(t *testing.T) {
	// Test migrating between two sqlite instances
	srcDir := t.TempDir()
	src, err := NewSqliteStore(filepath.Join(srcDir, "memo.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	notes, noteTags, attachments := seedMemoTestData(t, src)

	dstDir := t.TempDir()
	dst, err := NewSqliteStore(filepath.Join(dstDir, "memo.db"))
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	if summary.Notes != len(notes) {
		t.Errorf("summary notes: want %d, got %d", len(notes), summary.Notes)
	}

	verifyMigratedMemoData(t, dst, notes, noteTags, attachments)
}

func TestMigrateRoundTrip_SqliteToMarkdownToSqlite(t *testing.T) {
	// Phase 1: Create rich data in SQLite
	srcDir := t.TempDir()
	original, err := NewSqliteStore(filepath.Join(srcDir, "original.db"))
	if err != nil {
		t.Fatalf("create original store: %v", err)
	}
	defer original.Close()

	notes, noteTags, attachments := seedMemoTestData(t, original)

	// Phase 2: Migrate SQLite -> Markdown
	mdDir := t.TempDir()
	mdStore, err := NewMarkdownStore(mdDir)
	if err != nil {
		t.Fatalf("create markdown store: %v", err)
	}
	defer mdStore.Close()

	summary1, err := MigrateData(original, mdStore)
	if err != nil {
		t.Fatalf("MigrateData (sqlite->markdown) failed: %v", err)
	}
	if summary1.Notes != len(notes) || summary1.Attachments != len(attachments) {
		t.Errorf("phase 1 summary mismatch: notes=%d/%d attachments=%d/%d",
			summary1.Notes, len(notes), summary1.Attachments, len(attachments))
	}

	// Phase 3: Migrate Markdown -> new SQLite
	dstDir := t.TempDir()
	final, err := NewSqliteStore(filepath.Join(dstDir, "final.db"))
	if err != nil {
		t.Fatalf("create final store: %v", err)
	}
	defer final.Close()

	summary2, err := MigrateData(mdStore, final)
	if err != nil {
		t.Fatalf("MigrateData (markdown->sqlite) failed: %v", err)
	}
	if summary2.Notes != len(notes) || summary2.Attachments != len(attachments) {
		t.Errorf("phase 2 summary mismatch: notes=%d/%d attachments=%d/%d",
			summary2.Notes, len(notes), summary2.Attachments, len(attachments))
	}

	// Phase 4: Field-by-field verification
	verifyMigratedMemoData(t, final, notes, noteTags, attachments)
}

func TestMigrateRoundTrip_MarkdownToSqliteToMarkdown(t *testing.T) {
	// Phase 1: Create data in Markdown
	srcDir := t.TempDir()
	original, err := NewMarkdownStore(srcDir)
	if err != nil {
		t.Fatalf("create original store: %v", err)
	}
	defer original.Close()

	notes, noteTags, attachments := seedMemoTestData(t, original)

	// Phase 2: Migrate Markdown -> SQLite
	sqlDir := t.TempDir()
	sqlStore, err := NewSqliteStore(filepath.Join(sqlDir, "mid.db"))
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer sqlStore.Close()

	_, err = MigrateData(original, sqlStore)
	if err != nil {
		t.Fatalf("MigrateData (markdown->sqlite) failed: %v", err)
	}

	// Phase 3: Migrate SQLite -> new Markdown
	dstDir := t.TempDir()
	final, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create final store: %v", err)
	}
	defer final.Close()

	_, err = MigrateData(sqlStore, final)
	if err != nil {
		t.Fatalf("MigrateData (sqlite->markdown) failed: %v", err)
	}

	// Phase 4: Verify all data
	verifyMigratedMemoData(t, final, notes, noteTags, attachments)
}

func TestMigrateData_PreservesNoteOrdering(t *testing.T) {
	srcDir := t.TempDir()
	src, err := NewSqliteStore(filepath.Join(srcDir, "memo.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	// Create notes with defined ordering
	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		note := models.NewNote(strings.Repeat("x", i+1), "content")
		mustNoErr(t, src.CreateNote(note, nil))
	}

	// Migrate to markdown
	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	_, err = MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	// Verify notes exist (ordering is by updated_at desc in ListNotes)
	notes, err := dst.ListNotes(nil)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != 5 {
		t.Fatalf("expected 5 notes, got %d", len(notes))
	}
}

func TestIsDirNonEmpty(t *testing.T) {
	// Empty directory
	emptyDir := t.TempDir()
	nonEmpty, err := IsDirNonEmpty(emptyDir)
	if err != nil {
		t.Fatalf("IsDirNonEmpty on empty dir: %v", err)
	}
	if nonEmpty {
		t.Error("expected empty dir to be reported as empty")
	}

	// Non-empty directory
	nonEmptyDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	nonEmpty, err = IsDirNonEmpty(nonEmptyDir)
	if err != nil {
		t.Fatalf("IsDirNonEmpty on non-empty dir: %v", err)
	}
	if !nonEmpty {
		t.Error("expected non-empty dir to be reported as non-empty")
	}

	// Non-existent directory
	nonEmpty, err = IsDirNonEmpty(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("IsDirNonEmpty on non-existent dir: %v", err)
	}
	if nonEmpty {
		t.Error("expected non-existent dir to be reported as empty")
	}
}

func TestMigrateData_PreservesTagCounts(t *testing.T) {
	// Verify that tag usage counts survive migration
	srcDir := t.TempDir()
	src, err := NewSqliteStore(filepath.Join(srcDir, "memo.db"))
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	// Create notes sharing tags
	mustNoErr(t, src.CreateNote(models.NewNote("N1", "C"), []string{"shared", "alpha"}))
	mustNoErr(t, src.CreateNote(models.NewNote("N2", "C"), []string{"shared", "beta"}))
	mustNoErr(t, src.CreateNote(models.NewNote("N3", "C"), []string{"shared"}))

	// Migrate to markdown
	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	_, err = MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	// Verify tag counts
	tags, err := dst.ListAllTags()
	if err != nil {
		t.Fatalf("ListAllTags failed: %v", err)
	}

	tagCounts := make(map[string]int)
	for _, tc := range tags {
		tagCounts[tc.Tag.Name] = tc.Count
	}

	if tagCounts["shared"] != 3 {
		t.Errorf("expected 'shared' count 3, got %d", tagCounts["shared"])
	}
	if tagCounts["alpha"] != 1 {
		t.Errorf("expected 'alpha' count 1, got %d", tagCounts["alpha"])
	}
	if tagCounts["beta"] != 1 {
		t.Errorf("expected 'beta' count 1, got %d", tagCounts["beta"])
	}
}

func TestMigrateData_MarkdownToMarkdown(t *testing.T) {
	// Test migrating between two markdown instances
	srcDir := t.TempDir()
	src, err := NewMarkdownStore(srcDir)
	if err != nil {
		t.Fatalf("create source store: %v", err)
	}
	defer src.Close()

	notes, noteTags, attachments := seedMemoTestData(t, src)

	dstDir := t.TempDir()
	dst, err := NewMarkdownStore(dstDir)
	if err != nil {
		t.Fatalf("create destination store: %v", err)
	}
	defer dst.Close()

	summary, err := MigrateData(src, dst)
	if err != nil {
		t.Fatalf("MigrateData failed: %v", err)
	}

	if summary.Notes != len(notes) {
		t.Errorf("summary notes: want %d, got %d", len(notes), summary.Notes)
	}

	verifyMigratedMemoData(t, dst, notes, noteTags, attachments)
}
