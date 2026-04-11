package syncengine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunznx/syncer/internal/appdb"
	"github.com/sunznx/syncer/internal/fileops"
)

func TestSync_DirectoryWithFiles(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// 创建一个测试目录，包含文件
	testDir := filepath.Join(home, ".testdir")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	testFile := filepath.Join(testDir, "config.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	app := &appdb.AppConfig{
		Name:  "testdir",
		Files: []string{".testdir"},
	}

	engine := New(home, syncDir, WithCommand("backup"), WithProgressCallback(func(msg string) {
		t.Log("SYNC:", msg)
	}))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	t.Logf("Result: %v files processed", len(result.Files))

	if !result.App.IsLinkMode() {
		t.Errorf("expected ModeLink, got copy mode")
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file processed, got %d", len(result.Files))
	}

	// 验证目录已复制到 syncDir
	syncedDir := filepath.Join(syncDir, ".testdir")
	info, err := os.Stat(syncedDir)
	if err != nil {
		t.Fatalf("synced dir not found: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("synced path is not a directory")
	}

	// 验证目录中的文件也被复制了
	syncedFile := filepath.Join(syncedDir, "config.txt")
	data, err := os.ReadFile(syncedFile)
	if err != nil {
		t.Fatalf("synced file not found: %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("unexpected content: %q", string(data))
	}

	// 验证 home 目录中的目录已替换为符号链接
	homeInfo, err := os.Lstat(filepath.Join(home, ".testdir"))
	if err != nil {
		t.Fatalf("home path not found: %v", err)
	}
	if homeInfo.Mode()&os.ModeSymlink == 0 {
		t.Error(".testdir should be a symlink after sync")
	}
}

// TestSync_CopyMode tests copy mode (no symlinks)
func TestSync_CopyMode(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create a file in home
	homeFile := filepath.Join(home, "test.conf")
	if err := os.WriteFile(homeFile, []byte("config content"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "testapp",
		Files: []string{"test.conf"},
		Mode:  "copy",
	}

	engine := New(home, syncDir, WithCommand("backup"))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Files) != 1 {
		t.Errorf("expected 1 file processed, got %d", len(result.Files))
	}

	// Verify file was copied to syncDir
	syncFile := filepath.Join(syncDir, "test.conf")
	data, err := os.ReadFile(syncFile)
	if err != nil {
		t.Fatalf("sync file not found: %v", err)
	}
	if string(data) != "config content" {
		t.Errorf("unexpected content: %q", string(data))
	}

	// Verify home file is still a regular file (not a symlink)
	if fileops.IsSymlink(homeFile) {
		t.Error("home file should not be a symlink in copy mode")
	}
}

// TestSync_RestoreFromCloud tests restoring from syncDir to home
func TestSync_RestoreFromCloud(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create file in syncDir (simulating cloud backup)
	syncFile := filepath.Join(syncDir, "restore.conf")
	if err := os.WriteFile(syncFile, []byte("cloud content"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "restoreapp",
		Files: []string{"restore.conf"},
	}

	engine := New(home, syncDir, WithCommand("restore"))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Files) != 1 {
		t.Errorf("expected 1 file processed, got %d", len(result.Files))
	}

	// Verify file was restored to home
	homeFile := filepath.Join(home, "restore.conf")
	data, err := os.ReadFile(homeFile)
	if err != nil {
		t.Fatalf("home file not found: %v", err)
	}
	if string(data) != "cloud content" {
		t.Errorf("unexpected content: %q", string(data))
	}

	// In link mode, home should be a symlink
	target, err := os.Readlink(homeFile)
	if err != nil {
		t.Errorf("home file should be a symlink in link mode: %v", err)
	}
	// Convert relative symlink to absolute for comparison
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(homeFile), target)
	}
	if absTarget != syncFile {
		t.Errorf("symlink should point to syncDir: got %q (abs: %q), want %q", target, absTarget, syncFile)
	}
}

// TestSync_DryRun_DoesNotModifyFiles tests that dry-run mode doesn't modify anything
func TestSync_DryRun_DoesNotModifyFiles(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create initial state: file in home
	homeFile := filepath.Join(home, "dryrun.conf")
	originalContent := []byte("original content")
	if err := os.WriteFile(homeFile, originalContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Get file info to verify it doesn't change
	beforeInfo, err := os.Stat(homeFile)
	if err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "dryrun",
		Files: []string{"dryrun.conf"},
	}

	engine := New(home, syncDir, WithCommand("backup"), WithDryRun())
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// In dry-run mode, file should be "processed" but not modified
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file processed in dry-run, got %d", len(result.Files))
	}

	// Verify home file wasn't modified
	afterInfo, err := os.Stat(homeFile)
	if err != nil {
		t.Fatalf("home file was removed in dry-run mode: %v", err)
	}
	if beforeInfo.ModTime() != afterInfo.ModTime() {
		t.Error("home file was modified in dry-run mode (mod time changed)")
	}

	// Verify syncDir doesn't have the file
	syncFile := filepath.Join(syncDir, "dryrun.conf")
	if _, err := os.Stat(syncFile); !os.IsNotExist(err) {
		t.Error("syncDir should not have files in dry-run mode")
	}
}

