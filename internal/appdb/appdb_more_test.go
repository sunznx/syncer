package appdb

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppConfigString(t *testing.T) {
	app := &AppConfig{Name: "vim"}
	if got := app.String(); got != "vim" {
		t.Errorf("String() = %q, want vim", got)
	}
}

func TestAppConfigIsLinkMode(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"link", true},
		{"", true},
		{"copy", false},
		{"other", false},
	}
	for _, tt := range tests {
		app := &AppConfig{Mode: tt.mode}
		if got := app.IsLinkMode(); got != tt.want {
			t.Errorf("IsLinkMode(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestShouldIgnore(t *testing.T) {
	app := &AppConfig{
		Ignore: []string{"*.log", "tmp", "*.tmp"},
	}
	tests := []struct {
		path string
		want bool
	}{
		{"app.log", true},
		{"dir/app.log", true},
		{"tmp", true},
		{"dir/tmp", true},
		{"file.txt", false},
		{"data.tmp", true},
	}
	for _, tt := range tests {
		if got := app.ShouldIgnore(tt.path); got != tt.want {
			t.Errorf("ShouldIgnore(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestMatchPatternInvalid(t *testing.T) {
	// An invalid pattern should fall back to exact match
	if got := matchPattern("foo", "["); got {
		t.Error("invalid pattern should fall back to exact match")
	}
	if got := matchPattern("[", "["); !got {
		t.Error("exact match should work for invalid pattern")
	}
}

func TestParseConfig(t *testing.T) {
	_, err := ParseConfig(strings.NewReader(""))
	if err == nil {
		t.Error("expected error for legacy CFG format")
	}
	if !strings.Contains(err.Error(), "no longer supported") {
		t.Errorf("expected 'no longer supported' error, got %v", err)
	}
}

func TestDBOptions(t *testing.T) {
	var fs embed.FS
	db := NewDB(
		WithBuiltinDir("/builtin"),
		WithBuiltinFS(fs),
		WithSyncersDir("/syncers"),
	)
	if db.builtinDir != "/builtin" {
		t.Errorf("builtinDir = %q, want /builtin", db.builtinDir)
	}
	if !db.hasBuiltinFS {
		t.Error("expected hasBuiltinFS to be true")
	}
	if db.syncersDir != "/syncers" {
		t.Errorf("syncersDir = %q, want /syncers", db.syncersDir)
	}
}

func TestLoadFromBuiltinFS(t *testing.T) {
	// Create a temporary embedded FS is not possible in tests,
	// but we can test the error path with an empty FS.
	var fs embed.FS
	db := NewDB(WithBuiltinFS(fs))
	_, err := db.loadFromBuiltinFS("nonexistent")
	if err == nil {
		t.Error("expected error for missing config in empty FS")
	}
}

func TestAppNameFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"git.yaml", "git"},
		{"vim.cfg", "vim"},
		{"readme.md", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := appNameFromFilename(tt.filename); got != tt.want {
			t.Errorf("appNameFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestIsOverridden(t *testing.T) {
	dir := t.TempDir()
	db := NewDB(WithSyncersDir(dir))

	if db.IsOverridden("git") {
		t.Error("expected false when no override file exists")
	}

	// Create a .yaml override
	if err := writeTestFile(dir, "git.yaml", "name: git\n"); err != nil {
		t.Fatal(err)
	}
	if !db.IsOverridden("git") {
		t.Error("expected true when .yaml override exists")
	}

	// Create a .cfg override
	if err := writeTestFile(dir, "vim.cfg", "name=vim\n"); err != nil {
		t.Fatal(err)
	}
	if !db.IsOverridden("vim") {
		t.Error("expected true when .cfg override exists")
	}
}

func TestDBListAndLoad(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestFile(dir, "app1.yaml", "name: App1\nfiles:\n  - .a"); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(dir, "app2.yaml", "name: App2\nfiles:\n  - .b"); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(dir, "readme.txt", "hello"); err != nil {
		t.Fatal(err)
	}

	db := NewDB(WithSyncersDir(dir))
	names := db.List()
	if len(names) != 2 {
		t.Fatalf("expected 2 apps, got %d: %v", len(names), names)
	}

	app, err := db.Load("app1")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if app.Name != "App1" {
		t.Errorf("app.Name = %q, want App1", app.Name)
	}

	// Second load should use cache
	app2, err := db.Load("app1")
	if err != nil {
		t.Fatalf("cached Load failed: %v", err)
	}
	if app2 != app {
		t.Error("expected cached instance to be returned")
	}
}

func TestDBLoadMissing(t *testing.T) {
	db := NewDB()
	_, err := db.Load("missing")
	if err == nil {
		t.Error("expected error for missing app")
	}
}

func TestDBLoadFromBuiltinDir(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestFile(dir, "builtin.yaml", "name: BuiltIn\nfiles:\n  - .c"); err != nil {
		t.Fatal(err)
	}

	db := NewDB(WithBuiltinDir(dir))
	app, err := db.Load("builtin")
	if err != nil {
		t.Fatalf("Load from builtin dir failed: %v", err)
	}
	if app.Name != "BuiltIn" {
		t.Errorf("app.Name = %q, want BuiltIn", app.Name)
	}
}

func TestDBLoadFallbackToCFG(t *testing.T) {
	dir := t.TempDir()
	// No yaml, only cfg
	if err := writeTestFile(dir, "legacy.cfg", "name=legacy"); err != nil {
		t.Fatal(err)
	}

	db := NewDB(WithBuiltinDir(dir))
	_, err := db.Load("legacy")
	// ParseConfig always returns error, so this should fail
	if err == nil {
		t.Error("expected error when falling back to unsupported CFG")
	}
}

func TestAddDirEntriesEmpty(t *testing.T) {
	var names []string
	seen := map[string]bool{}
	addDirEntries("", &names, seen)
	if len(names) != 0 {
		t.Error("expected no names for empty dir")
	}
	addDirEntries("/nonexistent/path", &names, seen)
	if len(names) != 0 {
		t.Error("expected no names for missing dir")
	}
}

func writeTestFile(dir, name, content string) error {
	path := filepath.Join(dir, name)
	return os.WriteFile(path, []byte(content), 0644)
}
