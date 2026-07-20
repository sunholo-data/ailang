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

// TestExecute_PerTaskCacheDir covers M-MOTOKO-PARALLEL-EXECUTION-ISOLATION
// (v0.18.2) M2b: Execute MUST set AILANG_CACHE_DIR to a per-session unique
// path under TMPDIR, AND the dir must be cleaned up after Execute returns.
//
// This is the load-bearing fix for the dur=0 parallel-execution crash —
// without per-task cache isolation, parallel motoko sessions race on
// writes to MOTOKO_REPO/src/core/.ailang/cache/.../core.gob, corrupting
// the file and crashing subsequent reads before the runtime initializes.
func TestExecute_PerTaskCacheDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")
	envDump := filepath.Join(tmp, "received-cache-dir.txt")
	mockScript := `#!/bin/bash
set -e
LOGDIR="$WORKDIR/.motoko/logfile"
mkdir -p "$LOGDIR"
SESSION="${MOTOKO_SESSION_ID:-session_unknown}"
# Capture the per-task cache dir env var so the test can assert against it.
echo "AILANG_CACHE_DIR=${AILANG_CACHE_DIR:-UNSET}" > "` + envDump + `"
# Minimal JSONL so adapter parsing succeeds.
cat > "$LOGDIR/$SESSION.jsonl" <<EOF
{"schema_version":"1","session_id":"$SESSION","type":"session_start","task":"x","model":"x","brainVersion":"0.2.0"}
{"schema_version":"1","session_id":"$SESSION","type":"run_summary","finish_reason":"stop","duration_ms":1,"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}
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
	res, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "any",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Success {
		t.Fatalf("Success = false; want true. Error: %s", res.Error)
	}

	// Assert the env var was set to a non-empty per-task path.
	dump, err := os.ReadFile(envDump)
	if err != nil {
		t.Fatalf("reading env dump: %v", err)
	}
	got := strings.TrimSpace(string(dump))
	if got == "AILANG_CACHE_DIR=UNSET" || got == "AILANG_CACHE_DIR=" {
		t.Errorf("AILANG_CACHE_DIR not set on subprocess; got: %q", got)
	}
	// The path should include "motoko-task-" prefix indicating per-session isolation.
	if !strings.Contains(got, "motoko-task-") {
		t.Errorf("AILANG_CACHE_DIR doesn't look like a per-task path; got: %q", got)
	}

	// Assert the dir was cleaned up after Execute returned. Extract the path
	// from the dump line ("AILANG_CACHE_DIR=/tmp/motoko-task-...") and stat it.
	cacheDir := strings.TrimPrefix(got, "AILANG_CACHE_DIR=")
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Errorf("per-task cache dir %q not cleaned up after Execute; stat err=%v", cacheDir, err)
	}
}

// TestExecute_TwoSequentialCalls_DistinctCacheDirs verifies that two
// sequential Execute calls (same executor instance, distinct sessionIDs)
// receive DISTINCT AILANG_CACHE_DIR paths. This guards against accidental
// caching at the executor level that would leak the same dir between calls.
func TestExecute_TwoSequentialCalls_DistinctCacheDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")
	dumpA := filepath.Join(tmp, "dump-A.txt")
	dumpB := filepath.Join(tmp, "dump-B.txt")
	// The mock decides which dump to write based on a marker in the directive.
	mockScript := `#!/bin/bash
set -e
LOGDIR="$WORKDIR/.motoko/logfile"
mkdir -p "$LOGDIR"
SESSION="${MOTOKO_SESSION_ID:-session_unknown}"
# Pick which dump file based on directive content.
if echo "$@" | grep -q "DUMPA"; then
  echo "$AILANG_CACHE_DIR" > "` + dumpA + `"
else
  echo "$AILANG_CACHE_DIR" > "` + dumpB + `"
