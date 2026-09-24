# 验证记录

只记录实际执行的命令和观察结果。没有运行的测试不写「通过」。不记录真实 Secret，不粘贴完整敏感环境。

## 2026-09-24 / S0：规格准备

| 检查 | 结果 |
| --- | --- |
| 读取用户项目说明（UTF-8） | 完成；核心 M0–M5，后续 M6–M8 |
| 检查当前目录 | 起始为空 |
| 查找适用 AGENTS.md | 当前目录及检查的父目录未发现 |
| `git rev-parse --show-toplevel` | 解析到父仓库，尚无独立本项目仓库 |
| `git status --short --branch` | 父仓库 main，含其他项目未提交修改 |
| Go 可用性 | PATH 与检查的常见安装位置未找到 |
| DPAPI 文档 | 已核对 Microsoft Current User scope、安全边界与缓冲释放要求 |
| 单元测试 / 构建 | 未执行；尚无 Go 项目与工具链 |

D-01、D-02、D-03 已由用户确认。已执行 git init --initial-branch=main 建立独立仓库；未操作父仓库暂存区。

## 后续记录模板

```text
日期 / Milestone / Spec ID:
前置决策:
Red 命令:
Red 观察到的失败及原因:
Green 命令及结果:
Refactor / 回归结果:
真实 Windows 验收:
未验证项和原因:
对应 Git 提交:
```

## 2026-09-24 / M0 / CLI-01、SEC-03

- 工具链：Go 1.27.1 windows/amd64；官方 ZIP SHA-256 `a3911b5e0e1b1053f25ed0675f4c1c6aad1e2bfcf253df2b9be4caabd2edd95d` 已匹配。
- Red：`scripts/dev.ps1 test ./...`，帮助/短帮助/版本得到 exit 1；无效参数得到 exit 1 而期望 2，测试失败。
- Green：添加最小 CLI 实现后，相同命令全部通过。
- 修正：PowerShell 高级参数绑定会吞掉 Go 的 `-o`，改为原始参数转发，实际构建复测通过。
- 回归：`go vet ./...`、`go test ./...`、`go build -trimpath -o bin/dvl.exe ./cmd/dvl` 均成功。
- 冒烟：二进制 `--help` 输出用法，`--version` 输出 `dvl 0.1.0-dev`，均返回 0。
- Windows CI 文件已建立，但没有远程运行结果。
- 环境限制：沙箱创建的 `.git` 属于另一 Windows 身份；尝试调整该目录所有者被 OS 拒绝，未改变权限。开发命令使用进程级 `safe.directory` 精确信任本仓库根目录，未修改全局 Git 配置。普通沙箱启动仍异常。

## 2026-09-24 / M1 / PROJ-01..05、M1-01..10

- Red：`go test ./internal/project ./internal/cli`；根/子目录、隔离、worktree、目录联接因失败桩报错，status 返回 2，测试失败。
- 首次实现后，真实目录联接仍产生不同 ID。保留该失败并诊断为仅 `filepath.EvalSymlinks` 未得到该环境下的物理目标。
- 改为 `CreateFile` + `GetFinalPathNameByHandle` 获取实际目录，再规范化路径。Windows API 由固定的 `golang.org/x/sys v0.48.0` 提供。
- Green：`go test ./...` 全部通过，包括本机真实目录联接、大小写别名、最近仓库、gitdir 文件、非 Git、非法 marker 不回显等检查。
- `go vet ./...`、Windows 构建以及实际 `bin/dvl.exe status` 成功。
- 未验证：网络 UNC 共享、特殊文件系统、需要系统配置才能创建的区分大小写目录。实现保留实际目录名大小写，没有全路径 lower-case 合并。

## 2026-09-24 / M2 / SEC-07..09

- Red：存储层 CRUD、初始化、失败保留与并发测试因 not implemented 桩失败。
- Green：实现后 go test ./internal/vault 通过。覆盖重复 JSON 键/未知版本/结构损坏、当前项目隔离、大小写键、空值、加密/解密失败不回显敏感异常、替换失败旧文件保持逐字节一致。
- 12 个并发更新无丢失；本里程碑是多 goroutine，跨进程证据留待真实 DPAPI 集成。
- 故障注入验证临时密文文件清理。测试 Protector 仅存在于 *_test.go，磁盘没有测试假 Secret 明文。
- go test ./...、go vet ./...、Windows build 全部通过；生产 CLI 仍只提供帮助、版本、身份 status。

## 2026-09-24 / M3 / SEC-01/04、DPAPI 与进程并发

