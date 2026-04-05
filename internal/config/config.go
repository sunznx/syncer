// Package config loads user configuration for syncer.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the user's syncer configuration.
type Config struct {
	// Settings contains configuration settings
	Settings Settings `yaml:"settings"`
	// Applications contains app sync configuration
	Applications Applications `yaml:"applications"`
	// HomeDir is the user's home directory.
	HomeDir string `yaml:"-"`
	// ConfigFile is the path to the config file that was loaded (if any).
	ConfigFile string `yaml:"-"`
}

// Settings contains configuration settings
type Settings struct {
	// StoragePath overrides auto-detected storage location.
	StoragePath string `yaml:"storage_path"`
}

// Applications contains app sync configuration
type Applications struct {
	// Apps is the list of application names to sync. Empty means all.
	Apps []string `yaml:"apps"`
	// Ignore is the list of application names to ignore.
	Ignore []string `yaml:"ignore"`
}

// Option configures a Config instance.
type Option func(*Config)

// WithHomeDir sets a custom home directory.
func WithHomeDir(dir string) Option {
	return func(c *Config) { c.HomeDir = dir }
}

// Default returns a Config with sensible defaults.
func Default(opts ...Option) *Config {
	home, _ := os.UserHomeDir()
	c := &Config{
		HomeDir: home,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Load discovers and loads the user's syncer configuration.
// Search order:
//  1. SYNCER_CONFIG environment variable
//  2. Cloud storage folders (Dropbox, iCloud, OneDrive, Google Drive, Box)
//  3. ~/.config/syncer/syncer.yaml
//  4. Defaults (no config file found)
func Load(opts ...Option) (*Config, error) {
	cfg := Default(opts...)

	// Step 1: Check SYNCER_CONFIG env
	if envPath := os.Getenv("SYNCER_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return loadFromFile(cfg, envPath)
		}
	}

	// Step 2: Check cloud storage folders
	for _, path := range cloudStoragePaths(cfg.HomeDir) {
		if _, err := os.Stat(path); err == nil {
			return loadFromFile(cfg, path)
		}
	}

	// Step 3: Check ~/.config/syncer/syncer.yaml
	homeCfgPath := filepath.Join(cfg.HomeDir, ".config", "syncer", "syncer.yaml")
	if _, err := os.Stat(homeCfgPath); err == nil {
		return loadFromFile(cfg, homeCfgPath)
	}

	// Step 4: No config found — use defaults
	return cfg, nil
}

// cloudStoragePaths returns potential syncer.yaml locations in cloud storage folders.
func cloudStoragePaths(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "Dropbox", "syncer", "syncer.yaml"),
		filepath.Join(homeDir, "iCloud", "syncer", "syncer.yaml"),
		filepath.Join(homeDir, "OneDrive", "syncer", "syncer.yaml"),
		filepath.Join(homeDir, "Google Drive", "syncer", "syncer.yaml"),
		filepath.Join(homeDir, "Box", "syncer", "syncer.yaml"),
		filepath.Join(homeDir, "Library", "Mobile Documents", "syncer", "syncer.yaml"), // iCloud
	}
}

func loadFromFile(cfg *Config, path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.ConfigFile = path
	return cfg, nil
}
