//go:build windows

package daemon

import (
	"bytes"
	"context"
	"os"
	"time"

	"github.com/ConfabulousDev/confab/pkg/logger"
)

// inboxStopPollInterval is how often the Windows stop watcher checks the
// inbox for a session_end stop request. Short enough that session-end feels
// snappy; long enough that the poll is a trivial read on a tiny file.
const inboxStopPollInterval = 500 * time.Millisecond

// watchInboxForStop drives graceful shutdown on Windows, where the detached
// daemon cannot receive console signals. Every inboxStopPollInterval it
// checks the session's inbox file for a session_end stop request written by
// signalDaemonStop (the marker may omit the hook payload, as OpenCode does).
// When found, closes stopCh → the Run loop runs shutdown().
//
// The inbox path is derived from (provider, externalID) directly rather than
// d.state so the watcher can start before the state file is saved (Run's
// waitForTranscript window). A stop request arriving there still stops the
// daemon.
func (d *Daemon) watchInboxForStop(ctx context.Context) {
	ticker := time.NewTicker(inboxStopPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if d.inboxHasSessionEnd() {
				logger.Info("Stop request observed in inbox (session_end); signaling shutdown")
				d.Stop()
				return
			}
		}
	}
}

// inboxHasSessionEnd reports whether the session's inbox contains a
// session_end event. The inbox file is written by signalDaemonStop,
// so presence of a session_end line means a stop
// was requested. Missing/unreadable file → false (fail-open on transient
// writes: the next poll re-checks).
func (d *Daemon) inboxHasSessionEnd() bool {
	inboxPath, err := GetInboxPathForProvider(d.providerName, d.externalID)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(inboxPath)
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte(`"session_end"`))
}
