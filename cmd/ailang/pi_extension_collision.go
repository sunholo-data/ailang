package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// M-COORDINATOR-EXECUTION-TRUST M6 — the globally installed pi suite must not
// collide with a repo that ships its own copy.
//
// `ailang pi install` materialises the suite into ~/.pi/agent/extensions/ so it
// applies in EVERY workspace. That is deliberate and is what lets the session
// gate govern package repos that never adopted it (D2). It also means that when
// the workspace is the AILANG repo — which carries the same files in
// .pi/extensions/ — pi sees two registrations of the same tool and refuses to
// start:
//
//	Failed to load extension ".../ailang-lsp-lite.ts": Tool "ailang_check"
//	conflicts with /workspace/task-X/.pi/extensions/ailang-lsp-lite.ts
//	exit status 1
//
// Measured on the dev plane 2026-09-02: task-6825b3fc and task-48a365bb both
// died before turn 1; a package-repo task in the same window did not. It takes
// out design-doc-creator, sprint-planner, sprint-evaluator and coordinator on
// the pi provider, and had gone unseen because prod pi tasks target package
// repos.
//
// pi's loader is a third-party package, so the precedence cannot be fixed
// there. It is resolved here instead, and the rule is the defensible one: a
// workspace that ships its own suite is stating a preference, and honouring it
// is strictly more correct than failing. Deliberately NOT a special case for the
// AILANG repo — every package repo that later adds .pi/extensions/ would hit
// the same wall.
//
// Removing from the global dir is safe because the executor container is
// ephemeral and rebuilt per task.
func resolveExtensionCollisions(globalDir, workspaceDir string) ([]string, error) {
	workspaceExtDir := filepath.Join(workspaceDir, ".pi", "extensions")

	entries, err := os.ReadDir(workspaceExtDir)
	if err != nil {
		// The common case: a repo with no .pi/ of its own keeps the whole global
		// suite. Not an error — this runs on every dispatch, so failing hard here
		// would take out the fleet for the normal case.
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workspace extensions %s: %w", workspaceExtDir, err)
	}

	var removed []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		globalCopy := filepath.Join(globalDir, name)
		if _, statErr := os.Stat(globalCopy); statErr != nil {
			continue // no global counterpart, so no collision
		}
		if rmErr := os.Remove(globalCopy); rmErr != nil {
			return removed, fmt.Errorf("remove colliding global extension %s: %w", globalCopy, rmErr)
		}
		removed = append(removed, name)
	}
	return removed, nil
}

// piGlobalExtensionsDir is where `ailang pi install` puts the suite for the
// runtime user. Kept as a function so the executor and the tests agree.
func piGlobalExtensionsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "extensions")
}
