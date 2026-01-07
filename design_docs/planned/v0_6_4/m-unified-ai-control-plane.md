# M-UNIFIED-AI-CONTROL-PLANE

**Status**: Planned
**Target**: v0.6.4
**Priority**: P1 - Medium
**Estimated**: 16-20 hours
**Dependencies**: M-CONTROL-PLANE-V4-INTEGRATION (partially complete)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Unified tracing enables reproducible execution analysis |
| A2: Replayability | +1 | Complete trace hierarchy enables replay from any point |
| A3: Effect Legibility | +1 | All AI operations explicitly routed through single control point |
| A4: Explicit Authority | +1 | CLI flags explicitly grant AI execution capabilities |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured metadata enables automated trace analysis |
| A8: Minimal Syntax | 0 | No new syntax (CLI only) |
| A9: Cost Visibility | +1 | Unified cost tracking across all AI operations |
| A10: Composability | +1 | Composes with existing coordinator, eval, messaging systems |
| A11: Structured Failure | +1 | Unified error handling with structured JSON output |
| A12: System Boundary | +1 | Explicit boundary: ailang CLI → external AI CLIs |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Improves reproducibility via complete traces
- [x] A3 (Effects): Makes all AI invocation explicit
- [x] A4 (Authority): No ambient access - explicit CLI flags required
- [x] A7 (Machines First): Optimizes for machine analysis via structured traces

## Problem Statement

AI operations in AILANG are invoked through multiple disconnected paths, causing fragmented traces, duplicated setup code, and inconsistent hierarchy tracking.

**Current State:**
- **4 separate AI invocation paths**: Direct API (`internal/ai/`), CLI executors (`internal/executor/`), eval harness (own AIAgent wrapper), coordinator daemon
- **Duplicated environment setup**: OTEL endpoints, stdlib paths, resource attributes configured in 4+ locations (~200 LOC duplicated)
- **Fragmented traces**: Direct `ailang run --ai` bypasses coordinator tracing; eval suite creates separate root spans
- **Inconsistent hierarchy**: Coordinator tasks, eval runs, and direct CLI invocations don't share common ancestors

**Current Flow (Fragmented):**
```
Coordinator → executor/claude → claude CLI
     ↓
  Creates task/span

Eval Suite → eval_harness/ai_agent → ai/anthropic → HTTP API
     ↓
  Creates separate root span (no task link)

Direct CLI → cmd/ailang → ai/anthropic → HTTP API
     ↓
  Creates span (no task, no coordinator context)
```

**Impact:**
- **Debugging difficulty**: Can't trace full request lifecycle
- **Cost attribution**: AI costs scattered across unlinked spans
- **Hierarchy gaps**: Control Plane dashboard shows incomplete task trees
- **Code maintenance**: Bug fixes need replication across 4 paths

## Goals

**Primary Goal:** Route all AI operations through `ailang exec` command, making ailang CLI the single control point for trace hierarchy, metadata injection, and unified execution.

**Success Metrics:**
- All AI operations appear under unified trace hierarchy in Observatory
- Code duplication reduced by ~150 LOC (environment setup consolidation)
- Single source of truth for OTEL resource attributes
- Eval suite traces link to parent task when run via coordinator
- Cost rollup available at workspace → task → operation levels

## Solution Design

### Overview

Introduce `ailang exec` as the unified AI execution command that:
1. Sets up complete tracing context (parent span, resource attributes)
2. Handles both API-based and CLI-based AI operations
3. Provides consistent streaming output format (NDJSON events)
4. Records metrics to Observatory for hierarchy integration

All existing entry points (coordinator, eval, direct CLI) call `ailang exec` internally.

### Architecture

**Unified Control Hierarchy:**

The key insight is that **all operations flow through ailang CLI commands**, which ensures consistent metadata propagation:

```
Workspace
    ↓
Task (created by ailang messages)
    ↓
Sub-task (ailang messages or ailang exec)
    ↓
AI Execution (ailang exec claude/gemini/openai)
    ↓
Child operations (more messages or execs)
```

**Message-Driven Coordination:**

`ailang messages` serves as the coordination layer between agents:

