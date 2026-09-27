# Dotenvless

Windows 上的本地工程 ENV 管理中心。用中文桌面客户端管理配置，通过 CLI 把所选环境注入子进程。值由当前 Windows 用户的 DPAPI 加密，集中保存在工程目录之外。

当前版本：**0.2.0**，Windows x64。支持 Git 工程、独立 worktree 和普通目录；无运行时网络、遥测、常驻服务或应用主密码。关闭窗口即退出。实际验证范围与未执行的人工验收见[验证记录](docs/verification.md)。

## 开始使用

解压 Windows 发布包，运行 `dvl-desktop.exe`。使用 CLI 时可直接调用 `dvl.exe` 的绝对路径，或自行将其所在目录加入 PATH。

1. 在客户端点击「添加目录」，或「批量发现」后勾选 Git 工程。新工程包含基础配置和 `local`。
2. 在基础配置中填写共用变量，在 `local` 或新建的 `dev/test` 环境中填写覆盖值。保存空输入表示空字符串。
3. 在工程目录或其子目录中启动：

~~~powershell
dvl run -- npm run dev
dvl --env test run -- go test ./...
~~~

不指定环境始终使用 `local`；客户端正在浏览的环境不影响终端。普通目录需先明确登记，子目录归属最近的已登记根；嵌套 Git 根独立隔离。工程移动或重命名后视为新工程，旧配置保留为目录不可用记录。

## 客户端

- 左侧按工程名称或路径搜索，显示环境数量、配置状态和目录可用性。批量发现可取消，跳过目录链接、依赖和构建目录，不读取 dotenv 正文。
- 中间查看有效变量的来源、继承、共享引用和停用状态；右侧新增、替换、删除本层绑定或在当前环境停用。环境可创建、复制、重命名、删除，基础配置与 `local` 保留。
- 共享库通过稳定 ID 显式引用，支持不同工程使用不同变量名。更新前显示引用范围，下次启动生效；被引用条目不能直接删除。「保存独立值」只修改当前工程层。
- 导入先预览键名和冲突。冲突默认整批拒绝，勾选覆盖后才允许更新；取消不写入，原文件保留。
- 变量默认遮蔽；单条查看 15 秒后隐藏，切换工程、配置层、页面或失去焦点时隐藏。复制使用 Windows 后端排除剪贴板历史/云同步，30 秒后或正常退出时仅清理仍属于本次复制的内容。
- 配置被其他 CLI/客户端修改后，旧表单提交会要求刷新。编辑器中的值在保存、刷新或切换工程后清空。
- 「状态与接入」检查工作区并复制不含值的启动命令；可生成仅含键名的 `.env.example`，不会覆盖已有文件。

删除工程记录只删除 Vault 中的该工程配置，不删除工程文件或共享条目。客户端不提供启动器、明文导出、云同步或跨机器密钥迁移。

## CLI

~~~powershell
dvl init
dvl set API_KEY
dvl --base set DATABASE_URL
dvl --env dev set LOG_LEVEL
dvl --env dev list
dvl --env dev unset LOG_LEVEL
dvl --env dev import .env
dvl --env dev import --overwrite .env
dvl --env dev status
dvl --env dev status ./another-project
dvl --env dev example
dvl --env dev run -- npm run dev
dvl migrate
~~~

`set` 从交互终端隐藏读取值，不能把值作为命令参数提供。没有明文 `get`。

`--base` 与 `--env` 互斥，放在子命令之前；基础配置不能直接用于 `run`。`list/status/example` 使用所选层或环境的有效键名；`set/import` 写入所选层。`unset` 只删除本层绑定，删除覆盖后恢复基础继承。停用由客户端操作，启动时连父终端的同名变量也会移除。

最终子进程环境：继承父终端 → 基础配置 → 环境覆盖 → 删除停用键。先确定最终绑定，再解密所需值；引用缺失或解密失败时不启动。父终端和已运行进程的环境不变。

导入支持 UTF-8/BOM、LF/CRLF、注释、`export`、单/双引号及多行值；不展开变量、不执行命令，文件上限 1 MiB，单值上限 32 KiB。[导入语法](docs/specs/m6-import.md)

