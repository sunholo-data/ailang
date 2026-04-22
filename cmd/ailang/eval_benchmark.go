package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/claudehistory"
	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// runSingleBenchmark executes a single benchmark configuration
// condition is the experimental condition name ("baseline", "contract", "z3_guided", "full", or "" for legacy)
func runSingleBenchmark(ctx context.Context, model, benchmarkID, lang, condition string, seed int64, outputDir string, timeout time.Duration, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig, taskID string, evalChain *EvalChainContext) (bool, error) {
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

	// Load benchmark spec (uses evalBenchmarkDir set by --benchmark-dir flag)
	specPath := filepath.Join(evalBenchmarkDir, benchmarkID+".yml")
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

	// Resolve experimental condition (controls prompt content and verification)
	cond := eval_harness.ResolveCondition(condition, evalVerifyFlag, evalDevtoolsPromptFlag)

	// Agent mode: Use Claude Code headless evaluation
	if agentConfig != nil {
		// M-EVAL-CHAINS: Create chain stage for this benchmark
		var stageID string
		if evalChain != nil {
			stage, err := evalChain.Store.CreateStage(ctx, &observatory.StageCreateRequest{
				ChainID: evalChain.ChainID,
				AgentID: "eval-agent",
			})
			if err == nil {
				stageID = stage.ID
				_ = evalChain.Store.UpdateStageStatus(ctx, stageID, observatory.StageStatusRunning)
			}
		}

		// Create unique workspace for this benchmark session
		// Format: /tmp/ailang_eval/<benchmarkID>_<model>_<timestamp>_<pid>
		timestamp := time.Now().Format("20060102_150405")
		workspaceID := fmt.Sprintf("%s_%s_%s_%d", benchmarkID, model, timestamp, os.Getpid())
		sessionConfig := *agentConfig // Copy base config
		sessionConfig.WorkspaceDir = filepath.Join(os.TempDir(), "ailang_eval", workspaceID)
		sessionConfig.Condition = cond // Set experimental condition for prompt assembly

		// Use per-benchmark timeout from YAML if specified, otherwise use default from flag
		if spec.Timeout > 0 {
			sessionConfig.TimeoutSeconds = spec.Timeout
		}

		// Look up executor and model for multi-executor support
		// This enables switching between Claude Code, Gemini CLI, etc. based on model config
		// NO FALLBACK to legacy Claude-only runner -- fail fast if executor can't be determined
		executorName := ""
		modelName := sessionConfig.ClaudeModel // May already be set from --agent-model override

		if modelName == "" {
			// Look up executor and model from models.yml
			var err error
			executorName, modelName, err = eval_harness.GlobalModelsConfig.GetExecutorForModel(model)
			if err != nil {
				return false, fmt.Errorf("could not determine executor for model %q in agent mode: %w\n"+
					"Ensure model has agent_cli and agent_model_name configured in models.yml", model, err)
			}
			sessionConfig.ClaudeModel = modelName
		} else {
			// Model name overridden via --agent-model; still need executor name for routing
			var err error
			executorName, _, err = eval_harness.GlobalModelsConfig.GetExecutorForModel(model)
			if err != nil {
				return false, fmt.Errorf("could not determine executor for model %q in agent mode: %w\n"+
					"Ensure model has agent_cli configured in models.yml", model, err)
			}
		}

		// Always use multi-executor runner -- legacy RunAgentBenchmark() hardcodes Claude
		// and must NOT be used for non-Claude models
		multiConfig := eval_harness.MultiExecutorConfig{
			AgentBenchmarkConfig: sessionConfig,
			ExecutorName:         executorName,
			ModelName:            modelName,
			ConfigKey:            model, // original models.yml key for per-model timeout lookup
		}

		// M-EVAL-CHAINS: Structured tool/chat capture (executor-aware)
		// Claude: skip streaming capture (gives {} inputs due to input_json_delta protocol)
		//   → post-execution JSONL import provides complete data (see below)
		// Gemini/other: streaming capture works (full tool data in initial events)
		if evalChain != nil && stageID != "" && executorName != "claude" {
			multiConfig.ExtraHandler = NewObservatoryWriter(
				evalChain.Store, evalChain.ChainID, stageID, sessionConfig.WorkspaceDir,
			)
		}

		result, err := eval_harness.RunAgentBenchmarkWithExecutor(spec, multiConfig, lang)

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
				Condition:      condition,
			}
			_ = logger.Log(apiErrorMetrics) // Best effort - don't fail on logging error

			// M-EVAL-CHAINS: Record failure in chain stage
			if evalChain != nil && stageID != "" {
				assessment := &observatory.EvalAssessment{
					BenchmarkID:   spec.ID,
					Model:         model,
					Language:      lang,
					Condition:     condition,
					EvalMode:      "agent",
					Executor:      executorName,
					ErrorCategory: string(eval_harness.ErrorCategoryAPI),
					Stderr:        telemetry.Truncate(fmt.Sprintf("API Error: %v", err), 500),
				}
				_ = evalChain.Store.UpdateStageEvalAssessment(ctx, stageID, assessment)
				_ = evalChain.Store.UpdateStageError(ctx, stageID, err.Error())
			}

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
			Condition:      condition, // Experimental condition for this run
			// Store agent KPI metrics (turns, transcript) for comparison with standard mode
			AgentTurns:      result.NumTurns,
			AgentTranscript: result.SessionLog,
			EvalMode:        eval_harness.EvalModeAgent,                    // Mark as agent evaluation
			MicroragState:   eval_harness.MicroragModeAuto.ResolvedState(), // M-BRAIN-MICRORAG
		}

		// Append transcript to stderr for backward compatibility with existing tools
		if result.SessionLog != "" {
			execLabel := result.Executor
			if execLabel == "" {
				execLabel = "Agent"
			} else {
				// Capitalize first letter
				execLabel = strings.ToUpper(execLabel[:1]) + execLabel[1:]
			}
			metrics.Stderr += fmt.Sprintf("\n\n=== %s Session Transcript ===\n", execLabel) + result.SessionLog
		}

		if err := logger.Log(metrics); err != nil {
			benchSpan.RecordError(err)
			benchSpan.SetStatus(codes.Error, "failed to save result")
			return false, fmt.Errorf("failed to save result: %w", err)
		}

		// M-EVAL-CHAINS: Store assessment in chain stage
		if evalChain != nil && stageID != "" {
			assessment := &observatory.EvalAssessment{
				BenchmarkID:    spec.ID,
				Model:          model,
				Language:       lang,
				Condition:      condition,
				EvalMode:       "agent",
				Executor:       result.Executor,
				Seed:           seed,
				CompileOk:      result.CompileOk,
				RuntimeOk:      result.RuntimeOk,
				StdoutOk:       result.StdoutOk,
				ErrorCategory:  string(eval_harness.CategorizeError(result.CompileOk, result.RuntimeOk, result.StdoutOk)),
				FirstAttemptOk: result.Success,
				PromptVersion:  result.PromptVersion,
				CodeHash:       telemetry.ShortHash(result.SolutionCode, 8),
				Code:           telemetry.Truncate(result.SolutionCode, 2000),
				Stdout:         telemetry.Truncate(result.Stdout, 500),
				ExpectedStdout: telemetry.Truncate(spec.ExpectedOut, 500),
				Stderr:         telemetry.Truncate(result.Stderr, 500),
			}
			_ = evalChain.Store.UpdateStageEvalAssessment(ctx, stageID, assessment)

			tokensIn := result.Usage.InputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens
			_ = evalChain.Store.UpdateStageMetrics(ctx, stageID, result.Cost, tokensIn, result.Usage.OutputTokens, result.NumTurns, result.ToolCallCount, int64(result.DurationMS))

			stageStatus := observatory.StageStatusCompleted
			if !result.Success {
				stageStatus = observatory.StageStatusFailed
			}
			_ = evalChain.Store.UpdateStageStatus(ctx, stageID, stageStatus)

			// Link session to stage — prefer ObservatoryWriter session (has tool data)
			// over executor-reported session (just an ID with no local data)
			if multiConfig.ExtraHandler == nil && result.SessionID != "" {
				_ = evalChain.Store.UpdateStageSession(ctx, stageID, result.SessionID)
			}

			// Claude: import chat history from disk (complete tool inputs/outputs)
			// Claude Code saves JSONL to ~/.claude/projects/ with full tool_use blocks.
			// The claudehistory.Importer reads these files and populates chat_messages.
			if executorName == "claude" && result.SessionID != "" {
				corr := &observatory.SessionCorrelation{
					ChainID: evalChain.ChainID,
					StageID: stageID,
				}
				_ = evalChain.Store.UpsertSessionWithCorrelation(ctx, result.SessionID,
					sessionConfig.WorkspaceDir, "", "eval-agent", corr)

				chatImporter := claudehistory.NewImporter(evalChain.Store.DB())
				if n, importErr := chatImporter.SyncSession(ctx, result.SessionID); importErr != nil {
					fmt.Fprintf(os.Stderr, "[eval] chat import for %s: %v\n", result.SessionID, importErr)
				} else if os.Getenv("DEBUG_AGENT") != "" {
					fmt.Fprintf(os.Stderr, "[eval] imported %d chat messages for session %s\n", n, result.SessionID)
				}
			}
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

	// Standard mode: Create chain stage for this benchmark
	var stageID string
	if evalChain != nil {
		stage, err := evalChain.Store.CreateStage(ctx, &observatory.StageCreateRequest{
			ChainID: evalChain.ChainID,
			AgentID: "eval-standard",
		})
		if err == nil {
			stageID = stage.ID
			_ = evalChain.Store.UpdateStageStatus(ctx, stageID, observatory.StageStatusRunning)
		}
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
	// PromptForLanguage() handles both AILANG and Python correctly:
	// - AILANG: teaching prompt (1600 lines) + "## Task" + benchmark task
	// - Python: prompts/python.md guidelines + "## Task" + benchmark task
	var prompt string
	var actualPromptVersion string

	if promptVersion != "" && lang != "python" {
		// Explicit version specified via --prompt-version flag (AILANG only)
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
		// Append task description from spec
		taskDesc := spec.TaskPrompt
		if taskDesc == "" {
			taskDesc = spec.Prompt
		}
		if taskDesc != "" {
			langName := lang
			if lang == "python" {
				langName = "Python 3"
			} else if lang == "ailang" {
				langName = "AILANG"
			}
			prompt = prompt + "\n\n## Task\n\n" + strings.ReplaceAll(taskDesc, "<LANG>", langName)
		}
	} else {
		// Use PromptForLanguage() which correctly composes base + task for all languages
		prompt = spec.PromptForLanguage(lang)
		if lang == "python" {
			actualPromptVersion = "python"
		} else if prompt == "" {
			// No prompt from spec, fall back to active version from registry
			loader, err := eval_harness.NewPromptLoader("prompts/versions.json")
			if err != nil {
				return false, fmt.Errorf("failed to create prompt loader: %w", err)
			}
			activePrompt, err := loader.GetActivePrompt()
			if err != nil {
				return false, fmt.Errorf("failed to load active prompt: %w", err)
			}
			prompt = activePrompt
			actualPromptVersion = loader.GetActiveVersionID()
			// Append task description from spec
			taskDesc := spec.TaskPrompt
			if taskDesc == "" {
				taskDesc = spec.Prompt
			}
			if taskDesc != "" {
				prompt = prompt + "\n\n## Task\n\n" + strings.ReplaceAll(taskDesc, "<LANG>", "AILANG")
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
	repairRunner.SetVerify(cond.EnableVerify, evalVerifyTimeout)

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
			Condition:      condition,
		}
		_ = logger.Log(apiErrorMetrics) // Best effort - don't fail on logging error

		// M-EVAL-CHAINS: Record failure in chain stage (standard mode)
		if evalChain != nil && stageID != "" {
			assessment := &observatory.EvalAssessment{
				BenchmarkID:   spec.ID,
				Model:         model,
				Language:      lang,
				Condition:     condition,
				EvalMode:      "standard",
				ErrorCategory: string(eval_harness.ErrorCategoryAPI),
				Stderr:        telemetry.Truncate(fmt.Sprintf("API Error: %v", err), 500),
			}
			_ = evalChain.Store.UpdateStageEvalAssessment(ctx, stageID, assessment)
			_ = evalChain.Store.UpdateStageError(ctx, stageID, err.Error())
		}

		return false, fmt.Errorf("benchmark execution failed: %w", err)
	}

	// Tag metrics with experimental condition
	metrics.Condition = condition

	if err := logger.Log(metrics); err != nil {
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "failed to save result")
		return false, fmt.Errorf("failed to save result: %w", err)
	}

	// M-EVAL-CHAINS: Store assessment in chain stage (standard mode)
	if evalChain != nil && stageID != "" {
		assessment := &observatory.EvalAssessment{
			BenchmarkID:    spec.ID,
			Model:          model,
			Language:       lang,
			Condition:      condition,
			EvalMode:       "standard",
			Seed:           seed,
			CompileOk:      metrics.CompileOk,
			RuntimeOk:      metrics.RuntimeOk,
			StdoutOk:       metrics.StdoutOk,
			ErrorCategory:  string(metrics.ErrorCategory),
			FirstAttemptOk: metrics.StdoutOk,
			RepairUsed:     metrics.RepairUsed,
			RepairOk:       metrics.RepairOk,
			PromptVersion:  actualPromptVersion,
			CodeHash:       telemetry.ShortHash(metrics.Code, 8),
			Code:           telemetry.Truncate(metrics.Code, 2000),
			Stdout:         telemetry.Truncate(metrics.Stdout, 500),
			ExpectedStdout: telemetry.Truncate(spec.ExpectedOut, 500),
			Stderr:         telemetry.Truncate(metrics.Stderr, 500),
		}
		_ = evalChain.Store.UpdateStageEvalAssessment(ctx, stageID, assessment)
		_ = evalChain.Store.UpdateStageMetrics(ctx, stageID, metrics.CostUSD,
			metrics.InputTokens, metrics.OutputTokens, 0, 0, metrics.DurationMs)

		stageStatus := observatory.StageStatusCompleted
		if !metrics.StdoutOk {
			stageStatus = observatory.StageStatusFailed
		}
		_ = evalChain.Store.UpdateStageStatus(ctx, stageID, stageStatus)
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
