package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
)

// printTagsSection emits a per-tag breakdown table for all eval languages
// present in the results. The benchmark dir is passed so that the CLI can
// point at a non-standard location during tests.
func printTagsSection(results []*eval_analysis.BenchmarkResult, benchmarkDir string) {
	tags := eval_analysis.LoadBenchmarkTags(benchmarkDir)
	if len(tags) == 0 {
		fmt.Println("\n## By Tags\n\n(no benchmark tags loaded)")
		return
	}

	report := eval_analysis.GroupByTags(results, tags)

	// Collect all languages present across all tags, sorted.
	langSet := map[string]bool{}
	for _, agg := range report.Aggregates {
		for lang := range agg.LanguageBreakdown {
			langSet[lang] = true
		}
	}
	langs := sortedKeys(langSet)

	// Preferred display order: ailang first, then python, then alphabetical.
	preferred := []string{"ailang", "python"}
	langs = reorderLangs(langs, preferred)

	title := "By Tags (" + strings.Join(langs, " vs ") + ")"
	fmt.Println("\n## " + title)
	fmt.Println()

	// Build header dynamically.
	header := "| Tag |"
	sep := "|-----|"
	for _, lang := range langs {
		header += " " + lang + " |"
		sep += "-------:|"
	}
	// Add delta column only when ailang and python are both present.
	withDelta := langSet["ailang"] && langSet["python"]
	if withDelta {
		header += " Δ (ailang-python) |"
		sep += "------------------:|"
	}
	fmt.Println(header)
	fmt.Println(sep)

	for _, tag := range report.Tags {
		agg := report.Aggregates[tag]
		row := fmt.Sprintf("| %s |", tag)
		for _, lang := range langs {
			if ls := agg.LanguageBreakdown[lang]; ls != nil {
				row += " " + fmtTagRate(ls.Pass, ls.Total) + " |"
			} else {
				row += " n/a |"
			}
		}
		if withDelta {
			row += fmt.Sprintf(" %+.1fpp |", agg.Delta*100)
		}
		fmt.Println(row)
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

// sortedKeys returns the sorted keys of a string bool map.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// reorderLangs moves preferred languages to the front (in order), with
// remaining languages sorted alphabetically after them.
func reorderLangs(langs []string, preferred []string) []string {
	inLangs := make(map[string]bool, len(langs))
	for _, l := range langs {
		inLangs[l] = true
	}
	out := make([]string, 0, len(langs))
	for _, p := range preferred {
		if inLangs[p] {
			out = append(out, p)
			delete(inLangs, p)
		}
	}
	for _, l := range langs {
		if inLangs[l] {
			out = append(out, l)
		}
	}
	return out
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
