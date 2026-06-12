// Package main: `ailang eval-trend` subcommand (M-EVAL-OS-LONGITUDINAL Phase 4).
//
// Surfaces persistent-failure candidates from N-trial rotation data. Reads
// summary.json files (produced by Phase 3 --trials N runs) and emits a
// structured table of (benchmark, error_category, n_fail/n_trials) tuples
// where the same failure mode recurs across multiple trials. These are the
// inputs to the language-design feedback loop: each candidate becomes a
// candidate for a design doc + stdlib/prompt/syntax fix, then re-measured
// against the same (model, benchmark) at the next release.
//
// Output formats: text table (default) or JSON (--json) for piping into
// other tools.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// runEvalTrend is the top-level dispatcher for `ailang eval-trend <action>`.
func runEvalTrend() {
	if flag.NArg() < 2 {
		printEvalTrendUsage()
		os.Exit(1)
	}
	action := flag.Arg(1)
	switch action {
	case "candidates":
		runEvalTrendCandidates()
	case "tier-saturation":
		runEvalTrendSaturation()
	default:
		fmt.Fprintf(os.Stderr, "Unknown eval-trend action: %s\n\n", action)
		printEvalTrendUsage()
		os.Exit(1)
	}
}

func printEvalTrendUsage() {
	fmt.Println("Usage: ailang eval-trend <action> [options]")
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  candidates       Surface persistent-failure (benchmark, error_category) tuples")
	fmt.Println("                   from N-trial rotation summary.json data.")
	fmt.Println("  tier-saturation  Per-mode ELO saturation report (demotion candidates +")
	fmt.Println("                   recommendation) from the ratings DB.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang eval-trend candidates --rotation eval_results/rotation/2026-05-23")
	fmt.Println("  ailang eval-trend candidates --rotation eval_results/rotation/2026-05-23 --json")
	fmt.Println("  ailang eval-trend tier-saturation --db ~/.ailang/state/observatory.db")
}

// PersistentFailureCandidate is one row of the candidates output. The shape
// is stable (JSON-tagged) so external tools (the eval-analyzer skill, the
// docusaurus publication in Phase 5) can consume it.
type PersistentFailureCandidate struct {
	BenchmarkID   string  `json:"benchmark_id"`
	Model         string  `json:"model"`
	Lang          string  `json:"lang"`
	ErrorCategory string  `json:"error_category"`
	NFail         int     `json:"n_fail"`
	NTrials       int     `json:"n_trials"`
	PassRate      float64 `json:"pass_rate"`
	ExampleTokens int     `json:"example_tokens,omitempty"`
}

