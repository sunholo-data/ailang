# Sprint Plan: M-CLOUD-DISPATCH

## Summary
Close the last mile for end-to-end cloud task execution: add a `CloudDispatcher` interface with Cloud Run Jobs implementation, wire it into the coordinator's dispatch loop, and fix cloud logging visibility.

**Duration:** 1 session (~3 hours)
**Dependencies:** M-CLOUD-E2E (done), M-CLOUD-PUSH (done)
**Risk Level:** Low — small scope, well-understood APIs, existing patterns to follow

## Current Status Analysis

### Completed Recently
- M-CLOUD-PUSH: 640 LOC in 1 session (push endpoints, handler refactoring)
- M-CLOUD-E2E-FIXES: 43 LOC in 1 session (env var fix, Firestore fetch, dead code)
- M-CLOUD-E2E: ~300 LOC in 1 session (cloud wiring)

### Velocity
- Recent average: ~300 LOC/session for cloud infrastructure work
- This sprint: ~205 LOC estimated — well within single-session capacity

### Remaining from Design Doc
- Phase 1: Interface + Cloud Logging (~45 LOC)
- Phase 2: Cloud Run Jobs Implementation (~150 LOC)
- Phase 3: Documentation (~10 LOC changes)

## Proposed Milestones

### M1: CloudDispatcher Interface + Cloud Logging
**Goal:** Define the dispatcher abstraction and fix coordinator logging for Cloud Run
**Estimated:** 30 LOC impl + 15 LOC tests = ~45 LOC
**Duration:** 30 min

**Tasks:**
- Create `internal/coordinator/cloud_dispatcher.go` with `CloudDispatcher` interface and `DispatchParams` struct
- Add `cloudDispatcher CloudDispatcher` field to `Daemon` struct in `daemon.go`
- Add `io.MultiWriter(logFile, os.Stderr)` in cloud mode in `NewDaemon()`
- Test: verify MultiWriter outputs to both destinations

**Acceptance Criteria:**
- [x] `CloudDispatcher` interface defined with `Dispatch(ctx, DispatchParams) error`
- [x] `DispatchParams` struct has all 7 fields (TaskID, AgentID, Workspace, Provider, Directive, RepoURL, Branch)
- [x] Coordinator logs appear on stderr when `COORDINATOR_MODE=cloud`
- [x] All existing tests pass
- [x] Linting clean

**Risks:**
- None — pure additions, no existing behavior changes

### M2: Cloud Run Jobs Dispatcher Implementation
**Goal:** Implement the Cloud Run Jobs backend for the dispatcher interface
**Estimated:** 80 LOC impl + 60 LOC tests = ~140 LOC
**Duration:** 1 hour

**Tasks:**
- Create `internal/dispatch/cloudrun/dispatcher.go` implementing `CloudDispatcher`
- Add `cloud.google.com/go/run/apiv2` dependency via `go get`
- Implement `Dispatch()`: construct job name from config, build `RunJobRequest` with env var overrides
- Create `internal/dispatch/cloudrun/dispatcher_test.go` with mock client tests
- Verify request construction, env var mapping, error handling

**Acceptance Criteria:**
- [x] `CloudRunJobDispatcher` implements `coordinator.CloudDispatcher` (compile-time check)
- [x] Job name constructed as `projects/{project}/locations/{region}/jobs/{prefix}-agent-executor`
- [x] All 7 env vars set as container overrides (AILANG_TASK_ID, AILANG_AGENT_ID, AILANG_WORKSPACE, AILANG_PROVIDER, AILANG_DIRECTIVE, AILANG_REPO_URL, AILANG_BRANCH)
- [x] Error from `RunJob()` propagated correctly
- [x] 5 unit tests (8 sub-tests) passing
- [x] Linting clean

**Risks:**
- `cloud.google.com/go/run/apiv2` may pull in heavy dependencies — Mitigation: isolated in separate package, only imported in cloud mode
- RunJob API may require specific request format — Mitigation: check GCP docs during implementation

### M3: Wire Dispatcher + Update Dispatch Loop
**Goal:** Connect the dispatcher to the daemon and update `dispatchTasksCloud()` to trigger jobs
**Estimated:** 25 LOC impl + 0 LOC tests (tested via M2) = ~25 LOC
**Duration:** 30 min

**Tasks:**
- In `daemon_tasks_init.go`: create `CloudRunJobDispatcher` when `COORDINATOR_MODE=cloud`, store on daemon
- Read `AILANG_CLOUD_REGION` env var (default: `europe-west1`)
- In `daemon_tasks_exec.go`: after Pub/Sub publish, call `d.cloudDispatcher.Dispatch(ctx, params)`
- If dispatch fails, reset task to pending and log error
- Update CHANGELOG entry in `changelogs/v0.9-current.md`
- Update CLAUDE.md env var table with `AILANG_CLOUD_REGION`

**Acceptance Criteria:**
- [x] Dispatcher created and stored on daemon in cloud mode (via `SetCloudDispatcher()` from CLI entry point)
- [x] `dispatchTasksCloud()` calls dispatcher after Pub/Sub publish
- [x] Failed dispatch resets task to pending (not stuck in queued)
- [x] `AILANG_CLOUD_REGION` documented in CLAUDE.md
- [x] CHANGELOG updated
- [x] All tests pass
- [x] `go build ./...` succeeds

**Risks:**
- Circular import between coordinator and dispatch packages — Mitigation: interface in coordinator, implementation in dispatch/cloudrun

## Success Metrics
- All existing tests passing
- 3+ new tests for CloudRunJobDispatcher
- `go build ./...` clean
- `make lint` clean
- CHANGELOG and CLAUDE.md updated
- Coordinator logs visible on stderr in cloud mode

## Dependencies
- `cloud.google.com/go/run/apiv2` — new Go dependency for Cloud Run Jobs API
- Terraform already has: `agent-executor` Cloud Run Job, `roles/run.developer` IAM

## Open Questions
- None — design doc covers all decisions, user confirmed dispatcher interface approach
