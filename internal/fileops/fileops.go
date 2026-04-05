// Package fileops provides safe file operations for config sync.
package fileops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SafeCopy copies a file from src to dst, creating parent directories as needed.
// It overwrites dst if it already exists.
func SafeCopy(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("source file %q: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create parent dirs for %q: %w", dst, err)
	}

	return CopyFile(src, dst)
}

// CopyFile copies the contents of src to dst.
// If dst is a symlink, it will be removed before copying (to avoid following the symlink).
func CopyFile(src, dst string) error {
	// Remove dst first if it exists (especially important if it's a symlink)
	// os.Create follows symlinks, so we need to remove the symlink first
	os.RemoveAll(dst)

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source %q: %w", src, err)
	}
	defer f.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst %q: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, f); err != nil {
		return fmt.Errorf("copy %q → %q: %w", src, dst, err)
	}

	return nil
}

// CreateSymlink creates a symlink at linkPath pointing to target.
// If linkPath already exists (file or symlink), it is replaced.
// Returns error if target does not exist.
func CreateSymlink(target, linkPath string) error {
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("symlink target %q: %w", target, err)
	}

	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return fmt.Errorf("create parent dirs for %q: %w", linkPath, err)
	}

	// Remove existing file/symlink at linkPath
	_ = os.Remove(linkPath)

	if err := os.Symlink(target, linkPath); err != nil {
		return fmt.Errorf("create symlink %q → %q: %w", linkPath, target, err)
	}

	return nil
}

// IsSymlink returns true if path exists and is a symlink.
func IsSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

// SafeCopyAll recursively copies src to dst. Efficient for large directories.
func SafeCopyAll(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	if !srcInfo.IsDir() {
		return SafeCopy(src, dst)
	}

	// Source is a directory - use efficient walk and copy strategy
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory in destination
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		return CopyFile(path, dstPath)
	})
}
