package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var ifaceTestBinary string

func TestMain(m *testing.M) {
	if mode := os.Getenv("AILANG_IFACE_TEST_HELPER"); mode != "" {
		runIfaceTestHelper(mode)
		os.Exit(0)
	}

	tmpDir, err := os.MkdirTemp("", "ailang-iface-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)
	ifaceTestBinary = filepath.Join(tmpDir, "ailang")
	if runtime.GOOS == "windows" {
		ifaceTestBinary += ".exe"
	}
	build := exec.Command("go", "build", "-o", ifaceTestBinary, "./cmd/ailang")
	build.Dir = filepath.Join("..", "..")
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "build test ailang: %v\n%s", buildErr, output)
		os.Exit(1)
	}
	resolveIfaceBinary = func() (string, error) { return ifaceTestBinary, nil }
	os.Exit(m.Run())
}

func runIfaceTestHelper(mode string) {
	switch mode {
	case "fail":
		fmt.Fprintln(os.Stderr, "injected child failure")
		os.Exit(23)
	case "block":
		if marker := os.Getenv("AILANG_IFACE_SURVIVAL_MARKER"); marker != "" {
			child := exec.Command(os.Args[0])
			child.Env = append(os.Environ(), "AILANG_IFACE_TEST_HELPER=grandchild")
			if err := child.Start(); err != nil {
				os.Exit(24)
			}
		}
		if ready := os.Getenv("AILANG_IFACE_READY_MARKER"); ready != "" {
			_ = os.WriteFile(ready, []byte("ready"), 0o600)
		}
		time.Sleep(5 * time.Second)
	case "grandchild":
		time.Sleep(400 * time.Millisecond)
		_ = os.WriteFile(os.Getenv("AILANG_IFACE_SURVIVAL_MARKER"), []byte("survived"), 0o600)
	case "proxy":
		wantDir := os.Getenv("AILANG_IFACE_EXPECT_DIR")
		gotDir, err := os.Getwd()
		if err != nil || gotDir != wantDir {
			fmt.Fprintf(os.Stderr, "child working directory = %q, want %q: %v\n", gotDir, wantDir, err)
			os.Exit(25)
		}
		cmd := exec.Command(os.Getenv("AILANG_IFACE_REAL_BINARY"), os.Args[1:]...)
		cmd.Env = removeEnv(os.Environ(), "AILANG_IFACE_TEST_HELPER")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(26)
		}
	}
}

func TestBuildModuleIface_ReturnsError(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		dir := writeIfaceFixture(t, "test/pkg/missing", "")
		if _, err := BuildModuleIface(context.Background(), dir, "test/pkg/missing", DefaultPublishLimits()); err == nil {
			t.Fatal("BuildModuleIface succeeded for a missing module file")
		}
	})
	t.Run("type error", func(t *testing.T) {
		dir := writeIfaceFixture(t, "test/pkg/broken", "module test/pkg/broken\nexport func broken() -> int = \"wrong\"\n")
		if _, err := BuildModuleIface(context.Background(), dir, "test/pkg/broken", DefaultPublishLimits()); err == nil {
			t.Fatal("BuildModuleIface succeeded for a module with a type error")
		}
	})
	t.Run("non-zero child", func(t *testing.T) {
		dir := writeIfaceFixture(t, "test/pkg/main", validIfaceModule)
		withIfaceHelper(t, "fail", nil)
		if _, err := BuildModuleIface(context.Background(), dir, "test/pkg/main", DefaultPublishLimits()); err == nil || !strings.Contains(err.Error(), "injected child failure") {
			t.Fatalf("BuildModuleIface error = %v, want injected child failure", err)
		}
	})
}

func TestBuildModuleIface_Cancellation(t *testing.T) {
	dir := writeIfaceFixture(t, "test/pkg/main", validIfaceModule)
	ready := filepath.Join(t.TempDir(), "ready")
	survived := filepath.Join(t.TempDir(), "survived")
	withIfaceHelper(t, "block", map[string]string{
		"AILANG_IFACE_READY_MARKER":    ready,
		"AILANG_IFACE_SURVIVAL_MARKER": survived,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := BuildModuleIface(ctx, dir, "test/pkg/main", DefaultPublishLimits())
		done <- err
	}()
	waitForFile(t, ready)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("BuildModuleIface error = %v, want context.Canceled", err)
	}
	if runtime.GOOS != "windows" {
		time.Sleep(500 * time.Millisecond)
		if _, err := os.Stat(survived); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("cancelled subprocess left its process-group child alive: stat error = %v", err)
		}
	}
}

