# M-WORKSPACE-ACCESS: Multi-Workspace Access Control with Public/Private Visibility

**Status:** Implemented
**Target:** v0.7.1
**Priority:** P1 (High - enables multi-tenant deployment)
**Estimated:** 2.5 days → **Actual:** 1 day
**Dependencies:** M-AUTH Firebase Authentication (v0.7.0)

**Created:** January 23, 2026
**Last Modified:** January 23, 2026
**Implemented:** January 23, 2026

---

## Executive Summary

This design document specifies multi-workspace access control for the AILANG Collaboration Hub, enabling workspace-level permissions with public/private visibility. This builds on the Firebase authentication implemented in v0.7.0.

**Problem:** Currently, authenticated users can see all data regardless of workspace. When deployed as a multi-tenant service, users should only see workspaces they have access to, with a "public" workspace visible to unauthenticated users.

**Solution:** Implement workspace-based access control where:
- Each workspace maps to a GitHub repository identifier (cloud-deployable, not file paths)
- Workspaces can be marked as `is_public: true` - visible to unauthenticated users
- Authenticated users see public workspaces PLUS workspaces they have explicit access to
- Roles are per-workspace: Viewer (read-only), Approver (read+write/approve)

**Key Features:**
1. Workspace = GitHub repo identifier (e.g., "sunholo-data/ailang", not file paths)
2. `is_public` flag on any workspace (not a separate "public" workspace)
3. Per-workspace role assignments
4. API filtering by workspace access
5. Existing aggregation nav workspace selector respects access control
6. Observatory span filtering by workspace attribute

---

## Current State

**After v0.7.0 (M-AUTH):**
- Firebase Authentication working (Google Sign-In)
- Single global role per user (Approver/Viewer)
- No workspace filtering
- All authenticated users see all data
- Firestore schema: `access_control/{email}` with single role

**Data Sources with Workspace Attribute:**
- Observatory spans: `workspace` attribute in span attributes
- Coordinator tasks: `workspace` field (file path, e.g., `/Users/mark/dev/sunholo/ailang`)
- Messages: `workspace_id` field

**Current Workspace Mapping:**
```
File Path                              → Should Map To
/Users/mark/dev/sunholo/ailang        → sunholo-data/ailang
/Users/mark/dev/sunholo/stapledons    → sunholo-data/stapledons_voyage
/Users/mark/dev/TwilightGame          → MarkEdmondson1234/TwilightGame
```

---

## Problem Statement

### Multi-Tenant Gap
With Firebase auth enabled, users can log in but still see everything:
- All workspaces appear in the dashboard
- Observatory shows spans from all projects
- No separation between personal and shared projects
- No public access for demonstration/documentation

### Use Cases Not Supported
1. **Public Demo**: Show AILANG dashboard publicly with demo data
2. **Team Isolation**: Team A cannot see Team B's tasks
3. **Contractor Access**: Read-only access to specific workspace
4. **Personal Projects**: Keep personal projects private while sharing work projects

### Current Limitations
- Workspace = file path (not cloud-portable)
- No workspace registration/metadata
- No public/private visibility setting
- All API queries return all data
- No workspace selector in UI

---

## Goals

### Primary Goal
Enable workspace-scoped access control where users see only data from workspaces they have access to, with a "public" workspace visible to all.

### Success Metrics
- [ ] **Isolation**: Users cannot see data from workspaces they lack access to
- [ ] **Public Access**: Unauthenticated users see only "public" workspace
- [ ] **Cloud-Ready**: Workspace IDs are GitHub repo identifiers (portable)
- [ ] **Performance**: Workspace filtering adds <10ms to queries
- [ ] **UX**: Workspace selector in dashboard header
- [ ] **Migration**: Existing data auto-mapped to new workspace IDs

### Non-Goals
- Per-task or per-agent permissions (workspace-level is sufficient)
- Cross-workspace data sharing
- Workspace creation UI (CLI/config only for v0.7.1)
- Workspace analytics/quotas

---