- Red：真实 DPAPI 往返/随机性及持久化测试在失败桩上失败。
- 第一轮实现揭示 Windows DPAPI 对零长度输入失败；使用受保护的载荷版本字节支持空值，再次测试通过。
- Green：真实 Windows DPAPI 空值、Unicode、多行往返，篡改/错误上下文/非法密文拒绝，重复加密不同密文均通过。
- 真实 Vault 重开成功；6 个独立进程更新同一 Vault，最终 7 个键全部保留；磁盘没有测试假 Secret 明文。
- 持锁子进程直接退出、不执行 unlock，父进程随后成功 Init，验证 OS 释放锁；残留锁文件不造成阻塞。
- go test ./...、go vet ./...、Windows build 全部通过。
- 未验证：第二个 Windows 账户解密拒绝、Windows 10 独立机器；没有创建系统账户或声称这些验收通过。

## 2026-09-24 / M4 / INIT-01、CRUD-01/02、SEC-02/03/05

- Red：配置解析、init/set/list/unset 和输入编辑用例分别在失败桩上失败。
- Green：真实 DPAPI 的 CLI CRUD、重复 init 保留键、读取失败旧 Vault 不变、仅输出键名、未初始化/非终端拒绝、项目目录不生成文件均通过。
- 输入测试覆盖空值、Unicode 退格、Ctrl+C/EOF 取消、NUL/非法 UTF-8/过长值拒绝，不静默截断。官方 x/term v0.46.0 用于终端模式；本地读取循环控制取消与长度。
- 使用独立 Windows PTY 实际输入 FAKE_TTY_SECRET_2468；输出未出现该值，测试确认结果正确且前后 ConsoleMode 完全相同。
- 第二次 Windows PTY 输入测试假值后 Ctrl+C，取消成功、无返回值、前后 ConsoleMode 相同。
- go test ./...、go vet ./... 和 Windows 构建通过；交互测试默认在无人值守测试中显式跳过，以上单独执行结果为证据。

## 2026-09-24 / M5 / RUN-01、SEC-01/05

- Red：真实子进程参数/流/退出码、批处理、环境合并测试均在失败桩上失败；CLI run 集成初次返回 2 而非子进程的 23。
- Green：普通程序保留空参数、Unicode、引号、%、&、尾反斜杠；批处理支持安全子集并在启动前拒绝已列危险参数，gradlew.bat 自动解析成功。
- 验证继承 stdin/stdout/stderr/cwd，父环境不变，Windows 大小写变量覆盖，0/37/23 等退出码透传，没有临时 .env。
- 实际工具：Python 3.14.2、Node 25.2.1、Windows PowerShell、CMD、npm script 全部检查到假 Secret。
- 实际 Gradle 8.9 + Java 17：独立 GRADLE_USER_HOME，--offline --no-daemon，verifySecret 任务 BUILD SUCCESSFUL（17.77 秒）；没有运行其他项目构建。
- 真实 Windows 控制台 Ctrl+C 测试：子进程收到事件并退出 29，运行器得到 29，内部测试输出 CTRL_C_OK / PASS。承载测试的外层 PowerShell 同时收到事件，因此工具报告其退出 1；不把外层状态冒充 0。
- 完整二进制验收：隔离项目 init → 两次隐藏 set → list → run Python；Python 以哈希检查两值，输出 ENV_INJECTION_OK；Vault 无假值明文、项目没有 .env，整条流程 exit 0。
- go test ./...、go vet ./...、Windows build 通过。交互终端与 Gradle 的特定验收已单独执行，默认 CI 不假定具备这些条件。

## 2026-09-24 / M6 / IMPORT-01

- Red：dotenv 正常语法、Vault Import 与 CLI import 在失败桩上失败。
- Green：UTF-8/BOM/CRLF、注释、export、空值、单/双引号、转义、多行以及字面变量/命令文本均通过。
- 错误场景：非法语法、重复大小写键、非法 UTF-8/NUL、未知转义、超长文件/值；错误不回显原始 Secret 行。
- 默认冲突整次拒绝，显式 --overwrite 成功；真实 CLI 保留导入源文件。
- 故障注入：第一条和后续条目加密失败都保留旧 Vault 的逐字节内容，没有部分键落盘。
- go test ./...、go vet ./...、Windows build 全部通过。

## 2026-09-24 / M7 / EXAMPLE-01

- Red：CLI example 返回用法错误，根目录示例流程失败。
- Green：从项目子目录运行，根目录生成排序 KEY=，无假 Secret 值；重复执行返回错误，已有文件逐字节保持。
- 使用 O_EXCL 排他创建，不跟随已有路径覆盖文件；未实现强制覆盖。
- go test ./...、go vet ./...、Windows build 全部通过。

