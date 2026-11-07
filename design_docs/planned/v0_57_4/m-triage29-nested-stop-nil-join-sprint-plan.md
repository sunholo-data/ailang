# Sprint Plan: M-TRIAGE29-NESTED-STOP-NIL-JOIN

**Status:** Planned; execution not started or dispatched.
**Design:** [Diagnosis and fix design](m-triage29-nested-stop-nil-join.md)
**Duration:** 0.5 engineering day (about 4 hours), plus passive macOS CI soak over subsequent days.
**Scope:** Test fixture readiness only; no production, public API, ownership, process guarantees, deadline, workflow, or issue #2034 changes.

## Authorization and current status

The current handoff explicitly requests sprint planning and states prior work is approved. This supersedes the design's historical prohibition on planning for this planning stage only. It does not authorize execution, executor dispatch, merge, deployment, or paid/provider calls. No implementation is performed by this plan. Execution requires the separate user instruction to execute sprint. There are no unresolved technical human-decision gates: select design variant A1, the smallest proposed change; retain the existing helper ordering.

The design records synthetic mechanism reproduction (12/12 immediate-stop nil Joins), a strengthened-gate positive control (5/5 expected non-nil Joins), and no natural reproduction in original-gate Linux repetitions. Those are inherited evidence, not newly run results. The current source still polls only the nested marker. The source file is 360 lines; no companion split is required near the 800-line gate. Legitimate clean early shutdown remains a supported runtime behavior.

## Velocity and capacity

The skill velocity script was run for seven days. This checkout exposes one recent design-only commit (e999d154), no parent diff statistics, and historical changelog snippets rather than usable measured delivery rates. A numerical historical LOC/day would be unsupported. Use the design's bottom-up estimate: approximately 10 test lines plus 10 changelog lines, 60 report lines, and 4 hours of engineering. Total estimated changed/new lines: 80; planning capacity: 160 lines/day, not a measured throughput claim. CI wait time is excluded from engineering capacity.

## Registry reuse audit

M1 and M2: **none**. No package-like capability is introduced: this is an existing Go test's file-existence gate plus evidence documentation. Registry search/info/docs are not applicable, so no network registry queries are needed. Reuse the existing markers, helper, test and Make targets. No dependencies or packages are added.

## Milestones

### M1: Order stop after both helper bindings (~20 LOC)

**Duration:** 60 minutes. **Dependencies:** none.
**Files:** `internal/effects/process_runtime_lifetime_test.go` (~10 test lines); `changelogs/unreleased/2026-11-05-nested-stop-readiness.md` (~10 lines, opening with `### `).
**Examples:** none; no language or user-facing behavior changes. AILANG prompt/contracts/effects/inline-test gates do not apply because no `.ail` files are planned.

Require both `os.Stat(marker)` and `os.Stat(marker + ".nested")` to succeed before breaking the existing poll. Keep the 5-second readiness timeout and 1-millisecond sleep. Explain that the outer marker follows outer Bind, while the nested marker proves only nested readiness. Make readiness timeout text identify both helpers if useful. Do not reorder helpers or introduce fixture modes.

- [ ] Both existing markers are required before RequestStopContext.
- [ ] The 120 ms stop context, 370 ms upper bound, both remaining-budget assertions, non-nil Join assertion and process absence checks remain intact.
- [ ] Production code and helper mode behavior are unchanged; gofmt and diff review confirm the bounded test-only change plus changelog.
- [ ] Run a focused test once; record assertion site and elapsed time even if the known container zombie artifact prevents a pass.

### M2: Validate the deadline chain and bank evidence (~60 LOC)

**Duration:** Two 60–90 minute sessions if needed; each executor run stays below 90 minutes. **Dependencies:** M1.
**Files:** `design_docs/planned/v0_57_4/m-triage29-nested-stop-nil-join-validation.md` (~60 lines); update sprint JSON progress as evidence arrives. Keep the design planned until all acceptance criteria, including soak, are met; then reconcile/move it under `design_docs/implemented/v0_57_4/` and update links. Do not mark implementation complete based only on Linux results.
**Examples:** none; the existing nested-runtime test is the regression fixture.

- [ ] On a host with a reaping init, `go test ./internal/effects -run '^TestRuntimeNestedStopUsesRemainingDeadline$' -count=50 -v` has zero nil-Join occurrences and all assertions pass.
- [ ] `go test ./internal/effects -count=1` and `make test-changed` pass; retain logs and base revision. Run `make check-boundaries`, `make check-file-sizes`, and `make check-changelog`.
- [ ] If Linux container process absence fails, record PID/PPID/state to distinguish a zombie at the final absence assertion from the early nil-Join assertion. Report local validation incomplete; obtain a full run on a reaping host without deleting or weakening assertions.
- [ ] Record non-nil Join and both remaining-budget files as positive evidence that bound cleanup ran. Capture timing; investigate unexpectedly fast completions. Preserve the design's green duration shape of at least approximately 0.14 s as an observation, not a new wall-clock assertion.
- [ ] Observe at least 10 consecutive normal Build macos-latest legs containing the fix, with run/job IDs, commit, test outcome and duration; no nil-Join recurrence. Do not trigger paid/provider work or change workflows to manufacture soak.
- [ ] Final report reconciles historical recurrences into the existing design and clearly separates natural race non-reproduction, deterministic mechanism evidence, local validation, and macOS soak.

## Day-by-day execution

Day 1, first hour: M1 and focused smoke check. Next 60–90 minutes: M2 focused repetitions/package checks. Final 60–90 minutes: changed-package and static gates, evidence report and initial CI observation. Subsequent days: passive observation of existing macOS Build activity; resume solely to bank results. If ten qualifying legs are unavailable, keep M2 incomplete and report pending soak; do not spend an executor run polling indefinitely.

## Validation and risk handling

Use focused gates above; the full repository suite belongs to CI. No coverage percentage increase is expected: existing deadline, receipt and teardown assertions provide the regression coverage. No new tests mirroring file-existence implementation are needed. Run test repetitions serially and in isolation to avoid creating additional process pressure. No temporary synthetic test is required to repeat the already banked mechanism experiment.

A failure at the nil-Join assertion after A1 is new evidence: preserve it, reconcile into the existing diagnosis, and stop expanding the patch. A final process absence failure with an adopted zombie is a different failure and cannot be counted as a pass. Do not broaden into runtime receipt propagation, synchronous exec #2034, scheduler tuning, or protocol changes.

## Completion and handoff

Success requires all M1/M2 criteria, unchanged semantics and a complete validation report. Issue #2069 is recurrence bookkeeping only; use references without auto-closing it. Issue #2034 is explicitly excluded. No execution dispatch is part of this planning task. The plan and progress JSON are reviewable artifacts awaiting the execution instruction.
