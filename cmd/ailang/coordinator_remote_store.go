package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// openCoordinatorStore resolves which coordinator the command should act on.
//
// Precedence: --remote, then $AILANG_COORDINATOR_REMOTE, then
// $AILANG_STORAGE, then local. The mode is RETURNED so every caller can print
// it: a command that mutates approvals must say which plane it is mutating,
// because "approved" against the wrong store looks exactly like success.
func openCoordinatorStore(ctx context.Context, remoteFlag, stateDir string) (*coordinatorStoreBundle, error) {
	mode := remoteFlag
	if mode == "" {
		mode = os.Getenv("AILANG_COORDINATOR_REMOTE")
	}
	if mode == "" {
		mode = os.Getenv("AILANG_STORAGE")
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

	backends, err := storage.NewBackendsForMode(ctx, storage.Mode(mode))
	if err != nil {
		return nil, fmt.Errorf("failed to open %s coordinator store: %w", mode, err)
	}
	project := os.Getenv("AILANG_CLOUD_PROJECT")
	if project == "" {
		project = "(AILANG_CLOUD_PROJECT unset)"
	}
	return &coordinatorStoreBundle{
		Store:      backends.Coordinator,
		MsgStore:   backends.Messaging,
		ObsBackend: backends.Observatory,
		Mode:       fmt.Sprintf("%s (project %s)", mode, project),
		Close:      func() { _ = backends.Close() },
	}, nil
}

// remoteCoordinatorSelected reports whether the caller has named a non-local
// plane, either with --remote or through the environment.
//
// It is a pre-parse check because the local approve path has its own hand-rolled
// argument loop; routing has to happen before that consumes the flags.
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
	for _, env := range []string{"AILANG_COORDINATOR_REMOTE", "AILANG_STORAGE"} {
		if v := os.Getenv(env); v != "" && storage.Mode(v) != storage.ModeLocal {
			return true
		}
	}
	return false
}
