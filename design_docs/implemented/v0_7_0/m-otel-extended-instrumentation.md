# M-OTEL-EXTENDED: Extended OpenTelemetry Instrumentation

**Status:** Implemented
**Target:** v0.6.3
**Implemented:** 2026-01-06
**Priority:** P1 (Medium)

## Summary

Added comprehensive OpenTelemetry instrumentation across all major AILANG components:
- Compiler pipeline (8 spans)
- Eval harness (2 spans + task hierarchy)
- Message system (7 spans)
- REPL command (2 spans)
- Check command (2 spans)

Total: **21 new span types** with zero overhead when telemetry disabled.

## Implementation Report

### M1: Compiler Pipeline ✅ COMPLETE

**File:** `internal/pipeline/pipeline_single.go`, `internal/pipeline/pipeline_module.go`

**Spans implemented:**
| Span Name | Location | Attributes |
|-----------|----------|------------|
| `compile: <filename>` | pipeline_single.go:36 | `file.path`, `file.module` |
| `compile.parse` | pipeline_single.go:81 | `file.path` |
| `compile.elaborate` | pipeline_single.go:163 | - |
| `compile.typecheck` | pipeline_single.go:203 | - |
| `compile.validate` | pipeline_single.go:353 | - |
| `compile.lower` | pipeline_single.go:449 | - |
| `compile.load` | pipeline_module.go:77 | - |
| `compile.topo_sort` | pipeline_module.go:101 | - |
| `compile.modules` | pipeline_module.go:134 | - |

**Verification:**
```bash
grep -c "compilerTracer.Start" internal/pipeline/*.go
# Result: 9 span starts
```

### M2: Eval Harness ✅ COMPLETE

**File:** `cmd/ailang/eval_suite.go`

**Spans implemented:**
| Span Name | Attributes |
|-----------|------------|
| `eval.suite` | `ailang.task_id`, models, benchmark count |
| `eval.benchmark: <id>` | `benchmark.id`, `benchmark.lang`, `model` |

**Task Hierarchy Integration:**
- Creates Observatory tasks via `/api/observatory/tasks`
- Creates agent assignments via `/api/observatory/agents`
- Spans linked to tasks via `OTEL_RESOURCE_ATTRIBUTES`

**Verification:**
```sql
SELECT COUNT(*) FROM spans WHERE task_id IS NOT NULL;
-- Result: 3202 linked spans
```

### M3: Message System ✅ COMPLETE

**File:** `internal/messaging/inbox.go`, `internal/messaging/search.go`

**Spans implemented:**
| Span Name | Attributes |
|-----------|------------|
| `messages.send` | `inbox`, `title`, `from` |
| `messages.list` | `inbox`, `status`, `limit` |
| `messages.read` | `message.id` |
| `messages.ack` | `message.id` |
| `messages.unack` | `message.id` |
| `messages.cleanup` | `older_than`, `expired_only` |
| `messages.search` | `query`, `neural` |

**Verification:**
```bash
grep -c "messagingTracer.Start" internal/messaging/*.go
# Result: 7 span starts
```

### M4: REPL Command ✅ COMPLETE

**File:** `internal/repl/repl.go`

**Spans implemented:**
| Span Name | Attributes |
|-----------|------------|
| `repl.session` | `learn_mode`, `strict_syntax` |
| `repl.input` | `input.length`, `command_type` |

**Hierarchy:** Session span parents all input spans for a complete session trace.

**Verification:**
```bash
grep -c "replTracer.Start" internal/repl/*.go
# Result: 3 span starts (1 session + 2 input locations)
```

### M5: Check Command ✅ COMPLETE

**File:** `cmd/ailang/check.go`

**Spans implemented:**
| Span Name | Attributes |
|-----------|------------|
| `ailang.check` | `file.path`, `ailang.task_id`, `ailang.session_id` |
| `check.result` | `has_errors`, `error_count` |

**Cross-Process Linking:**
- Extracts TRACEPARENT from environment
- Extracts correlation IDs for span attributes

**Verification:**
```bash
grep -c "checkTracer.Start" cmd/ailang/check.go
# Result: 3 span starts
```

## Metrics

| Component | Spans | Files Modified | LOC Added |
|-----------|-------|----------------|-----------|
| Compiler | 9 | 2 | ~150 |
| Eval Harness | 2 | 1 | ~200 |
| Messages | 7 | 2 | ~100 |
| REPL | 2 | 1 | ~50 |
| Check | 2 | 1 | ~80 |
| **Total** | **22** | **7** | **~580** |

## Testing

All instrumentation verified with:
1. Unit tests for span creation
2. Integration tests with Observatory OTLP receiver
3. Live traces visible in Google Cloud Trace Explorer
4. `ailang trace list` shows instrumented operations

## Performance

Zero overhead when telemetry disabled:
- Tracer returns no-op spans (~2ns per call)
- No allocations for disabled spans
- No network I/O when no exporter configured

## Related Documentation

- [Telemetry Guide](docs/docs/guides/telemetry.md)
- [Observatory API](internal/observatory/api.go)
- [Context Propagation](internal/telemetry/context_propagation.go)
