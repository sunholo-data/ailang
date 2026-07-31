package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

func writeRow(t *testing.T, dir, name string, m RunMetrics) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTallyCosts_SplitsByProvenance is the guard on the headline number: a
// subscription lane's cost must never be added to metered spend, and an
// unlabelled legacy row must not be assumed metered either.
func TestTallyCosts_SplitsByProvenance(t *testing.T) {
	root := t.TempDir()
	writeRow(t, filepath.Join(root, "standard"), "a", RunMetrics{
		Model: "gpt5-6-sol", CostUSD: 2.00, CostProvenance: string(executor.CostMetered)})
	writeRow(t, filepath.Join(root, "agent"), "b", RunMetrics{
		Model: "gpt5-6-luna", EvalMode: "agent", CostUSD: 8.36,
		CostProvenance: string(executor.CostListPriceEquivalent)})
	writeRow(t, filepath.Join(root, "agent"), "c", RunMetrics{
		Model: "local-gemma", EvalMode: "agent", CostUSD: 0,
		CostProvenance: string(executor.CostFreeLocal)})
	// A row banked before the label existed.
	writeRow(t, filepath.Join(root, "standard"), "d", RunMetrics{
		Model: "or-glm-5-2", CostUSD: 1.50})

	tally, err := TallyCosts(root)
	if err != nil {
		t.Fatalf("TallyCosts: %v", err)
	}
	if tally.Metered != 2.00 {
		t.Errorf("Metered = %v, want 2.00 (subscription and unknown must stay out)", tally.Metered)
	}
	if tally.ListPriceEquivalent != 8.36 {
		t.Errorf("ListPriceEquivalent = %v, want 8.36", tally.ListPriceEquivalent)
	}
	if tally.UnknownRuns != 1 || tally.UnknownCost != 1.50 {
		t.Errorf("unknown = %d runs / $%v, want 1 / $1.50 — held apart, not summed into metered",
			tally.UnknownRuns, tally.UnknownCost)
	}
	if tally.FreeLocalRuns != 1 {
		t.Errorf("FreeLocalRuns = %d, want 1", tally.FreeLocalRuns)
	}
	if tally.TotalRuns != 4 {
		t.Errorf("TotalRuns = %d, want 4", tally.TotalRuns)
	}
	if got := tally.ByMode["agent"]; got != 2 {
		t.Errorf("agent runs = %d, want 2", got)
	}

	out := tally.Format()
	for _, want := range []string{"METERED (actually billed)", "$2.00", "subscription — not billed", "NOT counted as spend"} {
		if !strings.Contains(out, want) {
			t.Errorf("Format() missing %q:\n%s", want, out)
		}
	}
}

// TestTallyCosts_EmptyDirIsSilent: a run that banked nothing must not print a
// tally, and must not error.
func TestTallyCosts_EmptyDirIsSilent(t *testing.T) {
	tally, err := TallyCosts(t.TempDir())
	if err != nil {
		t.Fatalf("TallyCosts on empty dir: %v", err)
	}
	if s := tally.Format(); s != "" {
		t.Errorf("Format() on empty tally = %q, want empty", s)
	}
}

// TestTallyCosts_SkipsMalformed: a corrupt result file must not take down the
// cost report at the end of an otherwise-complete run.
func TestTallyCosts_SkipsMalformed(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "standard")
	writeRow(t, dir, "good", RunMetrics{Model: "m", CostUSD: 1, CostProvenance: string(executor.CostMetered)})
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	tally, err := TallyCosts(root)
	if err != nil {
		t.Fatalf("TallyCosts: %v", err)
	}
	if tally.TotalRuns != 1 || tally.Metered != 1 {
		t.Errorf("got %d runs / $%v, want 1 / $1 (malformed row skipped)", tally.TotalRuns, tally.Metered)
	}
}
