# M-TESTING-ADT-IMPORT-RESOLUTION: Fix inline test harness failing to resolve imported ADT constructors

**Status**: Planned
**Target**: v0.16.4
**Priority**: P1 — blocks `make test_core` in motoko_agent (19/19 parse_test.ail failures)
**Estimated**: ~3 hours (~35 LOC)
**Dependencies**: M-MOTOKO-INLINE-TEST-HARNESS (v0.16.3, completed — introduced CombinedResolver)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | 0 | Test harness only, no trace impact |
| A3: Effect Legibility | 0 | No effect handling changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Enables inline tests on functions that use imported ADT constructors |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents writing AILANG can now test functions that use Option/Result/custom ADTs |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | Resource costs unchanged |
| A10: Composability | +1 | Imported ADT constructors compose with inline test harness |
| A11: Structured Failure | 0 | No error handling surface changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine testability

## Problem Statement

`ailang test` fails with `harness evaluation failed: failed to resolve global $adt.make_Option_Some: module $adt not found` whenever a tested function uses ADT constructors imported from another module (e.g. `Some`/`None` from `std/option`, `Ok`/`Err` from `std/result`, custom ADTs from user modules).

**Root cause — single missing case in `CombinedResolver`:**

When the elaborator encounters a constructor call like `Some(n)` or `None`, it generates a Core `VarGlobal` with `module="$adt"` and `name="make_Option_Some"`. `$adt` is a synthetic namespace — not a real file-backed module. The production runtime resolver (`internal/runtime/resolver.go:58`) dispatches `$adt` refs to `resolveAdtFactory()` which searches `mod.Iface.Constructors`. But `CombinedResolver` in `internal/testing/executor_helpers.go` has no `$adt` case — it falls through to Case 2 (module-qualified lookup), which looks for `"$adt"` in `r.Modules`, never finds it, and returns an error.

