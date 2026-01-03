# M-COLLAB-PROVIDER-STATS Sprint Plan

**Sprint**: Provider & Workspace Tags and Statistics
**Duration**: 1.5 days
**Design Doc**: [m-collab-provider-stats.md](m-collab-provider-stats.md)

## Sprint Overview

Add provider and workspace as visible metadata with filtering and per-dimension statistics in the Collaboration Hub.

## Tasks

### Phase 1: Database Schema (1 hour)

#### Task 1.1: Add Workspace Column to Tasks Table
**Files**: `internal/coordinator/store_sqlite.go`
**Effort**: 30 min

```go
// In migrate() function, add:
alterQueries := []string{
    "ALTER TABLE tasks ADD COLUMN workspace TEXT",
}
// Add index
"CREATE INDEX IF NOT EXISTS idx_tasks_workspace ON tasks(workspace)"
```

**Acceptance Criteria:**
- [ ] Column exists in tasks table
- [ ] Index created for filtering performance
- [ ] Existing databases migrate cleanly

#### Task 1.2: Update TaskRecord Struct
**Files**: `internal/coordinator/types.go` or `store_sqlite.go`
**Effort**: 30 min

- Add `Workspace string` field to TaskRecord
- Update scanTask to read workspace
- Update CreateTask/UpdateTask to save workspace

### Phase 2: Coordinator Workspace Tracking (2 hours)

#### Task 2.1: Capture Thread Workspace in Daemon
**Files**: `internal/coordinator/daemon.go`
**Effort**: 45 min

```go
// In processTask or similar:
// Get workspace from thread
thread, _ := d.msgStore.GetThread(task.ThreadID)
sourceWorkspace := ""
if thread != nil && thread.Workspace != "" {
    sourceWorkspace = thread.Workspace
}

// Store on task
task.Workspace = sourceWorkspace
```

**Acceptance Criteria:**
- [ ] Thread workspace captured when task starts
- [ ] Stored in task record

#### Task 2.2: Include Workspace in Result Metadata
**Files**: `internal/coordinator/daemon.go`
**Effort**: 45 min

```go
// In postTaskResult, add to metadata:
metadata := map[string]interface{}{
    "provider": result.Provider,
    "source_workspace": task.Workspace,  // Add this
    "execution_stats": map[string]interface{}{
        // ...existing fields...
        "workspace": task.Workspace,  // Also add here
    },
}
```

**Acceptance Criteria:**
- [ ] Workspace included in message metadata
- [ ] Available for UI to read

#### Task 2.3: Update Store Methods
**Files**: `internal/coordinator/store_sqlite.go`
**Effort**: 30 min

- Update MarkTaskCompleted to save workspace
- Update any queries that should filter by workspace

### Phase 3: Provider Tag Display (1.5 hours)

#### Task 3.1: Create ProviderBadge Component
**Files**: `ui/src/components/ProviderBadge.tsx`
**Effort**: 45 min

```typescript
interface ProviderBadgeProps {
  provider?: string;
}

const providerColors: Record<string, { bg: string; text: string }> = {
  'claude-code': { bg: '#F97316', text: 'white' },  // Orange
  'gemini-cli': { bg: '#3B82F6', text: 'white' },   // Blue
  'gemini-api': { bg: '#22C55E', text: 'white' },   // Green
};

export const ProviderBadge: React.FC<ProviderBadgeProps> = ({ provider }) => {
  if (!provider) return null;
  const colors = providerColors[provider] || { bg: '#6B7280', text: 'white' };
  return (
    <span className="provider-badge" style={{ backgroundColor: colors.bg, color: colors.text }}>
      {provider}
    </span>
  );
};
```

**Acceptance Criteria:**
- [ ] Badge renders with correct color
- [ ] Handles undefined/null gracefully (returns null)
- [ ] Unknown providers get gray color

#### Task 3.2: Add Provider Badge to Message List
**Files**: `ui/src/components/MessageCenter/ConversationView.tsx`
**Effort**: 45 min

- Import ProviderBadge
- Extract provider from message metadata
- Add badge next to status indicator
- Style to align with existing badges

**Acceptance Criteria:**
- [ ] Provider badge visible on result messages
- [ ] Positioned consistently with other badges

### Phase 4: Workspace Tag Display (1.5 hours)

#### Task 4.1: Create WorkspaceBadge Component
**Files**: `ui/src/components/WorkspaceBadge.tsx`
**Effort**: 45 min

```typescript
interface WorkspaceBadgeProps {
  workspace?: string;
}

export const WorkspaceBadge: React.FC<WorkspaceBadgeProps> = ({ workspace }) => {
  if (!workspace) return null;

  // Get folder name (last path component)
  const folderName = workspace.split('/').pop() || workspace;

  return (
    <span
      className="workspace-badge"
      title={workspace}  // Full path on hover
    >
      📁 {folderName}
    </span>
  );
};
```

**Acceptance Criteria:**
- [ ] Badge shows folder name
- [ ] Tooltip shows full path
- [ ] Handles undefined/null gracefully

#### Task 4.2: Add Workspace Badge to Message List
**Files**: `ui/src/components/MessageCenter/ConversationView.tsx`
**Effort**: 45 min

- Import WorkspaceBadge
- Extract workspace from message metadata (source_workspace or thread.workspace)
- Add badge next to provider badge

