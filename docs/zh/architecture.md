# syncer 产品架构

## 概述

syncer 是一款用于在多设备间同步应用程序配置文件的命令行工具。它通过云存储（如 iCloud、Dropbox、Google Drive 等）作为中转，自动将本地 home 目录中的配置文件备份到云端，或从云端恢复到本地。

与同类工具最大的不同在于：syncer **自动判断每个应用应该使用符号链接（link）还是文件拷贝（copy）**，无需用户手动选择。

## 核心组件

### 应用定义系统

syncer 内置了数百个常见应用的配置文件定义，以 YAML 格式描述每个应用需要同步哪些文件。用户也可以添加自己的自定义定义。

应用定义的来源按优先级从高到低依次为：

1. **自定义覆盖目录**（`$SYNCER_DIR/.syncers/`）— 最高优先级
2. **内置定义库**（编译在二进制中的 `configs/` 目录）— 包含 600+ 常见应用

其中，`$SYNCER_DIR` 就是 syncer 的同步根目录，可以通过 `--syncer_dir` 参数或 `syncer.yaml` 中的 `storage_path` 指定。也就是说，**最高优先级的覆盖目录是由用户通过配置或命令行参数直接决定的**。只要在这个目录下创建 `.syncers/<应用名>.yaml`，就能覆盖内置定义。

> 测试验证：多级优先级覆盖、命令行参数指定自定义目录、自定义定义生效、合并去重

### 配置发现系统

syncer 会自动寻找用户配置文件 `syncer.yaml`，查找顺序为：

1. `SYNCER_CONFIG` 环境变量指定的路径
2. 各个云存储根目录（Dropbox、iCloud、Google Drive、OneDrive、Box 等）
3. `~/.config/syncer/syncer.yaml`
4. 如果都没有找到，则使用默认行为（同步所有可用应用）

> 测试验证：环境变量优先、云存储自动发现、本地兜底、无配置默认值

### 存储自动检测

如果用户没有在配置中指定存储路径，syncer 会自动检测本地已安装的云存储服务，按以下顺序尝试：

- iCloud（`~/Library/Mobile Documents/com~apple~CloudDocs`）
- Dropbox
- Google Drive
- OneDrive
- Box
- 本地兜底（`~/.config/syncer`）

检测到的第一个包含 `syncer/syncer.yaml` 的目录即为同步根目录。用户也可以通过配置文件或 `--syncer_dir` 命令行参数强制指定。

> 测试验证：自动检测逻辑、自定义路径、home 目录关联

### 同步引擎

同步引擎负责实际执行文件的备份和恢复。它支持三种运行模式：

- **备份（backup）**：将 home 中的最新内容复制到同步目录，并在 home 中建立符号链接（link 模式）或保持原样（copy 模式）
- **恢复（restore）**：将同步目录中的内容恢复到 home
- **Dry-run**：只预览将要执行的操作，不修改任何文件

引擎会自动跳过已经正确同步的文件，避免重复操作；也会检测并安全迁移由其他工具（如 Dropbox）管理的外部符号链接。

`doctor` 命令会同时以 dry-run 模式运行 backup 和 restore，并输出摘要。详见 [doctor.md](doctor.md)。

> 测试验证：备份/恢复流程、重复同步无操作、外部符号链接迁移、dry-run 不修改文件

### 外部资源管理

对于不直接存放在 home 目录中的资源（如 Git 仓库、下载的归档文件、远程文件），syncer 提供了外部资源同步能力。常见场景包括：

- 从 GitHub 克隆 oh-my-zsh 框架
- 下载并解压预编译二进制包
- 拉取远程单文件

外部资源会存放在 `sync_dir/.syncer_external/` 中，并通过双层符号链接结构连接到 home 目录。

> 测试验证：双层符号链接结构、整仓库同步、子路径同步

### 历史记录

所有同步操作都会被记录到 `sync_dir/.syncer-history.jsonl` 中，方便用户回溯操作历史。记录内容包括：

- 操作时间、命令类型（backup/restore）
- 处理的应用和文件数量
- 是否成功、错误信息
- 是否为 dry-run

> 测试验证：记录追加、列表查询、格式化输出
