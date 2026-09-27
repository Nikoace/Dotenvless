# 变更日志

## 0.2.0 — 本地工程 ENV 管理中心

- 新增 Windows Fyne 中文客户端；支持工程发现、普通目录登记、环境编辑、共享库、导入预览及状态接入。
- 基础配置 + 环境覆盖 + 显式共享引用；停用键从子进程环境移除，空字符串保留。
- CLI 新增全局 `--env` / `--base` 和显式 `migrate`，默认始终为 local。
- Vault v2 密文备份升级、不可变配置层 ID、上下文绑定、GUI 乐观版本检查；旧 Vault 在升级前仍可只读使用。
- 单条 15 秒查看、失焦隐藏、Windows 私密剪贴板及所有权检查；默认元数据列表不解密变量。
- 独立 GUI C 编译器构建、CLI 纯 Go 构建、双模块 CI、Windows 发布包及第三方许可证。
- 新增真实 CLI/GUI 子进程、并发修改、迁移进程中断恢复和无明文产物验证。未执行的独立机器/人工验收保留在验证记录中。


## 0.1.0-dev — 历史基线

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

- SDD/TDD 规格与验证记录、Windows CI、贡献和问题报告模板。
- 本机真实 DPAPI、控制台、Node/Python/npm/Gradle 及离线应用集成证据；跨账户拒绝、独立 Windows 系统与远程 CI 仍需分别验收。
