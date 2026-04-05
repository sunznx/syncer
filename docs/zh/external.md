# External 资源

External 资源允许您从外部源（如 Git 仓库、归档或 URL）直接同步文件到您的主目录。

## 支持的类型

### Git

克隆 Git 仓库到主目录：

```yaml
name: vim-plugins
application: Vim 插件
external:
  - type: git
    url: https://github.com/user/vim-plugins.git
    target: .vim/pack
```

执行流程：
1. 克隆仓库到 `sync_dir/.syncer_external/vim-plugins/`
2. 在 `~/.vim/pack` 创建指向克隆仓库的符号链接

### Archive

下载并解压归档：

```yaml
name: neovim
application: Neovim
external:
  - type: archive
    url: https://github.com/neovim/neovim/releases/download/v0.10.0/nvim-macos.tar.gz
    target: .local/bin
```

支持的归档格式：
- `.tar.gz`, `.tgz` - gzip 压缩 tar
- `.tar.bz2`, `.tbz2` - bzip2 压缩 tar
- `.tar.zst` - zstd 压缩 tar
- `.zip` - ZIP 归档

### File

下载单个文件：

```yaml
name: fzf
application: fzf (模糊查找器)
external:
  - type: file
    url: https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
    target: .vim/autoload/plug.vim
```

## 配置字段

### 通用字段

- `type`: 资源类型 (`git`, `archive`, `file`)
- `url`: 外部资源的 URL
- `target`: 主目录中的目标位置（相对路径，如 `.vim` 或 `./bin`）
- `executable`: 使目标文件可执行（`chmod +x`）

## 子路径 (Subpaths)

要从外部资源中选择性同步，使用 `subpaths`：

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

字段说明：
- `path`: 外部资源中的源路径（必需）
- `target`: 相对于主目录的目标路径（可选，默认为 path）
- `executable`: 使目标文件可执行（默认：false）

## 存储位置

External 资源存储在：
```
sync_dir/.syncer_external/
```

这样可以将外部仓库与同步文件分开存储。
