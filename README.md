# Dotenvless

Windows-first 本地 Secret 管理 CLI。将 Secret 保存在项目外的加密 Vault 中，在运行程序时注入子进程环境变量，使开发不再依赖真实 `.env` 文件。

当前阶段：M0 已完成。帮助与版本可运行，以下 Secret 工作流仍是目标用法。

```powershell
dvl init
dvl set OPENAI_API_KEY
dvl set DATABASE_URL
dvl list
dvl run -- python app.py
```

## 工作方式

采用 SDD（Specification-Driven Development，规格驱动开发）与 TDD：先定义可验证的行为，再运行失败测试，写最小实现，最后重构和回归。每个里程碑保持可编译，更新文档，并提交 Git。设计歧义先讨论，再实现相关行为。

- [V0.1 规格与验收条件](docs/specs/v0.1.md)
- [设计决策与待讨论事项](docs/design.md)
- [里程碑和测试计划](docs/plan.md)
- [验证记录](docs/verification.md)

## 安全范围

V0.1 使用 Windows DPAPI Current User scope 保护 Secret，不提供明文 `get` 命令。Dotenvless 自身不联网、不上传日志、不启动 daemon，不把 Secret 写入项目文件或持久环境变量。

这是本地存储保护工具。同一 Windows 身份运行的程序可能调用 DPAPI 解密；获得注入环境的子进程及其后代也可能读取或输出 Secret。V0.1 不提供 Agent 隔离，也不拦截目标应用的网络或输出。依据：[Microsoft DPAPI 文档](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata)。

## 当前状态

| 阶段 | 状态 |
| --- | --- |
| 规格、测试计划 | 基线已建立；D-01/D-02/D-03 已确认 |
| M0 项目初始化 | 完成：帮助/版本、测试、Windows CI 定义 |
| M1–M5 核心功能 | 未开始 |
| M6–M8 导入、示例、状态检查 | 未开始 |

运行时离线与开发工具下载是不同范围；开发工具链固定为 Go 1.27.1，位于忽略的 `.tools/go`，已核对官方 SHA-256。发布许可证尚未选定，不预设开源授权。

## 开发

已安装 Go 时直接使用 `go test ./...`。当前工作区也可使用隔离的工具链：

```powershell
.\scripts\dev.ps1 test ./...
.\scripts\dev.ps1 vet ./...
.\scripts\dev.ps1 build -trimpath -o bin/dvl.exe ./cmd/dvl
.\bin\dvl.exe --help
.\bin\dvl.exe --version
```

`bin`、工具链和构建缓存均不会进入 Git。CI 定义已添加，远程 CI 尚未触发。
