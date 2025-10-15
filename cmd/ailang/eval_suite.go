package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/eval_harness"
)

// SuiteResult captures the result of a single benchmark run in the suite
type SuiteResult struct {
	BenchmarkID string
	Language    string
	Model       string
	Success     bool
	Error       error
}

// discoverBenchmarks finds all .yml files in benchmarks/ directory
func discoverBenchmarks() []string {
	benchmarksDir := "benchmarks"
	entries, err := os.ReadDir(benchmarksDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not read benchmarks directory: %v\n", err)
		return nil
	}

	var benchmarks []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".yaml") {
			// Remove extension to get benchmark ID
			name := strings.TrimSuffix(entry.Name(), ".yml")
			name = strings.TrimSuffix(name, ".yaml")
			benchmarks = append(benchmarks, name)
		}
	}
	return benchmarks
}

func runEvalSuite() {
	// Parse eval-suite subcommand flags
	fs := flag.NewFlagSet("eval-suite", flag.ExitOnError)
	models := fs.String("models", "", "Comma-separated list of models (default: dev models)")
	fullSuite := fs.Bool("full", false, "Run full benchmark suite with all expensive models (gpt5, claude-sonnet-4-5, gemini-2-5-pro)")
	benchmarks := fs.String("benchmarks", "", "Comma-separated list of benchmarks (empty = auto-discover from benchmarks/)")
	langs := fs.String("langs", "python,ailang", "Comma-separated list of languages")
	seed := fs.Int64("seed", 42, "Random seed for deterministic runs")
	outputDir := fs.String("output", "eval_results", "Output directory for results")
	timeout := fs.Duration("timeout", 30*time.Second, "Timeout for code execution")
	maxConcurrent := fs.Int("parallel", 5, "Maximum concurrent API calls (0 = sequential)")
	selfRepair := fs.Bool("self-repair", false, "Enable single-shot self-repair on errors")
	promptVersion := fs.String("prompt-version", "", "Prompt version ID for all benchmarks")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Determine model list
	var modelList []string
	if *models != "" {
		// User specified models explicitly
		modelList = strings.Split(*models, ",")
	} else if *fullSuite {
		// Full suite: use expensive/comprehensive models
		modelList = []string{"gpt5", "claude-sonnet-4-5", "gemini-2-5-pro"}
	} else {
		// Default: use cheaper/faster dev models
		modelList = []string{"gpt5-mini", "gemini-2-5-flash"}
	}
	var benchmarkList []string
	if *benchmarks == "" {
		// Auto-discover benchmarks from benchmarks/ directory
		benchmarkList = discoverBenchmarks()
		if len(benchmarkList) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No benchmarks found in benchmarks/ directory\n")
			os.Exit(1)
		}
	} else {
		benchmarkList = strings.Split(*benchmarks, ",")
	}
	langList := strings.Split(*langs, ",")

	// Validate models and benchmarks
	for i := range modelList {
		modelList[i] = strings.TrimSpace(modelList[i])
	}
	for i := range benchmarkList {
		benchmarkList[i] = strings.TrimSpace(benchmarkList[i])
	}
	for i := range langList {
		langList[i] = strings.TrimSpace(langList[i])
	}

	// Calculate total runs
	totalRuns := len(modelList) * len(benchmarkList) * len(langList)

	fmt.Printf("%s AILANG Benchmark Suite\n", cyan("🚀"))
	fmt.Println("==========================")
	fmt.Println()
	fmt.Printf("Models:     %v\n", modelList)
	fmt.Printf("Benchmarks: %v\n", benchmarkList)
	fmt.Printf("Languages:  %v\n", langList)
	fmt.Printf("Seed:       %d\n", *seed)
	fmt.Printf("Parallel:   %d concurrent\n", *maxConcurrent)
	fmt.Printf("Total runs: %d\n", totalRuns)
	fmt.Println()

	// Check API keys
	checkAPIKeys(modelList)

	// Clean previous results
	fmt.Printf("%s Cleaning previous results...\n", cyan("→"))
	cleanResults(*outputDir)

	// Build job list
	var jobs []Job
	for _, model := range modelList {
		for _, benchmark := range benchmarkList {
			for _, lang := range langList {
				jobs = append(jobs, Job{
					Model:     model,
					Benchmark: benchmark,
					Language:  lang,
				})
			}
		}
	}

	// Run benchmarks with concurrency control
	startTime := time.Now()
	results := runBenchmarksParallel(jobs, *seed, *outputDir, *timeout, *maxConcurrent, *selfRepair, *promptVersion)
	duration := time.Since(startTime)

	// Summary
	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Println()
	fmt.Printf("%s Benchmark suite complete!\n", green("✓"))
	fmt.Printf("Duration: %s\n", duration.Round(time.Second))
	fmt.Printf("Success: %d/%d (%.1f%%)\n", successCount, totalRuns, float64(successCount)/float64(totalRuns)*100)
	fmt.Printf("Failed:  %d/%d\n", failCount, totalRuns)
	fmt.Println()
	fmt.Println("Results:")
	fmt.Printf("  - JSON: %s/*.json\n", *outputDir)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  ailang eval-summary %s\n", *outputDir)
	fmt.Printf("  ailang eval-matrix %s v0.3.0\n", *outputDir)
}

