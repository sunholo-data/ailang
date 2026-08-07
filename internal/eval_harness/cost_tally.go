package eval_harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// CostTally is the end-of-run cost report for a suite, split by whether the
// dollars were actually charged.
//
// The split is the whole point. A single grand total is misleading on this rig:
// agent-mode codex and claude authenticate by subscription, so their cost_usd is
// real arithmetic over real tokens that nobody paid. Summing that with metered
// OpenRouter/OpenAI spend produces a number that is neither a bill nor a
// list price. See executor.CostProvenance.
type CostTally struct {
	// Metered is spend an account was genuinely charged for — the only figure
	// that answers "what did this run cost us".
	Metered float64 `json:"metered_usd"`
	// ListPriceEquivalent is what the subscription lanes would have cost at list
	// price. Useful for comparing models on equal terms; not money.
	ListPriceEquivalent float64 `json:"list_price_equivalent_usd"`
	// FreeLocal is always 0 by construction (on-device models have no marginal
	// token cost); the run count is what carries the signal.
	FreeLocalRuns int `json:"free_local_runs"`
	// UnknownRuns counts rows with no provenance label — every row banked before
	// 2026-07-30, plus any executor that could not classify its auth lane. Their
	// dollars are held in UnknownCost rather than silently added to Metered.
	UnknownRuns int     `json:"unknown_runs"`
	UnknownCost float64 `json:"unknown_usd"`

	TotalRuns  int            `json:"total_runs"`
	ByMode     map[string]int `json:"runs_by_mode"`
	perModel   map[string]*modelTally
	ModelCount int `json:"model_count"`
}

type modelTally struct {
	model      string
	cost       float64
	runs       int
	provenance string
}

// TallyCosts walks a finished suite's banked result files and aggregates cost by
// provenance. Malformed files are skipped — a cost report must never be the
// thing that fails a completed run.
func TallyCosts(outputDir string) (*CostTally, error) {
	files, err := resultFilesIn(outputDir)
	if err != nil {
		return nil, err
	}
	t := &CostTally{ByMode: map[string]int{}, perModel: map[string]*modelTally{}}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var m RunMetrics
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		t.TotalRuns++
		mode := m.EvalMode
		if mode == "" {
			mode = "standard"
		}
		t.ByMode[mode]++

		switch m.CostProvenance {
		case string(executor.CostMetered):
			t.Metered += m.CostUSD
		case string(executor.CostListPriceEquivalent):
			t.ListPriceEquivalent += m.CostUSD
		case string(executor.CostFreeLocal):
			t.FreeLocalRuns++
		default:
			// Absent or unrecognised. Held apart deliberately: assuming metered
			// is exactly the error the provenance field exists to prevent.
			t.UnknownRuns++
			t.UnknownCost += m.CostUSD
		}

		key := m.Model + "\x00" + mode
		mt := t.perModel[key]
		if mt == nil {
			mt = &modelTally{model: m.Model + " (" + mode + ")"}
			t.perModel[key] = mt
		}
		mt.cost += m.CostUSD
		mt.runs++
		if mt.provenance == "" {
			mt.provenance = m.CostProvenance
		}
	}
	t.ModelCount = len(t.perModel)
	return t, nil
}

// Format renders the tally for the end of a suite run. Returns "" when there is
// nothing to report, so callers can print unconditionally.
func (t *CostTally) Format() string {
	if t == nil || t.TotalRuns == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Cost tally\n")

	rows := make([]*modelTally, 0, len(t.perModel))
	for _, mt := range t.perModel {
		rows = append(rows, mt)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].cost > rows[j].cost })
	for _, mt := range rows {
		label := provenanceLabel(mt.provenance)
		fmt.Fprintf(&b, "  %-38s %8.4f  %4d runs  %s\n", mt.model, mt.cost, mt.runs, label)
	}

	fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 72))
	fmt.Fprintf(&b, "  %-38s $%.2f\n", "METERED (actually billed)", t.Metered)
	if t.ListPriceEquivalent > 0 {
		fmt.Fprintf(&b, "  %-38s $%.2f   subscription — not billed\n",
			"list-price-equivalent", t.ListPriceEquivalent)
	}
	if t.UnknownRuns > 0 {
		fmt.Fprintf(&b, "  %-38s $%.2f   %d runs unlabelled — NOT counted as spend\n",
			"unknown provenance", t.UnknownCost, t.UnknownRuns)
	}
	if t.FreeLocalRuns > 0 {
		fmt.Fprintf(&b, "  %-38s %d runs, no marginal cost\n", "free-local", t.FreeLocalRuns)
	}
	return b.String()
}

func provenanceLabel(p string) string {
	switch p {
	case string(executor.CostMetered):
		return "metered"
	case string(executor.CostListPriceEquivalent):
		return "list-price-equiv"
	case string(executor.CostFreeLocal):
		return "free-local"
	default:
		return "unknown"
	}
}

// resultFilesIn returns every banked result JSON under outputDir, skipping the
// summary files the suite writes alongside them. Mirrors SummarizeRotation's
// walk so the two never disagree about what counts as a result.
func resultFilesIn(outputDir string) ([]string, error) {
	var files []string
	for _, mode := range []string{"standard", "agent"} {
		direct, _ := filepath.Glob(filepath.Join(outputDir, mode, "*.json"))
		files = append(files, direct...)
		condDirs, _ := filepath.Glob(filepath.Join(outputDir, mode, "*"))
		for _, cd := range condDirs {
			if info, err := os.Stat(cd); err == nil && info.IsDir() {
				condFiles, _ := filepath.Glob(filepath.Join(cd, "*.json"))
				files = append(files, condFiles...)
			}
		}
	}
	out := files[:0]
	for _, f := range files {
		base := filepath.Base(f)
		if base == "summary.json" || base == "baseline.json" {
			continue
		}
		out = append(out, f)
	}
	return out, nil
}
