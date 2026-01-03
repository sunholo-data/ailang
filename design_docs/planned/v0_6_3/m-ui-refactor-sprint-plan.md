# Sprint Plan: M-UI-REFACTOR - AI-Friendly File Sizes

## Summary
Refactor the ui/ folder to reduce all files to ≤400 lines by extracting ~2,400 lines of inline CSS to `.module.css` files, consolidating duplicated utilities (Icons, formatters), and splitting monolithic components into focused modules.

**Duration:** 3 sessions (~11 hours total)
**Dependencies:** None
**Risk Level:** Medium (refactor with UI regressions possible)

## Current Status Analysis

### Current Problem Files
| File | Lines | CSS Lines | After Refactor |
|------|-------|-----------|----------------|
| ConversationView.tsx | 1,383 | 778 | ~350 |
| ApprovalQueue.tsx | 1,073 | 691 | ~300 |
| Monitor.tsx | 999 | 476 | ~350 |
| App.tsx | 949 | 446 | ~400 |
| **Total** | **4,404** | **2,391** | **~1,400** |

### Velocity
- Recent average: ~150-200 LOC/day for new feature work
- Refactoring velocity: Higher (~300-400 LOC/day) since extracting existing code
- Estimated capacity: ~900 LOC/session (3 hours focused work)

### Duplication to Eliminate
- Icons object: ~300 lines duplicated across 4 files
- `formatTimestamp()`: 3 copies
- `formatDuration()`: 2 copies
- Approval state patterns: 2 implementations

## Proposed Milestones

### Milestone 1: Extract Utilities and CSS
**Goal:** Create shared utilities and extract all inline CSS to `.module.css` files
**Estimated:** ~2,400 LOC moved (CSS) + ~170 LOC consolidated (utilities)
**Duration:** 1 session (~3-4 hours)

**Tasks:**
1. Create `ui/src/utils/formatters.ts`:
   - Extract `formatTimestamp()` from ConversationView
   - Extract `formatDuration()` from Monitor
   - Extract `formatCost()` from Monitor
   - Extract `formatTokens()` from Monitor
   - Add `formatRelativeTime()` helper

2. Create `ui/src/components/common/Icons.tsx`:
   - Audit all Icons objects in codebase
   - Merge into single comprehensive set
   - Export named icon components

3. Create CSS module files:
   - `ui/src/App.module.css` (~450 lines from App.tsx)
   - `ui/src/components/ApprovalQueue.module.css` (~700 lines)
   - `ui/src/components/MessageCenter/ConversationView.module.css` (~780 lines)
   - `ui/src/components/Monitor.module.css` (~480 lines)

4. Update imports in all components

**Acceptance Criteria:**
- [ ] `npm run build` succeeds
- [ ] No inline `<style>` tags in files over 200 lines
- [ ] Single Icons.tsx with all icons
- [ ] Single formatters.ts with all format functions
- [ ] All 4 critical files reduced by ~500+ lines each

**Risks:**
- CSS specificity changes may break styles → Use CSS modules which scope automatically
- Import path updates may break → Run build after each file

---

### Milestone 2: Split ConversationView
**Goal:** Break ConversationView (1,383→350 lines) into focused components
**Estimated:** ~600 LOC restructured (150 new + 450 moved)
**Duration:** 1 session (~3-4 hours)

**Tasks:**
1. Create `ui/src/components/MessageCenter/MessageList.tsx` (~150 LOC):
   - Extract message rendering logic
   - Extract truncation logic (`truncateContent`, `truncateOutput`)
   - Handle `expandedMessages` state
   - Handle markdown rendering

2. Create `ui/src/components/MessageCenter/MessageInput.tsx` (~100 LOC):
   - Extract input area JSX
   - Extract message kind selector
   - Handle workspace input
   - Handle send button and loading state

3. Create `ui/src/components/MessageCenter/ApprovalPanel.tsx` (~150 LOC):
   - Extract inline approval request rendering
   - Extract `handledApprovals` and `approvalNotes` state
   - Handle approval/rejection callbacks

