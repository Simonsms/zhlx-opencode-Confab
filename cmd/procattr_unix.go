//go:build !windows

package cmd

import "syscall"

// detachedProcAttr detaches the spawned daemon from the hook's process group
// so it keeps running after the hook (and its parent Claude/Codex/OpenCode
// process) exits. POSIX: Setpgid puts the daemon in its own process group.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
