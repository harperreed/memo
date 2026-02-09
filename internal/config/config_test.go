// ABOUTME: Tests for memo configuration management.
// ABOUTME: Covers backend selection, data dir expansion, config load/save, and OpenStorage factory.

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetBackend_Default(t *testing.T) {
	cfg := &Config{}
	if got := cfg.GetBackend(); got != "sqlite" {
		t.Errorf("expected default backend 'sqlite', got %q", got)
	}
}

func TestGetBackend_Configured(t *testing.T) {
	cfg := &Config{Backend: "markdown"}
	if got := cfg.GetBackend(); got != "markdown" {
		t.Errorf("expected backend 'markdown', got %q", got)
	}
}

func TestGetDataDir_Default(t *testing.T) {
	cfg := &Config{}
	got := cfg.GetDataDir()
	if got == "" {
		t.Error("expected non-empty default data dir")
	}
}

func TestGetDataDir_Configured(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/test-memo"}
	if got := cfg.GetDataDir(); got != "/tmp/test-memo" {
		t.Errorf("expected '/tmp/test-memo', got %q", got)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", home},
		{"~/subdir", filepath.Join(home, "subdir")},
		{"~/a/b/c", filepath.Join(home, "a", "b", "c")},
	}

	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLoadSave(t *testing.T) {
	// Use a temp dir as XDG_CONFIG_HOME and XDG_DATA_HOME
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Load should return markdown backend for new users (no existing SQLite DB)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load (missing file) failed: %v", err)
	}
	if cfg.Backend != "markdown" {
		t.Errorf("expected backend 'markdown' for new user, got %q", cfg.Backend)
	}

	// Verify config file was auto-created
	configPath := filepath.Join(tmpDir, "memo", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not auto-created on first load")
	}

	// Save config with custom values
	cfg.Backend = "markdown"
	cfg.DataDir = "~/memo-data"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load back
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load (after save) failed: %v", err)
	}
	if cfg2.Backend != "markdown" {
		t.Errorf("expected backend 'markdown', got %q", cfg2.Backend)
	}
	if cfg2.DataDir != "~/memo-data" {
		t.Errorf("expected data_dir '~/memo-data', got %q", cfg2.DataDir)
	}
}

func TestLoadExistingSQLiteUser(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Create a fake memo.db to simulate an existing SQLite user
	dataDir := filepath.Join(tmpDir, "memo")
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	dbPath := filepath.Join(dataDir, "memo.db")
	if err := os.WriteFile(dbPath, []byte("fake-db"), 0600); err != nil {
		t.Fatalf("failed to create fake memo.db: %v", err)
	}

	// Load should detect existing SQLite DB and return sqlite backend
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Backend != "sqlite" {
		t.Errorf("expected backend 'sqlite' for existing user, got %q", cfg.Backend)
	}
}

func TestLoadAutoCreatedConfigValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// Load to trigger auto-creation
	_, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Read the auto-created config and verify it's valid JSON with backend: "markdown"
	configPath := filepath.Join(tmpDir, "memo", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read auto-created config: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("auto-created config is not valid JSON: %v", err)
	}
	if cfg.Backend != "markdown" {
		t.Errorf("expected auto-created config backend 'markdown', got %q", cfg.Backend)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write invalid JSON
	configDir := filepath.Join(tmpDir, "memo")
	os.MkdirAll(configDir, 0750)
	os.WriteFile(filepath.Join(configDir, "config.json"), []byte("not json"), 0600)

	_, err := Load()
	if err == nil {
		t.Error("expected error loading invalid JSON config")
	}
}

func TestOpenStorage_Sqlite(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Backend: "sqlite",
		DataDir: tmpDir,
	}

	store, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("OpenStorage (sqlite) failed: %v", err)
	}
	defer store.Close()

	// Verify memo.db was created
	dbPath := filepath.Join(tmpDir, "memo.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("memo.db was not created")
	}
}

func TestOpenStorage_Markdown(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Backend: "markdown",
		DataDir: tmpDir,
	}

	store, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("OpenStorage (markdown) failed: %v", err)
	}
	defer store.Close()

	// Verify notes directory was created
	notesDir := filepath.Join(tmpDir, "notes")
	if _, err := os.Stat(notesDir); os.IsNotExist(err) {
		t.Error("notes directory was not created")
	}
}

func TestOpenStorage_Unknown(t *testing.T) {
	cfg := &Config{
		Backend: "unknown",
		DataDir: t.TempDir(),
	}

	_, err := cfg.OpenStorage()
	if err == nil {
		t.Error("expected error for unknown backend")
	}
}

func TestOpenStorage_DefaultBackend(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		DataDir: tmpDir,
	}

	store, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("OpenStorage (default) failed: %v", err)
	}
	defer store.Close()

	// Default should be sqlite
	dbPath := filepath.Join(tmpDir, "memo.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected sqlite backend by default, but memo.db was not created")
	}
}

func TestGetConfigPath(t *testing.T) {
	// With XDG_CONFIG_HOME set
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	got := GetConfigPath()
	expected := "/custom/config/memo/config.json"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestConfigJSONFormat(t *testing.T) {
	cfg := &Config{
		Backend: "markdown",
		DataDir: "~/custom/path",
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Backend != "markdown" {
		t.Errorf("expected backend 'markdown', got %q", decoded.Backend)
	}
	if decoded.DataDir != "~/custom/path" {
		t.Errorf("expected data_dir '~/custom/path', got %q", decoded.DataDir)
	}
}

func TestConfigOmitsEmpty(t *testing.T) {
	cfg := &Config{}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// With omitempty, empty config should be "{}"
	if string(data) != "{}" {
		t.Errorf("expected empty config to marshal as '{}', got %q", string(data))
	}
}

func TestGetDataDir_TildeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()
	cfg := &Config{DataDir: "~/my-memo"}
	got := cfg.GetDataDir()
	expected := filepath.Join(home, "my-memo")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
