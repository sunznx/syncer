package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Storage represents a cloud storage location for syncing configs.
type Storage interface {
	// SyncDir returns the path to the sync directory.
	SyncDir() (string, error)
	// SyncersDir returns the path to the .syncers directory.
	SyncersDir() (string, error)
	// HomeDir returns the user's home directory.
	HomeDir() string
}

// customStorage is a storage with a custom path.
type customStorage struct {
	path string
	home string
}

// NewCustom creates a storage with a custom path.
func NewCustom(path string) (Storage, error) {
	if path == "" {
		return nil, fmt.Errorf("storage path cannot be empty")
	}
	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}
	home, _ := os.UserHomeDir()
	return &customStorage{path: absPath, home: home}, nil
}

// SyncDir returns the custom sync directory.
func (s *customStorage) SyncDir() (string, error) {
	return s.path, nil
}

// SyncersDir returns the .syncers directory.
func (s *customStorage) SyncersDir() (string, error) {
	return filepath.Join(s.path, ".syncers"), nil
}

// HomeDir returns the user's home directory.
func (s *customStorage) HomeDir() string {
	return s.home
}

// NewDefault creates a storage by auto-detecting cloud storage.
func NewDefault(homeDir string) (Storage, error) {
	home := homeDir
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
	}

	// Try each cloud storage location in order
	paths := []string{
		filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs"), // iCloud
		filepath.Join(home, "Dropbox"),
		filepath.Join(home, "Google Drive"),
		filepath.Join(home, "OneDrive"),
		filepath.Join(home, "Box"),
		filepath.Join(home, ".config", "syncer"),
	}

	for _, path := range paths {
		marker := filepath.Join(path, "syncer", "syncer.yaml")
		if _, err := os.Stat(marker); err == nil {
			return &customStorage{path: filepath.Join(path, "syncer"), home: home}, nil
		}
	}

	return nil, fmt.Errorf("no syncer/syncer.yaml found in cloud storage directories (tried: iCloud, Dropbox, Google Drive, OneDrive, Box, ~/.config/syncer)")
}
