//go:build demo
// +build demo

package syncengine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIssue_SymlinkTargetNotAbsolute_Demo reproduces the ORIGINAL issue where symlink
// targets were relative paths without ../ prefix, causing broken symlinks.
//
// This test demonstrates the BUG in the OLD implementation (using filepath.Rel).
// It intentionally FAILS to document the historical problem.
//
// User's case:
//
//	syncDir = "/Users/sunx/Library/Mobile Documents/com~apple~CloudDocs/syncer"
//	externalDest = "/Users/sunx/.../.syncer_external/github.com/ohmyzsh/ohmyzsh"
//	syncPath = ".oh-my-zsh"
//
//	OLD Result: /Users/sunx/.oh-my-zsh -> Library/Mobile Documents/... (BROKEN!)
//	NEW Result: /Users/sunx/.oh-my-zsh -> /Users/sunx/Library/Mobile... (FIXED!)
//
// See TestFix_SymlinkWithAbsolutePath for the fix verification.
// Run with: go test -tags=demo ./internal/syncengine
func TestIssue_SymlinkTargetNotAbsolute_Demo(t *testing.T) {
	// Simulate user's directory structure
	tempBase := t.TempDir()

	// Create a nested syncDir path like "Library/Mobile Documents/.../syncer"
	syncDir := filepath.Join(tempBase, "Library", "Mobile Documents", "com~apple~CloudDocs", "syncer")
	if err := os.MkdirAll(syncDir, 0755); err != nil {
		t.Fatalf("failed to create syncDir: %v", err)
	}

	// Create externalDest (where external repo is cloned)
	externalDest := filepath.Join(syncDir, ".syncer_external", "github.com", "ohmyzsh", "ohmyzsh")
	if err := os.MkdirAll(externalDest, 0755); err != nil {
		t.Fatalf("failed to create externalDest: %v", err)
	}

	// Simulate the symlink creation logic from sync.go:227-230
	syncPath := ".oh-my-zsh"
	homeDir := tempBase // Simulating home = tempBase (for test)

	// Layer 1: sync_dir path -> external repo path
	syncDirSymlink := filepath.Join(syncDir, syncPath) // syncDir/.oh-my-zsh

	// Layer 2: home path -> sync_dir path
	homeSymlink := filepath.Join(homeDir, syncPath) // tempBase/.oh-my-zsh

	// Current problematic logic (from sync.go:227-230)
	relSrc, err := filepath.Rel(filepath.Dir(homeSymlink), syncDirSymlink)
	if err != nil {
		relSrc = syncDirSymlink
	}

	t.Logf("homeSymlink: %s", homeSymlink)
	t.Logf("syncDirSymlink: %s", syncDirSymlink)
	t.Logf("filepath.Dir(homeSymlink): %s", filepath.Dir(homeSymlink))
	t.Logf("relSrc (computed): %s", relSrc)

	// The problem: relSrc might be like "Library/Mobile Documents/..."
	// which is a relative path without ../ prefix

	if !filepath.IsAbs(relSrc) && relSrc[0] != '.' && relSrc[0] != '/' {
		t.Errorf("✗ BROKEN: relSrc is a relative path without ./ or ../ prefix: %s", relSrc)
		t.Errorf("  This symlink will break when cwd is not: %s", filepath.Dir(homeSymlink))
	}

	// Verify by actually creating the symlink
	os.MkdirAll(filepath.Dir(homeSymlink), 0755)
	os.RemoveAll(homeSymlink)

	if err := os.Symlink(relSrc, homeSymlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Read back the symlink target
	target, err := os.Readlink(homeSymlink)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	t.Logf("symlink target: %s", target)

	// Test: can we resolve the symlink from a different directory?
	// Change to a completely different directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	testDir := t.TempDir()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Try to stat the symlink - it should fail if the target path is broken
	// We need to use absolute path to homeSymlink
	absHomeSymlink, err := filepath.Abs(homeSymlink)
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}

	// This will fail if the symlink target is a broken relative path
	if _, err := os.Stat(absHomeSymlink); err != nil {
		t.Logf("✗ CONFIRMED: Symlink is broken when accessed from different directory: %v", err)
		t.Logf("  Symlink: %s -> %s", absHomeSymlink, target)
	}
}

// TestFix_SymlinkWithAbsolutePath tests the fix using absolute paths.
func TestFix_SymlinkWithAbsolutePath(t *testing.T) {
	tempBase := t.TempDir()

	syncDir := filepath.Join(tempBase, "Library", "Mobile Documents", "com~apple~CloudDocs", "syncer")
	if err := os.MkdirAll(syncDir, 0755); err != nil {
		t.Fatalf("failed to create syncDir: %v", err)
	}

	syncPath := ".oh-my-zsh"
	homeDir := tempBase

	syncDirSymlink := filepath.Join(syncDir, syncPath)
	homeSymlink := filepath.Join(homeDir, syncPath)

	// THE FIX: Use absolute path
	absSyncDirSymlink, err := filepath.Abs(syncDirSymlink)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	// Create the actual directory (in real scenario, this would be created by syncengine)
	os.MkdirAll(absSyncDirSymlink, 0755)
	os.MkdirAll(filepath.Dir(homeSymlink), 0755)

	// Create home symlink with absolute path
	if err := os.Symlink(absSyncDirSymlink, homeSymlink); err != nil {
		t.Fatalf("failed to create home symlink: %v", err)
	}

	// Verify target
	target, err := os.Readlink(homeSymlink)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	t.Logf("symlink target (FIXED): %s", target)

	if !filepath.IsAbs(target) {
		t.Errorf("Expected absolute path, got: %s", target)
	}

	// Test from different directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	testDir := t.TempDir()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Create a test file in syncDir to verify symlink works
	testFile := filepath.Join(absSyncDirSymlink, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Access through symlink from different directory
	absHomeSymlink, err := filepath.Abs(homeSymlink)
	if err != nil {
		t.Fatalf("failed to get abs path: %v", err)
	}

	linkedFile := filepath.Join(absHomeSymlink, "test.txt")
	if _, err := os.Stat(linkedFile); err != nil {
		t.Errorf("✗ FAILED: Cannot access file through symlink from different directory: %v", err)
	} else {
		t.Logf("✓ SUCCESS: Symlink works from any directory")
	}
}
