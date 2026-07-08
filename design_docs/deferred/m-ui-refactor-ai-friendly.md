# M-UI-REFACTOR: Refactor UI Folder for AI-Friendly File Sizes

**Status**: Planned
**Target**: v0.6.2
**Priority**: P1 (Medium-High)
**Estimated**: 8-12 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a **developer tooling refactor** (React frontend), not a language feature. Most axioms are neutral as this doesn't affect AILANG semantics.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to runtime behavior |
| A2: Replayability | 0 | No change to traces |
| A3: Effect Legibility | 0 | UI code, no AILANG effects |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | **+1** | **Primary goal: reduce context window usage for AI development** |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost model changes |
| A10: Composability | **+1** | Smaller, composable components improve reuse |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): **Explicitly optimizing for machine (AI) analysis**

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| **≥ +2** | **✅ Proceed to implementation** |

## Problem Statement

The `ui/` folder contains 4 critical files that exceed the recommended 400-line limit for AI-maintainable code, causing context window exhaustion when working on the Collaboration Hub.

**Current State:**

| File | Lines | % CSS | Issue |
|------|-------|-------|-------|
| ConversationView.tsx | **1,383** | 56% (778 lines) | 5+ responsibilities, massive inline CSS |
| ApprovalQueue.tsx | **1,073** | 64% (691 lines) | Duplicate approval UI, massive inline CSS |
| Monitor.tsx | **999** | 48% (476 lines) | Complex state, telemetry merging |
| App.tsx | **949** | 47% (446 lines) | Kitchen sink component |
| **Total critical files** | **4,404** | | 48% of entire ui/ codebase |

**Additional issues:**
- Icons object duplicated in 4+ files (~300 wasted lines)
- `formatTimestamp()` defined 3 times
- Mixed styling approaches (inline `<style>` vs `.module.css`)
- No shared hooks for common patterns

**Impact:**
- AI assistants fill context window reading single files
- Edits to one concern (e.g., approval UI) touch 1000+ line files
- Difficult to understand component responsibilities
- Code duplication leads to inconsistent behavior

## Goals

**Primary Goal:** Split all files to ≤400 lines and extract ~2,400 lines of CSS to separate files.

**Success Metrics:**
- [ ] 0 files over 800 lines (currently: 4)
- [ ] 0 files over 400 lines (currently: 5)
- [ ] CSS extracted to `.module.css` files (~2,400 lines moved)
- [ ] Icons consolidated to single file (~300 lines deduplicated)
- [ ] Formatters consolidated to single utility (~50 lines deduplicated)
- [ ] All existing functionality preserved (no regressions)

## Solution Design

### Overview

Split monolithic components into focused, single-responsibility modules following the existing patterns (TaskExecution/, MetricsCard.module.css). Extract all inline CSS to `.module.css` files. Consolidate duplicated utilities.

### Architecture

**Target structure:**
```
ui/src/
├── components/
│   ├── common/                    # NEW: Shared UI primitives
│   │   ├── Icons.tsx              # NEW: Consolidated icons
│   │   └── index.ts
│   ├── approval/                  # NEW: Approval domain
│   │   ├── ApprovalQueue.tsx      # SHRUNK: <300 lines
│   │   ├── ApprovalCard.tsx       # NEW: Single approval item
│   │   ├── ApprovalHistory.tsx    # EXISTS: Already small
│   │   ├── ApprovalQueue.module.css  # NEW: Extracted CSS
│   │   └── index.ts
│   ├── messages/                  # NEW: Message domain
│   │   ├── ConversationView.tsx   # SHRUNK: <400 lines
│   │   ├── MessageList.tsx        # NEW: Message rendering
│   │   ├── MessageInput.tsx       # NEW: Input area
│   │   ├── ApprovalPanel.tsx      # NEW: Inline approval UI
│   │   ├── ConversationView.module.css  # NEW: Extracted CSS
│   │   └── index.ts
│   ├── monitoring/                # NEW: Monitor domain
│   │   ├── Monitor.tsx            # SHRUNK: <400 lines
│   │   ├── ProcessCard.tsx        # NEW: Single process item
│   │   ├── Monitor.module.css     # NEW: Extracted CSS
│   │   └── index.ts
│   ├── MessageCenter/             # EXISTS: Keep structure
│   └── TaskExecution/             # EXISTS: Good example
├── hooks/                         # EXISTS
│   ├── useWebSocket.ts
│   ├── useTaskStream.ts
│   ├── useTelemetryData.ts        # NEW: Extract from Monitor
│   └── index.ts
├── utils/                         # NEW: Shared utilities
│   ├── formatters.ts              # NEW: Consolidated formatters
│   └── index.ts
├── App.tsx                        # SHRUNK: <400 lines
├── AppShell.tsx                   # NEW: Layout component
└── App.module.css                 # NEW: Extracted CSS
```

