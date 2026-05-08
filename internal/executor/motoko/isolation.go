package motoko

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// tailString returns the last n bytes of s, prefixed with "..." when truncated.
// Used to bound the size of stderr captured from a crashed subprocess so it
// fits in a Result error message without flooding the eval-harness logs.
func tailString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}

// noopCleanup is the cleanup func returned on error paths. Defining it as a
// named var avoids the "empty function literal" lint warning AND lets callers
// `defer cleanup()` unconditionally even when setupTaskCacheDir errored.
var noopCleanup = func() { /* intentionally empty: nothing to clean up on error */ }

// setupTaskCacheDir creates a per-task AILANG cache directory and returns
// (path, cleanup, err). The path is intended to be set as AILANG_CACHE_DIR
// in the spawned motoko subprocess so AILANG's NewCacheStore writes to it
// instead of the shared <projectDir>/.ailang/cache/.
//
// Background (M-MOTOKO-PARALLEL-EXECUTION-ISOLATION v0.18.2): Phase 1
// investigation showed parallel motoko sessions racing on writes to the
// same .gob compile-cache files in MOTOKO_REPO/src/core/.ailang/cache/.
// Last writer wins, partial writes corrupt the file, subsequent reads
// crash before the AILANG runtime initializes — manifesting as dur_s=0
// and 0-byte session JSONLs in the adapter Result.
//
// Cleanup is idempotent and best-effort: if the deferred cleanup runs while
// the subprocess still has open file descriptors in the dir, os.RemoveAll
// will succeed (children are killed on adapter return) — but if a stale
// process lingers, the dir survives until the next tmpdir GC. We don't
// wait/fail on cleanup because that would mask earlier errors.
func setupTaskCacheDir(sessionID string) (path string, cleanup func(), err error) {
	// Sanitize sessionID for use in a path: strip "session_" prefix if
	// present (the adapter sets sessionID = "session_<uuid>"; we only
	// need the unique suffix for filesystem isolation).
	safeID := strings.TrimPrefix(sessionID, "session_")
	if safeID == "" {
		// Defensive: empty sessionID would produce a colliding "motoko-task-/"
		// dir. Caller should always provide one but fail loud if not.
		return "", noopCleanup, fmt.Errorf("setupTaskCacheDir: sessionID is empty; cannot create per-task isolation")
	}

	dir := filepath.Join(os.TempDir(), "motoko-task-"+safeID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", noopCleanup, fmt.Errorf("create per-task cache dir %q: %w", dir, err)
	}

	cleanup = func() {
		// Best-effort cleanup. Errors are non-fatal — the OS will GC tmpdir
		// eventually. If the subprocess crashed mid-flight, file descriptors
		// in this dir may still be open; RemoveAll handles that on POSIX.
		_ = os.RemoveAll(dir)
	}
	return dir, cleanup, nil
}
