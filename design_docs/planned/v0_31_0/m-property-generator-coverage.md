# M-PROPERTY-GENERATOR-COVERAGE: Vacuous-Pass Honesty + Structural Generator Derivation for Contract Property Tests

**Status**: Planned
**Target**: v0.31.0 (Lane A); Lane B may slip to a later minor without harming Lane A
**Priority**: P2 (explicitly NOT a v1.0 bar item — sized accordingly)
**Estimated**: Lane A ~0.5 day; Lane B = B1 only, ~1.5–2 days (independently shippable; Lane A first). B2 (user-supplied generators) **DEFERRED** — see Lane B2.
**Dependencies**: Lane A and Lane B1: None. **Deferred B2: BLOCKED ON a deterministic evaluator fuel/step budget** (fuel charged per evaluation reduction and function application — see Future Work; quorum 2026-07-29). Related-but-independent: [m-forall-properties-direct-core-eval](../v1_1_0/m-forall-properties-direct-core-eval.md) (P3, parked)
**Created**: 2026-07-29 (V1 mission iteration 121, DESIGNER role)
**Upstream filing**: sunholo-data/ailang#517 (open)

---

## Related Documents

- [m-named-test-blocks](../../implemented/v0_29_0/m-named-test-blocks.md) — **implemented v0.29.0** — introduced the *all*-skipped honesty guard this doc extends: `AllSkipped()`, `SuccessAllowingSkips()`, `--allow-skips`, "NO TESTS RAN" + exit 1. This doc closes the *partial*-skip gap that design deliberately left open.
- [m-forall-properties-direct-core-eval](../v1_1_0/m-forall-properties-direct-core-eval.md) — **planned, P3, deferred** — the `properties [forall(...) => ...]` surface is known-broken (`empty program`) on a *different* axis (evaluation path, not generator coverage). **Distinct scope**: this doc touches only `createGeneratorForType` and result/exit semantics; since the forall path shares `createGeneratorForType` (runner.go:247), Lane B's derived generators benefit it automatically *once that doc's fix lands* — no coupling in either direction.
- [m-dx26-property-test-empty-program](../../implemented/v0_21_0/m-dx26-property-test-empty-program.md) (Phases 5/5.1/5.2; sprint plan in [v0_20_0](../../implemented/v0_20_0/m-dx26-ensures-result-binding-sprint-plan.md)) — built the ensures/requires harness paths this doc extends, including the deliberate requires-out-of-contract-⇒-Skip semantics that constrain Lane A's design (see Conflict Surface).

Duplicate gate: SimHash search over 93 planned + 1003 implemented docs returned only keyword noise (top genuine hits found by targeted grep are the three above; none covers generator coverage or partial-skip honesty).

---

## Problem Statement

AILANG's contract-derived property tests (`ensures`/`requires` clauses) silently run **zero cases** for any parameter whose type `createGeneratorForType` does not cover, while the suite still reports `success: true` and exits 0 — provided at least one other test passes. This is the mission's **vacuous-pass class**: a check reporting success for work it never performed (prior instances: `ai-check` silently skipping when `z3` is absent; Go tests `t.Skip`-ing in CI).

**Root cause** (verified by reading `internal/testing/runner.go:630-661` at `origin/dev` @ `3901c14a8`): `createGeneratorForType` has exactly two arms — `*ast.SimpleType` with name in {`int`, `float`, `bool`, `string`}, and `*ast.ListType` — and the `*ast.ListType` arm is **unreachable from any parsed program** (see Verification Log V4: since DX-17 Phase 2, `[T]` parses to `ast.TypeApp{Constructor: "list"}` at `parser_type.go:56`; `ListType` is constructed only in Go test files). So the *effective* generator coverage is four scalar types. Everything else — lists, tuples, records (named or anonymous), ADTs, unit, refined types — produces `Status: skip, tests_run: 0`.

**Live minimal repro** (verified with a binary built from this worktree; `ailang check` clean):

```ailang
module mixed

export func dbl(x: int) -> int ! {}
ensures { result == x + x }
{
  x + x
}

export func headOr(xs: [int], d: int) -> int ! {}
ensures { result >= 0 || result < 0 }
{
  d
}
```

`ailang test --format json mixed.ail` → **rc=0**, `success: true`, `passed=1 failed=0 skipped=1`; properties: `dbl_property_1 pass tests_run=100`, `headOr_property_1 skip tests_run=0` with `"no generator for parameter xs: list[int]"`. Note the skipping parameter is `[int]` — the very shape the dead `ListType` arm exists to serve.

**Impact bound — do not oversell this.** The honesty guard is **half-present, not absent**:

- An **all-skipped** suite already exits 1 (`AllSkipped()` at `result.go:104`, "NO TESTS RAN" at `reporter.go:205-209`, shipped in m-named-test-blocks v0.29.0). A minimal single-function repro therefore exits 1 and **reads as already-fixed**.
- The silent shape requires **≥1 passing test alongside ≥1 vacuous skip** — which is every real module. Any regression test that does not use the MIXED shape passes for the wrong reason.
- This is a P2 testing-honesty item, not a soundness hole in the language: nothing wrong is *proven*; checks are silently *not run*.

**Measured in-repo blast radius** (sweep of `examples/runnable/contracts/*.ail` with the worktree binary, 2026-07-29): vacuous skips are pervasive — `{x: int, y: int}` anonymous records, named record aliases (`Cell`, `Mail`, `Proposal`), ADTs (`Role`, `Region`, `Season`, `TaxBracket`, `RiskTier`, `Doc`), `list[int]`, `list[Tree]`, unit `()`, and refined `string<email>`. **Five files currently exhibit the fully-silent mixed shape** (`success=true` with vacuous skips): `inbox_injection_v2.ail` (10 skipped properties), `inbox_v2_app.ail` (10), `list_verify.ail` (6), `park.ail` (6), `record_discovery_verify.ail` (6). The first two are the **prompt-injection safety demos — their safety properties have never executed**.

**The original filing reproduces exactly at HEAD** (verified): `ailang test --format json sketches/effectbroker.ail` (cwd = `ailang-world/design_docs`, cwd-sensitive for module resolution) → rc=0, `success=true`, `total=38 passed=33 skipped=5`, five properties skip with `no generator for parameter c: Capability` / `rec: EffectRecord` (both are **record type aliases**, not ADTs), two pass at `tests_run=100`.

**Note on #517's framing**: the filing says "ADTs and records". Reality is **wider**: tuples, `[T]`/`list[T]` (dead-arm bug), unit `()`, anonymous record types in parameter position, and refined types (`string<email>`) all skip too. The filing's own repro types are records, not ADTs.

**Two adjacent honesty defects, same class** (both verified):

