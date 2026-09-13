# ADT Equality Semantics — Position Independence and Instance Resolution

**Status**: Planned
**Target**: v0.39.0
**Priority**: P1 (High)
**Estimated**: 3 days
**Dependencies**: None (builds on M-DX19 derived-Eq machinery)
**Classification**: Type-system / eval semantics change (internal/types, internal/eval, internal/link)

## Problem Statement

Issue #713 reported two coupled defects; **both re-verified at HEAD (v0.38.5)** on 2026-09-13:

### Defect 1: ADT equality is position-dependent

For an ADT **without** `deriving (Eq)`, the *same construct* is rejected or accepted depending only on where it appears:

```ail
type Color = Red | Green | Blue

func f(c: Color) -> bool ! {}
{
  (c == Red) || (c == Green)     -- REJECTED: "No instance for Eq[Color] in scope ..."
}

func g(c: Color) -> Color ! {}
ensures { result == Red }        -- ACCEPTED: type-checks clean
{
  c
}
```

Verified: the body form fails with `No instance for Eq[Color] in scope. Equality (==, !=) needs an Eq instance; Color has none. Import std/prelude, or derive/define one.` The `ensures { result == Red }` form passes `ailang check` with no errors. Whitespace of context must not change semantics; today it does.

### Defect 2: The error's own suggested fix does not parse

The Eq hint (`internal/types/instances.go:91`, `actionableInstanceHint`) tells the agent: *"Import std/prelude, or derive/define one."*

Verified: `import std/prelude` fails with **IMP012_UNSUPPORTED_NAMESPACE** at the import site ("namespace imports not yet supported" — only selective `import std/x (sym)` or aliased forms parse), **and** `std/prelude.ail` does not exist as a module. The hint recommends (a) a syntax the parser rejects and (b) a module that is absent. An agent following the diagnostic verbatim lands in a second, unrelated error — the classic "hint says X, X is impossible" residual-failure class.

**Impact:** AI agents get an unrecoverable error in the common case (comparing enum/ADT values without remembering `deriving (Eq)`), and the recovery path in the message is a dead end. Both are agent-time and eval-pass-rate costs.

## Current State (mechanism, verified at HEAD)

**Type checking (general expressions).** `internal/types/typechecker_operators.go` (`case "==", "!="`) adds an `Eq` `ClassConstraint` on both operand types. Constraint solving (`internal/types/inference_helpers.go`, `InstanceEnv.Lookup` in `internal/types/instances.go`) fails for a user ADT unless: (a) the type declares `deriving (Eq)` — `AddDerivedEqForADT` registers `Eq[TypeName]` (M-DX19, invoked from `internal/elaborate/file_funcs.go`), (b) an `Ord` instance exists (superclass provision), or (c) a user-defined instance exists. This is correct per se — the rejection is the *dictionary-passing* discipline working as designed.

**Contract expressions (the divergent position).** `requires`/`ensures` blocks are parsed (`internal/parser/parser_contracts.go`) and elaborated to core properties (`internal/elaborate/file_funcs.go`), but **never run through type inference** — `inferCore` sees function bodies, not contract predicates. No `Eq` constraint is ever added for `result == Red`, so the type checker is silent. At eval time `applyBinOp` (`internal/eval/eval_operations.go:487`) falls back to `valuesStructurallyEqual` for `==`/`!=` on composite/ADT values, so the ensures check *runs* with structural semantics. Result: the two positions check with two different regimes — static dictionary discipline vs. runtime structural fallback.

