package eval_analysis

// Cross-dimensional aggregates (tier × model, tag × model, tier ×
// error-category, historical per-tier snapshots) for the dashboard.
//
// Kept in a sibling file because export_json.go is already near the 800
// line soft limit and these are orthogonal concerns — the main exporter
// just calls the builders here and splices the results into DashboardJSON.

// --- Accumulators -------------------------------------------------------

// dimAcc is the running counters for a single (model, lang) cell inside
// a tier or tag aggregate. Kept unexported; converted to
// ModelDimensionStats at the end of aggregation.
type dimAcc struct {
	total          int
	pass           int
	tokens         int
	apiErrors      int
	refusals       int
	firstAttemptOk int
	cost           float64
}

func (a *dimAcc) add(r *BenchmarkResult) {
	a.total++
	if r.StdoutOk {
		a.pass++
	}
	if r.OutputTokens > 0 {
		a.tokens += r.OutputTokens
	}
	if ShouldExcludeFromCapability(r.ErrorCategory) {
		a.apiErrors++
	}
	if r.RefusalDetected {
		a.refusals++
	}
	if r.FirstAttemptOk {
		a.firstAttemptOk++
	}
	a.cost += r.CostUSD
}

func (a *dimAcc) toStats() *ModelDimensionStats {
	if a.total == 0 {
		return nil
	}
	s := &ModelDimensionStats{
		TotalRuns:     a.total,
		SuccessRate:   float64(a.pass) / float64(a.total),
		AvgTokens:     float64(a.tokens) / float64(a.total),
		APIErrorCount: a.apiErrors,
		RefusalCount:  a.refusals,
	}
	return s
}

// --- Tier × model matrix ------------------------------------------------

// buildTierModelMatrix groups results by (tier, model, lang). The tier
// mapping is typically the in-flight benchmarkTier index built by the
// main exporter; callers without that index can pass tagsByID from
// LoadBenchmarkTags plus a parallel tier map.
//
// Benchmarks not in tierOf are skipped silently — matches the existing
// behaviour of the tier aggregate block in export_json.go.
func buildTierModelMatrix(
	results []*BenchmarkResult,
	tierOf map[string]string,
) map[string]map[string]map[string]*dimAcc {
	out := map[string]map[string]map[string]*dimAcc{}
	for _, r := range results {
		tier := tierOf[r.ID]
		if tier == "" {
			continue
		}
		if out[tier] == nil {
			out[tier] = map[string]map[string]*dimAcc{}
		}
		if out[tier][r.Model] == nil {
			out[tier][r.Model] = map[string]*dimAcc{}
		}
		if out[tier][r.Model][r.Lang] == nil {
			out[tier][r.Model][r.Lang] = &dimAcc{}
		}
		out[tier][r.Model][r.Lang].add(r)
	}
	return out
}

// finalizeTierModelMatrix converts the raw accumulator map into the
// exported ModelDimensionStats shape used in TierAggregate.ModelStats.
func finalizeTierModelMatrix(
	matrix map[string]map[string]map[string]*dimAcc,
) map[string]map[string]map[string]*ModelDimensionStats {
	out := map[string]map[string]map[string]*ModelDimensionStats{}
	for tier, byModel := range matrix {
		for model, byLang := range byModel {
			for lang, acc := range byLang {
				stats := acc.toStats()
				if stats == nil {
					continue
				}
				if out[tier] == nil {
					out[tier] = map[string]map[string]*ModelDimensionStats{}
				}
				if out[tier][model] == nil {
					out[tier][model] = map[string]*ModelDimensionStats{}
				}
				out[tier][model][lang] = stats
			}
		}
	}
	return out
}

// --- Tag × model matrix -------------------------------------------------

// buildTagModelMatrix groups results by (tag, model, lang). Each result
// contributes to every tag its benchmark carries, matching the
// GroupByTags convention. Refusal-detected results are included here
// (they carry apiErrors/refusals info the UI wants) — callers that
// want pass/total without refusals should continue using GroupByTags.
func buildTagModelMatrix(
	results []*BenchmarkResult,
	tagsOf map[string][]string,
) map[string]map[string]map[string]*ModelDimensionStats {
	accs := map[string]map[string]map[string]*dimAcc{}
	for _, r := range results {
		for _, tag := range tagsOf[r.ID] {
			if accs[tag] == nil {
				accs[tag] = map[string]map[string]*dimAcc{}
			}
			if accs[tag][r.Model] == nil {
				accs[tag][r.Model] = map[string]*dimAcc{}
			}
			if accs[tag][r.Model][r.Lang] == nil {
				accs[tag][r.Model][r.Lang] = &dimAcc{}
			}
			accs[tag][r.Model][r.Lang].add(r)
		}
	}

	out := map[string]map[string]map[string]*ModelDimensionStats{}
	for tag, byModel := range accs {
		for model, byLang := range byModel {
			for lang, acc := range byLang {
				stats := acc.toStats()
				if stats == nil {
					continue
				}
				if out[tag] == nil {
					out[tag] = map[string]map[string]*ModelDimensionStats{}
				}
				if out[tag][model] == nil {
					out[tag][model] = map[string]*ModelDimensionStats{}
				}
				out[tag][model][lang] = stats
			}
		}
	}
	return out
}

