// Package external handles external resources (git repos, archives, files).
package external

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunznx/syncer/internal/appdb"
	"github.com/sunznx/syncer/internal/color"
	"github.com/sunznx/syncer/internal/fileops"
)

var ErrAlreadySynced = errors.New("already synced")

// Manager handles external resources.
type Manager struct {
	syncDir string
	dryRun  bool
}

// New creates a new external manager.
func New(syncDir string, dryRun bool) *Manager {
	return &Manager{
		syncDir: syncDir,
		dryRun:  dryRun,
	}
}

// urlToPath converts a URL to a directory path.
// e.g., https://github.com/user/repo.git -> github.com/user/repo
func urlToPath(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git://")
	url = strings.TrimPrefix(url, "ssh://")

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Remove git@ prefix for SSH URLs
	url = strings.TrimPrefix(url, "git@")

	// Convert : to / for SSH URLs (git@github.com:user/repo -> github.com/user/repo)
	url = strings.Replace(url, ":", "/", 1)

	return url
}

// Pull fetches and updates external resources for the given app.
// Returns the destination path where the external resource was cloned/extracted.
// If ext.Target is set, also creates a symlink in syncDir pointing to the external resource.
func (m *Manager) Pull(appName string, ext *appdb.ExternalConfig) (string, error) {
	if ext == nil {
		return "", nil
	}

	destName := urlToPath(ext.URL)
	if destName == "" {
		destName = appName
	}

	destPath := filepath.Join(m.syncDir, ".syncer_external", destName)

	var err error
	switch ext.Type {
	case "git":
		err = m.pullGitRepo(ext.URL, destPath)
	case "archive":
		err = m.pullArchive(ext.URL, destPath, ext)
	case "file":
		err = m.pullFile(ext.URL, destPath, ext.Executable)
	default:
		return "", fmt.Errorf("unsupported external type: %s", ext.Type)
	}
	if err != nil {
		return destPath, err
	}

	// If target is specified, create a symlink in syncDir pointing to the external resource
	if ext.Target != "" {
		targetPath := filepath.Join(m.syncDir, ext.Target)
		if err := m.createSymlink(destPath, targetPath); err != nil && err != ErrAlreadySynced {
			return destPath, fmt.Errorf("link external target %s: %w", ext.Target, err)
		}
	}

	return destPath, nil
}

// pullGitRepo clones or updates a git repository.
func (m *Manager) pullGitRepo(url, dest string) error {
	// Check if repo already exists
	if _, err := os.Stat(dest); err == nil {
		// Repo exists, pull updates
		if m.dryRun {
			fmt.Printf("%s%s\n", color.DryRun("Would pull: "), color.Path(dest))
			return nil
		}
		return m.gitPull(dest)
	}

	// Clone new repo
	if m.dryRun {
		fmt.Printf("%s%s%s%s\n", color.DryRun("Would clone: "), color.Path(url), color.Arrow(), color.Path(dest))
		return nil
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	return m.gitClone(url, dest)
}

// gitClone runs git clone.
func (m *Manager) gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", url, err)
	}

	return nil
}

// gitPull runs git pull.
// Automatically stashes any uncommitted changes before pulling.
func (m *Manager) gitPull(dest string) error {
	// Check for uncommitted changes and stash them if necessary
	if m.hasUncommittedChanges(dest) {
		fmt.Printf("Stashing uncommitted changes in %s\n", dest)
		if err := m.gitStash(dest); err != nil {
			return fmt.Errorf("git stash in %s: %w", dest, err)
		}
	}

	cmd := exec.Command("git", "-C", dest, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull in %s: %w", dest, err)
	}

	return nil
}

