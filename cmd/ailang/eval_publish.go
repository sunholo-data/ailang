// Package main: `ailang eval-publish <release-tag>` (M-EVAL-OS-LONGITUDINAL Phase 5).
//
// Generates a per-release Docusaurus page comparing pass rates across models
// and against the previous release. The publishable longitudinal artifact
// that differentiates local-Ollama eval from cloud-cost-limited competitors —
// the rotation gives us N>=3 trial data per release, which the leaderboard
// can publish at granularity that pay-per-token cloud eval can't match.
//
// Inputs: one or more rotation directories (Phase-3 summary.json files).
// Output: docs/docs/reference/os-model-leaderboard/<release>.md
//
// Trend deltas vs. a prior release are included when --prev <release> is
// passed; otherwise only the per-release table is rendered.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// runEvalPublish implements `ailang eval-publish <release-tag> [flags]`.
func runEvalPublish() {
	fs := flag.NewFlagSet("eval-publish", flag.ExitOnError)
	rotationDir := fs.String("rotation", "",
		"Path to the rotation directory for THIS release (e.g. eval_results/rotation/2026-05-23). Required.")
	prevRelease := fs.String("prev", "",
		"Path to the previous release's rotation directory for trend-delta computation. Optional.")
	prevTag := fs.String("prev-tag", "",
		"Previous release tag (e.g. v0.22.0). Used in trend-delta column header. Required when --prev is set.")
	outputDir := fs.String("output-dir", "docs/docs/reference/os-model-leaderboard",
		"Output directory for the (legacy) Docusaurus markdown page. Only used with --markdown.")
	osJSON := fs.String("os-json", "docs/static/benchmarks/os/latest.json",
		"Path to write the OS/Local leaderboard JSON the dashboard reads (model x harness x language). Empty to skip.")
	emitMarkdown := fs.Bool("markdown", false,
		"Also emit the legacy per-release markdown snapshot (retired from the site; off by default).")
	minPpDelta := fs.Float64("min-delta-pp", 10.0,
		"Minimum percentage-point pass-rate delta required for a benchmark to appear in the trend table.")

	// Split positional release tag from flag args. Go's flag package stops
	// at the first non-flag token, so we must extract the tag (which can
	// appear before or after flags) before parsing.
	args := flag.Args()[1:]
	var releaseTag string
	var flagArgs []string
	for _, a := range args {
		if releaseTag == "" && !strings.HasPrefix(a, "-") {
			releaseTag = a
		} else {
			flagArgs = append(flagArgs, a)
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if releaseTag == "" {
		fmt.Fprintln(os.Stderr, "Error: release tag required.")
		fmt.Fprintln(os.Stderr, "Usage: ailang eval-publish <release-tag> --rotation <dir> [--prev <dir> --prev-tag <tag>]")
		os.Exit(1)
	}

	if *rotationDir == "" {
		fmt.Fprintln(os.Stderr, "Error: --rotation <dir> is required.")
		os.Exit(1)
	}
	if *prevRelease != "" && *prevTag == "" {
		fmt.Fprintln(os.Stderr, "Error: --prev-tag is required when --prev is set.")
		os.Exit(1)
	}

	current, err := loadReleaseBenchmarks(*rotationDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading current release: %v\n", err)
		os.Exit(1)
	}

	var previous map[string]eval_harness.BenchmarkSummary
	if *prevRelease != "" {
		prev, err := loadReleaseBenchmarks(*prevRelease)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load previous release: %v\n", err)
		} else {
			previous = prev
		}
	}

	// Primary output: the OS/Local leaderboard JSON the dashboard reads
	// (M-EVAL-BENCHMARK-UI-CONSOLIDATION). Aggregates the (benchmark, model, lang)
	// summaries into model x harness rows with per-language pass rates.
	if *osJSON != "" {
		data, err := buildOSLeaderboardJSON(releaseTag, readAilangVersion(), current)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building OS JSON: %v\n", err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Dir(*osJSON), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating OS JSON dir: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*osJSON, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing OS JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Published %s OS/Local leaderboard JSON to %s\n", releaseTag, *osJSON)
	}

	// Legacy markdown snapshot (retired from the site) — opt-in only.
	if *emitMarkdown {
		markdown := renderReleasePage(releaseTag, *prevTag, current, previous, *minPpDelta)
		if err := os.MkdirAll(*outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
			os.Exit(1)
		}
		outPath := filepath.Join(*outputDir, releaseTag+".md")
		if err := os.WriteFile(outPath, []byte(markdown), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing page: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ (legacy) markdown page → %s\n", outPath)
	}

	fmt.Printf("  Benchmarks: %d\n", uniqueBenchmarkCount(current))
	fmt.Printf("  Models: %d\n", uniqueModelCount(current))
	if previous != nil {
		fmt.Printf("  Trend deltas computed against %s (>=%.1fpp threshold)\n", *prevTag, *minPpDelta)
	}
}

// buildOSLeaderboardJSON aggregates the (benchmark, model, lang) summaries into
// the OS/Local leaderboard schema the dashboard reads: model × harness rows with
// per-language pass rates (M-EVAL-BENCHMARK-UI-CONSOLIDATION). Harness is resolved
// from models.yml (agent_cli), falling back to a name prefix.
func buildOSLeaderboardJSON(releaseTag, ailangVersion string, current map[string]eval_harness.BenchmarkSummary) ([]byte, error) {
	type acc struct{ passed, trials int }
	perModelLang := map[string]map[string]*acc{}
	langsSet := map[string]bool{}
	maxTrials := 0
	for _, s := range current {
		if perModelLang[s.Model] == nil {
			perModelLang[s.Model] = map[string]*acc{}
		}
		a := perModelLang[s.Model][s.Lang]
		if a == nil {
			a = &acc{}
			perModelLang[s.Model][s.Lang] = a
		}
		a.passed += s.Passed
		a.trials += s.Trials
		langsSet[s.Lang] = true
		if s.Trials > maxTrials {
			maxTrials = s.Trials
		}
	}

	langs := make([]string, 0, len(langsSet))
	for l := range langsSet {
		langs = append(langs, l)
	}
	sort.Strings(langs)

	harnessOf := func(model string) string {
		if cfg := eval_harness.GlobalModelsConfig; cfg != nil {
			if mc, ok := cfg.Models[model]; ok && mc.AgentCLI != nil && *mc.AgentCLI != "" {
				return *mc.AgentCLI
			}
		}
		for _, h := range []string{"opencode", "claude", "codex", "pi", "motoko"} {
			if strings.HasPrefix(model, h) {
				return h
			}
		}
		return "local"
	}

	models := make([]string, 0, len(perModelLang))
	for m := range perModelLang {
		models = append(models, m)
	}
	sort.Strings(models)

	rows := make([]map[string]interface{}, 0, len(models))
	for _, m := range models {
		langMap := map[string]float64{}
		for l, a := range perModelLang[m] {
			if a.trials > 0 {
				langMap[l] = float64(a.passed) / float64(a.trials)
			}
		}
		rows = append(rows, map[string]interface{}{
			"model":   m,
			"harness": harnessOf(m),
			"lang":    langMap,
		})
	}

	return json.MarshalIndent(map[string]interface{}{
		"version":        releaseTag,                       // rotation label (date)
		"ailang_version": ailangVersion,                    // AILANG release under test (std/VERSION)
		"generated":      time.Now().Format("2006-01-02"),
		"trials":         maxTrials,
		"languages":      langs,
		"rows":           rows,
	}, "", "  ")
}

// readAilangVersion returns the AILANG language version under test, read from
// std/VERSION (the canonical source) relative to the working directory. The
// rotation and post-release flow both run from the repo root. Returns "" if the
// file is unreadable — callers treat that as "unknown version" rather than
// fabricating one (the version tags the longitudinal trend, so a wrong value
// would be worse than an empty one).
func readAilangVersion() string {
	b, err := os.ReadFile("std/VERSION")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// summaryKey indexes a benchmark summary by the (benchmark, model, lang)
// tuple. Phase-5 publication treats different language runs as separate
// rows; cross-language merging is out of scope.
type summaryKey struct{ Benchmark, Model, Lang string }

// loadReleaseBenchmarks reads every summary.json under rotationDir and
// returns a map keyed by (benchmark, model, lang) -> the merged summary.
// When the same key appears across multiple slots, trials and passes are
// summed and pass_rate is recomputed.
func loadReleaseBenchmarks(rotationDir string) (map[string]eval_harness.BenchmarkSummary, error) {
	summaryFiles, err := findSummaryFiles(rotationDir)
	if err != nil {
		return nil, err
	}
	if len(summaryFiles) == 0 {
		return nil, fmt.Errorf("no summary.json files under %s — was the rotation run with --trials N?", rotationDir)
	}

	merged := map[summaryKey]*eval_harness.BenchmarkSummary{}
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
			k := summaryKey{Benchmark: bs.BenchmarkID, Model: bs.Model, Lang: bs.Lang}
			if existing, ok := merged[k]; ok {
				existing.Trials += bs.Trials
				existing.Passed += bs.Passed
				if existing.Trials > 0 {
					existing.PassRate = float64(existing.Passed) / float64(existing.Trials)
				}
			} else {
				copyBs := bs
				merged[k] = &copyBs
			}
		}
	}

	// Convert to flat map keyed by a stable string for trend lookup.
	out := make(map[string]eval_harness.BenchmarkSummary, len(merged))
	for k, v := range merged {
		out[flatKey(k.Benchmark, k.Model, k.Lang)] = *v
	}
	return out, nil
}

