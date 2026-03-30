# Sprint Plan: M-TYPE-ALIAS — Cross-Package Record Type Alias Unification

## Summary
Fix cross-package type alias propagation so that record type aliases (e.g., `Usage`) defined in Package A and used in Package B's function signatures are available for unification when called from Package C — without requiring explicit type annotations.

**Duration:** 1 day (3 milestones)
**Dependencies:** None (partial fix already landed in 575b5051)
**Risk Level:** Low — targeted changes to import resolution and interface building

## Current Status Analysis

### Completed Recently
- ✅ TCon vs TRecord alias expansion in unifier (`unification_core.go:188-197`)
- ✅ TypeName propagation through unification (`unification_core.go:244-309`)
- ✅ Cross-module record unification for type imports (`pipeline_module_imports.go:142-152`)
- ✅ Workaround documented (explicit type annotations)

### Velocity
- Recent average: ~300-500 LOC/day for type system changes
- This sprint: ~200 LOC total (targeted, well-scoped)

### Remaining from Design Doc
- ⏳ Alias propagation for value/constructor imports: ~20 LOC
- ⏳ Transitive alias embedding in interfaces: ~80 LOC
- ⏳ Integration tests + end-to-end verification: ~100 LOC

## Root Cause

Two gaps in the current alias propagation:

1. **Value imports don't trigger alias collection.** In `resolveSelectiveImports()`, the "import ALL type aliases" block (lines 142-152) only runs inside the `GetType` conditional. When Package C imports a function like `applyDelta` from Package A, it never enters this block, so Package A's aliases (including `Usage`) are not imported.

2. **Interfaces don't include transitive aliases.** Package B's interface only contains aliases for types it *defines*, not types it *imports* from dependencies. So even if we fix (1), the transitive case (C imports from B, B uses types from A, C doesn't import from A) still fails.

## Proposed Milestones

### Milestone 1: M1_VALUE_IMPORT_ALIASES — Fix alias propagation for all import types
**Goal:** Import type aliases from a dependency whenever *any* symbol is imported, not just type symbols.
**Estimated:** 20 LOC implementation + 0 LOC tests = 20 LOC
**Duration:** 30 minutes

**Tasks:**
- After the symbol loop in `resolveSelectiveImports()` (after line 170), add unconditional import of ALL type aliases from the dependency interface
- Remove the now-redundant block inside the `GetType` conditional (lines 142-152), or keep it as the per-symbol alias import and add the blanket import outside

**Key Change in `pipeline_module_imports.go`:**
```go
// After the for loop over symbols, always import all type aliases from this module
// M-TYPE-ALIAS: Ensure aliases are available for unification regardless of import type
for aliasName, aliasTarget := range depIface.TypeAliases {
    if _, exists := imports.ImportedTypeAliases[aliasName]; !exists {
        imports.ImportedTypeAliases[aliasName] = aliasTarget
    }
}
```

**Acceptance Criteria:**
- [ ] Importing a function from a module also imports that module's type aliases
- [ ] Existing `TestTypeAlias*` tests still pass
- [ ] `make test` passes

### Milestone 2: M2_TRANSITIVE_ALIAS_EMBED — Embed referenced aliases in module interfaces
**Goal:** When building a module's interface, include type aliases from dependencies that are referenced in exported function signatures. This handles the transitive case.
**Estimated:** 60 LOC implementation + 20 LOC tests = 80 LOC
**Duration:** 1-2 hours

**Tasks:**
- In `internal/iface/builder.go`, after building the interface, scan all exported function signatures for TCon references
- For each TCon that isn't already in the interface's TypeAliases, look it up in the module's imported aliases and add it
- This requires passing the module's imported aliases to the builder (or storing them during elaboration)

**Key Changes:**
1. `internal/iface/builder.go` — Add `collectReferencedAliases()` that walks exported function types
2. `internal/pipeline/pipeline_module_compile.go` — Pass imported aliases to interface builder
3. Helper function `collectTypeConsFromType(t types.Type) []string` to extract TCon names from a type

**Acceptance Criteria:**
- [ ] Package B's interface includes `Usage` alias when `getUsage() -> Result[Usage, string]` is exported
- [ ] No alias name conflicts (first-defined wins)
- [ ] `make test` passes

### Milestone 3: M3_TESTS_AND_VERIFY — Integration tests + end-to-end verification
**Goal:** Comprehensive test coverage and verification with real package code.
**Estimated:** 80 LOC tests + 20 LOC example = 100 LOC
**Duration:** 1 hour

**Tasks:**
- Write integration test: three-module chain where type alias crosses 2 package boundaries
- Write unit test: unifier with cross-package alias scenario (TCon from one package, TRecord from another)
- Verify existing `TestTypeAliasExpansion` still passes
- Test docparse_access_gate compiles without explicit type annotations (remove workaround)
- Update CHANGELOG.md

**Acceptance Criteria:**
- [ ] New `TestCrossPackageTypeAliasUnification` passes
- [ ] New `TestTransitiveTypeAliasPropagation` passes
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] CHANGELOG.md updated

## Success Metrics
- All tests passing: ✅
- All linting passing: ✅
- Docparse access gate compiles without workaround annotations
- Cross-package record types work transparently (no explicit type imports needed)
- Documentation: CHANGELOG.md, design doc moved to implemented/

## Dependencies
- None — partial fix already landed

## Notes
- The M1 fix alone may be sufficient for the specific docparse case (Package C imports from both A and B)
- M2 is needed for the general transitive case (C only imports from B, B uses types from A)
- Conservative approach: implement M1 first, verify, then M2
