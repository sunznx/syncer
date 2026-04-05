package syncengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunznx/syncer/internal/appdb"
)

// TestBackup_CrossStorageSymlink_NeedsAction tests that backup processes
// a symlink pointing to another storage — it should back up content and
// repoint the symlink to the current syncDir.
func TestBackup_CrossStorageSymlink_NeedsAction(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()
	otherSyncDir := t.TempDir() // simulates another storage (e.g. iCloud)

	// Create file in the "other" sync storage
	otherFile := filepath.Join(otherSyncDir, ".vimrc")
	if err := os.WriteFile(otherFile, []byte("vim config"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink in home pointing to the other storage
	homeFile := filepath.Join(home, ".vimrc")
	if err := os.Symlink(otherFile, homeFile); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "vim",
		Files: []string{".vimrc"},
	}

	engine := New(home, syncDir, WithCommand("backup"),
		WithProgressCallback(func(msg string) {
			t.Log("backup:", msg)
		}),
	)

	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Backup should process the file (symlink does NOT point to current syncDir)
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file (symlink points elsewhere, needs backup), got %d", len(result.Files))
	}

	// Verify content was copied to syncDir
	syncFile := filepath.Join(syncDir, ".vimrc")
	data, err := os.ReadFile(syncFile)
	if err != nil {
		t.Fatalf("sync file not found: %v", err)
	}
	if string(data) != "vim config" {
		t.Errorf("unexpected sync content: %q", string(data))
	}

	// Verify home symlink now points to current syncDir
	target, err := os.Readlink(homeFile)
	if err != nil {
		t.Fatal(err)
	}
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(homeFile), target)
	}
	if absTarget != syncFile {
		t.Errorf("symlink should point to current syncDir %q, got %q", syncFile, absTarget)
	}
}

// TestRestore_CrossStorageSymlink_NeedsAction tests that restore does NOT treat
// a symlink pointing to another storage as already synced — it should update
// the symlink to point to the current syncDir.
func TestRestore_CrossStorageSymlink_NeedsAction(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()
	otherSyncDir := t.TempDir() // simulates another storage (e.g. iCloud)

	// Create file in both sync storages
	for _, dir := range []string{syncDir, otherSyncDir} {
		if err := os.WriteFile(filepath.Join(dir, ".vimrc"), []byte("vim config"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create symlink in home pointing to the OTHER storage
	homeFile := filepath.Join(home, ".vimrc")
	if err := os.Symlink(filepath.Join(otherSyncDir, ".vimrc"), homeFile); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "vim",
		Files: []string{".vimrc"},
	}

	engine := New(home, syncDir, WithCommand("restore"),
		WithProgressCallback(func(msg string) {
			t.Log("restore:", msg)
		}),
	)

	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Restore should detect the symlink points elsewhere and update it
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file (symlink points to wrong storage, needs update), got %d", len(result.Files))
	}

	// Verify symlink now points to current syncDir
	target, err := os.Readlink(homeFile)
	if err != nil {
		t.Fatal(err)
	}
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(homeFile), target)
	}
	expectedTarget := filepath.Join(syncDir, ".vimrc")
	if absTarget != expectedTarget {
		t.Errorf("symlink should point to current syncDir %q, got %q", expectedTarget, absTarget)
	}
}

// TestRestore_AlreadySyncedCorrectly_NoOp tests that restore is a no-op when
// the symlink already points to the current syncDir.
func TestRestore_AlreadySyncedCorrectly_NoOp(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create file in syncDir
	syncFile := filepath.Join(syncDir, ".vimrc")
	if err := os.WriteFile(syncFile, []byte("vim config"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink in home pointing to current syncDir
	homeFile := filepath.Join(home, ".vimrc")
	if err := os.Symlink(syncFile, homeFile); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "vim",
		Files: []string{".vimrc"},
	}

	engine := New(home, syncDir, WithCommand("restore"))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("expected 0 files (already correctly synced), got %d", len(result.Files))
	}
}

// TestBackup_AlreadySyncedCorrectly_NoOp tests that backup is a no-op when
// the symlink already points to the current syncDir.
func TestBackup_AlreadySyncedCorrectly_NoOp(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create file in syncDir
	syncFile := filepath.Join(syncDir, ".vimrc")
	if err := os.WriteFile(syncFile, []byte("vim config"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink in home pointing to current syncDir
	homeFile := filepath.Join(home, ".vimrc")
	if err := os.Symlink(syncFile, homeFile); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "vim",
		Files: []string{".vimrc"},
	}

	engine := New(home, syncDir, WithCommand("backup"))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("expected 0 files (already correctly synced), got %d", len(result.Files))
	}
}

