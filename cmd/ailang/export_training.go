package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	ailtrace "github.com/sunholo-data/ailang/internal/trace"
)

func exportTraining() {
	fs := flag.NewFlagSet("export-training", flag.ExitOnError)
	minScore := fs.Float64("min-score", 0.0, "Minimum quality score (0.0-1.0)")
	outputFile := fs.String("output", "", "Output file (default: stdout)")
	sourceDir := fs.String("source-dir", "", "Directory to resolve source files from")
	scoreOnly := fs.Bool("score", false, "Only score traces, don't export")
	jsonOutput := fs.Bool("json", false, "Output scores as JSON (with --score)")
	quiet := fs.Bool("quiet", false, "Suppress progress messages")
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ailang export-training [flags] <traces...>")
		fmt.Fprintln(os.Stderr, "\nExport high-quality execution traces as training data for AI fine-tuning.")
		fmt.Fprintln(os.Stderr, "\nArguments:")
		fmt.Fprintln(os.Stderr, "  <traces...>   JSONL trace files or directories containing .jsonl files")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  ailang export-training traces/                          # Export all traces")
		fmt.Fprintln(os.Stderr, "  ailang export-training --min-score 0.7 traces/          # Filter low quality")
		fmt.Fprintln(os.Stderr, "  ailang export-training --score trace.jsonl               # Score a trace")
		fmt.Fprintln(os.Stderr, "  ailang export-training --score --json traces/            # Score all as JSON")
		os.Exit(2)
	}

	// Collect trace files from arguments
	traceFiles := collectTraceFiles(fs.Args())
	if len(traceFiles) == 0 {
		fmt.Fprintf(os.Stderr, "%s: no .jsonl trace files found\n", red("Error"))
		os.Exit(2)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Found %d trace file(s)\n", len(traceFiles))
	}

	// Score-only mode
	if *scoreOnly {
		runScoreMode(traceFiles, *jsonOutput, *quiet)
		return
	}

	// Export mode
	runExportMode(traceFiles, *minScore, *sourceDir, *outputFile, *quiet)
}

// runScoreMode scores each trace file and displays results.
func runScoreMode(files []string, jsonOut, quiet bool) {
	type scoreResult struct {
		File  string              `json:"file"`
		Score ailtrace.TraceScore `json:"score"`
	}

	var results []scoreResult
	for _, path := range files {
		score, err := ailtrace.ScoreTraceFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s %s: %v\n", red("✗"), path, err)
			continue
		}
		results = append(results, scoreResult{File: path, Score: score})
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(results)
		return
	}

	// Human-readable output
	for _, r := range results {
		s := r.Score
		fmt.Fprintf(os.Stderr, "\n%s  Score: %.2f\n", filepath.Base(r.File), s.Total)
		fmt.Fprintf(os.Stderr, "  Completion:  %.2f  Complexity:  %.2f\n", s.Completion, s.Complexity)
		fmt.Fprintf(os.Stderr, "  Contracts:   %.2f  Budget:      %.2f\n", s.ContractCoverage, s.BudgetEfficiency)
		fmt.Fprintf(os.Stderr, "  Effects:     %.2f\n", s.EffectDiversity)
		fmt.Fprintf(os.Stderr, "  Stats: %d events, %d functions, %d effects, depth %d\n",
			s.Stats.TotalEvents, s.Stats.DistinctFunctions, s.Stats.DistinctEffects, s.Stats.MaxDepth)
	}
}

// runExportMode exports qualifying traces as training data.
func runExportMode(files []string, minScore float64, sourceDir, outputFile string, quiet bool) {
	// Determine output writer
	out := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(2)
		}
		defer f.Close()
		out = f
	}

	cfg := ailtrace.ExportConfig{
		MinScore:  minScore,
		SourceDir: sourceDir,
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "Exporting with min score %.2f...\n", minScore)
	}

	exported, skipped, err := ailtrace.ExportTrainingData(out, files, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: export failed: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "\n%s Exported %d training examples (%d skipped)\n",
			green("✓"), exported, skipped)
		if outputFile != "" {
			fmt.Fprintf(os.Stderr, "Output: %s\n", outputFile)
		}
	}
}

// collectTraceFiles expands directories and globs into .jsonl file paths.
func collectTraceFiles(args []string) []string {
	var files []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			// Try as-is (might be a specific file path)
			if filepath.Ext(arg) == ".jsonl" {
				files = append(files, arg)
			}
			continue
		}

		if info.IsDir() {
			// Walk directory for .jsonl files
			filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && filepath.Ext(path) == ".jsonl" {
					files = append(files, path)
				}
				return nil
			})
		} else if filepath.Ext(arg) == ".jsonl" {
			files = append(files, arg)
		}
	}
	return files
}