// --- Tier aggregates (repair, cost, reliability) ------------------------

// tierExtras collects per-language repair-delta, cost, api-error, and
// refusal data for a single tier. Returned by computeTierExtras.
type tierExtras struct {
	langs         map[string]*TierLanguageStats
	apiErrorCount int
	refusalCount  int
}

// computeTierExtras walks results once per tier and computes per-language
// self-repair deltas, average cost, api-error counts, and refusal counts.
// All eval languages (python, ailang, javascript, go, …) are handled
// generically — no language is silently dropped.
func computeTierExtras(
	results []*BenchmarkResult,
	tierOf map[string]string,
) map[string]*tierExtras {
	type perLang struct {
		runs      int
		firstOk   int
		stdoutOk  int
		apiErrors int
		refusals  int
		cost      float64
	}
	type perTier struct {
		langs map[string]*perLang
	}
	tierAcc := map[string]*perTier{}
	for _, r := range results {
		tier := tierOf[r.ID]
		if tier == "" {
			continue
		}
		if tierAcc[tier] == nil {
			tierAcc[tier] = &perTier{langs: map[string]*perLang{}}
		}
		if tierAcc[tier].langs[r.Lang] == nil {
			tierAcc[tier].langs[r.Lang] = &perLang{}
		}
		pl := tierAcc[tier].langs[r.Lang]
		pl.runs++
		if r.FirstAttemptOk {
			pl.firstOk++
		}
		if r.StdoutOk {
			pl.stdoutOk++
		}
		if ShouldExcludeFromCapability(r.ErrorCategory) {
			pl.apiErrors++
		}
		if r.RefusalDetected {
			pl.refusals++
		}
		pl.cost += r.CostUSD
	}

	rate := func(num, denom int) float64 {
		if denom == 0 {
			return 0
		}
		return float64(num) / float64(denom)
	}
	avg := func(total float64, denom int) float64 {
		if denom == 0 {
			return 0
		}
		return total / float64(denom)
	}
	out := map[string]*tierExtras{}
	for tier, pt := range tierAcc {
		ex := &tierExtras{langs: make(map[string]*TierLanguageStats, len(pt.langs))}
		for lang, pl := range pt.langs {
			ex.langs[lang] = &TierLanguageStats{
				Runs:        pl.runs,
				Pass:        pl.stdoutOk,
				SuccessRate: rate(pl.stdoutOk, pl.runs),
				RepairDelta: rate(pl.stdoutOk, pl.runs) - rate(pl.firstOk, pl.runs),
				AvgCostUSD:  avg(pl.cost, pl.runs),
				APIErrors:   pl.apiErrors,
			}
			ex.apiErrorCount += pl.apiErrors
			ex.refusalCount += pl.refusals
		}
		out[tier] = ex
	}
	return out
}

// --- Top-level reliability counters ------------------------------------

// ReliabilityCounts is a small bag of counters surfaced at the top level
// of DashboardJSON.aggregates so the "API Reliability" card can render
// without drilling into tiers/models.
type ReliabilityCounts struct {
	APIErrorCount int                          `json:"apiErrorCount"`
	APIErrorRate  float64                      `json:"apiErrorRate"`
	RefusalCount  int                          `json:"refusalCount"`
	RefusalRate   float64                      `json:"refusalRate"`
	PerModel      map[string]*ModelReliability `json:"perModel,omitempty"`
}

// ModelReliability is the per-model counterpart: useful for the hover
// breakdown on the reliability card ("gemini-3-1-pro: 13/33 api errors").
type ModelReliability struct {
	APIErrorCount int     `json:"apiErrorCount"`
	APIErrorRate  float64 `json:"apiErrorRate"`
	RefusalCount  int     `json:"refusalCount"`
	RefusalRate   float64 `json:"refusalRate"`
	TotalRuns     int     `json:"totalRuns"`
	// Per-language api-error counts. AILANGAPIError/PythonAPIError kept for
	// backward compatibility; LanguageAPIErrors covers all eval languages.
	AILANGAPIError    int            `json:"ailangApiError,omitempty"`
	PythonAPIError    int            `json:"pythonApiError,omitempty"`
	LanguageAPIErrors map[string]int `json:"language_api_errors,omitempty"`
}

