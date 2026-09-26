// Package storage opens the three AILANG databases — coordinator, messaging,
// observatory — where the ONE plane switch says they live.
//
// Which store lives where is decided in internal/config (StoragePlane):
// AILANG_STORAGE=local|gcp|hybrid, with AILANG_STORAGE_{MESSAGING,
// COORDINATOR,OBSERVATORY}=local|gcp moving one store. This package reads
// no environment itself; it opens what the resolved selection says, one
// store at a time, so a per-store override and a whole plane go through the
// same code (M-V1-SIMPLIFY-S3 M3).
//
//   - local: SQLite under statedir.Dir()
//   - gcp:   Firestore in the cloud project
//   - hybrid: every store in SQLite, on the shared plane (project required)
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

// Mode is a plane value: local, gcp or hybrid. It is config.Plane under the
// name this package's callers have always used.
type Mode = config.Plane

// The planes.
const (
	ModeLocal  = config.PlaneLocal
	ModeGCP    = config.PlaneGCP
	ModeHybrid = config.PlaneHybrid
	// ModeInvalid is what GetMode returns when the environment does not
	// resolve — a retired selector is set, or a value is unknown. It is never
	// equal to ModeLocal, so a caller that branches on "not local" reaches
	// NewBackends, which returns the real error.
	ModeInvalid Mode = "invalid"
)

// Backends holds all three database backends used by AILANG services.
type Backends struct {
	Coordinator coordinator.Store
	Messaging   messaging.MessageStore
	Observatory observatory.Backend

	// Selection is what was opened, per store, with sources — for status
	// lines and logs.
	Selection config.Storage
	// Project is the cloud project the Firestore stores use ("" when none
	// was needed).
	Project string
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

// GetMode returns the resolved plane, or ModeInvalid when the environment
// does not resolve (the error surfaces from NewBackends).
func GetMode() Mode {
	s, err := config.StoragePlane()
	if err != nil {
		return ModeInvalid
	}
	return s.Plane
}

// NewBackends opens all three stores as the environment selects them.
func NewBackends(ctx context.Context) (*Backends, error) {
	sel, err := config.StoragePlane()
	if err != nil {
		return nil, err
	}
	return NewBackendsForSelection(ctx, sel, "")
}

// NewBackendsForMode opens the stores for an EXPLICIT plane, ignoring the
// environment's plane and overrides. It exists so a caller can open a
// SECOND set of backends alongside its own — the mission loop dual-writes
// its telemetry to a remote observatory while its coordinator and messaging
// stay local (M-MISSION-LOOP-UNIFIED-TELEMETRY M3). Same resolution, not a
// second selector.
func NewBackendsForMode(ctx context.Context, mode Mode) (*Backends, error) {
	plane, err := config.ParsePlane(string(mode))
	if err != nil {
		return nil, err
	}
	return NewBackendsForSelection(ctx, config.StorageForPlane(plane), "")
}

// NewBackendsForModeProject is NewBackendsForMode with the cloud project
// supplied by the caller instead of resolved through config.CloudProject —
// for the one caller that discovers the project by a wider search than the
// resolver performs (`ailang coordinator --remote`, which also accepts the
// messaging plane's pin).
func NewBackendsForModeProject(ctx context.Context, mode Mode, project string) (*Backends, error) {
	plane, err := config.ParsePlane(string(mode))
	if err != nil {
		return nil, err
	}
	return NewBackendsForSelection(ctx, config.StorageForPlane(plane), project)
}

// NewBackendsForSelection opens each store where sel says it lives. The
// project is needed when any store is in Firestore or the plane is shared
// (hybrid: the project is what the Pub/Sub publisher needs); an empty
// project is resolved through config.CloudProject, and a plane that needs
// one and has none is config.ErrNoCloudProject.
func NewBackendsForSelection(ctx context.Context, sel config.Storage, project string) (*Backends, error) {
	if sel.AnyGCP() || sel.Shared() {
		if project == "" {
			p, err := config.CloudProject(ctx)
			if err != nil {
				return nil, fmt.Errorf("%s=%s: %w", config.EnvStorage, sel.Plane, err)
			}
			project = p
		}
	}

	b := &Backends{Selection: sel, Project: project}
	var dir string
	if sel.Messaging.Mode == config.StoreLocal || sel.Coordinator.Mode == config.StoreLocal || sel.Observatory.Mode == config.StoreLocal {
		d, err := localStateDir()
		if err != nil {
			return nil, err
		}
		dir = d
	}
	var fsClient *fsstore.Client
	if sel.AnyGCP() {
		c, err := fsstore.NewClientForProject(ctx, project)
		if err != nil {
			return nil, fmt.Errorf("failed to create Firestore client (project: %s): %w", project, err)
		}
		fsClient = c
	}

	// 1. Coordinator
	switch sel.Coordinator.Mode {
	case config.StoreGCP:
		cs := fsstore.NewCoordinatorStore(fsClient)
		cs.StartCostSync(ctx)
		b.Coordinator = cs
	default:
		cs, err := coordinator.NewSQLiteStore(filepath.Join(dir, "coordinator.db"))
		if err != nil {
			return nil, fmt.Errorf("failed to open coordinator store: %w", err)
		}
		b.Coordinator = cs
	}

	// 2. Messaging
	switch sel.Messaging.Mode {
	case config.StoreGCP:
		b.Messaging = fsstore.NewMessagingStore(fsClient)
	default:
		ms, err := messaging.OpenStore(filepath.Join(dir, "collaboration.db"))
		if err != nil {
			_ = b.Coordinator.Close()
			return nil, fmt.Errorf("failed to open messaging store: %w", err)
		}
		b.Messaging = ms
	}

	// 3. Observatory
	switch sel.Observatory.Mode {
	case config.StoreGCP:
		b.Observatory = fsstore.NewObservatoryStore(fsClient)
	default:
		ob, err := observatory.NewSQLiteBackendFromPath(filepath.Join(dir, "observatory.db"))
		if err != nil {
			_ = b.Coordinator.Close()
			_ = b.Messaging.Close()
			return nil, fmt.Errorf("failed to open observatory backend: %w", err)
		}
		b.Observatory = ob
	}
	return b, nil
}

// localStateDir resolves and creates the directory the SQLite stores live in.
func localStateDir() (string, error) {
	dir, err := statedir.Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create state directory %s: %w", dir, err)
	}
	return dir, nil
}

