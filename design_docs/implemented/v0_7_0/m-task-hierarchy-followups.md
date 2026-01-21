# M-TASK-HIERARCHY Follow-ups

**Status**: Planned
**Target**: v0.6.5
**Priority**: P1 (High - Central to AILANG observability offering)
**Estimated**: 2-3 days
**Dependencies**: M-TASK-HIERARCHY (v0.6.4, implemented)

## Summary

Follow-up improvements to the M-TASK-HIERARCHY implementation based on production usage audit. The task hierarchy system is central to AILANG's observability offering, linking AI agent task execution with distributed traces for cost tracking and debugging.

## Audit Findings (January 2025)

### What Works Well

1. **Task ID Extraction Priority Chain** - All 4 sources now working:
   - `ailang.task_id` from resource attributes (OTEL_RESOURCE_ATTRIBUTES)
   - `task.id` span attribute (coordinator sets this on task execution spans)
   - `task.workspace` span attribute (executor sets worktree path)
   - `process.cwd` worktree path fallback (AILANG subprocesses)

2. **Token/Cost Tracking** - End-to-end verified:
   - Coordinator task `task-d6da20c4`: 5,857 tokens, $0.25 cost captured
   - Token extraction covers: `gen_ai.*`, `ailang.*`, `ai.*`, `task.*` naming conventions

3. **Hierarchy Building** - 92-97% test coverage:
   - Trace grouping by task works correctly
   - Chronological sorting implemented
   - Parent-child span relationships preserved

### Issues Identified

| Issue | Severity | Current State | Impact |
|-------|----------|---------------|--------|
| HTTP handler coverage | Medium | 0% for OTLP endpoints | Untested production code paths |
| Span limits hard-coded | Medium | 1000 spans per query | Large tasks may be incomplete |
| Log-to-span conversion | Low | 0% coverage | Untested MCP log ingestion |
| API handler coverage | Low | Many at 0% | CRUD operations untested |

### Span Linking Analysis

Current span distribution (13,409 total spans):

| Service | Total | Linked | Unlinked | Analysis |
|---------|-------|--------|----------|----------|
| claude-code | 7,704 | 0 | 7,704 | Direct CLI usage (not in worktree) - expected |
| ailang-run | 2,550 | 2,105 (83%) | 445 | Compilation spans - good linking |
| ailang-server | 2,345 | 0 | 2,345 | Server operations - not task-related |
| ailang-check | 288 | 108 (38%) | 180 | Mix of task and direct usage |
| ailang-eval | 257 | 0 | 257 | Eval runs - not task-related |
| ailang-messages | 159 | 18 (11%) | 141 | Message operations - mostly direct |
| ailang-coordinator | 37 | 2 (5%) | 35 | Most are daemon spans, not per-task |

**Conclusion**: 11,175 unlinked spans are expected. They represent:
- Direct CLI usage (user running `ailang run` outside tasks)
- Server/daemon operations (background processes)
- Eval runs (benchmark infrastructure, not coordinator tasks)

The backfill should only link spans that have task context (worktree path or explicit task ID).

## Proposed Improvements

### M1: OTLP HTTP Handler Tests (~150 LOC, 4 hours)

Add integration tests for the OTLP receiver HTTP handlers:

```go
// internal/observatory/otlp_receiver_http_test.go

func TestHandleTraces_ValidRequest(t *testing.T) {
    // POST valid protobuf to /v1/traces
    // Verify spans inserted into backend
}

func TestHandleTraces_InvalidProtobuf(t *testing.T) {
    // POST garbage data
    // Verify 400 response
}

func TestHandleLogs_ValidRequest(t *testing.T) {
    // POST valid log data
    // Verify converted to spans
}
```

**Files to modify:**
- `internal/observatory/otlp_receiver_test.go` (+150 LOC)

### M2: Configurable Hierarchy Limits (~80 LOC, 2 hours)

Replace hard-coded 1000 span limits with configurable options:

```go
// internal/observatory/hierarchy.go

type HierarchyOptions struct {
    MaxDepth     int
    IncludeSpans bool
    SpanLimit    int  // NEW: 0 = unlimited, default 1000
    Paginate     bool // NEW: Enable pagination for large hierarchies
}

func DefaultHierarchyOptions() HierarchyOptions {
    return HierarchyOptions{
        MaxDepth:     0,
        IncludeSpans: true,
        SpanLimit:    1000, // Sensible default
        Paginate:     false,
    }
}
```

