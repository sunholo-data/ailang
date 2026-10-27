package eval_analysis

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

type evidenceAccumulator struct {
	stats      *EvidenceStats
	benchmarks map[string]bool
	core       map[string]bool
	times      map[string][]float64
	first      time.Time
	last       time.Time
}

func newEvidenceAccumulator() *evidenceAccumulator {
	return &evidenceAccumulator{
		stats: &EvidenceStats{Versions: map[string]int{}, PromptVersions: map[string]int{},
			Outcomes: map[string]int{}, Costs: map[string]*EvidenceCost{}, Timing: map[string]*EvidenceTiming{}},
		benchmarks: map[string]bool{}, core: map[string]bool{}, times: map[string][]float64{},
	}
}

// buildBenchmarkEvidence deliberately leaves row selection to LoadRows. Sorting
// only stabilizes sums and output; it does not deduplicate trials a second time.
func buildBenchmarkEvidence(results []*BenchmarkResult, tiers map[string]string, requiredCore []string) *BenchmarkEvidence {
	core := map[string]bool{}
	for _, id := range requiredCore {
		core[id] = true
	}
	ordered := append([]*BenchmarkResult(nil), results...)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		ak := []string{a.Model, a.EvalMode, a.Executor, a.Lang, a.ID, a.Timestamp.UTC().Format(time.RFC3339Nano), a.AilangVersion}
		bk := []string{b.Model, b.EvalMode, b.Executor, b.Lang, b.ID, b.Timestamp.UTC().Format(time.RFC3339Nano), b.AilangVersion}
		for k := range ak {
			if ak[k] != bk[k] {
				return ak[k] < bk[k]
			}
		}
		if a.Seed != b.Seed {
			return a.Seed < b.Seed
		}
		return a.Trial < b.Trial
	})
	rows := map[string]*EvidenceRow{}
	accumulators := map[*EvidenceStats]*evidenceAccumulator{}
	for _, r := range ordered {
		mode, harness := "standard", "direct_api"
		if r.EvalMode == "agent" {
			mode, harness = "agent", r.Executor
			if harness == "" {
				harness = "unknown"
			}
		}
		// Structured JSON identity avoids delimiter collisions in model names.
		id := evidenceIdentity(r.Model, mode, harness)
		row := rows[id]
		if row == nil {
			row = &EvidenceRow{ID: id, Model: r.Model, Mode: mode, Harness: harness, Scopes: map[string]map[string]*EvidenceStats{}}
			rows[id] = row
		}
		tier := tiers[r.ID]
		if tier == "" {
			tier = "unclassified"
		}
		for _, scope := range []string{"all", tier} {
			if row.Scopes[scope] == nil {
				row.Scopes[scope] = map[string]*EvidenceStats{}
			}
			stats := row.Scopes[scope][r.Lang]
			if stats == nil {
				a := newEvidenceAccumulator()
				stats = a.stats
				row.Scopes[scope][r.Lang], accumulators[stats] = stats, a
			}
			accumulators[stats].add(r, core)
		}
	}
	for _, a := range accumulators {
		a.finish()
	}
	out := &BenchmarkEvidence{SchemaVersion: 1, RequiredCoreIDs: evidenceSortedKeys(core), Rows: []*EvidenceRow{}}
	for _, id := range evidenceSortedKeys(rows) {
		out.Rows = append(out.Rows, rows[id])
	}
	return out
}

