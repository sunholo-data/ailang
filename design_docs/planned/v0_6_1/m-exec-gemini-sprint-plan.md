# M-EXEC-GEMINI Sprint Plan: Gemini CLI Executor Implementation

**Sprint ID**: M-EXEC-GEMINI
**Duration**: 1 day (~6 hours)
**Target Version**: v0.6.1
**Risk Level**: Low (80% code reuse from Claude executor)

## Summary

Implement Gemini CLI executor for AILANG's agent system, enabling evaluation benchmarks and directive execution with Google's Gemini models. **Gemini 3 Flash will be the default model** due to its excellent performance/cost ratio.

## Why Gemini CLI First?

From the design doc analysis:

| Executor | Est. Hours | Complexity | Rationale |
|----------|------------|------------|-----------|
| **Gemini CLI** | **4-6 hours** | **Low** | Nearly identical CLI syntax to Claude Code |
| OpenAI Codex | 8-10 hours | Medium | Different flag syntax, no streaming NDJSON |
| Google Jules | 12-16 hours | High | REST API, not CLI wrapper |

**CLI Syntax Comparison** (shows 80% code reuse):
```bash
# Claude Code (current implementation)
claude --system-prompt "..." -p "task" --output-format stream-json --model haiku

# Gemini CLI (nearly identical!)
gemini --system-prompt "..." -p "task" --output-format stream-json -m gemini-3-flash-preview
```

## Current Status

**What exists:**
- `internal/agent/executor.go` - Claude-only DirectiveExecutor
- `internal/eval_harness/agent_runner_streaming.go` - Claude Code streaming (~289 LOC)
- `internal/ai/gemini/` - Gemini API client (for non-CLI use)
- `internal/eval_harness/models.yml` - Gemini 3 Flash already defined

**Gap:**
- No Gemini CLI executor implementation
- `agent_cli: null` for all Gemini models in models.yml
- Cannot run agent evals with Gemini models

## Milestones

### M1: Create Executor Interface & Factory (~1.5 hours, ~200 LOC)

**Files to create:**
- `internal/executor/executor.go` - Interface definitions
- `internal/executor/factory.go` - Executor factory

**Tasks:**
1. Define `Executor` interface with Execute, ExecuteStreaming, Name, Capabilities
2. Define `Task` struct (Directive, SystemPrompt, Workspace, Timeout, AllowedTools)
3. Define `Result` struct (Success, Output, DurationMS, NumTurns, CostUSD, TokenUsage)
4. Implement `ExecutorFactory` with registration pattern
5. Add HealthCheck for credential validation

**Acceptance Criteria:**
- [ ] Executor interface defined with all methods
- [ ] Task and Result types match existing DirectiveResult structure
- [ ] Factory can register and retrieve executors by name
- [ ] Unit tests for factory registration

### M2: Refactor Claude Executor to Interface (~1.5 hours, ~150 LOC)

**Files to modify:**
- `internal/executor/claude/claude.go` - New Claude executor implementation

**Tasks:**
1. Extract `RunHeadlessSessionStreaming` logic into ClaudeExecutor
2. Implement Executor interface for Claude
3. Add ClaudeExecutor to factory with name "claude"
4. Preserve backward compatibility in `internal/agent/executor.go`

**Acceptance Criteria:**
- [ ] ClaudeExecutor implements Executor interface
- [ ] Existing DirectiveExecutor continues working (calls ClaudeExecutor internally)
- [ ] Integration tests pass with Claude executor

### M3: Implement Gemini CLI Executor (~2 hours, ~250 LOC)

**Files to create:**
- `internal/executor/gemini/gemini.go` - Gemini CLI wrapper
- `internal/executor/gemini/gemini_test.go` - Tests

**Tasks:**
1. Copy ClaudeExecutor as starting point
2. Change CLI binary from `claude` to `gemini`
3. Adjust model flag from `--model` to `-m`
4. Parse streaming NDJSON (reuse Claude parser - identical format!)
5. Add `GEMINI_API_KEY` environment variable check
6. Implement cost calculation for Gemini 3 Flash pricing

**Gemini Executor Implementation (80% code reuse):**
```go
func (e *GeminiExecutor) buildCommand(task *Task) *exec.Cmd {
    args := []string{
        "-p", task.Directive,                    // Same as Claude!
        "--output-format", "stream-json",        // Same as Claude!
    }
    if task.SystemPrompt != "" {
        args = append(args, "--system-prompt", task.SystemPrompt)  // Same as Claude!
    }
    args = append(args, "-m", e.model)           // -m instead of --model

    cmd := exec.Command("gemini", args...)       // Only change: "claude" -> "gemini"
    cmd.Dir = task.Workspace
    return cmd
}
// Streaming parser: REUSE ENTIRE Claude NDJSON parser!
```

