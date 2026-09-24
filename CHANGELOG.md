# 变更日志

## Unreleased — 0.1.0-dev

### 功能

- Windows DPAPI Current User 加密 Vault，项目隔离、跨进程锁与原子写入。
- `init`、隐藏输入 `set`、`list`、`unset` 和子进程环境注入 `run`。
- `.env` 原子导入与显式 `--overwrite`、只含键名的 `.env.example` 生成。
- `status [DIRECTORY]`：当前/指定 Git 项目的键名、Vault 验证和环境文件 Git 状态。
- Windows `.cmd/.bat` 与 Gradle wrapper 的受限安全参数传递。

### 修复

- 保留多行引号值首行末尾的空白，包含 `export` 与 CRLF 场景。
- 含方括号的环境文件路径按字面量检查跟踪状态，忽略判定不再受其他索引文件影响。

### 工程

- 新增可复制的 `dotenvless` Agent Skill、命令参考、Codex/Claude Code 接入说明与使用边界。
- SDD/TDD 规格与验证记录、Windows CI、贡献和问题报告模板。
- 本机真实 DPAPI、控制台、Node/Python/npm/Gradle 及离线应用集成证据；跨账户拒绝、独立 Windows 系统与远程 CI 仍需分别验收。
