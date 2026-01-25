# M-COLLAB-PROVIDER-STATS Sprint Plan

**Sprint**: Provider & Workspace Tags and Statistics Integration
**Duration**: 0.5 days (4 hours) - reduced because components already exist
**Design Doc**: [m-collab-provider-stats.md](m-collab-provider-stats.md)
**Updated**: 2026-01-06

## Sprint Overview

**Key Discovery:** Most components already exist but are not integrated!

### Already Implemented ✅
| Component | Location | Status |
|-----------|----------|--------|
| ProviderBadge | `ui/src/components/badges/ProviderBadge/` | ✅ Built, **not used in ThreadList** |
| WorkspaceBadge | `ui/src/components/badges/WorkspaceBadge/` | ✅ Built, used in ThreadList |
| FilterBar | `ui/src/components/badges/FilterBar/` | ✅ Built, **not integrated** |
| GetTaskStats | `internal/coordinator/store_sqlite.go` | ✅ Returns ByProvider, ByWorkspace counts |
| StatsPanel | `ui/src/components/metrics/StatsPanel/` | ✅ Shows by_provider, by_workspace |

### Integration Gaps (This Sprint)
| Gap | Description | Est |
|-----|-------------|-----|
| ProviderBadge in ThreadList | Component exists but not rendered | 0.5h |
| FilterBar integration | Component exists but not wired | 1.5h |
| Cost/token stats | GetTaskStats returns counts only | 1.5h |

**Total Estimated: 3.5-4 hours**

## Milestones

### M1: Integrate ProviderBadge into ThreadList (~30 min)

**Files:**
- `ui/src/features/messaging/MessageCenter/ThreadList.tsx`

**Current State:** WorkspaceBadge is imported and used; ProviderBadge exists but not imported.

**Tasks:**
- [ ] Import ProviderBadge from `../../../components/badges/ProviderBadge`
- [ ] Add `<ProviderBadge provider={thread.target_agent} />` next to WorkspaceBadge
- [ ] Verify color scheme works (Claude=orange, Gemini=blue)

**Acceptance Criteria:**
- [ ] Provider badge visible on each thread
- [ ] Color-coded by provider

---

### M2: Integrate FilterBar into MessageCenter (~1.5 hours)

**Files:**
- `ui/src/features/messaging/MessageCenter/MessageCenter.tsx`
- `ui/src/features/messaging/MessageCenter/ThreadList.tsx`

**Current State:** FilterBar component exists with full implementation; not used anywhere.

**Tasks:**
- [ ] Import FilterBar into MessageCenter
- [ ] Add filter state: `const [selectedProvider, setSelectedProvider] = useState('')`
- [ ] Add filter state: `const [selectedWorkspace, setSelectedWorkspace] = useState('')`
- [ ] Extract unique providers from threads array
- [ ] Extract unique workspaces from threads array
- [ ] Filter threads before passing to ThreadList
- [ ] Add localStorage persistence for filter state
- [ ] Position FilterBar above ThreadList

**Implementation:**
```typescript
// In MessageCenter.tsx
import { FilterBar } from '../../../components/badges/FilterBar';

const [selectedProvider, setSelectedProvider] = useState(
  localStorage.getItem('collab_filter_provider') || ''
);
const [selectedWorkspace, setSelectedWorkspace] = useState(
  localStorage.getItem('collab_filter_workspace') || ''
);

// Extract unique values
const providers = [...new Set(threads.map(t => t.target_agent).filter(Boolean))];
const workspaces = [...new Set(threads.map(t => t.workspace).filter(Boolean))];

// Filter threads
const filteredThreads = threads.filter(t => {
  if (selectedProvider && t.target_agent !== selectedProvider) return false;
  if (selectedWorkspace && t.workspace !== selectedWorkspace) return false;
  return true;
});

// Save to localStorage on change
useEffect(() => {
  localStorage.setItem('collab_filter_provider', selectedProvider);
  localStorage.setItem('collab_filter_workspace', selectedWorkspace);
}, [selectedProvider, selectedWorkspace]);
```

**Acceptance Criteria:**
- [ ] FilterBar visible above thread list
- [ ] Provider dropdown populated with available providers
- [ ] Workspace dropdown populated with available workspaces
- [ ] Selecting filter updates visible threads
- [ ] Filters persist on refresh

---

### M3: Add Cost/Token Breakdown to Statistics (~1.5 hours)

**Files:**
- `internal/coordinator/store_sqlite.go` - Extend GetTaskStats
- `internal/server/handlers_statistics.go` - Update response struct
- `ui/src/components/metrics/StatsPanel/StatsPanel.tsx` - Display cost

**Current State:**
- GetTaskStats returns ByProvider/ByWorkspace with counts only
- StatsPanel shows counts but not cost/tokens

**Backend Tasks:**
- [ ] Add ProviderDetailedStats struct with cost, tokens
- [ ] Update SQL to aggregate cost/tokens per provider
- [ ] Update SQL to aggregate cost/tokens per workspace
- [ ] Return detailed breakdown in API response

**SQL Change:**
```sql
-- Per-provider with cost/tokens
SELECT
    COALESCE(provider, 'unknown') as provider,
    COUNT(*) as task_count,
    COALESCE(SUM(cost), 0) as total_cost,
    COALESCE(SUM(input_tokens), 0) as input_tokens,
    COALESCE(SUM(output_tokens), 0) as output_tokens
FROM tasks
WHERE provider IS NOT NULL
GROUP BY provider
```

**Frontend Tasks:**
- [ ] Update TypeScript interfaces for detailed stats
- [ ] Display cost next to count in provider cards
- [ ] Display cost next to count in workspace cards
- [ ] Format cost as currency ($X.XX)

**Acceptance Criteria:**
- [ ] API returns cost_usd, input_tokens, output_tokens per provider
- [ ] API returns cost_usd, input_tokens, output_tokens per workspace
- [ ] StatsPanel displays cost alongside task counts

---

## Files Modified

| File | Changes | Est LOC |
|------|---------|---------|
| `ThreadList.tsx` | Import + add ProviderBadge | +5 |
| `MessageCenter.tsx` | FilterBar + state + filtering | +60 |
| `store_sqlite.go` | Extend GetTaskStats with costs | +40 |
| `handlers_statistics.go` | Update structs + queries | +30 |
| `StatsPanel.tsx` | Display cost in breakdowns | +25 |
| **Total** | | **~160** |

## Success Metrics

- [ ] ProviderBadge visible on threads (M1)
- [ ] FilterBar filters threads by provider (M2)
- [ ] FilterBar filters threads by workspace (M2)
- [ ] Statistics show cost breakdown by provider (M3)
- [ ] Statistics show cost breakdown by workspace (M3)
- [ ] All existing tests pass
- [ ] UI builds: `cd ui && npm run build`

## Testing

### Manual Tests
- [ ] Start services: `make services-start`
- [ ] Open dashboard: http://localhost:1957
- [ ] Verify provider badge on threads
- [ ] Test provider filter dropdown
- [ ] Test workspace filter dropdown
- [ ] Verify filter persistence after refresh
- [ ] Check statistics panel shows cost breakdown

### Automated Tests
- [ ] Run `make test` - all pass
- [ ] Run `cd ui && npm run build` - builds successfully

## Definition of Done

- [ ] All acceptance criteria met
- [ ] No console errors in browser
- [ ] Build passes
- [ ] Tests pass

---

**Created**: 2024-12-30
**Updated**: 2026-01-06 (reduced scope after discovering existing components)
