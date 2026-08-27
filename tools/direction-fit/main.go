// Command direction-fit computes the per-release LANGUAGE-DIRECTION INDEX
// (M-EVAL-ROLLING-ELO M3): it holds the bridge models' strengths fixed (from
// the current placement fit in observatory.db) and refits the direction
// panel's benchmark difficulties from ONE release's linking-run trials.
// Falling difficulty release-over-release = the language/prompt got easier
// for the same models.
//
// The output is a STAMPED MEASUREMENT: the input bridge strengths are recorded
// alongside the result and the artifact is never recomputed (same contract as
// banked costs). Missing panel cells FAIL LOUDLY — a partial index is a silent
// fallback and is refused (design D3).
//
//	go run ./tools/direction-fit \
//	    --version v0.35.0 \
//	    --bridge claude-sonnet-5,gpt5-6-terra,gemini-3-7-flash,or-glm-5-3-flash,or-deepseek-v4-flash \
//	    --db ~/.ailang/state/observatory.db \
//	    --out eval_results/baselines/v0.35.0/direction_index.json \
//	    <release_results_dir>
//
// (flags before the positional dir — Go flag parsing stops at the first
// non-flag argument)
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/observatory"

	_ "github.com/mattn/go-sqlite3"
)

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	version := flag.String("version", "", "release version label (required)")
	bridge := flag.String("bridge", "", "comma-separated bridge model ids (required)")
	dbPath := flag.String("db", "", "observatory.db with the current placement fit (required)")
	panelPath := flag.String("panel", "internal/eval_harness/direction_panel_v1.json", "direction panel JSON")
	benchDir := flag.String("bench-dir", "benchmarks", "benchmark spec dir (for tier lookup)")
	outPath := flag.String("out", "", "output artifact path (required)")
	langsFlag := flag.String("langs", "ailang,python", "languages every panel cell must cover")
	flag.Parse()
	if flag.NArg() != 1 || *version == "" || *bridge == "" || *dbPath == "" || *outPath == "" {
		fatal("usage: direction-fit <release_results_dir> --version V --bridge a,b,c --db PATH --out PATH")
	}
	resultsDir := flag.Arg(0)
	bridgeModels := strings.Split(*bridge, ",")
	langs := strings.Split(*langsFlag, ",")

	// Panel (generated file; refuse an empty or unreadable one).
	var panel struct {
		Version    string             `json:"version"`
		Benchmarks map[string]float64 `json:"benchmarks"`
	}
	raw, err := os.ReadFile(*panelPath)
	if err != nil {
		fatal("read panel: %v", err)
	}
	if err := json.Unmarshal(raw, &panel); err != nil || len(panel.Benchmarks) == 0 {
		fatal("panel %s is invalid or empty (err=%v)", *panelPath, err)
	}

	// Bridge strengths = the current placement fit. A bridge member without a
	// stored standard rating cannot be held fixed — loud failure, no seeding.
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		fatal("open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	rows, err := observatory.LoadModelRatings(context.Background(), db, "standard")
	if err != nil {
		fatal("load placement ratings: %v", err)
	}
	stored := map[string]float64{}
	for _, r := range rows {
		stored[r.ModelID] = r.Rating
	}
	bridgeStrengths := map[string]float64{}
	for _, m := range bridgeModels {
		v, ok := stored[m]
		if !ok {
			fatal("bridge model %q has no standard placement rating in %s — run the placement persist first", m, *dbPath)
		}
		bridgeStrengths[m] = v
	}

	// Load the release's trials: standard mode, validity-filtered, restricted
	// to panel benchmarks x bridge models.
	results, err := eval_analysis.LoadResults(resultsDir)
	if err != nil {
		fatal("load results: %v", err)
	}
	inBridge := map[string]bool{}
	for _, m := range bridgeModels {
		inBridge[m] = true
	}
	var trials []eval_harness.Trial
	covered := map[string]bool{} // bench|model|lang
	for _, r := range results {
		if r.EvalMode == "agent" || !inBridge[r.Model] {
			continue
		}
		if _, ok := panel.Benchmarks[r.ID]; !ok {
			continue
		}
		trials = append(trials, eval_harness.Trial{Model: r.Model, Bench: r.ID, Pass: r.CompileOk && r.RuntimeOk && r.StdoutOk})
		covered[r.ID+"|"+r.Model+"|"+r.Lang] = true
	}

	// Completeness gate: every panel x bridge x lang cell must have a valid
	// banked row. A missing cell means the index would silently average over a
	// different set than last release — refused (no partial index).
	//
	// Languages are per-benchmark, NOT global: a spec may declare
	// `languages: ["ailang"]` (e.g. ai_effect_json_schema — AILANG-only by
	// construction), and demanding a python cell for it would make the index
	// permanently unachievable. Required set = requested langs INTERSECT the
	// benchmark's declared langs. Measured 2026-08-27 during the M3 rehearsal:
	// the first gate refused on 2 cells that can never exist.
	langsFor := func(id string) []string {
		spec, err := eval_harness.LoadSpec(filepath.Join(*benchDir, id+".yml"))
		if err != nil || len(spec.Languages) == 0 {
			return langs // no spec info: fall back to the requested set
		}
		declared := map[string]bool{}
		for _, l := range spec.Languages {
			declared[l] = true
		}
		var out []string
		for _, l := range langs {
			if declared[l] {
				out = append(out, l)
			}
		}
		return out
	}
	var missing []string
	for b := range panel.Benchmarks {
		want := langsFor(b)
		if len(want) == 0 {
			fatal("panel benchmark %q declares none of the requested languages %v — panel and --langs are incompatible", b, langs)
		}
		for _, m := range bridgeModels {
			for _, l := range want {
				if !covered[b+"|"+m+"|"+l] {
					missing = append(missing, b+" x "+m+" x "+l)
				}
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fmt.Fprintf(os.Stderr, "REFUSING to compute a partial direction index: %d missing panel cell(s):\n", len(missing))
		for _, m := range missing {
			fmt.Fprintf(os.Stderr, "  %s\n", m)
		}
		os.Exit(1)
	}

	// Direction fit: bridge strengths FIXED, panel difficulties free.
	_, benchRatings := eval_harness.FitFromTrialsAnchored(trials, nil, bridgeStrengths)

	tierOf := func(id string) string {
		spec, err := eval_harness.LoadSpec(filepath.Join(*benchDir, id+".yml"))
		if err != nil {
			return "core"
		}
		return spec.Tier
	}
	sum, n := 0.0, 0
	byTierSum := map[string]float64{}
	byTierN := map[string]int{}
	difficulties := map[string]float64{}
	for b := range panel.Benchmarks {
		d := benchRatings[b]
		difficulties[b] = float64(int(d*10)) / 10
		sum += d
		n++
		t := tierOf(b)
		byTierSum[t] += d
		byTierN[t]++
	}
	byTier := map[string]float64{}
	for t, s := range byTierSum {
		byTier[t] = float64(int(s/float64(byTierN[t])*10)) / 10
	}

	doc := struct {
		Version         string             `json:"version"`
		PanelVersion    string             `json:"panel_version"`
		Generated       string             `json:"generated"`
		IndexOverall    float64            `json:"index_overall"`
		IndexByTier     map[string]float64 `json:"index_by_tier"`
		BridgeStrengths map[string]float64 `json:"bridge_strengths_used"`
		Trials          int                `json:"trials"`
		Difficulties    map[string]float64 `json:"difficulties"`
		Note            string             `json:"note"`
	}{
		Version:         *version,
		PanelVersion:    panel.Version,
		Generated:       time.Now().UTC().Format(time.RFC3339),
		IndexOverall:    float64(int(sum/float64(n)*10)) / 10,
		IndexByTier:     byTier,
		BridgeStrengths: bridgeStrengths,
		Trials:          len(trials),
		Difficulties:    difficulties,
		Note:            "Stamped measurement: computed once at release time with the recorded bridge strengths; never recomputed. Falling index = AILANG/prompt got easier for the same models.",
	}
	buf, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fatal("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fatal("mkdir: %v", err)
	}
	if err := os.WriteFile(*outPath, append(buf, '\n'), 0o644); err != nil {
		fatal("write: %v", err)
	}
	fmt.Printf("direction index %s: overall %.1f, by tier %v (%d trials) -> %s\n",
		*version, doc.IndexOverall, byTier, len(trials), *outPath)
}
