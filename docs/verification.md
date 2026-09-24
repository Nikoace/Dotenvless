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
