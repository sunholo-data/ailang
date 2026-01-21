# M-OTEL-CROSS-PROCESS: Cross-Process Trace Linking

**Status:** Implemented
**Target:** v0.6.3
**Implemented:** 2026-01-06
**Priority:** P1 (High)

## Summary

Implemented end-to-end distributed tracing across process boundaries:
- TRACEPARENT injection when spawning CLI executors
- TRACEPARENT extraction in CLI commands
- Session-based correlation for Claude Code internal events
- Task hierarchy linking in Observatory

## Implementation Report

### Phase 1: Executor Context Injection ✅ COMPLETE

**Files:**
- `internal/executor/claude/claude.go`
- `internal/executor/gemini/gemini.go`
- `internal/telemetry/context_propagation.go`

**Implementation:**
```go
// Claude executor (line 79-82)
ctx, span := claudeTracer.Start(ctx, "claude.execute", ...)

// Inject trace context for distributed tracing
env = telemetry.InjectTraceContext(ctx, env)

// Inject correlation IDs for fallback linking
env = telemetry.InjectCorrelationIDs(env, task.ID, sessionID)
```

**Environment Variables Injected:**
| Variable | Format | Purpose |
|----------|--------|---------|
| `TRACEPARENT` | `00-{trace_id}-{span_id}-{flags}` | W3C trace context |
| `TRACESTATE` | vendor-specific | Extended context |
| `AILANG_TASK_ID` | task ID string | Fallback correlation |
| `AILANG_SESSION_ID` | session ID string | Fallback correlation |

### Phase 2: CLI Context Extraction ✅ COMPLETE

**Files:**
- `cmd/ailang/check.go`
- `cmd/ailang/eval_suite.go`
- `internal/telemetry/context_propagation.go`

**Implementation:**
```go
// Check command (line 34-35)
ctx = telemetry.ExtractTraceContext(ctx)
taskID, sessionID := telemetry.ExtractCorrelationIDs()

// Eval suite (line 97-98)
ctx = telemetry.ExtractTraceContext(ctx)
```

**Functions:**
| Function | Purpose |
|----------|---------|
| `ExtractTraceContext(ctx)` | Reads TRACEPARENT/TRACESTATE from env |
| `ExtractCorrelationIDs()` | Returns task_id, session_id from env |
| `HasTraceContext(ctx)` | Checks if context has trace |

### Phase 3: Session-Based Correlation ✅ COMPLETE

**Files:**
- `internal/observatory/store.go`
- `internal/observatory/backend.go`
- `internal/observatory/otlp_receiver.go`

**Problem:** Claude Code internal events (`claude_code.tool.*`) arrive via OTLP before the parent `claude.execute` span due to OTEL batching delays.

**Solution:** Bidirectional linking:
1. Forward lookup on event arrival (by session.id)
2. Backward linking when parent arrives

**Implementation:**
```go
// store.go - LinkOrphanedSpansBySession
// Updates spans that have matching session.id but no task_id
func (s *Store) LinkOrphanedSpansBySession(sessionID, taskID, assignmentID string) (int64, error) {
    result, err := s.db.Exec(`
        UPDATE spans SET
            task_id = ?,
            agent_assignment_id = ?
        WHERE json_extract(attributes, '$."session.id"') = ?
        AND (task_id IS NULL OR task_id = '')
        AND name != 'claude.execute'
    `, taskID, assignmentID, sessionID)
    ...
}

// otlp_receiver.go - Post-processing
if normalized.Name == "claude.execute" && normalized.TaskID != "" {
    sessionID := extractString(normalized.Attributes, "session.id")
    if sessionID != "" {
        linked, _ := r.backend.LinkOrphanedSpansBySession(ctx, sessionID, ...)
        if linked > 0 {
            r.backend.RecalculateTaskAggregates(ctx, normalized.TaskID)
        }
    }
}
```

### Phase 4: Task Hierarchy Linking ✅ COMPLETE

**Files:**
- `internal/observatory/hierarchy.go`
- `internal/observatory/otlp_receiver.go`

**Hierarchy Structure:**
```
Workspace
└── Task
    └── AgentAssignment
        └── Spans (linked via task_id/agent_assignment_id)
```

**Resource Attribute Linking:**
Spans with `ailang.task_id` in resource attributes automatically link to tasks:
```go
// otlp_receiver.go - linkToTaskHierarchy
taskID := extractString(span.ResourceAttributes, "ailang.task_id")
assignmentID := extractString(span.ResourceAttributes, "ailang.assignment_id")
```

## Verification

**Linked Spans:**
```sql
SELECT COUNT(*) FROM spans WHERE task_id IS NOT NULL;
-- Result: 3202
```

**Tasks with Metrics:**
```sql
SELECT id, title, span_count, total_cost_usd FROM tasks WHERE span_count > 0 LIMIT 3;
-- eval-1767705183959779000|Eval: 46 benchmarks × 2 models|202|0.42
-- eval-1767704866591054000||5|0.04
-- eval-1767684693165654000||3|0.02
```

**Session Linking Log:**
```
observatory: linked 11 orphaned Claude Code events to task eval-xxx (session sess-yyy)
```

## Trace Hierarchy Achieved

```
AILANG Trace (trace_id: abc123)
├── eval.suite
│   └── eval.benchmark: fibonacci
│       └── anthropic.generate (child span)
└── claude.execute (span_id: span002)
    └── [Claude Code events linked via session.id]
        ├── claude_code.tool.Read
        ├── claude_code.tool.Edit
        └── claude_code.assistant.response
```

## Metrics

| Component | LOC | Files |
|-----------|-----|-------|
| Context propagation | 145 | 1 |
| Session linking | 50 | 3 |
| Hierarchy linking | 269 | 1 |
| OTLP post-processing | 30 | 1 |
| **Total** | **~500** | **6** |

## Limitations

1. **Claude Code TRACEPARENT:** Claude Code CLI doesn't read TRACEPARENT for its own spans (creates separate trace). Mitigated by session-based correlation.

2. **GCP/Jaeger backends:** `LinkOrphanedSpansBySession` returns 0 (stubs) - only SQLite backend supports post-hoc linking.

3. **Timing window:** If `claude.execute` span arrives significantly after Claude Code events, those events may have been pruned.

## Related Documentation

- [Telemetry Context Propagation](internal/telemetry/context_propagation.go)
- [Observatory Store](internal/observatory/store.go)
- [Trace Debugger Skill](.claude/skills/trace-debugger/SKILL.md)