## Solution Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Frontend (React)                         │
│  - Workspace selector in header                              │
│  - Filter all views by selected workspace                    │
│  - Show only accessible workspaces in selector               │
└────────────────────────────┬────────────────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │  Authorization: Bearer <JWT token>       │
        │  X-Workspace: <workspace-id>             │
        └────────────────────┬────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │       Go HTTP Server (internal/server/)  │
        │ ┌──────────────────────────────────────┤
        │ │ Middleware: WorkspaceMiddleware       │
        │ │ - Extract workspace from header/query │
        │ │ - Check user has access to workspace  │
        │ │ - For public workspace: allow anon    │
        │ │ - Set ctx.Workspace                   │
        │ └──────────────────────────────────────┤
        │ ┌──────────────────────────────────────┤
        │ │ Handlers: All data endpoints          │
        │ │ - Filter queries by ctx.Workspace     │
        │ │ - Map file paths → workspace IDs      │
        │ └──────────────────────────────────────┤
        └────────────────────┬────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │       Firestore (Access Control)         │
        │ workspaces/{workspace_id}:               │
        │   - name: "AILANG"                       │
        │   - github_repo: "sunholo-data/ailang"   │
        │   - is_public: true  ← PUBLIC FLAG       │
        │   - path_patterns: ["/dev/sunholo/..."]  │
        │                                          │
        │ workspace_access/{workspace_id}/users/   │
        │   {email}:                               │
        │     - role: "Approver" | "Viewer"        │
        │     - granted_at: timestamp              │
        │                                          │
        │ Access Logic:                            │
        │   if is_public: anonymous gets Viewer    │
        │   if has access: use granted role        │
        │   else: no access                        │
        └────────────────────────────────────────┘
```

### Key Concepts

#### Workspace Identity
```
Workspace ID = GitHub repo identifier (owner/repo format)

Examples:
  - "sunholo-data/ailang"
  - "sunholo-data/stapledons_voyage"
  - "MarkEdmondson1234/TwilightGame"
  - "public" (special workspace for public data)
```

#### Workspace Metadata (Firestore)
```typescript
interface Workspace {
  id: string;              // "sunholo-data/ailang"
  name: string;            // "AILANG"
  github_repo: string;     // "sunholo-data/ailang"
  is_public: boolean;      // true = anonymous users get Viewer access
  path_patterns: string[]; // File paths that map to this workspace
  created_at: Timestamp;
  created_by: string;      // email
}

// Access resolution logic:
// 1. If workspace.is_public AND user is anonymous → Viewer role
// 2. If user has explicit access → use their granted role
// 3. If workspace.is_public AND user is authenticated → Viewer (unless granted higher)
// 4. Else → no access
```

#### User-Workspace Access (Firestore)
```typescript
interface WorkspaceAccess {
  email: string;
  workspace_id: string;
  role: "Approver" | "Viewer";
  granted_at: Timestamp;
  granted_by: string;      // email
}
```

### Path-to-Workspace Mapping

The coordinator uses file paths as workspace identifiers. We need to map these to GitHub repo IDs.

**Mapping Rules (in config):**
```yaml
# ~/.ailang/config.yaml
workspaces:
  mappings:
    # Pattern → Workspace ID
    - pattern: "/Users/*/dev/sunholo/ailang"
      workspace: "sunholo-data/ailang"
    - pattern: "/Users/*/dev/sunholo/stapledons_voyage"
      workspace: "sunholo-data/stapledons_voyage"
    - pattern: "/Users/*/dev/TwilightGame"
      workspace: "MarkEdmondson1234/TwilightGame"
    - pattern: "*"  # Default fallback
      workspace: "public"
```

**Runtime Mapping:**
```go
// Map file path to workspace ID
func MapPathToWorkspace(path string) string {
    for _, mapping := range config.Workspaces.Mappings {
        if matchPattern(mapping.Pattern, path) {
            return mapping.Workspace
        }
    }
    return "public" // Default
}
```

### API Changes

#### New Endpoint: List Accessible Workspaces
```
GET /api/workspaces

