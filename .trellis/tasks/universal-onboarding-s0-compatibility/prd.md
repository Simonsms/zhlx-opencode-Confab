# S0 原版 OpenCode 兼容性与最小接入探针

状态：pending。任务类型：AFK。

需求：[PRD](../../../docs/prd/universal-onboarding.md)；技术方案：[开发设计](../../../docs/design/universal-onboarding.md)；交接：[进度](../../../docs/handoff/universal-onboarding/progress.md)。

## 目标与依赖

无前置任务。方案已确认，新窗口从本任务开始，不重复确认总体需求。

在不修改 OpenCode 源码、程序文件或用户实际配置的前提下，跑通“标准宿主加载插件 → 状态提示 → 调用包内程序 → 退出清理”的最小路径。

## 范围与模块线索

- 新增正式插件工程的最小骨架：`packages/opencode-plugin/`。
- 参考 `pkg/provider/plugins/confab-sync.ts`、`pkg/provider/opencode.go`、`test/portable/` 的现有行为，不能复制后长期维护两套实现。
- 使用目标版本真实的 `@opencode-ai/plugin` / SDK 类型，不能以当前自建 `types/opencode-plugin.d.ts` 作为兼容证明。
- 已有核心能力见 `cmd/spawn.go`、`cmd/procattr_windows.go`、`pkg/git/procattr_windows.go`。
- 探针可先以绝对路径调用包内 EXE 的 `--help` 等无副作用命令。设计中的 integration 命令目前不存在，不为本探针提前实现完整认证和同步。

## 第一步

核对当前 Git 工作区，读取本机 OpenCode 启动脚本并定位其实际发行包，记录 OpenCode、Bun、Node、插件 SDK 版本及其来源。当前 PATH 能发现的是 `opencode.ps1`，其存在不等于已经确认原版宿主或具体版本。能从本机查证的信息不询问用户。

随后按该版本官方加载行为建立隔离测试配置。不得把 `CONFAB_OPENCODE_CONFIG_DIR` 当成控制 OpenCode 自身插件加载的配置项；应验证用户全局插件和旧 Confab 钩子不会混入测试。

## 验收标准

- [ ] 锁定实际宿主、SDK、Bun 和平台版本，记录兼容范围。
- [ ] 原版 OpenCode 真实加载插件；不是仅在 Bun 中手工 import 后直接调用处理器。
- [ ] 连接状态提示使用公开接口，记录 CLI/桌面展示能力的实际边界。
- [ ] 插件能通过绝对路径调用随包 EXE；无全局 confab、无 PATH 依赖。
- [ ] 宿主 → 插件 → EXE 这段调用不产生可见控制台，中文及空格路径正常。
- [ ] 验证目标版本 event/dispose 的真实行为，正常退出不遗留探针进程。
- [ ] 状态提示或程序调用失败不会阻塞宿主使用。
- [ ] 留下可重复执行的测试命令和验证报告；不包含密钥或真实会话正文。

## 验证与非范围

先添加最小有意义的测试，再实现探针。确定脚本后记录准确的 npm/Bun/宿主验证命令。已有基础可参考 `go test ./test/portable`，但不能用它替代真实插件加载验证。

本任务不连接真实后端、不调用付费模型、不发布 npm 包、不修改全局配置、不开始 S1 完整初始化。外网后端和内网私服尚未指定不阻塞 S0。

完成后更新本任务状态和交接进度，再将 S1 设为唯一下一步。
