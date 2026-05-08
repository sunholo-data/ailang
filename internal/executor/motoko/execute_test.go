package motoko

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestExecute_MockBinary_FullPipeline uses a POSIX shell stand-in that
// writes a known JSONL into ${WORKDIR}/.motoko/logfile/<MOTOKO_SESSION_ID>.jsonl
// and exits 0. Exercises the full subprocess→file-locate→parseSessionJSONL
// path without requiring real motoko on PATH (CI-safe).
func TestExecute_MockBinary_FullPipeline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")

	// The mock copies our success fixture into the expected location,
	// substituting the session id so findSessionJSONL's direct-match path
	// succeeds. Path to the fixture is resolved at runtime relative to the
	// test binary's working directory (which is the package dir under
	// `go test`).
	fixturePath := filepath.Join("testdata", "session_success.jsonl")
	absFixture, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("resolving fixture path: %v", err)
	}

	mockScript := `#!/bin/bash
set -e
LOGDIR="$WORKDIR/.motoko/logfile"
mkdir -p "$LOGDIR"
SESSION="${MOTOKO_SESSION_ID:-session_unknown}"
sed "s/session_test-success/$SESSION/g" "` + absFixture + `" > "$LOGDIR/$SESSION.jsonl"
exit 0
`
	if err := os.WriteFile(mockMotoko, []byte(mockScript), 0755); err != nil {
		t.Fatalf("writing mock binary: %v", err)
	}

	wsDir := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatalf("creating workspace: %v", err)
	}

	exec, err := New(&executor.Config{
		MotokoPath:    mockMotoko,
		MotokoModel:   "openrouter/anthropic/claude-haiku-4-5",
		MotokoProfile: "dogfood",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "say hello",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Fatalf("Success = false; want true. Error: %s", res.Error)
	}
	if res.InputTokens != 579 {
		t.Errorf("InputTokens = %d, want 579 (from fixture run_summary)", res.InputTokens)
	}
	if res.OutputTokens != 101 {
		t.Errorf("OutputTokens = %d, want 101", res.OutputTokens)
	}
	if res.CostUSD != 0.000813 {
		t.Errorf("CostUSD = %v, want 0.000813", res.CostUSD)
	}
	if res.CacheReadInputTokens != 128 || res.CacheCreationInputTokens != 128 {
		t.Errorf("cache fields = (%d, %d), want (128, 128)", res.CacheReadInputTokens, res.CacheCreationInputTokens)
	}
	if res.SessionID == "" {
		t.Errorf("SessionID is empty; expected MOTOKO_SESSION_ID we set")
	}
	if res.NumTurns != 2 {
		t.Errorf("NumTurns = %d, want 2", res.NumTurns)
	}
}

