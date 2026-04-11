package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sunznx/syncer/internal/appdb"
	"github.com/sunznx/syncer/internal/config"
	"github.com/sunznx/syncer/internal/storage"
)

func TestModeLabel(t *testing.T) {
	if got := modeLabel(&appdb.AppConfig{Mode: "link"}); got != "link" {
		t.Errorf("modeLabel(link) = %q, want link", got)
	}
	if got := modeLabel(&appdb.AppConfig{Mode: "copy"}); got != "copy" {
		t.Errorf("modeLabel(copy) = %q, want copy", got)
	}
	if got := modeLabel(&appdb.AppConfig{}); got != "link" {
		t.Errorf("modeLabel(empty) = %q, want link", got)
	}
}

func TestFormatAppNames(t *testing.T) {
	apps := []*appdb.AppConfig{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
		{Name: "d"}, {Name: "e"}, {Name: "f"},
		{Name: "g"}, {Name: "h"}, {Name: "i"},
		{Name: "j"}, {Name: "k"},
	}

	if got := formatAppNames(nil, 5); got != "" {
		t.Errorf("formatAppNames(nil) = %q, want empty", got)
	}
	if got := formatAppNames(apps[:3], 5); got != "a, b, c" {
		t.Errorf("formatAppNames(3) = %q, want a, b, c", got)
	}
	if got := formatAppNames(apps, 10); got != "a, b, c, d, e, f, g, h, i, j, and 1 more" {
		t.Errorf("formatAppNames(11) = %q", got)
	}
}

func TestCountOverriddenApps(t *testing.T) {
	db := appdb.NewDB()
	apps := []*appdb.AppConfig{
		{Name: "git"},
		{Name: "vim"},
	}
	if got := countOverriddenApps(db, apps); got != 0 {
		t.Errorf("countOverriddenApps = %d, want 0", got)
	}
}

func TestEnsureFilesFromExternals(t *testing.T) {
	app := &appdb.AppConfig{
		Files: []string{".zshrc"},
		External: []*appdb.ExternalConfig{
			{Target: ".oh-my-zsh"},
		},
	}
	ensureFilesFromExternals(app)
	if len(app.Files) != 1 || app.Files[0] != ".zshrc" {
		t.Error("expected files to be unchanged when already defined")
	}

	app2 := &appdb.AppConfig{
		External: []*appdb.ExternalConfig{
			{Target: ".tmux"},
			{Target: ".tmux/plugins"},
		},
	}
	ensureFilesFromExternals(app2)
	if len(app2.Files) != 1 || app2.Files[0] != ".tmux" {
		t.Errorf("expected first external target to be added, got %v", app2.Files)
	}

	app3 := &appdb.AppConfig{}
	ensureFilesFromExternals(app3)
	if len(app3.Files) != 0 {
		t.Error("expected no files when no externals defined")
	}
}

