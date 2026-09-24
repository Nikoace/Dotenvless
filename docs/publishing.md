# GitHub 发布

公开仓库：[Nikoace/Dotenvless](https://github.com/Nikoace/Dotenvless)，默认分支为 `main`。暂不添加许可证；当前版本仍为 0.1.0-dev。

## 推送更新

本地 origin 已指向上述仓库。进入本仓库，先检查工作区与远程地址，再推送 main：

```powershell
git status
git remote -v
git push -u origin main
```

如需重新获取项目，可运行 `git clone https://github.com/Nikoace/Dotenvless.git`。不要重复添加 origin，也不要将缓存、工具链、Vault 或真实环境文件加入提交。

## 仓库设置

- 默认分支为 main。Windows CI 通过后，可将该检查设为合并要求。
- 在 Security 设置中启用私密漏洞报告，为 SECURITY.md 中的报告流程提供实际入口。
- CI 构建产物不等于经过全部发布验收的稳定 Release。
- 暂无许可证。新增许可证、正式 Release 和模块路径变更应分别作出明确决定。

CI 使用固定提交版本的官方 [checkout](https://github.com/actions/checkout)、[setup-go](https://github.com/actions/setup-go) 与 [upload-artifact](https://github.com/actions/upload-artifact)；Go 版本从 go.mod 读取，上传范围仅为 bin/dvl.exe。执行结果见仓库 [Actions](https://github.com/Nikoace/Dotenvless/actions)。

## 历史清理

公开推送前，经用户确认，历史文档中的本机路径、账户说明和专属缓存目录已改为通用描述。原有 15 个提交及其里程碑、Red/Green 和集成验收事实全部保留，提交 SHA 已改变；逐提交比较确认代码、测试及 CI 配置未变。

原始历史备份与提交映射仅保存在本地，不加入 Git 或上传。历史验收记录提到的旧 SHA 对应清理前的执行状态。
