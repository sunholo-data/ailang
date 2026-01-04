# M-OTEL-CROSS-PROCESS: Cross-Process Trace Linking

**Status:** Planned
**Target:** v0.6.3
**Priority:** P1 (High)
**Estimated:** 1.5 days (12 hours)
**Dependencies:** M-OTEL-EXTENDED (in progress), Telemetry infrastructure (v0.6.1)
**Created:** 2026-01-04
**Updated:** 2026-01-04

## Problem Statement

AILANG's coordinator delegates tasks to external CLI executors (Claude Code, Gemini CLI). While both AILANG and the CLI tools have OpenTelemetry instrumentation, **traces are completely isolated**:

```
AILANG Trace (trace_id: abc123)             CLI Trace (trace_id: xyz789)
├── coordinator.task.execute                 ├── claude.session (separate!)
│   └── claude.execute                       │   ├── tool_use: Read
│       └── [subprocess starts here]         │   ├── tool_use: Edit
│           NO LINK TO CHILD →→→→→→→→→→→→→   │   └── ailang run ???
│                                            │       └── [yet another trace]
```

**The Gap:** When Claude Code or Gemini CLI spawns `ailang run`, that creates a **third isolated trace**. No parent-child relationship exists between:

1. Coordinator → Executor (AILANG spans)
2. Executor subprocess → CLI session (CLI spans)
3. CLI → `ailang run` (AILANG spans again)

**Impact:**
- Cannot trace a task from GitHub issue to final code execution
- Performance analysis requires manual correlation
- Cost attribution fragmented across isolated traces
- Debugging distributed workflows requires guesswork

## Current State: What Links Today?

### Metadata Already Available

| Component | Attribute | Passed To Child? | Queryable? |
|-----------|-----------|------------------|------------|
| Coordinator | `task.id` | ✅ Via message content | ✅ Span attribute |
| Executor | `session.id` | ✅ To Claude (`--session-id`) | ✅ Span attribute |
| Executor | `task.workspace` | ✅ Via `PWD` env | ❌ Not a span attr |
| Claude CLI | `session_id` | ❓ Unknown if passed to subprocesses | ✅ In CLI telemetry |
| Gemini CLI | `session_id` | ❓ Unknown if passed to subprocesses | ✅ In CLI telemetry |

### Can We Link Post-Run Today?

**Yes, but manually:**

1. Query AILANG traces by `task.id` or `session.id`
2. Query CLI traces by `session_id` (if same value)
3. Correlate by timestamp + workspace directory

**Problems:**
- Requires querying multiple trace backends
- No parent-child hierarchy for waterfall views
- Session ID passed to Claude but **not as an OTEL attribute by Claude**
- No programmatic correlation possible

## Goals

**Primary Goal:** Enable end-to-end distributed tracing from coordinator through CLI executors to `ailang run` subprocesses.

**Success Metrics:**
1. AILANG executor spans become parents of CLI subprocess spans (if CLIs support it)
2. CLI subprocess → `ailang run` spans linked (if CLIs propagate context)
3. Session ID recorded as span attribute for fallback correlation
4. Post-run analysis can link traces via shared task/session IDs
5. Zero breaking changes to existing telemetry

## Solution Design

### Architecture: Three-Layer Linking