// runEvalTrendCandidates implements `ailang eval-trend candidates`.
func runEvalTrendCandidates() {
	fs := flag.NewFlagSet("eval-trend candidates", flag.ExitOnError)
	rotationDir := fs.String("rotation", "",
		"Path to a rotation directory containing summary.json files (e.g. eval_results/rotation/2026-05-23). Required.")
	minFail := fs.Int("min-fail", 2,
		"Minimum number of failures of the same error_category to surface as a candidate.")
	minTrials := fs.Int("min-trials", 3,
		"Skip (benchmark, model, lang) tuples with fewer than this many trials (signal too weak).")
	maxPassRate := fs.Float64("max-pass-rate", 0.5,
		"Skip tuples whose pass_rate is >= this value (already mostly-passing benchmarks aren't candidates).")
	jsonOut := fs.Bool("json", false, "Emit candidates as JSON instead of a text table.")

	if err := fs.Parse(flag.Args()[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if *rotationDir == "" {
		fmt.Fprintln(os.Stderr, "Error: --rotation <dir> is required.")
		fs.Usage()
		os.Exit(1)
	}

	candidates, err := findCandidates(*rotationDir, *minFail, *minTrials, *maxPassRate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(candidates)
		return
	}
	printCandidatesTable(*rotationDir, candidates, *minFail, *minTrials, *maxPassRate)
}

// findCandidates loads summary.json files under rotationDir and filters
// per-(benchmark, model, lang, error_category) tuples that meet the thresholds.
//
// rotationDir can be:
//   - A single rotation directory containing one summary.json
//   - A parent directory containing multiple per-time-slot rotation subdirs
//     (the canonical layout from `make eval-smoke ... -output
//     eval_results/rotation/<date>/<hhmm>_<model>_<tier>/`)
//
// We walk for summary.json files recursively and union their benchmark
// summaries before filtering.
func findCandidates(rotationDir string, minFail, minTrials int, maxPassRate float64) ([]PersistentFailureCandidate, error) {
	summaryFiles, err := findSummaryFiles(rotationDir)
	if err != nil {
		return nil, fmt.Errorf("find summary.json files: %w", err)
	}
	if len(summaryFiles) == 0 {
		return nil, fmt.Errorf("no summary.json files found under %s — was the rotation run with --trials N?", rotationDir)
	}

	// Group cross-summary by (benchmark, model, lang) so a benchmark that
	// appears in multiple rotation subdirs gets aggregated. Pick the
	// dominant error_category (most-frequent failing one) for the candidate.
	type groupKey struct{ Benchmark, Model, Lang string }
	type groupAgg struct {
		Trials       int
		Passed       int
		ErrorCats    map[string]int
		SampleTokens int // a representative token count from a failing run
	}
	groups := map[groupKey]*groupAgg{}

	for _, sf := range summaryFiles {
		data, err := os.ReadFile(sf)
		if err != nil {
			continue
		}
		var rs eval_harness.RotationSummary
		if err := json.Unmarshal(data, &rs); err != nil {
			continue
		}
		for _, bs := range rs.BenchmarkSummary {
			k := groupKey{Benchmark: bs.BenchmarkID, Model: bs.Model, Lang: bs.Lang}
			g := groups[k]
			if g == nil {
				g = &groupAgg{ErrorCats: map[string]int{}}
				groups[k] = g
			}
			g.Trials += bs.Trials
			g.Passed += bs.Passed
			for cat, n := range bs.ErrorCategories {
				g.ErrorCats[cat] += n
			}
			if int(bs.TokensFailMean) > 0 && g.SampleTokens == 0 {
				g.SampleTokens = int(bs.TokensFailMean)
			}
		}
	}

	// Filter into candidates.
	var out []PersistentFailureCandidate
	for k, g := range groups {
		if g.Trials < minTrials {
			continue
		}
		passRate := float64(g.Passed) / float64(g.Trials)
		if passRate >= maxPassRate {
			continue
		}
		// Pick the dominant failure category for this tuple.
		domCat := ""
		domCount := 0
		for cat, n := range g.ErrorCats {
			if n > domCount {
				domCat = cat
				domCount = n
			}
		}
		if domCount < minFail {
			continue
		}
		out = append(out, PersistentFailureCandidate{
			BenchmarkID:   k.Benchmark,
			Model:         k.Model,
			Lang:          k.Lang,
			ErrorCategory: domCat,
			NFail:         domCount,
			NTrials:       g.Trials,
			PassRate:      passRate,
			ExampleTokens: g.SampleTokens,
		})
	}

	// Stable sort: pass_rate ASC (worst first), then benchmark, then model.
	sort.Slice(out, func(i, j int) bool {
		if out[i].PassRate != out[j].PassRate {
			return out[i].PassRate < out[j].PassRate
		}
		if out[i].BenchmarkID != out[j].BenchmarkID {
			return out[i].BenchmarkID < out[j].BenchmarkID
		}
		return out[i].Model < out[j].Model
	})

	return out, nil
}

// findSummaryFiles walks rotationDir looking for files literally named
// "summary.json", at any depth. Returns the list of absolute paths.
func findSummaryFiles(rotationDir string) ([]string, error) {
	var found []string
	err := filepath.Walk(rotationDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable paths silently
		}
		if !info.IsDir() && info.Name() == "summary.json" {
			found = append(found, path)
		}
		return nil
	})
	return found, err
}

// printCandidatesTable writes a human-readable table to stdout. Designed for
// quick triage from a terminal; the JSON output is the machine surface.
func printCandidatesTable(rotationDir string, candidates []PersistentFailureCandidate, minFail, minTrials int, maxPassRate float64) {
	fmt.Printf("Persistent failure candidates from %s\n", rotationDir)
	fmt.Printf("  Filters: pass_rate < %.2f, error_category count >= %d, total trials >= %d\n",
		maxPassRate, minFail, minTrials)
	fmt.Println()
	if len(candidates) == 0 {
		fmt.Println("  (no candidates — rotation looks healthy)")
		return
	}
	fmt.Printf("  %-30s %-25s %-9s %-18s %-7s %-10s\n",
		"Benchmark", "Model", "Lang", "Error category", "Fails", "Pass rate")
	fmt.Println("  " + repeat("-", 100))
	for _, c := range candidates {
		failCol := fmt.Sprintf("%d/%d", c.NFail, c.NTrials)
		passCol := fmt.Sprintf("%.1f%%", c.PassRate*100)
		fmt.Printf("  %-30s %-25s %-9s %-18s %-7s %-10s\n",
			truncStr(c.BenchmarkID, 30),
			truncStr(c.Model, 25),
			truncStr(c.Lang, 9),
			truncStr(c.ErrorCategory, 18),
			failCol,
			passCol,
		)
	}
	fmt.Println()
	fmt.Println("For per-candidate deep-dive (which AILANG construct is the model tripping on?):")
	fmt.Println("  Use the .claude/skills/eval-analyzer/ skill.")
	fmt.Println()
	fmt.Println("Pipe to --json for tool consumption (e.g. design-doc-creator integration):")
	fmt.Printf("  ailang eval-trend candidates --rotation %s --json\n", rotationDir)
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
