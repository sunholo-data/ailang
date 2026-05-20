package eval_analysis

import (
	"testing"
)

func mkAgentResult(id, lang, model, executor string, stdoutOk bool, cost float64, durMs int64) *BenchmarkResult {
	return &BenchmarkResult{
		ID:         id,
		Lang:       lang,
		Model:      model,
		Executor:   executor,
		EvalMode:   "agent",
		StdoutOk:   stdoutOk,
		CostUSD:    cost,
		DurationMs: durMs,
	}
}

// TestBuildHarnessAggregates verifies harness grouping by the Executor field (the historical
// record of which harness produced each row), success rate, and per-language breakdown.
//
// Per the implementation comment in export_json_executors.go, buildHarnessAggregates prefers
// r.Executor over cfg.GetAgentCLI to preserve the historical attribution: rows tagged with
// a retired harness (e.g. gemini-cli) must keep that attribution rather than silently
// re-bucketing under whatever the current ModelConfig says.
func TestBuildHarnessAggregates(t *testing.T) {
	results := []*BenchmarkResult{
		// claude harness — 2 ailang runs, 1 success
		mkAgentResult("fizzbuzz", "ailang", "claude-sonnet-4-6", "claude", true, 0.012, 45000),
		mkAgentResult("sort_list", "ailang", "claude-sonnet-4-6", "claude", false, 0.008, 60000),
		// opencode harness — 1 ailang success, 1 python success
		mkAgentResult("fizzbuzz", "ailang", "opencode-haiku", "opencode", true, 0.004, 80000),
		mkAgentResult("fizzbuzz", "python", "opencode-haiku", "opencode", true, 0.003, 75000),
	}

	harnessMap := buildHarnessAggregates(results, nil)
	if len(harnessMap) != 2 {
		t.Fatalf("expected 2 harness keys (claude, opencode), got %d: %v", len(harnessMap), keys(harnessMap))
	}

	// claude harness: 2 runs, 1 success (0.5 success rate), ailang only
	claudeRaw, ok := harnessMap["claude"]
	if !ok {
		t.Fatalf("expected 'claude' harness key, got keys: %v", keys(harnessMap))
	}
	claude := claudeRaw.(map[string]interface{})
	if runs, _ := claude["total_runs"].(int); runs != 2 {
		t.Errorf("claude total_runs: want 2, got %v", claude["total_runs"])
	}
	if sr, _ := claude["success_rate"].(float64); sr < 0.49 || sr > 0.51 {
		t.Errorf("claude success_rate: want ~0.5, got %v", sr)
	}
	claudeLangs := claude["languages"].(map[string]interface{})
	if _, ok := claudeLangs["ailang"]; !ok {
		t.Error("expected 'ailang' in claude harness language breakdown")
	}

	// opencode harness: 2 runs, 2 successes (1.0 success rate), ailang + python
	opencodeRaw, ok := harnessMap["opencode"]
	if !ok {
		t.Fatalf("expected 'opencode' harness key, got keys: %v", keys(harnessMap))
	}
	opencode := opencodeRaw.(map[string]interface{})
	if runs, _ := opencode["total_runs"].(int); runs != 2 {
		t.Errorf("opencode total_runs: want 2, got %v", opencode["total_runs"])
	}
	if sr, _ := opencode["success_rate"].(float64); sr < 0.99 || sr > 1.01 {
		t.Errorf("opencode success_rate: want ~1.0, got %v", sr)
	}
	opencodeLangs := opencode["languages"].(map[string]interface{})
	if _, ok := opencodeLangs["ailang"]; !ok {
		t.Error("expected 'ailang' in opencode harness language breakdown")
	}
	if _, ok := opencodeLangs["python"]; !ok {
		t.Error("expected 'python' in opencode harness language breakdown")
	}
}

// TestBuildHarnessAggregatesUnknownFallback verifies that when r.Executor is empty AND
// GlobalModelsConfig is nil, results fall back to the "unknown" bucket without panic.
func TestBuildHarnessAggregatesUnknownFallback(t *testing.T) {
	results := []*BenchmarkResult{
		mkAgentResult("fizzbuzz", "ailang", "some-model", "", true, 0.01, 50000),
		mkAgentResult("sort", "ailang", "some-model", "", false, 0.01, 50000),
	}
	harnessMap := buildHarnessAggregates(results, nil)
	unknownRaw, ok := harnessMap["unknown"]
	if !ok {
		t.Fatalf("expected 'unknown' fallback bucket when Executor is empty and cfg is nil, got keys: %v", keys(harnessMap))
	}
	unknown := unknownRaw.(map[string]interface{})
	if runs, _ := unknown["total_runs"].(int); runs != 2 {
		t.Errorf("unknown total_runs: want 2, got %v", unknown["total_runs"])
	}
}

// TestBuildHarnessAggregatesEmpty verifies no panic on empty input.
func TestBuildHarnessAggregatesEmpty(t *testing.T) {
	result := buildHarnessAggregates(nil, nil)
	if result == nil {
		t.Error("expected non-nil map for nil input")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map for nil input, got %v", result)
	}
}

func keys(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
