# Sprint Plan: Stapledon Workaround Retirement

**Sprint ID:** M-STAPLEDON-WORKAROUNDS
**Design:** [Approved workaround register](stapledon-v0-54-6-workarounds.md)
**Planning date:** 2026-10-15
**Status:** Implementation complete; final validation and independent evaluation in progress.
**Duration:** 4 working days, approximately 10 focused hours plus 3 hours contingency.
**Estimate:** 1,400 LOC including implementation, tests, examples and documentation.
**Risk:** Medium-high (new VM value representation and engine parity).

## Scope and approval

Implement the approved design's complete ten-builtin VM map port, deterministic re-locking,
and named-callee application diagnostics. Record downstream migration instructions for
readLineOpt and the existing #1576 track. The user's approved-design handoff is authority
for D2 (preserve generated_at, generator and ailang_version together) and D4 (no doc alias);
the design's unchecked freeze boxes predate that handoff. No implementation is performed
by this planning task. The v0_54_7 directory identifies the approved design's original target;
source is now v0.54.9, so execution targets the next unreleased version, without a version bump.

Downstream game edits require its own checkout and tests. They are not prerequisites for
completing this repository sprint and must never be reported as already retired.
#1576's B4 unsafe replay work remains parked pending A2 in the existing parity design.
Use `Refs #1576` and `Refs #1447` for context; neither issue is closed by this sprint.

## Current status and velocity

The working tree was clean at planning start, base 79ede6ee. Source inspection confirms:
all ten __map_* entries remain on the VM unported ratchet; bytecode.Value has no Map tag;
convert.go lacks map conversion; pkgLockCommand stamps new metadata in both normal and
empty-dependency paths; inferApp already calls coreCalleeName for collision hints.
readLineOpt is existing functionality per the approved verification register.

The seven-day velocity script found only the checkout's single commit and no usable recent
LOC/day data. Historical changelog matches are not a current velocity measurement. Thus
350 LOC/day is a conservative planning assumption, not measured throughput. Seven tasks
of 60–90 minutes fit individual executor runs; reserve approximately 30% extra time.
Current coverage was not remeasured for this documentation-only task. Execution measures
coverage of changed branches through targeted tests rather than imposing an invented
repository-wide percentage.

## Registry reuse audit

`ailang pkg search map` returned billing_stripe, a2ui and relativity, domain adapters rather
than VM implementations. `ailang pkg search lockfile` returned none. `ailang pkg search
diagnostics` returned sunholo/ail_diag@0.1.0; `pkg info` and `pkg docs` confirm it parses
emitted CLI text, not compiler inference. All milestones choose **none**: packages cannot
supply a Go VM value, native dispatch, CLI lock writer, or compiler diagnostic context.
M7 documents existing stdlib APIs and needs no additional dependency. These decisions are
also recorded per milestone in JSON. No plausible reusable implementation candidate was found.

## Milestones

### M1: VM map representation and conversion (~240 LOC)

**Estimate:** 140 implementation + 100 tests; 90 minutes. **Dependencies:** none.

Add the Map tag and object in `internal/bytecode/value_map.go`, integrating tag naming,
constructor/accessor, equality and display with `value.go`. Add evaluator conversions in
`internal/vm/convert_map.go` and dispatch from `convert.go`. Match eval.MapValue canonical
int/string/bool keys, sorted iteration and nested values without importing VM into core.
Audit every ValueTag switch for the new variant; include map-valued results crossing the
entrypoint/interop boundary. Companion tests: `value_map_test.go`, `convert_map_test.go`.
Example: existing `examples/runnable/map_basic.ail` is the parity fixture for M4.

- [ ] Map values round-trip across the evaluator bridge, including nested values and empty maps.
- [ ] Canonical key identity, stable display/iteration and equality match evaluator semantics.
- [ ] Targeted bytecode/VM tests and make test-changed pass; existing value kinds remain valid.

Risk: a missing tag switch can fail only at result conversion; explicit boundary tests cover it.

### M2: Native map point operations (~240 LOC)

**Estimate:** 150 implementation + 90 tests; 90 minutes. **Dependencies:** M1.

Create `internal/vm/builtins_map.go` and `builtins_map_test.go`: empty, insert, lookup,
member, remove and size, registered through existing native builtin dispatch. Preserve
copy-on-write insertion/removal, Option constructor identity, supported key types and
error behavior. Adjust compiler eligibility only as required for these six registrations.
Remove exactly their six ratchet rows in the same milestone so coverage stays green.
Example fixture: `examples/runnable/map_basic.ail` (final strict coverage in M4).

- [ ] Six point operations execute natively with int/string/bool keys and opaque values.
- [ ] Updating a map leaves aliases to the original unchanged; missing keys return None.
- [ ] TestPureBuiltinCoverage and targeted VM/compiler tests plus make test-changed pass.

