// Package history manages command execution history for syncer.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a single history entry.
type Entry struct {
	Timestamp int64    `json:"timestamp"`       // Unix timestamp
	Command   string   `json:"command"`         // "backup" or "restore"
	Apps      []string `json:"apps"`            // Application names processed
	FileCount int      `json:"file_count"`      // Total files processed
	Success   bool     `json:"success"`         // Whether the operation succeeded
	Error     string   `json:"error,omitempty"` // Error message if failed
	DryRun    bool     `json:"dry_run"`         // Whether this was a dry-run
}

// Manager manages history entries.
type Manager struct {
	historyFile string
}

// New creates a new history manager.
func New(syncDir string) *Manager {
	if syncDir == "" {
		homeDir, _ := os.UserHomeDir()
		syncDir = filepath.Join(homeDir, ".config", "syncer")
	}
	return &Manager{
		historyFile: filepath.Join(syncDir, ".syncer-history.jsonl"),
	}
}

// Record adds a new entry to the history.
func (m *Manager) Record(entry *Entry) error {
	entry.Timestamp = time.Now().Unix()

	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(m.historyFile), 0755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	// Open file in append mode
	f, err := os.OpenFile(m.historyFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}
	defer f.Close()

	// Write entry as JSON line
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}

	return nil
}

// List retrieves all history entries, most recent first.
func (m *Manager) List() ([]Entry, error) {
	// Read all lines
	data, err := os.ReadFile(m.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("read history file: %w", err)
	}

	lines := splitLines(data)
	entries := make([]Entry, 0, len(lines))

	// Parse each line as JSON
	for i := len(lines) - 1; i >= 0; i-- { // Reverse to get most recent first
		var entry Entry
		if err := json.Unmarshal(lines[i], &entry); err != nil {
			continue // Skip invalid entries
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// splitLines splits byte slice by newlines, handling both LF and CRLF.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		} else if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			lines = append(lines, data[start:i])
			start = i + 2
			i++ // Skip the \n
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// Format returns a human-readable string for the entry.
func (e *Entry) Format() string {
	t := time.Unix(e.Timestamp, 0).Format("2006-01-02 15:04:05")

	status := "✓"
	if !e.Success {
		status = "✗"
	}
	if e.DryRun {
		status += " (dry-run)"
	}

	appsStr := "all"
	if len(e.Apps) > 0 {
		appsStr = ""
		for i, app := range e.Apps {
			if i > 0 {
				appsStr += ", "
			}
			if i >= 3 {
				appsStr += fmt.Sprintf("... (%d more)", len(e.Apps)-i)
				break
			}
			appsStr += app
		}
	}

	return fmt.Sprintf("%s  [%s]  %s  %s, %d files", t, status, e.Command, appsStr, e.FileCount)
}
