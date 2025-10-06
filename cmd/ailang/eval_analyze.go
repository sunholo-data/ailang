package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/eval_analyzer"
)

func runEvalAnalyze() {
	// Parse eval-analyze subcommand flags
	fs := flag.NewFlagSet("eval-analyze", flag.ExitOnError)
	resultsDir := fs.String("results", "eval_results", "Directory containing eval result JSON files")
	outputDir := fs.String("output", "design_docs/planned", "Output directory for design documents")
	model := fs.String("model", "gpt5", "LLM model to use for design generation (gpt5, claude-sonnet-4-5)")
	seed := fs.Int64("seed", 42, "Random seed for deterministic generation")
	minFrequency := fs.Int("min-frequency", 2, "Minimum failure frequency to generate design doc")
	categories := fs.String("categories", "", "Comma-separated list of categories to analyze (empty = all)")
	dryRun := fs.Bool("dry-run", false, "Show issues without generating design docs")
	generateDesigns := fs.Bool("generate", true, "Generate design documents (set to false to only analyze)")
	forceNew := fs.Bool("force-new", false, "Always create new docs (disable deduplication)")
	mergeThreshold := fs.Float64("merge-threshold", 0.75, "Similarity threshold for merging (0.0-1.0)")
	skipWellDocumented := fs.Bool("skip-documented", false, "Skip generation if issue is already well-documented")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Analyzing eval results from %s...\n", cyan("→"), *resultsDir)

	// Parse categories
	var catList []string
	if *categories != "" {
		catList = strings.Split(*categories, ",")
		for i := range catList {
			catList[i] = strings.TrimSpace(catList[i])
		}
	}

	// Create analyzer
	analyzer := eval_analyzer.NewAnalyzer(*resultsDir, *minFrequency, catList)

	// Analyze results
	analysis, err := analyzer.Analyze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: analysis failed: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("\n%s Analysis Summary\n", cyan("━━━"))
	fmt.Printf("  Total Runs: %d\n", analysis.TotalRuns)
	fmt.Printf("  Failures: %d\n", analysis.FailureCount)
	fmt.Printf("  Success Rate: %.1f%%\n", analysis.SuccessRate)
	fmt.Printf("  Issues Found: %d\n", len(analysis.Issues))
	fmt.Println()

	if len(analysis.Issues) == 0 {
		fmt.Printf("%s No issues found meeting frequency threshold (%d)\n", green("✓"), *minFrequency)
		return
	}

	// Print issues
	fmt.Printf("%s Issues Discovered:\n", cyan("→"))
	for i, issue := range analysis.Issues {
		fmt.Printf("\n%d. %s [%s]\n", i+1, bold(issue.Title), issue.Impact)
		fmt.Printf("   Category: %s\n", issue.Category)
		fmt.Printf("   Frequency: %d failures\n", issue.Frequency)
		fmt.Printf("   Benchmarks: %s\n", strings.Join(issue.Benchmarks, ", "))
		fmt.Printf("   Language: %s\n", issue.Lang)
		fmt.Printf("   Models: %s\n", strings.Join(issue.Models, ", "))

		if len(issue.ErrorMessages) > 0 {
			fmt.Printf("   Sample Error: %s\n", truncateOutput(issue.ErrorMessages[0]))
		}
	}
	fmt.Println()

	// Dry run - stop here
	if *dryRun {
		fmt.Printf("%s Dry run mode - no design docs generated\n", yellow("⚠"))
		return
	}

	// Generate design docs
	if !*generateDesigns {
		fmt.Printf("%s Design generation disabled (--generate=false)\n", yellow("→"))
		return
	}

	fmt.Printf("%s Generating design documents with %s...\n", cyan("→"), *model)

	// Create design generator
	generator, err := eval_analyzer.NewDesignGenerator(*model, *seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to create design generator: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to create output directory: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Generate design doc for each issue (with deduplication)
	ctx := context.Background()
	generatedDocs := []string{}
	updatedDocs := []string{}
	skippedDocs := []string{}

	// Configure deduplication
	dedupConfig := eval_analyzer.DedupConfig{
		Enabled:            !*forceNew,
		MergeThreshold:     *mergeThreshold,
		ForceNew:           *forceNew,
		SkipWellDocumented: *skipWellDocumented,
	}

	for i, issue := range analysis.Issues {
		fmt.Printf("\n%s [%d/%d] Processing issue: %s\n",
			cyan("→"), i+1, len(analysis.Issues), issue.Title)

		// Check for similar existing docs
		similar, err := eval_analyzer.FindSimilarDesignDocs(issue, *outputDir, dedupConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to check for similar docs: %v\n", yellow("⚠"), err)
		}

		// Determine strategy
		strategy, bestMatch := eval_analyzer.DetermineMergeStrategy(issue, similar, dedupConfig)

		fmt.Printf("  Similar docs: %d found\n", len(similar))
		if bestMatch != nil {
			fmt.Printf("  Best match: %s (%.1f%% similar)\n", bestMatch.Filename, bestMatch.SimilarityScore*100)
		}
		fmt.Printf("  Strategy: %s\n", string(strategy))

		switch strategy {
		case eval_analyzer.StrategyCreate:
			// Generate new design doc
			designDoc, err := generator.Generate(ctx, issue, analysis.FailureCount)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to generate design doc: %v\n", red("✗"), err)
				continue
			}

			// Generate filename from issue title
			filename := generateFilename(issue.Title, issue.Category)
			filepath := filepath.Join(*outputDir, filename)

			// Write design doc
			if err := os.WriteFile(filepath, []byte(designDoc), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to write design doc: %v\n", red("✗"), err)
				continue
			}

			fmt.Printf("  %s Created: %s\n", green("✓"), filepath)
			generatedDocs = append(generatedDocs, filepath)

		case eval_analyzer.StrategyMerge:
			// Merge new evidence into existing doc
			if err := eval_analyzer.MergeDesignDoc(bestMatch.Path, issue, analysis.FailureCount); err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to merge design doc: %v\n", red("✗"), err)
				continue
			}

			fmt.Printf("  %s Updated: %s\n", green("✓"), bestMatch.Filename)
			fmt.Printf("     Added %d new failures, %d new benchmarks\n", issue.Frequency, len(issue.Benchmarks))
			updatedDocs = append(updatedDocs, bestMatch.Path)

		case eval_analyzer.StrategySkip:
			// Skip - already well-documented
			fmt.Printf("  %s Skipped: %s (already well-documented)\n", yellow("→"), bestMatch.Filename)
			skippedDocs = append(skippedDocs, bestMatch.Path)

		case eval_analyzer.StrategyLink:
			// Create new doc but reference related doc
			designDoc, err := generator.Generate(ctx, issue, analysis.FailureCount)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to generate design doc: %v\n", red("✗"), err)
				continue
			}

			// Add reference to related doc
			relatedNote := fmt.Sprintf("\n\n## Related Issues\n\nSee also: [%s](%s) (%.1f%% similar)\n",
				bestMatch.Filename, bestMatch.Filename, bestMatch.SimilarityScore*100)
			designDoc += relatedNote

			filename := generateFilename(issue.Title, issue.Category)
			filepath := filepath.Join(*outputDir, filename)

			if err := os.WriteFile(filepath, []byte(designDoc), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to write design doc: %v\n", red("✗"), err)
				continue
			}

			fmt.Printf("  %s Created: %s (linked to %s)\n", green("✓"), filepath, bestMatch.Filename)
			generatedDocs = append(generatedDocs, filepath)
		}

		// Rate limiting between API calls (only for CREATE operations)
		if strategy == eval_analyzer.StrategyCreate || strategy == eval_analyzer.StrategyLink {
			if i < len(analysis.Issues)-1 {
				time.Sleep(2 * time.Second)
			}
		}
	}

	// Generate summary report
	summaryPath := filepath.Join(*outputDir, fmt.Sprintf("EVAL_ANALYSIS_%s.md", time.Now().Format("20060102")))
	summary := generateSummaryReport(analysis, generatedDocs, *model)

	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to write summary: %v\n", yellow("⚠"), err)
	} else {
		fmt.Printf("\n%s Summary report: %s\n", green("✓"), summaryPath)
	}

	// Save analysis as JSON for further processing
	analysisPath := filepath.Join(*resultsDir, fmt.Sprintf("analysis_%s.json", time.Now().Format("20060102_150405")))
	analysisJSON, _ := json.MarshalIndent(analysis, "", "  ")
	if err := os.WriteFile(analysisPath, analysisJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to write analysis JSON: %v\n", yellow("⚠"), err)
	}

	fmt.Printf("\n%s Analysis complete!\n", green("✓"))
	fmt.Printf("  Design docs: %d created, %d updated, %d skipped\n", len(generatedDocs), len(updatedDocs), len(skippedDocs))
	fmt.Printf("  Summary: %s\n", summaryPath)
	fmt.Printf("  Analysis data: %s\n", analysisPath)

	if len(updatedDocs) > 0 {
		fmt.Printf("\n  Updated docs:\n")
		for _, doc := range updatedDocs {
			fmt.Printf("    - %s\n", filepath.Base(doc))
		}
	}

	if len(skippedDocs) > 0 {
		fmt.Printf("\n  Skipped docs (already well-documented):\n")
		for _, doc := range skippedDocs {
			fmt.Printf("    - %s\n", filepath.Base(doc))
		}
	}
}

