// Package eval_harness: SummarizeRotation aggregator (M-EVAL-OS-LONGITUDINAL Phase 3).
//
// Reads all RunMetrics JSON files in an output directory (e.g. one produced
// by `make eval-smoke ... --trials 3`), groups them by (benchmark_id, model,
// lang, condition), and writes a per-benchmark summary.json containing trial
// count, pass rate, and token distribution.
//
// The summary.json is the publishable artifact for Phase 4 (eval-trend
// candidates) and Phase 5 (eval-publish leaderboard). It's the difference
// between "we ran a benchmark once" and "we ran it N times and here's the
// distribution."
package eval_harness

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BenchmarkSummary aggregates the outcome of N trials of a single
// (benchmark, model, lang, condition) tuple.
type BenchmarkSummary struct {
	BenchmarkID string  `json:"benchmark_id"`
	Model       string  `json:"model"`
	Lang        string  `json:"lang"`
	Condition   string  `json:"condition,omitempty"`
	Trials      int     `json:"trials"`
	Passed      int     `json:"passed"`
	PassRate    float64 `json:"pass_rate"`

	// Best-of-N (M-AILANG-NATIVE-HARNESS P1, validated 96.9%→100% on postfix-broad-20260621).
	// AnyPass = the best-of-N ceiling (>=1 trial passed). BestOfNPass = the reference-free EXACT
	// selector outcome (pick the trial that runs > typechecks > neither, then check its stdout_ok) —
	// what `ailang select-best` / a deployed motoko-bestof would actually achieve. For benchmarks with
	// only flaky failures (the common post-fix case), BestOfNPass recovers the pass that pass@1 misses.
	AnyPass     bool `json:"any_pass"`
	BestOfNPass bool `json:"best_of_n_pass"`

	// Token distribution split by outcome. Failing runs often thrash to
	// much higher token counts than passing ones; reporting them separately
	// makes that signal visible.
	TokensPassMean   float64 `json:"tokens_pass_mean,omitempty"`
	TokensPassStddev float64 `json:"tokens_pass_stddev,omitempty"`
	TokensFailMean   float64 `json:"tokens_fail_mean,omitempty"`

	// Per-failure-category counts. Order-insensitive in the JSON encoding
	// because a map gives clean per-category access for Phase 4 filtering.
	ErrorCategories map[string]int `json:"error_categories,omitempty"`

	// Count of runs aborted by Phase-1 thrash detection.
	ThrashAborts int `json:"thrash_aborts,omitempty"`
	// TokensCacheUnaccounted counts trials excluded from the token KPIs because
	// they predate cache accounting (2026-08-11) and so cannot be decomposed into
	// fresh vs cached. Non-zero means the token means below rest on FEWER trials
	// than Trials — a shrunken sample stated out loud rather than a silent one.
	TokensCacheUnaccounted int `json:"tokens_cache_unaccounted,omitempty"`

	// InvalidExcluded counts rows dropped from Trials because they are not
	// measurements (RunMetrics.IsValid() == false — dominantly the api_error
	// backstop in validity.go). Trials/Passed/PassRate above are computed over
	// the SURVIVORS, so this is the same "shrunken sample stated out loud"
	// contract as TokensCacheUnaccounted.
	//
	// WHY THIS EXISTS. validity.go shipped the producer and the consumers, but
	// this aggregator — the one that feeds summary.json and from there the
	// published OS/Local leaderboard — never filtered. Its own doc comment
	// listed the consequence as known ("Published pass rates carried the crash
	// rate inside them") and it stayed true. Measured on the v0.34.0 rotation
	// (2026-09-02): motoko-local-qwen3-8-27b published 86.0% on AILANG against
	// pi's 92.7%, and was read as the weakest AILANG harness. 11 of its 12
	// failures were zero-token non-starts. On rows where the subject actually
	// ran, motoko is 98.7% — the BEST of the three, not the worst. The entire
	// reported gap was harness crash rate.
	InvalidExcluded int `json:"invalid_excluded,omitempty"`
	// InvalidReasons breaks InvalidExcluded down by Validity.Reason so an
	// operator can tell an ollama overload from a motoko startup crash without
	// opening the rows.
	InvalidReasons map[string]int `json:"invalid_reasons,omitempty"`
}

