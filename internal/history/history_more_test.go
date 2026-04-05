package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListEmpty(t *testing.T) {
	tempDir := t.TempDir()
	mgr := New(tempDir)

	entries, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListInvalidEntry(t *testing.T) {
	tempDir := t.TempDir()
	mgr := New(tempDir)

	historyFile := filepath.Join(tempDir, ".syncer-history.jsonl")
	data := `{"command":"backup","success":true}
invalid json
{"command":"restore","success":false}
`
	if err := os.WriteFile(historyFile, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 valid entries, got %d", len(entries))
	}
	// Most recent first
	if entries[0].Command != "restore" {
		t.Errorf("expected first entry to be restore, got %s", entries[0].Command)
	}
}

func TestSplitLinesCRLF(t *testing.T) {
	data := []byte("line1\r\nline2\r\nline3")
	lines := splitLines(data)
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if string(lines[0]) != "line1" {
		t.Errorf("expected line1, got %q", string(lines[0]))
	}
	if string(lines[2]) != "line3" {
		t.Errorf("expected line3, got %q", string(lines[2]))
	}
}

func TestSplitLinesTrailingNewline(t *testing.T) {
	data := []byte("line1\nline2\n")
	lines := splitLines(data)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestSplitLinesEmpty(t *testing.T) {
	lines := splitLines([]byte{})
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestFormatManyApps(t *testing.T) {
	entry := &Entry{
		Timestamp: time.Date(2026, 4, 5, 10, 30, 0, 0, time.Local).Unix(),
		Command:   "backup",
		Apps:      []string{"a", "b", "c", "d", "e"},
		FileCount: 5,
		Success:   true,
		DryRun:    false,
	}
	formatted := entry.Format()
	if !strings.Contains(formatted, "a, b, c, ... (2 more)") {
		t.Errorf("expected truncated app list, got: %s", formatted)
	}
}

func TestFormatNoApps(t *testing.T) {
	entry := &Entry{
		Timestamp: time.Date(2026, 4, 5, 10, 30, 0, 0, time.Local).Unix(),
		Command:   "backup",
		Apps:      []string{},
		FileCount: 0,
		Success:   false,
		DryRun:    true,
	}
	formatted := entry.Format()
	if !strings.Contains(formatted, "all") {
		t.Errorf("expected 'all' when no apps, got: %s", formatted)
	}
	if !strings.Contains(formatted, "✗ (dry-run)") {
		t.Errorf("expected failure dry-run status, got: %s", formatted)
	}
}
