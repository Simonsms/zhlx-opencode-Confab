//go:build !windows

package daemon

import "context"

// watchInboxForStop is a no-op on POSIX: the daemon stops via SIGTERM/SIGINT
// (daemonSignals) delivered by signalDaemonStop / the terminal. The inbox
// session_end event is still written by StopDaemonForProvider for the final-
// sync payload; nothing needs to poll it here.
func (d *Daemon) watchInboxForStop(ctx context.Context) {
	<-ctx.Done()
}