**Current hard-coded locations:**
- Line 123: `ListSpans(ctx, SpanListOptions{..., Limit: 1000})`
- Line 133: `ListSpans(ctx, SpanListOptions{..., Limit: 1000})`

**Files to modify:**
- `internal/observatory/hierarchy.go` (+30 LOC)
- `internal/observatory/hierarchy_test.go` (+50 LOC)

### M3: API Handler Test Coverage (~200 LOC, 4 hours)

Add tests for untested API handlers:

```go
// internal/observatory/api_test.go (expand existing)

func TestHandleAgentAssignments_CRUD(t *testing.T) {
    // Create, Read, Update, Delete agent assignments
}

func TestHandleGetAgentStats(t *testing.T) {
    // Verify stats aggregation
}

func TestHandleGetWorkspaceStats(t *testing.T) {
    // Verify workspace-level metrics
}
```

**Handlers at 0% coverage:**
- `handleUpdateWorkspace`
- `handleGetWorkspaceStats`
- `handleDeleteTask`
- `handleListAgentAssignments`
- `handleCreateAgentAssignment`
- `handleGetAgentAssignment`
- `handleUpdateAgentAssignment`
- `handleDeleteAgentAssignment`
- `handleGetAgentStats`

**Files to modify:**
- `internal/observatory/api_test.go` (+200 LOC)

### M4: Log-to-Span Conversion Tests (~100 LOC, 2 hours)

Test the log ingestion path for MCP servers:

```go
// internal/observatory/otlp_receiver_test.go (expand)

func TestConvertLogToSpan_MCPFormat(t *testing.T) {
    // MCP server log format
    // Verify converted to trace span
}

func TestProcessResourceLogs_BatchProcessing(t *testing.T) {
    // Multiple logs in single request
    // Verify all converted
}
```

**Files to modify:**
- `internal/observatory/otlp_receiver_test.go` (+100 LOC)

### M5: UI Lazy Loading for Large Traces (~100 LOC, 3 hours)

When a task has 2000+ spans, the UI should lazy-load:

```typescript
// ui/src/components/observatory/TraceView.tsx

interface TraceViewProps {
    traceId: string;
    initialLimit?: number; // Default 100
}

// Load first 100 spans, then paginate
const [spanPage, setSpanPage] = useState(0);
const spans = usePaginatedSpans(traceId, 100, spanPage);
```

**Files to modify:**
- `ui/src/components/observatory/TraceView.tsx` (+50 LOC)
- `ui/src/hooks/useSpans.ts` (+50 LOC)

### M6: Calculate Cost in OTLP Receiver (~80 LOC, 2 hours) ✅ IMPLEMENTED

**Problem:** AI providers emit tokens but NOT cost. Cost shows $0.00 for ALL traces.

**Current state:** 827,116 tokens captured across eval runs, but $0.00 cost.

**Why not fix in AI providers?** Circular dependency - `eval_harness` imports `ai`, so `ai` can't import `eval_harness` for `models.yml` pricing.

**Solution:** Calculate cost in the OTLP receiver using tokens + `models.yml` pricing:

```go
// internal/observatory/otlp_receiver.go (in convertSpan)

tokensIn := extractInt(attrs, "ai.tokens_in", ...)
tokensOut := extractInt(attrs, "ai.tokens_out", ...)
model := extractString(attrs, "gen_ai.request.model", "ailang.model")

// NEW: Calculate cost from tokens using models.yml pricing
costUSD := extractFloat(attrs, "ai.cost_usd", ...) // Try explicit first
if costUSD == 0 && tokensIn > 0 && model != "" {
    costUSD = CalculateCostFromTokens(model, tokensIn, tokensOut)
}
```

**Implementation (January 2025):**
- Created `internal/observatory/pricing.go` (~170 LOC) with:
  - `CalculateCostFromTokens(model, tokensIn, tokensOut)` - Main calculation function
  - `normalizeModelName(model)` - Model name normalization (date suffixes, API names)
  - Lazy loading of models.yml at first use (sync.Once pattern)
  - Support for 18 models with accurate pricing