// generateFilename creates a safe filename from issue title and category
func generateFilename(title, category string) string {
	// Convert to lowercase, replace spaces with underscores
	name := strings.ToLower(title)
	name = strings.ReplaceAll(name, " ", "_")

	// Remove special characters
	safe := strings.Builder{}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			safe.WriteRune(r)
		}
	}

	// Add date prefix and .md extension
	date := time.Now().Format("20060102")
	filename := fmt.Sprintf("%s_%s_%s.md", date, category, safe.String())

	// Truncate if too long
	if len(filename) > 100 {
		filename = filename[:100] + ".md"
	}

	return filename
}

// generateSummaryReport creates a summary markdown report
func generateSummaryReport(analysis *eval_analyzer.AnalysisResult, docs []string, model string) string {
	var buf strings.Builder

	buf.WriteString("# AI Eval Analysis Summary\n\n")
	buf.WriteString(fmt.Sprintf("**Generated**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	buf.WriteString(fmt.Sprintf("**Model**: %s\n\n", model))

	buf.WriteString("## Overview\n\n")
	buf.WriteString(fmt.Sprintf("- **Total Runs**: %d\n", analysis.TotalRuns))
	buf.WriteString(fmt.Sprintf("- **Failures**: %d\n", analysis.FailureCount))
	buf.WriteString(fmt.Sprintf("- **Success Rate**: %.1f%%\n", analysis.SuccessRate))
	buf.WriteString(fmt.Sprintf("- **Issues Identified**: %d\n\n", len(analysis.Issues)))

	buf.WriteString("## Issues by Impact\n\n")

	// Group by impact
	byImpact := make(map[string][]eval_analyzer.IssueReport)
	for _, issue := range analysis.Issues {
		byImpact[issue.Impact] = append(byImpact[issue.Impact], issue)
	}

	for _, impact := range []string{"critical", "high", "medium", "low"} {
		issues := byImpact[impact]
		if len(issues) == 0 {
			continue
		}

		buf.WriteString(fmt.Sprintf("### %s (%d)\n\n", strings.Title(impact), len(issues)))
		for _, issue := range issues {
			buf.WriteString(fmt.Sprintf("- **%s** (%s, %d failures)\n",
				issue.Title, issue.Lang, issue.Frequency))
			buf.WriteString(fmt.Sprintf("  - Benchmarks: %s\n", strings.Join(issue.Benchmarks, ", ")))
		}
		buf.WriteString("\n")
	}

	buf.WriteString("## Generated Design Documents\n\n")
	for _, doc := range docs {
		buf.WriteString(fmt.Sprintf("- [%s](%s)\n", filepath.Base(doc), doc))
	}
	buf.WriteString("\n")

	buf.WriteString("## Next Steps\n\n")
	buf.WriteString("1. Review generated design documents\n")
	buf.WriteString("2. Adjust priorities and estimates as needed\n")
	buf.WriteString("3. Move approved designs to milestone tracking\n")
	buf.WriteString("4. Create implementation branches\n")
	buf.WriteString("5. Re-run eval suite after fixes to measure improvement\n\n")

	buf.WriteString("---\n\n")
	buf.WriteString("*Generated by `ailang eval-analyze`*\n")

	return buf.String()
}
