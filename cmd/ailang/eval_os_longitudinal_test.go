package main

import (
	"encoding/json"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// M1 (M-EVAL-OS-LONGITUDINAL): the OS leaderboard JSON must carry the AILANG
// version it was measured against, so the longitudinal trend can key on it.
func TestBuildOSLeaderboardJSON_EmbedsAilangVersion(t *testing.T) {
	current := map[string]eval_harness.BenchmarkSummary{
		"fizzbuzz|opencode-qwen3-6|ailang": {Model: "opencode-qwen3-6", Lang: "ailang", Passed: 2, Trials: 3},
		"fizzbuzz|opencode-qwen3-6|python": {Model: "opencode-qwen3-6", Lang: "python", Passed: 3, Trials: 3},
	}
	data, err := buildOSLeaderboardJSON("rolling-20260616", "v0.25.0", current)
	if err != nil {
		t.Fatalf("buildOSLeaderboardJSON: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := out["ailang_version"]; got != "v0.25.0" {
		t.Errorf("ailang_version = %v, want v0.25.0", got)
	}
	if got := out["version"]; got != "rolling-20260616" {
		t.Errorf("version (date label) = %v, want rolling-20260616", got)
	}
	rows, ok := out["rows"].([]interface{})
	if !ok || len(rows) != 1 {
		t.Fatalf("rows = %v, want 1 model row", out["rows"])
	}
}