**Components to create:**

1. **Icons.tsx** - Single source of truth for all SVG icons
2. **formatters.ts** - `formatTimestamp()`, `formatDuration()`, `formatCost()`, `formatTokens()`
3. **ApprovalCard.tsx** - Single approval item rendering
4. **MessageList.tsx** - Message rendering with truncation
5. **MessageInput.tsx** - Input area component
6. **ApprovalPanel.tsx** - Inline approval UI (from ConversationView)
7. **ProcessCard.tsx** - Single process item (from Monitor)
8. **AppShell.tsx** - Header + sidebar layout
9. **useTelemetryData.ts** - Telemetry merging logic

### Implementation Plan

**Phase 1: Extract CSS and Utilities** (~3 hours)
- [ ] Create `utils/formatters.ts` with consolidated formatters
- [ ] Create `components/common/Icons.tsx` with all icons
- [ ] Create `App.module.css` from App.tsx inline styles
- [ ] Create `approval/ApprovalQueue.module.css` from inline styles
- [ ] Create `messages/ConversationView.module.css` from inline styles
- [ ] Create `monitoring/Monitor.module.css` from inline styles
- [ ] Update all imports to use new utilities

**Phase 2: Split ConversationView** (~3 hours)
- [ ] Extract `MessageList.tsx` (message rendering + truncation)
- [ ] Extract `MessageInput.tsx` (input area + message kind selector)
- [ ] Extract `ApprovalPanel.tsx` (approval request handling)
- [ ] Update `ConversationView.tsx` to compose sub-components
- [ ] Verify all functionality preserved

**Phase 3: Split App and Monitor** (~3 hours)
- [ ] Extract `AppShell.tsx` (header + sidebar layout)
- [ ] Extract `ProcessCard.tsx` from Monitor
- [ ] Extract `useTelemetryData.ts` hook
- [ ] Update parent components to use new modules
- [ ] Verify all functionality preserved

**Phase 4: Domain Organization** (~2 hours)
- [ ] Create `approval/` directory and move files
- [ ] Create `messages/` directory and move files
- [ ] Create `monitoring/` directory and move files
- [ ] Update all imports throughout codebase
- [ ] Add barrel exports (`index.ts`)

### Files to Modify/Create

**New files:**
- `ui/src/utils/formatters.ts` - Consolidated formatters (~50 LOC)
- `ui/src/components/common/Icons.tsx` - Consolidated icons (~120 LOC)
- `ui/src/components/messages/MessageList.tsx` - Message rendering (~150 LOC)
- `ui/src/components/messages/MessageInput.tsx` - Input component (~100 LOC)
- `ui/src/components/messages/ApprovalPanel.tsx` - Approval UI (~150 LOC)
- `ui/src/components/monitoring/ProcessCard.tsx` - Process item (~100 LOC)
- `ui/src/hooks/useTelemetryData.ts` - Telemetry hook (~80 LOC)
- `ui/src/AppShell.tsx` - Layout component (~100 LOC)
- `ui/src/App.module.css` - App styles (~450 LOC)
- `ui/src/components/approval/ApprovalQueue.module.css` - Queue styles (~700 LOC)
- `ui/src/components/messages/ConversationView.module.css` - Conv styles (~780 LOC)
- `ui/src/components/monitoring/Monitor.module.css` - Monitor styles (~480 LOC)

**Modified files:**
- `ui/src/App.tsx` - Remove CSS, extract layout (~400 LOC after)
- `ui/src/components/ApprovalQueue.tsx` - Remove CSS, extract card (~300 LOC after)
- `ui/src/components/MessageCenter/ConversationView.tsx` - Remove CSS, extract components (~350 LOC after)
- `ui/src/components/Monitor.tsx` - Remove CSS, extract card/hook (~350 LOC after)