// ModelRollupStats is the per-model headline rollup: pass@1 vs best-of-N across all of a model's benchmarks.
// This surfaces the validated best-of-N lift (96.9%→100% on the standard set) on EVERY rotation/release
// instead of only via the manual tools/eval_best_of_n.py.
type ModelRollupStats struct {
	Benchmarks     int     `json:"benchmarks"`
	Trials         int     `json:"trials"`
	PassAt1        float64 `json:"pass_at_1"`         // passing trials / total trials
	BestOfNExact   float64 `json:"best_of_n_exact"`   // benchmarks the EXACT selector passes / benchmarks
	BestOfNCeiling float64 `json:"best_of_n_ceiling"` // benchmarks with >=1 pass / benchmarks
}

// RotationSummary is the top-level summary.json shape for an eval-suite run.
type RotationSummary struct {
	OutputDir        string `json:"output_dir"`
	TotalResultFiles int    `json:"total_result_files"`
	// InvalidExcludedTotal counts result files that were read but are not
	// measurements, so they are absent from every pass rate below. Non-zero is
	// a HARNESS health signal, not a model signal — see BenchmarkSummary.
	InvalidExcluded int `json:"invalid_excluded,omitempty"`
	// InvalidReasons breaks that total down by Validity.Reason.
	InvalidReasons map[string]int `json:"invalid_reasons,omitempty"`
	// UnmeasuredTuples counts (benchmark, model, lang, condition) groups whose
	// every row was invalid. They are omitted from BenchmarkSummary entirely
	// rather than published as 0%.
	UnmeasuredTuples int                          `json:"unmeasured_tuples,omitempty"`
	TrialsPerBench   int                          `json:"trials_per_benchmark"` // max trial number seen
	ModelRollup      map[string]*ModelRollupStats `json:"model_rollup,omitempty"`
	BenchmarkSummary []BenchmarkSummary           `json:"benchmarks"`
}

// bestOfNExact applies the reference-free exact selector to a (benchmark,model) tuple's trials:
// rank runs(2) > typechecks(1) > neither(0), pick the best (ties keep first), and report whether the
// picked trial passed (stdout_ok). Mirrors `ailang select-best` and tools/eval_best_of_n.py. Also
// returns anyPass = the ceiling (>=1 trial passed).
func bestOfNExact(trials []*RunMetrics) (selectedPass bool, anyPass bool) {
	best, bestScore := -1, -1
	for i, m := range trials {
		score := 0
		if m.RuntimeOk {
			score = 2
		} else if m.CompileOk {
			score = 1
		}
		if score > bestScore {
			bestScore, best = score, i
		}
		if m.StdoutOk {
			anyPass = true
		}
	}
	if best >= 0 && trials[best].StdoutOk {
		selectedPass = true
	}
	return selectedPass, anyPass
}