// computeReliability scans the standard results once and returns the
// global + per-model reliability counters for all eval languages.
func computeReliability(results []*BenchmarkResult) *ReliabilityCounts {
	r := &ReliabilityCounts{PerModel: map[string]*ModelReliability{}}
	for _, res := range results {
		r.PerModel[res.Model] = getOrCreateReliability(r.PerModel, res.Model)
		m := r.PerModel[res.Model]
		m.TotalRuns++
		if ShouldExcludeFromCapability(res.ErrorCategory) {
			r.APIErrorCount++
			m.APIErrorCount++
			if m.LanguageAPIErrors == nil {
				m.LanguageAPIErrors = map[string]int{}
			}
			m.LanguageAPIErrors[res.Lang]++
		}
		if res.RefusalDetected {
			r.RefusalCount++
			m.RefusalCount++
		}
	}
	total := len(results)
	if total > 0 {
		r.APIErrorRate = float64(r.APIErrorCount) / float64(total)
		r.RefusalRate = float64(r.RefusalCount) / float64(total)
	}
	for _, m := range r.PerModel {
		if m.TotalRuns > 0 {
			m.APIErrorRate = float64(m.APIErrorCount) / float64(m.TotalRuns)
			m.RefusalRate = float64(m.RefusalCount) / float64(m.TotalRuns)
		}
		// backward-compat typed fields populated from the generic map
		m.AILANGAPIError = m.LanguageAPIErrors["ailang"]
		m.PythonAPIError = m.LanguageAPIErrors["python"]
	}
	return r
}

func getOrCreateReliability(m map[string]*ModelReliability, key string) *ModelReliability {
	if v, ok := m[key]; ok {
		return v
	}
	v := &ModelReliability{}
	m[key] = v
	return v
}

// --- Historical tier stats ---------------------------------------------

// buildHistoricalTierPoints recomputes per-tier per-model stats for a
// single baseline's results, applying the CURRENT tier tags retroactively.
// Pre-v0.14.0 baselines get the current mapping — that is an
// approximation (a benchmark's current tier is used for its historic
// runs too) but means the time-series chart can filter by tier without
// waiting for every old baseline to be re-run.
//
// Benchmarks absent from tierOf in the historic baseline are skipped
// silently (happens for benchmarks added AFTER the baseline was taken;
// there's no contribution to record).
func buildHistoricalTierPoints(
	results []*BenchmarkResult,
	tierOf map[string]string,
) map[string]*TierHistoryPoint {
	matrix := buildTierModelMatrix(results, tierOf)
	stats := finalizeTierModelMatrix(matrix)

	// Compute per-language aggregate pass rates per tier — all eval languages
	// (python, ailang, javascript, go, …) are handled generically.
	type langCount struct{ total, pass int }
	type perTier struct {
		langs    map[string]*langCount
		benchIDs map[string]struct{}
	}
	tier := map[string]*perTier{}
	for _, r := range results {
		t := tierOf[r.ID]
		if t == "" {
			continue
		}
		if tier[t] == nil {
			tier[t] = &perTier{langs: map[string]*langCount{}, benchIDs: map[string]struct{}{}}
		}
		pt := tier[t]
		pt.benchIDs[r.ID] = struct{}{}
		if pt.langs[r.Lang] == nil {
			pt.langs[r.Lang] = &langCount{}
		}
		pt.langs[r.Lang].total++
		if r.StdoutOk {
			pt.langs[r.Lang].pass++
		}
	}

	out := map[string]*TierHistoryPoint{}
	for t, pt := range tier {
		p := &TierHistoryPoint{
			BenchmarkCount: len(pt.benchIDs),
			LanguageStats:  make(map[string]*TierLanguageStats, len(pt.langs)),
		}
		for lang, lc := range pt.langs {
			sr := 0.0
			if lc.total > 0 {
				sr = float64(lc.pass) / float64(lc.total)
			}
			p.LanguageStats[lang] = &TierLanguageStats{Runs: lc.total, Pass: lc.pass, SuccessRate: sr}
		}
		// backward-compat typed fields
		if ls := p.LanguageStats["ailang"]; ls != nil {
			p.AILANGRuns, p.AILANGSuccessRate = ls.Runs, ls.SuccessRate
		}
		if ls := p.LanguageStats["python"]; ls != nil {
			p.PythonRuns, p.PythonSuccessRate = ls.Runs, ls.SuccessRate
		}
		if stats[t] != nil {
			p.ModelStats = stats[t]
		}
		out[t] = p
	}
	return out
}

