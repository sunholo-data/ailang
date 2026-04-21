package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/eval_analysis"
	"github.com/sunholo/ailang/internal/eval_harness"
)

// loadBenchmarkTags reads every YAML in dir and returns benchmark ID -> tags.
// Benchmarks that fail to load are skipped (silently here; LoadSpec already
// warns on unknown tags via spec.go). The full analysis-grade version lives
// in internal/eval_analysis/tags.go (arrives in M4).
func loadBenchmarkTags(dir string) map[string][]string {
	out := map[string][]string{}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.yml"))
	for _, path := range matches {
		spec, err := eval_harness.LoadSpec(path)
		if err != nil {
			continue
		}
		out[spec.ID] = spec.Tags
	}
	return out
}

// passCell counts pass/total events for a (tag, lang) cell.
type passCell struct{ pass, total int }

// printTagsSection emits a per-tag AILANG vs Python delta table.
// A benchmark result contributes to every tag it carries. Pass rate is
// computed on StdoutOk. Refusal filtering arrives in M4.
func printTagsSection(results []*eval_analysis.BenchmarkResult, benchmarkDir string) {
	tags := loadBenchmarkTags(benchmarkDir)
	if len(tags) == 0 {
		fmt.Println("\n## By Tags\n\n(no benchmark tags loaded)")
		return
	}

	byTag := map[string]map[string]*passCell{} // tag -> lang -> cell

	for _, r := range results {
		for _, tag := range tags[r.ID] {
			if byTag[tag] == nil {
				byTag[tag] = map[string]*passCell{}
			}
			if byTag[tag][r.Lang] == nil {
				byTag[tag][r.Lang] = &passCell{}
			}
			c := byTag[tag][r.Lang]
			c.total++
			if r.StdoutOk {
				c.pass++
			}
		}
	}

	tagNames := make([]string, 0, len(byTag))
	for t := range byTag {
		tagNames = append(tagNames, t)
	}
	sort.Strings(tagNames)

	fmt.Println("\n## By Tags (AILANG vs Python)")
	fmt.Println()
	fmt.Println("| Tag | AILANG | Python | Δ (AILANG - Python) |")
	fmt.Println("|-----|-------:|-------:|--------------------:|")
	for _, tag := range tagNames {
		ailCell := byTag[tag]["ailang"]
		pyCell := byTag[tag]["python"]
		delta := rate(ailCell) - rate(pyCell)
		fmt.Printf("| %s | %s | %s | %+.1fpp |\n", tag, fmtRate(ailCell), fmtRate(pyCell), delta*100)
	}
}

// printSaturatedSection lists benchmarks passing 100% across all models for
// every language in the given results. Multi-baseline saturation (requires
// the latest 2 baselines per the design) arrives in M4.
func printSaturatedSection(results []*eval_analysis.BenchmarkResult) {
	type key struct{ id, lang string }
	type agg struct{ pass, total int }
	byKey := map[key]*agg{}
	for _, r := range results {
		k := key{r.ID, r.Lang}
		if byKey[k] == nil {
			byKey[k] = &agg{}
		}
		byKey[k].total++
		if r.StdoutOk {
			byKey[k].pass++
		}
	}

	// Saturated = 100% pass in every language seen for that benchmark.
	langsByID := map[string]map[string]bool{}
	for k := range byKey {
		if langsByID[k.id] == nil {
			langsByID[k.id] = map[string]bool{}
		}
		langsByID[k.id][k.lang] = true
	}

	var saturated []string
	for id, langs := range langsByID {
		all := true
		for lang := range langs {
			a := byKey[key{id, lang}]
			if a.total == 0 || a.pass < a.total {
				all = false
				break
			}
		}
		if all {
			saturated = append(saturated, id)
		}
	}
	sort.Strings(saturated)

	fmt.Println("\n## Saturated Benchmarks (100% pass, all models + languages)")
	fmt.Println()
	if len(saturated) == 0 {
		fmt.Println("(none)")
		return
	}
	for _, id := range saturated {
		fmt.Printf("- `%s`\n", id)
	}
	fmt.Printf("\n**Count:** %d of %d benchmarks\n", len(saturated), len(langsByID))
}

// printAILANGWinsSection lists benchmarks where AILANG passes and Python fails
// for the same model. M4 will add cross-model pattern detection (3+ models
// agreeing) — this is the single-result view.
func printAILANGWinsSection(results []*eval_analysis.BenchmarkResult) {
	type key struct{ id, model string }
	byKey := map[key]map[string]bool{} // lang -> pass
	for _, r := range results {
		k := key{r.ID, r.Model}
		if byKey[k] == nil {
			byKey[k] = map[string]bool{}
		}
		byKey[k][r.Lang] = r.StdoutOk
	}

	var wins []key
	for k, langs := range byKey {
		if langs["ailang"] && !langs["python"] {
			wins = append(wins, k)
		}
	}
	sort.Slice(wins, func(i, j int) bool {
		if wins[i].id != wins[j].id {
			return wins[i].id < wins[j].id
		}
		return wins[i].model < wins[j].model
	})

	fmt.Println("\n## AILANG-Only Wins (AILANG passes, Python fails, same model)")
	fmt.Println()
	if len(wins) == 0 {
		fmt.Println("(none)")
		return
	}
	fmt.Println("| Benchmark | Model |")
	fmt.Println("|-----------|-------|")
	for _, w := range wins {
		fmt.Printf("| `%s` | %s |\n", w.id, w.model)
	}
	fmt.Printf("\n**Total:** %d (benchmark × model) wins\n", len(wins))
}

// rate returns pass/total as a [0,1] fraction, or 0 if total is 0.
func rate(c *passCell) float64 {
	if c == nil || c.total == 0 {
		return 0
	}
	return float64(c.pass) / float64(c.total)
}

// fmtRate formats a cell as "pp% (p/t)" or "n/a" when total is 0.
func fmtRate(c *passCell) string {
	if c == nil || c.total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%% (%d/%d)", rate(c)*100, c.pass, c.total)
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