**Key Insight:** Environment variables are inherited by child processes by default in Unix/Linux. Even if Claude/Gemini CLI don't explicitly use TRACEPARENT, they **pass it through** to subprocesses like `ailang run`.

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Layer 1: AILANG Executor → CLI Subprocess (We Control) ✅ GUARANTEED     │
│                                                                          │
│   coordinator.task.execute (trace_id: abc123, span_id: span001)         │
│   └── claude.execute (span_id: span002)                                  │
│       └── subprocess env: TRACEPARENT=00-abc123-span002-01               │
│                           AILANG_TASK_ID=task_456                        │
│                           AILANG_SESSION_ID=sess_789                     │
├─────────────────────────────────────────────────────────────────────────┤
│ Layer 2: CLI Tool Internal Spans (External Dependency) ❓ UNKNOWN        │
│                                                                          │
│   IF Claude/Gemini CLI reads TRACEPARENT for its OWN spans:              │
│   └── claude.session (parent: span002) ← CHILD of AILANG span!          │
│                                                                          │
│   IF NOT (most likely):                                                  │
│   └── claude.session (trace_id: NEW) ← Isolated trace                   │
│       └── CLI's internal spans are separate (not our concern)            │
│                                                                          │
│   EITHER WAY: Env vars are inherited by child processes!                 │
├─────────────────────────────────────────────────────────────────────────┤
│ Layer 3: CLI → ailang run (Works via Env Inheritance) ✅ GUARANTEED      │
│                                                                          │
│   Claude/Gemini CLI spawns: ailang run main.ail                          │
│   └── ailang run inherits TRACEPARENT from CLI's environment             │
│   └── ailang run extracts context, creates CHILD span of span002         │
│                                                                          │
│   Result: Direct parent-child link from executor → ailang run            │
│           (CLI's internal spans may be in separate trace - that's OK)    │
└─────────────────────────────────────────────────────────────────────────┘
```

### Environment Variable Inheritance Flow

```
AILANG Coordinator
    │
    │ exec.Command() with env: TRACEPARENT=00-abc123-span002-01
    ↓
Claude Code CLI (receives env, may ignore TRACEPARENT for own spans)
    │
    │ exec.Command("ailang", "run", ...) ← inherits parent's env by default!
    ↓
ailang run (receives TRACEPARENT via inherited env)
    │
    │ telemetry.ExtractTraceContext() reads TRACEPARENT
    ↓
Creates span as CHILD of span002 ✅
```

**This works because:**
1. Unix process model: child processes inherit parent's environment by default
2. CLI tools don't typically sanitize/clear inherited env vars
3. `ailang run` explicitly extracts and uses TRACEPARENT

### Implementation Plan

#### Phase 1: Context Propagation Helpers - 3 hours

**Add helper function to telemetry package:**

```go
// internal/telemetry/context_propagation.go

package telemetry

import (
    "context"
    "strings"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// InjectTraceContext adds W3C trace context to environment variables.
// Returns the updated environment slice.
//
// Injected variables:
//   - TRACEPARENT: W3C trace context (00-{trace_id}-{span_id}-{flags})
//   - TRACESTATE: Vendor-specific state (if present)
func InjectTraceContext(ctx context.Context, env []string) []string {
    carrier := make(map[string]string)
    otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))

    for key, value := range carrier {
        // W3C spec uses lowercase, but env vars conventionally uppercase
        envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
        env = append(env, envKey+"="+value)
    }

    return env
}

// InjectCorrelationIDs adds AILANG-specific correlation IDs to environment.
// These serve as fallback correlation when TRACEPARENT isn't supported.
func InjectCorrelationIDs(env []string, taskID, sessionID string) []string {
    if taskID != "" {
        env = append(env, "AILANG_TASK_ID="+taskID)
    }
    if sessionID != "" {
        env = append(env, "AILANG_SESSION_ID="+sessionID)
    }
    return env
}
```

**Update Claude executor:**

```go
// internal/executor/claude/claude.go (around line 130)

func (e *Executor) ExecuteStreaming(ctx context.Context, task *Task, handler EventHandler) (*Result, error) {
    ctx, span := claudeTracer.Start(ctx, "claude.execute", ...)
    defer span.End()

    sessionID := uuid.New().String()
    span.SetAttributes(attribute.String("session.id", sessionID))

    // Build command...
    cmd := exec.CommandContext(ctx, e.claudePath, args...)

    // Environment setup
    env := os.Environ()
    env = append(env, fmt.Sprintf("AILANG_STDLIB_PATH=%s", stdlibPath))
    if task.Workspace != "" {
        env = append(env, fmt.Sprintf("PWD=%s", task.Workspace))
    }

    // NEW: Inject trace context for distributed tracing
    env = telemetry.InjectTraceContext(ctx, env)

    // NEW: Inject correlation IDs for fallback linking
    env = telemetry.InjectCorrelationIDs(env, task.ID, sessionID)

    cmd.Env = env
    // ...
}
```

**Same change for Gemini executor** (`internal/executor/gemini/gemini.go`).

#### Phase 2: CLI Context Extraction (env + flags) - 4 hours

When `ailang run` starts, check for inherited trace context:

```go
// cmd/ailang/main.go or internal/telemetry/context_propagation.go

