//go:build windows

package daemon

import "github.com/ConfabulousDev/confab/pkg/types"

// Windows detached processes have no console for SIGTERM. The inbox marker is
// the shutdown signal itself: it is mandatory even for OpenCode's nil payload,
// and a failed write must propagate instead of reporting a successful stop.
func signalDaemonStop(state *State, hookInput *types.ClaudeHookInput) error {
	return writeInboxEvent(state.InboxPath, "session_end", hookInput)
}
