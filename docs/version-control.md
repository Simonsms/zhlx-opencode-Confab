# 版本管理约定

## 仓库与来源

- 开发仓库：[Simonsms/zhlx-opencode-Confab](https://github.com/Simonsms/zhlx-opencode-Confab)。
- 上游：[ConfabulousDev/confab](https://github.com/ConfabulousDev/confab)。保留其来源和 MIT 许可。
- `origin` 用于本派生仓库的获取和推送；`upstream` 保留原上游远程，用于对照和受控同步。
- Go module/import 路径暂保留上游名称，避免仅为仓库迁移批量修改代码。

首次推送完整上游历史时，GitHub Push Protection 命中了旧测试里的 Stripe/Slack 密钥格式样本。没有关闭保护或申请放行。当前测试已改用不含供应商凭据的通用字符串或新合成的重复字符数据，原字符串不再保留于发布快照中。

新仓库的 main 从该清理后快照开始独立历史。原 112 个上游提交及本机最初两个改造/规划提交保留在本地 `codex/upstream-history-20260916` 分支，可经 upstream 对照；不把该归档分支推到新仓库，不使用 `push --all` / `--mirror`。

## 日常开发

1. `main` 保留可追溯的集成基线。
2. 功能分支使用 `codex/` 前缀，例如 `codex/universal-onboarding-s0`。
3. 每个 Trellis 垂直切片单独实现、验证并提交，提交说明应包含具体行为及验证范围。
4. 合并前检查相关测试和 CI；失败、超时或未验证项目必须如实记录。
5. 不强推共享分支，不因同步上游覆盖本地改造。同步上游时审查并摘取必要差异，避免直接合并含被标记历史样本的归档分支。

## 纳入和排除

纳入源码、测试、文档、需求真源、`.trellis/tasks` 与交接记录。

不纳入 EXE、ZIP/下载缓存、`dist/`、`node_modules/`、Go 工具链缓存、用户配置、密钥和真实会话。验证文档只保留必要结论，不记录私有后端地址或凭据。

Git 提交使用当前账户的 GitHub noreply 邮箱，设置仅作用于本仓库，不修改全局 Git 身份。Windows 工作区禁用文件系统执行位比较，保留索引中原有 shell 脚本的可执行标记，避免无内容的权限变更。

## 发布

代码提交和推送不等于发布版本。创建并推送 `v*` 标签会触发 Release 工作流，必须作为明确的发布动作单独处理。

GoReleaser、安装脚本和 CLI 更新源均指向本仓库。当前 Windows 便携包仍通过本地构建脚本生成；通用插件的配套发布、内外网预设和受控升级按已确认开发设计实现。

## 新窗口与新机器

代码及交接文档提交后，可从 `origin` 获取再开始 S0。`dist/` 成品、依赖目录和本机临时工具不在 Git 中，需按文档重建。先读取 `docs/handoff/universal-onboarding/progress.md` 并核对实际 Git 状态，不能把历史工作区快照当作当前未提交列表。