Risk: dictionary/key conventions differ between engines; use evaluator behavior as oracle.

### M3: Native map bulk operations (~220 LOC)

**Estimate:** 130 implementation + 90 tests; 75 minutes. **Dependencies:** M2.

Add from_list, to_list, keys and values in `internal/vm/builtins_map_bulk.go` with companion
tests. Preserve canonical ordering, duplicate-key winner behavior and tuple/ADT/list value
shapes. Remove the remaining four map ratchet rows; keep unrelated unported families intact.
Example: M4's `examples/runnable/map_strict_parity.ail` will exercise all ten operations.

- [ ] All ten __map_* operations are native; the map family is absent from unportedPure.
- [ ] Empty input, duplicates and mixed positive/negative integer ordering match evaluator.
- [ ] Bulk-operation tests, coverage ratchet and make test-changed pass.

Risk: numeric and lexical ordering differ; compare canonical encoding, not host map iteration.

### M4: Strict-bytecode end-to-end parity (~220 LOC)

**Estimate:** 30 example/integration wiring + 190 tests; 90 minutes. **Dependencies:** M3.

Create `examples/runnable/map_strict_parity.ail` and focused engine parity tests in
`internal/vm/map_parity_test.go` using the repository's existing integration harness.
Exercise all ten operations, value aliasing, duplicate keys, sorted iteration, and bridge
results. Include the reporter's exact fromList/lookup shape as an executable regression.

- [ ] Reporter repro returns 2 under interpreter and --bytecode --strict-bytecode.
- [ ] Full-family parity fixture produces identical results on both engines, with no fallback.
- [ ] Still-unported XML builtin retains its explicit strict-bytecode failure and reason.
- [ ] make test-changed, make verify-examples and make check-boundaries pass.

Risk: native unit tests can pass while compiled imports fail; strict CLI tests are mandatory.

### M5: Preserve unchanged lockfile metadata (~200 LOC)

**Estimate:** 70 implementation + 130 tests; 90 minutes. **Dependencies:** none.

Add `cmd/ailang/pkg_lock_preserve.go` and `pkg_lock_preserve_test.go`, with small call-site
changes in pkg_commands.go. Compare current and expected resolution with pkg.LockDrift;
preserve all three generation fields only on no drift, including empty dependencies.
Missing locks still generate; unreadable or malformed locks must surface an actionable
error rather than silently treating them as absent. Preserve existing silent-ratchet
warnings and relative-path semantics. No .ail example: Go CLI fixture packages cover this.

- [ ] Consecutive unchanged locks are byte-identical, including across generator/version changes.
- [ ] Empty dependencies behave identically; changed resolution receives fresh metadata.
- [ ] Missing lock generates; malformed/unreadable lock errors are explicit.
- [ ] --check writes nothing and retains 0/current, 1/drift behavior; portability tests pass.
- [ ] Targeted cmd/ailang and internal/pkg tests plus make test-changed pass.

Risk: reserializing arbitrary handwritten JSON can normalize whitespace; byte identity is
required for CLI-generated locks. Do not skip resolution/cache population on unchanged locks.

### M6: Name the applied callee in diagnostics (~180 LOC)

**Estimate:** 60 implementation + 120 tests; 90 minutes. **Dependencies:** none.

Reuse coreCalleeName in `internal/types/typechecker_app_diagnostics.go` and add focused
application tests in a companion `_test.go`. typechecker_functions.go is 780 lines and
must stay below 800; typechecker_core.go is already 800 and must not grow. Attach context
at the application constraint path, including local solving, without changing raw Unify.
Retain collision hints and arity diagnostic wording/codes. Use an invalid .ail test fixture
embedded in the Go integration test rather than a runnable example that cannot typecheck.

- [ ] Swapped std/map.lookup arguments produce a diagnostic containing lookup and its span.
- [ ] Named local/imported applications carry their available names; anonymous heads stay neutral.
- [ ] Raw Unify and arity diagnostic tests retain their wording/codes; collision hints survive.
- [ ] Targeted internal/types tests and make test-changed pass.

Risk: avoid synthesizing formal parameter names or signatures that inference does not retain.
The design's illustrative full signature is not a requirement to invent missing context.

### M7: Workaround register and release evidence (~100 LOC)

**Estimate:** 100 documentation; 60 minutes. **Dependencies:** M4, M5, M6.

