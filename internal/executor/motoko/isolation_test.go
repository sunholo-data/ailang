package motoko

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSetupTaskCacheDir_CreatesUniqueDir covers M-MOTOKO-PARALLEL-EXECUTION-
// ISOLATION (v0.18.2) M2c: each call with a distinct sessionID returns a
// distinct path under TMPDIR. This is the load-bearing invariant that lets
// parallel motoko sessions write to non-overlapping AILANG_CACHE_DIRs.
func TestSetupTaskCacheDir_CreatesUniqueDir(t *testing.T) {
	pathA, cleanupA, err := setupTaskCacheDir("session_aaa-111")
	if err != nil {
		t.Fatalf("setupTaskCacheDir A: %v", err)
	}
	defer cleanupA()

	pathB, cleanupB, err := setupTaskCacheDir("session_bbb-222")
	if err != nil {
		t.Fatalf("setupTaskCacheDir B: %v", err)
	}
	defer cleanupB()

	if pathA == pathB {
		t.Errorf("paths collide: %q == %q (each session must get a unique dir)", pathA, pathB)
	}
	if !strings.Contains(pathA, "aaa-111") {
		t.Errorf("path %q should contain sessionID suffix aaa-111", pathA)
	}
	if !strings.Contains(pathB, "bbb-222") {
		t.Errorf("path %q should contain sessionID suffix bbb-222", pathB)
	}
	// Both dirs should actually exist on disk.
	if _, err := os.Stat(pathA); err != nil {
		t.Errorf("dir A not created: %v", err)
	}
	if _, err := os.Stat(pathB); err != nil {
		t.Errorf("dir B not created: %v", err)
	}
}

// TestSetupTaskCacheDir_CleanupRemovesDir verifies the deferred cleanup
// actually removes the temp dir from disk. Without this, parallel runs
// would leak ~50KB-1MB of cache per task, accumulating until tmpdir GC.
func TestSetupTaskCacheDir_CleanupRemovesDir(t *testing.T) {
	path, cleanup, err := setupTaskCacheDir("session_cleanup-test")
	if err != nil {
		t.Fatalf("setupTaskCacheDir: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("dir not created: %v", err)
	}

	// Write a file inside the dir to simulate AILANG cache contents.
	dummy := filepath.Join(path, "compile", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(dummy), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(dummy, []byte(`{"version":"v1","entries":{}}`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected dir removed; got err=%v", err)
	}
}

// TestSetupTaskCacheDir_RejectsEmptySessionID guards against a colliding
// "motoko-task-" dir name when the caller fails to populate sessionID.
// Returns an error rather than silently creating a shared dir (which would
// re-introduce the v0.18.2 race we're fixing).
func TestSetupTaskCacheDir_RejectsEmptySessionID(t *testing.T) {
	_, cleanup, err := setupTaskCacheDir("")
	defer cleanup() // safe even on error path
	if err == nil {
		t.Error("expected error for empty sessionID, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "sessionID is empty") {
		t.Errorf("error should mention empty sessionID; got: %v", err)
	}
}

// TestSetupTaskCacheDir_AcceptsBareSessionID verifies the function works
// when callers pass a sessionID WITHOUT the "session_" prefix (defensive
// — some upstream code paths may strip the prefix before calling).
func TestSetupTaskCacheDir_AcceptsBareSessionID(t *testing.T) {
	path, cleanup, err := setupTaskCacheDir("bare-id-no-prefix")
	defer cleanup()
	if err != nil {
		t.Fatalf("setupTaskCacheDir: %v", err)
	}
	if !strings.Contains(path, "bare-id-no-prefix") {
		t.Errorf("path %q should contain the bare sessionID", path)
	}
}
