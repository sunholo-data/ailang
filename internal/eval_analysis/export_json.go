package eval_analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/eval_harness"
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
	for _, r := range agentResults {
		agentBenchmarkIDs[r.ID] = true
	}
	agentBenchmarkList := make([]string, 0, len(agentBenchmarkIDs))
	for id := range agentBenchmarkIDs {
		agentBenchmarkList = append(agentBenchmarkList, id)
	}
	sort.Strings(agentBenchmarkList)

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

	// Collect agent-specific stats per benchmark+language
	agentBenchStats := make(map[string]map[string]struct {
		runs    int
		success int
		turns   int
		tokens  int
	}) // benchmarkID -> language -> stats

	for _, r := range agentResults {
		if agentBenchStats[r.ID] == nil {
			agentBenchStats[r.ID] = make(map[string]struct {
				runs    int
				success int
				turns   int
				tokens  int
			})
		}
		stats := agentBenchStats[r.ID][r.Lang]
		stats.runs++
		if r.StdoutOk {
			stats.success++
		}
		stats.turns += r.AgentTurns
		stats.tokens += r.TotalTokens
		agentBenchStats[r.ID][r.Lang] = stats
	}

	// Convert benchmarks to camelCase for JavaScript
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

		// Add agent-specific stats per language
		if agentStats, ok := agentBenchStats[id]; ok {
			agentLangStats := make(map[string]interface{})
			for lang, astats := range agentStats {
				if astats.runs > 0 {
					agentLangStats[lang] = map[string]interface{}{
						"runs":        astats.runs,
						"successRate": float64(astats.success) / float64(astats.runs),
						"avgTurns":    float64(astats.turns) / float64(astats.runs),
						"avgTokens":   float64(astats.tokens) / float64(astats.runs),
					}
				}
			}
			if len(agentLangStats) > 0 {
				benchmark["agentStats"] = agentLangStats
			}
		}

		benchmarksJS[id] = benchmark
	}

	// Calculate per-model agent statistics
	modelAgentStats := make(map[string]struct {
		runs        int
		success     int
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
		// Add baseline version if available
		if stats.BaselineVersion != "" {
			modelData["baselineVersion"] = stats.BaselineVersion
		}
		// Add per-language breakdown for this model
		if len(stats.Languages) > 0 {
			langBreakdown := make(map[string]interface{})
			for lang, lstats := range stats.Languages {
				langBreakdown[lang] = map[string]interface{}{
					"successRate": lstats.SuccessRate,
					"avgTokens":   lstats.AvgTokens,
					"totalRuns":   lstats.TotalRuns,
				}
			}
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
				"runs":        agentStats.runs,
				"successRate": float64(agentStats.success) / float64(agentStats.runs),
				"avgTurns":    float64(agentStats.totalTurns) / float64(agentStats.runs),
				"avgTokens":   float64(agentStats.totalTokens) / float64(agentStats.runs),
				"avgCost":     agentStats.totalCost / float64(agentStats.runs),
			}
		}
		modelsJS[name] = modelData
	}

	// Separate agent-only models (models that ran agents but not standard evals)
	agentModelsJS := make(map[string]interface{})
	for modelName, agentStats := range modelAgentStats {
		if _, exists := modelsJS[modelName]; !exists && agentStats.runs > 0 {
			agentModelsJS[modelName] = map[string]interface{}{
				"totalRuns": agentStats.runs,
				"agentStats": map[string]interface{}{
					"runs":        agentStats.runs,
					"successRate": float64(agentStats.success) / float64(agentStats.runs),
					"avgTurns":    float64(agentStats.totalTurns) / float64(agentStats.runs),
					"avgTokens":   float64(agentStats.totalTokens) / float64(agentStats.runs),
					"avgCost":     agentStats.totalCost / float64(agentStats.runs),
				},
			}
		}
	}

	// Calculate per-executor agent statistics (claude, gemini, etc.)
	type executorAgentStats struct {
		runs        int
		success     int
		totalTurns  int
		totalTokens int
		totalCost   float64
	}
	type executorLangAgentStats struct {
		runs         int
		success      int
		turns        int
		tokens       int
		cost         float64
		successTurns int
		successCount int
		failureTurns int
		failureCount int
	}
	perExecutorStats := make(map[string]*executorAgentStats)
	perExecutorLangStats := make(map[string]map[string]*executorLangAgentStats) // executor -> lang -> stats
	perExecutorModelStats := make(map[string]map[string]*executorAgentStats)    // executor -> model -> stats

	for _, r := range agentResults {
		executor := r.Executor
		if executor == "" {
			executor = "unknown"
		}

		// Per-executor totals
		if perExecutorStats[executor] == nil {
			perExecutorStats[executor] = &executorAgentStats{}
		}
		es := perExecutorStats[executor]
		es.runs++
		if r.StdoutOk {
			es.success++
		}
		es.totalTurns += r.AgentTurns
		es.totalTokens += r.TotalTokens
		es.totalCost += r.CostUSD

		// Per-executor per-model stats
		if perExecutorModelStats[executor] == nil {
			perExecutorModelStats[executor] = make(map[string]*executorAgentStats)
		}
		if perExecutorModelStats[executor][r.Model] == nil {
			perExecutorModelStats[executor][r.Model] = &executorAgentStats{}
		}
		ms := perExecutorModelStats[executor][r.Model]
		ms.runs++
		if r.StdoutOk {
			ms.success++
		}
		ms.totalTurns += r.AgentTurns
		ms.totalTokens += r.TotalTokens
		ms.totalCost += r.CostUSD

		// Per-executor per-language stats
		if perExecutorLangStats[executor] == nil {
			perExecutorLangStats[executor] = make(map[string]*executorLangAgentStats)
		}
		if perExecutorLangStats[executor][r.Lang] == nil {
			perExecutorLangStats[executor][r.Lang] = &executorLangAgentStats{}
		}
		ls := perExecutorLangStats[executor][r.Lang]
		ls.runs++
		ls.turns += r.AgentTurns
		ls.tokens += r.TotalTokens
		ls.cost += r.CostUSD
		if r.StdoutOk {
			ls.success++
			ls.successTurns += r.AgentTurns
			ls.successCount++
		} else {
			ls.failureTurns += r.AgentTurns
			ls.failureCount++
		}
	}

	// Build executors map for JSON output
	executorsJS := make(map[string]interface{})
	for executor, es := range perExecutorStats {
		if es.runs == 0 {
			continue
		}
		execData := map[string]interface{}{
			"runs":        es.runs,
			"successRate": float64(es.success) / float64(es.runs),
			"avgTurns":    float64(es.totalTurns) / float64(es.runs),
			"avgTokens":   float64(es.totalTokens) / float64(es.runs),
			"avgCost":     es.totalCost / float64(es.runs),
			"totalCost":   es.totalCost,
		}

		// Add per-language breakdown
		if langMap, ok := perExecutorLangStats[executor]; ok {
			langBreakdown := make(map[string]interface{})
			for lang, ls := range langMap {
				if ls.runs == 0 {
					continue
				}
				langEntry := map[string]interface{}{
					"runs":        ls.runs,
					"successRate": float64(ls.success) / float64(ls.runs),
					"avgTurns":    float64(ls.turns) / float64(ls.runs),
					"avgTokens":   float64(ls.tokens) / float64(ls.runs),
					"avgCost":     ls.cost / float64(ls.runs),
				}
				if ls.successCount > 0 {
					langEntry["avgTurnsSuccess"] = float64(ls.successTurns) / float64(ls.successCount)
				}
				if ls.failureCount > 0 {
					langEntry["avgTurnsFailure"] = float64(ls.failureTurns) / float64(ls.failureCount)
				}
				langBreakdown[lang] = langEntry
			}
			execData["languages"] = langBreakdown
		}
		// Add per-model breakdown within this executor
		if modelMap, ok := perExecutorModelStats[executor]; ok {
			modelBreakdown := make(map[string]interface{})
			for model, ms := range modelMap {
				if ms.runs == 0 {
					continue
				}
				modelBreakdown[model] = map[string]interface{}{
					"runs":        ms.runs,
					"successRate": float64(ms.success) / float64(ms.runs),
					"avgTurns":    float64(ms.totalTurns) / float64(ms.runs),
					"avgTokens":   float64(ms.totalTokens) / float64(ms.runs),
					"avgCost":     ms.totalCost / float64(ms.runs),
				}
			}
			execData["models"] = modelBreakdown
		}
		executorsJS[executor] = execData
	}

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
				// Merge this entry into the dashboard history
				mergeHistory(dashboard, histEntry)
			}
		}
	}

	// Build history entry for current version from matrix (only standard results for historical comparison)
	newHistoryEntry := buildHistoryEntryFromMatrix(matrix, standardResults)

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
		// Final (including repairs)
		finalRuns    int
		finalSuccess int
		finalTokens  int
		finalCost    float64
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
		finalRuns       int
		finalSuccess    int
		finalTokens     int
		finalCost       float64
	})

	for _, r := range standardResults {
		stats := langStandardStats[r.Lang]

		// Zero-shot metrics: Count ALL runs, but look at first attempt only
		// Success rate: based on first_attempt_ok across all runs
		// Tokens/cost: only from non-repair runs (repair inflates these metrics)
		stats.zeroShotRuns++
		if r.FirstAttemptOk {
			stats.zeroShotSuccess++
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

		// Track agent-comparable metrics (same benchmarks as agent ran)
		if agentBenchmarkIDs[r.ID] {
			comparableStats := langAgentComparableStats[r.Lang]

			// Zero-shot: Count ALL runs, look at first attempt only
			comparableStats.zeroShotRuns++
			if r.FirstAttemptOk {
				comparableStats.zeroShotSuccess++
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
			comparableStats.finalTokens += r.OutputTokens
			comparableStats.finalCost += r.CostUSD
			langAgentComparableStats[r.Lang] = comparableStats
		}
	}

	// Calculate per-language agent statistics with turn efficiency breakdown
	langAgentStats := make(map[string]struct {
		runs         int
		success      int
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

			// Final metrics (including repairs)
			if stdStats.finalRuns > 0 {
				langData["final_success_avg_tokens"] = float64(stdStats.finalTokens) / float64(stdStats.finalRuns)
				langData["final_success_avg_cost"] = stdStats.finalCost / float64(stdStats.finalRuns)
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

				if comparableStats.finalRuns > 0 {
					finalSuccess := float64(comparableStats.finalSuccess) / float64(comparableStats.finalRuns)
					finalAvgCost := comparableStats.finalCost / float64(comparableStats.finalRuns)
					langData["final_success_comparable"] = finalSuccess
					langData["final_success_avg_tokens_comparable"] = float64(comparableStats.finalTokens) / float64(comparableStats.finalRuns)
					langData["final_success_avg_cost_comparable"] = finalAvgCost

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
			if ls, ok := langMap[lang]; ok && ls.runs > 0 {
				langData["agent_success_rate_"+executor] = float64(ls.success) / float64(ls.runs)
				langData["agent_avg_turns_"+executor] = float64(ls.turns) / float64(ls.runs)
				langData["agent_avg_tokens_"+executor] = float64(ls.tokens) / float64(ls.runs)
				langData["agent_avg_cost_"+executor] = ls.cost / float64(ls.runs)
				langData["agent_runs_"+executor] = ls.runs
			}
		}

		languagesMap[lang] = langData
	}

	// Build sorted executor list for frontend
	executorList := make([]string, 0, len(perExecutorStats))
	for executor := range perExecutorStats {
		executorList = append(executorList, executor)
	}
	sort.Strings(executorList)
	aggregatesJS["agentExecutors"] = executorList

	// Update dashboard with current version data
	dashboard.Version = matrix.Version
	dashboard.Timestamp = time.Now().Format(time.RFC3339)
	dashboard.TotalRuns = matrix.TotalRuns
	dashboard.Aggregates = aggregatesJS
	dashboard.Models = modelsJS
	dashboard.AgentModels = agentModelsJS
	dashboard.Benchmarks = benchmarksJS
	dashboard.Languages = languagesMap
	dashboard.Executors = executorsJS

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
