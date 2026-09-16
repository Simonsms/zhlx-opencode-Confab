// Package process provides cross-platform process inspection primitives used
// by the daemon's parent-liveness monitoring and the providers' parent-PID
// discovery.
//
// The package is a stdlib-first leaf (like pkg/confabpath) so `pkg/provider`
// and `pkg/daemon` need not shell out to `ps` (unavailable on Windows) or
// repeat per-platform PID logic. The Windows implementation uses
// golang.org/x/sys/windows (Toolhelp32 snapshot for names/parents,
// OpenProcess + GetExitCodeProcess for liveness); the non-Windows
// implementation retains the original `ps`-based behavior byte-for-byte.
package process

// IsRunning reports whether a process with the given PID exists. Never errors
// on a missing PID — it returns false instead (like the old signal-0 probe).
// A PID <= 0 is never running.
func IsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	return platformIsRunning(pid)
}

// Name returns the basename of a process's executable (e.g. "claude" from
// "/usr/local/bin/claude" or "opencode.exe"). Empty string for a missing or
// unreadable PID. Normalized to a basename on every platform so callers can
// match consistently (macOS `ps -o comm=` returns a full path).
func Name(pid int) string {
	return platformName(pid)
}

// ParentPID returns the PID of a process's parent. 0 for a missing or
// unreadable PID.
func ParentPID(pid int) int {
	return platformParentPID(pid)
}

// Cmdline returns a process's command line (as reported by `ps -o command=`)
// or empty string when unavailable. On Windows the full command line of an
// arbitrary process is not cheaply readable; the executable basename is
// returned instead so provider process-name regexes (e.g. `\bclaude\b`)
// still match.
func Cmdline(pid int) string {
	return platformCmdline(pid)
}
