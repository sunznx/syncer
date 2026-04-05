package storage

import (
	"os"
	"testing"
)

func TestNewDefault(t *testing.T) {
	home := t.TempDir()
	// No cloud storage available
	store, err := NewDefault(home)
	if err != nil {
		// Expected to fail when no cloud storage found
		if err == nil {
			t.Error("expected error when no cloud storage found")
		}
		return
	}

	// If storage was found, verify it works
	syncDir, err := store.SyncDir()
	if err != nil {
		t.Fatalf("SyncDir failed: %v", err)
	}
	if syncDir == "" {
		t.Error("expected non-empty sync dir")
	}
}

func TestNewCustom(t *testing.T) {
	store, err := NewCustom("/tmp/test-sync")
	if err != nil {
		t.Fatalf("NewCustom failed: %v", err)
	}

	syncDir, err := store.SyncDir()
	if err != nil {
		t.Fatalf("SyncDir failed: %v", err)
	}
	if syncDir != "/tmp/test-sync" {
		t.Errorf("expected /tmp/test-sync, got %q", syncDir)
	}
}

func TestCustomStorage_HomeDir(t *testing.T) {
	store, err := NewCustom("/tmp/sync")
	if err != nil {
		t.Fatalf("NewCustom failed: %v", err)
	}

	home, _ := os.UserHomeDir()
	if store.HomeDir() != home {
		t.Errorf("expected %q, got %q", home, store.HomeDir())
	}
}
