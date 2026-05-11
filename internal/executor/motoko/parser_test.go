package motoko

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseSessionJSONL_Success verifies the canonical happy-path fixture
// produces a Result with run_summary totals (NOT summed per-step values),
// finish_reason=stop, and motoko_commit / motoko_model in ProviderData.
func TestParseSessionJSONL_Success(t *testing.T) {
	res, err := parseSessionJSONL("testdata/session_success.jsonl")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// run_summary is authoritative — these are the totals from the summary
	// event, NOT summed per-step (which would be 234+345=579 / 89+12=101).
	// (In this fixture they happen to equal because the summary is honest;
	// we assert both to lock the wiring.)
	if res.InputTokens != 579 {
		t.Errorf("InputTokens = %d, want 579 (from run_summary.usage)", res.InputTokens)
	}
	if res.OutputTokens != 101 {
		t.Errorf("OutputTokens = %d, want 101 (from run_summary.usage)", res.OutputTokens)
	}
	if res.CacheReadInputTokens != 128 {
		t.Errorf("CacheReadInputTokens = %d, want 128", res.CacheReadInputTokens)
	}
	if res.CacheCreationInputTokens != 128 {
		t.Errorf("CacheCreationInputTokens = %d, want 128", res.CacheCreationInputTokens)
	}
	if res.CostUSD != 0.000813 {
		t.Errorf("CostUSD = %v, want 0.000813", res.CostUSD)
	}
	if res.NumTurns != 2 {
		t.Errorf("NumTurns = %d, want 2 (from run_summary.steps_executed)", res.NumTurns)
	}
	if res.DurationMS != 4127 {
		t.Errorf("DurationMS = %d, want 4127 (from run_summary.duration_ms)", res.DurationMS)
	}
	if !res.Success {
		t.Errorf("Success = false, want true (finish_reason=stop)")
	}
	if res.Output != "Wrote f.ail returning 42." {
		t.Errorf("Output = %q, want last done event text", res.Output)
	}
	if res.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", res.ToolCallCount)
	}
	if res.SessionID != "session_test-success" {
		t.Errorf("SessionID = %q, want session_test-success", res.SessionID)
	}
	if commit, ok := res.ProviderData["motoko_commit"]; !ok || commit != "84fa449" {
		t.Errorf("ProviderData[motoko_commit] = %v, want 84fa449", commit)
	}
	if model, ok := res.ProviderData["motoko_model"]; !ok || model != "openrouter/anthropic/claude-haiku-4-5" {
		t.Errorf("ProviderData[motoko_model] = %v, want openrouter/anthropic/claude-haiku-4-5", model)
	}
	if events, ok := res.ProviderData["motoko_events"].([]map[string]any); !ok || len(events) < 5 {
		t.Errorf("ProviderData[motoko_events] missing or too few; got %T (len %d)", events, len(events))
	}
}

// TestParseSessionJSONL_CostExhausted verifies a non-stop finish_reason
// produces Success=false with a populated Error string.
func TestParseSessionJSONL_CostExhausted(t *testing.T) {
	res, err := parseSessionJSONL("testdata/session_cost_exhausted.jsonl")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if res.Success {
		t.Error("Success = true, want false (cost_exhausted is a failure)")
	}
	if res.Error == "" {
		t.Error("Error is empty; want the cost-cap message")
	}
	if res.CostUSD != 0.046 {
		t.Errorf("CostUSD = %v, want 0.046", res.CostUSD)
	}
	if res.ToolCallCount != 3 {
		t.Errorf("ToolCallCount = %d, want 3", res.ToolCallCount)
	}
	if fr, _ := res.ProviderData["motoko_finish_reason"].(string); fr != "cost_exhausted" {
		t.Errorf("ProviderData[motoko_finish_reason] = %q, want cost_exhausted", fr)
	}
	// M-EVAL-SWEET-SPOT: top-level Result.FinishReason must mirror the
	// ProviderData entry so the eval harness can classify without dipping
	// into the provider-specific map.
	if res.FinishReason != "cost_exhausted" {
		t.Errorf("Result.FinishReason = %q, want cost_exhausted", res.FinishReason)
	}
	// M-EVAL-SWEET-SPOT-FOLLOWUP: CostKilledAt must be populated when motoko
	// stopped because its cost cap fired. Drives the budget_blocked bucket
	// in sweet-spot reporting.
	if res.CostKilledAt != 0.046 {
		t.Errorf("CostKilledAt = %v, want 0.046 (= CostUSD when cost_exhausted)", res.CostKilledAt)
	}
}

