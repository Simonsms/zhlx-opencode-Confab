//go:build !windows

package daemon

import (
	"fmt"
	"os"
	"syscall"

	"github.com/ConfabulousDev/confab/pkg/logger"
	"github.com/ConfabulousDev/confab/pkg/types"
)

// signalDaemonStop asks a daemon process to shut down gracefully. POSIX:
// SIGTERM, which the daemon's Run loop turns into shutdown() → final sync →
// state/inbox cleanup.
func signalDaemonStop(state *State, hookInput *types.ClaudeHookInput) error {
	if hookInput != nil && state.InboxPath != "" {
		if err := writeInboxEvent(state.InboxPath, "session_end", hookInput); err != nil {
			// POSIX can still stop via SIGTERM when the optional metadata write fails.
			logger.Warn("Failed to write inbox event: %v", err)
		}
	}
	process, err := os.FindProcess(state.PID)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", state.PID, err)
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}
	return nil
}
