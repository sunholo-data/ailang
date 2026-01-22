package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sunholo/ailang/internal/eval_harness"
	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
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
	ctx := context.Background()

	// Check if we already have a task ID from coordinator/parent process
	taskID, sessionID := telemetry.ExtractCorrelationIDs()

	// If no task ID provided, generate one for this eval run
	// This enables eval runs to appear in the Observatory task hierarchy
	// Generate assignment ID as well - needed for span grouping in UI
	var assignmentID string
	if taskID == "" {
		taskID = fmt.Sprintf("eval-%d", time.Now().UnixNano())
		assignmentID = fmt.Sprintf("aa_%d", time.Now().UnixNano())
		// Set in environment so it gets picked up by telemetry.NewResource()
		// Both task_id and assignment_id are needed for full hierarchy visibility
		existingAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES")
		newAttrs := fmt.Sprintf("ailang.task_id=%s,ailang.assignment_id=%s", taskID, assignmentID)
		if existingAttrs != "" {
			os.Setenv("OTEL_RESOURCE_ATTRIBUTES", existingAttrs+","+newAttrs)
		} else {
			os.Setenv("OTEL_RESOURCE_ATTRIBUTES", newAttrs)
		}
	}

	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
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
	fullSuite := fs.Bool("full", false, "Run full benchmark suite with all models from extended_suite (gpt5-1-codex-max, claude-opus-4-5, claude-sonnet-4-5, gemini-3-pro, gemini-2-5-pro)")
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

	// Message-based coordination flags (M-UNIFIED-AI-CONTROL-PLANE)
	queueMode := fs.Bool("queue", false, "Run benchmarks via message queue (coordinator processes, crash recovery)")
	queueInbox := fs.String("queue-inbox", "eval-runner", "Inbox for queue mode benchmark jobs")
	queueWait := fs.Bool("queue-wait", true, "Wait for all queued benchmarks to complete (queue mode only)")

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
		// Full suite: use extended suite (5 models) from models.yml
		if eval_harness.GlobalModelsConfig != nil && len(eval_harness.GlobalModelsConfig.ExtendedSuite) > 0 {
			modelList = eval_harness.GlobalModelsConfig.ExtendedSuite
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5-1-codex-max", "claude-opus-4-5", "claude-sonnet-4-5", "gemini-3-pro", "gemini-2-5-pro"}
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

	// Create Task in Observatory for this eval run
	// This enables eval suites to appear in the task hierarchy
	createEvalTask(taskID, assignmentID, modelList, benchmarkList, langList, totalRuns, *agent)

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

	// Queue mode: Submit benchmarks as messages for coordinator processing (M-UNIFIED-AI-CONTROL-PLANE)
	if *queueMode {
		results := runBenchmarksViaQueue(ctx, jobs, taskID, *queueInbox, *seed, *outputDir, *agent, *queueWait, suiteSpan)
		_ = time.Since(time.Now()) // Duration tracked in queue function

		// Summary for queue mode
		successCount := 0
		failCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			} else {
				failCount++
			}
		}

		suiteSpan.SetAttributes(
			attribute.Int("eval.success_count", successCount),
			attribute.Int("eval.fail_count", failCount),
			attribute.Bool("eval.queue_mode", true),
		)
		if failCount > 0 {
			suiteSpan.SetStatus(codes.Error, fmt.Sprintf("%d/%d benchmarks failed", failCount, len(jobs)))
		} else {
			suiteSpan.SetStatus(codes.Ok, "all benchmarks passed")
		}

		fmt.Println()
		fmt.Printf("%s Benchmark suite complete (queue mode)!\n", green("✓"))
		fmt.Printf("Queued: %d jobs to inbox '%s'\n", len(jobs), *queueInbox)
		if *queueWait {
			fmt.Printf("Completed: %d/%d (%.1f%%)\n", successCount, len(jobs), float64(successCount)/float64(len(jobs))*100)
		} else {
			fmt.Println("Jobs queued - use 'ailang messages list --inbox eval-runner' to monitor")
		}
		fmt.Println()
		completeEvalTask(taskID, failCount == 0)
		return
	}

	// Open messaging store for dashboard event visibility (non-blocking on failure)
	var evalStore *messaging.Store
	if store, err := openStore(); err == nil {
		evalStore = store
		defer evalStore.Close()

		// Create "Suite Started" message for event queue visibility
		startPayload := map[string]interface{}{
			"task_id":    taskID,
			"models":     modelList,
			"benchmarks": benchmarkList,
			"languages":  langList,
			"total_jobs": len(jobs),
			"agent_mode": *agent,
		}
		payloadBytes, _ := json.Marshal(startPayload)

		startMsg := &messaging.InboxMessage{
			FromAgent:     "eval-suite",
			ToInbox:       "controlplane",
			MessageType:   messaging.InboxTypeNotification,
			Title:         fmt.Sprintf("Eval Suite Started: %d models, %d benchmarks", len(modelList), len(benchmarkList)),
			Payload:       string(payloadBytes),
			Category:      "eval",
			CorrelationID: taskID,
		}
		if err := evalStore.InsertInboxMessageWithContext(ctx, startMsg); err == nil {
			// Broadcast to dashboard via HTTP (non-blocking)
			go broadcastEvalEvent(startMsg)
			// Emit span so event appears in ExecHierarchy (Milestone 12)
			emitEventSpan(ctx, "suite_started", taskID, startMsg)
		}
	}

	// Run benchmarks with concurrency control (direct mode)
	startTime := time.Now()
	results := runBenchmarksParallel(ctx, jobs, *seed, *outputDir, *timeout, *maxConcurrent, finalSelfRepair, *promptVersion, agentConfig, taskID)
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

	// Create "Suite Completed" message for event queue visibility
	if evalStore != nil {
		status := "completed"
		if failCount > 0 {
			status = "partial"
		}

		completePayload := map[string]interface{}{
			"task_id":      taskID,
			"success":      successCount,
			"failed":       failCount,
			"total":        totalRuns,
			"duration_sec": duration.Seconds(),
			"success_rate": float64(successCount) / float64(totalRuns) * 100,
		}
		payloadBytes, _ := json.Marshal(completePayload)

		completeMsg := &messaging.InboxMessage{
			FromAgent:     "eval-suite",
			ToInbox:       "controlplane",
			MessageType:   messaging.InboxTypeNotification,
			Title:         fmt.Sprintf("Eval Suite %s: %d/%d passed (%.1f%%)", status, successCount, totalRuns, float64(successCount)/float64(totalRuns)*100),
			Payload:       string(payloadBytes),
			Category:      "eval",
			CorrelationID: taskID,
		}
		if err := evalStore.InsertInboxMessageWithContext(ctx, completeMsg); err == nil {
			// Broadcast to dashboard via HTTP (non-blocking)
			go broadcastEvalEvent(completeMsg)
			// Emit span so event appears in ExecHierarchy (Milestone 12)
			emitEventSpan(ctx, "suite_completed", taskID, completeMsg)
		}
	}

	// Update task status to completed
	completeEvalTask(taskID, failCount == 0)
}

