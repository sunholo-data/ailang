# Sprint Plan: M-NAMED-TEST-EFFECTFUL-HELPER

## Summary

Retain module-level helpers in named-test/property compilation and report synthetic entry effect failures as the existing purity contract. Unannotated pure helpers will pass; effectful test bodies will fail with their required effects and a usable verification route.

**Status:** Implemented; final validation recorded in the implementation report.
**Design:** [Approved design](m-named-test-effectful-helper.md)
**Duration:** 4 working days (approximately 24 hours, including 6 hours of contingency).
**Target:** v0.53.0
**Risk:** Medium (source mapping, batch fallback, both execution engines).
**Estimated total:** 420 LOC: 110 implementation, 260 tests/fixtures, 50 documentation/examples.
**Dependencies:** Existing named-test batch and D1 per-body fallback, already present in this checkout. No dependency on a future batching implementation.

## Current Status and Velocity

Planning baseline: source v0.52.5, HEAD c9273968, clean coordinator/task-0b4c1785 checkout. The handoff's coordinator/task-2802e11d is the predecessor branch; this plan stays on the current work branch.

Source inspection confirms the defect remains: named_batch.go and property_batch.go use stripWithLineMap with non-pure function deletion; executor.go's named-body path uses stripNonPureFunctions. The binding-extraction caller in executor.go must retain its existing policy. Existing named_batch_test.go and engine_parity_test.go already cover fallback, positions, parity, and strict routing.

The skill's seven-day velocity script found one visible commit, no usable recent LOC metrics, and no seven-day diff. This repository is shallow, so a historical LOC/day average cannot be supported. Historical changelog matches are not evidence of current throughput. Use a conservative planning capacity of 105 LOC/day across four days, not a measured velocity. The 18-hour task estimate has a 6-hour contingency (25% of total budget). Re-estimate if source mapping or strict-engine routing needs a broader change.

At the planning baseline, no implementation work was completed. The approved design supplies systemic analysis of batch tests, property generation, per-body evaluator/VM wrapping, import hints, binding extraction, contracts, and declaration ranges.

## Registry Reuse Audit

Executed `ailang pkg search testing`, then `ailang pkg info sunholo/testing_utils` and `ailang pkg docs sunholo/testing_utils`. Candidate: sunholo/testing_utils@0.1.1, pure Result-returning assertion helpers. It cannot modify the Go harness's source rewriting or error mapping. Registry calls succeeded, with a stale-binary warning; no compiler-behavior claim is inferred from that binary.

| Milestone | Decision | Rationale |
|---|---|---|
| M1 | none | Compiler-side source retention and line maps require internal Go changes; assertion package has no harness API. |
| M2 | none | Synthetic-entry diagnostics and engine parity belong to internal/testing, beyond the assertion package. |
| M3 | none | Documentation and local verification examples use existing std/fs and Go/CLI checks; no package dependency needed. |

## Milestones

### M1: Preserve helper declarations (~140 LOC)

**Duration:** Day 1, 5 hours.
**Estimated:** 40 implementation + 100 tests/fixtures.
**Dependencies:** None.

Add stripTestBlocks returning retained source and its original-line map without calling functionSkipRanges. Reuse the existing test/property range policy and line filtering; leave the brace scanner unchanged. Switch named_batch.go, property_batch.go's forallCaller, and executor.go's named-body base builder. Preserve the binding-extraction strip and its tests.

**Files:** internal/testing/source_strip.go, source_strip_test.go, named_batch.go, property_batch.go, executor.go. Add internal/testing/testdata/strip/named_test_unannotated_helper.ail. Extend the existing named_test_effectful.ail regression assertions to prove the unused effectful closure is retained without being executed. Fixtures are this milestone's runnable examples; no public language feature is introduced.

