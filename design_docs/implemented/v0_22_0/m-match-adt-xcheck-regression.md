# M-MATCH-ADT-XCHECK-REGRESSION — Foreign-constructor patterns silently accepted again

**Status**: IMPLEMENTED via [m-scheme-import-preserve-adt-head.md](m-scheme-import-preserve-adt-head.md) (2026-05-20, v0.22.0). Root cause was upstream scheme corruption, not a pattern-check gap. Fixing the schemes restored the existing M-MATCH-ADT-XCHECK fast-path's ability to fire for function-call scrutinees. New regression tests live in `internal/pipeline/match_foreign_constructor_function_call_test.go`.
**Target**: v0.22.0 (hotfix candidate)
**Priority**: P0 — silent runtime panics that pass `ailang check` + CI. Same bug class M-MATCH-ADT-XCHECK was supposed to close. Active production exposure (cognitive_commons, motoko_ext_*).
**Estimated**: ~1-2 days (root-cause + tightening the existing check; ~200 LOC + tests)
**Dependencies**: None — this is repair work on [M-MATCH-ADT-XCHECK (v0.18.10)](../implemented/v0_18_10/m-match-adt-xcheck.md)
**Source**: sunholo-demos/cognitive_commons (msg_20260520_111521_44c38751); reproduced 2026-05-20 on v0.20.1-41-g9dad56dc

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A5: Bounded Verification | **+2** | This is the canonical example of bounded verification — a static check that catches a class of runtime panic at compile time |
| A7: Machines First | **+2** | Agents spent days debugging "no pattern matched" when the actual bug was a constructor swap. The check exists to communicate this to agents. Failing silently is the worst-case outcome |
| A11: Structured Failure | **+2** | Restores a structured compile-time error that production has been silently relying on |
| A12: System Boundary | 0 | Internal check |

**Net Score: +6** → **Decision: Proceed urgently**

## Problem statement

The M-MATCH-ADT-XCHECK check (shipped v0.18.10) was supposed to permanently close this bug class:

> AILANG's match-arm typechecker allows any constructor pattern in any position regardless of whether that constructor belongs to the scrutinee's ADT.

It does not. The check still misses cases where the scrutinee is a function call whose return type is a concrete ADT.

### Reproducer (passes `ailang check` clean, but is a guaranteed runtime panic)

```ailang
module test/repro

import std/json (decode, getNumber, Json)
import std/result (Ok, Err)

export func main() -> float {
  let obj = decode("{\"x\": 3.14}") in
  match obj {
    Ok(j) => match getNumber(j, "x") {  -- returns Option[float]
      Ok(v)  => v,                      -- WRONG: Ok is from Result
      Err(_) => 0.0                     -- WRONG: Err is from Result
    },
    Err(_) => 0.0
  }
}
```

`getNumber` has a fully concrete signature `(Json, string) -> Option[float]` (std/json.ail:183). There is no polymorphism in the scrutinee. The xcheck *should* fire on the inner match.

```
$ ailang check /tmp/repro.ail
✓ No errors found!
```

At runtime, the inner match panics with `no pattern matched in match expression` when `getNumber` returns `Some(3.14)` — there's no `Some` arm and no fallthrough.

### Recurrence cost

- **arniwesth/motoko_agent#16** (May 2026): first surfaced — match `Result` arms against `Option` value, crash on first turn.
- **sunholo-demos/cognitive_commons** (May 2026): same bug across 3 modules (persuasion, citizen, commons_browser), passed `ailang check` + CI for an entire iteration. Only surfaced when JS called the WASM-exported function.
- The user's comment: *"This bit us hard porting cognitive_commons"* and *"Likely many similar latent bugs in extensions and user code, masked because the value happens to fall through to a working arm"*.

This is the same wording from the original M-MATCH-ADT-XCHECK problem statement. We are looking at the bug we already shipped a fix for.

## ⚠ ROOT CAUSE UPDATED 2026-05-20 — NOT a pattern-check bug, it's a SCHEME IMPORT bug