// ExtractTraceContext reads W3C trace context from environment variables.
// Returns a context with the extracted trace context, or the original context
// if no trace context is found.
func ExtractTraceContext(ctx context.Context) context.Context {
    carrier := make(map[string]string)

    // Read trace context from environment
    for _, env := range os.Environ() {
        parts := strings.SplitN(env, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := strings.ToLower(strings.ReplaceAll(parts[0], "_", "-"))
        // Only extract known propagation headers
        if key == "traceparent" || key == "tracestate" {
            carrier[key] = parts[1]
        }
    }

    if len(carrier) == 0 {
        return ctx
    }

    return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(carrier))
}

// ExtractCorrelationIDs reads AILANG-specific correlation IDs from environment.
func ExtractCorrelationIDs() (taskID, sessionID string) {
    return os.Getenv("AILANG_TASK_ID"), os.Getenv("AILANG_SESSION_ID")
}
```

**Usage in CLI entrypoint:**

```go
// cmd/ailang/run.go or wherever RunFile is called

func runCommand(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Extract parent trace context if running as subprocess
    ctx = telemetry.ExtractTraceContext(ctx)

    // Extract correlation IDs for span attributes
    taskID, sessionID := telemetry.ExtractCorrelationIDs()

    ctx, span := tracer.Start(ctx, "ailang.run",
        trace.WithAttributes(
            attribute.String("file.path", args[0]),
            attribute.String("ailang.task_id", taskID),     // For correlation
            attribute.String("ailang.session_id", sessionID),
        ))
    defer span.End()

    // ...
}
```

#### Phase 3: CLI Tool Verification - 3 hours

**Test whether Claude Code accepts TRACEPARENT:**

```bash
# Test script
export TRACEPARENT="00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
export CLAUDE_CODE_ENABLE_TELEMETRY=1
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

claude -p "echo hello" --output-format json

