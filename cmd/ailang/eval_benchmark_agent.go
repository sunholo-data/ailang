package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/sunholo-data/ailang/internal/modelreg"
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

// runSingleBenchmarkAgent runs the agent-mode (multi-turn CLI executor) path of
// runSingleBenchmark. Extracted verbatim (mechanical split, no logic change) to
// keep eval_benchmark.go under the 800-line file-size gate — this branch and the
// standard-mode branch it split from were the only two things in that file, each
// self-contained and always-returning, so the split has no behavioral effect.
// See eval_benchmark.go's runSingleBenchmark for the shared preamble (span setup,
// spec loading, HTTP mock fixture, condition resolution) and the standard-mode
// counterpart.
func runSingleBenchmarkAgent(ctx context.Context, benchSpan trace.Span, spec *eval_harness.BenchmarkSpec, model, benchmarkID, lang, condition string, cond eval_harness.EvalCondition, trial int, seed int64, outputDir string, agentConfig *eval_harness.AgentBenchmarkConfig, evalChain *EvalChainContext, onCost func(float64)) (bool, error) {
	// M-EVAL-CHAINS: Create chain stage for this benchmark.
	// M-EVAL-LOCAL-OBSERVABILITY M2: include benchmark id in agent_id so
	// `ailang chains view` can distinguish 4 concurrent eval-agent stages
	// (without this, all 4 stages show as "eval-agent [running]" — useless
	// for live monitoring on local Ollama where each stage takes 10-25 min).
	var stageID string
	if evalChain != nil {
		stage, err := evalChain.Store.CreateStage(ctx, &observatory.StageCreateRequest{
			ChainID: evalChain.ChainID,
			AgentID: "eval-agent:" + benchmarkID,
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
		executorName, modelName, err = modelreg.GlobalModelsConfig.GetExecutorForModel(model)
		if err != nil {
			return false, fmt.Errorf("could not determine executor for model %q in agent mode: %w\n"+
				"Ensure model has agent_cli and agent_model_name configured in models.yml", model, err)
		}
		sessionConfig.ClaudeModel = modelName
	} else {
		// Model name overridden via --agent-model; still need executor name for routing
		var err error
		executorName, _, err = modelreg.GlobalModelsConfig.GetExecutorForModel(model)
		if err != nil {
			return false, fmt.Errorf("could not determine executor for model %q in agent mode: %w\n"+
				"Ensure model has agent_cli configured in models.yml", model, err)
		}
	}

	// Multi-executor runner: the ONLY agent-mode runner. (The legacy
	// Claude-hardcoded RunAgentBenchmark() was deleted in M4a-3 — it had no
	// caller yet held the only agent-mode contract-verification call, which is
	// why no agent run ever banked verify_verified > 0. See BF-1.)
	multiConfig := eval_harness.MultiExecutorConfig{
		AgentBenchmarkConfig: sessionConfig,
		ExecutorName:         executorName,
		ModelName:            modelName,
		ConfigKey:            model, // original models.yml key for per-model timeout lookup
		Browser:              sessionConfig.Browser,
	}
	if multiConfig.Browser.ArtifactDir != "" {
		multiConfig.Browser.ArtifactDir = filepath.Join(multiConfig.Browser.ArtifactDir, workspaceID)
	}

	// M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP: thread chain_id/stage_id through
	// so the opencode subprocess receives them as OTEL_RESOURCE_ATTRIBUTES.
	// This is what makes `ailang chains live` show real "X s ago" last-span
	// ages instead of "(no spans yet)" for active chains.
	if evalChain != nil {
		multiConfig.ChainID = evalChain.ChainID
	}
	if stageID != "" {
		multiConfig.StageID = stageID
	}

	// M-EVAL-OS-LONGITUDINAL Phase 2: adaptive token-budget override.
	// If the eval-suite was invoked with --max-tokens-per-bench N (and we
	// have observatory access), look up the per-(model, benchmark) rolling
	// baseline. When N>=5 passing samples exist, replace the fixed N with
	// the adaptive `mean + 2σ` threshold. During bootstrap (n<5) the fixed
	// flag value is preserved, so behavior matches Phase 1 until enough
	// data accumulates. Best-effort: any DB error keeps the fixed budget.
	if multiConfig.AgentBenchmarkConfig.MaxTokensPerBench > 0 && evalChain != nil && evalChain.Store != nil {
		if baseline, blErr := observatory.GetEvalBaseline(ctx, evalChain.Store.DB(), model, spec.ID); blErr == nil {
			adaptive := observatory.ComputeAdaptiveThreshold(
				baseline,
				observatory.AdaptiveThresholdSigmas,
				multiConfig.AgentBenchmarkConfig.MaxTokensPerBench,
			)
			if adaptive > 0 && adaptive != multiConfig.AgentBenchmarkConfig.MaxTokensPerBench {
				multiConfig.AgentBenchmarkConfig.MaxTokensPerBench = adaptive
			}
		}
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

		// M-EVAL-SWEET-SPOT: classify the failure into a typed category
		// when the error string carries a recognizable signal (OpenRouter
		// quota kill, 429, context deadline). Otherwise stays as
		// api_error. result is nil on err-return so we can't read a
		// FinishReason here.
		errCategory := eval_harness.CategorizeAgentError(err, "")

		// Bank the executor's session transcript even though the run produced
		// no usable measurement. A crashed or thrashing run is EXACTLY the one
		// whose tool RESULTS you need — the fmt dialect bug was found in a run
		// that thrashed — and this path used to discard them. RunAgentBenchmark-
		// WithExecutor returns a diagnostics-only result (identifying fields and
		// the session path, nothing else) once the executor has actually run;
		// it is nil for the earlier setup failures, where no session exists.
		if result != nil && executorName == "motoko" && result.SessionJSONLPath != "" &&
			evalChain != nil && evalChain.Store != nil {
			if imp, impErr := evalChain.Store.ImportMotokoSession(ctx, result.SessionJSONLPath); impErr != nil {
				fmt.Fprintf(os.Stderr, "[eval] motoko session import (failed run) for %s: %v\n", result.SessionJSONLPath, impErr)
			} else {
				fmt.Fprintf(os.Stderr, "[eval] banked transcript of FAILED run: %s -> chain %s (%d steps, %d tool calls)\n",
					imp.SessionLabel, imp.ChainID, imp.Steps, imp.ToolCalls)
			}
		}

		apiErrorMetrics := &eval_harness.RunMetrics{
			ID:             spec.ID,
			Lang:           lang,
			Model:          model,
			Executor:       executorName,
			Seed:           seed,
			CompileOk:      false,
			RuntimeOk:      false,
			StdoutOk:       false,
			ErrorCategory:  errCategory,
			Stderr:         fmt.Sprintf("API Error: %v", err),
			ExpectedStdout: spec.ExpectedOut,
			Timestamp:      time.Now(),
			Caps:           spec.Caps,
			EvalMode:       eval_harness.EvalModeAgent,
			Condition:      condition,
			Trial:          trial, // M-EVAL-OS-LONGITUDINAL Phase 3
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
				ErrorCategory: errCategory,
				Stderr:        telemetry.Truncate(fmt.Sprintf("API Error: %v", err), 500),
			}
			_ = evalChain.Store.UpdateStageEvalAssessment(ctx, stageID, assessment)
			_ = evalChain.Store.UpdateStageError(ctx, stageID, err.Error())
		}

		return false, fmt.Errorf("agent benchmark failed: %w", err)
	}

	// Convert AgentBenchmarkResult to RunMetrics format for logging
	// Agent mode now uses standard validation fields (compile_ok, runtime_ok, stdout_ok)
	// M-EVAL-SWEET-SPOT: when the executor surfaces a structured
	// FinishReason (motoko cost_exhausted, future step_exhausted),
	// promote that to the canonical ErrorCategory. Otherwise fall back
	// to the standard compile/runtime/logic classification.
	// M-EVAL-MEM-GUARD: CategorizeRunError promotes memory-watchdog kills
	// (marker in validation stderr) to resource_limit.
	// M4 (m-eval-measurement-contract): assert the run measured what
	// models.yml said it would. Compares the CLAIMED motoko_profile
	// against the profile the subject reports it actually loaded; a
	// contradiction quarantines the row instead of counting it. Ten cloud
	// motoko entries silently ran `dogfood` for weeks while advertising
	// microRAG+DP7, and nothing noticed because nothing compared them.
	var claimedProfile string
	if modelreg.GlobalModelsConfig != nil {
		if mc, err := modelreg.GlobalModelsConfig.GetModel(model); err == nil {
			claimedProfile = mc.MotokoProfile
		}
	}
	configValidity := eval_harness.AssertResolvedProfile(claimedProfile, result.ResolvedProfile)

	// Treatment integrity (M6 verification gate + M5 void clause). An arm
	// that cannot demonstrate its treatment applied — or a control showing
	// the treatment leaked in — does not measure what it claims, so it is
	// quarantined rather than reported as a null. Config mismatch takes
	// precedence: if the run loaded the wrong profile entirely, that is the
	// more fundamental problem to report.
	if configValidity == nil {
		configValidity = eval_harness.AssertFmtTreatmentIntegrity(result.FmtHook, result.FmtHookEvents)
	}

	errCategory := eval_harness.CategorizeRunError(result.CompileOk, result.RuntimeOk, result.StdoutOk, result.Stderr)
	if !result.Success && result.FinishReason != "" {
		if typed := eval_harness.CategorizeAgentError(nil, result.FinishReason); typed != eval_harness.ErrorCategoryAPI {
			errCategory = typed
		}
	}

	metrics := &eval_harness.RunMetrics{
		Validity:    configValidity,
		ID:          result.BenchmarkID,
		Lang:        lang,
		Model:       model,
		Executor:    result.Executor,    // Track which executor was used (claude, gemini, etc.)
		ModelFamily: result.ModelFamily, // For cross-harness grouping (M-EVAL-CROSS-HARNESS)
		Seed:        seed,
		// InputTokens stays cache-INCLUSIVE so this stays comparable with every
		// pre-2026-08-11 baseline. The disjoint parts are banked alongside it
		// below, which is what makes the total decomposable.
		InputTokens:  result.Usage.InputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens,
		OutputTokens: result.Usage.OutputTokens,
		// Bank the cache split (2026-08-11). RunMetrics has carried these fields
		// since v0.31.0 but ONLY the standard-mode repair path ever set them, so
		// every agent-mode row omitted them (omitempty) and the agent cache hit
		// rate was unmeasurable from our own data — the exact gap v0.31.0 set out
		// to close, left open on the agent side. Without these, input_tokens is
		// cache-inclusive while cost_usd is priced cache-exclusive, so the two
		// describe different quantities and neither can be derived from the other.
		CacheReadInputTokens:     result.Usage.CacheReadInputTokens,
		CacheCreationInputTokens: result.Usage.CacheCreationInputTokens,
		// Always true on this path: the split above is populated for every agent
		// run from here on, INCLUDING genuine zeros (a local ollama model with no
		// prompt cache). That is the point — it marks the row as decomposable, so
		// the fresh-token KPI can exclude pre-2026-08-11 rows instead of scoring
		// them as 100% fresh and manufacturing an improvement.
		CacheAccounted: true,
		// Hidden reasoning tokens, kept disjoint from OutputTokens but counted
		// in TotalTokens (upstream bills them at the output rate). 0 = the
		// executor doesn't report a count, not "the model didn't think".
		ReasonTokens: result.Usage.ReasonTokens,
		TotalTokens:  result.Usage.InputTokens + result.Usage.OutputTokens + result.Usage.ReasonTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens,
		// NOT recomputed from tokens: agent CLIs report their own billed cost,
		// which already includes reasoning. Deriving it here would double-count.
		CostUSD: result.Cost,
		// Whether that cost was actually billed. The rig's codex and claude
		// lanes authenticate by subscription, so a non-zero cost there is a
		// list-price equivalent, not spend (see executor.CostProvenance).
		CostProvenance: result.CostProvenance,
		// Use standard validation fields from agent runner
		CompileOk:  result.CompileOk,
		RuntimeOk:  result.RuntimeOk,
		StdoutOk:   result.StdoutOk,
		DurationMs: int64(result.DurationMS),
		// Contract verification (M4a-3): the banked RESULT JSON now carries the
		// same 5 verify_* fields the chain assessment does. Standard mode has
		// always recorded these (via PopulateVerifyMetrics); agent mode dropped
		// them, so even once verification ran, eval-summary / eval-matrix could
		// not see it. Same evidence, both surfaces.
		VerifyOk:        result.VerifyOk,
		VerifyVerified:  result.VerifyVerified,
		VerifyCounterex: result.VerifyCounterex,
		VerifySkipped:   result.VerifySkipped,
		VerifyErrors:    result.VerifyErrors,
		VerifyJSON:      result.VerifyJSON,
		// M-EVAL-SWEET-SPOT: prefers FinishReason-derived category when available
		ErrorCategory:  errCategory,
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
		// Store agent KPI metrics (turns, tool calls, transcript) for comparison with standard mode
		AgentTurns:         result.NumTurns,
		AgentToolCalls:     result.ToolCallCount,
		AgentToolHistogram: result.ToolCalls,
		AgentTranscript:    result.SessionLog,
		EvalMode:           eval_harness.EvalModeAgent,                    // Mark as agent evaluation
		MicroragState:      eval_harness.MicroragModeAuto.ResolvedState(), // M-BRAIN-MICRORAG
		// Fmt-hook A/B (M-EVAL-FMT-WEAKMODEL-AB): resolved arm + hook reality,
		// banked for the config-diff review and M3's treatment-delivery metric.
		ResolvedProfile:    result.ResolvedProfile,
		ResolvedExtensions: result.ResolvedExtensions,
		FmtHookState:       result.FmtHook,
		FmtHookEvents:      result.FmtHookEvents,

		// Cost-and-speed budget metrics (M-EVAL-COST-AND-SPEED-BUDGETS, v0.15.1)
		CostKilledAt:   result.CostKilledAt,
		FirstAttemptMs: result.FirstAttemptMs,
		SuccessAtMs:    result.SuccessAtMs,
		TokensPerSec:   result.TokensPerSec,

		// Executor finish signal (M-EVAL-SWEET-SPOT, v0.19.0)
		FinishReason: result.FinishReason,

		// Context-compaction telemetry (M-AILANG-SEMANTIC-CONTEXT, v0.26.0)
		CompactionCount:     result.CompactionCount,
		CompactionFirstStep: result.CompactionFirstStep,
		CompactionMaxLevel:  result.CompactionMaxLevel,

		// Trial number (M-EVAL-OS-LONGITUDINAL Phase 3)
		Trial: trial,
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
	if onCost != nil {
		onCost(metrics.CostUSD)
	}

	// M-EVAL-OS-LONGITUDINAL Phase 2: extend the rolling token-budget
	// baseline ONLY on PASS outcomes (where the token count is a valid
	// "this is how much work it took to succeed" sample). Failures are
	// excluded — a thrashing run's 2M tokens would skew the baseline up
	// and disable the very abort it's meant to inform. Best-effort: DB
	// errors don't affect the result reported to the caller.
	if metrics.CompileOk && metrics.RuntimeOk && metrics.StdoutOk &&
		evalChain != nil && evalChain.Store != nil && metrics.TotalTokens > 0 {
		if upErr := observatory.UpdatePassedTrial(ctx, evalChain.Store.DB(), model, spec.ID, metrics.TotalTokens); upErr != nil {
			// Non-fatal — log and continue.
			fmt.Fprintf(os.Stderr, "warning: failed to update eval baseline for %s/%s: %v\n", model, spec.ID, upErr)
		}
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
			ErrorCategory:  string(eval_harness.CategorizeRunError(result.CompileOk, result.RuntimeOk, result.StdoutOk, result.Stderr)),
			FirstAttemptOk: result.Success,
			// Contract verification (M-COST-PER-SUCCESS-KPI M1): carry the 5
			// verify_* fields so the banked eval_assessment can distinguish a
			// golden-output pass from an independently verified program.
			VerifyOk:        result.VerifyOk,
			VerifyVerified:  result.VerifyVerified,
			VerifyCounterex: result.VerifyCounterex,
			VerifySkipped:   result.VerifySkipped,
			VerifyErrors:    result.VerifyErrors,
			PromptVersion:   result.PromptVersion,
			CodeHash:        telemetry.ShortHash(result.SolutionCode, 8),
			Code:            telemetry.Truncate(result.SolutionCode, 2000),
			Stdout:          telemetry.Truncate(result.Stdout, 500),
			ExpectedStdout:  telemetry.Truncate(spec.ExpectedOut, 500),
			Stderr:          telemetry.Truncate(result.Stderr, 500),
		}
		_ = evalChain.Store.UpdateStageEvalAssessment(ctx, stageID, assessment)

		tokensIn := result.Usage.InputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens
		_ = evalChain.Store.UpdateStageMetrics(ctx, stageID, result.Cost, tokensIn, result.Usage.OutputTokens, result.NumTurns, result.ToolCallCount, int64(result.DurationMS), result.CostProvenance)

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

		// motoko: import the session JSONL, which carries tool RESULTS.
		//
		// This is the motoko sibling of the Claude branch above, and it was
		// missing. The eval harness banks its own agent_transcript containing
		// tool CALLS only, so what the agent was TOLD — every compiler
		// diagnostic, and the fmt hook's "canonical AILANG would differ here"
		// message — never reached eval data. ImportMotokoSession has parsed
		// tool_result all along, but only ran from `ailang chains
		// import-motoko` by hand. The result: `ailang fmt` contradicted the
		// teaching prompt on every write for two weeks, at a measured +62%
		// output tokens, and three sessions of analysis blamed the model for
		// syntax it had been instructed to use.
		//
		// Best-effort by design: a failed import must never fail a benchmark
		// row. It is diagnostics, not measurement — but it is reported on
		// stderr rather than swallowed, so a silently-empty transcript cannot
		// recur unnoticed.
		if executorName == "motoko" && result.SessionJSONLPath != "" {
			if imp, importErr := evalChain.Store.ImportMotokoSession(ctx, result.SessionJSONLPath); importErr != nil {
				fmt.Fprintf(os.Stderr, "[eval] motoko session import for %s: %v\n", result.SessionJSONLPath, importErr)
			} else if os.Getenv("DEBUG_AGENT") != "" {
				fmt.Fprintf(os.Stderr, "[eval] imported motoko session %s -> chain %s (%d steps, %d tool calls)\n",
					imp.SessionLabel, imp.ChainID, imp.Steps, imp.ToolCalls)
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
		return true, nil
	}

	// Add error summary for failed benchmarks
	if result.Stderr != "" {
		benchSpan.SetAttributes(
			attribute.String("error.summary", telemetry.Truncate(result.Stderr, 200)),
			attribute.String("error.category", telemetry.CategorizeError(errors.New(result.Stderr))),
		)
	}
	benchSpan.SetStatus(codes.Error, "benchmark failed")

	// Return a descriptive (non-infra) error so the suite log surfaces the
	// reason instead of "<nil>" — mirrors standard mode below. The caller
	// treats any success=false as a failure regardless of the error value.
	switch {
	case !metrics.CompileOk:
		return false, fmt.Errorf("compilation failed (%s)", metrics.ErrorCategory)
	case !metrics.RuntimeOk:
		return false, fmt.Errorf("runtime error (%s)", metrics.ErrorCategory)
	default:
		return false, fmt.Errorf("output mismatch (%s)", metrics.ErrorCategory)
	}

}
