//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// daemonSignals returns the OS signals the daemon listens for to trigger a
// graceful shutdown. POSIX: SIGTERM (hooks/session-end) and SIGINT (Ctrl+C).
func daemonSignals() []os.Signal {
	return []os.Signal{syscall.SIGTERM, syscall.SIGINT}
}
