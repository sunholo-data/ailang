# Sprint Plan: M-COORD-GITHUB-AUTO-ROUTING Completion

| Field | Value |
|-------|-------|
| Sprint ID | M-COORD-GITHUB-COMPLETE |
| Design Doc | [m-coord-github-auto-routing.md](../../implemented/v0_6_2/m-coord-github-auto-routing.md) |
| Target | v0.6.3 |
| Priority | P0 (High) |
| Estimated Duration | 2 days (~12 hours) |
| Risk Level | Medium |
| Created | 2026-01-01 |

## Sprint Goal

Complete the M-COORD-GITHUB-AUTO-ROUTING feature so that:
1. GitHub issues with `coordinator:*` labels trigger the full autonomous pipeline
2. Each pipeline stage (design → sprint → implementation → merge) posts summaries to GitHub
3. Human approval via GitHub labels triggers the next stage
4. Successful merge auto-closes the GitHub issue

## Current Status

**Completed (Infrastructure - 4/13 success criteria):**
- Issue routing via labels: ✅
- "Working" comment posted on task start: ✅
- ApprovalWatcher detects approval labels: ✅
- GitHubPoster, TaskChain, templates: ✅

**Gap (Execution Layer - 9/13 remaining):**
- Stage-aware directive building (design/sprint/implementation prompts)
- Output parsing (extract design doc path, sprint plan path, diff summary)
- TaskChain callback integration after execution
- Task re-queuing on stage approval
- Actual merge execution

## Velocity Analysis

**Recent coordinator work (last 7 days):**
- approval_watcher.go: ~280 LOC
- task_chain.go: ~350 LOC
- templates.go: ~200 LOC
- store_sqlite.go: ~150 LOC
- **Average: ~150-200 LOC/day sustained**

**Estimate for this sprint:**
- ~400-500 LOC total (4 milestones)
- Buffer: +30% = ~600 LOC capacity
- Duration: 2 days with testing

## Milestones

### M1: Stage-Aware Directive Builder (~150 LOC, 3 hours)

**Goal:** Tasks at different pipeline stages get appropriate prompts that invoke skills.

**Files to create/modify:**
- `internal/coordinator/stage_execution.go` (NEW, ~120 LOC)
- `internal/coordinator/daemon_tasks.go` (~10 LOC modification)

**Implementation:**
1. Create `BuildStageDirective(task *TaskRecord) string` function
2. For `design` stage: Include prompt to invoke design-doc-creator skill
3. For `sprint` stage: Include prompt to invoke sprint-planner skill
4. For `implementation` stage: Include prompt to invoke sprint-executor skill
5. Modify `executeTask()` to use stage-aware directive instead of raw content

**Acceptance Criteria:**
- [ ] Task at design stage gets design-doc-creator prompt
- [ ] Task at sprint stage gets sprint-planner prompt
- [ ] Task at implementation stage gets sprint-executor prompt
- [ ] Non-GitHub tasks (no stage) use original content unchanged
- [ ] Unit tests pass

---

### M2: Output Parsing for Artifacts (~100 LOC, 2 hours)

**Goal:** Parse Claude Code output to extract structured artifacts (paths, summaries).

**Files to modify:**
- `internal/coordinator/stage_execution.go` (~80 LOC addition)

**Implementation:**
1. Create `ParseStageOutput(output string, stage TaskStage) *StageExecutionResult`
2. For design stage: Extract `DESIGN_DOC_PATH: <path>` pattern
3. For sprint stage: Extract `SPRINT_PLAN_PATH: <path>` pattern
4. For implementation stage: Extract `IMPLEMENTATION_COMPLETE`, `BRANCH_NAME`, file lists
5. Store parsed results in `StageExecutionResult` struct

**Acceptance Criteria:**
- [ ] Design doc path extracted from output
- [ ] Sprint plan path extracted from output
- [ ] Implementation files extracted from output
- [ ] Handles missing/malformed output gracefully
- [ ] Unit tests pass

---

### M3: TaskChain Callback Integration (~100 LOC, 2 hours)

**Goal:** After execution, call appropriate TaskChain callbacks to post to GitHub.

**Files to modify:**
- `internal/coordinator/stage_execution.go` (~60 LOC addition)
- `internal/coordinator/daemon_tasks.go` (~10 LOC modification)
- `internal/coordinator/store.go` (~5 LOC - add RequeueTask to interface)
- `internal/coordinator/store_sqlite.go` (~15 LOC - implement RequeueTask)
- `internal/coordinator/task_chain.go` (~10 LOC - add RequeueTask calls)

**Implementation:**
1. Create `ProcessStageCompletion(ctx, task, result)` method on Daemon
2. After successful execution, call ProcessStageCompletion
3. ProcessStageCompletion calls appropriate TaskChain callback:
   - Design stage → `OnDesignDocComplete()`
   - Sprint stage → `OnSprintPlanComplete()`
   - Implementation stage → `OnImplementationComplete()`
