# OpenCode Windows 免安装交付

需求真源：`docs/prd/opencode-portable-windows.md`。

## 切片 1：免 PATH 插件接入（AFK）

- 目标：setup 写入绑定当前 EXE 的插件，支持现有插件更新。
- 范围：`pkg/provider/opencode.go`、`pkg/provider/plugins/confab-sync.ts`、相应 Go / TypeScript 测试。
- 验收：绝对路径、中文空格与特殊字符、启动停止、旧插件重绑和幂等测试通过。
- 依赖：无。

## 切片 2：Windows 便携包与验证（AFK）

- 目标：交付可解压使用的 ZIP，修正可选 install 的 EXE 命名。
- 范围：`cmd/install.go`、构建脚本、Windows 使用文档、隔离环境冒烟测试。
- 验收：Windows 编译、插件测试、便携包解压及真实 EXE 调用验证通过；产物含说明与 SHA256。
- 依赖：切片 1。

## 执行记录

### 补充缺陷：Windows Git 子进程闪窗

- 目标：后台采集 Git 元数据时不创建控制台。
- 证据：本机 Win32_ProcessStartTrace 捕获旧 confab.exe PID 28052 → git.exe → conhost.exe，同时出现 Windows Terminal。
- 范围：`pkg/git/git.go`、平台进程属性文件及 Windows 回归测试；重打便携包。
- 验收：通过实际 gitCommand 调用测试子进程，GetConsoleWindow 返回 0；Git 原有行为测试及静态检查通过。
- 依赖：便携包切片。正在运行的旧实例不自动替换或停止。
- 边界：还捕获 Codex 桌面应用 ChatGPT.exe → gh.exe → conhost.exe，这是另一来源，本仓库修复不覆盖它。

- 2026-09-16：用户授权执行建议，明确先做本地验证。
- 仓库没有 Trellis 运行时，按项目要求使用本目录记录任务；不向外部 tracker 发布。
- 原有 Windows 改造为工作区未提交修改，保留这些修改，仅在相关模块上叠加本任务。
- 初始环境 PATH 无 Go；使用临时目录中的官方 Go 工具链验证，不全局安装。
- 原生冒烟复现 OpenCode nil payload 导致 Windows 停止标记缺失。补充 `pkg/daemon/stop_windows.go` / `stop_unix.go` 平台停止职责与回归测试，属于便携包可用性的必要修复。
- 2026-09-16：两个切片完成本地验收；最终交付文件与验证结果见同目录 `validation.md`。真实 OpenCode 应用及真实后端验收不属于本次已确认范围。
