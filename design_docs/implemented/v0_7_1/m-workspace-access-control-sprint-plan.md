# Sprint Plan: M-WORKSPACE-ACCESS

## Summary
Implement workspace-level access control for the AILANG Collaboration Hub, enabling secure multi-tenant deployment with public/private workspace visibility and per-workspace roles.

**Duration:** 3 days (conservative estimate with testing)
**Dependencies:** M-AUTH Firebase Authentication (v0.7.0) ✅ Complete
**Risk Level:** Medium (touches auth middleware + multiple API endpoints)

## Current Status Analysis

### Completed Recently
- ✅ M-AUTH Firebase Authentication: ~400 LOC in 2 days
- ✅ Trace Testing Framework: ~400 LOC in 2 days
- ✅ Observatory file refactor: ~300 LOC split

### Velocity
- Recent average: **150-200 LOC/day** (including tests)
- Estimated capacity: **450-600 LOC** for this sprint (3 days)
- Actual estimate: **~1150 LOC** → May need 4-5 days if velocity holds

### Existing Infrastructure
- ✅ `parseControlPlaneFilter()` already extracts workspace from query params
- ✅ All observatory queries support workspace filtering via `filter.Workspace`
- ✅ Firebase Auth context already available in handlers
- ✅ AggregationNav already has workspace selector UI

### Remaining from Design Doc
- ⏳ Phase 1: Firestore Schema & Backend (~700 LOC)
- ⏳ Phase 2: Frontend Implementation (~100 LOC)
- ⏳ Phase 3: CLI & Documentation (~350 LOC)

---

## Proposed Milestones

### Milestone 1: Workspace Service & Firestore Schema
**Goal:** Create workspace access control service with Firestore backend
**Estimated:** 200 LOC implementation + 300 LOC tests = **500 LOC**
**Duration:** 1.5 days

**Tasks:**
- Day 1 AM: Create Firestore schema (`workspaces/{id}`, `workspace_access/{id}/users/{email}`)
- Day 1 AM: Implement `internal/server/auth/workspace.go`:
  - `ListAccessibleWorkspaces(ctx, email)`
  - `HasWorkspaceAccess(ctx, email, workspaceID)`
  - `GetWorkspaceByPath(path)` (path mapping)
- Day 1 PM: Add workspace mappings config to `~/.ailang/config.yaml`
- Day 1 PM: Write unit tests for all workspace service methods
- Day 2 AM: Add caching layer (5 min TTL for access checks)

**Files to Create/Modify:**
```
NEW: internal/server/auth/workspace.go        (~200 LOC)
NEW: internal/server/auth/workspace_test.go   (~300 LOC)
MOD: internal/coordinator/agent_config.go     (+30 LOC - workspace mappings)
```

**Acceptance Criteria:**
- [ ] `ListAccessibleWorkspaces` returns public + granted workspaces
- [ ] `HasWorkspaceAccess` correctly checks Firestore permissions
- [ ] Path mapping resolves `/Users/mark/dev/sunholo/ailang` → `sunholo-data/ailang`
- [ ] Cache reduces Firestore calls by >90% on repeated checks
- [ ] Unit tests: >90% coverage on workspace.go
- [ ] `make test` passes

**Risks:**
- Firestore latency on cold starts → Mitigation: Add caching layer
- Path mapping edge cases → Mitigation: Comprehensive test cases + safe default to "public"

---

### Milestone 2: API Middleware & Endpoint Updates
**Goal:** Add WorkspaceMiddleware and update all data endpoints
**Estimated:** 150 LOC implementation + 100 LOC tests = **250 LOC**
**Duration:** 1 day

**Tasks:**
- Day 2 PM: Create `WorkspaceMiddleware` in `handlers_auth.go`:
  - Extract workspace from query/header
  - Check access via workspace service
  - Return 403 if unauthorized
  - Set workspace in request context
- Day 2 PM: Update `/api/workspaces` to filter by user access
- Day 3 AM: Apply middleware to all `/api/controlplane/*` endpoints
- Day 3 AM: Update query handlers to default to accessible workspaces when none specified
- Day 3 AM: Add workspace filtering to `/api/observatory/*` endpoints

**Files to Create/Modify:**
```
MOD: internal/server/handlers_auth.go         (+80 LOC - middleware)
MOD: internal/server/handlers_threads.go      (+40 LOC - filter workspaces)
MOD: internal/server/handlers_controlplane.go (+30 LOC - default filtering)
NEW: internal/server/handlers_workspace_test.go (~100 LOC)
```

