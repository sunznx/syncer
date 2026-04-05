package appdb

import (
	"strings"
	"testing"
)

func TestParseYAML_Basic(t *testing.T) {
	yamlContent := `
name: TestApp
files:
  - .config/test
  - .bashrc
mode: link
`
	app, err := ParseYAML(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("ParseYAML failed: %v", err)
	}

	if app.Name != "TestApp" {
		t.Errorf("expected name 'TestApp', got %q", app.Name)
	}

	if len(app.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(app.Files))
	}

	if app.Files[0] != ".config/test" {
		t.Errorf("expected first file '.config/test', got %q", app.Files[0])
	}

	if app.Files[1] != ".bashrc" {
		t.Errorf("expected second file '.bashrc', got %q", app.Files[1])
	}

	if !app.IsLinkMode() {
		t.Error("expected link mode by default")
	}
}

func TestParseYAML_CopyMode(t *testing.T) {
	yamlContent := `
name: CopyApp
files:
  - .copyrc
mode: copy
`
	app, err := ParseYAML(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("ParseYAML failed: %v", err)
	}

	if app.Mode != "copy" {
		t.Errorf("expected mode 'copy', got %q", app.Mode)
	}

	if app.IsLinkMode() {
		t.Error("should not be in link mode")
	}
}

func TestParseYAML_WithIgnore(t *testing.T) {
	yamlContent := `
name: IgnoreApp
files:
  - .testrc
ignore:
  - "*.tmp"
  - "*.log"
`
	app, err := ParseYAML(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("ParseYAML failed: %v", err)
	}

	if len(app.Ignore) != 2 {
		t.Errorf("expected 2 ignore patterns, got %d", len(app.Ignore))
	}

	if app.Ignore[0] != "*.tmp" {
		t.Errorf("expected first ignore '*.tmp', got %q", app.Ignore[0])
	}
}

func TestParseYAML_EmptyFiles(t *testing.T) {
	yamlContent := `
name: EmptyFilesApp
files: []
mode: link
`
	app, err := ParseYAML(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("ParseYAML failed: %v", err)
	}

	if len(app.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(app.Files))
	}
}
