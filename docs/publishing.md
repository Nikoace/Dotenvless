# 首次推送到 GitHub

当前工程已准备好以 `main` 作为默认分支。暂不添加许可证，也不预先填写未知的仓库 URL、作者联系方式或下载链接。

## 推送

在 GitHub 创建空仓库；如需保留本地历史，创建时不额外初始化 README、.gitignore 或许可证。进入本仓库，替换下面的仓库 URL：

```powershell
git status
git remote add origin <repository-url>
git push -u origin main
```

如果已有 origin，先运行 `git remote -v` 核对地址，勿重复添加或强制推送。本轮工程准备没有创建远程仓库或执行推送。

## 仓库设置

- 默认分支设为 main。首次 Windows CI 通过后，可将该检查设为合并要求。
- 在 Security 设置中启用私密漏洞报告，为 SECURITY.md 中的报告流程提供实际入口。
- 当前版本仍为 0.1.0-dev；CI 构建产物不等于经过全部发布验收的稳定 Release。
- 暂无许可证。新增许可证、正式 Release 和模块路径变更应分别作出明确决定。

CI 使用固定提交版本的官方 [checkout](https://github.com/actions/checkout)、[setup-go](https://github.com/actions/setup-go) 与 [upload-artifact](https://github.com/actions/upload-artifact)；Go 版本从 go.mod 读取，上传范围仅为 bin/dvl.exe。首轮远程 CI 结果需在推送后确认。

## 已有历史

保留每个里程碑、Red/Green 证据和集成验收记录。历史文档中的本机路径仅用于说明当时的执行环境；新用户无需这些路径、已下载工具链或测试缓存。当前整理不改写已有 Git 历史。
