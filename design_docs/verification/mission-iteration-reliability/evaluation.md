# Independent implementation review — M-MISSION-ITERATION-RELIABILITY

Date: 2026-09-08. Reviewer: independent_review agent, independent of implementation.
Scope: M1–M3 and their CLI integration in the working diff on
`sprint/mission-iteration`, based on `3d28f3dbb`. The reviewer made no production
or test implementation changes and performed no provider calls.

**Implementation verdict: PASS, 94/100, zero unresolved code blockers.**
**Full sprint/live adoption: PENDING.** This report does not establish useful live
acceptance, completed replay, two additional successful tasks, or the $5 smoke-series
outcome. It does not authorize merge, publication, or fleet rollout.

## Evidence and score

The sprint-evaluator rubric is applied to the implementation subset. All 13 M1–M3
criteria were verified independently against code and boundary tests. A bookkeeping
refresh verified M1–M3 `passes`, completion dates, notes and plan checkboxes. M4
remains correctly open; this is not a completed full-sprint acceptance calculation.

| Category | Score | Evidence |
| --- | ---: | --- |
| Tests | 20/20 | Final `make test` passed; `/private/tmp/reliability-full-tests-final.log` |
| Lint | 10/10 | Final `make lint`: zero issues; `/private/tmp/reliability-lint-final.log` |
| Implementation acceptance | 30/30 | 13/13 M1–M3 criteria verified; matrix below |
| Code quality/artifacts | 10/15 | Capped -5 for long orchestration functions; M1–M3 sprint artifacts complete |
| Documentation | 15/15 | Active changelog, operator guide command examples, honest design implementation/live status; CLI examples substitute for irrelevant `.ail` examples |
| Design fidelity | 9/10 | Imported historical prerequisite contracts cannot recover unavailable original acceptance hashes; manifests now state this explicitly (-1) |
| **Total** | **94/100** | **Implementation only** |

Additional author-run checks inspected/reported: internal mission/coordinator tests
and race suite passed, final focused CLI race passed, build passed, architecture
boundaries passed. Logs: `/private/tmp/reliability-internal-tests.log`,
`/private/tmp/reliability-race-tests.log`, `/private/tmp/reliability-cli-race-final.log`,
`/private/tmp/reliability-build.log`, `/private/tmp/reliability-boundaries.log`.
The author also reported a passing final iteration suite and focused race check for
the multi-role approval repair.
The reviewer additionally inspected the passing retained-binding/progress race result
at `/private/tmp/reliability-retained-race-final.log` and the retained `checks.md` packet.

Reviewer-run targeted activation/iteration/dispatch tests passed for packet bounds,
request replay, retry provenance and authority, progress journaling, ownership,
process-death recovery and legacy-start races. Reviewer-run CLI tests passed for
activation, progress/status and stop attestation. One newly introduced multi-role
test initially failed because its fixture crossed the real designer approval gate;
the fixture was repaired without weakening production gates, and the final full
suite passed afterward. The initial failed full-suite log is superseded by the
`-final.log` result, not erased.

Compiler regression-surface and CPU-performance conditional categories do not apply:
this change does not modify language/compiler semantics or claim CPU optimization.

## Acceptance verification

| Milestone | Verified implementation behavior |
| --- | --- |
| M1 (4 criteria) | Exact candidate/base, criteria, checks, provenance and result protocol in bounded packet; explicit incomplete diff; packet outside stage roots and immutable request replay; private bounded progress observer and durable journal; repetition remains diagnostic, with no fallback or new termination heuristic |
| M2 (5 criteria) | Terminal-only successor preparation, frozen request/report/author verification, original rows unchanged and evaluator-only dispatch; active/ambiguous/object/authority/destination rejection; explicit limits/routes with original scope/checks; typed status and actionable unknown causes; version-fenced stop attestation restricted to existing cancellation/deadline semantics |
| M3 (4 criteria) | Owned marker/binding transaction restores baselines after verified process stop; external changes and ambiguous work retain actionable holds; serialized host ownership and crash checkpoints; retained read-only evidence access after restoration, with no new writable runtime selection |

## Findings fixed before this verdict

1. Cleanup checked nonexistent child state `stopped` instead of actual terminal
   `cancelled`. The verifier and cancellation regression now match store semantics.
2. A declared review baseline could override stronger accepted-author provenance.
   It is now restricted to evaluator-only imported-executor work, and mismatches
   with local accepted input revisions are rejected.
3. Launcher handoff reread the mutable caller work-file path. It now publishes a
   private read-only canonical work file and binds/validates its digest and identity
   through the child process record. Source mutation and subprocess gate/EOF tests
   cover this boundary.
4. New approvals originally only bound executor output, leaving locally accepted
   designer/planner outputs impossible to import. Preparation now reports every
   missing approval, accepts supplied references only for exact imported artifacts,
   and still requires fresh review authority for executor output.
5. Pre-dispatch child failure with an absent runtime DB stranded cleanup. A durable
   typed `no_dispatch` receipt now permits that specific case after session stop;
   absent state without that receipt remains held.
6. Running/crashed progress was initially only available by reading raw journals.
   Status now reads bounded matching complete progress records and explicitly
   reports incomplete, oversized, redirected or ambiguous receipt evidence.

## Residual limits and next gate

- This is a local macOS Docs pilot. Process verification trusts the current executor
  adapters' session containment and the host's user-owned installation state. It is
  not an isolation boundary against a hostile process running as the same user.
- Historical imported prerequisites retain their content-bound artifact/authority
  contract, but do not magically gain original acceptance digests. The manifest's
  `provenance_sources` explains when those digests are unavailable.
- Progress inspection deliberately becomes unavailable beyond its bounded journal
  scan. It never interprets absent progress as successful completion or no dispatch.
- Several state-machine/orchestration functions exceed 50 lines, including
  `PrepareReview`, `Manager.Activate`, `Manager.Recover` and `runMissionActivation`.
  This is a maintainability deduction, not an observed correctness blocker.
- M1–M3 sprint bookkeeping is complete. Keep M4 open until retained
  live evidence proves the approved smoke-series criteria; unit tests and this
  implementation score do not demonstrate evaluator productivity.

Bookkeeping refresh: the initial 89/100 snapshot included a five-point pending-artifact
deduction. Reinspection confirmed completed M1–M3 artifacts, restoring those points.
The long-function and imported-provenance deductions remain. No implementation
changes or additional live results were introduced by this refresh.

## Reviewed key-file fingerprints

```text
c94bf988ead4b2ad061833f1241d5d5ddcb9bad58ea8fbf0a1ebc5d0b40ce22b  internal/mission/iteration/retry_review.go
2cc668081894ad850f3b721a76535e3c81f347c40d8ef3592b1637f17e9f6887  internal/mission/iteration/review_packet.go
f5e559d202f63caff4a0015dbd868f08916961d4cd1342123ee2d6007f0313eb  cmd/ailang/mission_activation.go
1a29e14677a7bca424504adc4a5d4a720b5f8a611285204152f86fa3bc901141  cmd/ailang/mission_activation_process.go
db0c4b1b25b214e12a71f3752e9ee09466f1f2f080f161e0ecae4dfd4aa96214  cmd/ailang/mission_activation_unix.go
ca260f95200b414a698fcf475c7094db08cd8a2085f550bcc3aa0e83a5be986a  internal/mission/activation/activation.go
7ce3bfa7e57a907c35e2b3e4ea23a66de33d3310fbbe786c3fc485c656423c2c  cmd/ailang/mission_iteration_status_progress.go
```
