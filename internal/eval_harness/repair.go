package eval_harness

import (
	"context"
	"fmt"
	"time"
)

// RepairRunner orchestrates self-repair logic for eval benchmarks
type RepairRunner struct {
	agent         *AIAgent
	runner        LanguageRunner
	spec          *BenchmarkSpec
	timeout       time.Duration
	selfRepair    bool
	promptVersion string // Optional prompt version ID for A/B testing
}

// NewRepairRunner creates a new repair runner
func NewRepairRunner(agent *AIAgent, runner LanguageRunner, spec *BenchmarkSpec, timeout time.Duration, selfRepair bool) *RepairRunner {
	return &RepairRunner{
		agent:      agent,
		runner:     runner,
		spec:       spec,
		timeout:    timeout,
		selfRepair: selfRepair,
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

	// Attempt self-repair
	// Check both generated code and stderr for error patterns
	errCode, hint := CategorizeErrorWithCode(firstResult.Code, firstResult.RunResult.Stderr)
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
		firstResult.RunResult.Stderr,
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
		// Repair succeeded - update metrics to reflect successful run
		metrics.Code = repairResult.Code
		metrics.CompileOk = true
		metrics.RuntimeOk = true
		metrics.StdoutOk = true
		// Add repair tokens to totals
		metrics.InputTokens += repairResult.InputTokens
		metrics.OutputTokens += repairResult.OutputTokens
		metrics.TotalTokens += repairResult.InputTokens + repairResult.OutputTokens
	}

	return metrics, nil
}

// attemptResult contains results from a single attempt
type attemptResult struct {
	Code         string
	InputTokens  int
	OutputTokens int
	RunResult    *RunResult
	CompileOk    bool
	RuntimeOk    bool
	StdoutOk     bool
}

// runSingleAttempt executes one code generation + execution cycle
func (r *RepairRunner) runSingleAttempt(ctx context.Context, prompt string) (*attemptResult, error) {
	// Generate code using AI
	genResult, err := r.agent.GenerateCode(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	// Execute generated code
	runResult, err := r.runner.Run(genResult.Code, r.timeout)
	if err != nil {
		return nil, fmt.Errorf("code execution failed: %w", err)
	}

	// Check if output matches expected
	stdoutOk := CompareOutput(r.spec.ExpectedOut, runResult.Stdout)

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
	metrics.Stdout = result.RunResult.Stdout
	metrics.Stderr = result.RunResult.Stderr
	metrics.ExpectedStdout = r.spec.ExpectedOut
	metrics.Code = result.Code
}
