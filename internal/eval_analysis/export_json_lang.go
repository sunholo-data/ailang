package eval_analysis

// Language-level statistics helpers for ExportBenchmarkJSON.
//
// Extracted from export_json.go to keep that file under the 800-line soft
// limit. The two public entry points are:
//
//   - buildLangStandardStats — accumulates per-language zero-shot / final /
//     repair counters from standard results, plus agent-comparable and
//     per-executor comparable subsets.
//   - buildLangAgentStats — accumulates per-language agent counters including
//     turn-efficiency breakdown.
//   - buildLanguagesMap — assembles the final languagesMap that goes into
//     DashboardJSON.Languages, weaving together standard, agent, comparable,
//     and per-executor statistics.

// langStdStat holds per-language zero-shot, final, and repair counters
// accumulated from standard (non-agent) results.
type langStdStat struct {
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
}

// langComparableStat holds the subset of standard-result counters filtered
// to benchmarks AND models that also ran in agent mode (fair comparison).
type langComparableStat struct {
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
}

// perExecComparable holds per-executor comparable baseline counters —
// standard-result metrics filtered to exactly the benchmarks and models each
// executor ran. Enables fair per-harness success-gap / cost-ratio deltas.
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

// langAgentStat holds per-language agent counters including turn-efficiency.
type langAgentStat struct {
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
}

