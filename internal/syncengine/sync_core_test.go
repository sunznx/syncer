package syncengine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunznx/syncer/internal/appdb"
	"github.com/sunznx/syncer/internal/fileops"
)

// TestSync_Backup_ExternalSymlink migrates external symlink to syncDir
func TestSync_Backup_ExternalSymlink(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create a cloud storage directory (Dropbox)
	cloudDir := filepath.Join(home, "Dropbox")
	if err := os.MkdirAll(cloudDir, 0755); err != nil {
		t.Fatal(err)
	}
	cloudConfigDir := filepath.Join(cloudDir, "config")
	if err := os.MkdirAll(cloudConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	cloudConfigFile := filepath.Join(cloudConfigDir, "app.conf")
	if err := os.WriteFile(cloudConfigFile, []byte("# app config"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create external symlink: ~/.config/app -> ~/Dropbox/config/app.conf
	homeConfigDir := filepath.Join(home, ".config")
	if err := os.MkdirAll(homeConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	homeConfigFile := filepath.Join(homeConfigDir, "app")
	if err := os.Symlink(cloudConfigDir, homeConfigFile); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "testapp",
		Files: []string{".config/app"},
	}

	engine := New(home, syncDir, WithCommand("backup"))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify content was copied to syncDir
	syncedConfigDir := filepath.Join(syncDir, ".config", "app")
	data, err := os.ReadFile(filepath.Join(syncedConfigDir, "app.conf"))
	if err != nil {
		t.Fatalf("synced config not found: %v", err)
	}
	if string(data) != "# app config" {
		t.Errorf("unexpected synced content: %q", string(data))
	}

	// Verify original symlink was replaced with symlink to syncDir
	homeConfigFile = filepath.Join(homeConfigDir, "app")
	target, err := os.Readlink(homeConfigFile)
	if err != nil {
		t.Fatalf("homeConfigFile is not a symlink: %v", err)
	}
	// Convert relative symlink to absolute for comparison
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(homeConfigFile), target)
	}
	if absTarget != syncedConfigDir {
		t.Errorf("symlink points to %q (abs: %q), want %q", target, absTarget, syncedConfigDir)
	}

	// Verify the file was processed
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file processed, got %d", len(result.Files))
	}
}

// TestSync_Backup_ExternalSymlinkAlreadyLinkedToSyncPath tests the scenario where:
// - homePath is a symlink pointing to syncPath
// - syncPath itself is an external symlink (pointing outside syncDir)
// Expected: content from external location should be copied to syncPath
func TestSync_Backup_ExternalSymlinkAlreadyLinkedToSyncPath(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create a cloud storage directory (Dropbox)
	cloudDir := filepath.Join(home, "Dropbox")
	if err := os.MkdirAll(cloudDir, 0755); err != nil {
		t.Fatal(err)
	}
	cloudConfigFile := filepath.Join(cloudDir, "app.conf")
	if err := os.WriteFile(cloudConfigFile, []byte("# app config"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create syncPath as an external symlink to Dropbox
	syncConfigFile := filepath.Join(syncDir, "app.conf")
	if err := os.Symlink(cloudConfigFile, syncConfigFile); err != nil {
		t.Fatal(err)
	}

	// Create homePath as a symlink to syncPath
	homeConfigFile := filepath.Join(home, "app.conf")
	if err := os.Symlink(syncConfigFile, homeConfigFile); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "testapp",
		Files: []string{"app.conf"},
	}

	// Test actual sync (not dry-run)
	engine := New(home, syncDir, WithCommand("backup"))
	result, err := engine.Sync(app)

	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify the file was processed
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file processed, got %d", len(result.Files))
	}

	// Verify content was copied from external location to syncPath
	data, err := os.ReadFile(syncConfigFile)
	if err != nil {
		t.Fatalf("failed to read syncPath: %v", err)
	}
	if string(data) != "# app config" {
		t.Errorf("unexpected content at syncPath: %q", string(data))
	}

	// Verify syncPath is now a regular file (not a symlink anymore)
	if fileops.IsSymlink(syncConfigFile) {
		t.Error("syncPath should be a regular file after migration, not a symlink")
	}

	// Verify homePath still points to syncPath location
	target, err := os.Readlink(homeConfigFile)
	if err != nil {
		t.Fatalf("homeConfigFile should still be a symlink: %v", err)
	}
	// Convert relative symlink to absolute for comparison
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(homeConfigFile), target)
	}
	if absTarget != syncConfigFile {
		t.Errorf("home symlink should point to syncPath: got %q (abs: %q), want %q", target, absTarget, syncConfigFile)
	}
}