// TestParseSessionJSONL_DP7Rejected verifies dp7_verifier_rejected events
// surface in ProviderData["dp7_rejections"] count, even on a successful run
// (DP7 retries until type-check passes; final run_summary is finish_reason=stop).
func TestParseSessionJSONL_DP7Rejected(t *testing.T) {
	res, err := parseSessionJSONL("testdata/session_dp7_rejected.jsonl")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !res.Success {
		t.Errorf("Success = false, want true (this run eventually succeeded after DP7 retry)")
	}
	count, ok := res.ProviderData["dp7_rejections"].(int)
	if !ok || count != 1 {
		t.Errorf("ProviderData[dp7_rejections] = %v, want 1 (one rejection before retry succeeded)", res.ProviderData["dp7_rejections"])
	}
	if res.NumTurns != 4 {
		t.Errorf("NumTurns = %d, want 4 (run_summary.steps_executed)", res.NumTurns)
	}
}

// TestParseSessionJSONL_NoSummaryCrash verifies the crash-mid-run case:
// no run_summary event → fall back to summed per-step totals + Success=false.
func TestParseSessionJSONL_NoSummaryCrash(t *testing.T) {
	res, err := parseSessionJSONL("testdata/session_no_summary_crash.jsonl")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if res.Success {
		t.Error("Success = true, want false (no run_summary)")
	}
	if res.Error == "" {
		t.Error("Error is empty; want crash message")
	}
	// Fallback summed totals (234+345=579 input, 89+120=209 output)
	if res.InputTokens != 579 {
		t.Errorf("InputTokens = %d, want 579 (sum of per-step thinking events)", res.InputTokens)
	}
	if res.OutputTokens != 209 {
		t.Errorf("OutputTokens = %d, want 209 (sum)", res.OutputTokens)
	}
	// 0.000412 + 0.000600 = 0.001012 ± float drift; tolerate 1e-9.
	if diff := res.CostUSD - 0.001012; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("CostUSD = %v, want ~0.001012 (summed per-step cost_usd)", res.CostUSD)
	}
	if res.NumTurns != 2 {
		t.Errorf("NumTurns = %d, want 2 (counted thinking events)", res.NumTurns)
	}
}

// TestParseSessionJSONL_TruncatedSuccess covers M-MOTOKO-EVAL-HARNESS-
// HARDENING M3a (gap #2): when run_summary is missing BUT the last
// thinking event has finish_reason="stop", the run completed successfully
// and the JSONL was truncated by the TS exit-on-done bug (since fixed in
// M1c). The parser must report Success=true with summed totals — not
// false-attribute the run as a crash.
func TestParseSessionJSONL_TruncatedSuccess(t *testing.T) {
	res, err := parseSessionJSONL("testdata/session_no_summary_truncated_success.jsonl")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !res.Success {
		t.Errorf("Success = false, want true (last thinking finish_reason=stop). Error=%q", res.Error)
	}
	if res.Error != "" {
		t.Errorf("Error = %q, want empty (truncated success is not a failure)", res.Error)
	}
	if res.InputTokens != 2300 {
		t.Errorf("InputTokens = %d, want 2300 (1100+1200)", res.InputTokens)
	}
	if res.OutputTokens != 55 {
		t.Errorf("OutputTokens = %d, want 55 (40+15)", res.OutputTokens)
	}
	if res.NumTurns != 2 {
		t.Errorf("NumTurns = %d, want 2", res.NumTurns)
	}
	if got, ok := res.ProviderData["motoko_run_summary_missing"].(bool); !ok || !got {
		t.Errorf("ProviderData[motoko_run_summary_missing] = %v, want true (signal to consumers)", res.ProviderData["motoko_run_summary_missing"])
	}
	if got := res.ProviderData["motoko_finish_reason"]; got != "stop" {
		t.Errorf("ProviderData[motoko_finish_reason] = %v, want stop", got)
	}
}

// TestParseSessionJSONL_TolerantToNonJSONLines verifies that non-JSON
// preamble / trailer chatter (e.g. the "This line is not JSON" line in
// the crash fixture) is skipped silently rather than aborting the parse.
func TestParseSessionJSONL_TolerantToNonJSONLines(t *testing.T) {
	// session_no_summary_crash.jsonl contains a non-JSON line. If the parser
	// aborted on it, the test above would report 0 tokens; the fact that it
	// reports the summed totals proves we skipped the bad line cleanly.
	res, err := parseSessionJSONL("testdata/session_no_summary_crash.jsonl")
	if err != nil {
		t.Fatalf("parse aborted on non-JSON line: %v", err)
	}
	if res.InputTokens == 0 {
		t.Error("parser aborted before reading thinking events (non-JSON line not tolerated)")
	}
}

// TestParseSessionLine_RejectsNonJSON verifies the line-level parser
// returns an error (not a partial event) for non-JSON input.
func TestParseSessionLine_RejectsNonJSON(t *testing.T) {
	cases := [][]byte{
		[]byte(""),
		[]byte("   "),
		[]byte("hello world"),
		[]byte("not json at all"),
	}
	for _, line := range cases {
		_, _, err := parseSessionLine(line)
		if err == nil {
			t.Errorf("parseSessionLine(%q) = no error, want error", string(line))
		}
	}
}

