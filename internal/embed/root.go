package embed

import (
	"fmt"
	"os"
	"path/filepath"
)

// Finding the tree that holds the AILANG modules a Go caller wants to run.
//
// Every caller used to do this itself, and all of them did it as
// `embed.New(".")` — the process working directory. That is silently wrong
// almost everywhere: `ailang budget` run from a user's home directory resolves
// no modules at all, and because each call site then fell back to a Go copy on
// error, the answer changed with the caller's cwd and nothing said so. The
// deployed dashboard image is the extreme case — it contains the binary and no
// `.ail` files whatsoever, so the AILANG path could never once have run there.
//
// Resolution order, most explicit first:
//
//  1. AILANG_PROJECT_ROOT — set it in a container, where there is nothing to
//     search upward for. Verified rather than trusted: a root that does not
//     contain the module is an error, not a silent miss.
//  2. the working directory and its ancestors — the repo case, and stable
//     under `cd` into a subdirectory or a worktree.
//
// A failure to resolve is returned. Callers must not paper over it with a
// second implementation: two copies of one rule drift, and the substitution is
// invisible.

// ProjectRoot returns the directory containing modulePath + ".ail".
//
// modulePath is the AILANG module path, e.g.
// "internal/dashboard_transforms/budget_checker". Passing the module the caller
// actually intends to run means resolution fails loudly when the tree is
// present but that module is missing, rather than succeeding and failing later
// inside Load.
func ProjectRoot(modulePath string) (string, error) {
	rel := filepath.FromSlash(modulePath) + ".ail"

	if root := os.Getenv("AILANG_PROJECT_ROOT"); root != "" {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			return "", fmt.Errorf("AILANG_PROJECT_ROOT=%q does not contain %s: %w", root, rel, err)
		}
		return root, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, rel)); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found in %s or any parent (set AILANG_PROJECT_ROOT)", rel, mustGetwd())
		}
		dir = parent
	}
}

// mustGetwd is only for the error message; a cwd we could not read has already
// been reported above.
func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "the working directory"
	}
	return dir
}

// NewForModule resolves the root for modulePath and returns an Engine with that
// module already loaded.
//
// The pairing matters: an Engine built on an unresolved root is indistinguishable
// from a working one until the first Call fails, which is precisely when a caller
// is most tempted to substitute a Go answer.
func NewForModule(modulePath string) (*Engine, error) {
	root, err := ProjectRoot(modulePath)
	if err != nil {
		return nil, err
	}
	eng := New(root)
	if err := eng.Load(modulePath); err != nil {
		_ = eng.Close()
		return nil, fmt.Errorf("loading %s from %s: %w", modulePath, root, err)
	}
	return eng, nil
}
