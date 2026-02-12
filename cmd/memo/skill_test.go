// ABOUTME: Tests for the install-skill command functionality.
// ABOUTME: Validates skill installation, directory creation, and file content.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallSkillWithYesFlag(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	tmpHome := t.TempDir()

	// Set the skip confirmation flag to simulate --yes
	oldSkipConfirm := skillSkipConfirm
	skillSkipConfirm = true
	defer func() { skillSkipConfirm = oldSkipConfirm }()

	// Run installation
	err := installSkillWithHome(tmpHome)
	if err != nil {
		t.Fatalf("installSkillWithHome failed: %v", err)
	}

	// Verify the file was created
	skillPath := filepath.Join(tmpHome, ".claude", "skills", "memo", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("SKILL.md was not created")
	}
}

func TestInstallSkillCreatesDirectories(t *testing.T) {
	tmpHome := t.TempDir()

	oldSkipConfirm := skillSkipConfirm
	skillSkipConfirm = true
	defer func() { skillSkipConfirm = oldSkipConfirm }()

	err := installSkillWithHome(tmpHome)
	if err != nil {
		t.Fatalf("installSkillWithHome failed: %v", err)
	}

	// Verify all directories in the path were created
	expectedDirs := []string{
		filepath.Join(tmpHome, ".claude"),
		filepath.Join(tmpHome, ".claude", "skills"),
		filepath.Join(tmpHome, ".claude", "skills", "memo"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist", dir)
			continue
		}
		if err != nil {
			t.Errorf("error checking directory %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}
}

func TestInstallSkillContent(t *testing.T) {
	tmpHome := t.TempDir()

	oldSkipConfirm := skillSkipConfirm
	skillSkipConfirm = true
	defer func() { skillSkipConfirm = oldSkipConfirm }()

	err := installSkillWithHome(tmpHome)
	if err != nil {
		t.Fatalf("installSkillWithHome failed: %v", err)
	}

	skillPath := filepath.Join(tmpHome, ".claude", "skills", "memo", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}

	// Verify expected content is present
	contentStr := string(content)

	expectedSubstrings := []string{
		"name: memo",
		"description:",
		"# memo",
		"mcp__notes__add_note",
		"mcp__notes__search_notes",
		"memo add",
		"memo list",
		"memo list -s",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("SKILL.md should contain %q", expected)
		}
	}

	// Verify the embedded content matches what we read
	embeddedContent, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		t.Fatalf("failed to read embedded skill: %v", err)
	}

	if string(content) != string(embeddedContent) {
		t.Error("installed SKILL.md content does not match embedded content")
	}
}

func TestInstallSkillOverwrite(t *testing.T) {
	tmpHome := t.TempDir()

	oldSkipConfirm := skillSkipConfirm
	skillSkipConfirm = true
	defer func() { skillSkipConfirm = oldSkipConfirm }()

	skillDir := filepath.Join(tmpHome, ".claude", "skills", "memo")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Pre-create directory and file with different content
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	oldContent := []byte("# Old content that should be overwritten")
	if err := os.WriteFile(skillPath, oldContent, 0644); err != nil {
		t.Fatalf("failed to write initial SKILL.md: %v", err)
	}

	// Verify old file exists
	beforeContent, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read initial SKILL.md: %v", err)
	}
	if string(beforeContent) != string(oldContent) {
		t.Fatal("initial content was not written correctly")
	}

	// Run installation (should overwrite)
	err = installSkillWithHome(tmpHome)
	if err != nil {
		t.Fatalf("installSkillWithHome failed: %v", err)
	}

	// Verify content was overwritten
	afterContent, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md after install: %v", err)
	}

	if string(afterContent) == string(oldContent) {
		t.Error("SKILL.md content was not overwritten")
	}

	// Verify it contains the expected new content
	if !strings.Contains(string(afterContent), "name: memo") {
		t.Error("SKILL.md should contain expected skill content after overwrite")
	}
}

func TestInstallSkillFilePermissions(t *testing.T) {
	tmpHome := t.TempDir()

	oldSkipConfirm := skillSkipConfirm
	skillSkipConfirm = true
	defer func() { skillSkipConfirm = oldSkipConfirm }()

	err := installSkillWithHome(tmpHome)
	if err != nil {
		t.Fatalf("installSkillWithHome failed: %v", err)
	}

	skillPath := filepath.Join(tmpHome, ".claude", "skills", "memo", "SKILL.md")
	info, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("failed to stat SKILL.md: %v", err)
	}

	// Check file permissions (0644)
	mode := info.Mode().Perm()
	// On some systems umask might affect this, so check it's at least readable
	if mode&0400 == 0 {
		t.Error("SKILL.md should be readable by owner")
	}
	if mode&0200 == 0 {
		t.Error("SKILL.md should be writable by owner")
	}
}

func TestInstallSkillDirectoryPermissions(t *testing.T) {
	tmpHome := t.TempDir()

	oldSkipConfirm := skillSkipConfirm
	skillSkipConfirm = true
	defer func() { skillSkipConfirm = oldSkipConfirm }()

	err := installSkillWithHome(tmpHome)
	if err != nil {
		t.Fatalf("installSkillWithHome failed: %v", err)
	}

	skillDir := filepath.Join(tmpHome, ".claude", "skills", "memo")
	info, err := os.Stat(skillDir)
	if err != nil {
		t.Fatalf("failed to stat skill directory: %v", err)
	}

	// Check directory permissions
	mode := info.Mode().Perm()
	if mode&0700 == 0 {
		t.Error("skill directory should have rwx permissions for owner")
	}
}
