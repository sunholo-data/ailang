# M-EVAL-UNIFIED-EXEC-CODEGEN: Unified Exec-Based Code Generation

## Status: PLANNED (v0.7.0)

## Background

In v0.6.3, we fixed the dashboard hierarchy issue where `ailang run` spans appeared at ROOT level instead of nested under `eval.benchmark`. The fix added TRACEPARENT propagation to `AILANGRunner`.

This design doc captures the remaining architectural improvements deferred from that work.

## Problem

The eval harness uses two different code paths for AI operations:

1. **Code generation**: Direct API calls via `internal/ai/` providers
   - `ai/anthropic/`, `ai/openai/`, `ai/gemini/`
   - Each has its own instrumentation

2. **Code verification**: `ailang run` subprocess
   - Now has TRACEPARENT propagation (v0.6.3)
   - Unified telemetry through subprocess

This creates:
- Inconsistent telemetry instrumentation patterns
- Maintenance burden (two code paths to update)
- Different retry/error handling logic

## Solution

Route all AI operations through `ailang exec` as the unified entry point:

```
eval.suite
  └── eval.benchmark: fizzbuzz
        ├── ailang exec --api-only --json  (code generation)
        │     └── anthropic.generate
        └── ailang run                      (code verification)
              └── compile
```

## Implementation

### Phase 1: ExecCodeGenerator

**New file: `internal/eval_harness/exec_codegen.go`**

```go
// ExecCodeGenerator generates code by calling ailang exec --api-only
type ExecCodeGenerator struct {
    execPath string
    model    string
    taskID   string
    ctx      context.Context
}

// NewExecCodeGenerator creates a code generator that routes through ailang exec
func NewExecCodeGenerator(ctx context.Context, model, taskID string) *ExecCodeGenerator {
    return &ExecCodeGenerator{
        execPath: "ailang",
        model:    model,
        taskID:   taskID,
        ctx:      ctx,
    }
}

// Generate calls ailang exec --api-only --json to generate code
func (g *ExecCodeGenerator) Generate(prompt string) (*GenerateResult, error) {
    args := []string{
        "exec",
        "--api-only",
        "--json",
        "--model", g.model,
        prompt,
    }

    cmd := exec.Command(g.execPath, args...)

    // Propagate telemetry context
    env := os.Environ()
    env = telemetry.InjectTraceContext(g.ctx, env)
    if g.taskID != "" {
        env = append(env, "AILANG_PARENT_TASK_ID="+g.taskID)
    }
    cmd.Env = env

    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("exec failed: %w", err)
    }

    // Parse JSON output
    var result struct {
        Success      bool    `json:"success"`
        Output       string  `json:"output"`
        InputTokens  int     `json:"input_tokens"`
        OutputTokens int     `json:"output_tokens"`
        CostUSD      float64 `json:"cost_usd"`
    }
    if err := json.Unmarshal(output, &result); err != nil {
        return nil, fmt.Errorf("parse output: %w", err)
    }

    return &GenerateResult{
        Code:         extractCode(result.Output),
        InputTokens:  result.InputTokens,
        OutputTokens: result.OutputTokens,
        CostUSD:      result.CostUSD,
    }, nil
}
```

### Phase 2: ExecBasedRepairRunner

**New file: `internal/eval_harness/exec_repair_runner.go`**

```go
// ExecBasedRepairRunner combines ExecCodeGenerator with task-aware AILANGRunner
type ExecBasedRepairRunner struct {
    codegen *ExecCodeGenerator
    runner  *AILANGRunner
    spec    *BenchmarkSpec
}

// NewExecBasedRepairRunner creates a repair runner using exec for all operations
func NewExecBasedRepairRunner(ctx context.Context, model string, spec *BenchmarkSpec, taskID string) *ExecBasedRepairRunner {
    return &ExecBasedRepairRunner{
        codegen: NewExecCodeGenerator(ctx, model, taskID),
        runner:  NewAILANGRunnerWithTask(ctx, "", spec.Caps, taskID),
        spec:    spec,
    }
}

// RunWithRepair generates code, verifies it, and optionally repairs
func (r *ExecBasedRepairRunner) RunWithRepair(prompt string, maxAttempts int) (*BenchmarkResult, error) {
    // Generate code
    genResult, err := r.codegen.Generate(prompt)
    if err != nil {
        return nil, err
    }

    // Verify code
    runResult, err := r.runner.Run(genResult.Code, r.spec.Timeout)
    if err != nil {
        return nil, err
    }

    // If failed and repair enabled, attempt repair
    if !runResult.StdoutOk && maxAttempts > 1 {
        return r.attemptRepair(genResult, runResult, maxAttempts-1)
    }

    return &BenchmarkResult{
        GenerateResult: genResult,
        RunResult:      runResult,
    }, nil
}
```

### Phase 3: Feature Flag

**File: `cmd/ailang/eval_suite.go`**

```go
execCodegen := fs.Bool("exec-codegen", false, "Route code generation through ailang exec (experimental)")

// In runSingleBenchmark:
if *execCodegen {
    runner := NewExecBasedRepairRunner(ctx, model, spec, taskID)
    result, err := runner.RunWithRepair(prompt, repairAttempts)
} else {
    // Existing behavior (direct API calls)
}
```

## Benefits

1. **Unified telemetry** - Single instrumentation path through `ailang exec`
2. **Simpler architecture** - One entry point for all AI operations
3. **Consistent retry/logging** - Handled by exec command
4. **Easier testing** - Mock at exec boundary

## Migration Path

| Version | Status |
|---------|--------|
| v0.7.0 | Add `--exec-codegen` feature flag |
| v0.7.1 | Validate telemetry parity, enable by default |
| v0.8.0 | Remove legacy direct API codegen |

## Files to Create/Modify

| File | Change |
|------|--------|
| `internal/eval_harness/exec_codegen.go` | NEW |
| `internal/eval_harness/exec_repair_runner.go` | NEW |
| `cmd/ailang/eval_suite.go` | Add `--exec-codegen` flag |

## Prerequisites

Completed in v0.6.3:
- [x] TRACEPARENT propagation in `AILANGRunner`
- [x] `--json` flag in `exec.go`

## Related Work

- v0.6.3: Dashboard hierarchy fix (TRACEPARENT propagation)
- `internal/executor/`: Agentic CLI executors (Claude Code, Gemini CLI)
- `internal/ai/`: Direct API providers
