# Sprint Plan: M-FS-DENY-WRITE-DIR-RENAME

Refs #1569

**Design:** [Approved design](m-fs-deny-write-dir-rename.md)
**Status:** Implementation complete; validation gate unmet due to unrelated whole-suite failures.
**Target:** v0.52.6
**Priority:** P0 — filesystem sandbox bypass
**Duration:** 2 working days, approximately 8 hours including 2 hours of review/validation buffer.
**Estimated changes:** 320 LOC (85 implementation, 220 tests, 15 documentation); counts are planning estimates, not quotas.
**Risk:** Medium — security enforcement must fail closed without blocking unrelated operations.
**Dependencies:** Approved design only; no new packages or language semantics.

## Goal and current status

Close the parent-directory rename bypass in #1569 with one shared structural-mutation matcher. Rename source and destination, and every removal entry point, must reject a path that a deny pattern matches directly or can match beneath. Preserve ordinary write/mkdir checks and base-name patterns that continue protecting files after a directory move.

Read issue #1569 and its comments through the GitHub API on 2026-10-08; the comments endpoint returned an empty list. The issue's `dir/**` workaround protects that directory but misses higher ancestors: `a/b/**` must also protect `a`. The approved design also covers `a/*/x.txt`, destination planting and empty ancestor removal. It audits policy-tool, archive helper calls, case/NFC folding and all structural effect entry points; no design revision is needed.

The working tree was clean at planning start on `coordinator/task-1f55def0`. The approved artifact is present even though its original work branch was `coordinator/task-1060a9ef`. Current source still has only `Protection.Check`/`MatchDenyWrite`, and structural handlers still call `fsCheckMutation`; no implementation is banked here. Existing denial tests dispatch through `effects.Call`, assert disk state and provide the harness for regression coverage.

