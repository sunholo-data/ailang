# M-EXEC: Multi-Executor Support for AI Coding Agents

**Status**: Planned
**Target**: v0.6.0 (postponed from v0.5.10)
**Priority**: P1 (Medium-High)
**Estimated**: 2 weeks (~38 hours for MVP, +14 hours optional for Jules)
**Dependencies**: None (current Claude Code executor is working)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Infrastructure change, no language syntax impact |
| Preserve Semantic Clarity | + | +1 | Abstracts executor details, cleaner interface |
| Increase Determinism | + | +1 | Consistent result format across all executors |
| Lower Token Cost | + | +1 | Enables model selection for cost optimization |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Rationale:** This feature enables AILANG to leverage the best AI coding agent for each task, optimizing for cost, speed, and capability. It abstracts executor details behind a unified interface, enabling deterministic benchmarking across providers.

## Problem Statement

**Current State:**
- AILANG agent system only supports Claude Code as an executor
- The `DirectiveExecutor` in `internal/agent/executor.go` is tightly coupled to Claude Code CLI
- Agent evals (`internal/eval_harness/agent_runner.go`) only benchmark Claude performance
- No ability to compare AI coding agent performance across providers
- Users locked into single provider pricing/quotas

**Impact:**
- Cannot evaluate which AI agent performs best on AILANG code generation
- No cost optimization through provider selection
- Single point of failure if Claude Code has issues
- Missing competitive benchmarking data for AILANG adoption

## Goals

**Primary Goal:** Enable AILANG's agent system and eval harness to use multiple AI coding agent executors (Claude Code, OpenAI Codex, Google Jules, Gemini CLI) through a unified interface.

**Success Metrics:**
- All 4 executors pass basic directive execution tests
- Eval harness can run benchmarks across all providers
- Unified result format captures metrics from all executors
- Cost tracking normalized across providers (USD)
- Workspace management consistent across executors

## Research Summary

### Available AI Coding Agent Executors

