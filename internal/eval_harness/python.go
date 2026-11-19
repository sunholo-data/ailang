package eval_harness

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/config"
)

// PinnedPythonVersion is the Python version the eval suite targets. Both the
// runtime (the interpreter `uv` resolves) and the prompt (what we tell the
// model) are driven from this single constant so they can never drift.
//
// 3.14 is the newest stable minor that `uv` resolves to a STABLE build on the
// rig and in CI (2026-11-16). 3.15 is out upstream, but a uv that predates its
// final release silently resolves `--python 3.15` to the 3.15.0b1 beta — so
// moving to 3.15 needs a uv upgrade on every runner first. Changing this pin
// changes the Python prompt and runtime: it starts a new Python baseline.
const (
	PinnedPythonMajor = 3
	PinnedPythonMinor = 14
)

// PinnedPythonVersion returns "3.14".
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

// substituteRuntimePlaceholders fills the runtime tokens a prompt template may
// carry ({{PYTHON_VERSION}}). It is the ONE place this happens, and every live
// path that hands a prompt to a model calls it: the agent system prompt
// (LoadSystemPromptForLanguage), the agent task prompt
// (GenerateAgentPromptsWithSystemPrompt) and standard mode
// (BenchmarkSpec.PromptPartsForLanguage). Until 2026-11-16 only the task
// prompt did, and models were shown a literal {{PYTHON_VERSION}}.
func substituteRuntimePlaceholders(s string) string {
	return strings.ReplaceAll(s, "{{PYTHON_VERSION}}", DetectedPythonVersion())
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
		if env := config.UV(); env != "" {
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
