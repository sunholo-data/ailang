# M-TYPE-ALIAS: Cross-Package Record Type Alias Unification

**Status**: Proposed
**Priority**: P1 (blocks clean cross-package record type usage)
**Estimated**: 1-2 days
**Dependencies**: None
**Author**: Mark + Claude
**Created**: 2026-03-23
**Triggered by**: DocParse billing integration — `cannot unify type constructor Usage with *types.TRecord`

---

## Problem Statement

When a record type alias is defined in Package A, used in Package B's function signatures, and called from Package C, the type checker fails to unify the nominal type constructor with the structural record.

```
Package A (billing_entitlements/usage_policy):
  export type Usage = { documentsParsed: int, pagesParsed: int, ... }
  export func applyDelta(current: Usage, delta: UsageDelta) -> Usage

Package B (billing_store/usage_repo):
  import pkg/.../usage_policy (Usage, emptyUsage)
  export func getUsage(...) -> Result[Usage, string]
    // Returns Ok({ documentsParsed: asInt(...), ... })  -- structural record literal

Package C (docparse_access_gate/usage_recording):
  import pkg/.../usage_repo (getUsage, putUsage)
  import pkg/.../usage_policy (applyDelta, parseDelta, emptyUsage)
  let current = match getUsage(...) { Ok(u) => u, ... }
  let updated = applyDelta(current, delta)  // FAILS: "cannot unify type constructor Usage with *types.TRecord"
```

### Current Workaround

Explicit type annotations + importing the type name:
```ailang
import pkg/.../usage_policy (Usage, UsageDelta, applyDelta, ...)
let current: Usage = match getUsage(...) { ... }
let updated: Usage = applyDelta(current, delta)
```

This works but shouldn't be necessary — type aliases are supposed to be transparent.

---

## Root Cause Analysis

### Two Issues

**Issue 1: TCon vs TRecord in unifier (partially fixed)**

In `internal/types/unification_core.go`, the `TCon` case didn't handle `TRecord` as t2. We added alias expansion for this case. However, the fix only works if the alias is **registered** in the unifier's `aliasEnv`.

**Issue 2: Cross-package alias propagation (the real blocker)**

The pipeline builds type alias environments per-module during compilation. When compiling Package C:

1. Package C imports `getUsage` from Package B and `applyDelta` from Package A
2. Package B's interface exports `getUsage() -> Result[Usage, string]` where `Usage` is a `TCon`
3. Package A's interface exports `applyDelta(current: Usage, ...) -> Usage`
4. The type checker needs `Usage` in its alias environment to expand `TCon("Usage")` → `TRecord{...}`

The alias propagation code (`pipeline_module_imports.go:142-152`) imports type aliases from **directly imported modules**. But:
- Package C imports from Package B (usage_repo) and Package A (usage_policy)
- Package B's interface may not include `Usage` in its `TypeAliases` map if it only re-exports the type implicitly through function signatures
- The alias is defined in Package A — Package C imports `applyDelta` from Package A which should bring `Usage` into scope

**The gap**: When a function signature references a type from another package (`Usage` from billing_entitlements appears in usage_repo's `getUsage` signature), the alias needs to be propagated through the calling chain. The current code only imports aliases from the **interface of the directly imported module**, not from transitive dependencies referenced in type signatures.

---

## Proposed Fix

### Approach: Transitive alias collection during import resolution

When building the type alias environment for a module, collect aliases not just from direct imports but also from types referenced in imported function signatures.

**In `pipeline_module_imports.go`**, after collecting aliases from direct imports:

```go
// Phase 2: Collect aliases from types referenced in imported function signatures
// If getUsage() returns Result[Usage, string], and Usage is defined in
// billing_entitlements, we need that alias even though we didn't directly
// import from billing_entitlements/usage_policy
for _, fn := range imports.ImportedFunctions {
    referencedTypes := collectTypeNames(fn.Type)
    for _, typeName := range referencedTypes {
        if _, exists := imports.ImportedTypeAliases[typeName]; !exists {
            // Search all loaded module interfaces for this alias
            if alias := findAliasInLoadedModules(typeName, modLinker); alias != nil {
                imports.ImportedTypeAliases[typeName] = alias
            }
        }
    }
}
```

### Alternative: Eager alias embedding in interfaces

When Package B's interface is built, embed all type aliases referenced in its function signatures — not just the ones defined locally. So `usage_repo`'s interface would include `Usage → {documentsParsed: int, ...}` even though `Usage` is defined elsewhere.

**Pros**: No change to the import resolution logic
**Cons**: Interfaces grow larger, aliases may conflict if different packages define same-named types

### Recommendation

Start with the transitive collection approach — it's more targeted and doesn't change interface size.

---

## Affected Code Paths

| File | Change |
|------|--------|
| `internal/types/unification_core.go:188-197` | TCon vs TRecord alias expansion (already added) |
| `internal/pipeline/pipeline_module_imports.go:125-152` | Transitive alias collection from imported signatures |
| `internal/iface/builder.go` | May need to include referenced aliases in interface |
| `internal/pipeline/pipeline_module_compile.go:198-204` | Alias registration |

---

## Testing Strategy

1. **Unit test**: Create unifier with cross-package alias scenario (TCon from one package, TRecord from another)
2. **Integration test**: Three-module compilation chain where type alias crosses 2 package boundaries
3. **Regression test**: Existing `TestTypeAliasExpansion` continues to pass
4. **End-to-end**: `docparse_access_gate` compiles without explicit type annotations

---

## Impact

Fixing this unblocks:
- Clean cross-package record type usage (no workaround annotations needed)
- The billing package suite working out of the box
- Any future multi-package system using shared domain types

---

## Current Workaround (until fixed)

Import the type names explicitly and add type annotations to let bindings:

```ailang
import pkg/.../usage_policy (Usage, UsageDelta, applyDelta, parseDelta, emptyUsage)

let current: Usage = match getUsage(...) { Ok(u) => u, Err(_) => emptyUsage() }
let delta: UsageDelta = parseDelta(pages, bytes, ocrPages)
let updated: Usage = applyDelta(current, delta)
```

This forces the type checker to resolve `Usage` via the direct import from the defining package.
