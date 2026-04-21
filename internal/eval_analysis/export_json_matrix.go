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
	if r.ErrorCategory == "api_error" {
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

// tierExtras collects the repair-delta and cost-per-lang metrics for a
// single tier. Returned by computeTierExtras for splicing into the
// existing TierAggregate struct.
type tierExtras struct {
	AILANGRepairDelta float64
	PythonRepairDelta float64
	AILANGAvgCost     float64
	PythonAvgCost     float64
	APIErrorCount     int
	AILANGAPIError    int
	PythonAPIError    int
	RefusalCount      int
}

// computeTierExtras walks results one more time and computes per-tier
// self-repair deltas, average cost (per lang), api-error counts, and
// refusal counts. Kept separate from buildTierModelMatrix so the two
// aren't coupled; the main exporter is free to call either or both.
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
		ailang perLang
		python perLang
	}
	tierAcc := map[string]*perTier{}
	for _, r := range results {
		tier := tierOf[r.ID]
		if tier == "" {
			continue
		}
		if tierAcc[tier] == nil {
			tierAcc[tier] = &perTier{}
		}
		var pl *perLang
		switch r.Lang {
		case "ailang":
			pl = &tierAcc[tier].ailang
		case "python":
			pl = &tierAcc[tier].python
		default:
			continue
		}
		pl.runs++
		if r.FirstAttemptOk {
			pl.firstOk++
		}
		if r.StdoutOk {
			pl.stdoutOk++
		}
		if r.ErrorCategory == "api_error" {
			pl.apiErrors++
		}
		if r.RefusalDetected {
			pl.refusals++
		}
		pl.cost += r.CostUSD
	}

	out := map[string]*tierExtras{}
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
	for tier, acc := range tierAcc {
		out[tier] = &tierExtras{
			AILANGRepairDelta: rate(acc.ailang.stdoutOk, acc.ailang.runs) -
				rate(acc.ailang.firstOk, acc.ailang.runs),
			PythonRepairDelta: rate(acc.python.stdoutOk, acc.python.runs) -
				rate(acc.python.firstOk, acc.python.runs),
			AILANGAvgCost:  avg(acc.ailang.cost, acc.ailang.runs),
			PythonAvgCost:  avg(acc.python.cost, acc.python.runs),
			APIErrorCount:  acc.ailang.apiErrors + acc.python.apiErrors,
			AILANGAPIError: acc.ailang.apiErrors,
			PythonAPIError: acc.python.apiErrors,
			RefusalCount:   acc.ailang.refusals + acc.python.refusals,
		}
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
	// Per-language api-error counts so the UI can say "AILANG 10, Python 3".
	AILANGAPIError int `json:"ailangApiError,omitempty"`
	PythonAPIError int `json:"pythonApiError,omitempty"`
}

// computeReliability scans the standard results once and returns the
// global + per-model reliability counters.
func computeReliability(results []*BenchmarkResult) *ReliabilityCounts {
	r := &ReliabilityCounts{PerModel: map[string]*ModelReliability{}}
	for _, res := range results {
		r.PerModel[res.Model] = getOrCreateReliability(r.PerModel, res.Model)
		m := r.PerModel[res.Model]
		m.TotalRuns++
		if res.ErrorCategory == "api_error" {
			r.APIErrorCount++
			m.APIErrorCount++
			if res.Lang == "ailang" {
				m.AILANGAPIError++
			} else if res.Lang == "python" {
				m.PythonAPIError++
			}
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

	// Separately compute AILANG/Python aggregate pass rates per tier so
	// the TierHistoryPoint can fill its own ailang/python runs fields
	// without the frontend having to sum ModelStats.
	type perTier struct {
		aTotal, aPass int
		pTotal, pPass int
		benchIDs      map[string]struct{}
	}
	tier := map[string]*perTier{}
	for _, r := range results {
		t := tierOf[r.ID]
		if t == "" {
			continue
		}
		if tier[t] == nil {
			tier[t] = &perTier{benchIDs: map[string]struct{}{}}
		}
		pt := tier[t]
		pt.benchIDs[r.ID] = struct{}{}
		switch r.Lang {
		case "ailang":
			pt.aTotal++
			if r.StdoutOk {
				pt.aPass++
			}
		case "python":
			pt.pTotal++
			if r.StdoutOk {
				pt.pPass++
			}
		}
	}

	out := map[string]*TierHistoryPoint{}
	for t, pt := range tier {
		p := &TierHistoryPoint{
			AILANGRuns:     pt.aTotal,
			PythonRuns:     pt.pTotal,
			BenchmarkCount: len(pt.benchIDs),
		}
		if pt.aTotal > 0 {
			p.AILANGSuccessRate = float64(pt.aPass) / float64(pt.aTotal)
		}
		if pt.pTotal > 0 {
			p.PythonSuccessRate = float64(pt.pPass) / float64(pt.pTotal)
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
	type tierAcc struct {
		ailangRuns, ailangPass int
		pythonRuns, pythonPass int
		benchIDs               map[string]bool
	}
	tierAccs := map[string]*tierAcc{}
	for _, r := range results {
		tier := tierOf[r.ID]
		if tier == "" {
			continue
		}
		acc, ok := tierAccs[tier]
		if !ok {
			acc = &tierAcc{benchIDs: map[string]bool{}}
			tierAccs[tier] = acc
		}
		acc.benchIDs[r.ID] = true
		switch r.Lang {
		case "ailang":
			acc.ailangRuns++
			if r.StdoutOk {
				acc.ailangPass++
			}
		case "python":
			acc.pythonRuns++
			if r.StdoutOk {
				acc.pythonPass++
			}
		}
	}
	modelStats := finalizeTierModelMatrix(buildTierModelMatrix(results, tierOf))
	extras := computeTierExtras(results, tierOf)

	out := make(map[string]TierAggregate, len(tierAccs))
	for tier, acc := range tierAccs {
		agg := TierAggregate{
			TotalRuns:      acc.ailangRuns + acc.pythonRuns,
			AILANGRuns:     acc.ailangRuns,
			PythonRuns:     acc.pythonRuns,
			BenchmarkCount: len(acc.benchIDs),
		}
		if acc.ailangRuns > 0 {
			agg.AILANGSuccessRate = float64(acc.ailangPass) / float64(acc.ailangRuns)
		}
		if acc.pythonRuns > 0 {
			agg.PythonSuccessRate = float64(acc.pythonPass) / float64(acc.pythonRuns)
		}
		if ms := modelStats[tier]; ms != nil {
			agg.ModelStats = ms
		}
		if ex := extras[tier]; ex != nil {
			agg.AILANGRepairDelta = ex.AILANGRepairDelta
			agg.PythonRepairDelta = ex.PythonRepairDelta
			agg.AILANGAvgCostUSD = ex.AILANGAvgCost
			agg.PythonAvgCostUSD = ex.PythonAvgCost
			agg.APIErrorCount = ex.APIErrorCount
			agg.AILANGAPIError = ex.AILANGAPIError
			agg.PythonAPIError = ex.PythonAPIError
			agg.RefusalCount = ex.RefusalCount
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
