package eval_harness

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// PinnedPythonVersion is the Python version the eval suite targets. Both the
// runtime (the interpreter `uv` resolves) and the prompt (what we tell the
// model) are driven from this single constant so they can never drift.
//
// 3.12 is a modern production Python that supports structural pattern
// matching (`match/case`, PEP 634) — so models are not penalised for
// reaching for idiomatic constructs available in current stable Python.
const (
	PinnedPythonMajor = 3
	PinnedPythonMinor = 12
)

// PinnedPythonVersion returns "3.12".
func PinnedPythonVersion() string {
	return fmt.Sprintf("%d.%d", PinnedPythonMajor, PinnedPythonMinor)
}

// DetectedPythonVersion is retained for prompt substitution so nothing in the
// prompt layer cares whether we're using uv, pyenv, or a raw interpreter.
// With uv managing the runtime we guarantee the pinned version, so this just
// returns the pin.
func DetectedPythonVersion() string {
	return PinnedPythonVersion()
}

// uvResolveOnce guarantees we only probe for uv and log the missing-uv error
// once per process.
var (
	uvResolveOnce sync.Once
	uvBinPath     string
	uvResolveErr  error
)

// ErrUvMissing is returned by Python benchmark runners when the `uv` binary
// is not on PATH. The eval suite depends on uv to pin the Python runtime so
// that every benchmark sees the exact same interpreter on every machine.
// Per CLAUDE.md §2 we fail loudly rather than silently falling back to a
// system `python3`, because a wrong-version fallback is the exact bug we
// are trying to eliminate.
type ErrUvMissing struct{ cause error }

func (e *ErrUvMissing) Error() string {
	return "uv is required to run Python benchmarks but was not found on PATH. " +
		"Install uv (https://docs.astral.sh/uv/getting-started/installation/) " +
		"— on macOS/Linux: `curl -LsSf https://astral.sh/uv/install.sh | sh`. " +
		"uv pins the Python runtime (currently " + PinnedPythonVersion() +
		") so benchmark results are reproducible across machines. " +
		"To override for local debugging, set AILANG_UV=/path/to/uv."
}
func (e *ErrUvMissing) Unwrap() error { return e.cause }

// resolveUv locates the uv binary (honoring $AILANG_UV) and caches the path.
// Returns an ErrUvMissing if uv can't be found.
func resolveUv() (string, error) {
	uvResolveOnce.Do(func() {
		if env := strings.TrimSpace(os.Getenv("AILANG_UV")); env != "" {
			uvBinPath = env
			return
		}
		path, err := exec.LookPath("uv")
		if err != nil {
			uvResolveErr = &ErrUvMissing{cause: err}
			return
		}
		uvBinPath = path
	})
	return uvBinPath, uvResolveErr
}

// newPythonCommand builds `uv run --python <pinned> -- <args...>`. The `--`
// separator keeps uv from misinterpreting flags meant for the script.
//
// Callers should treat a non-nil error as fatal for the Python benchmark and
// surface it to the eval report rather than substituting a different
// interpreter. Returning the error (rather than panicking) lets the harness
// record a clean "infra unavailable" outcome instead of a bogus test failure.
func newPythonCommand(args ...string) (*exec.Cmd, error) {
	uv, err := resolveUv()
	if err != nil {
		return nil, err
	}
	argv := append([]string{"run", "--python", PinnedPythonVersion(), "--"}, args...)
	return exec.Command(uv, argv...), nil
}
