package main

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/config"
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

// chainsReadMode resolves where Backend-shaped chains views read from:
// --remote when given, else the observatory store's mode from the ONE plane
// switch (AILANG_STORAGE, or AILANG_STORAGE_OBSERVATORY for this store alone
// — the scoped selector this replaced is a hard error naming it).
// "" means local.
func chainsReadMode(remoteFlag string) (string, error) {
	if remoteFlag != "" {
		if _, err := config.ParsePlane(remoteFlag); err != nil {
			return "", err
		}
		return remoteFlag, nil
	}
	sel, err := config.StoragePlane()
	if err != nil {
		return "", err
	}
	if sel.Observatory.Mode == config.StoreGCP {
		return string(config.StoreGCP), nil
	}
	return "", nil
}

// refuseRemoteReadForLocalOnlySurface prevents an explicitly local-only
// command from silently falling back to SQLite when remote reads are selected.
func refuseRemoteReadForLocalOnlySurface(command, remoteFlag string) error {
	mode, err := chainsReadMode(remoteFlag)
	if err != nil {
		return err
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

// openChainsReadBackend resolves the backend used by Backend-shaped chains
// views. Local remains the default; an explicit flag takes precedence over
// the plane's observatory store.
func openChainsReadBackend(ctx context.Context, remoteFlag string) (observatory.Backend, func(), error) {
	mode, err := chainsReadMode(remoteFlag)
	if err != nil {
		return nil, func() {}, err
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
