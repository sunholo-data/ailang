# M-AGENT: std/agent Module for AI Agent Orchestration

**Status**: Planned
**Target**: v0.10.0+
**Priority**: P3 (Exploratory — depends on M-PROCESS shipping first)
**Estimated**: 2-3 weeks (significant new effect + streaming infrastructure)
**Dependencies**: M-PROCESS (std/process), std/stream (streaming infrastructure)

## Motivation

AILANG's effect system, capability model, and trace infrastructure make it uniquely positioned to **govern AI agent execution**. Today, AI agents (Claude Code CLI, Gemini CLI) are orchestrated by unconstrained Go code in `internal/executor/`. Moving this orchestration into AILANG's effect system would give us:

1. **Typed governance** — `Agent` effect in function signatures makes AI invocation explicit
2. **Capability-bounded** — `--caps Agent` required, with per-agent allowlists
3. **Auditable** — Every agent invocation produces structured traces with cost, turns, tool usage
4. **Composable** — Chain agents with FS, Process, Net effects in typed pipelines
5. **Budget-bounded** — `Agent @limit=N` or `Agent @cost=$X` to cap spending

This is fundamentally different from `std/process` (which runs arbitrary commands synchronously). `std/agent` is specifically designed for long-running, streaming, AI-powered execution with session continuity.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | AI agents are inherently nondeterministic (LLM sampling). Explicit via Agent effect. Stronger nondeterminism than Process — output depends on model weights, temperature, context window. |
| A2: Replayability | +1 | Full structured traces: prompt, turns, tool calls, tool results, cost, tokens. Replay mode returns recorded outputs. Session IDs enable conversation continuity. |
| A3: Effect Legibility | +1 | Agent effect is explicit in function signatures: `! {Agent}` |
| A4: Explicit Authority | +1 | Requires `--caps Agent`. Per-agent allowlist (`--agent-allowlist claude,gemini`). Tool restrictions passed to agent. |
| A5: Bounded Verification | 0 | No impact on local type checking |
| A6: Safe Concurrency | 0 | Agent invocations are sequential within AILANG. Internal concurrency is handled by the agent runtime (Claude/Gemini). |
| A7: Machines First | +1 | Structured `AgentResult` type with typed fields: turns, cost, tools used, files changed. `AgentError` ADT for failures. |
| A8: Minimal Syntax | +1 | No new syntax — uses existing function call, Result type, effect declaration, and record types. |
| A9: Cost Visibility | +1 | Cost is a first-class field in AgentResult. Budget caps via `--agent-max-cost` and `--agent-max-turns`. Agent effect signals "this is expensive and unpredictable." |
| A10: Composability | +1 | Composes naturally: `FS → Agent → Process` pipelines. Agent reads files, generates code, Process compiles it. |
| A11: Structured Failure | +1 | `AgentError` ADT: Timeout, CostExceeded, TurnLimitExceeded, ProviderError, SessionNotFound. |
| A12: System Boundary | +1 | Explicit transition from AILANG to AI runtime — the most consequential system boundary in the project. |

**Net Score: +8** → **Decision: Move forward** (after M-PROCESS ships)

### Hard Violation Check

- [x] A1 (Determinism): Nondeterminism is explicit via Agent effect, not implicit
- [x] A3 (Effects): Agent side effect is declared in type signature
- [x] A4 (Authority): Requires explicit `--caps Agent` grant
- [x] A7 (Machines First): Returns structured AgentResult, not raw text

## Problem Statement

Today, the coordinator daemon (`internal/coordinator/`) orchestrates AI agents in Go:
- `internal/executor/claude/claude.go` — Runs Claude Code CLI with streaming NDJSON parsing
- `internal/executor/gemini/gemini.go` — Runs Gemini CLI with streaming NDJSON parsing
- Worktree management, session continuity, cost tracking all happen in unconstrained Go

**This works, but has no governance properties:**
- Any Go code can invoke any agent with any prompt — no capability check
- Cost tracking is after-the-fact — no pre-execution budget enforcement
- Tool restrictions are passed but not enforced at the AILANG level
- Session management is ad-hoc string passing, not typed
- No way for AILANG programs to invoke agents — it's Go-only infrastructure