#### 1. Claude Code (Current Implementation)
- **Interface**: CLI with headless JSON streaming
- **Command**: `claude -p <prompt> --output-format stream-json`
- **Auth**: Anthropic API key or ChatGPT subscription
- **Key Features**: Streaming events, tool allowlisting, session management
- **Pricing**: Per-token via API key, or included in Pro subscription
- **Docs**: [code.claude.com/docs/headless](https://code.claude.com/docs/headless)

#### 2. OpenAI Codex CLI
- **Interface**: CLI (`codex`) with optional TypeScript SDK (`@openai/codex-sdk`)
- **Install**: `npm i -g @openai/codex` or `brew install codex`
- **Auth**: `OPENAI_API_KEY` env var or ChatGPT subscription login
- **Key Features**: Thread persistence, structured output with JSON schema, MCP support
- **CLI Flags**:
  | Flag | Purpose | Example |
  |------|---------|---------|
  | `-q, --quiet` | Non-interactive mode | `codex -q "task"` |
  | `--json` | JSON output format | `codex -q --json "task"` |
  | `-a, --approval-mode` | Autonomy level | `codex -a full-auto "task"` |
  | `-m, --model` | Model selection | `codex -m o4-mini "task"` |
- **Headless Example**:
  ```bash
  codex -q --json -a full-auto "fix the bug in parser.go"
  ```
- **CI Environment**: `CODEX_QUIET_MODE=1` for headless operation
- **Pricing**: GPT-5-Codex rates via API, or included in Plus/Pro subscriptions
- **Docs**: [developers.openai.com/codex/cli](https://developers.openai.com/codex/cli/)

#### 3. Google Jules API
- **Interface**: REST API + CLI (`jules remote`)
- **Auth**: API key from Jules web app (X-Goog-Api-Key header)
- **Prerequisites**: Jules GitHub app installed on repository
- **Key Concepts**:
  - **Source**: GitHub repository (requires GitHub app installation)
  - **Session**: Continuous work unit with prompt and source context
  - **Activity**: Individual actions within a session (plans, messages, progress)
- **Key Endpoints**:
  | Endpoint | Method | Purpose |
  |----------|--------|---------|
  | `/v1alpha/sources` | GET | List connected sources |
  | `/v1alpha/sessions` | POST | Create new session |
  | `/v1alpha/sessions/{id}:approvePlan` | POST | Approve session plans |
  | `/v1alpha/sessions/{id}/activities` | GET | List session activities |
  | `/v1alpha/sessions/{id}:sendMessage` | POST | Send messages to agent |
- **CLI Example**: `jules remote new --source repo-id --prompt "Fix the bug"`
- **Pricing**: Free tier (15 daily tasks), AI Pro ($19.99/mo), AI Ultra ($124.99/mo)
- **Docs**: [developers.google.com/jules/api](https://developers.google.com/jules/api)

#### 4. Gemini CLI (RECOMMENDED NEXT)
- **Interface**: Open-source CLI, integrates with Gemini Code Assist
- **Install**: `npm i -g @google/gemini-cli` or `brew install gemini-cli`
- **Auth**: Google login, `GEMINI_API_KEY`, or Vertex AI credentials
- **Key Features**: 1M token context window, MCP server support, agent mode
- **CLI Flags** (nearly identical to Claude Code!):
  | Flag | Purpose | Example |
  |------|---------|---------|
  | `-p` | Non-interactive mode | `gemini -p "task"` |
  | `--output-format json` | JSON output | `gemini -p "task" --output-format json` |
  | `--output-format stream-json` | Streaming NDJSON | `gemini -p "task" --output-format stream-json` |
  | `-m` | Model selection | `gemini -m gemini-2.5-flash "task"` |
  | `--system-prompt` | System context | `gemini --system-prompt "..." -p "task"` |
- **Headless Example** (compare to Claude!):
  ```bash
  # Claude Code
  claude -p "fix the bug" --output-format stream-json

  # Gemini CLI (nearly identical!)
  gemini -p "fix the bug" --output-format stream-json
  ```
- **Pricing**: Free tier with personal Google account, Standard/Enterprise for teams
- **Models**: Gemini 2.5 Pro, Gemini 2.5 Flash (both GA)
- **Docs**: [github.com/google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli)

### Comparison Matrix

| Feature | Claude Code | OpenAI Codex | Google Jules | Gemini CLI |
|---------|-------------|--------------|--------------|------------|
| **Interface** | CLI (JSON stream) | CLI + SDK | REST API + CLI | CLI |
| **Session State** | File-based sessions | Thread persistence | Session/Activity | Agent mode |
| **Workspace Support** | `--add-dir` | Local dirs | GitHub repos only | Local dirs |
| **Streaming** | Yes (NDJSON) | Limited | Activities poll | Yes (NDJSON) |
| **Tool Control** | `--allowedTools` | MCP servers | N/A | MCP servers |
| **Cost Tracking** | In result JSON | SDK provides | API returns | Manual calc |
| **Approval Flow** | Permission modes | Sandbox modes | Plan approval | N/A |

### Implementation Complexity Analysis

**Prioritization based on implementation ease:**

| Rank | Executor | Est. Hours | Complexity | Rationale |
|------|----------|------------|------------|-----------|
| 1 | **Gemini CLI** | **4-6 hours** | **Low** | Nearly identical CLI syntax to Claude Code (`-p`, `--output-format stream-json`). Can reuse 80%+ of streaming parser. |
| 2 | OpenAI Codex | 8-10 hours | Medium | Different flag syntax (`-q`, `--json`), no streaming NDJSON, need to parse different JSON schema. |
| 3 | Google Jules | 12-16 hours | High | REST API (not CLI wrapper), requires HTTP client, session polling, GitHub app prerequisite. |

**CLI Syntax Comparison:**

```bash
# Claude Code (current implementation)
claude --system-prompt "..." -p "task" --output-format stream-json --model haiku

# Gemini CLI (nearly identical - EASIEST TO ADD)
gemini --system-prompt "..." -p "task" --output-format stream-json -m gemini-2.5-flash

# OpenAI Codex (different syntax)
codex -q --json -a full-auto -m gpt5-codex "task"

# Google Jules (REST API, not CLI)
curl -X POST https://jules.googleapis.com/v1alpha/sessions \
  -H "X-Goog-Api-Key: $JULES_API_KEY" \
  -d '{"prompt": "task", "source": "repo-id"}'
```

**Recommendation:** Implement Gemini CLI first (Phase 3) since it requires minimal code changes from the existing Claude executor.

## Solution Design

### Overview

Create an abstract `Executor` interface that normalizes the differences between AI coding agents. Each provider gets a concrete implementation that handles authentication, API specifics, and result mapping. The eval harness and agent system use the interface, enabling hot-swapping of executors.

### Architecture

```
                    ┌─────────────────────────────────────┐
                    │         ExecutorFactory             │
                    │  NewExecutor(config) -> Executor    │
                    └─────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │        Executor Interface      │
                    │  Execute(task, opts) -> Result │
                    │  Capabilities() -> []string    │
                    │  Name() -> string              │
                    │  CostPerToken() -> CostModel   │
                    └───────────────────────────────┘
                                    │
           ┌────────────┬───────────┼───────────┬────────────┐
           ▼            ▼           ▼           ▼            ▼
    ┌────────────┐ ┌────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
    │ClaudeExec  │ │ CodexExec  │ │JulesExec │ │GeminiExec│ │ MockExec │
    │(current)   │ │(TypeScript)│ │(REST API)│ │  (CLI)   │ │ (tests)  │
    └────────────┘ └────────────┘ └──────────┘ └──────────┘ └──────────┘
```

**Components:**

1. **Executor Interface** (`internal/executor/executor.go`): Common interface all executors implement
2. **ExecutorFactory** (`internal/executor/factory.go`): Creates executor instances from config
3. **ExecutorConfig** (`internal/executor/config.go`): Unified configuration schema
4. **Result Types** (`internal/executor/result.go`): Normalized result structure
5. **Provider Implementations**: One package per provider under `internal/executor/`

### Core Interface Design

```go
// internal/executor/executor.go

package executor

import (
    "context"
    "time"
)

// Executor is the common interface for all AI coding agent executors
type Executor interface {
    // Name returns the executor identifier (e.g., "claude", "codex", "jules", "gemini")
    Name() string

    // Execute runs a task and returns the result
    Execute(ctx context.Context, task *Task) (*Result, error)

    // ExecuteStreaming runs a task with real-time event callbacks
    ExecuteStreaming(ctx context.Context, task *Task, handler EventHandler) (*Result, error)

    // Capabilities returns the list of features this executor supports
    Capabilities() []Capability

    // CostModel returns pricing information for cost calculations
    CostModel() *CostModel

    // HealthCheck verifies the executor is configured and accessible
    HealthCheck(ctx context.Context) error

    // Close releases any resources held by the executor
    Close() error
}

// Task represents a coding task to execute
type Task struct {
    ID           string            // Unique task identifier
    Directive    string            // The instruction/prompt
    SystemPrompt string            // Optional system-level context
    Workspace    string            // Working directory (local path)
    Timeout      time.Duration     // Execution timeout
    AllowedTools []string          // Tools the agent can use
    Metadata     map[string]string // Provider-specific options
}

// Result is the normalized execution result
type Result struct {
    Success      bool              // Overall success
    Output       string            // Final text output from agent
    Error        string            // Error message if failed

    // Metrics
    DurationMS   int               // Total execution time
    NumTurns     int               // Conversation turns
    CostUSD      float64           // Total cost in USD

    // Token usage
    InputTokens  int
    OutputTokens int
    CacheHits    int               // Provider-specific caching

    // Session info
    SessionID    string            // Provider's session identifier
    Transcript   string            // Full conversation log

    // Artifacts
    FilesCreated []string          // Files created in workspace
    FilesModified []string         // Files modified

    // Provider-specific data
    ProviderData map[string]any    // Raw provider response data
}

// Capability flags for feature detection
type Capability string

const (
    CapStreaming       Capability = "streaming"
    CapToolControl     Capability = "tool_control"
    CapSessionResume   Capability = "session_resume"
    CapApprovalFlow    Capability = "approval_flow"
    CapGitHubIntegration Capability = "github_integration"
    CapLocalWorkspace  Capability = "local_workspace"
    CapStructuredOutput Capability = "structured_output"
)

// EventHandler receives streaming events during execution
type EventHandler interface {
    OnTurnStart(turnNum int)
    OnText(text string)
    OnToolUse(toolName string, input string)
    OnToolResult(toolName string, output string)
    OnTurnEnd(turnNum int)
    OnError(err error)
}

// CostModel contains pricing information
type CostModel struct {
    ProviderName     string
    InputTokenCost   float64  // Cost per 1K input tokens
    OutputTokenCost  float64  // Cost per 1K output tokens
    CacheReadCost    float64  // Cost per 1K cache read tokens
    CacheWriteCost   float64  // Cost per 1K cache write tokens
    MinimumCharge    float64  // Minimum per-request charge
}
```

### Implementation Plan

**Phase 1: Core Interface & Factory** (~10 hours)
- [ ] Create `internal/executor/` package structure
- [ ] Define `Executor` interface and related types
- [ ] Implement `ExecutorFactory` with registration pattern
- [ ] Create `ExecutorConfig` with YAML/JSON config loading
- [ ] Add config schema to `~/.ailang/config.yaml`
- [ ] Implement `MockExecutor` for testing
- [ ] Unit tests for factory and config

**Phase 2: Refactor Claude Executor** (~8 hours)
- [ ] Extract current `RunHeadlessSessionStreaming` into `ClaudeExecutor`
- [ ] Implement `Executor` interface for Claude
- [ ] Map Claude's JSON streaming to `EventHandler`
- [ ] Preserve backward compatibility in `internal/agent/executor.go`
- [ ] Add `ClaudeExecutor` config options (model, permission mode)
- [ ] Integration tests with existing eval harness

**Phase 3: Gemini CLI Executor (EASIEST - DO FIRST)** (~4-6 hours)
- [ ] Create `internal/executor/gemini/` package
- [ ] Copy Claude executor as starting point (80% code reuse)
- [ ] Change CLI binary from `claude` to `gemini`
- [ ] Adjust flags: `-p` (same), `--output-format stream-json` (same), `-m` (same)
- [ ] Parse streaming NDJSON (reuse Claude parser)
- [ ] Add `GEMINI_API_KEY` auth support
- [ ] Cost calculation for Gemini 2.5 Pro/Flash
- [ ] Integration tests with AILANG benchmarks

**Gemini Executor Implementation Sketch** (shows 80% code reuse):
```go
// internal/executor/gemini/gemini.go
func (e *GeminiExecutor) buildCommand(task *Task) *exec.Cmd {
    args := []string{
        "-p", task.Directive,                    // Same as Claude!
        "--output-format", "stream-json",        // Same as Claude!
    }
    if task.SystemPrompt != "" {
        args = append(args, "--system-prompt", task.SystemPrompt)  // Same as Claude!
    }
    if e.model != "" {
        args = append(args, "-m", e.model)       // Same as Claude! (just different flag)
    }

    cmd := exec.Command("gemini", args...)       // Only change: "claude" -> "gemini"
    cmd.Dir = task.Workspace
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("GEMINI_API_KEY=%s", e.apiKey))
    return cmd
}
// Streaming parser: REUSE ENTIRE Claude NDJSON parser!
```

**Phase 4: OpenAI Codex Executor** (~8-10 hours)
- [ ] Create `internal/executor/codex/` package
- [ ] Implement Codex CLI wrapper using `codex -q --json`
- [ ] Parse Codex JSON output to `Result` (different schema than Claude)
- [ ] Handle quiet mode flags (`-q`, `CODEX_QUIET_MODE=1`)
- [ ] Map approval modes (`-a full-auto`, `-a auto-edit`)
- [ ] Cost calculation from GPT-5-Codex rates
- [ ] Integration tests with AILANG eval benchmarks

**Phase 5: Google Jules Executor (HARDEST - DO LAST)** (~12-16 hours)
- [ ] Create `internal/executor/jules/` package
- [ ] Implement REST API client for Jules v1alpha (not CLI wrapper!)
- [ ] Handle session creation via POST `/v1alpha/sessions`
- [ ] Implement activity polling loop (GET `/v1alpha/sessions/{id}/activities`)
- [ ] Map Jules plan approval to `approval_flow` capability
- [ ] Handle GitHub-only workspace limitation (document as prerequisite)
- [ ] Cost estimation from Jules pricing tiers
- [ ] Integration tests (requires GitHub app setup - may skip in CI)

**Phase 6: Eval Harness Integration** (~6 hours)
- [ ] Update `AgentBenchmarkConfig` to support executor selection
- [ ] Modify `RunAgentBenchmark` to use executor interface
- [ ] Add multi-executor benchmark comparison mode
- [ ] Update `ailang eval-suite` to accept `--executor` flag
- [ ] Add executor comparison reports
- [ ] Update dashboard JSON schema for multi-provider data

### Files to Modify/Create

**New files:**
- `internal/executor/executor.go` - Interface definitions (~150 LOC)
- `internal/executor/factory.go` - Executor factory (~100 LOC)
- `internal/executor/config.go` - Configuration types (~80 LOC)
- `internal/executor/result.go` - Result normalization (~100 LOC)
- `internal/executor/mock.go` - Mock executor for tests (~80 LOC)
- `internal/executor/claude/claude.go` - Claude executor (~300 LOC)
- `internal/executor/codex/codex.go` - Codex executor (~350 LOC)
- `internal/executor/jules/jules.go` - Jules executor (~400 LOC)
- `internal/executor/jules/client.go` - Jules REST client (~200 LOC)
- `internal/executor/gemini/gemini.go` - Gemini executor (~250 LOC)
- `internal/executor/codex/codex_test.go` - Codex tests (~200 LOC)
- `internal/executor/jules/jules_test.go` - Jules tests (~200 LOC)
- `internal/executor/gemini/gemini_test.go` - Gemini tests (~150 LOC)

**Modified files:**
- `internal/agent/executor.go` - Use executor interface (~50 LOC changes)
- `internal/eval_harness/agent_runner.go` - Executor selection (~100 LOC changes)
- `internal/eval_harness/agent_runner_streaming.go` - Extract to Claude executor (~-200 LOC)
- `cmd/ailang/eval_suite.go` - Add `--executor` flag (~30 LOC changes)
- `internal/eval_harness/models.yml` - Add executor configs (~50 lines)

**Total: ~2,500 new LOC, ~380 modified LOC**

## Examples

### Example 1: Directive Execution with Different Executors

**Before (Claude only):**
```go
executor := agent.NewDirectiveExecutor(workspaceBase)
result, err := executor.Execute("Create a hello world program")
```

**After (Multi-executor):**
```go
import "github.com/sunholo/ailang/internal/executor"

// Load config from ~/.ailang/config.yaml or environment
cfg := executor.LoadConfig()

// Create executor factory
factory := executor.NewFactory(cfg)

// Get Claude executor (default)
claude, _ := factory.GetExecutor("claude")
result, _ := claude.Execute(ctx, &executor.Task{
    Directive: "Create a hello world program",
    Workspace: workspaceBase,
})

// Or use Codex
codex, _ := factory.GetExecutor("codex")
result, _ := codex.Execute(ctx, &executor.Task{
    Directive: "Create a hello world program",
    Workspace: workspaceBase,
})
```

### Example 2: Configuration Schema

**~/.ailang/config.yaml:**
```yaml
executors:
  # Default executor for agent commands
  default: claude

  claude:
    enabled: true
    model: haiku  # or sonnet, opus
    permission_mode: bypassPermissions
    allowed_tools:
      - Bash
      - Read
      - Write
      - Edit
      - Grep

  codex:
    enabled: true
    # Uses CODEX_API_KEY from environment
    sandbox: workspace-write  # or full-access
    model: gpt5-codex

  jules:
    enabled: false  # Requires GitHub app setup
    # Uses JULES_API_KEY from environment
    default_source: sunholo-data/ailang
    auto_approve_plans: false

  gemini:
    enabled: true
    # Uses GOOGLE_APPLICATION_CREDENTIALS or personal account
    model: gemini-2.5-pro
    context_window: 1000000  # 1M tokens
```

### Example 3: Eval Benchmark Comparison

**Before:**
```bash
# Only runs Claude
ailang eval-suite --models haiku
```

**After:**
```bash
# Run with specific executor
ailang eval-suite --executor codex --models gpt5-codex

# Compare executors on same benchmarks
ailang eval-compare-executors \
  --executors claude,codex,gemini \
  --benchmarks medium \
  --output eval_results/executor_comparison/

# Generate comparison report
ailang eval-report eval_results/executor_comparison/ v0.5.10 --format=json
```

### Example 4: Streaming Events

```go
// Create event handler
handler := &MyEventHandler{}

// Execute with streaming
result, err := executor.ExecuteStreaming(ctx, task, handler)

type MyEventHandler struct {
    turns int
}

func (h *MyEventHandler) OnTurnStart(turnNum int) {
    h.turns = turnNum
    fmt.Printf("[TURN %d] Starting...\n", turnNum)
}

func (h *MyEventHandler) OnText(text string) {
    fmt.Print(text)  // Stream to terminal
}

func (h *MyEventHandler) OnToolUse(name, input string) {
    fmt.Printf("[TOOL] %s\n", name)
}

func (h *MyEventHandler) OnToolResult(name, output string) {
    fmt.Printf("[RESULT] %s: %s\n", name, output[:100])
}

func (h *MyEventHandler) OnTurnEnd(turnNum int) {
    fmt.Printf("[TURN %d] Complete\n", turnNum)
}

func (h *MyEventHandler) OnError(err error) {
    fmt.Printf("[ERROR] %v\n", err)
}
```

## Success Criteria

- [ ] All 4 executors (Claude, Codex, Jules, Gemini) implement `Executor` interface
- [ ] `ExecutorFactory` can create any configured executor
- [ ] Existing Claude-based agent system continues working (backward compat)
- [ ] `ailang eval-suite --executor <name>` runs benchmarks with specified executor
- [ ] Cost tracking normalized to USD across all providers
- [ ] Result format consistent regardless of executor used
- [ ] Health check validates API keys/credentials before execution
- [ ] Streaming events work for executors that support it
- [ ] All tests passing (unit + integration)
- [ ] Documentation updated (CLAUDE.md, README, guides)
- [ ] Examples added for each executor

## Testing Strategy

**Unit tests:**
- Interface compliance for all executors
- Config parsing and validation
- Result normalization
- Cost calculations
- Mock executor for integration tests

**Integration tests:**
- Claude executor (existing tests + new interface)
- Codex executor (requires API key, skip in CI if not set)
- Jules executor (requires GitHub app, skip in CI if not set)
- Gemini executor (requires Google account, skip in CI if not set)

**Manual testing:**
- End-to-end directive execution with each executor
- Eval benchmark runs on all providers
- Cost report accuracy verification
- Streaming output visibility

**Test environment variables:**
```bash
# For CI/CD, mark tests as skipped if not set
ANTHROPIC_API_KEY=...      # Claude
CODEX_API_KEY=...          # Codex
JULES_API_KEY=...          # Jules
GOOGLE_APPLICATION_CREDENTIALS=... # Gemini
```

## Non-Goals

**Not in this feature:**
- **Custom executor plugins** - Third-party executor registration deferred to v0.6.0
- **Executor load balancing** - Automatic failover/distribution not needed yet
- **Fine-tuning integration** - Custom model support out of scope
- **IDE integration** - This is CLI/eval focused, not VS Code
- **Jules GitHub app automation** - Manual setup required initially
- **Gemini Code Assist enterprise** - Focus on free/personal tier first

## Timeline

**Week 1** (~22 hours):
- Phase 1: Core interface and factory (~10h)
- Phase 2: Refactor Claude to new interface (~8h)
- Phase 3: Gemini CLI executor (~4h) - Quick win, validates interface
- Unit tests for core infrastructure

**Week 2** (~16 hours):
- Phase 4: OpenAI Codex executor (~10h)
- Phase 6: Eval harness integration (~6h)
- Integration tests with eval harness
- Documentation updates

**Week 3 (Optional)** (~14 hours):
- Phase 5: Google Jules executor (~14h) - Can defer if not needed
- Final testing and release

**Total: ~38 hours for Claude + Gemini + Codex (core value)**
**Optional: +14 hours for Jules (GitHub-only, higher complexity)**

**Recommended MVP:** Ship Phases 1-4 + 6 first (without Jules), add Jules in v0.5.11 if needed.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Jules API is alpha, may change | Medium | Wrap in abstraction layer, pin API version |
| Codex SDK TypeScript-only | Medium | Use CLI wrapper (`codex exec`) for Go integration |
| Jules requires GitHub app | High | Make Jules optional, document setup, allow skip |
| API key management complexity | Medium | Clear config docs, environment variable fallbacks |
| Provider pricing changes | Low | Load costs from config file, not hardcoded |
| Rate limiting across providers | Medium | Implement backoff, expose rate limit errors |
| Inconsistent tool capabilities | Medium | Capability detection, graceful degradation |

## References

- **Current Implementation**: [internal/agent/executor.go](../../../internal/agent/executor.go)
- **Eval Harness**: [internal/eval_harness/agent_runner.go](../../../internal/eval_harness/agent_runner.go)
- **Claude Code Docs**: [code.claude.com/docs/headless](https://code.claude.com/docs/en/headless)
- **OpenAI Codex SDK**: [developers.openai.com/codex/sdk](https://developers.openai.com/codex/sdk/)
- **OpenAI Codex GitHub**: [github.com/openai/codex](https://github.com/openai/codex)
- **Google Jules API**: [developers.google.com/jules/api](https://developers.google.com/jules/api)
- **Jules Blog Post**: [blog.google - Jules Tools and API](https://blog.google/technology/google-labs/jules-tools-jules-api/)
- **Gemini Code Assist**: [developers.google.com/gemini-code-assist](https://developers.google.com/gemini-code-assist/docs/overview)
- **Gemini CLI Announcement**: [blog.google - Gemini CLI](https://blog.google/technology/developers/introducing-gemini-cli-open-source-ai-agent/)

## Future Work

- **Executor plugins**: Allow third-party executor registration for custom AI agents
- **Executor orchestration**: Multi-executor workflows (e.g., Claude for planning, Codex for implementation)
- **Cost-aware routing**: Automatically select cheapest executor for task complexity
- **Benchmark leaderboard**: Public dashboard comparing executor performance on AILANG
- **Session persistence**: Cross-session context for long-running agent tasks
- **Approval flow unification**: Standard approval UI across providers that support it

---

**Document created**: 2025-12-10
**Last updated**: 2025-12-10 (added implementation complexity analysis, reordered phases)
