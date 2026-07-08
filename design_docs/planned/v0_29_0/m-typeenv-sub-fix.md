# M-TYPEENV-SUB: TypeEnv Substitution Gap — ADT Return Types Lost in Cross-Module Exports

**Status**: Planned (design decision made: Option A + Approach 4)
**Target**: v0.9.2
**Priority**: P0 (type safety hole — silently accepts invalid programs)
**Estimated**: 2-3 days (revised from 4-6 hours after investigation)
**Dependencies**: None
**Author**: Mark + Claude
**Created**: 2026-03-27
**Last updated**: 2026-03-27 (design decision: Option A + Approach 4)
**Triggered by**: device_auth.ail passing `decode(jsonValue)` (Json->string mismatch) without compile error

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fixes nondeterministic type acceptance — programs that should fail now deterministically fail |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Restores local type checking correctness for cross-module calls |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents get correct type errors instead of silent runtime failures |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Cross-module function composition becomes type-safe |
| A11: Structured Failure | +1 | Type errors caught at compile time, not runtime |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly improves machine analysis

## Problem Statement

**The type checker silently accepts function calls where the return type of a function involving an imported ADT is used incorrectly — the return type is erased to a type variable.**

**Scope: Wider than initially thought.** This affects:
1. Cross-module exports (imported functions with ADT return types)
2. **Within-module chains** (local functions wrapping imported ADT constructors)
3. Any function whose return type is resolved via constraint solving involving imported types

**Root Cause:** `InferWithConstraints` in `typechecker_core.go` solves constraints and applies the substitution to the typed AST node (`applySubstitutionToTyped`) and to `CoreTI` (`CoreTI.ApplySubstitution`), but **never applies it to the `TypeEnv`** returned to callers. The type environment retains unresolved type variables from inference.

This has TWO downstream effects:
1. **Interface builder** reads unresolved types from the env, over-generalizes them into schemes with type variables (making functions appear to return "anything")
2. **Subsequent declarations in the same module** look up functions from the env and see dangling type variables that unify with anything

**Blast radius mapping:**

| Scenario | Caught? | Why |
|----------|---------|-----|
| Local ADT, direct call (`pick() -> Flavor`, then used wrong) | Yes | Local constructors resolve directly to TCon |
| Local ADT, chained via local func | Yes | Same — no imported type vars |
| Direct constructor use (`Ok(42)` where `string` expected) | Yes | Constructor types are concrete in the inference context |
| **Local func wrapping imported ADT** (`wrap() -> Result[...]`, then used wrong) | **No** | Return type resolved via constraint solving -> env gets type var |
| **Imported func returning ADT** (e.g., `jo()`, `decode()`) | **No** | Exported scheme has type var instead of concrete type |
| Functions returning only primitives (`string -> string`) | Yes | TCon types don't involve type variables |

**Concrete trace for `jo`:**

1. Type checker infers `jo`'s body: `JObject(kvs)` -> constraint `a1 = Json`
2. `inferCore` returns env with `jo : List[{key: string, value: Json}] -> a1`
3. `SolveConstraints()` produces substitution `{a1 -> Json}`
4. Substitution applied to typed AST and CoreTI
5. **Substitution NOT applied to env** -- env still has `a1`
6. Interface builder reads `jo : ... -> a1` from env
7. `generalizeType` wraps as `forall a1. List[...] -> a1`
8. Consumer module sees `jo` as returning "anything" — all type checks pass

**Impact:**
- **AI agents write type-incorrect code that compiles** — they only discover errors at runtime
- **The type system's #1 safety guarantee is broken** for any function returning an imported ADT
- Affects every module that uses Result, Option, Json, or any imported ADT in return types
- Within-module function chains are also affected, not just cross-module exports

## Goals

**Primary Goal:** Apply substitution to the type environment so exported function schemes have fully resolved types.

**Success Metrics:**
- `decode(jo([]))` produces a compile-time type error
- `needsString(jo([]))` produces a compile-time type error
- Within-module ADT chain mismatches caught (`wrap() -> Result, main -> string`)
- All existing tests pass (no regressions)
- All 152 example files pass
- Interface builder produces schemes with concrete ADT types, not type variables

---

## Investigation Report (2026-03-27)

### What was tried and why each approach failed

#### Approach 1: Naive `env.ApplySubstitution(sub)` in InferWithConstraints

**What:** Add `TypeEnv.ApplySubstitution` that walks all bindings in the current layer, applying the solve sub to resolve type variables.

