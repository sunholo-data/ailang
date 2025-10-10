package eval_analysis

import (
	"fmt"
	"sort"
	"time"
)

// GenerateMatrix generates a performance matrix from benchmark results
// This replaces the brittle jq-based bash script with type-safe Go code
func GenerateMatrix(results []*BenchmarkResult, version string) (*PerformanceMatrix, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to generate matrix from")
	}

	matrix := &PerformanceMatrix{
		Version:   version,
		Timestamp: time.Now(),
		TotalRuns: len(results),
	}

	// Calculate overall aggregates
	matrix.Aggregates = calculateAggregates(results)

	// Group by model
	matrix.Models = groupByModel(results)

	// Group by benchmark
	matrix.Benchmarks = groupByBenchmark(results)

	// Group by error code
	matrix.ErrorCodes = groupByErrorCode(results)

	// Group by language
	matrix.Languages = groupByLanguage(results)

	// Group by prompt version (if available)
	matrix.PromptVersions = groupByPromptVersion(results)

	return matrix, nil
}

// calculateAggregates computes overall statistics
func calculateAggregates(results []*BenchmarkResult) Aggregates {
	var agg Aggregates

	firstAttemptSuccess := 0
	finalSuccess := 0
	repairUsed := 0
	repairSuccess := 0
	totalTokens := 0
	totalCost := 0.0
	totalDuration := int64(0)

	for _, r := range results {
		if r.FirstAttemptOk {
			firstAttemptSuccess++
		}
		if r.StdoutOk {
			finalSuccess++
		}
		if r.RepairUsed {
			repairUsed++
			if r.RepairOk {
				repairSuccess++
			}
		}

		totalTokens += r.TotalTokens
		totalCost += r.CostUSD
		totalDuration += r.DurationMs
	}

	agg.ZeroShotSuccess = safeDiv(float64(firstAttemptSuccess), float64(len(results)))
	agg.FinalSuccess = safeDiv(float64(finalSuccess), float64(len(results)))
	agg.RepairUsed = repairUsed
	agg.RepairSuccessRate = safeDiv(float64(repairSuccess), float64(repairUsed))
	agg.TotalTokens = totalTokens
	agg.TotalCostUSD = totalCost
	agg.AvgDurationMs = safeDiv(float64(totalDuration), float64(len(results)))

	return agg
}

// groupByModel groups results by model
func groupByModel(results []*BenchmarkResult) map[string]*ModelStats {
	// Group results by model
	modelResults := make(map[string][]*BenchmarkResult)
	for _, r := range results {
		modelResults[r.Model] = append(modelResults[r.Model], r)
	}

	// Calculate stats for each model
	models := make(map[string]*ModelStats)
	for model, results := range modelResults {
		stats := &ModelStats{
			TotalRuns:  len(results),
			Aggregates: calculateAggregates(results),
			Benchmarks: make(map[string]*BenchmarkRun),
		}

		// Group by benchmark within this model
		benchResults := make(map[string]*BenchmarkResult)
		for _, r := range results {
			// Keep the latest result for each benchmark
			if existing, exists := benchResults[r.ID]; !exists || r.Timestamp.After(existing.Timestamp) {
				benchResults[r.ID] = r
			}
		}

		for benchID, r := range benchResults {
			stats.Benchmarks[benchID] = &BenchmarkRun{
				Success:        r.StdoutOk,
				FirstAttemptOk: r.FirstAttemptOk,
				RepairUsed:     r.RepairUsed,
				Tokens:         r.TotalTokens,
			}
		}

		models[model] = stats
	}

	return models
}

