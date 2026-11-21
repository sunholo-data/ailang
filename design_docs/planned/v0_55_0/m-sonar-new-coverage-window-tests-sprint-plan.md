# Sprint plan: bounded Sonar new-coverage window tests

**Status:** Proposed sprint; approved design input, execution not authorized.
**Design:** [Bounded product tests](m-sonar-new-coverage-window-tests.md).
**Sprint:** M-SONAR-NEW-COVERAGE-WINDOW-TESTS. **Duration:** 3 working days, 17 focused hours plus 3 hours contingency (20-hour budget), excluding CI/Sonar turnaround.
**Risk:** Medium: measurement drift and unverified line-level coverage projection.

## Scope and approval

The user approved the previous design and requested sprint planning. This records D1 (in-package tests) and D3 (five-file scope) as approved planning constraints. Sprint approval and an explicit execution instruction remain required. No executor message is sent in this planning step. No implementation is included.

Preserve the 80% new-code policy, previous-version baseline, overall 29% floor, existing dispositions and D-TRIAGE-21 go:S1764 reliability HOLD. No CI plumbing, exclusions, -coverpkg, production semantics or tail-file expansion. A failing contract that reveals a production defect stops this sprint for separate triage; park the failing test and evidence rather than fixing production code.

## Current status and estimates

The design diagnoses a legitimate v0.54.18 window and cross-package profile blind spots. Pinned analysis 27c1dd2e-9d70-47b6-b3b1-bebb920bed21 has 1431 new lines, 361 uncovered, 1070 covered: at least 75 additional lines are needed for 80%. Approximately 91 are projected across the five files (25+25+15+6+20), not yet measured. Local Go statement/block percentages do not equal Sonar executable-line coverage; M1 and M9 reconcile rather than treating the projection as proof.

Planning inspection confirms the target production APIs and existing harnesses. service_test.go is 228 lines, value_map_test.go 27, and list_iterative_test.go 533; anticipated additions remain below 800. Do not grow list_iterative_test.go: use the new array companion. If any file approaches 800, use service_validation_test.go or value_map_contract_test.go companions in the same package.

