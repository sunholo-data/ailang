# M-TYPE-EFFECT-ROW: Fix Effect Row Unification Regression

## Status: Fixed (a314e7fc)
## Version: v0.9.4
## Priority: Bug (P1)
## Effort: Complete

## Problem

Type-checker regression in dev build: `failed to unify effect rows: incompatible closed rows`. Affects 6 modules (layout_ai, output_formatter, epub_parser) that haven't changed.

## Root Cause

Commit b29c391f ("M-TYPE-V2-MIGRATION: Delete legacy TFunc, unify on TFunc2") introduced the bug during TFunc→TFunc2 migration.

When converting unannotated function types from AST to internal representation, the code was creating `TFunc2` with `EffectRow: nil`. During unification:
- `nil` was converted to `EmptyEffectRow()` which creates a **closed empty row** `{}`
- This conflicts with functions that have effects like `{AI, IO, FS}`
- Closed row `{}` vs closed row `{AI, IO}` → "incompatible closed rows" error

### Key semantic difference:
- **`nil` effect row** = "purity sentinel" (old behavior, flexible)
- **`EmptyEffectRow()`** = "closed empty row {}" (rigid constraint)
- **Open row with tail variable** = "effects unknown, determined by context" (correct for unannotated)

## Fix Applied (a314e7fc)

In `internal/elaborate/file_funcs.go:366-379`, unannotated functions now get an **open effect row** with a fresh tail variable instead of nil:

```go
openEffectRow := &types.Row{
    Kind:   types.EffectRow,
    Labels: make(map[string]types.Type),
    Tail:   &types.RowVar{Name: fmt.Sprintf("ε_annot%d", e.freshVarNum), Kind: types.EffectRow},
}
```

This allows unification to proceed via the "both open rows" path, creating fresh variables for any effect mismatch.

## Key Files

- `internal/types/row_unification.go` — Core row unification with the error
- `internal/types/unification_types.go:52-67` — Effect row unification in function unification
- `internal/elaborate/file_funcs.go:366-379` — AST-to-internal type conversion (fix location)
- `internal/types/row_unification_regression_test.go` — Regression tests

## Origin

Agent message 043b9620 from docparse (2026-03-19)