// Job represents a single benchmark task
type Job struct {
	Model     string
	Benchmark string
	Language  string
}

// runBenchmarksParallel executes benchmarks with concurrency control
func runBenchmarksParallel(ctx context.Context, jobs []Job, seed int64, outputDir string, timeout time.Duration, maxConcurrent int, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig, taskID string) []SuiteResult {

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
			success, err := runSingleBenchmark(ctx, j.Model, j.Benchmark, j.Language, seed, outputDir, timeout, selfRepair, promptVersion, agentConfig, taskID)

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
func runSingleBenchmark(ctx context.Context, model, benchmarkID, lang string, seed int64, outputDir string, timeout time.Duration, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig, taskID string) (bool, error) {
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

		// Save result to JSON (even on API error for observability)
		logger := eval_harness.NewMetricsLogger(outputDir)

		if err != nil {
			// API error - save result with api_error category for observability
			benchSpan.RecordError(err)
			benchSpan.SetStatus(codes.Error, "agent benchmark failed")

			apiErrorMetrics := &eval_harness.RunMetrics{
				ID:             spec.ID,
				Lang:           lang,
				Model:          model,
				Executor:       executorName,
				Seed:           seed,
				CompileOk:      false,
				RuntimeOk:      false,
				StdoutOk:       false,
				ErrorCategory:  eval_harness.ErrorCategoryAPI,
				Stderr:         fmt.Sprintf("API Error: %v", err),
				ExpectedStdout: spec.ExpectedOut,
				Timestamp:      time.Now(),
				Caps:           spec.Caps,
				EvalMode:       eval_harness.EvalModeAgent,
			}
			_ = logger.Log(apiErrorMetrics) // Best effort - don't fail on logging error
			return false, fmt.Errorf("agent benchmark failed: %w", err)
		}

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

	// Get runner with context for full telemetry hierarchy (TRACEPARENT + task ID)
	runner, err := eval_harness.GetRunnerWithContext(ctx, lang, spec, taskID)
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

	// Save result to JSON (moved up to handle API errors)
	logger := eval_harness.NewMetricsLogger(outputDir)

	metrics, err := repairRunner.Run(ctx, prompt)
	if err != nil {
		// API error - save result with api_error category for observability
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "benchmark execution failed")

		apiErrorMetrics := &eval_harness.RunMetrics{
			ID:             spec.ID,
			Lang:           lang,
			Model:          model,
			Seed:           seed,
			CompileOk:      false,
			RuntimeOk:      false,
			StdoutOk:       false,
			ErrorCategory:  eval_harness.ErrorCategoryAPI,
			Stderr:         fmt.Sprintf("API Error: %v", err),
			ExpectedStdout: spec.ExpectedOut,
			Timestamp:      time.Now(),
			Caps:           spec.Caps,
			EvalMode:       eval_harness.EvalModeStandard,
			PromptVersion:  actualPromptVersion,
		}
		_ = logger.Log(apiErrorMetrics) // Best effort - don't fail on logging error
		return false, fmt.Errorf("benchmark execution failed: %w", err)
	}
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

// createEvalTask creates a Task in Observatory for this eval run.
// This enables eval suites to appear in the task hierarchy alongside
// coordinator tasks. The task ID is used as ailang.task_id resource
// attribute so all child spans (benchmarks, API calls) are linked.
func createEvalTask(taskID, assignmentID string, models, benchmarks, langs []string, totalRuns int, agentMode bool) {
	// Try Observatory API endpoint (default server at localhost:1957)
	endpoint := os.Getenv("OBSERVATORY_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:1957"
	}

	// Get working directory for workspace lookup
	cwd, _ := os.Getwd()

	// Find workspace ID by looking up existing workspaces
	workspaceID := lookupWorkspaceID(endpoint, cwd)
	if workspaceID == "" {
		// No matching workspace found - skip task creation
		// (Task creation would fail due to foreign key constraint)
		return
	}

	// Build task title
	title := fmt.Sprintf("Eval: %d benchmarks × %d models", len(benchmarks), len(models))
	if agentMode {
		title += " (agent)"
	}

	// Build description
	description := fmt.Sprintf("Models: %s\nBenchmarks: %s\nLanguages: %s\nTotal runs: %d",
		strings.Join(models, ", "),
		strings.Join(benchmarks, ", "),
		strings.Join(langs, ", "),
		totalRuns,
	)

	// Create task object
	now := time.Now()
	task := observatory.Task{
		ID:          taskID,
		WorkspaceID: workspaceID,
		Title:       title,
		Description: description,
		SourceType:  observatory.TaskSourceManual,
		SourceRef:   "eval-suite",
		Status:      observatory.TaskStatusRunning,
		Priority:    "normal",
		CreatedAt:   now,
		StartedAt:   &now,
	}

	// POST to Observatory API
	taskJSON, err := json.Marshal(task)
	if err != nil {
		// Non-fatal: task creation failure shouldn't stop eval
		return
	}

	resp, err := http.Post(
		endpoint+"/api/observatory/tasks",
		"application/json",
		bytes.NewReader(taskJSON),
	)
	if err != nil {
		// Non-fatal: Observatory might not be running
		return
	}
	defer resp.Body.Close()

	// Log success if created
	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("%s Task created in Observatory: %s\n", cyan("📊"), taskID)

		// Also create an agent assignment so spans show up in UI
		// The UI groups spans under agent assignments
		createEvalAgentAssignment(endpoint, taskID, assignmentID, models, agentMode)
	}
}

