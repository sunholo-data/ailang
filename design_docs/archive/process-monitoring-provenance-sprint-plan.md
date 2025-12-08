# Sprint Plan: Process Monitoring & Provenance

**Design Doc**: [process-monitoring-provenance.md](process-monitoring-provenance.md)
**Sprint Duration**: 5 milestones
**Total Effort**: ~40 hours

## Sprint Overview

This sprint implements unified process monitoring with provenance tracking, enabling visibility into WHO started each process and HOW processes relate to each other in a hierarchy.

```
┌─────────────────────────────────────────────────────────────┐
│  M1: Process Registry        │  M2: Hierarchy          │   │
│  (10h) - Database +          │  (8h) - Parent-child    │   │
│  registration + detection    │  tracking + tree API    │   │
├─────────────────────────────────────────────────────────────┤
│  M3: Resource Monitor        │  M4: Unified UI         │   │
│  (6h) - CPU/RAM polling      │  (10h) - Tree view +    │   │
│  + history                   │  badges + WebSocket     │   │
├─────────────────────────────────────────────────────────────┤
│  M5: Claude Code Integration (6h)                           │
│  - Hooks + session linking + documentation                  │
└─────────────────────────────────────────────────────────────┘
```

## Milestones

### M1: Process Registry & Provenance (10 hours)

**Goal**: Track all processes with their initiators

**Day 1 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Add processes table schema | `internal/messaging/store.go` | +40 | Migration test |
| Create ProcessInfo struct | `internal/messaging/processes.go` | +80 | - |
| Implement RegisterProcess() | `internal/messaging/processes.go` | +60 | Unit test |
| Implement GetProcess() | `internal/messaging/processes.go` | +40 | Unit test |
| Implement ListProcesses() | `internal/messaging/processes.go` | +50 | Unit test |

**Day 2 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Add /api/processes/register endpoint | `internal/server/process_handlers.go` | +80 | Integration test |
| Add /api/processes list endpoint | `internal/server/process_handlers.go` | +50 | Integration test |
| Implement initiator detection | `cmd/ailang/run.go` | +60 | Manual test |
| Add auto-registration to ailang run | `cmd/ailang/run.go` | +40 | Manual test |
| Add AILANG_EVAL_ID support to eval harness | `internal/eval_harness/runner.go` | +30 | Unit test |

**Acceptance Criteria**:
- [ ] Process registration creates database record
- [ ] Initiator type detected from environment variables
- [ ] `/api/processes` returns all registered processes
- [ ] Eval suite processes show `initiator_type: eval-suite`

---

### M2: Process Hierarchy & Parent-Child (8 hours)

**Goal**: Track parent-child relationships between processes

**Day 3 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Add parent_process_id to schema | `internal/messaging/store.go` | +10 | Migration test |
| Implement GetProcessTree() | `internal/messaging/processes.go` | +80 | Unit test |
| Implement GetProcessAncestors() | `internal/messaging/processes.go` | +40 | Unit test |
| Implement GetProcessDescendants() | `internal/messaging/processes.go` | +40 | Unit test |
| Add AILANG_PARENT_PROCESS env var propagation | `internal/eval_harness/runner.go` | +20 | Manual test |

**Day 4 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Add /api/processes/tree endpoint | `internal/server/process_handlers.go` | +60 | Integration test |
| Add /api/processes/{id}/children | `internal/server/process_handlers.go` | +40 | Integration test |
| Add /api/processes/{id}/ancestors | `internal/server/process_handlers.go` | +40 | Integration test |
| Handle orphaned processes | `internal/messaging/processes.go` | +30 | Unit test |
| Add tree depth limit (max 10) | `internal/messaging/processes.go` | +20 | Unit test |

**Acceptance Criteria**:
- [ ] Parent-child relationships stored correctly
- [ ] Process tree API returns nested structure
- [ ] Eval suite benchmarks show as children of parent eval process
- [ ] Orphaned process handling (parent exits first)

---

### M3: Resource Monitoring Integration (6 hours)

**Goal**: Show CPU/RAM in hierarchy view

**Day 5 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Create ResourceMonitor worker | `internal/server/resource_monitor.go` | +100 | Unit test |
| Implement getProcessResourceUsage() | `internal/server/resource_monitor.go` | +60 | Manual test |
| Add periodic resource snapshot storage | `internal/messaging/processes.go` | +40 | Unit test |
| Include resources in process listing | `internal/server/process_handlers.go` | +20 | Integration test |
| Handle terminated processes | `internal/server/resource_monitor.go` | +30 | Unit test |

