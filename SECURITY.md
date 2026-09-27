# 安全说明

当前版本为 0.2.0，尚未承诺稳定版本维护周期。已知验证范围及未完成验收见 [docs/verification.md](docs/verification.md)。

## 报告问题

请勿在公开 Issue、PR 或附件中提供真实 Secret、Vault 文件、环境文件或完整进程环境。

如果本仓库 GitHub Security 页面提供 **Report a vulnerability**，请使用该私密入口，提供受影响版本、使用假值的最小复现、预期与实际结果。若尚未启用私密报告，可先在 Issue 中请求私密联系方式，不公开漏洞细节或凭据。维护者在首次发布仓库时应启用私密漏洞报告。

## 保护范围

- 使用 Windows DPAPI Current User 保护落盘值，项目、配置层与键名绑定，共享值绑定稳定共享 ID；不自行设计密码算法。
- 不防御管理员、同一 Windows 身份下的恶意进程、系统内存转储或目标程序主动泄露环境变量。
- CLI 与客户端不访问网络、不含遥测，不创建明文临时环境文件，不修改持久环境。
- `status` 的文件名检查不能替代内容扫描；Vault 加密不负责清理已经泄露的凭据或 Git 历史。

DPAPI 平台行为参考 [Microsoft 文档](https://learn.microsoft.com/en-us/windows/win32/api/dpapi/nf-dpapi-cryptprotectdata)。

客户端单条查看会短时显示明文；复制通过 Windows 剪贴板排除历史与云同步，定时/退出清理仅在序列号仍属于本次复制时执行。这些标记不防御同一身份程序主动读取剪贴板，异常终止可能来不及清理。界面偏好与日志不存储值；Vault 中工程路径、环境名、键名和引用关系属于元数据。v1 备份保留 DPAPI 绑定，不提供跨机器迁移。