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
	fullSuite := fs.Bool("full", false, "Run full benchmark suite with all 6 models from extended_suite (gpt5, gpt5-mini, claude-sonnet-4-5, claude-haiku-4-5, gemini-2-5-pro, gemini-2-5-flash)")
	benchmarks := fs.String("benchmarks", "", "Comma-separated list of benchmarks (empty = auto-discover from benchmarks/)")
	langs := fs.String("langs", "python,ailang", "Comma-separated list of languages")
	seed := fs.Int64("seed", 42, "Random seed for deterministic runs")
	outputDir := fs.String("output", "eval_results", "Output directory for results")
	timeout := fs.Duration("timeout", 30*time.Second, "Timeout for code execution")
	maxConcurrent := fs.Int("parallel", 10, "Maximum concurrent API calls across all providers (0 = sequential, recommended: 10-15)")
	selfRepair := fs.Bool("self-repair", true, "Enable single-shot self-repair on errors (default: true)")
	noSelfRepair := fs.Bool("no-self-repair", false, "Disable self-repair (run without error correction)")
	promptVersion := fs.String("prompt-version", "", "Prompt version ID for all benchmarks")
	skipExisting := fs.Bool("skip-existing", false, "Skip benchmarks that already have result files (resume interrupted run)")

	// Agent mode flags
	agent := fs.Bool("agent", false, "Use agent-based evaluation (Claude Code headless mode)")
	agentModel := fs.String("agent-model", "haiku", "Claude model for agent (haiku, sonnet, opus, or full name)")
	agentMaxConcurrent := fs.Int("agent-parallel", 10, "Max concurrent agent sessions (agent mode only)")
	agentRequestsPerSecond := fs.Int("agent-rate", 1, "API requests per second (agent mode only)")
	agentTimeout := fs.Int("agent-timeout", 300, "Timeout per benchmark in seconds (agent mode only)")
	agentMaxIterations := fs.Int("agent-iterations", 10, "Max agent iterations per benchmark (agent mode only)")

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
		// Full suite: use extended suite (all 6 models) from models.yml
		if eval_harness.GlobalModelsConfig != nil && len(eval_harness.GlobalModelsConfig.ExtendedSuite) > 0 {
			modelList = eval_harness.GlobalModelsConfig.ExtendedSuite
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5", "gpt5-mini", "claude-sonnet-4-5", "claude-haiku-4-5", "gemini-2-5-pro", "gemini-2-5-flash"}
		}
	} else {
		// Default: use dev models from models.yml
		if eval_harness.GlobalModelsConfig != nil && len(eval_harness.GlobalModelsConfig.DevModels) > 0 {
			modelList = eval_harness.GlobalModelsConfig.DevModels
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5-mini", "claude-haiku-4-5", "gemini-2-5-flash"}
		}
	}
	var benchmarkList []string
	if *benchmarks == "" {
		// SAFETY: Agent mode requires explicit benchmark list to prevent accidental large runs
		if *agent {
			fmt.Fprintf(os.Stderr, "Error: --agent mode requires explicit --benchmarks list\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Agent mode is expensive and time-consuming. You must explicitly specify which benchmarks to run.\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Example:\n")
			fmt.Fprintf(os.Stderr, "  ailang eval-suite --agent --benchmarks cli_args,fizzbuzz\n")
			fmt.Fprintf(os.Stderr, "\n")
			os.Exit(1)
		}

		// Auto-discover benchmarks from benchmarks/ directory (standard mode only)
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

	// Clean previous results (unless resuming)
	if !*skipExisting {
		fmt.Printf("%s Cleaning previous results...\n", cyan("→"))
		cleanResults(*outputDir)
	} else {
		fmt.Printf("%s Resuming run (skipping existing results)...\n", cyan("→"))
	}

	// Build job list in round-robin order by model
	// This interleaves models to distribute API calls across providers (OpenAI, Anthropic, Google)
	// allowing higher parallelism without hitting single-provider rate limits.
	//
	// Example with 3 models, 2 benchmarks, 2 languages:
	//   Old order: [m1/b1/l1, m1/b1/l2, m1/b2/l1, m1/b2/l2, m2/b1/l1, ...]
	//   New order: [m1/b1/l1, m2/b1/l1, m3/b1/l1, m1/b1/l2, m2/b1/l2, ...]
	//
	// With --parallel 10 and 3 providers, this means ~3-4 concurrent calls per provider
	// instead of 10 calls to the same provider.
	var jobs []Job
	skippedCount := 0
	for _, lang := range langList {
		// For each benchmark, create jobs for all models (round-robin)
		for benchIdx := 0; benchIdx < len(benchmarkList); benchIdx++ {
			for _, model := range modelList {
				benchmark := benchmarkList[benchIdx]
				job := Job{
					Model:     model,
					Benchmark: benchmark,
					Language:  lang,
				}

				// Check if result already exists (if resuming)
				if *skipExisting {
					// Result filename format: benchmarkID_lang_model_timestamp.json
					// We check for any file matching the pattern (ignoring timestamp)
					pattern := filepath.Join(*outputDir, fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model))
					matches, _ := filepath.Glob(pattern)
					if len(matches) > 0 {
						skippedCount++
						continue // Skip this job
					}
				}

				jobs = append(jobs, job)
			}
		}
	}

	if *skipExisting && skippedCount > 0 {
		fmt.Printf("Skipped %d existing results\n", skippedCount)
		fmt.Println()
	}

	// Handle --no-self-repair flag (overrides --self-repair=true default)
	finalSelfRepair := *selfRepair
	if *noSelfRepair {
		finalSelfRepair = false
		fmt.Println("Self-repair DISABLED (--no-self-repair)")
	} else {
		fmt.Println("Self-repair ENABLED (default)")
	}

	// Configure agent mode if requested
	var agentConfig *eval_harness.AgentBenchmarkConfig
	if *agent {
		fmt.Println()
		fmt.Printf("%s Agent mode ENABLED (Claude Code headless)\n", cyan("🤖"))
		fmt.Printf("  - Model: %s\n", *agentModel)
		fmt.Printf("  - Parallel sessions: %d\n", *agentMaxConcurrent)
		fmt.Printf("  - Rate limit: %d req/sec\n", *agentRequestsPerSecond)
		fmt.Printf("  - Timeout: %d seconds\n", *agentTimeout)
		fmt.Printf("  - Max iterations: %d\n", *agentMaxIterations)
		fmt.Println()

		agentConfig = &eval_harness.AgentBenchmarkConfig{
			MaxConcurrent:     *agentMaxConcurrent,
			RequestsPerSecond: *agentRequestsPerSecond,
			TimeoutSeconds:    *agentTimeout,
			MaxIterations:     *agentMaxIterations,
			WorkspaceDir:      filepath.Join(os.TempDir(), "ailang_eval"),
			AllowedTools:      []string{"Bash", "Read", "Write", "Edit", "Grep"},
			ClaudePath:        "claude",      // Use PATH
			ClaudeModel:       *agentModel,   // haiku, sonnet, opus, or full name
		}
	}

	// Run benchmarks with concurrency control
	startTime := time.Now()
	results := runBenchmarksParallel(jobs, *seed, *outputDir, *timeout, *maxConcurrent, finalSelfRepair, *promptVersion, agentConfig)
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
func runBenchmarksParallel(jobs []Job, seed int64, outputDir string, timeout time.Duration, maxConcurrent int, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig) []SuiteResult {

	if maxConcurrent <= 0 {
		maxConcurrent = 1 // Sequential
	}

	var (
		wg           sync.WaitGroup
		results      = make([]SuiteResult, len(jobs))
		sem          = make(chan struct{}, maxConcurrent) // Semaphore for concurrency control
		mu           sync.Mutex                           // Protect progress counter
		failureCount int                                  // Track consecutive failures
		aborted      bool                                 // Early abort flag
	)

	completed := 0
	totalJobs := len(jobs)

	for i, job := range jobs {
		// Check if we should abort early
		mu.Lock()
		if aborted {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		go func(idx int, j Job) {
			defer wg.Done()

			// Check abort flag before starting work
			mu.Lock()
			if aborted {
				mu.Unlock()
				return
			}
			mu.Unlock()

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
			success, err := runSingleBenchmark(j.Model, j.Benchmark, j.Language, seed, outputDir, timeout, selfRepair, promptVersion, agentConfig)

			results[idx] = SuiteResult{
				BenchmarkID: j.Benchmark,
				Language:    j.Language,
				Model:       j.Model,
				Success:     success,
				Error:       err,
			}

			if success {
				fmt.Printf("  %s Completed\n", green("✓"))
				mu.Lock()
				failureCount = 0 // Reset failure count on success
				mu.Unlock()
			} else {
				fmt.Printf("  %s Failed: %v\n", red("✗"), err)
				mu.Lock()
				failureCount++
				// Abort if first 50 results are all failures
				if completed >= 50 && failureCount >= 50 {
					if !aborted {
						aborted = true
						fmt.Printf("\n%s Aborting: First 50 results all failed - likely system issue!\n", red("🚨"))
						fmt.Printf("Check: interpreter debug output, missing API keys, or broken prompt.\n\n")
					}
				}
				mu.Unlock()
			}
		}(i, job)
	}

	wg.Wait()
	return results
}

// runSingleBenchmark executes a single benchmark configuration
func runSingleBenchmark(model, benchmarkID, lang string, seed int64, outputDir string, timeout time.Duration, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig) (bool, error) {
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

	// Agent mode: Use Claude Code headless evaluation
	if agentConfig != nil {
		// Create unique workspace for this benchmark session
		// Format: /tmp/ailang_eval/<benchmarkID>_<model>_<timestamp>_<pid>
		timestamp := time.Now().Format("20060102_150405")
		workspaceID := fmt.Sprintf("%s_%s_%s_%d", benchmarkID, model, timestamp, os.Getpid())
		sessionConfig := *agentConfig // Copy base config
		sessionConfig.WorkspaceDir = filepath.Join(os.TempDir(), "ailang_eval", workspaceID)

		result, err := eval_harness.RunAgentBenchmark(spec, sessionConfig, lang)
		if err != nil {
			return false, fmt.Errorf("agent benchmark failed: %w", err)
		}

		// Save result to JSON
		logger := eval_harness.NewMetricsLogger(outputDir)

		// Convert AgentBenchmarkResult to RunMetrics format for logging
		// Agent mode produces different result structure but we normalize to RunMetrics for consistency
		metrics := &eval_harness.RunMetrics{
			ID:             result.BenchmarkID,
			Lang:           lang,
			Model:          model,
			Seed:           seed,
			InputTokens:    result.Usage.InputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens,
			OutputTokens:   result.Usage.OutputTokens,
			TotalTokens:    result.Usage.InputTokens + result.Usage.OutputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens,
			CostUSD:        result.Cost,
			CompileOk:      result.Success, // Agent handles compile internally
			RuntimeOk:      result.Success, // Agent handles runtime internally
			StdoutOk:       result.Success, // Agent validates output
			DurationMs:     int64(result.DurationMS),
			ErrorCategory:  eval_harness.CategorizeError(result.Success, result.Success, result.Success),
			Timestamp:      time.Now(),
			PromptVersion:  "agent", // Mark as agent mode
			FirstAttemptOk: result.Success,
			RepairUsed:     false, // Agent mode doesn't use standard repair loop
			RepairOk:       false, // Agent mode doesn't use standard repair loop
			Caps:           spec.Caps,
		}

		// Add expected output (required for compatibility with dashboard/analysis tools)
		metrics.ExpectedStdout = spec.ExpectedOut

		// Add error message if failed
		if !result.Success {
			metrics.Stderr = result.Error
			if result.Error != "" {
				metrics.ErrorCategory = eval_harness.ErrorCategoryCompile // Default to compile error
			}
		}

		// Add solution code for inspection
		metrics.Code = result.SolutionCode

		// Store agent KPI metrics (turns, transcript) for comparison with standard mode
		metrics.AgentTurns = result.NumTurns
		metrics.AgentTranscript = result.SessionLog

		// Also append transcript to stderr for backward compatibility with existing tools
		if result.SessionLog != "" {
			metrics.Stderr += "\n\n=== Claude Session Transcript ===\n" + result.SessionLog
		}

		if err := logger.Log(metrics); err != nil {
			return false, fmt.Errorf("failed to save result: %w", err)
		}

		return result.Success, nil
	}

	// Standard mode: Create AI agent
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
	var actualPromptVersion string
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
		actualPromptVersion = promptVersion
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
			// Track the actual version used from registry
			actualPromptVersion = loader.GetActiveVersionID()
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
	if actualPromptVersion != "" {
		repairRunner.SetPromptVersion(actualPromptVersion)
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
