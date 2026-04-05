# Architecture

## Overview

syncer is a CLI tool that synchronizes application configuration files across machines using cloud storage (iCloud, Dropbox, Google Drive, etc.). Its key differentiator is **automatic mode detection** — it decides whether to use symlinks or file copies on a per-application basis, so you never have to choose manually.

## Core Components

### Application Definitions

syncer ships with 600+ built-in application definitions written in YAML. You can also add your own custom definitions to override or extend the defaults.

Loading priority (highest to lowest):

1. **`$SYNCER_DIR/.syncers/`** — custom overrides (highest priority)
2. **Built-in library** (`configs/` directory embedded in the binary) — 600+ common apps

`$SYNCER_DIR` is syncer's storage root, which you can set via `--syncer_dir` or `syncer.yaml`'s `storage_path`. By dropping a YAML file into `$SYNCER_DIR/.syncers/<app>.yaml`, you override the built-in definition for that app.

### Configuration Discovery

syncer automatically discovers `syncer.yaml` in this order:

1. `SYNCER_CONFIG` environment variable
2. Cloud storage root directories (Dropbox, iCloud, Google Drive, OneDrive, Box)
3. `~/.config/syncer/syncer.yaml`
4. Falls back to defaults (sync all available apps)

### Storage Auto-Detection

If no custom path is set, syncer detects installed cloud storage services in this order:

- iCloud (`~/Library/Mobile Documents/com~apple~CloudDocs`)
- Dropbox
- Google Drive
- OneDrive
- Box
- Local fallback (`~/.config/syncer`)

The first directory that contains a `syncer/syncer.yaml` file becomes the sync root. You can override this with `--syncer_dir` or `storage_path`.

### Sync Engine

The engine executes actual backups and restores. It supports three modes:

- **backup**: Copies the latest content from home to the sync directory, then replaces the original with a symlink (link mode) or leaves it as-is (copy mode)
- **restore**: Restores content from the sync directory to home
- **dry-run**: Previews operations without modifying any files

The engine skips already-synced files to avoid redundant work and safely migrates external symlinks managed by other tools (e.g., Dropbox).

The `doctor` command runs both backup and restore in dry-run mode, printing a summary of what would happen. See [doctor.md](doctor.md) for details.

### External Resources

For resources not directly in your home directory (Git repos, downloaded archives, remote files), syncer provides external resource syncing. Common use cases:

- Clone the oh-my-zsh framework from GitHub
- Download and extract prebuilt binaries
- Fetch a single remote file

External resources are stored in `sync_dir/.syncer_external/` and connected to home via a two-layer symlink structure.

### History

All sync operations are appended to `sync_dir/.syncer-history.jsonl` as JSON lines. Each entry includes:

- Timestamp and command type (backup/restore)
- Apps processed and file count
- Success status, error message, and dry-run indicator
