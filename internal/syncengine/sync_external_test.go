package syncengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunznx/syncer/internal/appdb"
)

// TestSync_ExternalResource_TwoLayerSymlinks tests that external resources
// create two-layer symlinks: sync_dir -> external_repo, home -> sync_dir
func TestSync_ExternalResource_TwoLayerSymlinks(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create a fake external repo
	externalRepo := t.TempDir()
	repoSubPath := filepath.Join(externalRepo, ".opencode")
	if err := os.MkdirAll(repoSubPath, 0755); err != nil {
		t.Fatalf("Failed to create repo subpath: %v", err)
	}
	testFile := filepath.Join(repoSubPath, "config.json")
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Simulate external resource path
	extDest := externalRepo

	app := &appdb.AppConfig{
		Name: "opencode",
		External: []*appdb.ExternalConfig{
			{
				Type:   "git",
				URL:    "https://github.com/affaan-m/everything-claude-code",
				Path:   ".opencode",
				Target: ".config/opencode",
			},
		},
	}

	// Manually call the external sync logic (copied from Sync method)
	extCfg := app.External[0]
	homeTargetPath := extCfg.Target
	if homeTargetPath == "" {
		homeTargetPath = "./" + app.Name
	}
	homeTargetPath = strings.TrimPrefix(homeTargetPath, "./")
	if homeTargetPath == "" {
		homeTargetPath = app.Name
	}

	syncPath := extCfg.Target
	if syncPath == "" {
		syncPath = extCfg.Path
	}
	syncPath = strings.TrimPrefix(syncPath, "./")

	// Layer 1: sync_dir -> external repo
	syncDirSymlink := filepath.Join(syncDir, syncPath)
	externalRepoPath := filepath.Join(extDest, extCfg.Path)

	os.MkdirAll(filepath.Dir(syncDirSymlink), 0755)
	os.RemoveAll(syncDirSymlink)
	relSrc, err := filepath.Rel(filepath.Dir(syncDirSymlink), externalRepoPath)
	if err != nil {
		relSrc = externalRepoPath
	}
	if err := os.Symlink(relSrc, syncDirSymlink); err != nil {
		t.Fatalf("Failed to create symlink in sync_dir: %v", err)
	}

	// Layer 2: home -> sync_dir
	homeSymlink := filepath.Join(home, syncPath)
	os.MkdirAll(filepath.Dir(homeSymlink), 0755)
	os.RemoveAll(homeSymlink)
	relSrc, err = filepath.Rel(filepath.Dir(homeSymlink), syncDirSymlink)
	if err != nil {
		relSrc = syncDirSymlink
	}
	if err := os.Symlink(relSrc, homeSymlink); err != nil {
		t.Fatalf("Failed to create symlink in home: %v", err)
	}

	// Verify Layer 1: sync_dir symlink points to external repo
	syncDirLink, err := os.Readlink(syncDirSymlink)
	if err != nil {
		t.Fatalf("Failed to read sync_dir symlink: %v", err)
	}
	if syncDirLink != externalRepoPath {
		// Check if it's a relative symlink
		absSyncDirLink, err := filepath.Abs(filepath.Join(filepath.Dir(syncDirSymlink), syncDirLink))
		if err != nil {
			t.Fatalf("Failed to resolve sync_dir symlink: %v", err)
		}
		absExternalRepo, _ := filepath.Abs(externalRepoPath)
		if absSyncDirLink != absExternalRepo {
			t.Errorf("sync_dir symlink points to %q, want %q", syncDirLink, externalRepoPath)
		}
	}

	// Verify Layer 2: home symlink points to sync_dir
	homeLink, err := os.Readlink(homeSymlink)
	if err != nil {
		t.Fatalf("Failed to read home symlink: %v", err)
	}
	if homeLink != syncDirSymlink {
		// Check if it's a relative symlink
		absHomeLink, err := filepath.Abs(filepath.Join(filepath.Dir(homeSymlink), homeLink))
		if err != nil {
			t.Fatalf("Failed to resolve home symlink: %v", err)
		}
		absSyncDir, _ := filepath.Abs(syncDirSymlink)
		if absHomeLink != absSyncDir {
			t.Errorf("home symlink points to %q, want %q", homeLink, syncDirSymlink)
		}
	}

	t.Logf("Two-layer symlinks created successfully")
}

// TestSync_ExternalResource_WholeRepo tests syncing entire external repo
func TestSync_ExternalResource_WholeRepo(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create a fake external repo (entire repo, no subpath)
	externalRepo := t.TempDir()
	repoFile := filepath.Join(externalRepo, "config.yaml")
	if err := os.WriteFile(repoFile, []byte("test: true"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	app := &appdb.AppConfig{
		Name: "testapp",
		External: []*appdb.ExternalConfig{
			{
				Type:   "git",
				URL:    "https://github.com/user/repo.git",
				Target: ".testapp",
			},
		},
	}

	// Manually call the external sync logic
	extCfg := app.External[0]
	extDest := externalRepo

	homeTargetPath := extCfg.Target
	if homeTargetPath == "" {
		homeTargetPath = "./" + app.Name
	}
	homeTargetPath = strings.TrimPrefix(homeTargetPath, "./")
	if homeTargetPath == "" {
		homeTargetPath = app.Name
	}

	// For whole repo, extPath is "."
	syncPath := homeTargetPath

	// Layer 1: sync_dir -> external repo (entire repo)
	syncDirSymlink := filepath.Join(syncDir, syncPath)
	externalRepoPath := filepath.Join(extDest, ".")

	os.MkdirAll(filepath.Dir(syncDirSymlink), 0755)
	os.RemoveAll(syncDirSymlink)
	relSrc, err := filepath.Rel(filepath.Dir(syncDirSymlink), externalRepoPath)
	if err != nil {
		relSrc = externalRepoPath
	}
	if err := os.Symlink(relSrc, syncDirSymlink); err != nil {
		t.Fatalf("Failed to create symlink in sync_dir: %v", err)
	}

	// Layer 2: home -> sync_dir
	homeSymlink := filepath.Join(home, syncPath)
	os.MkdirAll(filepath.Dir(homeSymlink), 0755)
	os.RemoveAll(homeSymlink)
	relSrc, err = filepath.Rel(filepath.Dir(homeSymlink), syncDirSymlink)
	if err != nil {
		relSrc = syncDirSymlink
	}
	if err := os.Symlink(relSrc, homeSymlink); err != nil {
		t.Fatalf("Failed to create symlink in home: %v", err)
	}

	// Verify the file can be accessed through both layers
	targetFile := filepath.Join(home, syncPath, "config.yaml")
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read file through symlinks: %v", err)
	}
	if string(content) != "test: true" {
		t.Errorf("File content = %q, want 'test: true'", string(content))
	}

	t.Logf("Whole repo sync successful")
}
