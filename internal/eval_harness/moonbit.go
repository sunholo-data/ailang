package eval_harness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// MoonBit is installed by the upstream installer at $MOON_HOME/bin (defaulting
// to $HOME/.moon/bin), which is NOT necessarily on PATH for child processes.
// resolveMoon looks for `moon` in PATH first (honoring $AILANG_MOON for
// overrides), then falls back to the standard installer location.

var (
	moonResolveOnce sync.Once
	moonBinPath     string
	moonResolveErr  error
)

// ErrMoonMissing is returned by MoonbitRunner when the `moon` binary is not
// available. Per CLAUDE.md §2 we surface this as a clean "infra unavailable"
// outcome rather than silently substituting another interpreter.
type ErrMoonMissing struct{ cause error }

func (e *ErrMoonMissing) Error() string {
	return "moon (MoonBit CLI) is required to run MoonBit benchmarks but was not found. " +
		"Install with: `curl -fsSL https://cli.moonbitlang.com/install/unix.sh | bash` " +
		"— this puts `moon` in $HOME/.moon/bin. The eval harness will find it there " +
		"automatically, or set AILANG_MOON=/path/to/moon to override."
}
func (e *ErrMoonMissing) Unwrap() error { return e.cause }

// resolveMoon locates the moon binary and caches the path.
func resolveMoon() (string, error) {
	moonResolveOnce.Do(func() {
		if env := strings.TrimSpace(os.Getenv("AILANG_MOON")); env != "" {
			moonBinPath = env
			return
		}
		if path, err := exec.LookPath("moon"); err == nil {
			moonBinPath = path
			return
		}
		// Fallback to the standard installer location.
		moonHome := strings.TrimSpace(os.Getenv("MOON_HOME"))
		if moonHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				moonResolveErr = &ErrMoonMissing{cause: err}
				return
			}
			moonHome = filepath.Join(home, ".moon")
		}
		candidate := filepath.Join(moonHome, "bin", "moon")
		if _, err := os.Stat(candidate); err == nil {
			moonBinPath = candidate
			return
		}
		moonResolveErr = &ErrMoonMissing{cause: fmt.Errorf("not found in PATH or %s", candidate)}
	})
	return moonBinPath, moonResolveErr
}

// newMoonbitCommand builds `moon run <file>.mbt [args...]`. Callers should
// treat a non-nil error as fatal for the MoonBit benchmark.
func newMoonbitCommand(args ...string) (*exec.Cmd, error) {
	moon, err := resolveMoon()
	if err != nil {
		return nil, err
	}
	argv := append([]string{"run"}, args...)
	return exec.Command(moon, argv...), nil
}
