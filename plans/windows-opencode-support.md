# Windows support for OpenCode collection install

This plan records the pre-change baseline and proposed Windows adaptation.
The platform split is now included in this fork; current behavior and validation
limits are documented in `docs/windows.md` and the portable test records.

## Goal

`confab setup --provider opencode` (and the auto-detected bare `setup`) must work on native Windows: compile, install the `confab-sync.ts` plugin, and run the daemon sync lifecycle (collect → materialize → upload, subagent sidechains, session-end shutdown). Other providers (Claude/Codex/Cursor) are out of scope for runtime correctness but must continue to compile on Windows since the whole module builds as one binary.

## Verified current state

`GOOS=windows GOARCH=amd64 go build ./...` fails with exactly one error:

- `cmd/spawn.go:133` — `syscall.SysProcAttr{Setpgid: true}`: the `Setpgid` field does not exist on Windows. Everything else compiles (`os.Getppid`, `syscall.SIGTERM/SIGINT`, `syscall.Signal(0)`, `modernc.org/sqlite` pure-Go driver, file/dir primitives).

Runtime blockers (compile fine, wrong on Windows):

1. `pkg/daemon/state.go` `isProcessRunning`: uses `os.FindProcess` + `process.Signal(syscall.Signal(0))`. Go's Windows `os.Process.Signal` only supports `os.Kill` (everything else returns `EWINDOWS`), so liveness always reports false. Breaks daemon duplicate detection, parent-PID liveness (`monitorParent` would kill a live daemon), `ReapStaleStates` (deletes live state files), and the `StopDaemonForProvider` running gate.
2. `pkg/provider/claude.go` `getProcCmdline` / `getProcName` / `getParentPID`: shell out to `ps`, which does not exist on Windows → `FindParentPID` always 0.
3. `pkg/daemon/daemon.go` `StopDaemonForProvider`: sends `SIGTERM` → on Windows becomes `TerminateProcess` (hard kill; no final sync / state cleanup). Also the daemon is spawned detached so it has no console to receive `GenerateConsoleCtrlEvent`.
4. Daemon spawn detach semantics rely on `Setpgid` (own process group so it outlives the hook process); the Windows equivalent is `CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS`.

Already correct (no change needed):

- OpenCode uses `xdg-basedir` on **all** platforms (sst/opencode `Global.Path` = `join(xdgBase, "opencode")`; it does **not** read `%APPDATA%` / `%LOCALAPPDATA%`). So `PluginDir` (`~/.config/opencode/plugins`, i.e. `C:\Users\<u>\.config\opencode\plugins`) and `OpenCodeDBPath` (`$XDG_DATA_HOME/opencode/opencode.db` → `~/.local/share/opencode/opencode.db`) already resolve correctly on Windows.
- Plugin `echo ${input} | confab hook ...`: runs under Bun's bundled cross-platform shell (not cmd.exe), so pipes should work; needs a live Windows verification (JSON quoting). Mitigation if needed: pass the payload via a temp file / `--bg-daemon` JSON arg (already the daemon's wire format).
- `pkg/provider` child sidechain backend `file_name`s already use `path.Join` (forward slashes).

## Design

Add the repo's first platform-split files. Keep POSIX behavior byte-identical; Windows is additive. `golang.org/x/sys` is already an (indirect) dependency — promote to direct and use `x/sys/windows` for process/console APIs.

### 1. New leaf package `pkg/process` (cross-platform process inspection)

Owns all PID introspection so no package shells out to `ps` anymore.

