# M-COORD-GITHUB-AUTO-ROUTING Sprint Plan

| Field | Value |
|-------|-------|
| Sprint ID | M-COORD-GITHUB-AUTO-ROUTING |
| Design Doc | [m-coord-github-auto-routing.md](m-coord-github-auto-routing.md) |
| Target | v0.6.3 |
| Estimated Duration | 3-4 days (~24 hours) |
| Risk Level | Medium |

## Sprint Summary

**Goal:** Enable GitHub-centric autonomous workflow where issues with `coordinator:*` labels auto-route through design-doc → sprint-planner → sprint-executor pipeline with GitHub comments for status updates and label-based approval gates.

**Key Deliverables:**
1. Label-based routing in `import-github` command
2. GitHub comment posting capability for coordinator
3. Approval detection via label polling
4. Task chaining (design-doc → sprint → execute)
5. Auto-close issues on successful merge

## Current State Analysis

### What Already Exists

| Component | Status | Location |
|-----------|--------|----------|
| GitHub sync daemon | ✅ Complete | `internal/coordinator/daemon_github.go` |
| Issue import | ✅ Complete | `cmd/ailang/messages_github.go` |
| GitHub client | ✅ Complete | `internal/messaging/github_client.go` |
| Coordinator task lifecycle | ✅ Complete | `internal/coordinator/daemon_tasks.go` |
| Worktree management | ✅ Complete | `internal/coordinator/worktree.go` |
| Merge workflow | ✅ Complete | `internal/coordinator/merge.go` |
| Approval checkpoint | ✅ Complete | `internal/coordinator/approval_checkpoint.go` |

### What Needs Building

| Component | Status | Est. LOC |
|-----------|--------|----------|
| Label-based routing | ❌ Not started | ~50 |
| GitHub comment posting | ❌ Not started | ~150 |
| Approval watcher | ❌ Not started | ~200 |
| Task chaining | ❌ Not started | ~250 |
| Comment templates | ❌ Not started | ~100 |
| Database schema updates | ❌ Not started | ~30 |
| Tests | ❌ Not started | ~300 |

**Total estimated LOC:** ~1,080

## Milestones

### M1: Label-Based Routing (~2 hours)

**Description:** Extend `import-github` to route issues with `coordinator:*` labels directly to the coordinator inbox.

**Tasks:**
- [ ] Add label routing logic to `messages_github.go`
- [ ] Support multiple routing labels: `coordinator:bug`, `coordinator:feature`, `coordinator:docs`, `coordinator:research`
- [ ] Add `--route-by-label` flag (enabled by default)
- [ ] Update daemon GitHub sync to use new routing
- [ ] Add tests for label routing

**Files:**
- `cmd/ailang/messages_github.go` - Add routing logic (~40 LOC)
- `internal/coordinator/daemon_github.go` - Wire up label routing (~10 LOC)
- `cmd/ailang/messages_github_test.go` - Tests (~50 LOC)

**Acceptance Criteria:**
- [ ] Issue with `coordinator:bug` label routes to `coordinator` inbox
- [ ] Issue without `coordinator:*` label routes to `user` inbox (default)
- [ ] Labels are configurable via `~/.ailang/config.yaml`
- [ ] Tests pass for all label routing scenarios

---

### M2: GitHub Comment Posting (~3 hours)

**Description:** Add ability for coordinator to post comments back to GitHub issues.

**Tasks:**
- [ ] Create `github_poster.go` with comment posting, label management, issue closing
- [ ] Integrate with GitHub client authentication
- [ ] Add error handling for API rate limits
- [ ] Add retry logic for transient failures
- [ ] Add tests with mock GitHub API

**Files:**
- `internal/coordinator/github_poster.go` - GitHub API wrapper (~150 LOC)
- `internal/coordinator/github_poster_test.go` - Tests (~100 LOC)

**Acceptance Criteria:**
- [ ] Can post comment to GitHub issue by number
- [ ] Can add/remove labels from issue
- [ ] Can close issue
- [ ] Handles API errors gracefully
- [ ] Respects rate limits

---

### M3: Approval Watcher (~3 hours)

**Description:** Poll GitHub for approval labels and trigger next pipeline stage.

**Tasks:**
- [ ] Create `approval_watcher.go` with polling loop
- [ ] Check for `design-approved`, `sprint-approved`, `merge-approved` labels
- [ ] Trigger appropriate handler on label detection
- [ ] Handle `needs-revision` label (pause pipeline)
- [ ] Add configurable poll interval
- [ ] Add tests

**Files:**
- `internal/coordinator/approval_watcher.go` - Label polling (~200 LOC)
- `internal/coordinator/approval_watcher_test.go` - Tests (~100 LOC)

**Acceptance Criteria:**
- [ ] Detects `design-approved` label and triggers sprint planning
- [ ] Detects `sprint-approved` label and triggers execution
- [ ] Detects `merge-approved` label and triggers merge
- [ ] Handles `needs-revision` by pausing pipeline
- [ ] Poll interval is configurable (default 60s)

---

### M4: Database Schema Updates (~1 hour)

**Description:** Add fields to tasks table for GitHub integration and task chaining.

**Tasks:**
- [ ] Add `github_issue` column (integer, nullable)
- [ ] Add `stage` column (text: 'design', 'sprint', 'implementation')
- [ ] Add migration for existing databases
- [ ] Update Store interface and SQLite implementation
- [ ] Add tests for schema changes

