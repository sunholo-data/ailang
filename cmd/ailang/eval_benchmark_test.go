package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// TestRunSingleBenchmark_StandardModeSkipsAgentWorkspaceOnly is the acceptance
// test for the M-EVAL-STANDARD-MODE-INPUT-FILES-GAP dispatch-time guard
// (design component 3): markdown_reimplement sets grade_entrypoint, so it is
// agent-mode-only. Called directly (as a `--benchmarks` invocation bypassing
// the scheduler filter) in standard mode it must short-circuit with a
// structured skip result — error_category "skipped_mode_incompatible" plus
// Validity invalid with reason "mode_incompatible" — and make ZERO AI provider
// API calls.
//
// The provider transport follows the in-package httptest mock precedent
// (configdriven_dispatch_test.go): a counting httptest server stands in for the
// model API. The ollama provider is the one standard-mode transport with an env
// seam (OLLAMA_HOST), so the sentinel model routes there; if the guard ever
// stops firing, the standard path reaches NewAIAgent → provider HTTP call and
// the counter reads non-zero, failing this test.
func TestRunSingleBenchmark_StandardModeSkipsAgentWorkspaceOnly(t *testing.T) {
	var providerCalls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"-- sentinel: no eval call should ever reach the provider"}}`))
	}))
	defer server.Close()

	// Route the sentinel model's provider transport at the counting server and
	// register it in the model registry (runSingleBenchmark resolves the model
	// via modelreg.GlobalModelsConfig). Both are restored on cleanup.
	t.Setenv("OLLAMA_HOST", server.URL)
	origModels := modelreg.GlobalModelsConfig
	modelreg.GlobalModelsConfig = &modelreg.ModelsConfig{
		Models: map[string]modelreg.ModelConfig{
			"mock-sentinel": {Provider: "ollama", APIName: "mock-sentinel"},
		},
	}
	defer func() { modelreg.GlobalModelsConfig = origModels }()

	// Load the REAL benchmark spec the way runSingleBenchmark does.
	origBenchmarkDir := evalBenchmarkDir
	evalBenchmarkDir = "../../benchmarks"
	defer func() { evalBenchmarkDir = origBenchmarkDir }()

	specPath := filepath.Join(evalBenchmarkDir, "markdown_reimplement.yml")
	if _, err := os.Stat(specPath); err != nil {
		t.Skipf("markdown_reimplement.yml not present (%v); skipping", err)
	}
	spec, err := eval_harness.LoadSpec(specPath)
	if err != nil {
		t.Fatalf("LoadSpec(markdown_reimplement): %v", err)
	}
	if spec.GradeEntrypoint == "" {
		t.Fatal("fixture drift: markdown_reimplement no longer sets grade_entrypoint; this test's premise is gone")
	}

	outputDir := t.TempDir()
	success, err := runSingleBenchmark(
		context.Background(),
		"mock-sentinel", // model
		"markdown_reimplement",
		"ailang", // lang
		"",       // condition (legacy)
		1,        // trial
		42,       // seed
		outputDir,
		30*time.Second,
		false, // selfRepair
		"",    // promptVersion
		nil,   // agentConfig — nil = STANDARD mode
		"",    // taskID
		nil,   // evalChain
		nil,   // onCost
	)

	// The skip is a structured outcome, not an error: the row was banked, the
	// suite loop just doesn't count it as a pass.
	if err != nil {
		t.Fatalf("runSingleBenchmark returned error %v, want nil (skip is not a failure)", err)
	}
	if success {
		t.Error("runSingleBenchmark returned success=true, want false (benchmark was skipped, not passed)")
	}

	// Exactly one skip row banked under the standard-mode results directory.
	rows, err := filepath.Glob(filepath.Join(outputDir, "standard", "*.json"))
	if err != nil {
		t.Fatalf("glob results: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("banked %d result rows, want exactly 1 skip row", len(rows))
	}
	data, err := os.ReadFile(rows[0])
	if err != nil {
		t.Fatalf("read %s: %v", rows[0], err)
	}
	var metrics eval_harness.RunMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("unmarshal %s: %v", rows[0], err)
	}

	if metrics.ID != "markdown_reimplement" {
		t.Errorf("row id = %q, want markdown_reimplement", metrics.ID)
	}
	if metrics.ErrorCategory != "skipped_mode_incompatible" {
		t.Errorf("row error_category = %q, want skipped_mode_incompatible (human-readable skip label)", metrics.ErrorCategory)
	}
	if metrics.Validity == nil {
		t.Fatal("row Validity is nil — absent decodes as VALID, so the skip row would leak into every aggregate")
	}
	if metrics.Validity.Valid {
		t.Error("row Validity.Valid = true, want false (the model was never invoked; not a measurement)")
	}
	if metrics.Validity.Reason != eval_harness.ReasonModeIncompatible {
		t.Errorf("row Validity.Reason = %q, want %q (machine-aggregation marker)", metrics.Validity.Reason, eval_harness.ReasonModeIncompatible)
	}
	if metrics.EvalMode != eval_harness.EvalModeStandard {
		t.Errorf("row eval_mode = %q, want %q", metrics.EvalMode, eval_harness.EvalModeStandard)
	}

	// The load-bearing half: ZERO provider API calls.
	if got := providerCalls.Load(); got != 0 {
		t.Errorf("provider call counter = %d, want 0 — the guard must short-circuit before any AI dispatch", got)
	}
}
