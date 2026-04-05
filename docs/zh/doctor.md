# Doctor 命令

`syncer doctor` 用于诊断配置，并预览 **backup** 和 **restore** 分别会执行什么操作——不会修改任何文件。

## 用法

```bash
# 诊断所有应用（默认只显示需要执行动作的应用）
syncer doctor

# 诊断指定应用
syncer doctor git zsh vscode

# 显示所有应用，包括已同步的
syncer doctor -v
syncer doctor --verbose
```

## 输出说明

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

### 字段说明

| 字段 | 说明 |
|------|------|
| `Storage Driver` | 解析后的同步根目录 |
| `Config File` | 加载的 `syncer.yaml` 路径（如果有） |
| `Syncers Dir` | 自定义覆盖目录 `.syncers/` |
| `Selected Apps` | 总应用数、自定义覆盖数、来源说明 |
| `Backup Preview` | `syncer backup` 的 dry-run 预览 |
| `Restore Preview` | `syncer restore` 的 dry-run 预览 |
| `Summary` | 需要执行动作的应用数 vs. 已同步的应用数 |

## 标志

- `-v`, `--verbose` — 显示所有应用，包括已经同步的。默认情况下 `doctor` 只显示有实际工作要做的应用。

## 与 `--dry-run` 的区别

| | `backup --dry-run` | `doctor` |
|---|---|---|
| 只运行一个命令 | 是 | 同时显示 backup **和** restore |
| 显示配置信息 | 否 | 是（存储路径、配置文件、覆盖数量） |
| 隐藏已同步应用 | 是 | 是（默认）；`-v` 可显示全部 |
| 统计摘要 | 否 | 是 |
