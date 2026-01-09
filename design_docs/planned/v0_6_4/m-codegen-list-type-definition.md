# M-CODEGEN-LIST-TYPE-DEFINITION: Fix Undefined List Type in Generated Code

**Status**: Planned
**Target**: v0.6.4
**Priority**: P1 (High)
**Estimated**: 4-6 hours (2h investigation + 2h implementation + 1h testing + 1h docs)
**Dependencies**: None
**Created**: 2026-01-09
**Author**: Claude Code (design-doc-creator)

## Problem Statement

**GitHub Issue #116**: AILANG codegen generates `*List` type references in `types.go` but does not define the List type. This causes `undefined: List` compilation errors when building.

### Current Pain Points

1. **Broken Compilation**: When AILANG ADTs contain list fields, the generated Go code references `*List` type that doesn't exist:
   ```go
   type StarSystem struct {
       Planets *List  // ❌ undefined: List
   }

   type MouseState struct {
       Buttons *List  // ❌ undefined: List
   }
   ```

2. **Silent Fallback**: The TypeMapper in `types.go` is supposed to convert `TApp("List", T)` to `[]T` (Go slices), but in some cases it falls back to an undefined `*List` type reference.

3. **Incomplete Architecture**: While `runtime.go` has List helper functions (`Cons`, `ListHead`, `ListTail`, `ListLen`), there's no corresponding Go type definition, causing a semantic mismatch.

4. **No Clear Fix Path**: The root cause isn't immediately obvious - appears to be a codegen path that bypasses the ListType special case handling.

### Scope: Why This Is Systemic

This is NOT just a single bug. Investigation reveals a pattern of incomplete type mapping:

- **ListType handling** in TypeMapper is incomplete (some paths hit fallback)
- **ADT slice converters** work around missing List type with reflection
- **Record types** with list fields need proper type generation
- **Potential future issues** with other generic types (Option, Either, etc.)

**The Fix:** Unified type mapping that handles all cases, not band-aids.

## Goals

**Primary Goal**: Eliminate undefined List type errors while maintaining clean Go code generation.

**Success Metrics**:
1. ✅ All AILANG ADTs compile to valid Go code (no `undefined: List`)
2. ✅ List fields use proper Go slice types `[]T` where possible
3. ✅ Type checking finds issues before code generation
4. ✅ ADT field types are consistently handled (no special cases)
5. ✅ All existing tests pass + 3+ new test cases for list field ADTs

## Solution Design

### Overview

The fix has three components:

1. **Root Cause Analysis**: Identify which codegen path creates `*List` references
2. **Type Mapping Fix**: Ensure all paths use TList → []T conversion
3. **Validation**: Add pre-codegen validation to catch type mapping issues early

### Architecture

#### Component 1: Identify Fallback Path

**Current TypeMapper behavior:**
- `TList(T)` → Maps to `[]T` ✅ (line 72-77 in types.go)
- `TApp("List", T)` → Maps to `[]T` ✅ (line 125-138 in types.go)
- **Some unknown path** → Falls back to `*List` ❌

**Investigation needed**: Find which AST type creates `List` without the special case handling.

**Candidates to check**:
- Elaboration of list syntax that produces different Core type
- ADT constructor type annotations using `List` directly
- Module imports where List is treated as a type name
- Record field types with implicit List wrapping

#### Component 2: Type Mapping Unification

**In `internal/gen/golang/types.go`**:

Add comprehensive List handling to `mapTypeWithVisited()`:

```go
// After TArray case (around line 84), add:

case *types.TData:
    // M-CODEGEN-UNIFIED: Handle data type instances
    // Includes TList(T) which is really TData("[]", [T])
    if typ.Name == "[]" || typ.Name == "List" {
        if len(typ.Args) > 0 {
            elemType, err := tm.mapTypeWithVisited(typ.Args[0], visited)
            if err != nil {
                return GoType("[]interface{}"), nil
            }
            return GoType(fmt.Sprintf("[]%s", elemType)), nil
        }
        return GoType("[]interface{}"), nil
    }
    // Other data types...
```

**In `mapTCon()` (line 158-180)**:

Add "List" as a reserved name that should never be used as a Go type name:

```go
case "List":
    return "", fmt.Errorf(
        "List type without type argument - use TList(T) or TApp(\"List\", T), not TCon(\"List\")")
```

This forces all List usage through the proper type application path.

#### Component 3: Codegen Validation

**New file**: `internal/gen/golang/validate_types.go`

Add pre-codegen validation:

```go
// ValidateTypeMapping checks that all types can be mapped to valid Go types.
// Returns a list of unmappable types (potential undefined reference sources).
func (tm *TypeMapper) ValidateTypeMapping(prog *core.Program) error {
    issues := []string{}

    // Walk all Core nodes, collect types
    types := extractAllTypes(prog)

    for t := range types {
        _, err := tm.MapType(t)
        if err != nil {
            issues = append(issues, fmt.Sprintf(
                "Type mapping failed: %T - %v", t, err))
        }
    }

    if len(issues) > 0 {
        return fmt.Errorf("Type validation failed:\n%s",
            strings.Join(issues, "\n"))
    }
    return nil
}
```

