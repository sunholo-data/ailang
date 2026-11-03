# M-PROPERTY-OUT-OF-CONTRACT-VACUITY: honest discard counters and a non-green rule for properties that never ran an in-contract case

**Status**: Planned (awaiting human design approval — freeze items D3/D4 below carry open decision **D-TRIAGE-25**; this document does not resolve it. No implementation, sprint planning, release or deployment is authorized by this document.)
**Target**: v0.57.1
**Priority**: P0 (data integrity of the `ailang test` green signal — a suite can today test nothing and read `success: true`)
**Estimated**: ~2 days (M1 + M2 + tests/docs; ≈90 LOC in `internal/testing`, ≈60 LOC of test updates, one CLI e2e fixture)
**Dependencies**: None. Supersedes no shipped milestone; extends the M-DX26 Phase 5.2 requires-path semantics and closes Future Work item F1 of [m-property-generator-coverage.md](../../implemented/v0_31_0/m-property-generator-coverage.md).
**Author**: design-doc-creator, coordinator task `task-a08bfe4d` (issue #1921, reported by stapledons_godot), 2026-11-03
**Source baseline**: HEAD `b23cadced75c7bbf1328eebcc5ab011453fba672` (v0.57.0) — all line numbers and live reproductions below are first-party on this commit unless marked "reporter-provided".
**Review status**: no off-Anthropic quorum run — attempted 2026-11-03, both shipped reviewers
unavailable from this environment and recorded by name per the no-silent-pass rule:
`gpt5-6-sol` (openai) absent_reason `auth` (no `OPENAI_API_KEY`); `gemini-3-1-pro`
absent_reason `unreachable` (Vertex ADC 403 on `aiplatform.endpoints.predict`). Quorum is an
optional pre-sprint step; the in-session verification log below is the load-bearing gate. The
human D-TRIAGE-25 ruling (Design Freeze) remains required before any sprint.

---

## Problem Statement

`ailang test` (file, multi-file and package modes) can report a fully green
suite — `success: true, failed_tests: 0, vacuous_skips: 0` — while the majority
of its property tests never executed a single in-contract case. Three
mechanisms compose into the hole (all verified on the v0.57.0 binary at HEAD,
see [Verification Log](#verification-log)):

1. **The `requires`-clause property path has no discard-and-retry loop.**
   `runRequiresProperty` (`internal/testing/runner.go:393`) generates one input
   at a time, and the **first** input that violates the function's `requires`
   predicate ends the property immediately as a Skip
   (`runner.go:499-508`). It never redraws. For contrast, the `ensures` path
   (`runEnsuresProperty`, `internal/testing/contract_domain.go:22`) already
   draws up to `maxAttempts = 1000` inputs, needs `requiredAccepted = 100`
   in-contract cases, and counts every rejection in `DiscardedInputs`
   (`contract_domain.go:86-87, 92, 129`). The requires path is the one M-DX26
   Phase 5.2 path that never got the retry semantics its sibling shipped with.

2. **The `requires` path's counters are dishonest at the moment of the skip.**
   At the skip it sets `result.TestsRun = testNum + 1`
   (`runner.go:506`) — so a property that executed **zero** in-contract cases
   reports `tests_run: 1` (or more) — and it never increments
   `DiscardedInputs` (the only writer in the codebase is
   `contract_domain.go:129`, verified by grep). A skip therefore reads as
   "1 test ran, nothing was discarded" when in fact nothing was tested and one
   input was silently thrown away.

3. **Out-of-contract skips are exempt from the vacuity count.**
   `SuiteResult.AddPropertyResult` (`internal/testing/result.go:118-127`)
   increments `VacuousSkips` for every skip **except**
   `SkipKindOutOfContract` (`result.go:122-124`), and `Success()`
   (`result.go:128-134`) requires `VacuousSkips == 0`. A suite whose
   properties all skip `out_of_contract` therefore reads green as long as any
   sibling test passes. This exclusion is a deliberate, test-pinned decision
   (see [Conflict Surface](#conflict-surface)) — but its blind spot is exactly
   the case the exclusion was not meant to forgive: a property that never ran
   an in-contract case at all.

**First-party reproduction** (v0.57.0 binary, HEAD `b23cadc`, this session;
fixture inline in the Verification Log):

```
module repro/tight
export pure func expm1ish(x: float) -> float ! {}
requires { x <= 700.0 && x >= -700.0 }
{ x }
export pure func positive(n: int) -> int ! {}
requires { n >= 0 }
{ n }
test "sibling" { true }
export func main() -> int ! {} { 0 }
```

`ailang test --json --seed 0` reports:

```
{'success': True, 'passed_tests': 1, 'failed_tests': 0, 'skipped_tests': 2, 'vacuous_skips': 0}
expm1ish_property_1  skip  tests_run=5  generated_inputs=5   discarded_inputs=0  skip_kind=out_of_contract
positive_property_1  skip  tests_run=2  generated_inputs=2   discarded_inputs=0  skip_kind=out_of_contract
```

Both properties' domains are trivially reachable by the default generators
(float ∈ ±1000 gives |x| ≤ 700 a ~70% acceptance rate; int ∈ ±1000 gives
n ≥ 0 a ~50% rate), yet both skip on their first violating draw, report
misleading `tests_run`, report zero discards, and the suite reads green. This
is the reporter's exact shape: `tests_run = 1, discarded_inputs = 0` on the
first generated input.

**Measured on shipped examples** (same binary, this session):

- `examples/runnable/contracts/list_recursive_verify.ail`: all 6 properties
  skip `out_of_contract`; four of them are requires-path one-shot skips
  (`generated_inputs` 1–2, `discarded_inputs` 0);
  `extractBounded_property_4` exhausts 1000 attempts with **0 accepted**
  (`tests_run=0, discarded_inputs=1000`) — a fully vacuous property that
  today does not count in `vacuous_skips` (the suite is red only via the
  separate all-skipped sentinel).
- `examples/runnable/contracts/basic.ail`: 3 of 8 properties are requires-path
  one-shot skips (`absolute_property_1`, `increment_property_1`,
  `clamp_property_1`) beside ensures-path properties that pass the same
  contracts with 100 accepted cases and honest discard counts.

**Reporter-provided motivation** (issue #1921, sunholo/relativity 0.13.0,
ailang-packages main `7682890`, not independently re-measured here — the
mechanism it describes is fully reproduced first-party above): 94 of 139
generated properties skip `out_of_contract`; **53 skip after
`generated_inputs = 1, tests_run = 1, discarded_inputs = 0`**; for the seven
functions whose contracts 0.13.0 tightened (expm1, sinhc, accelerate, doppler,
dopplerApparent, cmbSeenTemperature, cmbSeenTemperatureApparent), 12 of 14
generated properties test no in-contract input at all. The package CHANGELOG
can then say "the whole-directory property suite now reports 0 failures" —
true only because most properties no longer execute. The package maintainer
(sunholo) should be notified of the semantic change when it lands (see
[Goals](#goals), success metric 5).

**Who is affected**: every package that gates CI on `ailang test` exit codes
(the green signal is currently over-reporting verification), every agent
(motoko fleet, eval harness) that treats `success: true` as "properties
passed", and humans reading CHANGELOG claims derived from it.

This is issue **#1921**, held as **NEEDS-DECISION** with open decision
**D-TRIAGE-25** in the triage ledger
(`design_docs/triage-mission.md:157`): *choose (A) minimum
executed-case/maximum discard-rate coverage policy, making near-empty
properties non-green; (B) zero accepted inputs only becomes vacuous; (C)
retain current documented behavior. Recommend A, with thresholds/exit contract
reviewed in a design. Default C/current semantics until an operator ruling.*
This document is that design review; it concretizes the options and puts the
policy choice in the [Design Freeze](#design-freeze) for human ruling. It does
not pick the policy on the loop's own authority.

## Goals

**Primary Goal:** a property suite can no longer read green while any of its
properties never executed a single in-contract case, and every property's
`tests_run` / `discarded_inputs` counters describe what actually executed.

**Success Metrics:**

1. A requires-clause property over a generator-reachable domain (acceptance
   ≥ ~10%) reaches 100 in-contract cases and **passes** — no skip — with
   `discarded_inputs > 0` and `tests_run = 100` (today: one-shot skip,
   `tests_run` misleading, `discarded_inputs = 0`).
2. A property that skips having executed **zero** in-contract cases
   contributes to `vacuous_skips` (whatever its `skip_kind`), so
   `success` is false and the exit code is 1 unless `--allow-skips` is
   passed.
3. The reporter's repro shape (`success: true` + a property with
   `tests_run: 1, generated_inputs: 1, discarded_inputs: 0`) is impossible:
   on the requires path `tests_run` counts executed in-contract predicate
   evaluations only, and any out-of-contract draw increments
   `discarded_inputs`.
4. All existing ensures-path behaviors are unchanged (AC1/AC2/AC3 property
   counters, `unverified:` skip text, discard accounting).
5. The sunholo/relativity maintainer is notified when this ships (coordinator
   comment on issue #1921) that 0.13.0's "0 failures" claim was an artifact
   of the old forgiveness, with the new expected suite outcome.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: The requires path gets the **same discard-and-retry loop as the ensures path** — shared package-level constants `requiredAccepted = 100`, `maxAttempts = 1000` (hoisted out of `contract_domain.go` so the two paths cannot drift), `DiscardedInputs` counted, `tests_run` = accepted cases only | Closes the one-shot skip; also removes the perverse incentive to loosen `requires` clauses so properties stop flaking | agent (mechanism parity with shipped ensures path) | design | med |
| D2: Requires-path counters become honest — on any terminal skip, `tests_run` is the number of executed in-contract predicate evaluations (0 for a first-draw violation), never `testNum + 1`; the cap-exhausted skip message uses the ensures-path `unverified:` format with exact counts | `tests_run` is a public JSON contract consumed by gates and by the vacuity rule in D3; the current value is factually wrong | agent | design | low |
| D3: **The vacuity policy (D-TRIAGE-25)** — this doc recommends option **B refined**: a Skip counts as vacuous iff it executed zero in-contract cases (`tests_run == 0`), for **all** skip kinds including `out_of_contract`; skips that executed ≥ 1 case stay forgiven and visible | Changes the `Success()`/exit-code contract of `ailang test` — a human-owned semantic policy reserved by m-property-generator-coverage.md F1 and D-TRIAGE-25 | **human (D-TRIAGE-25 ruling; see Design Freeze)** | design (freeze) | med |
| D4: The intentional exit-contract flip: a suite containing a property that never ran an in-contract case exits 1 (today 0 when a sibling passes); `requires_unreachable.ail` (AC3) changes expected exit 0 → 1 and `impossible_property_1` changes expected counters | Breaks any package gate that relies on impossible-contract properties reading green; `--allow-skips` remains the escape hatch | human (rides with D3) | design (freeze) | med |
| D5: The human-reporter summary line generalizes from "properties never ran (no generator)" to "never ran an in-contract case (no generator or unreachable contract)" | The current text is wrong once out-of-contract vacuity exists | agent | design | low |
| D6: Generator shaping from numeric `requires` bounds (reporter's third suggestion) is **explicitly deferred** — it changes generator semantics, needs its own conflict surface, and overlaps parked Lane B2 (user-supplied generators, BLOCKED on the deterministic fuel budget) | Scoping it in would turn an honesty fix into a constraint-solving feature | agent (deferral), human may overrule | design | low |

### Design Freeze

Before implementation begins, these must be resolved (checked = frozen). D3/D4
are the open human ruling **D-TRIAGE-25** (issue #1921); until it is recorded,
current (C) semantics stand and no implementation is dispatched:

- [ ] **D3 ratified (D-TRIAGE-25):** choose the vacuity threshold —
      **B-refined (recommended)**: vacuous ⇔ skip with `tests_run == 0`, all
      skip kinds; **A-strict (ledger's recommendation, alternative)**: every
      `out_of_contract` skip is non-green (threshold = `requiredAccepted`,
      i.e. a skip can never be forgiven); or **C**: retain current behavior
      (the hole stays, documented). See [Options](#d-triage-25-options)
      below for the trade-off analysis.
- [ ] **D4 ratified:** the exit-code flip for suites containing
      never-executed properties (AC3 fixture `requires_unreachable.ail`
      expected exit changes 0 → 1 under B-refined; under A-strict every
      out-of-contract skip flips) and the `--allow-skips` escape hatch are
      accepted.
- [x] D1/D2 frozen (mechanism parity + counter honesty; no semantic choice
      beyond D3/D4 depends on them).

#### D-TRIAGE-25 options

| Option | Suite green when… | Blast radius | Incentives |
|---|---|---|---|
| **C (current)** | any sibling test passes; out-of-contract skips always forgiven | none (hole stays) | Tighter contracts silently reduce coverage; CHANGELOG claims like relativity 0.13.0's "0 failures" remain possible |
| **B-refined (recommended)** | every skipped property executed ≥ 1 in-contract case | Flips only suites that already test nothing in some property (e.g. AC3 fixture, relativity's 53 one-shot skips post-M1 become passes, so the real relativity suite stays green *because its domains are reachable*) | Reachable domain → must actually run; unreachable domain → red, fix the contract or use `--allow-skips`; deleting `requires` to go green no longer helps (property then always runs) |
| **A-strict** | no out-of-contract skip at all (any skip ⇒ non-green) | Also flips under-sampled properties (e.g. `containsImpliesNonEmpty_property_2`: 24 accepted of 1000 attempts — real work, honest skip) | Strictest; pushes maintainers to widen generators or drop `requires` even when the domain is genuinely narrow — a perverse incentive against honest contracts |

The design recommends **B-refined**: it exactly closes the reported defect
("tests nothing and reads green") with the smallest contract change, keeps the
`tests_run == 0` predicate machine-decidable with zero new thresholds in
`Success()`, and leaves the under-sampled middle case (1 ≤ `tests_run` < 100)
visible in JSON (`skip_kind`, counters, `skipped_tests`) without making
honestly-narrow domains gate-breaking. The ledger's "Recommend A" targeted the
pre-M1 world where one-shot skips made every out-of-contract property
"near-empty"; with D1's retry loop landed, the near-empty case collapses to
either pass (reachable) or zero-executed (unreachable), which B-refined
catches. If the human ruling picks A-strict, the implementation delta is one
line in `AddPropertyResult` (see M2) plus a wider expected-flip sweep — both
options are fully specified here so the ruling does not block planning.

## Solution Design

### Overview

Two milestones, each independently committable and red-first testable:

- **M1 — Requires-path retry and counter honesty** (`internal/testing/runner.go`,
  `internal/testing/contract_domain.go`): give `runRequiresProperty` the
  ensures path's draw-discard-retry loop with the same shared constants, and
  make its terminal counters and skip text state what actually happened.
  Purely mechanism parity; no `Success()` change.
- **M2 — Vacuity accounting** (`internal/testing/result.go`,
  `internal/testing/reporter.go`, CLI e2e expectations): a Skip that executed
  zero in-contract cases counts in `vacuous_skips` regardless of
  `skip_kind` (B-refined), so `Success()` — which already requires
  `vacuous_skips == 0` — makes such suites non-green. Update the pinned unit
  rows and the human summary text. (`--allow-skips` already forgives all
  skips via `SuccessAllowingSkips()`, `result.go:141-145`; no change needed.)

M1 without M2 already defuses the relativity-class incident (reachable
domains run 100 real cases and pass), but leaves impossible-contract suites
green; M2 closes that. M2 without M1 would flag properties whose counters lie
(`tests_run: 1` on a zero-execution skip) — which is why M1 ships first.

### Architecture

**M1 — `runRequiresProperty` loop** (`internal/testing/runner.go:393-538`):

1. Hoist the loop constants out of `contract_domain.go:85-88` to
   package level (e.g. in `result.go` beside the result types, or a small
   `const` block in `runner.go` referenced by both files):
   `requiredAccepted = 100`, `maxAttempts = 1000`. One definition, two
   consumers — the systemic-fix rule (no future drift between paths).
2. Replace the `for testNum := 0; testNum < numTests; testNum++` loop
   (`runner.go:450-453`) with the ensures-path shape
   (`contract_domain.go:92`):
   `for result.TestsRun < requiredAccepted && result.GeneratedInputs < maxAttempts`.
   On each draw: `result.GeneratedInputs++`; generate values (existing
   `valueToLiteral`/`astExprToCore` plumbing unchanged); evaluate the
   predicate; on predicate **false** → `result.DiscardedInputs++` and continue
   (no immediate skip); on predicate true → `result.TestsRun++` (this is the
   only place the requires path increments `TestsRun` — the
   `result.TestsRun = testNum + 1` assignments at `runner.go:464/485/494`
   (error branches) and `:506` (the old skip branch) become the current
   accepted count, which on error paths equals the count of executed cases).
3. Terminal states: all `StatusFail` error branches unchanged except the
   `TestsRun` fix; after the loop, if `result.TestsRun < requiredAccepted`
   → `StatusSkip` + `SkipKindOutOfContract` + the ensures-path message
   format (`contract_domain.go:159-164`): `unverified: requires filter
   accepted %d of %d generated inputs; need %d (%d discarded)`. The old
   "requires not satisfied by random input (consider tighter generators)"
   one-shot text is removed (it described the first violating input, which is
   now a discard, not a terminal state). `StatusPass` when 100 accepted —
   unchanged meaning (100 in-domain predicate evaluations completed).
4. Cost note: worst case per requires property goes from ≤ 100 to ≤ 1000
   predicate evaluations — the ensures path already pays exactly this, the
   predicates are pre-lowered Core evaluated via
   `EvaluateRequiresHarnessFromCore`, and the run stays deterministic
   (seed-derived RNG, `newRNG`). Bounded, replayable, no wall-clock
   fallback (`runner.go:691-693`).

**M2 — vacuity rule** (`internal/testing/result.go:118-127`):

```go
case StatusSkip:
    sr.SkippedTests++
    // A skip that executed zero in-contract cases tested nothing, whatever
    // its skip_kind (no generator, unsupported contract, or an unreachable
    // requires domain). D-TRIAGE-25 ruling: B-refined.
    if result.TestsRun == 0 {
        sr.VacuousSkips++
    }
```

- Under B-refined this is the whole change: `no_generator` and
  `unsupported` skips never execute anything (`TestsRun == 0` by
  construction — they return before any draw), so they stay counted exactly
  as today; the only behavior change is `out_of_contract` skips with zero
  executed cases. Under A-strict, the condition becomes
  `result.TestsRun < requiredAccepted` (every out-of-contract skip — a skip
  can by construction only occur short of the bar) — the same one line.
- `Success()` (`result.go:128-134`) is untouched: `VacuousSkips == 0` is
  already in its predicate.
- Human reporter (`reporter.go:251-257`): summary line becomes
  `"%d properties never ran an in-contract case (no generator or unreachable contract) — use --allow-skips to permit"`.
  The per-property `reportPropertyHuman` (`reporter.go:212-215`) already
  prints accepted/discarded/generated for out-of-contract skips — unchanged.
- JSON surface: **no new fields.** `vacuous_skips` (`reporter.go:56`) and the
  per-property `tests_run`/`discarded_inputs`/`skip_kind`/`generated_inputs`
  already give a gate everything needed to see why (`tests_run == 0 &&
  status == "skip"` ⇒ vacuous). Minimal surface per the simplicity gates.

### Implementation Plan

**Phase 1: M1 — requires-path retry + honest counters** (~1 day)
- [ ] Hoist `requiredAccepted`/`maxAttempts` to shared package constants.
- [ ] Rewrite the `runRequiresProperty` loop per Architecture §M1.2-3.
- [ ] Update `internal/testing/runner_skip_kind_test.go`
      `TestRunContractProperties_SkipKinds`: the `requires { false }`
      property still asserts `StatusSkip` + `SkipKindOutOfContract` (unchanged)
      but now with `GeneratedInputs == 1000`, `DiscardedInputs == 1000`,
      `TestsRun == 0`.
- [ ] New red-first unit: a `requires { n >= 0 }` int property (≈50%
      acceptance, fixed seed) passes with `TestsRun == 100`,
      `DiscardedInputs > 0`, `GeneratedInputs > 100` — red today (one-shot
      skip), green after M1.

**Phase 2: M2 — vacuity accounting** (~0.5 day)
- [ ] `AddPropertyResult` rule change per Architecture §M2.
- [ ] Update `internal/testing/result_vacuous_test.go`: the "out of contract
      forgiven" row gains `TestsRun: 1` (preserving its meaning — a skip that
      did run in-contract cases stays forgiven) and a new row pins the
      flipped semantics: `out_of_contract, TestsRun 0 → vacuous (want 1)` —
      red against pre-M2 code, the mutation test for this milestone.
- [ ] `result_vacuous_success_test.go`
      `TestSuiteResult_Success_MixedOutOfContractPropertyPasses` keeps
      passing unchanged (its fixture carries `TestsRun: 1`);
      `TestSuiteResult_Success_AllSkippedOutOfContractFails` keeps passing
      (already expects `!Success()`); add the mixed zero-executed variant:
      sibling pass + `out_of_contract` skip with `TestsRun: 0` ⇒ `!Success()`
      (red today).
- [ ] `reporter.go` summary text + `reporter_vacuous_test.go` text assertion.

**Phase 3: CLI e2e + docs + sweep** (~0.5 day)
- [ ] Update `cmd/ailang/test_contract_domain_test.go` **AC3**
      (`TestContractDomain_AC3_UnreachableDomainIsUnverified`):
      `runContractFixture(..., 0)` → wantExit **1** (the `impossible`
      properties executed zero cases; `ok` passes so the suite is red via
      `VacuousSkips`, not via all-skipped). Disambiguate the property
      selection — after M1 **both** impossible properties have
      `generated_inputs == 1000`, so the current "last match wins" loop must
      select by `tests_run`/kind or by exact name. `impossible_property_1`
      (requires path) now expects `generated_inputs 1000,
      discarded_inputs 1000, tests_run 0` (today: `1/1/0` — the reporter's
      exact shape); `impossible_property_2` expectations unchanged
      (`1000/1000/0`, `unverified:` prefix).
- [ ] AC1 (`precond2.ail`), AC2 (`precond_negative.ail`) and
      `TestContractDomain_OrderedRequiresBothInDomainPass` (`ordered.ail`)
      unchanged — `big_property_1` (requires, ≈45% acceptance) and the
      `g` requires properties move from *skip-forgiven* to *pass*, which no
      current assertion pins and no exit code depends on (verify in the
      sweep).
- [ ] New fixture `internal/testing/testdata/contracts/requires_tight.ail`
      (module + `requires { x >= 0 }` int function + passing named test) with
      a CLI e2e asserting: exit 0, the requires property `pass` with
      `tests_run == 100`, `discarded_inputs > 0` — the honest-green
      counterpart to AC3's honest-red.
- [ ] Docs: `docs/docs/guides/testing.md` gains a "Skips, discards and
      vacuity" subsection documenting: what `tests_run` /
      `generated_inputs` / `discarded_inputs` mean on each path, the three
      skip kinds, the vacuity rule (post-ruling), and `--allow-skips`.
      Today the guide shows the JSON shape but never defines these fields
      (verified: only an example at `testing.md:402`).
- [ ] Sweep: run `ailang test --json --seed 0` over
      `examples/runnable/contracts/` and `internal/testing/testdata/` before
      and after; record every rc/counter delta in the implementation report.
      Expected deltas only: requires-path skip→pass, and AC3's exit 0→1.
      Any other delta is a defect.

### Files to Modify/Create

**New files:**
- `internal/testing/testdata/contracts/requires_tight.ail` — honest-green e2e fixture (~10 LOC)

**Modified files:**
- `internal/testing/runner.go` — `runRequiresProperty` loop rewrite + counter honesty (~50 LOC delta)
- `internal/testing/contract_domain.go` — constants hoisted to shared scope (~8 LOC delta)
- `internal/testing/result.go` — `AddPropertyResult` vacuity rule (~6 LOC delta)
- `internal/testing/reporter.go` — summary text (~2 LOC delta)
- `internal/testing/result_vacuous_test.go`, `result_vacuous_success_test.go`, `runner_skip_kind_test.go`, `reporter_vacuous_test.go` — pinned-semantics updates + new red-first rows (~50 LOC delta)
- `cmd/ailang/test_contract_domain_test.go` — AC3 exit/counter updates, new e2e (~40 LOC delta)
- `docs/docs/guides/testing.md` — skip/discard/vacuity section (~40 LOC)

## Conflict Surface

This change does not touch `internal/parser|lexer|ast|types|elaborate|iface|codegen|eval|vm|effects`
or `cmd/ailang/exec.go`, so the mandatory parser/typechecker Conflict Surface
does not apply in its syntactic form. The changed surface is the
`ailang test` **result contract**, which has its own consumers enumerated
here (the honest answer is not "no conflicts"):

**Positions touched:**
1. `SuiteResult.AddPropertyResult` skip accounting (`result.go:118-127`) —
   every producer of `PropertyResult` funnels through it (`RunSuite`,
   `runner.go:89-90`; verified: no other caller in `internal/` or `cmd/`).
2. `SuiteResult.Success()` / `SuccessAllowingSkips()` (`result.go:128-138`) —
   consumed by **both** CLI aggregates for the exit code
   (`cmd/ailang/test.go:149` file/multi-file mode, `:293` package mode).
3. `PropertyResult.TestsRun` semantics on the requires path — consumed by the
   JSON reporter (`reporter.go:117` `tests_run`), the human reporter
   (`reporter.go:206-208` "N cases"), and (post-M2) the vacuity rule itself.

**What else lives in these positions:**
- `no_generator` / `unsupported` skips: unaffected (zero-execution by
  construction; still vacuous under both B-refined and A-strict).
- The forall-property path (`runProperty`, `runner.go:281`): never skips
  mid-run (only `no_generator` before the loop); **deliberately unchanged** —
  it has no `requires` filter to discard against.
- The all-skipped sentinel (`AllSkipped()`, `result.go:135-139`) and the
  "NO TESTS RAN" human summary: unchanged; a zero-executed out-of-contract
  skip in an otherwise-passing suite is a *different* failure mode (vacuous,
  `--allow-skips`-forgivable) than a suite where nothing ran at all.
- Package-mode extras: `checkUntestedSources` (`cmd/ailang/test.go:301-304`)
  and `NamedBatchFailures` — orthogonal, untouched.
- The `vacuous_skips` JSON field (`reporter.go:56`) — read by external
  package gates (e.g. sunholo packages' CHANGELOG claims) and reserved for the
  eval-harness bank schema (m-property-generator-coverage Future Work F3 —
  still unimplemented, verified: no consumer outside `cmd/ailang/test.go`
  aggregation, `test.go:123/268`).

**Programs/fixtures that MUST still work (all verified to exist):**
- `internal/testing/testdata/contracts/precond2.ail` (AC1): ensures path
  passes 100 with discards — unchanged.
- `internal/testing/testdata/contracts/precond_negative.ail` (AC2): in-domain
  violation still fails with counterexample — unchanged.
- `internal/testing/testdata/contracts/ordered.ail`: comma-separated requires
  enumeration, `g_property_3` pass with discards — unchanged.
- `internal/testing/testdata/contracts/requires_unreachable.ail` (AC3):
  counters unchanged for the ensures property; **intentionally changes**
  exit 0 → 1 (freeze item D4).
- `examples/runnable/contracts/*.ail`: runnable examples; sweep in Phase 3
  catches any unexpected delta.
- Pinned unit semantics in `result_vacuous_success_test.go` (both fixtures
  carry `TestsRun: 1` — they survive B-refined unchanged).

**What deliberately changes (intentional incompatibilities):**
1. Suites where every property skips `out_of_contract` **and** a sibling test
   passes: exit 0 → 1, `success` true → false, `vacuous_skips` 0 → N
   (N = zero-execution skips). This is the point of the fix.
2. `requires_unreachable.ail`-shaped suites (impossible contract + passing
   sibling): same flip.
3. Requires-path JSON `tests_run` becomes ≤ its old value and truthful
   (counts executions, not draws); `discarded_inputs` becomes ≥ 1 whenever a
   draw was rejected. Any consumer diffing these fields sees changed values —
   that is the honesty fix, not a regression.
4. Under A-strict (if the ruling picks it): additionally, every suite with
   any out-of-contract skip (including under-sampled ones like
   `containsImpliesNonEmpty_property_2`) flips non-green.

## Examples

### Example 1: Reachable domain — skip becomes an honest pass

**Before** (measured, `basic.ail` `absolute_property_1`, requires
`x >= 0`, int ∈ ±1000, `--seed 0`):
```
{"status": "skip", "tests_run": 2, "generated_inputs": 2,
 "discarded_inputs": 0, "skip_kind": "out_of_contract",
 "error": "requires not satisfied by random input (consider tighter generators): x=-841"}
```

**After** (M1): the violating draw is discarded and redrawn (counts
illustrative — exact values are seed-derived and asserted by AC-M1a):
```
{"status": "pass", "tests_run": 100, "generated_inputs": ~200,
 "discarded_inputs": ~100, "skip_kind": ""}
```

### Example 2: Unreachable domain — green lie becomes an honest red

**Before** (measured, repro fixture + passing `test "sibling"`):
```
success: true, passed_tests: 1, skipped_tests: 2, vacuous_skips: 0
positive_property_1: skip, tests_run 2 (lie: 0 executed), discarded 0
```

**After** (M1+M2, B-refined): `requires { n >= 0 }` is reachable, so this
particular property passes (Example 1). For a genuinely unreachable contract
(`requires { n > 1000 && n < -1000 }`):
```
success: false, vacuous_skips: 1, exit 1
impossible_property_1: skip, skip_kind out_of_contract,
  tests_run 0, generated_inputs 1000, discarded_inputs 1000,
  error "unverified: requires filter accepted 0 of 1000 generated inputs; need 100 (1000 discarded)"
```
…with `ailang test --allow-skips` as the explicit, recorded opt-out.

## Success Criteria

- [ ] AC-M1a: new unit — `requires { n >= 0 }` property passes with
      `tests_run == 100`, `discarded_inputs > 0` (red before M1, green after).
- [ ] AC-M1b: `requires { false }` property skips with
      `generated_inputs == 1000`, `discarded_inputs == 1000`,
      `tests_run == 0`, error starting `unverified:`
      (red before: `1/0/1`).
- [ ] AC-M2a: `AddPropertyResult` — `out_of_contract` skip with
      `TestsRun == 0` ⇒ `VacuousSkips == 1` (red before: 0).
- [ ] AC-M2b: mixed suite (sibling pass + zero-execution out-of-contract
      skip) ⇒ `Success() == false`; with `--allow-skips` ⇒ exit 0.
- [ ] AC-M2c: `out_of_contract` skip with `TestsRun == 1` in a mixed suite
      still ⇒ `Success() == true` under B-refined (pinned row updated, not
      deleted).
- [ ] AC-e2e-1: `requires_unreachable.ail` run exits 1 with the exact
      counters above for both impossible properties (updated AC3).
- [ ] AC-e2e-2: new `requires_tight.ail` fixture exits 0 with the requires
      property `pass`, `tests_run == 100`, `discarded_inputs > 0`.
- [ ] AC-regress: AC1 (`precond2.ail` exit 0, `big` pass 100/discards>0),
      AC2 (`precond_negative.ail` exit 1, counterexample), ordered.ail
      (g/h pass) — all unchanged.
- [ ] Sweep: examples/ + testdata/ show only the enumerated deltas
      (requires-path skip→pass; requires_unreachable exit 0→1).
- [ ] All tests passing (`make test`, `make fmt`, `make lint`,
      `make check-boundaries` — `internal/testing` is below the embed
      boundary; no import changes expected).
- [ ] Documentation updated (`docs/docs/guides/testing.md` skip/discard/
      vacuity subsection).
- [ ] Post-landing: coordinator comments on issue #1921 for the
      sunholo/relativity maintainer noting 0.13.0's "0 failures" was an
      artifact, and the D-TRIAGE-25 ledger row is marked RESOLVED with the
      ruling text.

## Testing Strategy

**Unit tests (red-first, per milestone):** the AC-M1a/M1b/M2a rows above are
written and committed against current code first — M1a and M2a must fail
before their milestones land (a guard is not a gate until something reds when
you remove it, per the repo's own rule 3j idiom).

**Integration/e2e:** the two CLI fixtures drive the built binary exactly like
`test_contract_domain_test.go` does today (same `runAilangBin` +
`--seed 42` reproducibility pattern).

**Manual/gates:** run the reporter's own repro (relativity package) once
post-landing and record `vacuous_skips` and per-property counters — expected:
0 vacuous (reachable domains all pass post-M1), `discarded_inputs > 0`
throughout; file the numbers in the implementation report.

## Deferred Decisions

- Whether to unify the two loops behind one shared driver helper
  (generate-params → filter-by-requires → callback) or keep two loops sharing
  only the constants and message format — **agent may choose** at
  implementation time; the design prefers whichever leaves
  `runEnsuresProperty` byte-stable for its pinned ACs.
- Exact wording of the human summary line (D5) — **agent may choose** within
  "never ran an in-contract case (…)".
- Whether the cap-exhausted requires-path skip should also carry the
  first-violating input in its message (helps humans debug domains) —
  **agent may choose**; cost is one retained value per property.

## Non-Goals

- **Generator shaping from `requires` bounds** (reporter's third suggestion):
  re-deriving generator ranges from numeric predicates is constraint
  interpretation — a different feature with its own conflict surface, and it
  overlaps parked Lane B2 (user-supplied generators, BLOCKED on the
  deterministic fuel budget, quorum 2026-07-29). Recorded in Future Work.
- **Any change to the ensures path's pass/fail/skip semantics** — it is
  shipped, pinned by AC1-AC3, and correct.
- **Any change to the forall-property path** (`runProperty`) or shrinking.
- **New JSON fields, new CLI flags, new skip kinds** — the existing surface
  suffices; adding kinds would churn the schema for no consumer.
- **Eval-harness bank schema surfacing of `vacuous_skips`** — already Future
  Work F3 of m-property-generator-coverage.md; not loaded onto this fix.
- **Changing `--allow-skips` semantics** — it already forgives vacuous skips
  (`SuccessAllowingSkips`, `result.go:141-145`).

## Timeline

**Total: ~2 days** (single sprint; phases are each independently committable)

- **Day 1**: M1 (loop rewrite, counter honesty, red-first M1a/M1b) —
  ~6 hours including the shared-constants refactor.
- **Day 2 AM**: M2 (vacuity rule, pinned-row updates, red-first M2a/M2b,
  reporter text) — ~4 hours.
- **Day 2 PM**: e2e fixtures (AC3 update, `requires_tight.ail`), docs
  subsection, sweep, `make test/lint/check-boundaries` — ~4 hours.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Exit-code flip breaks downstream package gates that (like relativity 0.13.0) unknowingly depend on the forgiveness | Med | The flip is *the requested fix* and is freeze-gated on D4; `--allow-skips` is a one-flag, per-gate opt-out; the sweep + issue #1921 notification give maintainers the list |
| The 1000-attempt cap is still too low for legitimately narrow multi-parameter domains (post-M1 they skip under-sampled) | Low | Under B-refined those skips stay green but fully visible (counters + skip_kind) — the failure mode the ledger's A option targets; raising `maxAttempts` or picking A-strict is a one-line follow-up once field data exists |
| Requires-path runtime up to 10× (≤1000 predicate evals per property) on pathological suites | Low | Same bound the ensures path already pays; deterministic; bounded by the same constants; measured in the sweep |
| AC3's property selection ("last with generated_inputs==1000") silently picks the wrong property after M1 makes both impossible properties cap-exhausted | Med | Called out in Phase 3: selection must disambiguate by exact name or `skip_kind`-adjacent counters; the updated test fails loudly if both match |
| Human ruling picks A-strict, widening the blast radius beyond the tested fixtures | Low | Both options are fully specified (one-line delta in `AddPropertyResult`); the sweep re-runs under the chosen option before landing |

## Related Documents

**This issue's prior art:**
- [m-property-generator-coverage.md](../../implemented/v0_31_0/m-property-generator-coverage.md) (implemented v0.31.0) — Lane A shipped the `out_of_contract` skip kind, the discard counters and the "honest, loud" principle; its **Future Work F1** explicitly reserved the discard-threshold policy for a human (D-TRIAGE-25). This doc is F1's design review, not a contradiction: Lane A made skips *visible*, this doc makes the zero-execution case *count*.
- [D-TRIAGE-25](../../triage-mission.md) (open, issue #1921) — the human ruling this doc freezes for. Default C until ruled.
- Triage rows `ailang-core-triage/imported-type-generator-skips-json.md` and `cross-file-property-generators-world-core.md` — same `ailang test` skip-classification neighborhood, different root cause (generator derivation for unreachable *types*, not unreachable *domains*); their suggested per-kind `skip_kind` split is a Non-Goal here.

**Deliberately distinct (checked for overlap per the duplicate gate):**
- [m-contract-verification-coverage.md](../m-contract-verification-coverage.md) — splits `verify_skipped` into `skipped` vs `not_applicable` for the **SMT verifier** (`ailang verify` / `ai-check`). Different subsystem, different counters, same data-integrity spirit. No overlap.
- [m-property-seed-determinism.md](../../implemented/v0_31_0/m-property-seed-determinism.md) — seed derivation contract this design relies on (per-property deterministic streams); unchanged here.

**References:**
- Issue #1921 (reporter: stapledons_godot, sunholo/relativity 0.13.0 evidence) and issue #1892 comment `6313645488` (verifier/judge CONCUR on NEEDS-DECISION).
- [Design Axioms](/docs/references/axioms) — scoring below.

## Verification Log

Every load-bearing claim, checked first-party at HEAD `b23cadc` (v0.57.0
binary, matching commit, live in this session) on 2026-11-03:

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | The requires path skips on the FIRST out-of-contract input, with no retry | Read `internal/testing/runner.go:450-508` (loop `for testNum < numTests`, immediate `return result` on `!boolVal.Value` at :499-508) | Confirmed |
| V2 | `DiscardedInputs` is incremented only on the ensures path | `grep -rn "DiscardedInputs" internal/ cmd/ --include=*.go \| grep -v _test` → sole writer `contract_domain.go:129` | Confirmed (negative-existence) |
| V3 | The requires path reports misleading `tests_run` and zero discards at the skip | Live repro (`repro/tight` fixture, `--seed 0`): `positive_property_1` `tests_run 2 / generated 2 / discarded 0`, skip; `expm1ish_property_1` `5/5/0` | Confirmed (matches reporter's `1/1/0` shape) |
| V4 | The ensures path already retries: `requiredAccepted=100`, `maxAttempts=1000`, discards counted, `unverified:` skip text | Read `contract_domain.go:85-87, 92, 129, 158-164` | Confirmed |
| V5 | Out-of-contract skips are excluded from `VacuousSkips`; `Success()` requires `VacuousSkips == 0` | Read `result.go:118-134` | Confirmed |
| V6 | A suite with zero-execution out-of-contract skips reads green when a sibling passes | Live repro: `repro/tight` → `success: true, vacuous_skips: 0` with 2 skips | Confirmed (the defect) |
| V7 | `list_recursive_verify.ail` has a zero-accepted out-of-contract property today | Live run `--seed 0`: `extractBounded_property_4` `tests_run 0 / generated 1000 / discarded 1000`, `vacuous_skips: 0` | Confirmed |
| V8 | The current exclusion is test-pinned, and which pinned rows flip under B-refined | Read `result_vacuous_test.go` ("out of contract forgiven" row has NO `TestsRun` ⇒ 0 ⇒ flips), `result_vacuous_success_test.go` (both fixtures carry `TestsRun: 1` ⇒ survive) | Confirmed — exactly one unit row flips (updated, not deleted) |
| V9 | AC3 pins exit 0 for the impossible-contract fixture and would flip to 1 | Read `cmd/ailang/test_contract_domain_test.go:122-158` (`runContractFixture(t, bin, ..., 0)`) | Confirmed |
| V10 | Both CLI aggregates (file and package mode) compute exit codes via `Success()` / `SuccessAllowingSkips()` and aggregate `VacuousSkips` | Read `cmd/ailang/test.go:118-151, 266-303` | Confirmed |
| V11 | All fixtures cited exist | `ls internal/testing/testdata/contracts/` → ordered, precond2, precond_negative, requires_unreachable; `examples/runnable/contracts/list_recursive_verify.ail` and `basic.ail` exist | Confirmed |
| V12 | No consumer of `vacuous_skips` outside `cmd/ailang/test.go` aggregation and the reporter | `grep -rln "vacuous_skips\|VacuousSkips" ui/ web/ tools/ mcp_tools/ serveapi/ internal/ cmd/` → only `cmd/ailang/test.go`, `internal/testing/` | Confirmed (negative-existence) |
| V13 | `--allow-skips` already forgives vacuous skips | Read `result.go:141-145` (`SuccessAllowingSkips = FailedTests == 0`) and both CLI call sites | Confirmed |
| V14 | Properties under `--bytecode` are unaffected (bytecode mode covers named-test bodies only) | Read `internal/testing/bytecode_engine.go:1-52` | Confirmed |
| V15 | The requires-property naming/index used in ACs (`_property_1` = first contract in source order) | Read `collector.go:133-145` (properties collected in decl order) + live repro property ordering | Confirmed |
| V16 | No doc currently defines `tests_run`/`discarded_inputs`/skip kinds | `grep -rn "vacuous\|out_of_contract" docs/docs/` → only the JSON example at `guides/testing.md:402` | Confirmed (negative-existence) — docs update in scope |
| V17 | D-TRIAGE-25 is OPEN, human-owned, with default C and "no implementation or design dispatch before ruling"; the operator-side coordinator dispatched this design review | Read `design_docs/triage-mission.md:157`, `docs/internal/backlog/2026-10-29/initial-issue-verification.md` (#1921 row), and this task's dispatch (coordinator `task-a08bfe4d`) | Confirmed — this doc freezes D3/D4 for the ruling rather than resolving it |
| V18 | Reporter's relativity numbers are reporter-provided, not re-measured | sunholo/ailang-packages checkout not present in this workspace | Recorded as motivation only; the mechanism it rests on is independently reproduced (V1-V6) |
| V19 | `AddPropertyResult` has exactly one production caller (`RunSuite`), and `no_generator`/`unsupported` skips execute zero cases by construction | `grep -rn "AddPropertyResult" internal/ cmd/ --include=*.go \| grep -v _test` → `runner.go:89-90` only; read the early-return skip branches (`runner.go:310-318` forall, `runner.go:428-436` and `441-446` requires, `contract_domain.go:39-45` and `75-80` ensures) — they return before any draw, `TestsRun` stays 0 | Confirmed (negative-existence): the M2 rule change is observable only through `out_of_contract` skips |

No new error codes are allocated by this design (no `MOD/PAR/TC/EFF` namespace
touch), so no error-code grep was needed beyond the above.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The retry loop uses the same seed-derived per-property RNG stream; counters become functions of the seed alone (today a skip's `tests_run` depends on which draw violated, already deterministic but *meaningless*; after, deterministic and meaningful) |
| A2: Replayability | +1 | `--seed` replay now reproduces identical `generated/discarded/tests_run` triples on both paths; the replay command is already reported |
| A3: Effect Legibility | 0 | No effect-surface change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | Verification honesty: a "pass" now means 100 in-contract executions; vacuity is detected instead of silently forgiven; the attempt cap bounds work |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | `success: true` becomes a machine-checkable claim that every property executed; zero new thresholds in the success predicate under B-refined |
| A8: Minimal Syntax | +1 | No syntax, flags, or JSON fields added |
| A9: Cost Visibility | 0 | Worst case per property rises to the bound the ensures path already pays; duration and per-property counters keep it visible |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | +1 | Skip reasons carry exact counts on both paths; the lying `tests_run` value is eliminated |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +5** → **Decision: Move forward** (contingent on the D3/D4 human
ruling; no hard violations).

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced (seed-derived streams only)
- [x] A3: no hidden side effects
- [x] A4: no ambient access granted
- [x] A7: the green signal gets *more* machine-checkable, not less

## Future Work

- **Requires-informed generation** (reporter's suggestion 3): shape numeric
  generators from simple `requires` bounds (`x >= 0`, `|x| <= 700`). Sits
  naturally with Lane B2 (user-supplied/registered generators, currently
  BLOCKED on the deterministic fuel budget) and the refinement-aware
  generation item in m-property-generator-coverage.md; needs its own design
  doc and conflict surface before any implementation.
- **Cap/coverage tuning**: if field data post-M1 shows many under-sampled
  skips (1 ≤ `tests_run` < 100), revisit `maxAttempts` (e.g. 10 000 for
  scalar-only domains) or move to A-strict — one line each, per the
  [options table](#d-triage-25-options).
- **Eval-harness bank schema**: surface `vacuous_skips` so agent-written
  contract tests are scored on *executed* properties (already F3 of
  m-property-generator-coverage.md; this fix makes the counter trustworthy
  enough to bank).

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03
