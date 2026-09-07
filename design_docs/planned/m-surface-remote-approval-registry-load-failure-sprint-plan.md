# Sprint Plan: Surface Remote Approval Registry Failure

## Summary
Make remote-plane approval fail before mutation when the local agent registry cannot be loaded or contains no dispatchable agents. The current branch already contains an apparent implementation, so execution begins by validating it and adds only missing regression coverage or corrections.

**Duration:** 0.5 day  
**Dependencies:** Approved design handoff `task-b0a3a274`  
**Risk:** Low

## Milestone 1: Lock the Failure Contract
**Estimate:** 30 test LOC, 0.25 day

- Add focused tests for registry-load failure and an empty registry on a remote approval path.
- Assert both cases return a clear error before approval state changes or handoff dispatch is attempted.
- Files: `cmd/ailang/coordinator_approvals_remote_test.go` (or the existing adjacent test file).

**Acceptance criteria:**
1. A registry load error is returned to the caller and the approval is not recorded.
2. A remote approval with no configured agents fails explicitly and does not report success.

## Milestone 2: Verify or Complete the Guard
**Estimate:** 15 implementation LOC, 0.25 day; depends on Milestone 1

- Confirm `cmd/ailang/coordinator_approvals_remote.go` passes the load error into the pre-mutation dispatch guard.
- If the inherited guard is incomplete, make the smallest correction covering both failure modes.
- Run focused coordinator tests, then `go test ./cmd/ailang/...` and formatting checks.

## Success Metrics
- Both regression tests pass and fail against the former error-discarding behavior.
- No remote approval success is printed after either registry failure mode.
- No example or user documentation change is required for this defensive CLI correction.

## Notes
The guard belongs before `ProcessApprovalRequest`: once approval state is persisted, surfacing a registry problem is too late to prevent a false-success state. Empty configuration must be treated separately because `LoadAgentRegistry` may legitimately return it with a nil error.