**The Vision:** AILANG programs that orchestrate AI agents with the same typed, capability-bounded, auditable semantics as file I/O or network requests.

## Goals

**Primary Goal:** Enable AILANG programs to invoke AI agents with typed governance: capability bounds, cost budgets, tool restrictions, and structured results.

**Success Metrics:**
- AILANG program can invoke Claude/Gemini agent and get structured result
- Missing `--caps Agent` produces CapabilityError
- Cost budget exceeded produces `Err(CostExceeded(...))`
- Turn limit exceeded produces `Err(TurnLimitExceeded(...))`
- Streaming events are accessible via callback (optional)
- Session continuity: resume previous agent sessions

## Solution Design

### Type Definitions

```ailang
-- Agent task configuration
type AgentTask = {
  provider: string,          -- "claude" | "gemini"
  directive: string,         -- the prompt/instruction
  model: string,             -- "claude-opus-4-6" | "gemini-2-5-flash" | etc.
  workspace: string,         -- working directory for agent
  allowedTools: [string],    -- tool restrictions (empty = all tools)
  maxTurns: int,             -- turn limit (0 = unlimited)
  maxCostUsd: float,         -- cost budget in USD (0.0 = unlimited)
  timeout: int,              -- ms (0 = default 3600000 = 1 hour)
  resumeSessionId: string    -- "" for new session, UUID to resume
}

-- Agent execution result
type AgentResult = {
  output: string,            -- final text output
  sessionId: string,         -- for future session resumption
  turns: int,                -- number of conversational turns
  toolCalls: int,            -- total tool invocations
  costUsd: float,            -- actual cost incurred
  inputTokens: int,          -- total input tokens
  outputTokens: int,         -- total output tokens
  filesCreated: [string],    -- files created by agent
  filesModified: [string],   -- files modified by agent
  durationMs: int            -- wall-clock time
}

-- Agent error ADT
type AgentError =
  | ProviderNotFound(string)        -- unknown provider name
  | ProviderUnavailable(string)     -- binary not installed or API down
  | Timeout(int)                    -- ms elapsed
  | CostExceeded(float, float)     -- actual, budget
  | TurnLimitExceeded(int, int)    -- actual, limit
  | SessionNotFound(string)         -- invalid session ID for resume
  | ProviderError(string)           -- provider-specific error
  | NotAllowed(string)              -- provider not in allowlist
```

### API Design

```ailang
import std/agent (invoke, AgentTask, AgentResult, AgentError)

-- Simple invocation
func fixBug(description: string) -> Result[AgentResult, AgentError] ! {Agent, FS} {
  invoke({
    provider: "claude",
    directive: "Fix this bug: " ++ description,
    model: "claude-opus-4-6",
    workspace: ".",
    allowedTools: ["Read", "Edit", "Bash"],
    maxTurns: 20,
    maxCostUsd: 5.0,
    timeout: 600000,
    resumeSessionId: ""
  })
}

-- Using the result
func main() -> () ! {Agent, FS, IO} {
  match fixBug("null pointer in parser.go line 42") {
    Ok(result) => {
      println("Fixed in " ++ show(result.turns) ++ " turns");
      println("Cost: $" ++ show(result.costUsd));
      println("Files changed: " ++ show(result.filesModified))
    },
    Err(CostExceeded(actual, budget)) =>
      println("Stopped: $" ++ show(actual) ++ " exceeded $" ++ show(budget) ++ " budget"),
    Err(TurnLimitExceeded(actual, limit)) =>
      println("Stopped: " ++ show(actual) ++ " turns exceeded " ++ show(limit) ++ " limit"),
    Err(Timeout(ms)) =>
      println("Timed out after " ++ show(ms) ++ "ms"),
    Err(e) =>
      println("Agent error: " ++ show(e))
  }
}
```

**Type signature:**
```
invoke : AgentTask -> Result[AgentResult, AgentError] ! {Agent}
```

