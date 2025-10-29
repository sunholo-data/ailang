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

// loadExistingDashboard reads the existing dashboard JSON file and returns its structure
// If the file doesn't exist, returns an empty dashboard with an empty history array
func loadExistingDashboard(path string) (*DashboardJSON, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &DashboardJSON{History: []HistoryEntry{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var dashboard DashboardJSON
	if err := json.Unmarshal(data, &dashboard); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return &dashboard, nil
}

// mergeHistory adds a new entry to the dashboard history or updates an existing entry
// If the version already exists, it updates that entry. Otherwise, prepends the new entry.
// History is maintained in reverse chronological order (newest first)
func mergeHistory(dashboard *DashboardJSON, newEntry HistoryEntry) {
	// Check for duplicate version
	for i, entry := range dashboard.History {
		if entry.Version == newEntry.Version {
			// Update existing entry
			dashboard.History[i] = newEntry
			return
		}
	}

	// Prepend new entry (reverse chronological order)
	dashboard.History = append([]HistoryEntry{newEntry}, dashboard.History...)
}

// buildHistoryEntryFromMatrix creates a HistoryEntry from a PerformanceMatrix and results
func buildHistoryEntryFromMatrix(matrix *PerformanceMatrix, results []*BenchmarkResult) HistoryEntry {
	successCount := 0
	for _, r := range results {
		if r.StdoutOk {
			successCount++
		}
	}

	successRate := 0.0
	if matrix.TotalRuns > 0 {
		successRate = float64(successCount) / float64(matrix.TotalRuns)
	}

	// Build language stats
	langStats := make(map[string]interface{})
	for lang, stats := range matrix.Languages {
		if stats.TotalRuns > 0 {
			langStats[lang] = map[string]interface{}{
				"success_rate": stats.SuccessRate,
				"total_runs":   stats.TotalRuns,
			}
		}
	}

	// Determine languages string
	languages := ""
	if len(matrix.Languages) > 0 {
		langList := make([]string, 0, len(matrix.Languages))
		for lang := range matrix.Languages {
			langList = append(langList, lang)
		}
		sort.Strings(langList)
		languages = strings.Join(langList, ",")
	}

	return HistoryEntry{
		Version:       matrix.Version,
		Timestamp:     matrix.Timestamp.Format(time.RFC3339),
		SuccessRate:   successRate,
		TotalRuns:     matrix.TotalRuns,
		SuccessCount:  successCount,
		Languages:     languages,
		LanguageStats: langStats,
	}
}

// writeJSONAtomic writes JSON data to a file atomically
// Uses a temp file + rename to ensure all-or-nothing writes
func writeJSONAtomic(path string, data interface{}) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	tmpPath := path + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tmpPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Validate temp file
	tmpData, err := os.ReadFile(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to read temp file: %w", err)
	}

	// Parse and validate
	if dashboard, ok := data.(*DashboardJSON); ok {
		var test DashboardJSON
		if err := json.Unmarshal(tmpData, &test); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("validation failed: %w", err)
		}

		if err := test.Validate(); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("validation failed: %w", err)
		}

		// Verify version matches
		if test.Version != dashboard.Version {
			os.Remove(tmpPath)
			return fmt.Errorf("version mismatch after marshaling: expected %s, got %s",
				dashboard.Version, test.Version)
		}
	}

	// Atomic rename (on Unix, overwrites atomically)
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename: %w", err)
	}

	return nil
}

