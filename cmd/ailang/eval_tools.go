package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// runEvalCompare compares two evaluation runs
// Usage: ailang eval-compare <baseline_dir|--chain CHAIN_ID> <new_dir|--chain CHAIN_ID>
func runEvalCompare() {
	if flag.NArg() < 3 {
		fmt.Fprintf(os.Stderr, "%s: missing arguments\n", red("Error"))
		fmt.Println("Usage: ailang eval-compare <baseline_dir> <new_dir>")
		fmt.Println("       ailang eval-compare --chain <id1> --chain <id2>")
		fmt.Println("")
		fmt.Println("Compare two evaluation runs and show what changed.")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --chain ID   Use chain as data source (specify twice for comparison)")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-compare eval_results/baselines/v0.3.0 eval_results/after_fix")
		fmt.Println("  ailang eval-compare --chain e9c7501d --chain a1b2c3d4")
		os.Exit(1)
	}

	// Check for --chain mode
	var chainIDs []string
	for i := 1; i < flag.NArg(); i++ {
		if flag.Arg(i) == "--chain" && i+1 < flag.NArg() {
			chainIDs = append(chainIDs, flag.Arg(i+1))
		}
	}

	var baseline, newResults []*eval_analysis.BenchmarkResult
	var baselineLabel, newLabel string
	var err error

	if len(chainIDs) == 2 {
		// M-EVAL-CHAINS: Load from chains
		fmt.Fprintf(os.Stderr, "Loading baseline from chain %s...\n", chainIDs[0])
		baseline, err = eval_analysis.LoadResultsFromChain(chainIDs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to load baseline chain: %v\n", red("Error"), err)
			os.Exit(1)
		}
		baselineLabel = "chain:" + chainIDs[0][:8]

		fmt.Fprintf(os.Stderr, "Loading new results from chain %s...\n", chainIDs[1])
		newResults, err = eval_analysis.LoadResultsFromChain(chainIDs[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to load new chain: %v\n", red("Error"), err)
			os.Exit(1)
		}
		newLabel = "chain:" + chainIDs[1][:8]
	} else if len(chainIDs) == 1 {
		fmt.Fprintf(os.Stderr, "%s: --chain requires exactly two chain IDs for comparison\n", red("Error"))
		os.Exit(1)
	} else {
		// Standard filesystem mode
		baselineDir := flag.Arg(1)
		newDir := flag.Arg(2)

		fmt.Fprintf(os.Stderr, "Loading baseline from %s...\n", baselineDir)
		baseline, err = eval_analysis.LoadResults(baselineDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to load baseline: %v\n", red("Error"), err)
			os.Exit(1)
		}
		baselineLabel = filepath.Base(baselineDir)

		fmt.Fprintf(os.Stderr, "Loading new results from %s...\n", newDir)
		newResults, err = eval_analysis.LoadResults(newDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to load new results: %v\n", red("Error"), err)
			os.Exit(1)
		}
		newLabel = filepath.Base(newDir)
	}

	// Compare
	fmt.Fprintf(os.Stderr, "Comparing results...\n\n")
	report, err := eval_analysis.Compare(baseline, newResults, baselineLabel, newLabel)
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
// Usage: ailang eval-matrix <results_dir> <version> [--by-tags] [--show-saturated] [--ailang-wins]
func runEvalMatrix() {
	if flag.NArg() < 3 {
		fmt.Fprintf(os.Stderr, "%s: missing arguments\n", red("Error"))
		fmt.Println("Usage: ailang eval-matrix <results_dir> <version> [--by-tags] [--show-saturated] [--ailang-wins] [--by-harness] [--group-by=model-family]")
		fmt.Println("")
		fmt.Println("Generate performance matrix with aggregated statistics.")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --by-tags                Append per-tag AILANG vs Python delta table")
		fmt.Println("  --show-saturated         Append list of benchmarks at 100% pass across all models + languages")
		fmt.Println("  --ailang-wins            Append list of (benchmark × model) cells where AILANG passes and Python fails")
		fmt.Println("  --by-harness             Append language × model × harness breakdown table")
		fmt.Println("  --group-by=model-family  Append cross-harness comparison grouped by model family (requires model_family in results)")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-matrix eval_results/baselines/v0.3.0 v0.3.0-alpha5")
		fmt.Println("  ailang eval-matrix eval_results/baselines/v0.13.0 v0.13.0 --by-tags --show-saturated")
		fmt.Println("  ailang eval-matrix eval_results/agent v0.14.1-harness --by-harness")
		fmt.Println("  ailang eval-matrix eval_results/baselines/v0.14.0 v0.14.0 --group-by=model-family")
		os.Exit(1)
	}

	resultsDir := flag.Arg(1)
	version := flag.Arg(2)

	// M-EVAL-SUITE-PREP M3: parse report-section flags. Positional args are
	// [1]=resultsDir, [2]=version; flags live from index 3 onward.
	byTags, showSaturated, ailangWins, byHarness := parseMatrixFlags(flag.Arg, flag.NArg())
	groupBy := parseGroupByFlag(flag.Arg, flag.NArg())

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

	// M-EVAL-SUITE-PREP M3: optional report sections.
	if byTags {
		printTagsSection(results, evalBenchmarkDir)
	}
	if showSaturated {
		printSaturatedSection(results)
	}
	if ailangWins {
		printAILANGWinsSection(results)
	}
	if byHarness {
		printByHarnessSection(results)
	}
	if groupBy == "model-family" {
		printGroupedByFamilySection(results)
	}
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

// runEvalReport generates a comprehensive evaluation report
// Usage: ailang eval-report <results_dir|--multi-model|--from-chain> <version> [--format=markdown|html|csv]
func runEvalReport() {
	if flag.NArg() < 3 {
		fmt.Fprintf(os.Stderr, "%s: missing arguments\n", red("Error"))
		fmt.Println("Usage: ailang eval-report <results_dir|--multi-model|--from-chain CHAIN_ID|--from-latest-chain> <version> [--format=markdown|html|docusaurus|json|csv]")
		fmt.Println("")
		fmt.Println("Generate comprehensive evaluation report.")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --multi-model         Aggregate latest results per model from all baselines")
		fmt.Println("  --from-chain ID       Load results from a specific eval chain")
		fmt.Println("  --from-latest-chain   Load results from the most recent eval chain")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-report eval_results/baselines/v0.3.0 v0.3.0")
		fmt.Println("  ailang eval-report --multi-model v0.3.5 --format=docusaurus")
		fmt.Println("  ailang eval-report --from-chain e9c7501d v0.8.1 --format=json")
		fmt.Println("  ailang eval-report --from-latest-chain v0.8.1 --format=json")
		os.Exit(1)
	}

	resultsDir := flag.Arg(1)
	version := flag.Arg(2)
	format := "markdown" // default
	multiModel := false
	fromChain := ""
	fromLatestChain := false

	// Check if using special modes
	if resultsDir == "--multi-model" {
		multiModel = true
	} else if resultsDir == "--from-latest-chain" {
		fromLatestChain = true
	} else if resultsDir == "--from-chain" {
		// --from-chain <id> <version> shifts args
		if flag.NArg() < 4 {
			fmt.Fprintf(os.Stderr, "%s: --from-chain requires chain ID and version\n", red("Error"))
			os.Exit(1)
		}
		fromChain = flag.Arg(2)
		version = flag.Arg(3)
	}

	// Check for format flag in remaining args
	for i := 1; i < flag.NArg(); i++ {
		arg := flag.Arg(i)
		if strings.HasPrefix(arg, "--format=") {
			format = strings.TrimPrefix(arg, "--format=")
		}
	}

	// Load models config so harness/provider_type lookups work in JSON export.
	// Non-fatal: export degrades gracefully when config unavailable.
	_ = eval_harness.InitModelsConfig()

	// Load results
	var results []*eval_analysis.BenchmarkResult
	var modelBaselines map[string]string
	var err error

	if fromChain != "" {
		// M-EVAL-CHAINS: Load from specific chain
		fmt.Fprintf(os.Stderr, "Loading results from chain %s...\n", fromChain)
		results, err = eval_analysis.LoadResultsFromChain(fromChain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to load results from chain: %v\n", red("Error"), err)
			os.Exit(1)
		}
	} else if fromLatestChain {
		// M-EVAL-CHAINS: Load from latest eval chain
		fmt.Fprintf(os.Stderr, "Loading results from latest eval chain...\n")
		var chainID string
		results, chainID, err = eval_analysis.LoadResultsFromLatestEvalChain()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to load results from latest chain: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "  Chain: %s\n", chainID[:8])
	} else if multiModel {
		fmt.Fprintf(os.Stderr, "Aggregating latest results per model from all baselines...\n")
		results, modelBaselines, err = eval_analysis.LoadLatestResultsPerModel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to load multi-model results: %v\n", red("Error"), err)
			os.Exit(1)
		}

		// Report which baselines were used per model
		fmt.Fprintf(os.Stderr, "Model sources:\n")
		for model, baseline := range modelBaselines {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", model, baseline)
		}
		fmt.Fprintf(os.Stderr, "\n")
	} else {
		fmt.Fprintf(os.Stderr, "Loading results from %s...\n", resultsDir)
		results, err = eval_analysis.LoadResults(resultsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to load results: %v\n", red("Error"), err)
			os.Exit(1)
		}
	}

	// Filter to only standard results for matrix/history (agent results tracked separately)
	// Exception: when loading from chain, include all results (chains are agent-only)
	standardResults := make([]*eval_analysis.BenchmarkResult, 0, len(results))
	if fromChain != "" || fromLatestChain {
		standardResults = results // Chain mode: all results included
	} else {
		for _, r := range results {
			if r.EvalMode != "agent" {
				standardResults = append(standardResults, r)
			}
		}
		// All-agent baseline: fall back to using agent results for matrix so
		// agent-only baselines (e.g. harness comparison runs) produce a valid report.
		if len(standardResults) == 0 {
			standardResults = results
		}
	}

	// Generate matrix (using standard results, or agent results when no standard exist)
	fmt.Fprintf(os.Stderr, "Generating performance matrix...\n")
	var matrix *eval_analysis.PerformanceMatrix
	if multiModel {
		matrix, err = eval_analysis.GenerateMatrixWithBaselines(standardResults, version, modelBaselines)
	} else {
		matrix, err = eval_analysis.GenerateMatrix(standardResults, version)
	}
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
		// Default output path for JSON (can be overridden by user redirecting stdout)
		jsonOutputPath := "docs/static/benchmarks/latest.json"
		output, err = eval_analysis.ExportBenchmarkJSON(matrix, history, results, jsonOutputPath)
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

// runEvalSweetSpot generates a sweet-spot report from a results directory
// (M-EVAL-SWEET-SPOT, v0.19.0).
//
// Usage: ailang eval-sweet-spot <results_dir> [--format=text|csv|json] [--slow-ms=N]
func runEvalSweetSpot() {
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "%s: missing results directory\n", red("Error"))
		fmt.Println("Usage: ailang eval-sweet-spot <results_dir> [--format=text|csv|json] [--slow-ms=N]")
		fmt.Println("")
		fmt.Println("Show per-model cost-vs-time-vs-success sweet-spot ranking.")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --format=text|csv|json   Output format (default: text)")
		fmt.Println("  --slow-ms=N              Slow-pass threshold in ms (default: 60000)")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-sweet-spot eval_results/standard")
		fmt.Println("  ailang eval-sweet-spot eval_results/v0_18_5_core_3harness --format=csv > sweet.csv")
		fmt.Println("  ailang eval-sweet-spot eval_results/standard --slow-ms=30000")
		os.Exit(1)
	}

	resultsDir := flag.Arg(1)

	// Parse flags from positional args 2+.
	format := "text"
	slowMs := int64(60_000)
	for i := 2; i < flag.NArg(); i++ {
		arg := flag.Arg(i)
		switch {
		case strings.HasPrefix(arg, "--format="):
			format = strings.TrimPrefix(arg, "--format=")
		case strings.HasPrefix(arg, "--slow-ms="):
			n, err := strconv.ParseInt(strings.TrimPrefix(arg, "--slow-ms="), 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: invalid --slow-ms value: %v\n", red("Error"), err)
				os.Exit(1)
			}
			slowMs = n
		}
	}

	results, err := eval_analysis.LoadResults(resultsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load results: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "%s: no results found in %s\n", yellow("Warning"), resultsDir)
		os.Exit(0)
	}

	report := eval_analysis.BuildSweetSpot(results, eval_analysis.SweetSpotOpts{SlowMs: slowMs})

	var out string
	switch format {
	case "text":
		out = eval_analysis.FormatSweetSpotText(report, isTerminal())
	case "csv":
		out, err = eval_analysis.FormatSweetSpotCSV(report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: CSV format failed: %v\n", red("Error"), err)
			os.Exit(1)
		}
	case "json":
		out, err = eval_analysis.FormatSweetSpotJSON(report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: JSON format failed: %v\n", red("Error"), err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown format '%s'\n", red("Error"), format)
		fmt.Fprintf(os.Stderr, "Supported formats: text, csv, json\n")
		os.Exit(1)
	}

	fmt.Print(out)
}
