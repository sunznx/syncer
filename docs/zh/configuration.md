# 配置系统

## 用户配置文件（`syncer.yaml`）

syncer 会自动发现 `syncer.yaml`，查找顺序如下：

1. **`SYNCER_CONFIG` 环境变量** 指定的文件路径
2. **云存储根目录** 中的 `syncer/syncer.yaml`（按 Dropbox → iCloud → OneDrive → Google Drive → Box → `Library/Mobile Documents` 的顺序，第一个存在的生效）
3. **`~/.config/syncer/syncer.yaml`**
4. **未找到任何配置** — 使用默认行为（同步所有可用应用）

> 测试验证：环境变量优先级、云存储自动发现、本地兜底、无配置默认值

## 配置格式

```yaml
applications:
  apps:
    - git
    - zsh
    - vscode
  ignore:
    - iterm2

settings:
  storage_path: ""  # 留空使用自动检测，或填写自定义路径
```

### 字段说明

- `applications.apps`：显式指定要同步的应用列表。如果为空数组或不填写，则同步**所有**可用应用。
- `applications.ignore`：要从同步中排除的应用名称列表。
- `settings.storage_path`：自定义同步根目录路径。如果填写，将覆盖自动检测到的云存储路径。

> 测试验证：应用白名单解析、黑名单解析、空列表处理

## 应用定义

每个应用通过一个 YAML 文件描述需要同步哪些配置文件。

### 标准格式示例

```yaml
name: Git
files:
  - .gitconfig
  - .gitignore_global
mode: link
ignore:
  - "*.tmp"
```

### 字段说明

- `name`：应用名称（在 `syncer list` 和 CLI 中使用）
- `files`：相对于 `$HOME` 的文件或目录路径列表
- `mode`：同步模式，可选 `"link"` 或 `"copy"`。如果不填，默认为 `link`
- `ignore`：glob 通配符列表，匹配的文件会被跳过
- `external`：外部资源定义（如 Git 仓库、远程归档），详见 [external.md](external.md)

### 自定义应用定义

你可以创建自己的应用定义来覆盖内置配置，或添加 syncer 未内置的应用。自定义定义放在：

- `$SYNCER_DIR/.syncers/<应用名>.yaml`

文件名即为应用在 CLI 中的标识符（不含 `.yaml` 后缀）。

**注意**：`$SYNCER_DIR` 就是 syncer 的同步根目录，可以通过 `--syncer_dir` 参数或 `storage_path` 设置来指定。这意味着应用覆盖目录完全由你控制——换一个同步根目录，就会读取该目录下 `.syncers/` 里的自定义定义。

> 测试验证：自定义定义覆盖内置、命令行参数指定优先级目录、回退到内置定义

### YAML 解析能力

syncer 支持解析以下形式的配置：
- 包含 `name`、`files` 和可选 `mode` 的基本配置
- 空的 `files: []`
- `ignore` 忽略模式列表
- 省略 `mode` 时自动默认为 link 模式

> 测试验证：基本解析、copy 模式解析、ignore 解析、空文件列表处理

## 存储路径

syncer 需要一个同步根目录来存放所有备份的配置文件。你可以通过以下方式指定：

### 自动检测（推荐）

不设置 `storage_path` 时，syncer 会自动检测本地安装的云存储服务，顺序为：

1. iCloud
2. Dropbox
3. Google Drive
4. OneDrive
5. Box
6. `~/.config/syncer`（本地兜底）

检测到的第一个可用目录即作为同步根目录。

> 测试验证：云存储自动检测、无可用存储时的错误处理

### 手动指定

优先级从高到低：
1. 命令行参数 `--syncer_dir` / `-R`
2. `syncer.yaml` 中的 `settings.storage_path`
3. 自动检测

> 测试验证：自定义路径保留、home 目录关联
