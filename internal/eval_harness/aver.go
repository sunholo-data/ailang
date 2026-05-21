package eval_harness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Aver is installed via `cargo install aver-lang`, which places the `aver`
// binary in $CARGO_HOME/bin (typically $HOME/.cargo/bin). That directory is
// NOT necessarily on PATH for child processes. resolveAver looks for `aver`
// in PATH first (honoring $AILANG_AVER for overrides), then falls back to
// the standard cargo install location.

var (
	averResolveOnce sync.Once
	averBinPath     string
	averResolveErr  error
)

// ErrAverMissing is returned by AverRunner when the `aver` binary cannot be
// located.
type ErrAverMissing struct{ cause error }

func (e *ErrAverMissing) Error() string {
	return "aver (Aver CLI) is required to run Aver benchmarks but was not found. " +
		"Install with: `cargo install aver-lang` — this puts `aver` in $HOME/.cargo/bin. " +
		"The eval harness will find it there automatically, or set " +
		"AILANG_AVER=/path/to/aver to override."
}
func (e *ErrAverMissing) Unwrap() error { return e.cause }

// resolveAver locates the aver binary and caches the path.
func resolveAver() (string, error) {
	averResolveOnce.Do(func() {
		if env := strings.TrimSpace(os.Getenv("AILANG_AVER")); env != "" {
			averBinPath = env
			return
		}
		if path, err := exec.LookPath("aver"); err == nil {
			averBinPath = path
			return
		}
		// Fallback to $CARGO_HOME/bin/aver (typically $HOME/.cargo/bin/aver).
		cargoHome := strings.TrimSpace(os.Getenv("CARGO_HOME"))
		if cargoHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				averResolveErr = &ErrAverMissing{cause: err}
				return
			}
			cargoHome = filepath.Join(home, ".cargo")
		}
		candidate := filepath.Join(cargoHome, "bin", "aver")
		if _, err := os.Stat(candidate); err == nil {
			averBinPath = candidate
			return
		}
		averResolveErr = &ErrAverMissing{cause: fmt.Errorf("not found in PATH or %s", candidate)}
	})
	return averBinPath, averResolveErr
}

// newAverCommand builds `aver run <file>.av [-- args...]`.
func newAverCommand(args ...string) (*exec.Cmd, error) {
	aver, err := resolveAver()
	if err != nil {
		return nil, err
	}
	argv := append([]string{"run"}, args...)
	return exec.Command(aver, argv...), nil
}
