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

	// Import React component
	sb.WriteString("import BenchmarkDashboard from '@site/src/components/BenchmarkDashboard';\n\n")

	// Hero section
	sb.WriteString("# AI Code Generation Benchmarks\n\n")
	sb.WriteString("Real-world performance metrics for AILANG vs Python across multiple AI models.\n\n")

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
		sb.WriteString("| Model | Runs | 0-Shot | Final | Avg Tokens | Cost/Run |\n")
		sb.WriteString("|-------|------|--------|-------|------------|---------|\n")

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

			sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %.1f%% | %.0f | $%.4f |\n",
				formatModelName(m.name),
				m.stats.TotalRuns,
				m.stats.Aggregates.ZeroShotSuccess*100,
				m.stats.Aggregates.FinalSuccess*100,
				avgTokens,
				avgCost))
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
func ExportBenchmarkJSON(matrix *PerformanceMatrix, history []*Baseline, results []*BenchmarkResult) (string, error) {
	// Convert aggregates to camelCase for JavaScript
	aggregatesJS := map[string]interface{}{
		"zeroShotSuccess":   matrix.Aggregates.ZeroShotSuccess,
		"finalSuccess":      matrix.Aggregates.FinalSuccess,
		"repairUsed":        matrix.Aggregates.RepairUsed,
		"repairSuccessRate": matrix.Aggregates.RepairSuccessRate,
		"totalTokens":       matrix.Aggregates.TotalTokens,
		"totalCostUSD":      matrix.Aggregates.TotalCostUSD,
		"avgDurationMs":     matrix.Aggregates.AvgDurationMs,
	}

	// Group results by benchmark ID and language for code samples and stats
	codeSamples := make(map[string]map[string]string)         // benchmarkID -> language -> code
	langStats := make(map[string]map[string]*LanguageStats)   // benchmarkID -> language -> stats

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
			stats.SuccessRate = float64(int(stats.SuccessRate*float64(stats.TotalRuns-1)) + 1) / float64(stats.TotalRuns)
		} else {
			stats.SuccessRate = float64(int(stats.SuccessRate*float64(stats.TotalRuns-1))) / float64(stats.TotalRuns)
		}
		// Use output tokens (not total)
		stats.AvgTokens = (stats.AvgTokens*float64(stats.TotalRuns-1) + float64(r.OutputTokens)) / float64(stats.TotalRuns)
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
		benchmarksJS[id] = benchmark
	}

	// Convert models to camelCase for JavaScript (nested aggregates)
	modelsJS := make(map[string]interface{})
	for name, stats := range matrix.Models {
		modelsJS[name] = map[string]interface{}{
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
	}

	// Transform history to include calculated success rates
	historyJS := make([]map[string]interface{}, len(history))
	for i, baseline := range history {
		successRate := 0.0
		if baseline.TotalBenchmarks > 0 {
			successRate = float64(baseline.SuccessCount) / float64(baseline.TotalBenchmarks)
		}
		historyJS[i] = map[string]interface{}{
			"version":       baseline.Version,
			"timestamp":     baseline.Timestamp.Format(time.RFC3339),
			"successRate":   successRate,
			"totalRuns":     baseline.TotalBenchmarks,
			"successCount":  baseline.SuccessCount,
			"languages":     baseline.Languages, // May be "ailang", "python", or "ailang,python"
		}
	}

	data := map[string]interface{}{
		"version":    matrix.Version,
		"timestamp":  time.Now().Format(time.RFC3339),
		"totalRuns":  matrix.TotalRuns,
		"aggregates": aggregatesJS,
		"models":     modelsJS,
		"benchmarks": benchmarksJS,
		"languages":  matrix.Languages,
		"history":    historyJS,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
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
