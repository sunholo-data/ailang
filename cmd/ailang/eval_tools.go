package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/eval_analysis"
)

// runEvalCompare compares two evaluation runs
// Usage: ailang eval-compare <baseline_dir> <new_dir>
func runEvalCompare() {
	if flag.NArg() < 3 {
		fmt.Fprintf(os.Stderr, "%s: missing arguments\n", red("Error"))
		fmt.Println("Usage: ailang eval-compare <baseline_dir> <new_dir>")
		fmt.Println("")
		fmt.Println("Compare two evaluation runs and show what changed.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-compare eval_results/baselines/v0.3.0 eval_results/after_fix")
		os.Exit(1)
	}

	baselineDir := flag.Arg(1)
	newDir := flag.Arg(2)

	// Load baseline results
	fmt.Fprintf(os.Stderr, "Loading baseline from %s...\n", baselineDir)
	baseline, err := eval_analysis.LoadResults(baselineDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load baseline: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Load new results
	fmt.Fprintf(os.Stderr, "Loading new results from %s...\n", newDir)
	newResults, err := eval_analysis.LoadResults(newDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load new results: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Compare
	fmt.Fprintf(os.Stderr, "Comparing results...\n\n")
	report, err := eval_analysis.Compare(baseline, newResults, filepath.Base(baselineDir), filepath.Base(newDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to compare: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Format and print
	output := eval_analysis.FormatComparison(report, true)
	fmt.Print(output)

	// Exit with error code if there are regressions
	if report.HasRegressions() {
		os.Exit(1)
	}
}

// runEvalMatrix generates a performance matrix from results
// Usage: ailang eval-matrix <results_dir> <version>
func runEvalMatrix() {
	if flag.NArg() < 3 {
		fmt.Fprintf(os.Stderr, "%s: missing arguments\n", red("Error"))
		fmt.Println("Usage: ailang eval-matrix <results_dir> <version>")
		fmt.Println("")
		fmt.Println("Generate performance matrix with aggregated statistics.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-matrix eval_results/baselines/v0.3.0 v0.3.0-alpha5")
		os.Exit(1)
	}

	resultsDir := flag.Arg(1)
	version := flag.Arg(2)

	// Load results
	fmt.Fprintf(os.Stderr, "Loading results from %s...\n", resultsDir)
	results, err := eval_analysis.LoadResults(resultsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load results: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Generate matrix
	fmt.Fprintf(os.Stderr, "Generating performance matrix for %s...\n", version)
	matrix, err := eval_analysis.GenerateMatrix(results, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to generate matrix: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Determine output path
	matrixOutput := fmt.Sprintf("eval_results/performance_tables/%s.json", version)

	// Ensure output directory exists
	outputDir := "eval_results/performance_tables"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to create output directory: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Write JSON
	jsonData, err := eval_analysis.FormatJSON(matrix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to format matrix as JSON: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if err := os.WriteFile(matrixOutput, []byte(jsonData), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to write matrix file: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\n%s Performance matrix generated\n", green("✓"))
	fmt.Fprintf(os.Stderr, "  Version:       %s\n", version)
	fmt.Fprintf(os.Stderr, "  Total runs:    %d\n", matrix.TotalRuns)
	fmt.Fprintf(os.Stderr, "  0-shot:        %.0f%%\n", matrix.Aggregates.ZeroShotSuccess*100)
	fmt.Fprintf(os.Stderr, "  Final success: %.0f%%\n", matrix.Aggregates.FinalSuccess*100)
	fmt.Fprintf(os.Stderr, "  Total cost:    $%.4f\n", matrix.Aggregates.TotalCostUSD)
	fmt.Fprintf(os.Stderr, "\n  Output: %s\n\n", matrixOutput)

	// Pretty-print summary
	prettyOutput := eval_analysis.FormatMatrix(matrix, true)
	fmt.Print(prettyOutput)
}

// runEvalSummary generates JSONL summary from results
// Usage: ailang eval-summary <results_dir>
func runEvalSummary() {
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "%s: missing argument\n", red("Error"))
		fmt.Println("Usage: ailang eval-summary <results_dir>")
		fmt.Println("")
		fmt.Println("Convert evaluation results to JSONL format (one JSON per line).")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-summary eval_results/baselines/v0.3.0")
		fmt.Println("  ailang eval-summary results/ | jq 'select(.stdout_ok == false)'")
		os.Exit(1)
	}

	resultsDir := flag.Arg(1)

	// Load results
	fmt.Fprintf(os.Stderr, "Loading results from %s...\n", resultsDir)
	results, err := eval_analysis.LoadResults(resultsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load results: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Generate JSONL
	fmt.Fprintf(os.Stderr, "Generating JSONL summary...\n")
	jsonl, err := eval_analysis.FormatJSONL(results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to format JSONL: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Determine output path
	summaryOutput := fmt.Sprintf("%s/summary.jsonl", resultsDir)

	// Write to file
	if err := os.WriteFile(summaryOutput, []byte(jsonl), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to write summary file: %v\n", red("Error"), err)
		os.Exit(1)
	}

	lineCount := len(results)
	successCount := 0
	for _, r := range results {
		if r.StdoutOk {
			successCount++
		}
	}

	fmt.Fprintf(os.Stderr, "\n%s Generated JSONL summary\n", green("✓"))
	fmt.Fprintf(os.Stderr, "  Input:  %s (%d JSON files)\n", resultsDir, lineCount)
	fmt.Fprintf(os.Stderr, "  Output: %s (%d lines)\n", summaryOutput, lineCount)
	fmt.Fprintf(os.Stderr, "  Success rate: %d/%d (%.1f%%)\n\n",
		successCount, lineCount, float64(successCount)/float64(lineCount)*100)

	fmt.Fprintf(os.Stderr, "Example queries:\n\n")
	fmt.Fprintf(os.Stderr, "  # Count successes\n")
	fmt.Fprintf(os.Stderr, "  jq -s 'map(select(.stdout_ok == true)) | length' %s\n\n", summaryOutput)
	fmt.Fprintf(os.Stderr, "  # Average tokens by model\n")
	fmt.Fprintf(os.Stderr, "  jq -s 'group_by(.model) | map({model: .[0].model, avg: (map(.total_tokens) | add / length)})' %s\n\n", summaryOutput)
	fmt.Fprintf(os.Stderr, "  # Error distribution\n")
	fmt.Fprintf(os.Stderr, "  jq -s 'group_by(.err_code) | map({code: .[0].err_code, count: length})' %s\n\n", summaryOutput)
}

// runEvalValidate validates a specific fix against baseline
// Usage: ailang eval-validate <benchmark_id> [baseline_version]
func runEvalValidate() {
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "%s: missing argument\n", red("Error"))
		fmt.Println("Usage: ailang eval-validate <benchmark_id> [baseline_version]")
		fmt.Println("")
		fmt.Println("Validate a fix by comparing current code to baseline.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-validate float_eq")
		fmt.Println("  ailang eval-validate records_person v0.3.0-alpha5")
		os.Exit(1)
	}

	benchmarkID := flag.Arg(1)
	baselineVersion := ""
	if flag.NArg() >= 3 {
		baselineVersion = flag.Arg(2)
	}

	// Run validation
	fmt.Fprintf(os.Stderr, "Validating fix for %s...\n\n", benchmarkID)
	result, err := eval_analysis.ValidateFix(benchmarkID, baselineVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Format and print
	output := eval_analysis.FormatValidationResult(result, true)
	fmt.Print(output)

	// Exit with error code if not fixed
	if result.Outcome == eval_analysis.OutcomeBroken || result.Outcome == eval_analysis.OutcomeStillFailing {
		os.Exit(1)
	}
}

// runEvalReport generates a comprehensive evaluation report
// Usage: ailang eval-report <results_dir> <version> [--format=markdown|html|csv]
func runEvalReport() {
	if flag.NArg() < 3 {
		fmt.Fprintf(os.Stderr, "%s: missing arguments\n", red("Error"))
		fmt.Println("Usage: ailang eval-report <results_dir> <version> [--format=markdown|html|docusaurus|json|csv]")
		fmt.Println("")
		fmt.Println("Generate comprehensive evaluation report.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-report eval_results/baselines/v0.3.0 v0.3.0")
		fmt.Println("  ailang eval-report results/ v0.3.1 --format=html > report.html")
		fmt.Println("  ailang eval-report results/ v0.3.1 --format=docusaurus > docs/docs/benchmarks/performance.md")
		fmt.Println("  ailang eval-report results/ v0.3.1 --format=json > docs/static/benchmarks/latest.json")
		os.Exit(1)
	}

	resultsDir := flag.Arg(1)
	version := flag.Arg(2)
	format := "markdown" // default

	// Check for format flag
	if flag.NArg() >= 4 {
		formatArg := flag.Arg(3)
		if strings.HasPrefix(formatArg, "--format=") {
			format = strings.TrimPrefix(formatArg, "--format=")
		}
	}

	// Load results
	fmt.Fprintf(os.Stderr, "Loading results from %s...\n", resultsDir)
	results, err := eval_analysis.LoadResults(resultsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load results: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Generate matrix
	fmt.Fprintf(os.Stderr, "Generating performance matrix...\n")
	matrix, err := eval_analysis.GenerateMatrix(results, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to generate matrix: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Load historical baselines if available
	baselines, _ := eval_analysis.ListBaselines()
	var history []*eval_analysis.Baseline
	for _, v := range baselines {
		if baseline, err := eval_analysis.LoadBaselineByVersion(v); err == nil {
			history = append(history, baseline)
		}
	}

	fmt.Fprintf(os.Stderr, "Generating %s report...\n\n", format)

	var output string
	switch format {
	case "markdown", "md":
		output = eval_analysis.ExportMarkdown(matrix, history)
	case "html":
		output, err = eval_analysis.ExportHTML(matrix, history)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to generate HTML: %v\n", red("Error"), err)
			os.Exit(1)
		}
	case "docusaurus", "mdx":
		output = eval_analysis.ExportDocusaurusMDX(matrix, history)
	case "json":
		output, err = eval_analysis.ExportBenchmarkJSON(matrix, history, results)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to generate JSON: %v\n", red("Error"), err)
			os.Exit(1)
		}
	case "csv":
		output, err = eval_analysis.ExportCSV(results)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to generate CSV: %v\n", red("Error"), err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown format '%s'\n", red("Error"), format)
		fmt.Fprintf(os.Stderr, "Supported formats: markdown, html, docusaurus, json, csv\n")
		os.Exit(1)
	}

	// Print to stdout
	fmt.Print(output)
}