# Check if traces in backend have parent span_id = 00f067aa0ba902b7
```

**Test whether Gemini CLI accepts TRACEPARENT:**

```bash
export TRACEPARENT="00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
# ... run gemini CLI and check backend
```

**Document findings and file feature requests if needed.**

#### Phase 4: Span Links as Fallback - 2 hours

If CLI tools don't accept TRACEPARENT, use **span links** instead of parent-child:

```go
// When we know the CLI created its own trace, link to it
span.AddLink(trace.Link{
    SpanContext: trace.SpanContextFromContext(cliCtx),
    Attributes: []attribute.KeyValue{
        attribute.String("link.type", "correlation"),
        attribute.String("link.session_id", sessionID),
    },
})
```

This creates a queryable relationship without requiring parent-child.

### Trace Hierarchy Scenarios

#### Scenario A: Best Case - CLI Also Uses TRACEPARENT

```
AILANG Trace (trace_id: abc123)
├── coordinator.task.execute
│   └── claude.execute (span_id: span002)
│       └── claude.session (CHILD of span002 - CLI reads TRACEPARENT)
│           ├── tool_use: Read
│           ├── tool_use: Edit
│           └── ailang.run (GRANDCHILD of span002)
│               ├── compile.pipeline
│               └── eval (if applicable)
```

**Result:** Complete waterfall view - all spans in one trace.
**Probability:** Low (CLI tools don't document TRACEPARENT support)

#### Scenario B: Expected Case - CLI Ignores TRACEPARENT, Env Inherited

```
AILANG Trace (trace_id: abc123)           CLI Trace (trace_id: xyz789)
├── coordinator.task.execute               ├── claude.session (own trace)
│   └── claude.execute (span002)           │   ├── tool_use: Read
│       │                                  │   ├── tool_use: Edit
│       │                                  │   └── tool_use: Bash "ailang run"
│       │                                  │
│       └── ailang.run (CHILD of span002!) ◀── env inherited through CLI
│           ├── compile.pipeline
│           └── eval
```

**Result:** AILANG spans linked (executor → ailang run), CLI spans separate.
**Probability:** High (expected behavior)

**Key insight:** Even though CLI's internal spans are in a separate trace, `ailang run` still receives TRACEPARENT via env inheritance and creates a direct child span of `claude.execute`.

#### Scenario C: Edge Case - CLI Sanitizes Environment

```
AILANG Trace                              AILANG Trace #2
├── coordinator.task.execute               ├── ailang.run (orphan - new trace)
│   └── claude.execute                     │   └── compile.pipeline
│       session.id=sess_001                │       task_id=task_456
│       task_id=task_456                   │       session_id=sess_001
│                                          │
│   [LINK: task_id + session_id] ─────────▶│
```

**Result:** Separate traces, linked by correlation IDs (fallback).
**Probability:** Very Low (CLI tools rarely sanitize env)
**Mitigation:** Correlation IDs (AILANG_TASK_ID, AILANG_SESSION_ID) recorded as span attributes for query-based linking.

### Post-Run Analysis Queries

**Query 1: Find all traces for a task**

```sql
-- Honeycomb/GCP Trace Query
SELECT trace_id, span_name, duration_ms
FROM spans
WHERE attributes['ailang.task_id'] = 'task_456'
   OR attributes['task.id'] = 'task_456'
ORDER BY start_time
```

**Query 2: Find CLI session for AILANG executor span**

```sql
SELECT * FROM spans
WHERE attributes['session.id'] = 'sess_001'
   OR attributes['session_id'] = 'sess_001'
```

**Query 3: Link traces by time window + workspace**

```sql
SELECT trace_id, span_name, attributes
FROM spans
WHERE start_time BETWEEN @executor_start AND @executor_end
  AND (attributes['task.workspace'] LIKE '/path/to/worktree/%'
       OR attributes['cwd'] LIKE '/path/to/worktree/%')
```

## Testing Strategy

### Unit Tests

```go
func TestInjectTraceContext(t *testing.T) {
    // Setup trace with known IDs
    ctx, span := tracer.Start(context.Background(), "test")
    defer span.End()

    env := telemetry.InjectTraceContext(ctx, []string{"PATH=/bin"})

    // Verify TRACEPARENT was injected
    var traceparent string
    for _, e := range env {
        if strings.HasPrefix(e, "TRACEPARENT=") {
            traceparent = strings.TrimPrefix(e, "TRACEPARENT=")
        }
    }

    require.NotEmpty(t, traceparent)
    require.True(t, strings.HasPrefix(traceparent, "00-"))

    // Verify format: 00-{trace_id}-{span_id}-{flags}
    parts := strings.Split(traceparent, "-")
    require.Len(t, parts, 4)
    require.Len(t, parts[1], 32) // trace_id
    require.Len(t, parts[2], 16) // span_id
}

func TestExtractTraceContext(t *testing.T) {
    // Simulate subprocess environment
    os.Setenv("TRACEPARENT", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
    defer os.Unsetenv("TRACEPARENT")

    ctx := telemetry.ExtractTraceContext(context.Background())
    spanCtx := trace.SpanContextFromContext(ctx)

    require.True(t, spanCtx.IsValid())
    require.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spanCtx.TraceID().String())
}
```

### Integration Tests

```bash
# Test full flow with local OTEL collector
make test-trace-propagation

