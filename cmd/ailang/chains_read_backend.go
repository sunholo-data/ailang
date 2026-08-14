package main

import (
	"context"
	"os"

	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/storage"
)

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