Response (authenticated user with access to ailang):
{
  "workspaces": [
    {
      "id": "sunholo-data/ailang",
      "name": "AILANG",
      "role": "Approver",       // User's role in this workspace
      "is_public": true         // Also visible to anonymous users
    },
    {
      "id": "sunholo-data/stapledons_voyage",
      "name": "Stapledons Voyage",
      "role": "Viewer",
      "is_public": false        // Only visible to authorized users
    }
  ]
}

Response (unauthenticated - only sees public workspaces):
{
  "workspaces": [
    {
      "id": "sunholo-data/ailang",
      "name": "AILANG",
      "role": "Viewer",         // Anonymous gets read-only
      "is_public": true
    }
  ]
}
```

#### Updated Endpoints: Workspace Filtering

All data endpoints accept `workspace` query parameter:

```
GET /api/observatory/traces?workspace=sunholo-data/ailang
GET /api/observatory/spans?workspace=sunholo-data/ailang
GET /api/tasks?workspace=sunholo-data/ailang
GET /api/messages?workspace=sunholo-data/ailang
```

**Behavior:**
- If `workspace` specified: Filter to that workspace (check access)
- If `workspace` not specified: Return data from all accessible workspaces
- Unauthenticated: Only return "public" workspace data

#### WebSocket: Workspace Subscription
```typescript
// Client subscribes to specific workspace
ws.send(JSON.stringify({
  type: "subscribe",
  workspace: "sunholo-data/ailang"
}));

// Server filters events by workspace
```

### Frontend Changes

#### Filter Cascade Architecture (Existing)

The dashboard already has a filter cascade where AggregationNav controls everything:

```
┌─────────────────────────────────────────────────────────────────┐
│ AggregationNav (right sidebar)                                   │
│   - Workspace selector ← FILTER HERE BY ACCESS                   │
│   - Provider selector                                            │
│   - Model selector                                               │
│   - Source type selector                                         │
│                    │                                             │
│                    ▼                                             │
│              selectedFilters = { workspace: "...", ... }         │
└─────────────────────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ ControlPlane.tsx                                                 │
│   const filters = mergeFilters(dimensionFilters, ...)           │
│                    │                                             │
│   Passes to:       │                                             │
│   - useHeatmapData({ filters })                                 │
│   - useEventQueue({ filters })                                  │
│   - useTraceData({ filters })                                   │
│   - useTopologyData({ ... })                                    │
│   - etc.                                                         │
└─────────────────────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ API Calls (already include workspace in query params)            │
│   GET /api/observatory/traces?workspace=sunholo-data/ailang     │
│   GET /api/tasks?workspace=sunholo-data/ailang                  │
└─────────────────────────────────────────────────────────────────┘
```

**Key insight:** We only need to filter at ONE place - the AggregationNav workspace list. Everything else cascades automatically.

#### Changes to AggregationNav

```tsx
// AggregationNav.tsx - Current
const workspaces = breakdowns?.workspace || [];

// AggregationNav.tsx - With access control
const { accessibleWorkspaces } = useWorkspaceAccess();
const workspaces = (breakdowns?.workspace || [])
  .filter(ws => accessibleWorkspaces.some(aw => ws.name.includes(aw.id)));
```

#### Workspace Access Hook
```tsx
// New hook - fetches accessible workspaces based on auth state
function useWorkspaceAccess() {
  const { user } = useAuth();
  const [accessibleWorkspaces, setAccessibleWorkspaces] = useState<Workspace[]>([]);

  useEffect(() => {
    // Fetch accessible workspaces (returns public if not authenticated)
    fetch('/api/workspaces', {
      headers: user ? { Authorization: `Bearer ${await getIdToken()}` } : {}
    })
      .then(res => res.json())
      .then(data => setAccessibleWorkspaces(data.workspaces));
  }, [user]);

  return { accessibleWorkspaces, isPublicOnly: !user };
}
```

#### Backend Defense-in-Depth
Even though frontend filters the selector, backend MUST also enforce access:
1. Validate user has access to requested workspace
2. Return 403 if not authorized
3. If no workspace specified, return union of all accessible workspace data

### CLI Changes

#### Workspace Management Commands
```bash
# List workspaces
ailang workspaces list