**Result:** Type safety tests pass (7/7). But causes **REPL corruption** — `TestEmbeddedStdlibLoading` fails.

**Root cause of failure:** The REPL path (`module_registry_load.go:282-289`) adds ALL exports from ALL loaded modules into a **single env layer** via `ExtendScheme`. When the current declaration's `InferWithConstraints` produces a sub that maps `a2 -> [int]` (for its own purposes), `ApplySubstitution` applies this to every binding in the layer, including `dot_helper` from `std/embedding` whose scheme also uses `a2` as a quantified variable. This is a **capture of bound variables** — different modules' quantified vars share names because their `freshCounter` starts from 0 independently.

**Debug output confirming corruption:**
```
[DEBUG-ENV-SUB] dot_helper: TypeVars [a2 a3 a4] -> [], type: ([int], [int], int) -> int
```
Expected: `dot_helper: ([float], [float], float) -> float` (annotated with `list[float]`).

#### Approach 2: Filtered sub with quantified-var exclusion

**What:** When applying sub to a Scheme, exclude the scheme's own TypeVars and RowVars from the substitution (prevent capture). Also filter out effect row variables (e prefix) and Row/RowVar values.

**Result:** REPL passes. But **3 type safety tests fail** — the exclusion prevents fixing over-generalized schemes because the incorrectly-quantified variable IS in TypeVars.

**Root cause of failure:** The variable we need to resolve (a1 -> Json) is ALSO listed in the scheme's `TypeVars` because it was incorrectly generalized. Excluding TypeVars from the sub prevents the fix for exactly the cases we need to fix. We cannot distinguish "correctly quantified" from "incorrectly over-quantified" by looking at TypeVars alone.

#### Approach 3: Apply solveSub in inferLet/inferLetRec before generalization

**What:** Instead of fixing the env post-hoc, use the discarded solveSub from `ctx.SolveConstraints()` in `inferLet`/`inferLetRec` to resolve type variables in `valueType` BEFORE generalization. Filter out effect/row variables.

**Result:** All 7 type safety tests pass. REPL passes. But **3 example files fail** (adt_list_fields.ail, json_array_extraction.ail, record_patterns.ail).

**Root cause of failure:** The solveSub from inferLet's internal `SolveConstraints()` resolves ALL type variables from the function body, not just imported ADT return types. This over-constrains:
- **Record types**: `getName = \obj. match obj { {name} => name }` — solveSub resolves the parameter to `{name: a2}` (closed record), which can't unify with `{name: "Grace", id: 123}` (extra field). Should stay open.
- **Intermediate types**: Within a function body, intermediate record/list types get resolved to specific shapes, preventing later broader use.

#### Approach 4: Two-phase targeted substitution

**What:** Phase 1: `ApplySubstitution` with quantified-var exclusion (safe for other modules). Phase 2: `ApplySubstitutionToBindings` targeting only the current declaration's names WITHOUT protecting quantified vars.

**Result:** All 7 type safety tests pass. REPL still fails — same `Fractional[int]` corruption in `std/embedding.dot`.

**Root cause of failure:** Same as Approach 1 for the REPL path. Even with targeted binding names, the outer InferWithConstraints sub maps variable names that collide with scheme variables from other modules loaded into the same env layer.

### The Fundamental Tension

The problem reduces to a **capture-avoidance dilemma**:

```
Need to resolve:   a1 -> Json   in current decl's scheme  (a1 is in TypeVars, INCORRECTLY)
Must NOT resolve:  a2 -> [int]  in other module's scheme  (a2 is in TypeVars, CORRECTLY)
```

Both variables appear in their scheme's TypeVars. The substitution cannot distinguish them because:
1. Different modules' TypeCheckers start `freshCounter` from 0 independently
2. The REPL path loads all exports into a single env layer
3. The outer InferWithConstraints sub maps ALL variables, including those that coincidentally share names

### Key Architectural Insights

1. **Pipeline path (non-REPL):** Each module gets its own TypeChecker with independent freshCounter. Env layers are nested (child per declaration). `ApplySubstitution` only touches the current layer. Name collisions between modules are unlikely but possible.

2. **REPL path:** All module exports loaded into a SINGLE env layer (`ExtendScheme` loop at `module_registry_load.go:282-289`). Name collisions are guaranteed because every module's schemes start from `a1, a2, ...`.