// LocalPath is the SQLite file a store uses under the local plane, or ""
// when no state directory resolves — for status output.
func LocalPath(store config.StoreName) string {
	dir, err := statedir.Dir()
	if err != nil {
		return ""
	}
	switch store {
	case config.StoreMessaging:
		return filepath.Join(dir, "collaboration.db")
	case config.StoreCoordinator:
		return filepath.Join(dir, "coordinator.db")
	default:
		return filepath.Join(dir, "observatory.db")
	}
}

// NewSQLiteBackends opens all three stores in local SQLite under
// statedir.Dir(), whatever the environment says — the migration source.
func NewSQLiteBackends() (*Backends, error) {
	return NewBackendsForSelection(context.Background(), config.StorageForPlane(config.PlaneLocal), "")
}

// NewGCPBackends opens all three stores in Firestore in the project
// config.CloudProject resolves — the migration destination.
func NewGCPBackends(ctx context.Context) (*Backends, error) {
	return NewBackendsForSelection(ctx, config.StorageForPlane(config.PlaneGCP), "")
}

// NewGCPBackendsForProject is NewGCPBackends for an explicit project.
func NewGCPBackendsForProject(ctx context.Context, project string) (*Backends, error) {
	if project == "" {
		return nil, fmt.Errorf("%s=gcp: %w", config.EnvStorage, config.ErrNoCloudProject)
	}
	return NewBackendsForSelection(ctx, config.StorageForPlane(config.PlaneGCP), project)
}

// NewHybridBackends opens every store in SQLite on the shared plane: the
// project config.CloudProject resolves is required even though no store
// uses it here, because hybrid is what the coordinator's Pub/Sub publisher
// and the secret approver key on.
func NewHybridBackends(ctx context.Context) (*Backends, error) {
	return NewBackendsForSelection(ctx, config.StorageForPlane(config.PlaneHybrid), "")
}

// NewHybridBackendsForProject is NewHybridBackends for an explicit project.
func NewHybridBackendsForProject(ctx context.Context, project string) (*Backends, error) {
	if project == "" {
		return nil, fmt.Errorf("%s=hybrid: %w", config.EnvStorage, config.ErrNoCloudProject)
	}
	return NewBackendsForSelection(ctx, config.StorageForPlane(config.PlaneHybrid), project)
}