**Derived-Eq dictionary machinery (shared with #963).** `internal/eval/eval_patterns.go` (`makeADTEqualityFn`, `taggedValuesEqual`, and the M-DX19 determinism fix that *synthesizes* structural Eq when the registry lookup misses) already implements the intended semantics: ADT equality is structural over constructors and fields. The runtime fallback and the dictionary method are the same comparison; only the type-checker's ruling diverges by position.

**Instance scope.** `LoadBuiltinInstances()` (`internal/types/instances.go`) is loaded unconditionally by the pipeline (`internal/pipeline/pipeline_module_phases.go:97`), so `Eq[int]/[float]/[string]/…` are always in scope; "import std/prelude" has no role to play for Eq at all — the hint is stale cargo from a pre-builtin-instance era.

## Proposed Design

### Ruling: what ADT equality SHOULD mean

**Every ADT (enum or data constructor type) gets a compiler-provided structural `Eq` instance automatically.** Rationale:

- ADTs are closed, nominal, fully-known shapes; structural equality is total, decidable, and deterministic — exactly the AILANG substrate properties. There is no lawful alternative meaning an ADT could have.
- The runtime already *implements* this semantics in two places (dictionary synthesis in `eval_patterns.go`, `applyBinOp` structural fallback in `eval_operations.go`). The proposal removes the discrepancy, not the semantics.
- This makes `deriving (Eq)` redundant-but-accepted (backwards compatible; becomes an explicit no-op affirmation, still useful documentation and still required for the *polymorphic* case below).

**Instance sources, in resolution order:**
1. User-defined `instance Eq[T]` (highest precedence; must keep coherence checking in `InstanceEnv.Add`).
2. `deriving (Eq)` declaration (explicit; unchanged).
3. **New: compiler-provided structural Eq for every monomorphic ADT**, registered alongside `AddDerivedEqForADT` (same key scheme, same `DerivedADTEquality` marker, same `makeADTEqualityFn` implementation). Ordering rule: user/derived declarations win; the compiler-provided instance is a default, never overlapping.
4. `Ord` superclass provision (unchanged).

**Scope limit:** polymorphic ADTs (`type T[a] = ...`) without field Eq constraints keep today's behavior — elaboration rejects `deriving (Eq)` for unconstrained polymorphic types (`file_funcs.go:162`), and the compiler-provided default applies **only to monomorphic ADTs** where the structural comparison is total. This preserves soundness without inventing qualified-instance machinery (deferred, matches existing v0.7+ note).

### Goal: position independence

The invariant to enforce (and to regression-test): **an expression's type-check result is a function of the expression and its bindings, not of which syntactic position it appears in.** Two consequences:

1. With the compiler-provided structural Eq above, `c == Red` type-checks identically in a body and in an `ensures` predicate — the general position stops being the strict one.
2. **Contract predicates must be type-checked** (the missing half of the fix): the pipeline should run `inferCore` over `requires`/`ensures`/invariant predicate expressions with `result` bound to the declared return type, so the *strict* regime applies in the contract position too. This closes the reverse hazard — today an ill-typed or Eq-less comparison inside `ensures` silently evaluates with structural fallback, which can mask contract bugs and, worse, lets ADT equality succeed in one position and fail in another for types where a user Eq instance *was* intended to differ from structural equality. (Contract predicate type-checking is included in scope here because the position-independence test cannot pass without both halves; it is the same semantics change, not a separate feature.)

### Sub-item: fix the IMP012 dead-end hint

The `actionableInstanceHint` Eq/Ord/Num strings must only suggest actions that are executable at HEAD. Options, in preference order:

1. **Preferred:** make the hints position-appropriate. Once structural Eq is compiler-provided, the Eq hint becomes obsolete for ADTs — replace the "Import std/prelude" clause with the real fix at that point (`add deriving (Eq)` for polymorphic ADTs, or "equality on ADTs is structural and automatic"). For the residual missing-instance classes where no auto-fix exists, reference `ailang check` documentation, not a nonexistent module.
2. **Fallback (if (1) is deferred):** at minimum, correct the message to stop recommending `import std/prelude` — it names both an unsupported import form (IMP012: namespace imports not supported, `internal/link/report.go:91`, emission site `internal/link/module_linker.go:156`) and a module that does not exist. Reword to the valid syntax family: e.g. "define an Eq instance or use `deriving (Eq)` on the type".

Out of scope: actually *supporting* namespace/bare imports (IMP012 relief) is a separate parser/feature change and is explicitly not proposed here; this doc only requires that no diagnostic recommend an unsupported form.

## Acceptance Criteria

1. `c == ctor` for a monomorphic ADT without `deriving (Eq)` **type-checks** in a function body (compiler-provided Eq; repro `eq1.ail` above goes green).
2. The identical expression in an `ensures` block behaves identically (both green or both red for every type/instance combination).
3. A user-defined `instance Eq[T]` that is *not* structural takes precedence over the compiler-provided default in **both** positions (once contract predicates are type-checked).
4. `deriving (Eq)` on polymorphic ADTs without Eq constraints continues to be rejected with its existing diagnostic.
5. No diagnostic in `internal/types` or `internal/errors` recommends `import std/prelude` or any bare/namespace import form (grep-enforced by test).
6. Regression: `examples/deriving_eq.ail` and the M-DX19 determinism tests in `internal/eval/eval_patterns.go` remain green; no change to runtime equality results (the runtime already computes structural equality).
7. Determinism: compiler-provided instances are registered deterministically per module (reuse the M-DX19 synthesis path), with a trace-guarantee test.

## Alternatives Considered

- **Type-check contracts only, keep requiring `deriving (Eq)`.** Rejected alone: it makes `ensures { result == Red }` fail where it currently passes, trading one position-dependence for a stricter-but-still-divergent regime and increasing agent friction (the original #713 complaint).
- **Lean fully on the runtime structural fallback.** Rejected: the type checker's Eq discipline carries real information (user-defined non-structural Eq); discarding it breaks dictionary passing for custom instances and undermines the effect/capability trace story.
- **Support namespace imports to make the hint true.** Out of scope (parser feature, separate design doc); the hint is wrong regardless because `std/prelude` doesn't exist.

## Related Documents

- Issue #713 (source report; measured on v0.30.0, defects re-verified at HEAD v0.38.5)
- Issue #963 (makeADTEq / derived-Eq dictionary machinery trace)
- M-DX19 (`deriving (Eq)`, derived ADT equality marker, determinism fix in `eval_patterns.go`)
- M-AILANG-SEMANTIC-CONTEXT R1b (actionable instance hints — this doc's IMP012 sub-item is the follow-through)