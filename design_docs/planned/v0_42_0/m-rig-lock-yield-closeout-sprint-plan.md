# Sprint Plan: M-RIG-LOCK-YIELD Closeout

## Summary

Verify at current HEAD that the already-shipped cooperative rig-lock yield protocol still satisfies its approved as-built design, then close GitHub issue #1136 with reproducible evidence. This sprint deliberately adds no second implementation: the shell/Go protocol, safe-point integration, and regression suite landed in v0.38.x and are present in v0.42.0.

**Duration:** 0.25 day (about 2 hours)
**Dependencies:** Implemented design `design_docs/implemented/v0_38_0/m-rig-lock-yield.md`; M-RIG-LOCK-ENFORCE
**Risk Level:** Low
**GitHub issue:** #1136

## Current Status Analysis

### Completed Already

- ✅ Shell request/clear/pending protocol and named-requester acquisition guard in `tools/launchd/rig-lock.sh`.
- ✅ Wire-compatible Go protocol in `internal/riglock/yield.go`.
- ✅ A safe checkpoint between benchmarks at `--parallel 1` in `cmd/ailang/eval_parallel.go`.
- ✅ Unit and shell/Go interoperability coverage in `internal/riglock/yield_test.go` and `internal/riglock/yield_shell_test.go`.
- ✅ Approved as-built design in `design_docs/implemented/v0_38_0/m-rig-lock-yield.md`.

### Velocity

The seven-day history contains only the v0.42.0 release squash, so it does not provide a defensible LOC/day estimate. Velocity is not material here: remaining work is bounded verification and issue bookkeeping, with **0 implementation LOC** planned.

### Remaining from Design Doc

- ⏳ Re-run the focused protocol and interoperability tests on current HEAD.
- 📋 Confirm the live safe-point and guard wiring still match the approved design.
- 📋 Post the evidence-backed closeout for issue #1136.

## Proposed Milestones

### Milestone 1: Protocol Verification at HEAD

**Goal:** Prove the shipped shell and Go halves remain wire-compatible and preserve the safety properties in the approved design.
**Estimated:** 0 implementation LOC + 0 test LOC = 0 LOC
**Duration:** 1.5 hours

**Files inspected (no changes expected):**

- `tools/launchd/rig-lock.sh`
- `internal/riglock/yield.go`
- `internal/riglock/yield_test.go`
- `internal/riglock/yield_shell_test.go`
- `cmd/ailang/eval_parallel.go`

**Tasks:**

- Run `go test ./internal/riglock` on current HEAD.
- Run the focused shell/Go interoperability and checkpoint tests with verbose output.
- Confirm the `eval-suite` checkpoint remains restricted to `maxConcurrent == 1` and occurs before the next benchmark dispatch.
- Confirm the shell acquisition guard reserves a yielded gap for the named requester.

**Acceptance Criteria:**

- [ ] `go test ./internal/riglock` passes.
- [ ] Shell-written markers are parsed by Go and Go-written markers are parsed by shell.
- [ ] Expired, malformed, and dead-requester markers are removed and treated as absent.
- [ ] A foreign requester cannot overwrite a live handoff, and a filler cannot steal the named gap.
- [ ] `Checkpoint` is inert for non-holders and for concurrent eval execution; it yields and re-acquires for a serial holder.
- [ ] `bash -n tools/launchd/rig-lock.sh` passes.

**Risks:**

- Platform-specific BSD/GNU `date` behavior could regress outside the Studio. Mitigation: retain the Linux-exercised cross-language tests and record the platform used in the closeout evidence.
- Timing-sensitive tests can expose genuine races. Mitigation: diagnose failures; do not weaken waits or skip assertions merely to obtain green output.

### Milestone 2: Design-to-Code Audit and Issue Closure

**Goal:** Record that #1136 is satisfied by the shipped implementation and approved as-built design, without introducing redundant code.
**Estimated:** 0 implementation LOC + 0 test LOC = 0 LOC
**Duration:** 0.5 hour
**Dependencies:** Milestone 1

**Artifacts inspected (no repository changes expected):**

- `design_docs/implemented/v0_38_0/m-rig-lock-yield.md`
- GitHub issue #1136

**Tasks:**

- Map each design acceptance criterion to its implementation and passing test evidence.
- Check that current callers still use the common lock protocol and that no competing yield-marker format exists.
- Post a closeout comment linking the implemented design and verification results; close #1136 only after the evidence is attached.

**Acceptance Criteria:**

- [ ] Every acceptance criterion in the implemented design has a code location and passing test reference.
- [ ] Repository search finds one cooperative-yield marker contract shared by shell and Go, not parallel incompatible protocols.
- [ ] The #1136 closeout states that implementation landed in v0.38.x and was re-verified at the executor's HEAD SHA.
- [ ] No implementation or test files are modified unless verification discovers a real regression; any regression returns through the feature/semantics gate before repair.

**Risks:**

- The issue may already be closed or inaccessible to the executor. Mitigation: preserve the exact proposed comment and report the external-state blocker without altering code.

## Day Plan

- **Hour 1:** Run focused Go, shell, and interoperability checks; capture exact commands and HEAD SHA.
- **Hour 2:** Audit design-to-code mappings, post the #1136 closeout, and record the final verdict.

## Success Metrics

- Focused rig-lock suite passes at current HEAD.
- All four safety classes are evidenced: ownership, expiry/death cleanup, named-gap exclusion, and serial safe-point yield/re-acquire.
- Example scenario verified by tests: a short named requester can be served by a long serial holder without concurrent GPU use.
- Documentation remains the implemented design; no duplicate design or protocol is created.
- Net implementation/test LOC: 0 unless a separately approved regression fix is required.

## Dependencies

- Current Go toolchain and bash.
- GitHub access for the final #1136 comment/closure.
- No live GPU or `rig.lock` acquisition is required; the verification uses isolated temporary directories.

## Open Questions

None. If verification finds a mismatch, stop and route that newly discovered implementation change through the repository gate rather than expanding this closeout sprint.

## Notes

- The current repository version is v0.42.0; the feature itself shipped in v0.38.x.
- The recent-velocity script found no granular post-release commits, so the estimate is based on the bounded commands above rather than an invented LOC/day rate.
- This plan is intentionally a close-with-verdict sprint. Reimplementing the accepted protocol would create drift across the shell and Go halves.