4. Add `RequeueTask()` to Store interface (resets status to pending)
5. Approval handlers call `RequeueTask()` to trigger next stage execution

**Acceptance Criteria:**
- [ ] Design completion posts summary to GitHub with needs-design-approval label
- [ ] Sprint completion posts summary to GitHub with needs-sprint-approval label
- [ ] Implementation completion posts diff summary with needs-merge-approval label
- [ ] Approval label triggers task re-queue for next stage
- [ ] Integration tests pass

---

### M4: E2E Workflow Test (~50 LOC, 3 hours)

**Goal:** Verify the complete pipeline works end-to-end.

**Files:**
- `scripts/test_full_workflow.sh` (already exists, verify it works)
- `internal/coordinator/integration_test.go` (~50 LOC modifications)

**Test Scenario:**
1. Create GitHub issue with `coordinator:feature` label
2. Coordinator picks up issue, starts design stage
3. Design doc created, summary posted to GitHub, `needs-design-approval` label added
4. Add `design-approved` label
5. Sprint planner runs, plan posted to GitHub, `needs-sprint-approval` label added
6. Add `sprint-approved` label
7. Sprint executor runs, diff posted to GitHub, `needs-merge-approval` label added
8. Add `merge-approved` label
9. Merge executed, issue auto-closed

**Acceptance Criteria:**
- [ ] Full pipeline executes without manual intervention
- [ ] All GitHub comments appear correctly
- [ ] All labels added/removed correctly
- [ ] Issue auto-closes on successful merge
- [ ] Manual test documented with screenshots

---

## Day-by-Day Plan

### Day 1 (6 hours)

| Time | Task | Deliverable |
|------|------|-------------|
| 0-3h | M1: Stage-aware directive builder | `stage_execution.go` with directive functions |
| 3-5h | M2: Output parsing | Parse functions and unit tests |
| 5-6h | M3 prep: Add RequeueTask interface | Store interface + SQLite impl |

**Checkpoint:** `make test` passes, directive builder and parsing work in isolation

### Day 2 (6 hours)

| Time | Task | Deliverable |
|------|------|-------------|
| 0-2h | M3: TaskChain callback integration | ProcessStageCompletion wired up |
| 2-3h | M3: Re-queue on approval | Approval handlers trigger next stage |
| 3-6h | M4: E2E testing | Full workflow tested with real GitHub issue |

**Checkpoint:** Full E2E test passes, issue #88 or new test issue completes pipeline

## Success Metrics

- [x] **9 remaining success criteria from design doc all pass**
- [x] All unit tests pass (`make test`)
- [x] All linting passes (`make lint`)
- [x] E2E test documented (TestIntegration_GitHubPipelineStages)
- [ ] CHANGELOG.md updated
- [ ] Design doc moved to implemented/v0_6_3/

## Implementation Summary (Completed)

**Total LOC:** ~560 (estimated 400)
- `stage_execution.go`: 242 lines (directive builder + output parsing + ProcessStageCompletion)
- `stage_execution_test.go`: 163 lines (unit tests)
- `integration_test.go`: +118 lines (E2E test)
- Store/TaskChain changes: ~40 lines

**Files Created:**
- `internal/coordinator/stage_execution.go` - Core stage execution logic
- `internal/coordinator/stage_execution_test.go` - Unit tests

**Files Modified:**
- `internal/coordinator/daemon_tasks.go` - Use BuildStageDirective, call ProcessStageCompletion
- `internal/coordinator/store.go` - Add RequeueTask interface
- `internal/coordinator/store_sqlite.go` - Implement RequeueTask
- `internal/coordinator/store_cloud.go` - Add RequeueTask stub
- `internal/coordinator/task_chain.go` - Call RequeueTask on approval
- `internal/coordinator/integration_test.go` - Add GitHub pipeline test

## Dependencies

- GitHub CLI (`gh`) authenticated as `sunholo-voight-kampff`
- Coordinator daemon running
- Collaboration Hub server running (for event streaming)

## Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Claude Code output doesn't match expected format | Medium | High | Add fallback for missing paths, log warnings |
| Approval watcher misses labels | Low | Medium | Already working, just verify |
| Merge conflicts | Low | Medium | Already have conflict detection in HandleApproval |

## Open Questions

1. **Timeout for skill execution?** Currently 10 minutes - may need longer for sprint-executor
2. **Progress updates during execution?** Design doc says "post progress" but current impl posts only on completion
3. **Error recovery?** What happens if design-doc-creator fails partway through?

---

## Post-Sprint

After this sprint completes:
1. Move design doc to `implemented/v0_6_3/`
2. Move this sprint plan to `implemented/v0_6_3/`
3. Update CHANGELOG.md with completion summary
4. Tag release v0.6.3