Update `docs/docs/guides/packages.md` with lock --check as CI gate and lock as the manifest
resolution/cache-population workflow on fresh machines. Update the approved register with
actual tested version/commit and W1–W4 statuses. Add
`changelogs/unreleased/2026-10-15-stapledon-workarounds.md` opening with `### `.
Record downstream instructions for sim/ship.ail readLineOpt migration and protocol tests
(blank, final partial line, sticky EOF), CI workaround deletion after upgrade, and fresh-binary
service-handler retest. Keep the ai/voice.ail doubling workaround pending #1576 closure.

- [ ] W1/W3 fixes have recorded local evidence; W2 downstream migration is explicitly pending.
- [ ] W4 links the existing B4/A2 dependency and requests confirmed-binary retest evidence.
- [ ] No doc alias or new readLine behavior/API is introduced; no issue is auto-closed.
- [ ] make check-file-sizes and make check-changelog pass; relevant examples/docs are accurate.

Risk: downstream repository is unavailable here; success means an actionable retirement
checklist, not a claim that the game changed. External failing evidence belongs on #1576.

## Day-by-day schedule

| Day | Work | Budget |
|---|---|---|
| 1 | M1 representation and M2 point operations | 3 focused hours |
| 2 | M3 bulk operations and M4 strict parity | 2.75 focused hours |
| 3 | M5 lock preservation and M6 diagnostics | 3 focused hours |
| 4 | M7 register/docs, focused integration checks and contingency | 1 hour + 3 hours reserve |

Critical path: M1 → M2 → M3 → M4 → M7. M5 and M6 are independent but scheduled
serially; this plan does not require parallel agents. If a milestone cannot fit a 90-minute
run, stop with concrete progress and split its remaining acceptance work before proceeding.

## AILANG syntax and examples gate

AILANG prompt version loaded: v0.16.7 (`ailang prompt`; `ailang prompt --version-active`).
This differs from source std/VERSION v0.54.9. Executor must build the checked-out source
using existing make targets, record version/commit and refresh its teaching prompt before
creating .ail fixtures. Do not use stale attached binary metadata as reproduction evidence.
No .ail source is written by this planning task.

| Showcase | Contracts | Effects | Inline tests |
|---|---|---|---|
| examples/runnable/map_strict_parity.ail | skip: map VM parity is covered by engine comparison, not SMT support | include: pure helpers/main, explicit empty row ! {} | include: pure result helper with tests [((), expected)] using validated syntax |
| existing examples/runnable/map_basic.ail | skip: preserve the existing example's API | skip: retain existing effect row, confirm pure map operations | skip: dedicated parity fixture and Go engine tests supply coverage |
| reporter fl repro (temporary test module) | skip: preserve exact reported shape | skip: preserve reporter's pure main signature | skip: exact CLI regression is the assertion |

## Validation and handoff

Each milestone runs focused tests for its packages and `make test-changed` (touched packages
plus test-core). Set TEST_CHANGED_BASE to the actual checked-out sprint base if origin/dev
is unavailable in the shallow cloud checkout; never silently omit changed packages.
Use targeted verification above, not the full make test suite, which is CI's responsibility.
Go edits are gofmt'd and relevant lint is run. Final focused gates include check-file-sizes,
check-changelog, check-boundaries, verify-examples, and verify-stdlib if std files change.
Avoid growing large files: new companions named above keep changes below the 800-line gate.

Success requires all seven acceptance sets, source-built strict engine evidence and a
populated JSON progress file with no completed milestones at initialization. No global
coverage threshold is invented. The executor retains each milestone's acceptance criteria
and records its test evidence as it progresses.

Coordinator artifacts are the Markdown plan and `.ailang/state/sprints/sprint_M-STAPLEDON-WORKAROUNDS.json`.
Plan approval/merge routes to sprint-executor through the coordinator; do not self-approve,
merge, or dispatch implementation while this new sprint is awaiting approval. The executor
then runs sprint-evaluator against the approved design and these criteria.

## Execution record (2026-10-15)

M1–M6 are committed on `coordinator/task-b299370e` (the execution checkout's branch).
The planner artifacts were recovered from `coordinator/task-3063755f`; no branch switch
or destructive git operation was used. M5/M6 ran independently while the VM chain ran
sequentially. Source-built CLI evidence uses v0.54.9 at code commit `0145da1ae`.
The VM native tests pass 20 repetitions; exact reporter and full map fixture agree on
both engines, including a map-valued entrypoint and strict XML refusal. Lock tests
cover empty/path dependencies, metadata across versions, malformed/unreadable files,
changed resolution and read-only checks. All downstream game work remains pending.

Examples, core, package, formatting, boundaries, sizes and changelog gates pass. The
initial changed-package suite's only failures were absent `ps` and zombie descendants
left under the container PID1; a temporary procps/tini environment rerun is in progress.
The initial examples manifest aggregate-count error was fixed and the gate rerun green.
Final lint and independent evaluation are pending; completion evidence is in sprint JSON.
