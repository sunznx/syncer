package appdb

import (
	"fmt"
	"io"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	// Name is the application identifier (e.g., "git", "vim")
	Name string `yaml:"name"`

	// Application is the application name (optional, for display purposes)
	Application string `yaml:"application,omitempty"`

	// Files is the list of file patterns to sync (relative to home directory)
	Files []string `yaml:"files"`

	// Mode specifies how files are synced: "link" (symlinks) or "copy" (copy files)
	Mode string `yaml:"mode"`

	// Ignore specifies file patterns to exclude from syncing
	Ignore []string `yaml:"ignore,omitempty"`

	// External defines external resources (git repos, archives, files)
	External []*ExternalConfig `yaml:"external,omitempty"`
}

// ExternalConfig defines external resources like git repos, archives, or files.
type ExternalConfig struct {
	// Type is the external type: "git", "archive", "file"
	Type string `yaml:"type"`

	// URL is the external resource URL
	URL string `yaml:"url,omitempty"`

	// Target is the target location relative to home directory
	Target string `yaml:"target,omitempty"`

	// Path is the source path within the external resource (for single subpath)
	Path string `yaml:"path,omitempty"`

	// Executable makes the target file executable
	Executable bool `yaml:"executable,omitempty"`

	// Subpaths defines paths to sync from external resource
	Subpaths []*ExternalSubpath `yaml:"subpaths,omitempty"`
}

// ExternalSubpath defines a subpath to sync from external resource.
type ExternalSubpath struct {
	// Path is the source path in external resource
	Path string `yaml:"path,omitempty"`

	// Target is the target path relative to home directory
	Target string `yaml:"target,omitempty"`

	// Executable makes the target file executable
	Executable bool `yaml:"executable,omitempty"`
}

// String returns a string representation of the app config.
func (a *AppConfig) String() string {
	return a.Name
}

// IsLinkMode returns true if the app should use symlinks.
func (a *AppConfig) IsLinkMode() bool {
	return a.Mode == "link" || a.Mode == "" // Default to link mode
}

// ShouldIgnore returns true if the given file should be ignored.
func (a *AppConfig) ShouldIgnore(relPath string) bool {
	for _, pattern := range a.Ignore {
		if matchPattern(relPath, pattern) {
			return true
		}
		// Also check the basename (last component) against the pattern
		// This allows patterns like "*.txt" to match "dir/file.txt"
		basename := filepath.Base(relPath)
		if matchPattern(basename, pattern) {
			return true
		}
	}
	return false
}

// matchPattern checks if a path matches a wildcard pattern.
func matchPattern(path, pattern string) bool {
	// Use filepath.Match for proper wildcard matching
	// filepath.Match supports *, ?, and [] character classes
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		// If pattern is invalid, fall back to exact match
		return pattern == path
	}
	return matched
}

// ParseYAML parses an app config from YAML format.
func ParseYAML(r io.Reader) (*AppConfig, error) {
	var cfg AppConfig
	if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return &cfg, nil
}

// ParseConfig parses an app config from legacy CFG format.
func ParseConfig(r io.Reader) (*AppConfig, error) {
	// Legacy CFG format support - simple key=value parsing
	// For now, just return an error
	return nil, fmt.Errorf("legacy CFG format no longer supported")
}
