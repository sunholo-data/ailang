package eval_analysis

import "testing"

// res is a tiny constructor for a trial result (pass = all three ok).
func res(model, lang, id string, pass bool) *BenchmarkResult {
	return &BenchmarkResult{
		ID: id, Lang: lang, Model: model,
		CompileOk: pass, RuntimeOk: pass, StdoutOk: pass,
	}
}

func findUplift(rows []UpliftRow, model, lang string) *UpliftRow {
	for i := range rows {
		if rows[i].Model == model && rows[i].Lang == lang {
			return &rows[i]
		}
	}
	return nil
}

func TestComputeUplift_SharedBenchmarksOnly(t *testing.T) {
	// standard ran A,B,C; agent ran B,C,D. Shared = {B,C}. D and A must be ignored.
	standard := []*BenchmarkResult{
		res("m1", "ailang", "A", false), // A: only standard → dropped
		res("m1", "ailang", "B", false), // B: std fail
		res("m1", "ailang", "C", true),  // C: std pass
	}
	agent := []*BenchmarkResult{
		res("m1", "ailang", "B", true), // B: agent pass
		res("m1", "ailang", "C", true), // C: agent pass
		res("m1", "ailang", "D", true), // D: only agent → dropped
	}
	rows := ComputeUplift(standard, agent)
	u := findUplift(rows, "m1", "ailang")
	if u == nil {
		t.Fatal("expected an uplift row for m1/ailang")
	}
	if u.SharedBenchmarks != 2 {
		t.Fatalf("shared should be 2 (B,C), got %d", u.SharedBenchmarks)
	}
	// standard: B=0, C=1 → macro-avg 0.5 ; agent: B=1, C=1 → 1.0 ; uplift 0.5
	if u.StandardPass != 0.5 || u.AgentPass != 1.0 || u.Uplift != 0.5 {
		t.Fatalf("std=%.3f agent=%.3f uplift=%.3f; want 0.5/1.0/0.5", u.StandardPass, u.AgentPass, u.Uplift)
	}
}

func TestComputeUplift_MatchingIdentityOnly(t *testing.T) {
	// `or-deepseek` (standard) vs `opencode-or-deepseek` (agent) must NOT pair —
	// different identity (a harness comparison, not uplift).
	standard := []*BenchmarkResult{res("or-deepseek", "ailang", "A", true)}
	agent := []*BenchmarkResult{res("opencode-or-deepseek", "ailang", "A", true)}
	rows := ComputeUplift(standard, agent)
	if len(rows) != 0 {
		t.Fatalf("mismatched identity must not produce an uplift row, got %d: %+v", len(rows), rows)
	}
}

func TestComputeUplift_MultiTrialMacroAverage(t *testing.T) {
	// Benchmark A has 4 trials, B has 1 trial. Macro-average weights A and B equally,
	// so A's extra trials must NOT dominate.
	standard := []*BenchmarkResult{
		res("m", "ailang", "A", true), res("m", "ailang", "A", true),
		res("m", "ailang", "A", false), res("m", "ailang", "A", false), // A std = 2/4 = 0.5
		res("m", "ailang", "B", false), // B std = 0
	}
	agent := []*BenchmarkResult{
		res("m", "ailang", "A", true), res("m", "ailang", "A", true),
		res("m", "ailang", "A", true), res("m", "ailang", "A", true), // A agent = 1.0
		res("m", "ailang", "B", true), // B agent = 1.0
	}
	rows := ComputeUplift(standard, agent)
	u := findUplift(rows, "m", "ailang")
	if u == nil {
		t.Fatal("expected uplift row")
	}
	// standard macro-avg = (0.5 + 0)/2 = 0.25 ; agent = (1.0 + 1.0)/2 = 1.0
	if u.StandardPass != 0.25 || u.AgentPass != 1.0 || u.Uplift != 0.75 {
		t.Fatalf("std=%.3f agent=%.3f uplift=%.3f; want 0.25/1.0/0.75", u.StandardPass, u.AgentPass, u.Uplift)
	}
}

func TestComputeUplift_LanguageScoped(t *testing.T) {
	// A model with both ailang and python shared benchmarks yields two rows,
	// each scoped to its own language (the axis is held constant).
	standard := []*BenchmarkResult{
		res("m", "ailang", "A", false),
		res("m", "python", "A", true),
	}
	agent := []*BenchmarkResult{
		res("m", "ailang", "A", true),
		res("m", "python", "A", true),
	}
	rows := ComputeUplift(standard, agent)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows (ailang, python), got %d", len(rows))
	}
	if a := findUplift(rows, "m", "ailang"); a == nil || a.Uplift != 1.0 {
		t.Fatalf("ailang uplift want 1.0, got %+v", a)
	}
	if p := findUplift(rows, "m", "python"); p == nil || p.Uplift != 0.0 {
		t.Fatalf("python uplift want 0.0, got %+v", p)
	}
	// deterministic order: ailang before python
	if rows[0].Lang != "ailang" || rows[1].Lang != "python" {
		t.Fatalf("rows not deterministically ordered: %+v", rows)
	}
}

func TestComputeUplift_NoAgentRun(t *testing.T) {
	// A model that only ran standard produces no uplift row.
	standard := []*BenchmarkResult{res("only-standard", "ailang", "A", true)}
	rows := ComputeUplift(standard, nil)
	if len(rows) != 0 {
		t.Fatalf("model without an agent run must not appear, got %+v", rows)
	}
}
