package eval_analysis

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// graderArtifactFlags marks benchmarks whose difficulty is a benchmark/grader
// artifact rather than genuine hardness (M-EVAL-DASHBOARD-REDESIGN guardrail), so
// the dashboard can badge them "measurement artifact — fix pending" instead of
// reading them as real difficulty. The bool-casing/numeric cases were fixed by
// M-EVAL-OUTPUT-NORMALIZE; these two remain.
var graderArtifactFlags = map[string]string{
	"contract_sorted_merge":  "set-repr",              // Python set {} vs expected list []
	"decision_block_capture": "free-text-exact-match", // free-text justification graded by exact match
}

// buildRatingsBlock fits per-mode ELO ratings over the run's results and returns
// the `ratings` block for latest.json: a model capability leaderboard and a
// benchmark difficulty list (with band, saturation, pass-rate, and grader flag)
// per mode. Mode-separated because standard and agent saturate differently.
func buildRatingsBlock(standard, agent []*BenchmarkResult) map[string]interface{} {
	out := map[string]interface{}{}
	if m := ratingsForMode(standard); m != nil {
		out["standard"] = m
	}
	if m := ratingsForMode(agent); m != nil {
		out["agent"] = m
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ratingsForMode fits the combined (all-language) leaderboard for a mode and
// also attaches a per-language "byLang" sub-map so consumers can read the
// AILANG-vs-Python story separately. The top-level models/benchmarks/saturation
// keys stay unchanged (both languages blended) for backward compatibility.
func ratingsForMode(results []*BenchmarkResult) map[string]interface{} {
	if len(results) == 0 {
		return nil
	}
	block := fitLeaderboard(results)

	// Per-language fits: group by .Lang and fit each language independently so a
	// benchmark's difficulty and a model's capability are measured against a
	// single language, not a python+ailang blend.
	byLangResults := map[string][]*BenchmarkResult{}
	for _, r := range results {
		byLangResults[r.Lang] = append(byLangResults[r.Lang], r)
	}
	byLang := map[string]interface{}{}
	for lang, rs := range byLangResults {
		if lang == "" {
			continue
		}
		if lb := fitLeaderboard(rs); lb != nil {
			byLang[lang] = lb
		}
	}
	if len(byLang) > 0 {
		block["byLang"] = byLang
	}
	return block
}

// fitLeaderboard fits ELO over the given results and returns the leaderboard
// block: {models, benchmarks, saturation}. A trial Pass = CompileOk && RuntimeOk
// && StdoutOk. This is the reusable body shared by the combined fit and each
// per-language fit in ratingsForMode.
func fitLeaderboard(results []*BenchmarkResult) map[string]interface{} {
	if len(results) == 0 {
		return nil
	}
	trials := make([]eval_harness.Trial, 0, len(results))
	pass := map[string][2]int{} // benchmark -> [passed, total]
	for _, r := range results {
		ok := r.CompileOk && r.RuntimeOk && r.StdoutOk
		trials = append(trials, eval_harness.Trial{Model: r.Model, Bench: r.ID, Pass: ok})
		v := pass[r.ID]
		v[1]++
		if ok {
			v[0]++
		}
		pass[r.ID] = v
	}
	modelRatings, benchRatings := eval_harness.FitFromTrials(trials)

	models := make([]map[string]interface{}, 0, len(modelRatings))
	for id, elo := range modelRatings {
		models = append(models, map[string]interface{}{
			"id": id, "elo": round1(elo), "band": eval_harness.Band(elo),
		})
	}
	sort.Slice(models, func(i, j int) bool { return models[i]["elo"].(float64) > models[j]["elo"].(float64) })

	benches := make([]map[string]interface{}, 0, len(benchRatings))
	saturated := 0
	for id, elo := range benchRatings {
		band := eval_harness.Band(elo)
		isSat := band == "Trivial"
		if isSat {
			saturated++
		}
		entry := map[string]interface{}{
			"id": id, "elo": round1(elo), "band": band, "saturated": isSat,
			"passRate": passRate(pass[id]),
		}
		if flag, ok := graderArtifactFlags[id]; ok {
			entry["graderFlag"] = flag
		}
		benches = append(benches, entry)
	}
	sort.Slice(benches, func(i, j int) bool { return benches[i]["elo"].(float64) > benches[j]["elo"].(float64) })

	return map[string]interface{}{
		"models":     models,
		"benchmarks": benches,
		"saturation": map[string]interface{}{
			"saturated":      saturated,
			"discriminating": len(benchRatings) - saturated,
			"total":          len(benchRatings),
		},
	}
}

func passRate(v [2]int) float64 {
	if v[1] == 0 {
		return 0
	}
	return round1(float64(v[0]) / float64(v[1]) * 100)
}

func round1(x float64) float64 {
	return float64(int(x*10+0.5)) / 10
}
