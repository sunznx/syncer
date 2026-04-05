package appdb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_SyncersDirOverrides(t *testing.T) {
	syncersDir := t.TempDir()
	builtinDir := t.TempDir()

	// Create app in syncers dir (highest priority)
	syncersCfg := filepath.Join(syncersDir, "test.yaml")
	content := []byte("name: Test\nfiles:\n  - .testrc\n")
	if err := os.WriteFile(syncersCfg, content, 0644); err != nil {
		t.Fatal(err)
	}

	db := NewDB(
		WithSyncersDir(syncersDir),
		WithBuiltinDir(builtinDir),
	)

	app, err := db.Load("test")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if app.Name != "Test" {
		t.Errorf("expected app name 'Test', got %q", app.Name)
	}
}

func TestLoad_BuiltinFallback(t *testing.T) {
	syncersDir := t.TempDir()
	builtinDir := t.TempDir()

	// Create app in builtin dir
	builtinCfg := filepath.Join(builtinDir, "myapp.yaml")
	content := []byte("name: MyApp\nfiles:\n  - .myapprc\n")
	if err := os.WriteFile(builtinCfg, content, 0644); err != nil {
		t.Fatal(err)
	}

	db := NewDB(
		WithSyncersDir(syncersDir),
		WithBuiltinDir(builtinDir),
	)

	app, err := db.Load("myapp")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if app.Name != "MyApp" {
		t.Errorf("expected app name 'MyApp', got %q", app.Name)
	}
}

func TestParseYAML(t *testing.T) {
	yamlContent := `
name: TestApp
files:
  - .config/test
  - .bashrc
mode: link
ignore:
  - "*.tmp"
`
	app, err := ParseYAML(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("ParseYAML failed: %v", err)
	}

	if app.Name != "TestApp" {
		t.Errorf("expected name 'TestApp', got %q", app.Name)
	}

	if len(app.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(app.Files))
	}

	if !app.IsLinkMode() {
		t.Error("expected link mode")
	}

	if len(app.Ignore) != 1 {
		t.Errorf("expected 1 ignore pattern, got %d", len(app.Ignore))
	}
}
