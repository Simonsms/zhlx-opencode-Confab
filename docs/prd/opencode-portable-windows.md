# OpenCode Windows 免安装接入

## 问题

当前 Windows 改造提供了进程生命周期支持，但 OpenCode 插件仍调用 PATH 中的 `confab`，用户误以为必须执行安装脚本。安装命令还会丢失 `.exe` 后缀。

## 已确认范围

用户已同意先交付免安装 ZIP，并选择只做本地验证。保留 Go 采集、脱敏、增量上传引擎；使用现有一次性认证和 setup 流程。此次不发布 npm 包、不连接真实后端、不改用户机器上的 OpenCode 配置和持久环境变量。

## 用户故事

1. 用户解压 ZIP 后可直接运行 EXE，无需管理员权限和安装脚本。
2. 用户执行一次 OpenCode setup 后，插件调用该 EXE 的绝对路径，不依赖 PATH。
3. 中文、空格、特殊字符路径下，启动和停止命令仍传递正确的 JSON。
4. 旧插件或移动后的 EXE 可通过 setup / hooks add 重新绑定。
5. 开发者可重复构建 Windows ZIP，并在本地隔离环境中验证插件与 EXE 链路。
6. 使用可选 install 命令的用户仍获得正确的 `.exe` 文件名。

## 实现决策

- 插件源码编译时嵌入 Go 程序，避免重复维护两份 TypeScript。
- setup / hooks add 将当前可执行文件绝对路径以 JSON 字符串写入插件。运行时不回退到另一份 PATH 二进制。
- setup 检查插件是否与当前版本和 EXE 路径一致；旧版或旧路径触发更新。
- 分发 ZIP 内含 EXE、使用说明和许可证；EXE 自带插件。解压目录应保持稳定，移动后重新绑定并重启 OpenCode。
- 同步协议、SQLite 采集和后端认证沿用现有模块。
- 本地冒烟测试发现 Windows 无 payload 的 session-end 没有写入停止标记；停止函数必须始终写入标记，并上报写入错误，POSIX 信号行为保持不变。
- 构建脚本供开发者生成 ZIP，不是用户安装前置步骤。

## 验收

- Go 测试覆盖绝对路径生成、特殊字符编码、旧插件升级和幂等行为。
- 现有插件生命周期测试和 TypeScript 检查通过。
- Windows 原生构建成功；隔离环境中 PATH 无 confab 时可通过插件启动及停止真实 EXE。
- ZIP 可解压运行，并附 SHA256 校验文件。
- 所有测试使用虚构会话和临时配置；真实 OpenCode + 后端上传明确标记未验证。

## 不在范围

纯 TypeScript 同步引擎、npm 发布、Windows 自动更新、其他 provider 的免 PATH 接入，以及生产环境部署。