// hasUncommittedChanges checks if a git repo has uncommitted changes.
func (m *Manager) hasUncommittedChanges(dest string) bool {
	cmd := exec.Command("git", "-C", dest, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// gitStash stashes uncommitted changes.
func (m *Manager) gitStash(dest string) error {
	args := []string{"-C", dest, "stash", "push", "-m", "syncer auto-stash before pull"}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pullFile downloads a single file from a URL.
func (m *Manager) pullFile(url, dest string, executable bool) error {
	if m.dryRun {
		fmt.Printf("%s%s%s%s\n", color.DryRun("Would download: "), color.Path(url), color.Arrow(), color.Path(dest))
		return nil
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	// Download file with timeout
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer out.Close()

	// Copy content
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}

	// Set executable bit if requested
	if executable {
		if err := os.Chmod(dest, 0755); err != nil {
			return fmt.Errorf("set executable: %w", err)
		}
	}

	return nil
}

// pullArchive downloads and extracts an archive, optionally syncing specific subpaths.
func (m *Manager) pullArchive(url, dest string, ext *appdb.ExternalConfig) error {
	if m.dryRun {
		fmt.Printf("%s%s%s%s\n", color.DryRun("Would download archive: "), color.Path(url), color.Arrow(), color.Path(dest))
		return nil
	}

	// Download to temporary file first
	tempFile := dest + ".tmp"
	if err := m.pullFile(url, tempFile, false); err != nil {
		return err
	}
	defer os.Remove(tempFile)

	// Detect archive format from URL extension
	format := ""
	if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		format = "tar.gz"
	} else if strings.HasSuffix(url, ".tar.bz2") || strings.HasSuffix(url, ".tbz2") {
		format = "tar.bz2"
	} else if strings.HasSuffix(url, ".tar.zst") {
		format = "tar.zst"
	} else if strings.HasSuffix(url, ".zip") {
		format = "zip"
	} else {
		return fmt.Errorf("unsupported archive format for URL: %s", url)
	}

	// Extract archive to temporary directory
	tempDir := dest + ".extract"
	defer os.RemoveAll(tempDir)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	var cmd *exec.Cmd
	switch format {
	case "zip":
		cmd = exec.Command("unzip", "-q", tempFile, "-d", tempDir)
	case "tar.bz2":
		cmd = exec.Command("tar", "-xjf", tempFile, "-C", tempDir)
	case "tar.zst":
		cmd = exec.Command("tar", "--zstd", "-xf", tempFile, "-C", tempDir)
	default:
		// tar.gz and tgz
		cmd = exec.Command("tar", "-xzf", tempFile, "-C", tempDir)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	// Process subpaths if specified
	if len(ext.Subpaths) > 0 {
		// Apply subpaths filtering
		for _, subpath := range ext.Subpaths {
			if subpath.Path == "" {
				continue
			}

			srcPath := filepath.Join(tempDir, subpath.Path)

			// Determine target path
			targetPath := subpath.Target
			if targetPath == "" {
				targetPath = subpath.Path
			}

			// Clean up target path (remove leading ./)
			targetPath = strings.TrimPrefix(targetPath, "./")
			if targetPath == "" {
				targetPath = subpath.Path
			}

			// Full destination path
			destPath := filepath.Join(dest, targetPath)

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("create parent dir: %w", err)
			}

			// Move or copy based on configuration
			fileInfo, err := os.Stat(srcPath)
			if err != nil {
				return fmt.Errorf("stat source: %w", err)
			}

			if fileInfo.IsDir() {
				// Directory - use recursive copy
				if err := fileops.SafeCopyAll(srcPath, destPath); err != nil {
					return fmt.Errorf("copy directory: %w", err)
				}
			} else {
				// File
				if subpath.Executable {
					// Copy and set executable
					if err := fileops.CopyFile(srcPath, destPath); err != nil {
						return fmt.Errorf("copy file: %w", err)
					}
					if err := os.Chmod(destPath, 0755); err != nil {
						return fmt.Errorf("set executable: %w", err)
					}
				} else {
					// Create symlink
					if err := m.createSymlink(srcPath, destPath); err != nil && err != ErrAlreadySynced {
						return fmt.Errorf("create symlink: %w", err)
					}
				}
			}
		}
	} else {
		// No subpaths, just rename tempDir to dest
		if err := os.Rename(tempDir, dest); err != nil {
			return fmt.Errorf("move extracted: %w", err)
		}
	}

	return nil
}

// createSymlink creates a symlink from src to dst.
func (m *Manager) createSymlink(src, dst string) error {
	// Use absolute path to avoid broken relative symlinks
	absSrc, err := filepath.Abs(src)
	if err != nil {
		absSrc = src
	}

	if m.dryRun {
		fmt.Printf("%s%s%s%s\n", color.DryRun("Would create symlink: "), color.Path(dst), color.Arrow(), color.Path(src))
		return nil
	}

	// First, clean up any macOS APFS file clones (numbered suffix) BEFORE creating new symlink
	dir := filepath.Dir(dst)
	base := filepath.Base(dst)
	matches, _ := filepath.Glob(filepath.Join(dir, base+" ?*"))
	for _, match := range matches {
		_ = os.Remove(match)
	}

	// Check if dst exists and what type it is
	if fileInfo, err := os.Lstat(dst); err == nil {
		// File exists
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			// It's a symlink, check if it points to correct target
			if existingTarget, err := os.Readlink(dst); err == nil {
				absExisting := existingTarget
				if !filepath.IsAbs(existingTarget) {
					absExisting = filepath.Join(filepath.Dir(dst), existingTarget)
				}
				absExisting, _ = filepath.Abs(absExisting)
				if absExisting == absSrc {
					// Already points to correct target
					return ErrAlreadySynced
				}
			}
			// Wrong target or other issue, remove it
			_ = os.Remove(dst)
		} else {
			// It's a directory or regular file, remove it
			// Use RemoveAll for directories
			_ = os.RemoveAll(dst)
		}
	}

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	// Create symlink
	if err := os.Symlink(absSrc, dst); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}

	// Delay briefly to let macOS APFS settle
	// Then check for and clean up any file clones that might have been created
	matches, _ = filepath.Glob(filepath.Join(dir, base+" ?*"))
	for _, match := range matches {
		_ = os.Remove(match)
	}

	return nil
}

// IsGitRepo checks if the given path is a git repository.
func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return true
	}
	return false
}
