# M-ARCH2: Daemon God Object Refactor

**Status**: Planned
**Target**: v0.6.5
**Priority**: P1 (Medium-High)
**Estimated**: 20-30 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to determinism |
| A2: Replayability | +1 | Clearer component boundaries improve trace correlation |
| A3: Effect Legibility | +1 | Effects isolated to specific components |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Smaller components enable focused testing |
| A6: Safe Concurrency | +1 | Clearer ownership reduces race condition risk |
| A7: Machines First | +1 | Smaller files, easier for AI to understand/modify |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Components can be used independently |
| A11: Structured Failure | +1 | Each component has clear error handling |
| A12: System Boundary | +1 | Clearer boundaries between subsystems |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Smaller components improve AI maintainability

## Problem Statement

The `Daemon` type in `internal/coordinator/` has grown into a "god object" with 39 methods spread across 5 files totaling ~2,500 lines. This violates single responsibility principle and makes the code difficult to understand, test, and modify.

**Current State:**
- `daemon.go` - Core struct definition
- `daemon_lifecycle.go` - 446 lines, 16 methods
- `daemon_tasks.go` - 892 lines, 9 methods (CRITICAL: >800 lines)
- `daemon_approval.go` - 461 lines, 6 methods
- `daemon_github.go` - Unknown lines, 6 methods

**Responsibilities Mixed in Single Type:**
1. Lifecycle management (Start, Stop, WaitForDone)
2. Task execution and polling
3. Approval checkpoint management
4. GitHub integration and label watching
5. Observatory trace linking
6. Provider selection and routing
7. Worktree management
8. Event broadcasting
9. Agent registry management

**Impact:**
- 892-line file (`daemon_tasks.go`) exceeds 800-line critical threshold
- Hard to test individual responsibilities in isolation
- Race conditions harder to identify with shared state
- New features add more methods to already-bloated type
- AI assistants struggle to understand full context

## Goals

**Primary Goal:** Split Daemon into focused components with single responsibilities, reducing largest file to <500 lines.

**Success Metrics:**
- No file exceeds 500 lines
- Each component has single, testable responsibility
- Total lines reduced by ~20% (eliminate dead code found during refactor)
- All existing tests pass
- Test coverage increases (smaller components = easier to test)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Daemon becomes orchestrator, not owner of state | All 39 methods move to sub-components; reverting means re-merging everything | human | design | high |
| Component communication via method calls (not channels/events) | Determines coupling model between components; channels would be more decoupled but harder to debug | human | design | high |
| Sub-packages under `internal/coordinator/` (lifecycle/, execution/, etc.) | Package layout locks import direction; components cannot import each other without cycles | human | design | high |
| Shared state protected by per-component mutexes (not one global lock) | Eliminates global Daemon.mu but requires careful analysis of which state belongs to which component | human | design | med |
| Delete original daemon_*.go files after migration (not keep as wrappers) | Clean break vs gradual deprecation; affects every caller of Daemon methods | human | design | med |
| Phase-by-phase extraction (lifecycle first, execution second) | Order determines which components are decoupled first; execution is highest-risk | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Define the exact interface each component exposes to the Daemon orchestrator
- [ ] Map every Daemon field to its owning component (especially shared fields like `store` and `registry`)
- [ ] Decide dependency direction: can `execution/` depend on `approval/` or must Daemon mediate?
- [ ] Confirm per-component mutex strategy and identify any cross-component locking needs
- [ ] Resolve whether EventBroadcaster is a dependency of other components or only called by Daemon
- [ ] Audit existing tests to determine which must be rewritten vs adapted

## Solution Design

### Overview

Extract Daemon responsibilities into dedicated components that the Daemon orchestrates. Each component owns its state and exposes a clean interface.

### Architecture

```
internal/coordinator/
├── daemon.go              # Orchestrator only (~150 lines)
├── lifecycle/
│   └── manager.go         # Start, Stop, WaitForDone (~200 lines)
├── execution/
│   ├── executor.go        # Task execution loop (~300 lines)
│   └── provider_router.go # Provider selection (~150 lines)
├── approval/
│   ├── manager.go         # Approval checkpoint logic (~250 lines)
│   └── watcher.go         # GitHub label watching (~200 lines)
├── github/
│   └── integration.go     # Issue sync, PR creation (~200 lines)
├── broadcast/
│   └── broadcaster.go     # Event streaming to dashboard (~150 lines)
└── worktree/
    └── manager.go         # Git worktree lifecycle (~200 lines)
```