3. **InferWithConstraints sub scope:** The sub from `SolveConstraints()` at `typechecker_core.go:349` maps ALL variables from the current declaration's inference. This includes variables created during scheme instantiation (fresh copies), NOT the original quantified names.

4. **inferLet internal sub scope:** The sub from `SolveConstraints()` at `typechecker_functions.go:180` maps ALL variables from the function body, including intermediate record/row variables that should stay polymorphic for generalization.

5. **Generalization timing:** Generalization in `inferLet`/`inferLetRec` happens INSIDE `tc.inferCore()`, BEFORE the outer `SolveConstraints()` in `InferWithConstraints`. The outer sub is computed after generalization — by then the scheme already has incorrect TypeVars.

---

## Proposed Solutions (Ordered by Preference)

### Option A: Alpha-rename schemes on env insertion (Recommended)

**Idea:** When adding a scheme to the env (in `ExtendScheme`, `BindScheme`), alpha-rename its quantified variables to globally unique names using a global counter. This ensures no two schemes in the same env ever share quantified variable names.

**Why it works:** Eliminates the name collision problem at the root. The sub from InferWithConstraints can safely map `a1 -> Json` without affecting `dot_helper` because dot_helper's variables would be `a_g42, a_g43, ...` (globally unique).

**Complexity:** Medium. Need a global counter and a rename pass on scheme insertion. ~40 LOC. Must handle TypeVars, RowVars, and the scheme's Type consistently.

**Risk:** Performance — extra allocation per scheme insertion. Mitigated by only renaming when schemes have TypeVars (monomorphic schemes are no-ops).

**Files:** `internal/types/env.go` (ExtendScheme, BindScheme), `internal/types/scheme.go` (rename helper)

### Option B: Track freshCounter range per InferWithConstraints call

**Idea:** Record the freshCounter range `[start, end)` for each InferWithConstraints call. When applying sub to the env, only substitute variables whose names fall within the current range.

**Why it works:** Variables from the current inference are in `[start, end)`. Variables from other modules' schemes are outside this range.

**Complexity:** Medium. Need counter tracking in TypeChecker and a range check in ApplySubstitution. ~30 LOC.

**Risk:** Fragile — depends on naming convention (`a{N}` with numeric N). Would break if variable naming changes.

**Files:** `internal/types/typechecker_core.go`, `internal/types/env.go`

### Option C: Apply sub only to current declaration, use env layering to isolate

**Idea:** In the REPL path, create a CHILD env for each module's declarations instead of adding everything to one layer. Then `ApplySubstitution` on the child layer only affects the current module.

**Why it works:** Each module's bindings are in separate env layers, so sub application is isolated.

**Complexity:** Low-Medium. Refactor REPL env construction in `module_registry_load.go`. ~20 LOC change.

**Risk:** May change lookup semantics if env layering affects name resolution order. Need careful testing.

**Files:** `internal/repl/module_registry_load.go`, `internal/types/env.go`

### Option D: Fix at generalization site (inferLet/inferLetRec) with selective sub

**Idea:** Don't discard the solveSub in inferLet. Apply it to valueType before generalization, but ONLY for variables that map to **named type constructors** (TCon, TApp with TCon head). Skip variables that map to records, lists, or other structural types.

**Why it works:** The over-generalization bug is specifically about ADT return types (TCon names like Json, Result, Option). Record/list shapes should stay polymorphic.

**Complexity:** Medium. Need a "is this a named type?" check. ~25 LOC.

**Risk:** May miss edge cases where a type variable should resolve to a structural type. Heuristic rather than principled.

**Files:** `internal/types/typechecker_functions.go`

---

## Design Decision: Option A + Approach 4 (Targeted Repair)

### Precise Diagnosis (from expert review)

This is not one bug but two:

- **Bug A (escaping metavariables):** Unresolved inference vars escape into generalized schemes.
  `inferLet` generalizes before the outer `SolveConstraints` runs, so vars like `a1` (which should
  be `Json`) get quantified as if they were legitimate polymorphic vars.

- **Bug B (non-alpha-safe quantified vars):** Quantified vars are represented as plain strings
  (`a1`, `a2`, ...) with no global uniqueness. Any post-hoc substitution on the env is
  capture-prone because different modules' schemes reuse the same names.

**The key invariant that is violated:**
> Env schemes must be closed under the current solve state before they become observable.

### Chosen Approach

**Ship: Option A + Approach 4 (targeted current-binding repair)**

