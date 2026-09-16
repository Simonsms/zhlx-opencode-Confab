package portable_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ConfabulousDev/confab/pkg/opencodetest"
	"github.com/ConfabulousDev/confab/pkg/process"
)

// Opt-in because this test exercises a separately built distribution binary.
func TestOpenCodePortable(t *testing.T) {
	binary := os.Getenv("CONFAB_PORTABLE_BINARY")
	if binary == "" {
		t.Skip("set CONFAB_PORTABLE_BINARY to a built confab executable; Bun is also required")
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Fatal("Bun is required for the portable plugin smoke test")
	}
	root := t.TempDir()
	work := filepath.Join(root, "中文 工作区 & $literal")
	binaryDir := filepath.Join(root, "中文 便携目录 & $literal 'quoted'")
	if err := os.MkdirAll(binaryDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(work, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	name := "confab"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	portableBinary := filepath.Join(binaryDir, name)
	if err := os.WriteFile(portableBinary, data, 0700); err != nil {
		t.Fatal(err)
	}
	const session = "ses_portable_smoke"
	db := opencodetest.NewDB(t)
	db.AddSessionWithDir(session, "", work).
		AddMessage(session, "msg_001", opencodetest.UserTextMessage("portable smoke")).
		AddPart("msg_001", "prt_001", opencodetest.TextPart("portable smoke"))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bun, "run", "opencode-smoke.ts", portableBinary, db.Path(), root, work)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log, _ := os.ReadFile(filepath.Join(root, "home", ".confab", "logs", "confab.log"))
		t.Fatalf("portable smoke: %v\n%s\nDaemon log:\n%s", err, out, log)
	}
	var result struct {
		PIDs []int `json:"pids"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("read smoke result: %v\n%s", err, out)
	}
	if len(result.PIDs) != 2 {
		t.Fatalf("expected new-session and resume daemons; got %v", result.PIDs)
	}
	for _, pid := range result.PIDs {
		deadline := time.Now().Add(5 * time.Second)
		for process.IsRunning(pid) && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
		if process.IsRunning(pid) {
			t.Errorf("test daemon %d did not exit after dispose", pid)
		}
	}
	t.Log("verified: setup migration/idempotency, executable rebinding, empty PATH, Chinese/space/special-character paths, SQLite materialization, session resume and graceful stop")
}
