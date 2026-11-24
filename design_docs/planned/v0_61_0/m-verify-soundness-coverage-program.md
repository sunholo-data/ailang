# M-VERIFY-SOUNDNESS-COVERAGE: close what VERIFIED does not mean, then widen what it reaches

**Status**: Planned. **Quorum guardrail spent** (round 1 and the one re-quorum both BLOCKED 3/3 present; codex seat absent on quota). Every objection was accepted and addressed in-session; see §Quorum History. Awaiting Mark's ratification of D1/D2/D4, not a third round. M0–M1 carry no freeze items. Umbrella program: each milestone below is its own sprint, and M3/M4 need their own design doc after their rulings.
**Target**: v0.61.x – v0.62.x
**Priority**: P1. M1–M2 are P1; M3 is P0 for soundness once D1 is ruled.
**Estimated**: M0 0.5d · M1 1.5d · M2 3–4d · M3 3–4d · M4 2–3d · M5 2d · M6 2d
**Dependencies**: none to start. M2 absorbs [verify-hof-inline-behind-callee-sort-gate](../ailang-core-triage/verify-hof-inline-behind-callee-sort-gate.md). M3 absorbs [verify-call-site-precondition-not-checked](../ailang-core-triage/verify-call-site-precondition-not-checked.md). M4 absorbs [verify-int-width-mismatch-z3-int-vs-int64](../ailang-core-triage/verify-int-width-mismatch-z3-int-vs-int64.md).

**Quorum trigger:** #1 fired (design-freeze items D1–D3). Run `ailang design-quorum` before M3/M4 are planned. M0–M2 have no freeze items and may proceed.

## Progress

