# 通用接入开发：进度与交接

## 新窗口从这里开始

**方案已经确认，不需要再次请求总体评审。唯一下一步是执行 S0：原版 OpenCode 兼容性与最小接入探针。**

开发仓库：[Simonsms/zhlx-opencode-Confab](https://github.com/Simonsms/zhlx-opencode-Confab)。可以继续使用当前本地工作目录，或从该仓库取得代码后开发：

`E:\zhlx-platform-ai\ai-3-years\confab\confab-exe\confab`

此前未提交的 Windows 改造已纳入代码基线，方案、任务与本文随独立规划提交入库。新窗口仍须核对实际 HEAD 和工作区。构建成品、依赖目录、临时工具和用户认证配置不在 Git 中，需要在新环境重建。

读取顺序：

1. 项目所有适用的 `AGENTS.md`。
2. [需求 PRD](../../prd/universal-onboarding.md)。
3. [已确认开发设计](../../design/universal-onboarding.md)。
4. 本进度文档与[纳入版本管理前的工作区快照](working-tree-snapshot.txt)（归档证据，不是当前未提交列表）。
5. [S0 任务](../../../.trellis/tasks/universal-onboarding-s0-compatibility/prd.md)及同目录 `task.json`。

## 当前快照与授权范围

- 更新时间：2026-09-16。
- 分支：`main`。
- 上游基线：`f4540ab138062715b516dd04ccfd7f5ea3bf440a`。
- 本机最初代码提交 `167b64dff77574d2a2867a8711484c673276e87e` 和规划提交 `c324ab2` 保留在本地归档分支。完整历史因旧测试样本触发 GitHub 推送保护，未进入新仓库；公开 main 从清理后快照开始，最新 HEAD 以 `git rev-parse HEAD` 为准。
- 远程：`origin` 指向新开发仓库，`upstream` 保留原上游远程；不向上游推送本派生改造。
- 版本基线：Windows 移植、便携包、Git 无窗口修复、测试、方案与任务均包含在新仓库的独立初始快照中。原始历史在本地 `codex/upstream-history-20260916` 保留，不向 origin 推送。后续脏工作区仍需单独识别和保留。
- 阶段：方案已确认、任务已落盘，等待新窗口开始 S0；本窗口没有开始通用初始化插件编码。
- 用户先确认方案和新窗口交接，随后明确提供新 GitHub 仓库并要求开展版本管理。
- 方案确认允许按推荐基线实施；代码仓库已提供，但真实外网测试后端、包发行源、发布凭据和精确宿主版本仍需落实。
- 本轮已经授权将已有代码、方案和任务提交并推送至指定仓库；这不等于授权创建发布标签、部署生产服务或修改实际宿主配置。后续开发按[版本管理约定](../../version-control.md)执行。

## 已确认的实施基线

| 项目 | 结论 |
| --- | --- |
| 产品目标 | 减少下发后的人工配置；不是仅将安装脚本换成手动 setup |
| 首个入口 | OpenCode 官方插件 |
| 宿主约束 | 不改 OpenCode 源码、程序文件，不 fork，不替换其安装包 |
| 用户步骤 | 添加插件一次＋首次网页授权；统一下发可预置插件 |
| 配置与程序 | 后端由组织预设提供；核心随包携带；无全局 CLI/PATH/Go 前置要求 |
| 通用性 | 复用 Go 核心与接入协议，各工具提供自己的薄适配器 |
| 首版平台 | 按方案先 Windows x64；精确宿主/SDK/Bun 版本由 S0 核定 |
| 第二工具 | 首版做另一个 provider 的合同测试，完整第二插件不默认纳入 |
| 更新 | 插件和核心固定配套、可回退，托管模式不查询公网 latest |
| 顺序 | 外网验证先于内网部署 |

## 执行真源与任务状态

执行细节以 `.trellis/tasks/<task>/prd.md` 和 `task.json` 为准。当前所有新任务都是 pending；新窗口开始执行时只将当前任务设为 in_progress。

| 任务 | 类型 | 状态 | 依赖 | 验收概要 |
| --- | --- | --- | --- | --- |
| [S0 兼容性](../../../.trellis/tasks/universal-onboarding-s0-compatibility/prd.md) | AFK | pending | 无 | 原版宿主真实加载、随包 EXE、状态提示与退出、版本冻结 |
| [S1 首次接入](../../../.trellis/tasks/universal-onboarding-s1-first-connection/prd.md) | AFK | pending | S0 | 预置环境、非阻塞授权、一条虚构会话闭环 |
| [S2 日常可靠性](../../../.trellis/tasks/universal-onboarding-s2-reliable-lifecycle/prd.md) | AFK | pending | S1 | 恢复、并发、错误分类、持久暂停、最终同步 |
| [S3 迁移升级](../../../.trellis/tasks/universal-onboarding-s3-managed-upgrades/prd.md) | AFK | pending | S2 | 新旧入口不双跑、版本回退、通用协议合同 |
| [S4 外网验收](../../../.trellis/tasks/universal-onboarding-s4-external-acceptance/prd.md) | HITL | pending | S3＋资源/发布授权 | 实际发行源、实际测试后端、真实宿主与虚构数据 |
| [S5 内网交付](../../../.trellis/tasks/universal-onboarding-s5-intranet-delivery/prd.md) | HITL | pending | S4＋内网条件 | 私服或离线安装，不依赖公网，不串环境 |

仓库当前只有 `.trellis/tasks`，尚未发现正式 Trellis workflow/scripts。这里已经落实用户要求的项目内执行台账，不声称已通过不存在的 Trellis CLI 调度；不得因此重建另一份需求真源或擅自全局安装工具。

## S0 的第一步、范围和完成条件

先核对 Git 状态，然后读取本机 OpenCode 启动脚本定位真正发行包，记录 OpenCode/SDK/Bun/Node 版本。当前 PATH 中的入口是 `opencode.ps1`，不能仅凭这个名称判断宿主版本或是否原版。

在该版本实际支持的隔离配置下验证插件加载，避免同时加载用户全局旧 Confab 插件。特别注意：`CONFAB_OPENCODE_CONFIG_DIR` 是 Confab 的参数，不应当成 OpenCode 自身配置隔离的证据。

修改范围限于最小插件工程 `packages/opencode-plugin/`、相关测试和版本/兼容报告。用包内核心的无副作用调用（如 `--help`）验证绝对路径、窗口行为和退出。暂不实现全部认证或新机器协议，也不调用付费模型或真实后端。

完成条件详见 S0：真实宿主加载；真实官方类型；可见状态；无 PATH 的随包 EXE；Windows 无非预期终端；退出无遗留。仅手工 import 插件或直接调用 mock 处理器不算通过。

## 现有可复用成果及验证边界

| 成果 | 状态与证据 |
| --- | --- |
| Windows 进程和 daemon 适配 | 在当前工作区；非全平台完整验收 |
| OpenCode 便携版绝对路径绑定 | 已有测试，位于 `pkg/provider/opencode.go` 与插件源码；仍不是自动初始化产品 |
| 停止标记修复 | Windows nil payload 必须写 marker；相关 Go 回归测试通过 |
| Git 子进程隐藏窗口 | `HideWindow`＋`CREATE_NO_WINDOW`；Git 测试及 Linux 交叉编译通过 |
| 真实包内 EXE 验证 | 重新解压包后，约 46 秒后台重复 Git 测试与可见窗口观察通过；没有遗留测试进程 |
| 首次设备码登录 | 历史上对用户指定后端通过；之后认证接口两次超时，不能将其当持续可用的测试环境 |
| 正式通用插件 | 未实现；设计中的 integration 命令、CONFAB_HOME、正式 npm 包均不是现有能力 |

详细证据见[便携版验证](../../../.trellis/tasks/opencode-portable-windows/validation.md)与[新包复测](../../../.trellis/tasks/opencode-portable-windows/package-retest-20260916.md)。

已验证的开发包位于 `dist/portable-no-console/confab_dev_windows_amd64.zip`，SHA256：

`3136a975f056ea9d2c3c1acf2fef9f8331937f20c084cc9232f2b3f5dcc16dca`

这是旧便携产品的修复包，可用于 S0 基础探针；它不包含本轮设计的新初始化功能。不要误用上级另一目录的旧现场程序 `E:\zhlx-platform-ai\ai-3-years\confab\confab.exe`。

尚未运行完整 Windows `go test ./...`；旧测试中仍有 Unix HOME/路径假设。已有局部通过结果不能扩大成完整项目通过。当前开发包版本为 dev，默认跳过自动更新；不能用它证明正式版托管模式不会访问公网。

## 新窗口的工具与验证提示

- 当前 PATH 可发现 Node.js、npm、Bun、OpenCode；Go 不在 PATH。
- 已下载并验证的 Go 1.26.3 位于 `%TEMP%\confab-portable-tools\go\bin\go.exe`，交接时文件仍存在。临时目录可能清理，新窗口先检查，不能假设永远存在。
- 该目录下已有 `gopath` 与 `gocache`。复用时只在当前测试进程设置 GOPATH/GOCACHE；不修改持久系统环境变量。
- 原插件测试依赖已安装在 `pkg/provider/plugins/node_modules`；它们不代表新发行工程已建立。
- 原验证命令可参考：`npm.cmd test`、`npm.cmd run typecheck`（旧插件目录）、`go test ./pkg/git`、`go test ./test/portable`。便携测试需要显式提供 CONFAB_PORTABLE_BINARY，否则会跳过。
- S0 应新增并记录自己的真实宿主加载验证命令，不能只复跑旧测试后宣称兼容完成。

## 现场状态与未决信息

- 本轮交接没有启动任何测试/部署进程。最终检查时 confab.exe 数量为 0。
- Codex 用户配置中的 Confab 受管钩子仍存在；此前只获准停止进程，未获得停用钩子的确认。新会话可能拉起旧版；不要把一次“0 进程”当成永久暂停，也不要据此自动编辑用户配置或关闭 Codex 主程序。
- 外网测试后端、制品源/scope、发布权限：待用户/负责人在 S4 前提供。S0–S3 可用隔离模拟后端推进。
- 内网私服还是完全离线、实际 CA/代理/下发责任人：S5 前落实。总体方案确认没有给出这些具体值。
- 精确平台和宿主版本：S0 可自行查证，不必再次向用户询问能够从本机读到的信息。
- 不在交接文档中保存 API Key、临时认证配置的内容、真实会话或账户标识。

## 本次交接变更

- PRD 与设计标记为已确认，记录推荐基线与尚未提供的真实环境信息。
- 建立六个任务目录，每项具有目标、范围、依赖、验收标准和模块线索。
- 更新本进度文件并保存 Git 工作区快照。
- 最初交接仅修改文档。用户随后要求版本管理，先在本地分开提交代码与规划；原始历史推送受阻后，保留本地归档并以清理后的源码快照初始化新仓库。没有关闭扫描、强推、创建发布标签或新的 Codex 任务。

本次交接验证已通过：六个 task.json 的格式与 pending 状态、依赖引用和无环检查、任务目标/范围字段、文档相互链接、JSON 示例、代码围栏，以及 `git diff --check`。也已检查交接文档不再包含要求重新进行总体评审的旧前置条件。本轮没有重复运行产品测试，因为改动仅为交接文档与任务台账。

纳入版本管理前的文件清单保存在[归档快照](working-tree-snapshot.txt)。进入新窗口仍须重新运行 Git 命令；不能用历史快照推断当前脏状态，也不要 reset、clean、批量格式化或回滚陌生改动。

## 唯一下一步

在开发仓库的工作目录核对实际 Git 状态，读取 S0 任务，将其标为 in_progress，开始锁定版本并实现原版 OpenCode 的最小插件接入探针。新建功能分支使用 codex/ 前缀。

## 可直接粘贴到新窗口的指令

```text
在 E:\zhlx-platform-ai\ai-3-years\confab\confab-exe\confab 当前本地工作目录继续开发。
先读取项目 AGENTS.md、docs/handoff/universal-onboarding/progress.md 及其引用的 PRD、已确认开发设计和 Git 快照。
用户已经确认方案，不重复请求总体评审，也不要把未指定的环境地址或版本号当成已知。
核对 branch、HEAD、git status 和相关未提交差异，保留同事的 Windows 适配与此前修复。
读取 .trellis/tasks/universal-onboarding-s0-compatibility/prd.md 和 task.json，从 S0 开始编码；不要一口气实现 S1–S5。
不改 OpenCode 源码，不修改实际用户配置，不连接真实后端或发布包；S0 使用真实宿主的隔离测试环境。
实际完成并验证当前切片后，更新 task.json、验证记录和进度文件，指定唯一下一步。
未验证的内容如实标注；按 docs/version-control.md 管理小切片提交，不创建发布标签或执行部署。
```
