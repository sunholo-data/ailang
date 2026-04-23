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

// parseMatrixFlags scans flag.Arg() from `start` onward for the four
// boolean flags. Unknown --flag=... / --flag args are left alone (they may
// be parsed elsewhere, e.g. --format= in eval-report shares this scan path).
func parseMatrixFlags(getArg func(int) string, n int) (byTags, showSaturated, ailangWins, byHarness bool) {
	for i := 0; i < n; i++ {
		switch strings.TrimSpace(getArg(i)) {
		case "--by-tags":
			byTags = true
		case "--show-saturated":
			showSaturated = true
		case "--ailang-wins":
			ailangWins = true
		case "--by-harness":
			byHarness = true
		}
	}
	return
}

// parseGroupByFlag returns the value of --group-by=<value> or --group-by <value>.
func parseGroupByFlag(getArg func(int) string, n int) string {
	for i := 0; i < n; i++ {
		arg := strings.TrimSpace(getArg(i))
		if strings.HasPrefix(arg, "--group-by=") {
			return strings.TrimPrefix(arg, "--group-by=")
		}
		if arg == "--group-by" && i+1 < n {
			return strings.TrimSpace(getArg(i + 1))
		}
	}
	return ""
}

// printByHarnessSection renders a language × model × harness breakdown table.
// Each row is one (language, model) pair; columns are harnesses (executors).
// This gives the full picture: which models, on which harnesses, pass in which languages.
func printByHarnessSection(results []*eval_analysis.BenchmarkResult) {
	type cell struct{ pass, total int }

	// Collect unique harnesses, languages, models in stable order
	harnessSet := map[string]bool{}
	langSet := map[string]bool{}
	modelSet := map[string]bool{}
	// data: lang+model → harness → cell
	data := map[string]map[string]*cell{}

	for _, r := range results {
		exec := r.Executor
		if exec == "" {
			exec = "standard"
		}
		harnessSet[exec] = true
		langSet[r.Lang] = true
		modelSet[r.Model] = true
		key := r.Lang + "\x00" + r.Model
		if data[key] == nil {
			data[key] = map[string]*cell{}
		}
		if data[key][exec] == nil {
			data[key][exec] = &cell{}
		}
		data[key][exec].total++
		if r.StdoutOk && r.CompileOk && r.RuntimeOk {
			data[key][exec].pass++
		}
	}

	var harnesses, langs, models []string
	for h := range harnessSet {
		harnesses = append(harnesses, h)
	}
	for l := range langSet {
		langs = append(langs, l)
	}
	for m := range modelSet {
		models = append(models, m)
	}
	sort.Strings(harnesses)
	sort.Strings(langs)
	sort.Strings(models)

	if len(harnesses) == 0 {
		fmt.Println("\n## By Language × Model × Harness\n\n(no results)")
		return
	}

	fmt.Print("\n## By Language × Model × Harness\n\n")

	// Header
	header := "| Language | Model |"
	sep := "|---|---|"
	for _, h := range harnesses {
		header += fmt.Sprintf(" %s |", h)
		sep += "---|"
	}
	fmt.Println(header)
	fmt.Println(sep)

	for _, lang := range langs {
		for _, model := range models {
			key := lang + "\x00" + model
			row := data[key]
			if row == nil {
				continue
			}
			line := fmt.Sprintf("| %s | %s |", lang, model)
			hasAny := false
			for _, h := range harnesses {
				c := row[h]
				if c == nil || c.total == 0 {
					line += " — |"
				} else {
					pct := float64(c.pass) / float64(c.total) * 100
					mark := "✓"
					if c.pass < c.total {
						mark = "⚠"
					}
					if c.pass == 0 {
						mark = "✗"
					}
					line += fmt.Sprintf(" %s %.0f%% (%d/%d) |", mark, pct, c.pass, c.total)
					hasAny = true
				}
			}
			if hasAny {
				fmt.Println(line)
			}
		}
	}
}

