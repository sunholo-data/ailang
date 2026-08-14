package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/storage"
)

var localOnlyChainsSurfaces = map[string]string{
	"chains live":    "Store().DB() raw SQL is not available on observatory.Backend",
	"chains journey": "GetChainJourney is not available on observatory.Backend",
	"chains stats --cost-per-verified-success": "Store is not available on observatory.Backend",
	"chains find --task":                       "GetTaskSpanSummary is not available on observatory.Backend",
	"observatory_*":                            "DB is not available on observatory.Backend",
}

// refuseRemoteReadForLocalOnlySurface prevents an explicitly local-only
// command from silently falling back to SQLite when remote reads are selected.
func refuseRemoteReadForLocalOnlySurface(command, remoteFlag string) error {
	mode := remoteFlag
	if mode == "" {
		mode = os.Getenv("AILANG_CHAINS_READ")
	}
	if mode == "" || storage.Mode(mode) == storage.ModeLocal {
		return nil
	}
	reason, localOnly := localOnlyChainsSurfaces[command]
	if !localOnly {
		return nil
	}
	return fmt.Errorf("%s cannot read remotely: %s; local-only under the D-15 narrowing", command, reason)
}

// openChainsReadBackend resolves the opt-in backend used by Backend-shaped
// chains views. Local remains the default; an explicit flag takes precedence
// over AILANG_CHAINS_READ.
func openChainsReadBackend(ctx context.Context, remoteFlag string) (observatory.Backend, func(), error) {
	mode := remoteFlag
	if mode == "" {
		mode = os.Getenv("AILANG_CHAINS_READ")
	}
	if mode == "" || storage.Mode(mode) == storage.ModeLocal {
		backend, err := observatory.NewSQLiteBackendFromPath(observatory.DefaultDatabasePath())
		if err != nil {
			return nil, func() {}, err
		}
		return backend, func() { _ = backend.Close() }, nil
	}

	backends, err := storage.NewBackendsForMode(ctx, storage.Mode(mode))
	if err != nil {
		return nil, func() {}, err
	}
	return backends.Observatory, func() { _ = backends.Close() }, nil
}
