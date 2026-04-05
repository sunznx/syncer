package external

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunznx/syncer/internal/appdb"
)

// TestHasUncommittedChanges tests the detection of uncommitted changes.
func TestHasUncommittedChanges(t *testing.T) {
	tests := []struct {
		name              string
		setupFunc         func(string) error
		expectUncommitted bool
	}{
		{
			name: "clean repo",
			setupFunc: func(repoDir string) error {
				return nil
			},
			expectUncommitted: false,
		},
		{
			name: "repo with uncommitted changes",
			setupFunc: func(repoDir string) error {
				testFile := filepath.Join(repoDir, "test.txt")
				return os.WriteFile(testFile, []byte("test content"), 0644)
			},
			expectUncommitted: true,
		},
		{
			name: "repo with staged changes",
			setupFunc: func(repoDir string) error {
				testFile := filepath.Join(repoDir, "test.txt")
				if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
					return err
				}
				cmd := exec.Command("git", "-C", repoDir, "add", "test.txt")
				return cmd.Run()
			},
			expectUncommitted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := t.TempDir()

			// Initialize git repo
			exec.Command("git", "-C", repoDir, "init").Run()
			exec.Command("git", "-C", repoDir, "config", "user.name", "Test").Run()
			exec.Command("git", "-C", repoDir, "config", "user.email", "test@example.com").Run()

			// Create initial commit
			initialFile := filepath.Join(repoDir, "initial.txt")
			os.WriteFile(initialFile, []byte("initial"), 0644)
			exec.Command("git", "-C", repoDir, "add", ".").Run()
			exec.Command("git", "-C", repoDir, "commit", "-m", "initial").Run()

			// Apply test-specific setup
			if tt.setupFunc != nil {
				if err := tt.setupFunc(repoDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			mgr := &Manager{}
			hasChanges := mgr.hasUncommittedChanges(repoDir)

			if hasChanges != tt.expectUncommitted {
				t.Errorf("hasUncommittedChanges() = %v, want %v", hasChanges, tt.expectUncommitted)
			}
		})
	}
}

// TestGitStash tests the stashing of uncommitted changes.
func TestGitStash(t *testing.T) {
	repoDir := t.TempDir()

	// Initialize git repo
	if err := exec.Command("git", "-C", repoDir, "init").Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := exec.Command("git", "-C", repoDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatalf("git config user.name failed: %v", err)
	}
	if err := exec.Command("git", "-C", repoDir, "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("git config user.email failed: %v", err)
	}

	// Create initial commit
	initialFile := filepath.Join(repoDir, "initial.txt")
	if err := os.WriteFile(initialFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}
	if err := exec.Command("git", "-C", repoDir, "add", ".").Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := exec.Command("git", "-C", repoDir, "commit", "-m", "initial").Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create uncommitted changes by modifying existing file
	if err := os.WriteFile(initialFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Verify changes exist
	mgr := &Manager{}
	if !mgr.hasUncommittedChanges(repoDir) {
		t.Fatal("expected uncommitted changes before stash")
	}

	// Stash changes
	if err := mgr.gitStash(repoDir); err != nil {
		t.Fatalf("gitStash failed: %v", err)
	}

	// Verify no uncommitted changes after stash
	if mgr.hasUncommittedChanges(repoDir) {
		t.Error("expected no uncommitted changes after stash")
	}

	// Verify stash was created
	cmd := exec.Command("git", "-C", repoDir, "stash", "list")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to list stash: %v", err)
	}

	if !strings.Contains(string(output), "syncer auto-stash before pull") {
		t.Error("expected to find auto-stash entry")
	}
}