1. **Human mode is softer than JSON mode.** `reportPropertyHuman` prints `prop.Error` only when `Status == StatusFail` (`reporter.go:181`), so the `no generator for parameter …` reason is **dropped entirely** in human output — while *test* skips DO show their reason (`reporter.go:148`). The headline is `✓ All tests passed!` (`reporter.go:210-214`) whenever `Success()` is true.
2. **`--format json` output is not JSON.** `runTestsV2` prints `→ Running tests in <file>` to **stdout** (`cmd/ailang/test.go:18`) before the JSON object (package mode adds more preamble at `test.go:163-173`), so raw stdout does not parse with `jq` without stripping. A JSON formatter whose output isn't JSON is the same honesty class; it is small and in scope for Lane A (sized at ~10 LOC; if it doesn't fit the sprint, drop it *explicitly* into a follow-up, not silently).

---

## Goals

**Primary goal**: a contract property that never ran can no longer be mistaken for a passing one — at the exit-code level, in JSON, and in human output — and the most common parameter shapes actually get generators so the question arises less often.

**Success metrics**:
- Mixed-shape suite (≥1 pass + ≥1 no-generator skip): exit code 0 → **1** (with `--allow-skips` escape).
- `[int]` / `list[int]` parameter properties: `tests_run` 0 → **100**.
- After Lane B1, the five silent-shape example files above run their skipped properties (or fail honestly).
- `ailang test --format json … | jq .` parses with **zero** preprocessing.
- Requires-out-of-contract skips (deliberate semantics, runner.go:532-540) **still** exit 0 — no false reds introduced.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Strictness is **skip-class-scoped** (only `no_generator` skips fail the suite), not blanket "any skip fails" | Blanket strictness breaks the deliberate requires-out-of-contract-⇒-Skip semantics (runner.go:532-540) and Z3-unencodable skips — would flip legitimate suites red repo-wide | human | design | med |
| `success` in JSON + exit code change semantics; `--allow-skips` is the single opt-out (forgives vacuous skips too, via unchanged `SuccessAllowingSkips()`) | External consumers gate on it ("JSON output for CI" is documented); in-repo the only programmatic consumer is `cmd/ailang/coordinator_cloud.go:586` (`:588` = escalate-to-AI branch) | human | design | med |
| Lane B derives generators from **surface-AST `TypeDecl`s in the same file** (`r.executor.sourceFile`), not from the typechecker env | Bounds scope: imported/cross-module named types stay vacuous-skip in v1 (honest, loud). A typechecker-env approach is strictly better but ~2× the work — wrong trade for P2 | human | design | high |
| Escape hatch = **naming convention** `gen<TypeName>(seed: int) -> T`, no new syntax **(deferred with B2)** | A8 (minimal syntax); no parser change ⇒ no parser conflict surface; discoverable via error text | human | design | med |
| `valueToLiteral` default arm becomes a **loud error**, not silent unit fallback | CLAUDE.md principle 2 (no silent fallbacks affecting data integrity) — a derived generator feeding `()` into an ensures harness is this bug all over again, one layer down | human | design | low |

### Design Freeze

Before implementation begins:

- [ ] Skip-class taxonomy field name + JSON spelling (`skip_kind`: `"no_generator"` \| `"out_of_contract"`; suite counter `vacuous_skips`) — approve or rename
- [ ] `Success()` becomes `ran > 0 && failed == 0 && vacuousSkips == 0`; `--allow-skips` restores old behavior — approve semantic change
- [ ] Lane B derivation source = same-file surface AST (imported types out of scope v1) — approve bound
- [ ] Escape-hatch naming convention `gen<TypeName>` — approve spelling *(deferred with B2; approve when B2 unparks)*

### Deferred Decisions

Intentionally left to the implementer (agent may choose):

- Whether the dead `*ast.ListType` arm in `createGeneratorForType` is kept (it IS reachable from Go tests that hand-construct ASTs: `verify_callee_sort_gate_test.go`, `cycles_test.go`, `adt_test.go`) or deleted with those constructions retargeted. Recommendation: **keep both arms** — one line, and coding-standards.md's "never delete because linter says unused" rule applies.
- Exact human-mode summary wording for vacuous-skip failure (must name the count and `--allow-skips`).
- Whether the runner.go:454 skip ("top-level requires not supported") is classified `no_generator`-adjacent or left unclassified. Recommendation: leave unclassified (out of scope; it is an unsupported-construct skip, not a coverage gap).
- Lane B: recursion depth default (recommendation: 3) and whether depth exhaustion with no non-recursive constructor available reports as vacuous skip (recommendation: yes — honest).
- Test fixture file organization.

---

## Solution Design

### Overview

