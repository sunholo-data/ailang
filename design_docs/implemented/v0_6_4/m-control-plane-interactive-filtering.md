# M-CONTROL-PLANE-INTERACTIVE-FILTERING

**Status**: Implemented
**Target**: v0.6.4
**Priority**: P1 - Medium
**Estimated**: 4-6 hours
**Dependencies**: M-CONTROL-PLANE-V4-INTEGRATION (partially complete)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to determinism - UI filtering only |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No new effects introduced |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Structured filter parameters enable programmatic access |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Improves cost drill-down visibility by dimension |
| A10: Composability | +1 | Filter parameters compose across all endpoints |
| A11: Structured Failure | 0 | No failure handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Enables better machine analysis via structured filters

## Problem Statement

The Control Plane dashboard displays aggregated telemetry data with a hierarchical breakdown in the left sidebar (By Source, By Provider, By Model, By Workspace). However, **selecting a breakdown item does not filter the rest of the dashboard**.

**Current State:**
- Left sidebar shows breakdown with costs/spans per dimension ✅
- Header stats show global totals (not filtered)
- Heatmap shows all activity (not filtered)
- Trace Waterfall shows recent spans (not filtered)
- Event Queue shows coordinator events (only coordinator source)

**Impact:**
- Users cannot drill down into specific sources (e.g., "show me only Eval costs")
- Dashboard feels disconnected - selection doesn't do anything useful
- Hard to investigate costs by dimension

## Goals

**Primary Goal:** When user selects a breakdown item (e.g., "Eval" source), ALL dashboard components filter to show only that data.

**Success Metrics:**
- Selecting "Eval" source shows only eval-related stats, heatmap, traces
- Selecting a specific model shows cost/tokens for that model only
- Filters compose: Source + Model selection narrows to intersection
- URL reflects filter state for shareability

## Solution Design

### Overview

Add filter query parameters to all Control Plane API endpoints. Connect the UI selection state to these filters. All components re-fetch with filter params when selection changes.

### Architecture

**Filter Flow:**
```
User clicks "Eval" in sidebar
    ↓
selectedLevel state updates to { source_type: "eval" }
    ↓
All hooks re-fetch with ?source_type=eval
    ↓
Backend queries filter by source type
    ↓
UI displays filtered data
```

**Components:**

1. **API Filter Parameters**: Add `source_type`, `provider`, `model`, `workspace` query params to:
   - `/api/controlplane/stats` - Header totals
   - `/api/controlplane/stats/breakdown` - Sidebar breakdown
   - `/api/controlplane/heatmap` - Calendar heatmap

2. **Backend Query Filtering**: Extend existing SQL queries with WHERE clauses based on filter params

3. **Hook Updates**: All hooks accept optional filter object, include in fetch URL

4. **Selection State**: Lift `selectedLevel` to parent, connect to all hooks

### Implementation Plan

**Phase 1: Backend API Filters** (~2 hours)
- [ ] Add filter params to `/api/controlplane/stats` endpoint
- [ ] Add filter params to `/api/controlplane/stats/breakdown` endpoint
- [ ] Add filter params to `/api/controlplane/heatmap` endpoint
- [ ] Add source type inference helper (reuse existing CASE logic)

**Phase 2: Frontend Hook Updates** (~1.5 hours)
- [ ] Update `useControlPlaneStats` to accept filter params
- [ ] Update `useBreakdownData` to accept filter params
- [ ] Update `useHeatmapData` to accept filter params
- [ ] Create shared filter type definition

**Phase 3: Connect Selection to Filters** (~1.5 hours)
- [ ] Lift selectedLevel state to ControlPlane component
- [ ] Pass filter to all hooks based on selection
- [ ] Update URL with filter state (optional, for sharing)
- [ ] Add "Clear filters" button in header

**Phase 4: UI Improvements** (~1 hour)
- [ ] Replace Event Queue with Task List (spans grouped by task_id)
- [ ] Update Trace Waterfall to show trace list with drill-down
- [ ] Add visual indicator for active filter

