// ABOUTME: Integration tests for memo CLI commands.
// ABOUTME: Tests full workflow from add to delete.

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var memoBin string

func TestMain(m *testing.M) {
	// Build memo binary
	cmd := exec.Command("go", "build", "-o", "bin/memo", "./cmd/memo")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	wd, _ := os.Getwd()
	memoBin = filepath.Join(wd, "..", "bin", "memo")

	os.Exit(m.Run())
}

func TestAddListShowDelete(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Add a note
	out, err := runMemo(dbPath, "add", "Test Note", "--content", "Test content here")
	if err != nil {
		t.Fatalf("add failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created note") {
		t.Errorf("expected 'Created note' in output: %s", out)
	}

	// List notes
	out, err = runMemo(dbPath, "list")
	if err != nil {
		t.Fatalf("list failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Test Note") {
		t.Errorf("expected 'Test Note' in list: %s", out)
	}

	// Extract ID prefix from list output
	lines := strings.Split(out, "\n")
	var idPrefix string
	for _, line := range lines {
		if strings.Contains(line, "Test Note") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				idPrefix = fields[0]
				break
			}
		}
	}

	if idPrefix == "" {
		t.Fatal("could not extract ID prefix")
	}

	// Show note
	out, err = runMemo(dbPath, "show", idPrefix)
	if err != nil {
		t.Fatalf("show failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Test content") {
		t.Errorf("expected 'Test content' in show: %s", out)
	}

	// Delete note
	out, err = runMemo(dbPath, "rm", idPrefix, "--force")
	if err != nil {
		t.Fatalf("rm failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Deleted") {
		t.Errorf("expected 'Deleted' in output: %s", out)
	}
}

func TestTagOperations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Add note with tags
	_, _ = runMemo(dbPath, "add", "Tagged Note", "--content", "Content", "--tags", "work,urgent")

	// List by tag
	out, _ := runMemo(dbPath, "list", "--tag", "work")
	if !strings.Contains(out, "Tagged Note") {
		t.Errorf("expected note in tag filter: %s", out)
	}

	// Tag list
	out, _ = runMemo(dbPath, "tag", "list")
	if !strings.Contains(out, "work") {
		t.Errorf("expected 'work' tag in list: %s", out)
	}
}

func TestSearch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	_, _ = runMemo(dbPath, "add", "Go Programming", "--content", "Learn about goroutines")
	_, _ = runMemo(dbPath, "add", "Cooking", "--content", "How to make pasta")

	out, _ := runMemo(dbPath, "list", "--search", "goroutines")
	if !strings.Contains(out, "Go Programming") {
		t.Errorf("expected 'Go Programming' in search: %s", out)
	}
	if strings.Contains(out, "Cooking") {
		t.Errorf("did not expect 'Cooking' in search: %s", out)
	}
}

func runMemo(dbPath string, args ...string) (string, error) {
	allArgs := append([]string{"--db", dbPath}, args...)
	cmd := exec.Command(memoBin, allArgs...) //nolint:gosec // Running our own test binary is expected in integration tests
	out, err := cmd.CombinedOutput()
	return string(out), err
}