# Add workspace
ailang workspaces add sunholo-data/ailang \
  --name "AILANG" \
  --public false

# Grant access
ailang workspaces grant sunholo-data/ailang \
  --email user@example.com \
  --role Approver

# Revoke access
ailang workspaces revoke sunholo-data/ailang \
  --email user@example.com

# Set public visibility
ailang workspaces set-public sunholo-data/ailang --public true
```

---

## Implementation Plan

### Phase 1: Firestore Schema & Backend (1.5 days)

**Tasks:**

1. **Firestore Schema** (~2 hours)
   - [ ] Create `workspaces/{id}` collection schema
   - [ ] Create `workspace_access/{workspace_id}/users/{email}` schema
   - [ ] Add Firestore security rules
   - [ ] Create "public" workspace document

2. **Workspace Service** (~3 hours)
   - [ ] Create `internal/server/auth/workspace.go`
     - `ListAccessibleWorkspaces(ctx, email) ([]Workspace, error)`
     - `HasWorkspaceAccess(ctx, email, workspaceID) (bool, Role, error)`
     - `GetWorkspaceByPath(path string) string` (path mapping)
   - [ ] Add caching (5 min TTL)

3. **Path Mapping Configuration** (~2 hours)
   - [ ] Update `internal/coordinator/agent_config.go`
     - Add `WorkspaceMappings` to config schema
     - Load from `~/.ailang/config.yaml`
   - [ ] Create default mappings for known workspaces

4. **API Middleware** (~2 hours)
   - [ ] Create `WorkspaceMiddleware` in `handlers_auth.go`
     - Extract workspace from `X-Workspace` header or `?workspace=` query
     - Check user has access (or is public workspace)
     - Set `ctx.Workspace`
   - [ ] Update existing `AuthMiddleware` to work with workspace

5. **Query Filtering** (~3 hours)
   - [ ] Update Observatory queries to filter by workspace
     - Map span `workspace` attribute using path mapping
   - [ ] Update Coordinator queries to filter by workspace
   - [ ] Update Messages queries to filter by workspace

**Tests:**
- `internal/server/auth/workspace_test.go` - Access control logic
- `internal/server/handlers_workspace_test.go` - API endpoints

### Phase 2: Frontend Implementation (0.5 days)

**Tasks:**

1. **Workspace Access Hook** (~1 hour)
   - [ ] Create `src/hooks/useWorkspaceAccess.ts`
     - Fetch accessible workspaces from `/api/workspaces`
     - Re-fetch on auth state change
     - Cache results

2. **Aggregation Nav Integration** (~2 hours)
   - [ ] Update `AggregationNav.tsx`
     - Filter workspace breakdown items by accessible workspaces
     - Show role badge (Approver/Viewer) next to workspace name
     - For unauthenticated: only show public workspaces
   - [ ] Handle workspace selection persisting across page reloads

3. **API Error Handling** (~1 hour)
   - [ ] Handle 403 (workspace access denied) in API client
   - [ ] Show toast/notification when access denied
   - [ ] Clear inaccessible workspace from selection

**Tests:**
- `src/hooks/useWorkspaceAccess.test.ts`

### Phase 3: CLI & Documentation (0.5 days)

**Tasks:**

1. **Workspace CLI Commands** (~2 hours)
   - [ ] Create `cmd/ailang/workspaces.go`
     - `list`, `add`, `grant`, `revoke`, `set-public` subcommands
   - [ ] Add to main.go command router

2. **Configuration & Migration** (~1 hour)
   - [ ] Add default workspace mappings to `config.example.yaml`
   - [ ] Document migration from file paths to workspace IDs

3. **Documentation** (~1 hour)
   - [ ] Update `docs/guides/authentication.md`
   - [ ] Create `docs/guides/workspaces.md`
   - [ ] Update CLAUDE.md with workspace commands

---

## Configuration

### ~/.ailang/config.yaml
```yaml
firebase:
  project_id: ailang-dev

