package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("sandbox path tests use POSIX-style paths and are not applicable on Windows")
	}
}

func TestResolveSandboxPathCheck_Relative(t *testing.T) {
	skipOnWindows(t)
	sandbox := "/tmp/sandbox"
	got, err := resolveSandboxPathCheck(sandbox, "config.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(sandbox, "config.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveSandboxPathCheck_AbsoluteWithin(t *testing.T) {
	skipOnWindows(t)
	sandbox := "/tmp/sandbox"
	path := "/tmp/sandbox/subdir/file.txt"
	got, err := resolveSandboxPathCheck(sandbox, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

func TestResolveSandboxPathCheck_AbsoluteEscapes(t *testing.T) {
	skipOnWindows(t)
	sandbox := "/tmp/sandbox"
	_, err := resolveSandboxPathCheck(sandbox, "/etc/passwd")
	if err == nil {
		t.Error("expected error for path escaping sandbox, got nil")
	}
}

func TestResolveSandboxPathCheck_SandboxRootExact(t *testing.T) {
	skipOnWindows(t)
	sandbox := "/tmp/sandbox"
	got, err := resolveSandboxPathCheck(sandbox, "/tmp/sandbox")
	if err != nil {
		t.Fatalf("unexpected error for exact sandbox root: %v", err)
	}
	if got != sandbox {
		t.Errorf("got %q, want %q", got, sandbox)
	}
}

func TestResolveSandboxPathCheck_TraversalAttempt(t *testing.T) {
	skipOnWindows(t)
	sandbox := "/tmp/sandbox"
	_, err := resolveSandboxPathCheck(sandbox, "/tmp/sandbox/../other")
	if err == nil {
		t.Error("expected error for path traversal attempt, got nil")
	}
}