**Components:**

1. **LifecycleManager**: Handles daemon start/stop, graceful shutdown, health checks
2. **TaskExecutor**: Polls queue, executes tasks, handles retries
3. **ProviderRouter**: Selects appropriate provider for task type
4. **ApprovalManager**: Creates checkpoints, handles approval/rejection
5. **ApprovalWatcher**: Monitors GitHub labels for approvals
6. **GitHubIntegration**: Syncs issues, creates PRs
7. **EventBroadcaster**: Streams events to WebSocket clients
8. **WorktreeManager**: Creates/cleans git worktrees

### Implementation Plan

**Phase 1: Extract Lifecycle** (~4 hours)
- [ ] Create `lifecycle/manager.go` with LifecycleManager
- [ ] Move Start, Stop, WaitForDone, health check methods
- [ ] Daemon embeds LifecycleManager
- [ ] Tests pass

**Phase 2: Extract Execution** (~6 hours)
- [ ] Create `execution/executor.go` with TaskExecutor
- [ ] Move task polling, execution loop, retry logic
- [ ] Create `execution/provider_router.go` with ProviderRouter
- [ ] Move provider selection logic from daemon_tasks.go
- [ ] Tests pass

**Phase 3: Extract Approval** (~5 hours)
- [ ] Create `approval/manager.go` with ApprovalManager
- [ ] Move checkpoint creation, approval handling
- [ ] Create `approval/watcher.go` with ApprovalWatcher (already partially exists)
- [ ] Tests pass

**Phase 4: Extract GitHub** (~3 hours)
- [ ] Create `github/integration.go` with GitHubIntegration
- [ ] Move issue sync, PR creation logic
- [ ] Tests pass

**Phase 5: Extract Supporting Components** (~4 hours)
- [ ] Create `broadcast/broadcaster.go` with EventBroadcaster
- [ ] Create `worktree/manager.go` with WorktreeManager
- [ ] Move respective logic from daemon files
- [ ] Tests pass

**Phase 6: Cleanup & Documentation** (~4 hours)
- [ ] Remove dead code discovered during refactor
- [ ] Update package documentation
- [ ] Add architecture diagram to README
- [ ] Final test verification

### Files to Modify/Create

**New files:**
- `internal/coordinator/lifecycle/manager.go` (~200 LOC)
- `internal/coordinator/execution/executor.go` (~300 LOC)
- `internal/coordinator/execution/provider_router.go` (~150 LOC)
- `internal/coordinator/approval/manager.go` (~250 LOC)
- `internal/coordinator/approval/watcher.go` (~200 LOC)
- `internal/coordinator/github/integration.go` (~200 LOC)
- `internal/coordinator/broadcast/broadcaster.go` (~150 LOC)
- `internal/coordinator/worktree/manager.go` (~200 LOC)
- Test files for each component (~800 LOC total)

**Modified/Removed files:**
- `internal/coordinator/daemon.go` - Reduce to orchestrator (~150 LOC from current)
- `internal/coordinator/daemon_lifecycle.go` - DELETE (moved to lifecycle/)
- `internal/coordinator/daemon_tasks.go` - DELETE (moved to execution/)
- `internal/coordinator/daemon_approval.go` - DELETE (moved to approval/)
- `internal/coordinator/daemon_github.go` - DELETE (moved to github/)

## Examples

### Example 1: Daemon as Orchestrator

**Before (daemon owns everything):**
```go
type Daemon struct {
    store           Store
    registry        *AgentRegistry
    broadcaster     *HTTPBroadcaster
    providers       []Provider
    // ... 15+ more fields

    mu              sync.RWMutex
    running         bool
    stopCh          chan struct{}
    // ... state management
}

// 39 methods on Daemon type
func (d *Daemon) Start() error { ... }
func (d *Daemon) executeTask(task *Task) error { ... }
func (d *Daemon) createApprovalCheckpoint(...) error { ... }
func (d *Daemon) syncGitHubIssues() error { ... }
// ... 35 more methods
```