**Acceptance Criteria:**
- [ ] GeminiExecutor implements Executor interface
- [ ] `gemini -p "test" --output-format stream-json` works in headless mode
- [ ] Streaming events parsed correctly (reuses Claude parser)
- [ ] Cost calculation uses Gemini 3 Flash pricing ($0.50/$3.00 per 1M)
- [ ] Health check validates GEMINI_API_KEY or gcloud auth

### M4: Update models.yml & Integration (~1 hour, ~50 LOC)

**Files to modify:**
- `internal/eval_harness/models.yml` - Add agent_cli support
- `internal/agent/executor.go` - Use executor factory

**Tasks:**
1. Update gemini-3-flash entry with `agent_cli: "gemini"`
2. Add `agent_model_name: "gemini-3-flash-preview"`
3. Update DirectiveExecutor to optionally use Gemini
4. Add `--executor` flag concept (or environment variable)

**models.yml changes:**
```yaml
gemini-3-flash:
  api_name: "gemini-3-flash-preview"
  provider: "google"
  agent_cli: "gemini"  # NEW - enables agent evals
  agent_model_name: "gemini-3-flash-preview"  # NEW
  # ... rest unchanged
```

**Acceptance Criteria:**
- [ ] `agent_cli: "gemini"` added to gemini-3-flash
- [ ] DirectiveExecutor can use Gemini when AILANG_EXECUTOR=gemini
- [ ] Eval harness recognizes Gemini as valid executor

## Configuration

**Default executor:** Gemini 3 Flash (best performance/cost ratio)

**Environment variables:**
```bash
# Select executor (default: gemini)
AILANG_EXECUTOR=gemini

# Gemini auth (one of these required)
GEMINI_API_KEY=xxx              # Personal API key
# OR
GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json  # Service account
# OR
gcloud auth login               # User credentials
```

**Example usage:**
```bash
# Run directive with Gemini (default)
ailang agent exec "Create a hello world function"

# Explicitly use Claude
AILANG_EXECUTOR=claude ailang agent exec "Create a hello world function"

# Run eval benchmark with Gemini
ailang eval-suite --models gemini-3-flash
```

## Success Metrics

- [ ] Gemini CLI executor passes all unit tests
- [ ] Can execute directive with Gemini 3 Flash
- [ ] Streaming output visible in DEBUG_AGENT mode
- [ ] Cost tracking works with Gemini pricing
- [ ] models.yml updated with agent_cli support
- [ ] Backward compatibility maintained (Claude still works)

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Gemini CLI not installed | Health check returns clear error with install instructions |
| Different NDJSON event format | Verify format with `gemini -p "test" --output-format stream-json` before implementation |
| Auth issues | Support multiple auth methods (API key, gcloud, ADC) |

## Testing Plan

**Unit tests:**
- Factory registration and retrieval
- Task/Result struct validation
- Cost calculations

**Integration tests (require Gemini CLI):**
- Simple directive execution
- Streaming event parsing
- Timeout handling

**Manual verification:**
```bash
# Verify Gemini CLI is installed and works
gemini -p "Say hello" --output-format stream-json

# Run test directive
DEBUG_AGENT=1 ailang agent exec "Create a function that adds two numbers"
```

## Files Summary

**New files (~600 LOC):**
- `internal/executor/executor.go` (~100 LOC)
- `internal/executor/factory.go` (~80 LOC)
- `internal/executor/claude/claude.go` (~200 LOC)
- `internal/executor/gemini/gemini.go` (~180 LOC)
- `internal/executor/gemini/gemini_test.go` (~40 LOC)

**Modified files (~80 LOC):**
- `internal/eval_harness/models.yml` (~10 lines)
- `internal/agent/executor.go` (~50 LOC changes)
- `internal/eval_harness/agent_runner.go` (~20 LOC - optional)

## Timeline

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| M1: Interface & Factory | 1.5h | Executor interface, factory |
| M2: Claude Refactor | 1.5h | ClaudeExecutor implements interface |
| M3: Gemini Executor | 2h | GeminiExecutor with streaming |
| M4: Integration | 1h | models.yml update, testing |
| **Total** | **6h** | Full Gemini CLI support |

---

**Document created**: 2025-12-17
**Based on**: design_docs/planned/v0_6_1/m-exec-multi-executor-support.md (Phases 1-3)
