package motoko

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/executor"
)

// D1 provider-preflight refusal-branch matrix (sprint plan §3.2). The
// mechanism's value set is ai.EnvVarForProvider's switch over providers, so
// these tests are TABLES over (model → expected env var), not booleans. Every
// row names the mutation that would kill it, and asserts on a string UNIQUE to
// the branch (the resolved provider + missing variable, never just "an error
// was returned").
//
// Requirement (earned by D1's whole point): a row with a provider that needs
// NO credential (ollama), asserted to PASS with every credential env var
// cleared. That is the one row the deleted unconditional refusal could not
// make pass — it is the row the original bug fails and this matrix guards.

// preflightCases: model -> expected env var ("" = no credential required).
// VERIFY these against internal/ai/config.go GuessProvider + EnvVarForProvider
// on 2026-08-20:
//
//	ollama/qwen3.6:35b-a3b-mxfp8  -> "ollama:" prefix     -> provider ollama -> env ""
//	ollama:qwen3.5                -> "ollama:" prefix     -> provider ollama -> env ""
//	openrouter/anthropic/claude-haiku-4-5 -> vendor/model -> openrouter -> OPENROUTER_API_KEY
//	anthropic/claude-sonnet-4-6            -> vendor/model -> openrouter (NOT direct anthropic) -> OPENROUTER_API_KEY
//	claude-3-5-sonnet             -> bare claude- prefix -> anthropic -> ANTHROPIC_API_KEY
//	gpt-5-codex                   -> bare gpt- prefix    -> openai    -> OPENAI_API_KEY
//	gemini-3-1-pro                -> bare gemini- prefix -> google    -> GOOGLE_API_KEY
var preflightCases = []struct {
	model    string
	envVar   string
	provider string
}{
	{"ollama/qwen3.6:35b-a3b-mvp", "", "ollama"},
	{"ollama:qwen3.5", "", "ollama"},
	{"openrouter/anthropic/claude-haiku-4-5", "OPENROUTER_API_KEY", "openrouter"},
	{"anthropic/claude-sonnet-4-6", "OPENROUTER_API_KEY", "openrouter"},
	{"claude-3-5-sonnet", "ANTHROPIC_API_KEY", "anthropic"},
	{"gpt-5-codex", "OPENAI_API_KEY", "openai"},
	{"gemini-3-1-pro", "GOOGLE_API_KEY", "google"},
}