**After (daemon orchestrates components):**
```go
type Daemon struct {
    lifecycle  *lifecycle.Manager
    executor   *execution.TaskExecutor
    approval   *approval.Manager
    github     *github.Integration
    broadcast  *broadcast.Broadcaster
    worktree   *worktree.Manager
}

func NewDaemon(cfg *Config) (*Daemon, error) {
    d := &Daemon{
        lifecycle: lifecycle.NewManager(),
        executor:  execution.NewExecutor(cfg.Store, cfg.Providers),
        approval:  approval.NewManager(cfg.Store),
        github:    github.NewIntegration(cfg.GitHubConfig),
        broadcast: broadcast.NewBroadcaster(cfg.WebhookURL),
        worktree:  worktree.NewManager(cfg.WorktreeDir),
    }
    return d, nil
}

func (d *Daemon) Start() error {
    return d.lifecycle.Start(d.runLoop)
}

func (d *Daemon) runLoop(ctx context.Context) error {
    // Orchestrate components
    for {
        task := d.executor.Poll(ctx)
        result := d.executor.Execute(ctx, task)
        d.approval.CreateCheckpoint(ctx, task, result)
        d.broadcast.Send(ctx, result)
    }
}
```

### Example 2: Testing Individual Components

**Before (must test entire daemon):**
```go
func TestApprovalCreation(t *testing.T) {
    // Must set up entire daemon with all dependencies
    daemon := NewDaemon(fullConfig)
    daemon.Start()
    defer daemon.Stop()
    // Test approval through daemon methods
}
```

**After (test component in isolation):**
```go
func TestApprovalCreation(t *testing.T) {
    // Test just the approval manager
    store := NewMockStore()
    mgr := approval.NewManager(store)

    err := mgr.CreateCheckpoint(ctx, task, result)
    assert.NoError(t, err)
    assert.Equal(t, 1, store.CheckpointCount())
}
```

## Success Criteria

- [ ] No coordinator file exceeds 500 lines
- [ ] Daemon struct has <10 fields (component references)
- [ ] Daemon has <10 methods (orchestration only)
- [ ] Each component has dedicated test file
- [ ] All existing integration tests pass
- [ ] Test coverage for coordinator package increases
- [ ] Documentation includes architecture diagram

## Testing Strategy

**Unit tests:**
- Test each component in isolation with mock dependencies
- Test error handling paths in each component
- Test state transitions in LifecycleManager

**Integration tests:**
- Existing daemon tests should pass unchanged
- Add tests for component interactions

**Manual testing:**
- Run coordinator with all components
- Verify dashboard receives events
- Test approval workflow end-to-end

## Deferred Decisions

The following are intentionally left open for the implementer:

- Internal struct field layout within each component — [agent may resolve]
- Whether LifecycleManager exposes health check as HTTP handler or just bool method — [agent may resolve]
- Logging strategy within components (structured logger per-component vs shared) — [agent may resolve]
- Naming convention for component constructors (`NewManager` vs `NewLifecycleManager`) — [agent may resolve]
- Whether to introduce a shared `Config` struct or pass dependencies individually to constructors — [agent may resolve]
- Worktree cleanup strategy (eager vs lazy) within WorktreeManager — [human may resolve]

## Non-Goals

**Not in this feature:**
- Changing coordinator behavior - Purely structural refactor
- Adding new features - Same functionality, better structure
- Changing API - External interface unchanged

## Timeline

**Week 1** (12 hours):
- Phase 1: Extract Lifecycle
- Phase 2: Extract Execution

**Week 2** (10 hours):
- Phase 3: Extract Approval
- Phase 4: Extract GitHub
- Phase 5: Extract Supporting Components

**Week 3** (4 hours):
- Phase 6: Cleanup & Documentation

**Total: ~26 hours across 3 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking daemon behavior | High | Run integration tests after each phase |
| Race conditions in refactored code | High | Add race detector to CI, review mutex usage |
| Import cycles between components | Medium | Design dependency direction upfront |
| Test coverage gaps | Medium | Write component tests before refactoring |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_2/m-coord-stable.md](design_docs/implemented/v0_6_2/m-coord-stable.md) (0.50)

**Planned (check for overlap):**
- [design_docs/planned/v0_7_0/m-coordinator-always-on-daemon.md](design_docs/planned/v0_7_0/m-coordinator-always-on-daemon.md) (0.49)

## References

- [Design Axioms](/docs/references/axioms)
- `internal/coordinator/daemon_tasks.go` - Current largest file (892 lines)
- Effective Go: Embedding - https://go.dev/doc/effective_go#embedding

## Future Work

- Add metrics collection to each component
- Add circuit breakers to TaskExecutor
- Add component health endpoints for monitoring

---

**Document created**: 2026-01-05
**Last updated**: 2026-01-05