Call this in `cmd/ailang/compile.go` after type checking, before codegen.

### Implementation Plan

#### Phase 1: Root Cause Investigation (1 hour)
- [ ] Create minimal test case that triggers `*List` generation
- [ ] Trace codegen path with `DEBUG_CODEGEN=1`
- [ ] Identify exact location where `*List` reference is created
- [ ] Document the issue in comments

#### Phase 2: Fix Type Mapping (1 hour)
- [ ] Add "List" reserved name check to mapTCon()
- [ ] Verify TList and TApp("List", T) both work
- [ ] Handle edge case: List with no type args
- [ ] Run existing tests to ensure no regression

#### Phase 3: Add Validation (1 hour)
- [ ] Create validate_types.go with ValidateTypeMapping()
- [ ] Call validation in compile pipeline
- [ ] Add error reporting that explains the issue
- [ ] Ensure error messages guide users to fix

#### Phase 4: Test Coverage (1.5 hours)
- [ ] Add test case: ADT with list field of primitives
- [ ] Add test case: ADT with list field of other ADTs
- [ ] Add test case: Nested list types
- [ ] Verify generated Go code compiles
- [ ] Add integration test in examples/

#### Phase 5: Documentation (0.5 hours)
- [ ] Update CHANGELOG.md
- [ ] Add example to docs/ showing list field ADTs
- [ ] Update limitations doc if needed

### Files to Modify

| File | Changes | Estimate |
|------|---------|----------|
| `internal/gen/golang/types.go` | Add List handling + reserved name check | 150 LOC |
| `internal/gen/golang/validate_types.go` | NEW - Type validation | 120 LOC |
| `cmd/ailang/compile.go` | Call ValidateTypeMapping() | 10 LOC |
| `internal/gen/golang/types_test.go` | Test List type mapping | 100 LOC |
| `internal/gen/golang/adt_test.go` | Test ADT with list fields | 150 LOC |
| `examples/adt_with_lists.ail` | Example showing list fields | 30 LOC |
| `CHANGELOG.md` | Document the fix | 10 LOC |

**Total new/modified**: ~570 LOC

### Examples

#### Before (Broken)
```ailang
module examples/game_state

type SolarSystem =
  { name: string
  , planets: [string]  -- List of planet names
  }

type Game =
  { systems: [SolarSystem]
  }
```

Generated Go (❌ BROKEN):
```go
type SolarSystem struct {
    Name string
    Planets *List  // ❌ undefined: List
}

type Game struct {
    Systems *List  // ❌ undefined: List
}
```

#### After (Fixed)
```go
type SolarSystem struct {
    Name string
    Planets []string  // ✅ Proper Go slice
}

type Game struct {
    Systems []*SolarSystem  // ✅ Proper Go slice
}
```

## Success Criteria

- [ ] Root cause identified and documented
- [ ] All paths through TypeMapper use consistent List → []T mapping
- [ ] New validation catches type mapping failures early
- [ ] All existing tests pass
- [ ] New test cases for ADTs with list fields all pass
- [ ] Example programs compile and run correctly
- [ ] CHANGELOG.md updated with fix description
- [ ] No performance regression in codegen

## Timeline

**Week 1** (4-6 hours total):
- **Day 1 (2h)**: Root cause investigation, minimal test case
- **Day 2 (1h)**: Type mapping fix + validation implementation
- **Day 3 (1.5h)**: Test coverage for all scenarios
- **Day 4 (0.5h)**: Documentation, examples, changelog

## Related Documents

- [M-DX25-LIST-TYPE-CODEGEN.md](../implemented/v0_6_3/m-dx25-list-type-codegen.md) - How List types are supposed to work
- [M-CODEGEN-UNIFIED-SLICE-CONVERTERS.md](../implemented/v0_6_3/m-codegen-unified-slice-converters.md) - ADT slice converter generation
- [types.go](../../../../internal/gen/golang/types.go) - Type mapper implementation
- [adt.go](../../../../internal/gen/golang/adt.go) - ADT code generation

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Regression in type mapping | Comprehensive test suite (3+ new test cases) |
| Performance impact | Validation runs only during compile, not runtime |
| Breaking change | Validation is additive (only catches missing mappings) |
| Incomplete root cause | Early minimal test case to isolate issue |

## Open Questions

1. **Why isn't existing code catching this?** The TypeMapper should already handle List types. Why does `*List` slip through?
2. **Where is the fallback happening?** Which code path bypasses the special case?
3. **Are there other generic types with similar issues?** (Option, Either, etc.)

## Implementation Notes

**Testing strategy:**
- Unit test: TypeMapper.MapType() with various TList inputs
- Integration test: ADT → Go codegen → compile with `go build`
- Example test: Run adt_with_lists.ail and verify generated Go compiles

**Validation placement:**
- After type checking passes (types are correct)
- Before codegen starts (fail fast if types unmappable)
- Not in runtime (no performance impact on execution)

**Error messages:**
Should clearly indicate:
- Which type couldn't be mapped
- Why (what was expected vs. actual)
- Hint: "Use TList(T) or [T] in AILANG for list fields"
