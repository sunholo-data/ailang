package pkg

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestResolveIfaceBinary_PinnedWinsOverPath pins the CLI's own binary as the
// identity computer even when it is not named `ailang` and another `ailang`
// sits on PATH. That pair is the 2026-10-19 PUB005 case: a renamed v0.54.15
// build silently hashed with the installed v0.52.0.
//
// Kills: a resolver that ignores the pin, or prefers PATH over it.
func TestResolveIfaceBinary_PinnedWinsOverPath(t *testing.T) {
	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "ailang"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)
	pinned := filepath.Join(t.TempDir(), "ailang-v0.54.15")
	UseIfaceBinary(pinned)
	t.Cleanup(func() { UseIfaceBinary("") })

	got, err := realResolveIfaceBinary()
	if err != nil {
		t.Fatalf("resolveIfaceBinary: %v", err)
	}
	if got != pinned {
		t.Fatalf("resolved %q, want the pinned binary %q", got, pinned)
	}
}

// TestResolveIfaceBinary_FallsBackToPath exercises the REAL resolver's
// unpinned branch (the registry validator's): resolution must go through
// exec.LookPath and find the binary on PATH.
//
// Kills: an inverted or short-circuited fallback that returns the test binary,
// and a resolver that ignores PATH entirely.
func TestResolveIfaceBinary_FallsBackToPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub binary permissions differ on windows; the predicate test covers the naming half")
	}

	// Precondition, asserted rather than assumed: nothing is pinned in a test
	// process, or this test would take the pinned branch and prove nothing.
	if pinnedIfaceBinary != "" {
		t.Fatalf("instrument failure: %q is pinned, so the PATH branch is not reachable here", pinnedIfaceBinary)
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
// PATH and nothing pinned must produce an error, never a bare path
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
