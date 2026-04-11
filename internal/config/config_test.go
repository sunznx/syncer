package config

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Config struct tests ---

func TestConfig_Defaults(t *testing.T) {
	cfg := Default()
	if cfg.HomeDir == "" {
		t.Error("expected non-empty HomeDir")
	}
	if cfg.Applications.Apps != nil {
		t.Errorf("expected nil Apps, got %v", cfg.Applications.Apps)
	}
}

func TestConfig_WithHomeDir(t *testing.T) {
	cfg := Default(WithHomeDir("/tmp/test"))
	if cfg.HomeDir != "/tmp/test" {
		t.Errorf("expected /tmp/test, got %q", cfg.HomeDir)
	}
}

// --- Load tests ---

func TestLoad_EnvVariable(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "syncer.yaml")
	content := []byte("applications:\n  apps:\n    - git\n")
	if err := os.WriteFile(cfgFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SYNCER_CONFIG", cfgFile)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ConfigFile != cfgFile {
		t.Errorf("expected ConfigFile %q, got %q", cfgFile, cfg.ConfigFile)
	}
}

func TestLoad_SpecifiedApps(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "syncer.yaml")
	content := []byte("applications:\n  apps:\n    - git\n    - zsh\n    - vscode\n")
	if err := os.WriteFile(cfgFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SYNCER_CONFIG", cfgFile)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Applications.Apps) != 3 {
		t.Fatalf("expected 3 apps, got %d", len(cfg.Applications.Apps))
	}
	expected := []string{"git", "zsh", "vscode"}
	for i, app := range expected {
		if cfg.Applications.Apps[i] != app {
			t.Errorf("expected app[%d] = %q, got %q", i, app, cfg.Applications.Apps[i])
		}
	}
}

func TestLoad_IgnoreApps(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "syncer.yaml")
	content := []byte("applications:\n  ignore:\n    - iterm2\n    - firefox\n")
	if err := os.WriteFile(cfgFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SYNCER_CONFIG", cfgFile)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Applications.Ignore) != 2 {
		t.Fatalf("expected 2 ignored apps, got %d", len(cfg.Applications.Ignore))
	}
}

func TestLoad_FallbackConfigDir(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "syncer")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(configDir, "syncer.yaml")
	content := []byte("applications:\n  apps:\n    - git\n")
	if err := os.WriteFile(cfgFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// No SYNCER_CONFIG set, no cloud storage config
	t.Setenv("SYNCER_CONFIG", "")
	cfg, err := Load(WithHomeDir(home))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ConfigFile != cfgFile {
		t.Errorf("expected ConfigFile %q, got %q", cfgFile, cfg.ConfigFile)
	}
}

func TestLoad_NoConfig_Defaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SYNCER_CONFIG", "")
	cfg, err := Load(WithHomeDir(home))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ConfigFile != "" {
		t.Errorf("expected empty ConfigFile, got %q", cfg.ConfigFile)
	}
}

func TestLoad_CloudStoragePath(t *testing.T) {
	home := t.TempDir()
	dropboxDir := filepath.Join(home, "Dropbox", "syncer")
	if err := os.MkdirAll(dropboxDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(dropboxDir, "syncer.yaml")
	content := []byte("applications:\n  apps:\n    - git\n")
	if err := os.WriteFile(cfgFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SYNCER_CONFIG", "")
	cfg, err := Load(WithHomeDir(home))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ConfigFile != cfgFile {
		t.Errorf("expected ConfigFile %q, got %q", cfgFile, cfg.ConfigFile)
	}
}

func TestLoad_EmptyAppsList(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "syncer.yaml")
	content := []byte("applications:\n  apps: []\n")
	if err := os.WriteFile(cfgFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SYNCER_CONFIG", cfgFile)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Applications.Apps) != 0 {
		t.Errorf("expected 0 apps, got %d", len(cfg.Applications.Apps))
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "syncer.yaml")
	content := []byte("applications:\n  apps: [unclosed bracket\n")
	if err := os.WriteFile(cfgFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SYNCER_CONFIG", cfgFile)
	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}
