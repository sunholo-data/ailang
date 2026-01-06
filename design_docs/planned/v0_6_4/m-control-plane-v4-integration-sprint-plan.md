# Sprint Plan: M-CONTROL-PLANE-V4-INTEGRATION

## Summary
Connect the Control Plane v4 UI mockup to real data from existing AILANG APIs (Observatory, Coordinator, Statistics) to enable production monitoring of AI agent activity.

**Duration:** 2-3 days (~13 hours)
**Dependencies:** Observatory API (M1-M4 complete), Coordinator running
**Risk Level:** Low (existing APIs, straightforward integration)

## Current Status Analysis

### Completed Recently
- Observatory API foundation (M1-M4): ~1,500 LOC
- Control Plane v4 UI mockup: ~1,800 LOC (ControlPlane.tsx + CSS)
- GCP Trace backend: ~400 LOC
- Task hierarchy API: ~200 LOC

### Velocity
- Recent average: ~150-200 LOC/day (mixed frontend/backend)
- Estimated capacity: ~500 LOC for this sprint
- Note: Recent work includes significant UI development

### Remaining from Design Doc
- ⏳ Backend API additions: ~150 LOC
- ⏳ Frontend hooks: ~200 LOC
- ⏳ Component integration: ~150 LOC
- 📋 Testing & polish: ~50 LOC

**Total estimated:** ~550 LOC

## Proposed Milestones

### Milestone 1: Backend Control Plane APIs
**Goal:** Create new server endpoints for Control Plane-specific data aggregations
**Estimated:** 120 LOC implementation + 30 LOC tests = 150 LOC
**Duration:** 0.5 days (4 hours)

**Tasks:**
1. Create `internal/server/handlers_controlplane.go`:
   - `GET /api/controlplane/heatmap` - Daily task aggregations
   - `GET /api/controlplane/topology` - Agent graph with status
   - `GET /api/controlplane/aggregations` - Filter options

2. Enhance `internal/server/handlers_statistics.go`:
   - Add `active_agents` count
   - Add `pending_approvals` count
   - Add `success_rate` calculation

3. Register routes in `internal/server/server.go`

**Files to create/modify:**
- NEW: `internal/server/handlers_controlplane.go` (~100 LOC)
- MODIFY: `internal/server/handlers_statistics.go` (+30 LOC)
- MODIFY: `internal/server/server.go` (+20 LOC)

**Acceptance Criteria:**
- [ ] `curl /api/controlplane/heatmap?days=90` returns daily task data
- [ ] `curl /api/controlplane/topology` returns agent graph with edges
- [ ] `curl /api/statistics` includes `active_agents`, `pending_approvals`, `success_rate`
- [ ] All new endpoints return valid JSON
- [ ] Server starts without errors

**Risks:**
- Coordinator DB schema might not have all needed fields - Mitigation: Add computed fields or defaults

---

### Milestone 2: Frontend Data Hooks
**Goal:** Create React hooks to fetch data from APIs and manage state
**Estimated:** 150 LOC implementation + 50 LOC types = 200 LOC
**Duration:** 0.5 days (4 hours)

**Tasks:**
1. Create `ui/src/features/controlplane/hooks/` directory
2. Implement data hooks:
   - `useHeatmapData.ts` - Fetch and transform heatmap data
   - `useTopologyData.ts` - Fetch topology, poll for updates
   - `useTraceData.ts` - Fetch Observatory spans
   - `useEventQueue.ts` - WebSocket event subscription
   - `useControlPlaneStats.ts` - Statistics fetching

3. Add shared types in `ui/src/features/controlplane/types.ts`

**Files to create:**
- NEW: `ui/src/features/controlplane/hooks/useHeatmapData.ts` (~40 LOC)
- NEW: `ui/src/features/controlplane/hooks/useTopologyData.ts` (~50 LOC)
- NEW: `ui/src/features/controlplane/hooks/useTraceData.ts` (~30 LOC)
- NEW: `ui/src/features/controlplane/hooks/useEventQueue.ts` (~40 LOC)
- NEW: `ui/src/features/controlplane/hooks/useControlPlaneStats.ts` (~25 LOC)
- NEW: `ui/src/features/controlplane/hooks/index.ts` (~10 LOC)
- NEW: `ui/src/features/controlplane/types.ts` (~30 LOC)