// printGroupedByFamilySection renders a cross-harness comparison table.
// Results are clustered by ModelFamily; within each family, each unique
// Executor is a column. Delta rows show pass-rate difference between
// the first and second harness in order of appearance.
func printGroupedByFamilySection(results []*eval_analysis.BenchmarkResult) {
	// Index: family → executor → benchmark → results
	type cell struct {
		pass  int
		total int
		cost  float64
		durMs int64
	}

	// Collect families and executors in stable order.
	var families []string
	familySet := map[string]bool{}
	// executor order per family
	familyExecutors := map[string][]string{}
	familyExecutorSet := map[string]map[string]bool{}
	// data: family → executor → benchID → []pass
	data := map[string]map[string]map[string]*cell{}

	for _, r := range results {
		if r.ModelFamily == "" {
			continue
		}
		fam := r.ModelFamily
		exec := r.Executor
		if exec == "" {
			exec = r.Model
		}

		if !familySet[fam] {
			familySet[fam] = true
			families = append(families, fam)
		}
		if familyExecutorSet[fam] == nil {
			familyExecutorSet[fam] = map[string]bool{}
		}
		if !familyExecutorSet[fam][exec] {
			familyExecutorSet[fam][exec] = true
			familyExecutors[fam] = append(familyExecutors[fam], exec)
		}
		if data[fam] == nil {
			data[fam] = map[string]map[string]*cell{}
		}
		if data[fam][exec] == nil {
			data[fam][exec] = map[string]*cell{}
		}
		c := data[fam][exec][r.ID]
		if c == nil {
			c = &cell{}
			data[fam][exec][r.ID] = c
		}
		c.total++
		if r.StdoutOk {
			c.pass++
		}
		c.cost += r.CostUSD
		c.durMs += r.DurationMs
	}

	if len(families) == 0 {
		fmt.Println("\n## Cross-Harness Comparison\n\n(no results with model_family set)")
		return
	}

	sort.Strings(families)

	fmt.Println("\n## Cross-Harness Comparison (by Model Family)")

	for _, fam := range families {
		execs := familyExecutors[fam]
		fmt.Printf("\n### %s (%d harness(es): %s)\n\n", fam, len(execs), strings.Join(execs, ", "))

		if len(execs) == 0 {
			continue
		}

		// Collect all benchmark IDs for this family.
		benchSet := map[string]bool{}
		for _, exec := range execs {
			for benchID := range data[fam][exec] {
				benchSet[benchID] = true
			}
		}
		var benchIDs []string
		for b := range benchSet {
			benchIDs = append(benchIDs, b)
		}
		sort.Strings(benchIDs)

		// Header
		header := "| Benchmark |"
		sep := "|-----------|"
		for _, exec := range execs {
			header += fmt.Sprintf(" %s |", exec)
			sep += "--------:|"
		}
		if len(execs) >= 2 {
			header += fmt.Sprintf(" Δ (%s→%s) |", execs[0], execs[1])
			sep += "-----------:|"
		}
		fmt.Println(header)
		fmt.Println(sep)

		// Per-benchmark rows
		execPass := make(map[string]int)
		execTotal := make(map[string]int)
		for _, benchID := range benchIDs {
			row := fmt.Sprintf("| `%s` |", benchID)
			var rates []float64
			for _, exec := range execs {
				c := data[fam][exec][benchID]
				if c == nil || c.total == 0 {
					row += " n/a |"
					rates = append(rates, -1)
				} else {
					rate := float64(c.pass) / float64(c.total)
					row += fmt.Sprintf(" %.0f%% (%d/%d) |", rate*100, c.pass, c.total)
					rates = append(rates, rate)
					execPass[exec] += c.pass
					execTotal[exec] += c.total
				}
			}
			if len(execs) >= 2 && len(rates) >= 2 && rates[0] >= 0 && rates[1] >= 0 {
				delta := rates[1] - rates[0]
				row += fmt.Sprintf(" %+.1fpp |", delta*100)
			} else if len(execs) >= 2 {
				row += " n/a |"
			}
			fmt.Println(row)
		}

		// Summary row
		fmt.Println()
		fmt.Print("**Summary:** ")
		parts := make([]string, 0, len(execs))
		var summaryRates []float64
		for _, exec := range execs {
			t := execTotal[exec]
			p := execPass[exec]
			if t == 0 {
				parts = append(parts, fmt.Sprintf("%s: n/a", exec))
				summaryRates = append(summaryRates, -1)
			} else {
				rate := float64(p) / float64(t)
				parts = append(parts, fmt.Sprintf("%s: %d/%d (%.1f%%)", exec, p, t, rate*100))
				summaryRates = append(summaryRates, rate)
			}
		}
		fmt.Print(strings.Join(parts, ", "))
		if len(execs) >= 2 && len(summaryRates) >= 2 && summaryRates[0] >= 0 && summaryRates[1] >= 0 {
			fmt.Printf(", Δ = %+.1fpp\n", (summaryRates[1]-summaryRates[0])*100)
		} else {
			fmt.Println()
		}
	}
}