# Manual verification
DEBUG_TRACE=1 ailang coordinator start &
ailang messages send coordinator "test task" --type bug --from test
# Check GCP Cloud Trace for linked spans
```

## Success Criteria

**Layer 1: Executor → CLI Subprocess**
- [ ] TRACEPARENT injected into Claude/Gemini executor subprocesses
- [ ] Correlation IDs (task_id, session_id) injected into subprocess env

**Layer 3: CLI → ailang run (via env inheritance)**
- [ ] `ailang run` extracts TRACEPARENT from inherited environment
- [ ] `ailang run` creates child span under parent trace
- [ ] `ailang run` records correlation IDs as span attributes
- [ ] Verified: `ailang run` spawned by CLI is direct child of executor span

**Fallback & Analysis**
- [ ] Post-run queries can link traces by session_id or task_id
- [ ] Documentation of expected scenario (B) and edge cases

**Optional: Layer 2 (CLI internal spans)**
- [ ] Test whether Claude Code reads TRACEPARENT for own spans
- [ ] Test whether Gemini CLI reads TRACEPARENT for own spans
- [ ] File feature requests if useful (nice-to-have, not blocking)

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/telemetry/context_propagation.go` | New file - inject/extract helpers | +100 |
| `internal/executor/claude/claude.go` | Inject trace context + correlation IDs | +10 |
| `internal/executor/gemini/gemini.go` | Same as Claude | +10 |
| `cmd/ailang/run.go` | Extract context (env + CLI flags) | +25 |
| `cmd/ailang/check.go` | Same as run | +25 |
| `cmd/ailang/eval_suite.go` | Same as run | +25 |
| `cmd/ailang/repl.go` | Same as run | +25 |
| `docs/docs/guides/telemetry.md` | Document cross-process linking | +60 |
| `internal/telemetry/context_propagation_test.go` | Unit tests | +120 |

**Total: ~400 LOC**

### CLI Flags Added

| Command | New Flags |
|---------|-----------|
| `ailang run` | `--trace-parent`, `--trace-id`, `--parent-span`, `--task-id`, `--session-id` |
| `ailang check` | Same |
| `ailang eval-suite` | Same |
| `ailang repl` | `--trace-parent`, `--trace-id`, `--session-id` (no task-id) |

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Claude Code ignores TRACEPARENT for own spans | High | Low | Only affects CLI's internal spans; `ailang run` still links via env inheritance |
| Gemini CLI ignores TRACEPARENT for own spans | High | Low | Same as above |
| CLI tools sanitize/clear inherited env vars | Very Low | Medium | Correlation IDs (task_id, session_id) as fallback |
| Performance overhead from context injection | Low | Low | <1μs per injection |

**Important:** The main risk (CLI ignoring TRACEPARENT) has **low impact** because env vars are inherited by child processes. The `ailang run` commands spawned by CLI tools will still receive and use TRACEPARENT, creating direct parent-child links to the executor span.

## Design Decision: Environment Variables vs CLI Arguments

### Options Comparison

| Aspect | Environment Variables | CLI Arguments |
|--------|----------------------|---------------|
| **OTEL Standard** | ✅ W3C spec uses env vars | ❌ Non-standard |
| **Transparency** | ✅ Automatic, invisible | ⚠️ Visible in logs/history |
| **Subprocess inheritance** | ✅ Automatic | ❌ Must pass explicitly |
| **CI/CD integration** | ⚠️ Some CIs sanitize env | ✅ Always works |
| **Debugging** | ⚠️ Less visible | ✅ Easy to see in command |
| **CLI pollution** | ✅ No extra flags | ❌ Adds `--trace-parent` etc |
| **Orchestration tools** | ✅ Easy to inject | ⚠️ Must modify command line |

### Recommendation: Hybrid Approach

**Primary: Environment Variables (OTEL standard)**
```bash
# Automatic propagation - standard OTEL pattern
TRACEPARENT=00-abc123-span456-01 ailang run main.ail
```