**Acceptance Criteria:**
- [ ] Workspace badge visible on messages
- [ ] Shows thread workspace when available

### Phase 5: Filtering UI (2 hours)

#### Task 5.1: Create FilterBar Component
**Files**: `ui/src/components/FilterBar.tsx`
**Effort**: 1 hour

```typescript
interface FilterBarProps {
  providers: string[];        // Available providers
  workspaces: string[];       // Available workspaces
  selectedProviders: string[];
  selectedWorkspaces: string[];
  onProviderChange: (providers: string[]) => void;
  onWorkspaceChange: (workspaces: string[]) => void;
}

export const FilterBar: React.FC<FilterBarProps> = (props) => {
  // Render provider chips (toggleable)
  // Render workspace dropdown
};
```

**Acceptance Criteria:**
- [ ] Provider chips are toggleable
- [ ] Workspace dropdown shows available options
- [ ] Clear filters button

#### Task 5.2: Integrate FilterBar into ThreadList/MessageCenter
**Files**: `ui/src/components/MessageCenter/MessageCenter.tsx`
**Effort**: 45 min

- Add filter state
- Pass to FilterBar
- Filter displayed messages based on selections
- Persist to localStorage

**Acceptance Criteria:**
- [ ] Filters affect displayed messages
- [ ] Filter state persists on refresh

#### Task 5.3: Add CSS Styles
**Files**: `ui/src/components/MessageCenter/MessageCenter.css`
**Effort**: 15 min

- Style filter bar
- Style badges
- Responsive layout

### Phase 6: Filter API (1.5 hours)

#### Task 6.1: Add Filter Params to Messages API
**Files**: `internal/server/handlers_messages.go`
**Effort**: 45 min

```go
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
    threadID := r.URL.Query().Get("thread_id")
    provider := r.URL.Query().Get("provider")
    workspace := r.URL.Query().Get("workspace")

    // Build SQL with optional WHERE clauses
    query := "SELECT ... FROM messages m"
    // Join with tasks if filtering by provider/workspace
}
```

**Acceptance Criteria:**
- [ ] provider filter works
- [ ] workspace filter works
- [ ] Multiple filters combine (AND)

#### Task 6.2: Add Workspaces List Endpoint
**Files**: `internal/server/handlers_messages.go`
**Effort**: 45 min

```go
// GET /api/workspaces
func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
    // SELECT DISTINCT workspace FROM threads WHERE workspace IS NOT NULL
    // OR SELECT DISTINCT workspace FROM tasks WHERE workspace IS NOT NULL
}
```

**Acceptance Criteria:**
- [ ] Returns unique workspaces
- [ ] Used to populate filter dropdown

### Phase 7: Statistics (2.5 hours)

#### Task 7.1: Create Statistics API Endpoint
**Files**: `internal/server/handlers_statistics.go` (new)
**Effort**: 1 hour

```go
// GET /api/statistics

type StatisticsResponse struct {
    Global      GlobalStats      `json:"global"`
    ByProvider  []ProviderStats  `json:"by_provider"`
    ByWorkspace []WorkspaceStats `json:"by_workspace"`
}

func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
    // Query for global stats
    // Query for per-provider breakdown
    // Query for per-workspace breakdown
}
```

**Acceptance Criteria:**
- [ ] Returns global totals
- [ ] Returns per-provider breakdown
- [ ] Returns per-workspace breakdown

#### Task 7.2: Create Statistics UI Components
**Files**: `ui/src/components/StatsPanel.tsx`
**Effort**: 1 hour

- ProviderStatsPanel: Cards for each provider
- WorkspaceStatsPanel: Cards for each workspace
- Fetch from /api/statistics

**Acceptance Criteria:**
- [ ] Shows provider breakdown cards
- [ ] Shows workspace breakdown cards
- [ ] Loading and error states

#### Task 7.3: Integrate into Dashboard
**Files**: `ui/src/components/MessageCenter/MessageCenter.tsx` or new page
**Effort**: 30 min

- Add stats panels to UI
- Fetch on mount
- Refresh button

**Acceptance Criteria:**
- [ ] Stats visible in dashboard
- [ ] Data updates on refresh

## Testing

### Manual Tests
- [ ] Start services: `make services-start`
- [ ] Set workspace on a thread via folder picker
- [ ] Run a task (send directive to coordinator)
- [ ] Verify provider badge appears on result message
- [ ] Verify workspace badge appears on result message
- [ ] Test provider filter: select claude-code only
- [ ] Test workspace filter: select specific folder only
- [ ] Check statistics panel shows breakdown
- [ ] Verify filters persist on page refresh

### Automated Tests
- [ ] Store migration adds workspace column
- [ ] ProviderBadge renders correctly for each provider
- [ ] WorkspaceBadge shows folder name with tooltip
- [ ] Statistics aggregation returns correct totals

## Definition of Done

- [ ] All acceptance criteria met
- [ ] No console errors in browser
- [ ] Responsive design works
- [ ] Build passes: `cd ui && npm run build`
- [ ] Go tests pass: `make test`
- [ ] Documentation updated

## Rollback Plan

If issues arise:
1. Workspace column is additive (no breaking changes)
2. Filters can be disabled in UI
3. Statistics endpoint can return empty arrays
4. Badges are optional (check for null)

---

**Created**: 2024-12-30
