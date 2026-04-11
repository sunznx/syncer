package syncengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSymlinkAbsolutePath verifies that symlinks created in home directory
// use absolute paths or valid relative paths that work from any directory.
func TestSymlinkAbsolutePath(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Simulate iCloud sync directory path (like "Library/Mobile Documents/...")
	syncSubDir := filepath.Join(syncDir, ".oh-my-zsh")
	if err := os.MkdirAll(syncSubDir, 0755); err != nil {
		t.Fatalf("failed to create sync subdirectory: %v", err)
	}

	// Create a file in sync directory
	testFile := filepath.Join(syncSubDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create symlink in home directory
	homeSymlink := filepath.Join(home, ".oh-my-zsh")

	// Current implementation (problematic)
	relSrc, err := filepath.Rel(filepath.Dir(homeSymlink), syncSubDir)
	if err != nil {
		relSrc = syncSubDir
	}

	if err := os.Symlink(relSrc, homeSymlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Verify symlink target
	target, err := os.Readlink(homeSymlink)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	// Check if target is absolute or starts with ..
	// If it's a relative path without ../ prefix, it's broken
	if filepath.IsAbs(target) {
		t.Logf("✓ Target is absolute path: %s", target)
	} else if strings.HasPrefix(target, "../") {
		t.Logf("✓ Target is valid relative path: %s", target)
	} else {
		t.Errorf("✗ Target is broken relative path: %s (must be absolute or start with ../)", target)
	}

	// Test that symlink works from different directories
	testDirs := []string{
		home,
		filepath.Join(home, "subdir"),
		os.TempDir(),
	}

	for _, testDir := range testDirs {
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Logf("Skipping test in %s: %v", testDir, err)
			continue
		}

		// Get absolute path to verify file exists
		absTarget, err := filepath.EvalSymlinks(homeSymlink)
		if err != nil {
			t.Errorf("Symlink broken when accessed from %s: %v", testDir, err)
			continue
		}

		// Verify the resolved path points to an existing file
		if _, err := os.Stat(absTarget); err != nil {
			t.Errorf("Cannot access file through symlink from %s: %v", testDir, err)
		} else {
			t.Logf("✓ Symlink works from %s", testDir)
		}
	}
}

// TestSymlinkCorrectness verifies the fix using absolute paths.
func TestSymlinkCorrectness(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Simulate directory structure
	syncSubDir := filepath.Join(syncDir, ".oh-my-zsh")
	if err := os.MkdirAll(syncSubDir, 0755); err != nil {
		t.Fatalf("failed to create sync subdirectory: %v", err)
	}

	// Create symlink using ABSOLUTE path (the fix)
	homeSymlink := filepath.Join(home, ".oh-my-zsh")

	// Convert to absolute path explicitly
	absSyncSubDir, err := filepath.Abs(syncSubDir)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	if err := os.Symlink(absSyncSubDir, homeSymlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Verify symlink target
	target, err := os.Readlink(homeSymlink)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	if !filepath.IsAbs(target) {
		t.Errorf("Expected absolute path, got: %s", target)
	}

	// Verify symlink works
	testFile := filepath.Join(syncSubDir, "oh-my-zsh.sh")
	if err := os.WriteFile(testFile, []byte("#!/bin/sh"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Access through symlink
	linkedFile := filepath.Join(homeSymlink, "oh-my-zsh.sh")
	if _, err := os.Stat(linkedFile); err != nil {
		t.Errorf("Cannot access file through symlink: %v", err)
	}
}
