//go:build !windows

package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// platformIsRunning probes the process with signal 0. Signal 0 performs error
// checking only — it never delivers a signal, so a nil error means the process
// exists. Zombies that haven't been reaped still report running (same
// behavior as the original daemon code).
func platformIsRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

// platformCmdline reads the full command line via `ps -o command=`.
// Errors degrade to "" so callers treat it as "unknown".
func platformCmdline(pid int) string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// platformName reads the process name via `ps -o comm=`. macOS returns the
// full executable path; Linux returns just the basename. Normalize to the
// basename so the Name() contract is identical on every non-Windows platform.
func platformName(pid int) string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return ""
	}
	return filepath.Base(name)
}

// platformParentPID reads the parent PID via `ps -o ppid=`.
func platformParentPID(pid int) int {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "ppid=").Output()
	if err != nil {
		return 0
	}
	ppid, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return ppid
}
