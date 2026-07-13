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

	// Per-model, per-language pass counts per benchmark — powers the gallery detail
	// view's "which models pass this benchmark" strip. bench -> model -> lang -> counts.
	type benchModelStat struct{ runs, passes int }
	modelBenchStats := make(map[string]map[string]map[string]*benchModelStat)

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

		// Per-model per-language pass counts for this benchmark.
		if r.Model != "" {
			if modelBenchStats[r.ID] == nil {
				modelBenchStats[r.ID] = make(map[string]map[string]*benchModelStat)
			}
			if modelBenchStats[r.ID][r.Model] == nil {
				modelBenchStats[r.ID][r.Model] = make(map[string]*benchModelStat)
			}
			ms := modelBenchStats[r.ID][r.Model][r.Lang]
			if ms == nil {
				ms = &benchModelStat{}
				modelBenchStats[r.ID][r.Model][r.Lang] = ms
			}
			ms.runs++
			if r.StdoutOk {
				ms.passes++
			}
		}
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
		if ShouldExcludeFromCapability(r.ErrorCategory) {
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
			if ShouldExcludeFromCapability(r.ErrorCategory) {
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
		// Skip benchmarks whose YAML spec no longer exists on disk. Historical
		// runs for renamed/removed benchmarks remain in summary.jsonl, but the
		// gallery should only show the *current* benchmark set — otherwise dead
		// IDs appear with no prompt and a frozen, misleading success rate.
		specPath := filepath.Join("benchmarks", id+".yml")
		spec, err := eval_harness.LoadSpec(specPath)
		if err != nil {
			continue
		}

		benchmark := map[string]interface{}{
			"totalRuns":   stats.TotalRuns,
			"successRate": stats.SuccessRate,
			"avgTokens":   stats.AvgTokens,
			"languages":   stats.Languages,
		}

		// Use TaskPrompt if available, otherwise fall back to Prompt
		if spec.TaskPrompt != "" {
			benchmark["taskPrompt"] = spec.TaskPrompt
		} else if spec.Prompt != "" {
			benchmark["taskPrompt"] = spec.Prompt
		}
		// Expected stdout — the other half of "the benchmark code" (what a correct
		// solution must print), shown alongside the prompt in the gallery detail.
		if spec.ExpectedOut != "" {
			benchmark["expectedStdout"] = spec.ExpectedOut
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

		// Per-model breakdown for this benchmark: {model: {lang: {passRate, runs}}}.
		// Powers the gallery detail view's per-model strip (which of the N models
		// pass THIS task, by language).
		if perModel, ok := modelBenchStats[id]; ok && len(perModel) > 0 {
			modelStatsJS := make(map[string]interface{}, len(perModel))
			for model, langs := range perModel {
				langJS := make(map[string]interface{}, len(langs))
				for lang, ms := range langs {
					if ms.runs == 0 {
						continue
					}
					langJS[lang] = map[string]interface{}{
						"passRate": float64(ms.passes) / float64(ms.runs),
						"runs":     ms.runs,
					}
				}
				if len(langJS) > 0 {
					modelStatsJS[model] = langJS
				}
			}
			if len(modelStatsJS) > 0 {
				benchmark["modelStats"] = modelStatsJS
			}
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

	// Build per-model JS objects, agent-only model entries, and the pooled
	// sweet-spot report. Extracted to export_json_models.go to keep this
	// function under the 800-line soft limit.
	mres := buildModelsJS(matrix, results, agentResults, reliability)
	modelsJS := mres.ModelsJS
	agentModelsJS := mres.AgentModelsJS
	sweetSpotReport := mres.SweetSpotReport

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

	// Calculate per-language standard, agent-comparable, and per-executor stats.
	// Extracted to export_json_lang.go (buildLangStandardStats / buildLangAgentStats
	// / buildLanguagesMap) to keep this file under the 800-line soft limit.
	langStandardStats, langAgentComparableStats, langPerExecComparableStats :=
		buildLangStandardStats(standardResults, agentBenchmarkIDs, agentModelsSet,
			perExecBenchmarkIDs, perExecModelsSet)

	langAgentStats := buildLangAgentStats(agentResults)

	languagesMap := buildLanguagesMap(matrix, langStandardStats, langAgentComparableStats,
		langPerExecComparableStats, langAgentStats, perExecutorLangStats)

	aggregatesJS["agentExecutors"] = executorList

	// Tier/tag aggregates + suite-change events (M-EVAL-SUITE-PREP M6 +
	// M-DASH-V2). Extracted to sibling file because the main exporter is
	// already past the 800-line soft limit.
	tiersJS := buildTierAggregates(resultsForTiers, benchmarkTier)
	tagsJS := buildTagAggregates(resultsForTiers)
	harnessesJS := buildHarnessAggregates(agentResults, benchmarkTier)
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
	// M-EVAL-DASHBOARD-REDESIGN: per-mode ELO ratings + grading provenance.
	dashboard.Ratings = buildRatingsBlock(standardResults, agentResults)
	dashboard.Grading = map[string]interface{}{
		"regraded": true,
		"method":   "M-EVAL-OUTPUT-NORMALIZE (boolean-case + numeric parity)",
	}
	dashboard.Languages = languagesMap
	dashboard.Executors = executorsJS
	dashboard.Harnesses = harnessesJS
	dashboard.Events = suiteEvents

	// M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION (v0.19.0): top-level sweet-spot
	// block. Carries per-benchmark cheapest/fastest pass champions plus the
	// slow threshold used. Per-model sweet_spot lives in dashboard.Models.
	champions := make([]map[string]interface{}, 0, len(sweetSpotReport.Champions))
	for _, c := range sweetSpotReport.Champions {
		champions = append(champions, map[string]interface{}{
			"benchmark_id":      c.BenchmarkID,
			"cheapest_model":    c.CheapestModel,
			"cheapest_cost_usd": c.CheapestCost,
			"cheapest_tts_ms":   c.CheapestTTSMs,
			"fastest_model":     c.FastestModel,
			"fastest_tts_ms":    c.FastestTTSMs,
			"fastest_cost_usd":  c.FastestCost,
		})
	}
	dashboard.SweetSpotGlobal = map[string]interface{}{
		"champions":         champions,
		"slow_threshold_ms": sweetSpotReport.SlowMs,
		"total_runs":        sweetSpotReport.TotalRuns,
	}

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