func flatKey(benchmark, model, lang string) string {
	return benchmark + "|" + model + "|" + lang
}

// renderReleasePage produces the Docusaurus markdown for a release.
// Layout:
//   - YAML frontmatter (sidebar_position, title)
//   - intro paragraph
//   - per-benchmark pass-rate matrix (rows = benchmark, columns = model)
//   - optional trend-deltas section vs prev release
func renderReleasePage(releaseTag, prevTag string,
	current, previous map[string]eval_harness.BenchmarkSummary,
	minPpDelta float64,
) string {
	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString("title: " + releaseTag + " OS-model leaderboard\n")
	sb.WriteString("---\n\n")

	sb.WriteString("# " + releaseTag + " OS-model smoke leaderboard\n\n")
	sb.WriteString("Auto-generated by `ailang eval-publish " + releaseTag + "`.\n\n")

	// Build benchmark / model axes for stable iteration.
	benches := uniqueBenchmarks(current)
	models := uniqueModels(current)

	// Per-benchmark pass-rate matrix.
	sb.WriteString("## Per-benchmark pass rate\n\n")
	sb.WriteString("| Benchmark |")
	for _, m := range models {
		sb.WriteString(" " + m + " |")
	}
	sb.WriteString("\n|---|")
	for range models {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")
	for _, b := range benches {
		sb.WriteString("| " + b + " |")
		for _, m := range models {
			cell := "—"
			for _, lang := range uniqueLangs(current, b, m) {
				bs, ok := current[flatKey(b, m, lang)]
				if !ok {
					continue
				}
				cell = fmt.Sprintf("%.0f%% (n=%d)", bs.PassRate*100, bs.Trials)
				break
			}
			sb.WriteString(" " + cell + " |")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Trend deltas vs previous release.
	if previous != nil && prevTag != "" {
		sb.WriteString("## Trend deltas since " + prevTag + "\n\n")
		sb.WriteString(fmt.Sprintf("Benchmarks where pass rate moved by >=%.1f percentage points.\n\n", minPpDelta))
		sb.WriteString("| Benchmark | Model | " + prevTag + " | " + releaseTag + " | Δ |\n")
		sb.WriteString("|---|---|---|---|---|\n")
		anyRow := false
		for _, b := range benches {
			for _, m := range models {
				for _, lang := range uniqueLangs(current, b, m) {
					curr, ok := current[flatKey(b, m, lang)]
					if !ok {
						continue
					}
					prev, ok2 := previous[flatKey(b, m, lang)]
					if !ok2 {
						continue
					}
					deltaPp := (curr.PassRate - prev.PassRate) * 100
					if absFloat(deltaPp) < minPpDelta {
						continue
					}
					arrow := "▲"
					if deltaPp < 0 {
						arrow = "▼"
					}
					sb.WriteString(fmt.Sprintf("| %s | %s | %.0f%% (n=%d) | %.0f%% (n=%d) | %s %+.1fpp |\n",
						b, m,
						prev.PassRate*100, prev.Trials,
						curr.PassRate*100, curr.Trials,
						arrow, deltaPp))
					anyRow = true
				}
			}
		}
		if !anyRow {
			sb.WriteString("(no benchmarks moved by >=" + fmt.Sprintf("%.1fpp", minPpDelta) + ")\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("_Generated from N-trial rotation data via the [local-ollama eval rig](/docs/guides/evaluation/local-ollama)._\n")
	return sb.String()
}

// Helper: collect unique benchmark IDs in stable sorted order.
func uniqueBenchmarks(m map[string]eval_harness.BenchmarkSummary) []string {
	seen := map[string]bool{}
	for _, bs := range m {
		seen[bs.BenchmarkID] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Helper: collect unique model IDs in stable sorted order.
func uniqueModels(m map[string]eval_harness.BenchmarkSummary) []string {
	seen := map[string]bool{}
	for _, bs := range m {
		seen[bs.Model] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Helper: unique langs for a given (benchmark, model). Lang is usually
// single-valued for the local rig but the publication supports multi-lang.
func uniqueLangs(m map[string]eval_harness.BenchmarkSummary, benchmark, model string) []string {
	seen := map[string]bool{}
	for _, bs := range m {
		if bs.BenchmarkID == benchmark && bs.Model == model {
			seen[bs.Lang] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func uniqueBenchmarkCount(m map[string]eval_harness.BenchmarkSummary) int {
	return len(uniqueBenchmarks(m))
}
func uniqueModelCount(m map[string]eval_harness.BenchmarkSummary) int { return len(uniqueModels(m)) }

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