## 2026-09-24 / M8 / STATUS-01 与最终回归

- Red：工作区检查失败桩使实际 Git/目录联接测试失败；CLI status 缺少 Vault/环境文件报告而失败。
- Green：实际 DPAPI 校验、损坏 Vault 返回错误、输出无值、未初始化不创建 Vault；嵌套环境文件、模板/缓存排除、目录联接不越界均通过。
- 真实临时 Git 仓库验证 ignored、强制加入索引后的 tracked、去掉忽略规则后的 not ignored；无效 Git 显示 unknown。
- 最终强制回归：go test -json -count=1 ./...，58 项测试/子测试通过。
- 默认跳过 4 项：TestDPAPIAcrossAccounts、TestInteractiveCtrlC、TestInteractiveHiddenInput、TestInstalledGradle。其中交互输入、Ctrl+C、真实 Gradle 已在本次任务单独执行并记录；跨账户仅 write/read-same 基线通过，read-other 未执行。
- 跨账户专用程序已编译到 .cache/dpapi-account.test.exe；第二身份沙箱再次尝试仍因 helper setup refresh 错误无法启动，没有把启动失败记作 DPAPI 拒绝。
- gofmt 检查无差异，go vet ./... 与 Windows 构建通过；实际二进制 help/status 正常。
- 本机系统版本：Windows NT 10.0.26200.0。未声称已测试独立 Windows 10/11、UNC 或远程 CI。
- 所有测试仅使用假值；工具链/缓存/编译结果未加入 Git。实际项目根 status 显示 Vault 尚未初始化，没有创建开发者真实 Vault。

## 2026-09-24 / relations 实际项目集成 / REL-01..08

- 按用户指定，在外部测试项目 relations 的实际工作区执行；先定义 docs/specs/integration-relations.md，再从当前源码构建测试二进制。这轮仅增加验收文档，没有修改产品实现，未声称新增 TDD Red/Green。
- 测试输出保存在被 Git 忽略的隔离测试目录，包含逐步日志、前后文件快照、result.json 和测试驱动；其中所有凭据及 SQLite 内容均为生成的假数据。
- 隔离 APPDATA、Vault、SQLite、TEMP 和 npm 缓存。没有读取真实 .env.local 或真实 Vault，没有调用 dev/start 或外部模型/Tavily 服务；未修改 relations 的 Git 索引。
- REL-01/02：未初始化 status 不创建 Vault；init → import → list → status 通过，实际 DPAPI 解密验证成功。根目录与 src 子目录得到相同项目 ID。8 个假值键包含 Unicode 与多行值；Vault 没有假值明文，CLI 输出不含测试值，导入源保留。
- REL-03：重复 import 返回 1，Vault 字节不变；--overwrite 成功。另导入不同的新 MODEL_API_KEY 并由 Node 对比，确认覆盖更新实际生效。
- REL-04/05：`dvl run -- node.exe <隔离探针>` 从 relations 目录加载该项目实际 configuration、API route、Store、runResearch 和 FixtureProvider。API 创建任务返回 201，state 返回 200，注入配置 ready；研究产生 3 个节点、2 条关系，全部关系引用指向持久化来源。SQLite 关闭/重新打开后数量一致。探针禁止 fetch，外部请求计数为 0；API 响应未返回测试凭据。cwd、父变量覆盖与 Unicode/多行值均校验成功，显式退出 37 被正确透传。
- REL-06：通过 `dvl run -- npm test` 执行实际 npm.cmd 和 tsx 测试，44 项通过、0 失败、0 跳过（Node 测试报告耗时约 1.30 秒）。`dvl run -- npm run typecheck -- --incremental false` 返回 0；关闭增量避免写入项目 tsbuildinfo。
- REL-07：已有 .env.example 返回拒绝覆盖；unset 后测试键消失。初次跨项目探针选用了工具仓库，但隔离 Vault 正位于其缓存目录中，因此收到正确的“Vault 必须位于项目外”拒绝；这是测试目录选择错误，不是产品故障。仅将第二项目替换为隔离 APPDATA 旁边生成的 Git 项目后，init/list/run 均通过，未收到 relations 值，保留父进程哨兵变量。未重复已通过的项目测试。
- REL-08：前后比较 37 个源文件/根文件/数据库文件记录；普通文件校验哈希，.env 与 data 文件仅校验大小/修改时间而不读取内容，全部一致。relations 的完整 Git porcelain 输出前后一致，保留该仓库原有未提交文件。status 正确列出 .env.local 为 ignored，未读取文件内容。
- 本轮结论：Dotenvless 与 relations 的离线 Node/npm/实际应用模块集成通过。没有运行浏览器/Next dev、生产构建、真实 API 或真实凭据迁移，不据此声称这些场景通过。
- 已知待修复：前次全量审查复现的多行引号值首行尾空白丢失，以及方括号路径 Git 状态误报仍未修复；本轮正常多行样本不包含首行尾空白。

