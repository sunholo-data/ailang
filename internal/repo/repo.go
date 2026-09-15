// Package repo finds the root of the project the process is running in by
// walking up from the working directory until a marker file or directory
// appears. It is a LEAF (stdlib only) because both language-core packages
// (internal/module) and language roots (internal/prompt) need it and neither
// may import the other; each used to carry its own copy of the walk
// (M-V1-SIMPLIFY-S4 M3B).
//
// The walk is the shared part. WHICH markers mean "root" is the caller's
// policy — the module resolver looks for go.mod/.git/ailang.yaml/.ailang, the
// prompt loader for go.mod/.git/prompts — so markers are an argument, never a
// package default.
package repo

import (
	"os"
	"path/filepath"
)

// FindRoot returns the nearest ancestor of the working directory (including
// itself) that contains any of markers, and true. When no ancestor does, it
// returns the working directory itself and false, so a caller that only
// wants a best-effort base path can ignore the bool. When the working
// directory cannot be determined at all it returns "." and false.
func FindRoot(markers ...string) (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return ".", false
	}
	cwd := dir
	for {
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd, false
		}
		dir = parent
	}
}
