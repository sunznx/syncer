# syncer

> 一个通过云存储在多台机器间同步应用程序配置文件的单一二进制 Go 工具。

灵感来自 [Mackup](https://github.com/lra/mackup)，用 Go 重写，零 Python 依赖。

## 为什么选择 syncer？

Mackup 要求你为每个应用在**复制**和**链接**模式之间做选择。但实际上：

- **链接模式**（符号链接）在 99% 的场景下都是最佳选择 — 变更实时同步，无需维护
- **复制模式** 只适用于少数由 macOS `cfprefsd` 管理的 plist 文件
- 在 macOS Sonoma+ 上，`~/Library/Preferences/*.plist` 的符号链接根本不可靠

**syncer 消除了这个选择。** 它会根据文件路径自动为每个应用选择正确的模式。无需手动配置，无需猜测。

## 它能做什么

1. 扫描 600+ 内置应用定义（以及你添加的自定义定义）
2. 自动为每个应用决定同步模式：
   - `Library/Preferences/*.plist` → **复制模式**
   - 其他所有路径 → **链接模式**
3. 将你的 dotfiles 备份到云存储并保持同步

## 快速开始

```bash
# 备份所有应用
syncer backup

# 在新机器上恢复所有配置
syncer restore

# 仅同步指定应用
syncer backup git zsh vscode

# 预览变更（不修改任何文件）
syncer backup --dry-run
```

## 安装

```bash
go install github.com/sunznx/syncer@latest
```

或从 [Releases](../../releases) 下载预编译二进制文件。

## 命令

```
syncer backup  [app...]    # 将 home 中的配置备份到同步目录
syncer restore [app...]    # 从同步目录恢复配置到 home
syncer list                # 列出所有支持的应用
syncer doctor  [app...]    # 诊断配置并预览操作
syncer version             # 显示版本
```

全局标志：
- `--dry-run` — 预览将要执行的操作，不修改任何文件
- `--syncer_dir` / `-R` — 覆盖自动检测到的存储路径

### Doctor 诊断

`syncer doctor` 显示当前配置并预览 backup 和 restore 的操作 — 不会修改任何文件。适合在新机器上初始化或排查同步问题时使用。

```bash
# 检查所有应用（仅显示需要操作的应用）
syncer doctor

# 检查指定应用
syncer doctor git zsh

# 显示所有应用（包括已同步的）
syncer doctor -v
```

## 存储自动检测

syncer 按以下顺序自动寻找云存储。它会依次检查这些目录，并使用**第一个包含 `syncer/syncer.yaml` 文件的目录**作为同步根目录：

1. iCloud（`~/Library/Mobile Documents/com~apple~CloudDocs`）
2. Dropbox
3. Google Drive
4. OneDrive
5. Box
6. 本地回退：`~/.config/syncer`

如果以上目录都不包含 `syncer/syncer.yaml`，syncer 会报错。你可以通过 `--syncer_dir` 或 `syncer.yaml` 覆盖此行为。

## 配置

创建 `syncer.yaml` 来控制同步范围：

```yaml
applications:
  apps:
    - git
    - zsh
    - vscode
  ignore:
    - iterm2

settings:
  storage_path: ""  # 留空表示自动检测
```

配置文件查找顺序：
1. `SYNCER_CONFIG` 环境变量
2. 云存储根目录（`~/Dropbox/syncer/syncer.yaml` 等）
3. `~/.config/syncer/syncer.yaml`
4. 默认值（同步所有应用）

## 自定义应用定义

在 `$SYNCER_DIR/.syncers/<应用名>.yaml` 中放置 YAML 文件即可覆盖或新增应用：

```yaml
name: MyApp
files:
  - .myapp/config
  - .myapp/settings.json
```

自定义配置永远优先于内置定义。

## 外部资源

你可以拉取 Git 仓库、归档文件或远程文件：

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

支持类型：`git`、`archive`（`.tar.gz`、`.tar.bz2`、`.tar.zst`、`.zip`）、`file`

完整说明见 [docs/zh/external.md](docs/zh/external.md)。

## 同步机制

### 链接模式（默认）
- 将文件/目录复制到同步目录
- 在原位置创建指向同步目录的符号链接
- 通过符号链接，变更会即时反映

### 复制模式
- 将文件/目录复制到同步目录
- 保留原始文件不变
- 自动用于 macOS `Library/Preferences/` 下的 plist 文件

### 重新同步
当 home 和同步目录都存在同名文件时，**以 home 中的版本为准**。syncer 会将 home 的内容复制到同步目录并更新符号链接。

## 文档

- [产品架构](docs/zh/architecture.md)
- [配置系统](docs/zh/configuration.md)
- [外部资源](docs/zh/external.md)
- [Doctor 命令](docs/zh/doctor.md)

## 许可证

MIT
