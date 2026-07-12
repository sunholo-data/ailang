# M-MATCH-XCHECK-ERROR-QUALITY — `MatchForeignConstructorError` shows empty constructor list for non-imported ADTs

**Status**: ✅ Implemented v0.30.0 (mission iteration 15) — **Option A** chosen (diagnostic-only
transitive constructor registry). Sprint plan: [m-match-xcheck-error-quality-sprint-plan.md](m-match-xcheck-error-quality-sprint-plan.md).
**Target**: v0.30.0 (was: v0.22.x/v0.23.0)
**Priority**: P2 — error message degradation. Correctness is fine; the suggestion list was empty.
**Estimated**: ~0.5 day (~50 LOC) — Actual: ~55 LOC (impl + test assertion)
**Dependencies**: None
**Source**: Discovered 2026-05-20 during sprint evaluation of [M-SCHEME-IMPORT-PRESERVE-ADT-HEAD](../v0_22_0/m-scheme-import-preserve-adt-head.md).

## Implementation note (v0.30.0)

Shipped **Option A**. A diagnostic-only `Constructor → ADT` registry
(`moduleImports.AllCtorTypes`) is built from every transitively-loaded interface via
`modLinker.GetLoadedModules()` and passed to the typechecker with
`SetDiagnosticConstructorTypes`. `lookupADTConstructors`
([`internal/types/typechecker_core.go`](../../../internal/types/typechecker_core.go)) consults it
**only** when the primary (direct-import + local) scan yields nothing for the requested ADT — so
direct/local constructors always win and the registry never affects scope or inference. Verified:
the repro (`match asNumber(...) { Ok ... Err ... }` importing only `std/result`) now prints
`Option's constructors are: None, Some`. Non-vacuity of the guarding test proven by disabling the
fallback (message reverts to empty). Option B's import-hint text remains a possible follow-up.

## Problem

After M-SCHEME-IMPORT-PRESERVE-ADT-HEAD landed, the foreign-constructor reproducer at `/tmp/foreign_ctor_repro.ail` correctly fires:

```
Error: type error in tmp/foreign_ctor_repro (decl 0):
  at constructor pattern Ok: match arm constructor 'Ok' belongs to ADT 'Result', not 'Option' (the scrutinee's type).
  Option's constructors are:
  Result's constructors are: Err, Ok
```

The actionable diagnosis is correct: `Ok` belongs to Result, scrutinee is Option, mismatch. But the **"Option's constructors are: " line is empty** because the user's module only imports `std/result (Ok, Err)` — it does NOT import `std/option`. So the typechecker's `constructorTypes` map has no entries for Option's `Some`/`None`.

Compare with a module that DOES import std/option directly — the same error correctly lists `Some, None`.

## Root cause

[`internal/pipeline/pipeline_module_imports.go:175-184`](../../internal/pipeline/pipeline_module_imports.go#L175-L184) (the M-CTOR-AUTO block) iterates over each *directly imported* module's constructors and adds them to `imports.ImportedCtorTypes`. It does NOT walk transitive dependencies. So when `std/json.getNumber` returns `Option[float]` and the user matches on it without importing `std/option`, the typechecker knows the constructor patterns are wrong (because the scrutinee's TApp head IS `Option`) but cannot enumerate Option's ctors for the suggestion list.

[`internal/types/typechecker_core.go:295-306`](../../internal/types/typechecker_core.go#L295-L306) (`lookupADTConstructors`) reverse-scans `tc.constructorTypes`, returning empty when the requested ADT name has no entries.

## Why this is P2

- The error correctly identifies the WRONG constructor and names BOTH ADTs.
- The user knows what's wrong; they're missing only the "did you mean Some/None?" hint.
- The fix is non-trivial (need to either walk transitive deps or maintain a global ADT registry) and could create naming-conflict surprises if done naively.

## Proposed fixes (pick one)

### Option A — Transitive ctor discovery only at error time

Plumb the module linker (or a slimmed-down ADT registry) into the typechecker. When `NewMatchForeignConstructorError` is built, fall back from `tc.constructorTypes` to the global registry for the missing ADT name. Constructors are still NOT in scope (the user can't call `Some(x)` without importing it); we just enumerate them for the suggestion list.

Pros: minimal scope, no scoping changes, fixes exactly the cosmetic gap.
Cons: requires plumbing the registry through the CoreTypeChecker constructor, slight architectural cost.

### Option B — Append an import-hint to the error

When `lookupADTConstructors(scrutADT)` returns empty, append an explicit hint to the error:

```
  Option's constructors are: (not visible in this module — try `import std/option (Some, None)`)
```

This needs to know which module Option came from. The TApp's TCon doesn't carry module info directly. The typechecker would need to look up the ADT's defining module from the import dep graph.

Pros: even better UX than just listing ctors — gives a concrete import statement.
Cons: requires similar registry plumbing as Option A.

### Option C — Defer

Mark the issue as `expected-cosmetic-degradation`, add a unit test that asserts the error still fires (the correctness invariant) and ignore the suggestion-list emptiness. Revisit when a user reports it as confusing.

Pros: zero work.
Cons: leaves a small UX paper-cut that future users may stumble on.

**Recommendation:** Option A. The registry plumbing is the smaller fix, and once available it can support Option B as a follow-up enhancement.

## Acceptance criteria

- Reproducer in [/tmp/foreign_ctor_repro.ail](/tmp/foreign_ctor_repro.ail) (or equivalent fixture) produces a non-empty constructor list for the scrutinee's ADT, EVEN when the user hasn't imported that ADT's module directly.
- New test in `internal/pipeline/match_foreign_constructor_function_call_test.go` asserts the constructor list is non-empty in the foreign-ctor error message for a transitively-known ADT.
- No regression in existing M-MATCH-ADT-XCHECK tests.

## Out of scope

- Auto-importing transitive constructors into the user's scope (would create naming-conflict surprises).
- Changing the `MatchForeignConstructorError` message format.