| Method | POSIX | Windows |
| --- | --- | --- |
| `IsRunning(pid)` | `os.FindProcess` + `Signal(0)` | `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` + `GetExitCodeProcess == STILL_ACTIVE` |
| `Name(pid)` (basename) | `ps -p <pid> -o comm=` → basename | `CreateToolhelp32Snapshot` + `Process32First/Next` `ExeFile` |
| `ParentPID(pid)` | `ps -p <pid> -o ppid=` | same snapshot, `ParentProcessID` |
| `Cmdline(pid)` | `ps -p <pid> -o command=` | best-effort fallback to `Name(pid)` (full command line would require reading the target's PEB; provider `IsProcess` regexes like `\bclaude\b` still match `claude.exe`) |

Files: `process.go` (doc), `process_unix.go` (`//go:build !windows`), `process_windows.go`, `process_test.go` (portable: self PID, negative/bogus PID, basename shape).

### 2. `pkg/provider` — delete `ps` helpers, call `pkg/process`

- Delete `getProcCmdline` / `getProcName` / `getParentPID` from `claude.go`.
- Update callers: `claude.go` (`IsProcess`, `findParentOrGrandparent`), `codex.go`, `cursor.go`, `opencode.go` (`FindParentPID` loop, `IsProcess`).
- Move the `getProcName` basename contract test out of `claude_test.go` into `pkg/process`.

### 3. `pkg/daemon` — platform-split liveness, signals, and stop mechanism

- `state.go`: `isProcessRunning` becomes a one-line delegator to `process.IsRunning` (drop the `syscall` import).
- New `signals_unix.go` / `signals_windows.go`: `daemonSignals() []os.Signal` for `signal.Notify`. Windows list: `{os.Interrupt, syscall.SIGTERM}` (Go's runtime maps `CTRL_BREAK_EVENT` → `SIGINT`, so a console `CTRL_BREAK` stops the daemon; `SIGTERM` covers console-close). POSIX list unchanged `{syscall.SIGTERM, syscall.SIGINT}`.
- `StopDaemonForProvider` stop vector split into `signalProcessStop(pid) error`: POSIX sends `SIGTERM` (unchanged); Windows is a no-op — shutdown is driven by the stop-request watcher (below). The inbox `session_end` event write is platform-neutral and already happens first.
- New Windows-only `watchStopRequests` goroutine started in `Run`: polls `state.InboxPath` (`~/.confab/sync/<provider>/<id>.inbox.jsonl`) at ~500ms and calls `d.Stop()` (→ `stopCh` → graceful `shutdown`, which does final sync + reads the inbox event + deletes files) when it sees a `session_end` line. `state` is saved before the watcher starts. POSIX build compiles this out.
  - Alternative considered and rejected: `GenerateConsoleCtrlEvent(CTRL_BREAK_EVENT)` — requires the daemon to share a console, which contradicts the detached spawn (`DETACHED_PROCESS`).

### 4. `cmd/spawn.go` — detached spawn attributes

- `cmd.SysProcAttr = detachedProcAttr()`.
- `procattr_unix.go`: `&syscall.SysProcAttr{Setpgid: true}` (unchanged).
- `procattr_windows.go`: `&syscall.SysProcAttr{CreationFlags: DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP}` so the daemon outlives the `confab hook session-start` process and gets a clean stop path.

### 5. `pkg/provider/opencode.go` — config-dir parity

- `StateDir`: honor `XDG_CONFIG_HOME` before falling back to `~/.config/opencode` (opencode honors it on all platforms). `CONFAB_OPENCODE_CONFIG_DIR` stays the top-priority override.

### 6. Verification

- `GOOS=windows go build ./...` in CI (cross-compile gate).
- windows-latest CI job running `make test` (tests gated/skipped where a tool is POSIX-only, e.g. `sleep`).
- Manual Windows QA: `confab setup --provider opencode`, start a session, confirm materialized `~/.confab/opencode/<id>/messages.jsonl` + upload + `session-end` graceful shutdown (inbox watcher), resume, subagent sidechains.
- Live-verify the Bun-shell `echo | confab` pipe on Windows; adjust plugin invocation if JSON quoting misbehaves.

### Out of scope / flagged

- `pkg/pathcanon.CanonicalDir` binding keys on case-insensitive Windows filesystems (per-`--config-dir` creds) — documented, not fixed.
- Claude/Codex/Cursor runtime parity on Windows (they still must *compile*).
- GoReleaser Windows release artifacts + `cmd/update.go` artifact-name matching for self-update — follow-up ticket.

## Test changes

- New `pkg/process/process_test.go`: portable self/bogus-PID liveness + basename assertions.
- `pkg/provider/claude_test.go`: `TestGetProcName` removed (coverage moves to `pkg/process`).
- `pkg/daemon/state_test.go`: `TestIsProcessRunning_WithRealSubprocess` gated to non-Windows (`sleep` unavailable), plus a portable self/negative-PID assertion covering the delegator.
- Existing `GOOS` build gate in CI.