Two lanes, **separable**; Lane A lands first and is independently shippable. Recommendation: **sprint Lane A alone first** — it is the honesty fix (the mission's actual concern) plus a one-arm bug fix; Lane B is capability work that reduces how often the honest failure fires.

### Lane A — "make it loud" (~0.5 day)

**A1. Fix the dead list arm** (straight bug fix, not a design question). In `createGeneratorForType` (runner.go:630), add:

```go
// [T] / list[T] parses to TypeApp{Constructor: "list"} since DX-17 Phase 2.
if app, ok := typ.(*ast.TypeApp); ok {
    if app.Constructor == "list" && len(app.Args) == 1 {
        elemGen, elemShrink := r.createGeneratorForType(app.Args[0])
        if elemGen == nil {
            return nil, nil
        }
        config := DefaultConfig()
        return NewListGenerator(elemGen, 0, config.MaxSize), NewListShrinker(elemShrink)
    }
}
```

The existing `*ast.ListType` arm stays (or goes — deferred decision above).

**A2. Skip-class taxonomy.** `PropertyResult` gains `SkipKind string` (`"no_generator"`, `"out_of_contract"`, else empty). The three no-generator sites (runner.go:249 forall, :373 ensures, :490 requires) set `no_generator`; the out-of-contract site (:536) sets `out_of_contract`. `SuiteResult` gains `VacuousSkips int`, incremented in `AddPropertyResult` when `SkipKind == "no_generator"` (and merged in the two aggregation loops in `cmd/ailang/test.go:52-59, 181-188`).

**A3. Exit/`success` semantics.**

```go
// Success: tests ran, none failed, and no property was silently
// skipped for lack of a generator (vacuous pass — see #517).
func (sr *SuiteResult) Success() bool {
    ran := sr.PassedTests + sr.FailedTests
    return ran > 0 && sr.FailedTests == 0 && sr.VacuousSkips == 0
}
```

`AllSkipped()` and `SuccessAllowingSkips()` are **unchanged** — `--allow-skips` therefore already forgives vacuous skips through the existing `succeeded := Success() || (allowSkips && SuccessAllowingSkips())` at test.go:80/:208. No new flag. Composition with m-named-test-blocks is strictly additive: all-skipped stays exit 1, mixed-vacuous becomes exit 1, `--allow-skips` remains the single escape for both.

**A4. Reporter honesty.**
- JSON: add top-level `"vacuous_skips": N` and per-property `"skip_kind"`. (`success` reflects A3 automatically since reporter.go:56 calls `Success()`.)
- Human: show the skip reason for properties exactly as `reportTestHuman` already does for tests (extend the condition at reporter.go:181 to include `StatusSkip`); add a summary branch between the `AllSkipped()` and `Success()` cases: `✗ N properties never ran (no generator) — use --allow-skips to permit` + the existing non-zero exit.

**A5. JSON is JSON.** In `runTestsV2` and `runPackageTests`, route the preamble lines (test.go:18 and :163-173) to **stderr when `--format json`** (human mode unchanged). ~10 LOC. If the sprint runs over, this drops to a named follow-up — explicitly, in the sprint close-out.

**A6. Mixed-shape regression tests** — see Testing Strategy. The fixture MUST be the mixed shape (fact: a single-function repro already exits 1 via `AllSkipped()` and would pass for the wrong reason).

**No new diagnostic code is allocated** — Lane A changes exit/reporting semantics only; the existing `no generator for parameter …` message text is kept (now surfaced in human mode and classified in JSON).

### Lane B — derive generators structurally (B1 only, ~1.5–2 days; B2 deferred)

**Discovery that shrinks this lane** (verified): the combinator layer already exists in `internal/testing/generator_advanced.go` — `NewRecordGenerator`, `NewTupleGenerator`, `NewADTGenerator`, `NewOneOfGenerator`, `NewFrequencyGenerator`, `NewSizedGenerator` — plus `NewADTShrinker` in shrink.go. **None of it is wired into `createGeneratorForType`.** This is the repo's recurring guard-the-call-site-not-the-helper shape, a second time in the same function. Lane B is therefore mostly *derivation + wiring + value-splice plumbing*, not generator implementation.

**B1. Structural derivation** (~1.5–2 days):

- **Type resolution**: for `*ast.SimpleType{Name: "Point"}` not in the scalar set, look up a `TypeDecl` named `Point` in `r.executor.sourceFile.Decls` (surface AST — the runner already holds it via `SetSourceFile`). Same-file only in v1; imported types remain vacuous skips (honest, and now loud per Lane A).
- **Derivation rules**:
  - `TypeDecl{Definition: *ast.RecordType}` and **anonymous** `*ast.RecordType` in parameter position (the single most common shape in the examples sweep) → `NewRecordGenerator` over per-field derived generators.
  - `TypeDecl{Definition: *ast.AlgebraicType}` → sum-over-constructors: `NewOneOfGenerator` of one `NewADTGenerator(ctor, fieldGens, true)` per constructor. **Prerequisite fix**: `ADTGenerator.Generate` hardcodes `ModulePath: "test", TypeName: "ADT"` (generator_advanced.go:171-176) — parameterize with the real module path + type name or generated `TaggedValue`s won't match/typecheck in the harness.
  - `*ast.TupleType` → `NewTupleGenerator`.
  - `()` (unit) → constant generator (one line; in-repo occurrence verified: `cross_module_types.ail` `_: ()`).
  - `*ast.TypeApp` over a user type with args → substitute args into the decl's `TypeParams`, bounded by depth.
- **Recursion bound**: depth budget (default 3) threaded through derivation; at the bound, restrict ADT choice to non-recursive constructors (compute per-constructor recursiveness once per decl); if none exist → return no generator (vacuous skip, loud per Lane A). `NewSizedGenerator` exists if the implementer prefers it.
- **Value splice** (load-bearing, easy to miss): generated values flow into the harness via `astExprToCore(r.valueToLiteral(v))` (runner.go:393, :510). Verified: `astExprToCore` (harness.go:156) already handles `ast.Record`, `ast.Tuple`, `ast.List`, `ast.FuncCall` (constructor application), `ast.Identifier` — but **`valueToLiteral` (runner.go:691) falls through to a silent unit literal** for anything beyond scalar/list. B1 must add arms: `RecordValue → ast.Record`, `TupleValue → ast.Tuple`, `TaggedValue → ast.FuncCall(Identifier(ctor), fields)` (bare `Identifier` for nullary constructors) — and change the default arm to a loud error (signature becomes `(ast.Expr, error)`; a failure surfaces as a property **Fail**, not a skip). Without this, a derived record generator would silently test `f(())` — the same vacuous-pass bug one layer down.
- **Shrinking**: minimal honest scope — per-element `TupleShrinker` and per-field `RecordShrinker` (composing existing field shrinkers); ADTs shrink within-constructor via existing `NewADTShrinker`. Cross-constructor shrinking (e.g. `Dark(n)` → `Light`) is out of scope v1. **Shrinking behaviour for derived generators was NOT investigated beyond code reading — treat quality of shrunk counterexamples as unvalidated until B1's tests exist.**

**B2. User-supplied generator escape hatch — DEFERRED, not in Lane B's shippable scope** (~1 day when unparked) — ask (3) of #517:

> **DEFERRAL (quorum 2026-07-29).** The 2-reviewer quorum BLOCKED this doc on B2, and this doc
> adopts reviewer gpt5-6-sol's own first proposed option: **defer B2** rather than ship
> user-supplied generator execution under the recursion-depth guard alone. **B2 is BLOCKED ON a
> deterministic evaluator fuel/step budget** — one that charges fuel for every evaluation
> reduction and function application, so total evaluator work (not just call-stack depth) is
> deterministically bounded, with fuel exhaustion mapping to the same structured property Fail
> specced below. That fuel budget is a separate, larger piece of work (see Future Work) and is
> deliberately NOT designed here.
>
> **Why depth alone was judged insufficient** (credit: gpt5-6-sol's objection, adopted): the
> depth guard is real and verified — `maxRecursionDepth = 10,000` enforced at every function
> application in `evalCoreApp` (eval_operations.go:55-60, error `RT_REC_003`), no TCO in the
> tree-walking harness evaluator (TCO exists only in the bytecode VM), deterministic, and
> live-verified through the exact call path B2 would reuse: an infinitely tail-recursive pure
> generator dies in ~8ms with identical output across runs (V25/V26/V27). But that guard bounds
> **call-stack depth, not total evaluator work**: a depth-9,999 exponential-fanout generator
> passes the guard and runs arbitrarily slowly. Determinism does not make that wait operationally
> bounded, and B-8 below proves only the recursive-divergence half — it does not cover
> bounded-depth computations with explosive step or allocation growth. An earlier revision of
> this doc argued the residual was acceptable by symmetry with the function-under-test path; the
> quorum rejected that position and it is withdrawn, not overridden — B2 waits for fuel.
>
> **Consequence for v1 scope**: shapes B1 cannot derive — refined types (`string<email>`),
> imported types, function-typed fields — have **no escape hatch in v1**; they remain vacuous
> skips, made loud by Lane A (exit 1 + reason). Controller-measured 2026-07-29: 17 in-repo
> example files under `examples/runnable/contracts/` currently have skipping properties (e.g.
> `invoice.ail` 2 passed / 18 skipped, `record_discovery_verify.ail` 2/8, `inbox_injection_v2.ail`
> 1/10; six are all-skipped and therefore already exit 1 today). B1 covers the structurally
> derivable subset; the remainder waits on B2.
>
> The spec below is retained (with the quorum's hint-text correction applied) so B2 can unpark
> without re-design once the fuel budget lands.

Convention: an **exported, pure** function `gen<TypeName>(seed: int) -> <TypeName>` in the same module takes precedence over structural derivation for `<TypeName>`. Verified to compile (`ailang check` clean):

> **Escape-hatch naming applies to NAMED types only.** `<TypeName>` in this convention is always a
> declared type name (`Mail` → `genMail`). For anonymous/compound shapes there is no valid
> identifier to derive — see the hint-text branching below, which routes users through a type
> alias instead of instructing them to write invalid syntax.

```ailang
module escape

export type Point = { x: int, y: int }

export func genPoint(seed: int) -> Point ! {}
{
  { x: seed % 100, y: (seed / 100) % 100 }
}
```

- Runner detects `gen<TypeName>` while resolving a named type; per generated case it evaluates `gen<TypeName>(seed_i)` (seeds from the existing suite RNG) through the same executor machinery the harness already uses to call the function under test (`ExtractFunctionBinding` + the Core-call path; exact plumbing is implementer's choice — the harness demonstrably supports calling module functions with literal args).
- **Bounded evaluation (REQUIRED — bounded-waits axiom): deterministic fuel budget, not depth
  alone.** Every `gen<TypeName>(seed_i)` invocation MUST run under the deterministic evaluator
  **fuel/step budget** named in the deferral block above (fuel charged per evaluation reduction
  and function application), *in addition to* the existing recursion-depth guard
  (`maxRecursionDepth = 10,000`, `RT_REC_003`, verified V25/V26/V27). The depth guard alone
  deterministically stops recursive divergence but not total-work explosion — see "Why depth
  alone was judged insufficient" above. Both bounds are step-deterministic (never wall clock),
  so pass/fail results stay reproducible. **B2 does not unpark until the fuel budget exists.**
- **Exhaustion = structured property Fail, never skip or fallback.** Any error from a
  `gen<TypeName>` invocation — `RT_REC_003` depth exhaustion or fuel exhaustion alike — flows
  through the runner's existing error branch (runner.go:397-404 shape) into `Status: fail` with
  the diagnostic text, exactly as harness-evaluation errors already do. Explicitly forbidden:
  (a) reporting the property as *skip* (that would re-open the vacuous-pass hole this doc
  exists to close), and (b) silently falling back to structural derivation (a user generator
  that errors must surface, not be second-guessed). **No wall-clock timeout is used anywhere in
  this lane** — a wall-clock bound would make pass/fail nondeterministic across machines.
- Once unparked, this is the *only* answer for shapes that cannot be derived: refined types (`string<email>`), imported types, function-typed fields. Until then those shapes remain vacuous skips, made loud by Lane A.
- **Vacuous-skip hint text branches on type shape** (A11: never instruct the user to write
  invalid syntax). Naive formatting (`gen<%v>` over the AST type) would produce identifiers like
  `genstring<email>` or `gen{x: int}` — syntactically invalid AILANG. The message logic:
  - **Named `*ast.SimpleType`** (e.g. `Mail`): the type name IS a valid identifier suffix —
    `no generator for parameter m: Mail (define export func genMail(seed: int) -> Mail to supply one)`.
  - **Anything else** (anonymous records `{x: int}`, tuples, refined types `string<email>`,
    `TypeApp`s): no valid identifier can be derived — emit the alias-first path instead:
    `no generator for parameter p: {x: int} (to supply one, alias this type via export type T = ..., define export func genT(seed: int) -> T, and change the parameter type to T)`.
    The **`change the parameter type to T` clause is mandatory** (credit: gemini-3-1-pro, quorum
    2026-07-29): `gen<TypeName>` discovery only triggers when the runner resolves a *named* type,
    so aliasing + defining `genT` without changing the function signature leaves the runner
    facing the same anonymous/refined shape — structural derivation still fails and the same
    error re-emits. The pre-revision wording omitted the clause and was a DX trap: structurally
    ineffective instructions that could not resolve the error they accompany.
  Both instructed shapes are live-verified to compile: `genMail(seed: int) -> Mail` for a named
  record type, and the alias-first `export type T = { x: int }` + `genT(seed: int) -> T` +
  a function whose parameter type was changed to `p: T` (V28).
- Ordering within Lane B: **B2 is deferred entirely; Lane B ships as B1 alone.** When unparked, B1 before B2 (B2 reuses B1's type-resolution walk). The hint-text branching above is part of B2 and defers with it (the hint advertises B2's convention) — until then, Lane A keeps the existing `no generator for parameter …` message text unchanged.

### Files to Modify/Create

**Lane A:**
- `internal/testing/runner.go` (+25/-5) — TypeApp list arm; SkipKind at 4 skip sites
- `internal/testing/result.go` (+15/-3) — `SkipKind`, `VacuousSkips`, `Success()` change
- `internal/testing/reporter.go` (+25/-2) — JSON fields; human skip reasons + summary branch
- `cmd/ailang/test.go` (+15/-5) — aggregate `VacuousSkips`; preamble→stderr in JSON mode
- `internal/testing/result_test.go`, `reporter_test.go`, `named_test_test.go` — new mixed-shape tests + **deliberate** updates to tests asserting old semantics (enumerated in Conflict Surface)
- `examples/runnable/contracts/` — mixed-shape fixture file (new, small)

**Lane B (B1 only — B2 file impact deferred with B2):**
- `internal/testing/derive.go` (new, ~200 LOC) — type resolution, structural derivation, depth budget (`gen<TypeName>` lookup deferred with B2)
- `internal/testing/runner.go` (+40/-10) — wire derivation; `valueToLiteral` record/tuple/tagged arms + loud default
- `internal/testing/generator_advanced.go` (+15/-8) — parameterize ADTGenerator module/type
- `internal/testing/shrink.go` (+80) — TupleShrinker, RecordShrinker
- `internal/testing/derive_test.go` (new, ~300 LOC)
- `examples/` — one feature example per coding-standards (record+ADT+tuple ensures module; the `shapes.ail` in Examples below is ready-made)

### Examples

All snippets below were verified with `ailang check` and `ailang test` using a binary built from this worktree (`go build -o /tmp/ailang_designer ./cmd/ailang` @ `3901c14a8`).

**The canonical mixed-shape module** (today: `success=true`, 1 pass + 3 vacuous skips; post-Lane-A: exit 1; post-Lane-B: 4 × 100 cases):

```ailang
module shapes

export type Point = { x: int, y: int }

export type Shade = Light | Dark(level: int)

export func shiftX(p: Point, dx: int) -> int ! {}
ensures { result == p.x + dx }
{
  p.x + dx
}

export func level(s: Shade) -> int ! {}
ensures { result >= 0 }
{
  match s { Light => 0, Dark(l) => if l >= 0 then l else 0 }
}

export func fst2(pair: (int, int)) -> int ! {}
ensures { result == result }
{
  match pair { (a, _) => a }
}

export func anchor(x: int) -> int ! {}
ensures { result == x }
{
  x
}
```

Verified current output: `shiftX_property_1` skip (`no generator for parameter p: Point`), `level_property_1` skip (`s: Shade`), `fst2_property_1` skip (`pair: (int, int)`), `anchor_property_1` pass ×100 — `success=true`, rc=0.

**Before/after, Lane A** (the `mixed` module from Problem Statement):

| | Before | After |
|---|---|---|
| exit code | 0 | 1 |
| JSON `success` | `true` | `false` |
| JSON extras | — | `"vacuous_skips": 1`, `"skip_kind": "no_generator"` |
| human summary | `✓ All tests passed!` | `✗ 1 property never ran (no generator) …` + reason line under `⊘` |
| with `--allow-skips` | 0 | 0 |
| `headOr` (`[int]`) | skip 0 cases | **pass, 100 cases** (list-arm fix) |
| `jq .` on stdout | parse error (preamble) | parses |

---

## Conflict Surface

This design touches `internal/testing/` + `cmd/ailang/test.go` and **reads** `internal/ast` type nodes. No parser, lexer, typechecker, or codegen changes — the conflict surface is semantic (exit codes, schema, skip semantics), not syntactic.

### Consumers of the touched surfaces (enumerated)

| Surface | Consumers (verified by grep/read) | Effect of change |
|---|---|---|
| `SuiteResult.Success()` | `cmd/ailang/test.go:80,208` (exit codes); `reporter.go:56` (JSON `success`); `reporter.go:210` (human headline) | All four are the *point* of Lane A; change together |
| `AllSkipped()` | `reporter.go:205` | Unchanged |
| `SuccessAllowingSkips()` | test.go both call sites | Unchanged — becomes the vacuous-skip escape automatically |
| `createGeneratorForType` | runner.go:247 (forall), :371 (ensures), :488 (requires) — no callers outside runner.go | Lane A/B extend all three uniformly; forall path stays broken upstream of generation (separate doc) |
| `--format json` schema | In-repo: only `internal/testing/reporter_test.go`, `integration_test.go:117`. External: documented "JSON output for CI"; agents told to run `ailang test` in eval-harness prompts (`agent_prompt.go:397`, `agent_task_ailang.txt:39`); ailang-world sketches | Additive fields safe; `success` semantic change is deliberate (see below) |
| `ailang test` exit code | **No CI workflow, Makefile target, or tools/ script invokes `ailang test`** (verified; `tools/test-concurrency.sh`'s `"test"` is an API arg string, not this command). Only programmatic consumer: `cmd/ailang/coordinator_cloud.go:586` (`ailang test --package .` deterministic gate in execute-job, no `--allow-skips`) | A package whose properties vacuously skip flips from silent-green to escalate-to-AI — the honest behavior; noted, accepted |
| Go tests asserting current semantics | `named_test_test.go` (AllSkipped/Success suite, lines 153-310), `result_test.go`, `reporter_test.go` | AllSkipped tests keep passing (semantics unchanged). Tests asserting `Success()==true` for pass+skip mixes without kind classification: **updated deliberately** — the executor enumerates each in the sprint log; anything updated beyond that list is a regression |

### Programs that MUST still work (regression fixtures)

All verified to exist and their current behavior measured (2026-07-29 sweep):

1. `examples/runnable/contracts/basic.ail` — scalar-typed properties; its passing properties must keep passing with unchanged case counts.
2. `examples/runnable/contracts/quantifier_verify.ail` — 4 skips with **vacuous=0** (non-generator skip class); must still exit as today: its skips are NOT reclassified.
3. `examples/runnable/contracts/unencodable_callee_skip.ail` — the Z3-unencodable skip class; must remain unaffected by property-skip taxonomy.
4. `examples/runnable/contracts/ensures_violation_demo.ail` — a failing suite; must keep failing for the *failure* reason, not acquire a vacuous-skip label.
5. New mixed-shape fixture (this doc) — the Lane A acceptance fixture.
6. A requires-out-of-contract case (runner.go:532-540 semantics): out-of-contract skip + passing sibling test ⇒ **exit 0 preserved**.

### What deliberately changes

- Mixed suites with `no_generator` property skips: exit 0 → 1 (escape: `--allow-skips`). Anything relying on the current false green — that reliance is the bug (same stance as m-named-test-blocks' deliberate change).
- JSON `success` may flip `true→false` for such suites; two additive JSON fields.
- JSON-mode stdout carries only JSON (preamble → stderr).
- Lane B: previously-skipped properties on derivable types now run — a *latent-bug detector*: ensures clauses that were never exercised may genuinely fail (e.g. `record_verify.ail`'s deliberately-`brokenDistance` properties will start counterexampling, and `park.ail`/`list_verify.ail`'s never-run properties may fail). Expected, honest; the sprint must budget triage time for newly-failing examples rather than treat them as regressions.

---

## Testing Strategy

**Every acceptance criterion names a concrete production-code mutation that turns it red.**

### Lane A acceptance criteria

| # | Criterion | Red-turning mutation |
|---|---|---|
| A-1 | Mixed-shape fixture (1 passing scalar property + 1 `Point`-param property): `ailang test` exits **1**, JSON `success=false`, `vacuous_skips=1` | Revert `Success()` to `ran > 0 && failed == 0` (drop the `VacuousSkips` term) |
| A-2 | Same fixture with `--allow-skips`: exit **0** | Make the allow-skips branch check `VacuousSkips == 0` too (i.e. stop forgiving vacuous) |
| A-3 | `headOr(xs: [int], …)` ensures-property runs **100 cases** and passes | Delete the new `*ast.TypeApp` "list" arm in `createGeneratorForType` |
| A-4 | Requires-out-of-contract skip + passing sibling: exit **0**, `vacuous_skips=0`, `skip_kind="out_of_contract"` | Set `SkipKind = "no_generator"` at runner.go:536 (misclassify) |
| A-5 | `ailang test --format json f.ail \| jq .` succeeds on raw stdout | Restore the `fmt.Printf("→ Running tests…")` to stdout in JSON mode |
| A-6 | Human output for the mixed fixture contains `no generator for parameter` and does NOT contain `All tests passed!` | Revert reporter.go:181 condition to `StatusFail`-only (reason drop) / revert summary branch (headline) |
| A-7 | All-skipped suite still prints `NO TESTS RAN` + exit 1 (m-named-test-blocks invariant) | Make `Success()` return `FailedTests == 0` (would claim success for all-skipped) |

### Lane B acceptance criteria (B1 — in shippable scope)

| # | Criterion | Red-turning mutation |
|---|---|---|
| B-1 | `shapes.ail` (above): all four properties run 100 cases, exit 0 | Delete the `RecordType` (resp. `AlgebraicType`, `TupleType`) derivation arm — the property reverts to vacuous skip, which Lane A makes exit 1 |
| B-2 | Anonymous record param `{x: int, y: int}` (e.g. `record_verify.ail`'s `getX`) runs 100 cases | Restrict derivation to named `TypeDecl`s only (drop the direct `*ast.RecordType` arm) |
| B-3 | ADT-generated `TaggedValue`s pattern-match correctly in `match` inside the function under test (the `level`/`Shade` property passes, both constructors observed across the run) | Restore hardcoded `ModulePath: "test", TypeName: "ADT"` in `ADTGenerator.Generate` |
| B-4 | `valueToLiteral` on an unknown `eval.Value` produces a property **Fail** with an explicit error, never a silent `()` splice | Restore the silent unit-literal default arm |
| B-5 | Recursive ADT (`type Tree = Leaf \| Node(kids: [Tree])`) generation terminates within depth bound; property runs 100 cases | Remove the depth budget (test then hangs/overflows — enforce with a test timeout) |
| B-7 | Imported named type still vacuous-skips **loudly** (exit 1, existing `no generator for parameter …` message text — the `gen<TypeName>` hint is deferred with B2) | Make unresolvable types silently return a unit-constant generator |

### Deferred B2 acceptance criteria (apply only when B2 unparks; not part of any sprint until the fuel budget lands)

| # | Criterion | Red-turning mutation |
|---|---|---|
| B-6 | `genPoint` escape hatch: with a `gen<TypeName>` producing only `x >= 0` values and an `ensures { p.x >= 0 => … }`-style property that structural generation would counterexample, the suite passes (user generator took precedence) | Remove the `gen<TypeName>` lookup (structural derivation generates negative `x`, property fails) |
| B-8 | **Nonterminating generator is deterministically bounded**: a fixture whose `genT` infinitely recurses (tail-recursively, to also pin no-TCO) terminates the suite promptly with property `Status: fail` whose error contains `RT_REC_003`, identical JSON (modulo durations) across two consecutive runs; never a skip, never a hang (enforce with a Go test timeout). **Scope note (quorum 2026-07-29): this proves only the recursive-divergence half. On unpark, add a sibling criterion: a bounded-depth exponential-fanout `genT` fails with the fuel-exhaustion diagnostic under the fuel budget** | In the B2 generator-invocation path, catch the evaluation error and fall back to structural derivation (or map it to `StatusSkip`) instead of failing — B-8's fail-status and diagnostic assertions go red |
| B-9 | **Hint text never emits an invalid identifier AND its instructions are structurally effective**: for a named-type parameter (`m: Mail`) the skip hint contains `genMail(seed: int) -> Mail`; for an anonymous-record parameter (`p: {x: int}`) and a refined-type parameter (`s: string<email>`) the hint contains the alias-first wording — `export type T = ...`, `define export func genT(seed: int) -> T`, **and `change the parameter type to T`** — and does NOT contain `gen{` or `genstring<` | Collapse the two hint branches into the naive single-branch `fmt.Sprintf("gen%v", typ)` formatting (invalid identifiers emit, negative assertions go red); OR drop the `change the parameter type to T` clause — the instructions become structurally ineffective (`gen<TypeName>` discovery only fires on named-type resolution, so the runner re-encounters the anonymous shape and re-emits the same error) and B-9's clause assertion goes red |

**Unit tests**: skip-classification per runner site; `Success()`/`VacuousSkips` truth table; derivation per type shape; `valueToLiteral` round-trips (value → literal → core → harness-evaluated equality). **Integration**: CLI-level runs of the fixture files pinning exit code + parsed JSON. **Regression-surface**: one test per "Programs that MUST still work" entry above.

---

## Non-Goals

- **Fixing the `properties [forall(...) => ...]` evaluation path** — known-broken (`empty program`), separately documented and parked (P3) in [m-forall-properties-direct-core-eval](../v1_1_0/m-forall-properties-direct-core-eval.md). Not re-derived here; not a new finding.
- **User-supplied generator execution (B2)** — DEFERRED, blocked on a deterministic evaluator fuel budget (quorum 2026-07-29; see Lane B2).
- **Imported / cross-module named types** in Lane B v1 — remain honest vacuous skips (loud per Lane A); the deferred B2 escape hatch covers them when unparked.
- **Refined types** (`string<email>`) — no structural derivation (would need refinement-aware generation); no v1 path — deferred B2 escape hatch when unparked.
- **Cross-constructor ADT shrinking** and shrink-quality guarantees for derived generators.
- **Polymorphic parameter types** (`a` with class constraints) — out of scope.
- **New CLI flags or syntax** — `--allow-skips` is reused; the escape hatch is a naming convention.
- **Changing test-level (non-property) skip semantics** — Z3-unencodable and other test skips keep current behavior.

---

## Timeline

**Lane A (~0.5 day)**: list arm + taxonomy + `Success()` + reporter (3h) · preamble-to-stderr (0.5h) · mixed-shape fixtures + deliberate test updates (2h).

**Lane B (B1 only, ~1.5–2 days)**: B1 derivation + splice + shrinkers + tests (1.5–2d) · triage of newly-running example properties (budgeted inside B1 — triage volume is measured, not speculative: controller sweep 2026-07-29 found 17 contract example files with skipping properties, e.g. `invoice.ail` 2 passed / 18 skipped, `record_discovery_verify.ail` 2/8).

**B2 (~1 day when unparked): DEFERRED** — blocked on the deterministic evaluator fuel budget (quorum 2026-07-29; see Lane B2 and Future Work). Not scheduled in any sprint from this doc.

Honest sizing note: this is a P2. If only Lane A ever ships, the mission's vacuous-pass concern is addressed; Lane B is capability, not honesty.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Exit-code strictness flips consumer suites red (external users, ailang-world sketches, coordinator execute-job) | Med | Scoped to `no_generator` class only; `--allow-skips` escape; CHANGELOG breaking-change entry; in-repo audit found no CI consumer |
| Blanket-skip misclassification breaks requires-discard semantics | High if missed | A-4 regression test pins it; taxonomy is per-site explicit, not inferred from message text |
| Lane B generated values don't survive the `valueToLiteral → astExprToCore` splice for some shape | Med | B-4's loud-error default converts silent corruption into visible Fail; round-trip unit tests |
| Newly-running properties fail on real bugs in examples (`brokenDistance` is deliberate; others unknown) | Med (sprint time) | Budgeted triage in B1; failures are the feature working |
| `gen<TypeName>` convention collides with an existing user function of that shape but different intent *(deferred with B2)* | Low | Precedence documented + hint text; type-checked signature must match `(int) -> T` exactly or it is ignored with a stderr note |
| Depth-bound recursion misses non-obvious cycles (mutual recursion via records-of-ADTs) | Med | Depth budget decrements on *every* derivation step, not just ADT arms — B-5 uses `list[Tree]` (verified in-repo occurrence, `cross_module_types.ail`) |
| User-supplied `gen<TypeName>` runs unbounded total work — depth guard bounds call-stack depth, not evaluator steps (bounded-waits axiom) | High if shipped under depth alone | **Resolved by deferral (quorum 2026-07-29)**: B2 is out of shippable scope, BLOCKED ON a deterministic evaluator fuel budget. The verified depth guard (V25/V26/V27) closes only the recursive-divergence case; exhaustion-as-structured-Fail and no-wall-clock-timeout remain specced for unpark (B-8) |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Derived generators use the existing seeded RNG (`newRNG(config.Seed)`); no new nondeterminism |
| A2: Replayability | 0 | No trace-surface change |
| A3: Effect Legibility | 0 | Properties remain over pure code; escape-hatch generators are pure functions |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +2 | The headline: contract checks that reported success without running now either run (Lane B) or fail loudly (Lane A). Five in-repo modules' never-executed safety properties start executing |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | `--format json` becomes actually machine-parseable; exit codes stop lying to agents that gate on them |
| A8: Minimal Syntax | 0 | Zero new syntax (escape hatch is a naming convention; flag reused) |
| A9: Cost Visibility | 0 | No cost-model change |
| A10: Composability | +1 | Wires the existing-but-orphaned generator combinator layer into the type-driven path; composes with m-named-test-blocks' exit semantics instead of adding a parallel mechanism |
| A11: Structured Failure | +1 | Skip taxonomy (`skip_kind`) turns an ambiguous skip string into structured, classifiable failure data |
| A12: System Boundary | 0 | No FFI/boundary change |

**Net Score: +5** → Proceed.

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): actively improves machine consumers

---

## Verification Log

All commands run 2026-07-29 in worktree `.wt-iter121` (branch `sprint/m-property-generator-coverage`, base `origin/dev` @ `3901c14a8`), binary `go build -o /tmp/ailang_designer ./cmd/ailang`.

| # | Claim | Evidence |
|---|---|---|
| V1 | `createGeneratorForType` has exactly 2 arms (4 scalars + ListType); all else `nil, nil` | Read `runner.go:630-661` |
| V2 | Honesty guard half-present: `Success()` requires ran>0; `AllSkipped()`, `SuccessAllowingSkips()`, `--allow-skips` exist and are wired | Read `result.go:95-112`, `test.go:80,208`, `main.go:117` |
| V3 | Mixed-shape repro: rc=0, `success=true`, 1 pass + 1 vacuous skip; `ailang check` clean | Live run of `mixed.ail` (Problem Statement); JSON captured |
| V4 | `*ast.ListType` arm unreachable from parsed programs | (a) grep `ListType{` over non-test `.go`: constructions ONLY in `verify_callee_sort_gate_test.go`, `cycles_test.go`, `format/node_coverage_test.go`, `apiserver/named_args_test.go`, `gen/golang/adt_test.go`; (b) `parser_type.go:56` builds `TypeApp{Constructor: "list"}`; (c) **behavioral**: V3's `[int]` param skips with `list[int]` in the message |
| V5 | #517 reproduces exactly at HEAD | Live run in `ailang-world/design_docs` (cwd-sensitive: from repo root, module resolution fails with LDR001 instead): `success=true, total=38, passed=33, skipped=5, len(tests)=31, len(properties)=7`; skip reasons `c: Capability`, `rec: EffectRecord` |
| V6 | `Capability`/`EffectRecord` are record aliases, not ADTs | `grep "type Capability" effectbroker.ail` → `export type Capability = {` (line 16); `EffectRecord` line 154 |
| V7 | Human mode drops property-skip reasons but shows test-skip reasons; headline `✓ All tests passed!` on `Success()` | Read `reporter.go:148` (tests: Fail *or* Skip) vs `:181` (properties: Fail only); `:205-219` summary switch |
| V8 | JSON preamble on stdout | Read `test.go:18` (Printf before reporter, same stream); live: first stdout line `→ Running tests in mixed.ail`, then JSON |
| V9 | Requires out-of-contract ⇒ whole-property Skip is deliberate | Read `runner.go:431-443` (doc comment), `:532-540` (implementation) |
| V10 | Explicit `properties [forall…]` surface is known-broken (`empty program`), separately planned | `design_docs/planned/v1_1_0/m-forall-properties-direct-core-eval.md` exists (P3, 2026-05-15); runner.go:220-223 comment confirms "broken EvaluateExpression source-synthesis path" |
| V11 | **Negative**: no CI workflow / Makefile target / tools script consumes `ailang test` exit code | grep `.github/workflows/*.yml`, `Makefile`, `tools/` — only `tools/test-concurrency.sh` matches on an unrelated API arg string; sole programmatic consumer `cmd/ailang/coordinator_cloud.go:586` (path corrected in revision pass — see V30) |
| V12 | **Negative**: JSON schema fields consumed in-repo only by the package's own tests | grep `passed_tests\|total_tests\|skipped_tests` repo-wide → `reporter.go`, `reporter_test.go`, `integration_test.go:117` only |
| V13 | Record/Tuple/ADT/OneOf/Frequency/Sized generators + ADTShrinker already exist, unwired | Read `generator_advanced.go` (constructors at :155-261), `shrink.go:255`; V1 shows no wiring |
| V14 | `ADTGenerator.Generate` hardcodes `ModulePath: "test", TypeName: "ADT"` | Read `generator_advanced.go:171-176` |
| V15 | **Negative**: no RecordShrinker/TupleShrinker exists | grep `func New.*Shrinker` in `shrink.go` → Int/Float/String/List/ADT/NoOp only |
| V16 | `valueToLiteral` silently falls back to unit for non-scalar/list values | Read `runner.go:691-727` (default arm returns `UnitLit`) |
| V17 | `astExprToCore` covers `Record`/`Tuple`/`List`/`FuncCall`/`Identifier`; panics loudly on default | Read `harness.go:156-249` |
| V18 | All AILANG snippets in this doc compile | `ailang check` clean on `mixed.ail`, `shapes.ail`, `escape.ail` (worktree binary) |
| V19 | `shapes.ail` behavior: record/ADT/tuple params all vacuous-skip; scalar passes; `success=true` | Live `ailang test --format json` run (Examples section) |
| V20 | In-repo blast radius: 5 example files in silent mixed shape; vacuous skips across ~15 contract examples incl. anonymous records, unit, `string<email>`, `list[Tree]` | Per-file sweep of `examples/runnable/contracts/*.ail` with worktree binary (output archived in iteration log) |
| V21 | `quantifier_verify.ail` has 4 skips with vacuous=0 (distinct skip class exists in the wild) | Same sweep |
| V22 | **Negative**: no new diagnostic code allocated by this design (nothing to collide) | Design decision — Lane A reuses existing message text + adds JSON fields |
| V23 | Escape-hatch function shape compiles | `ailang check` clean on `escape.ail` (`genPoint(seed: int) -> Point`) |
| V24 | `AddPropertyResult` counts property skips into the same counters as tests; aggregation duplicated in `test.go:52-59,181-188` | Read `result.go:80-93`, `test.go` |
| V25 | **The bounded-evaluation API exists and is on the property-test call path**: `CoreEvaluator.maxRecursionDepth` (default 10,000, `eval_evaluator.go:148`; setter `SetMaxRecursionDepth` at `:519-521`; CLI `--max-recursion-depth`), enforced on every `*FunctionValue` application in `evalCoreApp` (`eval_operations.go:55-60`, error `RT_REC_003`); the harness reaches it via `evaluateEnsuresHarnessCore` → fresh `eval.NewCoreEvaluator()` (`executor.go:150`) → `EvalCoreProgram` (`:162`) | Read all four sites (revision pass, binary `go build -o /tmp/ailang_designer2 ./cmd/ailang`) |
| V26 | **Live**: a nonterminating pure function invoked through the ensures-property harness terminates deterministically — `spin(x) = spin(x)` (tail-recursive, so this also proves the guard is not TCO-evaded) → property `status: fail`, `error: "test 0: ensures harness evaluation failed: RT_REC_003: max recursion depth 10000 exceeded…"`, `tests_run: 1`, suite `success: false`, rc=1, wall time ~8ms; two consecutive runs byte-identical modulo `duration` fields | Live `ailang test --format json` on `diverge.ail` (worktree binary), run twice + diff |
| V27 | TCO exists **only** in the bytecode VM (`internal/vm`, `OpTailCall`), not in the tree-walking `CoreEvaluator` the test harness uses; v0_7_3 design doc states "No TCO in AILANG. Recursion guarded by depth counter" | grep `TailCall\|tail-recursion` repo-wide → hits only in `internal/vm`, `internal/bytecode`, archived codegen docs; behavioral proof in V26 |
| V28 | **Both hint-text instructed shapes compile**: named-type `genMail(seed: int) -> Mail` and alias-first `export type T = { x: int }` + `genT(seed: int) -> T` + consumer `getX(p: T)` — `ailang check` clean | Live check of `aliashint.ail` (worktree binary) |
| V29 | **Negative, instrument-proven**: the evaluator has NO step/fuel counter — only the depth budget (V25) and *effect* budgets (`EffectBudgets`/`@limit`, `value.go:311`, which bound effect operations, not pure computation steps). The same grep instrument that found the effect-budget positives found no step/fuel machinery | grep `fuel\|Fuel\|StepBudget\|MaxSteps\|StepLimit` over `internal/` → 0 hits; grep `budget\|Budget\|[Ll]imit` over `internal/eval/` → effect budgets + recursion depth only (positive control: same pattern found both known mechanisms) |
| V30 | `cmd/ailang/coordinator_cloud.go:586` is `exec.CommandContext(ctx, "ailang", "test", "--package", ".")`; `:588` is the `testErr != nil` → `"ailang test failed (escalating to AI)"` branch | Read `cmd/ailang/coordinator_cloud.go:578-592` (revision pass; corrects V11's path, which omitted the `cmd/ailang/` directory) |

**Not verified / open for the executor**: shrink quality of derived generators (V-none — flagged in Lane B); whether external (out-of-repo) consumers parse `success` (unknowable from here; mitigated by `--allow-skips` + CHANGELOG). The `gen<TypeName>` call path's *depth-bounded evaluation and error-to-Fail behavior* is verified (V25/V26), but B2 is deferred regardless — depth bounds divergence, not total work (quorum 2026-07-29); the verified envelope is retained for unpark.

### Quorum verification log

**2026-07-29 — 2-reviewer quorum BLOCKED (rc=3); bounded revision pass applied same day.**
Artifact: `.ailang/state/mission-quorum/m-property-generator-coverage-2026-07-29T21-50-41Z.json`.
Both objections landed exclusively on Lane B2; neither reviewer objected to Lane A or B1.

| Reviewer | Verdict | Objection (one line) | Resolution applied |
|---|---|---|---|
| gpt5-6-sol | REJECT | B2's recursion-depth guard bounds call-stack depth, not total evaluator work — the doc explicitly permitted a depth-9,999 exponential-fanout generator to run arbitrarily slowly; determinism does not make that wait operationally bounded, and B-8 proves only the divergence half | Adopted the reviewer's **first** proposed option: **B2 DEFERRED** out of Lane B's shippable scope, BLOCKED ON a deterministic evaluator fuel budget (fuel per evaluation reduction / function application) — **deferred on the reviewer's own proposal, not overridden**. No fuel-budget design was invented here. Updated: Dependencies + Estimated headers, Lane B heading, B2 section (deferral block), Files, acceptance tables (B-6/B-8/B-9 moved to a deferred table; B-7 re-scoped), Timeline, Risks, Non-Goals, Future Work. Lane B = B1 only. The withdrawn depth-alone-sufficiency prose ("Bounded evaluation (REQUIRED…)" block + "Honest residual" paragraph) was relocated under deferred B2 as "why depth alone was judged insufficient", crediting the objection |
| gemini-3-1-pro | REJECT | Anonymous/refined-type hint text told users to alias the type to `T` and define `genT` but omitted changing the function's parameter type to `T` — since `gen<TypeName>` discovery only fires on named-type resolution, following the hint re-emits the same error (structurally ineffective instructions; DX trap) | Hint wording in the (deferred) B2 spec now reads `…, define export func genT(seed: int) -> T, and change the parameter type to T`, with the mandatory-clause rationale recorded inline; acceptance criterion B-9 updated to assert the signature-change clause and to name its omission as a red-turning mutation |

Lane A (A1–A6, its acceptance-criteria table, and mutation column) was left **byte-identical** in
this revision — no reviewer objected to it; it routes to the sprint-planner unchanged.

---

## References

- **Upstream filing**: sunholo-data/ailang#517 (open) — asks: (1) derive record/ADT generators, (2) loud skips, (3) user-supplied generator hook. This doc covers all three; notes the filing's type coverage framing is narrower than measured reality.
- [m-named-test-blocks](../../implemented/v0_29_0/m-named-test-blocks.md) — the half of the guard that already shipped
- [m-forall-properties-direct-core-eval](../v1_1_0/m-forall-properties-direct-core-eval.md) — the parked sibling surface
- [m-dx26-property-test-empty-program](../../implemented/v0_21_0/m-dx26-property-test-empty-program.md) — harness paths extended here
- Program north star: [design_docs/PROGRAM.md](../../PROGRAM.md) — vacuous-pass class tracking

## Future Work

- **Evaluator step-fuel budget** (deterministic total-work bound: fuel charged per evaluation reduction and function application, not just call depth) — **now the named BLOCKING DEPENDENCY for deferred B2** (quorum 2026-07-29): the depth guard bounds recursive divergence, not total work. Benefits the whole test harness (function-under-test calls share the same exposure), not just generators. When it lands, B2 unparks with no semantic change to its retained spec (exhaustion is already specced as structured Fail).
- Typechecker-env-based type resolution (unlocks imported types) — supersedes the same-file bound
- Refinement-aware generation for `string<...>` taint/refined types
- Cross-constructor ADT shrinking; integrated shrink-quality metrics
- Surface `vacuous_skips` in the eval-harness bank schema so agent-written contract tests are scored on *executed* properties
