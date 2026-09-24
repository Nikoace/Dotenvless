# M4：初始化与隐藏输入 CRUD

- Vault 默认路径为 APPDATA/dotenvless/vault.dat。APPDATA 必须是绝对路径；解析现存祖先的物理路径后，拒绝位于当前项目根内的 Vault。非普通 Vault 文件拒绝使用。
- init 注册项目且保留现有键。list/unset/set 未初始化时提示先 init。
- set KEY 只接受隐藏终端输入，不接受明文参数，也不自动读取重定向 stdin。成功输出规范化键名；读取失败/取消/校验/加密失败不改旧值。
- 键名规则沿用 M2；空值允许。Secret 输入限制为无 NUL 的有效 UTF-8 文本、至多 32 KiB。
- 使用 Go 官方 x/term 的终端模式管理；Ctrl+C 取消路径必须恢复输入模式。终端是否可隐藏检查失败时不读取内容。
- list 仅输出排序键名，unset 输出删除键名；不提供 get。
- status 本阶段仍只展示身份，Vault 与工作区完整状态到 M8 补齐。

验收：真实加密的 init/set/list/unset、幂等/覆盖、空值、拒绝非终端、参数不回显、读取失败保留旧值、项目内 AppData 拒绝、无项目明文文件；另用交互终端验证不回显和取消恢复。
