# Dotenvless

Windows-first 本地 Secret 管理 CLI。真实值保存在项目外的 DPAPI 加密 Vault 中，通过子进程环境变量供应用使用。

M0–M8 功能已实现，当前版本为 0.1.0-dev。跨 Windows 账户、独立 Windows 10/11 与远程 CI 验收仍有待办，见[验证记录](docs/verification.md)和[独立环境验收](docs/manual-verification.md)。

## 使用

在 Git 项目中运行：

```powershell
dvl init
dvl set OPENAI_API_KEY
dvl set DATABASE_URL
dvl list
dvl run -- python app.py
dvl run -- uv run main.py
dvl run -- npm run dev
dvl run -- ./gradlew bootRun
```

set 需要交互式 Windows 终端，输入不会显示字符。Enter 保存，Backspace 删除字符，Ctrl+C 取消并恢复终端。变量名规范成大写；允许空值，值必须是无 NUL 的 UTF-8 文本且至多 32 KiB。不接受 Secret 位置参数或重定向 stdin。

| 命令 | 行为 |
| --- | --- |
| dvl init | 幂等注册当前项目 |
| dvl set KEY | 隐藏输入并保存/覆盖该键 |
| dvl list | 仅输出排序键名 |
| dvl unset KEY | 删除该键，不存在时报错 |
| dvl run -- COMMAND [ARG...] | 注入当前项目 Secret，继承工作目录/标准流，透传退出码 |
| dvl import FILE | 原子导入；同名键默认使整次失败 |
| dvl import --overwrite FILE | 明确允许覆盖，源文件始终保留 |
| dvl example | 在 Git 根创建只含 KEY= 的 .env.example；拒绝覆盖 |
| dvl status [DIRECTORY] | 查看当前或指定目录所属项目的身份、键名、Vault 与环境文件/Git 状态 |
| dvl --help / --version | 帮助/版本，不要求 Git 项目 |

status 无参数时查看当前目录所属的 Git 项目；指定目录时查看该目录所属项目，支持相对路径和项目子目录，也可从非 Git 目录调用。含空格路径用引号包裹：

```powershell
dvl status
dvl status ..\example-project
dvl status ..\relations\src
```

import 支持 UTF-8/BOM、CRLF、注释、export、单/双引号和多行；不执行变量展开或命令替换。源文件上限 1 MiB。[完整语法](docs/specs/m6-import.md)

run 对普通程序直接传参，自动识别 .cmd/.bat；./gradlew 优先使用相邻 gradlew.bat。批处理的引号、%、!、^、&、|、<、>、控制字符和末尾反斜杠参数会被明确拒绝；需要此类语法时显式选择 shell。普通程序不受这一批处理限制。[参数契约](docs/specs/m5-run.md)

## 存储与项目身份

默认位置为 %APPDATA%\dotenvless\vault.dat。Vault 必须位于当前项目之外；测试使用隔离路径，不触碰真实 Vault。每个 Secret 由 Windows DPAPI Current User scope 加密，项目 ID 与键名绑定到记录。文件修改使用跨进程锁与密文原子替换；损坏或未知格式不会被自动重置。

项目 ID 为规范化物理 Git 根路径的 SHA-256。子目录与目录联接识别为同一项目；不同 worktree、同名不同路径仓库隔离。移动或重命名目录后需要重新导入；非 Git 目录报错。

## 安全范围

Dotenvless 自身没有网络访问、遥测、云账户、更新检查或 daemon，不写入持久环境变量，不生成临时明文 .env。list/status/example 和自身错误信息不显示 Secret value；V0.1 没有 get 命令。

Secret 会存在于 dvl 和目标进程内存及目标环境中。目标应用及其后代可以主动输出或传递 Secret，输出会原样传回。DPAPI 不隔离同一 Windows 身份下的其他程序，V0.1 不提供 Agent sandbox。依据：[Microsoft DPAPI](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata)。

status 检查 .env / .env.* 文件名，不读取其内容；模板、Git/依赖/构建/缓存目录和目录链接被排除，命令会显示范围。Git 状态区分 tracked / ignored / not ignored / unknown；发现环境文件属于提示，损坏 Vault 或扫描失败返回非零。[检查范围](docs/specs/m8-status.md)

APPDATA 可能由系统重定向或漫游；工具自身离线并不改变系统对该目录的同步设置。

## 构建与测试

目标平台：Windows 10/11。开发使用 Go 1.27.1。运行生成的 dvl.exe 不需要 Go 或额外运行时。

```powershell
go test ./...
go vet ./...
go build -trimpath -o bin/dvl.exe ./cmd/dvl
.\bin\dvl.exe --help
```

本工作区已下载并校验官方 Go ZIP，工具链位于忽略的 .tools/go；隔离缓存包装脚本也可使用：

```powershell
.\scripts\dev.ps1 test ./...
.\scripts\dev.ps1 vet ./...
.\scripts\dev.ps1 build -buildvcs=false -trimpath -o bin/dvl.exe ./cmd/dvl
```

本任务的 Git 目录由沙箱身份创建。如本机 Git 报 dubious ownership，可仅对该次命令使用 `git -c safe.directory="$(Get-Location)" status`。上面的 -buildvcs=false 可避开构建时读取 Git 状态；没有更改全局 Git 信任设置。

已实际验证：真实 DPAPI、跨进程写入与退出释放锁、隐藏输入/取消、Ctrl+C、完整 CLI→Python 流程，以及 Python/Node/PowerShell/CMD/npm/Gradle 的环境注入。交互和跨账户测试在默认测试中显式跳过，需要专门运行；真实 Gradle 测试通过 DVL_TEST_GRADLE 指向本机发行版，使用独立缓存、离线与 --no-daemon。

实际项目集成：已在外部测试项目 relations 中使用隔离 Vault 和假值运行 Node/npm，44 项项目测试及类型检查通过；实际 API/研究模块完成离线图谱持久化验证，项目原有文件保持不变。见 [relations 验收范围](docs/specs/integration-relations.md)和[执行记录](docs/verification.md)。

## 开发流程

采用 SDD（规格驱动）与 TDD：规格/验收 ID → 真实失败测试 → 最小实现 → 回归与重构 → 文档更新 → Git 提交。关键设计先讨论，按 M0–M8 顺序推进。

- [V0.1 规格](docs/specs/v0.1.md)
- [设计决策](docs/design.md)
- [里程碑与测试计划](docs/plan.md)
- [执行证据](docs/verification.md)
- [独立环境验收](docs/manual-verification.md)

云同步、团队分享、Web UI、Secret 内容扫描、Linux/macOS 与 Agent 权限隔离均未加入 V0.1。许可证尚未指定，不预设开源授权。