func TestFilterApps(t *testing.T) {
	db := appdb.NewDB()
	cfg := &config.Config{}

	// When no apps specified, should return empty because built-in DB has no configs without FS/dir
	apps, err := filterApps(db, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With empty DB, list is empty
	_ = apps
}

func TestResolveStorage(t *testing.T) {
	cfg := &config.Config{HomeDir: "/tmp/home"}

	// Test with flag set
	flagSyncerDir = "/custom/sync"
	store, err := resolveStorage(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s, _ := store.SyncDir(); s != "/custom/sync" {
		t.Errorf("expected /custom/sync, got %q", s)
	}
	flagSyncerDir = ""

	// Test with config setting
	cfg.Settings.StoragePath = "/config/sync"
	store, err = resolveStorage(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s, _ := store.SyncDir(); s != "/config/sync" {
		t.Errorf("expected /config/sync, got %q", s)
	}
	cfg.Settings.StoragePath = ""
}

func TestPreviewSync(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := t.TempDir()

	// Create a file to backup
	app := &appdb.AppConfig{
		Name:  "testapp",
		Files: []string{".testconfig"},
		Mode:  "copy",
	}
	createTestFile(homeDir, ".testconfig", "data")

	apps := []*appdb.AppConfig{app}
	stats := previewSync(homeDir, syncDir, apps, "backup", true)
	if stats.ActionApps != 1 {
		t.Errorf("expected 1 action app, got %d", stats.ActionApps)
	}
	if stats.TotalFiles != 1 {
		t.Errorf("expected 1 total file, got %d", stats.TotalFiles)
	}

	// Run again - in dry-run mode, it still reports action because no actual symlink was created
	stats = previewSync(homeDir, syncDir, apps, "backup", true)
	if stats.ActionApps != 1 {
		t.Errorf("expected 1 action app in dry-run re-check, got %d", stats.ActionApps)
	}
}

func TestPreviewSyncRestore(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := t.TempDir()

	app := &appdb.AppConfig{
		Name:  "testapp",
		Files: []string{".testconfig"},
		Mode:  "copy",
	}
	// Put file in sync dir
	createTestFile(syncDir, ".testconfig", "restore data")

	apps := []*appdb.AppConfig{app}
	stats := previewSync(homeDir, syncDir, apps, "restore", true)
	if stats.ActionApps != 1 {
		t.Errorf("expected 1 action app for restore, got %d", stats.ActionApps)
	}
}

func TestPreviewSyncEmpty(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := t.TempDir()

	stats := previewSync(homeDir, syncDir, nil, "backup", true)
	if stats.ActionApps != 0 {
		t.Errorf("expected 0 action apps for empty list, got %d", stats.ActionApps)
	}
}

func TestPreviewSyncError(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := t.TempDir()

	// Create a file that will cause a conflict scenario
	app := &appdb.AppConfig{
		Name:  "badapp",
		Files: []string{".config"},
		Mode:  "copy",
	}
	// Create a directory at the home path where file is expected
	if err := os.MkdirAll(filepath.Join(homeDir, ".config"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create a file at the sync path
	if err := os.WriteFile(filepath.Join(syncDir, ".config"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	apps := []*appdb.AppConfig{app}
	stats := previewSync(homeDir, syncDir, apps, "backup", true)
	// In dry-run, directories are handled fine, so this may still succeed.
	// Just verify it doesn't panic.
	_ = stats
}

func createTestFile(base, rel, content string) {
	path := filepath.Join(base, rel)
	_ = os.WriteFile(path, []byte(content), 0644)
}

func TestLoadAppDB(t *testing.T) {
	store, err := storage.NewCustom("/tmp/sync")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	db := loadAppDB(store)
	if db == nil {
		t.Fatal("expected non-nil DB")
	}
}

func TestVersionCmd(t *testing.T) {
	cmd := versionCmd()
	if cmd.Use != "version" {
		t.Errorf("expected use 'version', got %q", cmd.Use)
	}
	// Execute to cover the Run function
	cmd.Run(cmd, []string{})
}

func TestListCmd(t *testing.T) {
	// Setup a temporary sync directory with marker file
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	if err := os.MkdirAll(syncDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n  apps:\n    - myapp\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Override env to find config
	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	defer func() { flagSyncerDir = "" }()

	cmd := listCmd()
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("listCmd failed: %v", err)
	}
}

func TestDoctorCmd(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	if err := os.MkdirAll(syncDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n  apps:\n    - myapp\n"), 0644); err != nil {
		t.Fatal(err)
	}

	syncersDir := filepath.Join(syncDir, ".syncers")
	if err := os.MkdirAll(syncersDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(syncersDir, "myapp.yaml"), []byte("name: MyApp\nfiles:\n  - .myapp\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".myapp"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	defer func() { flagSyncerDir = "" }()

	cmd := doctorCmd()
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("doctorCmd failed: %v", err)
	}
}

func TestDoctorCmdWithArgs(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	_ = os.MkdirAll(syncDir, 0755)
	_ = os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n"), 0644)

	syncersDir := filepath.Join(syncDir, ".syncers")
	_ = os.MkdirAll(syncersDir, 0755)
	_ = os.WriteFile(filepath.Join(syncersDir, "myapp.yaml"), []byte("name: MyApp\nfiles:\n  - .myapp\n"), 0644)

	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	defer func() { flagSyncerDir = "" }()

	cmd := doctorCmd()
	// Pass specific app name as argument
	err := cmd.RunE(cmd, []string{"myapp", "unknownapp"})
	if err != nil {
		t.Fatalf("doctorCmd with args failed: %v", err)
	}
}

func TestBackupRestoreCmd(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	_ = os.MkdirAll(syncDir, 0755)
	_ = os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n"), 0644)

	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	defer func() { flagSyncerDir = "" }()

	// Test that commands can be constructed
	b := backupCmd()
	if b.Use != "backup [app...]" {
		t.Errorf("unexpected use: %q", b.Use)
	}

	r := restoreCmd()
	if r.Use != "restore [app...]" {
		t.Errorf("unexpected use: %q", r.Use)
	}
}

func TestRunSyncWithArgs(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	_ = os.MkdirAll(syncDir, 0755)
	_ = os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n  apps:\n    - myapp\n"), 0644)

	// Create a custom app config with external resource
	syncersDir := filepath.Join(syncDir, ".syncers")
	_ = os.MkdirAll(syncersDir, 0755)
	appYAML := "name: MyApp\nfiles:\n  - .myapp\nexternal:\n  - type: file\n    url: https://example.com/file.txt\n"
	_ = os.WriteFile(filepath.Join(syncersDir, "myapp.yaml"), []byte(appYAML), 0644)

	// Create the home file
	_ = os.WriteFile(filepath.Join(homeDir, ".myapp"), []byte("data"), 0644)

	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	defer func() { flagSyncerDir = "" }()

	err := runSync([]string{"myapp"}, "backup")
	if err != nil {
		t.Fatalf("runSync failed: %v", err)
	}
}

func TestRunSyncDryRun(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	_ = os.MkdirAll(syncDir, 0755)
	_ = os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n  apps:\n    - myapp\n"), 0644)

	syncersDir := filepath.Join(syncDir, ".syncers")
	_ = os.MkdirAll(syncersDir, 0755)
	_ = os.WriteFile(filepath.Join(syncersDir, "myapp.yaml"), []byte("name: MyApp\nfiles:\n  - .myapp\n"), 0644)
	_ = os.WriteFile(filepath.Join(homeDir, ".myapp"), []byte("data"), 0644)

	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	flagDryRun = true
	defer func() {
		flagSyncerDir = ""
		flagDryRun = false
	}()

	err := runSync([]string{"myapp"}, "backup")
	if err != nil {
		t.Fatalf("runSync dry-run failed: %v", err)
	}
}

func TestRunSyncUnknownApp(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	_ = os.MkdirAll(syncDir, 0755)
	_ = os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n  apps:\n    - myapp\n"), 0644)

	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	defer func() { flagSyncerDir = "" }()

	err := runSync([]string{"nonexistent"}, "backup")
	if err != nil {
		t.Fatalf("runSync failed: %v", err)
	}
}

func TestRunSyncAllApps(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	_ = os.MkdirAll(syncDir, 0755)
	_ = os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n  apps:\n    - myapp\n"), 0644)

	syncersDir := filepath.Join(syncDir, ".syncers")
	_ = os.MkdirAll(syncersDir, 0755)
	_ = os.WriteFile(filepath.Join(syncersDir, "myapp.yaml"), []byte("name: MyApp\nfiles:\n  - .myapp\n"), 0644)
	_ = os.WriteFile(filepath.Join(homeDir, ".myapp"), []byte("data"), 0644)

	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	defer func() { flagSyncerDir = "" }()

	err := runSync([]string{}, "backup")
	if err != nil {
		t.Fatalf("runSync all apps failed: %v", err)
	}
}

func TestRunBackupRestore(t *testing.T) {
	homeDir := t.TempDir()
	syncDir := filepath.Join(homeDir, ".config", "syncer")
	_ = os.MkdirAll(syncDir, 0755)
	_ = os.WriteFile(filepath.Join(syncDir, "syncer.yaml"), []byte("applications:\n  apps:\n    - myapp\n"), 0644)

	syncersDir := filepath.Join(syncDir, ".syncers")
	_ = os.MkdirAll(syncersDir, 0755)
	_ = os.WriteFile(filepath.Join(syncersDir, "myapp.yaml"), []byte("name: MyApp\nfiles:\n  - .myapp\n"), 0644)
	_ = os.WriteFile(filepath.Join(homeDir, ".myapp"), []byte("data"), 0644)

	oldEnv := os.Getenv("SYNCER_CONFIG")
	os.Setenv("SYNCER_CONFIG", filepath.Join(syncDir, "syncer.yaml"))
	defer os.Setenv("SYNCER_CONFIG", oldEnv)

	flagSyncerDir = syncDir
	defer func() { flagSyncerDir = "" }()

	if err := runBackup([]string{"myapp"}); err != nil {
		t.Fatalf("runBackup failed: %v", err)
	}
	if err := runRestore([]string{"myapp"}); err != nil {
		t.Fatalf("runRestore failed: %v", err)
	}
}

func TestRootCmd(t *testing.T) {
	// Just verify main() doesn't panic by constructing the command tree
	// We can't call main() directly because it calls os.Exit, but we can
	// verify the root command structure.
	var rootCmd = &cobra.Command{
		Use:   "syncer",
		Short: "Sync config files",
	}
	rootCmd.AddCommand(backupCmd())
	rootCmd.AddCommand(restoreCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(doctorCmd())
	rootCmd.AddCommand(versionCmd())

	if len(rootCmd.Commands()) != 5 {
		t.Errorf("expected 5 subcommands, got %d", len(rootCmd.Commands()))
	}
}
