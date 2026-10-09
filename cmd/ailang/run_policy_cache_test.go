package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/policy"
)

func TestWorkerCacheEnvironment(t *testing.T) {
	res := &policy.Resolved{Mode: policy.ModeRestricted}
	for _, value := range []string{"", "operator-cache"} {
		t.Setenv("AILANG_CACHE_DIR", value)
		got := workerEnv(res, "private-cache")
		want := value
		if want == "" {
			want = "private-cache"
		}
		count := 0
		for _, item := range got {
			if strings.HasPrefix(item, "AILANG_CACHE_DIR=") {
				count++
				if item != "AILANG_CACHE_DIR="+want {
					t.Fatal(item)
				}
			}
		}
		if count != 1 {
			t.Fatalf("cache entries: %v", got)
		}
	}
}

func TestPolicyCachePlacement(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skip(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{relative, root, filepath.Join(root, "missing"), filepath.Join(alias, "missing", "leaf")} {
		if !cacheInsideSandbox(root, path) {
			t.Fatalf("missed inside path %s", path)
		}
	}
	if cacheInsideSandbox(root, t.TempDir()) {
		t.Fatal("outside classified inside")
	}
}

func TestPolicyCacheStartFailureCleanup(t *testing.T) {
	skipWithoutRestrictedMode(t)
	base := t.TempDir()
	t.Setenv("TMPDIR", base)
	t.Setenv("AILANG_CACHE_DIR", "")
	res := &policy.Resolved{Mode: policy.ModeRestricted, Root: t.TempDir()}
	_, err := startPolicyWorker(exec.Command(filepath.Join(base, "missing-executable")), res)
	if err == nil {
		t.Fatal("expected start failure")
	}
	entries, _ := os.ReadDir(base)
	if len(entries) != 0 {
		t.Fatalf("leaked cache: %v", entries)
	}
	res.Root = base
	_, err = startPolicyWorker(exec.Command("unused"), res)
	if err == nil || !strings.Contains(err.Error(), "inside fs_sandbox") {
		t.Fatalf("unsafe placement: %v", err)
	}
	entries, _ = os.ReadDir(base)
	if len(entries) != 0 {
		t.Fatalf("leaked refused cache: %v", entries)
	}
	t.Setenv("TMPDIR", filepath.Join(base, "missing"))
	if _, err = startPolicyWorker(exec.Command("unused"), res); err == nil {
		t.Fatal("invalid temporary base accepted")
	}
}

func TestWorkerCacheTrustedEnvironment(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", "")
	t.Setenv("CACHE_TEST_SENTINEL", "host")
	env := workerEnv(&policy.Resolved{Mode: policy.ModeTrustedHost}, "ignored-default")
	if strings.Join(env, "\n") != strings.Join(os.Environ(), "\n") {
		t.Fatal("trusted environment changed")
	}
}

func TestPolicyCachePrivateAndOperatorOwned(t *testing.T) {
	skipWithoutRestrictedMode(t)
	base := t.TempDir()
	t.Setenv("TMPDIR", base)
	t.Setenv("AILANG_CACHE_DIR", "")
	res := &policy.Resolved{Mode: policy.ModeRestricted, Root: t.TempDir()}
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cache, err := startPolicyWorker(cmd, res)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cache)
	info, err := os.Stat(cache)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("private directory: %v, %v", info, err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	operator := t.TempDir()
	t.Setenv("AILANG_CACHE_DIR", operator)
	if _, err := startPolicyWorker(exec.Command(filepath.Join(base, "missing")), res); err == nil {
		t.Fatal("expected start failure")
	}
	if _, err := os.Stat(operator); err != nil {
		t.Fatalf("operator cache removed: %v", err)
	}
}

func TestPolicyCacheWithoutSandbox(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if cacheInsideSandbox("", cwd) {
		t.Fatal("absent sandbox treated as cwd")
	}
}
