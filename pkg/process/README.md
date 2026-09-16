# pkg/process

Cross-platform process inspection primitives used by daemon parent-liveness
monitoring and provider parent-PID discovery.

## Files

- `process.go` — exported API + contract docs.
- `process_unix.go` — `//go:build !windows`: `ps`-based name/parent/cmdline and
  signal-0 liveness (byte-for-byte the behavior previously inlined in
  `pkg/provider/claude.go` and `pkg/daemon/state.go`).
- `process_windows.go` — `//go:build windows`: Toolhelp32 snapshot for
  name/parent, `OpenProcess`+`GetExitCodeProcess` for liveness, name fallback
  for cmdline.
- `process_test.go` — portable self/bogus-PID liveness + basename assertions.

## API

| Function | POSIX | Windows |
|----------|-------|---------|
| `IsRunning(pid)` | `os.FindProcess` + `Signal(0)` | `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` + `GetExitCodeProcess == STILL_ACTIVE` |
| `Name(pid)` (basename) | `ps -p <pid> -o comm=` → `filepath.Base` | Toolhelp32 `ExeFile` |
| `ParentPID(pid)` | `ps -p <pid> -o ppid=` | Toolhelp32 `ParentProcessID` |
| `Cmdline(pid)` | `ps -p <pid> -o command=` | `Name(pid)` fallback (full command line would require reading the target's PEB) |

## Design notes

- **Stdlib-first leaf**, like `pkg/confabpath`. No confab imports; Windows uses
  `golang.org/x/sys/windows` (already an indirect dependency, promoted to
  direct).
- **Liveness must not lie**: the Windows implementation reads the exit code
  rather than trusting `os.FindProcess` (which succeeds even for dead PIDs) or
  `os.Process.Signal` (Windows only supports `os.Kill`). A PID <= 0 is never
  running; a missing process is `false`, never an error.
- **`Cmdline` on Windows is best-effort.** Provider `IsProcess` regexes like
  `\bclaude\b` / `\bcodex\b` still match the basename (`claude.exe`), which is
  all a detached-process parent walk needs.
- **Don't cache a Toolhelp32 snapshot** across calls — PIDs are recycled and
  callers want current reality. Each call takes one snapshot.
- Zone of use: `pkg/provider` (parent-PID discovery, `IsProcess`) and
  `pkg/daemon` (`isProcessRunning`, reaper, parent liveness). Do not add
  confab-specific behavior here.

## Extension checklist

Add a new primitive when a caller needs a PID fact (`IsRunning`, `Name`,
`ParentPID`, `Cmdline` cover every current call site). Each primitive needs a
POSIX and a Windows implementation plus a portable test.