// SummarizeRotation walks outputDir/{standard,agent}/[<condition>/]*.json,
// loads each RunMetrics, groups by (benchmark_id, model, lang, condition),
// and writes a summary.json at outputDir/summary.json.
//
// Returns the computed summary so callers can inspect it without re-reading
// the file. Returns an error only on I/O failure; malformed individual
// result files are skipped with a stderr warning (eval-suite already logs
// per-file errors at write time, so duplicating here is noise).
func SummarizeRotation(outputDir string) (*RotationSummary, error) {
	// Walk both standard/ and agent/ subdirectories (and any condition nesting).
	pattern := filepath.Join(outputDir, "**", "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob result files: %w", err)
	}
	// fallback for filesystems where Go's ** matching is shallow
	if len(files) == 0 {
		standard, _ := filepath.Glob(filepath.Join(outputDir, "standard", "*.json"))
		agent, _ := filepath.Glob(filepath.Join(outputDir, "agent", "*.json"))
		files = append(standard, agent...)
		// Also handle condition-nested dirs (one level deeper)
		for _, mode := range []string{"standard", "agent"} {
			condDirs, _ := filepath.Glob(filepath.Join(outputDir, mode, "*"))
			for _, cd := range condDirs {
				if info, err := os.Stat(cd); err == nil && info.IsDir() {
					condFiles, _ := filepath.Glob(filepath.Join(cd, "*.json"))
					files = append(files, condFiles...)
				}
			}
		}
	}

	// Exclude summary.json itself if it already exists at the root.
	filtered := files[:0]
	for _, f := range files {
		if filepath.Base(f) == "summary.json" {
			continue
		}
		filtered = append(filtered, f)
	}
	files = filtered

	// Group by (benchmark, model, lang, condition).
	type groupKey struct {
		Benchmark, Model, Lang, Condition string
	}
	type groupAccum struct {
		Trials       []*RunMetrics
		MaxTrial     int
		ErrorCats    map[string]int
		ThrashAborts int
		// Invalid rows are counted but NOT appended to Trials: they are not
		// measurements, so they must not sit in a pass-rate denominator.
		Invalid        int
		InvalidReasons map[string]int
	}
	groups := map[groupKey]*groupAccum{}
	totalFiles := 0
	maxTrialSeen := 1

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m RunMetrics
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		totalFiles++
		key := groupKey{
			Benchmark: m.ID,
			Model:     m.Model,
			Lang:      m.Lang,
			Condition: m.Condition,
		}
		g := groups[key]
		if g == nil {
			g = &groupAccum{ErrorCats: map[string]int{}, InvalidReasons: map[string]int{}}
			groups[key] = g
		}
		// A row that is not a measurement is quarantined here rather than
		// deleted: it stays on disk as evidence of the bug (validity.go), and it
		// is still counted so the shrunken sample is visible — but it never
		// reaches Trials, so it cannot be charged to the model as a failure.
		if !m.IsValid() {
			g.Invalid++
			reason := m.InvalidReason()
			if reason == "" {
				reason = ReasonHarnessError
			}
			g.InvalidReasons[reason]++
			continue
		}
		g.Trials = append(g.Trials, &m)
		trial := m.Trial
		if trial < 1 {
			trial = 1
		}
		if trial > g.MaxTrial {
			g.MaxTrial = trial
		}
		if trial > maxTrialSeen {
			maxTrialSeen = trial
		}
		if m.ErrorCategory != "" && m.ErrorCategory != ErrorCategoryNone {
			g.ErrorCats[m.ErrorCategory]++
		}
		if m.ErrorCategory == ErrorCategoryThrashAborted {
			g.ThrashAborts++
		}
	}

	// Compute per-group summary.
	var summaries []BenchmarkSummary
	invalidTotal := 0
	unmeasuredTuples := 0
	invalidReasonTotals := map[string]int{}
	for _, g := range groups {
		invalidTotal += g.Invalid
		for reason, n := range g.InvalidReasons {
			invalidReasonTotals[reason] += n
		}
	}
	for key, g := range groups {
		// Every trial for this tuple was a non-measurement. Emitting a summary
		// here would publish 0/0 — which renders as 0% and is exactly the
		// phantom-failure shape validity.go exists to prevent. The tuple is
		// simply UNMEASURED, so it is omitted from the leaderboard and surfaced
		// in the top-level tally instead. It stays eligible for --skip-existing
		// retry, because no valid row was ever banked for it.
		if len(g.Trials) == 0 {
			unmeasuredTuples++
			continue
		}
		// Token KPIs count UNCACHED work only (Mark, 2026-08-11). A cache read
		// costs ~20% of a fresh token, so charging it as a whole one makes a run
		// that caches well look more expensive than one that does not — the metric
		// would penalise the behaviour we want, and penalise AILANG hardest, since
		// its large teaching prompt is the most cacheable thing in the system
		// (measured: total_tokens reads ~3.9x the real work at a 75% hit rate).
		//
		// Rows predating cache accounting are EXCLUDED, not assumed all-fresh:
		// counting them whole would inflate every historical row and manufacture
		// an improvement out of a schema change. The count is surfaced so a
		// shrunken sample is visible rather than silent.
		var passTokens, failTokens []int
		passed := 0
		unaccounted := 0
		for _, m := range g.Trials {
			isPass := m.CompileOk && m.RuntimeOk && m.StdoutOk
			if isPass {
				passed++
			}
			fresh, ok := m.FreshTokens()
			if !ok {
				unaccounted++
				continue
			}
			if isPass {
				passTokens = append(passTokens, fresh)
			} else {
				failTokens = append(failTokens, fresh)
			}
		}
		bestPass, anyPass := bestOfNExact(g.Trials)
		s := BenchmarkSummary{
			BenchmarkID:     key.Benchmark,
			Model:           key.Model,
			Lang:            key.Lang,
			Condition:       key.Condition,
			Trials:          len(g.Trials),
			Passed:          passed,
			PassRate:        float64(passed) / float64(len(g.Trials)),
			AnyPass:         anyPass,
			BestOfNPass:     bestPass,
			ErrorCategories: g.ErrorCats,
			ThrashAborts:    g.ThrashAborts,
		}
		if len(passTokens) > 0 {
			s.TokensPassMean = mean(passTokens)
			s.TokensPassStddev = stddev(passTokens, s.TokensPassMean)
		}
		if len(failTokens) > 0 {
			s.TokensFailMean = mean(failTokens)
		}
		s.TokensCacheUnaccounted = unaccounted
		s.InvalidExcluded = g.Invalid
		if len(g.InvalidReasons) > 0 {
			s.InvalidReasons = g.InvalidReasons
		}
		if len(s.ErrorCategories) == 0 {
			s.ErrorCategories = nil
		}
		summaries = append(summaries, s)
	}

	// Stable sort: benchmark id, then model, then lang, then condition.
	sort.Slice(summaries, func(i, j int) bool {
		a, b := summaries[i], summaries[j]
		if a.BenchmarkID != b.BenchmarkID {
			return a.BenchmarkID < b.BenchmarkID
		}
		if a.Model != b.Model {
			return a.Model < b.Model
		}
		if a.Lang != b.Lang {
			return a.Lang < b.Lang
		}
		return a.Condition < b.Condition
	})

	// Per-model rollup: pass@1 (trial-mean) vs best-of-N (EXACT selector) vs ceiling, across the
	// model's benchmarks. Makes the validated best-of-N lift visible on every rotation/release.
	type rollupAccum struct {
		benches, trials, passTrials, bestOfN, ceiling int
	}
	racc := map[string]*rollupAccum{}
	for _, s := range summaries {
		r := racc[s.Model]
		if r == nil {
			r = &rollupAccum{}
			racc[s.Model] = r
		}
		r.benches++
		r.trials += s.Trials
		r.passTrials += s.Passed
		if s.BestOfNPass {
			r.bestOfN++
		}
		if s.AnyPass {
			r.ceiling++
		}
	}
	rollup := map[string]*ModelRollupStats{}
	for model, r := range racc {
		ms := &ModelRollupStats{Benchmarks: r.benches, Trials: r.trials}
		if r.trials > 0 {
			ms.PassAt1 = float64(r.passTrials) / float64(r.trials)
		}
		if r.benches > 0 {
			ms.BestOfNExact = float64(r.bestOfN) / float64(r.benches)
			ms.BestOfNCeiling = float64(r.ceiling) / float64(r.benches)
		}
		rollup[model] = ms
	}
	if len(rollup) == 0 {
		rollup = nil
	}

	rs := &RotationSummary{
		OutputDir:        outputDir,
		TotalResultFiles: totalFiles,
		InvalidExcluded:  invalidTotal,
		UnmeasuredTuples: unmeasuredTuples,
		TrialsPerBench:   maxTrialSeen,
		ModelRollup:      rollup,
		BenchmarkSummary: summaries,
	}
	if len(invalidReasonTotals) > 0 {
		rs.InvalidReasons = invalidReasonTotals
	}

	// Write summary.json at the root of outputDir.
	summaryPath := filepath.Join(outputDir, "summary.json")
	data, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		return rs, fmt.Errorf("marshal summary: %w", err)
	}
	// Ensure outputDir exists (caller may have given us a not-yet-created path
	// during testing).
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return rs, fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(summaryPath, data, 0644); err != nil {
		return rs, fmt.Errorf("write summary.json: %w", err)
	}
	return rs, nil
}

// mean returns the arithmetic mean of ints, as float64. Returns 0 for empty input.
func mean(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0
	for _, x := range xs {
		sum += x
	}
	return float64(sum) / float64(len(xs))
}

// stddev returns the sample standard deviation. Returns 0 for n<2.
func stddev(xs []int, m float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var sq float64
	for _, x := range xs {
		d := float64(x) - m
		sq += d * d
	}
	return math.Sqrt(sq / float64(len(xs)-1))
}

// Used only to keep the import set tidy when `strings` becomes unused after
// future refactoring — leaving here as a no-op anchor to avoid go vet warnings
// during incremental editing.
var _ = strings.HasPrefix