### Streaming API (Optional)

For real-time event handling during agent execution:

```ailang
import std/agent (invokeStreaming, AgentEvent)

type AgentEvent =
  | TurnStart(int)                  -- turn number
  | Text(string)                    -- text output chunk
  | ToolUse(string, string)         -- tool name, input summary
  | ToolResult(string, string)      -- tool name, output summary
  | TurnEnd(int)                    -- turn number
  | CostUpdate(float)              -- cumulative cost so far

-- Streaming invocation with callback
func watchAgent(task: AgentTask) -> Result[AgentResult, AgentError] ! {Agent, IO} {
  invokeStreaming(task, \event. {
    match event {
      TurnStart(n) => println("--- Turn " ++ show(n) ++ " ---"),
      Text(t) => print(t),
      ToolUse(name, _) => println("[tool: " ++ name ++ "]"),
      CostUpdate(c) => println("[cost: $" ++ show(c) ++ "]"),
      _ => ()
    }
  })
}
```

### Security Design

**1. Capability requirement**
```bash
ailang run --caps Agent,FS --entry main orchestrator.ail
```

**2. Provider allowlist**
```bash
# Only permit specific providers
ailang run --caps Agent --agent-allowlist "claude,gemini" module.ail
```

**3. Budget enforcement (pre-execution)**
```bash
# Global cost cap across all agent invocations in this run
ailang run --caps Agent --agent-max-cost 10.00 module.ail

# Global turn cap
ailang run --caps Agent --agent-max-turns 100 module.ail
```

Per-invocation budgets are specified in `AgentTask` and enforced by the runtime.

**4. Tool restrictions**
The `allowedTools` field in `AgentTask` is passed to the underlying CLI executor. This restricts what tools the AI agent can use during its session.

**5. Workspace isolation**
Agent workspace is set via `AgentTask.workspace`. Combined with `AILANG_FS_SANDBOX`, this constrains where the agent can operate.

### Relationship to Existing Infrastructure

| Component | Current (Go) | Future (AILANG) |
|-----------|-------------|-----------------|
| Executor interface | `internal/executor/executor.go` | `std/agent` effect handler wraps executors |
| Claude CLI | `internal/executor/claude/` | Reused as-is, called via Agent effect |
| Gemini CLI | `internal/executor/gemini/` | Reused as-is, called via Agent effect |
| Coordinator | `internal/coordinator/` | Can invoke AILANG programs that use Agent effect |
| Cost tracking | After-the-fact in observatory.db | Pre-execution budget + real-time monitoring |
| Session management | Ad-hoc string passing | Typed `resumeSessionId` field |

**Key principle:** `std/agent` is a **typed interface layer** over the existing Go executors, not a replacement. The Go executors handle the actual CLI interaction, NDJSON parsing, and environment setup. The AILANG Agent effect adds governance.

### How It Differs from std/process

| Property | std/process | std/agent |
|----------|-------------|-----------|
| Execution model | Synchronous, blocks until exit | Long-running with streaming events |
| Output | Raw bytes (stdout/stderr) | Structured AgentResult with cost, turns, tools |
| Duration | Seconds | Minutes to hours |
| Cost model | Wall-clock time only | Token-based cost with USD tracking |
| Session | Stateless | Stateful (session continuity) |
| Error types | ProcessError (spawn failures) | AgentError (budget, turns, provider) |
| Security | Allowlist of binaries | Allowlist of providers + tool restrictions |
| Primary use | Host tool integration | AI-powered code generation and reasoning |

## Implementation Plan (High-Level)

**Phase 1: Core Agent effect** (~1 week)
- Define AgentTask, AgentResult, AgentError types in std/agent.ail
- Create Agent effect handler that wraps existing Go executors
- Register `_agent_invoke` builtin
- CLI flags: --agent-allowlist, --agent-max-cost, --agent-max-turns

**Phase 2: Streaming support** (~1 week)
- Add AgentEvent type and streaming callback API
- Wire streaming handler to executor's EventHandler interface
- Real-time cost monitoring with budget enforcement

