// Package storage provides a unified backend selector for all AILANG databases.
// It enables pluggable storage backends via the AILANG_STORAGE environment variable:
//   - "local" (default): SQLite databases on local filesystem
//   - "gcp": Firestore (coordinator + messaging) + BigQuery (observatory)
//   - "hybrid": SQLite for coordinator/messaging, BigQuery for observatory
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/statedir"
	fsstore "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// Mode represents the storage backend mode.
type Mode string

const (
	ModeLocal  Mode = "local"
	ModeGCP    Mode = "gcp"
	ModeHybrid Mode = "hybrid"
)

// Backends holds all three database backends used by AILANG services.
type Backends struct {
	Coordinator coordinator.Store
	Messaging   messaging.MessageStore
	Observatory observatory.Backend
}

// Close closes all backends.
func (b *Backends) Close() error {
	var firstErr error
	if b.Coordinator != nil {
		if err := b.Coordinator.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if b.Messaging != nil {
		if err := b.Messaging.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if b.Observatory != nil {
		if err := b.Observatory.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// GetMode reads the AILANG_STORAGE environment variable and returns the storage mode.
// Returns ModeLocal if unset or empty.
func GetMode() Mode {
	mode := Mode(os.Getenv("AILANG_STORAGE"))
	switch mode {
	case ModeLocal, ModeGCP, ModeHybrid:
		return mode
	case "":
		return ModeLocal
	default:
		return mode // Will be caught by NewBackends validation
	}
}

// NewBackends creates all three backends based on the AILANG_STORAGE environment variable.
func NewBackends(ctx context.Context) (*Backends, error) {
	return NewBackendsForMode(ctx, GetMode())
}

// NewBackendsForMode resolves an EXPLICIT mode instead of reading the process-wide
// AILANG_STORAGE. It exists so a caller can open a SECOND set of backends alongside
// its own — the mission loop dual-writes its telemetry to a remote observatory while
// its coordinator and messaging stay local (M-MISSION-LOOP-UNIFIED-TELEMETRY M3).
//
// This is the same resolution NewBackends performs, factored out — deliberately NOT
// a second local/gcp/hybrid selector, since two selectors drift.
func NewBackendsForMode(ctx context.Context, mode Mode) (*Backends, error) {
	switch mode {
	case ModeLocal, "":
		return NewSQLiteBackends()
	case ModeGCP, ModeHybrid:
		project, err := config.CloudProject(ctx)
		if err != nil {
			return nil, fmt.Errorf("AILANG_STORAGE=%s: %w", mode, err)
		}
		return NewBackendsForModeProject(ctx, mode, project)
	default:
		return nil, fmt.Errorf("unknown AILANG_STORAGE mode: %q (valid: local, gcp, hybrid)", mode)
	}
}

// NewBackendsForModeProject is NewBackendsForMode with the cloud project
// supplied by the caller instead of resolved through config.CloudProject. It
// exists for the one caller that discovers the project by a wider search than
// the resolver performs (`ailang coordinator --remote`, which also accepts the
// messaging plane's pin) and used to hand the answer down by mutating the
// process environment. Plumbing it is the honest alternative.
func NewBackendsForModeProject(ctx context.Context, mode Mode, project string) (*Backends, error) {
	switch mode {
	case ModeLocal, "":
		return NewSQLiteBackends()
	case ModeGCP:
		return NewGCPBackendsForProject(ctx, project)
	case ModeHybrid:
		return NewHybridBackendsForProject(ctx, project)
	default:
		return nil, fmt.Errorf("unknown AILANG_STORAGE mode: %q (valid: local, gcp, hybrid)", mode)
	}
}

// NewSQLiteBackends creates all three backends using local SQLite databases
// under statedir.Dir().
func NewSQLiteBackends() (*Backends, error) {
	dir, err := statedir.Dir()
	if err != nil {
		return nil, err
	}

	// Ensure state directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory %s: %w", dir, err)
	}

	// 1. Coordinator store
	coordStore, err := coordinator.NewSQLiteStore(filepath.Join(dir, "coordinator.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to open coordinator store: %w", err)
	}

	// 2. Messaging store
	msgStore, err := messaging.OpenStore(filepath.Join(dir, "collaboration.db"))
	if err != nil {
		coordStore.Close()
		return nil, fmt.Errorf("failed to open messaging store: %w", err)
	}

	// 3. Observatory backend
	obsBackend, err := observatory.NewSQLiteBackendFromPath(filepath.Join(dir, "observatory.db"))
	if err != nil {
		coordStore.Close()
		msgStore.Close()
		return nil, fmt.Errorf("failed to open observatory backend: %w", err)
	}

	return &Backends{
		Coordinator: coordStore,
		Messaging:   msgStore,
		Observatory: obsBackend,
	}, nil
}

// NewGCPBackends creates all three backends using GCP services (Firestore)
// in the project config.CloudProject resolves.
func NewGCPBackends(ctx context.Context) (*Backends, error) {
	project, err := config.CloudProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("AILANG_STORAGE=gcp: %w", err)
	}
	return NewGCPBackendsForProject(ctx, project)
}

// NewGCPBackendsForProject is NewGCPBackends for an explicit project.
func NewGCPBackendsForProject(ctx context.Context, project string) (*Backends, error) {
	if project == "" {
		return nil, fmt.Errorf("AILANG_STORAGE=gcp: %w", config.ErrNoCloudProject)
	}

	// Firestore client (shared by coordinator and messaging)
	fsClient, err := fsstore.NewClientForProject(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client (project: %s): %w", project, err)
	}

	// 1. Coordinator → Firestore
	coordStore := fsstore.NewCoordinatorStore(fsClient)
	coordStore.StartCostSync(ctx)

	// 2. Messaging → Firestore
	msgStore := fsstore.NewMessagingStore(fsClient)

	// 3. Observatory → Firestore
	obsBackend := fsstore.NewObservatoryStore(fsClient)

	return &Backends{
		Coordinator: coordStore,
		Messaging:   msgStore,
		Observatory: obsBackend,
	}, nil
}

// NewHybridBackends creates a hybrid setup: SQLite for coordinator/messaging,
// BigQuery for observatory (analytics scale), in the project
// config.CloudProject resolves.
func NewHybridBackends(ctx context.Context) (*Backends, error) {
	project, err := config.CloudProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("AILANG_STORAGE=hybrid: %w", err)
	}
	return NewHybridBackendsForProject(ctx, project)
}

// NewHybridBackendsForProject is NewHybridBackends for an explicit project.
func NewHybridBackendsForProject(_ context.Context, project string) (*Backends, error) {
	if project == "" {
		return nil, fmt.Errorf("AILANG_STORAGE=hybrid: %w", config.ErrNoCloudProject)
	}

	dir, err := statedir.Dir()
	if err != nil {
		return nil, err
	}

	// Ensure state directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory %s: %w", dir, err)
	}

	// Coordinator and messaging: local SQLite (fast writes)
	coordStore, err := coordinator.NewSQLiteStore(filepath.Join(dir, "coordinator.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to open coordinator store: %w", err)
	}

	msgStore, err := messaging.OpenStore(filepath.Join(dir, "collaboration.db"))
	if err != nil {
		coordStore.Close()
		return nil, fmt.Errorf("failed to open messaging store: %w", err)
	}

	// Observatory: BigQuery (analytics scale)
	// TODO: Implement BigQuery observatory backend (M4)
	// For now, fall back to SQLite
	obsBackend, err := observatory.NewSQLiteBackendFromPath(filepath.Join(dir, "observatory.db"))
	if err != nil {
		coordStore.Close()
		msgStore.Close()
		return nil, fmt.Errorf("failed to open observatory backend: %w", err)
	}

	return &Backends{
		Coordinator: coordStore,
		Messaging:   msgStore,
		Observatory: obsBackend,
	}, nil
}