**Day 6 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Add resource history (last 5 min) | `internal/messaging/processes.go` | +50 | Unit test |
| Include resources in /api/hierarchy | `internal/server/hierarchy.go` | +30 | Integration test |
| Add resource thresholds config | `internal/server/resource_monitor.go` | +20 | - |
| Tests for resource polling | `internal/server/resource_monitor_test.go` | +80 | - |

**Acceptance Criteria**:
- [ ] CPU/RAM updated every 2 seconds for running processes
- [ ] `/api/hierarchy` includes resource usage
- [ ] Resource history available for last 5 minutes
- [ ] Terminated processes show final resource values

---

### M4: Unified Hierarchy UI (10 hours)

**Goal**: Display processes nested in hierarchy with resources

**Day 7 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Create UnifiedHierarchy component | `ui/src/components/UnifiedHierarchy/UnifiedHierarchy.tsx` | +200 | - |
| Create UnifiedHierarchy.module.css | `ui/src/components/UnifiedHierarchy/UnifiedHierarchy.module.css` | +100 | - |
| Create ProcessNode component | `ui/src/components/ProcessNode/ProcessNode.tsx` | +100 | - |
| Create ResourceBadge component | `ui/src/components/ResourceBadge/ResourceBadge.tsx` | +80 | - |

**Day 8 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Create ProvenanceBadge component | `ui/src/components/ProvenanceBadge/ProvenanceBadge.tsx` | +60 | - |
| Add expand/collapse for process trees | `ui/src/components/UnifiedHierarchy/UnifiedHierarchy.tsx` | +50 | - |
| Integrate into Monitor tab | `ui/src/components/Monitor/Monitor.tsx` | +40 | - |
| Add WebSocket updates for processes | `ui/src/hooks/useProcessUpdates.ts` | +80 | - |

**Day 9 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Color-coded warnings (high CPU/RAM) | `ui/src/components/ResourceBadge/ResourceBadge.module.css` | +40 | - |
| Add "Kill process" action | `ui/src/components/ProcessNode/ProcessNode.tsx` | +40 | Manual test |
| Add process types to types/index.ts | `ui/src/types/index.ts` | +30 | - |
| Build and test UI | - | - | Manual test |

**Acceptance Criteria**:
- [ ] Process tree displays with proper nesting
- [ ] Initiator icons show (user/claude-code/eval-suite)
- [ ] CPU/RAM badges with color warnings
- [ ] Real-time updates via WebSocket
- [ ] Kill button works for running processes

---

### M5: Claude Code Integration (6 hours)

**Goal**: Link Claude Code sessions to spawned processes

**Day 10 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Document hook configuration | `.claude/hooks/process-tracking.md` | +100 | - |
| Create example pre-exec hook | `.claude/hooks/examples/process-tracking.json` | +30 | - |
| Add CLAUDE_SESSION_ID detection | `cmd/ailang/run.go` | +20 | Manual test |
| Test with actual Claude Code | - | - | Manual test |

**Day 11 Tasks**:

| Task | File | LOC | Test |
|------|------|-----|------|
| Show Claude sessions as top-level nodes | `ui/src/components/UnifiedHierarchy/UnifiedHierarchy.tsx` | +40 | Manual test |
| Link threads to sessions | `internal/messaging/processes.go` | +30 | Unit test |
| Write integration guide | `docs/guides/claude-code-integration.md` | +200 | - |
| Final testing and polish | - | - | E2E test |

**Acceptance Criteria**:
- [ ] Claude Code sessions appear in hierarchy
- [ ] Processes spawned by Claude Code show as children
- [ ] Documentation complete for hook setup
- [ ] E2E test passes

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Process exits before registration | Add retry logic, handle orphans |
| High polling overhead | Configurable intervals, batch updates |
| WebSocket flooding with updates | Throttle to max 1 update/second |
| Cross-platform resource monitoring | Use `ps` command (works on macOS/Linux) |

## Definition of Done

- [ ] All processes show initiator type in UI
- [ ] Parent-child hierarchy visible in tree
- [ ] CPU/RAM updated in real-time
- [ ] Kill action works from UI
- [ ] Claude Code integration documented
- [ ] All tests passing
- [ ] Documentation updated

---

**Sprint created**: 2025-11-30
**Author**: Claude (sprint-planner)
