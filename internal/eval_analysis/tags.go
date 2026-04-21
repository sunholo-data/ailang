package eval_analysis

import (
	"path/filepath"
	"sort"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// TagAggregate summarises pass/total counts for one tag, per language,
// plus the AILANG vs Python delta in [-1,1].
type TagAggregate struct {
	Tag         string  `json:"tag"`
	AILANGPass  int     `json:"ailang_pass"`
	AILANGTotal int     `json:"ailang_total"`
	PythonPass  int     `json:"python_pass"`
	PythonTotal int     `json:"python_total"`
	Delta       float64 `json:"delta"` // ailangRate - pythonRate
	// M-DASH-V2: unique benchmark IDs carrying this tag (useful for the
	// UI "N benchmarks in tag" chip).
	BenchmarkCount int `json:"benchmark_count,omitempty"`
	// M-DASH-V2: per-model cross-section so the dashboard can render
	// per-model bars filtered to this tag. Outer key is model name,
	// inner key is language.
	ModelStats map[string]map[string]*ModelDimensionStats `json:"model_stats,omitempty"`
}

// TagReport is the output of GroupByTags: a sorted tag list plus the
// per-tag aggregates.
type TagReport struct {
	Tags       []string                 `json:"tags"`
	Aggregates map[string]*TagAggregate `json:"aggregates"`
}

// AILANGWin names a (benchmark, model) cell where AILANG passed and Python
// failed — the atom of the AILANG-only-wins report.
type AILANGWin struct {
	ID    string `json:"id"`
	Model string `json:"model"`
}

// AILANGWinsReport aggregates wins at the cell level plus a pattern list
// of benchmarks where ≥3 distinct models agree that AILANG wins.
type AILANGWinsReport struct {
	Wins         []AILANGWin    `json:"wins"`
	PerBenchmark map[string]int `json:"per_benchmark"` // benchmark -> distinct models winning
	Patterns     []string       `json:"patterns"`      // benchmarks with ≥3 models winning
}

// SaturatedBenchmark names a benchmark that hit 100% pass across every
// model × language pair in the considered baselines.
type SaturatedBenchmark struct {
	ID            string   `json:"id"`
	BaselinesSeen []string `json:"baselines_seen"` // versions contributing
	TotalCells    int      `json:"total_cells"`    // model × lang pairs
}

// LoadBenchmarkTags reads every YAML in dir and returns a map of
// benchmark ID -> tag list. Benchmarks with unreadable specs are
// skipped silently — LoadSpec already warns on unknown tags.
func LoadBenchmarkTags(dir string) map[string][]string {
	out := map[string][]string{}
	matches, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		return out
	}
	for _, path := range matches {
		spec, err := eval_harness.LoadSpec(path)
		if err != nil {
			continue
		}
		out[spec.ID] = spec.Tags
	}
	return out
}

// GroupByTags builds a TagReport from benchmark results and the tag index
// from LoadBenchmarkTags. Results flagged RefusalDetected are excluded
// from pass/total counts so refusals do not inflate failure rates for
// every tag the benchmark happened to carry.
//
// Each benchmark contributes to every tag it carries; a (benchmark, lang,
// model) run is one unit, so a benchmark tagged adt_pattern_match +
// recursion counts once in each column.
func GroupByTags(results []*BenchmarkResult, tags map[string][]string) *TagReport {
	type cell struct{ pass, total int }
	byTag := map[string]map[string]*cell{}

	for _, r := range results {
		if r.RefusalDetected {
			continue
		}
		for _, tag := range tags[r.ID] {
			if byTag[tag] == nil {
				byTag[tag] = map[string]*cell{}
			}
			if byTag[tag][r.Lang] == nil {
				byTag[tag][r.Lang] = &cell{}
			}
			c := byTag[tag][r.Lang]
			c.total++
			if r.StdoutOk {
				c.pass++
			}
		}
	}

	report := &TagReport{
		Aggregates: map[string]*TagAggregate{},
	}
	for tag, langs := range byTag {
		ail := langs["ailang"]
		py := langs["python"]
		agg := &TagAggregate{Tag: tag}
		if ail != nil {
			agg.AILANGPass, agg.AILANGTotal = ail.pass, ail.total
		}
		if py != nil {
			agg.PythonPass, agg.PythonTotal = py.pass, py.total
		}
		agg.Delta = cellRate(agg.AILANGPass, agg.AILANGTotal) - cellRate(agg.PythonPass, agg.PythonTotal)
		report.Aggregates[tag] = agg
		report.Tags = append(report.Tags, tag)
	}
	sort.Strings(report.Tags)
	return report
}

