package eval_analysis

import (
	"path/filepath"
	"testing"
	"time"
)

// TestExportSplit_AgentOnlyModelsExcludedFromStandard pins the standard/agent
// split (M-EVAL-BENCHMARK-UI-CONSOLIDATION): a model that ran ONLY in agent mode
// must not appear in the standard `models` map — it would render as a phantom-zero
// row on the Model Leaderboard (the bug fixed 2026-06-12). Standard-ran models
// (incl. dual-mode) stay in `models`; agent-only goes to `agentModels`.
func TestExportSplit_AgentOnlyModelsExcludedFromStandard(t *testing.T) {
	now := time.Now()
	mk := func(model, mode string) *BenchmarkResult {
		return &BenchmarkResult{
			ID: "fizzbuzz", Lang: "ailang", Model: model, EvalMode: mode,
			CompileOk: true, RuntimeOk: true, StdoutOk: true, Timestamp: now,
		}
	}
	results := []*BenchmarkResult{
		mk("std-model", "standard"),
		mk("dual-model", "standard"),
		mk("dual-model", "agent"),
		mk("agent-only", "agent"),
	}

	// The matrix is the STANDARD performance matrix (it drives the standard models loop).
	var standard []*BenchmarkResult
	for _, r := range results {
		if r.EvalMode != "agent" {
			standard = append(standard, r)
		}
	}
	matrix, err := GenerateMatrix(standard, "v0.test")
	if err != nil {
		t.Fatalf("GenerateMatrix: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "dashboard.json")
	if _, err := ExportBenchmarkJSON(matrix, nil, results, tmpFile); err != nil {
		t.Fatalf("ExportBenchmarkJSON: %v", err)
	}
	d := readDashboard(t, tmpFile)

	if _, leaked := d.Models["agent-only"]; leaked {
		t.Errorf("agent-only model leaked into the standard `models` map (phantom-zero row)")
	}
	for _, want := range []string{"std-model", "dual-model"} {
		if _, ok := d.Models[want]; !ok {
			t.Errorf("standard-ran model %q missing from `models`", want)
		}
	}
	if _, ok := d.AgentModels["agent-only"]; !ok {
		t.Errorf("agent-only model missing from `agentModels`")
	}
}
