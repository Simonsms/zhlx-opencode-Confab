//go:build windows

package cmd

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// detachedProcAttr detaches the spawned daemon from the hook's console so it
// keeps running after the hook (and its parent OpenCode process) exits.
// Windows:
//   - CREATE_NEW_PROCESS_GROUP puts the daemon in its own process group so
//     console events (Ctrl+C etc.) targeted at the foreground group never hit
//     it.
//   - DETACHED_PROCESS gives it no console at all; combined with the inbox
//     stop watcher (pkg/daemon watchInboxForStop) this is how shutdown is
//     signaled, since a console-less process can't receive
//     GenerateConsoleCtrlEvent.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
}
