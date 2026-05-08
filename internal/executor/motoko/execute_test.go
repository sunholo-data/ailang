package motoko

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