## Examples

### Example 1: ConversationView Split

**Before (1,383 lines):**
```tsx
// ConversationView.tsx - 1,383 lines
export function ConversationView(...) {
  // 80 lines: Icons object (duplicated)
  // 150 lines: truncation logic
  // 200 lines: approval handling
  // 100 lines: message rendering
  // 75 lines: input area
  // 778 lines: inline CSS <style>
}
```

**After (350 lines + 4 new files):**
```tsx
// ConversationView.tsx - 350 lines
import { Icons } from '../common/Icons';
import { MessageList } from './MessageList';
import { MessageInput } from './MessageInput';
import { ApprovalPanel } from './ApprovalPanel';
import styles from './ConversationView.module.css';

export function ConversationView(...) {
  return (
    <div className={styles.container}>
      <MessageList messages={messages} ... />
      <ApprovalPanel approvals={pending} ... />
      <MessageInput onSend={handleSend} ... />
    </div>
  );
}
```

### Example 2: Icons Consolidation

**Before (duplicated 4x = ~300 lines):**
```tsx
// ConversationView.tsx
const Icons = { Send: () => <svg>...</svg>, ... };

// ApprovalQueue.tsx
const Icons = { Check: () => <svg>...</svg>, ... };

// Monitor.tsx
const Icons = { Activity: () => <svg>...</svg>, ... };
```

**After (single file ~120 lines):**
```tsx
// components/common/Icons.tsx
export const Icons = {
  Send: () => <svg>...</svg>,
  Check: () => <svg>...</svg>,
  Activity: () => <svg>...</svg>,
  // ... all icons in one place
};

// Usage everywhere:
import { Icons } from '../common/Icons';
```

## Success Criteria

- [ ] All 4 critical files under 400 lines
- [ ] No inline `<style>` tags in any component over 200 lines
- [ ] Icons consolidated to single source (~300 lines saved)
- [ ] Formatters consolidated (~50 lines saved)
- [ ] All existing UI functionality preserved
- [ ] `npm run build` succeeds
- [ ] Visual inspection shows no regressions
- [ ] All tests passing (if any exist)

## Testing Strategy

**Visual testing:**
- [ ] Conversation view renders messages correctly
- [ ] Approval queue shows pending/history correctly
- [ ] Monitor shows processes and telemetry correctly
- [ ] App navigation and layout unchanged

**Functional testing:**
- [ ] Message sending works
- [ ] Approval/rejection works
- [ ] WebSocket updates work
- [ ] All existing features work

**Build verification:**
- [ ] `npm run build` succeeds
- [ ] No TypeScript errors
- [ ] No console errors in browser

## Non-Goals

**Not in this feature:**
- Adding new UI functionality - Pure refactor, no new features
- Rewriting in different framework - Keep React/TypeScript
- Adding comprehensive tests - Just preserve existing behavior
- Changing visual design - Only code organization

## Timeline

**Day 1** (~4 hours):
- Phase 1: Extract CSS and utilities

**Day 2** (~4 hours):
- Phase 2: Split ConversationView

**Day 3** (~4 hours):
- Phase 3: Split App and Monitor
- Phase 4: Domain organization
- Verification and cleanup

**Total: ~11 hours across 3 sessions**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking imports | High | Run `npm run build` after each phase |
| CSS specificity issues | Medium | Use CSS modules which scope styles |
| Missing functionality | High | Visual check each component after split |
| Git conflicts | Medium | Work on single branch, commit often |

## Related Documents

**Implemented (inform patterns):**
- TaskExecution/ folder - Good example of split components
- MetricsCard.module.css - Good example of CSS modules

**Project guidelines:**
- [CLAUDE.md](../../CLAUDE.md) - File size targets (200-500 lines ideal, 800+ critical)
- [.claude/skills/codebase-organizer/](../../.claude/skills/codebase-organizer/) - Refactoring patterns

## References

- CLAUDE.md file organization principles
- React component composition patterns
- CSS Modules documentation

## Future Work

- Add Storybook for component documentation
- Add unit tests for utilities and hooks
- Consider state management library if props drilling persists
- Component library extraction if patterns stabilize

---

**Document created**: 2025-12-30
**Last updated**: 2025-12-30