// TestExecute_BudgetEnvVarPassthrough covers M-MOTOKO-EVAL-HARNESS-HARDENING
// M5a-c (gaps #3, #9): when Task.Budget is set, the adapter must convert
// per-1K USD rates to per-1M millicents and pass them as
// MOTOKO_COST_INPUT_PER_1M_MILLICENTS / MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS
// env vars on the spawn. This lets motoko's cost_warning + cost_exhausted
// thresholds fire and run_summary report a non-zero cost_usd without
// duplicating pricing in every motoko profile config.
func TestExecute_BudgetEnvVarPassthrough(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")

	// Mock writes the env vars it received to a side-channel file we can
	// then assert against. Also writes a minimal JSONL so the adapter
	// doesn't fail on the missing-session-file path.
	envDump := filepath.Join(tmp, "received-env.txt")
	mockScript := `#!/bin/bash
set -e
LOGDIR="$WORKDIR/.motoko/logfile"
mkdir -p "$LOGDIR"
SESSION="${MOTOKO_SESSION_ID:-session_unknown}"
# Write the relevant env vars to a side-channel for the test to assert.
{
  echo "MOTOKO_COST_INPUT_PER_1M_MILLICENTS=${MOTOKO_COST_INPUT_PER_1M_MILLICENTS:-UNSET}"
  echo "MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS=${MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS:-UNSET}"
} > "` + envDump + `"
# Minimal JSONL so the adapter completes parsing without error.
cat > "$LOGDIR/$SESSION.jsonl" <<EOF
{"schema_version":"1","session_id":"$SESSION","type":"session_start","task":"x","model":"openrouter/anthropic/claude-haiku-4-5","brainVersion":"0.2.0"}
{"schema_version":"1","session_id":"$SESSION","type":"thinking","step":0,"text":"done","finish_reason":"stop","tool_calls":0,"input_tokens":100,"output_tokens":50}
{"schema_version":"1","session_id":"$SESSION","type":"run_summary","finish_reason":"stop","duration_ms":100,"usage":{"input_tokens":100,"output_tokens":50,"total_tokens":150},"total_cost_usd":0.0001}
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

	exec, err := New(&executor.Config{MotokoPath: mockMotoko})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// claude-haiku-4-5 rates: 0.00025 input / 0.00125 output per 1K USD.
	// Expected per-1M millicents: 0.00025 × 1e8 = 25000  (input)
	//                              0.00125 × 1e8 = 125000 (output)
	res, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "any",
		Budget:    executor.NewCostBudget(0, 0.00025, 0.00125),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Fatalf("Success = false; want true. Error: %s", res.Error)
	}

	dump, err := os.ReadFile(envDump)
	if err != nil {
		t.Fatalf("reading env dump: %v", err)
	}
	got := string(dump)
	if !strings.Contains(got, "MOTOKO_COST_INPUT_PER_1M_MILLICENTS=25000\n") {
		t.Errorf("input rate env var missing or wrong; got:\n%s", got)
	}
	if !strings.Contains(got, "MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS=125000\n") {
		t.Errorf("output rate env var missing or wrong; got:\n%s", got)
	}
}

// TestExecute_NoBudget_NoEnvVar verifies that when Task.Budget is nil
// (e.g. coordinator-driven runs that don't enforce cost budgets), the
// adapter must NOT emit MOTOKO_COST_* env vars — motoko then uses its
// profile-config rates as the fallback (typically 0).
func TestExecute_NoBudget_NoEnvVar(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")
	envDump := filepath.Join(tmp, "received-env.txt")
	mockScript := `#!/bin/bash
set -e
LOGDIR="$WORKDIR/.motoko/logfile"
mkdir -p "$LOGDIR"
SESSION="${MOTOKO_SESSION_ID:-session_unknown}"
{
  echo "MOTOKO_COST_INPUT_PER_1M_MILLICENTS=${MOTOKO_COST_INPUT_PER_1M_MILLICENTS:-UNSET}"
  echo "MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS=${MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS:-UNSET}"
} > "` + envDump + `"
cat > "$LOGDIR/$SESSION.jsonl" <<EOF
{"schema_version":"1","session_id":"$SESSION","type":"session_start","task":"x","model":"x","brainVersion":"0.2.0"}
{"schema_version":"1","session_id":"$SESSION","type":"run_summary","finish_reason":"stop","duration_ms":1,"usage":{"input_tokens":0,"output_tokens":0,"total_tokens":0}}
EOF
exit 0
`
	if err := os.WriteFile(mockMotoko, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}
	wsDir := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}
	exec, _ := New(&executor.Config{MotokoPath: mockMotoko})
	_, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "any",
		// Budget intentionally nil
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	dump, _ := os.ReadFile(envDump)
	got := string(dump)
	if !strings.Contains(got, "MOTOKO_COST_INPUT_PER_1M_MILLICENTS=UNSET\n") {
		t.Errorf("input rate env var should be UNSET when Budget is nil; got:\n%s", got)
	}
	if !strings.Contains(got, "MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS=UNSET\n") {
		t.Errorf("output rate env var should be UNSET when Budget is nil; got:\n%s", got)
	}
}

// TestExecute_NoJSONLFile verifies that when the mock binary exits 0 but
// writes no JSONL, Execute returns a Result with Success=false and a clear
// error pointing at the missing file (rather than panicking or returning a
// confusing zero-Result).
func TestExecute_NoJSONLFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")
	if err := os.WriteFile(mockMotoko, []byte("#!/bin/bash\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	wsDir := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	exec, _ := New(&executor.Config{MotokoPath: mockMotoko})
	res, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "any",
	})
	if err != nil {
		t.Fatalf("Execute returned hard error (should return Result with Error instead): %v", err)
	}
	if res.Success {
		t.Errorf("Success = true; want false")
	}
	if res.Error == "" {
		t.Errorf("Error is empty; expected a session-jsonl-not-found message")
	}
}

// TestLiveRun_Motoko (gated): runs against the real motoko binary if
// AILANG_MOTOKO_LIVE=1 is set AND `motoko` is on PATH. Skipped by default
// so CI does not spawn real LLM calls. Use this to validate adapter
// behaviour against the actual binary on a developer machine.
func TestLiveRun_Motoko(t *testing.T) {
	if os.Getenv("AILANG_MOTOKO_LIVE") != "1" {
		t.Skip("set AILANG_MOTOKO_LIVE=1 to run live motoko tests")
	}
	exec, err := New(executor.DefaultConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := exec.HealthCheck(context.Background()); err != nil {
		t.Skipf("motoko binary not healthy on PATH: %v", err)
	}
	wsDir := t.TempDir()
	res, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "Print 'Hello from motoko' to stdout in a one-line python script and run it.",
	})
	if err != nil {
		t.Fatalf("live Execute: %v", err)
	}
	t.Logf("live result: success=%v turns=%d cost=$%.6f tokens=%d/%d cache=%d/%d",
		res.Success, res.NumTurns, res.CostUSD, res.InputTokens, res.OutputTokens,
		res.CacheReadInputTokens, res.CacheCreationInputTokens)
}
