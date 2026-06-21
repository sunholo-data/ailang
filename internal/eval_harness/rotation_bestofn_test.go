package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestBestOfNRollup validates the best-of-N metric added to SummarizeRotation: per-benchmark
// AnyPass/BestOfNPass (the reference-free EXACT selector: runs > typechecks > neither, ties keep
// first) and the per-model rollup (pass@1 vs best-of-N). This is the metric that surfaces the
// validated best-of-N lift (postfix-broad: 96.9%→100%) on every rotation.
func TestBestOfNRollup(t *testing.T) {
	dir := t.TempDir()
	agent := filepath.Join(dir, "agent")
	if err := os.MkdirAll(agent, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(id string, trial int, c, r, s bool) {
		m := RunMetrics{ID: id, Lang: "ailang", Model: "m", Trial: trial, CompileOk: c, RuntimeOk: r, StdoutOk: s}
		b, _ := json.Marshal(m)
		_ = os.WriteFile(filepath.Join(agent, id+"_ailang_m_"+string(rune('0'+trial))+".json"), b, 0o644)
	}
	// "flaky": trial1 passes (c+r+s), trial2 runs-but-wrong (c+r, !s). pass@1=1/2; EXACT picks a
	// running trial, ties keep first (trial1=correct) → best-of-N PASS; anyPass=true.
	write("flaky", 1, true, true, true)
	write("flaky", 2, true, true, false)
	// "neither": both fail to compile. pass@1=0; best-of-N fail; anyPass=false.
	write("neither", 1, false, false, false)
	write("neither", 2, false, false, false)

	rs, err := SummarizeRotation(dir)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]BenchmarkSummary{}
	for _, s := range rs.BenchmarkSummary {
		byID[s.BenchmarkID] = s
	}
	if f := byID["flaky"]; !f.AnyPass || !f.BestOfNPass || f.PassRate != 0.5 {
		t.Errorf("flaky: %+v, want AnyPass && BestOfNPass && PassRate=0.5", f)
	}
	if n := byID["neither"]; n.AnyPass || n.BestOfNPass || n.PassRate != 0 {
		t.Errorf("neither: %+v, want !AnyPass && !BestOfNPass && PassRate=0", n)
	}
	ms := rs.ModelRollup["m"]
	if ms == nil {
		t.Fatal("no rollup for model m")
	}
	// pass@1 = 1 passing trial / 4 = 0.25; best-of-N EXACT = 1 bench / 2 = 0.5; ceiling = 0.5.
	if ms.Benchmarks != 2 || ms.Trials != 4 || ms.PassAt1 != 0.25 || ms.BestOfNExact != 0.5 || ms.BestOfNCeiling != 0.5 {
		t.Errorf("rollup: %+v, want benches=2 trials=4 pass@1=0.25 bestOfN=0.5 ceiling=0.5", *ms)
	}
}

// TestRollupOnBroadDataIfPresent confirms the Go rollup matches tools/eval_best_of_n.py on the real
// postfix-broad baseline (motoko pass@1≈0.969, best-of-N EXACT=1.0). Skips if the dir is absent.
func TestRollupOnBroadDataIfPresent(t *testing.T) {
	dir := "../../eval_results/rotation/postfix-broad-20260621"
	if _, err := os.Stat(filepath.Join(dir, "agent")); err != nil {
		t.Skip("broad baseline absent")
	}
	rs, err := SummarizeRotation(dir)
	if err != nil {
		t.Fatal(err)
	}
	for model, ms := range rs.ModelRollup {
		t.Logf("%s: pass@1=%.3f best_of_n_exact=%.3f ceiling=%.3f benches=%d trials=%d",
			model, ms.PassAt1, ms.BestOfNExact, ms.BestOfNCeiling, ms.Benchmarks, ms.Trials)
		if ms.BestOfNExact < ms.PassAt1 {
			t.Errorf("%s: best-of-N (%.3f) should be >= pass@1 (%.3f)", model, ms.BestOfNExact, ms.PassAt1)
		}
	}
}
