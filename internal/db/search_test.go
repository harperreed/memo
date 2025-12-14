// ABOUTME: Tests for FTS5 full-text search operations.
// ABOUTME: Validates search queries and result ranking.

package db

import (
	"path/filepath"
	"testing"

	"github.com/harper/memo/internal/models"
)

func TestSearchNotes(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	note1 := models.NewNote("Go Programming", "Learn about goroutines and channels")
	note2 := models.NewNote("Python Basics", "Introduction to Python programming")
	note3 := models.NewNote("Cooking Recipes", "How to make pasta")
	_ = CreateNote(db, note1)
	_ = CreateNote(db, note2)
	_ = CreateNote(db, note3)

	results, err := SearchNotes(db, "programming", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results for 'programming', got %d", len(results))
	}
}

func TestSearchNotesNoResults(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	note := models.NewNote("Test", "Content")
	_ = CreateNote(db, note)

	results, err := SearchNotes(db, "nonexistent", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchNotesRanking(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Note with term in title should rank higher
	note1 := models.NewNote("Other Topic", "This mentions golang once")
	note2 := models.NewNote("Golang Guide", "Golang golang golang everywhere")
	_ = CreateNote(db, note1)
	_ = CreateNote(db, note2)

	results, err := SearchNotes(db, "golang", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Note with more matches should be first
	if results[0].ID != note2.ID {
		t.Error("expected note with more matches to rank first")
	}
}
