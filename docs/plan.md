# SDD / TDD 推进计划

## 每个工作切片

1. **Spec**：在 `docs/specs` 写行为、边界、错误路径与验收 ID；设计歧义记录在 `docs/design.md` 并先讨论。
2. **Red**：先写可运行的行为测试，执行并确认因功能缺失而失败；编译失败与行为失败分别记录。
3. **Green**：实现满足测试的最小代码，不顺手提前实现后续功能。
4. **Refactor**：在通过状态下整理实现，运行相关回归。
5. **Verify**：格式检查、`go vet ./...`、`go test ./...`、构建；必要时真实 Windows 进程/控制台验收。
6. **Commit**：审查 diff、确认未包含 Secret/运行时数据、更新 README 与验证记录，提交可编译的切片。

测试和最小实现可一起进入 Green 提交，Red 的命令及结果保留在验证记录；不需要把失败测试提交到稳定分支。版本历史使用 `docs:`、`feat:`、`fix:`、`test:` 前缀，提交粒度按规格或里程碑。不会未经要求推送远程仓库。

## 里程碑

| 里程碑 | Spec / 首批失败测试 | 最小交付与验收 | Git 切片 |
| --- | --- | --- | --- |
| S0 | 需求、边界、设计分歧 | 规格、决策草案、测试计划 | `docs: define v0.1 specification and delivery plan` |
| M0 | CLI-01 帮助/版本、无效参数 | Go 项目、CLI 骨架、Windows CI、构建成功 | `feat: bootstrap Windows CLI with help and version` |
| M1 | PROJ-01..05 Git 根/子目录/非 Git/同名不同路径 | 项目身份、基础 status，README 说明移动路径语义 | `feat: resolve Git project identity` |
| M2 | SEC-07..09 新建/覆盖/删除/损坏/并发更新 | Vault 存储层，测试使用内存/假数据边界，生产不写明文 | `feat: add transactional vault storage` |
| M3 | SEC-01/04 加密往返/损坏/失败/跨用户拒绝 | 真实 DPAPI 后端；此后才开放生产 Secret 存储 | `feat: protect vault records with Windows DPAPI` |
| M4 | INIT-01、CRUD-01/02、SEC-02/03 | init、隐藏 set、list、unset，失败不覆盖 | `feat: manage project secrets without exposing values` |
| M5 | RUN-01、SEC-01/05 参数/环境/退出码/脚本 | run 与真实工具链验证，完成核心使用路径 | `feat: inject secrets into child process environments` |
| M6 | IMPORT-01 先固定语法/冲突策略 | 原子导入，原文件保留，错误不泄露行内容 | `feat: import dotenv files safely` |
| M7 | EXAMPLE-01 空值模板/已有文件 | 只生成键名，不破坏已存在文件 | `feat: generate secret-free environment examples` |
| M8 | STATUS-01 明文文件/Git 忽略/状态失败 | 如实显示检查范围与安全状态 | `feat: report vault and workspace status` |

M1 提供身份识别，M2/M3 完成后注册与加密能力就绪，M4 组合为完整 `init`。M0/M1 不放置可写明文 Secret 的临时实现。

## 测试层次

### 单元测试

- Project：临时 Git 根、子目录、`.git` 文件、非 Git、同名不同路径、路径规范化。
- Vault：新建、CRUD、隔离、未知版本、截断文件、非法结构、读写失败、锁竞争、原子替换失败。
- DPAPI：随机假值、空值、Unicode、篡改数据、解密错误；测试替身仅用于触发错误路径。
- CLI：语法和返回码；任何错误均不包含假 Secret、原始 Secret 输入或导入失败行。
- Runner：环境合并、大小写覆盖、父环境不变、参数列表；启动策略和 shell 特殊字符。

### Windows 集成测试

- 真正启动 helper 子进程；由子进程校验假 Secret 并返回状态，成功时不打印 Secret。
- 测试每次使用临时存储目录，不接触 `%APPDATA%\dotenvless` 的真实数据。
- 多个独立进程更新同一测试 Vault，验证无丢失和退出后锁恢复。
- 子进程返回 0、非零、找不到命令；项目目录前后文件清单一致。
- PowerShell、CMD、Python、Node、npm `.cmd`、Gradle `.bat` 分别记录证据。
- 真正安装并执行 Gradle wrapper 的验证与仅测试 `.bat` helper 分开记录；不使用外部 Secret。

### 人工/独立账户验收

- 真正交互式 `set` 不回显，取消后终端模式恢复。
- Ctrl+C 的目标及后代退出行为。
- 账户 A 加密，账户 B 拿到密文文件后解密失败；只允许该测试使用临时测试账户/临时假值。

涉及创建 Windows 账户/系统设置的动作不作为普通单元测试自动执行。缺少独立账户条件时保留为未验证，不能用 mock 结果标为通过。

## CI 计划

Windows runner、固定 Go 版本、格式检查、vet、测试、构建。支持时运行 race 检查；如额外编译器缺失要单独记录，不用跳过掩盖测试失败。CI 中交互/第二账户验收与无人值守自动测试区分。工作流的首次远程结果需实际触发后才能声称通过。
