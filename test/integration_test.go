// ABOUTME: Integration tests for memo CLI commands.
// ABOUTME: Tests use a temporary database directory for isolation.

package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var memoBin string
var testDataDir string

func TestMain(m *testing.M) {
	// Build memo binary
	cmd := exec.Command("go", "build", "-o", "bin/memo", "./cmd/memo")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	wd, _ := os.Getwd()
	memoBin = filepath.Join(wd, "..", "bin", "memo")

	// Create temp directory for test data
	var err error
	testDataDir, err = os.MkdirTemp("", "memo-integration-test-*")
	if err != nil {
		panic(err)
	}

	// Set XDG_DATA_HOME to use temp directory
	os.Setenv("XDG_DATA_HOME", testDataDir)

	code := m.Run()

	// Cleanup
	os.RemoveAll(testDataDir)

	os.Exit(code)
}

func TestAddListShowDelete(t *testing.T) {
	// Add a note
	out, err := runMemo("add", "Test Note", "--content", "Test content here")
	if err != nil {
		t.Fatalf("add failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created note") {
		t.Errorf("expected 'Created note' in output: %s", out)
	}

	// List notes
	out, err = runMemo("list")
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
	out, err = runMemo("show", idPrefix)
	if err != nil {
		t.Fatalf("show failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Test content") {
		t.Errorf("expected 'Test content' in show: %s", out)
	}

	// Delete note
	out, err = runMemo("rm", idPrefix, "--force")
	if err != nil {
		t.Fatalf("rm failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Deleted") {
		t.Errorf("expected 'Deleted' in output: %s", out)
	}
}

func TestTagOperations(t *testing.T) {
	// Add note with tags
	_, _ = runMemo("add", "Tagged Note", "--content", "Content", "--tags", "work,urgent")

	// List by tag
	out, _ := runMemo("list", "--tag", "work")
	if !strings.Contains(out, "Tagged Note") {
		t.Errorf("expected note in tag filter: %s", out)
	}

	// Tag list
	out, _ = runMemo("tag", "list")
	if !strings.Contains(out, "work") {
		t.Errorf("expected 'work' tag in list: %s", out)
	}
}

func TestSearch(t *testing.T) {
	_, _ = runMemo("add", "Go Programming", "--content", "Learn about goroutines")
	_, _ = runMemo("add", "Cooking", "--content", "How to make pasta")

	out, _ := runMemo("list", "--search", "goroutines")
	if !strings.Contains(out, "Go Programming") {
		t.Errorf("expected 'Go Programming' in search: %s", out)
	}
	if strings.Contains(out, "Cooking") {
		t.Errorf("did not expect 'Cooking' in search: %s", out)
	}
}

func TestExportImport(t *testing.T) {
	// Add a note
	_, _ = runMemo("add", "Export Test", "--content", "Content for export", "--tags", "export")

	// Export to JSON
	exportPath := filepath.Join(testDataDir, "export.json")
	out, err := runMemo("export", "--format", "json", "-o", exportPath)
	if err != nil {
		t.Fatalf("export failed: %v\n%s", err, out)
	}

	// Check export file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("export file was not created")
	}

	// Export to YAML
	yamlPath := filepath.Join(testDataDir, "export.yaml")
	out, err = runMemo("export", "--format", "yaml", "-o", yamlPath)
	if err != nil {
		t.Fatalf("yaml export failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Error("yaml export file was not created")
	}
}

func runMemo(args ...string) (string, error) {
	cmd := exec.Command(memoBin, args...) //nolint:gosec // Running our own test binary is expected in integration tests
	out, err := cmd.CombinedOutput()
	return string(out), err
}
