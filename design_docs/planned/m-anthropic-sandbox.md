# M-ANTHROPIC-SANDBOX: Anthropic Self-Hosted Sandbox Executor

**Status**: PLANNED
**Target**: v0.23.0 (post-M-EVAL-OS-LONGITUDINAL)
**Priority**: P1 — new executor capability; Claude Opus 4.8 access without CLI dependency
**Estimated**: 3-4 days
**Dependencies**:
- `github.com/anthropics/anthropic-sdk-go v1.46.0` (verified available; SDK confirmed with managed-agents support)
- `ANTHROPIC_API_KEY` — session creation and environment management
- `ANTHROPIC_ENVIRONMENT_KEY` — environment-scoped worker auth (NOT the API key)
- `ANTHROPIC_ENVIRONMENT_ID` — pre-created self-hosted environment in Anthropic console
- `ANTHROPIC_AGENT_ID` — agent configured in Anthropic console (model set there, e.g. Opus 4.8)
- `internal/executor/` conformance contract (`docs/internal/EXECUTOR_SHAPE.md`)

---

## Executive Summary

Anthropic's **self-hosted sandbox** (beta, `managed-agents-2026-04-01`) is a new execution model where:

- **Anthropic orchestrates Claude** (the model runs on Anthropic's side)
- **Your infrastructure executes tools** (bash, read, write, edit, glob, grep run locally via a worker process you control)

This is the **architectural inverse** of AILANG's existing `managed_agents` executor (Vertex AI), where Google orchestrates AND executes everything in a remote sandbox. It eliminates the primary weakness of Vertex managed agents: the `CapRemoteSandbox` file-bridge hack.

**Key win**: Claude's file edits go directly to `task.Workspace` on our machine — no `managed_agents_bridge.go` hack, no code-block extraction, no manual file writing. It also unlocks Claude Opus 4.8 (and future models) without upgrading any CLI binary.

**Spike status**: SDK verified (v1.46.0), all required APIs confirmed present, prototype in `internal/executor/anthropic_sandbox/`. See [Spike Findings](#spike-findings) below.

---

## Architecture Comparison

| | `claude` executor (current) | `managed_agents` executor (Vertex AI) | `anthropic_sandbox` (proposed) |
|---|---|---|---|
| **Model** | Claude Sonnet/Opus via Claude Code CLI | Gemini 3.5 Flash (antigravity-preview) | Claude Opus 4.8 (configured on agent) |
| **Orchestration** | Local subprocess | Google-hosted | Anthropic-hosted |
| **Tool execution** | Local (direct filesystem) | Remote (Google sandbox) | Local (our worker) |
| **File access** | Direct | Bridge hack required | Direct |
| **Auth** | Claude Code CLI session | GCP ADC (gcloud) | `ANTHROPIC_API_KEY` + `ANTHROPIC_ENVIRONMENT_KEY` |
| **`CapRemoteSandbox`** | No | **Yes** (bridge hack) | **No** (direct access) |
| **Binary dependency** | `claude` CLI must be installed | None | None |
| **Worker lifecycle** | Claude Code subprocess per task | None (fire-and-forget SSE) | In-process worker per task |

The `managed_agents` executor sends one POST and streams SSE until done. The `anthropic_sandbox` executor requires a **two-component concurrent pattern**:

1. **Session side** (`ANTHROPIC_API_KEY`): creates session → sends `user.message` directive → streams session events to capture text output
2. **Worker side** (`ANTHROPIC_ENVIRONMENT_KEY`): polls environment work queue → `HandleItem` blocks executing all tool calls locally until session ends

Both run concurrently inside `ExecuteStreaming`, driven by goroutines. No daemon required.

---

## Verified API Surface (2026-05-28)

### SDK
```
github.com/anthropics/anthropic-sdk-go v1.46.0
```
Available packages:
- `github.com/anthropics/anthropic-sdk-go` — client, `Beta.Sessions`, `Beta.Environments`
- `github.com/anthropics/anthropic-sdk-go/lib/environments` — `EnvironmentWorker`, `WorkPoller`, `HandleItemOptions`
- `github.com/anthropics/anthropic-sdk-go/option` — `WithAPIKey`, `WithAuthToken`
- `github.com/anthropics/anthropic-sdk-go/tools/agenttoolset` — `AgentToolContext`, `BetaAgentToolset20260401`

### Session creation
```go
session, err := client.Beta.Sessions.New(ctx, anthropic.BetaSessionNewParams{
    Agent:         anthropic.BetaSessionNewParamsAgentUnion{OfString: anthropic.String(agentID)},
    EnvironmentID: environmentID,
    Metadata:      map[string]string{"ailang_task_id": task.ID},
})
```

### Directive delivery (user.message event)
```go
client.Beta.Sessions.Events.Send(ctx, session.ID, anthropic.BetaSessionEventSendParams{
    Events: []anthropic.BetaManagedAgentsEventParamsUnion{
        anthropic.BetaManagedAgentsEventParamsOfUserMessage(
            []anthropic.BetaManagedAgentsUserMessageEventParamsContentUnion{
                {OfText: &anthropic.BetaManagedAgentsTextBlockParam{Text: task.Directive}},
            },
        ),
    },
})
```

### Worker (tool execution side)
```go
worker := environments.NewEnvironmentWorker(workerClient, environments.EnvironmentWorkerOptions{
    EnvironmentID:  environmentID,
    EnvironmentKey: environmentKey,
    Workdir:        task.Workspace,  // Claude's tools write to task workspace directly
})
// Poll for the session's work item, then handle it (blocks until session ends)
poller := environments.NewWorkPoller(ctx, workerClient, environments.WorkPollerOptions{
    EnvironmentID:  environmentID,
    EnvironmentKey: environmentKey,
    Drain:          true,  // stop when queue empty
})
```

### Output capture (text events)
```go
stream := apiClient.Beta.Sessions.Events.StreamEvents(ctx, session.ID,
    anthropic.BetaSessionEventStreamParams{})
for stream.Next() {
    ev := stream.Current()
    if ev.Type == "agent.message" {
        for _, block := range ev.Content.OfBetaManagedAgentsTextBlockArray {
            // block.Text is Claude's output
        }
    }
}
```

### Token accounting
```go
finalSession, _ := apiClient.Beta.Sessions.Get(ctx, session.ID, ...)
result.InputTokens  = int(finalSession.Stats.InputTokens)
result.OutputTokens = int(finalSession.Stats.OutputTokens)
```

---

## Integration Design

### New Executor Package

```
internal/executor/anthropic_sandbox/
  anthropic_sandbox.go      # Main executor: session + worker concurrency pattern
  anthropic_sandbox_test.go # Unit tests with fixture replay
  README.md                 # Auth setup, env vars, agent console setup
  testdata/                 # SSE event fixtures for replay tests
```

### Executor Interface Conformance

All four required symbols from `EXECUTOR_SHAPE.md`:
- `New(cfg *executor.Config) (*Executor, error)` ✓
- `Register()` ✓
- `init()` → calls `Register()` ✓
- `*Executor` implements all `executor.Executor` methods ✓

**Capabilities**: `[]executor.Capability{executor.CapStreaming}` — no `CapRemoteSandbox` because tools execute locally.

### Coordinator Wiring

One-line blank import in `internal/coordinator/provider_executor.go`:
```go
_ "github.com/sunholo-data/ailang/internal/executor/anthropic_sandbox"
```

### models.yml Entry

```yaml
anthropic-opus-4-8:
  agent_cli: "anthropic_sandbox"
  pricing:
    input_per_1k: 0.015     # Claude Opus 4.8 pricing (placeholder — verify before release)
    output_per_1k: 0.075
  budget:
    max_cost_usd: 0.50
    hard_timeout_secs: 600
  notes: >-
    Requires ANTHROPIC_API_KEY, ANTHROPIC_ENVIRONMENT_KEY, ANTHROPIC_ENVIRONMENT_ID,
    ANTHROPIC_AGENT_ID. Model is set in the Anthropic console agent config, not here.
    Worker executes tools locally in task.Workspace — no CapRemoteSandbox bridge needed.
```

### Concurrency Pattern in ExecuteStreaming

```
ExecuteStreaming(ctx, task, handler)
│
├── client.Beta.Sessions.New()          → session.ID
├── client.Beta.Sessions.Events.Send()  → sends directive (user.message)
│
├── goroutine A: WorkPoller + HandleItem  (worker side)
│     poll env queue → claim work item for session
│     HandleItem → blocks, executing ALL tool calls in task.Workspace
│     done signal via channel when session ends
│
├── goroutine B: StreamEvents           (output side)
│     ev.Type == "agent.message" → ev.Content.OfBetaManagedAgentsTextBlockArray
│     ev.Type == "session.status_terminated" → exit loop
│     forward text to handler(EventOutput)
│
└── <-done → read finalSession stats → return Result
```

---

## Differences from `managed_agents` Bridge Pattern

The `managed_agents` executor advertises `CapRemoteSandbox` and the eval harness in `agent_runner_multi.go` detects it to:
1. Append bridge instruction to system prompt ("output your solution as a fenced code block")
2. Call `writeSolutionFromResponse()` to extract and write files locally

**None of this is needed with `anthropic_sandbox`**. The worker's `agenttoolset` provides `bash`, `read`, `write`, `edit`, `glob`, `grep` executing directly on `task.Workspace`. Claude writes `solution.ail` for real. The eval harness reads it normally.

This means `anthropic_sandbox` can be added to the **standard eval rotation** without special-casing — it behaves exactly like the `claude` executor from the eval harness's perspective.

---

## Spike Findings

Spike code: `internal/executor/anthropic_sandbox/anthropic_sandbox.go`

**Verified**:
- `github.com/anthropics/anthropic-sdk-go v1.46.0` compiles; all required types present
- `environments.EnvironmentWorker` and `environments.WorkPoller` both available with correct signatures
- `BetaSessionEventSendParams` accepts `user.message` events for directive delivery
- `BetaManagedAgentsStreamSessionEventsUnion.Type == "agent.message"` + `.Content.OfBetaManagedAgentsTextBlockArray` for output capture
- `BetaManagedAgentsSessionStats.InputTokens` / `.OutputTokens` for billing
- Go SDK worker pattern fits AILANG's `ExecuteStreaming` concurrency model

**Not yet tested live** (requires Anthropic console setup: environment + agent with Opus 4.8):
- Actual session execution end-to-end
- File write latency (does `write` tool call take meaningful extra time vs. local Claude?)
- Parallel session isolation (two concurrent sessions in same environment — any crosstalk?)
- Work queue timing (race between session creation and poller start)
- Session termination events (which `Type` value signals normal completion?)

**Open questions**:
1. **Environment reuse vs. per-task**: Should AILANG pre-create one shared environment (lower overhead) or create one per task (better isolation)? The SDK supports both.
2. **Work queue race**: `NewWorkPoller` starts before `Events.Send` to avoid missing the work item. Is there a safer way to wait for a specific session's work item?
3. **Cost model**: Managed agents pricing adds orchestration overhead on top of model token costs. Need to verify actual billing amounts with a live run.
4. **Session timeout**: `session.status_terminated` is the termination event. What triggers it on idle? Does it respect `task.Timeout`?
5. **Cloud Run deployment**: Worker needs to run somewhere. For cloud coordinator, the Cloud Run Job would need the environment key injected. Separate design needed for Pillar 2.

---

## Milestones

### M1: Core Executor (1.5 days)
- `internal/executor/anthropic_sandbox/anthropic_sandbox.go` — full production implementation
  - Proper context cancellation, timeout handling
  - Session termination detection (`session.status_terminated` / `session.status_idle` + `end_turn`)
  - Error surfacing from `session.error` events
  - Token accounting with `Stats.InputTokens` / `Stats.OutputTokens`
  - Cost budget integration (same `task.Budget.Add()` pattern as other executors)
- `internal/executor/anthropic_sandbox/anthropic_sandbox_test.go`
  - Fixture replay test with a recorded SSE event stream
  - Compile-only test for offline CI
- README with console setup steps (create environment, generate key, create agent with Opus 4.8)

### M2: Wire into Eval Harness (0.5 days)
- Blank import in `internal/coordinator/provider_executor.go`
- `models.yml` entry with correct pricing and budget
- Verify eval harness does NOT inject bridge instruction (no `CapRemoteSandbox`)
- 3-benchmark smoke gate: `fibonacci.ail`, `hello_world.ail`, `balanced_parens.ail`

### M3: Observability + Docs (0.5 days)
- OTEL spans mirroring `claude` executor pattern
- `ailang chains` integration (session ID stored in chain metadata)
- Design doc moved to `implemented/v0_23_0/`
- CHANGELOG entry

### M4: Eval Rotation (1 day, after M3)
- Add `anthropic-opus-4-8` to standard eval rotation
- Baseline benchmark run: compare vs. `claude` executor on same benchmarks
- Dashboard entry

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Session event schema drift (beta API) | Medium | High | Version-pin via `anthropic-beta: managed-agents-2026-04-01` header; record SSE fixtures for regression tests |
| Work queue race (create before poll claims item) | Low | Medium | Start poller before `Events.Send`; add retry with short backoff |
| Cloud Run Pillar 2 complexity | Medium | Medium | Defer to separate design; local-only is sufficient for v0.23.0 |
| Cost surprises (orchestration overhead) | Low | Low | Benchmark cost vs. direct Claude API in smoke tests |
| `session.status_terminated` ambiguity | Medium | Medium | Treat `session.status_idle` with `end_turn` stop reason as success; `session.error` as failure |