```
User: ailang messages send design-doc-creator "Add semantic caching"
                    ↓
        Message stored with metadata:
        - message_id: msg_abc123
        - parent_task_id: (none - root task)
        - workspace_id: ws_xyz
        - correlation_id: corr_123
                    ↓
    Coordinator picks up message, spawns:
        ailang exec claude "Create design doc for semantic caching" \
          --task-id=msg_abc123 \
          --workspace=/path/to/repo
                    ↓
        Claude agent decides to delegate to sprint-planner:
        ailang messages send sprint-planner "Plan sprint for semantic caching" \
          --parent-task=msg_abc123 \
          --correlation-id=corr_123
                    ↓
        Sprint-planner receives message with inherited context:
        - message_id: msg_def456
        - parent_task_id: msg_abc123  ← Links to parent!
        - workspace_id: ws_xyz
        - correlation_id: corr_123
                    ↓
        Coordinator spawns for sprint-planner:
        ailang exec claude "Create sprint plan..." \
          --task-id=msg_def456 \
          --parent-task-id=msg_abc123 \
          --workspace=/path/to/repo
```

**New Unified Flow:**
```
User/Coordinator/Eval
         ↓
   ailang messages send <inbox> "task" [--parent-task=...] [--correlation-id=...]
         ↓
   Message stored with hierarchy metadata
         ↓
   Coordinator daemon picks up message
         ↓
   ailang exec [claude|gemini|openai|ollama] "directive" --task-id=<msg_id>
         ↓
   ┌─────────────────────────────────────────────┐
   │ UnifiedExecutor (cmd/ailang/exec.go)        │
   │ - Creates parent span with task context     │
   │ - Injects OTEL resource attributes          │
   │ - Handles provider selection                │
   │ - Normalizes streaming output               │
   └─────────────────────────────────────────────┘
         ↓
   ┌──────────┬──────────────┐
   ↓          ↓              ↓
executor/  executor/     ai/provider
claude     gemini        (API mode)
   ↓          ↓              ↓
claude CLI  gemini CLI   HTTP API
```

**Components:**

1. **`ailang exec` Command** (`cmd/ailang/exec.go`):
   - Entry point for all AI operations
   - Parses provider, model, workspace, and task options
   - Creates root span with hierarchy context
   - Dispatches to executor or API provider
   - Emits standardized NDJSON events

2. **Unified Environment Setup** (`internal/executor/environment.go`):
   - Single function for OTEL configuration
   - Consolidates resource attribute building
   - Handles trace context propagation
   - Sets stdlib path, GCP project, etc.

3. **Observatory Integration** (`cmd/ailang/exec.go`):
   - Auto-registers task in Observatory when `--register-task` flag
   - Links spans to task via resource attributes
   - Emits final metrics (tokens, cost, duration)

4. **Message Metadata Propagation** (`cmd/ailang/messages.go`):
   - `ailang messages send` accepts `--parent-task` and `--correlation-id`
   - Messages store hierarchy metadata in `inbox_messages` table
   - Coordinator extracts metadata when spawning `ailang exec`
   - Child tasks inherit parent context automatically

5. **Task Hierarchy in Observatory**:
   - Messages create Observatory tasks on send (optional `--register-task`)
   - `ailang exec` links to parent task via `--parent-task-id`
   - Observatory aggregates costs up the hierarchy tree
   - Control Plane shows: Workspace → Root Task → Sub-tasks → Spans

6. **Eval Suite via Messages** (resurrects M-EVAL-AGENT-QUEUE):
   - Each benchmark becomes a message to `eval-runner` inbox
   - Coordinator processes benchmarks using `ailang exec` per benchmark
   - Results posted back as messages with hierarchy metadata
   - Crash recovery: unacked benchmarks resume on restart
   - Replaces 300+ LOC of custom parallel job management

### Eval Suite Unification

**Before (custom parallel job management):**
```go
// cmd/ailang/eval_suite.go - 300+ LOC of custom code
func runBenchmarksParallel(jobs []BenchmarkJob, parallel int, ...) {
    sem := make(chan struct{}, parallel)
    for _, job := range jobs {
        go func(j BenchmarkJob) {
            // Custom spawn logic
            // Custom result collection
            // Custom error handling
        }(job)
    }
}
```

**After (unified via messages):**
```bash
# Each benchmark is a message
for benchmark in benchmarks; do
    ailang messages send eval-runner "$benchmark" \
        --parent-task=$suite_task_id \
        --correlation-id=$eval_correlation
done

# Coordinator processes via ailang exec
# Results flow back as messages with hierarchy
# Crash recovery automatic via message acknowledgement
```

**Code Reduction:**
- Delete `runBenchmarksParallel()` (~150 LOC)
- Delete custom job queue management (~100 LOC)
- Delete custom result aggregation (~50 LOC)
- Reuse existing messaging infrastructure
- **Net: ~300 LOC removed, unified architecture**

