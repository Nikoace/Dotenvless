# 需要独立环境的验收

当前已在本机验证隐藏输入/取消恢复、Ctrl+C、真实工具链和完整 CLI 流程。以下项目不以 mock 或跳过代替通过。

## 第二 Windows 账户：DPAPI 拒绝解密

测试只产生固定假值的密文，独立于真实 Vault。无需输入真实 Secret，也不需要安装 Go 到账户 B。

在 Windows 账户 A 中构建专用测试程序：

```powershell
.\scripts\dev.ps1 test -c -o .cache/dpapi-account.test.exe ./internal/crypto
$env:DVL_DPAPI_CHECK_FILE = Join-Path (Get-Location).Path 'dotenvless-dpapi-fixture.json'
$env:DVL_DPAPI_CHECK_MODE = 'write'
.\.cache\dpapi-account.test.exe '-test.run=^TestDPAPIAcrossAccounts$' '-test.v'
$env:DVL_DPAPI_CHECK_MODE = 'read-same'
.\.cache\dpapi-account.test.exe '-test.run=^TestDPAPIAcrossAccounts$' '-test.v'
```

先准备一个专用测试目录，替换示例路径；write 要求目标文件尚不存在。将测试 exe 和密文 JSON 复制到账户 B 可读取的位置。可以用已有普通测试账户，不需要为此自动创建账户。

在实际登录的账户 B 下运行：

```powershell
$env:DVL_DPAPI_CHECK_FILE = Join-Path (Get-Location).Path 'dotenvless-dpapi-fixture.json'
$env:DVL_DPAPI_CHECK_MODE = 'read-other'
.\dpapi-account.test.exe '-test.run=^TestDPAPIAcrossAccounts$' '-test.v'
```

通过条件：程序实际读到密文，确认 Windows SID 与写入账户不同，然后 DPAPI 解密失败。文件不可读、同一账户、无效 JSON 都判失败，不冒充 DPAPI 安全验收。完成后自行移除专用假值测试文件及当前 shell 的测试环境变量。

执行记录应注明两个不同账户、Windows 版本、write/read-same/read-other 的实际结果。当前工作环境的第二身份沙箱启动失败，因此 read-other 尚未执行。

## 其他发布前检查

- Windows 10 与 Windows 11 的独立系统验收；当前本机结果不代表全部 Windows 版本。
- 远程 Windows CI：工作流已写入，但本项目没有配置远程仓库或实际运行记录。
- UNC 共享与特殊文件系统没有现场测试；本版本以本地 Windows Git 项目为主要场景。
- 发布许可证尚未指定，仓库不预设开源授权。