// createEvalAgentAssignment creates an agent assignment for eval runs.
// This is needed because the UI groups spans under agent assignments.
// The assignmentID is passed from the caller to ensure it matches what was
// set in OTEL_RESOURCE_ATTRIBUTES (so spans link to this assignment).
func createEvalAgentAssignment(endpoint, taskID, assignmentID string, models []string, agentMode bool) {
	now := time.Now()

	// Determine provider from models
	provider := observatory.ProviderClaude // default
	for _, model := range models {
		if strings.Contains(model, "gemini") {
			provider = observatory.ProviderGemini
			break
		}
	}

	agentID := "eval-harness"
	if agentMode {
		agentID = "eval-agent"
	}

	assignment := observatory.AgentAssignment{
		ID:         assignmentID,
		TaskID:     taskID,
		AgentID:    agentID,
		Provider:   provider,
		Status:     observatory.AgentStatusRunning,
		AssignedAt: now,
		StartedAt:  &now,
	}

	assignmentJSON, err := json.Marshal(assignment)
	if err != nil {
		return
	}

	resp, err := http.Post(
		endpoint+"/api/observatory/agents",
		"application/json",
		bytes.NewReader(assignmentJSON),
	)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// completeEvalTask updates the task status to completed when the eval finishes.
func completeEvalTask(taskID string, success bool) {
	endpoint := os.Getenv("OBSERVATORY_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:1957"
	}

	status := observatory.TaskStatusCompleted
	if !success {
		status = observatory.TaskStatusFailed
	}

	now := time.Now()
	update := map[string]any{
		"status":       status,
		"completed_at": now.Format(time.RFC3339),
	}

	updateJSON, err := json.Marshal(update)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPut, endpoint+"/api/observatory/tasks/"+taskID, bytes.NewReader(updateJSON))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// lookupWorkspaceID finds the workspace ID for a given path by querying the Observatory API.
// Returns empty string if no workspace found or API unavailable.
func lookupWorkspaceID(endpoint, path string) string {
	resp, err := http.Get(endpoint + "/api/observatory/workspaces")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var workspaces []observatory.Workspace
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return ""
	}

	// Find workspace matching our path
	for _, ws := range workspaces {
		if ws.Path == path {
			return ws.ID
		}
	}

	return ""
}

