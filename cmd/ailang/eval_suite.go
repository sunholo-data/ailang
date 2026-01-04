package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sunholo/ailang/internal/eval_harness"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// evalTracer is the OpenTelemetry tracer for eval harness instrumentation.
var evalTracer = otel.Tracer("ailang.eval")

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
	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-eval")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: telemetry init failed: %v\n", err)
	} else {
		defer shutdownTelemetry(ctx)
	}

	// Extract trace context from environment (enables cross-process trace linking)
	// If TRACEPARENT is set (e.g., by coordinator or CI), this eval suite will be
	// a child span in the parent trace
	ctx = telemetry.ExtractTraceContext(ctx)
	taskID, sessionID := telemetry.ExtractCorrelationIDs()

	// Start parent span for the entire eval suite
	ctx, suiteSpan := evalTracer.Start(ctx, "eval.suite")
	defer suiteSpan.End()

	// Record correlation IDs as span attributes for fallback trace linking
	if taskID != "" {
		suiteSpan.SetAttributes(attribute.String("ailang.task_id", taskID))
	}
	if sessionID != "" {
		suiteSpan.SetAttributes(attribute.String("ailang.session_id", sessionID))
	}

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
	agentModel := fs.String("agent-model", "", "Override agent CLI model (default: use first model from -models flag). Advanced use only.")
	agentMaxConcurrent := fs.Int("agent-parallel", 10, "Max concurrent agent sessions (agent mode only)")
	agentRequestsPerSecond := fs.Int("agent-rate", 1, "API requests per second (agent mode only)")
	agentTimeout := fs.Int("agent-timeout", 60, "Timeout per benchmark in seconds (agent mode only)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Initialize models configuration
	if err := eval_harness.InitModelsConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load models.yml: %v\n", err)
		fmt.Fprintf(os.Stderr, "Continuing with fallback model lists\n")
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

	// Filter models for agent mode
	if *agent {
		if eval_harness.GlobalModelsConfig == nil {
			fmt.Fprintf(os.Stderr, "Error: models.yml not loaded, cannot determine agent support\n")
			os.Exit(1)
		}

		// Filter to only models that support agent eval
		originalModels := modelList
		modelList = eval_harness.GlobalModelsConfig.FilterAgentSupportedModels(modelList)

		// Warn about skipped models
		if len(modelList) < len(originalModels) {
			skipped := []string{}
			for _, model := range originalModels {
				if !eval_harness.GlobalModelsConfig.SupportsAgentEval(model) {
					skipped = append(skipped, model)
				}
			}
			fmt.Fprintf(os.Stderr, "%s Agent mode: Skipping %d unsupported model(s): %v\n",
				yellow("⚠️"), len(skipped), skipped)
			fmt.Fprintf(os.Stderr, "   These models require CLI integration (not yet implemented)\n")
			fmt.Fprintf(os.Stderr, "   Only Claude models support agent eval currently\n")
			fmt.Println()
		}

		if len(modelList) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No models support agent evaluation\n")
			fmt.Fprintf(os.Stderr, "Agent mode currently only supports Claude models (claude-sonnet-4-5, claude-haiku-4-5)\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Example:\n")
			fmt.Fprintf(os.Stderr, "  ailang eval-suite --agent --models claude-haiku-4-5 --benchmarks fizzbuzz\n")
			fmt.Fprintf(os.Stderr, "\n")
			os.Exit(1)
		}
	}

	// Calculate total runs
	totalRuns := len(modelList) * len(benchmarkList) * len(langList)

	// Set suite span attributes now that we know the configuration
	suiteSpan.SetAttributes(
		attribute.StringSlice("eval.models", modelList),
		attribute.StringSlice("eval.benchmarks", benchmarkList),
		attribute.StringSlice("eval.languages", langList),
		attribute.Int("eval.total_runs", totalRuns),
		attribute.Bool("eval.agent_mode", *agent),
		attribute.Bool("eval.full_suite", *fullSuite),
		attribute.Int64("eval.seed", *seed),
	)

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

	// M-EVAL-GUARD: Start watchdog to detect and kill orphaned eval processes
	watchdog := eval_harness.NewWatchdog(15*time.Minute, 60*time.Second)
	watchdogDone := make(chan struct{})
	go watchdog.Start(watchdogDone)
	defer func() {
		close(watchdogDone)
		if report := watchdog.Report(); report != "No orphaned processes detected" {
			fmt.Printf("%s %s\n", yellow("⚠️"), report)
		}
	}()

	// M-EVAL-GUARD: Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Printf("\n%s Received interrupt, cleaning up...\n", yellow("⚠️"))
		// Kill any remaining orphaned processes
		killed := watchdog.KillOrphans()
		if killed > 0 {
			fmt.Printf("Killed %d orphaned process(es)\n", killed)
		}
		os.Exit(1)
	}()

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
					// Check in appropriate subdirectory based on eval mode
					var patterns []string
					if *agent {
						// Agent mode: check agent/ subdirectory
						patterns = append(patterns, filepath.Join(*outputDir, "agent", fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model)))
					} else {
						// Standard mode: check standard/ subdirectory
						patterns = append(patterns, filepath.Join(*outputDir, "standard", fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model)))
					}
					// Also check root directory for legacy results
					patterns = append(patterns, filepath.Join(*outputDir, fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model)))

					foundExisting := false
					for _, pattern := range patterns {
						matches, _ := filepath.Glob(pattern)
						if len(matches) > 0 {
							foundExisting = true
							break
						}
					}

					if foundExisting {
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
		// Agent CLI model will be determined per-job based on the code generation model
		// (unless --agent-model is explicitly provided as override)
		agentModelOverride := *agentModel

		fmt.Println()
		fmt.Printf("%s Agent mode ENABLED (Claude Code)\n", cyan("🤖"))
		fmt.Printf("  - Models: %v\n", modelList)
		if agentModelOverride != "" {
			fmt.Printf("  - Agent CLI model: %s (override)\n", agentModelOverride)
		} else {
			fmt.Printf("  - Agent CLI model: per-model lookup from models.yml\n")
		}
		fmt.Printf("  - Parallel sessions: %d\n", *agentMaxConcurrent)
		fmt.Printf("  - Rate limit: %d req/sec\n", *agentRequestsPerSecond)
		fmt.Printf("  - Timeout: %d seconds\n", *agentTimeout)
		fmt.Println()

		agentConfig = &eval_harness.AgentBenchmarkConfig{
			MaxConcurrent:     *agentMaxConcurrent,
			RequestsPerSecond: *agentRequestsPerSecond,
			TimeoutSeconds:    *agentTimeout,
			WorkspaceDir:      filepath.Join(os.TempDir(), "ailang_eval"),
			AllowedTools:      []string{"Bash", "Read", "Write", "Edit", "Grep"},
			ClaudePath:        "claude",           // Use PATH
			ClaudeModel:       agentModelOverride, // Empty unless override specified
		}
	}

	// Run benchmarks with concurrency control
	startTime := time.Now()
	results := runBenchmarksParallel(ctx, jobs, *seed, *outputDir, *timeout, *maxConcurrent, finalSelfRepair, *promptVersion, agentConfig)
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

	// Record suite results on span
	suiteSpan.SetAttributes(
		attribute.Int("eval.success_count", successCount),
		attribute.Int("eval.fail_count", failCount),
		attribute.Int64("eval.duration_ms", duration.Milliseconds()),
		attribute.Float64("eval.success_rate", float64(successCount)/float64(totalRuns)*100),
	)
	if failCount > 0 {
		suiteSpan.SetStatus(codes.Error, fmt.Sprintf("%d/%d benchmarks failed", failCount, totalRuns))
	} else {
		suiteSpan.SetStatus(codes.Ok, "all benchmarks passed")
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
func runBenchmarksParallel(ctx context.Context, jobs []Job, seed int64, outputDir string, timeout time.Duration, maxConcurrent int, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig) []SuiteResult {

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
			success, err := runSingleBenchmark(ctx, j.Model, j.Benchmark, j.Language, seed, outputDir, timeout, selfRepair, promptVersion, agentConfig)

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
func runSingleBenchmark(ctx context.Context, model, benchmarkID, lang string, seed int64, outputDir string, timeout time.Duration, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig) (bool, error) {
	// Start span for this benchmark
	// Include benchmark ID in span name for easy identification in trace viewers
	ctx, benchSpan := evalTracer.Start(ctx, fmt.Sprintf("eval.benchmark: %s", benchmarkID),
		trace.WithAttributes(
			attribute.String("benchmark.id", benchmarkID),
			attribute.String("benchmark.model", model),
			attribute.String("benchmark.language", lang),
			attribute.Int64("benchmark.seed", seed),
			attribute.Bool("benchmark.agent_mode", agentConfig != nil),
		),
	)
	defer benchSpan.End()

	// Load benchmark spec
	specPath := filepath.Join("benchmarks", benchmarkID+".yml")
	spec, err := eval_harness.LoadSpec(specPath)
	if err != nil {
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "failed to load benchmark spec")
		return false, fmt.Errorf("failed to load benchmark: %w", err)
	}

	// Check if language is supported
	if !spec.SupportsLanguage(lang) {
		err := fmt.Errorf("language %s not supported by benchmark %s", lang, benchmarkID)
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "unsupported language")
		return false, err
	}

	// Agent mode: Use Claude Code headless evaluation
	if agentConfig != nil {
		// Create unique workspace for this benchmark session
		// Format: /tmp/ailang_eval/<benchmarkID>_<model>_<timestamp>_<pid>
		timestamp := time.Now().Format("20060102_150405")
		workspaceID := fmt.Sprintf("%s_%s_%s_%d", benchmarkID, model, timestamp, os.Getpid())
		sessionConfig := *agentConfig // Copy base config
		sessionConfig.WorkspaceDir = filepath.Join(os.TempDir(), "ailang_eval", workspaceID)

		// Use per-benchmark timeout from YAML if specified, otherwise use default from flag
		if spec.Timeout > 0 {
			sessionConfig.TimeoutSeconds = spec.Timeout
		}

		// Look up executor and model for multi-executor support
		// This enables switching between Claude Code, Gemini CLI, etc. based on model config
		executorName := ""
		modelName := ""
		if sessionConfig.ClaudeModel == "" {
			var err error
			executorName, modelName, err = eval_harness.GlobalModelsConfig.GetExecutorForModel(model)
			if err != nil {
				// Fall back to getting agent model name (backwards compatibility)
				modelName, err = eval_harness.GlobalModelsConfig.GetAgentModelName(model)
				if err != nil {
					return false, fmt.Errorf("could not determine agent model for %s: %w", model, err)
				}
			}
			sessionConfig.ClaudeModel = modelName
		}

		// Use multi-executor runner if executor is specified, otherwise fall back to legacy runner
		var result *eval_harness.AgentBenchmarkResult
		var err error
		if executorName != "" {
			multiConfig := eval_harness.MultiExecutorConfig{
				AgentBenchmarkConfig: sessionConfig,
				ExecutorName:         executorName,
				ModelName:            modelName,
			}
			result, err = eval_harness.RunAgentBenchmarkWithExecutor(spec, multiConfig, lang)
		} else {
			result, err = eval_harness.RunAgentBenchmark(spec, sessionConfig, lang)
		}
		if err != nil {
			benchSpan.RecordError(err)
			benchSpan.SetStatus(codes.Error, "agent benchmark failed")
			return false, fmt.Errorf("agent benchmark failed: %w", err)
		}

		// Save result to JSON
		logger := eval_harness.NewMetricsLogger(outputDir)

		// Convert AgentBenchmarkResult to RunMetrics format for logging
		// Agent mode now uses standard validation fields (compile_ok, runtime_ok, stdout_ok)
		metrics := &eval_harness.RunMetrics{
			ID:           result.BenchmarkID,
			Lang:         lang,
			Model:        model,
			Executor:     result.Executor, // Track which executor was used (claude, gemini, etc.)
			Seed:         seed,
			InputTokens:  result.Usage.InputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens,
			OutputTokens: result.Usage.OutputTokens,
			TotalTokens:  result.Usage.InputTokens + result.Usage.OutputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens,
			CostUSD:      result.Cost,
			// Use standard validation fields from agent runner
			CompileOk:  result.CompileOk,
			RuntimeOk:  result.RuntimeOk,
			StdoutOk:   result.StdoutOk,
			DurationMs: int64(result.DurationMS),
			// Use standard error categorization (same as standard eval mode)
			ErrorCategory:  eval_harness.CategorizeError(result.CompileOk, result.RuntimeOk, result.StdoutOk),
			Stdout:         result.Stdout,
			Stderr:         result.Stderr,
			ExpectedStdout: spec.ExpectedOut,
			Timestamp:      time.Now(),
			PromptVersion:  result.PromptVersion, // Track actual prompt version used (e.g., v0.3.22 for AILANG, python for Python)
			FirstAttemptOk: result.Success,
			RepairUsed:     false, // Agent mode doesn't use standard repair loop
			RepairOk:       false, // Agent mode doesn't use standard repair loop
			Caps:           spec.Caps,
			Code:           result.SolutionCode,
			// Store agent KPI metrics (turns, transcript) for comparison with standard mode
			AgentTurns:      result.NumTurns,
			AgentTranscript: result.SessionLog,
			EvalMode:        eval_harness.EvalModeAgent, // Mark as agent evaluation
		}

		// Append transcript to stderr for backward compatibility with existing tools
		if result.SessionLog != "" {
			metrics.Stderr += "\n\n=== Claude Session Transcript ===\n" + result.SessionLog
		}

		if err := logger.Log(metrics); err != nil {
			benchSpan.RecordError(err)
			benchSpan.SetStatus(codes.Error, "failed to save result")
			return false, fmt.Errorf("failed to save result: %w", err)
		}

		// Record success attributes on span
		benchSpan.SetAttributes(
			attribute.Bool("benchmark.success", result.Success),
			attribute.Int64("benchmark.duration_ms", int64(result.DurationMS)),
			attribute.Int("benchmark.input_tokens", result.Usage.InputTokens),
			attribute.Int("benchmark.output_tokens", result.Usage.OutputTokens),
			attribute.Float64("benchmark.cost_usd", result.Cost),
			attribute.Int("benchmark.turns", result.NumTurns),
		)

		// Add code preview and hash for debugging and deduplication
		if result.SolutionCode != "" {
			benchSpan.SetAttributes(
				attribute.String("code.preview", telemetry.Truncate(result.SolutionCode, 100)),
				attribute.String("code.hash", telemetry.ShortHash(result.SolutionCode, 8)),
			)
		}

		if result.Success {
			benchSpan.SetStatus(codes.Ok, "benchmark passed")
		} else {
			// Add error summary for failed benchmarks
			if result.Stderr != "" {
				benchSpan.SetAttributes(
					attribute.String("error.summary", telemetry.Truncate(result.Stderr, 200)),
					attribute.String("error.category", telemetry.CategorizeError(errors.New(result.Stderr))),
				)
			}
			benchSpan.SetStatus(codes.Error, "benchmark failed")
		}

		return result.Success, nil
	}

	// Standard mode: Create AI agent
	agent, err := eval_harness.NewAIAgent(model, seed)
	if err != nil {
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "failed to create AI agent")
		return false, fmt.Errorf("failed to create AI agent: %w", err)
	}

	// Get runner
	runner, err := eval_harness.GetRunner(lang, spec)
	if err != nil {
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "failed to get runner")
		return false, fmt.Errorf("failed to get runner: %w", err)
	}

	// Generate prompt
	var prompt string
	var actualPromptVersion string
	if promptVersion != "" {
		// Explicit version specified via --prompt-version flag
		if lang == "python" {
			// Python always uses prompts/python.md (not versioned)
			pythonPromptData, err := os.ReadFile("prompts/python.md")
			if err != nil {
				return false, fmt.Errorf("failed to load Python prompt: %w", err)
			}
			prompt = string(pythonPromptData)
			actualPromptVersion = "python"
			if spec.TaskPrompt != "" {
				prompt = prompt + "\n\n## Task\n\n" + spec.TaskPrompt
			}
		} else {
			// AILANG uses versioned prompts from prompts/versions.json
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
		}
	} else {
		// No explicit prompt version specified
		if lang == "python" {
			// Python always uses prompts/python.md (not versioned)
			pythonPromptData, err := os.ReadFile("prompts/python.md")
			if err != nil {
				return false, fmt.Errorf("failed to load Python prompt: %w", err)
			}
			prompt = string(pythonPromptData)
			actualPromptVersion = "python"
			if spec.TaskPrompt != "" {
				prompt = prompt + "\n\n## Task\n\n" + spec.TaskPrompt
			}
		} else {
			// AILANG: Try spec.PromptFiles first, then fall back to active version from registry
			prompt = spec.PromptForLanguage(lang)
			if prompt == "" {
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
			}
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

	// Execute with repair runner (use ctx from span, not new context)
	repairRunner := eval_harness.NewRepairRunner(agent, runner, spec, timeout, selfRepair)
	if actualPromptVersion != "" {
		repairRunner.SetPromptVersion(actualPromptVersion)
	}

	metrics, err := repairRunner.Run(ctx, prompt)
	if err != nil {
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "benchmark execution failed")
		return false, fmt.Errorf("benchmark execution failed: %w", err)
	}

	// Save result to JSON
	logger := eval_harness.NewMetricsLogger(outputDir)
	if err := logger.Log(metrics); err != nil {
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "failed to save result")
		return false, fmt.Errorf("failed to save result: %w", err)
	}

	// Record benchmark metrics on span
	benchSpan.SetAttributes(
		attribute.Bool("benchmark.success", metrics.StdoutOk),
		attribute.Bool("benchmark.compile_ok", metrics.CompileOk),
		attribute.Bool("benchmark.runtime_ok", metrics.RuntimeOk),
		attribute.Int64("benchmark.duration_ms", metrics.DurationMs),
		attribute.Int64("benchmark.input_tokens", int64(metrics.InputTokens)),
		attribute.Int64("benchmark.output_tokens", int64(metrics.OutputTokens)),
		attribute.Float64("benchmark.cost_usd", metrics.CostUSD),
		attribute.String("benchmark.error_category", string(metrics.ErrorCategory)),
		attribute.Bool("benchmark.repair_used", metrics.RepairUsed),
		attribute.Bool("benchmark.repair_successful", metrics.RepairOk),
	)

	// Add code preview and hash for debugging and deduplication
	if metrics.Code != "" {
		benchSpan.SetAttributes(
			attribute.String("code.preview", telemetry.Truncate(metrics.Code, 100)),
			attribute.String("code.hash", telemetry.ShortHash(metrics.Code, 8)),
		)
	}

	// Add error summary for failed benchmarks
	if !metrics.StdoutOk && metrics.Stderr != "" {
		benchSpan.SetAttributes(
			attribute.String("error.summary", telemetry.Truncate(metrics.Stderr, 200)),
		)
	}

	// Return error with failure details if benchmark failed
	if !metrics.StdoutOk {
		if !metrics.CompileOk {
			benchSpan.SetStatus(codes.Error, "compilation failed")
			return false, fmt.Errorf("compilation failed (%s)", metrics.ErrorCategory)
		}
		if !metrics.RuntimeOk {
			benchSpan.SetStatus(codes.Error, "runtime error")
			return false, fmt.Errorf("runtime error (%s)", metrics.ErrorCategory)
		}
		benchSpan.SetStatus(codes.Error, "output mismatch")
		return false, fmt.Errorf("output mismatch (%s)", metrics.ErrorCategory)
	}

	benchSpan.SetStatus(codes.Ok, "benchmark passed")
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