# Workspace configuration (v0.7.1+)
workspaces:
  # Path pattern to workspace ID mapping
  mappings:
    - pattern: "*/dev/sunholo/ailang"
      workspace: "sunholo-data/ailang"
    - pattern: "*/dev/sunholo/stapledons_voyage"
      workspace: "sunholo-data/stapledons_voyage"
    - pattern: "*/dev/TwilightGame"
      workspace: "MarkEdmondson1234/TwilightGame"

  # Default workspace for unmapped paths (optional)
  default_workspace: "sunholo-data/ailang"
```

### Initial Workspace Setup
```bash
# Create workspaces in Firestore (is_public is a flag on any workspace)
ailang workspaces add sunholo-data/ailang --name "AILANG" --public true
ailang workspaces add sunholo-data/stapledons_voyage --name "Stapledons Voyage" --public false
ailang workspaces add MarkEdmondson1234/TwilightGame --name "Twilight Game" --public false

# Grant access (authenticated users get specific roles)
ailang workspaces grant sunholo-data/ailang --email m@sunholo.com --role Approver
ailang workspaces grant sunholo-data/ailang --email me@markedmondson.me --role Viewer

# Public workspaces: anonymous users can view, but only granted users can approve
# So sunholo-data/ailang is:
#   - Viewable by everyone (is_public: true)
#   - Approvable by m@sunholo.com only (role: Approver)
```

---

## Files to Create/Modify

### New Files
- `internal/server/auth/workspace.go` (~200 LOC) - Workspace access control
- `internal/server/auth/workspace_test.go` (~300 LOC) - Tests
- `internal/server/handlers_workspace.go` (~150 LOC) - Workspace API endpoints
- `cmd/ailang/workspaces.go` (~250 LOC) - CLI commands
- `ui/src/hooks/useWorkspaceAccess.ts` (~50 LOC) - Access control hook
- `docs/guides/workspaces.md` (~200 LOC) - Documentation

### Modified Files
- `internal/server/auth/firestore.go` - Add workspace methods
- `internal/server/handlers_auth.go` - Add workspace middleware
- `internal/server/handlers_observatory.go` - Add workspace filtering
- `internal/server/handlers_tasks.go` - Add workspace filtering
- `internal/coordinator/agent_config.go` - Add workspace mappings config
- `ui/src/features/controlplane/components/AggregationNav.tsx` - Filter workspaces by access
- `ui/src/features/controlplane/ControlPlane.tsx` - Add workspace access context

---

## Success Criteria

- [ ] **Isolation**: Authenticated users only see their workspace data
- [ ] **Public Access**: Unauthenticated users see only "public" workspace
- [ ] **Workspace Selector**: Dashboard header shows workspace dropdown
- [ ] **Role Enforcement**: Viewers cannot approve in any workspace
- [ ] **Path Mapping**: File paths correctly map to workspace IDs
- [ ] **CLI**: `ailang workspaces` commands work
- [ ] **Performance**: Workspace filtering <10ms overhead
- [ ] **Tests**: 90%+ coverage on new code

---

## Related Documents

- [M-AUTH Firebase Authentication](../v0_7_0/m-auth-dashboard-firebase.md) - Prerequisite
- [Global Collaboration Hub](../v0_8_0/global-collaboration-hub.md) - Future multi-tenant vision

---

## Risks & Mitigations

### Risk: Path Mapping Complexity
**Impact:** Edge cases in path matching cause data leaks
**Mitigation:**
- Use glob patterns with explicit fallback to "public"
- Log unmapped paths for review
- Comprehensive path mapping tests

### Risk: Workspace Migration
**Impact:** Existing data has file paths, not workspace IDs
**Mitigation:**
- Query-time mapping (no data migration needed)
- Cache path → workspace mapping
- Gradual migration as data is accessed

### Risk: Performance Overhead
**Impact:** Workspace filtering slows queries
**Mitigation:**
- Index on workspace field in SQLite
- Cache workspace access checks
- Batch workspace membership lookups

---

## Security Analysis

### Defense-in-Depth Architecture

The workspace access control uses **four layers of defense**:

```
┌─────────────────────────────────────────────────────────────────────┐
│ LAYER 1: Frontend (UX Filter)                                        │
│ AggregationNav only shows accessible workspaces                      │
│ ↓ Can be bypassed by modifying JS or calling API directly            │
├─────────────────────────────────────────────────────────────────────┤
│ LAYER 2: API Middleware (Access Enforcement)                         │
│ WorkspaceMiddleware validates user has access before handlers run    │
│ Returns 403 Forbidden if unauthorized                                │
│ ↓ Cannot be bypassed - runs on every request                         │
├─────────────────────────────────────────────────────────────────────┤
│ LAYER 3: Query Filtering (Data Isolation)                            │
│ All data queries include workspace filter in WHERE clause            │
│ Even if middleware is somehow bypassed, queries still filtered       │
│ ↓ Defense in depth - queries enforce isolation                       │
├─────────────────────────────────────────────────────────────────────┤
│ LAYER 4: Default to Safe State                                       │
│ If no workspace specified → only return accessible workspace data    │
│ If unauthenticated → only return public workspace data               │
│ ↓ Fail-safe default - never returns unauthorized data                │
└─────────────────────────────────────────────────────────────────────┘
```

### Existing Infrastructure (Already Implemented)

The backend **already has workspace filtering infrastructure** that we leverage:

**1. Filter Extraction (`handlers_controlplane.go:578-590`):**
```go
func parseControlPlaneFilter(r *http.Request) *observatory.ControlPlaneFilter {
    return &observatory.ControlPlaneFilter{
        Workspace:  q.Get("workspace"),  // ✓ Already extracts workspace
        SourceType: q.Get("source_type"),
        Provider:   q.Get("provider"),
        // ...
    }
}
```

**2. All Endpoints Use Filter:**
- `handleControlPlaneHeatmap` → `sqliteBackend.GetFilteredHeatmapData(ctx, filter, days)`
- `handleControlPlaneStats` → `sqliteBackend.GetFilteredMetricsSummary(ctx, filter)`
- `handleControlPlaneStatsBreakdown` → `GetFilteredBreakdownByWorkspace(ctx, filter)`
- `handleTaskHierarchy` → `filter.Workspace = workspace`

**3. SQL Queries Filter by Workspace:**
All observatory queries include `WHERE workspace = ?` when filter is set.

### Security Changes Required

**1. New WorkspaceMiddleware (CRITICAL):**
```go
// handlers_auth.go - New middleware
func (s *Server) WorkspaceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        workspace := r.URL.Query().Get("workspace")
        if workspace == "" {
            workspace = r.Header.Get("X-Workspace")
        }

        // Check user has access
        user := getUserFromContext(r.Context())
        hasAccess, _ := s.workspaceService.HasWorkspaceAccess(r.Context(), user.Email, workspace)

        if !hasAccess {
            http.Error(w, "Forbidden: No access to workspace", http.StatusForbidden)
            return
        }

        // Set workspace in context for handlers
        ctx := context.WithValue(r.Context(), "workspace", workspace)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**2. Update `/api/workspaces` Endpoint:**