Recent changelog releases v0.52.1–v0.52.5 include the shared folded matcher (#1559) and policy hardening. The velocity script was run for seven days, but this shallow checkout contains one root snapshot dated 2026-10-08 and no usable parent diff. Its changelog LOC scan mixes older releases, so no historical LOC/day can be inferred. Use the design's six-hour task estimate plus a two-hour buffer, spread over two working days; 160 LOC/day is planning capacity only.

## Registry reuse audit

`ailang pkg search fileguard` and `ailang pkg search sandbox` both returned no packages. The installed binary warned it may be stale; these results are advisory, and the architectural reason for no dependency is decisive: an AILANG package cannot replace the Go host's pre-effect security gate. No candidates existed to inspect with `pkg info`/`pkg docs`.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | Extend the existing Go leaf package `internal/fileguard`; no package dependency can enforce host mutations. |
| M2 | none | Wire existing Go FS handlers to the shared gate; keep archive helper callers covered through `FSRemove`. |
| M3 | none | Document and validate the existing policy/runtime contract; no package capability is added. |

## Milestones

### M1: ✅ Shared structural-mutation matcher (~160 LOC)

**Estimated:** 60 implementation + 100 tests; 2 hours, Day 1.
**Dependencies:** None.
**Files:** `internal/fileguard/protect.go`, `internal/fileguard/protect_test.go`.
**Executable examples:** table-driven Go matcher cases in `protect_test.go`; no new public AILANG feature or standalone example is needed.

Add `Violation.Ancestor`, `MatchDenyMove`, `mayMatchBeneath` and `Protection.CheckMove`. Reuse existing cleaning/folding rules; preserve `Check` and `MatchDenyWrite` behavior. Apply direct protection first, retaining `.git` precedence, then component-prefix matching with a component remaining. Exclude base-name matching from the beneath predicate. Treat a `**` component and encountered malformed component globs conservatively as specified by the design. Return the original operator pattern and distinguish direct versus ancestor denial.

- [x] Literal `.claude/settings.json`, `a/b/**` and `a/*/x.txt` refuse their matching ancestors; direct protected paths remain refused.
- [x] Case/NFC folding, leading `./`, cleaned paths, wildcard components, middle `**` and malformed-glob fail-closed behavior have explicit unit cases.
- [x] Unrelated paths and `.claude/sub` remain allowed under `.claude/settings.json`; base patterns `*.yml` and `Makefile` do not freeze unrelated directory names.
- [x] `CheckMove` preserves `.git` precedence and marks only additional ancestor violations with `Ancestor`; pattern text remains original.
- [x] `go test ./internal/fileguard` passes and existing plain-check tests remain unchanged in behavior.

**Risk:** A second normalization rule or an overly broad basename check changes existing policy semantics. Mitigate by using the current helpers and paired allowed/denied cases. Include root-relative `.` in the unit audit; do not invent root-move semantics if the backend already rejects it. Escalate any conflict with the frozen predicate before changing it.

### M2: ✅ Structural FS enforcement and denial evidence (~145 LOC)

**Estimated:** 25 implementation + 120 tests; 3 hours, Day 1 and Day 2.
**Dependencies:** M1.
**Files:** `internal/effects/fs_root.go`, `internal/effects/fs_dir.go`, `internal/effects/fs_deny_write_test.go`.
**Executable examples:** regression fixtures through `effects.Call` in `fs_deny_write_test.go`; use the existing design's CLI repro as an optional end-to-end check after obtaining `ailang prompt`.

Add `fsCheckMove` with the existing sandbox resolution, early-return, `.git` and error-code behavior, calling `CheckMove`. Wire both operands of `renameFile` and `renameFileResult`; wire `removeFile`, `removeFileResult`, `removeDirResult` and exported `FSRemove`. Keep writes, appends and mkdir handlers on `fsCheckMutation`. Audit the call-site list against current source rather than design line numbers.

- [x] For literal, nested `/**` and mid-glob policies, source moves and independently prepared destination planting are refused before any mutation.
- [x] Both rename variants exercise both operands through `effects.Call`; throwing variants report `E_FS_PROTECTED`, Result variants return `Err` containing the same code.
- [x] All three dispatched removal variants refuse an empty protected ancestor with `E_FS_PROTECTED`, avoiding a false pass from the backend's non-empty-directory error; `FSRemove` is tested directly as an exported helper.
- [x] Every denied mutation asserts unchanged file content and source/destination directory state; do not demand denial of an arbitrary temporary-path write that the policy does not protect.
- [x] Unrelated directory moves, `run.json.tmp` to `run.json`, ancestor mkdir/write behavior, no-deny operation behavior and base-glob directory moves remain allowed; moved base-glob files remain write-protected.
- [x] Existing `.git`, sandbox-boundary and direct-write protection tests pass; `go test ./internal/effects ./internal/fileguard ./internal/policytool` passes.

**Risk:** Tests can mistake ordinary filesystem failure for security denial or omit destination-only attacks. Use empty ancestors and independently prepared trees, and assert the security code plus disk state. Policy-tool currently has no structural operation, so its write/edit gate remains unchanged; future structural ops must explicitly call `CheckMove`.

### M3: Policy documentation and release validation (~15 LOC)

**Estimated:** 15 documentation LOC; 1 hour changes plus 2 hours shared buffer, Day 2.
**Dependencies:** M1, M2.
**Files:** `std/fs.ail` (comments only), `docs/docs/guides/agent-tool-policy.md`, `changelogs/v0.32-current.md`.
**Examples:** Document the literal ancestor, nested `/**` ancestor and mid-glob cases using existing design examples/policy prose. No new `.ail` program is required; fetch `ailang prompt` before any `.ail` edit, and type-check if code is added.

- [ ] FS rename/remove comments and the policy guide state that structural operations protect matching ancestor directories, including both rename operands.
- [ ] The guide explains `dir/**` still protects all contents and now protects its higher ancestors; basename patterns keep their current behavior.
- [ ] Add a Fixed entry under Unreleased referencing #1569 and the two sibling variants; do not bump versions or open a new issue.
- [ ] Run `make fmt`, `make test`, `make lint`, and `make check-boundaries`; record actual results and surface any environmental blockers instead of claiming unrun checks passed.
- [ ] Inspect the final diff for one shared matcher, correct handler coverage and unchanged write/mkdir semantics; carry `Refs #1569` in the implementation PR body.
- [ ] Run sprint-evaluator against the approved design and this plan after execution; resolve failures before reporting completion.

**Risk:** Full repository checks may exceed the small coding effort. Reserve two hours for validation/repair; report unrelated failures distinctly with evidence.

## Day-by-day execution

| Day | Work | Estimated effort |
|---|---|---|
| 1 | M1: matcher tests then implementation; M2: denial fixtures, helper and handler wiring | 4 hours |
| 2 | Finish M2 removal/allowed-operation evidence; M3 docs, repository checks, review and evaluator handoff | 4 hours including buffer |

Execute sequentially: M1 → M2 → M3. No parallel agents are required for this small, coupled security patch.

## Success metrics and boundaries

All three reproduced bypass classes are denied on both rename operands; every structural removal entry point is covered. Every security test asserts the named denial and observable disk state. Target complete branch-case coverage of the new matcher predicate rather than an invented global coverage percentage. Existing direct-denial and allowed-operation fixtures continue passing. No second matcher, new dependencies, policy grammar, effect signatures or directory traversal is introduced.

Symlink write-through, shell/exec mutation policy, recursive deletion APIs, policy validation of middle `**`, and sandbox-check CLI enhancements remain outside this sprint. Exact ancestor refusal wording is finalized in M2 within the approved message shape: operation path, original pattern, ancestor relationship and `E_FS_PROTECTED`; existing direct-denial wording stays stable. No blocking design question remains.

## Coordinator handoff

Progress file: `.ailang/state/sprints/sprint_M-FS-DENY-WRITE-DIR-RENAME.json`.
The coordinator handoff explicitly approved execution of this sprint plan. Implementation runs on `coordinator/task-6bda97e1`; the original planner branch remains `coordinator/task-1f55def0`. The prepared implementation PR body includes `Refs #1569`; no new issue was created.


## Execution evidence (2026-10-08)

Implementation completed on `coordinator/task-6bda97e1`. The new structural regression
matrix failed before wiring and now passes. Unit matcher tests, effects regressions,
policy-tool tests, Windows cross-compilation, and `ailang check std/fs.ail` pass.
`make fmt`, `make fmt-check`, `make check-boundaries`, and `make check-file-sizes` pass.
`make lint` passes with zero issues using `GOFLAGS=-p=1 GOMAXPROCS=2 GOGC=30` after an
initial memory-killed run.

Full `make test` is non-green outside this patch: CLI/validator/package helper linking
was killed; coordinator timeout (confirmed isolated: 10 seconds vs <1 second),
eval-harness Python/process-monitor tests, and executor process-group termination tests
failed in this minimal container. The affected effects, fileguard and policytool packages
pass. Independent evaluation must remain FAIL until the whole-suite gate passes on a
suitable runner; M3 and sprint completion are not marked passed. No unrelated code was changed.

Prepared PR body: `.ailang/state/sprints/M-FS-DENY-WRITE-DIR-RENAME-pr-body.md`.
