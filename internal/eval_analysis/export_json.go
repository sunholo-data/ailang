package eval_analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// ExportBenchmarkJSON exports benchmark data as JSON for client-side rendering
func ExportBenchmarkJSON(matrix *PerformanceMatrix, history []*Baseline, results []*BenchmarkResult, outputPath string) (string, error) {
	// Load existing dashboard to preserve history
	dashboard, err := loadExistingDashboard(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to load existing dashboard: %w", err)
	}
	// Separate standard vs agent results for different metrics
	var standardResults, agentResults []*BenchmarkResult
	for _, r := range results {
		if r.EvalMode == "agent" {
			agentResults = append(agentResults, r)
		} else {
			// Default to standard if eval_mode not set (legacy results)
			standardResults = append(standardResults, r)
		}
	}
	// Agent-only baselines (e.g. harness comparison runs) have no standard results.
	// Fall back to all results so tier/tag aggregates are still populated.
	resultsForTiers := standardResults
	if len(resultsForTiers) == 0 {
		resultsForTiers = results
	}

	// Calculate agent-specific metrics
	agentSuccessCount := 0
	totalAgentTurns := 0
	agentTotalTokens := 0
	totalAgentCost := 0.0
	for _, r := range agentResults {
		if r.StdoutOk {
			agentSuccessCount++
		}
		totalAgentTurns += r.AgentTurns
		agentTotalTokens += r.TotalTokens
		totalAgentCost += r.CostUSD
	}

	avgAgentTurns := 0.0
	if len(agentResults) > 0 {
		avgAgentTurns = float64(totalAgentTurns) / float64(len(agentResults))
	}

	agentSuccessRate := 0.0
	if len(agentResults) > 0 {
		agentSuccessRate = float64(agentSuccessCount) / float64(len(agentResults))
	}

	avgAgentCost := 0.0
	if len(agentResults) > 0 {
		avgAgentCost = totalAgentCost / float64(len(agentResults))
	}

	// Build list of agent benchmark IDs (sorted for consistent output)
	agentBenchmarkIDs := make(map[string]bool)
	// Per-executor benchmark IDs — used downstream to compute fair per-executor
	// success-gap / cost-ratio comparisons. The blended `agentBenchmarkIDs` is
	// the union; per-executor sets capture exactly which benchmarks each harness
	// actually ran, so deltas vs zero-shot use matching denominators when
	// harness coverage diverges (e.g., one harness crashes mid-run).
	perExecBenchmarkIDs := make(map[string]map[string]bool) // executor -> benchmarkID -> true
	// Set of models that ran in agent mode. The cross-method comparison
	// (standard vs agent) is only fair when both sides use the same model
	// pool — otherwise standard's flagship pool (e.g., Opus, GPT-5) gets
	// compared to agent's cheap pool (e.g., Sonnet, Haiku, Flash) and the
	// agent column looks artificially worse.
	agentModelsSet := make(map[string]bool)
	perExecModelsSet := make(map[string]map[string]bool) // executor -> model -> true
	for _, r := range agentResults {
		agentBenchmarkIDs[r.ID] = true
		if r.Model != "" {
			agentModelsSet[r.Model] = true
		}
		if r.Executor != "" {
			if perExecBenchmarkIDs[r.Executor] == nil {
				perExecBenchmarkIDs[r.Executor] = make(map[string]bool)
			}
			perExecBenchmarkIDs[r.Executor][r.ID] = true
			if r.Model != "" {
				if perExecModelsSet[r.Executor] == nil {
					perExecModelsSet[r.Executor] = make(map[string]bool)
				}
				perExecModelsSet[r.Executor][r.Model] = true
			}
		}
	}
	agentBenchmarkList := make([]string, 0, len(agentBenchmarkIDs))
	for id := range agentBenchmarkIDs {
		agentBenchmarkList = append(agentBenchmarkList, id)
	}
	sort.Strings(agentBenchmarkList)

	// Sorted list of agent models — exposed so the UI can label the cross-method
	// comparison as "matched models only" and surface which model pool the
	// 0-shot baseline used.
	agentModelsList := make([]string, 0, len(agentModelsSet))
	for m := range agentModelsSet {
		agentModelsList = append(agentModelsList, m)
	}
	sort.Strings(agentModelsList)

	// Compute reliability counters (api-error + refusal) across standard
	// results so the dashboard can render an API Reliability card and
	// distinguish infra-failure 0% dips from real code-quality 0%s
	// (M-DASH-V2).
	reliability := computeReliability(standardResults)

	// Convert aggregates to camelCase for JavaScript
	aggregatesJS := map[string]interface{}{
		"zeroShotSuccess":   matrix.Aggregates.ZeroShotSuccess,
		"finalSuccess":      matrix.Aggregates.FinalSuccess,
		"repairUsed":        matrix.Aggregates.RepairUsed,
		"repairSuccessRate": matrix.Aggregates.RepairSuccessRate,
		"totalTokens":       matrix.Aggregates.TotalTokens,
		"totalCostUSD":      matrix.Aggregates.TotalCostUSD,
		"avgDurationMs":     matrix.Aggregates.AvgDurationMs,
		// Agent metrics (M-EVAL-AGENT)
		"agentRuns":        len(agentResults),
		"agentSuccessRate": agentSuccessRate,
		"avgAgentTurns":    avgAgentTurns,
		"agentTotalTokens": agentTotalTokens,
		"avgAgentCost":     avgAgentCost,
		"agentBenchmarks":  agentBenchmarkList, // Sorted list of benchmark IDs for fair comparison
		"agentModels":      agentModelsList,    // Models used in agent mode — also the 0-shot baseline pool
		// API reliability + refusal (M-DASH-V2). Keys are camelCase.
		"apiErrorCount": reliability.APIErrorCount,
		"apiErrorRate":  reliability.APIErrorRate,
		"refusalCount":  reliability.RefusalCount,
		"refusalRate":   reliability.RefusalRate,
	}

	// Group results by benchmark ID and language for code samples and stats
	codeSamples := make(map[string]map[string]string)       // benchmarkID -> language -> code
	langStats := make(map[string]map[string]*LanguageStats) // benchmarkID -> language -> stats

	for _, r := range results {
		// Collect code samples
		if r.Code != "" {
			if codeSamples[r.ID] == nil {
				codeSamples[r.ID] = make(map[string]string)
			}
			// Only keep one sample per language (preferably successful ones)
			if existing, exists := codeSamples[r.ID][r.Lang]; !exists || (r.RuntimeOk && !strings.Contains(existing, "def ")) {
				codeSamples[r.ID][r.Lang] = r.Code
			}
		}

		// Collect language-specific stats for each benchmark
		if langStats[r.ID] == nil {
			langStats[r.ID] = make(map[string]*LanguageStats)
		}
		if langStats[r.ID][r.Lang] == nil {
			langStats[r.ID][r.Lang] = &LanguageStats{}
		}
		stats := langStats[r.ID][r.Lang]
		stats.TotalRuns++
		if r.StdoutOk {
			stats.SuccessRate = float64(int(stats.SuccessRate*float64(stats.TotalRuns-1))+1) / float64(stats.TotalRuns)
		} else {
			stats.SuccessRate = float64(int(stats.SuccessRate*float64(stats.TotalRuns-1))) / float64(stats.TotalRuns)
		}
		// Use output tokens (not total)
		stats.AvgTokens = (stats.AvgTokens*float64(stats.TotalRuns-1) + float64(r.OutputTokens)) / float64(stats.TotalRuns)
	}

	// Collect agent-specific stats per benchmark+language, including api_errors
	// and per-harness (executor) buckets so the gallery can show adjusted pass
	// rates and per-harness breakdowns. Using pointers because we mutate nested
	// maps; the prior anonymous-struct + reassign pattern doesn't compose well
	// once we have per-harness sub-maps.
	type agentBenchHarnessStat struct {
		runs      int
		success   int
		apiErrors int
	}
	type agentBenchStat struct {
		runs       int
		success    int
		apiErrors  int
		turns      int
		tokens     int
		perHarness map[string]*agentBenchHarnessStat // executor -> per-harness stats
	}
	agentBenchStats := make(map[string]map[string]*agentBenchStat) // benchmarkID -> lang -> stats

	for _, r := range agentResults {
		if agentBenchStats[r.ID] == nil {
			agentBenchStats[r.ID] = make(map[string]*agentBenchStat)
		}
		stats := agentBenchStats[r.ID][r.Lang]
		if stats == nil {
			stats = &agentBenchStat{perHarness: make(map[string]*agentBenchHarnessStat)}
			agentBenchStats[r.ID][r.Lang] = stats
		}
		stats.runs++
		if r.StdoutOk {
			stats.success++
		}
		if r.ErrorCategory == "api_error" {
			stats.apiErrors++
		}
		stats.turns += r.AgentTurns
		stats.tokens += r.TotalTokens

		// Per-harness bucket. Skip if executor is empty (legacy or non-agent rows).
		if r.Executor != "" {
			hs := stats.perHarness[r.Executor]
			if hs == nil {
				hs = &agentBenchHarnessStat{}
				stats.perHarness[r.Executor] = hs
			}
			hs.runs++
			if r.StdoutOk {
				hs.success++
			}
			if r.ErrorCategory == "api_error" {
				hs.apiErrors++
			}
		}
	}

	// Convert benchmarks to camelCase for JavaScript. While iterating,
	// capture each benchmark's tier (from the YAML) into an index so we
	// can compute per-tier aggregates below without reloading the specs
	// (M-EVAL-SUITE-PREP M6). Tags are emitted on each benchmark but do
	// not need a separate index here.
	benchmarkTier := make(map[string]string) // benchmarkID -> tier
	benchmarksJS := make(map[string]interface{})
	for id, stats := range matrix.Benchmarks {
		benchmark := map[string]interface{}{
			"totalRuns":   stats.TotalRuns,
			"successRate": stats.SuccessRate,
			"avgTokens":   stats.AvgTokens,
			"languages":   stats.Languages,
		}

		// Load task prompt from benchmark YAML file
		specPath := filepath.Join("benchmarks", id+".yml")
		if _, err := os.Stat(specPath); err == nil {
			if spec, err := eval_harness.LoadSpec(specPath); err == nil {
				// Use TaskPrompt if available, otherwise fall back to Prompt
				if spec.TaskPrompt != "" {
					benchmark["taskPrompt"] = spec.TaskPrompt
				} else if spec.Prompt != "" {
					benchmark["taskPrompt"] = spec.Prompt
				}
				// Expose tier + tags so the dashboard can filter per-tier
				// and render tag chips. Tier defaults to "core" in LoadSpec.
				if spec.Tier != "" {
					benchmark["tier"] = spec.Tier
					benchmarkTier[id] = spec.Tier
				}
				if len(spec.Tags) > 0 {
					benchmark["tags"] = spec.Tags
				}
			}
		}

		// Add code samples if available
		if samples, ok := codeSamples[id]; ok {
			benchmark["codeSamples"] = samples
		}
		// Add per-language stats if available
		if perLangStats, ok := langStats[id]; ok {
			langStatsJS := make(map[string]interface{})
			for lang, lstats := range perLangStats {
				langStatsJS[lang] = map[string]interface{}{
					"successRate": lstats.SuccessRate,
					"avgTokens":   lstats.AvgTokens,
					"totalRuns":   lstats.TotalRuns,
				}
			}
			benchmark["languageStats"] = langStatsJS
		}

		// Add agent-specific stats per language. Includes api_error counts
		// and per-harness breakdown (claude/codex/gemini/opencode) so the
		// gallery can show adjusted pass rates and per-harness reliability.
		if agentStats, ok := agentBenchStats[id]; ok {
			agentLangStats := make(map[string]interface{})
			for lang, astats := range agentStats {
				if astats.runs == 0 {
					continue
				}
				byHarness := make(map[string]interface{})
				for hname, hs := range astats.perHarness {
					if hs.runs == 0 {
						continue
					}
					byHarness[hname] = map[string]interface{}{
						"runs":         hs.runs,
						"successRate":  float64(hs.success) / float64(hs.runs),
						"apiErrors":    hs.apiErrors,
						"apiErrorRate": float64(hs.apiErrors) / float64(hs.runs),
					}
				}
				agentLangStats[lang] = map[string]interface{}{
					"runs":         astats.runs,
					"successRate":  float64(astats.success) / float64(astats.runs),
					"avgTurns":     float64(astats.turns) / float64(astats.runs),
					"avgTokens":    float64(astats.tokens) / float64(astats.runs),
					"apiErrors":    astats.apiErrors,
					"apiErrorRate": float64(astats.apiErrors) / float64(astats.runs),
					"byHarness":    byHarness,
				}
			}
			if len(agentLangStats) > 0 {
				benchmark["agentStats"] = agentLangStats
			}
		}

		benchmarksJS[id] = benchmark
	}

	// Per-model, per-language agent stats (for JS/Go in BenchmarkExplorer).
	// Tracks api_error count so the explorer can surface adjusted pass rates
	// at the per-(model × language) granularity its mini-bars use.
	type agentLangStat struct{ runs, success, apiErrors int }
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
		if r.ErrorCategory == "api_error" {
			ls.apiErrors++
		}
		modelAgentLangStats[r.Model][r.Lang] = ls
	}

	// Calculate per-model agent statistics
	modelAgentStats := make(map[string]struct {
		runs        int
		success     int
		apiErrors   int
		totalTurns  int
		totalTokens int
		totalCost   float64
	})
	for _, r := range agentResults {
		stats := modelAgentStats[r.Model]
		stats.runs++
		if r.StdoutOk {
			stats.success++
		}
		if r.ErrorCategory == "api_error" {
			stats.apiErrors++
		}
		stats.totalTurns += r.AgentTurns
		stats.totalTokens += r.TotalTokens
		stats.totalCost += r.CostUSD
		modelAgentStats[r.Model] = stats
	}

	// Convert models to camelCase for JavaScript (nested aggregates)
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
					// timeout_scale relative to the default 30s cloud TTFT budget
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
		// Add baseline version if available
		if stats.BaselineVersion != "" {
			modelData["baselineVersion"] = stats.BaselineVersion
		}
		// Add per-language breakdown for this model (standard evals)
		langBreakdown := make(map[string]interface{})
		for lang, lstats := range stats.Languages {
			entry := map[string]interface{}{
				"successRate": lstats.SuccessRate,
				"avgTokens":   lstats.AvgTokens,
				"totalRuns":   lstats.TotalRuns,
			}
			// Attach agent api_error info if this model also ran in agent mode for this lang.
			// Lets the explorer's per-model mini-bars show adjusted rates.
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
		// Augment with agent-only languages (JS/Go from lang_harness_suite)
		if agentLangs, ok := modelAgentLangStats[name]; ok {
			for lang, als := range agentLangs {
				if _, exists := langBreakdown[lang]; !exists && als.runs > 0 {
					entry := map[string]interface{}{
						"successRate":       float64(als.success) / float64(als.runs),
						"avgTokens":         0,
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
		// Add per-benchmark breakdown for this model
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
		// Add agent-specific stats for this model
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
		// M-DASH-V2: attach per-model reliability counters so the dashboard
		// can tooltip the API Reliability card ("gemini-3-1-pro: 13/33").
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

	// Agent-only models (no standard eval — e.g. lang_harness_suite): add to modelsJS
	// so BenchmarkExplorer can display them alongside standard models.
	agentModelsJS := make(map[string]interface{})
	for modelName, agentStats := range modelAgentStats {
		if _, exists := modelsJS[modelName]; !exists && agentStats.runs > 0 {
			entry := map[string]interface{}{
				"totalRuns": agentStats.runs,
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
			// Per-language agent success rates (JS/Go from lang_harness_suite).
			// Includes adjusted rate + api_error counts so the Explorer's mini-bars
			// and per-model heatmap row can flip raw→adjusted just like models that
			// also have standard runs.
			if agentLangs, ok := modelAgentLangStats[modelName]; ok && len(agentLangs) > 0 {
				langBreakdown := make(map[string]interface{})
				for lang, als := range agentLangs {
					if als.runs > 0 {
						rawRate := float64(als.success) / float64(als.runs)
						langEntry := map[string]interface{}{
							"successRate":       rawRate,
							"avgTokens":         0,
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
			// Attach models.yml metadata (agent_cli, model_family, provider_type)
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
			// Add to both modelsJS (for BenchmarkExplorer) and agentModelsJS (legacy)
			modelsJS[modelName] = entry
			agentModelsJS[modelName] = entry
		}
	}

	// Per-executor agent aggregates (claude, gemini, etc.) — extracted to
	// export_json_executors.go to keep this file under the 800-line soft
	// limit. perExecutorLangStats is consumed by the language loop below to
	// attach agent_*_<executor> fields per language.
	executorsJS, perExecutorLangStats, executorList := buildExecutorAggregates(agentResults)

	// Process historical baselines to build complete history with per-model stats
	// Load results for each baseline and generate matrix to get modelStats
	for _, baseline := range history {
		// Load results for this baseline if not already loaded
		if len(baseline.Results) == 0 {
			// Try to find the baseline directory (handle versions with/without "v" prefix)
			var loadedResults []*BenchmarkResult
			var err error

			// Try version as-is first
			baselineDir := fmt.Sprintf("eval_results/baselines/%s", baseline.Version)
			loadedResults, err = LoadResults(baselineDir)

			// If failed and version doesn't start with "v", try adding "v" prefix
			if err != nil && !strings.HasPrefix(baseline.Version, "v") {
				baselineDir = fmt.Sprintf("eval_results/baselines/v%s", baseline.Version)
				loadedResults, err = LoadResults(baselineDir)
			}

			// If failed and version starts with "v", try removing "v" prefix
			if err != nil && strings.HasPrefix(baseline.Version, "v") {
				baselineDir = fmt.Sprintf("eval_results/baselines/%s", strings.TrimPrefix(baseline.Version, "v"))
				loadedResults, err = LoadResults(baselineDir)
			}

			if err == nil {
				baseline.Results = loadedResults
			}
		}

		// If we have results, generate a matrix and use buildHistoryEntryFromMatrix
		// to get complete stats including per-model breakdown
		if len(baseline.Results) > 0 {
			// Generate matrix from baseline results
			baselineMatrix, err := GenerateMatrix(baseline.Results, baseline.Version)
			if err == nil {
				// Build history entry with modelStats
				histEntry := buildHistoryEntryFromMatrix(baselineMatrix, baseline.Results)
				// Preserve the original baseline timestamp (don't use current time)
				histEntry.Timestamp = baseline.Timestamp.Format(time.RFC3339)
				// M-DASH-V2: attach per-tier snapshots using the CURRENT
				// tier mapping (documented approximation for pre-v0.14.0
				// baselines) so PerModelTrend can filter the time series
				// by tier retroactively.
				if tp := buildHistoricalTierPoints(baseline.Results, benchmarkTier); len(tp) > 0 {
					histEntry.Tiers = tp
				}
				// Merge this entry into the dashboard history
				mergeHistory(dashboard, histEntry)
			}
		}
	}

	// Build history entry for current version from matrix (only standard results for historical comparison)
	newHistoryEntry := buildHistoryEntryFromMatrix(matrix, standardResults)
	if tp := buildHistoricalTierPoints(standardResults, benchmarkTier); len(tp) > 0 {
		newHistoryEntry.Tiers = tp
	}

	// Merge with existing history (preserves old entries, updates if version exists)
	mergeHistory(dashboard, newHistoryEntry)

	// Calculate per-language zero-shot, final, and cost metrics from standard results
	// (agentBenchmarkIDs already built earlier for aggregatesJS)
	langStandardStats := make(map[string]struct {
		// Zero-shot only (firstAttemptOk)
		zeroShotRuns    int
		zeroShotSuccess int
		zeroShotTokens  int
		zeroShotCost    float64
		zeroShotApiErr  int // api_error count — excluded from "adjusted" rate
		// Final (including repairs)
		finalRuns    int
		finalSuccess int
		finalTokens  int
		finalCost    float64
		finalApiErr  int
		// Repair-specific
		repairAttempts int
		repairSuccess  int
	})

	// Also track stats for agent-comparable benchmarks only (fair comparison)
	langAgentComparableStats := make(map[string]struct {
		zeroShotRuns    int
		zeroShotSuccess int
		zeroShotTokens  int
		zeroShotCost    float64
		zeroShotApiErr  int
		finalRuns       int
		finalSuccess    int
		finalTokens     int
		finalCost       float64
		finalApiErr     int
	})

	// Per-executor comparable stats: zero-shot/repair metrics filtered to the
	// exact benchmarks each executor ran. Lets the dashboard compute fair
	// per-harness deltas (success gap, cost ratio) without mixing denominators.
	type perExecComparable struct {
		zeroShotRuns    int
		zeroShotSuccess int
		zeroShotTokens  int
		zeroShotCost    float64
		zeroShotApiErr  int
		finalRuns       int
		finalSuccess    int
		finalCost       float64
		finalApiErr     int
	}
	langPerExecComparableStats := make(map[string]map[string]*perExecComparable) // exec -> lang -> stats

	for _, r := range standardResults {
		stats := langStandardStats[r.Lang]
		isApiErr := r.ErrorCategory == "api_error"

		// Zero-shot metrics: Count ALL runs, but look at first attempt only
		// Success rate: based on first_attempt_ok across all runs
		// Tokens/cost: only from non-repair runs (repair inflates these metrics)
		stats.zeroShotRuns++
		if r.FirstAttemptOk {
			stats.zeroShotSuccess++
		}
		if isApiErr {
			stats.zeroShotApiErr++
		}
		if !r.RepairUsed {
			// Only count tokens/cost from non-repair runs for accurate per-attempt metrics
			if r.OutputTokens > 0 {
				stats.zeroShotTokens += r.OutputTokens
			}
			stats.zeroShotCost += r.CostUSD
		}

		// Final metrics (including repair attempts) - use ALL runs
		stats.finalRuns++
		if r.StdoutOk {
			stats.finalSuccess++
		}
		if isApiErr {
			stats.finalApiErr++
		}
		stats.finalTokens += r.OutputTokens
		stats.finalCost += r.CostUSD

		// Repair metrics
		if r.RepairUsed {
			stats.repairAttempts++
			if r.StdoutOk {
				stats.repairSuccess++
			}
		}

		langStandardStats[r.Lang] = stats

		// Track agent-comparable metrics: same benchmarks AND same models as
		// agent ran. Without the model filter, standard's flagship pool gets
		// compared against agent's cheap pool — see comment on agentModelsSet.
		if agentBenchmarkIDs[r.ID] && agentModelsSet[r.Model] {
			comparableStats := langAgentComparableStats[r.Lang]

			// Zero-shot: Count ALL runs, look at first attempt only
			comparableStats.zeroShotRuns++
			if r.FirstAttemptOk {
				comparableStats.zeroShotSuccess++
			}
			if isApiErr {
				comparableStats.zeroShotApiErr++
			}
			if !r.RepairUsed {
				// Only count tokens/cost from non-repair runs for accurate per-attempt metrics
				if r.OutputTokens > 0 {
					comparableStats.zeroShotTokens += r.OutputTokens
				}
				comparableStats.zeroShotCost += r.CostUSD
			}

			// Final: all runs
			comparableStats.finalRuns++
			if r.StdoutOk {
				comparableStats.finalSuccess++
			}
			if isApiErr {
				comparableStats.finalApiErr++
			}
			comparableStats.finalTokens += r.OutputTokens
			comparableStats.finalCost += r.CostUSD
			langAgentComparableStats[r.Lang] = comparableStats
		}

		// Per-executor comparable: count this standard run for every executor
		// where (a) the benchmark was run by that executor AND (b) the model
		// was used by that executor. The model filter ensures we don't
		// compare e.g. Opus 0-shot against Sonnet agent runs for "claude".
		for executor, benchSet := range perExecBenchmarkIDs {
			if !benchSet[r.ID] {
				continue
			}
			if execModels := perExecModelsSet[executor]; execModels != nil && !execModels[r.Model] {
				continue
			}
			if langPerExecComparableStats[executor] == nil {
				langPerExecComparableStats[executor] = make(map[string]*perExecComparable)
			}
			pec := langPerExecComparableStats[executor][r.Lang]
			if pec == nil {
				pec = &perExecComparable{}
				langPerExecComparableStats[executor][r.Lang] = pec
			}
			pec.zeroShotRuns++
			if r.FirstAttemptOk {
				pec.zeroShotSuccess++
			}
			if isApiErr {
				pec.zeroShotApiErr++
			}
			if !r.RepairUsed {
				if r.OutputTokens > 0 {
					pec.zeroShotTokens += r.OutputTokens
				}
				pec.zeroShotCost += r.CostUSD
			}
			pec.finalRuns++
			if r.StdoutOk {
				pec.finalSuccess++
			}
			if isApiErr {
				pec.finalApiErr++
			}
			pec.finalCost += r.CostUSD
		}
	}

	// Calculate per-language agent statistics with turn efficiency breakdown
	langAgentStats := make(map[string]struct {
		runs         int
		success      int
		apiErrors    int // for "adjusted" rate that excludes infra failures
		turns        int
		tokens       int
		cost         float64
		successTurns int // Total turns for successful runs
		successCount int // Count of successful runs (for avg)
		failureTurns int // Total turns for failed runs
		failureCount int // Count of failed runs (for avg)
	})
	for _, r := range agentResults {
		stats := langAgentStats[r.Lang]
		stats.runs++
		stats.turns += r.AgentTurns
		stats.tokens += r.TotalTokens
		stats.cost += r.CostUSD
		if r.ErrorCategory == "api_error" {
			stats.apiErrors++
		}

		if r.StdoutOk {
			stats.success++
			stats.successTurns += r.AgentTurns
			stats.successCount++
		} else {
			stats.failureTurns += r.AgentTurns
			stats.failureCount++
		}
		langAgentStats[r.Lang] = stats
	}

	// Build languages map for dashboard (matches existing format)
	languagesMap := make(map[string]interface{})
	for lang, stats := range matrix.Languages {
		langData := map[string]interface{}{
			"total_runs":   stats.TotalRuns,
			"success_rate": stats.SuccessRate,
			"avg_tokens":   stats.AvgTokens,
		}

		// Add zero-shot, final, and cost metrics from standard results
		if stdStats, ok := langStandardStats[lang]; ok && stdStats.zeroShotRuns > 0 {
			// Zero-shot only metrics
			langData["zero_shot_success"] = float64(stdStats.zeroShotSuccess) / float64(stdStats.zeroShotRuns)
			langData["zero_shot_avg_tokens"] = float64(stdStats.zeroShotTokens) / float64(stdStats.zeroShotRuns)
			langData["zero_shot_avg_cost"] = stdStats.zeroShotCost / float64(stdStats.zeroShotRuns)

			// Adjusted = passes / non-api-error runs. Surfaces "true" model
			// strength when infrastructure works (4% baseline of api_errors
			// in standard mode for v0.14.x).
			if zsNonApi := stdStats.zeroShotRuns - stdStats.zeroShotApiErr; zsNonApi > 0 {
				langData["zero_shot_success_adjusted"] = float64(stdStats.zeroShotSuccess) / float64(zsNonApi)
				langData["zero_shot_api_errors"] = stdStats.zeroShotApiErr
				langData["zero_shot_api_error_rate"] = float64(stdStats.zeroShotApiErr) / float64(stdStats.zeroShotRuns)
			}

			// Final metrics (including repairs)
			if stdStats.finalRuns > 0 {
				langData["final_success_avg_tokens"] = float64(stdStats.finalTokens) / float64(stdStats.finalRuns)
				langData["final_success_avg_cost"] = stdStats.finalCost / float64(stdStats.finalRuns)
				if finalNonApi := stdStats.finalRuns - stdStats.finalApiErr; finalNonApi > 0 {
					langData["final_success_adjusted"] = float64(stdStats.finalSuccess) / float64(finalNonApi)
				}
			}

			// Repair metrics
			if stdStats.repairAttempts > 0 {
				langData["repair_success_rate"] = float64(stdStats.repairSuccess) / float64(stdStats.repairAttempts)
			}

			// Overall average cost
			langData["avg_cost_usd"] = stdStats.finalCost / float64(stdStats.finalRuns)
		}

		// Add agent metrics if available for this language
		if agentStats, ok := langAgentStats[lang]; ok && agentStats.runs > 0 {
			langData["agent_runs"] = agentStats.runs
			agentSuccessRate := float64(agentStats.success) / float64(agentStats.runs)
			langData["agent_success_rate"] = agentSuccessRate
			langData["agent_avg_turns"] = float64(agentStats.turns) / float64(agentStats.runs)
			langData["agent_avg_tokens"] = float64(agentStats.tokens) / float64(agentStats.runs)
			agentAvgCost := agentStats.cost / float64(agentStats.runs)
			langData["agent_avg_cost"] = agentAvgCost

			// Adjusted agent rate — pass count / runs that did not hit api_error.
			// In v0.14.x ~45% of agent runs are api_errors (gemini quota, codex
			// CLI version, opencode infra) so this is the headline "model
			// strength when the harness works" number.
			langData["agent_api_errors"] = agentStats.apiErrors
			langData["agent_api_error_rate"] = float64(agentStats.apiErrors) / float64(agentStats.runs)
			if nonApi := agentStats.runs - agentStats.apiErrors; nonApi > 0 {
				langData["agent_success_rate_adjusted"] = float64(agentStats.success) / float64(nonApi)
			}

			// Agent turn efficiency breakdown
			if agentStats.successCount > 0 {
				langData["agent_avg_turns_success"] = float64(agentStats.successTurns) / float64(agentStats.successCount)
			}
			if agentStats.failureCount > 0 {
				langData["agent_avg_turns_failure"] = float64(agentStats.failureTurns) / float64(agentStats.failureCount)
			}

			// Add agent-comparable standard metrics (same benchmarks as agent) for fair comparison
			if comparableStats, ok := langAgentComparableStats[lang]; ok && comparableStats.zeroShotRuns > 0 {
				zeroShotSuccess := float64(comparableStats.zeroShotSuccess) / float64(comparableStats.zeroShotRuns)
				zeroShotAvgCost := comparableStats.zeroShotCost / float64(comparableStats.zeroShotRuns)

				langData["zero_shot_success_comparable"] = zeroShotSuccess
				langData["zero_shot_avg_tokens_comparable"] = float64(comparableStats.zeroShotTokens) / float64(comparableStats.zeroShotRuns)
				langData["zero_shot_avg_cost_comparable"] = zeroShotAvgCost
				if zsNonApi := comparableStats.zeroShotRuns - comparableStats.zeroShotApiErr; zsNonApi > 0 {
					langData["zero_shot_success_comparable_adjusted"] = float64(comparableStats.zeroShotSuccess) / float64(zsNonApi)
				}

				if comparableStats.finalRuns > 0 {
					finalSuccess := float64(comparableStats.finalSuccess) / float64(comparableStats.finalRuns)
					finalAvgCost := comparableStats.finalCost / float64(comparableStats.finalRuns)
					langData["final_success_comparable"] = finalSuccess
					langData["final_success_avg_tokens_comparable"] = float64(comparableStats.finalTokens) / float64(comparableStats.finalRuns)
					langData["final_success_avg_cost_comparable"] = finalAvgCost
					if finalNonApi := comparableStats.finalRuns - comparableStats.finalApiErr; finalNonApi > 0 {
						langData["final_success_comparable_adjusted"] = float64(comparableStats.finalSuccess) / float64(finalNonApi)
					}

					// Cost per success for repair approach
					if finalSuccess > 0 {
						langData["final_cost_per_success_comparable"] = finalAvgCost / finalSuccess
					}
				}

				// Derived metrics showing agent superiority on hard benchmarks
				// 1. Success rate gap (agent - 0-shot): higher = agent more valuable
				langData["agent_success_gap"] = agentSuccessRate - zeroShotSuccess

				// 2. Impossibility coverage: of 0-shot failures, what % did agent solve?
				if zeroShotSuccess < 1.0 {
					impossibilityCoverage := (agentSuccessRate - zeroShotSuccess) / (1.0 - zeroShotSuccess)
					langData["agent_impossibility_coverage"] = impossibilityCoverage
				}

				// 3. Cost efficiency ratio: agent cost/success vs 0-shot cost/success
				// Lower ratio = better (agent justifies its cost)
				if zeroShotSuccess > 0 && agentSuccessRate > 0 {
					zeroShotCostPerSuccess := zeroShotAvgCost / zeroShotSuccess
					agentCostPerSuccess := agentAvgCost / agentSuccessRate
					langData["agent_cost_per_success"] = agentCostPerSuccess
					langData["zero_shot_cost_per_success"] = zeroShotCostPerSuccess
					langData["agent_cost_efficiency_ratio"] = agentCostPerSuccess / zeroShotCostPerSuccess
				}
			}
		}

		// Add per-executor agent breakdown for this language
		for executor, langMap := range perExecutorLangStats {
			if ls, ok := langMap[lang]; ok && ls.Runs > 0 {
				langData["agent_success_rate_"+executor] = float64(ls.Success) / float64(ls.Runs)
				langData["agent_avg_turns_"+executor] = float64(ls.Turns) / float64(ls.Runs)
				langData["agent_avg_tokens_"+executor] = float64(ls.Tokens) / float64(ls.Runs)
				langData["agent_avg_cost_"+executor] = ls.Cost / float64(ls.Runs)
				langData["agent_runs_"+executor] = ls.Runs

				// Adjusted per-executor: passes / non-api-error runs.
				langData["agent_api_errors_"+executor] = ls.APIErrors
				langData["agent_api_error_rate_"+executor] = float64(ls.APIErrors) / float64(ls.Runs)
				if nonApi := ls.Runs - ls.APIErrors; nonApi > 0 {
					langData["agent_success_rate_adjusted_"+executor] = float64(ls.Success) / float64(nonApi)
				}

				// Per-executor comparable baseline + derived deltas. Filtered to
				// exactly the benchmarks this executor ran, so the success-gap
				// and cost-ratio comparisons hold even when harness coverage
				// drifts (one harness skips/crashes a benchmark another runs).
				if execLangMap, ok := langPerExecComparableStats[executor]; ok {
					if pec, ok := execLangMap[lang]; ok && pec.zeroShotRuns > 0 {
						zeroShotSuccessExec := float64(pec.zeroShotSuccess) / float64(pec.zeroShotRuns)
						zeroShotAvgCostExec := pec.zeroShotCost / float64(pec.zeroShotRuns)

						langData["zero_shot_success_comparable_"+executor] = zeroShotSuccessExec
						langData["zero_shot_avg_cost_comparable_"+executor] = zeroShotAvgCostExec

						// Adjusted = success / non-api-error runs. Surfaces true model
						// strength when infrastructure works.
						var zsAdjExec, agentAdjExec float64
						haveZsAdj, haveAgentAdj := false, false
						if zsNonApi := pec.zeroShotRuns - pec.zeroShotApiErr; zsNonApi > 0 {
							zsAdjExec = float64(pec.zeroShotSuccess) / float64(zsNonApi)
							langData["zero_shot_success_comparable_adjusted_"+executor] = zsAdjExec
							haveZsAdj = true
						}
						if agentNonApi := ls.Runs - ls.APIErrors; agentNonApi > 0 {
							agentAdjExec = float64(ls.Success) / float64(agentNonApi)
							haveAgentAdj = true
						}

						agentSuccessRateExec := float64(ls.Success) / float64(ls.Runs)
						agentAvgCostExec := ls.Cost / float64(ls.Runs)

						// Raw success gap (kept for backwards-compat).
						langData["agent_success_gap_"+executor] = agentSuccessRateExec - zeroShotSuccessExec

						// Adjusted success gap — apples-to-apples with the headline
						// "Success Rate" row which now defaults to adjusted.
						if haveZsAdj && haveAgentAdj {
							langData["agent_success_gap_adjusted_"+executor] = agentAdjExec - zsAdjExec
						}

						// Cost efficiency ratio (lower is better; agent vs zero-shot).
						if zeroShotSuccessExec > 0 && agentSuccessRateExec > 0 {
							zeroShotCPS := zeroShotAvgCostExec / zeroShotSuccessExec
							agentCPS := agentAvgCostExec / agentSuccessRateExec
							if zeroShotCPS > 0 {
								langData["agent_cost_efficiency_ratio_"+executor] = agentCPS / zeroShotCPS
							}
						}

						// Adjusted cost efficiency: keep total spend in numerator but
						// divide by adjusted success count (excluding api-error runs).
						// This says "cost per success when the harness actually runs".
						if haveZsAdj && haveAgentAdj && zsAdjExec > 0 && agentAdjExec > 0 {
							zsCPSAdj := zeroShotAvgCostExec / zsAdjExec
							agentCPSAdj := agentAvgCostExec / agentAdjExec
							if zsCPSAdj > 0 {
								langData["agent_cost_efficiency_ratio_adjusted_"+executor] = agentCPSAdj / zsCPSAdj
							}
						}
					}
				}
			}
		}

		languagesMap[lang] = langData
	}

	aggregatesJS["agentExecutors"] = executorList

	// Tier/tag aggregates + suite-change events (M-EVAL-SUITE-PREP M6 +
	// M-DASH-V2). Extracted to sibling file because the main exporter is
	// already past the 800-line soft limit.
	tiersJS := buildTierAggregates(resultsForTiers, benchmarkTier)
	tagsJS := buildTagAggregates(resultsForTiers)
	harnessesJS := buildHarnessAggregates(agentResults)
	suiteEvents, err := LoadSuiteEvents("benchmarks/events.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: load events.yml: %v\n", err)
	}

	// Update dashboard with current version data
	dashboard.Version = matrix.Version
	dashboard.Timestamp = time.Now().Format(time.RFC3339)
	dashboard.TotalRuns = matrix.TotalRuns
	dashboard.Aggregates = aggregatesJS
	dashboard.Tiers = tiersJS
	dashboard.Tags = tagsJS
	dashboard.Models = modelsJS
	dashboard.AgentModels = agentModelsJS
	dashboard.Benchmarks = benchmarksJS
	dashboard.Languages = languagesMap
	dashboard.Executors = executorsJS
	dashboard.Harnesses = harnessesJS
	dashboard.Events = suiteEvents

	// Write atomically
	if err := writeJSONAtomic(outputPath, dashboard); err != nil {
		return "", fmt.Errorf("failed to write dashboard: %w", err)
	}

	// Return JSON string for backwards compatibility (stdout redirection)
	jsonBytes, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonBytes), nil
}