## 2026-09-24 / status 可选目录 / STATUS-DIR-01..04

- SDD：先在 docs/specs/m8-status.md 定义 `status [DIRECTORY]`、项目根选择、错误码和只读要求；保持既有项目身份与配置范围。
- Red：在未修改实现前执行 `go test -count=1 -run 'TestStatusDirectory|TestStatusHelp' -v ./internal/cli`，3 个新增测试函数失败。`.`、绝对/相对/Unicode 空格/子目录/非 Git 调用位置均被旧解析器拒绝为 exit 2；错误目标返回 2 而非期望 1；help 缺少目录语法。无参数原有行为通过。
- Green：为 status 添加单个可选目录参数，复用 project.Discover 选择目标，不调用 chdir。上述测试全部通过；在共享的隔离 Vault 中验证当前/目标项目键名和文件隔离、查询前后 cwd/Vault 不变、值不出现在输出中。
- 无效目录、文件、非 Git 目录返回 1；空参数、选项形式、多余参数返回 2，不回显原始参数，不创建 Vault。
- 全量 `go test -json -count=1 ./...`：73 项测试/子测试通过，4 项既有交互/跨账户/Gradle 测试跳过。原始事件保存在忽略文件 `.cache/status-directory-tests.jsonl`。gofmt 无差异，go vet ./... 和 Windows build 通过。
- 实际二进制：继续使用上一轮 relations 集成测试生成的假 Vault，从工具仓库通过绝对路径查询目标项目、通过相对路径查询其 src 子目录，再在目标项目中执行无参数 status；三份完整输出相同，目标项目 DPAPI 验证成功，Vault SHA-256 与 relations Git 状态不变。未访问真实 Vault 或真实 .env 内容。
- help、README 与设计文档已同步。没有修改其他命令的目录选择；此前审查的两个独立缺陷仍未修复。本轮未重做交互、跨账户或真实外部 API 验收。

## 2026-09-24 / GitHub 提交准备 / GH-01..06

