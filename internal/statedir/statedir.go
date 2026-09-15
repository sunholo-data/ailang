package statedir

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnvVar is the override honoured by Dir.
const EnvVar = "AILANG_STATE_DIR"

// Dir returns the per-user state directory: $AILANG_STATE_DIR when set, else
// $HOME/.ailang/state. It does not create the directory.
//
// It errors when neither resolves, rather than returning a relative path:
// see the package comment for why.
func Dir() (string, error) {
	if d := os.Getenv(EnvVar); d != "" {
		return filepath.Clean(d), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("statedir: %s is unset and the home directory is unresolvable: %w", EnvVar, err)
	}
	if home == "" {
		return "", fmt.Errorf("statedir: %s is unset and the home directory is empty", EnvVar)
	}
	return UnderHome(home), nil
}

// Path joins elem beneath Dir. It fails exactly when Dir fails.
func Path(elem ...string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{dir}, elem...)...), nil
}

// UnderHome is the default state directory beneath an EXPLICIT home. It exists
// for code that injects a home for testability (mission.Paths) and must derive
// the same tree the driver scripts use; everything else calls Dir, which is
// what honours AILANG_STATE_DIR.
func UnderHome(home string) string {
	return filepath.Join(home, ".ailang", "state")
}

// Project returns the PROJECT-scoped state directory beneath root — the
// repository's own .ailang/state, which holds tracked artifacts (sprints,
// evaluations) and the project brain — optionally extended by elem. It is
// deliberately not affected by AILANG_STATE_DIR: that variable relocates the
// per-user tree, and a repository's tracked state cannot move with it.
func Project(root string, elem ...string) string {
	return filepath.Join(append([]string{root, ".ailang", "state"}, elem...)...)
}