| Milestone | State (2026-11-23) |
|---|---|
| M1 (S7 callee identity) | **Landed on dev.** Deterministic bare-name lookups, plus a gate refusing any call that would bind another module's function (`internal/smt/callee_identity.go`, tests in `callee_identity_test.go`). One false positive (a parameter sharing a function's name) was found by the corpus run and fixed. S4/S5/S6 are still open |
| M2 (HOF callers) | **Landed on dev, extended.** map/filter/foldl/any/flatMap over any element sort, with captures, inside callees and inside contracts. Bounded (depth 3) and classified like recursion. Let and capture hygiene (`$l_`/`$c_`) fixes #215. See §M2 result below |
| M5 (partial) | **Landed on dev.** Named-function HOF arguments (eta-expanded). Quantifier-free map/filter frontiers, so length relations are proved for any list. Direct encodings for std/list `nthOr`/`headOr`/`lastOr`/`head`/`nth`/`last` and std/option `getOrElse`/`isSome`/`isNone`. Fixed `floatToInt` floor-vs-truncate (a false VERIFIED). Still open: generic *user* callees, Result-returning callees, math builtins |
| M6 (witness replay) | **Landed on dev.** Each inconclusive model's inputs are run through the real function, using the property-test harness. A runtime ensures failure, with every requires holding, is a confirmed VIOLATION. Int/float/bool/string/list/record inputs are replayed; Option/ADT inputs are not yet. Corpus: 2 of 25 inconclusives became genuine violations (crew_core `ruleSource`: a missing `requires { label != "" }`; `yieldsOf`: `length(result) <= 4` is false for scenarios with more than four yields per recipe). 8 replays hit the contracts holding; 13 models were not convertible (let-terms, unsupported sorts). Induction is still open |
| M5 (cont.) | **Landed on dev.** std/math: floor/ceil/round/abs/PI exact; sqrt/exp/pow/log/sin/cos uninterpreted with per-call IEEE-true facts, and a model over them is inconclusive (77 false counterexamples avoided in the corpus). take/drop encoded (they had never been reachable). `[]` under `let` is typed from the return sort. Replay reads let-terms and enum constructors. Corpus: **verified 748 → 840** |
| M5 (strings) | **Landed on dev.** std/string trim/toLower/toUpper/split/chars/join/repeat/replace (uninterpreted, rune-exact facts) and charAt (exact). Encoder hygiene: `[]` sorted from its use, helper/unrolled parameters bound as `$a_`. An undecidable string/math encoding is a skip. Replay found docparse `csvParseLine` (`split("", "")` is `[]`). Corpus: **verified 840 → 872**, errors 5 → 2 |
| M0 (gap map) | **Landed on dev.** `make verify-gap-map` (`tools/verify-gap-map/`) verifies every contracted file in the `tools/downstream/corpora.yaml` corpora (the same inventory as `make downstream-check`), buckets status / normalised reason / blamed callee, type and builtin, reports the D1 blast radius from the new per-result `requires_callees` field, banks a snapshot and diffs against the last one. A status change in a file whose content hash changed is drift, never a regression. Snapshots bank to `~/.ailang/state/verify-gaps/`, not the repo: docparse, email-parse and daneel are private and a snapshot names their functions. See §M0 baseline |
| M5 (concat, honest skips) | **Landed on dev.** `std/list.concat` = `seq.++` (an empty operand takes the other's sort); undeclarable records skip instead of erroring; tuple types render as AILANG in skip reasons. First gap-map diff: error 1 → 0, nothing else moved. The ten `concat`-blamed functions are large parsers whose next blockers are `foldChars` (8), `bytes`/`Option` helper signatures, and effects: coverage there needs generic callees, not more builtins |
| M3 (call-site requires) | **Landed on dev.** D1 ruled (a). Each call in a verified body must meet its callee's `requires` (path-scoped through the call-facts channel); a possible break is `counterexample` / `requires_violation`. Lambdas, recursion, helper bodies and contracts not yet checked. Gap map: no verdict changed; all 11 verified results that call a function with a `requires` had every such call checked. Design: [m-verify-call-site-requires.md](m-verify-call-site-requires.md) |
| M5 (recursive helpers) | **Landed on dev.** A self-recursive callee without an `ensures` is unrolled to depth 3 (it was refused, and its callers skipped as "could not resolve a required sort"); with an `ensures` its contract is still used. Unknowns through the unrolling stay skips. Gap map: +9 verified, 4 skipped → inconclusive, no regressions |
| M3/M4 (inside definitions) | **Landed on dev.** Requires, overflow and NaN obligations inside helpers, `map`/`filter`/`foldl` lambdas and recursion are checked at each call: each define-fun the encoder emits for runtime code gets one `$ob_<name>@<def>` Bool define-fun per obligation (path-scoped, `(=> assumptions obligations)`), and every call adds it on the actual arguments (`internal/smt/codegen_def_obligations.go`). Recursive helpers and lambdas are checked to the unrolling depth (`obligations_checked_to_depth` note). A recursive contracted function's self-calls must meet its own `requires`, which makes its obligations hold by induction. The postcondition query no longer assumes the int64 obligations (that made one docparse function time out). Gap map: no verdict changed; overflow notes 56 → 61, 10 depth notes. Measured first: 32 verified results had int arithmetic in a helper (22), recursive helper (9), lambda (1) or their own recursion (5); none had an unchecked division, `sqrt`/`log` or `requires` |
| M4 (numeric model) | **Landed on dev.** D2 (a′): int64 overflow obligations, a caveat by default and a violation under `--int-model=int64`. D4 (b′): float `/`, `%` by zero, `sqrt` of a negative and `log` of a non-positive are violations; float inputs assumed non-NaN (caveat). Default-mode gap map: no verdict changed; 56 overflow notes. Design: [m-verify-numeric-model.md](m-verify-numeric-model.md) |
| M5 (parametric callees) | **Landed on dev.** The callee-sort gate admits a parametric ADT applied to encodable arguments (`Option[int]`, `Result[float, MyErr]`, a user `Box[int]`): `astTypeEncodable` now reads `CollectDeclarableTypes` (name → type-parameter count, Option/Result built in) and checks arity. The encoder already declared these with Z3 binders, so no codegen change was needed. Still gated: type variables (`foldChars`, `isErr`'s `Result[a, e]`), tuples (`Result[(), string]`), function types, `bytes`; record-returning callees fail later (separate). Gap map: 2 skipped → verified (ailang-parse `pdfAiPagesCountFromInfo`; the gate's own example `unencodable_callee_skip.ail`, moved to a tuple callee to keep demonstrating the skip), 0 regressions. Most former Option/Result skips now stop at the next blocker: an **unannotated helper** (`std/json.get`/`decode`/`asString`, crew `distOf`/`levelsOf`) is refused by `smtAdmissible` (25 rows, now the top skip reason), which m-pure-row-and-iface-purity Rule 3 ratified (nil row admits nothing). Its skip hint now names the fix instead of the generic list-pattern text |

**Five-repo corpus, 2026-11-23 (1,840 obligations):**
- Verified: 698 → **748**.
- 25 functions that were skipped now reach the solver as inconclusive.
- Errors: 12 → 6.
- No verdict regressions, and no new errors or counterexamples.

Inconclusive dominates what is left of the HOF gain: real contracts rarely bound list length in `requires`. Witness replay (M6) is now the lever for those.

## Why this program, and why not Lean

Question asked on 2026-11-23: "we use Z3, a lot of people use Lean for verification. Should we?"
Answer, grounded below: **no, not as the core.** The verifier's measured limits are all on the
*encoder* side, where Z3 is the right tool (push-button, no proof scripts, deterministic verdicts).
Dafny, Verus and F* reach inductive proofs on Z3 through auto-active encoding. Lean would add a
second semantics of AILANG to keep in sync, and token-heavy proof synthesis.

A Lean *export* of obligations Z3 leaves inconclusive stays a possible motoko **extension** for
later (PROGRAM.md routing). The trigger is data: M6's inconclusive count staying material after
M5. Today that count is **0 of 1,840 obligations in real downstream code** (V9).

## Problem Statement

Measured on v0.60.6-128-g3fb62e616c with Z3 4.16.0, 2026-11-23. Probe files and corpus JSON from the
attended session are summarised in the Verification Log.

**Soundness: what a `✓ VERIFIED` does not mean.**

| # | Gap | Probe | Today |
|---|-----|-------|-------|
| S1 | Callee `requires` never discharged at call sites | caller `requires { n < 0 }` → `one(n)` with `requires { n > 0 }` | both VERIFIED (V1) |
| S2 | `int` proved as unbounded `Int`; runtime wraps int64 | `inc(x) ensures { result > x } { x + 1 }` | VERIFIED; `inc(maxInt)` = `-9223372036854775808` (V2) |
| S3 | `float` proved as `Real` (no NaN) | `x == x` | VERIFIED; false at runtime for NaN (V3) |
| S4 | Callee-sort gate keys callees by **bare name**, first-wins over a Go-map iteration | `std/list.map` caller | skip reason cites `Option[b]` (std/option.map's signature) (V4) |
| S5 | Unannotated generic parameter reaches Z3 | `pure func inc(x) requires {…}` | `! ERROR unknown constant x` (#2400) (V5) |
| S7 | **Callee bodies resolved by bare name: a false VERIFIED, nondeterministically** | `a/main` imports `a/good (clamp0 as goodClamp)` and `a/bad (clamp0)`; `usesBad(x) ensures { result >= 0 } { clamp0(x) }` where `a/bad.clamp0 = x - 1` | **VERIFIED in 5 of 6 runs**, VIOLATION in 1 (V16). The correct verdict is VIOLATION (`x = 0`) |
| S6 | Counterexample `result` values are raw solver terms | `basic.ail` before its fix | `result: Int = (ite (= (>= $p_dividend 0) …` truncated (V6) |

S2 and S3 are now *disclosed* on every run (`Scope:` line and `numeric_model`), but not *checked*. Both are handled by M4 under decisions D2 (int) and D4 (float).

**Coverage: what the verifier cannot reach.** These are real repositories, not our curated examples.
The curated examples come out 116 verified / 17 counterexample (all but two deliberate) / 7 skipped
/ 0 inconclusive, which flatters the verifier.

| Corpus (files with `ensures`) | verified | skipped (encoding) | missing_postcondition | error / unknown | counterexample | inconclusive |
|---|---|---|---|---|---|---|
| daneel/tools (43) | 160 | 0 | 0 | 0 | 0 | 0 |
| ailang-world (31) | 141 | 1 | 0 | 2 | 1 | 0 |
| ailang-packages-sd-sortedjson (167) | 251 | 585 | 85 | 11 | 4 | 0 |
| ailang-parse-wt (24) | 87 | 232 | 0 | 0 | 0 | 0 |
| stapledons-godot (125; 17 failed to load) | 59 | 304 | 106 | 2 | 0 | 0 |
| **Total** | **698** | **1,122** | **191** | **15** | **5** | **0** |

Obligations = verified + encoding skips + error/unknown + counterexample = 698 + 1,122 + 15 + 5 =
**1,840**. `missing_postcondition` rows (191) are not obligations.

Of the 1,122 encoding-skipped functions (V8), counted **per function**. The categories **overlap**:
one skip reason can name a builtin, a gated callee and "uses higher-order functions" together, so
the rows do not sum to 1,122.
- **736** hit the callee-sort gate. The gate reports one callee per function, so the type split sums exactly: `Result[..]` 204, function type 159, `Option[..]` 137, type variable 108, `list[..]` 105, tuple 8, array 7, `bytes` 6, `Map` 2.
- **441** are HOF-attributed (a gated or builtin callee among `any`/`all`/`filter`/`map`/`foldl`/`flatMap`/`findFirst`, or the reason says "uses higher-order functions"). The top gated callees are `any` 150, `filter` 104, `map` 69 and `foldl` 61.
- **322** name an unencodable stdlib builtin. **112** of those name a `std/string` one (top: `trim` 30, `split` 27, `join` 16, `toLower` 9). `std/list.head` (returns `Option`) appears 34 times.
- **306** fail the declaration closure on a user-function callee. **119** of those name a math function (`sqrt` 57, `pow` 28, `exp` 22, `atanh` 6, `pi` 5).
- **45** are effectful (fundamental; not targeted).

**Who is affected:** every agent told by the teaching prompt to "maximize the surface area of
verified code". Today, writing idiomatic AILANG (HOFs, `Option`) mostly gets you SKIPPED. And the
VERIFIED you do get silently assumes the caller honours `requires` and that no int overflows.

## Goals

**Primary goal:** a VERIFIED result means what an agent reading it would assume, and the share of
real contracts that reach a verdict roughly doubles.

**Success metrics** (each re-measured by the M0 instrument on the same five corpora):
1. S1: a call that can violate a callee's `requires` is reported, not VERIFIED. The probe flips.
2. S7: the V16 two-module probe is VIOLATION on 20 of 20 runs, and no callee lookup is keyed by bare name. S4/S5/S6: zero Z3 errors from encodable-but-unannotated input; skip reasons name the actual callee; counterexample values are literals.
3. Coverage: encoding skips drop from 1,122 to ≤ 600 (M2 + M5), and verified rises from 698 to ≥ 1,100. No verified-to-counterexample flips go unexplained.
4. S2: an `int` overflow reachable under `requires` is surfaced, per D2's ruling.
5. S3: per D4's ruling, either a float contract that is false for NaN is no longer VERIFIED, or every float-involving VERIFIED carries a machine-readable `float_model: real` caveat on the result row itself (not only the report header).

## High-Impact Decisions

| # | Decision | Options | Who decides | Change cost |
|---|----------|---------|-------------|-------------|
| D1 | What a call that may violate a callee's `requires` reports | (a) a checked obligation under the caller's path condition: a violation is a `counterexample` with a `requires_violation` reason code. (b) A new status `precondition_unproved`, a non-verified outcome. (c) Warn only. | **Mark** | Changes the trust contract of VERIFIED. Some currently VERIFIED callers will flip; M0 measures how many first |
| D2 | int64 semantics | (a) overflow obligations: every `+ - *` on `int` asserts no wrap, reported as a separate `overflow` reason. Opt-out flag `--int-model=unbounded`. (b) A `(_ BitVec 64)` encoding mode, slower, and some current proofs become unknown. (c) Keep disclosure only | **Mark** | (a) is additive and reports per-operation; (b) changes the solver theory for everything |
| D4 | float semantics | (a) IEEE encoding: `(_ FloatingPoint 11 53)` with `fp.eq`/`fp.lt` and NaN, exact but slower, and some current `Real` proofs become unknown. (b) NaN obligations: a float `==`/`<`/`<=` whose operands are not guarded by `requires { !isNaN(x) }` (or derived from literals or ints) makes the result `unproved` with reason `nan_unguarded`. (c) Keep `Real` and attach a per-result `float_model: real` caveat to every VERIFIED row that touches a float | **Mark** | (c) is pure reporting; (b) changes outcomes for float contracts; (a) changes the solver theory for floats. The `internal/smt/` code uses no `FloatingPoint` sort today (V14) |
| D3 | Does `make verify-examples` pin verdicts (VERIFIED/VIOLATION per function), so a regression to SKIP fails CI? | (a) yes, pin via each file's `-- Expect:` header; (b) no | agent may choose (a) | Small. `hof_verify.ail` has been silently skipping since #1375 because SKIP is not a failure |

### Design Freeze

- [x] D1 ruled 2026-11-24 by Mark: **(a)**, a checked obligation; a possible violation is a `counterexample` with reason code `requires_violation`. M3 design: [m-verify-call-site-requires.md](m-verify-call-site-requires.md)

**D2/D4 measurements (2026-11-24, throwaway prototype on the M3 obligation channel, not committed).**
Parameters are assumed valid for their type (an `int` is an int64; a `float` is not NaN), and only
operations are checked, each as a path-scoped obligation.
- *D2(a), int64:* every `int` `+ - *` must stay in [-2^63, 2^63-1]. Inventory (8 corpora): **64 of
  590 VERIFIED change** (56 → counterexample, 7 → unknown, 1 → skipped), in every corpus: examples
  21, daneel 11, world 10, ailang-parse 8, docparse 5, ailang-packages 5, motoko-fork 2, email-parse 2.
  Stapledons physics (one copy per file): 4 of 55. Read by hand, each is a real wrap only at absurd
  inputs (`pagesWithText * 10` at 9.2e17 pages; `price * ratePct` at a 4.6e18 price;
  `day - deadline` at the int64 extremes); the fix is an upper bound in `requires`. A first run
  without the parameter bound gave 71: unbounded `Int` parameters inflate the count.
- *D4, floats:* the inventory has 12 verified results touching `Real` and none with a `float`
  parameter. Stapledons: 38 of 55 unique verified results use floats, 36 take a float parameter,
  28 divide. Checking float division by a possibly-zero divisor (the operation that makes ±Inf/NaN
  from ordinary inputs) changes **0** proofs in either corpus: the divisors are already guarded.
  Option (b) as written (any comparison on an unguarded float input is unproved) would reach up to
  the 36 float-parameter proofs; (a) IEEE was not measured.
- [x] D2 ruled 2026-11-24 by Mark: **(a′)**: int64 overflow obligations, reported on the VERIFIED result (`caveats`, kind `int_overflow`) by default; `--int-model=int64` makes them violations. The default flips to (a) once downstream packages carry bounds
- [x] D4 ruled 2026-11-24 by Mark: **(b′)**: operations that make NaN/±Inf from ordinary inputs (float `/` and `%` by a possible zero, `sqrt` of a negative, `log` of a non-positive) are checked, and a possible one is a violation (`float_nan_or_inf`); every float result notes that its inputs are assumed non-NaN. M4 design: [m-verify-numeric-model.md](m-verify-numeric-model.md)
- [x] D3: agent latitude, default (a)

## Solution Design

### Overview

Measure first, fix honesty bugs, then widen coverage where the data says it pays, then close the
two soundness gaps once their semantics are ruled. Each milestone re-runs the M0 instrument and
banks the delta.

### Implementation Plan

**M0 — Gap-map instrument (0.5d).** `make verify-gap-map` runs `ailang verify --json` over a
corpus list (repo examples plus a configurable list of downstream checkouts). It buckets results by
status, `reason_code` and normalised reason, and attributes skips to the callee, type and builtin.
It banks `.ailang/state/verify-gaps/<date>.json` and diffs against the previous snapshot, the same
shape as `make simplicity-audit`. This is the measurement every later milestone is accepted against.
It also reports the D1 blast radius ahead of M3: the count of call sites whose callee has a `requires`.

**M0 baseline (2026-11-24, v0.60.6-185, 8 corpora, 255 files).** This is the inventory corpus, not
the ad-hoc five-repo corpus of the earlier rows (that one counted four worktree copies of docparse
and a repo outside the inventory), so its numbers are not comparable with them. 765 contracted
results: verified 590, counterexample 17, inconclusive 3, unknown 1, error 1, skipped 125,
missing-postcondition 28; 880 uncontracted exports. All 16 counterexamples not confirmed by replay
are deliberate (`examples/runnable/contracts` `broken*` demos and world's `refuted.ail` fixture); the
17th is docparse `csvParseLine` (fix: ailang-parse#171). Largest skip buckets: a callee signature
with a type variable or Option/Result (35), a callee whose own encoding is inadmissible (11), effects
(10), and the S7 gate refusing stdlib name collisions such as `std/list.concat` against
`std/bytes.concat` (12).

**D1 blast radius, measured.** 20 results reach a callee whose `requires` is assumed: 11 VERIFIED,
9 already skipped. Seven of the 11 are repo examples. The four downstream ones (deontic
`interestFor`, `penaltyFor`; docparse `pdfTextLayerSparseCore`; daneel `offsetFor`) were read by
hand: in each the caller's own `requires` (or the guarding branch) implies the callee's, so under
D1(a) all four would still verify. D1(a) adds a checked obligation; on today's corpus it is not
expected to flip any downstream proof.

**M1 — Honesty fixes (1.5d; S7 is P0).**
- S7 + S4, one systemic fix: key every callee lookup by `(module, name)` rather than the bare name. Sites:
  - `collectCalleeCallsInner` (`callee_resolver.go:252`, `seen[vg.Ref.Name]`)
  - `findFuncBodyInAnyProg` (`callee_resolver.go:197`, first hit over a Go-map `range importedPrograms`)
  - `calleeASTFuncs` (`verify.go:315-330`)
  - the emitted `define-fun` names, which must be qualified so two same-named callees cannot collide

  V16–V17 list the sites. Regression fixture: the V16 two-module probe, asserted VIOLATION over 20 runs.
- S5 (#2400): a type-variable parameter becomes a SKIP with reason `unannotated_generic_param` and the hint "annotate `x: int`", never a Z3 error.
- S6: evaluate the model's `result` in Z3 (`(get-value (result))` or model eval) so counterexamples print literals.
- Sort `coreProg.Meta` iteration at `verify.go:337`. This is useful on its own and a prerequisite of the total-run budget.
- D3(a): pin each file's **current** verdicts (`-- Expect:` header) in `make verify-examples`. `hof_verify.ail` is pinned to its present `SKIPPED` × 2 with a `TODO(M2)` note, so M1 lands green. Any *change* of verdict, including an unexpected new SKIP, fails CI. M2 re-pins it to VERIFIED.

**M2 — HOF callers (3d; largest coverage lever, 441 HOF-attributed skips).** Option A of the triage doc:
exempt literal-lambda `map`/`filter`/`foldl` from the gate when `InlineHOFCalls` will consume them.
These are the only three kinds the inliner knows today: `knownHOFStdlib`/`knownHOFBuiltins` (`hof_inline.go:50-61`, V18).
`any`/`all`/`flatMap` (≈170 of the 441) **do not** share an existing shape. Each needs a new
`HOFKind` (`any`/`all` as a boolean fold, `flatMap` as a seq-concat fold). Estimate +1d, or defer them to an M2b. Thread the real element and result sorts
instead of hard-coded `Int`/`(Seq Int)` (`hof_inline.go:332-349`: `TCon{"int"}` params, `returnSort = "(Seq Int)"`/`"Int"`, V18). Prefix encoded `let` names so they cannot shadow accessors
(#215). Where the element sort is unencodable, keep an honest SKIP naming the element type.
Closes #1375 and #215.

**M3 — Call-site preconditions (3–4d, after D1).** At every encoded call site, emit the callee's
`requires`, with params substituted for args, as a checked obligation under the caller's path
condition. Param→arg substitution already exists as `buildParamSubstitution`
(`codegen_xmod_contract.go:146-153`), used by both the contract path (`:45`, which since fd6140dc0
emits `(=> requires ensures)` rather than an axiom, `:98`) and the call-facts path
(`codegen_call_facts.go:34`) (V19). What does **not** exist is the same substitution on the
*inline* (`define-fun`) path, which `ResolveCallees` uses for most encodable callees. The work is
to emit `requires(args)` as a checked obligation under the path condition on all three paths. M0 supplies the before/after flip list; every flip is either a real bug in the
caller or a `requires` to add.

**M4 — Numeric model: int64 and float (2–3d, after D2 and D4).** The int half follows D2. Under (a):
per-operation `(<= INT64_MIN e INT64_MAX)` obligations, reported as reason `overflow`, with
`numeric_model` gaining `int=Int+overflow_checked`. The float half follows D4. Each half ships
independently, behind its own ruling.

**M5 — Parametric callees and math (2d).** Today the gate rejects them in
`astTypeEncodable` (`internal/smt/from_ast.go:256-261`: any `TypeApp` other than `list[T]` is
unencodable). Math callees fail through the unresolved-user-function guard
(`internal/smt/codegen.go:157-175`, `activeUserFunctions`) because `std/math` has no entry in
`ResolveStdlibToBuiltin` (`internal/smt/encodable.go`). Monomorphise `Option[T]`/`Result[T,E]` callees at a
concrete call site (the 341 `Option`/`Result` gate rejections). What exists is the *declaration*
side: `DeclareParametricDatatype` (`parametric.go:10`) declares parametric ADTs with native Z3
binders. Admitting a parametric **callee** signature is new work, built on that, not a reuse (V20).
Encode `sqrt`/`pow`/`exp`/`pi` as uninterpreted functions, never exact values, under two rules:
- **Only facts true of IEEE-754 float64 results.** No `sqrt(x)*sqrt(x) = x`: rounding breaks it
  (`sqrt(2.0)*sqrt(2.0) ≠ 2.0`). The candidates are `x >= 0 ⇒ sqrt(x) >= 0`, `exp(x) >= 0` (not `> 0`:
  underflow gives 0) and `pi` as a bounded constant `3.14159 < pi < 3.14160`. Each fact gets a
  property test against the runtime builtin.
- **Ground facts per call site, never quantified axioms.** Each `sqrt(e)` application asserts its
  fact for that `e` only. This stays out of quantified nonlinear real arithmetic, where Z3
  routinely returns `unknown` (A5).

**M6 — Recursion: witness replay then induction (2d, lowest measured demand).** First, replay an
inconclusive model through the evaluator: a confirmed violation becomes a counterexample. Second,
the induction-hypothesis frontier (assume the function's own `ensures` at `f_0`). Both are
designed in [m-verify-bounded-unrolling-false-counterexample](../../implemented/v0_57_5/m-verify-bounded-unrolling-false-counterexample.md)
§Future Work. Re-prioritise if M0 shows inconclusive rising once M2 brings recursive HOF
specialisations into the fragment.

### Files to Modify/Create

- `internal/smt/verify.go` — callee keying, sorted iteration (M1); call-site obligations (M3)
- `internal/smt/from_ast.go` — gate exemption for inlinable HOFs (M2); generic-param skip (M1)
- `internal/smt/hof_inline.go` — element/result sort threading (M2)
- `internal/smt/codegen_control.go` — `let` name hygiene (M2)
- `internal/smt/codegen_xmod_contract.go` — `requires` as obligation (M3)
- `internal/smt/codegen_operators.go` — overflow obligations, float model (M4)
- `internal/smt/types.go` — float sort under D4(a) (M4)
- `internal/smt/from_ast.go` (`astTypeEncodable`) and `internal/smt/parametric*.go` — admit concrete `Option[T]`/`Result[T,E]` callees, reusing the parametric-ADT encoder (M5)
- `internal/smt/encodable.go` (`ResolveStdlibToBuiltin`) plus a new `internal/smt/math_axioms.go` — `std/math` uninterpreted functions and their axioms (M5)
- `internal/smt/solver.go` — model value evaluation (M1)
- `tools/verify-gap-map/` or a `scripts/verify_gap_map.sh` plus Makefile target — the M0 instrument
- `scripts/verify_examples.sh` — pin `-- Expect:` verdicts (M1)
- `docs/docs/guides/contracts.mdx`, `docs/docs/reference/limitations.md`, teaching prompt — each milestone removes its own "Open gaps" line

## Examples

**S1 before / after (M3, under D1(a)):**
```text
Before:  ✓ VERIFIED one   ✓ VERIFIED caller
After:   ✓ VERIFIED one   ✗ VIOLATION caller  [requires_violation: one(n) needs n > 0]
           Counterexample: $p_n: Int = -1
```

**M2 before / after:**
```text
Before:  ⚠ SKIPPED incrementAll — calls "map" whose signature uses an unencodable type "Option[b]"
After:   ✓ VERIFIED incrementAll  (bounded: depth 2)
```

## Success Criteria

- [x] M0: `make verify-gap-map` banks a snapshot and diffs against the previous one. Re-based on the corpora.yaml inventory rather than reproducing the earlier ad-hoc corpus table (see §M0 baseline)
- [ ] M1: the S7 two-module probe is VIOLATION ×20; the S4/S5/S6 probes and a sorted-order test pass; each is mutation-tested (revert the fix, watch its test fail)
- [ ] M1: `make verify-examples` pins current verdicts and stays green. A mutation that changes any pinned verdict (e.g. makes `finance.ail`'s `brokenDiscount` SKIP) turns it red
- [ ] M2: `hof_verify.ail` re-pinned to VERIFIED
- [ ] M2: `hof_verify.ail` VERIFIED; a `[record]`-list `filter` with a field-named `let` never reaches a Z3 error; gap-map HOF-attributed skips ≤ 50
- [x] M3: the S1 probe flips (`caller` VIOLATION, `requires_violation`); the gap-map flip list is empty
- [x] M4: per D2 (a′) the S2 probe carries an `int_overflow` caveat (a violation under `--int-model=int64`); per D4 (b′) the S3 probe carries `float_inputs_assumed_non_nan`, and NaN-producing operations are violations
- [ ] Each milestone: contracts guide, limitations page and prompt "open gaps" lines updated; changelog fragment
- [ ] `make test-core`, `make verify-examples`, `make check-doc-samples` green

## Testing Strategy

**Unit tests:** one `internal/smt` test per probe in the Problem table, asserting the verdict
*and* reason code. Positive and negative controls, per the Daneel rule: break each contract and
expect a VIOLATION.

**Integration:** `make verify-gap-map` before and after each milestone; `make verify-examples` with pinned verdicts.

**Mutation:** revert each fix through a scratchpad copy (never `git checkout <file>`) and confirm its test fails.

## Deferred Decisions

- Instrument form (Go tool vs shell script), its corpus list and storage path — agent may choose; follow `simplicity-audit`'s banked-snapshot shape
- Which HOFs beyond map/filter/foldl M2 admits (`any`/`all`/`flatMap`) — agent may choose, gated on each having a sound inliner shape and a test
- Axiom set for M5 math functions — agent proposes, and the M5 design review checks soundness

## Non-Goals

- **Lean / proof-assistant backend.** See "Why this program". Reopen only on M6 data.
- **Effectful-function verification.** Fundamental (`contracts.mdx`: effects break proofs).
- **Total-run budget for single-file verify.** Stays its own row (verify-total-run-budget). M1's sorted iteration is its prerequisite only.
- **Runtime contract parity** — covered by M-RUNTIME-CONTRACT-PARITY.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| M3 flips many VERIFIED results to violations in downstream packages (World, docparse) | M0 measures the flip list first. Ship with a clear reason code; announce to package owners |
| M2 re-opens #215 (Z3 errors on non-int lists) | Sort threading and `let` hygiene land in the same PR; the record-list fixture is a required test |
| M4 overflow obligations make proofs slow or unknown | Per-operation obligations are linear; measure solve time on the corpus; `--int-model=unbounded` escape hatch |
| Corpus drift (downstream repos change) makes before/after incomparable | M0 snapshots carry each corpus repo's HEAD SHA |

## Verification Log

| # | Claim | Evidence (2026-11-23, v0.60.6-128-g3fb62e616c, Z3 4.16.0) |
|---|-------|-----------|
| V1 | Call-site `requires` unchecked | Probe `one`/`caller` → `✓ VERIFIED one`, `✓ VERIFIED caller` |
| V2 | int64 overflow unmodelled | `inc` VERIFIED; `ailang run` of `inc(9223372036854775807)` prints `-9223372036854775808` |
| V3 | NaN unmodelled | `selfEq(x: float) ensures { result == true } { x == x }` VERIFIED |
| V4 | Callee gate keyed by bare name, first-wins | `internal/smt/verify.go:315-330`: `calleeASTFuncs[fd.Name]` guarded by `!exists`, over `range modules` where `modules map[string]VerifyModule` (`verify.go:177`). `hof_verify.ail` skip cites `Option[b]`; `std/option.ail:13` declares `map[a, b](f, opt: Option[a]) -> Option[b]` |
| V5 | #2400 reproduces | `pure func inc(x) …` → `! ERROR … unknown constant x` |
| V6 | Counterexample `result` is a raw term | `verify --json basic.ail` (pre-fix) model value `"(ite (= (>= $p_dividend 0) (>= $p_divisor 0)"` |
| V7 | Contract-path `requires` is an implication, not an axiom (triage note's 3rd point is stale) | `internal/smt/codegen_xmod_contract.go:98` returns `(=> requires ensures)`; comment `:39` "no global quantified axiom is emitted" |
| V8 | Corpus numbers | `verify --json --relax-modules` over every non-test `.ail` with an `ensures` line in the five repos. Totals: verified 698, skipped 1,313 (191 `missing_postcondition`), error 12, counterexample 5, unknown 3, inconclusive 0. 17 stapledons files failed to load (`package … not found in ailang.lock`), so they are excluded |
| V9 | Recursion demand is low | Inconclusive = 0 across all five downstream corpora and 0 in `examples/` |
| V10 | No existing gap-map instrument | `make help \| grep -i verif` and `ls tools` show only example/stdlib verifiers; `git grep "verify --json" tools scripts Makefile` is empty |
| V11 | HOF skip still present | `hof_verify.ail` → `unproved`, 2 skipped (#1375) |
| V12 | `ensures` calling a user `int -> bool` helper is symmetric now | `good` VERIFIED, `bad` VIOLATION with counterexample |
| V13 | Where M5's two gaps are rejected today | `from_ast.go:256-261` (`TypeApp` other than `list` → false); `codegen.go:157-175` comment cites the 2026-11-02 stapledons `(sqrt s)` Z3 crash that led to the user-function guard; corpus reason text `calls user function "sqrt" that is not SMT-encodable` |
| V14 | No IEEE float sort in the encoder today | `grep -rn "FloatingPoint\|Float64" internal/smt/*.go` (non-test) is empty; `numeric_model.float_sort` = `Real` in every report |
| V16 | S7 false VERIFIED, nondeterministic | Two-module probe above, `ailang verify a/main.ail` ×6 → 5 VERIFIED, 1 VIOLATION. `ailang check` passes |
| V17 | Bare-name keying sites | `callee_resolver.go:197-207` (`findFuncBodyInAnyProg`: `findFuncBody(prog, funcName)` then first hit over `range importedPrograms`); `callee_resolver.go:252+` (`seen[vg.Ref.Name]`, `findFuncBodyInAnyProg(vg.Ref.Name, …)`); `verify.go:315-330` |
| V18 | HOF inliner kinds and hard-coded sorts | `hof_inline.go:14-16` (`HOFMap`, `HOFFilter`, `HOFFoldl` only); `:50-61` known names map/filter/foldl (`$builtin` `map_List`/`filter_List`/`foldl_List`); `:332-349` `TCon{Name: "int"}` and `"(Seq Int)"`/`"Int"` return sorts |
| V19 | Substitution machinery | `buildParamSubstitution` defined at `codegen_xmod_contract.go:153`, called at `:45` (contract path) and `codegen_call_facts.go:34` (call-facts path); not called from `callee_resolver.go` (inline path) |
| V20 | Parametric support is declaration-side only | `parametric.go:10` `DeclareParametricDatatype(name, params, variants)`; the gate still rejects every non-`list` `TypeApp` in callee signatures (`from_ast.go:256-261`) |
| V15 | Obligation total | 698 + 1,122 + 12 error + 3 unknown + 5 counterexample = 1,840 (recomputed per function from the banked JSON) |

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | M1 removes Go-map iteration from verdict-affecting paths. S7 is a verdict that varies between runs on identical source today |
| A2: Replayability | 0 | No trace change |
| A3: Effect Legibility | 0 | No effect change |
| A4: Explicit Authority | 0 | — |
| A5: Bounded Verification | +1 | The core of the program: verification that means what it says, locally |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | Agents get honest verdicts and literal counterexamples; fewer silent SKIPs |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | M3 makes contracts compose across calls (assume-guarantee) |
| A11: Structured Failure | +1 | New reasons are typed codes (`requires_violation`, `overflow`, `unannotated_generic_param`) |
| A12: System Boundary | 0 | — |

**Net Score: +5** → Move forward. No −1 on A1/A3/A4/A7.

## Quorum History

| Round | Reviewers present | Verdict | Objections → disposition |
|---|---|---|---|
| 1 (2026-11-23) | gemini-4-argon, oc-glm-5-3, oc-kimi-k3 (codex: quota) | BLOCKED 3/3 | S3 had no disposition → **D4 + M4 float half + metric 5**. Skip arithmetic did not reconcile → **recomputed per function, overlap stated, V15**. M1's pin turned CI red → **pin current verdicts, re-pin in M2**. M5 named no files → **files + V13/V14** |
| 2 (2026-11-23) | same three | BLOCKED 3/3 | `sqrt*sqrt = x` unsound under IEEE, and quantified NRA axioms risk `unknown` → **IEEE-true facts only, ground per call site**. M2/M3/M5 machinery claims lacked log rows → **V17–V20**. These showed that `any`/`all`/`flatMap` are new inliner kinds (M2 re-estimated) and that parametric callees are new work (M5). Investigating CollectCalleeNames' keying for the glm objection **found S7** (V16) |

Guardrail spent after round 2; no third round. The doc goes to Mark with these dispositions.

## Related Documents

- [m-verify-bounded-unrolling-false-counterexample](../../implemented/v0_57_5/m-verify-bounded-unrolling-false-counterexample.md) — inconclusive status; M6's designs
- [m-smt-well-sorted-preflight](../m-smt-well-sorted-preflight.md) — sort resolution by elaborated type; M1/M2 should reuse its resolver, not add a second
- [m-contract-verification-coverage](../v1_1_0/m-contract-verification-coverage.md) — skipped vs not_applicable split
- [m-smt-string-casefold-charfold](../v0_38_0/m-smt-string-casefold-charfold.md), [m-smt-datetime-sprint-plan](../v0_38_6/m-smt-datetime-sprint-plan.md) — builtin coverage, already planned, not re-owned here
- [m-runtime-contract-parity](../m-runtime-contract-parity.md) — runtime side
- Triage rows: verify-call-site-precondition-not-checked, verify-hof-inline-behind-callee-sort-gate, verify-int-width-mismatch-z3-int-vs-int64, verify-total-run-budget, verify-ensures-verified-without-encodable-refutation (all re-verified 2026-11-23)

## Future Work

- Lean (or other ITP) export of residual inconclusive obligations, as a motoko extension, gated on M6 data
- Bit-vector encoding for bitwise operators (currently skipped)
