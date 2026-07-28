# Hierarchy Algorithm (Virtual Re-parenting)

The hierarchy API in `internal/observatory/hierarchy.go` applies three levels of span correlation to produce a logical hierarchy that differs from the raw DB structure.

## Data Model Overview

```
WORKSPACE
  └── TASK (created from message)
        └── AGENT_ASSIGNMENT (which executor is running)
              └── SPANS (telemetry data)
                    └── TRACES (span groups)
```

**Key relationships:**
- **Message ID → Task ID**: Tasks created with ID format `task-{first8chars of message_id}`
- **Task → Traces**: A task may have multiple traces (coordinator + execution)
- **Trace → Spans**: A trace contains all spans with the same `trace_id`

## Level 1: Timestamp Correlation

**Function:** `applyTimestampCorrelation()`

Claude Code doesn't propagate TRACEPARENT to subprocess environments. Child spans from `ailang check` are parented to `claude.execute` instead of `exec.tool_use`.

```
Before (DB structure):
claude.execute
  ├── exec.turn
  ├── exec.tool_use (Bash: ailang check)
  └── ailang.check ← sibling, wrong!

After (virtual hierarchy):
claude.execute
  └── exec.turn
        └── exec.tool_use (Bash: ailang check)
              └── ailang.check ← correctly nested!
```

**Algorithm:**
1. Find executor spans (`claude.execute`, `gemini.execute`)
2. Collect `exec.tool_use` children (potential parents)
3. For each `ailang.*` child of executor:
   - Find tool whose time window contains the child's start time
   - Move child under that tool (in-memory only)

## Level 2: Cross-Trace Merging

**Function:** `mergeRelatedTraces()`

When TRACEPARENT is propagated, child processes create spans in a NEW trace but with `parent_span_id` pointing to the original trace. This creates "orphan root" spans.

```
Trace A (coordinator): coordinator.task.execute
Trace B (executor):    claude.execute (parent_span_id = coordinator span ID)

Merged view:
coordinator.task.execute
  └── claude.execute ← linked across traces!
```

**Algorithm:**
1. Build global span index across all traces
2. Find orphan roots (have `parent_span_id` in another trace)
3. Re-parent under the actual parent
4. Merge summary stats

## Level 3: Session-Based Merging

**Function:** `mergeSessionRelatedTraces()`

Claude Code emits telemetry (api_request, user_prompt, tool events) in separate traces with NO `parent_span_id`. However, these spans share `session.id` attribute with executor spans.

**Algorithm:**
1. Find main trace (has `coordinator.task.execute` or `claude.execute` root)
2. Extract `session.id` from main trace
3. Find orphan traces with matching `session.id`
4. Nest orphan spans under appropriate `exec.turn` spans using timestamp correlation
5. Fallback: 30-second window after turn end for API request spans

## Task ID Correlation

When clicking an event in the UI, we find its traces:

```typescript
// Priority order for finding task ID:
// 1. metadata.task_id (direct span attribute)
// 2. metadata.parent_task_id (from coordinator)
// 3. metadata.correlation_id (message correlation)
// 4. Construct task-{first8chars} from event.id
let lookupId = metadata?.task_id || metadata?.parent_task_id || metadata?.correlation_id;
if (!lookupId && event.id) {
  lookupId = `task-${event.id.substring(0, 8)}`;
}
```

## Hierarchy API vs Direct Spans API

**Problem**: Child spans (claude.execute, exec.turn, etc.) don't have `task_id` set.

**Solution**: Use hierarchy endpoint which expands to include ALL spans in linked traces:

```typescript
// In useTraceData.ts:
// 1. Call hierarchy endpoint (does proper trace expansion)
const response = await fetch(`/api/observatory/tasks/${tid}/hierarchy`);
const hierarchy = await response.json();

// 2. Extract all spans from nested structure
const allSpans = [];
for (const agent of hierarchy.agents) {
  for (const trace of agent.traces) {
    for (const spanNode of trace.spans) {
      allSpans.push(spanNode.span);
    }
  }
}
```

The hierarchy endpoint (`/api/observatory/tasks/{id}/hierarchy`):
1. Gets spans by `task_id` first
2. Collects all unique `trace_id`s from those spans
3. Fetches ALL spans in those traces (including children without task_id)
4. Applies timestamp correlation for proper nesting

## Example Hierarchy Response

```json
{
  "task": {
    "id": "task-50b9518d",
    "title": "Dashboard Trace Test",
    "status": "pending"
  },
  "agents": [{
    "agent": {
      "id": "aa_3a4daa5f8e42ecc8",
      "agent_id": "hierarchy-test",
      "provider": "claude"
    },
    "traces": [
      {
        "trace_id": "79c9736d...",
        "spans": [{"span": {"name": "coordinator.task.execute"}}],
        "summary": {"span_count": 1}
      },
      {
        "trace_id": "f87170ae...",
        "root_span": {
          "span": {"name": "ailang.exec"},
          "children": [{
            "span": {"name": "claude.execute"},
            "children": [
              {"span": {"name": "exec.turn"}},
              {"span": {"name": "exec.tool_use"}},
              {"span": {"name": "ailang.check", "children": [...]}}
            ]
          }]
        },
        "spans": [...],
        "summary": {"span_count": 15}
      }
    ]
  }]
}
```

## Known Limitation: TRACEPARENT Not Propagated

Claude Code does NOT propagate TRACEPARENT to Bash tool subprocess environments. This is documented in CLAUDE.md. DO NOT attempt to fix at runtime.

**Impact:** Child spans from `ailang run` appear as siblings in DB, but virtual re-parenting corrects this at query time.

**Workaround:** The eval harness now propagates TRACEPARENT explicitly (v0.6.3+) for `ailang run` calls.

## Backend Files

| File | Purpose |
|------|---------|
| `internal/observatory/hierarchy.go` | Task hierarchy building, timestamp correlation |
| `internal/observatory/api.go` | REST API handlers |
| `internal/observatory/backend.go` | Backend interface |
| `internal/observatory/store.go` | SQLite implementation |

## Frontend Files

| File | Purpose |
|------|---------|
| `ui/src/features/controlplane/ControlPlane.tsx` | Event click handling |
| `ui/src/features/controlplane/hooks/useTraceData.ts` | Spans via hierarchy API |
| `ui/src/features/controlplane/components/TraceWaterfall.tsx` | Visualization |
