package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func pluginBinaryPath(t *testing.T, source string) string {
	t.Helper()
	match := regexp.MustCompile(`(?m)^const confabBinary = (.+)$`).FindStringSubmatch(source)
	if len(match) != 2 {
		t.Fatal("plugin has no explicit executable binding")
	}
	var binary string
	if err := json.Unmarshal([]byte(strings.TrimSpace(match[1])), &binary); err != nil {
		t.Fatalf("decode executable path: %v", err)
	}
	return binary
}

func TestOpencodeInstallHooksBindsCurrentExecutable(t *testing.T) {
	t.Setenv("CONFAB_OPENCODE_CONFIG_DIR", t.TempDir())
	pluginPath, err := (Opencode{}).InstallHooks()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	got := pluginBinaryPath(t, string(source))
	if !filepath.IsAbs(got) || got != want {
		t.Fatalf("plugin binary = %q, want current executable %q", got, want)
	}
}

func TestOpencodePluginPreservesExecutablePath(t *testing.T) {
	for _, executable := range []string{
		`C:\Users\测试 用户\tools & apps\confab.exe`,
		`C:\tools\$dollar` + "`tick" + `\confab.exe`,
		`/opt/quoted "path"/confab`,
		"/opt/line\nbreak/confab",
	} {
		t.Run(executable, func(t *testing.T) {
			got := pluginBinaryPath(t, opencodePluginForExecutable(executable))
			if got != executable {
				t.Fatalf("decoded path = %q, want %q", got, executable)
			}
		})
	}
}

func TestOpencodeIsHooksInstalledRejectsStalePlugin(t *testing.T) {
	t.Setenv("CONFAB_OPENCODE_CONFIG_DIR", t.TempDir())
	p := Opencode{}
	pluginPath, err := p.InstallHooks()
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, stale := range []string{
		"// legacy plugin invoking confab via PATH\n",
		strings.Replace(string(current), "const confabBinary = ", "const oldConfabBinary = ", 1),
	} {
		if err := os.WriteFile(pluginPath, []byte(stale), 0600); err != nil {
			t.Fatal(err)
		}
		installed, err := p.IsHooksInstalled()
		if err != nil || installed {
			t.Fatalf("stale plugin installed = %v, err = %v; want false, nil", installed, err)
		}
		if _, err := p.InstallHooks(); err != nil {
			t.Fatal(err)
		}
		installed, err = p.IsHooksInstalled()
		if err != nil || !installed {
			t.Fatalf("updated plugin installed = %v, err = %v; want true, nil", installed, err)
		}
	}
}
