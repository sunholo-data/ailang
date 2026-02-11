package trace

import "math"

// TraceScore holds the quality assessment of a trace.
type TraceScore struct {
	Total            float64        `json:"total"`             // Weighted total 0.0-1.0
	Completion       float64        `json:"completion"`        // Did program complete without errors?
	Complexity       float64        `json:"complexity"`        // Number of functions, effects, depth
	ContractCoverage float64        `json:"contract_coverage"` // Percentage of contracts that passed
	BudgetEfficiency float64        `json:"budget_efficiency"` // Used budget vs declared budget
	EffectDiversity  float64        `json:"effect_diversity"`  // Uses multiple effect types
	Stats            TraceStats     `json:"stats"`             // Raw statistics
	EffectBreakdown  map[string]int `json:"effect_breakdown"`  // Per-effect invocation counts
	FunctionCounts   map[string]int `json:"function_counts"`   // Per-function call counts
}

// TraceStats holds raw statistical data extracted from a trace.
type TraceStats struct {
	TotalEvents       int  `json:"total_events"`
	FunctionCalls     int  `json:"function_calls"` // Enter events
	DistinctFunctions int  `json:"distinct_functions"`
	EffectCalls       int  `json:"effect_calls"`
	DistinctEffects   int  `json:"distinct_effects"`
	ContractChecks    int  `json:"contract_checks"`
	ContractsPassed   int  `json:"contracts_passed"`
	BudgetEvents      int  `json:"budget_events"`
	MaxDepth          int  `json:"max_depth"`
	HasErrors         bool `json:"has_errors"`
	HasModuleEnd      bool `json:"has_module_end"`
}

// Default scoring weights (from design doc).
const (
	weightCompletion       = 0.30
	weightComplexity       = 0.25
	weightContractCoverage = 0.20
	weightBudgetEfficiency = 0.15
	weightEffectDiversity  = 0.10
)

// ScoreTrace evaluates the quality of a trace for training data purposes.
// Returns a score between 0.0 and 1.0 with component breakdowns.
func ScoreTrace(events []TraceEvent) TraceScore {
	stats := collectStats(events)
	score := TraceScore{
		Stats:           stats,
		EffectBreakdown: collectEffectBreakdown(events),
		FunctionCounts:  collectFunctionCounts(events),
	}

	score.Completion = scoreCompletion(stats)
	score.Complexity = scoreComplexity(stats)
	score.ContractCoverage = scoreContractCoverage(stats)
	score.BudgetEfficiency = scoreBudgetEfficiency(events, stats)
	score.EffectDiversity = scoreEffectDiversity(stats)

	score.Total = weightCompletion*score.Completion +
		weightComplexity*score.Complexity +
		weightContractCoverage*score.ContractCoverage +
		weightBudgetEfficiency*score.BudgetEfficiency +
		weightEffectDiversity*score.EffectDiversity

	// Clamp to [0, 1]
	score.Total = math.Max(0, math.Min(1, score.Total))

	return score
}

// collectStats gathers raw statistics from events.
func collectStats(events []TraceEvent) TraceStats {
	stats := TraceStats{TotalEvents: len(events)}
	functions := make(map[string]bool)
	effects := make(map[string]bool)

	for _, e := range events {
		if e.Depth > stats.MaxDepth {
			stats.MaxDepth = e.Depth
		}

		switch e.Event {
		case EventFunctionEnter:
			stats.FunctionCalls++
			if e.Function != nil {
				functions[e.Function.Name] = true
			}
		case EventEffect:
			stats.EffectCalls++
			if e.Effect != nil {
				effects[e.Effect.EffectName] = true
			}
		case EventContractCheck:
			stats.ContractChecks++
			if e.Contract != nil && e.Contract.Passed {
				stats.ContractsPassed++
			}
		case EventBudgetDelta:
			stats.BudgetEvents++
		case EventError:
			stats.HasErrors = true
		case EventModuleEnd:
			stats.HasModuleEnd = true
		}
	}

	stats.DistinctFunctions = len(functions)
	stats.DistinctEffects = len(effects)
	return stats
}

