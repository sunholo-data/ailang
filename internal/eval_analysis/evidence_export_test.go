package eval_analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func evidenceFixture(model, id, lang, mode string, pass bool) *BenchmarkResult {
	return &BenchmarkResult{RunMetrics: eval_harness.RunMetrics{
		Model: model, ID: id, Lang: lang, EvalMode: mode, CompileOk: pass,
		RuntimeOk: pass, StdoutOk: pass, FirstAttemptOk: pass,
		Timestamp:     time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
		AilangVersion: "v0.54.8", CostUSD: .1, CostProvenance: "metered",
		OutputTokens: 400, DurationMs: 50,
	}}
}

func TestEvidenceMissingTokenTelemetry(t *testing.T) {
	a := evidenceFixture("model", "task", "ailang", "standard", true)
	b := evidenceFixture("model", "other", "ailang", "standard", true)
	b.OutputTokens = 0
	s := buildBenchmarkEvidence([]*BenchmarkResult{a, b}, nil, nil).Rows[0].Scopes["all"]["ailang"]
	if s.OutputTokens != 400 || s.OutputTokenSamples != 1 || s.TotalTokenSamples != 0 {
		t.Fatalf("missing tokens became recorded zero: %+v", s)
	}
}

func TestEvidenceScopesCoverageAndProvenance(t *testing.T) {
	tiers := map[string]string{}
	var core []string
	var results []*BenchmarkResult
	for i := 0; i < 49; i++ {
		id := fmt.Sprintf("task-%02d", i)
		tiers[id] = "smoke"
		if i < 23 {
			tiers[id] = "core"
			core = append(core, id)
		}
		r := evidenceFixture("argon", id, "ailang", "standard", true)
		if i == 48 {
			r.FirstAttemptOk, r.RepairUsed = false, true
			r.AilangVersion = "v0.54.9"
			r.Timestamp = r.Timestamp.Add(time.Hour)
		}
		results = append(results, r)
	}
	for i := 0; i < 3; i++ {
		results = append(results, evidenceFixture("astra", fmt.Sprintf("hard-%d", i), "ailang", "standard", true))
	}
	agent := evidenceFixture("argon", "task-00", "ailang", "agent", false)
	agent.Executor, agent.CostUSD, agent.SuccessAtMs = "gemini_agents", 2, 9000
	results = append(results, agent)
	py := evidenceFixture("argon", "task-00", "python", "standard", false)
	results = append(results, py)
	got := buildBenchmarkEvidence(results, tiers, core)
	if got.SchemaVersion != 1 || len(got.Rows) != 3 {
		t.Fatalf("evidence shape: %+v", got)
	}
	var direct, agentRow, astra *EvidenceRow
	for _, row := range got.Rows {
		switch {
		case row.Model == "astra":
			astra = row
		case row.Mode == "agent":
			agentRow = row
		default:
			direct = row
		}
	}
	a := direct.Scopes["all"]["ailang"]
	if a.Runs != 49 || a.Passes != 49 || a.FirstAttemptPasses != 48 || a.Repairs != 1 {
		t.Fatalf("Argon counts: %+v", a)
	}
	if len(a.CoreCompleted) != 23 || len(a.Benchmarks) != 49 || a.Versions["v0.54.8"] != 48 || a.LatestVersion != "v0.54.9" {
		t.Fatalf("coverage/provenance: %+v", a)
	}
	if len(astra.Scopes["all"]["ailang"].CoreCompleted) != 0 || astra.Scopes["core"] != nil {
		t.Fatal("Astra's three hard tasks became Core coverage")
	}
	if direct.Scopes["core"]["ailang"].Runs != 23 || direct.Scopes["all"]["python"].Passes != 0 {
		t.Fatal("tier or language split lost")
	}
	if agentRow.Harness != "gemini_agents" || agentRow.Scopes["all"]["ailang"].Runs != 1 {
		t.Fatal("agent and direct model evidence pooled")
	}
	if *a.Costs["metered"].PerSuccessUSD < .099 || *a.Costs["metered"].PerSuccessUSD > .101 {
		t.Fatal("agent/Python cost leaked into AILANG direct scope")
	}
	if _, exists := a.Timing["generation_wall"]; exists {
		t.Fatal("legacy execution duration relabeled as generation time")
	}
	if a.Timing["recorded_duration"].MedianMs != 50 {
		t.Fatal("legacy duration discarded")
	}
}