**Acceptance criteria:**
- [x] Unannotated pure helper passes under evaluator and bytecode, matching pure and explicit-empty-row controls.
- [x] All functions, annotations, contracts, imports, types, and bindings survive; only named-test/property blocks are removed, with correct original-line mapping.
- [x] Property batch and forced forallCaller fallback resolve an unannotated pure helper with the same seed/results.
- [x] Existing binding-extraction and malformed-control tests retain their behavior; uncalled effectful helpers do not execute.
- [x] Focused source-strip and named-batch tests pass; edited Go files are formatted.

**Risk:** Broader module validation now exposes invalid unused helpers. Assert the original module error and source location rather than treating that consistency change as a regression.

### M2: Map entry effect failures truthfully (~220 LOC)

**Duration:** Days 2–3, 9 hours.
**Estimated:** 70 implementation + 150 tests/fixtures.
**Dependencies:** M1.

Implement a narrowly matched entry-effect diagnostic mapper in internal/testing and wire batch and per-body evaluator/VM errors, including property fallback. Match the exact effect-check heading for __namedtest_<number>, __namedtest_entry, or $tmp<number>; do not use substring matching over arbitrary error text. Preserve required effect rows and meaningful compiler detail while removing synthetic signature advice and temporary paths from user-facing output. Position and construct identity come from the original test/property metadata and retained-line map.

**Files:** internal/testing/named_batch.go, property_batch.go, executor.go, bytecode_engine.go; new named_test_effectful_diagnostic_test.go; extend named_batch_test.go and engine_parity_test.go. Add testdata/strip/named_test_effectful_helper.ail, named_test_effectful_direct.ail, named_test_inferred_effectful.ail, and fixture.txt if needed by the integration harness. Cover exported/non-exported helpers and mixed pure/effectful sibling tests.

**Acceptance criteria:**
- [x] Effectful helper and direct FS calls report the original construct name/file/line, pure-body contract, Missing effects: FS, and exported-entry `ailang run --caps FS` workaround.
- [x] Public test reports and fallback notices contain no synthetic function advice, temp paths, or std/debug import recommendation for a locally declared helper.
- [x] Mapper unit tests cover all three synthetic forms, multiple effect rows, and negative controls for user functions, undefined names, and unrelated errors.
- [x] Invalid unannotated effectful helpers retain the compiler's module-level effect error and original helper location; they are not falsely attributed to a pure test entry.
- [x] Evaluator, --bytecode, and --strict-bytecode agree on semantic results; expected engine-routing metadata is asserted separately.
- [x] An effectful test triggers truthful D1 fallback; valid pure sibling tests still pass, and no user error is called a harness bug.
- [x] Property effect errors identify the property rather than a named test, and property fallback preserves source mapping and seeded behavior.

**Design clarification:** The design's class-5 summary asks for an entry-contract message, but its architecture explicitly forbids rewriting user-function effect errors. Follow that architecture: class 5 must surface the real module-level effect error, with no undefined-name/import lie. Likewise preserve useful error detail, not raw synthetic identifiers. These distinctions prevent acceptance tests from demanding mutually exclusive output.

**Risk:** Batch compile failure precedes execution and its notice may expose raw errors. Include the complete CLI output in assertions, not only per-test result strings. Strict mode may fail at compilation before VM invocation; verify sibling isolation and don't impose impossible VM execution counts on failed entries.

### M3: Document and validate the contract (~60 LOC)

**Duration:** Day 4, 4 hours plus remaining contingency.
**Estimated:** 10 tests/fixtures + 50 documentation/examples.
**Dependencies:** M1, M2.

Update docs/docs/guides/testing.md as the canonical user-facing contract page and add a short cross-reference in docs/docs/reference/limitations.md. Include pure-helper usage and the exported FS verification route. Add examples/tests/named_helper_purity.ail (passing pure helper with an unused effectful declaration) and examples/tests/effectful_fixture_verification.ail with a small tracked fixture file. Place any guide-specific review metadata updates alongside the page edits. Add changelogs/unreleased/2026-10-08-m-named-test-effectful-helper.md following the fragment conventions.