- Modified `internal/observatory/otlp_receiver.go` (+10 LOC):
  - Cost calculation in `convertSpan()` and `convertLogToSpan()`
- Added tests in `internal/observatory/pricing_test.go` (~120 LOC):
  - `TestNormalizeModelName` - All model name variants
  - `TestCalculateCostFromTokens_*` - Zero tokens, with config, unknown model

**Files modified:**
- `internal/observatory/pricing.go` (+170 LOC, new file)
- `internal/observatory/pricing_test.go` (+120 LOC, new file)
- `internal/observatory/otlp_receiver.go` (+10 LOC)
- `internal/observatory/otlp_receiver_test.go` (+90 LOC)

**Impact:** Observatory now shows accurate costs for:
- ✅ Eval suite runs (new spans going forward)
- ✅ Coordinator tasks
- ✅ Direct `ailang` CLI usage
- ✅ Any trace with tokens + model name

**Note:** Historical spans already in database keep their original cost values. Only new incoming spans get calculated costs.

## Success Criteria

- [x] **Cost calculation works for all traces** (eval suites show accurate $ amounts) - ✅ M6 IMPLEMENTED
- [ ] OTLP HTTP handler test coverage > 70%
- [ ] API handler test coverage > 60%
- [ ] Hierarchy respects configurable span limits
- [ ] Log-to-span conversion has test coverage
- [ ] UI handles 2000+ span traces without performance issues
- [ ] Overall observatory package coverage > 65% (currently 52.2%)

## Estimated Effort

| Milestone | LOC | Time | Priority | Status |
|-----------|-----|------|----------|--------|
| M6: Cost calculation | 100 | 2h | **High** ⭐ | ✅ DONE |
| M1: OTLP HTTP tests | 150 | 4h | High | Pending |
| M2: Configurable limits | 80 | 2h | Medium | Pending |
| M3: API handler tests | 200 | 4h | Medium | Pending |
| M4: Log conversion tests | 100 | 2h | Low | Pending |
| M5: UI lazy loading | 100 | 3h | Medium | Pending |
| **Total** | **730** | **17h** | | |

## Non-Goals

- **Backfill improvements**: Current backfill logic is correct. Unlinked spans are from non-task contexts and should remain unlinked.
- **Real-time streaming**: WebSocket streaming is working; no changes needed.
- **Schema changes**: Database schema is stable; no migrations needed.

## Related Documents

- [M-TASK-HIERARCHY Design](./m-task-hierarchy-linking.md) - Original implementation
- [M-TASK-HIERARCHY Sprint Plan](./m-task-hierarchy-sprint-plan.md) - Sprint execution plan
- [M-OTEL-DASHBOARD Foundation](./m-otel-dashboard-foundation.md) - UI foundation

## Implementation Notes

### Task ID Extraction (Current Implementation)

The OTLP receiver uses a 4-source priority chain for task ID extraction:

```go
// internal/observatory/otlp_receiver.go (lines 285-300, 512-527)

// Priority order:
// 1. ailang.task_id from resource attributes (OTEL_RESOURCE_ATTRIBUTES)
taskID := extractString(resourceAttrs, "ailang.task_id")

// 2. task.id span attribute (coordinator sets this on task execution spans)
if taskID == "" {
    taskID = extractString(attrs, "task.id")
}

// 3. task.workspace span attribute (executor sets worktree path)
if taskID == "" {
    if workspace := extractString(attrs, "task.workspace"); workspace != "" {
        taskID = extractTaskIDFromPath(workspace)
    }
}

// 4. process.cwd worktree path fallback (Claude Code subprocesses)
if taskID == "" {
    taskID = extractTaskIDFromCwd(resourceAttrs)
}
```

### Token Extraction (Current Implementation)

The OTLP receiver extracts tokens from multiple naming conventions:

```go
// internal/observatory/otlp_receiver.go (lines 550-590)

// Naming conventions checked:
// - gen_ai.usage.input_tokens / gen_ai.usage.output_tokens (OpenTelemetry GenAI SemConv)
// - ailang.tokens.in / ailang.tokens.out (AILANG native)
// - ai.token_count.input / ai.token_count.output (AI provider convention)
// - task.tokens_in / task.tokens_out (Coordinator task spans)
```

This covers all common AI telemetry patterns used by Claude Code, Gemini CLI, and AILANG tooling.
