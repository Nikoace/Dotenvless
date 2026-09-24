# relations 项目集成验收

范围：在外部测试项目 relations 的实际工作区执行 Dotenvless 0.1.0-dev；不迁移真实凭据，不运行自动读取 .env.local 的 dev/start，不进行外部 API 调用。

先定义以下验收，再执行现有产品；这轮不修改产品实现，不将集成测试冒充新增实现的 TDD Red/Green。

| ID | 验收 |
| --- | --- |
| REL-01 | 使用独立 APPDATA 和假值，初始化前 status 不创建 Vault；根目录/现有 src 子目录身份一致。 |
| REL-02 | init/import/list/status 正常；导入源保留；生成的 Vault 不含测试值明文；输出不含测试值。 |
| REL-03 | 同名键重复导入失败且 Vault 字节不变；显式 --overwrite 可以更新。 |
| REL-04 | dvl run 启动真实 Node 和 npm.cmd；继承 cwd、覆盖父进程同名变量、保留 Unicode/多行值与退出码。 |
| REL-05 | 通过注入配置直接调用 relations 的配置检查、API 和 FixtureProvider 研究流程，隔离 SQLite 持久化 3 个节点/2 条关系，API 不返回凭据。 |
| REL-06 | dvl run -- npm test 与 npm run typecheck -- --incremental false 通过。 |
| REL-07 | 已有 .env.example 拒绝覆盖；unset 删除测试键；其他项目不获得 relations 的 Vault 值。 |
| REL-08 | relations Git 状态、源文件哈希及已有 .env/数据库文件大小和修改时间前后一致；不读取真实 .env 或数据库内容，不操作其索引。 |

结果记录在 docs/verification.md。此前全量审查发现的多行引号首行尾空白丢失、方括号路径 Git 状态误报仍是独立缺陷，本轮测试不隐含修复。