// EvalBenchmarkJob represents a benchmark job payload for queue mode (M-UNIFIED-AI-CONTROL-PLANE)
type EvalBenchmarkJob struct {
	Model       string `json:"model"`
	Benchmark   string `json:"benchmark"`
	Language    string `json:"language"`
	Seed        int64  `json:"seed"`
	OutputDir   string `json:"output_dir"`
	AgentMode   bool   `json:"agent_mode"`
	SuiteTaskID string `json:"suite_task_id"`
}

// runBenchmarksViaQueue submits benchmarks as messages to the specified inbox.
// Coordinator daemon picks up messages and processes via ailang exec.
// This enables crash recovery (unacked messages resume) and distributed execution.
func runBenchmarksViaQueue(ctx context.Context, jobs []Job, suiteTaskID, inbox string, seed int64, outputDir string, agentMode, wait bool, suiteSpan trace.Span) []SuiteResult {
	// Open message store
	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Failed to open message store: %v\n", red("✗"), err)
		suiteSpan.RecordError(err)
		return nil
	}
	defer store.Close()

	// Generate correlation ID for this eval suite run
	correlationID := fmt.Sprintf("eval_%s", suiteTaskID)

	// Submit each benchmark as a message
	var messageIDs []string
	fmt.Printf("%s Submitting %d benchmark jobs to queue '%s'...\n", cyan("→"), len(jobs), inbox)

	for i, job := range jobs {
		// Create job payload
		payload := EvalBenchmarkJob{
			Model:       job.Model,
			Benchmark:   job.Benchmark,
			Language:    job.Language,
			Seed:        seed,
			OutputDir:   outputDir,
			AgentMode:   agentMode,
			SuiteTaskID: suiteTaskID,
		}
		payloadBytes, _ := json.Marshal(payload)

		// Create message with hierarchy metadata
		msg := &messaging.InboxMessage{
			FromAgent:     "eval-suite",
			ToInbox:       inbox,
			MessageType:   messaging.InboxTypeNotification,
			Title:         fmt.Sprintf("Benchmark: %s/%s/%s", job.Benchmark, job.Language, job.Model),
			Payload:       string(payloadBytes),
			CorrelationID: correlationID,
			ParentTaskID:  suiteTaskID,
			Category:      "eval",
		}

		if err := store.InsertInboxMessage(msg); err != nil {
			fmt.Fprintf(os.Stderr, "%s Failed to queue job %d: %v\n", red("✗"), i+1, err)
			continue
		}
		messageIDs = append(messageIDs, msg.MessageID)

		// Progress indicator
		if (i+1)%50 == 0 || i+1 == len(jobs) {
			fmt.Printf("  Queued %d/%d jobs\n", i+1, len(jobs))
		}
	}

	suiteSpan.SetAttributes(
		attribute.Int("eval.jobs_queued", len(messageIDs)),
		attribute.String("eval.correlation_id", correlationID),
	)

	fmt.Printf("%s Queued %d benchmark jobs (correlation: %s)\n", green("✓"), len(messageIDs), correlationID)

	// If not waiting, return empty results (async mode)
	if !wait {
		fmt.Println()
		fmt.Println("Jobs queued for processing by coordinator daemon.")
		fmt.Println("Monitor progress:")
		fmt.Printf("  ailang messages list --inbox %s --unread\n", inbox)
		fmt.Println()
		return nil
	}

	// Wait for all jobs to complete
	fmt.Println()
	fmt.Printf("%s Waiting for %d jobs to complete...\n", cyan("→"), len(messageIDs))
	fmt.Println("  (Coordinator daemon processes these - ensure it's running)")
	fmt.Println()

	// Poll for completion
	pollInterval := 5 * time.Second
	maxWait := 2 * time.Hour
	startTime := time.Now()

	results := make([]SuiteResult, len(jobs))
	completed := make(map[string]bool)

	for time.Since(startTime) < maxWait {
		// Check how many messages are still unread
		messages, err := store.ListInboxMessages(messaging.InboxListOptions{
			Inbox:      inbox,
			UnreadOnly: true,
		})
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		// Count remaining from our correlation
		remaining := 0
		for _, msg := range messages {
			if msg.CorrelationID == correlationID {
				remaining++
			}
		}

		completedCount := len(messageIDs) - remaining
		if completedCount > len(completed) {
			// New completions - mark them
			for _, msg := range messages {
				if msg.CorrelationID == correlationID {
					completed[msg.MessageID] = true
				}
			}
			fmt.Printf("  Progress: %d/%d completed\n", completedCount, len(messageIDs))
		}

		if remaining == 0 {
			// All done
			break
		}

		time.Sleep(pollInterval)
	}

	// Build results from completed messages
	// For now, assume success if message was acknowledged
	for i, job := range jobs {
		results[i] = SuiteResult{
			BenchmarkID: job.Benchmark,
			Language:    job.Language,
			Model:       job.Model,
			Success:     true, // Message was processed
		}
	}

	return results
}