// Job represents a single benchmark task
type Job struct {
	Model     string
	Benchmark string
	Language  string
}

// runBenchmarksParallel executes benchmarks with concurrency control
func runBenchmarksParallel(jobs []Job, seed int64, outputDir string, timeout time.Duration, maxConcurrent int, selfRepair bool, promptVersion string) []SuiteResult {

	if maxConcurrent <= 0 {
		maxConcurrent = 1 // Sequential
	}

	var (
		wg      sync.WaitGroup
		results = make([]SuiteResult, len(jobs))
		sem     = make(chan struct{}, maxConcurrent) // Semaphore for concurrency control
		mu      sync.Mutex                           // Protect progress counter
	)

	completed := 0
	totalJobs := len(jobs)

	for i, job := range jobs {
		wg.Add(1)
		go func(idx int, j Job) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Update progress
			mu.Lock()
			completed++
			currentProgress := completed
			mu.Unlock()

			fmt.Printf("[%d/%d] Running %s with %s (%s)...\n",
				currentProgress, totalJobs,
				cyan(j.Benchmark), green(j.Model), j.Language)

			// Run the benchmark
			success, err := runSingleBenchmark(j.Model, j.Benchmark, j.Language, seed, outputDir, timeout, selfRepair, promptVersion)

			results[idx] = SuiteResult{
				BenchmarkID: j.Benchmark,
				Language:    j.Language,
				Model:       j.Model,
				Success:     success,
				Error:       err,
			}

			if success {
				fmt.Printf("  %s Completed\n", green("✓"))
			} else {
				fmt.Printf("  %s Failed: %v\n", red("✗"), err)
			}
		}(i, job)
	}

	wg.Wait()
	return results
}

