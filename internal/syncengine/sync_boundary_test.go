package syncengine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunznx/syncer/internal/appdb"
)

func TestSync_SkipsIgnoredFiles(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	// Create files
	homeConfigDir := filepath.Join(home, ".config", "test")
	if err := os.MkdirAll(homeConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	homeFile := filepath.Join(homeConfigDir, "config.txt")
	if err := os.WriteFile(homeFile, []byte("config"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &appdb.AppConfig{
		Name:   "test",
		Files:  []string{".config/test/config.txt"}, // Sync the file directly
		Mode:   "link",
		Ignore: []string{"*.txt"},
	}

	engine := New(home, syncDir, WithProgressCallback(func(msg string) {
		t.Log("SYNC:", msg)
	}))
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("Result: %d files processed", len(result.Files))

	// Should be skipped due to ignore pattern
	if len(result.Files) != 0 {
		t.Errorf("expected 0 files (ignored), got %d", len(result.Files))
	}
}

func TestSync_EmptyAppConfig(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()

	app := &appdb.AppConfig{
		Name:  "empty",
		Files: []string{},
		Mode:  "link",
	}

	engine := New(home, syncDir)
	result, err := engine.Sync(app)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(result.Files))
	}
}
