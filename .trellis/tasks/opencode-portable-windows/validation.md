# 本地验收记录

日期：2026-09-16。环境：Windows amd64、Go 1.26.3、Bun 1.3.13。

最新修复包：`dist/portable-no-console/confab_dev_windows_amd64.zip`，SHA256 `3136a975f056ea9d2c3c1acf2fef9f8331937f20c084cc9232f2b3f5dcc16dca`。下方 `dist/portable/` 是此前的验证码登录验收版本，未包含后续 Git 无窗口启动修复。

用户要求的后续复测见 [package-retest-20260916.md](package-retest-20260916.md)：实际包内 EXE 的后台 Git 和窗口可见性测试通过，连续约 46 秒未观察到可见终端；真实后端本轮认证接口两次超时，TCP 可达。

## 交付文件

- ZIP：`dist/portable/confab_dev_windows_amd64.zip`（5,481,903 字节）
- 校验文件：`dist/portable/confab_dev_windows_amd64.zip.sha256`
- SHA256：`816e9ba1ef0a4f44c35c24e4a922333fe5d0f83a57fc68096ff75411ccc212af`
- ZIP 内容：`confab.exe`、`README.md`、`LICENSE`。
- 说明：`docs/windows-portable.md`。
- `dist/` 其他目录为开发过程的构建产物；仅上述 `dist/portable/` ZIP 是最终验收版本。

## 已通过

1. 先写回归测试，复现路径未绑定、旧插件误判和 `.exe` 后缀丢失；修复后通过。
2. `go test ./pkg/provider -run 'TestOpencode(InstallHooks|UninstallHooks|IsHooksInstalled|Plugin)|TestCollector|TestReadSession|TestMaterializedEnvelope' -count=1 -timeout 120s`。
3. 插件 22 个 Vitest 测试与 `tsc --noEmit`（也由上述 Go 测试调用）。
4. `go test ./pkg/process ./pkg/opencodetest -count=1 -timeout 60s`。
5. `go test ./pkg/daemon -run '^TestWindowsStop' -count=1 -timeout 30s`：无 payload 必须写停止标记、标记写入失败必须返回错误。
6. `go test ./cmd -run '^TestInstallPreservesPlatformExecutableName$' -count=1 -timeout 30s`。
7. `go vet ./...`。
8. Windows 原生 ZIP 构建及 SHA256 比对。
9. `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...` 交叉编译。
10. 从最终 ZIP 解压后设置 `CONFAB_PORTABLE_BINARY`，执行 `go test ./test/portable -v -count=1 -timeout 90s`。实际验证：
    - 回环地址认证替身下，完整 setup 更新旧插件，重复 setup 幂等。
    - 从另一目录 hooks add 后绑定到新 EXE。
    - 调用进程 PATH 为空，中文、空格、`&` 等路径仍可启动。
    - 真实 Windows 后台 EXE 读取虚构 OpenCode SQLite，会话 cwd JSON 正确传递。
    - 新建会话与恢复会话均物化正确消息，没有重复追加。
    - dispose 写入停止标记；后台进程退出，state/inbox 清理。

## 验证边界

### 补充：验证码登录（2026-09-16）

使用同一最终 ZIP 解压出的 EXE，在隔离 HOME / USERPROFILE / 配置目录、空 PATH 下，执行不带 `--api-key` 的 `setup --provider opencode --backend-url <本地回环地址>`。本地模拟服务器返回设备码，先返回一次 `authorization_pending`；测试脚本提交匹配的虚构验证码后，第二次轮询领取凭据。已检查验证链接与验证码输出、`Authentication successful!`、`Setup complete.`、凭据及脱敏配置保存、插件绑定 EXE 路径，结果通过。

此测试没有经过真实网页登录或真实后端。当前 `cmd/login.go` 的 `openBrowser` 仅处理 macOS/Linux；Windows 会返回 unsupported platform，但主登录流程忽略此辅助错误并继续轮询。因此 Windows 用户需要手动打开终端显示的链接完成网页授权，尚不支持自动弹出浏览器。

### 其余边界