**Benefits (from original M-EVAL-AGENT-QUEUE design):**
- **Crash Recovery**: Unacked benchmarks remain for resume
- **State Persistence**: Task queue survives process crashes
- **Observability**: `ailang messages list --inbox eval-runner`
- **Dogfooding**: Uses AILANG's own messaging at scale (264 tasks)
- **Distributed**: Multiple workers can poll same queue (future)

### Implementation Plan

**Phase 1: Unified exec Command** (~4 hours)
- [ ] Create `cmd/ailang/exec.go` with `ailang exec` subcommand
- [ ] Add provider selection: `ailang exec claude|gemini|openai|ollama`
- [ ] Add common flags: `--workspace`, `--model`, `--timeout`, `--task-id`
- [ ] Implement NDJSON streaming output format
- [ ] Create root span with resource attributes

**Phase 2: Environment Consolidation** (~2 hours)
- [ ] Create `internal/executor/environment.go`
- [ ] Extract `BuildEnvironment()` from claude/gemini executors
- [ ] Consolidate OTEL endpoint, resource attributes, stdlib path
- [ ] Update `executor/claude/claude.go` to use shared function
- [ ] Update `executor/gemini/gemini.go` to use shared function

**Phase 3: Integration** (~3 hours)
- [ ] Update coordinator to use `ailang exec` for task delegation
- [ ] Update eval harness agent mode to use `ailang exec`
- [ ] Add `--register-task` flag for Observatory task creation
- [ ] Wire task_id from coordinator → exec → Observatory

**Phase 4: API Provider Mode** (~2 hours)
- [ ] Add API provider support: `ailang exec openai --api-only "prompt"`
- [ ] Integrate `internal/ai/` providers via exec command
- [ ] Ensure same tracing/metadata for API vs CLI modes

**Phase 5: Message Hierarchy Integration** (~3 hours)
- [ ] Add `--parent-task` and `--correlation-id` flags to `ailang messages send`
- [ ] Store hierarchy metadata in `inbox_messages` table (new columns)
- [ ] Update coordinator to extract and pass hierarchy to `ailang exec`
- [ ] Add `--parent-task-id` flag to `ailang exec` for span linking
- [ ] Update Observatory to aggregate costs up task hierarchy

**Phase 6: Eval Suite Migration to Messages** (~4 hours)
- [ ] Create `eval-runner` inbox configuration in coordinator
- [ ] Refactor `cmd/ailang/eval_suite.go` to use `ailang messages send` per benchmark
- [ ] Delete `runBenchmarksParallel()` and custom job queue (~300 LOC)
- [ ] Add `--queue` flag to enable message-based coordination
- [ ] Implement result collection via message acknowledgement
- [ ] Add crash recovery: resume from unacked messages
- [ ] Test with full suite (264 benchmarks × 3 models)

### Files to Modify/Create