This is the combination that matches all evidence:
- Approach 4 already fixed all 7 type safety tests in investigation
- Approach 4 only failed because of name collisions in REPL/shared-layer case
- Option A removes exactly that collision class

**Phase 1: Alpha-rename quantified vars on scheme insertion**
- On `ExtendScheme`/`BindScheme`, allocate globally unique IDs for all TypeVars/RowVars
- Rename scheme body consistently
- For v0.9.2: use unique textual prefix (e.g., `q$42$`, `rq$43$`) to avoid collision with
  inference vars (which use `a{N}`, `e{N}`, `r{N}`)
- Monomorphic schemes (no TypeVars) are no-ops

**Phase 2: Targeted current-binding repair after outer solve**
- After `InferWithConstraints` solves constraints, apply sub ONLY to bindings introduced
  by the current declaration
- Do NOT shield that declaration's quantified vars (they are the ones that were over-generalized)
- After Option A, the sub cannot accidentally hit binders from unrelated schemes

**Phase 3: Invariant assertions**
- `assertNoEscapingMetaVars(scheme)` in debug/test builds after generalization
- Turns silent soundness failure into hard internal error during development

### Why other options were rejected

| Option | Verdict | Reason |
|--------|---------|--------|
| **B (freshCounter range)** | Reject | Couples correctness to naming syntax; fragile under refactors; ages badly |
| **C (env layering)** | Optional hardening | Reduces blast radius but doesn't fix core soundness issue |
| **D (selective sub in inferLet)** | Reject as main fix | Attacks the symptom at wrong semantic boundary; over-constrains record/row types |

### Why A + targeted repair preserves passing examples

The failing "inferLet solveSub" attempt (Approach 3) broke examples because it substituted too
much structure into parameter-side types BEFORE generalization. The targeted post-solve repair
operates on the already-generalized/exported binding, not on intermediate body constraints. It
fixes only the exported observable type.

### Interface builder caution

The interface builder must NOT independently re-generalize already-repaired schemes. It should
consume repaired schemes directly, not re-run `generalizeType` on raw types that still contain
unresolved vars. Verify this path during implementation.

### Long-term direction (post v0.9.2)

Separate type variable concepts in the representation:

```go
type TVarID uint64

type TypeVar struct {
    ID   TVarID
    Kind TypeVarKind // Meta, Quant, RowMeta, RowQuant
    Name string      // pretty-print hint only
}
```

This makes the entire bug class structurally impossible:
- Env stores schemes over QuantVars
- Inference operates on MetaVars
- Generalization converts unsolved MetaVars -> QuantVars
- Instantiation replaces QuantVars with fresh MetaVars

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Solution approach: A + Approach 4 | Fixes both bugs with bounded risk | expert review | 2026-03-27 | medium |
| REPL path gets same fix as pipeline | REPL is used for WASM/stdlib loading | agent | sprint | low |
| Apply sub in InferWithConstraints (not inferLet) | Post-solve repair on exported type, not body internals | expert review | 2026-03-27 | n/a |

### Design Freeze

- [x] Fix location: inside `InferWithConstraints` (post-solve targeted repair)
- [x] Solution approach: Option A + Approach 4

## Existing Test Coverage

**Already written (in `internal/pipeline/type_safety_test.go`):**
- `TestTypeSafety_StringNotJson` — encode result passed to Json param (PASSES today, not affected by bug)
- `TestTypeSafety_StringNotJson_Positive` — valid Json usage compiles (PASSES)
- `TestTypeSafety_DecodeJsonNotString` — **currently FAILS (documents the bug)**
- `TestTypeSafety_CrossModule_JsonToString` — **currently FAILS (documents the bug)**
- `TestTypeSafety_CrossModule_StringParam` — cross-module int-to-string (PASSES)
- `TestTypeSafety_WithinModule_ImportedADTChain` — **currently FAILS (documents the bug)**
- `TestTypeSafety_WithinModule_ImportedADTChain_Positive` — valid chain compiles (PASSES)

**Regression tests that must not break:**
- `TestEmbeddedStdlibLoading` (REPL stdlib with float annotations in std/embedding)
- `TestCrossPackageTypeAliasUnification` (cross-package record aliases)
- `TestTransitiveTypeAliasPropagation` (transitive type alias chains)
- All 152 example files (`make verify-examples`)
- Full `make test`

## Success Criteria