- 用户选择暂不添加许可证。先在 docs/specs/github-preparation.md 定义验收；本轮不创建远程仓库、不推送、不发布 Release。
- Red：新增多行引号值首行空白回归测试，单/双引号、export、CRLF 四例全部失败；Git 字面量路径测试把未跟踪/已忽略的 .env.[dev] 误报为 tracked。
- Green：解析器只清理行首语法空白，保留引号内行尾空白。Git ls-files 使用 --literal-pathspecs；真实回归进一步发现 check-ignore 默认索引匹配同样受方括号影响，将同一 literal 选项加到 check-ignore 会被 Git 拒绝。最终在已经精确确认未跟踪后使用 check-ignore --no-index，三种 Git 状态均通过。参考 [Git check-ignore](https://git-scm.com/docs/git-check-ignore)。
- 上述两项已修复，替代此前记录中的“仍未修复”状态；未加引号值的既有空白清理行为保持。
- README 改为通用安装/构建/使用说明，补充 CONTRIBUTING、SECURITY、CHANGELOG、首次推送指南、Issue/PR 模板和 editorconfig。历史验收记录保留，不改写旧提交；新用户无需本机路径和缓存。
- Windows CI 使用官方 Actions 当前 v7 的固定提交 SHA（通过官方仓库 git ls-remote 核对），Go 版本从 go.mod 获取；包含依赖下载/校验、格式、vet、测试、构建和 help/version 冒烟，产物上传仅包含 bin/dvl.exe。参考 [setup-go](https://github.com/actions/setup-go)、[checkout](https://github.com/actions/checkout)、[upload-artifact](https://github.com/actions/upload-artifact)。当前没有远程 CI 运行结果。
- 完整本地回归：Windows x64、CGO_ENABLED=0，79 项测试/子测试通过，4 项既有可选验收跳过；go mod verify、go vet、gofmt、Windows build 与二进制 help/version 通过。本轮未重做交互/第二账户/独立系统验收。
- 检查当时 61 个历史文件路径、112 个可达历史 blob：没有工具链、构建/缓存产物、Vault 或真实环境文件路径；常见 GitHub/AWS/OpenAI token 及私钥格式匹配为 0。该模式检查不是不存在任何秘密的证明。候选文件的 YAML/Markdown 本地链接与忽略规则检查通过。脚本与汇总仅保存在忽略的 .cache 中。

### GH-07：干净克隆与默认分支

- 对提交 ab3cd34 执行本地 `git clone --no-hardlinks`，副本仅包含 Git 版本文件，没有原工作区 .tools、测试 Vault 或其他缓存。使用外部 Go 1.27.1 可执行文件和已校验的模块缓存，GOPROXY=off、CGO_ENABLED=0，构建 Windows CLI 成功，help/version 返回 0，构建后副本 Git 工作区干净。这验证源码完整性，不冒充全新机器的联网依赖安装或 GitHub CI。
- 初次本地 clone 因源仓库 .git 属于另一 Windows 身份而失败；仅为当前进程增加源仓库及其 .git 的精确 safe.directory 后成功，未修改全局信任或文件权限。
- 默认分支 main 使用 fast-forward 合并已通过验证的开发历史，保留全部原提交及 codex/v0.1 分支；不重写历史、不创建远程、不推送。暂不添加许可证。

### GH-08：文档路径清理

- README 目录示例改为相对路径；跨账户测试从仓库根目录生成假值密文，复制到第二账户后在文件所在目录解析路径。没有执行跨账户测试或访问真实 Vault。
- 验收记录中的本机绝对路径、具体账户名和专属缓存目录改为通用描述，保留原有 Red/Green 结果、测试数量与未验证项；当前文本清理不改写旧提交。
- 扫描 24 个受版本管理的文档与 GitHub 模板：未发现绝对 Windows 路径、用户主目录路径、UNC 路径、已知本机身份或专属缓存目录。20 个相对 Markdown 链接均有效；README 与手动验收文档中的 7 段 PowerShell 示例语法检查通过；git diff --check 通过。
- 本轮只修改文档，没有产品行为变更，未新增 TDD 测试，也未重复 Go 测试、vet 或构建。

### GH-09：公开仓库与历史清理

- 用户明确要求创建公开仓库，并选择先清理旧提交中的本机信息再推送。已创建公开空仓库 Nikoace/Dotenvless，origin 指向该仓库。
- 原始历史先写入本地忽略目录中的 Git bundle，bundle verify 通过；备份不加入 Git，也不上传。
- 使用 git-filter-repo 2.47.0 对两条本地分支进行精确文本替换，保留原有 15 个提交。逐提交比较非 Markdown 文件的 Git 对象 ID：全部一致；清理前后最新完整文件树一致。旧提交 SHA 已改变，历史测试结果仍对应原执行事实。
- 扫描清理后 136 个可达 blob，其中 63 个为文档或配置版本：未发现本机路径、历史账户说明、专属缓存目录或运行时产物，常见凭据模式匹配为 0。初次扫描命中的单元测试目录名为临时生成的通用占位目录，核实后保留测试代码；模式扫描不构成不存在任何秘密的证明。
- 本轮没有产品行为变化，没有重做 TDD、Go 测试或构建；公开推送后的 Windows CI 以仓库 Actions 的实际结果为准。

### GH-10：首次 CI 路径别名修复

- 公开 main 的首次 [Windows CI](https://github.com/Nikoace/Dotenvless/actions/runs/35979927750) 在配置测试中报 unexpected vault path；格式、依赖校验和 vet 已通过。失败来自测试直接比较路径文本，而 VaultPath 返回规范化物理路径。
- SDD：先增加 GH-10，要求配置路径与子进程工作目录按物理目录身份验收，不改变产品实现。
- Red：将 TEMP/TMP 指向隔离目录的大小写别名，运行 `go test -count=1 -run '^TestVaultPathOutsideProjectWithoutWriting$' -v ./internal/config`，复现同一断言失败。
- 首次扩大全量复现时，临时目录设在仓库内部，使非 Git 测试继承了仓库祖先；这是测试环境选择错误。改为系统临时区内的独立目录后，单独运行 `go test -count=1 -run '^TestBatchArgumentsAndGradleWrapperResolution$' -v ./internal/runner`，仍复现测试辅助进程退出 92，定位为 cwd 字符串比较同样不接受路径别名。
- Green：两个测试改用 os.Stat/os.SameFile 验证目录身份，保留固定文件名、无写入、环境变量、参数、流及退出码断言。相同别名条件下，配置和批处理针对性测试均通过。
- 完整回归：仓库外的隔离路径别名环境下，`go test -count=1 -json ./...` 得到 79 项通过、4 项既有可选验收跳过、0 失败；gofmt、go vet、Windows 构建和二进制版本冒烟通过。此次修改只涉及测试与文档，产品实现保持不变。