// buildLangStandardStats walks standardResults once and returns:
//   - langStandardStats: per-language overall counters
//   - langAgentComparableStats: same-benchmark+model subset for fair agent comparison
//   - langPerExecComparableStats: per-executor filtered subset
func buildLangStandardStats(
	standardResults []*BenchmarkResult,
	agentBenchmarkIDs map[string]bool,
	agentModelsSet map[string]bool,
	perExecBenchmarkIDs map[string]map[string]bool,
	perExecModelsSet map[string]map[string]bool,
) (
	map[string]langStdStat,
	map[string]langComparableStat,
	map[string]map[string]*perExecComparable,
) {
	langStandardStats := make(map[string]langStdStat)
	langAgentComparableStats := make(map[string]langComparableStat)
	langPerExecComparableStats := make(map[string]map[string]*perExecComparable)

	for _, r := range standardResults {
		stats := langStandardStats[r.Lang]
		isApiErr := ShouldExcludeFromCapability(r.ErrorCategory)

		// Zero-shot metrics: Count ALL runs, but look at first attempt only.
		// Tokens/cost: only from non-repair runs (repair inflates these metrics).
		stats.zeroShotRuns++
		if r.FirstAttemptOk {
			stats.zeroShotSuccess++
		}
		if isApiErr {
			stats.zeroShotApiErr++
		}
		if !r.RepairUsed {
			if r.OutputTokens > 0 {
				stats.zeroShotTokens += r.OutputTokens
			}
			stats.zeroShotCost += r.CostUSD
		}

		// Final metrics (including repair attempts) — use ALL runs.
		stats.finalRuns++
		if r.StdoutOk {
			stats.finalSuccess++
		}
		if isApiErr {
			stats.finalApiErr++
		}
		stats.finalTokens += r.OutputTokens
		stats.finalCost += r.CostUSD

		// Repair metrics.
		if r.RepairUsed {
			stats.repairAttempts++
			if r.StdoutOk {
				stats.repairSuccess++
			}
		}

		langStandardStats[r.Lang] = stats

		// Track agent-comparable metrics: same benchmarks AND same models as
		// agent ran. Without the model filter, standard's flagship pool gets
		// compared against agent's cheap pool.
		if agentBenchmarkIDs[r.ID] && agentModelsSet[r.Model] {
			comp := langAgentComparableStats[r.Lang]

			comp.zeroShotRuns++
			if r.FirstAttemptOk {
				comp.zeroShotSuccess++
			}
			if isApiErr {
				comp.zeroShotApiErr++
			}
			if !r.RepairUsed {
				if r.OutputTokens > 0 {
					comp.zeroShotTokens += r.OutputTokens
				}
				comp.zeroShotCost += r.CostUSD
			}

			comp.finalRuns++
			if r.StdoutOk {
				comp.finalSuccess++
			}
			if isApiErr {
				comp.finalApiErr++
			}
			comp.finalTokens += r.OutputTokens
			comp.finalCost += r.CostUSD
			langAgentComparableStats[r.Lang] = comp
		}

		// Per-executor comparable: count this standard run for every executor
		// where (a) the benchmark was run by that executor AND (b) the model
		// was used by that executor.
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

	return langStandardStats, langAgentComparableStats, langPerExecComparableStats
}

// buildLangAgentStats walks agentResults once and returns per-language agent
// counters including turn-efficiency breakdowns for success vs failure paths.
func buildLangAgentStats(agentResults []*BenchmarkResult) map[string]langAgentStat {
	langAgentStats := make(map[string]langAgentStat)
	for _, r := range agentResults {
		stats := langAgentStats[r.Lang]
		stats.runs++
		stats.turns += r.AgentTurns
		stats.tokens += r.TotalTokens
		stats.cost += r.CostUSD
		if ShouldExcludeFromCapability(r.ErrorCategory) {
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
	return langAgentStats
}

// buildLanguagesMap assembles the DashboardJSON.Languages map, weaving
// together standard, agent, comparable, and per-executor statistics for each
// source language.
func buildLanguagesMap(
	matrix *PerformanceMatrix,
	langStandardStats map[string]langStdStat,
	langAgentComparableStats map[string]langComparableStat,
	langPerExecComparableStats map[string]map[string]*perExecComparable,
	langAgentStats map[string]langAgentStat,
	perExecutorLangStats map[string]map[string]ExecutorLangStats,
) map[string]interface{} {
	languagesMap := make(map[string]interface{})
	for lang, stats := range matrix.Languages {
		langData := map[string]interface{}{
			"total_runs":   stats.TotalRuns,
			"success_rate": stats.SuccessRate,
			"avg_tokens":   stats.AvgTokens,
		}

		// Add zero-shot, final, and cost metrics from standard results.
		if stdStats, ok := langStandardStats[lang]; ok && stdStats.zeroShotRuns > 0 {
			langData["zero_shot_success"] = float64(stdStats.zeroShotSuccess) / float64(stdStats.zeroShotRuns)
			langData["zero_shot_avg_tokens"] = float64(stdStats.zeroShotTokens) / float64(stdStats.zeroShotRuns)
			langData["zero_shot_avg_cost"] = stdStats.zeroShotCost / float64(stdStats.zeroShotRuns)

			if zsNonApi := stdStats.zeroShotRuns - stdStats.zeroShotApiErr; zsNonApi > 0 {
				langData["zero_shot_success_adjusted"] = float64(stdStats.zeroShotSuccess) / float64(zsNonApi)
				langData["zero_shot_api_errors"] = stdStats.zeroShotApiErr
				langData["zero_shot_api_error_rate"] = float64(stdStats.zeroShotApiErr) / float64(stdStats.zeroShotRuns)
			}

			if stdStats.finalRuns > 0 {
				langData["final_success_avg_tokens"] = float64(stdStats.finalTokens) / float64(stdStats.finalRuns)
				langData["final_success_avg_cost"] = stdStats.finalCost / float64(stdStats.finalRuns)
				if finalNonApi := stdStats.finalRuns - stdStats.finalApiErr; finalNonApi > 0 {
					langData["final_success_adjusted"] = float64(stdStats.finalSuccess) / float64(finalNonApi)
				}
			}

			if stdStats.repairAttempts > 0 {
				langData["repair_success_rate"] = float64(stdStats.repairSuccess) / float64(stdStats.repairAttempts)
			}

			langData["avg_cost_usd"] = stdStats.finalCost / float64(stdStats.finalRuns)
		}

		// Add agent metrics if available for this language.
		if agentStats, ok := langAgentStats[lang]; ok && agentStats.runs > 0 {
			agentSuccessRate := float64(agentStats.success) / float64(agentStats.runs)
			agentAvgCost := agentStats.cost / float64(agentStats.runs)

			langData["agent_runs"] = agentStats.runs
			langData["agent_success_rate"] = agentSuccessRate
			langData["agent_avg_turns"] = float64(agentStats.turns) / float64(agentStats.runs)
			langData["agent_avg_tokens"] = float64(agentStats.tokens) / float64(agentStats.runs)
			langData["agent_avg_cost"] = agentAvgCost

			langData["agent_api_errors"] = agentStats.apiErrors
			langData["agent_api_error_rate"] = float64(agentStats.apiErrors) / float64(agentStats.runs)
			if nonApi := agentStats.runs - agentStats.apiErrors; nonApi > 0 {
				langData["agent_success_rate_adjusted"] = float64(agentStats.success) / float64(nonApi)
			}

			if agentStats.successCount > 0 {
				langData["agent_avg_turns_success"] = float64(agentStats.successTurns) / float64(agentStats.successCount)
			}
			if agentStats.failureCount > 0 {
				langData["agent_avg_turns_failure"] = float64(agentStats.failureTurns) / float64(agentStats.failureCount)
			}

			// Add agent-comparable standard metrics for fair comparison.
			if comp, ok := langAgentComparableStats[lang]; ok && comp.zeroShotRuns > 0 {
				zeroShotSuccess := float64(comp.zeroShotSuccess) / float64(comp.zeroShotRuns)
				zeroShotAvgCost := comp.zeroShotCost / float64(comp.zeroShotRuns)

				langData["zero_shot_success_comparable"] = zeroShotSuccess
				langData["zero_shot_avg_tokens_comparable"] = float64(comp.zeroShotTokens) / float64(comp.zeroShotRuns)
				langData["zero_shot_avg_cost_comparable"] = zeroShotAvgCost
				if zsNonApi := comp.zeroShotRuns - comp.zeroShotApiErr; zsNonApi > 0 {
					langData["zero_shot_success_comparable_adjusted"] = float64(comp.zeroShotSuccess) / float64(zsNonApi)
				}

				if comp.finalRuns > 0 {
					finalSuccess := float64(comp.finalSuccess) / float64(comp.finalRuns)
					finalAvgCost := comp.finalCost / float64(comp.finalRuns)
					langData["final_success_comparable"] = finalSuccess
					langData["final_success_avg_tokens_comparable"] = float64(comp.finalTokens) / float64(comp.finalRuns)
					langData["final_success_avg_cost_comparable"] = finalAvgCost
					if finalNonApi := comp.finalRuns - comp.finalApiErr; finalNonApi > 0 {
						langData["final_success_comparable_adjusted"] = float64(comp.finalSuccess) / float64(finalNonApi)
					}

					if finalSuccess > 0 {
						langData["final_cost_per_success_comparable"] = finalAvgCost / finalSuccess
					}
				}

				// Derived metrics showing agent superiority on hard benchmarks.
				langData["agent_success_gap"] = agentSuccessRate - zeroShotSuccess

				if zeroShotSuccess < 1.0 {
					langData["agent_impossibility_coverage"] = (agentSuccessRate - zeroShotSuccess) / (1.0 - zeroShotSuccess)
				}

				if zeroShotSuccess > 0 && agentSuccessRate > 0 {
					zeroShotCostPerSuccess := zeroShotAvgCost / zeroShotSuccess
					agentCostPerSuccess := agentAvgCost / agentSuccessRate
					langData["agent_cost_per_success"] = agentCostPerSuccess
					langData["zero_shot_cost_per_success"] = zeroShotCostPerSuccess
					langData["agent_cost_efficiency_ratio"] = agentCostPerSuccess / zeroShotCostPerSuccess
				}
			}
		}

		// Add per-executor agent breakdown for this language.
		for executor, langMap := range perExecutorLangStats {
			if ls, ok := langMap[lang]; ok && ls.Runs > 0 {
				langData["agent_success_rate_"+executor] = float64(ls.Success) / float64(ls.Runs)
				langData["agent_avg_turns_"+executor] = float64(ls.Turns) / float64(ls.Runs)
				langData["agent_avg_tokens_"+executor] = float64(ls.Tokens) / float64(ls.Runs)
				langData["agent_avg_cost_"+executor] = ls.Cost / float64(ls.Runs)
				langData["agent_runs_"+executor] = ls.Runs

				langData["agent_api_errors_"+executor] = ls.APIErrors
				langData["agent_api_error_rate_"+executor] = float64(ls.APIErrors) / float64(ls.Runs)
				if nonApi := ls.Runs - ls.APIErrors; nonApi > 0 {
					langData["agent_success_rate_adjusted_"+executor] = float64(ls.Success) / float64(nonApi)
				}

				// Per-executor comparable baseline + derived deltas.
				if execLangMap, ok := langPerExecComparableStats[executor]; ok {
					if pec, ok := execLangMap[lang]; ok && pec.zeroShotRuns > 0 {
						zeroShotSuccessExec := float64(pec.zeroShotSuccess) / float64(pec.zeroShotRuns)
						zeroShotAvgCostExec := pec.zeroShotCost / float64(pec.zeroShotRuns)

						langData["zero_shot_success_comparable_"+executor] = zeroShotSuccessExec
						langData["zero_shot_avg_cost_comparable_"+executor] = zeroShotAvgCostExec

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

						langData["agent_success_gap_"+executor] = agentSuccessRateExec - zeroShotSuccessExec

						if haveZsAdj && haveAgentAdj {
							langData["agent_success_gap_adjusted_"+executor] = agentAdjExec - zsAdjExec
						}

						if zeroShotSuccessExec > 0 && agentSuccessRateExec > 0 {
							zeroShotCPS := zeroShotAvgCostExec / zeroShotSuccessExec
							agentCPS := agentAvgCostExec / agentSuccessRateExec
							if zeroShotCPS > 0 {
								langData["agent_cost_efficiency_ratio_"+executor] = agentCPS / zeroShotCPS
							}
						}

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
	return languagesMap
}
