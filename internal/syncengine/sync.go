package syncengine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunznx/syncer/internal/appdb"
	"github.com/sunznx/syncer/internal/color"
	"github.com/sunznx/syncer/internal/fileops"
)

// ErrAlreadySynced indicates the file is already in sync.
var ErrAlreadySynced = errors.New("already synced")

const (
	// ModeLink uses symlinks.
	ModeLink = "link"
	// ModeCopy copies files.
	ModeCopy = "copy"
)

// Engine handles syncing files between home and sync directory.
type Engine struct {
	homeDir   string
	syncDir   string
	dryRun    bool
	command   string // "backup" or "restore"
	callbacks []func(string)
}

// Option configures an Engine.
type Option func(*Engine)

// WithDryRun enables dry-run mode.
func WithDryRun() Option {
	return func(e *Engine) { e.dryRun = true }
}

// WithProgressCallback sets a callback for progress updates.
func WithProgressCallback(cb func(string)) Option {
	return func(e *Engine) { e.callbacks = append(e.callbacks, cb) }
}

// WithCommand sets the sync command (backup or restore).
func WithCommand(cmd string) Option {
	return func(e *Engine) { e.command = cmd }
}

// New creates a new sync engine.
func New(homeDir, syncDir string, opts ...Option) *Engine {
	e := &Engine{
		homeDir: homeDir,
		syncDir: syncDir,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Result represents the result of syncing an app.
type Result struct {
	App   *appdb.AppConfig
	Files []string
}

// String returns a string representation of the result.
func (r *Result) String() string {
	var mode string
	if r.App.IsLinkMode() {
		mode = "link"
	} else {
		mode = "copy"
	}
	return fmt.Sprintf("[%s] %s (%d files)", mode, r.App.Name, len(r.Files))
}

// Sync syncs an app's files between home and sync directory.
func (e *Engine) Sync(app *appdb.AppConfig) (*Result, error) {
	result := &Result{
		App:   app,
		Files: []string{},
	}

	for _, filePattern := range app.Files {
		// Expand file pattern (may contain wildcards)
		homePath := filepath.Join(e.homeDir, filePattern)

		// Check if it's a wildcard pattern
		if strings.Contains(filePattern, "*") || strings.Contains(filePattern, "?") {
			matches, err := filepath.Glob(homePath)
			if err != nil {
				e.log(fmt.Sprintf("Error globbing %s: %v", filePattern, err))
				continue
			}
			for _, match := range matches {
				relPath, err := filepath.Rel(e.homeDir, match)
				if err != nil {
					continue
				}
				if app.ShouldIgnore(relPath) {
					continue
				}
				if err := e.syncFile(relPath, app); err != nil {
					if err != ErrAlreadySynced {
						e.log(fmt.Sprintf("Error syncing %s: %v", relPath, err))
					}
				} else {
					result.Files = append(result.Files, relPath)
				}
			}
		} else {
			// Single file
			if app.ShouldIgnore(filePattern) {
				continue
			}
			if err := e.syncFile(filePattern, app); err != nil {
				if err != ErrAlreadySynced {
					e.log(fmt.Sprintf("Error syncing %s: %v", filePattern, err))
				}
			} else {
				result.Files = append(result.Files, filePattern)
			}
		}
	}

	return result, nil
}

// syncFile syncs a single file.
func (e *Engine) syncFile(relPath string, app *appdb.AppConfig) error {
	homePath := filepath.Join(e.homeDir, relPath)
	syncPath := filepath.Join(e.syncDir, relPath)

	homeExists := false
	syncExists := false

	// Check home file
	_, err := os.Lstat(homePath)
	if err == nil {
		homeExists = true
	}

	// Check sync file
	_, err = os.Lstat(syncPath)
	if err == nil {
		syncExists = true
	}

	mode := e.detectMode(relPath, app)

	// Handle based on command type
	if e.command == "backup" {
		return e.syncFileBackup(homePath, syncPath, relPath, mode, homeExists, syncExists)
	}
	return e.syncFileRestore(homePath, syncPath, relPath, mode, homeExists, syncExists)
}

// syncFileBackup handles backup command: home -> cloud, then home -> symlink -> cloud
func (e *Engine) syncFileBackup(homePath, syncPath, relPath string, mode string, homeExists, syncExists bool) error {
	// If home file doesn't exist, nothing to backup
	if !homeExists {
		return ErrAlreadySynced
	}

	// Check if home is a symlink
	isHomeSymlink := fileops.IsSymlink(homePath)

	// If home is a symlink, check if it's broken
	if isHomeSymlink {
		if _, err := os.Stat(homePath); err != nil {
			// Broken symlink - treat as if home doesn't exist
			return ErrAlreadySynced
		}
	}

	// If home is already a correct symlink to syncPath or any known sync storage
	if mode == ModeLink && isHomeSymlink {
		target, err := os.Readlink(homePath)
		if err == nil {
			absTarget := target
			if !filepath.IsAbs(target) {
				absTarget = filepath.Join(filepath.Dir(homePath), target)
			}
			absSyncPath, _ := filepath.Abs(syncPath)
			if absTarget == absSyncPath {
				// Check if syncPath itself is an external symlink needing migration
				if migrated, err := e.migrateExternalSyncPath(syncPath); err != nil {
					return err
				} else if migrated {
					return nil
				}
				if pathAccessible(syncPath) {
					return ErrAlreadySynced
				}
			}

		}
	}

	// Resolve symlink target for reading source content
	sourcePath := homePath
	if isHomeSymlink {
		if resolved, err := filepath.EvalSymlinks(homePath); err == nil {
			sourcePath = resolved
		}
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	syncParentDir := filepath.Dir(syncPath)
	if !e.dryRun {
		if err := os.MkdirAll(syncParentDir, 0755); err != nil {
			return fmt.Errorf("create sync dir: %w", err)
		}
	}

	if sourceInfo.IsDir() {
		if e.dryRun {
			e.log(color.DryRun("Would copy directory: ") + color.Path(sourcePath) + color.Arrow() + color.Path(syncPath))
		} else {
			if err := fileops.SafeCopyAll(sourcePath, syncPath); err != nil {
				return fmt.Errorf("copy directory: %w", err)
			}
		}
	} else {
		if e.dryRun {
			e.log(color.Action("Copying: ") + color.Path(sourcePath) + color.Arrow() + color.Path(syncPath))
		} else {
			if err := fileops.CopyFile(sourcePath, syncPath); err != nil {
				return fmt.Errorf("copy file: %w", err)
			}
		}
	}

	// Now replace home with symlink/copy pointing to syncPath
	homeParentDir := filepath.Dir(homePath)
	if !e.dryRun {
		if err := os.MkdirAll(homeParentDir, 0755); err != nil {
			return fmt.Errorf("create home dir: %w", err)
		}
	}

	if mode == ModeLink {
		if e.dryRun {
			e.log(color.DryRun("Would create symlink: ") + color.Path(homePath) + color.Arrow() + color.Path(syncPath))
			return nil
		}
		return e.createSymlink(syncPath, homePath)
	}

	if e.dryRun {
		e.log(color.DryRun("Would copy: ") + color.Path(syncPath) + color.Arrow() + color.Path(homePath))
		return nil
	}
	return e.createCopy(syncPath, homePath)
}

// syncFileRestore handles restore command: cloud -> home (create symlink)
func (e *Engine) syncFileRestore(homePath, syncPath, relPath string, mode string, homeExists, syncExists bool) error {
	// If nothing in sync, nothing to restore
	if !syncExists {
		return ErrAlreadySynced
	}

	// Check if home already correctly points to sync
	if homeExists && mode == ModeLink {
		if fileops.IsSymlink(homePath) {
			target, err := os.Readlink(homePath)
			if err == nil {
				absTarget := target
				if !filepath.IsAbs(target) {
					absTarget = filepath.Join(filepath.Dir(homePath), target)
				}
				absSyncPath, _ := filepath.Abs(syncPath)
				if absTarget == absSyncPath {
					return ErrAlreadySynced
				}

			}
		}
	}

	// Ensure home directory exists
	homeParentDir := filepath.Dir(homePath)
	if !e.dryRun {
		if err := os.MkdirAll(homeParentDir, 0755); err != nil {
			return fmt.Errorf("create home dir: %w", err)
		}
	}

	if mode == ModeLink {
		if e.dryRun {
			e.log(color.DryRun("Would create symlink: ") + color.Path(homePath) + color.Arrow() + color.Path(syncPath))
			return nil
		}
		return e.createSymlink(syncPath, homePath)
	}

	if e.dryRun {
		e.log(color.DryRun("Would copy: ") + color.Path(syncPath) + color.Arrow() + color.Path(homePath))
		return nil
	}
	return e.createCopy(syncPath, homePath)
}

// migrateExternalSyncPath checks if syncPath is a symlink pointing outside syncDir,
// and if so, copies the external content to replace the symlink with actual data.
// Returns true if migration was performed.
func (e *Engine) migrateExternalSyncPath(syncPath string) (bool, error) {
	if !fileops.IsSymlink(syncPath) {
		return false, nil
	}

	syncTarget, err := os.Readlink(syncPath)
	if err != nil {
		return false, nil
	}

	absSyncTarget := syncTarget
	if !filepath.IsAbs(syncTarget) {
		absSyncTarget = filepath.Join(filepath.Dir(syncPath), syncTarget)
	}

	// Only migrate if pointing outside syncDir
	// Use filepath.Rel to prevent path traversal (e.g. syncer/../etc/passwd)
	absSyncDir, _ := filepath.Abs(e.syncDir)
	rel, err := filepath.Rel(absSyncDir, absSyncTarget)
	if err != nil {
		return false, nil
	}
	if !strings.HasPrefix(rel, "..") && rel != "." {
		// Target is inside syncDir, no migration needed
		return false, nil
	}

	// External symlink - copy content to replace the symlink
	sourceInfo, err := os.Stat(absSyncTarget)
	if err != nil {
		return false, nil
	}

	os.Remove(syncPath)

	if sourceInfo.IsDir() {
		if err := fileops.SafeCopyAll(absSyncTarget, syncPath); err != nil {
			return false, fmt.Errorf("migrate external dir to sync: %w", err)
		}
	} else {
		if err := fileops.CopyFile(absSyncTarget, syncPath); err != nil {
			return false, fmt.Errorf("migrate external file to sync: %w", err)
		}
	}

	return true, nil
}

// pathAccessible checks if a path actually exists and is accessible.
// More reliable than os.Stat for cloud filesystems which may return cached metadata.
func pathAccessible(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// createSymlink creates a symlink from src to dst.
func (e *Engine) createSymlink(src, dst string) error {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		absSrc = src
	}

	if e.dryRun {
		if existingTarget, err := os.Readlink(dst); err == nil {
			absExisting, _ := filepath.Abs(filepath.Join(filepath.Dir(dst), existingTarget))
			if absExisting == absSrc {
				return ErrAlreadySynced
			}
		}
		e.log(color.DryRun("Would create symlink: ") + color.Path(dst) + color.Arrow() + color.Path(src))
		return nil
	}

	dir := filepath.Dir(dst)
	base := filepath.Base(dst)
	matches, _ := filepath.Glob(filepath.Join(dir, base+" ?*"))
	for _, match := range matches {
		os.Remove(match)
	}

	os.RemoveAll(dst)

	if err := os.Symlink(absSrc, dst); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}

	matches, _ = filepath.Glob(filepath.Join(dir, base+" ?*"))
	for _, match := range matches {
		os.Remove(match)
	}

	return nil
}

// createCopy copies src to dst.
func (e *Engine) createCopy(src, dst string) error {
	os.RemoveAll(dst)

	if e.dryRun {
		e.log(color.DryRun("Would copy: ") + color.Path(src) + color.Arrow() + color.Path(dst))
		return nil
	}

	if err := fileops.CopyFile(src, dst); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	return nil
}

// detectMode determines the sync mode for a given file.
// macOS system files (plist in Library/Preferences) must use copy mode
// due to macOS Sonoma+ symlink restrictions. Everything else defaults to link.
func (e *Engine) detectMode(relPath string, app *appdb.AppConfig) string {
	if app.Mode == "copy" {
		return ModeCopy
	}
	if app.Mode == "link" {
		return ModeLink
	}

	if strings.Contains(relPath, "Library/Preferences/") && strings.HasSuffix(relPath, ".plist") {
		return ModeCopy
	}

	return ModeLink
}

// log logs a message via callbacks.
func (e *Engine) log(msg string) {
	for _, cb := range e.callbacks {
		cb(msg)
	}
}