**Secondary: CLI Flags (explicit override)**
```bash
# Explicit override - useful for CI, debugging
ailang run --trace-parent "00-abc123-span456-01" main.ail
ailang run --trace-id abc123 --parent-span span456 main.ail
```

**Precedence:** CLI flags > Environment variables

### Implementation

```go
// cmd/ailang/run.go
func extractTraceContext(cmd *cobra.Command) context.Context {
    ctx := context.Background()

    // 1. Try CLI flags first (highest priority)
    if tp, _ := cmd.Flags().GetString("trace-parent"); tp != "" {
        carrier := propagation.MapCarrier{"traceparent": tp}
        return otel.GetTextMapPropagator().Extract(ctx, carrier)
    }

    // 2. Fall back to environment variables (OTEL standard)
    return telemetry.ExtractTraceContext(ctx)
}
```

### CLI Flags to Add

```go
// For all traced commands: run, check, eval-suite, repl
cmd.Flags().String("trace-parent", "", "W3C traceparent header for distributed tracing")
cmd.Flags().String("trace-id", "", "Trace ID to join (alternative to --trace-parent)")
cmd.Flags().String("parent-span", "", "Parent span ID (requires --trace-id)")
```

### CI/CD Examples

**GitHub Actions (env var):**
```yaml
- name: Run AILANG tests
  env:
    TRACEPARENT: "00-${{ github.run_id }}-${{ github.job }}-01"
  run: ailang run tests/integration.ail
```

**GitHub Actions (CLI flag):**
```yaml
- name: Run AILANG tests
  run: |
    ailang run --trace-id "${{ github.run_id }}" tests/integration.ail
```

**Cloud Build:**
```yaml
steps:
  - name: 'ailang'
    args: ['run', '--trace-parent', '$_TRACEPARENT', 'main.ail']
```

### Correlation IDs

Same hybrid pattern for `AILANG_TASK_ID` and `AILANG_SESSION_ID`:

```bash
# Environment (automatic)
AILANG_TASK_ID=task_123 AILANG_SESSION_ID=sess_456 ailang run main.ail

# CLI (explicit)
ailang run --task-id task_123 --session-id sess_456 main.ail
```

## Open Questions

1. **Should we file feature requests to CLI tools now?**
   - Recommendation: Yes, regardless of implementation. Industry-standard feature.

2. **Should correlation IDs use task_id or session_id as primary?**
   - Recommendation: Both. Task_id for AILANG context, session_id for CLI context.

3. **Should we add span events for context propagation status?**
   - Recommendation: Yes, log whether TRACEPARENT was found in child.

## Related Documents

- [M-OTEL-EXTENDED](./m-otel-extended-instrumentation.md) - Extended instrumentation (prerequisite)
- [docs/docs/guides/telemetry.md](../../../docs/docs/guides/telemetry.md) - Current telemetry docs
- [OpenTelemetry Env Carriers Spec](https://opentelemetry.io/docs/specs/otel/context/env-carriers/)

## Appendix: CLI Telemetry Documentation Links

**Claude Code:**
- [Monitoring Usage](https://code.claude.com/docs/en/monitoring-usage) - Metrics/logs export, no TRACEPARENT docs

**Gemini CLI:**
- [Telemetry Docs](https://google-gemini.github.io/gemini-cli/docs/cli/telemetry.html) - OTEL export, no TRACEPARENT docs

**Feature Request Templates:**

Claude Code:
> Title: Support W3C TRACEPARENT environment variable for distributed tracing
>
> When Claude Code is invoked as a subprocess, it should read the TRACEPARENT
> environment variable (W3C Trace Context) and create spans as children of the
> parent trace. This enables end-to-end distributed tracing through orchestration
> tools that delegate tasks to Claude Code.

Gemini CLI:
> Title: Support TRACEPARENT environment variable for distributed trace context
>
> When Gemini CLI is invoked by a parent process that sets TRACEPARENT,
> the CLI should continue that distributed trace rather than starting a new one.
> This is standard OpenTelemetry practice for subprocess context propagation.
