# 贡献说明

当前项目处于 0.1.0-dev 阶段，产品目标是 Windows 上的本地 Git 项目 Secret 管理。先阅读 [AGENTS.md](AGENTS.md)、[设计决策](docs/design.md)和对应[规格](docs/specs/v0.1.md)。许可证尚未指定。

## 环境

Windows、Git for Windows，以及满足 go.mod 要求的 Go。无需本地 API Key、真实 Vault 或其他项目；自动测试创建自己的临时目录和假值。

## 修改流程

1. 在 docs/specs 中明确行为、错误边界和验收 ID。涉及新安全边界或不兼容行为时先讨论。
2. 添加行为测试，并在实现前实际运行以确认失败；在 docs/verification.md 记录 Red 命令和结果。
3. 实现最小修改，运行回归，记录 Green。纯文档修改不需要为了覆盖率编写文案测试。
4. 更新 README 或命令帮助，审查 diff 后提交。PR 说明问题、最终行为和验证证据。

```powershell
$files = git ls-files '*.go'
gofmt -w $files
go mod verify
go vet ./...
go test -count=1 ./...
go build -trimpath -o bin/dvl.exe ./cmd/dvl
.\bin\dvl.exe --help
git diff --check
```

新建的 Go 文件也需要 gofmt。依赖变更时运行 `go mod tidy` 并审查 go.mod/go.sum。

## 测试约束

- 使用生成的假 Secret、临时项目、临时 Vault；不要在测试中读取或修改个人真实 Vault。
- 不把 Secret、环境文件内容或潜在敏感参数写进日志、Issue、PR 或断言失败信息。
- 交互终端、跨账户和 Gradle 测试的 opt-in 方式见[人工验收](docs/manual-verification.md)。跳过不等于通过。
- 本项目当前不提供 Linux/macOS 产品支持；CI 与平台行为测试使用 Windows。

提交前检查 `git diff --cached`，确保工具链、缓存、可执行文件、Vault 和真实 `.env` 没有进入提交。Git 操作仅限本仓库。
