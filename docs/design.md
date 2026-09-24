# 设计决策

状态：M0–M8 实现基线。规格、实现与执行证据分别保存在 docs/specs、源代码与 docs/verification.md；未通过的独立环境验收单独列出。

## 用户已确认的选择（2026-09-24）

| ID | 决策 | 结果 |
| --- | --- | --- |
| D-01 | 在本项目目录建立独立 Git 仓库 | 只管理本项目；未操作父仓库的索引或其他项目修改 |
| D-02 | 规范化 Git 根路径的 SHA-256；非 Git 拒绝；worktree 隔离 | 子目录/联接共享身份；移动/重命名需重新导入 |
| D-03 | .cmd/.bat 自动通过 cmd.exe；普通程序直接启动 | 支持 npm 与 Gradle 常见用法 |
| D-04 | 批处理不可靠的参数明确拒绝 | 限定字符与引号规则已写入 M5 规格并通过真实进程测试 |
| D-05 | 导入同名键默认整次失败，显式 --overwrite 才覆盖 | 解析、冲突、加密、提交均采用整批事务 |

## 来自原始需求的约束

- Windows 10/11 优先、Go、单个 CLI，无运行时网络访问和 daemon。
- 直接使用 Windows DPAPI Current User 保护每个值；不增加应用主密钥或自制密码算法。
- 隐藏输入；不接受明文位置参数；没有 get；不生成临时明文 .env，不写持久环境。
- M0–M8 依次推进；不实现 V0.2 内容扫描、云服务、Agent sandbox 或其他平台产品支持。

## 已落地的实现决策

- 本地模块名 dotenvless；开发工具链 Go 1.27.1；只依赖固定版本的 Go 官方 x/sys、x/term。远程仓库与发布许可证未指定。
- Windows 文件句柄解析真实目录，再规范化路径并哈希；保留实际目录名大小写。测试发现仅 EvalSymlinks 不足后作出该修正。
- Vault 默认 APPDATA/dotenvless/vault.dat；没有实际配置需求，未生成空 config.json。拒绝当前项目内的存储路径。
- Vault 版本化 JSON、DPAPI 密文、上下文绑定；Windows OS 锁覆盖读—改—写，密文临时文件同步后原子替换。
- 键名为 ASCII 环境变量名，规范为大写；空值允许，拒绝无效 UTF-8/NUL 和超过 32 KiB 的值。
- 终端只提供隐藏交互输入；处理 Unicode 退格、Ctrl+C、长度上限，并恢复模式/清理未消费输入。重定向 stdin 明确拒绝。
- dotenv 使用明确的语法子集，不插值或执行命令；导入后保留原文件。
- example 排他创建，不覆盖原文件。status [DIRECTORY] 选择当前或指定目录所属的 Git 项目，不切换调用者目录；只在声明的范围检查文件名，Git 失败时显示 unknown。

## SDD、TDD 与 Git

SDD 指 Specification-Driven Development：先写验收 ID、边界和错误行为，再实现。TDD 使用实际执行的 Red → Green → Refactor；测试与最小实现一起进入可编译提交，失败证据保留在验证记录。每个里程碑更新 README、运行相应检查并单独提交。

## 安全边界

DPAPI 默认通常关联同用户凭据和机器；本项目不使用 CRYPTPROTECT_LOCAL_MACHINE。见 [Microsoft CryptProtectData](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata)。

同 Windows 身份的程序仍可能解密；目标进程及其后代可以读取、输出或传递已注入的值。V0.1 不隔离管理员、同身份恶意程序、系统分页/转储，也不证明没有其他类型的 Secret 文件。APPDATA 的系统漫游/重定向设置不由本工具改变。

## 环境限制与待验收

- .git 由沙箱身份创建；本机使用进程级 safe.directory 完成 Git 操作，没有更改全局信任设置。所有者调整被 OS 拒绝；普通沙箱启动仍报 setup refresh 错误。
- 跨账户验收工具已准备，写入与同账户读取已验证；不同 SID 读取后的 DPAPI 拒绝仍未运行。
- 独立 Windows 10/11、UNC/特殊文件系统和远程 CI 尚未完成现场验证。
- 不因这些待办而声称已经完成全部发布验收；当前仍为 0.1.0-dev。