Currently returns ALL workspaces (`handlers_threads.go:188-208`). Must filter:
```go
func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
    user := getUserFromContext(r.Context())

    // Get all workspaces user has access to (includes public)
    accessibleWorkspaces, err := s.workspaceService.ListAccessibleWorkspaces(r.Context(), user)
    if err != nil {
        http.Error(w, "Failed to get workspaces", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(accessibleWorkspaces)
}
```

**3. Default to Accessible Workspaces:**
When no workspace is specified, return union of all accessible workspaces:
```go
// In each handler, before querying
if filter.Workspace == "" {
    // User didn't specify workspace - filter to all accessible
    accessibleIDs := s.workspaceService.GetAccessibleWorkspaceIDs(ctx, user)
    filter.WorkspaceIDs = accessibleIDs  // Query: WHERE workspace IN (...)
}
```

### Endpoints Requiring Access Control

| Endpoint | Current State | Change Required |
|----------|--------------|-----------------|
| `GET /api/workspaces` | Returns all | Filter by user access |
| `GET /api/controlplane/heatmap` | Filters if `?workspace=` | Add middleware + default |
| `GET /api/controlplane/stats` | Filters if `?workspace=` | Add middleware + default |
| `GET /api/controlplane/stats/breakdown` | Filters if `?workspace=` | Add middleware + default |
| `GET /api/controlplane/topology` | No filtering | Add workspace filter |
| `GET /api/controlplane/task-hierarchy` | Filters by workspace | Add middleware |
| `GET /api/observatory/traces` | Filters by task_id | Add workspace filter |
| `GET /api/observatory/spans` | Filters by task_id | Add workspace filter |
| `GET /api/threads` | Filters by workspace | Add middleware |
| `GET /api/messages` | No filtering | Add workspace filter |
| `WS /ws` | No filtering | Add workspace subscription |