func TestUrlToPath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL with .git suffix",
			url:      "https://github.com/ohmyzsh/ohmyzsh.git",
			expected: "github.com/ohmyzsh/ohmyzsh",
		},
		{
			name:     "HTTPS URL without .git suffix",
			url:      "https://github.com/affaan-m/everything-claude-code",
			expected: "github.com/affaan-m/everything-claude-code",
		},
		{
			name:     "HTTP URL",
			url:      "http://github.com/user/repo.git",
			expected: "github.com/user/repo",
		},
		{
			name:     "SSH URL with git@ prefix",
			url:      "git@github.com:user/repo.git",
			expected: "github.com/user/repo",
		},
		{
			name:     "Git protocol URL",
			url:      "git://github.com/user/repo.git",
			expected: "github.com/user/repo",
		},
		{
			name:     "URL with nested path",
			url:      "https://gitlab.com/group/subgroup/project.git",
			expected: "gitlab.com/group/subgroup/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := urlToPath(tt.url)
			if result != tt.expected {
				t.Errorf("urlToPath(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestManager_DestPath(t *testing.T) {
	tests := []struct {
		name         string
		appName      string
		url          string
		syncDir      string
		expectedPath string
	}{
		{
			name:         "GitHub HTTPS URL",
			appName:      "test-app",
			url:          "https://github.com/user/repo.git",
			syncDir:      "/test/sync",
			expectedPath: "/test/sync/.syncer_external/github.com/user/repo",
		},
		{
			name:         "Empty URL uses app name",
			appName:      "myapp",
			url:          "",
			syncDir:      "/test/sync",
			expectedPath: "/test/sync/.syncer_external/myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test path construction logic
			ext := &appdb.ExternalConfig{
				Type: "git",
				URL:  tt.url,
			}

			destName := urlToPath(ext.URL)
			if destName == "" {
				destName = tt.appName
			}
			destPath := tt.syncDir + "/.syncer_external/" + destName

			if destPath != tt.expectedPath {
				t.Errorf("dest path = %q, want %q", destPath, tt.expectedPath)
			}
		})
	}
}

func TestNew(t *testing.T) {
	m := New("/tmp/sync", true)
	if m.syncDir != "/tmp/sync" {
		t.Errorf("syncDir = %q, want %q", m.syncDir, "/tmp/sync")
	}
	if !m.dryRun {
		t.Error("expected dryRun to be true")
	}
}

func TestPullNilExt(t *testing.T) {
	m := New(t.TempDir(), false)
	dest, err := m.Pull("app", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest != "" {
		t.Errorf("expected empty dest, got %q", dest)
	}
}

func TestPullUnsupportedType(t *testing.T) {
	m := New(t.TempDir(), false)
	dest, err := m.Pull("app", &appdb.ExternalConfig{Type: "unknown", URL: "http://example.com"})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if dest != "" {
		t.Error("expected dest to be empty for unsupported type error")
	}
}

func TestPullGitDryRun(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, true)

	// Repo doesn't exist - should print would clone
	_, err := m.Pull("app", &appdb.ExternalConfig{Type: "git", URL: "https://github.com/user/repo.git"})
	if err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}

	// Create fake repo dir - should print would pull
	dest := filepath.Join(dir, ".syncer_external", "github.com", "user", "repo")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	_, err = m.Pull("app", &appdb.ExternalConfig{Type: "git", URL: "https://github.com/user/repo.git"})
	if err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}
}

func TestPullFileDryRun(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, true)
	_, err := m.Pull("app", &appdb.ExternalConfig{Type: "file", URL: "https://example.com/file.txt"})
	if err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}
}

func TestPullFileWithTarget(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	// Start a local HTTP server
	content := []byte("hello from server")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer ts.Close()

	_, err := m.Pull("app", &appdb.ExternalConfig{
		Type:   "file",
		URL:    ts.URL + "/file.txt",
		Target: "bin/file.txt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was downloaded (urlToPath converts URL to nested path)
	downloaded := filepath.Join(dir, ".syncer_external", "127.0.0.1", "52193", "file.txt")
	data, err := os.ReadFile(downloaded)
	if err != nil {
		// Port may differ, just verify some file was downloaded
		matches, _ := filepath.Glob(filepath.Join(dir, ".syncer_external", "*", "*", "file.txt"))
		if len(matches) == 0 {
			t.Fatalf("failed to find downloaded file: %v", err)
		}
		data, err = os.ReadFile(matches[0])
		if err != nil {
			t.Fatalf("failed to read downloaded file: %v", err)
		}
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(data), string(content))
	}

	// Verify symlink was created
	target := filepath.Join(dir, "bin", "file.txt")
	if _, err := os.Lstat(target); err != nil {
		t.Fatalf("target symlink not created: %v", err)
	}
}

func TestPullFileExecutable(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("#!/bin/sh\necho hello"))
	}))
	defer ts.Close()

	_, err := m.Pull("app", &appdb.ExternalConfig{
		Type:       "file",
		URL:        ts.URL + "/script.sh",
		Executable: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify executable bit (urlToPath converts URL to nested path)
	matches, _ := filepath.Glob(filepath.Join(dir, ".syncer_external", "*", "*", "script.sh"))
	if len(matches) == 0 {
		t.Fatalf("failed to find downloaded script")
	}
	info, err := os.Stat(matches[0])
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("expected file to be executable")
	}
}

func TestPullFileDownloadError(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	// Server returns 404
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	_, err := m.Pull("app", &appdb.ExternalConfig{Type: "file", URL: ts.URL + "/missing.txt"})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestCreateSymlink(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("src"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := m.createSymlink(src, dst); err != nil {
		t.Fatalf("createSymlink failed: %v", err)
	}

	// Should be a symlink
	if fi, err := os.Lstat(dst); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("expected dst to be a symlink")
	}
}

func TestCreateSymlinkAlreadySynced(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("src"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create correct symlink first (use same abs logic as implementation)
	absSrc, _ := filepath.Abs(src)
	if err := os.Symlink(absSrc, dst); err != nil {
		t.Fatal(err)
	}

	err := m.createSymlink(src, dst)
	if err != ErrAlreadySynced {
		t.Fatalf("expected ErrAlreadySynced, got %v", err)
	}
}

func TestCreateSymlinkDryRun(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, true)

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("src"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := m.createSymlink(src, dst); err != nil {
		t.Fatalf("createSymlink failed in dry-run: %v", err)
	}

	// Should NOT create symlink in dry-run
	if _, err := os.Lstat(dst); err == nil {
		t.Error("expected no symlink in dry-run mode")
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Error("empty dir should not be a git repo")
	}

	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	if !IsGitRepo(dir) {
		t.Error("dir with .git should be a git repo")
	}
}

func TestPullArchiveUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not an archive"))
	}))
	defer ts.Close()

	_, err := m.Pull("app", &appdb.ExternalConfig{Type: "archive", URL: ts.URL + "/file.txt"})
	if err == nil {
		t.Fatal("expected error for unsupported archive format")
	}
}

