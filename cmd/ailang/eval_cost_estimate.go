// Package main: pre-flight cost estimation for `ailang eval-suite --dry-run`
// (M-EVAL-STANDARD-CONFIDENCE-GATING).
//
// Projects a $ total for a planned (models × benchmarks × langs × trials) run
// by looking up each benchmark's historical mean input/output tokens from the
// most recent baseline's banked results, and multiplying by each model's
// models.yml pricing. Benchmarks with no history (new benchmark, no baseline
// yet) fall back to a conservative flat default, explicitly flagged — never a
// silent omission (NO SILENT FALLBACKS, CLAUDE.md #2). This lives in cmd/ailang
// rather than internal/eval_harness because it needs eval_analysis.LoadResults
// for historical data, and internal/eval_analysis already imports
// internal/eval_harness (for the ELO fit) — the reverse import would cycle.
package main

import (
	"github.com/sunholo-data/ailang/internal/modelreg"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// Flat fallback used when a benchmark has no historical token data at all.
// Deliberately on the high side: a pre-flight estimate should err toward
// "this might cost more than you think," not under-warn before real spend.
const (
	defaultMeanInputTokens  = 3000.0
	defaultMeanOutputTokens = 1500.0
)

// benchmarkTokenMeans holds historical mean input/output tokens per
// benchmark ID, derived from a completed baseline's banked standard-mode
// results. Zero value (no baseline found) is valid and yields an all-default
// estimate rather than an error — a fresh checkout with no baseline yet
// should still produce a (rough) number, not a crash.
type benchmarkTokenMeans struct {
	input   map[string]float64
	output  map[string]float64
	covered map[string]bool // true = this benchmark had real history
}

// loadBenchmarkTokenMeans computes per-benchmark mean input/output tokens
// from the most recent available baseline's standard-mode results. Agent-mode
// rows are excluded — agent mode has its own weak-model cost story
// (agent_suite is deliberately cheap by model selection, not by this
// estimator) and mixing the two regimes' token profiles would distort both.
func loadBenchmarkTokenMeans() *benchmarkTokenMeans {
	means := &benchmarkTokenMeans{
		input:   map[string]float64{},
		output:  map[string]float64{},
		covered: map[string]bool{},
	}

	versions, err := eval_analysis.ListBaselines()
	if err != nil || len(versions) == 0 {
		return means
	}
	dir := filepath.Join("eval_results", "baselines", versions[0])
	results, err := eval_analysis.LoadResults(dir)
	if err != nil || len(results) == 0 {
		return means
	}

	sumIn := map[string]float64{}
	sumOut := map[string]float64{}
	count := map[string]int{}
	for _, r := range results {
		if r.EvalMode == "agent" {
			continue
		}
		sumIn[r.ID] += float64(r.InputTokens)
		sumOut[r.ID] += float64(r.OutputTokens)
		count[r.ID]++
	}
	for id, n := range count {
		if n == 0 {
			continue
		}
		means.input[id] = sumIn[id] / float64(n)
		means.output[id] = sumOut[id] / float64(n)
		means.covered[id] = true
	}
	return means
}

// costEstimate is the total projected result for a planned run.
type costEstimate struct {
	TotalUSD       float64
	Pairs          int // total (model, benchmark) pairs priced
	NoHistoryPairs int // pairs that fell back to the flat default
	UnknownPricing int // pairs where the model had no pricing in models.yml (priced as $0, flagged separately from NoHistoryPairs)
}

// estimateRunCostUSD projects a $ total for a planned (models × benchmarks ×
// langs × trials) run, using live historical data and the global models
// config. Thin wrapper over estimateRunCostUSDWithMeans so tests can inject
// a fixture instead of touching the filesystem/global state.
func estimateRunCostUSD(models, benchmarks []string, langs, trials int) costEstimate {
	return estimateRunCostUSDWithMeans(models, benchmarks, langs, trials, loadBenchmarkTokenMeans(), modelreg.GlobalModelsConfig)
}

// estimateRunCostUSDWithMeans is the pure computation: no filesystem or
// global-state access, so it's exactly hand-computable in a test. langs and
// trials scale the total uniformly — token means are per-benchmark-run, not
// tracked per-language, which is a rougher approximation than a per-language
// mean but is the right precision for a pre-flight order-of-magnitude
// estimate, not a promise.
func estimateRunCostUSDWithMeans(models, benchmarks []string, langs, trials int, means *benchmarkTokenMeans, cfg *eval_harness.ModelsConfig) costEstimate {
	var est costEstimate

	for _, m := range models {
		var pricing eval_harness.Pricing
		havePricing := false
		if cfg != nil {
			if mc, ok := cfg.Models[m]; ok {
				pricing = mc.Pricing
				havePricing = true
			}
		}
		for _, b := range benchmarks {
			est.Pairs++
			if !havePricing {
				est.UnknownPricing++
				continue
			}
			inTok, outTok := defaultMeanInputTokens, defaultMeanOutputTokens
			if means != nil && means.covered[b] {
				inTok, outTok = means.input[b], means.output[b]
			} else {
				est.NoHistoryPairs++
			}
			perRunUSD := (inTok*pricing.InputPer1K + outTok*pricing.OutputPer1K) / 1000.0
			est.TotalUSD += perRunUSD * float64(langs) * float64(trials)
		}
	}
	return est
}
