# Windows / OpenCode 免安装使用说明

## 首次配置

1. 将 ZIP 解压到长期保留的目录，例如 `D:\tools\confab`。支持中文和空格路径。
2. 在该目录打开 PowerShell，执行：

   ```powershell
   .\confab.exe setup --provider opencode --backend-url https://你的测试后端地址
   ```

3. 按提示完成浏览器认证；重启 OpenCode 后开始新会话或恢复会话。

无需执行 `install.sh` 或 `confab.exe install`，无需管理员权限、注册服务或修改 PATH。EXE 内已包含插件，setup 默认写入 `%USERPROFILE%\.config\opencode\plugins\confab-sync.ts`，并将本次 EXE 的绝对路径写进插件。OpenCode 从桌面或其他终端启动也可调用它。

setup 仍然需要访问你指定的后端；免安装不等于免认证。后端凭据保存在 `%USERPROFILE%\.confab\config.json`，不会打进分发包。`CONFAB_OPENCODE_CONFIG_DIR` / `XDG_CONFIG_HOME` 仍可覆盖插件目录。

## 更新、移动和回退

退出 OpenCode 并等待 confab 同步进程结束，再解压新版本。移动 EXE 或切换到旧版本后，在目标目录执行：

```powershell
.\confab.exe hooks add --provider opencode
```

然后重启 OpenCode。该命令只重新绑定插件，不需要重新登录。使用 `setup` 也会更新旧版或旧路径插件，但会先检查后端认证。不要手工编辑生成的插件；再次 setup 会用当前 EXE 的版本和路径重建它。

保留旧版本目录即可回退：从旧目录重新执行上述 hooks add 命令并重启 OpenCode。

## 检查

```powershell
.\confab.exe status
.\confab.exe sync status
.\confab.exe list --provider opencode
```

会话开始后，后台进程读取 OpenCode 本地 SQLite，生成 `%USERPROFILE%\.confab\opencode\<会话ID>\messages.jsonl`，再脱敏并上传。日志位于 `%USERPROFILE%\.confab\logs\confab.log`。首次 EXE 启动失败的错误由 OpenCode 插件输出到 OpenCode 的日志/控制台。

没有活动会话时不一定有 confab 进程，这是正常行为。运行目录必须在整个会话期间保持可访问；移动后需要重新绑定。

## 范围与限制

- 后台采集 Git 信息时，Windows 子进程使用无控制台启动，避免 Git 命令周期性唤起终端。已运行的旧 EXE 不会因解压新包而自动更新，需要结束旧实例并从新版本启动。
- 免 PATH 支持针对 OpenCode 自动同步。`/retro` 等手工技能仍使用裸命令；可直接用 EXE 完整路径执行对应命令。
- Windows 自动更新不属于本次交付。使用 ZIP 版本替换与重新绑定流程升级。
- 开发版不等于已签名发行版；真实 OpenCode 版本及后端上传需要在目标环境验收。本次默认只做虚构会话和本地隔离验证。

## 开发者构建

在仓库根目录执行（需要项目 go.mod 指定的 Go 工具链）：

```powershell
.\scripts\build-windows-portable.ps1
```

默认输出 `dist\confab_dev_windows_amd64.zip` 及 `.sha256`，不会覆盖已有产物。通过 `-OutputDirectory` 指定新的构建目录；ARM64 使用 `-Architecture arm64`。用户使用成品 ZIP 不需要 Go 或构建脚本。

## 本地回归验证

需要 Go、Node.js/npm，以及 Bun 1.3.13（Windows CI 固定此版本）。在仓库根目录运行：

```powershell
npm.cmd ci --prefix pkg/provider/plugins
go test ./pkg/provider -run 'TestOpencode(InstallHooks|UninstallHooks|IsHooksInstalled|Plugin)' -count=1 -timeout 120s
go test ./pkg/daemon -run '^TestWindowsStop' -count=1 -timeout 30s
go test ./cmd -run '^TestInstallPreservesPlatformExecutableName$' -count=1 -timeout 30s
go vet ./...
$env:CONFAB_PORTABLE_BINARY = (Resolve-Path dist/confab_dev_windows_amd64/confab.exe).Path
go test ./test/portable -v -count=1 -timeout 90s
```

冒烟测试只使用临时用户目录、虚构 SQLite 会话和回环地址的认证替身；验证旧插件更新、重复 setup、路径重绑、清空 PATH 后的新会话/恢复会话、本地采集及停止清理。没有连接真实后端，也没有修改实际用户的 OpenCode 配置。Windows CI 另从 ZIP 解压 EXE 执行同一测试。
