package main

// Opt-in remote reads for `ailang chains` (M-MISSION-LOOP-UNIFIED-TELEMETRY M3).
//
// RATIFIED (Mark, 2026-08-13): local analysis may read cloud, but OPT-IN. The
// local store stays the default so offline nodes remain first-class, and opt-in
// is the reversible choice — if `--remote` turns out to be always-passed,
// flipping the default later is one line with evidence behind it, whereas
// canonical-first is the version you cannot walk back.
//
// Consequence enforced here: `--remote` with no cloud configured is an ERROR,
// never a quiet fall back to the local store. A command that says "remote" and
// silently answers from SQLite would report a mission iteration as absent
// cloud-side when it is merely being read from the wrong place.

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/storage"
)

// remoteFlagUsage is the shared help text, so every command that opts in
// describes the same behaviour.
const remoteFlagUsage = "Read from this node's configured cloud observatory instead of the local store (requires AILANG_STORAGE=gcp)"

// openChainBackend opens the observatory a chains command should READ from.
// The returned closer is always non-nil.
func openChainBackend(ctx context.Context, remote bool) (observatory.Backend, func(), error) {
	if !remote {
		backend, err := observatory.NewSQLiteBackendFromPath(observatory.DefaultDatabasePath())
		if err != nil {
			return nil, func() {}, err
		}
		return backend, func() { _ = backend.Close() }, nil
	}

	if mode := storage.GetMode(); mode != storage.ModeGCP {
		return nil, func() {}, fmt.Errorf("--remote requires a cloud observatory: AILANG_STORAGE is %q (set AILANG_STORAGE=gcp and AILANG_CLOUD_PROJECT); refusing to answer a remote query from the local store", mode)
	}

	backends, err := storage.NewBackends(ctx)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open cloud observatory: %w", err)
	}
	return backends.Observatory, func() { _ = backends.Close() }, nil
}
