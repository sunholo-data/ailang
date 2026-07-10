package eval_harness

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RepairRunner orchestrates self-repair logic for eval benchmarks
type RepairRunner struct {
	agent         *AIAgent
	runner        LanguageRunner
	spec          *BenchmarkSpec
	timeout       time.Duration
	selfRepair    bool
	promptVersion string        // Optional prompt version ID for A/B testing
	verify        bool          // M-CONTRACT-EVAL: run Z3 verification
	verifyTimeout time.Duration // M-CONTRACT-EVAL: per-function Z3 timeout
}

// NewRepairRunner creates a new repair runner
func NewRepairRunner(agent *AIAgent, runner LanguageRunner, spec *BenchmarkSpec, timeout time.Duration, selfRepair bool) *RepairRunner {
	return &RepairRunner{
		agent:         agent,
		runner:        runner,
		spec:          spec,
		timeout:       timeout,
		selfRepair:    selfRepair,
		verifyTimeout: 5 * time.Second, // default
	}
}

// SetVerify enables contract verification (M-CONTRACT-EVAL)
func (r *RepairRunner) SetVerify(verify bool, timeout time.Duration) {
	r.verify = verify
	if timeout > 0 {
		r.verifyTimeout = timeout
	}
}

// SetPromptVersion sets the prompt version ID for metrics tracking
func (r *RepairRunner) SetPromptVersion(version string) {
	r.promptVersion = version
}

// Run executes the benchmark with optional self-repair
func (r *RepairRunner) Run(ctx context.Context, prompt string) (*RunMetrics, error) {
	metrics := NewRunMetrics(r.spec.ID, r.runner.Language(), r.agent.friendlyName, r.agent.seed)
	metrics.PromptVersion = r.promptVersion // Track prompt version for A/B testing
	metrics.EvalMode = EvalModeStandard     // Mark as standard evaluation (0-shot + self-repair)

	// First attempt
	firstResult, err := r.runSingleAttempt(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("first attempt failed: %w", err)
	}

	// Populate metrics from first attempt
	r.populateMetrics(metrics, firstResult)
	metrics.FirstAttemptOk = firstResult.CompileOk && firstResult.RuntimeOk && firstResult.StdoutOk

	// If first attempt succeeded or self-repair is disabled, return
	if metrics.FirstAttemptOk || !r.selfRepair {
		return metrics, nil
	}

	// Determine error category and repair hint
	var errCode ErrCode
	var hint *RepairHint
	var repairStderr string

	// M-CONTRACT-EVAL: If verify is active, code compiles, and spec has contracts,
	// try Z3 verification as an additional error signal
	if r.verify && r.spec.ContractSpec != "" && firstResult.CompileOk && firstResult.RunResult.WorkspaceDir != "" {
		verifyResult, rawJSON, verifyErr := RunAICheck("", firstResult.RunResult.WorkspaceDir+"/benchmark/solution.ail", r.verifyTimeout)
		if verifyErr == nil && verifyResult != nil {
			PopulateVerifyMetrics(metrics, verifyResult, rawJSON)
			if verifyResult.Verify.Counterexample > 0 {
				// Z3 found counterexample — use it as the repair error
				errCode = VERIFY_COUNTEREXAMPLE
				hint = FormatZ3RepairHint(rawJSON)
				repairStderr = rawJSON
			}
		}
	}

	// Constraint violations carry their own precise repair feedback (byte
	// deltas, offending lines) — route them to the repair attempt directly.
	if hint == nil && len(firstResult.ConstraintViolations) > 0 {
		errCode = CONSTRAINT_VIOLATION
		hint = &RepairHint{
			Title: "Generated source violates the benchmark's source constraints",
			Why:   "This benchmark grades the program TEXT itself; the code was rejected before execution.",
			How:   "Adjust the source to satisfy every constraint listed in the error, then re-check byte counts and banned characters over the ENTIRE source including comments and identifiers.",
		}
		repairStderr = firstResult.RunResult.Stderr
	}

	// If no Z3 error found (or verify not active), fall back to standard error categorization
	if hint == nil {
		errCode, hint = CategorizeErrorWithCode(firstResult.Code, firstResult.RunResult.Stderr, r.runner.Language())
		repairStderr = firstResult.RunResult.Stderr
	}

	if hint == nil {
		// Unknown error, can't repair
		return metrics, nil
	}

	// We have a categorized error and repair hint
	metrics.ErrCode = string(errCode)
	metrics.RepairUsed = true

	// Build repair prompt with failed code and error context
	repairPrompt := prompt + "\n\n" + FormatRepairPrompt(
		errCode,
		hint,
		r.spec.ID,
		r.runner.Language(),
		firstResult.Code,
		repairStderr,
	)

	// Second attempt with repair guidance
	repairResult, err := r.runSingleAttempt(ctx, repairPrompt)
	if err != nil {
		// Repair attempt failed to execute, but not a failure - just log
		return metrics, nil
	}

	// Update metrics with repair results
	metrics.RepairTokensIn = repairResult.InputTokens
	metrics.RepairTokensOut = repairResult.OutputTokens
	metrics.RepairOk = repairResult.CompileOk && repairResult.RuntimeOk && repairResult.StdoutOk

	if metrics.RepairOk {
		// Repair succeeded - update metrics to reflect the REPAIRED run.
		metrics.Code = repairResult.Code
		metrics.CompileOk = true
		metrics.RuntimeOk = true
		metrics.StdoutOk = true
		// Store the repaired attempt's actual output + clear the error so the
		// PERSISTED stdout matches the graded result. Without this, the first
		// (failed) attempt's stdout was stored alongside a passing stdout_ok — a
		// data-integrity bug surfaced by the M-EVAL-OUTPUT-NORMALIZE re-grade
		// (a verbose first attempt looked like a "stored pass with wrong stdout").
		if repairResult.RunResult != nil {
			metrics.Stdout = repairResult.RunResult.Stdout
			metrics.Stderr = repairResult.RunResult.Stderr
		}
		metrics.ErrorCategory = "none"
		// Add repair tokens to totals
		metrics.InputTokens += repairResult.InputTokens
		metrics.OutputTokens += repairResult.OutputTokens
		metrics.TotalTokens += repairResult.InputTokens + repairResult.OutputTokens
	}

	return metrics, nil
}

