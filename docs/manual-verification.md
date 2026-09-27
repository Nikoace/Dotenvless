# 需要独立环境的验收

当前已在本机验证隐藏输入/取消恢复、Ctrl+C、真实工具链和完整 CLI 流程。以下项目不以 mock 或跳过代替通过。

## 第二 Windows 账户：DPAPI 拒绝解密

测试只产生固定假值的密文，独立于真实 Vault。无需输入真实 Secret，也不需要安装 Go 到账户 B。

在 Windows 账户 A 中，从本仓库根目录构建专用测试程序：

```powershell
.\scripts\dev.ps1 test -c -o .cache/dpapi-account.test.exe ./internal/crypto
$env:DVL_DPAPI_CHECK_FILE = Join-Path (Get-Location).Path '.cache/dotenvless-dpapi-fixture.json'
$env:DVL_DPAPI_CHECK_MODE = 'write'
.\.cache\dpapi-account.test.exe '-test.run=^TestDPAPIAcrossAccounts$' '-test.v'
$env:DVL_DPAPI_CHECK_MODE = 'read-same'
.\.cache\dpapi-account.test.exe '-test.run=^TestDPAPIAcrossAccounts$' '-test.v'
```

write 要求目标文件尚不存在。将生成的测试 exe 和密文 JSON 一起复制到账户 B 可读取的专用测试目录。可以用已有普通测试账户，不需要为此自动创建账户。

实际登录账户 B，进入包含上述两个文件的目录后运行：

```powershell
$env:DVL_DPAPI_CHECK_FILE = (Resolve-Path './dotenvless-dpapi-fixture.json').Path
$env:DVL_DPAPI_CHECK_MODE = 'read-other'
.\dpapi-account.test.exe '-test.run=^TestDPAPIAcrossAccounts$' '-test.v'
```

通过条件：程序实际读到密文，确认 Windows SID 与写入账户不同，然后 DPAPI 解密失败。文件不可读、同一账户、无效 JSON 都判失败，不冒充 DPAPI 安全验收。完成后自行移除专用假值测试文件及当前 shell 的测试环境变量。

执行记录应注明两个不同账户、Windows 版本、write/read-same/read-other 的实际结果。当前工作环境的第二身份沙箱启动失败，因此 read-other 尚未执行。

## 其他发布前检查

- Windows 10 与 Windows 11 的独立系统验收；当前本机结果不代表全部 Windows 版本。
- 远程 Windows CI：工作流已写入，执行状态与日志以仓库 [Actions](https://github.com/Nikoace/Dotenvless/actions) 记录为准。
- UNC 共享与特殊文件系统没有现场测试；本版本以本地 Windows Git 项目为主要场景。
- 发布许可证尚未指定，仓库不预设开源授权。

## v0.2 桌面专项（目前未执行完整人工验收）

仅使用新建临时目录、临时 APPDATA 和生成的假值。不要把日常真实 Vault 作为验收对象，也不要为测试复制真实密钥。

1. 在独立 Windows 10 和 Windows 11 x64 机器解压同一发布包，记录系统、显卡/驱动、DPI、输入法及包的 SHA-256。分别打开客户端，确认不需 Go/C 编译器或浏览器运行时，关闭窗口后进程退出。
2. 在假工程内使用中文 IME 输入共享条目名、中文多行假值；保存后通过真实 CLI helper 校验值。覆盖 100%、150%、200% 缩放及不同 DPI 显示器间移动，确认文字、对话框、表格与滚动区可用。
3. 查看一个假值：15 秒后隐藏；在倒计时内切换工程/环境/页面、切换到其他应用或锁屏，应隐藏。再次查看不受旧计时器影响。
4. 准备可丢弃的系统剪贴板内容，复制假值并确认普通粘贴正确。用 Win+V 确认其未进入历史；开启云剪贴板的两台专用测试机验证不同步。30 秒和正常退出均应清理本次复制；之后再复制无关内容，确认过期清理不会覆盖它。剪贴板被其他程序占用时应显示可理解的失败提示。
5. 打开两份客户端/CLI，修改同一 Vault：旧编辑页提交要求刷新。共享更新前展示工程/环境/变量名，引用存在时删除受阻；取消确认后文件摘要不变。
6. 大目录发现中取消；取消导入、环境/工程删除、共享修改。核对 Vault 摘要和原工程文件均未改变。导入覆盖明确勾选后整批提交，原文件保留。
7. 在禁止应用联网的专用测试环境完整操作登记、编辑、复制、导入、CLI run；记录出站与监听监测。构建时下载依赖不属于运行时联网。
8. 按既有跨账户步骤验证 DPAPI；只转交生成的假值密文。不要把同一账户的往返测试当成跨账户拒绝证明。

当前本机已执行原生 Fyne/OpenGL 假数据窗口启动/截图/退出，中文标签在当前系统 DPI 显示正常；自动化计时与剪贴板所有权通过替身验证。它们不替代上述人工项目。
