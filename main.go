package main

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sunznx/syncer/internal/appdb"
	"github.com/sunznx/syncer/internal/color"
	"github.com/sunznx/syncer/internal/config"
	"github.com/sunznx/syncer/internal/external"
	"github.com/sunznx/syncer/internal/history"
	"github.com/sunznx/syncer/internal/storage"
	"github.com/sunznx/syncer/internal/syncengine"
)

var (
	flagDryRun        bool
	flagSyncerDir     string
	flagDoctorVerbose bool
	version           = "dev"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "syncer",
		Short: "Sync config files between home directory and cloud storage",
		Long: `Syncer backs up and restores configuration files from your home directory
to cloud storage (iCloud, Dropbox, Google Drive, etc.).

Applications are defined as YAML configs in the configs/ directory.
Use "syncer list" to see all supported applications.`,
	}

	// Disable cobra auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Add subcommands
	rootCmd.AddCommand(backupCmd())
	rootCmd.AddCommand(restoreCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(doctorCmd())
	rootCmd.AddCommand(versionCmd())

	// Persistent flags
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "show what would be done without making changes")
	rootCmd.PersistentFlags().StringVarP(&flagSyncerDir, "syncer_dir", "R", "", "syncer root directory (overrides auto-detected storage path)")

	// Execute
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func backupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup [app...]",
		Short: "Backup config files from home to sync directory",
		Long:  "Backup config files from home directory to sync directory.\nIf app names are specified, only backup those applications. Otherwise backup all.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackup(args)
		},
	}
	return cmd
}

func restoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [app...]",
		Short: "Restore config files from sync directory to home",
		Long:  "Restore config files from sync directory to home directory.\nIf app names are specified, only restore those applications. Otherwise restore all.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(args)
		},
	}
	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List supported applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			store, err := resolveStorage(cfg)
			if err != nil {
				return err
			}
			db := loadAppDB(store)

			names := db.List()
			for _, name := range names {
				fmt.Println(name)
			}
			return nil
		},
	}
}

func doctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor [app...]",
		Short: "Diagnose syncer configuration and preview operations",
		Long:  "Diagnose syncer configuration and preview backup/restore operations.\nBy default, only apps with pending changes are shown. Use -v to see all apps.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			store, err := resolveStorage(cfg)
			if err != nil {
				return err
			}
			syncDir, err := store.SyncDir()
			if err != nil {
				return fmt.Errorf("get sync dir: %w", err)
			}
			syncersDir, _ := store.SyncersDir()

			fmt.Println("=== syncer doctor ===")
			fmt.Println()
			fmt.Printf("Storage Driver:  %s\n", syncDir)
			if cfg.ConfigFile != "" {
				fmt.Printf("Config File:     %s\n", cfg.ConfigFile)
			} else {
				fmt.Printf("Config File:     (none, using defaults)\n")
			}
			fmt.Printf("Home Directory:  %s\n", cfg.HomeDir)
			fmt.Printf("Syncers Dir:     %s\n", syncersDir)
			fmt.Println()

			db := loadAppDB(store)

			var apps []*appdb.AppConfig
			if len(args) > 0 {
				for _, name := range args {
					app, err := db.Load(name)
					if err != nil {
						fmt.Fprintf(os.Stderr, "warning: unknown app %q\n", name)
						continue
					}
					apps = append(apps, app)
				}
			} else {
				apps, err = filterApps(db, cfg)
				if err != nil {
					return err
				}
			}

			overridden := countOverriddenApps(db, apps)
			fmt.Printf("Selected Apps:   %d total", len(apps))
			if overridden > 0 {
				fmt.Printf(" (%d custom override)", overridden)
			}
			if len(args) > 0 {
				fmt.Println("  [from command line]")
			} else if len(cfg.Applications.Apps) > 0 {
				fmt.Println("  [from config]")
			} else {
				fmt.Println("  [all available]")
			}
			if names := formatAppNames(apps, 10); names != "" {
				fmt.Printf("                 %s\n", names)
			}
			fmt.Println()

			fmt.Println("--- Backup Preview (dry-run) ---")
			statsB := previewSync(cfg.HomeDir, syncDir, apps, "backup", flagDoctorVerbose)
			fmt.Println()
			fmt.Println("--- Restore Preview (dry-run) ---")
			statsR := previewSync(cfg.HomeDir, syncDir, apps, "restore", flagDoctorVerbose)

			fmt.Println()
			fmt.Println("=== Summary ===")
			fmt.Printf("Backup:  %d apps need action, %d already synced\n", statsB.ActionApps, statsB.SyncedApps)
			fmt.Printf("Restore: %d apps need action, %d already synced\n", statsR.ActionApps, statsR.SyncedApps)

			return nil
		},
	}
	cmd.Flags().BoolVarP(&flagDoctorVerbose, "verbose", "v", false, "show all apps including already-synced ones")
	return cmd
}