func TestEvidenceMissingZeroTrialsAndDeterminism(t *testing.T) {
	local := evidenceFixture("local", "c1", "ailang", "agent", true)
	local.CostUSD, local.CostProvenance = 0, "free-local"
	unknown := evidenceFixture("legacy", "c1", "ailang", "standard", true)
	unknown.CostUSD, unknown.CostProvenance, unknown.AilangVersion = 0, "", ""
	trial := *local
	trial.Trial = 2
	trial.LLMWallMs = 1200
	results := []*BenchmarkResult{local, &trial, unknown}
	got := buildBenchmarkEvidence(results, map[string]string{"c1": "core"}, []string{"c1", "c2"})
	for _, row := range got.Rows {
		s := row.Scopes["all"]["ailang"]
		if row.Model == "local" {
			if s.Runs != 2 || len(s.Benchmarks) != 1 || len(s.CoreCompleted) != 1 {
				t.Fatal("repeated trials inflated task coverage")
			}
			if s.Costs["free-local"].PerSuccessUSD == nil || *s.Costs["free-local"].PerSuccessUSD != 0 {
				t.Fatal("known free-local zero became missing")
			}
		} else if s.Costs["unknown"].PerSuccessUSD != nil || s.UnknownVersionRuns != 1 {
			t.Fatal("missing cost/version manufactured")
		}
	}
	before, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		results[0], results[2] = results[2], results[0]
		after, err := json.Marshal(buildBenchmarkEvidence(results, map[string]string{"c1": "core"}, []string{"c2", "c1"}))
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatalf("export depends on input/map order: %v", err)
		}
	}
}

func TestEvidenceExportPreservesHistoryAndCostKinds(t *testing.T) {
	a := evidenceFixture("dual", "fizzbuzz", "ailang", "standard", true)
	a.LLMWallMs = 1000
	b := evidenceFixture("dual", "other", "ailang", "standard", false)
	b.CostProvenance = "list-price-equivalent"
	b.CostUSD = 9
	results := []*BenchmarkResult{a, b}
	block := buildBenchmarkEvidence(results, nil, nil)
	s := block.Rows[0].Scopes["all"]["ailang"]
	if len(s.Costs) != 2 || s.Costs["list-price-equivalent"].PerSuccessUSD != nil {
		t.Fatal("mixed billing categories pooled or zero-pass cost invented")
	}
	if s.Timing["generation_wall"].Samples != 1 || s.Timing["recorded_duration"].Samples != 2 {
		t.Fatal("different timing boundaries pooled")
	}
	path := filepath.Join(t.TempDir(), "latest.json")
	old := HistoryEntry{Version: "v0.saved", TotalRuns: 7, SuccessCount: 3, SuccessRate: 3.0 / 7}
	data, err := json.Marshal(DashboardJSON{Version: "v0.saved", History: []HistoryEntry{old}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	matrix, err := GenerateMatrix(results, "v0.new")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExportBenchmarkJSON(matrix, nil, results, path); err != nil {
		t.Fatal(err)
	}
	d := readDashboard(t, path)
	if d.BenchmarkEvidence == nil || len(d.BenchmarkEvidence.Rows) != 1 || d.Models["dual"] == nil {
		t.Fatal("additive evidence or existing models missing")
	}
	for _, h := range d.History {
		if h.Version == old.Version {
			if !reflect.DeepEqual(h, old) {
				t.Fatalf("saved history rewritten: %+v", h)
			}
			return
		}
	}
	t.Fatal("saved history disappeared")
}
