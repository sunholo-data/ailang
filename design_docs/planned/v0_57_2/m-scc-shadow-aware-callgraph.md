# M-SCC-SHADOW-AWARE-CALLGRAPH: binder-shadowed names must not create call-graph edges (#2078)

**Status**: Planned
**Target**: v0.57.2 (release after v0.57.0 / the queued v0.57.1)
**Priority**: P1 (High) — a sound, documented-language program is rejected with a wrong diagnostic; no data or soundness loss
**Estimated**: 2.5 days (1 day walker + binder classification, 1 day tests/e2e matrix, 0.5 day corpus sweep + changelog)
**Dependencies**: none (independent of the #1958 re-land; found while verifying it)

**Issue**: https://github.com/sunholo-data/ailang/issues/2078

## Problem Statement

A row-polymorphic recursive function is rejected with "Missing effects" as soon as a sibling
module function (a) is named exactly like a **binder** inside the recursive function's body, and
(b) calls the recursive function with a **closed-row** callback. The issue's repro, verified on
the v0.57.0 binary (commit `b23cadc`) on 2026-11-03:

```ailang
module r2b
import std/io (println)
func each[e](f: (int) -> () ! {e}, xs: [int]) -> () ! {e} = match xs { [] => (), h :: t => { f(h); each(f, t) } }
export func t() -> () ! {IO} = each(\x. println("x"), [1, 2])
```

```
Error: effect checking failed in r2b: Effect checking failed for function 'each'
  Function uses effects not declared in signature
  Missing effects: IO
  Current signature: func each(...) -> T ! {e}
  Suggested fix:     func each(...) -> T ! {IO, e}
```

`each` alone checks clean. The diagnostic is wrong twice: `each` uses no IO, and the suggested
fix (`! {IO, e}`) poisons `each` — any pure caller would then be charged IO.

### Root cause (traced, not guessed)

`internal/elaborate/scc.go:181 findReferences` builds the module call graph from the **surface
AST** and is scope-blind: it walks `*ast.Match` case bodies and guards but **never case
patterns**, and it does not track lambda params or `let` binders either. So in `each`'s body
the pattern-bound `t` (`h :: t`) is counted as a reference to the module function `t`.

That false edge `each → t` closes a cycle with the sibling's **genuine** call edge
`t → each` (`t`'s body really calls `each`), merging the two into one Tarjan SCC `{t, each}`
(observed: with the sibling named `t` the core program is `LetRec{t, each}`; renamed to `u`
the false edge is gone and it is `LetRec{each}` + `Let{u}`; a non-calling sibling `t` stays
`Let{t}` + `LetRec{each}` — a one-way false edge alone does not group). Inside one SCC, `inferLetRec` (internal/types/typechecker_functions.go:286) infers the
group **monomorphically** — one shared effect-row variable for `each`'s declared `! {e}` across
all group call sites. The sibling `t`'s call `each(\x. println("x"), …)` unifies the callback
slot's row with the lambda's **closed** `{IO}` row, resolving `e := {IO}` for the whole group.
The recursive self-call inside `each` then publishes its `ApplicationEffects.CallRow` as the
resolved `{IO}` (`DEBUG_EFFECTS` trace: `Published App 17 call row: {IO}` in the failing
module vs `Published App 6 call row: {...e}` in the passing control), and the post-check effect
validator's App case (internal/pipeline/validate_effects.go, `!usedDeclared || calleeEffects.Tail != nil`
override) charges that site row to `each`'s **definition**. `{IO}` has no tail, `! {e}` covers no
concrete label, so `SubsumeEffectRows({IO}, {e})` fails.

### Verified reproduction matrix (all run 2026-11-03, v0.57.0 binary)

| # | Variant | Result |
|---|---------|--------|
| 1 | Issue repro (pattern var `t`, sibling caller `t` with IO callback) | FAIL "Missing effects: IO" on `each` |
| 2 | `each` alone (no sibling) | PASS |
| 3 | Sibling renamed `u` (pattern var still `t`) | PASS — recursive self-call publishes `{...e}` |
| 4 | Pattern var renamed `rest` (sibling still `t`) | PASS |
| 5 | Shadow via **lambda param**: `each(\t. t, r)` | FAIL — same mischarge |
| 6 | Shadow via **let binder**: `let t = r in each(f, t)` | FAIL — same mischarge |
| 7 | Sibling `t` calls `each` with a pure unannotated callback | PASS (open row aliases `e`, no resolution) |
| 8 | Sibling `t` does not call `each` at all | PASS — `Let{t}` + `LetRec{each}`: the one-way false edge makes no cycle, `e` stays free |
| 9 | Genuine mutual recursion `even`/`odd`, both `! {e}`, no shadowing | PASS |
| 10 | True SCC where a member passes a closed-row callback to a group-mate | FAIL on that member — **correct** (its body really performs IO) |
| 11 | Self-recursive `d[e]` whose self-call passes a concrete IO lambda | FAIL on `d` — **correct** (per-site charging at self-calls is load-bearing soundness) |

Row 10/11 matter for fix design: the validator and the `ApplicationEffects` publication are
behaving *correctly per-site*; the wrongness enters purely through the false SCC. Rows 5/6 show
this is a **defect class** (all binder forms), not a match-pattern one-off.

### Impact

- Who: any module that shadows a module-level name with a local binder (match pattern, lambda
  param, `let`) **and** references the shadowed binder — then any sibling that calls the
  shadowing-adjacent function at a closed row. AI-generated code does shadow names like `t`.
- Prevalence: the issue reports 0 hits in the 20,046-file v0.57.0 pre-release sweep (this repo +
  28 repos + 15,447 eval-generated programs); std combinators (`forEachE`, `mapE`, …) are
  unaffected. So this is a correctness-of-diagnosis fix, not an epidemic.
- Not a regression: v0.56.1 behaves identically. It surfaced while verifying the #1958 re-land
  because the old union bug used to mask variant #1's sibling call's IO.

## Goals

**Primary Goal:** make `findReferences` (and therefore `BuildCallGraph` / `isSelfRecursive`)
scope-aware, so a binder-shadowed name never creates a call-graph edge; the issue repro and the
matrix rows 1/5/6 then type-check, and rows 10/11 keep being rejected.

**Success Metrics:**
- Issue repro (r2b.ail) checks clean; matrix rows 1–9 all pass.
- Matrix rows 10–11 (genuinely unsound) are still rejected — no soundness relaxation.
- `internal/elaborate/module_let_resolution_test.go` (TestModuleLetResolutionMatrix) and the
  `scope_test.go` shadowing matrix pass unchanged.
- Corpus sweep: `ailang check` status flips = 0 over this repo's `examples/` + `std/` + the
  1,066-file sweep corpus used by recent fixes (the same harness as the #2074/#1958 checks).
- New unit tests exist for the call graph itself (`scc.go` today has **no** test file).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix at the call graph (elaboration), not at the validator or `ApplicationEffects` publication | The validator's per-site charging is load-bearing (matrix rows 10/11); the publication is only contaminated because the false SCC forced group-wide row sharing | human (this doc) | design | high |
| Binder classification for surface patterns is syntactic (lowercase identifier = binder), mirroring `elaboratePattern` | Wrong classification in the *permissive* direction drops genuine edges → #327 false "undefined variable"; in the *strict* direction keeps the status quo bug | human (this doc) | design | med |
| Thread a bound-names set through the whole `findReferences` walk (all binder forms), not a match-pattern-only patch | Rows 5/6 show the same false edge from lambda params and let binders; a one-position patch re-sells the audit later | agent (may pick representation) | implementation | low |
| Complete the walker's missing cases (`ForallExpr`, `Send`, `Recv`, `AssertStmt`, `QuasiQuote` interpolations) in the same pass | Same function, same #327 edge-dropping family; `ForallExpr` additionally binds `Var` | human (this doc) | design | low |
| Do NOT add polymorphic recursion to SCC inference | HM monomorphic groups are the documented design; true-SCC programs are rejected only when genuinely unsound | human (this doc) | design | high |

## Design Freeze

- [x] Fix lives in `internal/elaborate/scc.go` (+ a new `internal/elaborate/scc_test.go`); no
      changes to `internal/types/` or `internal/pipeline/`.
- [x] Surface binder classification: only **lowercase** pattern identifiers bind (see
      "Disambiguation strategy" for why that is exact, not heuristic).
- [x] The validator (`validate_effects.go`) and `ApplicationEffects` publication are untouched
      and gain regression tests instead.
- [ ] Exact e2e test file placement (internal/pipeline vs internal/elaborate external test
      package) — resolved in M1 after checking import-direction constraints (see Deferred).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Internal representation of the bound set (parameter vs. walker struct vs. copied map) — agent
  may choose; note `ownSets`-style memoisation is *not* needed here (each walk is once-per-body).
- Whether the pattern-binder helper (`astPatternBinders`) lives in `scc.go` beside its only
  caller or in `patterns.go` beside `corePatternBinders` — agent may choose.
- Test naming/organization inside `scc_test.go` — agent may choose.
- Whether to also enumerate `tests [...]` / contract property expressions in `BuildCallGraph`
  (today only `f.Body` is walked) — **do not expand scope**; no known repro; note it in the
  changelog as an observed adjacent limit if touched by review.

## Solution Design

### Overview

`findReferences` answers "which module-level names does this body reference?" for the SCC
grouping (and `isSelfRecursive`). Today it answers "which names are *mentioned*?", which is
wrong whenever a binder shadows a module-level name. The fix makes the walk track lexical
binders, exactly the way the elaborator's own scope resolution (internal/elaborate/scope.go,
`pushScope`, and the `TestLexicalScope_BinderShadowsGlobals` matrix) already does for *name
resolution*. The call graph then only gets edges from genuinely free references.

This is deliberately the same shape as prior fixes in this family: #327 made the walk
exhaustive over expression *positions*; this makes it correct about *scope*.

### Architecture

**Components:**

1. **`astPatternBinders(pat ast.Pattern) []string`** — surface-pattern binder walker.
   Mirrors `corePatternBinders` (internal/elaborate/scope.go:53) at the AST level:
   `*ast.Identifier` → binder iff not `_` and not uppercase; `ConstructorPattern` → recurse
   `Patterns` (its `Name` is a constructor, not a binder); `ConsPattern` → `Head`, `Tail`;
   `ListPattern` → `Elements`, `Rest`; `TuplePattern` → `Elements`; `RecordPattern` → each
   `FieldPattern.Pattern` (field names bind nothing); `WildcardPattern`/`LiteralPattern`/`nil`
   → none.

2. **Scope-aware `findReferences`** — same public signature (`findReferences(expr ast.Expr) []string`,
   still used by `isSelfRecursive`), internally threading a `bound map[string]bool`:
   - `*ast.Identifier`: emit only if not bound.
   - `*ast.Lambda`, `*ast.FuncLit`: body walked with params bound.
   - `*ast.Let`: value walked with the current bound (a `let` binder is not in
     scope in its own value); body walked with the binder added.
   - `*ast.LetRec`, split by shape (file.go:134 distinguishes the same way):
     - `Body == nil` — a **module-level bodyless `letrec x = v`**: the value is
       walked with the current bound and the binder **not** filtered. Its
       self-reference is exactly what `isSelfRecursive`/Tarjan use to emit it
       as `core.LetRec` (file.go:118) — filtering it would dangle the binding.
     - `Body != nil` — a **local `letrec x = v in b`**: value and body walked
       with the binder added (local binder shadows the module name; the
       scope matrix row `letrec_value_and_body` pins the resolution).
   - `*ast.Match`: scrutinee with current bound; per case, guard and body walked with that
     case's `astPatternBinders(pattern)` added.
   - `*ast.Block`: statements walked in order, binders threaded forward (a `let`-statement
     binds for *subsequent* statements only — this preserves the "use before let" case from the
     scope matrix, where the *earlier* statement's reference must stay). Note the parser folds
     `let t = 2; rest` inside a block into a `*ast.Let` element whose `Body` is the rest — the
     `Let` case then carries the scoping; forward threading covers the shapes that stay flat.
   - `*ast.ForallExpr` (new case): `Lo`/`Hi` with current bound; `Body` with `Var` bound.
   - `*ast.Send` (new case): `Channel`, `Value`. `*ast.Recv` (new case): `Channel`.
   - `*ast.AssertStmt` (new case): `Condition`.
   - `*ast.QuasiQuote` (new case): each `Interpolation.Expr` — interpolations are evaluated
     expressions; the template string binds nothing.
3. **Regression tests** — see Testing Strategy.

### Implementation Plan

**Phase 1: Walker** (~4 hours)
- [ ] Add `astPatternBinders` with the classification above; unit-test the classification
      against `elaboratePattern`'s output (`corePatternBinders` of the elaborated pattern) for
      a table of patterns — the two must agree.
- [ ] Rework `findReferences` to thread `bound`; add the five missing `ast.Expr` cases.
- [ ] Keep the `DEBUG_STRICT` default-case panic intact and its test coverage (a new variant
      added without a case still fails loudly).
- [ ] `isSelfRecursive`/emitter check: a module-level bodyless `letrec x = \n. x(n)` still emits
      `core.LetRec` (self-edge survives); a local `letrec t = … in …` that shadows a module func
      `t` no longer contributes the false edge.

**Phase 2: Tests** (~6 hours)
- [ ] `internal/elaborate/scc_test.go` (new): unit tests over `BuildCallGraph` —
      shadow-in-pattern, shadow-in-lambda-param, shadow-in-let (the matrix rows 1/5/6 shapes)
      assert **no** `each→t` edge; non-shadow controls assert the edge stays.
- [ ] `isSelfRecursive` tests: `func t() = match x { t => t }`-shaped body (self-reference
      shadowed by a pattern) is NOT self-recursive; genuine `letrec`-shaped body still is.
- [ ] e2e repro matrix (see Success Criteria) through the real check pipeline, in the external
      test package that can import `internal/pipeline` (mirror
      `module_let_resolution_test.go`'s harness).
- [ ] Guard tests for rows 10/11 (must stay rejected) — these pin that the validator semantics
      are untouched.

**Phase 3: Sweep & docs** (~4 hours)
- [ ] Corpus sweep with the same harness as the #2074/#1958 checks: pre-fix vs post-fix status
      flips must be empty; the repro matrix rows are the positive control.
- [ ] `make verify-examples`; `make test-core`; changelog entry under Unreleased referencing
      #2078 (`refs #2078`; `Fixes #2078` only if this lands with code changes, which it will).

### Files to Modify/Create

**New files:**
- `internal/elaborate/scc_test.go` (~250 LOC) — first direct tests for `scc.go` (verified today:
  no test file exists for it)
- e2e regression test file (~120 LOC, placement per Deferred Decisions)

**Modified files:**
- `internal/elaborate/scc.go` (+90/−25 LOC) — `astPatternBinders`, scope-aware
  `findReferences`, five new `ast.Expr` cases

**Unchanged (with new guard tests):** `internal/types/typechecker_app_effects.go`,
`internal/pipeline/validate_effects.go`.

## Examples

**Before (rejected, #2078):** the issue repro as given above.

**After (accepted):** identical source, but the module elaborates as `LetRec{each}` + `Let{t}`
(verified today by the `u`-renamed control), `each`'s recursive self-call publishes `{...e}`,
`t`'s call site still publishes `{IO}` and correctly charges `t`, which declares `! {IO}`.

**Still rejected (unchanged, load-bearing):**

```ailang
func d[e](f: (int) -> () ! {e}) -> () ! {e} = { f(1); d(\x. println("boom")) }
```

`d`'s self-call site genuinely introduces a closed IO row; charging it to `d` is correct, and
this fix must not weaken it (matrix row 11, verified still failing today).

## Success Criteria

- [ ] Issue repro (r2b.ail shape) and matrix rows 1/5/6 pass `ailang check`
- [ ] Matrix rows 10/11 still fail with "Missing effects" (no soundness change)
- [ ] No `each→t`-class false edge from any binder form (unit test, all three forms)
- [ ] Genuine mutual recursion (row 9 shape) still resolves (edge preserved)
- [ ] `TestModuleLetResolutionMatrix` and `TestLexicalScope_BinderShadowsGlobals` unchanged & passing
- [ ] Corpus sweep: 0 status flips; positive control (the repro rows) flips all
- [ ] `make verify-examples`, `make test-core`, `make fmt`, `make lint` pass
- [ ] Changelog entry under Unreleased (`refs #2078`)
- [ ] Documentation: none required beyond changelog (no user-facing surface changes; the
      rejected error disappears)

## Conflict Surface

`internal/elaborate` is on the mandatory list; this change rewrites the traversal that feeds
module decl ordering.

### Syntactic positions touched

- `findReferences`'s switch over `*ast.Expr` (internal/elaborate/scc.go:181): extends the
  `Identifier`, `Lambda`, `FuncLit`, `Let`, `LetRec`, `Match`, `Block` cases with binder
  tracking; adds `ForallExpr`, `Send`, `Recv`, `AssertStmt`, `QuasiQuote` cases.
- New helper reads `ast.Pattern` positions: `Identifier`, `ConstructorPattern(+Patterns)`,
  `ConsPattern(Head, Tail)`, `ListPattern(Elements, Rest)`, `TuplePattern(Elements)`,
  `RecordPattern(Fields[].Pattern)`, `WildcardPattern`.
- Consumers: `BuildCallGraph` (edges) and `isSelfRecursive` (file.go:475); both consume only the
  returned `[]string`.

### What else lives here

Every *binder* form in the language now filters references; every *reference* position the old
walk already visited keeps its semantics:

| Position | Existing behavior | After change |
|----------|-------------------|--------------|
| `let t = t(1) in t` — value of the binder | reference **kept** (binder not in scope in its own value — scope matrix row `let_value_outside_scope`) | kept |
| `{ let a = tick(1); let tick = 2; a + tick }` — use before let | reference **kept** (earlier statement sees outer binding — scope matrix row `block_use_before_let`) | kept (forward-threaded binders) |
| `match 1 { tick => tick } + tick(1)` — arm closes | arm ref shadowed, outer ref kept (scope matrix row `scope_closes_after_arm`) | same, now also at graph level |
| Module-level bodyless `letrec x = \n. x(n)` (no `in` body) | self-reference **kept** — it is what makes the emitter produce `core.LetRec` (file.go:118) | kept (LetRec case splits on `Body == nil`) |
| Local `letrec t = v in b` shadowing a module func `t` | self/false refs **kept** (today's bug, `letrec` flavor) | filtered; binding resolves locally, no module edge |
| Uppercase pattern identifier (`None`) | treated as constructor, not a binder by `elaboratePattern` (isUpperIdent) | same classification, now shared by the graph |
| QuasiQuote template | never walked | still not; interpolations now walked |
| Contract `requires/ensures`, `tests` expressions | **not walked today** (only `f.Body` feeds the graph) | unchanged (out of scope, no repro) |

### Disambiguation strategy

The binder classification must not be a guess. `elaboratePattern`
(internal/elaborate/patterns.go:96) classifies a pattern-position `*ast.Identifier` as:
wildcard if `_`; nullary/bare constructor if the name is in `e.constructors` **or is
upper-case** (`isUpperIdent`, parser enforces UpperCamelCase at declaration — see the
`PAR_VARIANT_NEEDS_UIDENT` note in patterns.go); a `VarPattern` otherwise. The call graph runs
**before** elaboration (BuildCallGraph at file.go:267 gets `symbols`/`imports`, not the
constructor table), so `astPatternBinders` cannot consult `e.constructors`. It uses the
conservative syntactic projection: a binder is a **lower-case** identifier (≠ `_`).

Why this is exact and not heuristic:
- A **lower-case** pattern identifier is always a `VarPattern` (the two other classes —
  wildcard and constructor — require `_` or UpperCamelCase), so binding it can never drop a
  real reference.
- An **upper-case** pattern identifier is never a `VarPattern` — `elaboratePattern`
  (patterns.go:113–135) turns it into a `ConstructorPattern` (registered nullary ctor, or
  bare upper-case per the #323 rule). So in pattern position it is *neither a binder nor a
  reference to a module func*, even when a module func shares the name (verified: upper-case
  module funcs like `func Foo()` are legal — `PAR_VARIANT_NEEDS_UIDENT` constrains variant
  constructors, not func declarations). If that same name appears as a plain **expression**
  identifier (`Foo()` in a body), it is a genuine reference and the `*ast.Identifier` case —
  not the pattern classifier — handles it: not bound, edge kept.

The failure modes are asymmetric and were checked against the #327 history: dropping a genuine
edge re-introduces false "undefined variable" (the #327 family this walk is commented to
prevent); keeping a false edge is exactly today's #2078. The classification above can only do
the second, never the first.

### Programs that MUST still work

Regression fixtures (all exist and check clean on today's binary — verified 2026-11-03):

1. `examples/inline_tests_recursive.ail` — `sumList`/`listLength` recursive over
   `head :: tail` cons patterns; `tail` shadows an imported std name.
2. `std/list.ail` — `length`/`head`/`tail` over `x :: rest` cons patterns in a module with many
   sibling functions.
3. `examples/record_cons_pattern.ail` — record-literal head in a cons pattern.
4. `examples/first_non_repeat.ail` — cons patterns + multiple module funcs.
5. `internal/elaborate/module_let_resolution_test.go` — the #366 decl-ordering matrix (module
   let → func dependencies in both orders).
6. The `mutual.ail` shape (genuine mutual `even`/`odd` recursion, both `! {e}`) — the e2e
   matrix's row 9 becomes a checked-in test.

### What deliberately changes

Nothing user-visible is *removed*. Programs that were spuriously **rejected** are now
accepted. One behavior changes invisibly: modules with shadow-shaped names stop being forced
into one mutual-recursion group, which can change inferred generality (in the observed cases it
restores the author's declared polymorphism). No previously accepted program is rejected: the
walk only *removes* edges whose endpoints are provably not references, and Tarjan ordering
under fewer edges still emits every callee before its (genuine) callers — the #366 invariant.

## Testing Strategy

**Unit tests (`internal/elaborate/scc_test.go`, new):**
- `astPatternBinders` classification table (all pattern forms, incl. `_`, upper-case, nullary
  ctor shapes) cross-checked against `corePatternBinders(elaboratePattern(p))` for the same
  inputs — agreement by construction, tested anyway.
- `BuildCallGraph`: one case per binder form (pattern / lambda param / let) asserting the
  shadowed edge is absent and control edges are present.
- `isSelfRecursive`: shadowed self-reference → false; genuine self-reference → true;
  module-level bodyless `letrec x = \n. x(n)` → still **true** (emission guard).

**Integration/e2e (external test package, real pipeline):**
- The reproduction matrix as a table: rows 1–9 must pass, rows 10–11 must fail with "Missing
  effects" (guard against relaxing the validator).

**Regression-surface tests:** the six "Programs that MUST still work" fixtures, plus the full
`examples/` + `std/` sweep in Phase 3.

**Manual verification:** `ailang check` on the repro; `DEBUG_EFFECTS=1` trace shows the
recursive self-call publishing `{...e}` and the caller's site publishing `{IO}` (mirror of
today's passing control, which is the post-fix state).

## Non-Goals

- **Polymorphic recursion / per-site instantiation inside SCCs.** HM monomorphic groups are the
  design; row 10 shows group-mate instantiation must be charged. Not revisited here.
- **Validator semantics** (`validate_effects.go` per-site charging, the
  `!usedDeclared || calleeEffects.Tail != nil` override). Proven load-bearing by rows 10/11.
  A blanket "charge the declared row at self-calls" hardening was analyzed and **rejected**: it
  would accept row 11's genuinely-unsound `d`.
- **`ApplicationEffects` publication/substitution** (`typechecker_app_effects.go`). Contaminated
  only via the false SCC; untouched.
- **Walking `tests [...]` / contract expressions in the call graph.** Pre-existing gap, no
  repro, tracked as an observation only (Deferred Decisions).
- **The #1958 callback-slot soundness work** — this doc fixes its discovery vehicle, not it.

## Timeline

**Week 1** (~14 hours over 2 days):
- Phase 1: walker + binder classification (4h), Phase 2: unit + e2e tests (6h),
  Phase 3: sweep + changelog (4h)

**Total: ~14 hours, 2 days** (estimate doubled per convention → budget 2.5 days)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Dropping a genuine edge re-introduces #327 false "undefined variable" | High | Conservative classifier (lower-case only, the exact `elaboratePattern` projection); corpus sweep with 0-flip gate; module_let_resolution matrix as pinned ordering test |
| Binder threading gets block/`let`-desugar shapes wrong (use-before-let) | Medium | Reuse the `TestLexicalScope_BinderShadowsGlobals` rows as the semantic spec; port the same cases as graph-level unit tests |
| `isSelfRecursive` flip changes Let vs LetRec emission for shadowed-self functions | Low | Emission stays valid either way (a `let` binding a non-recursive lambda); e2e matrix covers it |
| Hidden dependency of the emitter on SCC *size* (singleton vs multi) | Medium | The `u`-control already exercises singleton-then-let through the real pipeline and passes today; sweep gate confirms no other module regrouping flips |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No nondeterminism added or removed; graph construction was and stays deterministic |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | +1 | The effect validator stops reporting a phantom IO on a polymorphic function and stops suggesting a signature that would hide real polymorphism |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Declared-row-vs-body validation becomes correct exactly at the definition boundary it describes; group-mate instantiations no longer leak into it |
| A6: Safe Concurrency | 0 | Unrelated (the `Send`/`Recv` cases only keep the walk exhaustive) |
| A7: Machines First | +1 | AI-generated code that shadows names (`t`, `x`, `acc`) gets correct, deterministic checking instead of a misdirected diagnostic |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No resource-surface changes |
| A10: Composability | +1 | Row-polymorphic helpers become usable from any sibling again, restoring the composition the declaration promises |
| A11: Structured Failure | +1 | The failure that disappears was attributed to the wrong function with a harmful suggested fix |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: proceed**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects; one *spurious* side-effect accusation removed
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Verification Log

Every load-bearing claim above, with its check (all run 2026-11-03 against the v0.57.0 binary,
commit `b23cadc`, built from this workspace):

| # | Claim | Check | Result |
|---|-------|-------|--------|
| 1 | Issue repro fails exactly as reported | `ailang check r2b.ail` (issue source verbatim) | FAIL: "Effect checking failed for function 'each' … Missing effects: IO … Suggested fix: func each(...) -> T ! {IO, e}" |
| 2 | `each` alone is clean | same source minus the sibling, `ailang check` | PASS |
| 3 | Sibling name (not its content) is the switch | rename sibling `t`→`u`, keep pattern var `t` | PASS; `DEBUG_EFFECTS=1` shows `LetRec binding: each` singleton + `Let binding: u` |
| 4 | The failing module is `LetRec{t, each}` | `DEBUG_EFFECTS=1` on the repro | `=== Validating LetRec binding: t ===` then `=== Validating LetRec binding: each ===` |
| 4b | A non-calling sibling `t` does NOT group with `each` | `DEBUG_EFFECTS=1` on `t() = println("direct")` variant | `=== Validating Let binding: t ===` + `=== Validating LetRec binding: each ===` — one-way false edge, no cycle |
| 5 | The recursive self-call publishes `{IO}` in the failing case and `{...e}` in the control | `DEBUG_EFFECTS=1`, grep `Published App … call row` | failing: `App 17 {IO}`, `App 20 {...e}`; control: `App 6 {...e}`, `App 9 {...e}` |
| 6 | Lambda-param shadow reproduces | `each(\t. t, r)` variant | FAIL, same mischarge |
| 7 | Let-binder shadow reproduces | `let t = r in each(f, t)` variant | FAIL, same mischarge |
| 8 | Pure-callback sibling passes (open-row aliasing) | `each(\x. (), [1])` sibling + second IO caller `w` | PASS (both callers) — no silent polymorphism loss either |
| 9 | Genuine mutual recursion passes | `even`/`odd` both `! {e}` | PASS |
| 10 | Group-mate closed-row callback is *correctly* rejected | `odd` calls `even(\x. println("concrete"), 0)` | FAIL on `odd` — genuine: its body performs IO |
| 11 | Per-site charging at self-calls is load-bearing | `d[e]` self-call with concrete IO lambda | FAIL on `d` — genuine; blanket validator hardening would wrongly accept this |
| 12 | `findReferences` never looks at case patterns | read internal/elaborate/scc.go:181–296, `case *ast.Match` | confirmed: walks `ex.Expr`, `c.Guard`, `c.Body`; no pattern walk, no bound set |
| 13 | `ForallExpr`/`Send`/`Recv`/`AssertStmt`/`QuasiQuote` unhandled in the walk | `grep ForallExpr\|Send\|Recv\|AssertStmt\|QuasiQuote internal/elaborate/scc.go` | empty — they fall to the silent `default` (DEBUG_STRICT only panics) |
| 14 | No test file exists for `scc.go` / no direct `BuildCallGraph` coverage | `ls internal/elaborate/scc*` → `scc.go`; grep tests for `BuildCallGraph\|findReferences` | only a comment mention in scope_test.go — negative-existence confirmed |
| 15 | The validator override charges the site row for tailed callees | read internal/pipeline/validate_effects.go App case | confirmed: `if collector.maskLookup != nil && (!usedDeclared \|\| calleeEffects != nil && calleeEffects.Tail != nil) { calleeEffects = cloneEffectRow(publication.CallRow) … }` |
| 16 | The elaborator's *scope* resolution already handles shadowing correctly | read internal/elaborate/scope_test.go `TestLexicalScope_BinderShadowsGlobals` | confirmed: 26-row matrix incl. `match_binder`, `list_tail_binder`, `lambda_param`, `let_body`, `block_use_before_let` |
| 17 | Decl-ordering machinery exists and is tested | `ls internal/elaborate/module_let_resolution_test.go`, `grep TestModuleLetResolutionMatrix` | exists; e2e via pipeline.Run ModeCheck |
| 18 | Fixtures exist and pass today | `ailang check` on the six MUST-work fixtures | all PASS |
| 19 | `elaboratePattern` classification is `_`/constructor(upper or registered)/lower-case-binder | read internal/elaborate/patterns.go:96–135 | confirmed |
| 21 | Upper-case module funcs are legal (the classifier must not assume otherwise) | `ailang check` on `func Foo() -> int = 1; export func main() -> int = Foo()` | PASS — classifier reasoning above does not rely on func naming case |
| 20 | "Not a regression; found verifying #1958" | issue text + changelogs/v0.32-current.md #1958/#2074 sections | consistent (v0.56.1 same behavior per issue; verified #1958 sections exist) |

## Related Documents

- [m-effect-row-open-closed-reland-sprint-plan.md](../v0_57_0/m-effect-row-open-closed-reland-sprint-plan.md)
  (implemented, v0.57.0) — the #1958 re-land whose verification found this.
- [m-module-let-func-resolution.md](../../implemented/v0_30_0/m-module-let-func-resolution.md)
  (implemented, v0.30.0) — the unified func+let SCC this walk feeds; its ordering invariant is
  the one this fix must not break.
- `internal/elaborate/scc.go` header comments — the #327 exhaustive-position doctrine this fix
  extends to scope.

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Issue #2078 — the repro, the v0.57.0/v0.56.1 verification, the scope estimate (0/20,046 sweep
  files), and the original "likely area" pointer at the validator; this doc's traced root cause
  sits one layer below that pointer (the call graph that feeds the grouping).
- `changelogs/v0.32-current.md` — #1958, #2074, and the pre-release sweep notes.

## Future Work

- Walk contract `requires`/`ensures` and `tests [...]` expressions in `BuildCallGraph` if a
  repro appears (observed adjacent gap, deliberately not fixed here).
- A "shadow audit" lint/diagnostic when a local binder shadows a module-level name — the
  language allows it, but it surprised a careful reader here.

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03