// TestRequireProviderCredential_NoKeyRefusalMatrix (T-B5) is the primary
// table: each row with its env var cleared must either pass (no env required)
// or refuse naming the model AND the specific missing variable.
func TestRequireProviderCredential_Row_KeyedRefusalTable(t *testing.T) {
	for _, c := range preflightCases {
		envVar := c.envVar
		if envVar != "" {
			t.Setenv(envVar, "")
		}
		err := requireProviderCredential(c.model)
		if envVar == "" {
			if err != nil {
				t.Errorf("model %q needs no credential; requireProviderCredential refused: %v", c.model, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("model %q requires %s (unset) but requireProviderCredential passed", c.model, envVar)
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, c.model) {
			t.Errorf("model %q refusal does not name the model; got: %q", c.model, msg)
		}
		if !strings.Contains(msg, envVar) {
			t.Errorf("model %q refusal does not name the missing variable %s; got: %q", c.model, envVar, msg)
		}
	}
}

// TestRequireProviderCredential_KeyPresent_T_B5b prohibits the naive
// "any API key is set" mutant: setting the CORRECT variable for a keyed model
// must admit, but setting a DIFFERENT provider's variable must still refuse.
// The observable is the (model, which-var-was-set) pairing — two-dimensional,
// so a single-var check collapses it.
func TestRequireProviderCredential_KeyPresent_AndWrongKeyStillRefuses(t *testing.T) {
	keyed := []struct{ model, envVar string }{
		{"openrouter/anthropic/claude-haiku-4-5", "OPENROUTER_API_KEY"},
		{"claude-3-5-sonnet", "ANTHROPIC_API_KEY"},
		{"gpt-5-codex", "OPENAI_API_KEY"},
		{"gemini-3-1-pro", "GOOGLE_API_KEY"},
	}
	for _, c := range keyed {
		// Correct var set -> admit.
		for _, other := range []string{"OPENROUTER_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY"} {
			// set ALL keys to empty first, then set only the correct one —
			// so only the correct var being present is what admits the run.
			t.Setenv(other, "")
		}
		t.Setenv(c.envVar, "sk-test-key")
		if err := requireProviderCredential(c.model); err != nil {
			t.Errorf("model %q with %s set must pass; refused: %v", c.model, c.envVar, err)
		}
		// A DIFFERENT provider's var set (correct one cleared) -> must refuse.
		for _, k := range []string{"OPENROUTER_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY"} {
			t.Setenv(k, "")
		}
		t.Setenv(differentVarFor(c.envVar), "sk-test-key")
		err := requireProviderCredential(c.model)
		if err == nil {
			t.Errorf("model %q must refuse when only a DIFFERENT provider's var is set (got a pass)", c.model)
		}
	}
}

// differentVarFor returns any credential env var that is NOT the one given.
// Used to prove the check keys on the resolved provider, not on "some key is set".
func differentVarFor(envVar string) string {
	switch envVar {
	case "OPENROUTER_API_KEY":
		return "OPENAI_API_KEY"
	case "OPENAI_API_KEY":
		return "ANTHROPIC_API_KEY"
	case "ANTHROPIC_API_KEY":
		return "GOOGLE_API_KEY"
	default:
		return "OPENROUTER_API_KEY"
	}
}

// TestRequireProviderCredential_UnresolvableModel_T_B6: GuessProvider returns
// "" for an unresolvable model string — measured ai.GuessProvider("") == ""
// (internal/ai/config.go terminal `return ""`). EnvVarForProvider("") then
// returns "", which a naive implementation would silently PASS. D1 requires a
// loud refusal HERE instead, naming the model and the unresolvable-provider
// fact.
func TestRequireProviderCredential_UnresolvableModel_LoudRefusal(t *testing.T) {
	// Never leave a real key lying around while asserting the unresolvable leg.
	for _, c := range preflightCases {
		if c.envVar != "" {
			t.Setenv(c.envVar, "")
		}
	}
	for _, model := range []string{"", "not-a-real-model-xyz", "gibberish"} {
		err := requireProviderCredential(model)
		if err == nil {
			t.Errorf("unresolvable model %q passed; must REFUSE loudly (no silent fallback)", model)
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, "could not resolve provider") {
			t.Errorf("unresolvable-model error does not say the provider could not be resolved; got: %q", msg)
		}
		if model != "" && !strings.Contains(msg, model) {
			t.Errorf("unresolvable-model error does not name the model; got: %q", msg)
		}
	}
}

// TestRequireProviderCredential_AllMotokoLanesResolve (T-B6b): every
// agent_model_name across the 17 `agent_cli: "motoko"` lanes in
// internal/eval_harness/models.yml must resolve to a non-empty provider via
// ai.GuessProvider. This is the coverage gate that fails a future motoko lane
// whose model string is unresolvable — TODAY, all 17 begin with ollama/ or
// openrouter/ (measured this session; design doc §12.2 enumerates them).
func TestRequireProviderCredential_AllMotokoLanesResolve(t *testing.T) {
	lanes := motokoLaneModels(t)
	if len(lanes) == 0 {
		t.Skip("could not locate internal/eval_harness/models.yml; lane-coverage gate not run")
	}
	if len(lanes) != 17 {
		t.Errorf("expected 17 motoko lanes in models.yml, found %d", len(lanes))
	}
	for _, m := range lanes {
		if ai.GuessProvider(m) == "" {
			t.Errorf("motoko lane model %q is unresolvable by GuessProvider; a future motoko lane must not ship unresolvable", m)
		}
	}
}

// findRepoRoot walks up from the test cwd until it finds internal/eval_harness/models.yml.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "internal", "eval_harness", "models.yml")); statErr == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return ""
}

// motokoLaneModels reads the agent_model_name values of every `agent_cli:
// "motoko"` block in the repo models.yml.
func motokoLaneModels(t *testing.T) []string {
	repo := findRepoRoot(t)
	if repo == "" {
		return []string{}
	}
	data, err := os.ReadFile(filepath.Join(repo, "internal", "eval_harness", "models.yml"))
	if err != nil {
		return []string{}
	}
	var lanes []string
	inMotoko := false
	for _, line := range strings.Split(string(data), "\n") {
		tr := strings.TrimSpace(line)
		if strings.HasPrefix(tr, "agent_cli:") {
			inMotoko = strings.Contains(tr, "\"motoko\"")
		} else if inMotoko && strings.HasPrefix(tr, "agent_model_name:") {
			val := tr[len("agent_model_name:"):]
			// Drop an inline YAML comment ("...  # note") before parsing the value.
			if ci := strings.Index(val, "#"); ci >= 0 {
				val = val[:ci]
			}
			val = strings.TrimSpace(val)
			val = strings.Trim(val, "\"")
			lanes = append(lanes, val)
		}
	}
	return lanes
}

// T-CALLSITE — the WIRING arm, added by the controller at iteration 14 after
// mutation MUT-4 SURVIVED. MUT-4 neutered the requireProviderCredential call in
// ExecuteStreaming (`if err := error(nil); err != nil {`) and the ENTIRE package
// stayed green (rc=0, 43.9s): every other arm calls the helper DIRECTLY, and the
// one Execute-level arm (T-ORDER in execute_test.go) drives an `ollama/…` task,
// which the guard ADMITS — so its observable is not downstream of the wiring at
// all. That is this repo's named recurring shape, guard the helper and miss the
// call site, reproduced inside the milestone whose whole subject is a guard.
//
// The discriminating observable is a REFUSAL that reaches Execute, asserted two
// independent ways, neither of which "the run never started" can satisfy:
//
//	(a) the error text names the resolved provider AND the missing variable AND
//	    the ExecuteStreaming wrapper's own prefix — a string only this code path
//	    produces, never a bare "an error was returned";
//	(b) the mock binary must NOT have run. That half is an ABSENCE, so it is
//	    admissible only because the control arm below proves, on the same
//	    instrument in the same test, that the marker DOES appear when the task
//	    is admitted. Without the control, (b) would be satisfied equally by a
//	    preflight that refused and by a mock that was never wired up.
func TestExecuteStreaming_CallSite_RefusesKeyRequiringLane(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}
	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")
	marker := filepath.Join(tmp, "mock-was-invoked")

	// The mock records that it ran, then emits the minimal JSONL a successful
	// parse needs. Reaching it at all means the preflight did not refuse.
	mockScript := `#!/bin/bash
set -e
echo ran > "` + marker + `"
for arg in "$@"; do
  if [ "$arg" = "--version" ]; then
    echo "motoko_repo=/tmp/fake-motoko-repo"
    exit 0
  fi
done
LOGDIR="$WORKDIR/.motoko/logfile"
mkdir -p "$LOGDIR"
SESSION="${MOTOKO_SESSION_ID:-session_unknown}"
cat > "$LOGDIR/$SESSION.jsonl" <<EOF
{"schema_version":"1","session_id":"$SESSION","type":"session_start","task":"x","model":"x","brainVersion":"0.2.0"}
{"schema_version":"1","session_id":"$SESSION","type":"run_summary","finish_reason":"stop","duration_ms":1,"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}
EOF
exit 0
`
	if err := os.WriteFile(mockMotoko, []byte(mockScript), 0755); err != nil {
		t.Fatalf("writing mock binary: %v", err)
	}
	wsDir := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("creating workspace: %v", err)
	}

	t.Setenv("OPENROUTER_API_KEY", "")
	exe, err := New(&executor.Config{MotokoPath: mockMotoko, MotokoModel: "ollama/qwen3.6:35b-a3b-mxfp8"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// TREATMENT: a key-requiring per-task lane with the key unset.
	_, err = exe.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "any",
		Model:     "openrouter/anthropic/claude-haiku-4-5",
	})
	if err == nil {
		t.Fatal("Execute admitted an openrouter lane with OPENROUTER_API_KEY unset — the ExecuteStreaming call site is not wired (MUT-4 would survive)")
	}
	for _, want := range []string{
		"motoko provider preflight refused",
		`resolves to provider "openrouter"`,
		"requires OPENROUTER_API_KEY",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal text missing %q; got: %v", want, err)
		}
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("the mock motoko binary RAN — the refusal happened after the subprocess, not before it")
	}

	// CONTROL, same instrument, same test: an admitted lane must reach the mock.
	// This is what makes the marker-absence above a measurement rather than a
	// vacuous pass.
	wsDir2 := filepath.Join(tmp, "ws2")
	if err := os.MkdirAll(wsDir2, 0755); err != nil {
		t.Fatalf("creating control workspace: %v", err)
	}
	if _, err := exe.Execute(context.Background(), &executor.Task{
		Workspace: wsDir2,
		Directive: "any",
		Model:     "ollama/qwen3.6:35b-a3b-mxfp8",
	}); err != nil {
		t.Fatalf("control: admitted ollama lane failed: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatal("CONTROL DID NOT FIRE: the mock never ran even for an admitted lane, so the marker proves nothing about the treatment arm")
	}
}