### Files to Modify/Create

**Modified files:**
- `internal/server/handlers_controlplane.go` - Add filter parsing to all handlers (~50 LOC)
- `internal/observatory/backend.go` - Add filter params to query methods (~100 LOC)
- `ui/src/features/controlplane/hooks/useControlPlaneStats.ts` - Accept filters (~20 LOC)
- `ui/src/features/controlplane/hooks/useBreakdownData.ts` - Accept filters (~20 LOC)
- `ui/src/features/controlplane/hooks/useHeatmapData.ts` - Accept filters (~20 LOC)
- `ui/src/features/controlplane/ControlPlane.tsx` - Connect selection to filters (~40 LOC)

**New files:**
- `ui/src/features/controlplane/types/filters.ts` - Shared filter type (~20 LOC)

## Examples

### Example 1: Filter by Source Type

**API Request:**
```
GET /api/controlplane/stats?source_type=eval
```

**Response (filtered to eval only):**
```json
{
  "total_cost": 245.67,
  "total_tokens": 1234567,
  "total_spans": 892,
  "period_start": "2026-01-01T00:00:00Z",
  "period_end": "2026-01-06T23:59:59Z"
}
```

### Example 2: Multiple Filters

**API Request:**
```
GET /api/controlplane/stats?source_type=eval&model=claude-sonnet-4-5
```

**Response (eval + specific model):**
```json
{
  "total_cost": 89.23,
  "total_tokens": 456789,
  "total_spans": 234
}
```

### Example 3: UI Selection State

```typescript
// User clicks "Eval" in sidebar
const [filters, setFilters] = useState<ControlPlaneFilters>({});

// Selection handler
const handleSelect = (level: string, value: string) => {
  setFilters(prev => ({ ...prev, [level]: value }));
};

// All hooks receive filters
const { stats } = useControlPlaneStats({ filters });
const { breakdowns } = useBreakdownData({ filters });
const { heatmap } = useHeatmapData({ filters });
```

## Success Criteria

- [ ] Selecting source type filters all dashboard components
- [ ] Selecting provider filters all dashboard components
- [ ] Selecting model filters all dashboard components
- [ ] Multiple filters compose (intersection)
- [ ] "Clear filters" returns to global view
- [ ] All tests passing
- [ ] No performance regression (queries still fast)

## Testing Strategy

**Unit tests:**
- Filter parsing in handlers
- SQL query generation with filters
- Hook re-fetch on filter change

**Integration tests:**
- API returns filtered data correctly
- Filter combinations work

**Manual testing:**
- Click through all breakdown items
- Verify header stats update
- Verify heatmap updates
- Verify trace list updates

## Non-Goals

**Not in this feature:**
- Time range filtering - Separate feature, different UX
- Saved filter presets - Nice to have, defer
- Export filtered data - Different feature

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Performance with filters | Medium | Add indexes on source_type, model columns |
| Complex filter combinations | Low | Start with single-dimension filters, add multi later |
| URL state management | Low | Use simple query params, not complex state |

## Related Documents

**Implemented (informs design):**
- [m-control-plane-v4-integration.md](m-control-plane-v4-integration.md) - Parent feature
- [semantic-caching-complete.md](../../../implemented/v0_6_0/semantic-caching-complete.md) - Similar query patterns

**Planned (check for overlap):**
- [m-control-plane-v4.md](m-control-plane-v4.md) - Original Control Plane design

## References

- [Design Axioms](/docs/references/axioms)
- Observatory Backend: `internal/observatory/backend.go`
- Control Plane Handlers: `internal/server/handlers_controlplane.go`

## Future Work

- Time range filtering (date picker)
- Cost alerting when filtered total exceeds threshold
- Comparison view (filter A vs filter B)

---

## Implementation Report

**Completed**: 2026-01-06

### What Was Built

**Phase 1: Dimension Filtering (sidebar → dashboard)**