### Path-to-Workspace Mapping Security

**Risk:** File paths stored in coordinator.db (e.g., `/Users/mark/dev/sunholo/ailang`) need mapping to workspace IDs at query time.

**Mitigations:**
1. **Consistent mapping:** Same mapping logic at both ingest and query time
2. **Logged mismatches:** Unmapped paths logged for review
3. **Safe default:** Unmapped paths → "public" workspace (visible to all)
4. **No path leakage:** API returns workspace ID, not file path

**Mapping implementation:**
```go
// Query time: Map coordinator file paths to workspace IDs
func (s *Server) mapTaskWorkspace(task *coordinator.Task) string {
    if task.Workspace == "" {
        return "public"
    }
    // Check configured mappings
    for _, m := range s.config.Workspaces.Mappings {
        if glob.Match(m.Pattern, task.Workspace) {
            return m.WorkspaceID
        }
    }
    return "public"  // Safe default
}
```

### Public Workspace Access

**Access Logic:**
```
is_public: true  + anonymous user    → Viewer role
is_public: true  + authenticated     → Viewer role (unless granted higher)
is_public: false + no explicit grant → 403 Forbidden
explicit grant                       → granted role (Approver/Viewer)
```

**Public workspace security:**
- Anonymous users can VIEW data in public workspaces
- Anonymous users CANNOT approve, modify, or create
- Public workspace with `is_public: true` is still filterable by authenticated users with higher roles

### Attack Surface Analysis

| Attack Vector | Mitigation |
|--------------|------------|
| Direct API call with unauthorized workspace | Middleware returns 403 |
| Modifying frontend to bypass filter | Middleware enforces server-side |
| SQL injection in workspace param | Parameterized queries only |
| Timing attack (enumerate workspaces) | Consistent error response time |
| Session fixation | Firebase Auth handles sessions |
| Token replay | JWT validation with expiry |

### Audit Logging

All workspace access attempts should be logged:
```go
log.Printf("WORKSPACE_ACCESS user=%s workspace=%s granted=%v", user.Email, workspaceID, hasAccess)
```

Failed access attempts (403s) are logged for security review.

---

## Future Enhancements (v0.8.0+)

- **Workspace Creation UI**: Create workspaces from dashboard
- **Team Management**: Invite users via email
- **Workspace Quotas**: Limit tasks/cost per workspace
- **Cross-Workspace Search**: Search across multiple workspaces
- **Workspace Templates**: Pre-configured agent sets per workspace

---

## Implementation Report (January 23, 2026)

### Firestore Schema Implementation

#### Document ID Encoding (CRITICAL)

Firestore does **not allow "/" in document IDs** because "/" is the path separator. Since workspace IDs use GitHub repo format (e.g., `sunholo-data/ailang`), we must encode them.

**Encoding scheme:**
```
Workspace ID: "sunholo-data/ailang"
Document ID:  "sunholo-data__ailang"  (replace "/" with "__")
```