// TestSync_BothExist_ResyncsContent tests resyncing when both home and syncDir have files
func TestSync_BothExist_ResyncsContent(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create file in syncDir with old content
	syncFile := filepath.Join(syncDir, "resync.conf")
	if err := os.WriteFile(syncFile, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create file in home with new content
	homeFile := filepath.Join(home, "resync.conf")
	if err := os.WriteFile(homeFile, []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "resync",
		Files: []string{"resync.conf"},
	}

	engine := New(home, syncDir, WithCommand("backup"))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Files) != 1 {
		t.Errorf("expected 1 file processed, got %d", len(result.Files))
	}

	// Verify syncDir now has the new content
	data, err := os.ReadFile(syncFile)
	if err != nil {
		t.Fatalf("failed to read sync file: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("syncDir should have new content, got: %q", string(data))
	}

	// Verify home points to syncDir
	target, err := os.Readlink(homeFile)
	if err != nil {
		t.Errorf("home should be a symlink: %v", err)
	}
	absTarget := target
	if !filepath.IsAbs(target) {
		absTarget = filepath.Join(filepath.Dir(homeFile), target)
	}
	if absTarget != syncFile {
		t.Errorf("home should point to syncDir: got %q (abs: %q), want %q", target, absTarget, syncFile)
	}
}

// TestSync_NeitherExist_ReturnsAlreadySynced tests when neither home nor syncDir has the file
func TestSync_NeitherExist_ReturnsAlreadySynced(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	app := &appdb.AppConfig{
		Name:  "empty",
		Files: []string{"nonexistent.conf"},
	}

	engine := New(home, syncDir)
	result, err := engine.Sync(app)

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	// Result is returned but with 0 files (since nothing was processed)
	if result == nil {
		t.Fatal("expected result object, got nil")
	}
	if len(result.Files) != 0 {
		t.Errorf("expected 0 files processed, got %d", len(result.Files))
	}
}

// TestSync_RelativeSymlink tests that relative symlinks work correctly
func TestSync_RelativeSymlink(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create a subdirectory structure
	subDir := filepath.Join(home, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	homeFile := filepath.Join(subDir, "rel.conf")
	if err := os.WriteFile(homeFile, []byte("relative content"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "reltest",
		Files: []string{"subdir/rel.conf"},
	}

	engine := New(home, syncDir, WithCommand("backup"))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Files) != 1 {
		t.Errorf("expected 1 file processed, got %d", len(result.Files))
	}

	// Verify the symlink works
	content, err := os.ReadFile(homeFile)
	if err != nil {
		t.Fatalf("failed to read home file: %v", err)
	}
	if string(content) != "relative content" {
		t.Errorf("unexpected content: %q", string(content))
	}

	// Verify it's an absolute symlink (fix for broken relative paths)
	target, err := os.Readlink(homeFile)
	if err != nil {
		t.Fatalf("home file should be a symlink: %v", err)
	}
	// After fix: symlinks use absolute paths to avoid broken relative paths
	if !filepath.IsAbs(target) {
		t.Errorf("expected absolute symlink, got relative: %q", target)
	}
}

// TestSync_AlreadySynced_SecondSyncNoOp tests that syncing an already synced file is a no-op
func TestSync_AlreadySynced_SecondSyncNoOp(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create file in syncDir
	syncFile := filepath.Join(syncDir, "already.conf")
	if err := os.WriteFile(syncFile, []byte("synced content"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:  "synced",
		Files: []string{"already.conf"},
	}

	engine := New(home, syncDir, WithCommand("restore"))

	// First sync: restore from cloud
	result1, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("First sync failed: %v", err)
	}
	if len(result1.Files) != 1 {
		t.Errorf("expected 1 file in first sync, got %d", len(result1.Files))
	}

	// Get initial mod time
	homeFile := filepath.Join(home, "already.conf")
	info1, err := os.Stat(homeFile)
	if err != nil {
		t.Fatal(err)
	}

	// Second sync: should be no-op
	result2, err := engine.Sync(app)
	if err != nil {
		t.Errorf("Second sync should not error, got: %v", err)
	}
	// Result is returned but with 0 files (since nothing was processed)
	if len(result2.Files) != 0 {
		t.Errorf("Second sync should process 0 files, got %d", len(result2.Files))
	}

	// Verify mod time didn't change
	info2, err := os.Stat(homeFile)
	if err != nil {
		t.Fatal(err)
	}
	if info1.ModTime() != info2.ModTime() {
		t.Error("Second sync should not modify files")
	}
}
