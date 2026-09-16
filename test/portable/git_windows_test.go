//go:build windows

package portable_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/ConfabulousDev/confab/pkg/backendtest"
	"github.com/ConfabulousDev/confab/pkg/opencodetest"
	pkgsync "github.com/ConfabulousDev/confab/pkg/sync"
	"golang.org/x/sys/windows"
)

// Runs the ZIP's executable, not a freshly built substitute. Repeated init
// failures exercise the background Git path that previously flashed terminals.
// ProcessStartTrace can observe this test; the daemon PID is logged explicitly.
func TestPackagedBackgroundGit(t *testing.T) {
	binary := os.Getenv("CONFAB_PORTABLE_BINARY")
	if binary == "" {
		t.Skip("set CONFAB_PORTABLE_BINARY to the extracted Windows executable")
	}
	root := t.TempDir()
	work := filepath.Join(root, "git workspace")
	if err := os.MkdirAll(work, 0700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q", "--initial-branch=portable-test"},
		{"remote", "add", "origin", "https://example.invalid/portable.git"},
	} {
		command := exec.Command("git", args...)
		command.Dir = work
		command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("prepare local Git fixture: %v\n%s", err, out)
		}
	}
	const session = "ses_packaged_git_test"
	db := opencodetest.NewDB(t)
	db.AddSessionWithDir(session, "", work).
		AddMessage(session, "msg_001", opencodetest.UserTextMessage("isolated package test")).
		AddPart("msg_001", "prt_001", opencodetest.TextPart("isolated package test"))
	var inits, chunks atomic.Int32
	var gitSeen atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, err := backendtest.ReadRequestBody(r)
		if err != nil {
			t.Errorf("decode local request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch r.URL.Path {
		case "/api/v1/sync/init":
			var request pkgsync.InitRequest
			if err := json.Unmarshal(body, &request); err != nil {
				t.Errorf("decode init: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if request.Metadata != nil && strings.Contains(string(request.Metadata.GitInfo), "https://example.invalid/portable.git") {
				gitSeen.Store(true)
			}
			// Exercise multiple daemon cycles, not only HTTP-client retries.
			if inits.Add(1) <= 6 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"intentional local test retry"}`))
				return
			}
			json.NewEncoder(w).Encode(pkgsync.InitResponse{SessionID: "local-portable-test", Files: map[string]pkgsync.FileState{}})
		case "/api/v1/sync/chunk":
			var request pkgsync.ChunkRequest
			if err := json.Unmarshal(body, &request); err != nil {
				t.Errorf("decode chunk: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if len(request.Lines) != 1 || !strings.Contains(request.Lines[0], "isolated package test") {
				t.Errorf("unexpected synthetic transcript chunk")
			}
			chunks.Add(1)
			json.NewEncoder(w).Encode(pkgsync.ChunkResponse{LastSyncedLine: request.FirstLine + len(request.Lines) - 1})
		case "/api/v1/capabilities":
			json.NewEncoder(w).Encode(pkgsync.Capabilities{})
		default:
			t.Errorf("unexpected local endpoint: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	config, err := json.Marshal(struct {
		BackendURL string `json:"backend_url"`
		APIKey     string `json:"api_key"`
		AutoUpdate bool   `json:"auto_update"`
	}{server.URL, "cfb_isolated-package-test-only", false})
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.json")
	if err := os.WriteFile(configPath, config, 0600); err != nil {
		t.Fatal(err)
	}
	launch, err := json.Marshal(struct {
		Provider   string `json:"provider"`
		ExternalID string `json:"external_id"`
		CWD        string `json:"cwd"`
		ParentPID  int    `json:"parent_pid"`
	}{"opencode", session, work, os.Getpid()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "hook", "session-start", "--provider", "opencode", "--bg-daemon", string(launch))
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	command.Env = append(os.Environ(),
		"USERPROFILE="+root, "HOME="+root,
		"CONFAB_CONFIG_PATH="+configPath, "CONFAB_OPENCODE_DB="+db.Path(),
		"CONFAB_OPENCODE_CONFIG_DIR="+filepath.Join(root, "opencode"),
		"CONFAB_SYNC_INTERVAL_MS=6000", "CONFAB_SYNC_JITTER_MS=0")
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Logf("PACKAGED_DAEMON_PID=%d", command.Process.Pid)
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	waited := false
	defer func() {
		if !waited {
			cancel()
			<-done
		}
	}()
	deadline := time.Now().Add(60 * time.Second)
	for chunks.Load() == 0 && time.Now().Before(deadline) {
		select {
		case err := <-done:
			waited = true
			t.Fatalf("packaged daemon exited before sync: %v\n%s", err, output.String())
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	if chunks.Load() == 0 || !gitSeen.Load() {
		t.Fatalf("background Git/sync not observed: inits=%d chunks=%d git=%v", inits.Load(), chunks.Load(), gitSeen.Load())
	}
	stateDir := filepath.Join(root, ".confab", "sync", "opencode")
	inbox := filepath.Join(stateDir, session+".inbox.jsonl")
	if err := os.WriteFile(inbox, []byte("{\"type\":\"session_end\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		waited = true
		if err != nil {
			t.Fatalf("packaged daemon stop: %v\n%s", err, output.String())
		}
	case <-time.After(8 * time.Second):
		t.Fatal("packaged daemon did not stop")
	}
	if _, err := os.Stat(filepath.Join(stateDir, session+".json")); !os.IsNotExist(err) {
		t.Fatalf("daemon state not cleaned up: %v", err)
	}
	t.Logf("PACKAGED_GIT_RESULT init_attempts=%d chunks=%d git_metadata=true graceful_stop=true", inits.Load(), chunks.Load())
}