The velocity script was run for seven days. This shallow checkout exposes only design commit 4c3dc018 (#1997); it yielded no usable implementation LOC/day. Historic changelog excerpts are not current velocity. The estimate therefore uses the design's 17 focused hours with approximately 18% contingency: 500 test LOC and 30 release-note/evidence LOC, zero production LOC. The nominal 177 LOC/day is a capacity assumption, not measured throughput. Nine milestones target 60–90 minute implementation sessions; M1 and M9 each allow two hours for local evidence, with external waits outside the budget. Split a milestone at a documented checkpoint if a run reaches its time limit.

No matching sprint artifact was found in the local plan/state search. M1 must recheck current dev for overlapping tests before adding any. Existing cross-engine golden and runner prelude tests remain authorities; this sprint adds distinct contracts rather than copying them.

## Registry reuse audit

Every milestone has action `none` in JSON. No package-like capability or AILANG package is introduced, so registry search/info/docs are inapplicable. Reuse the repository's Go callback harness, coordinator store API, compiler stmt harness, pcoll and existing coverage tools. No dependency changes.

## Milestones

### M1: Measure profile completeness and window (~0 LOC)

**Dependencies:** None. **Files/evidence:** Coverage evidence in PR/sprint notes; no production or measurement configuration edits..

- [ ] Record current dev SHA, Sonar analysis ID/date, previous-version baseline, new lines/uncovered counts and every gate condition; distinguish the pinned analysis from current measurements.
- [ ] Capture before profiles for the five packages with per-package instrumentation and no -coverpkg; map zero-count blocks to actual proposed contracts.
- [ ] Record current required line delta and deduplicate against tests merged since the design; stop for scope review if remaining approved contracts cannot plausibly close the gap.

### M2: Array accumulation order and persistence (~65 LOC)

**Dependencies:** M1. **Files/evidence:** internal/builtins/array_accumulate_test.go (new).

- [ ] Assert histogram output and ordered repeated-index combines using a real FnCallerN callback.
- [ ] Assert the source array remains unchanged and callback inputs observe the previous write.
- [ ] Run go test ./internal/builtins -count=1.

### M3: Array accumulation structured failures (~55 LOC)

**Dependencies:** M2. **Files/evidence:** internal/builtins/array_accumulate_test.go.

- [ ] Assert wrong array/list, malformed tuple, non-int index, nil context and unwired callback failures.
- [ ] Assert negative and upper-bound indices name index and array length; callback failure identifies the failing write and retains the error cause as supported by callbackErr.
- [ ] Run go test ./internal/builtins -count=1.

### M4: Lazy evaluator map keys (~60 LOC)

**Dependencies:** M1. **Files/evidence:** internal/eval/value_mapkey_test.go (new).

- [ ] Assert IndirectValue is forced before encoding and yields the forced value key.
- [ ] Assert forcing failure propagates through KeyWriter.Fail; do not copy structured-key golden cases.
- [ ] Run go test ./internal/eval -count=1.

### M5: Approval record and filesystem guards (~80 LOC)

**Dependencies:** M1. **Files/evidence:** internal/accessapproval/service_test.go additions.

- [ ] Assert unknown command reports invalid_command and valid command names; corrupt stored ContextJSON reports invalid_record using the existing store API.
- [ ] Assert unsafe state directory permissions and non-0600 coordinator.db are rejected with isolated temp directories; no ambient home/state access.
- [ ] Run go test ./internal/accessapproval -count=1.

### M6: Approval expiry and list validation (~70 LOC)

**Dependencies:** M5. **Files/evidence:** internal/accessapproval/service_test.go additions.

- [ ] Assert past and beyond-seven-day expiry rejections with times safely away from boundaries.
- [ ] Assert zero limit uses the default, limit above 50 and negative offset are rejected, and invalid status is rejected.
- [ ] Run go test ./internal/accessapproval -count=1; leave CAS retry-exhaustion branch deferred without racing writers or adding a test seam.

### M7: Map object persistence and comparator contracts (~80 LOC)

**Dependencies:** M1. **Files/evidence:** internal/bytecode/value_map_test.go additions.

- [ ] Assert Lookup on an unencodable key errors rather than silently missing.
- [ ] Assert removing a missing key preserves MapObj identity and removing an existing key preserves the original map.
- [ ] Assert Equal invokes the supplied comparator and AsMap on a non-map panics with the tag; run go test ./internal/bytecode -count=1.

### M8: Compiler prelude image contract (~90 LOC)

**Dependencies:** M1. **Files/evidence:** internal/bytecode/compiler/prelude_test.go (new).

- [ ] Compile a bare println via the in-package stmt harness and assert exactly one synthetic $prelude.println prototype with arity one.
- [ ] Assert its instructions dispatch to __io_println; repeated prelude resolution reuses the prototype, and an unknown name returns ok=false without modifying the image.
- [ ] Run go test ./internal/bytecode/compiler -count=1; keep existing runner behavior tests as the end-to-end authority.

### M9: Reconcile coverage and publish evidence (~30 LOC)

**Dependencies:** M3, M4, M6, M7, M8. **Files/evidence:** changelogs/unreleased/2026-10-28-sonar-window-product-tests.md (new, starts ###); sprint notes and PR evidence.

- [ ] Run make test-changed with an explicit available pre-sprint TEST_CHANGED_BASE plus make check-file-sizes and make check-changelog; format all new Go tests.
- [ ] Compare before/after profiles using identical instrumentation and source mapping; itemize file, uncovered window line, asserted contract and test name without double counting overlapping blocks.
- [ ] Obtain CI full-suite and overall 29% coverage gate results and a current Sonar analysis showing new_coverage >=80% and reliability rating 1; require next dev analysis after merge for final closure.
- [ ] Verify zero production or coverage configuration changes and unchanged D-TRIAGE-21 HOLD; if the gate remains red, report exact residual counts and stop for a bounded follow-up design rather than expanding into tail files.

## Day-by-day schedule

- Day 1: M1 measurement and deduplication; M2–M3 accumulation contracts; M4 lazy-key contracts.
- Day 2: M5–M6 approval guards and M7 MapObj contracts.
- Day 3: M8 prototype contracts; M9 focused verification, evidence and CI review. Reserve contingency for harness setup and measurement reconciliation.

## Validation and completion

Use identical before/after package-only profiles, for example `go test ./internal/accessapproval ./internal/bytecode ./internal/builtins ./internal/bytecode/compiler ./internal/eval -count=1 -covermode=set -coverprofile=/tmp/sonar-window-before.out` before edits, then a distinct after path. Inspect with `go tool cover -func` and `-html`; map covered blocks to actual source lines and the Sonar window. Do not replace the repository-wide CI coverage.out with this partial profile or run its overall gate against partial data.

For the final local gate use `make test-changed TEST_CHANGED_BASE=<recorded-pre-sprint-sha>` plus file-size and changelog checks. This deliberately supersedes the design's local full-suite instruction per the current sprint-planner executor guidance: full-suite tests, profile derivation and the 29% overall gate run in CI. Use existing CI artifacts instead of repeating the whole suite locally. No source examples, .ail edits, stdlib or syntax changes are planned, so teaching-prompt, contracts/effects/inline-demo and example-generation gates are inapplicable. Existing difference_array_accumulate.ail and runner tests are references, not substitutes for these unit assertions.

Each test must name a behavioral contract in a comment and assert outputs, immutability, authority guard, prototype structure or error content. Reject execute-only tests, copies of cross-engine key goldens, and racing CAS exhaustion tests. Audit test assertions against the original contract, including implementation errors; never weaken an assertion to make the current code pass.

Completion requires measured coverage gain of at least 75 previously uncovered pinned-window lines when that window is still applicable, and current Sonar new_coverage >=80% with reliability rating 1. If the window changes, preserve and report both analyses and recompute the required delta; a new release tag's green gate alone does not demonstrate the original gap was closed. PR analysis is provisional; record the next dev analysis after merge before claiming final gate closure. Failed or inaccessible remote checks stay pending, never inferred successful.

## Risks and stop rules

Coverage estimates may overstate executable-line gain; M1 validates before tests are committed to scope. If insufficient, request a follow-up design; tail compiler/VM files are not automatically authorized. Permission tests need isolated files and explicit mode assertions, with platform constraints documented. No deterministic seam is authorized for CAS exhaustion. A production defect, policy/configuration change or widening of scope returns to the applicable design/human gate. CI latency may extend elapsed duration without changing implementation effort.

No issue is auto-closed: #1938 is a merged dependency and #1997 is the design PR, not the defect issue. The JSON therefore has an empty github_issues array intentionally. Planning artifacts remain uncommitted for review; after sprint approval and execution authorization, the executor can use the populated JSON.
