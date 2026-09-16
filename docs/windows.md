# Windows support

Confab runs natively on Windows. This page covers how it behaves there, how to
build the `.exe`, and how to verify it is working after `confab setup`.

## How confab runs on Windows (no service, by design)

`confab setup` does **not** install a Windows service, a scheduled task, or a
startup entry. There is nothing to "start" after setup — that is intentional.

The sync engine is **on-demand, per session**:

1. When a provider session starts (e.g. OpenCode's `session.created` event, or
   a Cursor `sessionStart` hook), the hook/plugin shells out to
   `confab hook session-start`, which spawns a background **sync daemon**.
2. The daemon is spawned detached (`CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS`,
   `cmd/procattr_windows.go`), so it survives the hook process that launched it.
   It uploads transcript chunks incrementally (~every 30s).
3. When the session ends, the hook/plugin runs `confab hook session-end`, which
   appends a `session_end` marker to the daemon's inbox
   (`~/.confab/sync/<provider>/<id>.inbox.jsonl`). Because a detached Windows
   process has no console to receive `SIGTERM`, the daemon runs an inbox watcher
   (`watchInboxForStop`, `pkg/daemon/watchstop_windows.go`) that polls for that
   marker (~500ms), then does a graceful shutdown with a final sync.
4. There is also a **parent-PID liveness backstop**: if the provider process
   exits without a clean `session-end` (crash, task-kill), the daemon notices and
   shuts down. Stale `state`/`inbox` files are reaped by `ReapStaleStates` on the
   next session start.

So you should see a `confab.exe` process **only while a provider session is
active** — and usually none between sessions. Between hook invocations nothing
runs in the background, and there is nothing to uninstall at the OS level beyond
the hooks themselves.

## Getting confab.exe

Official Windows release artifacts and `confab update` self-update for Windows
are a follow-up; for now build the binary yourself.

Cross-compile from Linux/macOS (no Windows toolchain needed — the Windows code
paths are pure Go):

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -ldflags "-X main.date=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" -o confab.exe .
```

Or build natively on Windows:

```powershell
go build -o confab.exe .
```

For Windows on ARM64 use `GOARCH=arm64`. `CGO_ENABLED=0` is safe — the Windows
code (process inspection via Toolhelp32, daemon lifecycle) is pure Go and needs
no linking against a Windows C toolchain.

## OpenCode portable setup (no PATH required)

For OpenCode, keep `confab.exe` in a stable directory and run it directly:

```powershell
.\confab.exe setup --provider opencode --backend-url https://confab.yourcompany.com
```

Setup installs the bundled plugin with the current executable's absolute path.
There is no installer, service registration, or PATH requirement for OpenCode sync.
Restart OpenCode afterwards. Existing plugins are refreshed when their content or
executable binding differs. After moving the binary, run
`.\confab.exe hooks add --provider opencode` from the new directory and restart
OpenCode. See [the portable guide](windows-portable.md) for ZIP distribution,
rollback, and the separate PATH limitation of manually invoked skills such as `/retro`.

## Optional PATH installation

Other providers' hooks and manually invoked skills may still call the bare
command `confab`, requiring its directory on their process's PATH.

Convenient placement:

```powershell
.\confab.exe install        # copies confab.exe to %USERPROFILE%\.local\bin
```

Then add `%USERPROFILE%\.local\bin` to PATH (System Properties → Environment
Variables), open a **new** terminal, and **restart your provider** so the plugin
picks up the PATH change. Verify:

```powershell
confab status
```

## Setup

```powershell
# Full auto-detect (OpenCode, Cursor, etc. on PATH or with state dirs present)
confab setup --backend-url https://confab.yourcompany.com

