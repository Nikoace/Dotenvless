# 给 AI 使用的 Dotenvless Skill

[dotenvless skill](../skills/dotenvless/SKILL.md) 指导 AI 通过现有 `dvl` CLI 检查配置、迁移 `.env`、生成键名模板和启动应用。它是一组 Agent Skills 指令，不新增 CLI 命令、MCP 服务或自动读取 Secret 的脚本。

## 前提

- 在 Windows 10/11 上按 [README](../README.md#构建) 构建并配置可信的 `dvl.exe`；skill 本身不安装 CLI。
- 实际命令须在目标 Windows 身份和应用 Git 项目下执行。云端/Linux 的 AI 可以给出操作步骤，但不能因此获得本机 DPAPI Vault。
- 真实值由你在本机交互终端的 `dvl set KEY` 隐藏提示中输入；不要发到对话、工具参数或复制到云端。已有 `.env` 可由本机 `dvl import` 直接处理，AI 不需要先读文件。

## 安装到客户端

完整复制 `skills/dotenvless` 目录，保留 `SKILL.md`、`references/cli-contract.md` 和 `agents/openai.yaml` 的相对位置。仓库中的 `skills/` 是分发源目录，不会自动成为所有客户端的发现目录。

| 客户端 | 当前项目安装位置 | 当前用户安装位置 | 显式调用 |
| --- | --- | --- | --- |
| Codex CLI / IDE | `<应用项目>/.agents/skills/dotenvless/` | `~/.agents/skills/dotenvless/` | `$dotenvless` |
| Claude Code | `<应用项目>/.claude/skills/dotenvless/` | `~/.claude/skills/dotenvless/` | `/dotenvless` |

位置与调用方式依据 [OpenAI 官方 skill 文档](https://developers.openai.com/codex/skills) 和 [Claude Code 官方 skill 文档](https://code.claude.com/docs/en/skills)，核对日期为 2026-09-24。`~` 指客户端所在账户的主目录；Windows 原生客户端通常对应 `$env:USERPROFILE`。客户端仍须遵守本身的文件、终端和命令授权机制。

例如，在 Dotenvless 仓库根目录的 PowerShell 中执行下面的**用户级 Codex 安装**。目标存在时主动停止，避免覆盖已有定制：

```powershell
$source = (Resolve-Path -LiteralPath '.\skills\dotenvless').Path
$skillParent = Join-Path $env:USERPROFILE '.agents\skills'
$skillTarget = Join-Path $skillParent 'dotenvless'
if (Test-Path -LiteralPath $skillTarget) {
    throw 'dotenvless skill already exists; review it before updating.'
}
New-Item -ItemType Directory -Path $skillParent -Force -ErrorAction Stop | Out-Null
Copy-Item -LiteralPath $source -Destination $skillTarget -Recurse -ErrorAction Stop
Test-Path -LiteralPath (Join-Path $skillTarget 'SKILL.md')
```

Claude Code 使用同一段命令，将 `.agents\skills` 改成 `.claude\skills`。项目级安装则将 `$skillParent` 设为目标应用项目中的对应目录；不要把 Dotenvless 工具仓库误当成应用项目。

在客户端的 skill 列表中确认出现 `dotenvless`。其他支持 Agent Skills 的客户端可使用同一目录，具体安装入口以该客户端文档为准。本仓库不默认安装到你的个人 skill 目录，也没有宣称完成所有客户端的加载验收。

## 调用示例

Codex CLI / IDE：

```text
$dotenvless 检查当前项目的密钥配置，只报告已有和缺少的变量名。
$dotenvless 把当前项目的 .env 导入 Vault；保留源文件，已有键冲突时先告诉我。
$dotenvless 用已保存的密钥运行 npm run dev，先检查启动脚本是否会输出密钥。
```

Claude Code 可将开头换成 `/dotenvless`。使用自然语言提到 Dotenvless 和相应操作，也可以由支持自动匹配的客户端选择 skill。

首次配置时，AI 会确定项目并执行已授权的初始化，再给你本机录入命令，例如：

```powershell
dvl set OPENAI_API_KEY
```

你在隐藏提示中输入真实值后，AI 用 `list/status` 确认键名，再执行已经检查过的启动命令。没有安全的本机交互入口时，录入步骤保持待完成；不会用假值、管道或临时文件代替。

## 调用边界

- `status` 返回 0 也可能表示尚未初始化，或发现了已跟踪的 `.env`；skill 会解读字段，不把退出码当成安全检查通过。
- `status DIRECTORY` 只查询，不切换后续命令的项目。worktree、移动或重新克隆后的路径具有独立身份；同目录切换分支不改变身份。
- 导入冲突时不会自行加 `--overwrite`；不会默认删除源 `.env`、覆盖已有 `.env.example` 或删除其他键。已经明确授权的操作不重复询问。
- `run` 会传入该项目全部 Secret，子进程日志原样返回。skill 会避开环境转储等调用，但不构成沙箱、权限隔离或输出脱敏保证。
- 保留的 `.env` 仍可能被应用的 dotenv loader 读取；导入成功和启动成功不单独证明已经摆脱文件依赖。清理源文件前须确认应用配置行为。

## 维护

修改 CLI 后同步 [skill](../skills/dotenvless/SKILL.md) 和[命令参考](../skills/dotenvless/references/cli-contract.md)。包内参考保持自包含，用户复制 skill 后不需要保留整个源码仓库。验收约定见[规格](specs/ai-skill.md)，本轮实际检查及未验证项见[验证记录](verification.md)。
