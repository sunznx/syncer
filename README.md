# syncer

**[中文](README.zh-CN.md)**

> A single-binary Go tool that syncs your app configs across machines via cloud storage.

Inspired by [Mackup](https://github.com/lra/mackup), rewritten in Go with zero Python dependencies.

## Why syncer?

Mackup makes you choose between **copy** and **link** modes. In practice:

- **Link mode** (symlink) is what you want 99% of the time — changes sync instantly, zero maintenance
- **Copy mode** is only needed for macOS plist files managed by `cfprefsd`
- On macOS Sonoma+, symlinks to `~/Library/Preferences/*.plist` simply don't work

**syncer eliminates the choice.** It automatically picks the right mode for every file path. No manual configuration, no guesswork.

## What It Does

1. Scans 600+ built-in app definitions (and any custom ones you add)
2. Automatically decides per app:
   - `Library/Preferences/*.plist` → **copy mode**
   - everything else → **link mode**
3. Backs up your dotfiles to cloud storage and keeps them in sync

## Quick Start

```bash
# Backup everything
syncer backup

# Restore everything on a new machine
syncer restore

# Sync specific apps only
syncer backup git zsh vscode

# Preview changes without touching files
syncer backup --dry-run
```

## Installation

```bash
go install github.com/sunznx/syncer@latest
```

Or grab a prebuilt binary from [Releases](../../releases).

## Commands

```
syncer backup  [app...]    # Backup home configs to sync directory
syncer restore [app...]    # Restore configs from sync directory
syncer list                # List all supported applications
syncer doctor  [app...]    # Diagnose config and preview operations
syncer version             # Show version
```

Global flags:
- `--dry-run` — Preview what would happen without making changes
- `--syncer_dir` / `-R` — Override the auto-detected storage path

### Doctor

`syncer doctor` shows your current configuration and previews what backup and restore would do — without touching any files. Useful for onboarding a new machine or troubleshooting sync issues.

```bash
# Check all apps (only shows apps that need action)
syncer doctor

# Check specific apps
syncer doctor git zsh

# Show all apps including already-synced ones
syncer doctor -v
```

## Storage Auto-Detection

syncer finds your cloud storage automatically. It checks each directory in order and uses the **first one that contains a `syncer/syncer.yaml` file** as the sync root:

1. iCloud (`~/Library/Mobile Documents/com~apple~CloudDocs`)
2. Dropbox
3. Google Drive
4. OneDrive
5. Box
6. Fallback: `~/.config/syncer`

If none of these contain `syncer/syncer.yaml`, syncer returns an error. You can override this behavior with `--syncer_dir` or in `syncer.yaml`.

## Configuration

Create a `syncer.yaml` to control what gets synced:

```yaml
applications:
  apps:
    - git
    - zsh
    - vscode
  ignore:
    - iterm2

settings:
  storage_path: ""  # empty = auto-detect
```

Config discovery order:
1. `SYNCER_CONFIG` env var
2. Cloud storage root (`~/Dropbox/syncer/syncer.yaml`, etc.)
3. `~/.config/syncer/syncer.yaml`
4. Defaults (sync all apps)

## Custom App Definitions

Drop a YAML file into `$SYNCER_DIR/.syncers/<app>.yaml` to override or add apps:

```yaml
name: MyApp
files:
  - .myapp/config
  - .myapp/settings.json
```

Custom configs always take priority over built-in definitions.

## External Resources

Pull in Git repos, archives, or remote files:

```yaml
name: oh-my-zsh
application: Oh My Zsh
external:
  - type: git
    url: https://github.com/ohmyzsh/ohmyzsh.git
    target: .oh-my-zsh

  - type: git
    url: https://github.com/zsh-users/zsh-syntax-highlighting.git
    target: .oh-my-zsh-custom/plugins/zsh-syntax-highlighting
    subpaths:
      - path: .
        target: .oh-my-zsh/custom/plugins/zsh-syntax-highlighting
```

Supported types: `git`, `archive` (`.tar.gz`, `.tar.bz2`, `.tar.zst`, `.zip`), `file`

See [docs/external.md](docs/external.md) for the full guide.

## How Sync Works

### Link Mode (default)
- Copies your file/dir to the sync directory
- Replaces the original with a symlink pointing to the sync dir
- Changes are instantly reflected via the symlink

### Copy Mode
- Copies your file/dir to the sync directory
- Leaves the original untouched
- Used automatically for macOS plist files in `Library/Preferences/`

### Resync
If both home and the sync directory have the file, **home wins**. syncer copies the home version to the sync directory and updates the symlink.

## Documentation

- [Architecture](docs/architecture.md)
- [Configuration](docs/configuration.md)
- [External Resources](docs/external.md)
- [Doctor Command](docs/doctor.md)

## License

MIT