# OpenCode-only
confab setup --provider opencode --backend-url https://confab.yourcompany.com
```

Setup installs the bundled `/retro` skill and the sync lifecycle wiring for each
detected provider. For OpenCode that is `%USERPROFILE%\.config\opencode\plugins\confab-sync.ts`;
for Cursor it merges `sessionStart`/`sessionEnd` (+ tool-use) hooks into
`%USERPROFILE%\.cursor\hooks.json`.

## Where things live on Windows

Everything confab owns is under `%USERPROFILE%`. OpenCode in particular resolves
its dirs via `xdg-basedir` on **all** platforms — it never reads `%APPDATA%`:

| Path | Purpose |
|------|---------|
| `%USERPROFILE%\.confab\config.json` | Backend URL, API key, redaction settings |
| `%USERPROFILE%\.confab\logs\confab.log` | Operation logs (auto-rotated) |
| `%USERPROFILE%\.confab\sync\<provider>\<id>.json` | Daemon state (PID, paths) |
| `%USERPROFILE%\.confab\sync\<provider>\<id>.inbox.jsonl` | Session-end stop marker |
| `%USERPROFILE%\.confab\opencode\<root>\messages.jsonl` | Materialized OpenCode transcript |
| `%USERPROFILE%\.confab\opencode\<root>\children\<child>\messages.jsonl` | Materialized OpenCode subagent transcripts |
| `%USERPROFILE%\.config\opencode\plugins\confab-sync.ts` | OpenCode lifecycle plugin |
| `%USERPROFILE%\.config\opencode\skills\` | `/retro` skill |
| `%USERPROFILE%\.local\share\opencode\opencode.db` | OpenCode's own session database (read by confab) |
| `%USERPROFILE%\.cursor\hooks.json` | Cursor hooks |
| `%USERPROFILE%\.cursor\skills\` | `/retro` skill for Cursor |

## Verifying it works

```powershell
# Backend connection + hook install status
confab status

# Per-session daemons (empty output between sessions = normal)
confab sync status

# List / upload offline
confab list --provider opencode
confab save --provider opencode <session-id>
```

End-to-end check:

1. `confab status` → connected to your backend.
2. Start an OpenCode session and send a message.
3. Confirm a daemon is alive:
   ```powershell
   Get-Process | Where-Object { $_.ProcessName -like "confab*" }
   ```
4. Confirm the transcript materialized:
   ```powershell
   Get-ChildItem "$env:USERPROFILE\.confab\opencode" -Recurse
   ```
5. Close the session (`dispose` → `session-end`) and confirm the daemon exits and
   its state file under `%USERPROFILE%\.confab\sync\opencode\` is removed.

## Provider status on Windows

- **OpenCode** — local portable smoke coverage includes the installed plugin,
  native daemon start/stop, SQLite materialization and resume with synthetic
  sessions. Real OpenCode UI lifecycle, backend uploads and subagent sidechains
  still require target-environment acceptance; local smoke coverage is not that QA.
- **Cursor** — shares the same file-first daemon + `sessionStart`/`sessionEnd`
  hook lifecycle and is designed to work on Windows.
- **Claude Code / Codex** — compile and reuse the same daemon/process machinery,
  but are not yet QA-verified on Windows.

## Troubleshooting

- **No `confab` process while a session is active.** For OpenCode, check the
  `confabBinary` path in the installed plugin. After moving/upgrading the binary,
  rerun `hooks add --provider opencode` from the target executable and restart
  OpenCode. Initial executable launch failures are reported by the plugin to
  OpenCode's console/logs; a process that never started cannot write confab logs.
  Other providers still require the hook command to resolve through PATH.
- **SmartScreen "Windows protected your PC".** The binary is unsigned; choose
  *More info → Run anyway*, or sign it with your own cert. No admin rights are
  needed to run it — it writes only under `%USERPROFILE%`.
- **A stale daemon from a hard-killed provider.** Kill it from Task
  Manager/`Stop-Process`; the next `session-start` runs `ReapStaleStates`, which
  deletes orphaned `state` + `inbox` files whose PID is no longer alive.
- **`confab sync status` shows stale entries.** Same as above — they are cleaned
  up automatically, or by removing the stale `*.json`/`*.inbox.jsonl` pair under
  `%USERPROFILE%\.confab\sync\<provider>\`.
- **SQLite busy after an OpenCode crash.** The collector opens the DB read-only
  with an incremental query and retries indefinitely with a warn-once-per-minute
  cadence; a transient lock resolves itself.

## Uninstall / cleanup

```powershell
confab hooks remove --provider opencode   # removes the plugin file
confab logout
# Optional: delete local state
Remove-Item -Recurse "$env:USERPROFILE\.confab"
```
