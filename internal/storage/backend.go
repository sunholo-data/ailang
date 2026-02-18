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

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
	fsstore "github.com/sunholo/ailang/internal/storage/firestore"
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
	mode := GetMode()
	switch mode {
	case ModeLocal, "":
		return NewSQLiteBackends()
	case ModeGCP:
		return NewGCPBackends(ctx)
	case ModeHybrid:
		return NewHybridBackends(ctx)
	default:
		return nil, fmt.Errorf("unknown AILANG_STORAGE mode: %q (valid: local, gcp, hybrid)", mode)
	}
}

// stateDir returns the AILANG state directory for database files.
func stateDir() string {
	dir := os.Getenv("AILANG_STATE_DIR")
	if dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".ailang", "state")
	}
	return filepath.Join(home, ".ailang", "state")
}

// NewSQLiteBackends creates all three backends using local SQLite databases.
func NewSQLiteBackends() (*Backends, error) {
	dir := stateDir()

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

// NewGCPBackends creates all three backends using GCP services (Firestore).
// Requires GOOGLE_CLOUD_PROJECT to be set.
func NewGCPBackends(ctx context.Context) (*Backends, error) {
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT must be set for AILANG_STORAGE=gcp")
	}

	// Firestore client (shared by coordinator and messaging)
	fsClient, err := fsstore.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client (project: %s): %w", project, err)
	}

	// 1. Coordinator → Firestore
	coordStore := fsstore.NewCoordinatorStore(fsClient)

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
// BigQuery for observatory (analytics scale).
func NewHybridBackends(ctx context.Context) (*Backends, error) {
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT must be set for AILANG_STORAGE=hybrid")
	}

	dir := stateDir()

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
