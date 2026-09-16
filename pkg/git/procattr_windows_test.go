//go:build windows

package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// Invoke the real gitCommand path with a helper executable that reports whether
// Windows attached it to a console. Inspecting flags alone would miss a caller
// that forgot to apply them.
func TestGitCommandWithoutConsole(t *testing.T) {
	if os.Getenv("CONFAB_TEST_GIT_CONSOLE") == "1" {
		var startup windows.StartupInfo
		if err := windows.GetStartupInfo(&startup); err != nil {
			panic(err)
		}
		console, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
		hidden := startup.Flags&windows.STARTF_USESHOWWINDOW != 0 && startup.ShowWindow == windows.SW_HIDE
		fmt.Printf("console=%d hidden=%t", console, hidden)
		os.Exit(0)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "git.exe"), data, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CONFAB_TEST_GIT_CONSOLE", "1")
	out, err := gitCommand(dir, "-test.run=^TestGitCommandWithoutConsole$")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "console=0 hidden=true" {
		t.Fatalf("git child was not started without a console and with hidden startup settings: %s", out)
	}
}
