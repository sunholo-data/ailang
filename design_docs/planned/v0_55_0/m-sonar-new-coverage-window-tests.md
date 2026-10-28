# Bounded product tests for the v0.54.18→dev Sonar new-coverage window

**Status**: Planned
**Target**: v0.55.0 (in-flight window; see §Window Diagnosis)
**Priority**: P1
**Estimated**: 2–3 days (~17 focused hours, plus CI/Sonar turnaround)
**Dependencies**: None blocking. In-window code under test is already merged
(`internal/accessapproval` #1938, collections 1a/1c, M-VM-COVERAGE G2, M-VM-QUICK-WINS).
**Routing**: DESIGN-ONLY per triage iteration 17 (task `task-9843bd9f`). No
implementation or executor handoff in this step; the doc stops at the human-approval
gate. Verified by an independent OpenAI verifier and an Anthropic Sonnet judge
(CONCUR) before drafting.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Test-only change: no production semantics, syntax, or effects are altered. The
tests pin existing contracts; two of them pin *error* contracts that are
currently asserted nowhere.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pins deterministic contracts: list-order write semantics of `_array_accumulate`, canonical map-key encoding of forced lazy values, image-shape of the prelude prototype. |
| A2: Replayability | 0 | No trace/replay changes. |
| A3: Effect Legibility | 0 | No effect changes; the prelude test asserts the existing `__io_println` effect routing stays explicit. |
| A4: Explicit Authority | +1 | `accessapproval` tests pin the fail-loud guards (unsafe state dir, unregistered argument) rather than letting them stay silently unexercised. |
| A5: Bounded Verification | +1 | Moves verification of five in-window files from "only observable via full CI + Sonar" to local, package-bounded `go test` runs. |
| A6: Safe Concurrency | 0 | No concurrency changes; existing CAS tests are untouched. |
| A7: Machines First | 0 | No surface change. |
| A8: Minimal Syntax | +1 | No syntax; explicitly rejects CI semantics changes (`-coverpkg`) that would alter what the machine measures. |
| A9: Cost Visibility | 0 | No cost-model changes. |
| A10: Composability | 0 | Tests compose with existing in-package harnesses only. |
| A11: Structured Failure | +1 | Three new tests assert *error* contracts: out-of-bounds index naming index+length, `Lookup` on an unencodable key erroring (never a silent miss), `KeyWriter.Fail` propagation from a thunk that cannot force. |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

**Current State:**

- SonarCloud quality gate run `122959090663` on `dev` @ `07ce4a8d` (analysis
  `27c1dd2e-9d70-47b6-b3b1-bebb920bed21`) FAILs on exactly one condition:
  `new_coverage` **74.8% < 80%**. `new_reliability_rating` = 1 **passes** (live
  re-read this session via the sonarcloud-triage read-only API, §Verification
  Log V1).
- Pinned numbers reconcile exactly with the live per-file measures API (V2):
  **1431 new lines to cover, 361 uncovered → 74.77%**. To pass the preserved 80%
  floor, covered lines must go from 1070 to ≥ ⌈0.8 × 1431⌉ = 1145 — i.e. **≥ 75
  additional new lines must be covered**.
- The window is legitimate, not a misconfiguration (§Window Diagnosis): 115
  commits since the previous-version baseline tag `v0.54.18` (2026-10-27).
- This is a **coverage gap**, not a proved production defect and not a
  latest-commit regression: the same gate was green on the immediately
  preceding Sonar-success PR heads, and no failing reliability/security
  condition exists.
- **Profile-completeness diagnosis (the load-bearing finding):** CI's coverage
  profile is per-package only — `go test ./... -coverprofile` with **no
  `-coverpkg`** (V3) — so behavior tested from a *different* package is
  invisible to the profile. Three of the five top uncovered files are
  behavior-tested in other packages today:
  - `internal/bytecode/compiler/prelude.go` (24/25 uncovered) — behavior is
    tested by `internal/runner/vm_prelude_test.go` (153 LOC: bare `println`
    runs natively, library rejection, drift test), which runs whole programs
    through the VM but credits only `internal/runner`.
  - `internal/eval/value_mapkey.go` (24/34) and `internal/bytecode/value_map.go`
    (27/33) — every structured key kind is golden-tested byte-for-byte by
    `internal/vm/mapkey_cross_engine_test.go`, and `MapObj` methods are
    exercised via the VM map builtins in `internal/vm` tests; none of that
    credits the defining packages.
  - The remaining two are genuine gaps: `internal/builtins/array_accumulate.go`
    (25/33) is exercised only by `examples/runnable/difference_array_accumulate.ail`
    through the built binary (invisible to the profile by construction), with
    **no unit test anywhere in the repo** (V6); `internal/accessapproval/service.go`
    (34/162) has 13 in-package tests but 34 uncovered branches (V2/V15).

**Impact:**

- `dev` carries a red (non-required for merge, but reported) Sonar gate; per the
  standing v1-mission-log reading, a `new_coverage` red is "a machine naming
  shipped lines nothing exercises" — left alone, the next releases keep
  re-accumulating the same artifact.
- The per-package blind spot will keep recurring: every feature whose product
  behavior is tested from a sibling package (the VM lane's established pattern)
  re-creates uncovered-looking files on every release window.

## Window Diagnosis (measurements, not assumptions)

1. **Window definition** — `sonar.leak.period = previous_version`, read live (V1).
   `sonar.projectVersion` is computed in CI as `git describe --tags --abbrev=0`
   (V3), so the window opens at the last release tag. Baseline `v0.54.18`
   (2026-10-27) is correct and **must not be rebased** (a `set_new_code` call is
   a policy/baseline change this doc explicitly forbids).
2. **Window contents** — `v0.54.18..HEAD` = 115 commits (M-VM-QUICK-WINS,
   M-VM-COVERAGE G1–G4, collections 1a/1c, access approvals #1938, prompt
   v0.16.8, fleet/i66…). ~7.3k added non-test Go lines in `cmd`/`internal`;
   Sonar narrows these to 1431 executable "lines to cover" across 67 files (V2).
3. **Profile completeness** — the profile reaches exactly the packages with
   `TestGoFiles` (V4, `make/coverage.mk` `COVERAGE_PKGS_CMD`); all five target
   packages have in-package test files, so **in-package tests will register**.
   `sonar-project.properties` excludes `**/*_test.go` from sources (V10), so
   adding tests does **not** widen the window.
4. **Top uncovered contributors** (live measures API, V2):

   | file | uncovered / to-cover | tested anywhere? |
   |---|---|---|
   | `internal/accessapproval/service.go` | 34 / 162 | in-package suite exists; 34 branch lines unexercised anywhere |
   | `internal/bytecode/value_map.go` | 27 / 33 | cross-package only (`internal/vm`) |
   | `internal/builtins/array_accumulate.go` | 25 / 33 | **nowhere** (example runs via binary) |
   | `internal/eval/value_mapkey.go` | 24 / 34 | cross-package only (`internal/vm` golden) |
   | `internal/bytecode/compiler/prelude.go` | 24 / 25 | cross-package only (`internal/runner`) |
   | *next tail* (`compiler/expr.go` 22, `letrec.go` 20, `vm/vm.go` 20, `vm/closure_call.go` 20) | ~84 | mixed |

   The five named files carry 134 of the 361 uncovered lines. A bounded
   ~90-line test increment (below) yields projected `new_coverage ≈ 81.1%`
   (1160/1431), clearing 80% with margin; the tail files are named as
   contingency only.

## Goals

**Primary Goal:** Close the v0.54.18→dev `new_coverage` gap (74.8% → ≥80%) with
bounded, meaningful in-package product tests for the five top uncovered files —
without weakening the 80% policy, rebasing the baseline, adding coverage
exclusions, or writing empty/mirrored tests.

**Success Metrics:**
1. Sonar `new_coverage` ≥ 80% on the next `dev` analysis (and reliability stays 1).
2. ≥ 75 previously-uncovered new lines covered by tests that assert behavior
   (each test states a contract; none executes code solely for profile credit).
3. Zero changes to: `sonar-project.properties`, CI coverage plumbing,
   `make/coverage.mk`, the five production files' semantics.
4. The D-TRIAGE-21 `go:S1764` reliability HOLD is untouched and unchanged.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Fix via in-package tests, NOT `-coverpkg=./...` in CI | Changing the profile's semantics alters what the preserved 80% policy *measures* (cross-package credit would inflate the repo-wide 29% gate too); it is a policy-adjacent change needing its own doc | human (this doc proposes; ratification at approval gate) | design | high |
| D2: No mirrored re-execution of the cross-engine golden | Re-running `internal/vm`'s golden cases in `internal/eval`/`internal/bytecode` would be a mirror test — forbidden by the task and by test honesty | agent (this doc binds it) | design | low |
| D3: Scope = the five named files only; tail files are contingency | Bounded by task; prevents gate-chasing sprawl into 20+ files | human (approve scope) | design | low |
| D4: If a new test finds a real production defect, STOP and route to triage — do not fix in this sprint | Keeps the test-only guarantee; a defect fix is a separate gate (trivial-fix/feature routing) | agent (process rule) | runtime | low |
| D5: Tests land before the next release tag | After a re-tag, the window resets and the gate greenness is moot — but the tests remain durable product value either way | agent (sequencing note, not a hard gate) | runtime | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] D1 ratified: in-package tests over `-coverpkg` (reject-by-default reviewers
  should specifically confirm they are not being asked to bless a measurement-semantics change)
- [ ] D3 scope confirmed by human at the approval gate

## Solution Design

### Overview

Add one bounded test file (or additions to existing ones) per target package,
asserting contracts that no existing test anywhere asserts. No production code
changes. The projection is ~90 newly covered lines against a 75-line
requirement; the two largest levers are the genuinely-untested
`array_accumulate` builtin and the tested-but-invisible compiler prelude.

### Architecture

**Components:**

1. **`internal/builtins/array_accumulate_test.go` (new, ~120 LOC)** — the only
   file with zero existing unit coverage anywhere. Follows the established
   in-package callback-builtin pattern (`newTestEffCtx`/`goFn` from
   `list_iterative_test.go`, V11) so `FnCallerN` really calls back into a
   combine function:
   - Histogram via `+` combine and in-list-order double-write semantics
     (write j's combine sees write j−1's result — the property that separates
     `accumulate` from `update_many`'s later-wins, and the documented reason
     both exist).
   - Input-shape rejections: non-`Array` arg 0, non-`List` arg 1, non-pair write,
     non-int index, `FnCallerN`-nil ("evaluator not wired").
   - Out-of-bounds error **names the index and the array length** (the doc
     comment's contract; currently asserted nowhere).
   - Combine-callback error propagates wrapped with the failing write index.
   Expected: ~25 lines covered (the whole executable body).

2. **`internal/accessapproval/service_test.go` (additions, ~150 LOC)** — 34
   uncovered branch lines in a file whose happy paths are well-tested. Cover
   the *guard* branches, each a fail-loud contract (CLAUDE.md §2):
   - `Handle` unknown command → `invalid_command` naming the valid commands.
   - `load` on a store row with corrupt `ContextJSON` → `invalid_record`
     (write the corrupt row through `coordinator`'s store API directly, then `get`).
   - `Open` unsafe-state-dir guards: state dir with group/other permission bits;
     `coordinator.db` as a non-0600 regular file (chmod + reopen).
   - `request` expiry-window guards: `expires_at` in the past; beyond 7 days.
   - `list` validation: `limit` 0→default, `limit > 50`, negative offset, bad status.
   - The `mutate` 32-retry CAS-exhaustion `conflict` branch is **excluded**:
     forcing it deterministically requires racing a second writer between load
     and CAS; a racy test would violate the repo's determinism bar. Recorded as
     a known-remaining-uncovered line (see Deferred Decisions).
   Expected: ~25 of the 34 lines.

3. **`internal/bytecode/value_map_test.go` (additions, ~80 LOC)** — the existing
   single test covers `Insert`/`String`/`Equal` basics. Add `MapObj`-level unit
   contracts (engine-level tests in `internal/vm` exercise these only through
   the bridge/builtins, with different assertions — not mirrors):
   - `Lookup` on an **unencodable key errors** — "never a silent miss" is the
     file's own doc-comment contract; unit-untested anywhere.
   - `Remove` of a missing key returns the **same** `MapObj` (identity, no copy);
     `Remove` of an existing key leaves the original untouched.
   - `Equal` actually consults the caller's comparator (comparator that returns
     false on identical values must make `Equal` false — proves it isn't
     structural-equality short-circuiting).
   - `AsMap` on a non-map `Value` panics with the tag in the message (recovered
     panic assertion).
   Expected: ~15 lines.

4. **`internal/eval/value_mapkey_test.go` (new, ~60 LOC)** — the cross-engine
   golden in `internal/vm` covers every key *kind*; per D2 we do not re-run its
   cases. The one arm with **no test anywhere** (V8) is `*IndirectValue`:
   - A lazy thunk as a map key **forces before encoding** (key equals the
     forced value's key).
   - A thunk whose `Force()` errors propagates the error through
     `pcoll.KeyWriter.Fail` (the `Fail` API exists precisely for this, V8).
   These are real evaluator properties (laziness interacting with the key
   grammar), not golden copies. Expected: ~6 lines. The remaining
   tuple/record/ADT arms stay uncovered **by decision** — covering them
   in-package would mirror the golden (D2), and this doc prefers an honest 81%
   over a mirrored 84%.

5. **`internal/bytecode/compiler/prelude_test.go` (new, ~90 LOC)** — behavior
   is tested end-to-end in `internal/runner`; the **image-shape contract** of
   the synthetic prototype is asserted nowhere. Using the existing in-package
   `Compile(&stmt.Program{...})` harness (V12), compile a function whose body is
   a bare `println(...)` call and assert on the produced image:
   - exactly one prototype keyed `$prelude.println` (a key no source identifier
     can spell), arity 1, registered once — and calling `preludeFunc` again
     returns the cached key (memoization branch).
   - its body dispatches to the `__io_println` effect builtin (the capability
     gate therefore applies — this is the security-relevant part of G2).
   - an unknown bare name returns `ok=false` without compiling anything.
   Expected: ~20 of 25 lines.

### Implementation Plan

**Phase 0: Measure-first (~2 hours)**
- [ ] Reproduce per-file uncovered lines locally: run the five packages'
      tests with `-coverprofile` (or reuse CI's `coverage/coverage.out`),
      `go tool cover -html` / `-func`, and tick each proposed test against the
      actual uncovered set — adjust targets if a branch assumed uncovered is
      already covered (this doc's branch lists come from code reading + the
      Sonar per-file numbers, not line-level Sonar data).
- [ ] Re-read the live measures API after any rebase of `dev` (numbers drift
      with every merge; the 1431/361 split is pinned to analysis `27c1dd2e`).

**Phase 1: The genuine gap (~4 hours)**
- [ ] `array_accumulate_test.go` (Component 1)
- [ ] `value_mapkey_test.go` IndirectValue tests (Component 4)

**Phase 2: Guard branches (~5 hours)**
- [ ] `service_test.go` additions (Component 2)
- [ ] `value_map_test.go` additions (Component 3)

**Phase 3: Image-shape contract (~4 hours)**
- [ ] `prelude_test.go` (Component 5)

**Phase 4: Verify at the gate level (~2 hours + CI/Sonar turnaround)**
- [ ] `make test` + `make test-coverage` locally; per-file covered-line delta
      printed in the PR body (before/after table).
- [ ] CI green including `check-coverage-threshold` (overall 29% floor — only rises).
- [ ] Sonar `new_coverage` ≥ 80 on the PR analysis; confirm reliability stays 1.
- [ ] Confirm zero diff in `sonar-project.properties`, `ci.yml`, `make/*.mk`.

### Files to Modify/Create

**New files:**
- `internal/builtins/array_accumulate_test.go` — product semantics + error contracts (~120 LOC)
- `internal/eval/value_mapkey_test.go` — lazy-force key encoding (~60 LOC)
- `internal/bytecode/compiler/prelude_test.go` — synthetic-prototype image shape (~90 LOC)

**Modified files:**
- `internal/accessapproval/service_test.go` — guard-branch tests (~150 LOC added)
- `internal/bytecode/value_map_test.go` — MapObj unit contracts (~80 LOC added)

**Production files touched: none.** If any test exposes a defect (D4), the sprint
stops and the finding is routed to triage with the failing test parked.

## Examples

### Example 1: order-sensitivity contract (Component 1)

```go
// Two writes to index 1 in list order: the second combine must see the
// first combine's result — accumulate, not update_many's later-wins.
out, err := arrayAccumulateImpl(newTestEffCtx(), []eval.Value{
    intArray(1, 1, 1),
    pairList(intPair(1, 10), intPair(1, 100)),
    goFn(func(xs []eval.Value) (eval.Value, error) { /* x+y */ ... }),
})
// want [1, 111, 1]  — 1+10 then +100, not [1, 100, 1]
```

### Example 2: never-a-silent-miss (Component 3)

```go
// Lookup with an unencodable key must ERROR — returning (zero,false,nil)
// here would silently treat "cannot key" as "not found" (CLAUDE.md §2).
_, _, err := m.Lookup(NewList(nil))
if err == nil { t.Fatal("unencodable key silently missed") }
```

### Example 3: prelude image shape (Component 5)

```go
img, err := Compile(progWithBarePrintln())
key, ok, err := fc.preludeFunc("println") // or assert via img lookup
// exactly one prototype named "$prelude.println", arity 1,
// body = effect builtin __io_println — capability-checked like std/io.
```

## Success Criteria

- [ ] Sonar `new_coverage` ≥ 80.0 on the next `dev` analysis after merge; `new_reliability_rating` remains 1 (acceptance: quality-gate API read linked in PR)
- [ ] ≥ 75 previously-uncovered window lines covered, itemized in a PR-body table (file → branch → test name)
- [ ] Every new test asserts a contract stated in a comment; no test whose body only executes code (acceptance: reviewer check; gatelint/`check-skills` unaffected)
- [ ] No mirrored golden cases from `internal/vm/mapkey_cross_engine_test.go` (acceptance: grep for its golden key strings in the new tests returns nothing)
- [ ] All tests passing (`make test`); overall coverage threshold (29%) still green
- [ ] Zero production-code diff; zero changes to `sonar-project.properties`, `ci.yml`, `make/coverage.mk`
- [ ] D-TRIAGE-21 (`go:S1764` reliability HOLD) unchanged — no issue transitions, no `mark_fp`/`mark_safe` calls in this sprint
- [ ] Documentation: this doc updated with the measured before/after and moved per the design-doc-creator flow when shipped

## Testing Strategy

**Unit tests:** the deliverable itself (Components 1–5), all runnable per-package
with `go test ./internal/<pkg>` in <15s each (aligned with `make test-core`'s
inner-loop budget).

**Integration tests:** none added; the existing cross-package tests
(`internal/vm` goldens, `internal/runner` VM runs, `verify-examples`) remain the
integration layer and are explicitly NOT duplicated.

**Manual verification:** after merge, read the quality-gate API for the `dev`
head and record the `new_coverage` value in this doc's implementation report;
post the run link to the triage bookkeeping thread.

## Verification Log

Every load-bearing claim above, with its evidence (all commands run this
session on `dev` @ `07ce4a8d`; Sonar reads via the sonarcloud-triage skill's
read-only API):

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Gate fails only on `new_coverage` 74.8 < 80; period `previous_version`; reliability 1 passes | `gate_status.sh` live output this session |
| V2 | 1431 to cover / 361 uncovered = 74.77%; per-file table incl. the five named files and their uncovered counts | `api/measures/component_tree` (metricKeys `new_lines_to_cover,new_uncovered_lines,new_coverage`, 4×500 leaves) summed to exactly the pinned totals |
| V3 | CI coverage pass has no `-coverpkg`; `sonar.projectVersion` = `git describe --tags --abbrev=0`; full clone for new-code attribution | `.github/workflows/ci.yml` "Run tests", "Compute Sonar projectVersion", checkout comment (read) |
| V4 | Profile includes only packages with `TestGoFiles`; gate threshold 29% overall (separate from Sonar's 80% new-code) | `make/coverage.mk` `COVERAGE_PKGS_CMD`, `COVERAGE_THRESHOLD := 29` (read) |
| V5 | All structured key kinds of both MapKey twins are golden-tested cross-engine (byte-identical) | `internal/vm/mapkey_cross_engine_test.go:66–133` (read; cases: int, string, bool, unit, float incl. −0.0, tuple, record, nullary/std ctor, nested; unsupported: function, list, map, NaN) |
| V6 | NEGATIVE: no unit test for `_array_accumulate` exists anywhere | `grep -rn "_array_accumulate\|arrayAccumulate" internal/ --include="*_test.go"` → 0 hits; only runner is `examples/runnable/difference_array_accumulate.ail` (exists, read) driven by the built binary via `verify-examples` (V9 pattern) |
| V7 | Compiler prelude behavior is tested cross-package; NEGATIVE: no in-package compiler test references it | `internal/runner/vm_prelude_test.go` (read: `TestPreludePrintlnOnVM` runs real programs on the VM, `TestPreludePrintlnStillRejectedInLibrary`, `TestPreludeSurfaceHasVMImplementation`); `grep prelude internal/bytecode/compiler/*_test.go` → 0 hits |
| V8 | NEGATIVE: `*IndirectValue` arm of `eval.writeMapKey` is untested anywhere; `KeyWriter.Fail` exists for the force-error path | `grep -rn "IndirectValue" internal/vm/*_test.go internal/eval/*_test.go` → 0 hits; `internal/pcoll/key.go:91` `Fail` (read) |
| V9 | The five target packages all have in-package test files, so new tests register in the profile | `ls`: `internal/accessapproval/*_test.go` (3 files), `internal/bytecode/value_map_test.go`, `internal/builtins/*_test.go` (many), `internal/bytecode/compiler/*_test.go` (16), `internal/eval/map_value_test.go` |
| V10 | Adding tests does not widen the Sonar window (tests excluded from sources); no target file is in `sonar.coverage.exclusions` | `sonar-project.properties`: `sonar.exclusions` `**/*_test.go`; coverage exclusions list = `cmd/ailang/**, internal/server/**, internal/storage/firestore/**, internal/agentprompt/**, internal/devtoolsprompt/**, internal/docsearch/**` (read; none of the five appear) |
| V11 | Established in-package harness for callback builtins (`FnCallerN` wiring) exists | `internal/builtins/list_iterative_test.go:16–35` `newTestEffCtx` (read) |
| V12 | Established in-package compile harness exists for the prelude image-shape test | `internal/bytecode/compiler/compiler_test.go:17–40` `runCompiled`/`Compile(&stmt.Program{...})` (read) |
| V13 | Gate arithmetic: need ≥ 75 newly covered lines (⌈0.8×1431⌉=1145; current 1070) | arithmetic from V2 |
| V14 | `dev` @ `07ce4a8d` is 115 commits after `v0.54.18`; the tag exists and is an ancestor | `git rev-list --count v0.54.18..HEAD` = 115 after `--unshallow` (workspace arrived shallow; fetched, non-destructive) |
| V15 | `accessapproval` in-package tests exist and still leave 34 uncovered lines | 13 `func Test` across `service_test.go`/`concurrency_test.go`/`crash_test.go` (read); Sonar 34/162 (V2) |
| V16 | Overall project coverage 70.1%, 29% make-gate unchanged by test additions | `gate_status.sh`; `make/coverage.mk` |

**Known-limitation honesty:** the *branch-level* lists in §Solution Design
derive from reading each file plus the per-file Sonar counts; line-level
Sonar coverage was not pulled. Phase 0's measure-first step exists precisely to
reconcile this before any test is written, and the ~90-line projection carries
margin over the 75-line requirement.

## Conflict Surface

Test-only design: no parser/typechecker/codegen/eval semantics are touched, so
no syntactic position is extended and no fixture can change meaning. What must
still hold (each verified to exist where cited):

- `internal/vm/mapkey_cross_engine_test.go` goldens stay the single authority
  for cross-engine key bytes (this doc adds no competing golden).
- `internal/runner/vm_prelude_test.go` stays the end-to-end authority for bare
  `println` on the VM (the new compiler test asserts image shape only).
- `examples/runnable/difference_array_accumulate.ail` and the `verify-examples`
  gate stay green (the new builtin test must not change registry behavior —
  registration is `init()`-time and untouched).
- All 13 existing `accessapproval` tests keep passing; new tests must not
  weaken `Open`'s privacy guards to make setup easier.
- Deliberate incompatibilities: none.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact harness reuse vs. local helpers in each new test file — agent may
  choose, following each package's existing conventions.
- Whether `service_test.go` additions live in `service_test.go` or a new
  `service_guards_test.go` — agent may choose by file-size hygiene.
- The `mutate` CAS-exhaustion branch (32-retry conflict): left uncovered unless
  the implementer finds a *deterministic* way to force it (e.g., a store-level
  seam); a racy test is not acceptable — agent may resolve, default is skip.
- If the projected ~90 lines land under 75 after Phase 0 measurement (e.g.,
  some assumed branches are already covered), the contingency order is
  `compiler/expr.go` (22), `letrec.go` (20), `vm/closure_call.go` (20) —
  agent may pick the smallest set that clears 80% with the same
  no-mirror rule; anything beyond that returns to the human gate.

## Non-Goals

**Not attempted in this feature:**
- **Changing the CI coverage profile (`-coverpkg`)** — would change what the
  preserved 80% policy measures (and the repo-wide 29% gate); needs its own
  design doc and human gate. Recorded in Future Work, not smuggled in here.
- **Rebasing the new-code baseline** (`set_new_code.sh`) or touching the 80%
  threshold — explicitly forbidden by the task and this doc.
- **Any Sonar disposition changes** (`mark_safe`/`mark_fp`, hotspot reviews) —
  the gate red is a coverage condition, not an issue triage; D-TRIAGE-21
  (`go:S1764` reliability HOLD) is a separate, unchanged ruling outside this
  doc's scope.
- **Mirrored tests** — no re-execution of the cross-engine golden's cases in
  `internal/eval`/`internal/bytecode`; no empty "coverage-only" tests.
- **Production-code changes of any kind**, including refactors "to make it
  testable" (e.g., injecting a clock seam into `accessapproval` beyond the
  existing `s.now` hook) — that would be a different scope and gate.
- **Chasing the remaining tail files** beyond the named contingency — the goal
  is durable product tests for the named gaps, not gate-chasing across 67 files.

## Timeline

**Week 1** (~17 hours over 2–3 working days):
- Phase 0 measure-first (2h) → Phase 1 genuine gaps (4h) → Phase 2 guards (5h)
- Phase 3 image shape (4h) → Phase 4 gate verification (2h + CI/Sonar turnaround)

**Total: ~17 focused hours.** (Estimate follows the 2× realism rule from ~8h of
raw work, which includes reading each target file's uncovered branches.)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `dev` moves; window numbers drift before merge | Med | Phase 0 re-reads the measures API at sprint start; the 75-line requirement is re-derived from the live analysis, not the pinned one |
| Next release tag resets the window, making the gate moot mid-sprint | Low | Tests are durable product value regardless; D5 notes sequencing — land before the next tag if practical |
| A new test exposes a real defect in a target file | Med | D4: stop, route to triage; do not fix in a test-only sprint |
| Branch lists in this doc are wrong (already covered / uncoverable) | Med | Phase 0 measure-first reconciles before writing; 90-line projection carries 15-line margin over the requirement, plus named contingency files |
| Reviewer reads this as gate-gaming | Med | Every test maps to a stated contract; no-mirror rule is greppable (Success Criteria); production diff is zero |

## Related Documents

**Implemented (may inform design):**
- [m-sonar-gate-cleanup](../../implemented/v0_14_1/m-sonar-gate-cleanup.md) — the Sonar gate triage precedent (issue cleanup; this doc covers the *coverage* condition, a different lever)
- [m-persistent-collections](m-persistent-collections.md) — collections 1a/1c: shipped the MapKey twins and `_array_accumulate` this window must cover

**Planned (check for overlap):**
- [m-vm-coverage-letrec-prelude-bridge](m-vm-coverage-letrec-prelude-bridge.md) — M-VM-COVERAGE G2 shipped `compiler/prelude.go`; its tests live in `internal/runner` (cross-package), which is why the file reads uncovered here
- [m-vm-quick-wins](m-vm-quick-wins.md) — sibling in-window work; its `vm.go`/`closure_call.go` tails are this doc's contingency scope only
- [m-contract-verification-coverage](../m-contract-verification-coverage.md) — language-level verification coverage (different meaning of "coverage"; checked for overlap, none)

**Dedupe gate:** the create-script search (SimHash + neural) returned no
matches ≥ 0.45 for this topic; the manual sweep above found no doc proposing
in-package tests for the new-coverage window. No duplicate.

## References

- Gate run: https://github.com/sunholo-data/ailang/runs/122959090663
- Pinned analysis: https://sonarcloud.io/api/qualitygates/project_status?analysisId=27c1dd2e-9d70-47b6-b3b1-bebb920bed21
- [sonarcloud-triage skill](../../../.agents/skills/sonarcloud-triage/SKILL.md) — read-only API used for V1/V2
- [make/coverage.mk](../../../make/coverage.mk) — profile derivation and the 29% overall gate
- [CLAUDE.md §2](../../../CLAUDE.md) — no-silent-fallback rule that three of the new tests pin
- D-TRIAGE-21 (`go:S1764` reliability HOLD) — triage charter decision ledger; separate and unchanged

## Future Work

- **`-coverpkg` profile semantics** — if the tested-but-invisible pattern
  recurs across release windows (three of five files this window), a dedicated
  design doc should weigh per-package honesty vs. cross-package credit, and
  what it does to both the 80% new-code floor and the 29% overall floor. That
  doc needs its own human gate; this one deliberately does not attempt it.
- **Line-level Sonar coverage in triage** — the sonarcloud-triage skill could
  gain a `fetch_uncovered_lines.sh` so future iterations get branch-level data
  instead of code-reading (the Phase 0 friction this doc hits).
- **`mutate` CAS-exhaustion coverage** — requires a store-level test seam; only
  worth a design row if `accessapproval` gains another uncovered-window report.

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28
**Author**: design-doc-creator, triage iteration 17 routing (task `task-9843bd9f`)
**Next gate**: human approval → `sprint-planner` (not requested in this task; doc stops here by design)