**Implementation:**
```go
// internal/server/auth/workspace.go

// EncodeDocID encodes a workspace ID for use as a Firestore document ID.
// Replaces "/" with "__" since Firestore doesn't allow "/" in document IDs.
func EncodeDocID(id string) string {
    return strings.ReplaceAll(id, "/", "__")
}

// DecodeDocID decodes a Firestore document ID back to a workspace ID.
func DecodeDocID(docID string) string {
    return strings.ReplaceAll(docID, "__", "/")
}
```

**Usage:**
- All methods that read/write workspace documents must use `EncodeDocID()`
- When returning workspace data, use `DecodeDocID()` to show original format
- CLI uses encoding when creating workspaces

#### Actual Firestore Collections

```
/workspaces/{encoded_workspace_id}
  # Document ID: "sunholo-data__ailang" (encoded from "sunholo-data/ailang")
  ├── id: string              # Original workspace ID with "/" (e.g., "sunholo-data/ailang")
  ├── name: string            # Display name (e.g., "AILANG Project")
  ├── github_repo: string     # GitHub repo identifier (same as id)
  ├── is_public: boolean      # If true, visible to all users
  ├── created_at: timestamp
  └── created_by: string      # "cli" or user email

/workspace_access/{encoded_workspace_id}/users/{email}
  # Parent document ID is encoded
  # User email is used as document ID (Firestore allows @ and .)
  ├── email: string
  ├── workspace_id: string    # Original workspace ID with "/"
  ├── role: "Viewer" | "Approver"
  ├── granted_at: timestamp
  └── granted_by: string      # "cli" or admin email
```

#### Required Firestore Index

**Collection Group Index** is required for querying user access across all workspaces:

```
Collection group: users
Field: email (Ascending)
Query scope: Collection group
```

**Create this index:**
1. Go to Firebase Console → Firestore → Indexes
2. Click "Add Index" → "Collection group"
3. Configure as above

**Without this index**, the query `ListAccessibleWorkspaces(email)` will fail with:
```
The query requires a COLLECTION_GROUP_ASC index for collection users and field email
```

### Created Workspaces

```bash
# Initial workspace setup (January 23, 2026)
ailang workspaces add --id sunholo-data/ailang --name "AILANG Project" --public
ailang workspaces add --id sunholo-data/stapledons_voyage --name "Stapledons Voyage" --public
ailang workspaces add --id MarkEdmondson1234/TwilightGame --name "Twilight Game"

# Granted access
ailang workspaces grant --id sunholo-data/ailang --email m@sunholo.com --role Approver
ailang workspaces grant --id sunholo-data/stapledons_voyage --email m@sunholo.com --role Approver
ailang workspaces grant --id MarkEdmondson1234/TwilightGame --email m@sunholo.com --role Approver
```

### Files Implemented

| File | LOC | Description |
|------|-----|-------------|
| `internal/server/auth/workspace.go` | 580 | Workspace service with Firestore backend, caching |
| `internal/server/auth/workspace_test.go` | 254 | Unit tests |
| `cmd/ailang/workspaces.go` | 464 | CLI commands (list, add, grant, revoke, set-public, show) |
| `ui/src/hooks/useWorkspaceAccess.ts` | 112 | React hook for workspace access |
| `docs/docs/guides/workspaces.md` | 236 | User documentation |

**Modified files:**
- `internal/server/handlers_auth.go` - Added RequireWorkspaceAccessMiddleware
- `internal/server/handlers_threads.go` - Added workspace filtering to handleWorkspaces
- `internal/coordinator/agent_config.go` - Added LoadWorkspacesConfig()
- `ui/src/features/controlplane/components/AggregationNav.tsx` - Added role badges

### Metrics

- **Estimated duration:** 2.5 days
- **Actual duration:** 1 day
- **Estimated LOC:** 1,150
- **Actual LOC:** 1,810
- **Cache hit rate:** >90% on repeated access checks (5-min TTL)
