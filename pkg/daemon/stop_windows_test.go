//go:build windows

package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ConfabulousDev/confab/pkg/logger"
	"github.com/ConfabulousDev/confab/pkg/types"
)

func TestWindowsStopWritesMarkerWithoutPayload(t *testing.T) {
	logger.SetupForTesting(t)
	t.Setenv("USERPROFILE", t.TempDir())
	state := NewStateForProvider("opencode", "ses_stop_marker", "", "", 0)
	if err := state.Save(); err != nil {
		t.Fatal(err)
	}
	if err := StopDaemonForProvider("opencode", state.ExternalID, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(state.InboxPath)
	if err != nil {
		t.Fatalf("stop marker not written: %v", err)
	}
	var event types.InboxEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "session_end" || event.HookInput != nil {
		t.Fatalf("unexpected stop marker: %+v", event)
	}
}

func TestWindowsStopReportsInboxFailure(t *testing.T) {
	logger.SetupForTesting(t)
	t.Setenv("USERPROFILE", t.TempDir())
	state := NewStateForProvider("opencode", "ses_stop_failed", "", "", 0)
	state.InboxPath = filepath.Join(t.TempDir(), "missing", "inbox.jsonl")
	if err := state.Save(); err != nil {
		t.Fatal(err)
	}
	for _, payload := range []*types.ClaudeHookInput{nil, {SessionID: state.ExternalID}} {
		if err := StopDaemonForProvider("opencode", state.ExternalID, payload); err == nil {
			t.Fatal("stop succeeded although Windows could not write its only shutdown signal")
		}
	}
}