// groupByBenchmark groups results by benchmark ID
func groupByBenchmark(results []*BenchmarkResult) map[string]*BenchmarkStats {
	// Group results by benchmark
	benchResults := make(map[string][]*BenchmarkResult)
	for _, r := range results {
		benchResults[r.ID] = append(benchResults[r.ID], r)
	}

	// Calculate stats for each benchmark
	benchmarks := make(map[string]*BenchmarkStats)
	for benchID, results := range benchResults {
		successCount := 0
		totalTokens := 0
		langs := make(map[string]bool)

		for _, r := range results {
			if r.StdoutOk {
				successCount++
			}
			totalTokens += r.TotalTokens
			langs[r.Lang] = true
		}

		// Extract language list
		langList := make([]string, 0, len(langs))
		for lang := range langs {
			langList = append(langList, lang)
		}
		sort.Strings(langList)

		benchmarks[benchID] = &BenchmarkStats{
			TotalRuns:   len(results),
			SuccessRate: safeDiv(float64(successCount), float64(len(results))),
			AvgTokens:   safeDiv(float64(totalTokens), float64(len(results))),
			Languages:   langList,
		}
	}

	return benchmarks
}

// groupByErrorCode groups failures by error code
func groupByErrorCode(results []*BenchmarkResult) []*ErrorCodeStats {
	// Only consider failures with error codes
	errorResults := make(map[string][]*BenchmarkResult)
	for _, r := range results {
		if r.ErrCode != "" && !r.StdoutOk {
			errorResults[r.ErrCode] = append(errorResults[r.ErrCode], r)
		}
	}

	// Calculate stats for each error code
	var errorCodes []*ErrorCodeStats
	for code, results := range errorResults {
		repairSuccess := 0
		for _, r := range results {
			if r.RepairOk {
				repairSuccess++
			}
		}

		errorCodes = append(errorCodes, &ErrorCodeStats{
			Code:          code,
			Count:         len(results),
			RepairSuccess: safeDiv(float64(repairSuccess), float64(len(results))),
		})
	}

	// Sort by count (descending)
	sort.Slice(errorCodes, func(i, j int) bool {
		return errorCodes[i].Count > errorCodes[j].Count
	})

	return errorCodes
}

// groupByLanguage groups results by language
func groupByLanguage(results []*BenchmarkResult) map[string]*LanguageStats {
	// Group results by language
	langResults := make(map[string][]*BenchmarkResult)
	for _, r := range results {
		langResults[r.Lang] = append(langResults[r.Lang], r)
	}

	// Calculate stats for each language
	languages := make(map[string]*LanguageStats)
	for lang, results := range langResults {
		successCount := 0
		totalOutputTokens := 0

		for _, r := range results {
			if r.StdoutOk {
				successCount++
			}
			// Use OutputTokens instead of TotalTokens to exclude input prompt
			totalOutputTokens += r.OutputTokens
		}

		languages[lang] = &LanguageStats{
			TotalRuns:   len(results),
			SuccessRate: safeDiv(float64(successCount), float64(len(results))),
			AvgTokens:   safeDiv(float64(totalOutputTokens), float64(len(results))),
		}
	}

	return languages
}

// groupByPromptVersion groups results by prompt version
func groupByPromptVersion(results []*BenchmarkResult) map[string]*PromptStats {
	// Group results by prompt version
	promptResults := make(map[string][]*BenchmarkResult)
	for _, r := range results {
		if r.PromptVersion != "" {
			promptResults[r.PromptVersion] = append(promptResults[r.PromptVersion], r)
		}
	}

	// If no prompt versions, return empty map
	if len(promptResults) == 0 {
		return nil
	}

	// Calculate stats for each prompt version
	prompts := make(map[string]*PromptStats)
	for version, results := range promptResults {
		firstAttemptSuccess := 0
		finalSuccess := 0
		totalTokens := 0

		for _, r := range results {
			if r.FirstAttemptOk {
				firstAttemptSuccess++
			}
			if r.StdoutOk {
				finalSuccess++
			}
			totalTokens += r.TotalTokens
		}

		prompts[version] = &PromptStats{
			TotalRuns:       len(results),
			ZeroShotSuccess: safeDiv(float64(firstAttemptSuccess), float64(len(results))),
			FinalSuccess:    safeDiv(float64(finalSuccess), float64(len(results))),
			AvgTokens:       safeDiv(float64(totalTokens), float64(len(results))),
		}
	}

	return prompts
}