**Phase 3: Integration + testing** (~1 week)
- Integration tests with mock executor
- Cost budget enforcement tests
- Session continuity tests
- Documentation and examples

## Non-Goals (v1)

- **Multi-agent coordination** — No agent-to-agent communication in AILANG. Use coordinator for that.
- **Model selection logic** — No built-in routing. User specifies provider + model explicitly.
- **Training/fine-tuning** — Read-only interaction with AI providers.
- **Prompt engineering** — No prompt templates or optimization. User writes prompts as strings.
- **Concurrent agents** — Sequential invocation only. Parallel agents violate A6.

## Use Cases

### 1. AILANG as AI Governance Layer

```ailang
-- Central policy: all AI invocations go through typed, auditable, bounded pipeline
func governedAgentCall(task: AgentTask) -> Result[AgentResult, AgentError] ! {Agent, IO} {
  -- Pre-execution: log intent
  println("Invoking " ++ task.provider ++ " with budget $" ++ show(task.maxCostUsd));

  let result = invoke(task);

  -- Post-execution: structured audit
  match result {
    Ok(r) => {
      println("Completed: " ++ show(r.turns) ++ " turns, $" ++ show(r.costUsd));
      Ok(r)
    },
    Err(e) => {
      println("Failed: " ++ show(e));
      Err(e)
    }
  }
}
```

### 2. Multi-Stage Pipeline

```ailang
-- Design → Implement → Test pipeline (sequential, typed)
func developFeature(spec: string) -> () ! {Agent, FS, IO} {
  -- Stage 1: Design
  let design = invoke({
    provider: "claude", directive: "Create design doc for: " ++ spec,
    model: "claude-opus-4-6", workspace: ".", allowedTools: ["Read"],
    maxTurns: 10, maxCostUsd: 2.0, timeout: 300000, resumeSessionId: ""
  });

  match design {
    Ok(d) => {
      -- Stage 2: Implement (resume same session for context)
      let impl = invoke({
        provider: "claude", directive: "Implement the design you just created",
        model: "claude-opus-4-6", workspace: ".", allowedTools: ["Read", "Edit", "Bash"],
        maxTurns: 30, maxCostUsd: 10.0, timeout: 1800000,
        resumeSessionId: d.sessionId
      });

      match impl {
        Ok(i) => println("Feature implemented: " ++ show(i.filesModified)),
        Err(e) => println("Implementation failed: " ++ show(e))
      }
    },
    Err(e) => println("Design failed: " ++ show(e))
  }
}
```

### 3. Cost-Bounded Research

```ailang
func research(topic: string) -> Result[string, AgentError] ! {Agent} {
  match invoke({
    provider: "gemini", directive: "Research: " ++ topic,
    model: "gemini-2-5-pro", workspace: "/tmp",
    allowedTools: [],  -- no tools, just text generation
    maxTurns: 5, maxCostUsd: 1.0, timeout: 120000, resumeSessionId: ""
  }) {
    Ok(r) => Ok(r.output),
    Err(e) => Err(e)
  }
}
```

## Related Documents

- [design_docs/planned/m-process-exec.md](design_docs/planned/m-process-exec.md) — std/process (prerequisite, simpler scope)
- [design_docs/planned/v0_8_1/m-arch4-executor-stream-processor.md](design_docs/planned/v0_8_1/m-arch4-executor-stream-processor.md) — Executor stream processing architecture

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- `internal/executor/executor.go` — Current Go executor interface (to be wrapped)
- `internal/coordinator/` — Current Go coordinator (potential consumer of Agent-effect AILANG programs)

## Open Questions

1. **Should Agent subsume Process?** An agent *is* a process, but with richer semantics. Keep them separate (Agent for AI, Process for tools) or unify?
2. **Cost tracking granularity**: Per-invocation only, or cumulative across a program run with a global budget?
3. **Session storage**: Where do session transcripts live? In observatory.db? Separate store?
4. **WASM support**: Could Agent work in browser via API calls (not CLI)? Different executor backend.

---

**Document created**: 2026-02-18
**Last updated**: 2026-02-18