// runSingleBenchmark executes a single benchmark configuration
func runSingleBenchmark(model, benchmarkID, lang string, seed int64, outputDir string, timeout time.Duration, selfRepair bool, promptVersion string) (bool, error) {
	// Load benchmark spec
	specPath := filepath.Join("benchmarks", benchmarkID+".yml")
	spec, err := eval_harness.LoadSpec(specPath)
	if err != nil {
		return false, fmt.Errorf("failed to load benchmark: %w", err)
	}

	// Check if language is supported
	if !spec.SupportsLanguage(lang) {
		return false, fmt.Errorf("language %s not supported by benchmark %s", lang, benchmarkID)
	}

	// Create AI agent
	agent, err := eval_harness.NewAIAgent(model, seed)
	if err != nil {
		return false, fmt.Errorf("failed to create AI agent: %w", err)
	}

	// Get runner
	runner, err := eval_harness.GetRunner(lang, spec)
	if err != nil {
		return false, fmt.Errorf("failed to get runner: %w", err)
	}

	// Generate prompt
	var prompt string
	if promptVersion != "" {
		// Explicit version specified via --prompt-version flag
		loader, err := eval_harness.NewPromptLoader("prompts/versions.json")
		if err != nil {
			return false, fmt.Errorf("failed to create prompt loader: %w", err)
		}
		customPrompt, err := loader.LoadPrompt(promptVersion)
		if err != nil {
			return false, fmt.Errorf("failed to load prompt version: %w", err)
		}
		prompt = customPrompt
		if spec.TaskPrompt != "" {
			prompt = prompt + "\n\n## Task\n\n" + spec.TaskPrompt
		}
	} else {
		// Try spec.PromptFiles first, then fall back to active version from registry
		prompt = spec.PromptForLanguage(lang)
		if prompt == "" && lang == "ailang" {
			// No prompt in spec, use active version from registry
			loader, err := eval_harness.NewPromptLoader("prompts/versions.json")
			if err != nil {
				return false, fmt.Errorf("failed to create prompt loader: %w", err)
			}
			activePrompt, err := loader.GetActivePrompt()
			if err != nil {
				return false, fmt.Errorf("failed to load active prompt: %w", err)
			}
			prompt = activePrompt
			if spec.TaskPrompt != "" {
				prompt = prompt + "\n\n## Task\n\n" + spec.TaskPrompt
			}
		}
	}

	// Debug: Print prompt info
	if os.Getenv("DEBUG_PROMPT") != "" {
		fmt.Printf("[DEBUG] Prompt length: %d bytes\n", len(prompt))
		fmt.Printf("[DEBUG] First 300 chars: %s\n", prompt[:min(300, len(prompt))])
	}

	// Execute with repair runner
	ctx := context.Background()
	repairRunner := eval_harness.NewRepairRunner(agent, runner, spec, timeout, selfRepair)
	if promptVersion != "" {
		repairRunner.SetPromptVersion(promptVersion)
	}

	metrics, err := repairRunner.Run(ctx, prompt)
	if err != nil {
		return false, fmt.Errorf("benchmark execution failed: %w", err)
	}

	// Save result to JSON
	logger := eval_harness.NewMetricsLogger(outputDir)
	if err := logger.Log(metrics); err != nil {
		return false, fmt.Errorf("failed to save result: %w", err)
	}

	// Return error with failure details if benchmark failed
	if !metrics.StdoutOk {
		if !metrics.CompileOk {
			return false, fmt.Errorf("compilation failed (%s)", metrics.ErrorCategory)
		}
		if !metrics.RuntimeOk {
			return false, fmt.Errorf("runtime error (%s)", metrics.ErrorCategory)
		}
		return false, fmt.Errorf("output mismatch (%s)", metrics.ErrorCategory)
	}

	return true, nil
}

// checkAPIKeys validates that required API keys are set
func checkAPIKeys(models []string) {
	warnings := []string{}

	for _, model := range models {
		switch {
		case strings.Contains(model, "gpt"):
			if os.Getenv("OPENAI_API_KEY") == "" {
				warnings = append(warnings, fmt.Sprintf("%s OPENAI_API_KEY not set (needed for %s)", yellow("⚠️"), model))
			}
		case strings.Contains(model, "claude"):
			if os.Getenv("ANTHROPIC_API_KEY") == "" {
				warnings = append(warnings, fmt.Sprintf("%s ANTHROPIC_API_KEY not set (needed for %s)", yellow("⚠️"), model))
			}
		case strings.Contains(model, "gemini"):
			if os.Getenv("GOOGLE_API_KEY") == "" {
				warnings = append(warnings, fmt.Sprintf("%s GOOGLE_API_KEY not set (needed for %s)", yellow("⚠️"), model))
			}
		}
	}

	if len(warnings) > 0 {
		for _, w := range warnings {
			fmt.Println(w)
		}
		fmt.Println()
		fmt.Println("Set API keys to run with real models:")
		fmt.Println("  export OPENAI_API_KEY='sk-...'")
		fmt.Println("  export ANTHROPIC_API_KEY='sk-ant-...'")
		fmt.Println("  export GOOGLE_API_KEY='...'")
		fmt.Println()
	}
}

// cleanResults removes old result files
func cleanResults(outputDir string) {
	// Remove JSON files but keep directory structure
	pattern := filepath.Join(outputDir, "*.json")
	files, _ := filepath.Glob(pattern)
	for _, f := range files {
		_ = os.Remove(f)
	}
}
