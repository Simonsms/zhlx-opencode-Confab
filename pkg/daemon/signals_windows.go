//go:build windows

package daemon

import (
	"os"
	"syscall"
)

// daemonSignals returns the OS signals the daemon listens for to trigger a
// graceful shutdown.
//
// Windows daemons are spawned detached (DETACHED_PROCESS, no console) and
// normally stop via the inbox stop watcher (watchInboxForStop) rather than a
// signal. os.Interrupt (== SIGINT) is registered anyway so Ctrl+C and
// CTRL_BREAK_EVENT (Go's runtime maps both to SIGINT) stop the daemon when it
// happens to share a console; syscall.SIGTERM covers console-close events
// (runtime maps those to SIGTERM). Signal values are compatible with
// os/signal.Notify; undeliverable ones are simply never received.
func daemonSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
