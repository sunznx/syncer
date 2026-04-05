// Package history provides unit tests for history management.
package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManager_Record(t *testing.T) {
	tempDir := t.TempDir()
	mgr := New(tempDir)

	entry := &Entry{
		Command:   "backup",
		Apps:      []string{"git", "zsh"},
		FileCount: 3,
		Success:   true,
		DryRun:    true,
	}

	err := mgr.Record(entry)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Verify file was created
	historyFile := filepath.Join(tempDir, ".syncer-history.jsonl")
	data, err := os.ReadFile(historyFile)
	if err != nil {
		t.Fatalf("history file not created: %v", err)
	}

	if len(data) == 0 {
		t.Error("history file is empty")
	}
}

func TestManager_List(t *testing.T) {
	tempDir := t.TempDir()
	mgr := New(tempDir)

	// Record multiple entries
	entries := []*Entry{
		{Command: "backup", Apps: []string{"git"}, FileCount: 1, Success: true, DryRun: false},
		{Command: "restore", Apps: []string{"zsh"}, FileCount: 2, Success: true, DryRun: false},
	}

	for _, entry := range entries {
		if err := mgr.Record(entry); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	// List should return entries in reverse order (most recent first)
	listed, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != len(entries) {
		t.Errorf("expected %d entries, got %d", len(entries), len(listed))
	}

	// Verify order (most recent first)
	if listed[0].Command != "restore" {
		t.Errorf("expected first entry to be restore, got %s", listed[0].Command)
	}
	if listed[1].Command != "backup" {
		t.Errorf("expected second entry to be backup, got %s", listed[1].Command)
	}
}

func TestEntry_Format(t *testing.T) {
	entry := &Entry{
		Timestamp: time.Date(2026, 4, 5, 10, 30, 0, 0, time.Local).Unix(),
		Command:   "backup",
		Apps:      []string{"git", "zsh"},
		FileCount: 3,
		Success:   true,
		DryRun:    false,
	}

	formatted := entry.Format()
	// Check key parts
	if !strings.Contains(formatted, "backup") {
		t.Errorf("Format missing command: got %s", formatted)
	}
	if !strings.Contains(formatted, "git, zsh") {
		t.Errorf("Format missing app names: got %s", formatted)
	}
	if !strings.Contains(formatted, "3 files") {
		t.Errorf("Format missing file count: got %s", formatted)
	}
	if !strings.Contains(formatted, "✓") {
		t.Errorf("Format missing OK status: got %s", formatted)
	}
}

func TestEntry_Format_DryRun(t *testing.T) {
	entry := &Entry{
		Timestamp: time.Date(2026, 4, 5, 10, 30, 0, 0, time.Local).Unix(),
		Command:   "backup",
		Apps:      []string{"git"},
		FileCount: 1,
		Success:   true,
		DryRun:    true,
	}

	formatted := entry.Format()
	// Check key parts
	if !strings.Contains(formatted, "backup") {
		t.Errorf("Format missing command: got %s", formatted)
	}
	if !strings.Contains(formatted, "(dry-run)") {
		t.Errorf("Format missing dry-run indicator: got %s", formatted)
	}
}

func TestEntry_Format_Failure(t *testing.T) {
	entry := &Entry{
		Timestamp: time.Date(2026, 4, 5, 10, 30, 0, 0, time.Local).Unix(),
		Command:   "backup",
		Apps:      []string{"git"},
		FileCount: 1,
		Success:   false,
		Error:     "file not found",
		DryRun:    false,
	}

	formatted := entry.Format()
	if formatted == "" {
		t.Error("Format returned empty string")
	}

	// Should include error indicator (NOT OK)
	if strings.Contains(formatted, "✓") {
		t.Error("failure should not show OK status")
	}
}

func TestNew_DefaultLocation(t *testing.T) {
	mgr := New("")

	// Should default to ~/.config/syncer
	homeDir, _ := os.UserHomeDir()
	expectedPath := filepath.Join(homeDir, ".config", "syncer", ".syncer-history.jsonl")

	if mgr.historyFile != expectedPath {
		t.Errorf("history file path mismatch:\ngot:  %s\nwant: %s", mgr.historyFile, expectedPath)
	}
}