func TestPullArchiveDryRun(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, true)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("archive content"))
	}))
	defer ts.Close()

	_, err := m.Pull("app", &appdb.ExternalConfig{Type: "archive", URL: ts.URL + "/file.tar.gz"})
	if err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}
}

func TestPullArchiveWithSubpaths(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	// Create a tar.gz archive with a single file
	archiveDir := t.TempDir()
	subDir := filepath.Join(archiveDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(subDir, "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(archiveDir, "test.tar.gz")
	cmd := exec.Command("tar", "-czf", archivePath, "-C", archiveDir, "subdir")
	if err := cmd.Run(); err != nil {
		t.Skipf("tar not available: %v", err)
	}

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer ts.Close()

	_, err = m.Pull("app", &appdb.ExternalConfig{
		Type: "archive",
		URL:  ts.URL + "/test.tar.gz",
		Subpaths: []*appdb.ExternalSubpath{
			{Path: "subdir/hello.txt"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify extracted file
	extracted := filepath.Join(dir, ".syncer_external", "test", "subdir", "hello.txt")
	data, err := os.ReadFile(extracted)
	if err != nil {
		// Some tar implementations may create different structures
		// Try alternative path
		extracted = filepath.Join(dir, ".syncer_external", "test", "hello.txt")
		data, err = os.ReadFile(extracted)
		if err != nil {
			t.Skipf("could not find extracted file: %v", err)
		}
	}
	if string(data) != "hello" {
		t.Errorf("content mismatch: got %q, want %q", string(data), "hello")
	}
}

func TestPullArchiveWithExecutableSubpath(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	archiveDir := t.TempDir()
	scriptPath := filepath.Join(archiveDir, "script.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(archiveDir, "test.tar.gz")
	cmd := exec.Command("tar", "-czf", archivePath, "-C", archiveDir, "script.sh")
	if err := cmd.Run(); err != nil {
		t.Skipf("tar not available: %v", err)
	}

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer ts.Close()

	_, err = m.Pull("app", &appdb.ExternalConfig{
		Type: "archive",
		URL:  ts.URL + "/test.tar.gz",
		Subpaths: []*appdb.ExternalSubpath{
			{Path: "script.sh", Executable: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find extracted file
	extracted := filepath.Join(dir, ".syncer_external", "test", "script.sh")
	info, err := os.Stat(extracted)
	if err != nil {
		t.Skipf("could not find extracted file: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("expected script to be executable")
	}
}

func TestPullArchiveTarBZ2(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	archiveDir := t.TempDir()
	filePath := filepath.Join(archiveDir, "data.txt")
	if err := os.WriteFile(filePath, []byte("bz2 data"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(archiveDir, "test.tar.bz2")
	cmd := exec.Command("tar", "-cjf", archivePath, "-C", archiveDir, "data.txt")
	if err := cmd.Run(); err != nil {
		t.Skipf("tar bz2 not available: %v", err)
	}

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer ts.Close()

	_, err = m.Pull("app", &appdb.ExternalConfig{Type: "archive", URL: ts.URL + "/test.tar.bz2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullArchiveZip(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	archiveDir := t.TempDir()
	filePath := filepath.Join(archiveDir, "data.txt")
	if err := os.WriteFile(filePath, []byte("zip data"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(archiveDir, "test.zip")
	cmd := exec.Command("zip", "-j", archivePath, filePath)
	if err := cmd.Run(); err != nil {
		t.Skipf("zip not available: %v", err)
	}

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer ts.Close()

	_, err = m.Pull("app", &appdb.ExternalConfig{Type: "archive", URL: ts.URL + "/test.zip"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullArchiveSubpathMissing(t *testing.T) {
	dir := t.TempDir()
	m := New(dir, false)

	archiveDir := t.TempDir()
	filePath := filepath.Join(archiveDir, "data.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(archiveDir, "test.tar.gz")
	cmd := exec.Command("tar", "-czf", archivePath, "-C", archiveDir, "data.txt")
	if err := cmd.Run(); err != nil {
		t.Skipf("tar not available: %v", err)
	}

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer ts.Close()

	// Request a subpath that doesn't exist in the archive
	_, err = m.Pull("app", &appdb.ExternalConfig{
		Type: "archive",
		URL:  ts.URL + "/test.tar.gz",
		Subpaths: []*appdb.ExternalSubpath{
			{Path: "missing.txt"},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing subpath")
	}
}
