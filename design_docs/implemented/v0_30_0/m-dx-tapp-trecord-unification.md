# M-DX-TAPP-TRECORD: Type Inference Bug with Nested [[RecordType]]

**Status**: Resolved — ghost (verified fixed, no change needed; closed mission iteration 18, 2026-07-13)
**Priority**: High
**Source**: DX feedback from docparse-demo (Feb 2026)
**Milestone**: v0.8.0 (closed in the v0.30.0 line)

## Resolution (2026-07-13, mission iteration 18)

VERIFY-then-route reality-check: the bug **no longer reproduces at HEAD** (`v0.29.2`). A helper
`makeTable(rows: [[TableCell]]) -> Block` — the exact "fails when extracted to a function" case —
type-checks cleanly in both the modern `func` form and the doc's original
`let makeTable: [[TableCell]] -> Block = \rows. …` lambda-binding form.

The doc cited the error `cannot unify type application with *types.TRecord` from
`internal/types/unification_types.go:unifyTypeApps()`. That `%T`-formatted string was reworded to
`.String()` in commit `5cf6287bf`; the remaining fallback (`cannot unify type application %s with
%s`) fires only for genuinely-unrelated type applications, not for `[[RecordAlias]]` element
unification — `unifyTypeApps` decomposes `TApp ~ TApp` and recursively unifies element types
(alias expansion resolves the record alias before comparison). No code change was required.

CI-enforced regression guard added:

- `examples/record_list_extraction.ail` — top-level example whose `[[TableCell]]` extraction is
  type-checked (and run, printing `headers: 2`) by `verify-examples-toplevel` in CI.

*(Everything below is the original 2026-02 analysis, retained for provenance.)*

## Problem

Type inference fails when extracting a function that takes `[[RecordType]]` (a list of lists of record type aliases) as a parameter. The exact same code works when inlined at the call site but fails when extracted to a helper function.

### Minimal Reproduction

```ailang
type TableCell = { text: string, bold: bool }

-- This works (inlined):
let result = match rows with
  | firstRow :: dataRows -> TableBlock({ rows: dataRows, headers: firstRow })
  | [] -> EmptyBlock

-- This FAILS when extracted to a function:
let makeTable: [[TableCell]] -> Block = \rows.
  match rows with
  | firstRow :: dataRows -> TableBlock({ rows: dataRows, headers: firstRow })
  | [] -> EmptyBlock
```

**Error**: `cannot unify type application with *types.TRecord`

### Analysis

The bug is in `internal/types/unification_types.go:unifyTypeApps()` (line 164-223).

**Root cause**: There is **no explicit case** for `TApp ~ TRecord` unification. The function handles:
- `TApp ~ TApp` (decompose and unify)
- `TApp("list", a) ~ TList` (special case, M-DX17)
- `TApp("Array", a) ~ TArray` (special case)
- `TApp ~ TVar2` (swap and retry)
- `TApp ~ TCon` (error message)
- **DEFAULT**: Falls through to `"cannot unify type application with %T"`

When a record type alias (e.g., `TableCell = {text: string, bold: bool}`) is the element type of a nested list `[[TableCell]]`, the type system represents it as `TApp("list", TApp("list", TCon("TableCell")))`. During unification across a function boundary, one side may have the expanded `TRecord` form while the other has the `TApp` form, and there's no case to handle this mismatch.

### Why It Works Inline But Not Extracted

When code is inlined, type inference sees the concrete record literal and the pattern match in the same context. The type checker unifies the record fields directly.

When extracted to a function, the function signature introduces a type application boundary. The parameter type `[[TableCell]]` becomes `TApp("list", TApp("list", TCon("TableCell")))`, but the pattern match infers the inner type as a `TRecord`. The unifier then tries to unify `TApp(..., TRecord{...})` with `TApp(..., TCon("TableCell"))` and fails because:

1. The outer `TApp ~ TApp` decomposition works (both are lists)
2. The inner `TApp ~ TApp` decomposition works (both are lists)
3. The element-level unification tries `TCon("TableCell") ~ TRecord{...}`
4. Alias expansion should handle this, but may not fire if the aliasEnv is incomplete at this point in the function's type checking

### Proposed Fix

**Option A: Expand aliases eagerly in TApp decomposition** (Recommended)

Add alias expansion before element-type unification in `unifyTypeApps()`:

```go
// After decomposing TApp ~ TApp into constructor + args:
for i, arg := range args1 {
    arg1 := u.expandAlias(arg)
    arg2 := u.expandAlias(args2[i])
    if err := u.Unify(arg1, arg2, sub); err != nil {
        return err
    }
}
```

This ensures nested type aliases are always expanded before comparing element types.

**Option B: Add TApp ~ TRecord case**

Add explicit handling in `unifyTypeApps()` for when a TApp encounters a TRecord:

```go
case *TRecord:
    // TApp might be wrapping a record alias - expand and retry
    expanded := u.expandAlias(t1)
    if expanded != t1 {
        return u.Unify(expanded, t2, sub)
    }
    return fmt.Errorf("cannot unify type application with record type")
```

**Option C: Ensure aliasEnv is complete at function boundaries**

The real issue might be that when type-checking the extracted function, the aliasEnv doesn't include the `TableCell` alias from the current module. This would require ensuring that `expandAlias()` has access to all in-scope type aliases when checking function parameter types.

### Implementation Plan

1. Write a regression test with the minimal reproduction case
2. Add `DEBUG_MONO_VERBOSE=1` tracing to capture the exact unification failure
3. Implement Option A (most conservative, least risk of breaking other cases)
4. If Option A is insufficient, layer on Option C
5. Run full test suite + `make verify-examples`

### Files to Modify

| File | Change |
|------|--------|
| `internal/types/unification_types.go` | Add alias expansion in TApp arg unification |
| `internal/types/unification_types_test.go` | Add test for `[[RecordAlias]]` unification |
| `examples/record_list_extraction.ail` | Regression test example file |

### Risk Assessment

- **Low risk**: Option A is purely additive (extra expansion step)
- **Testing**: Existing 400+ type inference tests will catch regressions
- **Performance**: `expandAlias` is O(1) hash lookup, negligible cost

### Workaround

Until fixed, inline the pattern match at the call site instead of extracting to a helper function.
