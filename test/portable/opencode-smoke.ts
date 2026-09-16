import { $ } from "bun"
import assert from "node:assert/strict"
import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs"
import { join } from "node:path"
import { pathToFileURL } from "node:url"

const [binary, database, root, work] = process.argv.slice(2)
assert(binary && database && root && work, "expected binary, database, root and work arguments")
const home = join(root, "home")
const configDir = join(home, "opencode")
mkdirSync(join(home, ".confab"), { recursive: true })
// 没有上传凭据，因此测试不会连接真实后端；也关闭任意版本的自动更新。
const configPath = join(home, ".confab", "config.json")
writeFileSync(configPath, JSON.stringify({ auto_update: false }))

const environment: Record<string, string> = {}
for (const [key, value] of Object.entries(process.env)) {
  if (value !== undefined && key.toLowerCase() !== "path") environment[key] = value
}
Object.assign(environment, {
  PATH: "",
  HOME: home,
  USERPROFILE: home,
  CONFAB_CONFIG_PATH: configPath,
  CONFAB_OPENCODE_CONFIG_DIR: configDir,
  CONFAB_OPENCODE_DB: database,
  CONFAB_SYNC_INTERVAL_MS: "100",
  CONFAB_SYNC_JITTER_MS: "0",
})
const shell = (template: TemplateStringsArray, ...values: string[]) => $(template, ...values).env(environment)
const pluginPath = join(configDir, "plugins", "confab-sync.ts")
mkdirSync(join(configDir, "plugins"), { recursive: true })
writeFileSync(pluginPath, "// stale PATH-based plugin\n")
// 仅在回环地址模拟认证，验证完整 setup 能更新旧插件且重复运行幂等。
const auth = Bun.serve({
  hostname: "127.0.0.1",
  port: 0,
  fetch(request) {
    assert.equal(new URL(request.url).pathname, "/api/v1/auth/validate")
    return Response.json({ valid: true })
  },
})
try {
  const url = auth.url.toString().replace(/\/$/, "")
  const key = "cfb_portable-smoke-test-only"
  await shell`${binary} setup --provider opencode --backend-url ${url} --api-key ${key}`.quiet()
  const repeat = await shell`${binary} setup --provider opencode --backend-url ${url}`.quiet()
  assert(repeat.stdout.toString().includes("hooks already installed (no changes)"))
  const config = JSON.parse(readFileSync(configPath, "utf8"))
  assert.equal(config.redaction.enabled, true)
} finally {
  auth.stop(true)
  writeFileSync(configPath, JSON.stringify({ auto_update: false }))
}

// 从另一个解压目录重新绑定，验证升级/移动后的 hooks add 路径。
const relocated = join(root, "移动 后目录 & space", "confab.exe")
mkdirSync(join(root, "移动 后目录 & space"), { recursive: true })
copyFileSync(binary, relocated)
await shell`${relocated} hooks add --provider opencode`.quiet()
const source = readFileSync(pluginPath, "utf8")
const binding = source.match(/^const confabBinary = (.+)$/m)
assert(binding, "installed plugin must bind an executable")
assert.equal(JSON.parse(binding[1]), relocated)

const { ConfabSync } = await import(pathToFileURL(pluginPath).href)
const session = "ses_portable_smoke"
const statePath = join(home, ".confab", "sync", "opencode", `${session}.json`)
const inboxPath = join(home, ".confab", "sync", "opencode", `${session}.inbox.jsonl`)
const transcript = join(home, ".confab", "opencode", session, "messages.jsonl")
const pids: number[] = []

async function waitFor(check: () => boolean, description: string) {
  const deadline = Date.now() + 15_000
  while (Date.now() < deadline) {
    if (check()) return
    await Bun.sleep(50)
  }
  throw new Error(`Timed out: ${description}`)
}

for (const resumed of [false, true]) {
  const hooks = await ConfabSync({ $: shell })
  try {
    await hooks.event({ event: resumed
      ? { type: "session.status", properties: { sessionID: session, status: { type: "busy" } } }
      : { type: "session.created", properties: { info: { id: session, directory: work } } },
    })
    await waitFor(() => existsSync(statePath) && existsSync(transcript), "daemon and materialized transcript")
    const state = JSON.parse(readFileSync(statePath, "utf8"))
    assert.equal(state.cwd, work, "JSON cwd must survive the Bun shell pipe")
    assert.equal(state.parent_pid, process.pid)
    assert.equal(state.provider, "opencode")
    assert(state.pid > 0)
    pids.push(state.pid)
  } finally {
    await hooks.dispose()
    await waitFor(() => !existsSync(statePath) && !existsSync(inboxPath), "graceful stop and state cleanup")
  }
  const lines = readFileSync(transcript, "utf8").trim().split("\n")
  assert.equal(lines.length, 1, "resume must not duplicate already materialized messages")
  const message = JSON.parse(lines[0])
  assert.equal(message.info.sessionID, session)
  assert.equal(message.parts[0].text, "portable smoke")
}
console.log(JSON.stringify({ pids }))