**Systemic audit (pre-existing, not a regression):**
- `$adt` and `$builtin` are the only two synthetic module names in the elaborator
- `$builtin` is handled in `CombinedResolver` Case 1 (`ref.Module == "$builtin"` or name starts with `_`)
- `$adt` was simply never added alongside it — this fix closes the last gap
- This has been broken since `CombinedResolver` was introduced in M-DX25 (v0.7.4)
- `make test` (AILANG's own Go tests) is not affected — the gap only shows when running `ailang test` on `.ail` files that use imported ADTs in test bodies

**Observed failures (motoko_agent `make test_core`):**
```
✗ ef_some_test_1 — failed to resolve global $adt.make_Option_Some: module $adt not found
✗ ef_some_test_2 — failed to resolve global $adt.make_Option_None: module $adt not found
... (19/19 tests in parse_test.ail)
```

## Goals

**Primary Goal:** `CombinedResolver` resolves `$adt` references by searching `r.Modules[*].Iface.Constructors`, mirroring what the production resolver already does.

**Success Metrics:**
- `parse_test.ail` goes from 19/19 failing → 0 failing in `make test_core`
- `TestADTConstructorFromImportedModule` passes (harness path)
- `TestADTConstructorInCluster` passes (cluster path)
- `make test` full suite remains green

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Use `mod.Iface.Constructors` for arity lookup | Iface is authoritative; Types AST may not always have arity computed | agent | design | low |
| Return `ConstructorClosure` (not `BuiltinFunction`) for n-ary | `ConstructorClosure` is what `injectADTConstructors` already uses; `BuiltinFunction` is the runtime approach | agent | design | low |

### Design Freeze

All decisions above are low change-cost and resolvable by the implementer. No pre-coding pause required.

## Solution Design

### Overview

Add a `$adt` case to `CombinedResolver.ResolveValue` (3 LOC) and a `resolveAdtFactory` helper method (~25 LOC) to `internal/testing/executor_helpers.go`. The helper parses `make_Option_Some` → `typeName="Option"`, `ctorName="Some"`, searches `r.Modules` for the matching `Iface.Constructors` entry, and returns a `TaggedValue` (nullary) or `ConstructorClosure` (n-ary).

This is a pure test-harness change — no parser, pipeline, or runtime changes.

### Architecture

**Pattern: mirror `injectADTConstructors` but for imported modules**

`injectADTConstructors` (already in `executor_helpers.go`) handles ADT constructors from the *current source file* by scanning `e.sourceFile.Decls` for `*ast.TypeDecl`. The new `resolveAdtFactory` handles ADT constructors from *imported modules* by scanning `r.Modules[*].Iface.Constructors`.

**Data available in `CombinedResolver`:**
- `r.Modules map[string]*loader.LoadedModule` — all loaded modules, each with `Iface *iface.Iface`
- `mod.Iface.Constructors map[string]*ConstructorScheme` — keyed by ctor name, has `TypeName` and `Arity`

**Resolution logic:**
```
$adt.make_Option_Some
  → parse: typeName="Option", ctorName="Some"
  → search r.Modules for mod where mod.Iface.Constructors["Some"].TypeName == "Option"
  → found in std/option: Arity=1
  → return &eval.ConstructorClosure{TypeName:"Option", CtorName:"Some", Arity:1}
```

### Implementation Plan

**M1: Add `$adt` case to `CombinedResolver`** (~1 hour, ~30 LOC)

In `internal/testing/executor_helpers.go`:

1. In `CombinedResolver.ResolveValue`, add before Case 1:
   ```go
   // Case 0: $adt synthetic module — ADT constructors from imported modules
   if ref.Module == "$adt" {
       return r.resolveAdtFactory(ref.Name)
   }
   ```

2. Add `resolveAdtFactory` method:
   ```go
   func (r *CombinedResolver) resolveAdtFactory(factoryName string) (eval.Value, error) {
       if !strings.HasPrefix(factoryName, "make_") {
           return nil, fmt.Errorf("invalid $adt factory name: %s", factoryName)
       }
       parts := strings.SplitN(factoryName[5:], "_", 2)
       if len(parts) != 2 {
           return nil, fmt.Errorf("invalid $adt factory format: %s (expected make_TypeName_CtorName)", factoryName)
       }
       typeName, ctorName := parts[0], parts[1]

       for _, mod := range r.Modules {
           if mod == nil || mod.Iface == nil || mod.Iface.Constructors == nil {
               continue
           }
           ctor, ok := mod.Iface.Constructors[ctorName]
           if !ok || ctor.TypeName != typeName {
               continue
           }
           if ctor.Arity == 0 {
               return &eval.TaggedValue{
                   TypeName: typeName,
                   CtorName: ctorName,
                   Fields:   []eval.Value{},
               }, nil
           }
           return &eval.ConstructorClosure{
               TypeName: typeName,
               CtorName: ctorName,
               Arity:    ctor.Arity,
           }, nil
       }
       return nil, fmt.Errorf("constructor %s.%s not found in any loaded module", typeName, ctorName)
   }
   ```

**M2: Regression tests** (~1 hour, ~60 LOC)

Add to `internal/testing/executor_regression_test.go`:

1. **`TestADTConstructorFromImportedModule`** — harness path (no cross-function deps):
   ```ailang
   module test_adt_harness
   import std/option (Some, None)
   func wrap_some(n: int) -> bool
     tests [(1, true), (0, true)]
     { match Some(n) { Some(_) => true, None => false } }
   func is_none(n: int) -> bool
     tests [(0, false)]
     { match None { Some(_) => true, None => false } }
   ```

2. **`TestADTConstructorInCluster`** — cluster path (cross-function dep + imported ADT):
   ```ailang
   module test_adt_cluster
   import std/option (Some, None)
   func wrap(n: int) -> bool = match Some(n) { Some(_) => true, None => false }
   func tested(n: int) -> bool
     tests [(1, true), (0, true)]
     { wrap(n) }
   ```
   (`tested` depends on `wrap` → cluster path; `wrap` uses `Some` → triggers `$adt` resolution)

**M3: Cross-repo verify + CHANGELOG** (~1 hour)

- `make quick-install` → `make test_core` in motoko_agent → `parse_test.ail` all green
- `make test` in ailang → full suite green
- CHANGELOG entry under v0.16.4
- Move design doc to `design_docs/implemented/v0_16_4/`

### Files to Modify

| File | Change | ΔLoC |
|------|--------|------|
| `internal/testing/executor_helpers.go` | Add `$adt` case to `ResolveValue` + `resolveAdtFactory` method | +30 |
| `internal/testing/executor_regression_test.go` | Two new regression tests | +55 |
| `changelogs/v0.10-current.md` | CHANGELOG entry under v0.16.4 | +10 |

**Total: ~95 LOC**

## Examples

### Before (broken)

```ailang
module parse_test
import std/option (Some, None)

func ef_some(text: string, fence: string) -> bool
  tests [
    (("```bash\necho hello\n```", "```bash"), true)
  ]
  { match extract_fence(text, fence) { Some(_) => true, None => false } }
