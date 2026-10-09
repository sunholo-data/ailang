# Sprint Plan: M-VM-LIST-TAIL-SHARE

## Summary
Make bytecode cons-pattern list walks linear by sharing immutable list views, and make `run --strict-bytecode` select the strict VM. Design approval is supplied by coordinator handoff task-006b0353; this plan awaits the coordinator's sprint approval before execution.

**Duration:** 4 days (approximately 24 engineering hours, including measurement and contingency).
**Estimated total:** 600 LOC, predominantly fixtures/tests/benchmark documentation.
**Risk:** Medium: alias safety, benchmark reproducibility and strict-flag compatibility.
**Design:** [m-vm-list-tail-share.md](m-vm-list-tail-share.md). Related issue: #676.
**Lane:** AILANG fix. No representation change, OpCons/append optimization, interpreter change or D-19 decision.

## Current status and velocity
The three copying sites and the uncombined runFile flags remain in source. The design audited related view operations, mutation paths, bridge conversions and pattern lowering; systemic analysis is sufficient. Existing CLI subprocess tests in cmd/ailang/run_bytecode_test.go and benchmark scaffolding in bench/vm_hof_callbacks/run.sh provide conventions.

The seven-day velocity script found only one design-document commit and no usable recent LOC metrics in this shallow checkout. Historical changelog snippets are not measured current velocity. Capacity of 150 LOC/day is a planning assumption, not an observed rate. The design's two-day estimate is expanded to four days for fresh-build reproduction, alias safety and measurement. No implementation or coverage run was performed during planning: go, make and jq are absent. Executor must provision the supported Go toolchain and make before AC-0; do not substitute /usr/local/bin/ailang for a fresh build.

Quorum was controller-only with all three external routes absent. Approval in the handoff permits planning; optional external review should be retried when available, without claiming quorum clearance.

## Registry reuse audit
`ailang pkg search list` returned sunholo/a2ui, sunholo/duckdb, sunholo/decisions and sunholo/mcp_oauth. Their returned descriptions concern application protocols/clients, not VM storage or CLI dispatch; no candidate warrants pkg info/docs inspection. M1–M4 each record **none**: existing compiler/runtime, CLI, benchmark harness and repository documentation must be changed in-tree. No package dependency or publish operation is required.

## Milestones and daily schedule

### M1: Fresh-build baseline and benchmark fixtures (~180 LOC)

**Day:** 1. **Effort:** approximately 6 hours. **Dependencies:** None.

**Files:** bench/vm_tail_walk/{walk.ail,foldl.ail,build.ail,decoded_walk.ail,run.sh,README.md} (new); follow bench/vm_hof_callbacks conventions, but fail loudly on unsuccessful runs.

**Tasks:** Build and identify baseline binary before any runtime edit. Create equivalent linearly built walk/fold fixtures and separate quadratic construction control. Type-check all modules; collect unchanged-tree timings, outputs and disassembly. Add a linear-built float fixture to isolate the downstream decoded-list consumption shape without introducing quadratic mk construction.

**Acceptance criteria:**

- [ ] AC-0: build the unchanged checkout; record commit, binary version/hash, OS/architecture and startup baseline; reproduce V6–V9 using that binary.
- [ ] Add walk, foldl and build fixtures and fail-fast min-of-5 harness; record 40k/80k/160k both-engine results and disassembly showing _list_tail and TAIL_CALL.
- [ ] Benchmark inputs are equivalent, outputs checked, and list construction separated from consumption; no strict-only run is labelled VM before M3.

**Risk / mitigation:** Stale binary or mixed construction costs invalidate evidence: freeze baseline binary path and provenance, check exit codes and expected output, use identical hardware and workload..

### M2: Shared immutable list views and safety regression tests (~200 LOC)

**Day:** 2. **Effort:** approximately 6 hours. **Dependencies:** M1.

**Files:** internal/vm/builtins.go; internal/vm/builtins_list_poly.go; internal/vm/list_tail_share_test.go (new); extend existing VM parity tests as needed.

**Tasks:** Reconfirm every shared-slice consumer is copy-on-write. Write regression tests first, then replace the three copy expressions with NewList(subslice). Capture allocation bytes with Go benchmarks and address-sharing tests, plus clamping/error and alias-safety parity. Run focused VM/eval tests.

**Acceptance criteria:**

- [ ] AC-1: _list_tail, take and drop use shared subslices with unchanged clamping, arity/type errors and builtin indices.
- [ ] Nonempty result backing addresses match input offsets; allocation counts and bytes remain constant with list length, permitting the ListObj wrapper allocation.
- [ ] Negative/zero/length/overlength and empty cases pass; cons, sortBy, array conversion and bridge tests preserve original values and evaluator parity.

**Risk / mitigation:** Hidden writes could expose sharing: inspect all consumers and test original values after cons, sorting, array conversion and engine crossing..

### M3: Strict-bytecode run flag implies VM (~110 LOC)

**Day:** 3. **Effort:** approximately 6 hours. **Dependencies:** M1.

**Files:** cmd/ailang/main_run.go; cmd/ailang/run_bytecode_test.go; cmd/ailang/help.go if central help repeats the old flag contract.

**Tasks:** Extend CLI subprocess tests to prove engine selection and strict rejection using an entry already known to be EvalOnly. OR flags at the actual runFile call; update help; run focused CLI/runner tests. Do not assume all effectful entries fail: current transparent bridge support makes fixture selection material.

**Acceptance criteria:**