type previewStats struct {
	ActionApps int
	SyncedApps int
	TotalFiles int
}

func countOverriddenApps(db *appdb.DB, apps []*appdb.AppConfig) int {
	count := 0
	for _, app := range apps {
		if db.IsOverridden(app.Name) {
			count++
		}
	}
	return count
}

func formatAppNames(apps []*appdb.AppConfig, limit int) string {
	if len(apps) == 0 {
		return ""
	}
	var names []string
	for _, app := range apps {
		names = append(names, app.Name)
	}
	if len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:limit], ", ") + fmt.Sprintf(", and %d more", len(names)-limit)
}

func previewSync(homeDir, syncDir string, apps []*appdb.AppConfig, command string, verbose bool) previewStats {
	var msgs []string
	engine := syncengine.New(homeDir, syncDir,
		syncengine.WithDryRun(),
		syncengine.WithCommand(command),
		syncengine.WithProgressCallback(func(msg string) {
			msgs = append(msgs, msg)
		}),
	)

	var stats previewStats
	for _, app := range apps {
		ensureFilesFromExternals(app)
		result, err := engine.Sync(app)
		if err != nil && err != syncengine.ErrAlreadySynced {
			fmt.Printf("  [%s] %s  error: %v\n", modeLabel(app), app.Name, err)
			stats.ActionApps++
			continue
		}
		if len(result.Files) > 0 {
			fmt.Printf("  [%s] %s  (%d files)\n", modeLabel(app), app.Name, len(result.Files))
			stats.ActionApps++
			stats.TotalFiles += len(result.Files)
			for _, msg := range msgs {
				fmt.Printf("      %s\n", msg)
			}
		} else if verbose {
			fmt.Printf("  [%s] %s  (already synced)\n", modeLabel(app), app.Name)
		}
		if len(result.Files) == 0 {
			stats.SyncedApps++
		}
		msgs = msgs[:0]
	}
	if stats.ActionApps == 0 && len(apps) > 0 {
		fmt.Println("  (all apps already synced, no actions needed)")
	} else if len(apps) == 0 {
		fmt.Println("  (no apps configured)")
	}
	return stats
}

func modeLabel(app *appdb.AppConfig) string {
	// Check app-level explicit mode first
	if app.Mode == "copy" {
		return "copy"
	}
	if app.Mode == "link" {
		return "link"
	}
	// Auto-detect: if any file matches plist pattern, show copy
	for _, f := range app.Files {
		if strings.Contains(f, "Library/Preferences/") && strings.HasSuffix(f, ".plist") {
			return "copy"
		}
	}
	return "link"
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("syncer version %s\n", version)
		},
	}
}

func runBackup(args []string) error {
	return runSync(args, "backup")
}

func runRestore(args []string) error {
	return runSync(args, "restore")
}