// TestRestore_DryRun_CrossStorageSymlink tests dry-run restore with cross-storage symlink.
func TestRestore_DryRun_CrossStorageSymlink(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()
	otherSyncDir := t.TempDir()

	// Create file in both storages
	for _, dir := range []string{syncDir, otherSyncDir} {
		if err := os.WriteFile(filepath.Join(dir, ".vimrc"), []byte("vim config"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Symlink points to other storage
	homeFile := filepath.Join(home, ".vimrc")
	otherTarget := filepath.Join(otherSyncDir, ".vimrc")
	if err := os.Symlink(otherTarget, homeFile); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "vim",
		Files: []string{".vimrc"},
	}

	var msgs []string
	engine := New(home, syncDir, WithCommand("restore"), WithDryRun(),
		WithProgressCallback(func(msg string) {
			msgs = append(msgs, msg)
		}),
	)

	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Dry-run should still report 1 file needing action
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file in dry-run, got %d", len(result.Files))
	}

	// Should have logged a "Would create symlink" message
	found := false
	for _, msg := range msgs {
		if strings.Contains(msg, "Would create symlink") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Would create symlink' in dry-run output")
	}

	// Symlink should NOT have changed (dry-run)
	target, _ := os.Readlink(homeFile)
	if target != otherTarget {
		t.Errorf("dry-run should not modify symlink, got target %q", target)
	}
}

// TestMigrateExternalSyncPath tests migrating an external symlink to a real file.
func TestMigrateExternalSyncPath(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create an external file
	externalFile := filepath.Join(home, "external", "data.conf")
	if err := os.MkdirAll(filepath.Dir(externalFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(externalFile, []byte("external data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside syncDir pointing outside
	syncPath := filepath.Join(syncDir, "data.conf")
	if err := os.Symlink(externalFile, syncPath); err != nil {
		t.Fatal(err)
	}

	engine := New(home, syncDir)
	migrated, err := engine.migrateExternalSyncPath(syncPath)
	if err != nil {
		t.Fatalf("migrateExternalSyncPath failed: %v", err)
	}
	if !migrated {
		t.Error("expected migration to happen for external symlink")
	}

	// Verify syncPath is now a regular file (not symlink)
	if fi, err := os.Lstat(syncPath); err != nil {
		t.Fatalf("syncPath not found: %v", err)
	} else if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("syncPath should be a regular file after migration, not a symlink")
	}

	// Verify content
	data, err := os.ReadFile(syncPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "external data" {
		t.Errorf("unexpected content: %q", string(data))
	}
}

// TestMigrateExternalSyncPath_InsideSyncDir_NoMigration tests that symlinks
// pointing inside syncDir are not migrated (prevents path traversal).
func TestMigrateExternalSyncPath_InsideSyncDir_NoMigration(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create a real file inside syncDir
	realFile := filepath.Join(syncDir, "real.conf")
	if err := os.WriteFile(realFile, []byte("real data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside syncDir pointing to another location inside syncDir
	linkPath := filepath.Join(syncDir, "subdir", "link.conf")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realFile, linkPath); err != nil {
		t.Fatal(err)
	}

	engine := New(home, syncDir)
	migrated, err := engine.migrateExternalSyncPath(linkPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Error("symlink inside syncDir should NOT be migrated")
	}
}

// TestMigrateExternalSyncPath_PathTraversal tests that path traversal attempts
// like syncer/../etc/passwd are correctly detected as inside syncDir.
func TestMigrateExternalSyncPath_PathTraversal(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create a real file outside but with path traversal-like path
	externalFile := filepath.Join(home, "secret.conf")
	if err := os.WriteFile(externalFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink that uses path traversal pattern
	syncPath := filepath.Join(syncDir, "traversal.conf")
	// Point to a path that would resolve inside syncDir via ..
	traversalTarget := filepath.Join(syncDir, "..", filepath.Base(home), "secret.conf")
	if err := os.Symlink(traversalTarget, syncPath); err != nil {
		t.Fatal(err)
	}

	engine := New(home, syncDir)
	migrated, err := engine.migrateExternalSyncPath(syncPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// filepath.Rel should resolve the .. and detect it's outside syncDir
	// so migration SHOULD happen (it points outside)
	if !migrated {
		t.Error("external symlink with path traversal should be migrated")
	}
}