func TestBuildModuleIface_PerModuleDeadline(t *testing.T) {
	defaults := DefaultPublishLimits()
	if defaults.Overall != 60*time.Second || defaults.PerModule != 10*time.Second || defaults.MaxExportedModules != 64 {
		t.Fatalf("DefaultPublishLimits = %+v, want overall=60s per-module=10s max-exports=64", defaults)
	}
	dir := writeIfaceFixture(t, "test/pkg/main", validIfaceModule)
	withIfaceHelper(t, "block", nil)
	lim := DefaultPublishLimits()
	lim.PerModule = 50 * time.Millisecond
	_, err := BuildModuleIface(context.Background(), dir, "test/pkg/main", lim)
	var timeoutErr *ModuleIfaceTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("BuildModuleIface error = %T %v, want *ModuleIfaceTimeoutError", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("BuildModuleIface error = %v, want context.DeadlineExceeded compatibility", err)
	}
}

func TestBuildModuleIface_ExportLimit(t *testing.T) {
	dir := t.TempDir()
	mods := make([]string, 65)
	for i := range mods {
		mods[i] = fmt.Sprintf("\"test/pkg/mod%d\"", i)
	}
	writeFile(t, filepath.Join(dir, "ailang.toml"), manifestFor(strings.Join(mods, ", ")))
	lim := DefaultPublishLimits()
	lim.MaxExportedModules = 64
	_, err := BuildModuleIface(context.Background(), dir, "test/pkg/mod0", lim)
	if err == nil || !strings.Contains(err.Error(), "exceeding limit of 64") {
		t.Fatalf("BuildModuleIface error = %v, want export-limit refusal", err)
	}
}

func TestBuildModuleIface_MatchesInProcess(t *testing.T) {
	dir := writeIfaceFixture(t, "test/pkg/main", validIfaceModule)
	withIfaceHelper(t, "proxy", map[string]string{
		"AILANG_IFACE_EXPECT_DIR":  dir,
		"AILANG_IFACE_REAL_BINARY": ifaceTestBinary,
	})
	got, err := BuildModuleIface(context.Background(), dir, "test/pkg/main", DefaultPublishLimits())
	if err != nil {
		t.Fatalf("BuildModuleIface: %v", err)
	}
	gotBytes, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal BuildModuleIface result: %v", err)
	}
	cmd := exec.Command(ifaceTestBinary, "internal-dump-iface", dir, "test/pkg/main")
	cmd.Dir = dir
	wantBytes, err := cmd.Output()
	if err != nil {
		t.Fatalf("internal-dump-iface: %v", err)
	}
	if !bytes.Equal(gotBytes, bytes.TrimSpace(wantBytes)) {
		t.Fatalf("wrapper JSON differs from canonical subprocess bytes\ngot:\n%s\nwant:\n%s", gotBytes, wantBytes)
	}
}

func TestBuildModuleIface_ModulePathResolution(t *testing.T) {
	dir := writeIfaceFixture(t, "test/pkg/declared", "")
	_, err := BuildModuleIface(context.Background(), dir, "test/pkg/declared", DefaultPublishLimits())
	if err == nil || !strings.Contains(err.Error(), "declared.ail") {
		t.Fatalf("BuildModuleIface error = %v, want missing resolved module file", err)
	}
}

const validIfaceModule = "module test/pkg/main\nexport func answer() -> int = 42\n"

func writeIfaceFixture(t *testing.T, modulePath, source string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ailang.toml"), manifestFor(fmt.Sprintf("%q", modulePath)))
	if source != "" {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(modulePath)+".ail"), source)
	}
	return dir
}

func manifestFor(modules string) string {
	return "[package]\nname = \"test/pkg\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[exports]\nmodules = [" + modules + "]\n"
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func withIfaceHelper(t *testing.T, mode string, extra map[string]string) {
	t.Helper()
	oldResolver := resolveIfaceBinary
	resolveIfaceBinary = func() (string, error) { return os.Args[0], nil }
	t.Cleanup(func() { resolveIfaceBinary = oldResolver })
	t.Setenv("AILANG_IFACE_TEST_HELPER", mode)
	for key, value := range extra {
		t.Setenv(key, value)
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for helper marker %s", path)
}

func removeEnv(env []string, key string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return result
}
