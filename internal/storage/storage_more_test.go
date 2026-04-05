package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCustomEmptyPath(t *testing.T) {
	_, err := NewCustom("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestCustomStorageSyncersDir(t *testing.T) {
	store, err := NewCustom("/tmp/sync")
	if err != nil {
		t.Fatalf("NewCustom failed: %v", err)
	}

	syncersDir, err := store.SyncersDir()
	if err != nil {
		t.Fatalf("SyncersDir failed: %v", err)
	}
	expected := filepath.Join("/tmp", "sync", ".syncers")
	if syncersDir != expected {
		t.Errorf("SyncersDir = %q, want %q", syncersDir, expected)
	}
}

func TestNewDefaultWithMarker(t *testing.T) {
	home := t.TempDir()
	dropbox := filepath.Join(home, "Dropbox", "syncer")
	if err := os.MkdirAll(dropbox, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dropbox, "syncer.yaml")
	if err := os.WriteFile(marker, []byte("settings:\n"), 0644); err != nil {
		t.Fatal(err)
	}

	store, err := NewDefault(home)
	if err != nil {
		t.Fatalf("NewDefault failed: %v", err)
	}

	syncDir, err := store.SyncDir()
	if err != nil {
		t.Fatalf("SyncDir failed: %v", err)
	}
	if syncDir != dropbox {
		t.Errorf("SyncDir = %q, want %q", syncDir, dropbox)
	}
}

func TestNewDefaultAutoHome(t *testing.T) {
	// This will likely fail because no cloud storage exists in the real home dir,
	// but it exercises the homeDir=="" branch.
	_, err := NewDefault("")
	// Just verify it doesn't panic; error is expected in most environments.
	_ = err
}
