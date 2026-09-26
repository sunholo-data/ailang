package pkg

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestIsAilangBinaryPath pins the predicate that decides WHICH binary
// BuildModuleIface exec's.
//
// Why this file exists: every test that drives BuildModuleIface overrides
// resolveIfaceBinary in TestMain, so replacing that function's whole body with
// an always-failing stub left all six named arms PASSing (measured, iteration
// 313 — reproduced first-party from the sprint evaluator's blocking finding).
// Neither branch was pinned, including the primary one. The naming decision is
// now a pure function and is pinned here directly.
func TestIsAilangBinaryPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/usr/local/bin/ailang", true},
		{"ailang", true},
		{filepath.Join("some", "dir", "ailang"), true},
		{"ailang.exe", true},     // the Windows publisher
		{"/tmp/pkg.test", false}, // the go test binary — the case that matters
		{"/usr/local/bin/ailangx", false},
		{"/usr/local/bin/notailang", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isAilangBinaryPath(tc.path); got != tc.want {
			t.Errorf("isAilangBinaryPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestResolveIfaceBinary_FallsBackToPath exercises the REAL resolver's fallback
// branch: when the running executable is not itself `ailang` (which is always
// true under `go test`, where os.Executable() is `<pkg>.test`), resolution must
// go through exec.LookPath and find the binary on PATH.
//
// Kills: an inverted or short-circuited fallback that returns the test binary,
// and a resolver that ignores PATH entirely.
func TestResolveIfaceBinary_FallsBackToPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub binary permissions differ on windows; the predicate test covers the naming half")
	}

	// Precondition, asserted rather than assumed: under `go test` the running
	// executable must NOT look like ailang, or this test would take the primary
	// branch and prove nothing about the fallback.
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if isAilangBinaryPath(self) {
		t.Fatalf("instrument failure: test binary %q looks like ailang, so the fallback branch is not reachable here", self)
	}

	stubDir := t.TempDir()
	stub := filepath.Join(stubDir, "ailang")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)

	// Call the real implementation, not the TestMain override.
	got, err := realResolveIfaceBinary()
	if err != nil {
		t.Fatalf("resolveIfaceBinary: %v", err)
	}
	if got != stub {
		t.Fatalf("resolved %q, want the stub on PATH %q", got, stub)
	}
}

// TestResolveIfaceBinary_ErrorsWhenAbsent pins the refusal branch: no ailang on
// PATH and a non-ailang executable must produce an error, never a bare path
// that would later exec something arbitrary.
func TestResolveIfaceBinary_ErrorsWhenAbsent(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir: nothing named ailang
	got, err := realResolveIfaceBinary()
	if err == nil {
		t.Fatalf("expected an error when no ailang is on PATH, got %q", got)
	}
}

// TestBuildModuleIface_ExportLimitBoundary pins the exactly-at-limit case.
//
// Kills: flipping `>` to `>=` in the export-limit check. Measured at iteration
// 313 (the evaluator's non-blocking finding, reproduced first-party): that
// mutation reddened NOTHING in the whole package, because only the
// over-the-limit case was exercised.
func TestBuildModuleIface_ExportLimitBoundary(t *testing.T) {
	const modulePath = "test/pkg/boundary"
	dir := writeIfaceFixture(t, modulePath, "module test/pkg/boundary\n\nexport func ok() -> int { 1 }\n")

	lim := DefaultPublishLimits()
	lim.MaxExportedModules = 1 // the fixture declares exactly one exported module

	if _, err := BuildModuleIface(context.Background(), dir, modulePath, lim); err != nil {
		t.Fatalf("exactly-at-limit package must be allowed, got: %v", err)
	}
}
