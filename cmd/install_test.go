package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ConfabulousDev/confab/pkg/logger"
)

func TestInstallPreservesPlatformExecutableName(t *testing.T) {
	logger.SetupForTesting(t)
	previous := installDest
	t.Cleanup(func() { installDest = previous })
	installDest = t.TempDir()
	if err := runInstall(nil, nil); err != nil {
		t.Fatal(err)
	}
	name := "confab"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	got, err := os.Stat(filepath.Join(installDest, name))
	if err != nil {
		t.Fatalf("installed executable: %v", err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(executable)
	if err != nil {
		t.Fatal(err)
	}
	if got.Size() != want.Size() {
		t.Fatalf("installed size = %d, want %d", got.Size(), want.Size())
	}
}
