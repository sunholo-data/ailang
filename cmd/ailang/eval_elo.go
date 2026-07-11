// Package main: `ailang eval-elo` subcommand.
//
// Fits per-language (ailang vs python) ELO ratings over an eval results dir and
// prints a model leaderboard + a benchmark difficulty table side-by-side, so the
// AILANG-vs-Python capability story is visible and benchmark tier promotion /
// demotion candidates surface. This is the CLI surface over the same
// deterministic chess-style fit used by the dashboard ratings block
// (eval_harness.FitFromTrials); here we group trials by .Lang instead of blending
// both languages into one leaderboard.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// eloModelRow is one row of the model leaderboard (per mode).
type eloModelRow struct {
	Model      string   `json:"model"`
	AilangELO  *float64 `json:"ailang_elo"`
	PythonELO  *float64 `json:"python_elo"`
	Delta      *float64 `json:"delta_ailang_minus_python"` // set only when both langs present
	AilangBand string   `json:"ailang_band,omitempty"`
	PythonBand string   `json:"python_band,omitempty"`
}

// eloBenchRow is one row of the benchmark difficulty table (per mode).
type eloBenchRow struct {
	Benchmark  string   `json:"benchmark"`
	Tier       string   `json:"tier"`
	AilangELO  *float64 `json:"ailang_elo"`
	PythonELO  *float64 `json:"python_elo"`
	AilangPass *float64 `json:"ailang_pass_pct"`
	PythonPass *float64 `json:"python_pass_pct"`
	AilangBand string   `json:"ailang_band,omitempty"`
	PythonBand string   `json:"python_band,omitempty"`
	Flag       string   `json:"flag,omitempty"`
}

// eloModeReport is the per-mode payload emitted by --json.
type eloModeReport struct {
	Mode       string        `json:"mode"`
	Models     []eloModelRow `json:"models"`
	Benchmarks []eloBenchRow `json:"benchmarks"`
}

// langFit holds the per-language fit outputs.
type langFit struct {
	modelELO map[string]float64
	benchELO map[string]float64
	pass     map[string][2]int // benchmark -> [passed, total]
}

