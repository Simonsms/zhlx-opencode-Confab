package process

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsRunningSelf asserts the current process (and its ancestor chain) is
// reported running — the positive path of the liveness probe.
func TestIsRunningSelf(t *testing.T) {
	if !IsRunning(os.Getpid()) {
		t.Errorf("IsRunning(%d) = false for the current test process; want true", os.Getpid())
	}
}

// TestIsRunningSelfParent asserts the parent of the test runner is alive. On
// Windows the runner's parent may legitimately be gone (e.g. a reaped launcher),
// so only require the snapshot walk to resolve a nonzero PID — liveness of that
// PID is asserted when we could resolve one.
func TestIsRunningSelfParent(t *testing.T) {
	ppid := ParentPID(os.Getpid())
	if ppid <= 0 {
		t.Skipf("ParentPID(%d) = %d; cannot assert ancestor liveness", os.Getpid(), ppid)
	}
	t.Logf("parent of runner = %d", ppid)
	if !IsRunning(ppid) {
		t.Errorf("IsRunning(%d) = false for the test runner's live parent; want true", ppid)
	}
}

// TestIsRunningNeverErrors asserts the negative path (missing/bogus PIDs) is
// false rather than an error, including non-positive PIDs.
func TestIsRunningNeverErrors(t *testing.T) {
	for _, pid := range []int{0, -1, 999999999} {
		if IsRunning(pid) {
			t.Errorf("IsRunning(%d) = true for a non-existent PID; want false", pid)
		}
	}
}

// TestNameSelf asserts Name returns a non-empty executable basename (not a
// full path, not a command line) for the current process, mirroring the old
// pkg/provider getProcName contract.
func TestNameSelf(t *testing.T) {
	name := Name(os.Getpid())
	if name == "" {
		t.Fatalf("Name(%d) returned empty string", os.Getpid())
	}
	if strings.Contains(name, string(filepath.Separator)) {
		t.Errorf("Name(%d) = %q, should be basename not full path", os.Getpid(), name)
	}
	if strings.Contains(name, " ") {
		t.Errorf("Name(%d) = %q, should be just process name not full cmdline", os.Getpid(), name)
	}
}

// TestNameBogus asserts a missing PID yields the empty string, not an error.
func TestNameBogus(t *testing.T) {
	if got := Name(999999999); got != "" {
		t.Errorf("Name(999999999) = %q, want empty string for non-existent PID", got)
	}
}

// TestCmdlineSelf asserts Cmdline resolves to something non-empty for the
// current process (real command line on POSIX; executable name on Windows).
func TestCmdlineSelf(t *testing.T) {
	if got := Cmdline(os.Getpid()); got == "" {
		t.Errorf("Cmdline(%d) returned empty string", os.Getpid())
	}
}
