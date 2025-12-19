// ABOUTME: Integration tests for memo CLI commands.
// ABOUTME: Tests require a running Charm server or CHARM_DATA_DIR to be set.

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

// skipIfNoCharm skips the test if Charm is not configured for testing.
func skipIfNoCharm(t *testing.T) {
	t.Helper()
	// Tests require CHARM_DATA_DIR to be set to a temp directory
	// or a running charm server. For CI, use the testserver package.
	if os.Getenv("CHARM_DATA_DIR") == "" && os.Getenv("CHARM_HOST") == "" {
		t.Skip("Skipping: set CHARM_DATA_DIR or CHARM_HOST for integration tests")
	}
}

func TestAddListShowDelete(t *testing.T) {
	skipIfNoCharm(t)

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
	skipIfNoCharm(t)

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
	skipIfNoCharm(t)

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

func runMemo(args ...string) (string, error) {
	cmd := exec.Command(memoBin, args...) //nolint:gosec // Running our own test binary is expected in integration tests
	out, err := cmd.CombinedOutput()
	return string(out), err
}