- [ ] `decode(jo([]))` produces compile-time type error
- [ ] `needsString(jo([]))` produces compile-time type error
- [ ] Within-module: `wrap() -> Result, main -> string` caught
- [ ] `jo([kv("name", js("Alice"))])` still correctly types as `Json`
- [ ] `encode(jo([...]))` still correctly types as `string`
- [ ] `dot_helper` in std/embedding retains `float` types (not corrupted to `int`)
- [ ] `record_patterns.ail` — open record matching still works
- [ ] `adt_list_fields.ail` — nested record lists still compile
- [ ] `json_array_extraction.ail` — JSON number arrays still compile
- [ ] All existing tests passing (`make test`)
- [ ] All 152 examples valid (`make verify-examples`)
- [ ] No regression in cross-module type alias unification (M-TYPE-ALIAS)

## Implementation Plan

**Phase 1: Alpha-rename on scheme insertion** (~0.5 day)
- [ ] Add global scheme var counter to `env.go` (package-level `atomic.Uint64` or similar)
- [ ] Add `AlphaRenameScheme(scheme *Scheme) *Scheme` helper
  - Allocate fresh globally unique IDs for all TypeVars and RowVars
  - Use prefix like `q${globalID}$` to avoid collision with inference vars (`a{N}`)
  - Build rename map, walk scheme Type consistently
  - No-op for monomorphic schemes (empty TypeVars)
- [ ] Call alpha-rename in `ExtendScheme` and `BindScheme`
- [ ] Verify: `TestEmbeddedStdlibLoading` still passes (REPL path)
- [ ] Verify: `make test` still passes (no regressions from renaming alone)

**Phase 2: Targeted current-binding repair** (~0.5 day)
- [ ] In `InferWithConstraints` (typechecker_core.go), after outer `SolveConstraints`:
  - Extract current declaration names from `expr` (`*core.Let` -> name, `*core.LetRec` -> names)
  - For each name, look up scheme in `updatedEnv`
  - Apply filtered sub (exclude effect/row vars) WITHOUT protecting quantified vars
  - Write repaired scheme back
- [ ] Verify: all 7 type safety tests pass
- [ ] Verify: `TestEmbeddedStdlibLoading` still passes

**Phase 3: Invariant assertions + regression check** (~0.5 day)
- [ ] Add `assertNoEscapingMetaVars(scheme)` check in debug builds
  - After generalization, before interface extraction
  - Detects any unification var from current solve session in exported schemes
- [ ] Verify interface builder consumes repaired schemes (not re-generalizing)
- [ ] Pass all 152 examples (`make verify-examples`)
- [ ] Full `make test`

**Phase 4: Cleanup** (~0.5 day)
- [ ] Remove `filterTypeVarSub` function if unused
- [ ] Remove any debug print statements
- [ ] Update CHANGELOG.md
- [ ] Move design doc to implemented/

### Files to Modify

| File | Change | LOC estimate |
|------|--------|-------------|
| `internal/types/env.go` | Alpha-rename on scheme insertion, global counter | ~50 |
| `internal/types/typechecker_core.go` | Targeted repair after outer solve | ~25 |
| `internal/pipeline/type_safety_test.go` | Already written, verify tests pass | ~0 |

## Non-Goals

- **Separating MetaVar/QuantVar in type representation** — Correct long-term fix but too large for v0.9.2
- **Improving error messages** — The unification error is already informative enough
- **Fixing unrelated type inference bugs** — Focus only on the ADT return type gap

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Alpha-renaming adds allocation overhead | Low | No-op for monomorphic schemes; only allocates for polymorphic |
| Renamed vars break pretty-printing / error messages | Low | Keep `Name` field as display hint; rename only internal identity |
| Interface builder re-generalizes repaired schemes | High | Audit interface builder path; it must consume repaired schemes directly |
| Fix breaks Num/Fractional defaulting | High | Verified: defaulting happens in `defaultAmbiguitiesTopLevel` which already applies sub to constraints |

## Related Documents

- [M-TYPE-ALIAS: Cross-Package Record Type Alias Unification](../m-type-cross-package-alias-unification.md) — Related fix for record type aliases across packages
- [M-PKG-INTERREF: Package Inter-Function References](m-pkg-interref-fix.md) — Another cross-module resolution fix in the same sprint
- [M-FIX-FLOAT-OP](../../implemented/v0_5_9/m-fix-float-op-summary.md) — Prior type annotation preservation fix (similar pattern: type info lost between phases)

---

**Document created**: 2026-03-27
**Last updated**: 2026-03-27 (design decision: Option A + Approach 4)
