package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source.txt")
	dst := filepath.Join(t.TempDir(), "dest.txt")
	content := []byte("test content")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	result, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(result) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(result), string(content))
	}
}

func TestCopyFileOverwritesSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	middle := filepath.Join(dir, "middle.txt")
	link := filepath.Join(dir, "link.txt")
	content := []byte("new content")

	if err := os.WriteFile(src, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(middle, []byte("middle"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(middle, link); err != nil {
		t.Fatal(err)
	}

	// Copy over the symlink
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, link); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// link should now be a regular file with new content
	if IsSymlink(link) {
		t.Error("expected link to be replaced with regular file")
	}
	result, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(result), string(content))
	}
}

func TestSafeCopy(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "file.txt")
	dst := filepath.Join(dstDir, "subdir", "file.txt")
	content := []byte("safe copy content")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := SafeCopy(src, dst); err != nil {
		t.Fatalf("SafeCopy failed: %v", err)
	}

	result, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(result) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(result), string(content))
	}
}

func TestSafeCopyMissingSource(t *testing.T) {
	src := filepath.Join(t.TempDir(), "missing.txt")
	dst := filepath.Join(t.TempDir(), "dest.txt")

	if err := SafeCopy(src, dst); err == nil {
		t.Error("expected error for missing source file")
	}
}

func TestIsSymlink(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	link := filepath.Join(dir, "link.txt")

	// Create a regular file
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Not a symlink yet
	if IsSymlink(file) {
		t.Error("regular file should not be a symlink")
	}

	// Create symlink
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}

	// Now it's a symlink
	if !IsSymlink(link) {
		t.Error("symlink should be detected as symlink")
	}

	// Original file still not a symlink
	if IsSymlink(file) {
		t.Error("original file should not be a symlink")
	}

	// Non-existent path
	if IsSymlink(filepath.Join(dir, "nonexistent")) {
		t.Error("non-existent path should not be a symlink")
	}
}

func TestCreateSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	linkPath := filepath.Join(dir, "link.txt")

	if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CreateSymlink(target, linkPath); err != nil {
		t.Fatalf("CreateSymlink failed: %v", err)
	}

	if !IsSymlink(linkPath) {
		t.Error("expected created path to be a symlink")
	}

	readTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if readTarget != target {
		t.Errorf("symlink target = %q, want %q", readTarget, target)
	}
}

func TestCreateSymlinkReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	oldFile := filepath.Join(dir, "old.txt")
	linkPath := filepath.Join(dir, "link.txt")

	if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldFile, linkPath); err != nil {
		t.Fatal(err)
	}

	if err := CreateSymlink(target, linkPath); err != nil {
		t.Fatalf("CreateSymlink failed: %v", err)
	}

	readTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if readTarget != target {
		t.Errorf("symlink target = %q, want %q", readTarget, target)
	}
}

func TestCreateSymlinkMissingTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "missing.txt")
	linkPath := filepath.Join(dir, "link.txt")

	if err := CreateSymlink(target, linkPath); err == nil {
		t.Error("expected error when target does not exist")
	}
}

func TestSafeCopyAll_File(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "file.txt")
	dst := filepath.Join(dstDir, "file.txt")
	content := []byte("file content")

	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := SafeCopyAll(src, dst); err != nil {
		t.Fatalf("SafeCopyAll failed: %v", err)
	}

	result, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(result), string(content))
	}
}

func TestSafeCopyAll_Directory(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dst")

	// Create nested structure
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	file1 := filepath.Join(srcDir, "a.txt")
	file2 := filepath.Join(subDir, "b.txt")
	if err := os.WriteFile(file1, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SafeCopyAll(srcDir, dstDir); err != nil {
		t.Fatalf("SafeCopyAll failed: %v", err)
	}

	// Verify structure
	result1, err := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(result1) != "a" {
		t.Errorf("a.txt content mismatch")
	}

	result2, err := os.ReadFile(filepath.Join(dstDir, "subdir", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(result2) != "b" {
		t.Errorf("b.txt content mismatch")
	}
}

func TestSafeCopyAll_MissingSource(t *testing.T) {
	src := filepath.Join(t.TempDir(), "missing")
	dst := filepath.Join(t.TempDir(), "dst")

	if err := SafeCopyAll(src, dst); err == nil {
		t.Error("expected error for missing source")
	}
}