**Files:**
- `internal/coordinator/store.go` - Update interface (~10 LOC)
- `internal/coordinator/store_sqlite.go` - Add columns, migration (~50 LOC)
- `internal/coordinator/store_sqlite_test.go` - Tests (~50 LOC)

**Acceptance Criteria:**
- [ ] Tasks can store associated GitHub issue number
- [ ] Tasks track their current stage
- [ ] Migration runs automatically on startup
- [ ] Existing data preserved after migration

---

### M5: Task Chaining (~4 hours)

**Description:** Implement the pipeline: design-doc → sprint-planner → sprint-executor.

**Tasks:**
- [ ] Create `task_chain.go` with stage transition handlers
- [ ] Implement `OnDesignDocComplete` - posts summary, requests approval
- [ ] Implement `OnSprintPlanComplete` - posts summary, requests approval
- [ ] Implement `OnImplementationComplete` - posts diff summary, requests approval
- [ ] Implement `OnMergeComplete` - closes issue, posts summary
- [ ] Wire chain into daemon task lifecycle
- [ ] Add tests for all transitions

**Files:**
- `internal/coordinator/task_chain.go` - Chaining logic (~250 LOC)
- `internal/coordinator/task_chain_test.go` - Tests (~150 LOC)
- `internal/coordinator/daemon_tasks.go` - Wire chain (~30 LOC)

**Acceptance Criteria:**
- [ ] Design doc completion triggers GitHub comment + label
- [ ] Sprint plan completion triggers GitHub comment + label
- [ ] Implementation completion triggers GitHub comment + label
- [ ] Merge completion closes issue with summary
- [ ] Each stage waits for human approval before proceeding

---

### M6: Comment Templates (~2 hours)

**Description:** Create templates for GitHub comments at each stage.

**Tasks:**
- [ ] Create template for "working" status
- [ ] Create template for design doc summary
- [ ] Create template for sprint plan summary
- [ ] Create template for implementation diff summary
- [ ] Create template for completion/closure
- [ ] Add template rendering with task data
- [ ] Add tests for template rendering

**Files:**
- `internal/coordinator/templates.go` - Template definitions and rendering (~100 LOC)
- `internal/coordinator/templates_test.go` - Tests (~50 LOC)

**Acceptance Criteria:**
- [ ] Templates produce well-formatted GitHub markdown
- [ ] Templates include relevant task data (cost, duration, etc.)
- [ ] Templates render correctly for all stages
- [ ] Error handling for missing data

---

### M7: Integration & Documentation (~3 hours)

**Description:** Wire everything together, add configuration, update documentation.

**Tasks:**
- [ ] Update coordinator config in `config.yaml` schema
- [ ] Add GitHub auto-routing configuration options
- [ ] Wire approval watcher into daemon startup
- [ ] Add end-to-end integration test
- [ ] Update coordinator documentation
- [ ] Update CHANGELOG.md

**Files:**
- `internal/coordinator/daemon.go` - Wire components (~50 LOC)
- `internal/coordinator/integration_test.go` - E2E test (~150 LOC)
- `docs/docs/guides/coordinator.md` - Documentation (~100 lines)
- `CHANGELOG.md` - Update

**Acceptance Criteria:**
- [ ] Full workflow works end-to-end with real GitHub issue
- [ ] Configuration documented
- [ ] All tests pass
- [ ] Documentation updated

---

## Day-by-Day Breakdown

### Day 1 (~8 hours)
- **M1: Label-Based Routing** (2h)
- **M2: GitHub Comment Posting** (3h)
- **M4: Database Schema Updates** (1h)
- Buffer/review (2h)

### Day 2 (~8 hours)
- **M3: Approval Watcher** (3h)
- **M5: Task Chaining** (4h)
- Buffer/testing (1h)

### Day 3 (~8 hours)
- **M5: Task Chaining** - completion (if needed)
- **M6: Comment Templates** (2h)
- **M7: Integration & Documentation** (3h)
- Final testing and polish (3h)

## Success Metrics

- [ ] Issues with `coordinator:*` labels auto-route to coordinator
- [ ] Agents post status updates to GitHub issue threads
- [ ] Human approval via GitHub labels triggers pipeline stages
- [ ] Full pipeline: issue → design → sprint → implement → merge → close
- [ ] All 847+ existing tests still pass
- [ ] New tests for all components (target: 300+ LOC of tests)
- [ ] Documentation updated with user workflow guide

## Dependencies

- GitHub CLI (`gh`) must be authenticated
- `~/.ailang/config.yaml` must have valid GitHub configuration
- Coordinator daemon must be running

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| GitHub API rate limits | High | Add rate limit handling, use conditional requests |
| Label polling latency | Low | Configurable interval, webhook support in future |
| Complex state machine | Medium | Clear stage transitions, comprehensive tests |
| Existing tests break | Medium | Run tests after each milestone |

## Open Questions

1. Should we support GitHub webhooks for instant response? (Future enhancement)
2. Should rejected tasks be retryable? (Yes - keep worktree, allow re-triggering)
3. Maximum comment length for large diffs? (Truncate with link to full diff)

---

**Created:** 2025-12-31
**Last Updated:** 2025-12-31