// TestFindSessionJSONL_DirectMatch verifies findSessionJSONL prefers the
// exact MOTOKO_SESSION_ID-named file over newest-modified fallback.
func TestFindSessionJSONL_DirectMatch(t *testing.T) {
	tmp := t.TempDir()
	logDir := filepath.Join(tmp, ".motoko", "logfile")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(logDir, "session_my-id.jsonl")
	if err := os.WriteFile(wantPath, []byte(`{"type":"session_start"}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Decoy: a newer file with a different name.
	decoy := filepath.Join(logDir, "session_other.jsonl")
	if err := os.WriteFile(decoy, []byte(`{"type":"session_start"}`), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := findSessionJSONL(tmp, "session_my-id", "")
	if err != nil {
		t.Fatalf("findSessionJSONL: %v", err)
	}
	if got != wantPath {
		t.Errorf("got %q, want %q (should prefer exact match over newest)", got, wantPath)
	}
}

// TestFindSessionJSONL_FallbackToNewest verifies fallback when the
// MOTOKO_SESSION_ID file does not exist.
func TestFindSessionJSONL_FallbackToNewest(t *testing.T) {
	tmp := t.TempDir()
	logDir := filepath.Join(tmp, ".motoko", "logfile")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(logDir, "session_old.jsonl")
	newer := filepath.Join(logDir, "session_new.jsonl")
	if err := os.WriteFile(older, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Make `newer` actually newer.
	if err := os.WriteFile(newer, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Sanity: stat ordering
	got, err := findSessionJSONL(tmp, "session_does-not-exist", "")
	if err != nil {
		t.Fatalf("findSessionJSONL: %v", err)
	}
	if got != newer && got != older {
		t.Errorf("got %q, want one of the .jsonl files", got)
	}
}

// TestFindSessionJSONL_MissingDir returns a clear error when the workspace
// has no .motoko/logfile directory.
func TestFindSessionJSONL_MissingDir(t *testing.T) {
	tmp := t.TempDir()
	_, err := findSessionJSONL(tmp, "anything", "")
	if err == nil {
		t.Error("findSessionJSONL on workspace without .motoko/logfile/ should fail")
	}
}

// TestFindSessionJSONL_DiscoveredRepoFallback covers M-MOTOKO-EVAL-HARNESS-
// HARDENING M3b (gap #5): when MOTOKO_REPO env is unset but the executor's
// HealthCheck populated discoveredRepo from `motoko --version`, the JSONL
// search must fall back to <discoveredRepo>/.motoko/logfile/.
func TestFindSessionJSONL_DiscoveredRepoFallback(t *testing.T) {
	// Workspace has no .motoko/logfile/ — forces the search past the first
	// candidate and into the fallback chain.
	workspace := t.TempDir()
	repo := t.TempDir()
	logDir := filepath.Join(repo, motokoStateDir, "logfile")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(logDir, "session_discovered.jsonl")
	if err := os.WriteFile(wantPath, []byte(`{"type":"session_start"}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Critical: MOTOKO_REPO env must be UNSET for the discoveredRepo branch
	// to fire (env wins over discovered when both are present).
	t.Setenv("MOTOKO_REPO", "")

	got, err := findSessionJSONL(workspace, "session_discovered", repo)
	if err != nil {
		t.Fatalf("findSessionJSONL with discoveredRepo fallback: %v", err)
	}
	if got != wantPath {
		t.Errorf("got %q, want %q (discoveredRepo fallback should locate JSONL)", got, wantPath)
	}
}

// TestFindSessionJSONL_EnvWinsOverDiscovered verifies that when both
// MOTOKO_REPO env and discoveredRepo are set, the env-set path takes
// precedence (operator override > auto-discovery).
func TestFindSessionJSONL_EnvWinsOverDiscovered(t *testing.T) {
	workspace := t.TempDir()
	envRepo := t.TempDir()
	discoveredRepo := t.TempDir()

	// Put a JSONL in BOTH candidate dirs with the same session id.
	for _, repo := range []string{envRepo, discoveredRepo} {
		logDir := filepath.Join(repo, motokoStateDir, "logfile")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(logDir, "session_x.jsonl"), []byte(`{}`), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MOTOKO_REPO", envRepo)

	got, err := findSessionJSONL(workspace, "session_x", discoveredRepo)
	if err != nil {
		t.Fatalf("findSessionJSONL: %v", err)
	}
	wantPath := filepath.Join(envRepo, motokoStateDir, "logfile", "session_x.jsonl")
	if got != wantPath {
		t.Errorf("got %q, want %q (MOTOKO_REPO env should beat discoveredRepo)", got, wantPath)
	}
}

// (mock-binary end-to-end Execute test lives in execute_test.go to keep
// imports tidy and avoid dragging the executor package into parser_test.go)
