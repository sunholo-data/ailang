package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/storage"
)

// Opening the coordinator store the CLI should actually act on
// (2026-09-04, Mark: "we should put it on the cli as well though").
//
// `ailang coordinator approve|reject|pending` opened a hardcoded local SQLite
// path. On a machine whose real coordinator is Firestore that is not a missing
// feature — it is the CLI confidently operating on the wrong store: `coordinator
// list` returned a stale local task from May while production held live work,
// and an approve would have resolved nothing while reporting success.
//
// That is the same defect class as the provider bug fixed earlier today: a
// value silently defaulted instead of resolved, with no signal that the answer
// was wrong. `ailang chains` already had the remote flag; the approval path did
// not.

// coordinatorStoreBundle is a store plus the collaborators the approval
// processor injects, and a closer.
type coordinatorStoreBundle struct {
	Store      coordinator.Store
	MsgStore   messaging.MessageStore
	ObsBackend observatory.Backend
	Mode       string
	Close      func()
}

// coordinatorPlane resolves the plane the coordinator commands act on:
// --remote when given, else the coordinator store's mode from the ONE plane
// switch (AILANG_STORAGE, or AILANG_STORAGE_COORDINATOR for this store
// alone — the scoped AILANG_COORDINATOR_REMOTE this replaced is a hard error
// naming it). The source is returned so the caller can print it.
func coordinatorPlane(remoteFlag string) (mode string, source config.Source, err error) {
	if remoteFlag != "" {
		if _, err := config.ParsePlane(remoteFlag); err != nil {
			return "", "", err
		}
		return remoteFlag, "--remote", nil
	}
	sel, err := config.StoragePlane()
	if err != nil {
		return "", "", err
	}
	if sel.Coordinator.Source != sel.PlaneSource {
		return string(sel.Coordinator.Mode), sel.Coordinator.Source, nil
	}
	return string(sel.Plane), sel.PlaneSource, nil
}

// openCoordinatorStore resolves which coordinator the command should act on.
//
// Precedence: --remote, then the plane (see coordinatorPlane), then local.
// The mode is RETURNED so every caller can print it: a command that mutates
// approvals must say which plane it is mutating, because "approved" against
// the wrong store looks exactly like success.
func openCoordinatorStore(ctx context.Context, remoteFlag, stateDir string) (*coordinatorStoreBundle, error) {
	mode, _, err := coordinatorPlane(remoteFlag)
	if err != nil {
		return nil, err
	}

	if mode == "" || storage.Mode(mode) == storage.ModeLocal {
		cfg := coordinator.DefaultConfig()
		if stateDir != "" {
			cfg.StateDir = stateDir
		}
		dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
		store, err := coordinator.NewSQLiteStore(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open local coordinator database at %s: %w", dbPath, err)
		}
		return &coordinatorStoreBundle{
			Store: store,
			Mode:  "local (" + dbPath + ")",
			Close: func() { _ = store.Close() },
		}, nil
	}

	// Resolve the project BEFORE opening backends, so the error can say what
	// to set rather than merely what was missing — and hand it to the backends
	// EXPLICITLY. This used to os.Setenv the discovered value for the whole
	// process; the backends now take the project as an argument.
	project, source, err := resolveCloudProject(ctx, mode)
	if err != nil {
		return nil, err
	}

	backends, err := storage.NewBackendsForModeProject(ctx, storage.Mode(mode), project)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s coordinator store: %w", mode, err)
	}
	return &coordinatorStoreBundle{
		Store:      backends.Coordinator,
		MsgStore:   backends.Messaging,
		ObsBackend: backends.Observatory,
		// Report the SOURCE too: a silently-guessed project is the same failure
		// class as a silently-guessed store.
		Mode:  fmt.Sprintf("%s (project %s, via %s)", mode, project, source),
		Close: func() { _ = backends.Close() },
	}, nil
}

// remoteCoordinatorSelected reports whether the caller has named a non-local
// plane, either with --remote or through the environment.
//
// It is a pre-parse check because the local approve path has its own hand-rolled
// argument loop; routing has to happen before that consumes the flags. An
// environment that does not resolve (a retired selector, an unknown value)
// counts as remote so the command reaches openCoordinatorStore and fails with
// the real error instead of silently acting on local SQLite.
func remoteCoordinatorSelected(args []string) bool {
	for i, a := range args {
		if a == "--remote" && i+1 < len(args) {
			return storage.Mode(args[i+1]) != storage.ModeLocal && args[i+1] != ""
		}
		if strings.HasPrefix(a, "--remote=") {
			v := strings.TrimPrefix(a, "--remote=")
			return storage.Mode(v) != storage.ModeLocal && v != ""
		}
	}
	mode, _, err := coordinatorPlane("")
	if err != nil {
		return true
	}
	return mode != "" && storage.Mode(mode) != storage.ModeLocal
}
