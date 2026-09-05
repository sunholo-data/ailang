package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// runSingleBenchmark executes a single benchmark configuration
// condition is the experimental condition name ("baseline", "contract", "z3_guided", "full", or "" for legacy)
// onCost, when non-nil, is called with the trial's banked CostUSD immediately
// after its result is successfully logged (M-EVAL-STANDARD-CONFIDENCE-GATING
// --budget-usd). Deliberately NOT wired into every early-return path in this
// function — a trial that errors out before its result is ever logged has no
// banked cost figure to report, and (for API-calling paths) typically incurred
// little to no billable cost either. This means an aggregate budget cap fed by
// onCost is a "sum of what got banked" gauge, not a byte-exact real-time meter
// — consistent with the accepted graceful-stop tolerance in the design doc.
func runSingleBenchmark(ctx context.Context, model, benchmarkID, lang, condition string, trial int, seed int64, outputDir string, timeout time.Duration, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig, taskID string, evalChain *EvalChainContext, onCost func(float64)) (bool, error) {
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

	// M-EVAL-NETWORK-MOCK-FIXTURE: if the benchmark's prompt references the mock-URL
	// token, start a local deterministic HTTP mock for THIS run and substitute the
	// token with the live server URL. This replaces non-deterministic external calls
	// (e.g. httpbin.org returning intermittent 503s) with a reproducible, offline,
	// concurrency-safe local server. The server binds an ephemeral port (one per run,
	// never shared) and is torn down when this benchmark finishes — including agent-
	// mode repair iterations, which all run while the server is up. spec is a per-job
	// pointer from LoadSpec, so mutating TaskPrompt here is local to this run.
	if eval_harness.PromptUsesHTTPMock(spec.TaskPrompt) {
		mock := eval_harness.StartHTTPMock()
		defer mock.Close()
		spec.TaskPrompt = strings.ReplaceAll(spec.TaskPrompt, eval_harness.MockHTTPURLToken, mock.URL)
		benchSpan.SetAttributes(attribute.String("benchmark.mock_http_url", mock.URL))
	}

	// Resolve experimental condition (controls prompt content and verification)
	cond := eval_harness.ResolveCondition(condition, evalVerifyFlag, evalDevtoolsPromptFlag)

	// Agent mode: Use Claude Code headless evaluation
	// Agent mode: Use Claude Code headless evaluation (extracted to
	// eval_benchmark_agent.go — this branch always returns, see its doc comment)
	if agentConfig != nil {
		return runSingleBenchmarkAgent(ctx, benchSpan, spec, model, benchmarkID, lang, condition, cond, trial, seed, outputDir, agentConfig, evalChain, onCost)
	}

	// Standard mode: Create chain stage for this benchmark
	// M-EVAL-LOCAL-OBSERVABILITY M2: include benchmark id in agent_id (see comment above)
	var stageID string
	if evalChain != nil {
		stage, err := evalChain.Store.CreateStage(ctx, &observatory.StageCreateRequest{
			ChainID: evalChain.ChainID,
			AgentID: "eval-standard:" + benchmarkID,
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

	// Tag OpenRouter requests so the trace OpenRouter broadcasts back can be
	// joined to this chain and benchmark (M-OPENROUTER-BROADCAST-INGEST M3).
	// No chain (e.g. a bare `ailang eval-benchmark`) leaves the request
	// wire-identical to before.
	if evalChain != nil {
		agent.SetCorrelation(evalChain.ChainID, spec.ID, spec.Difficulty)
		registerStandardEvalSession(ctx, evalChain, stageID, outputDir)
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
		// Generation failed — classify (refusal is model behavior, not
		// infrastructure) and save for observability.
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "benchmark execution failed")
		stdErrCategory := eval_harness.CategorizeStandardAPIError(err)

		apiErrorMetrics := &eval_harness.RunMetrics{
			ID:             spec.ID,
			Lang:           lang,
			Model:          model,
			Seed:           seed,
			CompileOk:      false,
			RuntimeOk:      false,
			StdoutOk:       false,
			ErrorCategory:  stdErrCategory,
			Stderr:         fmt.Sprintf("API Error: %v", err),
			ExpectedStdout: spec.ExpectedOut,
			Timestamp:      time.Now(),
			Caps:           spec.Caps,
			EvalMode:       eval_harness.EvalModeStandard,
			PromptVersion:  actualPromptVersion,
			Condition:      condition,
			Trial:          trial, // M-EVAL-OS-LONGITUDINAL Phase 3
		}
		// M-LYCEUM-PROVIDER M3: carry the FAILED call's latency when the client
		// measured it (provider clients stamp WallMS on their error paths). This
		// is what distinguishes "gateway 504 after 30s" from "after 6 minutes"
		// in the banked rows instead of only in the console log.
		var pe *ai.ProviderError
		if errors.As(err, &pe) && pe.WallMS > 0 {
			apiErrorMetrics.LLMWallMs = pe.WallMS
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
				ErrorCategory: stdErrCategory,
				Stderr:        telemetry.Truncate(fmt.Sprintf("API Error: %v", err), 500),
			}
			_ = evalChain.Store.UpdateStageEvalAssessment(ctx, stageID, assessment)
			_ = evalChain.Store.UpdateStageError(ctx, stageID, err.Error())
		}

		return false, fmt.Errorf("benchmark execution failed: %w", err)
	}

	// Tag metrics with experimental condition + trial number (M-EVAL-OS-LONGITUDINAL Phase 3).
	// metrics comes from repairRunner.Run which doesn't know about trials —
	// we own this annotation here.
	metrics.Condition = condition
	metrics.Trial = trial

	if err := logger.Log(metrics); err != nil {
		benchSpan.RecordError(err)
		benchSpan.SetStatus(codes.Error, "failed to save result")
		return false, fmt.Errorf("failed to save result: %w", err)
	}
	if onCost != nil {
		onCost(metrics.CostUSD)
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
			// Contract verification (M-COST-PER-SUCCESS-KPI M1): standard-mode
			// verification evidence, banked identically to the agent path.
			VerifyOk:        metrics.VerifyOk,
			VerifyVerified:  metrics.VerifyVerified,
			VerifyCounterex: metrics.VerifyCounterex,
			VerifySkipped:   metrics.VerifySkipped,
			VerifyErrors:    metrics.VerifyErrors,
			PromptVersion:   actualPromptVersion,
			CodeHash:        telemetry.ShortHash(metrics.Code, 8),
			Code:            telemetry.Truncate(metrics.Code, 2000),
			Stdout:          telemetry.Truncate(metrics.Stdout, 500),
			ExpectedStdout:  telemetry.Truncate(spec.ExpectedOut, 500),
			Stderr:          telemetry.Truncate(metrics.Stderr, 500),
		}
		_ = evalChain.Store.UpdateStageEvalAssessment(ctx, stageID, assessment)
		_ = evalChain.Store.UpdateStageMetrics(ctx, stageID, metrics.CostUSD,
			metrics.InputTokens, metrics.OutputTokens, 0, 0, metrics.DurationMs, metrics.CostProvenance)

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

// registerStandardEvalSession makes the OpenRouter session_id resolvable by the
// observatory receiver. Session registration is telemetry-only and must never
// prevent an evaluation from running.
func registerStandardEvalSession(ctx context.Context, evalChain *EvalChainContext, stageID, workspace string) {
	if evalChain == nil || stageID == "" {
		return
	}

	_ = evalChain.Store.UpsertSessionWithCorrelation(ctx, evalChain.ChainID,
		workspace, "", "eval-standard", &observatory.SessionCorrelation{
			ChainID: evalChain.ChainID,
			StageID: stageID,
		})
}
