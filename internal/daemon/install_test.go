//go:build darwin
// +build darwin

// install_test.go covers macOS launchd plist installer behavior. The tests
// stub launchctl via the launchctlRun seam, but the file paths
// (~/Library/LaunchAgents) and CLI semantics they assert against are
// genuinely macOS-only — build-tagged here so non-darwin CI matrices
// (Windows, Linux) skip them rather than failing on the unmodelled
// platform. The implementation file install.go is intentionally NOT tagged
// — it still compiles on all platforms but errors at runtime on non-macOS,
// which is the user-facing behaviour we want.

package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeLaunchctl records launchctl invocations for assertions.
type fakeLaunchctl struct {
	commands [][]string
	listOut  string
	listErr  error
}

func (f *fakeLaunchctl) run(args ...string) ([]byte, error) {
	f.commands = append(f.commands, args)
	if len(args) > 0 && args[0] == "list" && f.listOut != "" {
		return []byte(f.listOut), f.listErr
	}
	return nil, nil
}

func withLaunchctl(t *testing.T, l *fakeLaunchctl) func() {
	t.Helper()
	prev := launchctlRun
	launchctlRun = l.run
	return func() { launchctlRun = prev }
}

func tempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func TestInstall_CreatesPlist(t *testing.T) {
	home := tempHome(t)
	l := &fakeLaunchctl{}
	defer withLaunchctl(t, l)()

	if err := Install(InstallOpts{Env: "dev", BinaryPath: "/usr/local/bin/ailang"}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	plist := filepath.Join(home, "Library", "LaunchAgents", "com.sunholo.ailang.daemon.plist")
	data, err := os.ReadFile(plist)
	if err != nil {
		t.Fatalf("plist not created: %v", err)
	}
	for _, want := range []string{"com.sunholo.ailang.daemon", "/usr/local/bin/ailang", "<string>dev</string>", "AILANG_CLOUD_PROJECT", "ailang-multivac-dev"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("plist missing %q", want)
		}
	}
	// load should have been called.
	if len(l.commands) == 0 {
		t.Fatal("expected launchctl load call")
	}
	if l.commands[0][0] != "load" {
		t.Errorf("expected load, got %s", l.commands[0][0])
	}
}

func TestInstall_RefusesOverwriteWithoutForce(t *testing.T) {
	home := tempHome(t)
	defer withLaunchctl(t, &fakeLaunchctl{})()
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(plistDir, 0o755); err != nil {
		t.Fatal(err)
	}
	plist := filepath.Join(plistDir, "com.sunholo.ailang.daemon.plist")
	if err := os.WriteFile(plist, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Install(InstallOpts{Env: "dev", BinaryPath: "/usr/local/bin/ailang"})
	if err == nil {
		t.Fatal("expected error on overwrite without --force")
	}
	if !strings.Contains(err.Error(), "already installed") {
		t.Errorf("unexpected error: %v", err)
	}
	// file should be unchanged
	data, _ := os.ReadFile(plist)
	if string(data) != "existing" {
		t.Errorf("plist was overwritten, got: %q", data)
	}
}

func TestInstall_ForceOverwrites(t *testing.T) {
	home := tempHome(t)
	defer withLaunchctl(t, &fakeLaunchctl{})()
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	_ = os.MkdirAll(plistDir, 0o755)
	plist := filepath.Join(plistDir, "com.sunholo.ailang.daemon.plist")
	_ = os.WriteFile(plist, []byte("existing"), 0o644)

	if err := Install(InstallOpts{Env: "dev", BinaryPath: "/usr/local/bin/ailang", Force: true}); err != nil {
		t.Fatalf("force install failed: %v", err)
	}
	data, _ := os.ReadFile(plist)
	if !strings.Contains(string(data), "com.sunholo.ailang.daemon") {
		t.Errorf("plist not overwritten by --force, got: %q", data[:min(len(data), 80)])
	}
}

func TestInstall_RejectsUnknownEnv(t *testing.T) {
	tempHome(t)
	defer withLaunchctl(t, &fakeLaunchctl{})()
	err := Install(InstallOpts{Env: "bogus", BinaryPath: "/usr/local/bin/ailang"})
	if err == nil {
		t.Fatal("expected error for unknown env")
	}
}

func TestUninstall_RemovesPlist(t *testing.T) {
	home := tempHome(t)
	l := &fakeLaunchctl{}
	defer withLaunchctl(t, l)()
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	_ = os.MkdirAll(plistDir, 0o755)
	plist := filepath.Join(plistDir, "com.sunholo.ailang.daemon.plist")
	_ = os.WriteFile(plist, []byte("dummy"), 0o644)

	if err := Uninstall(); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(plist); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected plist removed, got err=%v", err)
	}
	if len(l.commands) == 0 || l.commands[0][0] != "unload" {
		t.Errorf("expected unload as first launchctl call, got %v", l.commands)
	}
}

func TestUninstall_NoPlistOK(t *testing.T) {
	tempHome(t)
	defer withLaunchctl(t, &fakeLaunchctl{})()
	if err := Uninstall(); err != nil {
		t.Errorf("Uninstall on missing plist should be no-op, got %v", err)
	}
}

func TestStatus_ReportsRunning(t *testing.T) {
	tempHome(t)
	l := &fakeLaunchctl{listOut: "12345\t0\tcom.sunholo.ailang.daemon\n"}
	defer withLaunchctl(t, l)()

	out, err := Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !strings.Contains(out, "running") {
		t.Errorf("expected 'running' in status, got: %q", out)
	}
	if !strings.Contains(out, "com.sunholo.ailang.daemon") {
		t.Errorf("expected daemon label in status, got: %q", out)
	}
}

func TestStatus_ReportsNotInstalled(t *testing.T) {
	tempHome(t)
	l := &fakeLaunchctl{listOut: ""} // empty list = not loaded
	defer withLaunchctl(t, l)()

	out, _ := Status()
	if !strings.Contains(out, "not installed") && !strings.Contains(out, "not running") {
		t.Errorf("expected 'not installed/running' in status, got: %q", out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