func (a *evidenceAccumulator) add(r *BenchmarkResult, required map[string]bool) {
	s := a.stats
	s.Runs++
	if r.Passed() {
		s.Passes++
	}
	if r.FirstAttemptOk {
		s.FirstAttemptPasses++
	} else if r.Passed() && !r.RepairUsed {
		// A legacy successful row with neither first-attempt nor repair flags
		// does not establish a measured first-attempt failure.
		s.FirstAttemptUnknown++
	}
	if r.RepairUsed {
		s.Repairs++
	}
	a.benchmarks[r.ID] = true
	if required[r.ID] {
		a.core[r.ID] = true
	}
	if r.AilangVersion == "" {
		s.UnknownVersionRuns++
	} else {
		s.Versions[r.AilangVersion]++
	}
	if r.PromptVersion != "" {
		s.PromptVersions[r.PromptVersion]++
	}
	if !r.Timestamp.IsZero() {
		if a.first.IsZero() || r.Timestamp.Before(a.first) {
			a.first = r.Timestamp
		}
		if a.last.IsZero() || !r.Timestamp.Before(a.last) {
			a.last, s.LatestVersion = r.Timestamp, r.AilangVersion
		}
	}
	// Legacy integer fields lose JSON presence. A zero cannot establish that
	// token usage was actually measured, so retain positive-telemetry coverage.
	if r.OutputTokens > 0 {
		s.OutputTokens += r.OutputTokens
		s.OutputTokenSamples++
	}
	if r.TotalTokens > 0 {
		s.TotalTokens += r.TotalTokens
		s.TotalTokenSamples++
	}
	category := r.ErrorCategory
	if category == "" {
		category = "unclassified"
	}
	s.Outcomes[category]++
	a.addCost(r)
	if r.LLMWallMs > 0 {
		a.times["generation_wall"] = append(a.times["generation_wall"], float64(r.LLMWallMs))
	}
	if r.EvalMode == "agent" && r.SuccessAtMs > 0 && r.Passed() {
		a.times["agent_success"] = append(a.times["agent_success"], float64(r.SuccessAtMs))
	}
	if r.DurationMs > 0 {
		a.times["recorded_duration"] = append(a.times["recorded_duration"], float64(r.DurationMs))
	}
}

func (a *evidenceAccumulator) addCost(r *BenchmarkResult) {
	kind := r.CostProvenance
	switch kind {
	case "metered", "list-price-equivalent", "free-local":
	default:
		kind = "unknown"
	}
	c := a.stats.Costs[kind]
	if c == nil {
		c = &EvidenceCost{Complete: true}
		a.stats.Costs[kind] = c
	}
	c.Runs++
	if r.Passed() {
		c.Passes++
	}
	valid := !math.IsNaN(r.CostUSD) && !math.IsInf(r.CostUSD, 0) && r.CostUSD >= 0
	if !valid || (kind == "unknown" && r.CostUSD == 0) {
		c.Complete = false
		return
	}
	c.TotalUSD += r.CostUSD
}

func (a *evidenceAccumulator) finish() {
	s := a.stats
	s.Benchmarks, s.CoreCompleted = evidenceSortedKeys(a.benchmarks), evidenceSortedKeys(a.core)
	if !a.first.IsZero() {
		s.FirstRun = a.first.UTC().Format(time.RFC3339Nano)
		s.LastRun = a.last.UTC().Format(time.RFC3339Nano)
	}
	for _, c := range s.Costs {
		if c.Complete && c.Passes > 0 {
			v := c.TotalUSD / float64(c.Passes)
			c.PerSuccessUSD = &v
		}
	}
	for kind, values := range a.times {
		s.Timing[kind] = &EvidenceTiming{Samples: len(values), MedianMs: median(values)}
	}
}

func evidenceSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// evidenceTierIndex supplements standard-only gallery tiers for agent-only tasks.
func evidenceTierIndex(results []*BenchmarkResult, known map[string]string) map[string]string {
	out := map[string]string{}
	for _, r := range results {
		if _, exists := out[r.ID]; exists {
			continue
		}
		out[r.ID] = known[r.ID]
		if out[r.ID] == "" && filepath.Base(r.ID) == r.ID && !strings.ContainsAny(r.ID, `/\`) {
			if spec, err := eval_harness.LoadSpec(filepath.Join("benchmarks", r.ID+".yml")); err == nil {
				out[r.ID] = spec.Tier
			}
		}
	}
	return out
}