普通可执行文件直接启动；`.cmd/.bat` 通过系统 `cmd.exe` 执行，`./gradlew` 优先使用相邻 `gradlew.bat`。无法可靠传递的批处理特殊参数会被拒绝。[参数契约](docs/specs/m5-run.md)

## Vault 升级与安全范围

默认位置：`%APPDATA%\dotenvless\vault.dat`，必须位于所有已登记工程目录之外。普通配置和密钥统一加密；根路径、键名、环境名等是元数据。跨进程锁、版本检查和密文原子替换保护并发修改。

v1 Vault 可以由新 CLI 只读使用，修改前执行 `dvl migrate` 或客户端升级入口。升级先保留原始密文 `vault.dat.v1.bak`，再在内存中重新加密并一次提交；失败保留原文件。旧变量迁入各工程 `local`，基础为空。旧记录没有工程路径，添加原目录后按路径哈希关联。升级后旧 CLI 会拒绝 v2，不能继续修改。

备份仍绑定原 Windows 用户/DPAPI 条件，不是跨机器可移植备份。不要手动用旧备份覆盖已经更新的 v2 Vault。备份与当前原文件不一致时，迁移拒绝覆盖备份。

项目身份为规范化物理根路径的 SHA-256，worktree 隔离。密文上下文绑定工程、配置层及键名，或共享条目 ID。DPAPI 不防御同一 Windows 身份下的恶意程序、管理员、内存转储或目标子进程主动泄露。剪贴板排除标记也不能阻止其他程序主动读取当前剪贴板；异常终止可能来不及执行定时清理。

`status` 仅检查声明范围内的 dotenv 文件名和 Git 状态，不读取正文，也不证明工作区没有其他凭据。导入不会清理 Git 历史或撤销曾泄露的凭据。[检查范围](docs/specs/m8-status.md) · [安全说明](SECURITY.md)

## 构建与验证

需要满足 `go.mod` 的 Go。CLI 保持纯 Go：

~~~powershell
$env:CGO_ENABLED='0'
go vet ./...
go test -count=1 ./...
go build -trimpath -o bin/dvl.exe ./cmd/dvl
~~~

桌面是独立模块，Fyne 需要 C 编译器：

~~~powershell
.\scripts\setup-desktop.ps1
.\scripts\desktop.ps1 vet -tags ci ./...
.\scripts\desktop.ps1 test -tags ci -count=1 ./...
$env:CGO_ENABLED='1'
.\scripts\desktop.ps1 build -trimpath -ldflags '-H=windowsgui' -o ../bin/dvl-desktop.exe .
~~~

安装脚本仅下载固定版本且校验 SHA-256 的 LLVM-MinGW 到忽略的 `.tools`。开发脚本优先使用 `.tools/go` 中的 Go，否则使用 PATH，缓存写入 `.cache`；不修改系统持久设置。首次获取编译器与依赖需要开发时联网，应用运行无需联网。

完整交付检查及打包：

~~~powershell
.\scripts\package.ps1
~~~

脚本执行两个模块的格式、依赖、vet、测试、CLI/桌面 Windows 构建，并启用真实 CLI 端到端测试；输出 `dist/dotenvless-0.2.0-windows-amd64.zip`、SHA-256 文件、包内二进制清单与依赖许可证。应用未做代码签名。

单独运行端到端测试：将进程变量 `DVL_E2E_CLI` 设为已构建 `dvl.exe` 的绝对路径，再运行桌面测试。测试使用临时 Vault 和生成的假值，禁止读取真实凭据。

[Windows CI](.github/workflows/windows.yml) 分开验证纯 Go CLI 与 Fyne 桌面构建。远程结果以推送后实际运行为准。中文 IME、多 DPI、焦点和真实剪贴板、独立 Windows 10/11 及跨账户验收不能由无界面测试代替，见[人工验收](docs/manual-verification.md)。

- [v0.2 规格](docs/specs/v0.2.md) · [设计决策](docs/design.md) · [推进计划](docs/plan.md)
- [验证记录](docs/verification.md) · [变更日志](CHANGELOG.md) · [贡献说明](CONTRIBUTING.md)

## 许可证

Dotenvless 源码暂未声明许可证。发布包中的第三方依赖遵循各自许可证，详见包内 `THIRD_PARTY_NOTICES.txt`。