Backend:
- `ControlPlaneFilter` struct with `SourceType`, `Provider`, `Model`, `Workspace` fields
- `buildSourceTypeCondition()` - SQL WHERE clause builder for source type inference
- `GetFilteredMetricsSummary()` - Filtered observatory metrics
- `GetFilteredBreakdownByProvider/SourceType/Model()` - Filtered breakdown queries
- `parseControlPlaneFilter()` - HTTP query param parser

Frontend:
- `ControlPlaneFilters` TypeScript interface
- `buildFilterQueryString()` - Query string builder
- `parseSelectedLevelToFilters()` - Selection state to filter conversion
- Updated all hooks (`useControlPlaneStats`, `useBreakdownData`, `useHeatmapData`) to accept filters
- Filter badge in GlobalStats header with clear button

**Phase 2: Bidirectional Time Filtering (heatmap ↔ dashboard)**

Backend:
- Extended `ControlPlaneFilter` with `StartDate`, `EndDate` fields (YYYY-MM-DD format)
- `buildFilterConditions()` - Shared WHERE clause builder for all filter types
- `GetFilteredHeatmapData()` - Observatory-based heatmap with filter support
- Updated heatmap handler to use observatory data instead of coordinator
- All breakdown methods now support time range filtering

Frontend:
- Extended `ControlPlaneFilters` type with `start_date`, `end_date`
- `hasTimeRangeFilter()`, `mergeFilters()`, `clearTimeRangeFilter()` helper functions
- `createDateFilter()`, `createDateRangeFilter()` for heatmap click handling
- Merged time selection with dimension filters in `ControlPlane.tsx`
- Bidirectional state: heatmap selection updates all dashboard components

### Files Modified

**Backend (~200 LOC):**
- `internal/observatory/backend.go` - Filter type, helper functions, heatmap query, filtered methods
- `internal/server/handlers_controlplane.go` - Filter parsing, observatory-based heatmap handler

**Frontend (~150 LOC):**
- `ui/src/features/controlplane/types/filters.ts` - Extended filter type and helpers
- `ui/src/features/controlplane/hooks/useControlPlaneStats.ts` - Accept filters
- `ui/src/features/controlplane/hooks/useBreakdownData.ts` - Accept filters
- `ui/src/features/controlplane/hooks/useHeatmapData.ts` - Accept filters
- `ui/src/features/controlplane/ControlPlane.tsx` - Bidirectional filter state

### Verified Working

```bash
# Unfiltered: 31,216 spans, $479.89
curl "http://localhost:1957/api/controlplane/stats"

# Eval filter: 8,896 spans, $470.38
curl "http://localhost:1957/api/controlplane/stats?source_type=eval"

# Time range filter (single day):
# Jan 4: 181 spans, $3.28
curl "http://localhost:1957/api/controlplane/stats?start_date=2026-01-04&end_date=2026-01-04"

# Jan 5: 15,666 spans, $297.28
curl "http://localhost:1957/api/controlplane/stats?start_date=2026-01-05&end_date=2026-01-05"

# Combined filter (eval + Jan 6): 3,479 spans, $173.06
curl "http://localhost:1957/api/controlplane/stats?source_type=eval&start_date=2026-01-06&end_date=2026-01-06"

# Heatmap with filters
curl "http://localhost:1957/api/controlplane/heatmap?source_type=eval"
```

### Filter Flow

```
User clicks "Eval" in sidebar          User clicks date in heatmap
         ↓                                       ↓
selectedLevel = "source-eval"          selectedDateRange = {start, end}
         ↓                                       ↓
         └──────────────────┬──────────────────┘
                            ↓
              filters = mergeFilters(
                parseSelectedLevelToFilters(selectedLevel),
                {start_date, end_date}
              )
                            ↓
         ┌──────────────────┴──────────────────┐
         ↓                  ↓                  ↓
   useControlPlaneStats  useBreakdownData  useHeatmapData
         ↓                  ↓                  ↓
       Header             Sidebar            Heatmap
       (stats)         (breakdowns)        (activity)
```

---

**Document created**: 2026-01-06
**Last updated**: 2026-01-06