func runSync(args []string, command string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	store, err := resolveStorage(cfg)
	if err != nil {
		return err
	}
	syncDir, err := store.SyncDir()
	if err != nil {
		return fmt.Errorf("get sync dir: %w", err)
	}

	db := loadAppDB(store)

	// Initialize history manager
	histMgr := history.New(syncDir)

	// If specific app names provided, only process those
	if len(args) > 0 {
		var opts []syncengine.Option
		if flagDryRun {
			opts = append(opts, syncengine.WithDryRun())
		}
		opts = append(opts, syncengine.WithCommand(command))
		opts = append(opts, syncengine.WithProgressCallback(func(msg string) {
			fmt.Println(msg)
		}))
		engine := syncengine.New(cfg.HomeDir, syncDir, opts...)

		var appNames []string
		var totalFiles int
		var success bool
		var errMsg string

		for _, appName := range args {
			app, err := db.Load(appName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: unknown app %q\n", appName)
				errMsg = fmt.Sprintf("unknown app %q", appName)
				continue
			}

			// Pull external resources (git repos, archives, files)
			if len(app.External) > 0 {
				extMgr := external.New(syncDir, flagDryRun)
				for _, ext := range app.External {
					destPath, err := extMgr.Pull(app.Name, ext)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error pulling external %s: %v\n", ext.URL, err)
						errMsg = err.Error()
						continue
					}
					if destPath != "" {
						fmt.Printf("%s%s%s%s%s\n", color.Action("Pulled: "), color.Path(ext.URL), color.Arrow(), color.Path(destPath), color.Reset)
					}
				}
			}

			// Auto-generate files from external targets if no files defined
			ensureFilesFromExternals(app)

			result, err := engine.Sync(app)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error syncing %s: %v\n", app.Name, err)
				errMsg = err.Error()
				continue
			}
			if len(result.Files) == 0 {
				continue
			}
			fmt.Println(result.String())
			appNames = append(appNames, app.Name)
			totalFiles += len(result.Files)
		}

		success = errMsg == ""

		// Feedback when no actions needed
		if len(appNames) == 0 && success {
			verb := "backed up"
			if command == "restore" {
				verb = "restored"
			}
			fmt.Println(color.Info(fmt.Sprintf("All apps already %s, no actions needed.", verb)))
		}

		// Record history
		histMgr.Record(&history.Entry{
			Command:   command,
			Apps:      appNames,
			FileCount: totalFiles,
			Success:   success,
			Error:     errMsg,
			DryRun:    flagDryRun,
		})

		return nil
	}

	// Otherwise process all configured apps
	apps, err := filterApps(db, cfg)
	if err != nil {
		return err
	}

	var opts []syncengine.Option
	if flagDryRun {
		opts = append(opts, syncengine.WithDryRun())
	}
	opts = append(opts, syncengine.WithCommand(command))
	opts = append(opts, syncengine.WithProgressCallback(func(msg string) {
		fmt.Println(msg)
	}))
	engine := syncengine.New(cfg.HomeDir, syncDir, opts...)

	var appNames []string
	var totalFiles int
	var success bool
	var errMsg string

	for _, app := range apps {
		ensureFilesFromExternals(app)
		result, err := engine.Sync(app)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error syncing %s: %v\n", app.Name, err)
			errMsg = err.Error()
			continue
		}
		if len(result.Files) == 0 {
			continue
		}
		fmt.Println(result.String())
		appNames = append(appNames, app.Name)
		totalFiles += len(result.Files)
	}

	success = errMsg == ""

	// Feedback when no actions needed
	if len(appNames) == 0 && success {
		verb := "backed up"
		if command == "restore" {
			verb = "restored"
		}
		fmt.Println(color.Info(fmt.Sprintf("All apps already %s, no actions needed.", verb)))
	}

	// Record history
	histMgr.Record(&history.Entry{
		Command:   command,
		Apps:      appNames,
		FileCount: totalFiles,
		Success:   success,
		Error:     errMsg,
		DryRun:    flagDryRun,
	})

	return nil
}

func filterApps(db *appdb.DB, cfg *config.Config) ([]*appdb.AppConfig, error) {
	// Apps specified in config file
	if len(cfg.Applications.Apps) > 0 {
		var apps []*appdb.AppConfig
		for _, name := range cfg.Applications.Apps {
			app, err := db.Load(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: unknown app %q\n", name)
				continue
			}
			apps = append(apps, app)
		}
		return apps, nil
	}

	// All available apps
	names := db.List()
	var apps []*appdb.AppConfig
	for _, name := range names {
		app, err := db.Load(name)
		if err != nil {
			continue
		}
		apps = append(apps, app)
	}
	return apps, nil
}

// ensureFilesFromExternals auto-generates file entries from external targets
// when the app has no explicit files defined.
// Only adds the first external target to avoid duplicating nested targets.
func ensureFilesFromExternals(app *appdb.AppConfig) {
	if len(app.Files) > 0 || len(app.External) == 0 {
		return
	}
	// Only add the first external target (usually the main repository)
	// This avoids adding nested targets like .oh-my-zsh/custom/plugins/*
	if app.External[0].Target != "" {
		app.Files = append(app.Files, app.External[0].Target)
	}
}

func resolveStorage(cfg *config.Config) (storage.Storage, error) {
	// CLI flag takes highest priority
	if flagSyncerDir != "" {
		return storage.NewCustom(flagSyncerDir)
	}
	if cfg.Settings.StoragePath != "" {
		return storage.NewCustom(cfg.Settings.StoragePath)
	}
	return storage.NewDefault(cfg.HomeDir)
}

//go:embed configs/*
var builtinConfigsFS embed.FS

func loadAppDB(store storage.Storage) *appdb.DB {
	syncersDir, err := store.SyncersDir()
	if err != nil {
		// If SyncersDir fails, we'll still create a DB without syncersDir
		// and rely on builtin configs
		syncersDir = ""
	}

	return appdb.NewDB(
		appdb.WithSyncersDir(syncersDir),
		appdb.WithBuiltinFS(builtinConfigsFS),
	)
}