**Acceptance criteria:**
- [x] Guide explicitly states named-test/property entry purity and the capability-granted exported-entry route for effectful verification.
- [x] Obtain `ailang prompt` before writing .ail fixtures/examples; `ailang check` accepts intended-valid examples, while the intentionally invalid class-5 fixture has an asserted failure.
- [x] Passing example succeeds under evaluator and bytecode; exported verification example succeeds with FS capability and fails without it.
- [x] Reporter repro and pure control are verified with a freshly built binary; effectful case fails truthfully and pure control passes.
- [x] Focused testing package, boundaries, formatting, and lint validated; CLI/core CGO/sqlite environment limitations recorded in the completion report.
- [x] Focused coverage includes each changed strip/mapper branch and negative controls; existing package coverage does not regress without an explained reason.
- [x] Testing guide, limitations note, examples, and changelog describe the resulting behavior consistently.

**Risk:** The installed binary reports possible staleness. Build the repository CLI for verification and record its version; do not count old-binary checks as implementation evidence.

## Day-by-Day Execution

| Day | Tasks | Deliverable |
|---|---|---|
| 1 | Read teaching prompt; add retention regressions; implement the three base-source switches and line-map coverage | M1 passing focused checks |
| 2 | Table-test mapper, synthetic heading recognition, real user-error controls; integrate evaluator and batch notices | Correct positioned contract diagnostics |
| 3 | Wire VM/property fallback; full-output CLI assertions across engines and mixed siblings | M2 parity and isolation evidence |
| 4 | Add verified examples, guide note and changelog; run scoped package/core/boundary/style checks | M3 reviewable evidence and completed sprint state |

## Validation and Success Metrics

Use `go test ./internal/testing -run 'TestStrip|TestNamedBatch|TestEngineParity|TestNamedTestEffect'` during implementation, expanding the filter if new property-specific test names require it. Record a baseline with `go test ./internal/testing -cover` and compare after implementation; no historical coverage percentage is asserted by this planning stage. Run the full testing/CLI package suites at completion because make test-core alone does not establish harness coverage.

Success means the legal unannotated helper becomes green, effectful entries fail truthfully on both engines, invalid modules preserve their real errors, seeded properties remain deterministic, existing binding extraction is unchanged, and pure siblings survive D1 fallback. No changes to internal/types import hints, test capability flags, or the string-brace scanner are part of this sprint.

## Handoff

Machine state: `.ailang/state/sprints/sprint_M-NAMED-TEST-EFFECTFUL-HELPER.json`.
Execution was explicitly authorized by the user's sprint dispatch, including the
plain effect-free export regression from #1640. All milestones are implemented;
final validation limitations and evaluation are recorded with the completion
artifacts. The coordinator pushes the local commits and opens the implementation
PR with `Closes #1640`. No push was attempted by the executor.

## Execution record (2026-10-09)

Executed on `coordinator/task-708afb41`, refs #1640. M1 and M2 were committed
separately as `6132aef2` and `88a4efe6`; M3 records final validation and documents
below. Plain `export func inc(x: int) -> int { x + 1 }` is included as an explicit
regression. The evaluator fallback now uses a pure synthetic function: retaining
helpers exposed that its prior free expression did not enforce the purity contract.
Invalid user-helper effect errors also receive the original declaration location.

The user replaced full `make test` with focused package checks, `make test-core`,
lint, boundaries, and file sizes because of the 8 GiB job and prior SIGBUS failures.
No full suite, push, or compiler installation was performed. Package coverage rose
from 82.1% in an isolated archived baseline to 83.8%; the strip policy and effect
mapper functions have 100% statement coverage. New examples were checked and run
with the freshly built CLI; short-module internal fixtures used relaxed module
validation. Manifest validation reports zero module drift. CGO/sqlite failures and
final gate statuses are recorded separately in the completion artifact.

PR body must contain `Closes #1640`; the coordinator owns push and PR creation.
