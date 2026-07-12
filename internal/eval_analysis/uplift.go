package eval_analysis

import "sort"

// M-EVAL-VALIDITY-DISCIPLINE (W3): like-for-like agent uplift.
//
// "What does agent mode add?" is only a valid question when a model is compared to
// ITSELF, on the SAME benchmarks. Comparing a cheap agent model's score to a
// frontier standard model's score, or blending in benchmarks only one mode ran,
// produces a bogus "uplift" (this is the exact bug this discipline exists to stop).
//
// ComputeUplift enforces both invariants:
//   - matching identity: the model string must match EXACTLY across modes, so
//     `or-deepseek` (standard API) and `opencode-or-deepseek` (agent CLI) do NOT
//     pair — that is a *harness* comparison, not uplift, and is excluded here.
//   - shared benchmarks: pass rates are computed over ONLY the (benchmark) set both
//     modes ran for that (model, lang); benchmarks unique to one mode are dropped.
//   - held axis: every row is scoped to a single language.

// UpliftRow is a like-for-like standard→agent comparison for one (model, lang).
type UpliftRow struct {
	Model            string  `json:"model"`
	Lang             string  `json:"lang"`
	SharedBenchmarks int     `json:"sharedBenchmarks"` // benchmarks BOTH modes ran for this (model,lang)
	StandardPass     float64 `json:"standardPass"`     // macro-avg pass rate over the shared set, 0..1
	AgentPass        float64 `json:"agentPass"`
	Uplift           float64 `json:"uplift"` // agentPass - standardPass
}

// upliftKey identifies one (model, lang, benchmark) cell.
type upliftKey struct{ model, lang, bench string }

// ComputeUplift returns per-(model, lang) uplift rows over the shared-benchmark
// intersection, sorted deterministically by (model, lang). A (model, lang) is
// included only if it ran in BOTH modes and shares ≥1 benchmark. Pass rate is the
// macro-average of per-benchmark pass fractions (equal weight per benchmark) so a
// benchmark with many trials doesn't dominate.
func ComputeUplift(standard, agent []*BenchmarkResult) []UpliftRow {
	std := aggUpliftPass(standard)
	ag := aggUpliftPass(agent)

	type ml struct{ model, lang string }
	group := func(agg map[upliftKey][2]int) map[ml]map[string][2]int {
		out := map[ml]map[string][2]int{}
		for k, v := range agg {
			key := ml{k.model, k.lang}
			if out[key] == nil {
				out[key] = map[string][2]int{}
			}
			out[key][k.bench] = v
		}
		return out
	}
	stdML := group(std)
	agML := group(ag)

	var rows []UpliftRow
	for key, sbench := range stdML {
		abench, ok := agML[key]
		if !ok {
			continue // model+lang didn't run in agent mode → no like-for-like uplift
		}
		var shared []string
		for b := range sbench {
			if _, ok := abench[b]; ok {
				shared = append(shared, b)
			}
		}
		if len(shared) == 0 {
			continue
		}
		var sSum, aSum float64
		for _, b := range shared {
			sSum += passFraction(sbench[b])
			aSum += passFraction(abench[b])
		}
		n := float64(len(shared))
		sp := round3(sSum / n)
		ap := round3(aSum / n)
		rows = append(rows, UpliftRow{
			Model:            key.model,
			Lang:             key.lang,
			SharedBenchmarks: len(shared),
			StandardPass:     sp,
			AgentPass:        ap,
			Uplift:           round3(ap - sp),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Model != rows[j].Model {
			return rows[i].Model < rows[j].Model
		}
		return rows[i].Lang < rows[j].Lang
	})
	return rows
}

// aggUpliftPass accumulates [passed, total] trials per (model, lang, benchmark).
func aggUpliftPass(results []*BenchmarkResult) map[upliftKey][2]int {
	m := map[upliftKey][2]int{}
	for _, r := range results {
		if r == nil || r.Model == "" || r.Lang == "" || r.ID == "" {
			continue
		}
		k := upliftKey{r.Model, r.Lang, r.ID}
		v := m[k]
		v[1]++
		if r.CompileOk && r.RuntimeOk && r.StdoutOk {
			v[0]++
		}
		m[k] = v
	}
	return m
}

// passFraction is passed/total for one benchmark's trials (0..1).
func passFraction(v [2]int) float64 {
	if v[1] == 0 {
		return 0
	}
	return float64(v[0]) / float64(v[1])
}

func round3(x float64) float64 {
	return float64(int(x*1000+0.5)) / 1000
}