- 未运行真实 OpenCode 应用的 UI/插件加载生命周期，未验证真实会话或子会话上传。真实后端的验证码登录已在后续复测通过，见下文。
- 未执行完整 Windows `go test ./...`，仅执行上述针对性测试；已有其他测试中仍有 Unix HOME/路径假设。
- Linux 只做交叉编译，未做 Linux 运行时测试；ARM64 未验证。
- 新增 Windows CI 工作流，但尚未推送或触发远端 CI。
- 没有修改实际用户的 OpenCode 配置或持久 PATH，没有发布 npm 包或创建 Git 提交。
- 本地 Go 工具链位于临时目录，未做全局安装。

## 后续：真实后端验证码登录验证通过

用户提供的内网测试后端曾完成真实网页设备授权（公开仓库中不记录内网地址）。使用最终 ZIP 中的 EXE、不带 `--api-key`，在独立临时配置和 OpenCode 插件目录中运行 setup。首个设备码过期；重新申请后用户确认授权，EXE 正常退出（退出码 0）。

实际输出（后端地址已脱敏）：

```text
Authentication successful!
Redaction enabled (default patterns)
hooks installed
Setup complete. opencode sessions will sync to <test-backend-url>
```

随后在同一隔离环境执行 `confab.exe status`，后端返回 `Validating API key... ✓ Valid`、`Status: ✓ Authenticated and ready`，OpenCode 插件状态为 Installed。新凭据、默认脱敏设置和插件文件均已保存；不在文档中记录 API Key。

原有用户配置文件修改时间仍为 `2026-09-16 11:20:53`。实际使用的 OpenCode 插件和系统环境变量没有切换，本次没有启动会话采集或上传。

## 后续：周期性终端闪窗

45 秒 Win32_ProcessStartTrace 记录捕获到两条独立链路：

- 原有 confab.exe（PID 28052）→ git.exe → conhost.exe；现场 EXE 位于另一目录 `E:\zhlx-platform-ai\ai-3-years\confab\confab.exe`，不是本次产物。
- Codex 桌面进程 ChatGPT.exe → gh.exe → conhost.exe；不属于本仓库可修改范围。

`pkg/git.gitCommand` 缺少 Windows 进程属性。现已设置 `HideWindow` 和 `CREATE_NO_WINDOW`，其他平台保持默认属性。子进程回归测试先复现 hidden=false，修复后确认 console=0、hidden=true。`go test ./pkg/git -count=1 -timeout 120s`、相关包 go vet、Linux 全项目交叉编译通过。新 ZIP 解压后的原生 OpenCode 冒烟测试也通过，校验值一致。尚未在现场替换旧 EXE，不宣称所有可见闪窗已经消失。

用户随后明确要求先停止后台服务。通过已验证的 Windows inbox 停止标记请求旧 Codex daemon 正常退出；PID 28052 已退出，随后进程枚举确认 confab.exe 数量为 0。没有删除配置、修改系统环境变量、关闭 Codex/OpenCode 或替换现场二进制。Hook 仍保留，新会话可能重新启动旧版同步，恢复前需要切换到修复版。

## 版本管理前检查

最初本机代码基线提交：`167b64dff77574d2a2867a8711484c673276e87e`（现保留于本地归档分支；公开仓库另从清理后快照起步）。

- Windows 平台 `staticcheck ./...` 与 `deadcode -test ./...` 通过，无报告。
- provider/OpenCode、采集器、SQLite 读取、process、Git、Windows 停止及安装相关针对性测试通过；插件 Vitest/TypeScript 由 provider 测试执行通过。
- updater 提取测试原先假定存在 Unix 执行位；已将该断言限定到非 Windows，内容校验保留，相关测试重新通过。
- 安装脚本语法检查、发行/安装/更新仓库一致性检查、差异格式及针对性凭据扫描通过。
- 提交目标为新开发仓库；未发布标签。完整 Windows 测试集和远端 CI 结果不由上述局部检查替代。
- GitHub Push Protection 拒绝原始历史里的供应商密钥格式测试样本后，当前配置/字符串截断测试改用通用字符串，Slack/Stripe 正则测试改用新合成的重复字符数据。相关配置测试、utils/redactor 全包测试及 Staticcheck 重新通过；未保留被标记值、未关闭或绕过扫描。