After live tracing with debug printfs in `inferVar`, `inferApp`, `SolveConstraints`, and `extractADTName`, the actual root cause is **scheme corruption on import**, not a pattern-check gap.

### Evidence (debug trace from /tmp/repro_step1.ail)

```
[XCHECK-VAR] decode: scheme.TypeVars=[α4] scheme.Type=string -> α4 ! {...ε5}
[XCHECK-VAR] getNumber: scheme.TypeVars=[α134] scheme.Type=(Json, string) -> α134
```

`getNumber`'s declared signature is `(Json, string) -> Option[float]`. After import, its scheme is stored as `forall α134. (Json, string) -> α134` — the **Option[float] head is completely lost** and replaced with a single unconstrained type variable. Same for `decode` (declared `string -> Result[Json, string]`, imported as `forall α4. string -> α4 ! ε5`).

### Constraint dump

```
[XCHECK-SOLVE] 6 constraints
  [0] TypeEq: (Json, string) -> α1 ~ (Json, string) -> α2 ! {...ε3}   ← application
  [1] TypeEq: α2 ~ Result[α4, α5]                                       ← Ok pattern
  [2] TypeEq: α2 ~ Result[α7, α8]                                       ← Err pattern
  [3] Class: Fractional[α10]
  [4] TypeEq: α10 ~ α6                                                  ← match arm
  [5] TypeEq: α6 ~ float                                                ← return type
```

After solve, `sub[α2] = Result[α4, α5]` (bound ONLY by the foreign pattern, with no competing constraint from the function call). Unification succeeds because there is no `α2 ~ Option[float]` constraint to conflict with it. The scrutinee silently becomes `Result[...]` — exactly the opposite of what the user intended.

### Why M-MATCH-ADT-XCHECK (v0.18.10) still works for some cases

The fast-path xcheck fires when `extractADTName(scrutType) != ""`. That only happens when the scrutinee is bound to a concrete TApp at pattern-check time — which requires either:
- A let-binding with an explicit type annotation (`let x: Option[int] = ...`)
- A constructor literal scrutinee (`match Some(42) { ... }`)

For function-call scrutinees, the scrutinee is a fresh TVar. The constraint-based fallback would normally catch the conflict — *if* the function's return-type constraint were added. But because the imported scheme has no concrete return type, no such constraint exists.

### Locus of the bug

Almost certainly in **module interface serialization / scheme construction** when exporting from `std/json` (or wherever `Option[float]` returns get flattened). The candidates:

1. `iface.BuildInterfaceWithTypesAndConstructors` (called per module) — converts the inferred type into an exportable Scheme. If `Option[float]` is being seen as polymorphic (because `float` is unconstrained via Num defaulting?), it might be over-generalized.
2. `iface` deserialization — if the iface is round-tripped through JSON/disk, a non-TApp wrap step might be dropping the head constructor.
3. `Scheme.Instantiate` or `generalize` — if generalization over-quantifies, capturing the WHOLE return type as a free var.

Hypothesis #1 is most likely: the function body of `getNumber` returns `Option[float]` but the typechecker treats `float` as a Num-class variable and generalizes the return as `forall a: Num. Option[a]` — but then somewhere the Option head is lost and only `a` survives.

## Original Root cause hypothesis (PRE-INVESTIGATION, partially incorrect)

The existing check in [internal/types/typechecker_patterns.go:146-200](../../internal/types/typechecker_patterns.go) has two paths:

1. **Fast-path (lines 166-174)**: when `extractADTName(scrutType) != ""` and disagrees with the constructor's ADT, raise `MatchForeignConstructorError` directly.
2. **Constraint path (lines 175-198)**: when the scrutinee is unresolved or matches, add a unification constraint linking the scrutinee type to the constructor's ADT.

The comment claims (lines 161-165):
> Polymorphic scrutinees (fresh TVars) fall through to the constraint-based path as before — extractADTName returns "" for unresolved types, so the check is conservative and doesn't fire on legitimately polymorphic code.

