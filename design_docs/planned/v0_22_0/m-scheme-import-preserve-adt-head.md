# M-SCHEME-IMPORT-PRESERVE-ADT-HEAD — Imported function schemes lose ADT head constructors

**Status**: Planned — P0
**Target**: v0.22.0 or v0.23.0 (depends on depth of fix)
**Priority**: P0 — root cause of M-MATCH-ADT-XCHECK regression and likely many silent type bugs
**Estimated**: 2-4 days (investigation-heavy; possible interface-format change)
**Dependencies**: None
**Source**: Investigation 2026-05-20 while attempting to fix [m-match-adt-xcheck-regression.md](m-match-adt-xcheck-regression.md). Spawning doc.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A5: Bounded Verification | **+2** | Core invariant of HM type inference — function signatures determine call-site behavior. If schemes are wrong, every downstream check fails silently |
| A7: Machines First | **+2** | Agents rely on type signatures being honest. A function declared `Option[float]` that types as `forall a. a` is a foundational violation of trust |
| A11: Structured Failure | **+2** | Silent corruption of cross-module type information is the worst-case failure mode |
| A12: System Boundary | **+1** | Module interface boundary is precisely the place where contracts should be machine-verifiable |

**Net Score: +7** → **Decision: Proceed urgently**

## Problem statement

When `std/json.getNumber` is declared as:

```ailang
export func getNumber(obj: Json, key: string) -> Option[float] { ... }
```

…and imported into another module, the imported scheme in the type env is:

```
scheme.TypeVars = [α134]
scheme.Type     = (Json, string) -> α134
```

That is: `forall α134. (Json, string) -> α134`. The concrete `Option[float]` return is replaced with an unconstrained type variable. Same pattern observed for `std/json.decode`: declared `string -> Result[Json, string]`, imported as `forall α4. string -> α4 ! ε5`.

This means:
- Every call to `getNumber(obj, key)` produces a result with type *fresh TVar*, with no constraint linking it back to `Option[float]`.
- Pattern matches on the result can use ANY constructor (Ok, Err, Some, None, custom ADTs) and the typechecker has nothing to compare against — no error is produced.
- At runtime, the value actually IS an Option, so any non-Option arm misses and the runtime panics with `no pattern matched in match expression`.

This is the **actual root cause** of the symptom reported in [m-match-adt-xcheck-regression.md](m-match-adt-xcheck-regression.md). The MATCH-XCHECK code in `internal/types/typechecker_patterns.go` is sound; it just has no usable information to check against.

## Evidence

Live trace from `/tmp/repro_step1.ail` with debug printfs added to `inferVar` and `SolveConstraints`:

```
[XCHECK-VAR] decode: scheme.TypeVars=[α4] scheme.Type=string -> α4 ! {...ε5}
[XCHECK-VAR] getNumber: scheme.TypeVars=[α134] scheme.Type=(Json, string) -> α134
```

For comparison, locally-defined functions in the same module (e.g. `myGetNum` defined inline as `func myGetNum(x: int) -> Option[float]`) show the same broken pattern. Imported constructors like `Some`, `Ok` come through correctly with `forall a. a -> Option[a]` and `forall a, e. a -> Result[a, e]` — so this is specifically about **function return types**, not all imports.

A clean test using a concrete-typed local match scrutinee:

```ailang
let o: Option[int] = None in
match o { Ok(_) => ..., Err(_) => ... }
```

DOES correctly produce a `MatchForeignConstructorError`. So the pattern-check layer is fine; the issue is upstream.

## Hypothesis

Most likely: when building a function's exported scheme, the body's return type is being subject to **over-generalization**. The function returns `Option[float]`, but at the export boundary:

1. `float` is a `Num`-class type variable (because numeric literals default to `Num a => a`).
2. Generalization sees `α` as free and quantifies it: `forall α. Option[α]`.
3. **Somewhere between generalization and storage, the Option wrapper is lost** and the scheme degrades to `forall α. α`.

Alternative: the iface JSON serialization round-trip is dropping the head constructor for parameterized types when the parameter is a quantified variable.

## Investigation plan (M1)

1. **Confirm scheme construction site**: add tracing to `iface.BuildInterfaceWithTypesAndConstructors` to log the scheme being stored for each export, immediately after construction.
2. **Confirm scheme serialization round-trip**: if ifaces are persisted, dump the on-disk representation for `std/json.getNumber` and check whether the head constructor is present there.
3. **Determine which step drops the head**:
   - If the scheme is correct at construction time but wrong at lookup → serialization bug.
   - If the scheme is already wrong at construction → generalization bug.
4. **Compare with constructor schemes** which DO survive (`Some: forall a. a -> Option[a]`). Identify what the constructor-export path does differently.

## Fix sketch (M2 — pending M1 outcome)

Two likely fix locations:

- **If serialization**: fix the iface format / unmarshal to preserve TApp head when the args contain quantified vars.
- **If generalization**: ensure that when a return type is `TApp{Con, Args}` and only the args contain free vars, the head Con is preserved and ONLY the args are generalized.

## Acceptance criteria

- [ ] A new test in `internal/iface/` or `internal/pipeline/` that imports `std/json.getNumber` and asserts:
  - `scheme.Type` is a `TFunc2` whose `Return` is a `TApp` with `Constructor = TCon{Option}` and `Args = [TVar{...float-related...}]` OR `Args = [TCon{float}]`.
  - The scheme does NOT have an unconstrained return-type variable.
- [ ] `/tmp/foreign_ctor_repro.ail` (the M-MATCH-ADT-XCHECK reproducer) now fails `ailang check` with the structured `MatchForeignConstructorError`.
- [ ] Every existing test in `internal/iface/`, `internal/pipeline/`, and `internal/types/` continues to pass.
- [ ] CHANGELOG entry under v0.22.0 (or v0.23.0) documenting that imported function schemes now preserve concrete ADT heads.

## Risks

- **Performance**: if generalization currently produces `forall α. α` shortcuts for speed, the fix may slow down compilation by adding ADT-aware generalization. Mitigation: profile after the fix, look for hot paths.
- **Cascading test failures**: other tests may have been silently passing only because schemes were overly polymorphic. Each one must be investigated, not blindly fixed.
- **Backwards compatibility of cached ifaces**: if ifaces are cached on disk, the fix should bump the cache key / version to force re-export of all module ifaces.

## Out of scope

- Pattern-check improvements (handled by [m-match-adt-xcheck-regression.md](m-match-adt-xcheck-regression.md), which is unblocked by this fix).
- Constructor scheme handling (already correct).
- Effect-row preservation in imported schemes (separate concern — though may share infrastructure).

## Related

- [m-match-adt-xcheck-regression.md](m-match-adt-xcheck-regression.md) — symptom-level doc; this is the underlying cause.
- [M-MATCH-ADT-XCHECK (v0.18.10)](../implemented/v0_18_10/m-match-adt-xcheck.md) — the original pattern-check work, sound but rendered ineffective by this bug.
- msg_20260520_111521_44c38751 — bug report from cognitive_commons that started the investigation.
