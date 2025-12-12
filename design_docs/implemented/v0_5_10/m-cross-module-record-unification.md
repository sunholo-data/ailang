# M-CROSS-MODULE-RECORD-UNIFICATION: Fix Type Unification for Cross-Module Nested Record Fields

**Status**: Implemented
**Target**: v0.5.10
**Priority**: P0 (High) - Blocked stapledons_voyage compilation
**Actual Time**: ~30 minutes
**Dependencies**: None
**Created**: 2025-12-12
**Implemented**: 2025-12-12

## Problem Statement

When importing a record type from another module that contains nested record fields, type unification fails with the error:

```
type unification failed at [return type annotation at sim/step.ail:138:8]:
failed to unify record field 'currentSystem':
failed to unify record field 'position':
cannot unify old record type with *types.TCon
```

**Root Cause:**
- Module A defines: `type SystemPos = {x: float, y: float, z: float}`
- Module A defines: `type StarSystem = {..., position: SystemPos, ...}`
- Module B imports: `import sim/celestial (StarSystem)`
- When B registers the `StarSystem` type alias, the nested `position` field's type remains as `TCon{Name: "SystemPos"}` (a reference)
- During unification, one side has expanded `TRecord{x: float, y: float, z: float}` and the other has `TCon{Name: "SystemPos"}`
- `unifyRecord` doesn't have a case for TCon, AND `SystemPos` wasn't in the aliasEnv

## Solution Implemented

### Two-Part Fix

**Part 1: Transitive Type Alias Import** (`internal/pipeline/pipeline_module.go:192-202`)

When importing any type from a dependency module, now imports ALL type aliases from that module. This ensures nested record types (like `SystemPos` inside `StarSystem`) are available for unification.

```go
// M-CROSS-MODULE-RECORD-UNIFICATION: Import ALL type aliases from this module
// This ensures nested record types (e.g., SystemPos inside StarSystem) are available
// for unification when the parent type is imported
for aliasName, aliasTarget := range depIface.TypeAliases {
    if _, exists := importedTypeAliases[aliasName]; !exists {
        importedTypeAliases[aliasName] = aliasTarget
        if cfg.TraceDefaulting {
            fmt.Printf("  Import transitive type alias %s -> %s\n", aliasName, aliasTarget)
        }
    }
}
```

**Part 2: Defensive TCon Handling** (`internal/types/unification_records.go:106-117`)

Added TCon case to `unifyRecord()` that attempts alias expansion. Provides clearer error messages when expansion fails.

```go
// M-CROSS-MODULE-RECORD-UNIFICATION: Handle TCon by expanding alias
// This occurs when a nested record field type is imported from another module
// and hasn't been expanded yet (e.g., position: SystemPos where SystemPos is TCon)
if t2Con, ok := t2.(*TCon); ok {
    expanded := u.expandAlias(t2Con)
    if expanded != t2Con {
        // Successfully expanded - retry unification with expanded type
        return u.Unify(t1, expanded, sub)
    }
    // Can't expand - might be an ADT or unknown type
    return nil, fmt.Errorf("cannot unify record with unexpandable type constructor %s", t2Con.Name)
}
```

### Why This Fix Is Correct

1. **Transitive import is complete:** All type aliases from a dependency are available, not just explicitly imported ones
2. **Consistent with existing pattern:** The main `Unify` function already calls `expandAlias` at the top
3. **Safe retry:** If expansion succeeds, we delegate back to `Unify` which handles all type combinations
4. **Good error message:** If expansion fails, we provide a specific error naming the type

## Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/pipeline/pipeline_module.go` | Transitive type alias import | +11 |
| `internal/types/unification_records.go` | TCon case in `unifyRecord()` | +12 |

**Total:** ~23 LOC

## Test Results

- [x] stapledons_voyage `sim/step.ail` type checks successfully
- [x] stapledons_voyage `sim/celestial.ail` compiles to Go
- [x] All type system tests pass (`go test ./internal/types/...`)
- [x] All pipeline tests pass (`go test ./internal/pipeline/...`)
- [x] Full test suite passes (`make test`)

## Success Criteria Met

- [x] stapledons_voyage compiles without unification errors
- [x] Nested record types unify correctly across modules
- [x] No regression in existing type unification tests
- [x] No performance regression

## References

- Related: M-TYPE-ALIAS-UNIFICATION (v0.5.8) - established aliasEnv pattern
- Related: M-FIX-RECORD-UPDATE (v0.5.8) - cross-module type alias imports
- Also fixed in same session: M-CODEGEN-ADT-TYPE-ASSERT (nullary ADT constructor type assertions)
