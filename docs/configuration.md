# Configuration

## User Config (`syncer.yaml`)

syncer discovers `syncer.yaml` in the following order:

1. **`SYNCER_CONFIG`** environment variable
2. **Cloud storage root directories** (Dropbox → iCloud → OneDrive → Google Drive → Box → `Library/Mobile Documents`) — first existing file wins
3. **`~/.config/syncer/syncer.yaml`**
4. **No config found** — use defaults (sync all available apps)

## Config Format

```yaml
applications:
  apps:
    - git
    - zsh
    - vscode
  ignore:
    - iterm2

settings:
  storage_path: ""  # Leave empty for auto-detect, or set a custom path
```

### Fields

- `applications.apps` — explicit allowlist of apps to sync. Empty means **all**.
- `applications.ignore` — list of app names to exclude.
- `settings.storage_path` — custom sync root path. If set, overrides auto-detection.

## Application Definitions

Each app is described by a YAML file:

```yaml
name: Git
files:
  - .gitconfig
  - .gitignore_global
mode: link
ignore:
  - "*.tmp"
```

### Fields

- `name` — app identifier (used in `syncer list` and CLI)
- `files` — list of paths relative to `$HOME`
- `mode` — `"link"` or `"copy"` (optional, defaults to link)
- `ignore` — glob patterns to skip
- `external` — external resources (see [external.md](external.md))

### Custom Definitions

You can create custom definitions to override built-ins or add apps syncer doesn't know about:

- `$SYNCER_DIR/.syncers/<app>.yaml` (highest priority)

The filename (without `.yaml`) becomes the app's CLI identifier.

### Storage Path

You can specify the sync root via:

1. `--syncer_dir` / `-R` CLI flag (highest priority)
2. `syncer.yaml` → `settings.storage_path`
3. Auto-detection — finds the first cloud storage directory containing `syncer/syncer.yaml`