// runEvalELO implements `ailang eval-elo <results_dir> [--json]`.
func runEvalELO() {
	args := flag.Args() // flag.Arg(0) == "eval-elo"
	jsonOut := false
	resultsDir := ""
	for _, a := range args[1:] {
		switch a {
		case "--json":
			jsonOut = true
		default:
			if resultsDir == "" {
				resultsDir = a
			}
		}
	}
	if resultsDir == "" {
		fmt.Fprintf(os.Stderr, "%s: missing <results_dir>\n", red("Error"))
		fmt.Println("Usage: ailang eval-elo <results_dir> [--json]")
		fmt.Println()
		fmt.Println("Fit per-language (AILANG vs Python) ELO over an eval results directory.")
		fmt.Println("Prints a model leaderboard and a benchmark difficulty table per mode.")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-elo eval_results/baselines/v0.29.2")
		fmt.Println("  ailang eval-elo eval_results/baselines/v0.29.2 --json")
		os.Exit(1)
	}

	results, err := eval_analysis.LoadResults(resultsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load results: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Split standard vs agent. Matches eval-report's convention: EvalMode=="agent"
	// is agent, everything else (including empty) is standard. Standard baseline
	// results are all standard mode.
	var standard, agent []*eval_analysis.BenchmarkResult
	for _, r := range results {
		if r.EvalMode == "agent" {
			agent = append(agent, r)
		} else {
			standard = append(standard, r)
		}
	}

	var reports []eloModeReport
	if r := buildELOModeReport("standard", standard); r != nil {
		reports = append(reports, *r)
	}
	if r := buildELOModeReport("agent", agent); r != nil {
		reports = append(reports, *r)
	}
	if len(reports) == 0 {
		fmt.Fprintf(os.Stderr, "%s: no results with a language found in %s\n", red("Error"), resultsDir)
		os.Exit(1)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(reports); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		return
	}
	printELOReports(resultsDir, reports)
}

// buildELOModeReport fits per-language ELO for one mode's results and assembles
// the leaderboard + benchmark rows. Returns nil when there are no results.
func buildELOModeReport(mode string, results []*eval_analysis.BenchmarkResult) *eloModeReport {
	if len(results) == 0 {
		return nil
	}
	ailang := fitLang(results, "ailang")
	python := fitLang(results, "python")

	// --- Model leaderboard: union of models across both languages ---
	modelSet := map[string]bool{}
	for m := range ailang.modelELO {
		modelSet[m] = true
	}
	for m := range python.modelELO {
		modelSet[m] = true
	}
	var models []eloModelRow
	for m := range modelSet {
		row := eloModelRow{Model: m}
		if v, ok := ailang.modelELO[m]; ok {
			row.AilangELO = ptr(round1(v))
			row.AilangBand = eval_harness.Band(v)
		}
		if v, ok := python.modelELO[m]; ok {
			row.PythonELO = ptr(round1(v))
			row.PythonBand = eval_harness.Band(v)
		}
		if row.AilangELO != nil && row.PythonELO != nil {
			row.Delta = ptr(round1(*row.AilangELO - *row.PythonELO))
		}
		models = append(models, row)
	}
	sortByELODesc(models, func(r eloModelRow) *float64 { return r.AilangELO }, func(r eloModelRow) string { return r.Model })

	// --- Benchmark table: union of benchmarks across both languages ---
	benchSet := map[string]bool{}
	for b := range ailang.benchELO {
		benchSet[b] = true
	}
	for b := range python.benchELO {
		benchSet[b] = true
	}
	var benches []eloBenchRow
	for b := range benchSet {
		tier := benchTier(b)
		row := eloBenchRow{Benchmark: b, Tier: tier}
		if v, ok := ailang.benchELO[b]; ok {
			row.AilangELO = ptr(round1(v))
			row.AilangBand = eval_harness.Band(v)
		}
		if v, ok := python.benchELO[b]; ok {
			row.PythonELO = ptr(round1(v))
			row.PythonBand = eval_harness.Band(v)
		}
		if pv, ok := ailang.pass[b]; ok {
			row.AilangPass = ptr(passPct(pv))
		}
		if pv, ok := python.pass[b]; ok {
			row.PythonPass = ptr(passPct(pv))
		}
		row.Flag = eloFlag(tier, row.AilangBand, row.PythonBand, row.AilangELO)
		benches = append(benches, row)
	}
	sortByELODesc(benches, func(r eloBenchRow) *float64 { return r.AilangELO }, func(r eloBenchRow) string { return r.Benchmark })

	return &eloModeReport{Mode: mode, Models: models, Benchmarks: benches}
}

// fitLang runs the deterministic ELO fit over only the results for one language.
func fitLang(results []*eval_analysis.BenchmarkResult, lang string) langFit {
	trials := make([]eval_harness.Trial, 0, len(results))
	pass := map[string][2]int{}
	for _, r := range results {
		if r.Lang != lang {
			continue
		}
		ok := r.CompileOk && r.RuntimeOk && r.StdoutOk
		trials = append(trials, eval_harness.Trial{Model: r.Model, Bench: r.ID, Pass: ok})
		v := pass[r.ID]
		v[1]++
		if ok {
			v[0]++
		}
		pass[r.ID] = v
	}
	modelELO, benchELO := eval_harness.FitFromTrials(trials)
	return langFit{modelELO: modelELO, benchELO: benchELO, pass: pass}
}

// eloFlag computes a promote/demote heuristic for a benchmark.
//
// Heuristic (an ELO-based APPROXIMATION of the formal tier rules):
//   - DEMOTE                 if the benchmark is Trivial-band on BOTH languages and
//     its current tier is stretch/frontier (nobody is challenged by it there).
//   - PROMOTE→stretch        if AILANG-ELO > 1900 and current tier == core.
//   - PROMOTE→frontier       if AILANG-ELO > 2100 and current tier == stretch.
//   - frontier-easy-on-AILANG if tier == frontier but AILANG-ELO < 1600 (a
//     frontier benchmark that AILANG models find easy — candidate for review).
//
// NOTE: the FORMAL frontier-demotion rule is "demote if all *frontier models*
// pass". This ELO heuristic approximates that with band/threshold checks;
// refining it with an explicit frontier-model-set pass check is a follow-up.
func eloFlag(tier, ailangBand, pythonBand string, ailangELO *float64) string {
	if ailangBand == "Trivial" && pythonBand == "Trivial" && (tier == "stretch" || tier == "frontier") {
		return "DEMOTE"
	}
	if ailangELO != nil {
		switch {
		case tier == "core" && *ailangELO > 1900:
			return "PROMOTE->stretch"
		case tier == "stretch" && *ailangELO > 2100:
			return "PROMOTE->frontier"
		case tier == "frontier" && *ailangELO < 1600:
			return "frontier-easy-on-AILANG"
		}
	}
	return ""
}

// benchTier reads a benchmark's current tier from benchmarks/<id>.yml. Defaults
// to "core" (LoadSpec's default) when the spec is missing/unreadable.
func benchTier(id string) string {
	spec, err := eval_harness.LoadSpec(filepath.Join("benchmarks", id+".yml"))
	if err != nil || spec.Tier == "" {
		return "core"
	}
	return spec.Tier
}

// --- printing ---

func printELOReports(resultsDir string, reports []eloModeReport) {
	fmt.Printf("Per-language ELO — %s\n", resultsDir)
	for _, rep := range reports {
		fmt.Printf("\n=== %s mode ===\n", rep.Mode)

		fmt.Println("\nModel leaderboard (sorted by AILANG-ELO):")
		fmt.Printf("  %-28s %11s %11s %9s\n", "Model", "AILANG-ELO", "Python-ELO", "Δ(A−P)")
		fmt.Println("  " + repeat("-", 62))
		for _, m := range rep.Models {
			fmt.Printf("  %-28s %11s %11s %9s\n",
				truncStr(m.Model, 28), eloStr(m.AilangELO), eloStr(m.PythonELO), deltaStr(m.Delta))
		}

		fmt.Println("\nBenchmark difficulty (sorted by AILANG-ELO):")
		fmt.Printf("  %-26s %-9s %10s %10s %8s %8s  %s\n",
			"Benchmark", "Tier", "AILANG", "Python", "A-pass%", "P-pass%", "Flag")
		fmt.Println("  " + repeat("-", 96))
		for _, b := range rep.Benchmarks {
			fmt.Printf("  %-26s %-9s %10s %10s %8s %8s  %s\n",
				truncStr(b.Benchmark, 26), truncStr(b.Tier, 9),
				eloStr(b.AilangELO), eloStr(b.PythonELO),
				pctStr(b.AilangPass), pctStr(b.PythonPass), b.Flag)
		}
	}
	fmt.Println()
	fmt.Println("Flags: DEMOTE (Trivial on both langs at stretch/frontier), PROMOTE->stretch (AILANG-ELO>1900 @ core),")
	fmt.Println("       PROMOTE->frontier (AILANG-ELO>2100 @ stretch), frontier-easy-on-AILANG (AILANG-ELO<1600 @ frontier).")
}

// --- small helpers ---

func ptr(f float64) *float64 { return &f }

func passPct(v [2]int) float64 {
	if v[1] == 0 {
		return 0
	}
	return round1(float64(v[0]) / float64(v[1]) * 100)
}

func eloStr(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f", *p)
}

func deltaStr(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%+.1f", *p)
}

func pctStr(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", *p)
}

// sortByELODesc sorts by the given ELO accessor descending (nil ELO sinks to the
// bottom), tie-broken by the name accessor ascending for determinism.
func sortByELODesc[T any](rows []T, elo func(T) *float64, name func(T) string) {
	sort.Slice(rows, func(i, j int) bool {
		ei, ej := elo(rows[i]), elo(rows[j])
		switch {
		case ei == nil && ej == nil:
			return name(rows[i]) < name(rows[j])
		case ei == nil:
			return false
		case ej == nil:
			return true
		case *ei != *ej:
			return *ei > *ej
		default:
			return name(rows[i]) < name(rows[j])
		}
	})
}

func round1(x float64) float64 {
	return float64(int(x*10+0.5)) / 10
}
