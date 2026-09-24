# M8：状态检查

status 输出项目身份、当前项目键名、Vault 状态和工作区环境文件名；不输出值、不自动修改项目文件。

Vault：尚未初始化明确显示；存在时实际解密当前项目记录验证 DPAPI，失败返回错误，不显示已验证加密。零键项目显示已初始化/无 Secret。状态读取不创建缺失 Vault；已有 Vault 使用与 CRUD 相同的进程锁。

工作区：递归检查文件名 .env / .env.*，排除 .env.example 与 .env.*.example。不读取环境文件内容，不声称已发现真实凭据。排除 .git、node_modules、.venv、venv、vendor、build、dist、target、.gradle、.cache、.tools、__pycache__ 目录；不跟随目录链接。输出始终声明范围；无匹配只表示检查范围内未发现这些文件名。

Git：检查根 .env 的忽略状态，并对检测文件区分 tracked、ignored、not ignored、unknown。已跟踪的文件即使匹配 .gitignore 也显示 tracked。使用只读 Git 命令，禁用 fsmonitor 和可选锁；Git 不可用/仓库无效/超时显示 unknown，不据此声称安全。

Vault 或扫描失败返回 1。发现文件/未忽略属于提示，status 返回 0；它是信息命令，不是 CI Secret 扫描器。V0.2 才讨论内容扫描。

验收：嵌套文件、排除模板与缓存、真实目录联接不越界、真实 Git 的 tracked/ignored/not ignored、未初始化不创建 Vault、损坏/解密失败不显示成功、输出没有值。