// DetectAILANGOnlyWins finds cells where AILANG passes and Python fails
// for the same (benchmark, model). A benchmark is a "pattern" win when
// ≥3 distinct models agree on it. If either language refused at a cell,
// the whole cell is dropped — a Python refusal masquerading as a failure
// would otherwise produce a false-positive win.
func DetectAILANGOnlyWins(results []*BenchmarkResult) *AILANGWinsReport {
	type key struct{ id, model string }
	type cellState struct {
		langPass map[string]bool
		refused  bool
	}
	byKey := map[key]*cellState{}
	for _, r := range results {
		k := key{r.ID, r.Model}
		if byKey[k] == nil {
			byKey[k] = &cellState{langPass: map[string]bool{}}
		}
		s := byKey[k]
		if r.RefusalDetected {
			s.refused = true
			continue
		}
		if r.StdoutOk {
			s.langPass[r.Lang] = true
		} else if _, seen := s.langPass[r.Lang]; !seen {
			s.langPass[r.Lang] = false
		}
	}

	report := &AILANGWinsReport{PerBenchmark: map[string]int{}}
	winModels := map[string]map[string]bool{} // id -> set of models

	for k, s := range byKey {
		if s.refused {
			continue
		}
		langs := s.langPass
		if langs["ailang"] && !langs["python"] {
			report.Wins = append(report.Wins, AILANGWin{ID: k.id, Model: k.model})
			if winModels[k.id] == nil {
				winModels[k.id] = map[string]bool{}
			}
			winModels[k.id][k.model] = true
		}
	}

	for id, models := range winModels {
		report.PerBenchmark[id] = len(models)
		if len(models) >= 3 {
			report.Patterns = append(report.Patterns, id)
		}
	}

	sort.Slice(report.Wins, func(i, j int) bool {
		if report.Wins[i].ID != report.Wins[j].ID {
			return report.Wins[i].ID < report.Wins[j].ID
		}
		return report.Wins[i].Model < report.Wins[j].Model
	})
	sort.Strings(report.Patterns)
	return report
}

// DetectSaturation returns benchmarks that pass 100% across every
// (model, language) cell in all considered baselines. Only baselines
// with ≥1 AILANG result are considered, to avoid "saturated" Python-only
// baselines reporting spurious wins.
//
// If fewer than minBaselines baselines are available, saturation is
// computed over the ones that exist — better to return partial data
// with a clear "BaselinesSeen" list than nothing.
func DetectSaturation(baselines []*Baseline, minBaselines int) []*SaturatedBenchmark {
	if len(baselines) == 0 {
		return nil
	}

	// Filter to baselines containing AILANG results (skip Python-only
	// legacy baselines).
	var usable []*Baseline
	for _, b := range baselines {
		for _, r := range b.Results {
			if r.Lang == "ailang" {
				usable = append(usable, b)
				break
			}
		}
	}
	if len(usable) == 0 {
		return nil
	}

	// Track per-(benchmark, baseline) cell pass/total. Refusals excluded.
	type cellKey struct{ id, baseline, lang, model string }
	cells := map[cellKey]*struct{ pass, total int }{}
	for _, b := range usable {
		for _, r := range b.Results {
			if r.RefusalDetected {
				continue
			}
			k := cellKey{r.ID, b.Version, r.Lang, r.Model}
			if cells[k] == nil {
				cells[k] = &struct{ pass, total int }{}
			}
			cells[k].total++
			if r.StdoutOk {
				cells[k].pass++
			}
		}
	}

	// For each benchmark, check every cell across every usable baseline.
	benchCells := map[string]map[string][]cellKey{} // id -> baseline -> cells
	for k := range cells {
		if benchCells[k.id] == nil {
			benchCells[k.id] = map[string][]cellKey{}
		}
		benchCells[k.id][k.baseline] = append(benchCells[k.id][k.baseline], k)
	}

	var out []*SaturatedBenchmark
	for id, byBaseline := range benchCells {
		// Require data in every usable baseline (or at least minBaselines).
		needed := len(usable)
		if minBaselines > 0 && minBaselines < needed {
			needed = minBaselines
		}
		if len(byBaseline) < needed {
			continue
		}
		allPass := true
		totalCells := 0
		var seenVersions []string
		for version, ks := range byBaseline {
			seenVersions = append(seenVersions, version)
			totalCells += len(ks)
			for _, k := range ks {
				c := cells[k]
				if c.total == 0 || c.pass < c.total {
					allPass = false
					break
				}
			}
			if !allPass {
				break
			}
		}
		if allPass {
			sort.Strings(seenVersions)
			out = append(out, &SaturatedBenchmark{
				ID:            id,
				BaselinesSeen: seenVersions,
				TotalCells:    totalCells,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// cellRate returns pass/total in [0,1], zero when total is zero. Split out
// so callers that already have the int pair don't need to thread a
// *passCell through the CLI/library boundary.
func cellRate(pass, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(pass) / float64(total)
}