// attemptResult contains results from a single attempt
type attemptResult struct {
	Code                 string
	InputTokens          int
	OutputTokens         int
	RunResult            *RunResult
	CompileOk            bool
	RuntimeOk            bool
	StdoutOk             bool
	ConstraintViolations []string // non-empty: source rejected before execution
}

// runSingleAttempt executes one code generation + execution cycle
func (r *RepairRunner) runSingleAttempt(ctx context.Context, prompt string) (*attemptResult, error) {
	// Generate code using AI
	genResult, err := r.agent.GenerateCode(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	// Source-constraint gate (constrained-construction benchmarks): the
	// constraint is on the program TEXT, so it's checked before execution and
	// a violation fails the attempt regardless of what the code would print.
	if r.spec.SourceConstraints != nil {
		if violations := r.spec.SourceConstraints.Check(genResult.Code); len(violations) > 0 {
			stderrMsg := "SOURCE CONSTRAINT VIOLATION (code was not executed):\n- " + strings.Join(violations, "\n- ")
			return &attemptResult{
				Code:                 genResult.Code,
				InputTokens:          genResult.InputTokens,
				OutputTokens:         genResult.OutputTokens,
				RunResult:            &RunResult{Stderr: stderrMsg},
				CompileOk:            false,
				RuntimeOk:            false,
				StdoutOk:             false,
				ConstraintViolations: violations,
			}, nil
		}
	}

	// Execute generated code
	runResult, err := r.runner.Run(genResult.Code, r.timeout)
	if err != nil {
		return nil, fmt.Errorf("code execution failed: %w", err)
	}

	// Check if output matches expected. Quine grading compares stdout against
	// the submitted source itself — the only deterministic grader for a quine.
	var stdoutOk bool
	if r.spec.Grading == "quine" {
		stdoutOk = NormalizeSource(runResult.Stdout) == NormalizeSource(genResult.Code)
	} else {
		stdoutOk = CompareOutput(r.spec.ExpectedOut, runResult.Stdout)
	}

	return &attemptResult{
		Code:         genResult.Code,
		InputTokens:  genResult.InputTokens,
		OutputTokens: genResult.OutputTokens,
		RunResult:    runResult,
		CompileOk:    runResult.CompileOk,
		RuntimeOk:    runResult.RuntimeOk,
		StdoutOk:     stdoutOk,
	}, nil
}

// populateMetrics fills in RunMetrics from an attemptResult
func (r *RepairRunner) populateMetrics(metrics *RunMetrics, result *attemptResult) {
	metrics.InputTokens = result.InputTokens
	metrics.OutputTokens = result.OutputTokens
	metrics.TotalTokens = result.InputTokens + result.OutputTokens
	metrics.CostUSD = CalculateCostWithBreakdown(metrics.Model, metrics.InputTokens, metrics.OutputTokens)

	metrics.CompileOk = result.CompileOk
	metrics.RuntimeOk = result.RuntimeOk
	metrics.StdoutOk = result.StdoutOk

	metrics.DurationMs = result.RunResult.Duration.Milliseconds()
	metrics.CompileMs = result.RunResult.CompileTime.Milliseconds()
	metrics.ExecuteMs = result.RunResult.ExecuteTime.Milliseconds()

	metrics.ErrorCategory = CategorizeError(result.CompileOk, result.RuntimeOk, result.StdoutOk)
	if len(result.ConstraintViolations) > 0 {
		metrics.ErrorCategory = ErrorCategoryConstraint
		metrics.ConstraintViolations = result.ConstraintViolations
	}
	metrics.Stdout = result.RunResult.Stdout
	metrics.Stderr = result.RunResult.Stderr
	metrics.ExpectedStdout = r.spec.ExpectedOut
	metrics.Code = result.Code
}