**New files:**
- `cmd/ailang/exec.go` - Unified exec command (~250 LOC)
- `internal/executor/environment.go` - Shared environment builder (~100 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Register exec subcommand (~10 LOC)
- `cmd/ailang/messages.go` - Add `--parent-task`, `--correlation-id` flags (~30 LOC)
- `internal/messaging/store.go` - Add hierarchy columns to schema (~20 LOC)
- `internal/executor/claude/claude.go` - Use shared environment (~-50 LOC)
- `internal/executor/gemini/gemini.go` - Use shared environment (~-50 LOC)
- `internal/coordinator/daemon_tasks.go` - Extract hierarchy, call `ailang exec` (~50 LOC)
- `cmd/ailang/eval_suite.go` - Migrate to messages, delete parallel runner (~-300 LOC)
- `internal/observatory/backend.go` - Add task hierarchy aggregation (~40 LOC)

**Net change:** ~400 LOC reduction (consolidation + eval suite unification)

## Examples

### Example 1: Direct AI Execution

**Before (multiple invocation patterns):**
```bash
# Via coordinator (creates task + traces)
ailang messages send coordinator "Fix the bug" --type bug

# Via eval (separate trace tree)
ailang eval-suite --models claude-sonnet-4-5 --agent

# Direct API (minimal tracing)
ailang run --ai claude-haiku-4-5 --caps IO file.ail
```

**After (unified via exec):**
```bash
# All operations route through exec
ailang exec claude "Fix the bug in parser.go" --workspace /repo

# With explicit task registration
ailang exec claude "Fix the bug" --register-task --workspace /repo

# API-only mode (no CLI tool)
ailang exec openai "Generate test cases" --api-only

# With parent task context (from coordinator)
AILANG_TASK_ID=task_123 ailang exec gemini "Implement feature"
```

### Example 2: Trace Hierarchy

**Before (fragmented):**
```
Trace A (coordinator):
  └─ claude.execute (task_123)
       └─ [claude CLI internal spans - no link to parent]

Trace B (eval suite):
  └─ eval.suite (no task link)
       └─ eval.benchmark (standalone)
```

**After (unified):**
```
Trace A (coordinator):
  └─ coordinator.task (task_123)
       └─ ailang.exec (provider=claude)
            └─ claude.execute
                 └─ [claude CLI spans - linked via resource attrs]

Trace B (eval via coordinator):
  └─ coordinator.task (task_456)
       └─ ailang.exec (provider=claude, mode=agent)
            └─ eval.benchmark
                 └─ claude.execute
```

### Example 3: Multi-Agent Coordination via Messages

**Full workflow: User → design-doc-creator → sprint-planner → sprint-executor**

```bash
# Step 1: User initiates task (root task)
$ ailang messages send design-doc-creator "Add semantic caching feature" \
    --title "Feature: Semantic Caching" \
    --from user

# Message created with:
#   message_id: msg_001
#   parent_task_id: null (root)
#   correlation_id: corr_abc123 (auto-generated)

# Step 2: Coordinator picks up message, spawns design-doc-creator agent
# (internally runs):
$ ailang exec claude "Create design doc for semantic caching" \
    --task-id=msg_001 \
    --workspace=/path/to/ailang \
    --register-task

# Step 3: design-doc-creator completes, sends to sprint-planner
# (Claude agent runs via ailang messages):
$ ailang messages send sprint-planner "Plan sprint for semantic caching" \
    --title "Sprint: Semantic Caching" \
    --from design-doc-creator \
    --parent-task msg_001 \
    --correlation-id corr_abc123

# Message created with:
#   message_id: msg_002
#   parent_task_id: msg_001 (linked!)
#   correlation_id: corr_abc123 (inherited)

# Step 4: Coordinator spawns sprint-planner
$ ailang exec claude "Create sprint plan..." \
    --task-id=msg_002 \
    --parent-task-id=msg_001 \
    --workspace=/path/to/ailang

# Step 5: sprint-planner completes, sends to sprint-executor
$ ailang messages send sprint-executor "Execute sprint plan" \
    --parent-task msg_002 \
    --correlation-id corr_abc123

# Message created with:
#   message_id: msg_003
#   parent_task_id: msg_002 (sprint-planner)
#   correlation_id: corr_abc123 (same chain)
```

**Result: Complete hierarchy in Observatory**

```
Workspace: /path/to/ailang
└── Task: msg_001 (design-doc-creator) [$0.45, 15min]
    ├── Span: ailang.exec (provider=claude)
    │   └── claude.execute
    │       └── tool spans...
    └── Task: msg_002 (sprint-planner) [$0.32, 8min]
        ├── Span: ailang.exec (provider=claude)
        │   └── claude.execute
        └── Task: msg_003 (sprint-executor) [$1.23, 45min]
            └── Span: ailang.exec (provider=claude)

Total cost for correlation corr_abc123: $2.00
```

### Example 4: NDJSON Streaming Output

```bash
$ ailang exec claude "Add tests for parser" --workspace /repo --output-format stream-json

{"type":"exec_start","provider":"claude","model":"haiku","task_id":"exec_abc123"}
{"type":"turn_start","turn":1}
{"type":"text","content":"I'll add tests for the parser..."}
{"type":"tool_use","tool":"Read","input":"/repo/internal/parser/parser.go"}
{"type":"tool_result","tool":"Read","output":"[file contents]"}
{"type":"text","content":"Now let me write the tests..."}
{"type":"tool_use","tool":"Write","input":"/repo/internal/parser/parser_test.go"}
{"type":"turn_end","turn":1,"tokens_in":1500,"tokens_out":800}
{"type":"exec_end","success":true,"duration_ms":45000,"cost_usd":0.023}
```

### Example 5: Eval Suite with Hierarchy

```bash
# Run eval suite as a tracked task
$ ailang messages send eval-runner "Run baseline v0.6.4" \
    --title "Eval: v0.6.4 Baseline" \
    --from user

# Coordinator spawns eval via exec
$ ailang exec eval-suite --models claude-sonnet-4-5,gpt5-mini \
    --task-id=msg_eval_001 \
    --register-task

# Each benchmark creates child spans under the eval task
# Observatory shows:
#   Task: msg_eval_001 (eval-runner)
#   └── eval.suite
#       ├── eval.benchmark: fizzbuzz (claude-sonnet-4-5)
#       ├── eval.benchmark: fizzbuzz (gpt5-mini)
#       ├── eval.benchmark: cli_args (claude-sonnet-4-5)
#       └── ...
```

## Success Criteria

- [ ] `ailang exec claude "prompt"` works with streaming output
- [ ] `ailang exec gemini "prompt"` works with streaming output
- [ ] `ailang exec openai "prompt" --api-only` works for API-only mode
- [ ] `ailang messages send` accepts `--parent-task` and `--correlation-id`
- [ ] Messages store and propagate hierarchy metadata
- [ ] Coordinator delegates via `ailang exec` with hierarchy context
- [ ] Eval agent mode uses `ailang exec` (traces link to eval suite span)
- [ ] Environment setup code deduplicated (~100 LOC removed)
- [ ] Observatory shows unified hierarchy: Workspace → Task → Sub-tasks → Spans
- [ ] Cost aggregation works up the task hierarchy tree
- [ ] All existing tests passing
- [ ] Documentation updated with new exec and message flags

## Testing Strategy

**Unit tests:**
- `cmd/ailang/exec_test.go` - Provider selection, flag parsing
- `cmd/ailang/messages_test.go` - Hierarchy flag parsing, metadata storage
- `internal/executor/environment_test.go` - Environment building
- `internal/messaging/store_test.go` - Hierarchy column storage/retrieval

**Integration tests:**
- Run `ailang exec claude "hello" --dry-run` and verify span creation
- Run `ailang messages send ... --parent-task=X` and verify hierarchy stored
- Run via coordinator, verify task→exec span linkage with parent context
- Run eval suite, verify hierarchy in Observatory
- Test multi-level hierarchy: task → subtask → sub-subtask

**Manual testing:**
- Execute real task: `ailang exec claude "ls"`
- Full workflow: messages → exec → child messages → exec
- Verify traces appear in Observatory dashboard with correct hierarchy
- Confirm cost attribution rolls up the task tree
- Test Control Plane filtering by correlation_id

## Non-Goals

**Not in this feature:**
- **Approval flow in exec** - Keep in coordinator, exec is just execution
- **Session resume in exec** - Delegates to underlying provider
- **New AI providers** - Uses existing `internal/ai/` and `internal/executor/`

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking coordinator | High | Gradual migration: add exec path, test, then switch |
| Performance overhead | Low | Single subprocess spawn, minimal latency |
| Complex flag handling | Medium | Start with essential flags, add more iteratively |

## Related Documents

**Implemented (informs design):**
- [m-unified-ai-providers.md](../../implemented/v0_5_10/m-unified-ai-providers.md) - Provider abstraction
- [m-coordinator-feedback-loop.md](../../implemented/v0_6_2/m-coordinator-feedback-loop.md) - Coordinator patterns
- [ui-collaboration-hub.md](../../implemented/v0_4_6/ui-collaboration-hub.md) - Dashboard integration
- [M-EVAL-AGENT.md](../../implemented/v0_5_0/M-EVAL-AGENT.md) - Original eval agent design

**Planned (related):**
- [m-control-plane-v4-integration.md](m-control-plane-v4-integration.md) - Control Plane dashboard
- [m-control-plane-interactive-filtering.md](m-control-plane-interactive-filtering.md) - Filtering features
- [M-EVAL-AGENT-QUEUE.md](../v0_6_3/M-EVAL-AGENT-QUEUE.md) - **Deferred design now incorporated here**

## References

- [Design Axioms](/docs/references/axioms)
- Executor Interface: `internal/executor/executor.go`
- Context Propagation: `internal/telemetry/context_propagation.go`
- Claude Executor: `internal/executor/claude/claude.go`

## Future Work

- **Cost budgeting**: `ailang exec claude "task" --budget $5.00`
- **Retry policies**: Automatic retry with exponential backoff
- **Multi-provider fallback**: Try claude, fallback to gemini on failure
- **Real-time cost alerts**: Emit warning events as cost approaches budget
- **Correlation-based queries**: `ailang trace list --correlation-id=corr_abc123`
- **Task tree visualization**: Show full task hierarchy in Control Plane sidebar
- **Auto-inheritance**: Child execs automatically inherit parent context from env
- **Message replay**: Replay a correlation chain from any point

---

**Document created**: 2026-01-07
**Last updated**: 2026-01-07