// --- Tier + tag aggregate builders used by the main exporter -----------

// buildTierAggregates groups standard results by tier and emits the
// ExportBenchmarkJSON-facing TierAggregate map — pass rates, per-model
// cross-sections, and the repair/cost/reliability extras. Extracted from
// export_json.go so that file stays under the 800-line soft limit.
func buildTierAggregates(
	results []*BenchmarkResult,
	tierOf map[string]string,
) map[string]TierAggregate {
	type langCount struct{ runs, pass int }
	type tierAcc struct {
		langs    map[string]*langCount
		benchIDs map[string]bool
	}
	tierAccs := map[string]*tierAcc{}
	for _, r := range results {
		tier := tierOf[r.ID]
		if tier == "" {
			continue
		}
		acc, ok := tierAccs[tier]
		if !ok {
			acc = &tierAcc{langs: map[string]*langCount{}, benchIDs: map[string]bool{}}
			tierAccs[tier] = acc
		}
		acc.benchIDs[r.ID] = true
		if acc.langs[r.Lang] == nil {
			acc.langs[r.Lang] = &langCount{}
		}
		acc.langs[r.Lang].runs++
		if r.StdoutOk {
			acc.langs[r.Lang].pass++
		}
	}
	modelStats := finalizeTierModelMatrix(buildTierModelMatrix(results, tierOf))
	extras := computeTierExtras(results, tierOf)

	out := make(map[string]TierAggregate, len(tierAccs))
	for tier, acc := range tierAccs {
		agg := TierAggregate{
			BenchmarkCount: len(acc.benchIDs),
			LanguageStats:  make(map[string]*TierLanguageStats, len(acc.langs)),
		}
		for lang, lc := range acc.langs {
			sr := 0.0
			if lc.runs > 0 {
				sr = float64(lc.pass) / float64(lc.runs)
			}
			agg.LanguageStats[lang] = &TierLanguageStats{Runs: lc.runs, Pass: lc.pass, SuccessRate: sr}
			agg.TotalRuns += lc.runs
		}
		// backward-compat typed fields
		if ls := agg.LanguageStats["ailang"]; ls != nil {
			agg.AILANGRuns, agg.AILANGSuccessRate = ls.Runs, ls.SuccessRate
		}
		if ls := agg.LanguageStats["python"]; ls != nil {
			agg.PythonRuns, agg.PythonSuccessRate = ls.Runs, ls.SuccessRate
		}
		if ms := modelStats[tier]; ms != nil {
			agg.ModelStats = ms
		}
		if ex := extras[tier]; ex != nil {
			agg.APIErrorCount = ex.apiErrorCount
			agg.RefusalCount = ex.refusalCount
			for lang, ls := range ex.langs {
				if tls := agg.LanguageStats[lang]; tls != nil {
					tls.RepairDelta = ls.RepairDelta
					tls.AvgCostUSD = ls.AvgCostUSD
					tls.APIErrors = ls.APIErrors
				} else {
					agg.LanguageStats[lang] = ls
				}
			}
			// backward-compat typed extras
			if ls := ex.langs["ailang"]; ls != nil {
				agg.AILANGRepairDelta = ls.RepairDelta
				agg.AILANGAvgCostUSD = ls.AvgCostUSD
				agg.AILANGAPIError = ls.APIErrors
			}
			if ls := ex.langs["python"]; ls != nil {
				agg.PythonRepairDelta = ls.RepairDelta
				agg.PythonAvgCostUSD = ls.AvgCostUSD
				agg.PythonAPIError = ls.APIErrors
			}
		}
		out[tier] = agg
	}
	return out
}

// buildTagAggregates loads the benchmark tag index and emits the
// TagAggregate map that DashboardJSON.Tags serialises. Uses GroupByTags
// for refusal-filtered pass/total and buildTagModelMatrix for the
// per-model cross-section.
func buildTagAggregates(results []*BenchmarkResult) map[string]*TagAggregate {
	tagsOf := LoadBenchmarkTags("benchmarks")
	report := GroupByTags(results, tagsOf)
	modelStats := buildTagModelMatrix(results, tagsOf)

	benchCount := map[string]map[string]bool{}
	for benchID, tagList := range tagsOf {
		for _, tag := range tagList {
			if benchCount[tag] == nil {
				benchCount[tag] = map[string]bool{}
			}
			benchCount[tag][benchID] = true
		}
	}

	out := map[string]*TagAggregate{}
	for tag, agg := range report.Aggregates {
		if ms := modelStats[tag]; ms != nil {
			agg.ModelStats = ms
		}
		agg.BenchmarkCount = len(benchCount[tag])
		out[tag] = agg
	}
	return out
}