4. Refactor ConversationView.tsx (~350 LOC):
   - Import and compose new components
   - Keep WebSocket connection logic
   - Keep high-level state management

**Acceptance Criteria:**
- [ ] ConversationView.tsx ≤400 lines
- [ ] All message types render correctly
- [ ] Approval flow works end-to-end
- [ ] Input and send works
- [ ] `npm run build` succeeds

**Risks:**
- Props drilling complexity → Accept for now, context later if needed
- State coordination bugs → Test each component independently

---

### Milestone 3: Split App and Monitor
**Goal:** Break App (949→400) and Monitor (999→350) into focused components
**Estimated:** ~600 LOC restructured
**Duration:** 1 session (~3-4 hours)

**Tasks:**
1. Create `ui/src/AppShell.tsx` (~100 LOC):
   - Extract header JSX
   - Extract sidebar JSX
   - Extract layout container
   - Handle navigation state

2. Create `ui/src/components/monitoring/ProcessCard.tsx` (~100 LOC):
   - Extract single process rendering
   - Extract status color logic
   - Extract metric displays

3. Create `ui/src/hooks/useTelemetryData.ts` (~80 LOC):
   - Extract telemetry merging logic
   - Extract trend calculation
   - Handle live vs historical data merge

4. Refactor App.tsx (~400 LOC):
   - Import AppShell for layout
   - Keep routing logic
   - Keep hierarchy fetching

5. Refactor Monitor.tsx (~350 LOC):
   - Import ProcessCard for rendering
   - Import useTelemetryData hook
   - Keep polling and WebSocket setup

**Acceptance Criteria:**
- [ ] App.tsx ≤400 lines
- [ ] Monitor.tsx ≤400 lines
- [ ] Navigation works correctly
- [ ] Process monitoring works
- [ ] Telemetry updates work
- [ ] `npm run build` succeeds

**Risks:**
- Hook extraction may break reactivity → Test with live data
- Layout changes may shift UI → Visual inspection required

---

### Milestone 4: Domain Organization (Optional)
**Goal:** Organize components by domain for discoverability
**Estimated:** ~50 LOC (barrel exports only)
**Duration:** 30 minutes

**Tasks:**
1. Create domain directories (if not exists):
   - `ui/src/components/approval/`
   - `ui/src/components/messages/`
   - `ui/src/components/monitoring/`

2. Move files to appropriate directories

3. Create barrel exports (`index.ts`) for each domain

4. Update all imports

**Acceptance Criteria:**
- [ ] Clear domain organization
- [ ] All imports work
- [ ] `npm run build` succeeds

**Note:** This milestone is optional if time constrained. The file size reductions are the priority.

---

## Success Metrics

| Metric | Before | After | Target |
|--------|--------|-------|--------|
| Files >800 lines | 4 | 0 | 0 |
| Files >400 lines | 5 | 0 | 0 |
| Largest file | 1,383 | ~400 | ≤400 |
| Duplicated Icons | ~300 lines | 0 | 0 |
| Duplicated formatters | ~50 lines | 0 | 0 |
| `npm run build` | ✅ | ✅ | ✅ |

## Dependencies
- None - this is standalone refactoring work

## Testing Checklist

**After each milestone:**
- [ ] `npm run build` succeeds
- [ ] No TypeScript errors
- [ ] Visual inspection in browser:
  - [ ] Messages render correctly
  - [ ] Approvals work
  - [ ] Monitor shows data
  - [ ] Navigation works
  - [ ] WebSocket updates work

## Open Questions
- Should we add component tests after refactor? (Deferred to future work)
- Should we adopt a state management library? (Out of scope for this sprint)

## Notes
- This is a pure refactor - no new features
- Prioritize file size reduction over perfect organization
- If time runs short, skip Milestone 4 (domain organization)
- CSS modules provide automatic scoping - no specificity conflicts expected
- Visual inspection is critical - no automated UI tests exist

---

**Sprint created**: 2025-12-30
**Design doc**: [m-ui-refactor-ai-friendly.md](m-ui-refactor-ai-friendly.md)