// emitEventSpan creates an OTEL span for eval events so they appear in the ExecHierarchy.
// This bridges inbox messages (collaboration.db) with spans (observatory.db) by creating
// a span whenever we create an inbox message. The span name follows the pattern
// "eval.event.{eventType}" and includes attributes linking to the task/message.
func emitEventSpan(ctx context.Context, eventType, taskID string, msg *messaging.InboxMessage) {
	_, span := evalTracer.Start(ctx, "eval.event."+eventType)
	span.SetAttributes(
		attribute.String("event.type", eventType),
		attribute.String("event.title", msg.Title),
		attribute.String("event.message_id", msg.MessageID),
		attribute.String("ailang.task_id", taskID),
		attribute.String("ailang.category", "eval"),
	)
	// End immediately (event is instantaneous)
	span.End()
}

// broadcastEvalEvent sends an eval event to the dashboard server for real-time updates.
// This is non-blocking and best-effort - failures are silently ignored.
func broadcastEvalEvent(msg *messaging.InboxMessage) {
	// Create a simplified event payload for the dashboard
	event := map[string]interface{}{
		"type":      "eval_event",
		"message":   msg.Title,
		"category":  msg.Category,
		"from":      msg.FromAgent,
		"task_id":   msg.CorrelationID,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	// POST to dashboard server (non-blocking, best-effort)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(
		"http://127.0.0.1:1957/api/coordinator/events",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return // Silently ignore - dashboard may not be running
	}
	defer resp.Body.Close()
}
