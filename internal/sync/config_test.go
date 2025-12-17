// ABOUTME: Tests for sync configuration management
// ABOUTME: Verifies config loading, saving, and environment overrides

package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("ConfigPath should return absolute path, got %s", path)
	}
}

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	path := ConfigPath()
	if dir != filepath.Dir(path) {
		t.Errorf("ConfigDir() = %s, want %s", dir, filepath.Dir(path))
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg == nil {
		t.Fatal("defaultConfig returned nil")
	}
	if cfg.VaultDB == "" {
		t.Error("defaultConfig should set VaultDB")
	}
}

func TestConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected bool
	}{
		{
			name:     "empty config",
			cfg:      &Config{},
			expected: false,
		},
		{
			name: "partial config - missing derived key",
			cfg: &Config{
				Server: "https://api.example.com",
				Token:  "token123",
				UserID: "user123",
			},
			expected: false,
		},
		{
			name: "partial config - missing token",
			cfg: &Config{
				Server:     "https://api.example.com",
				UserID:     "user123",
				DerivedKey: "key123",
			},
			expected: false,
		},
		{
			name: "fully configured",
			cfg: &Config{
				Server:     "https://api.example.com",
				Token:      "token123",
				UserID:     "user123",
				DerivedKey: "key123",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsConfigured(); got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "absolute path unchanged",
			input:    "/foo/bar",
			expected: "/foo/bar",
		},
		{
			name:     "relative path unchanged",
			input:    "foo/bar",
			expected: "foo/bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandPath(tt.input); got != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}

	// Test home expansion separately since it depends on environment
	t.Run("tilde expansion", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("cannot get home dir")
		}
		got := expandPath("~/test")
		expected := filepath.Join(home, "test")
		if got != expected {
			t.Errorf("expandPath(\"~/test\") = %q, want %q", got, expected)
		}
	})
}

func TestApplyEnvOverrides(t *testing.T) {
	// Save original env vars
	origServer := os.Getenv("MEMO_SYNC_SERVER")
	origToken := os.Getenv("MEMO_SYNC_TOKEN")
	origAuto := os.Getenv("MEMO_SYNC_AUTO")
	defer func() {
		_ = os.Setenv("MEMO_SYNC_SERVER", origServer)
		_ = os.Setenv("MEMO_SYNC_TOKEN", origToken)
		_ = os.Setenv("MEMO_SYNC_AUTO", origAuto)
	}()

	t.Run("overrides from env", func(t *testing.T) {
		t.Setenv("MEMO_SYNC_SERVER", "https://test.example.com")
		t.Setenv("MEMO_SYNC_TOKEN", "testtoken")
		t.Setenv("MEMO_SYNC_AUTO", "true")

		cfg := &Config{}
		applyEnvOverrides(cfg)

		if cfg.Server != "https://test.example.com" {
			t.Errorf("Server = %q, want %q", cfg.Server, "https://test.example.com")
		}
		if cfg.Token != "testtoken" {
			t.Errorf("Token = %q, want %q", cfg.Token, "testtoken")
		}
		if !cfg.AutoSync {
			t.Error("AutoSync should be true")
		}
	})

	t.Run("auto sync with 1", func(t *testing.T) {
		t.Setenv("MEMO_SYNC_SERVER", "")
		t.Setenv("MEMO_SYNC_TOKEN", "")
		t.Setenv("MEMO_SYNC_AUTO", "1")

		cfg := &Config{}
		applyEnvOverrides(cfg)

		if !cfg.AutoSync {
			t.Error("AutoSync should be true when MEMO_SYNC_AUTO=1")
		}
	})
}

func TestLoadConfig(t *testing.T) {
	// Test loading non-existent config
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig should not error on missing config, got: %v", err)
	}
	if cfg == nil {
		t.Error("LoadConfig should return default config when file doesn't exist")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := &Config{
		Server:     "https://test.example.com",
		UserID:     "test-user",
		Token:      "test-token",
		DerivedKey: "test-key",
		DeviceID:   "test-device",
		VaultDB:    filepath.Join(tmpDir, "vault.db"),
		AutoSync:   true,
	}

	// Save config
	err := SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load it back
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Server != cfg.Server {
		t.Errorf("Server = %q, want %q", loaded.Server, cfg.Server)
	}
	if loaded.UserID != cfg.UserID {
		t.Errorf("UserID = %q, want %q", loaded.UserID, cfg.UserID)
	}
	if loaded.Token != cfg.Token {
		t.Errorf("Token = %q, want %q", loaded.Token, cfg.Token)
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create config
	cfg := defaultConfig()
	err := SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Now should exist
	if !ConfigExists() {
		t.Error("ConfigExists should return true after SaveConfig")
	}
}
