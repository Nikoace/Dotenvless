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
