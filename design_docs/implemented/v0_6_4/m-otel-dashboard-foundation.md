# M-OTEL-DASHBOARD Foundation Implementation Report

**Status**: Implemented (Foundation Phase)
**Version**: v0.6.4
**Implemented**: 2026-01-04
**Design Doc**: [../planned/v0_6_4/m-otel-dashboard.md](../../planned/v0_6_4/m-otel-dashboard.md)

## Executive Summary

Implemented the foundational OTLP receiver and UI enhancements for the Observatory dashboard. This enables Claude Code and Gemini CLI telemetry to flow into the local Observatory for unified observability.

**Key Achievement:** Claude Code sessions now send telemetry to the local Observatory dashboard, providing real-time visibility into token usage, costs, and session activity.

## What Was Implemented

### 1. OTLP Receiver Enhancements (`internal/observatory/otlp_receiver.go`)

**Problem:** Claude Code sends telemetry via OTLP HTTP JSON format, but the receiver was using `json.Unmarshal` which doesn't handle OTEL's protobuf-to-JSON mapping.

**Solution:**
- Changed to `protojson.Unmarshal` for correct OTEL JSON parsing
- Added string-to-number extraction (Claude Code sends numbers as strings)
- Support for both protobuf and JSON content types

```go
// Before (broken):
json.Unmarshal(body, &exportReq)

// After (working):
protojson.Unmarshal(body, &exportReq)
```

**String extraction for Claude Code:**
```go
case string:
    // Claude Code sends numbers as strings
    if i, err := strconv.Atoi(val); err == nil {
        return i
    }
```

### 2. Trace Filtering (`internal/server/server.go`)

**Problem:** Dashboard was flooded with 4000+ noisy spans from HTTP polling endpoints.

**Solution:** Added OTEL filter to exclude:
- `/api/approvals`, `/api/hierarchy`, `/api/statistics`
- `/api/observatory/*`, `/api/metrics/*`
- `/assets/*`, `/v1/*` (OTLP endpoints)
- `/health`, `/ws`, `/ws/observatory`

### 3. Timestamp Parsing (`internal/observatory/store.go`)

**Problem:** Trace summaries showed "0001-01-01T00:00:00Z" because SQLite uses space-separated timestamps.

**Solution:** Added multiple time format parsers:
```go
// SQLite format (space, not T)
"2006-01-02 15:04:05.999999999-07:00"
"2006-01-02 15:04:05-07:00"
```

### 4. Observatory UI Enhancements (`ui/src/features/observatory/`)

**Inline Trace Expansion:**
- Click row to expand details below (not at bottom)
- Collapsible trace detail view
- Visual indicator (▶/▼) for expand state

**Enhanced Span Detail:**
- Trace summary with totals (spans, tokens, cost, duration)
- Per-span metrics: input/output tokens, cache tokens, cost
- Provider and model badges
- Expandable attributes section

**Format Function Fixes:**
- All format functions now handle string inputs via `Number()` conversion
- Prevents `TypeError: e.toFixed is not a function`

### 5. Documentation

**Website docs** (`docs/docs/guides/telemetry.md`):
- New "Observatory Dashboard Integration (v0.6.3+)" section
- Architecture diagram
- Step-by-step setup for Claude Code and Gemini CLI
- Environment variables reference
- OTLP endpoint documentation

**Trace-debugger skill** (`.claude/skills/trace-debugger/skill.md`):
- Observatory as "Option 3" in prerequisites
- Full configuration guide

## Configuration Required

### Claude Code (`~/.claude/settings.json`)
```json
{
  "env": {
    "CLAUDE_CODE_ENABLE_TELEMETRY": "1",
    "OTEL_LOGS_EXPORTER": "otlp",
    "OTEL_METRICS_EXPORTER": "otlp",
    "OTEL_EXPORTER_OTLP_PROTOCOL": "http/json",
    "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:1957",
    "OTEL_RESOURCE_ATTRIBUTES": "ailang.source=user"
  }
}
```

### Gemini CLI (shell profile)
```bash
export GEMINI_TELEMETRY_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957
export OTEL_EXPORTER_OTLP_PROTOCOL=http/json
export OTEL_RESOURCE_ATTRIBUTES="ailang.source=user"
```

## Files Modified

| File | Changes |
|------|---------|
| `internal/observatory/otlp_receiver.go` | protojson parsing, string extraction |
| `internal/server/server.go` | OTEL filter for noisy endpoints |
| `internal/observatory/store.go` | Timestamp format parsing |
| `ui/src/features/observatory/Observatory.tsx` | Inline expansion, span details, format fixes |
| `ui/src/features/observatory/Observatory.module.css` | New styles for expansion UI |
| `docs/docs/guides/telemetry.md` | Observatory integration docs |
| `.claude/skills/trace-debugger/skill.md` | Observatory setup guide |

## Remaining Work (Future Sprints)

From the original M-OTEL-DASHBOARD design doc, these phases remain:

### Not Started
- [ ] **M1-M4**: Full schema refactoring (workspaces, tasks, agent_assignments tables)
- [ ] **M5**: GCP Trace backend adapter
- [ ] **M6**: Jaeger backend adapter
- [ ] **M7**: Composite backend (local write, remote read)
- [ ] **M8**: Full REST API (20+ endpoints)
- [ ] **M9**: WebSocket hub with subscriptions
- [ ] **M12**: Frontend data hooks (useWorkspaces, useTasks, useAgents)

### Partially Complete
- [x] **M2**: SQLite store (basic, not full schema)
- [x] **M3**: Provider normalization (Claude logs, Gemini spans)
- [~] **M10**: CLI command rename (`ailang server` works, `serve` is alias)
- [~] **M11**: Server integration (OTLP endpoints working)

### Complete
- [x] OTLP receiver for traces, logs, metrics
- [x] Claude Code telemetry ingestion
- [x] Gemini CLI telemetry support (coordinator-injected)
- [x] Observatory UI with trace/span viewing
- [x] Documentation

## Metrics

| Metric | Value |
|--------|-------|
| LOC added/modified | ~300 |
| Test coverage | Existing tests pass |
| Build status | Passing |
| Documentation | Complete |

## Success Criteria Met

- [x] Claude Code telemetry flows to Observatory
- [x] Traces appear in UI with correct data
- [x] Span details show tokens, cost, attributes
- [x] Documentation enables user self-service setup
- [x] Silent failure mode (no impact if server down)

## Known Limitations

1. **Claude Code sends events, not traces** - We create synthetic spans from log records
2. **No full trace hierarchy from Claude Code** - Only Gemini CLI provides full span trees
3. **Schema is legacy** - Full M-OTEL-DASHBOARD schema migration not done
4. **No backend adapters** - GCP/Jaeger query not implemented yet

## Next Steps

1. Run full M-OTEL-DASHBOARD sprint for v0.6.5
2. Add GCP Trace backend adapter for cloud viewing
3. Implement full schema with workspaces/tasks hierarchy
4. Build provider comparison dashboard

---

**Implementation Date**: 2026-01-04
**Sprint Duration**: 1 session (~2 hours)
