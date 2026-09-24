# Dotenvless

Windows-first 本地 Secret 管理 CLI。把项目 Secret 存入项目目录之外的 Windows DPAPI 加密 Vault，在启动程序时通过环境变量注入。

当前版本：**0.1.0-dev**。支持 Windows 10/11，当前自动测试与构建目标为 Windows x64。尚未完成的独立系统和跨账户验收见[验证记录](docs/verification.md)，远程 CI 状态见 [GitHub Actions](https://github.com/Nikoace/Dotenvless/actions)。

## 构建

需要 [Go](https://go.dev/dl/) 1.26 或更新版本，以及 [Git for Windows](https://git-scm.com/downloads/win)。Go 版本要求以 [go.mod](go.mod) 为准；已在本机使用 Go 1.27.1 验证。运行构建后的 CLI 不需要安装 Go。

克隆仓库后，在仓库根目录执行：

```powershell
go mod download
go build -trimpath -o bin/dvl.exe ./cmd/dvl
.\bin\dvl.exe --help
```

以下示例假设 `dvl.exe` 所在目录已经加入 PATH；也可使用可执行文件的完整路径。首次下载 Go 依赖需要网络，CLI 自身运行时不访问网络。

## 使用

进入要管理的 Git 项目：

```powershell
dvl init
dvl set OPENAI_API_KEY
dvl set DATABASE_URL
dvl list
dvl run -- python app.py
dvl run -- npm run dev
dvl run -- ./gradlew bootRun
```

`set` 从 Windows 交互终端隐藏读取输入；Enter 保存，Backspace 删除字符，Ctrl+C 取消。变量名规范为大写，允许空值，值须为无 NUL 的 UTF-8 文本且不超过 32 KiB。不接受 `set KEY VALUE` 或重定向 stdin。

| 命令 | 行为 |
| --- | --- |
| `dvl init` | 幂等注册当前 Git 项目 |
| `dvl set KEY` | 隐藏输入并保存/覆盖该键 |
| `dvl list` | 仅输出排序键名 |
| `dvl unset KEY` | 删除该键；不存在时报错 |
| `dvl run -- COMMAND [ARG...]` | 注入当前项目 Secret，继承工作目录/标准流并透传退出码 |
| `dvl import FILE` | 整批导入；同名键冲突则全部拒绝，保留源文件 |
| `dvl import --overwrite FILE` | 显式允许覆盖已有键 |
| `dvl example` | 在 Git 根创建只含 `KEY=` 的 `.env.example`；拒绝覆盖 |
| `dvl status [DIRECTORY]` | 查看当前或指定目录所属项目的身份、键名、Vault 与 Git 状态 |
| `dvl --help` / `dvl --version` | 查看帮助/版本，无需位于 Git 项目 |

`status` 支持相对路径和项目子目录，也可从非 Git 目录查询；含空格路径用引号包裹：

```powershell
dvl status
dvl status "..\my app"
dvl status ..\my-app\src
```

### 从已有 .env 迁移

```powershell
dvl init
dvl import .env
dvl list
dvl run -- npm run dev
```

导入不会删除 `.env`；确认应用通过环境变量正常工作后，由你决定是否移除原文件。真实环境文件仍应被 Git 忽略；如果曾提交真实凭据，迁移不会清除 Git 历史或撤销凭据。

导入支持 UTF-8/BOM、LF/CRLF、注释、`export`、单/双引号和多行值；不展开变量、不执行命令，源文件上限 1 MiB。详见[导入语法](docs/specs/m6-import.md)。

### Windows 命令兼容

普通可执行文件直接启动；`.cmd/.bat` 自动通过系统 `cmd.exe` 执行，`./gradlew` 优先使用相邻 `gradlew.bat`。批处理参数中的引号、`% ! ^ & | < >`、控制字符和末尾反斜杠会被明确拒绝；需要 shell 语法时显式选择 shell。详见[参数契约](docs/specs/m5-run.md)。

## AI Agent Skill

仓库提供 [dotenvless skill](skills/dotenvless/SKILL.md)，用于让 AI 按现有 CLI 契约检查键名、迁移 `.env` 和注入环境变量启动应用。支持复制到 Codex CLI / IDE、Claude Code 等 Agent Skills 客户端；完整安装步骤见 [AI 接入说明](docs/ai-skill.md)。

安装后可这样调用（Codex 示例）：

```text
$dotenvless 检查当前项目的密钥配置，只报告已有和缺少的变量名。
$dotenvless 使用 Vault 中的密钥运行 npm run dev，先检查启动脚本是否会输出密钥。
```

Skill 不读取真实 `.env` 内容或接收密钥值；已有文件交给本机 `dvl import`，新值由用户在本机隐藏输入。它不提供沙箱或日志脱敏能力，实际执行仍要求访问对应 Windows 身份下的项目与 CLI。

## 存储与安全范围

- 默认 Vault：`%APPDATA%\dotenvless\vault.dat`，必须位于当前项目之外。每个值使用 DPAPI Current User 加密，并绑定项目 ID 与键名；写入采用进程锁和密文原子替换。
- 项目 ID 来自规范化物理 Git 根路径的 SHA-256。子目录与目录联接共享身份；不同 worktree 隔离。移动或重命名项目后需要重新导入。
- `list/status/example` 和 CLI 自身错误信息不显示 Secret 值，没有明文 `get` 命令。不创建临时明文 `.env`，不修改持久环境，无遥测、daemon 或运行时联网。
- Secret 会存在于 CLI 和子进程内存；子进程及其后代可以主动输出或发送它。DPAPI 不能隔离同一 Windows 身份下的其他程序，本工具不是 Agent sandbox。
- `status` 只检查声明范围内的 `.env` 文件名和 Git 状态，不检查文件内容，也不证明工作区不存在其他凭据。[检查范围](docs/specs/m8-status.md)
- APPDATA 可能受系统漫游或重定向设置影响。安全问题报告方式见 [SECURITY.md](SECURITY.md)。

## 开发与验证

```powershell
go vet ./...
go test -count=1 ./...
go build -trimpath -o bin/dvl.exe ./cmd/dvl
```

可选的 `scripts/dev.ps1` 优先使用 `.tools/go` 下的本地工具链，否则使用 PATH 中的 Go，并将 Go 缓存放入忽略的 `.cache` 目录；它不下载工具链。

GitHub [Windows CI](.github/workflows/windows.yml) 执行格式检查、依赖校验、vet、测试和构建，成功时提供 `dvl-windows-amd64` 构建产物。交互终端、第二 Windows 账户和可选 Gradle 验收需[单独执行](docs/manual-verification.md)；远程 CI 的结果以推送后实际运行为准。

采用 SDD/TDD：先定义行为与验收，再执行失败测试、实现和回归，记录真实证据后提交。

- [贡献说明](CONTRIBUTING.md) · [变更日志](CHANGELOG.md)
- [V0.1 规格](docs/specs/v0.1.md) · [设计决策](docs/design.md) · [推进计划](docs/plan.md)
- [验证记录](docs/verification.md) · [首次推送指南](docs/publishing.md)

## 许可证

当前暂不添加许可证；未声明采用 MIT、Apache-2.0 或其他开源许可证。