For the reproducer, the scrutinee `getNumber(j, "x")` should have its return type concretely resolved before the patterns are visited, because `inferMatch` ([typechecker_patterns.go:13](../../internal/types/typechecker_patterns.go#L13)) calls `tc.inferCore(ctx, match.Scrutinee)` first. So we'd expect `getType(scrutineeNode)` to be `Option[float]` (concrete TApp). The fast-path should fire.

**Candidate causes** (M1 investigation):

1. **`getType(scrutineeNode)` returns a TVar, not Option[float]** — `inferCore` for a function call returns the unsolved return-type variable. The substitution that would resolve it to `Option[float]` is added as a constraint, but not yet applied to the type stored on the node. `extractADTName(TVar) == ""`, so the fast-path falls through.

2. **`Ok`/`Err` not in `constructorTypes` for this module's typechecker** — if `Ok` isn't found at line 150, the entire ADT-resolution block (including the fast-path AND the constraint addition) is skipped. The pattern is then type-checked structurally with fresh TVars (lines 202+) and never linked back to `Result`. No constraint, no conflict, no error.

3. **The constraint IS added but the unifier doesn't conflict** — `scrutType ~ Result[a, e]` and `scrutType ~ Option[float]` are both added, but a bug in unification reconciles them. Less likely; would show up on many tests.

The most likely cause is **#1**: the fast-path is gated on the scrutinee's type being *already concrete at pattern-check time*, but for function-call scrutinees that's after-the-fact knowledge — it requires applying the current substitution to the scrutinee node before extracting the ADT name. This is what the existing tests miss: they use scrutinees that are immediately-bound let values (`let x: Option[int] = ... in match x { ... }`), not function calls.

Cause **#2** is also plausible and would be more pernicious: it would mean ANY foreign constructor on ANY scrutinee passes silently, and the existing tests pass only because they happen to use scrutinees whose type is locked down before pattern check. Worth checking with a let-bound scrutinee: `let x: Option[int] = Some(0) in match x { Ok(_) => ... }` — does the existing v0.18.10 test still catch this today?

## Goals

1. The reproducer above produces a `MatchForeignConstructorError` at compile time, naming `Ok` and `Err` as foreign to `Option`.
2. The same error fires when the scrutinee is *any* function call whose return type is concretely a different ADT — not just immediately-bound values.
3. The existing M-MATCH-ADT-XCHECK regression tests in [internal/pipeline/match_foreign_constructor_test.go](../../internal/pipeline/match_foreign_constructor_test.go) all still pass.
4. A new test suite covers function-call scrutinees, nested matches, and cross-module ADTs to prevent this regression class from re-appearing.

## Revised acceptance criteria (post-investigation)

The original M-MATCH-ADT-XCHECK regression CANNOT be closed without first fixing the scheme-corruption bug. A pattern-check-level fix (post-solve walk, applySubst, etc.) cannot work because *no constraint* connects the scrutinee to its true ADT — the imported scheme has lost the head constructor.

**Recommended sprint pivot**:
1. **Close this design doc as "investigation complete, deferred"** — the reported symptom is real but the fix lives in a different layer.
2. **Open a follow-up M-SCHEME-IMPORT-PRESERVE-ADT-HEAD design doc** (or similar) covering the actual repair: ensure that imported function schemes preserve the concrete ADT head in their return type. Add a regression test for `getNumber: (Json, string) -> Option[float]` that asserts `tc.globalTypes["std/json.getNumber"].Type` reads back as a `TApp{Constructor: Option, Args:[float]}` NOT a TVar.
3. Once that lands, the existing M-MATCH-ADT-XCHECK fast-path will fire correctly for function-call scrutinees with no further changes — the constraint conflict will be detected at solve time.

## Original Proposed fix (OBSOLETED — see ROOT CAUSE UPDATED above)

Apply the current substitution to the scrutinee type before invoking the fast-path xcheck:

```go
// Before:
if scrutADT := extractADTName(scrutType); scrutADT != "" && scrutADT != adtTypeName {

// After:
resolved := ctx.applySubst(scrutType)  // or whatever the in-flight substitution accessor is
if scrutADT := extractADTName(resolved); scrutADT != "" && scrutADT != adtTypeName {
```

If the in-flight constraints haven't been solved yet (constraints are accumulated and solved at the end of inference), the substitution may still be incomplete. In that case, a stronger approach:

**Move the xcheck to a post-solve pass.** After `solveConstraints`, walk the typed AST, find every `TypedMatch`, look at the scrutinee's *final* type, and re-check each arm's constructor against it. This is `O(n)` over match arms and guaranteed correct because all unification has completed.

Trade-off: post-solve passes mean the error appears later, possibly behind a less-helpful generic unification error if one fires first. Mitigation: in the constraint-path branch (lines 175+), suppress generic `cannot unify Result[a,e] with Option[float]` errors when both sides are TApps over different constructorTypes entries — defer those to the post-solve xcheck pass which produces a much better message.

## Acceptance criteria

- [ ] Reproducer at the top of this doc fails `ailang check` with a structured `MatchForeignConstructorError` naming `Ok` and the available `Option` constructors.
- [ ] A new test in `internal/pipeline/match_foreign_constructor_test.go` (or sibling file) covers the function-call-scrutinee case for at least: (a) std/json `getNumber` returning Option, (b) cross-module ADT imported alongside foreign-ADT constructors, (c) nested match where the inner scrutinee is a function call.
- [ ] All existing M-MATCH-ADT-XCHECK tests continue to pass.
- [ ] Bug message msg_20260520_111521_44c38751 acknowledged with a verified fix on cognitive_commons.

## M1 investigation tasks (sequence)

1. Add a debug printf in [typechecker_patterns.go:166](../../internal/types/typechecker_patterns.go#L166) capturing `scrutType` and `extractADTName(scrutType)` for the reproducer. Confirm hypothesis #1 vs #2.
2. If hypothesis #1: implement the `applySubst` variant. Check whether it's enough or whether constraints are still unsolved at pattern-check time.
3. If unsolved: implement the post-solve pass.
4. If hypothesis #2: trace why `Ok` isn't in `constructorTypes` despite the `import std/result (Ok, Err)`. Likely a gap between elaborator's `RegisterConstructor` calls and the pipeline's `ImportedCtorTypes` map population for explicitly-imported (not just brought-in-by-type-import) constructors.

## Out of scope

- Exhaustiveness checking (separate concern — M-MATCH-EXHAUSTIVE would cover dead arms and missing arms; this doc only covers foreign-ADT constructors).
- Pattern matching for refinement types, GADTs, or anything beyond plain tagged unions.
- Changing the runtime panic message — fixing the static check makes the panic unreachable.

## Risks

- **False positives**: making the check more aggressive could reject legitimate polymorphic code. Mitigation: the post-solve pass is the safest place — by the time it runs, all unification has settled, so "this constructor is foreign to this ADT" is a fact, not a guess.
- **Error ordering**: users may see a confusing generic unification error before the nice xcheck error. Mitigation: in the constraint path, suppress generic mismatch errors when both sides are recognizably ADTs that disagree; let the post-solve pass produce the message.
- **Test coverage cliff**: this regression existed because the existing test suite only covered let-bound and locally-typed scrutinees. The new tests must explicitly target function-call and import-flavored scrutinees, otherwise the bug returns again.

## Related

- [M-MATCH-ADT-XCHECK (v0.18.10)](../implemented/v0_18_10/m-match-adt-xcheck.md) — original implementation this doc repairs.
- msg_20260520_111521_44c38751 — bug report from cognitive_commons.
- arniwesth/motoko_agent#16 — original 2026-05-11 incident M-MATCH-ADT-XCHECK was created to fix.