fi
cat > "$LOGDIR/$SESSION.jsonl" <<EOF
{"schema_version":"1","session_id":"$SESSION","type":"session_start","task":"x","model":"x","brainVersion":"0.2.0"}
{"schema_version":"1","session_id":"$SESSION","type":"run_summary","finish_reason":"stop","duration_ms":1,"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}
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

	if _, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "DUMPA marker",
	}); err != nil {
		t.Fatalf("Execute A: %v", err)
	}
	if _, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "DUMPB marker",
	}); err != nil {
		t.Fatalf("Execute B: %v", err)
	}

	pathA := strings.TrimSpace(string(mustRead(t, dumpA)))
	pathB := strings.TrimSpace(string(mustRead(t, dumpB)))
	if pathA == "" || pathB == "" {
		t.Fatalf("one of the dumps is empty: A=%q B=%q", pathA, pathB)
	}
	if pathA == pathB {
		t.Errorf("two Execute calls reused the same cache dir: %q (must be distinct per session)", pathA)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %q: %v", path, err)
	}
	return data
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
  echo "AI_MAX_COST_USD_CENTS=${AI_MAX_COST_USD_CENTS:-UNSET}"
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
	// Cost cap: $0.50 → AI_MAX_COST_USD_CENTS=50 (M-EVAL-SWEET-SPOT-FOLLOWUP)
	res, err := exec.Execute(context.Background(), &executor.Task{
		Workspace: wsDir,
		Directive: "any",
		Budget:    executor.NewCostBudget(0.50, 0.00025, 0.00125),
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
	// M-EVAL-SWEET-SPOT-FOLLOWUP: the hard cost cap must reach motoko as
	// AI_MAX_COST_USD_CENTS so motoko's internal budget gate fires
	// finish_reason="cost_exhausted" on overrun.
	if !strings.Contains(got, "AI_MAX_COST_USD_CENTS=50\n") {
		t.Errorf("AI_MAX_COST_USD_CENTS env var missing or wrong; got:\n%s", got)
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
  echo "AI_MAX_COST_USD_CENTS=${AI_MAX_COST_USD_CENTS:-UNSET}"
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
	if !strings.Contains(got, "AI_MAX_COST_USD_CENTS=UNSET\n") {
		t.Errorf("AI_MAX_COST_USD_CENTS should be UNSET when Budget is nil; got:\n%s", got)
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

// TestExecute_StartupCrash_StderrTailInError reproduces the 2026-07-16
// incident shape: motoko exits nonzero BEFORE writing any session JSONL
// (an AILANG compile error on stdlib schema drift — std/ai.Message gained an
// `images` field the core's 4-field literals didn't have). The compile error
// lands on stderr; Result.Error must carry its tail + the stderr log path so
// triage reads the actual cause instead of chasing the "no session JSONL" /
// delivery-regression classes.
func TestExecute_StartupCrash_StderrTailInError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")
	mockScript := `#!/bin/bash
echo "error[TC001]: record literal is missing field 'images' required by std/ai.Message" >&2
echo "  --> src/core/backend.ail:42:7" >&2
exit 1
`
	if err := os.WriteFile(mockMotoko, []byte(mockScript), 0755); err != nil {
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
		t.Error("Success = true; want false")
	}
	if !strings.Contains(res.Error, "no session JSONL found") {
		t.Errorf("Error should still name the missing-JSONL symptom; got: %s", res.Error)
	}
	if !strings.Contains(res.Error, "missing field 'images'") {
		t.Errorf("Error must include the stderr tail (the actual crash cause); got: %s", res.Error)
	}
	if !strings.Contains(res.Error, "motoko-stderr-") {
		t.Errorf("Error must include the on-disk stderr log path; got: %s", res.Error)
	}
}

// TestExecute_MidRunCrash_NoRunSummary_StderrTailInError: the session JSONL
// exists but truncates before run_summary (crash mid-run) and NO system prompt
// is configured — so the delivery guard never runs. The stderr tail must still
// be attached by the central post-parse attribution, not only via the guard.
func TestExecute_MidRunCrash_NoRunSummary_StderrTailInError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")
	mockScript := `#!/bin/bash
LOGDIR="$WORKDIR/.motoko/logfile"
mkdir -p "$LOGDIR"
SESSION="${MOTOKO_SESSION_ID:-session_unknown}"
cat > "$LOGDIR/$SESSION.jsonl" <<EOF
{"schema_version":"1","session_id":"$SESSION","type":"session_start","task":"x","model":"m"}
{"schema_version":"1","session_id":"$SESSION","type":"thinking","step":1,"text":"...","finish_reason":"length","input_tokens":10,"output_tokens":5}
EOF
echo "RuntimeError: env-server connection reset mid-step" >&2
exit 1
`
	if err := os.WriteFile(mockMotoko, []byte(mockScript), 0755); err != nil {
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
		// SystemPrompt intentionally empty: the delivery guard must not be the
		// only path that surfaces stderr.
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Success {
		t.Error("Success = true; want false")
	}
	if !strings.Contains(res.Error, "no run_summary") {
		t.Errorf("Error should name the missing-run_summary symptom; got: %s", res.Error)
	}
	if !strings.Contains(res.Error, "env-server connection reset") {
		t.Errorf("Error must include the stderr tail; got: %s", res.Error)
	}
}

// TestExecute_StartupCrashWithSystemPrompt_StderrAttachedOnce: when the
// delivery guard fires its startup-crash verdict it embeds the stderr tail in
// Result.Error itself; the central post-parse attribution must then NOT append
// the same tail a second time.
func TestExecute_StartupCrashWithSystemPrompt_StderrAttachedOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}

	tmp := t.TempDir()
	mockMotoko := filepath.Join(tmp, "motoko")
	mockScript := `#!/bin/bash
LOGDIR="$WORKDIR/.motoko/logfile"
mkdir -p "$LOGDIR"
SESSION="${MOTOKO_SESSION_ID:-session_unknown}"
cat > "$LOGDIR/$SESSION.jsonl" <<EOF
{"schema_version":"1","session_id":"$SESSION","type":"session_start","task":"x","model":"m"}
EOF
echo "BOOM_UNIQUE_MARKER: startup compile failure" >&2
exit 1
`
	if err := os.WriteFile(mockMotoko, []byte(mockScript), 0755); err != nil {
		t.Fatal(err)
	}
	wsDir := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	exec, _ := New(&executor.Config{MotokoPath: mockMotoko})
	res, err := exec.Execute(context.Background(), &executor.Task{
		Workspace:    wsDir,
		Directive:    "any",
		SystemPrompt: "AILANG teaching content",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Success {
		t.Error("Success = true; want false")
	}
	if got := strings.Count(res.Error, "BOOM_UNIQUE_MARKER"); got != 1 {
		t.Errorf("stderr tail appears %d times in Error, want exactly 1 (guard + central attribution must not double-append):\n%s", got, res.Error)
	}
	if _, has := res.ProviderData["motoko_startup_crash"]; !has {
		t.Error("guard should have classified this as a startup crash")
	}
}

// TestAttachStderrTail covers the helper's two shapes: non-empty stderr gets
// the tail + log path; empty stderr still points at the log path so the reader
// knows stderr WAS captured and had nothing (vs. not captured at all).
func TestAttachStderrTail(t *testing.T) {
	got := attachStderrTail("crashed", "/tmp/motoko-stderr-x.log", "line1\nline2\n")
	if !strings.Contains(got, "crashed") || !strings.Contains(got, "line2") ||
		!strings.Contains(got, "/tmp/motoko-stderr-x.log") {
		t.Errorf("non-empty stderr: got %q", got)
	}

	got = attachStderrTail("crashed", "/tmp/motoko-stderr-x.log", "  \n")
	if !strings.Contains(got, "stderr was empty") || !strings.Contains(got, "/tmp/motoko-stderr-x.log") {
		t.Errorf("empty stderr: got %q", got)
	}

	// Long stderr is bounded to the last ~1KB.
	long := strings.Repeat("x", 4096) + "TAIL_END"
	got = attachStderrTail("crashed", "/tmp/l.log", long)
	if !strings.Contains(got, "TAIL_END") {
		t.Errorf("tail lost the end of stderr")
	}
	if len(got) > 1024+300 {
		t.Errorf("attached message too long (%d bytes); tail must be bounded to ~1KB", len(got))
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
