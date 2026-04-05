# External Resources

External resources allow you to sync files from external sources like Git repositories, archives, or URLs directly to your home directory.

## Supported Types

### Git

Clone a Git repository to your home directory:

```yaml
name: vim-plugins
application: Vim Plugins
external:
  - type: git
    url: https://github.com/user/vim-plugins.git
    target: .vim/pack
```

This will:
1. Clone the repository to `sync_dir/.syncer_external/vim-plugins/`
2. Create a symlink at `~/.vim/pack` pointing to the cloned repository

### Archive

Download and extract an archive:

```yaml
name: neovim
application: Neovim
external:
  - type: archive
    url: https://github.com/neovim/neovim/releases/download/v0.10.0/nvim-macos.tar.gz
    target: .local/bin
```

Supported archive formats:
- `.tar.gz`, `.tgz` - gzip compressed tar
- `.tar.bz2`, `.tbz2` - bzip2 compressed tar
- `.tar.zst` - zstd compressed tar
- `.zip` - ZIP archive

### File

Download a single file:

```yaml
name: fzf
application: fzf (fuzzy finder)
external:
  - type: file
    url: https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
    target: .vim/autoload/plug.vim
```

## Configuration Fields

### Common Fields

- `type`: Resource type (`git`, `archive`, `file`)
- `url`: URL of the external resource
- `target`: Target location in home directory (relative, like `.vim` or `./bin`)
- `executable`: Make the target file executable (`chmod +x`)

## Subpaths

For selective syncing from external resources, use `subpaths`:

```yaml
name: oh-my-zsh
application: Oh My Zsh
external:
  - type: git
    url: https://github.com/ohmyzsh/ohmyzsh.git
    target: .oh-my-zsh
    subpaths:
      - path: custom/
      - path: plugins/zsh-syntax-highlighting
        target: .oh-my-zsh/custom/plugins/zsh-syntax-highlighting
```

Fields:
- `path`: Source path in external resource (required)
- `target`: Target path relative to home directory (optional, defaults to path)
- `executable`: Make the target file executable (default: false)

## Storage Location

External resources are stored in:
```
sync_dir/.syncer_external/
```

This keeps external repositories separate from your synced files.