```
```
✗ ef_some_test_1 — harness evaluation failed: failed to resolve global
  $adt.make_Option_Some: module $adt not found or function make_Option_Some not in module
```

### After (fixed)

```
✓ ef_some_test_1 (32ms)
✓ ef_some_test_2 (14ms)
... (all 19 tests pass)
```

## Success Criteria

- [ ] `TestADTConstructorFromImportedModule` passes (harness path, nullary + n-ary constructors)
- [ ] `TestADTConstructorInCluster` passes (cluster path, imported ADT + cross-function dep)
- [ ] `make test_core` in motoko_agent: `parse_test.ail` 0 failures (was 19/19)
- [ ] `make test` full AILANG suite green
- [ ] `go test ./internal/testing/... -count=5` green (determinism check)
- [ ] CHANGELOG entry under v0.16.4
- [ ] Design doc moved to `implemented/v0_16_4/`

## Testing Strategy

**Regression tests** (in `internal/testing/executor_regression_test.go`):
- Harness path: nullary (`None`) and n-ary (`Some(n)`) imported constructors
- Cluster path: imported constructor used inside a local helper function

**Cross-repo** (motoko_agent `make test_core`):
- `parse_test.ail` — 19 tests using `Some`/`None` in match expressions in test bodies
- `agents_md.ail` — already passes (M-MOTOKO-INLINE-TEST-HARNESS); must not regress

**Determinism**: run with `-count=5` (Go map iteration over `r.Modules` must not affect correctness)

## Non-Goals

- Ambiguity detection when two loaded modules export a same-named constructor of the same type (rare; production resolver handles it; test harness can leave it as first-match for now)
- Caching `TaggedValue` singletons for nullary constructors across test cases (not needed at test harness scale)
- Fixing other pre-existing harness gaps (e.g. `context_usage.ail` partial failures — separate issue)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `mod.Iface` is nil for some modules (e.g. `$builtin` pseudo-module) | Low | nil-guard in loop: `if mod == nil \|\| mod.Iface == nil` |
| Same ctor name in two imported modules (ambiguous) | Low | First-match semantics; same ambiguity already exists in bare-name env injection |
| `make_` factory name format changes | Low | Already stable in elaborator since v0.6.0; format documented in runtime resolver |

## Related Documents

- [design_docs/implemented/v0_16_3/m-motoko-inline-test-harness.md](../v0_16_3/m-motoko-inline-test-harness.md) — M1/M2 fixes that introduced `CombinedResolver`; this is the M3 follow-up
- [design_docs/implemented/v0_7_4/m-dx25-inline-tests-import-resolution.md](../../implemented/v0_7_4/m-dx25-inline-tests-import-resolution.md) — original `CombinedResolver` introduction (M-DX25)
- [design_docs/implemented/v0_6_2/m-dx22-adt-constructor-resolution.md](../../implemented/v0_6_2/m-dx22-adt-constructor-resolution.md) — ADT constructor resolution in codegen (related pattern)
- `internal/runtime/resolver.go:167` — `resolveAdtFactory` — the production version of what we're adding to `CombinedResolver`
- `internal/iface/iface.go:35` — `ConstructorScheme` — `TypeName`, `CtorName`, `Arity` fields used by fix

---

**Document created**: 2026-05-06
**Last updated**: 2026-05-06
