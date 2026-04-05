package appdb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDB(t *testing.T) {
	syncersDir := t.TempDir()
	builtinDir := t.TempDir()

	db := NewDB(
		WithSyncersDir(syncersDir),
		WithBuiltinDir(builtinDir),
	)

	if db == nil {
		t.Fatal("NewDB returned nil")
	}
}

func TestList_MergesAllSources(t *testing.T) {
	syncersDir := t.TempDir()
	builtinDir := t.TempDir()

	// Create app in builtin dir
	builtinCfg := filepath.Join(builtinDir, "test.yaml")
	content := []byte("name: builtin-test\nfiles:\n  - .testrc\n")
	if err := os.WriteFile(builtinCfg, content, 0644); err != nil {
		t.Fatal(err)
	}

	db := NewDB(
		WithSyncersDir(syncersDir),
		WithBuiltinDir(builtinDir),
	)

	names := db.List()
	if len(names) == 0 {
		t.Error("expected at least one app")
	}

	// Should contain test (derived from filename test.yaml)
	found := false
	for _, name := range names {
		if name == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'test' in list, got %v", names)
	}
}

func TestLoad_PriorityOrder(t *testing.T) {
	syncersDir := t.TempDir()
	builtinDir := t.TempDir()

	// Create app in both locations
	syncersCfg := filepath.Join(syncersDir, "priority.yaml")
	content := []byte("name: priority\nfiles:\n  - .priorityrc\nmode: copy\n")
	if err := os.WriteFile(syncersCfg, content, 0644); err != nil {
		t.Fatal(err)
	}

	builtinCfg := filepath.Join(builtinDir, "priority.yaml")
	content = []byte("name: priority\nfiles:\n  - .priorityrc\nmode: link\n")
	if err := os.WriteFile(builtinCfg, content, 0644); err != nil {
		t.Fatal(err)
	}

	db := NewDB(
		WithSyncersDir(syncersDir),
		WithBuiltinDir(builtinDir),
	)

	app, err := db.Load("priority")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should use syncersDir (highest priority)
	if app.Mode != "copy" {
		t.Errorf("expected mode 'copy' from syncersDir, got %q", app.Mode)
	}
}

func TestLoad_Caching(t *testing.T) {
	syncersDir := t.TempDir()
	builtinDir := t.TempDir()

	// Create app in builtin dir
	builtinCfg := filepath.Join(builtinDir, "cached.yaml")
	content := []byte("name: cached\nfiles:\n  - .cachedrc\n")
	if err := os.WriteFile(builtinCfg, content, 0644); err != nil {
		t.Fatal(err)
	}

	db := NewDB(
		WithSyncersDir(syncersDir),
		WithBuiltinDir(builtinDir),
	)

	// Load same app twice
	app1, err1 := db.Load("cached")
	if err1 != nil {
		t.Fatalf("First Load failed: %v", err1)
	}

	app2, err2 := db.Load("cached")
	if err2 != nil {
		t.Fatalf("Second Load failed: %v", err2)
	}

	// Should return same instance (cached)
	if app1 != app2 {
		t.Error("expected cached instance")
	}
}
