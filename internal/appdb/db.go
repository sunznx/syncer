package appdb

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DB loads and caches application configurations from multiple sources.
// Priority order (highest to lowest):
// 1. syncersDir ($SYNCER_DIR/.syncers/) - highest priority, overrides everything
// 2. builtinDir / builtinFS - built-in configs from configs/ directory
type DB struct {
	syncersDir   string   // $SYNCER_DIR/.syncers/ for custom configs (highest priority)
	builtinDir   string   // built-in configs directory (configs/)
	builtinFS    embed.FS // built-in configs as embedded FS
	hasBuiltinFS bool     // true if builtinFS is set
	cache        map[string]*AppConfig
}

// DBOption configures a DB instance.
type DBOption func(*DB)

// WithBuiltinDir sets the built-in configs directory.
func WithBuiltinDir(dir string) DBOption {
	return func(db *DB) { db.builtinDir = dir }
}

// WithBuiltinFS sets the built-in configs as an embedded FS.
func WithBuiltinFS(fs embed.FS) DBOption {
	return func(db *DB) { db.builtinFS = fs; db.hasBuiltinFS = true }
}

// WithSyncersDir sets the $SYNCER_DIR/.syncers/ directory for custom configs.
// This takes highest priority when loading app configs.
func WithSyncersDir(dir string) DBOption {
	return func(db *DB) { db.syncersDir = dir }
}

// NewDB creates a DB with the given options.
func NewDB(opts ...DBOption) *DB {
	db := &DB{cache: make(map[string]*AppConfig)}
	for _, opt := range opts {
		opt(db)
	}
	return db
}

// Load returns the AppConfig for the named application.
// Priority: syncersDir > builtin.
func (db *DB) Load(name string) (*AppConfig, error) {
	if cached, ok := db.cache[name]; ok {
		return cached, nil
	}

	// Try syncersDir first (highest priority), then builtin
	app, err := db.loadFromDir(db.syncersDir, name)
	if err != nil {
		app, err = db.loadFromBuiltin(name)
	}
	if err != nil {
		return nil, fmt.Errorf("app %q not found: %w", name, err)
	}

	db.cache[name] = app
	return app, nil
}

// List returns all available application names (merged from both sources).
func (db *DB) List() []string {
	seen := map[string]bool{}
	var names []string

	// List from syncersDir first (highest priority)
	addDirEntries(db.syncersDir, &names, seen)

	// List from builtin embedded FS
	if db.hasBuiltinFS {
		entries, err := db.builtinFS.ReadDir("configs")
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if name := appNameFromFilename(e.Name()); name != "" && !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
		}
	}

	// List from builtin directory (fallback for non-embedded usage)
	addDirEntries(db.builtinDir, &names, seen)

	return names
}

func (db *DB) loadFromDir(dir, name string) (*AppConfig, error) {
	if dir == "" {
		return nil, fmt.Errorf("no directory configured")
	}
	// Try .yaml first, then fallback to .cfg
	yamlPath := filepath.Join(dir, name+".yaml")
	f, err := os.Open(yamlPath)
	if err == nil {
		defer f.Close()
		return ParseYAML(f)
	}
	cfgPath := filepath.Join(dir, name+".cfg")
	f, err = os.Open(cfgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseConfig(f)
}

func (db *DB) loadFromBuiltin(name string) (*AppConfig, error) {
	// Try embedded FS first
	if db.hasBuiltinFS {
		app, err := db.loadFromBuiltinFS(name)
		if err == nil {
			return app, nil
		}
	}
	// Fallback to directory
	if db.builtinDir != "" {
		return db.loadFromDir(db.builtinDir, name)
	}
	return nil, fmt.Errorf("no builtin source configured")
}

func (db *DB) loadFromBuiltinFS(name string) (*AppConfig, error) {
	// Try .yaml first, then fallback to .cfg
	yamlPath := "configs/" + name + ".yaml"
	f, err := db.builtinFS.Open(yamlPath)
	if err == nil {
		defer f.Close()
		return ParseYAML(f)
	}
	cfgPath := "configs/" + name + ".cfg"
	f, err = db.builtinFS.Open(cfgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseConfig(f)
}

// appNameFromFilename extracts the application name from a config filename.
// Returns the name without extension for .yaml and .cfg files, empty string otherwise.
func appNameFromFilename(filename string) string {
	if strings.HasSuffix(filename, ".yaml") {
		return strings.TrimSuffix(filename, ".yaml")
	}
	if strings.HasSuffix(filename, ".cfg") {
		return strings.TrimSuffix(filename, ".cfg")
	}
	return ""
}

// IsOverridden reports whether the named app has a custom config in syncersDir.
func (db *DB) IsOverridden(name string) bool {
	if db.syncersDir == "" {
		return false
	}
	yamlPath := filepath.Join(db.syncersDir, name+".yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return true
	}
	cfgPath := filepath.Join(db.syncersDir, name+".cfg")
	if _, err := os.Stat(cfgPath); err == nil {
		return true
	}
	return false
}

// addDirEntries scans a directory for .yaml/.cfg files and adds unseen app names.
func addDirEntries(dir string, names *[]string, seen map[string]bool) {
	if dir == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if name := appNameFromFilename(e.Name()); name != "" && !seen[name] {
			seen[name] = true
			*names = append(*names, name)
		}
	}
}
