package eval_analysis

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// modelsResult bundles the outputs of buildModelsJS so ExportBenchmarkJSON
// can receive them as a single return value rather than a sprawling tuple.
type modelsResult struct {
	ModelsJS        map[string]interface{}
	AgentModelsJS   map[string]interface{}
	SweetSpotReport SweetSpotReport
}

// agentLangStat tracks per-language agent run counts for a single model.
// Used only within this file; the type lives here (not export_json.go) since
// the accumulation loop that creates it was extracted here.
type agentLangStat struct {
	runs, success, apiErrors int
	tokens                   int
	cost                     float64
}

// modelAgentTotals tracks per-model agent aggregates (all languages pooled).
type modelAgentTotals struct {
	runs        int
	success     int
	apiErrors   int
	totalTurns  int
	totalTokens int
	totalCost   float64
}

// buildModelsJS computes per-model and agent-only-model JS objects for the
// dashboard, along with the pooled SweetSpotReport (champions + per-model
// sweet-spot data).
//
// Extracted from ExportBenchmarkJSON (lines 333–721 of the original file) to
// keep that function under the 800-line soft limit.
func buildModelsJS(
	matrix *PerformanceMatrix,
	results []*BenchmarkResult,
	agentResults []*BenchmarkResult,
	reliability *ReliabilityCounts,
) modelsResult {
	// --- Per-model, per-language agent stats ---
	modelAgentLangStats := make(map[string]map[string]agentLangStat)
	for _, r := range agentResults {
		if modelAgentLangStats[r.Model] == nil {
			modelAgentLangStats[r.Model] = make(map[string]agentLangStat)
		}
		ls := modelAgentLangStats[r.Model][r.Lang]
		ls.runs++
		if r.StdoutOk {
			ls.success++
		}
		if ShouldExcludeFromCapability(r.ErrorCategory) {
			ls.apiErrors++
		}
		ls.tokens += r.OutputTokens
		ls.cost += r.CostUSD
		modelAgentLangStats[r.Model][r.Lang] = ls
	}

	// --- Per-model agent aggregates (all languages pooled) ---
	modelAgentStats := make(map[string]*modelAgentTotals)
	for _, r := range agentResults {
		s := modelAgentStats[r.Model]
		if s == nil {
			s = &modelAgentTotals{}
			modelAgentStats[r.Model] = s
		}
		s.runs++
		if r.StdoutOk {
			s.success++
		}
		if ShouldExcludeFromCapability(r.ErrorCategory) {
			s.apiErrors++
		}
		s.totalTurns += r.AgentTurns
		s.totalTokens += r.TotalTokens
		s.totalCost += r.CostUSD
	}

	// --- Per-model efficiency (standard + agent, all languages) ---
	resultsByModel := make(map[string][]*BenchmarkResult)
	for _, r := range results {
		resultsByModel[r.Model] = append(resultsByModel[r.Model], r)
	}
	modelEfficiency := make(map[string]EfficiencyAggregates)
	for modelName, modelResults := range resultsByModel {
		modelEfficiency[modelName] = ComputeEfficiency(modelResults)
	}

	// --- Sweet-spot report (pooled) ---
	sweetSpotReport := BuildSweetSpot(results, SweetSpotOpts{})
	sweetSpotByModel := make(map[string][]SweetSpotRow)
	for _, row := range sweetSpotReport.Rows {
		sweetSpotByModel[row.Model] = append(sweetSpotByModel[row.Model], row)
	}
	for m := range sweetSpotByModel {
		rs := sweetSpotByModel[m]
		sort.SliceStable(rs, func(i, j int) bool {
			return rs[i].TotalRuns > rs[j].TotalRuns
		})
		sweetSpotByModel[m] = rs
	}

	// --- Per-language sweet-spot (agent-only runs) ---
	resultsByLang := map[string][]*BenchmarkResult{}
	for _, r := range results {
		if r.Lang == "" || r.EvalMode != "agent" {
			continue
		}
		resultsByLang[r.Lang] = append(resultsByLang[r.Lang], r)
	}
	sweetSpotByModelByLang := map[string]map[string][]SweetSpotRow{}
	for lang, rs := range resultsByLang {
		langReport := BuildSweetSpot(rs, SweetSpotOpts{})
		for _, row := range langReport.Rows {
			if sweetSpotByModelByLang[row.Model] == nil {
				sweetSpotByModelByLang[row.Model] = map[string][]SweetSpotRow{}
			}
			sweetSpotByModelByLang[row.Model][lang] = append(sweetSpotByModelByLang[row.Model][lang], row)
		}
	}

	// --- Standard models loop ---
	modelsJS := make(map[string]interface{})
	for name, stats := range matrix.Models {
		modelData := map[string]interface{}{
			"totalRuns": stats.TotalRuns,
			"aggregates": map[string]interface{}{
				"zeroShotSuccess":   stats.Aggregates.ZeroShotSuccess,
				"finalSuccess":      stats.Aggregates.FinalSuccess,
				"repairUsed":        stats.Aggregates.RepairUsed,
				"repairSuccessRate": stats.Aggregates.RepairSuccessRate,
				"totalTokens":       stats.Aggregates.TotalTokens,
				"totalCostUSD":      stats.Aggregates.TotalCostUSD,
				"avgDurationMs":     stats.Aggregates.AvgDurationMs,
			},
		}
		// M-BENCHMARK-SECTION: provider_type and timeout_scale for cloud/local badge.
		if cfg := eval_harness.GlobalModelsConfig; cfg != nil {
			if mc, ok := cfg.Models[name]; ok {
				providerType := "cloud"
				if mc.Provider == "ollama" {
					providerType = "local"
				}
				modelData["provider_type"] = providerType
				if mc.TTFTTimeoutSeconds > 0 {
					const cloudTTFTDefault = 30
					modelData["timeout_scale"] = float64(mc.TTFTTimeoutSeconds) / cloudTTFTDefault
				}
				if mc.AgentCLI != nil && *mc.AgentCLI != "" {
					modelData["agent_cli"] = *mc.AgentCLI
				}
				if mc.ModelFamily != "" {
					modelData["model_family"] = mc.ModelFamily
				}
			}
		}
		if stats.BaselineVersion != "" {
			modelData["baselineVersion"] = stats.BaselineVersion
		}
		// Per-language breakdown (standard evals).
		langBreakdown := make(map[string]interface{})
		for lang, lstats := range stats.Languages {
			entry := map[string]interface{}{
				"successRate": lstats.SuccessRate,
				"avgTokens":   lstats.AvgTokens,
				"totalRuns":   lstats.TotalRuns,
			}
			if agentLangs, ok := modelAgentLangStats[name]; ok {
				if als, ok := agentLangs[lang]; ok && als.runs > 0 {
					entry["agentRuns"] = als.runs
					entry["agentApiErrors"] = als.apiErrors
					entry["agentApiErrorRate"] = float64(als.apiErrors) / float64(als.runs)
					if nonApi := als.runs - als.apiErrors; nonApi > 0 {
						entry["agentSuccessRate"] = float64(als.success) / float64(als.runs)
						entry["agentSuccessRateAdjusted"] = float64(als.success) / float64(nonApi)
					}
				}
			}
			langBreakdown[lang] = entry
		}
		// Augment with agent-only languages (JS/Go from lang_harness_suite).
		if agentLangs, ok := modelAgentLangStats[name]; ok {
			for lang, als := range agentLangs {
				if _, exists := langBreakdown[lang]; !exists && als.runs > 0 {
					entry := map[string]interface{}{
						"successRate":       float64(als.success) / float64(als.runs),
						"avgTokens":         float64(als.tokens) / float64(als.runs),
						"avgCost":           als.cost / float64(als.runs),
						"totalRuns":         als.runs,
						"agentOnly":         true,
						"agentApiErrors":    als.apiErrors,
						"agentApiErrorRate": float64(als.apiErrors) / float64(als.runs),
					}
					if nonApi := als.runs - als.apiErrors; nonApi > 0 {
						entry["agentSuccessRateAdjusted"] = float64(als.success) / float64(nonApi)
					}
					langBreakdown[lang] = entry
				}
			}
		}
		if len(langBreakdown) > 0 {
			modelData["languages"] = langBreakdown
		}
		// Per-benchmark breakdown.
		if len(stats.Benchmarks) > 0 {
			benchBreakdown := make(map[string]interface{})
			for benchID, run := range stats.Benchmarks {
				benchBreakdown[benchID] = map[string]interface{}{
					"success":        run.Success,
					"firstAttemptOk": run.FirstAttemptOk,
					"repairUsed":     run.RepairUsed,
					"tokens":         run.Tokens,
				}
			}
			modelData["benchmarks"] = benchBreakdown
		}
		// Agent-specific stats for this model.
		if agentStats, ok := modelAgentStats[name]; ok && agentStats.runs > 0 {
			modelData["agentStats"] = map[string]interface{}{
				"runs":         agentStats.runs,
				"successRate":  float64(agentStats.success) / float64(agentStats.runs),
				"apiErrors":    agentStats.apiErrors,
				"apiErrorRate": float64(agentStats.apiErrors) / float64(agentStats.runs),
				"avgTurns":     float64(agentStats.totalTurns) / float64(agentStats.runs),
				"avgTokens":    float64(agentStats.totalTokens) / float64(agentStats.runs),
				"avgCost":      agentStats.totalCost / float64(agentStats.runs),
			}
		}
		if eff, ok := modelEfficiency[name]; ok {
			modelData["efficiency"] = eff
		}
		if ssRows, ok := sweetSpotByModel[name]; ok && len(ssRows) > 0 {
			modelData["sweet_spot"] = renderSweetSpotRow(ssRows[0])
			if len(ssRows) > 1 {
				byHarness := make(map[string]interface{}, len(ssRows))
				for _, r := range ssRows {
					byHarness[r.Harness] = renderSweetSpotRow(r)
				}
				modelData["sweet_spot_by_harness"] = byHarness
			}
		}
		if langMap, ok := sweetSpotByModelByLang[name]; ok && len(langMap) > 0 {
			byLang := make(map[string]interface{}, len(langMap))
			for lang, rows := range langMap {
				if len(rows) == 0 {
					continue
				}
				byLang[lang] = renderSweetSpotRow(rows[0])
			}
			if len(byLang) > 0 {
				modelData["sweet_spot_by_lang"] = byLang
			}
		}
		// M-DASH-V2: per-model reliability counters.
		if rel, ok := reliability.PerModel[name]; ok {
			modelData["reliability"] = map[string]interface{}{
				"apiErrorCount":  rel.APIErrorCount,
				"apiErrorRate":   rel.APIErrorRate,
				"refusalCount":   rel.RefusalCount,
				"refusalRate":    rel.RefusalRate,
				"totalRuns":      rel.TotalRuns,
				"ailangApiError": rel.AILANGAPIError,
				"pythonApiError": rel.PythonAPIError,
			}
		}
		modelsJS[name] = modelData
	}

	// --- Agent-only models (no standard eval) ---
	agentModelsJS := make(map[string]interface{})
	for modelName, agentStats := range modelAgentStats {
		if _, exists := modelsJS[modelName]; !exists && agentStats.runs > 0 {
			entry := map[string]interface{}{
				"totalRuns": agentStats.runs,
				"aggregates": map[string]interface{}{
					"totalRuns":     agentStats.runs,
					"totalTokens":   agentStats.totalTokens,
					"totalCostUSD":  agentStats.totalCost,
					"finalSuccess":  float64(agentStats.success) / float64(agentStats.runs),
					"avgDurationMs": 0,
				},
				"agentStats": map[string]interface{}{
					"runs":         agentStats.runs,
					"successRate":  float64(agentStats.success) / float64(agentStats.runs),
					"apiErrors":    agentStats.apiErrors,
					"apiErrorRate": float64(agentStats.apiErrors) / float64(agentStats.runs),
					"avgTurns":     float64(agentStats.totalTurns) / float64(agentStats.runs),
					"avgTokens":    float64(agentStats.totalTokens) / float64(agentStats.runs),
					"avgCost":      agentStats.totalCost / float64(agentStats.runs),
				},
			}
			if eff, ok := modelEfficiency[modelName]; ok {
				entry["efficiency"] = eff
			}
			if ssRows, ok := sweetSpotByModel[modelName]; ok && len(ssRows) > 0 {
				entry["sweet_spot"] = renderSweetSpotRow(ssRows[0])
				if len(ssRows) > 1 {
					byHarness := make(map[string]interface{}, len(ssRows))
					for _, r := range ssRows {
						byHarness[r.Harness] = renderSweetSpotRow(r)
					}
					entry["sweet_spot_by_harness"] = byHarness
				}
			}
			if langMap, ok := sweetSpotByModelByLang[modelName]; ok && len(langMap) > 0 {
				byLang := make(map[string]interface{}, len(langMap))
				for lang, rows := range langMap {
					if len(rows) == 0 {
						continue
					}
					byLang[lang] = renderSweetSpotRow(rows[0])
				}
				if len(byLang) > 0 {
					entry["sweet_spot_by_lang"] = byLang
				}
			}
			// Per-language agent success rates.
			if agentLangs, ok := modelAgentLangStats[modelName]; ok && len(agentLangs) > 0 {
				langBreakdown := make(map[string]interface{})
				for lang, als := range agentLangs {
					if als.runs > 0 {
						rawRate := float64(als.success) / float64(als.runs)
						langEntry := map[string]interface{}{
							"successRate":       rawRate,
							"avgTokens":         float64(als.tokens) / float64(als.runs),
							"avgCost":           als.cost / float64(als.runs),
							"totalRuns":         als.runs,
							"agentOnly":         true,
							"agentRuns":         als.runs,
							"agentSuccessRate":  rawRate,
							"agentApiErrors":    als.apiErrors,
							"agentApiErrorRate": float64(als.apiErrors) / float64(als.runs),
						}
						if nonApi := als.runs - als.apiErrors; nonApi > 0 {
							langEntry["agentSuccessRateAdjusted"] = float64(als.success) / float64(nonApi)
						}
						langBreakdown[lang] = langEntry
					}
				}
				if len(langBreakdown) > 0 {
					entry["languages"] = langBreakdown
				}
			}
			// Attach models.yml metadata.
			if cfg := eval_harness.GlobalModelsConfig; cfg != nil {
				if mc, ok := cfg.Models[modelName]; ok {
					if mc.AgentCLI != nil && *mc.AgentCLI != "" {
						entry["agent_cli"] = *mc.AgentCLI
					}
					if mc.ModelFamily != "" {
						entry["model_family"] = mc.ModelFamily
					}
					providerType := "cloud"
					if mc.Provider == "ollama" {
						providerType = "local"
					}
					entry["provider_type"] = providerType
				}
			}
			// Agent-only models belong in agentModels ONLY. Adding them to the
			// standard `models` map put them on the Model Leaderboard with empty
			// standard data — the phantom-zero rows. Standard and agent must stay
			// split: `models` = ran standard, `agentModels` = agent.
			agentModelsJS[modelName] = entry
		}
	}

	return modelsResult{
		ModelsJS:        modelsJS,
		AgentModelsJS:   agentModelsJS,
		SweetSpotReport: sweetSpotReport,
	}
}
