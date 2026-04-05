# Doctor Command

`syncer doctor` diagnoses your configuration and previews what **backup** and **restore** would do — without touching any files.

## Usage

```bash
# Diagnose all apps (only shows apps that need action)
syncer doctor

# Diagnose specific apps
syncer doctor git zsh vscode

# Show all apps, including already-synced ones
syncer doctor -v
syncer doctor --verbose
```

## What It Shows

```text
=== syncer doctor ===

Storage Driver:  /Users/you/Library/Mobile Documents/com~apple~CloudDocs/syncer
Config File:     /Users/you/Dropbox/syncer/syncer.yaml
Home Directory:  /Users/you
Syncers Dir:     /Users/you/Library/Mobile Documents/com~apple~CloudDocs/syncer/.syncers

Selected Apps:   3 total (1 custom override)  [from command line]
                 git, zsh, vscode

--- Backup Preview (dry-run) ---
  [link] git  (3 files)
      Copying: ~/.gitconfig -> ~/Library/Mobile Documents/com~apple~CloudDocs/syncer/.gitconfig
      Would create symlink: ~/.gitconfig -> ~/Library/Mobile Documents/com~apple~CloudDocs/syncer/.gitconfig
  [link] zsh  (already synced, no actions needed)

--- Restore Preview (dry-run) ---
  [link] git  (3 files)
      Would create symlink: ~/.gitconfig -> ~/Library/Mobile Documents/com~apple~CloudDocs/syncer/.gitconfig

=== Summary ===
Backup:  1 apps need action, 1 already synced
Restore: 1 apps need action, 1 already synced
```

### Output Sections

| Field | Description |
|-------|-------------|
| `Storage Driver` | The resolved sync root directory |
| `Config File` | The loaded `syncer.yaml` path, if any |
| `Syncers Dir` | The `.syncers/` override directory |
| `Selected Apps` | Total apps, custom override count, and source |
| `Backup Preview` | Dry-run output for `syncer backup` |
| `Restore Preview` | Dry-run output for `syncer restore` |
| `Summary` | Count of apps needing action vs. already synced |

## Flags

- `-v`, `--verbose` — Show all apps, including those that are already in sync. By default, `doctor` only prints apps that would perform actual work.

## Difference from `--dry-run`

| | `backup --dry-run` | `doctor` |
|---|---|---|
| Runs one command | Yes | Shows both backup **and** restore |
| Shows config info | No | Yes (storage, config path, overrides) |
| Hides synced apps | Yes | Yes (default); use `-v` to show all |
| Summary stats | No | Yes |
