package main

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/eval_analysis"
)

// printTagsSection emits a per-tag AILANG vs Python delta table using
// the eval_analysis primitives. The benchmark dir is passed so that
// the CLI can point at a non-standard location during tests.
func printTagsSection(results []*eval_analysis.BenchmarkResult, benchmarkDir string) {
	tags := eval_analysis.LoadBenchmarkTags(benchmarkDir)
	if len(tags) == 0 {
		fmt.Println("\n## By Tags\n\n(no benchmark tags loaded)")
		return
	}

	report := eval_analysis.GroupByTags(results, tags)

	fmt.Println("\n## By Tags (AILANG vs Python)")
	fmt.Println()
	fmt.Println("| Tag | AILANG | Python | Δ (AILANG - Python) |")
	fmt.Println("|-----|-------:|-------:|--------------------:|")
	for _, tag := range report.Tags {
		agg := report.Aggregates[tag]
		fmt.Printf("| %s | %s | %s | %+.1fpp |\n",
			tag,
			fmtTagRate(agg.AILANGPass, agg.AILANGTotal),
			fmtTagRate(agg.PythonPass, agg.PythonTotal),
			agg.Delta*100,
		)
	}
}

// printSaturatedSection wraps the current results as a single synthetic
// baseline and runs DetectSaturation. This keeps the CLI single-batch
// behaviour while using the same primitive the multi-baseline analysis
// does — M5+ can call DetectSaturation directly with the latest 2
// baselines when the pipeline loads them.
func printSaturatedSection(results []*eval_analysis.BenchmarkResult) {
	baseline := &eval_analysis.Baseline{Version: "current", Results: results}
	saturated := eval_analysis.DetectSaturation([]*eval_analysis.Baseline{baseline}, 1)

	// Count benchmarks that participated in the analysis (for the
	// "X of Y" footer). Mirror DetectSaturation's refusal filter.
	benchSet := map[string]bool{}
	for _, r := range results {
		if r.RefusalDetected {
			continue
		}
		benchSet[r.ID] = true
	}

	fmt.Println("\n## Saturated Benchmarks (100% pass, all models + languages)")
	fmt.Println()
	if len(saturated) == 0 {
		fmt.Println("(none)")
		return
	}
	for _, s := range saturated {
		fmt.Printf("- `%s`\n", s.ID)
	}
	fmt.Printf("\n**Count:** %d of %d benchmarks\n", len(saturated), len(benchSet))
}

// printAILANGWinsSection renders the per-cell win list plus the 3+ model
// pattern list from DetectAILANGOnlyWins.
func printAILANGWinsSection(results []*eval_analysis.BenchmarkResult) {
	report := eval_analysis.DetectAILANGOnlyWins(results)

	fmt.Println("\n## AILANG-Only Wins (AILANG passes, Python fails, same model)")
	fmt.Println()
	if len(report.Wins) == 0 {
		fmt.Println("(none)")
		return
	}
	fmt.Println("| Benchmark | Model |")
	fmt.Println("|-----------|-------|")
	for _, w := range report.Wins {
		fmt.Printf("| `%s` | %s |\n", w.ID, w.Model)
	}
	fmt.Printf("\n**Total:** %d (benchmark × model) wins\n", len(report.Wins))

	if len(report.Patterns) > 0 {
		fmt.Println("\n### Cross-Model Patterns (≥3 models agreeing)")
		fmt.Println()
		for _, id := range report.Patterns {
			fmt.Printf("- `%s` (%d models)\n", id, report.PerBenchmark[id])
		}
	}
}

// fmtTagRate formats a (pass, total) pair as "pp% (p/t)" or "n/a".
// Kept CLI-local because it is purely a table-rendering helper.
func fmtTagRate(pass, total int) string {
	if total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%% (%d/%d)", 100*float64(pass)/float64(total), pass, total)
}

// parseMatrixFlags scans flag.Arg() from `start` onward for the three M3
// boolean flags. Unknown --flag=... / --flag args are left alone (they may
// be parsed elsewhere, e.g. --format= in eval-report shares this scan path).
func parseMatrixFlags(getArg func(int) string, n int) (byTags, showSaturated, ailangWins bool) {
	for i := 0; i < n; i++ {
		switch strings.TrimSpace(getArg(i)) {
		case "--by-tags":
			byTags = true
		case "--show-saturated":
			showSaturated = true
		case "--ailang-wins":
			ailangWins = true
		}
	}
	return
}