// scoreCompletion: 1.0 if no errors and module completed, 0.0 if errors.
func scoreCompletion(stats TraceStats) float64 {
	if stats.HasErrors {
		return 0.0
	}
	if stats.HasModuleEnd {
		return 1.0
	}
	// No module events but also no errors — simple program
	if stats.TotalEvents > 0 {
		return 0.8
	}
	return 0.0
}

// scoreComplexity: higher for programs with more functions, depth, and variety.
// Uses diminishing returns (log scale) to avoid rewarding bloat.
func scoreComplexity(stats TraceStats) float64 {
	if stats.TotalEvents == 0 {
		return 0.0
	}

	// Function diversity: 1 function = 0.2, 5+ = 1.0
	funcScore := math.Min(1.0, float64(stats.DistinctFunctions)/5.0)

	// Depth: depth 1 = 0.2, depth 5+ = 1.0
	depthScore := math.Min(1.0, float64(stats.MaxDepth)/5.0)

	// Call count: use log scale. 1 call = 0.0, 10 calls = 0.5, 100+ = 1.0
	callScore := 0.0
	if stats.FunctionCalls > 0 {
		callScore = math.Min(1.0, math.Log10(float64(stats.FunctionCalls))/2.0)
	}

	return (funcScore + depthScore + callScore) / 3.0
}

// scoreContractCoverage: 1.0 if all contracts passed, 0.0 if none.
// Programs without contracts get 0.5 (neutral — neither penalized nor rewarded).
func scoreContractCoverage(stats TraceStats) float64 {
	if stats.ContractChecks == 0 {
		return 0.5 // No contracts = neutral
	}
	return float64(stats.ContractsPassed) / float64(stats.ContractChecks)
}

// scoreBudgetEfficiency: rewards programs that use budgets without exhausting them.
// Programs without budgets get 0.5 (neutral).
func scoreBudgetEfficiency(events []TraceEvent, stats TraceStats) float64 {
	if stats.BudgetEvents == 0 {
		return 0.5 // No budgets = neutral
	}

	// Collect final budget states per effect
	budgets := make(map[string]*BudgetEvent)
	for _, e := range events {
		if e.Event == EventBudgetDelta && e.Budget != nil {
			b := e.Budget
			budgets[b.Effect] = b
		}
	}

	if len(budgets) == 0 {
		return 0.5
	}

	totalEfficiency := 0.0
	counted := 0
	for _, b := range budgets {
		if b.Limit <= 0 {
			continue // Unlimited — skip
		}
		// Efficiency = used/limit. 0% usage = wasteful (0.3), 100% = exhausted (0.5), 20-80% = good (1.0)
		ratio := float64(b.Used) / float64(b.Limit)
		if ratio >= 0.2 && ratio <= 0.8 {
			totalEfficiency += 1.0
		} else if ratio > 0.8 {
			totalEfficiency += 0.5 // Near-exhaustion
		} else {
			totalEfficiency += 0.3 // Barely used
		}
		counted++
	}

	if counted == 0 {
		return 0.5
	}
	return totalEfficiency / float64(counted)
}

// scoreEffectDiversity: rewards programs using multiple effect types.
// 0 effects = 0.0, 1 effect = 0.4, 2 = 0.7, 3+ = 1.0.
func scoreEffectDiversity(stats TraceStats) float64 {
	switch stats.DistinctEffects {
	case 0:
		return 0.0
	case 1:
		return 0.4
	case 2:
		return 0.7
	default:
		return 1.0
	}
}

// collectEffectBreakdown counts invocations per effect type.
func collectEffectBreakdown(events []TraceEvent) map[string]int {
	breakdown := make(map[string]int)
	for _, e := range events {
		if e.Event == EventEffect && e.Effect != nil {
			breakdown[e.Effect.EffectName]++
		}
	}
	return breakdown
}

// collectFunctionCounts counts calls per function name.
func collectFunctionCounts(events []TraceEvent) map[string]int {
	counts := make(map[string]int)
	for _, e := range events {
		if e.Event == EventFunctionEnter && e.Function != nil {
			counts[e.Function.Name]++
		}
	}
	return counts
}