**Acceptance Criteria:**
- [ ] Unauthorized workspace access returns 403 Forbidden
- [ ] `/api/workspaces` only returns accessible workspaces
- [ ] Unauthenticated users only see public workspace data
- [ ] No workspace specified → returns union of accessible workspace data
- [ ] All API tests pass with workspace restrictions
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Breaking existing API consumers → Mitigation: Default behavior unchanged for authenticated users
- Performance overhead → Mitigation: Cache workspace access checks

---

### Milestone 3: Frontend Integration
**Goal:** Update AggregationNav to filter workspaces by access
**Estimated:** 50 LOC implementation + 20 LOC tests = **70 LOC**
**Duration:** 0.5 days

**Tasks:**
- Day 3 PM: Create `useWorkspaceAccess` hook in `ui/src/hooks/`
- Day 3 PM: Update `AggregationNav.tsx` to filter workspace breakdown by accessible workspaces
- Day 3 PM: Add role badge next to workspace names in selector
- Day 3 PM: Handle 403 errors gracefully in API client

**Files to Create/Modify:**
```
NEW: ui/src/hooks/useWorkspaceAccess.ts       (~50 LOC)
MOD: ui/src/features/controlplane/components/AggregationNav.tsx (+20 LOC)
```

**Acceptance Criteria:**
- [ ] Only accessible workspaces appear in AggregationNav
- [ ] Role badge shows (Approver/Viewer) next to workspace name
- [ ] 403 errors show user-friendly toast notification
- [ ] Unauthenticated users only see public workspaces
- [ ] `npm run build` succeeds
- [ ] Manual testing in browser confirms filtering works

**Risks:**
- React state sync issues → Mitigation: Re-fetch on auth state change

---

### Milestone 4: CLI Commands & Documentation
**Goal:** Add `ailang workspaces` CLI commands and documentation
**Estimated:** 250 LOC implementation + 80 LOC tests = **330 LOC**
**Duration:** 0.5 days

**Tasks:**
- Day 3 PM: Create `cmd/ailang/workspaces.go` with subcommands:
  - `list` - Show accessible workspaces
  - `add` - Create workspace in Firestore
  - `grant` - Grant user access
  - `revoke` - Remove user access
  - `set-public` - Toggle public visibility
- Day 3 PM: Update `cmd/ailang/main.go` to register workspace commands
- Day 3 PM: Create `docs/guides/workspaces.md`
- Day 3 PM: Update CLAUDE.md with workspace commands

**Files to Create/Modify:**
```
NEW: cmd/ailang/workspaces.go                 (~250 LOC)
MOD: cmd/ailang/main.go                       (+10 LOC - register commands)
NEW: docs/docs/guides/workspaces.md           (~150 LOC)
MOD: CLAUDE.md                                (+30 LOC - workspace commands)
```

**Acceptance Criteria:**
- [ ] `ailang workspaces list` shows accessible workspaces
- [ ] `ailang workspaces add sunholo-data/ailang --public true` works
- [ ] `ailang workspaces grant sunholo-data/ailang --email x@y.com --role Approver` works
- [ ] Documentation explains workspace concepts and setup
- [ ] `make build` succeeds
- [ ] `make lint` passes

**Risks:**
- Firestore permissions for CLI → Mitigation: Use ADC (Application Default Credentials)

---

## Success Metrics
- Test coverage: >85% on new code
- All existing tests passing: ✅
- All linting passing: ✅
- Documentation: `docs/guides/workspaces.md` created
- Manual verification: Workspace filtering works end-to-end
- Security: 403 returned for unauthorized workspace access

## Dependencies
- ✅ Firebase Authentication (M-AUTH v0.7.0)
- ✅ Firebase Admin SDK initialized
- ✅ Firestore database accessible via ADC
- ✅ AggregationNav workspace selector exists

## Open Questions
- None - design doc is comprehensive

## Day-by-Day Schedule

| Day | Morning | Afternoon |
|-----|---------|-----------|
| **Day 1** | M1: Firestore schema + workspace service | M1: Config + tests |
| **Day 2** | M1: Caching + complete tests | M2: Middleware + API updates |
| **Day 3** | M2: Default filtering + tests | M3: Frontend + M4: CLI + docs |

## Total LOC Estimate

| Component | Implementation | Tests | Total |
|-----------|---------------|-------|-------|
| Workspace Service | 200 | 300 | 500 |
| API Middleware | 150 | 100 | 250 |
| Frontend | 70 | 0 | 70 |
| CLI + Docs | 280 | 50 | 330 |
| **Total** | **700** | **450** | **1150** |

## Notes
- Leverages existing workspace filtering infrastructure (no data migration needed)
- Path-to-workspace mapping happens at query time
- Default unmapped paths to "public" (safe default)
- Design doc includes comprehensive security analysis