// ExportDocusaurusMDX generates an MDX file with React components for Docusaurus
func ExportDocusaurusMDX(matrix *PerformanceMatrix, history []*Baseline) string {
	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString("sidebar_position: 6\n")
	sb.WriteString("title: Benchmark Performance\n")
	sb.WriteString("description: Real-world AI code generation performance metrics for AILANG\n")
	sb.WriteString(fmt.Sprintf("last_updated: %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString("---\n\n")

	// Import React components
	sb.WriteString("import BenchmarkDashboard from '@site/src/components/BenchmarkDashboard';\n")
	sb.WriteString("import ModelRadarComparison from '@site/src/components/ModelRadarComparison';\n\n")

	// Hero section
	sb.WriteString("# AI Code Generation Benchmarks\n\n")
	sb.WriteString("Real-world performance metrics for AILANG vs Python across multiple AI models.\n\n")

	// Model Radar Comparison (at the top)
	sb.WriteString("## Model Comparison\n\n")
	sb.WriteString("Compare AI model performance across multiple dimensions:\n\n")
	sb.WriteString("<ModelRadarComparison />\n\n")

	// Dashboard component
	sb.WriteString("<BenchmarkDashboard />\n\n")

	// Explanation section
	sb.WriteString("## What These Numbers Mean\n\n")
	sb.WriteString("Our benchmark suite tests AI models' ability to generate correct, working code in both AILANG and Python.\n\n")

	sb.WriteString("### Success Metrics\n\n")
	sb.WriteString("- **0-Shot Success**: Code works on first try (no repairs)\n")
	sb.WriteString("- **Final Success**: Code works after M-EVAL-LOOP self-repair\n")
	sb.WriteString("- **Token Efficiency**: Lower tokens = more concise code\n\n")

	sb.WriteString("### Why This Matters\n\n")
	sb.WriteString("These benchmarks demonstrate:\n\n")
	sb.WriteString("1. **Type Safety Works**: AILANG's type system catches errors early\n")
	sb.WriteString("2. **Effects Are Clear**: Explicit effect annotations help AI models\n")
	sb.WriteString("3. **Patterns Are Learnable**: AI models understand functional programming\n")
	sb.WriteString("4. **Room to Grow**: Benchmarks identify language gaps and guide development\n\n")

	// Success stories
	sb.WriteString("## Where AILANG Shines\n\n")
	if len(matrix.Benchmarks) > 0 {
		// Find top performing benchmarks
		type benchEntry struct {
			id    string
			stats *BenchmarkStats
		}
		var benchmarks []benchEntry
		for id, stats := range matrix.Benchmarks {
			if stats.SuccessRate >= 0.8 { // 80%+ success
				benchmarks = append(benchmarks, benchEntry{id, stats})
			}
		}
		sort.Slice(benchmarks, func(i, j int) bool {
			return benchmarks[i].stats.SuccessRate > benchmarks[j].stats.SuccessRate
		})

		if len(benchmarks) > 0 {
			sb.WriteString("AILANG excels at these problem types:\n\n")
			for i, b := range benchmarks {
				if i >= 5 {
					break // Top 5
				}
				sb.WriteString(fmt.Sprintf("- **%s**: %.1f%% success rate\n",
					formatBenchmarkName(b.id), b.stats.SuccessRate*100))
			}
			sb.WriteString("\n")
		}
	}

	// Development impact
	sb.WriteString("## How Benchmarks Guide Development\n\n")
	sb.WriteString("The M-EVAL-LOOP system uses these benchmarks to:\n\n")
	sb.WriteString("1. **Identify Bugs**: Failing benchmarks reveal language issues\n")
	sb.WriteString("2. **Validate Fixes**: Compare before/after to confirm improvements\n")
	sb.WriteString("3. **Track Progress**: Historical data shows language evolution\n")
	sb.WriteString("4. **Prioritize Features**: High-impact failures guide roadmap\n\n")

	// Case study
	sb.WriteString("### Case Study: Float Equality Bug\n\n")
	sb.WriteString("The `adt_option` benchmark caught a critical bug where float comparisons ")
	sb.WriteString("with variables called `eq_Int` instead of `eq_Float`. ")
	sb.WriteString("The benchmark suite detected it, guided the fix, and validated the solution.\n\n")
	sb.WriteString("**Result**: Benchmark went from runtime_error → PASSING ✅\n\n")

	// Try it yourself
	sb.WriteString("## Try It Yourself\n\n")
	sb.WriteString("Want to see AILANG in action?\n\n")
	sb.WriteString("- **[Interactive REPL](/ailang/docs/reference/repl-commands)** - Try AILANG in your browser\n")
	sb.WriteString("- **[Code Examples](https://github.com/sunholo-data/ailang/tree/main/examples)** - 48+ working examples\n")
	sb.WriteString("- **[Getting Started](/ailang/docs/guides/getting-started)** - Install and run locally\n\n")

	// Technical details
	sb.WriteString("## Technical Details\n\n")
	sb.WriteString(fmt.Sprintf("**Version**: %s\n\n", matrix.Version))
	sb.WriteString(fmt.Sprintf("**Total Runs**: %d\n\n", matrix.TotalRuns))
	sb.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Model breakdown
	if len(matrix.Models) > 0 {
		sb.WriteString("### Model Performance Details\n\n")
		sb.WriteString("| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run | Baseline |\n")
		sb.WriteString("|-------|------|--------|-------|------------|----------|----------|\n")

		type modelEntry struct {
			name  string
			stats *ModelStats
		}
		var models []modelEntry
		for name, stats := range matrix.Models {
			models = append(models, modelEntry{name, stats})
		}
		sort.Slice(models, func(i, j int) bool {
			return models[i].stats.Aggregates.FinalSuccess > models[j].stats.Aggregates.FinalSuccess
		})

		for _, m := range models {
			avgCost := 0.0
			if m.stats.TotalRuns > 0 {
				avgCost = m.stats.Aggregates.TotalCostUSD / float64(m.stats.TotalRuns)
			}
			avgTokens := 0.0
			if m.stats.TotalRuns > 0 {
				avgTokens = float64(m.stats.Aggregates.TotalTokens) / float64(m.stats.TotalRuns)
			}

			baselineVersion := m.stats.BaselineVersion
			if baselineVersion == "" {
				baselineVersion = matrix.Version
			}

			sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %.1f%% | %.0f | $%.4f | %s |\n",
				formatModelName(m.name),
				m.stats.TotalRuns,
				m.stats.Aggregates.ZeroShotSuccess*100,
				m.stats.Aggregates.FinalSuccess*100,
				avgTokens,
				avgCost,
				baselineVersion))
		}
		sb.WriteString("\n")
	}

	// Benchmark details
	if len(matrix.Benchmarks) > 0 {
		sb.WriteString("### Benchmark Details\n\n")
		sb.WriteString("| Benchmark | Success Rate | Avg Tokens | Languages |\n")
		sb.WriteString("|-----------|--------------|------------|-----------|\n")

		type benchEntry struct {
			id    string
			stats *BenchmarkStats
		}
		var benchmarks []benchEntry
		for id, stats := range matrix.Benchmarks {
			benchmarks = append(benchmarks, benchEntry{id, stats})
		}
		sort.Slice(benchmarks, func(i, j int) bool {
			// Sort by success rate, then by ID
			if benchmarks[i].stats.SuccessRate != benchmarks[j].stats.SuccessRate {
				return benchmarks[i].stats.SuccessRate > benchmarks[j].stats.SuccessRate
			}
			return benchmarks[i].id < benchmarks[j].id
		})

		for _, b := range benchmarks {
			status := "✅"
			if b.stats.SuccessRate < 0.5 {
				status = "❌"
			} else if b.stats.SuccessRate < 1.0 {
				status = "⚠️"
			}

			sb.WriteString(fmt.Sprintf("| %s %s | %.1f%% | %.0f | %s |\n",
				status,
				formatBenchmarkName(b.id),
				b.stats.SuccessRate*100,
				b.stats.AvgTokens,
				strings.Join(b.stats.Languages, ", ")))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString("**Methodology**: Benchmarks use deterministic seeds across multiple AI models. ")
	sb.WriteString("Each benchmark tests code generation, compilation, and execution. ")
	sb.WriteString("The M-EVAL-LOOP system provides structured error feedback for automatic repair.\n\n")
	sb.WriteString("**Learn More**: ")
	sb.WriteString("[M-EVAL-LOOP Design](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md) | ")
	sb.WriteString("[Evaluation Guide](/ailang/docs/guides/evaluation/eval-loop)\n")

	return sb.String()
}

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

	// Add agent-only models (models that ran agents but not standard evals)
	for modelName, agentStats := range modelAgentStats {
		if _, exists := modelsJS[modelName]; !exists && agentStats.runs > 0 {
			// Create minimal model entry with only agent stats
			modelsJS[modelName] = map[string]interface{}{
				"totalRuns": 0, // No standard runs
				"aggregates": map[string]interface{}{
					"zeroShotSuccess":   0.0,
					"finalSuccess":      0.0,
					"repairUsed":        0,
					"repairSuccessRate": 0.0,
					"totalTokens":       0,
					"totalCostUSD":      0.0,
					"avgDurationMs":     0.0,
				},
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

	// Transform history to include calculated success rates and per-language breakdown
	historyJS := make([]map[string]interface{}, len(history))
	for i, baseline := range history {
		successRate := 0.0
		if baseline.TotalBenchmarks > 0 {
			successRate = float64(baseline.SuccessCount) / float64(baseline.TotalBenchmarks)
		}

		histEntry := map[string]interface{}{
			"version":      baseline.Version,
			"timestamp":    baseline.Timestamp.Format(time.RFC3339),
			"successRate":  successRate,
			"totalRuns":    baseline.TotalBenchmarks,
			"successCount": baseline.SuccessCount,
			"languages":    baseline.Languages, // May be "ailang", "python", or "ailang,python"
		}

		// Calculate per-language stats from results if available
		if len(baseline.Results) > 0 {
			langStats := make(map[string]*LanguageStats)
			for _, result := range baseline.Results {
				lang := result.Lang
				if lang == "" {
					continue
				}
				if langStats[lang] == nil {
					langStats[lang] = &LanguageStats{}
				}
				langStats[lang].TotalRuns++
				// Success = compile_ok && runtime_ok && stdout_ok
				if result.CompileOk && result.RuntimeOk && result.StdoutOk {
					langStats[lang].SuccessRate += 1.0
				}
			}

			// Calculate final success rates
			langStatsJS := make(map[string]interface{})
			for lang, stats := range langStats {
				if stats.TotalRuns > 0 {
					langStatsJS[lang] = map[string]interface{}{
						"success_rate": stats.SuccessRate / float64(stats.TotalRuns),
						"total_runs":   stats.TotalRuns,
					}
				}
			}
			if len(langStatsJS) > 0 {
				histEntry["languageStats"] = langStatsJS
			}
		}

		historyJS[i] = histEntry
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

		languagesMap[lang] = langData
	}

	// Update dashboard with current version data
	dashboard.Version = matrix.Version
	dashboard.Timestamp = time.Now().Format(time.RFC3339)
	dashboard.TotalRuns = matrix.TotalRuns
	dashboard.Aggregates = aggregatesJS
	dashboard.Models = modelsJS
	dashboard.Benchmarks = benchmarksJS
	dashboard.Languages = languagesMap

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

// Helper functions

func formatBenchmarkName(id string) string {
	// Convert snake_case to Title Case
	words := strings.Split(id, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func formatModelName(name string) string {
	// Shorten long model names for table display
	switch {
	case strings.Contains(name, "claude-sonnet-4-5"):
		return "Claude Sonnet 4.5"
	case strings.Contains(name, "gpt-4o-mini"):
		return "GPT-4o Mini"
	case strings.Contains(name, "gpt-4"):
		return "GPT-4"
	case strings.Contains(name, "gemini-2-5-pro"):
		return "Gemini 2.5 Pro"
	default:
		return name
	}
}
