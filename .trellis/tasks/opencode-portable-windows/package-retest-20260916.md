# 修复包复测：2026-09-16

## 测试对象

- 包：`dist/portable-no-console/confab_dev_windows_amd64.zip`
- SHA256：`3136a975f056ea9d2c3c1acf2fef9f8331937f20c084cc9232f2b3f5dcc16dca`
- 从 ZIP 重新解压到 `dist/package-verification-20260916-162318/extracted/`，没有以重新编译的 EXE 替代交付包。
- 未切换日常运行路径，未修改实际用户的 Codex/OpenCode 配置，未恢复自动同步。

## 已通过

1. ZIP 校验值与 `.sha256` 一致。
2. `TestOpenCodePortable`：新旧插件迁移、setup 幂等、移动目录重新绑定、空 PATH、中文/空格/特殊字符路径、SQLite 采集、恢复不重复追加、正常停止。
3. 新增 `TestPackagedBackgroundGit`，实际启动包内 EXE 并访问隔离 Git 仓库及虚构 OpenCode SQLite。只连接本地模拟同步后端：先返回 6 次可重试到下一轮的初始化错误，随后允许初始化与上传。
4. 第一轮后台测试 46.87 秒，完成 7 次初始化、1 次虚构消息上传，Git 元数据正确，进程及状态正常清理。Win32_ProcessStartTrace 捕获测试 EXE 后代中的 108 个 Git 进程。
5. 第二轮后台测试 46.37 秒；同时以约 10ms 间隔读取顶层窗口可见性和前台焦点，观察持续 46.49 秒，可见控制台/终端事件为 0。观察范围包括 ConsoleWindowClass、CASCADIA_HOSTING_WINDOW_CLASS，以及终端、Git、Confab 等进程拥有的新可见窗口；不读取窗口内容。
6. `go vet ./test/portable` 通过。
7. 测试后进程枚举没有遗留 `confab.exe`。

注意：`CREATE_NO_WINDOW` 下仍可能创建无界面的 `conhost.exe`。第一轮确有隐藏控制台宿主，不能以宿主进程数量推断可见闪窗；第二轮采用实际窗口可见性观测。结论限于本次测试场景与采样窗口，不承诺所有 Codex 外部子进程都不会闪窗。

## 真实后端复查未通过（超时）

使用此前用户真实授权后保存在隔离目录中的凭据，调用新包 `status`。两次请求 `/api/v1/auth/validate` 均在 5 秒后超时：

```text
context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

内网测试后端的 TCP 连接可以建立，但未收到该认证接口的 HTTP 响应（公开仓库中不记录内网地址）。CLI 把请求错误统一显示成 Invalid，不能据此断言 API Key 失效。本次未重新登录、未修改凭据、未发送真实会话数据；真实 OpenCode/后端上传全链路仍未验收。

## 原始记录

- `dist/package-verification-20260916-162318/background-git-test.stdout.txt`
- `dist/package-verification-20260916-162318/process-starts.jsonl`
- `dist/package-verification-20260916-162318/visible-window-test.stdout.txt`
- `dist/package-verification-20260916-162318/visible-window-observation.json`
- `dist/package-verification-20260916-162318/real-backend-status.stdout.txt`
- `dist/package-verification-20260916-162318/real-backend-status-retry.stdout.txt`

可复用测试：`test/portable/git_windows_test.go`。窗口观察脚本位于上述产物目录的 `observe_visible_windows.py`。