**Acceptance Criteria:**
- [ ] `useHeatmapData` returns typed HeatmapCell[]
- [ ] `useTopologyData` returns agents and edges, polls every 5s
- [ ] `useTraceData` fetches spans from Observatory API
- [ ] `useEventQueue` connects to WebSocket and filters events
- [ ] All hooks handle loading/error states
- [ ] TypeScript compiles without errors

**Risks:**
- WebSocket event format might differ - Mitigation: Add adapter layer

---

### Milestone 3: Component Integration
**Goal:** Replace mock data with real API calls in ControlPlane.tsx
**Estimated:** 120 LOC implementation + 30 LOC cleanup = 150 LOC
**Duration:** 0.5 days (3 hours)

**Tasks:**
1. Import hooks into ControlPlane.tsx
2. Replace mock data arrays:
   - `mockHeatmapData` → `useHeatmapData()`
   - `mockAgents` + `topologyEdges` → `useTopologyData()`
   - `mockSpans` → `useTraceData()`
   - `mockEvents` → `useEventQueue()`
   - Header stats → `useControlPlaneStats()`

3. Add loading skeletons for each component
4. Add error boundaries
5. Remove mock data arrays
6. Update footer version from mock

**Files to modify:**
- MODIFY: `ui/src/features/controlplane/ControlPlane.tsx` (~150 LOC changed)

**Acceptance Criteria:**
- [ ] Activity Heatmap shows real task data (or empty state if no data)
- [ ] Agent Topology shows agents from config with live status
- [ ] Trace Waterfall shows real Observatory spans
- [ ] Message Queue shows WebSocket events
- [ ] Statistics cards match `ailang coordinator status` output
- [ ] Loading states display correctly
- [ ] Error states display correctly
- [ ] No hardcoded mock data remains

**Risks:**
- API data format mismatch - Mitigation: Use adapter functions in hooks

---

### Milestone 4: Testing & Polish
**Goal:** Verify integration works end-to-end, fix edge cases
**Estimated:** 50 LOC fixes
**Duration:** 0.25 days (2 hours)

**Tasks:**
1. Start coordinator and run some tasks
2. Verify heatmap populates with real data
3. Verify topology reflects config agents
4. Test WebSocket event flow
5. Check edge cases (no data, errors, slow network)
6. Performance review (memoization, debounce)
7. Build production bundle

**Acceptance Criteria:**
- [ ] `npm run build` succeeds
- [ ] Dashboard works with real coordinator data
- [ ] Dashboard works with empty database (graceful empty state)
- [ ] No console errors in browser
- [ ] Reasonable performance (no layout thrashing)

**Risks:**
- Performance issues with large datasets - Mitigation: Add pagination/limits

---

## Success Metrics
- All 4 milestones complete: ✅
- Test coverage: Existing coverage maintained
- New endpoints working: 3 new + 1 enhanced
- Frontend hooks: 5 new hooks
- Mock data removed: All hardcoded arrays replaced
- Build succeeds: `npm run build` ✅
- Manual testing: Dashboard works with real data

## Dependencies
- Observatory API (M1-M4) - COMPLETE
- Coordinator daemon - Available
- Collaboration Hub server - Available
- React Flow - Already installed

## Open Questions
1. **Trust Scores:** Keep mock for v0.6.4? (Recommended: Yes, defer to v0.7.0)
2. **Polling interval:** 5 seconds for topology? (Recommended: Yes)
3. **Historical range:** Default 90 days for heatmap? (Recommended: Yes, use existing range selector)

## Notes
- This sprint is primarily integration work (glue code)
- Most complexity is already implemented in Observatory/Coordinator
- Focus is on data transformation and UI binding
- Trust configuration remains mocked (no backend support yet)
- The Aggregation Navigator in sidebar can be implemented later (filter chips work)