- [ ] AC-4: run --strict-bytecode alone selects the VM, pure supported entry succeeds, and a proven evaluator-only entry fails with the strict error and nonzero exit.
- [ ] run --bytecode --strict-bytecode and test --strict-bytecode behavior remain covered; ordinary --bytecode retains existing bridge behavior.
- [ ] Flag help says it implies --bytecode; change the actual runFile argument in main_run.go and audit central help text for consistency.

**Risk / mitigation:** An apparently evaluator-only fixture may now bridge successfully: inspect current strict rejection tests and assert a real strict error, not an unrelated capability or syntax failure..

### M4: Performance acceptance, documentation and release evidence (~110 LOC)

**Day:** 4. **Effort:** approximately 6 hours. **Dependencies:** M2, M3.

**Files:** std/list.ail (comments); docs/LIMITATIONS.md; changelogs/unreleased/m-vm-list-tail-share.md (new); bench/vm_tail_walk/README.md; sprint execution log.

**Tasks:** Rerun the exact baseline harness against rebuilt changed source. Publish before/after, scaling, VM/interpreter and foldl comparisons, and the large float-list measurement. Update cost documentation and changelog; run full required checks and evaluator review.

**Acceptance criteria:**

- [ ] AC-2/2b: min-of-5 VM 160k walk <=2x evaluator and 160k/80k <=2; record 786432-element linear-built float-list walk and <=2s target when startup is about 0.35s.
- [ ] AC-3: commit runnable harness and before/after table with timing provenance and allocation evidence; record failures without suppressing stderr.
- [ ] AC-5: make test, make lint and make verify-examples pass; first_non_repeat, record_cons_pattern and url_route_dispatch outputs agree across engines.
- [ ] AC-6: std/list header and take/drop/tail comments describe sharing; LIMITATIONS states backing-array retention and preserves D-19 construction warnings; unreleased changelog explains strict-flag behavior.
- [ ] Run sprint-evaluator against design AC-0 through AC-6; log all results and unresolved thresholds before marking sprint complete.

**Risk / mitigation:** Noise or a slower machine may miss absolute thresholds: retain all samples/startup baseline and report unmet targets explicitly; investigate rather than relax approved ACs..

## Validation details and design clarifications
AC-3's “approximately zero allocations” means **no element-sized backing allocation or copy**, not zero heap objects: bytecode.NewList wraps a slice in a ListObj. AllocsPerRun alone cannot detect the regression reliably because both old and new versions allocate a constant number of objects while old allocated bytes grow with input size. Assert shared addresses for nonempty views and O(1) bytes/op across small and large lists; permit a bounded wrapper allocation with escaping results.

The design's retention mitigation `take(len, tail)` becomes a shared view too; do not publish it as an explicit copy workaround. State the retention trade-off without promising a copying API. No new copy builtin belongs in this sprint.

Min-of-5 raw wall-clock values must include matching startup/compilation conditions. Report net times additionally, without replacing the approved raw-wall thresholds. Pure exported benchmark entries can run under both --bytecode and --bytecode --strict-bytecode; an IO main can bridge and obscure strict checks. Use explicit flags and record engine selection. The 786432 float-list fixture may use a verified linear builtin builder; record when decodeF32LE itself was not reproduced. Preserve the construction control's quadratic behavior as evidence of the D-19 boundary.

Coverage target: exercise every changed clamp/error branch and aliasing class, rather than inventing an unmeasured repository percentage. Run focused go tests for internal/vm, internal/eval, internal/runner and cmd/ailang; then make test, make lint and make verify-examples (inspect current targets and use tools/verify_examples.sh if the target is absent, recording that choice). Run make check-boundaries if the implementation becomes cross-cutting. No new imports across architectural layers are planned.

## AILANG syntax gate and showcase checklist
AILANG prompt version loaded: v0.16.6 (reported by `ailang prompt --version-active`; full current prompt fetched during planning). Executor reloads the fresh binary's teaching prompt before writing .ail code and type-checks, formats and runs every fixture. Planned function signatures below are specifications, not unchecked runnable source.

| Module | Contracts | Effects | Inline tests |
|---|---|---|---|
| walk.ail | skip: general recursive performance fixture; output parity and Go safety tests cover correctness | include: pure countGt(xs: [int], thr: int, acc: int) -> int; exported synth(n: int) -> int ! {} | include: synth small sizes 0, 1 and a threshold-crossing size with computed counts |
| foldl.ail | skip: reference fold; compare against the walk oracle | include: exported synth(n: int) -> int ! {} with identical count predicate | include: same small-size expected counts as walk |
| build.ail | skip: construction-only control intentionally preserves existing complexity | include: pure mk(n: int, acc: [float]) -> [float]; exported synth(n: int) -> int ! {} returning length | include: synth tests [(0, 0), (1, 1), (3, 3)] |
| decoded_walk.ail | skip: large float workload validated by known expected count and fold parity | include: pure countAbove(xs: [float], thr: float, acc: int) -> int; exported synth(n: int) -> int ! {} | include: small equivalent float-list sizes and threshold boundary |

## Completion and handoff
All milestones start with passes=null; dependencies must reference exact IDs. Baseline AC-0 is a prerequisite for runtime edits. M2 and M3 may be scheduled independently after M1, but this estimate assumes sequential execution. No source implementation, branch switch, commit or sprint execution is authorized by creation of this plan alone. Coordinator artifact markers carry the handoff for approval; merging the sprint-plan PR starts sprint-executor according to the skill's coordinator integration. After implementation, sprint-evaluator assesses all approved criteria and any deviations.
