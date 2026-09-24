# AI Skill：Dotenvless 调用约定

范围：为现有 0.1.0-dev CLI 提供可复制的 Agent Skill。只增加调用指导、客户端接入说明与验证记录，不改变命令、Vault 格式或产品安全边界。

## 行为与验收

| ID | 验收 |
| --- | --- |
| SKILL-01 | `skills/dotenvless/SKILL.md` 包含有效的 `name`、`description`；随目录复制的参考文件可独立读取，不依赖 Dotenvless 源码目录。附 Codex UI 元数据和客户端安装/调用说明。 |
| SKILL-02 | 仅使用现有命令；先检查版本、帮助和目标项目。明确只有 `status` 接受目录参数，其他操作使用目标项目的工作目录；不能把 `status` 返回 0 当作已初始化或无凭据风险。 |
| SKILL-03 | 说明 Windows/DPAPI 与隐藏交互终端要求；真实值不进入 AI 对话、参数、工具输入或日志。无本机终端时交给用户执行 `set KEY`，不以管道、假值或明文文件绕过。 |
| SKILL-04 | 迁移直接调用 `import`，AI 不预读源文件；同名冲突不自动加 `--overwrite`，成功后保留源文件。覆盖、删除键和删除源文件须属于已授权范围；拒绝覆盖已有 `.env.example` 时保留原文件。 |
| SKILL-05 | `run` 仅执行已授权且已检查的目标命令；说明会注入该项目全部键、继承环境和原样传递子进程输出。禁止环境转储探针；不声称 skill 是沙箱或日志脱敏器。 |
| SKILL-06 | 排错覆盖未初始化、缺键、非 Windows、无终端、worktree/移动路径、导入冲突、批处理参数和 DPAPI 失败；结果只汇报键名、状态、退出码与未完成事项。 |

## 验证方式

- 对照 `internal/cli`、`internal/runner` 和现有规格核对命令及输出语义。
- 校验 skill frontmatter、UI 元数据、相对链接和 diff；以不含真实值的任务场景检查调用选择。
- 按 CONTRIBUTING.md 的纯文档约定，不新增文案匹配测试，不虚构 TDD Red/Green；本轮不改变 Go 实现。
- 无 Windows 终端或客户端实际加载条件时，明确记录未验证项；不访问真实 Vault 或 `.env` 内容。
