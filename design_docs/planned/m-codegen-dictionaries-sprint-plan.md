# M-CODEGEN-DICTIONARIES Sprint Plan

**Sprint Goal**: Generate type class dictionary implementations for Go codegen to fix broken `ailang compile --emit-go`

**Duration**: 1 day (~4-5 hours)
**Risk Level**: Low (well-scoped, clear implementation path)
**Priority**: High (blocks integration tests)

## Current Status

- **Design Doc**: `design_docs/planned/m-codegen-dictionaries.md` ✅
- **Problem**: Go codegen emits `dict_Num_Int.Add()` but never defines the dictionary structs
- **Impact**: `TestContractViolation_Integration` fails, `--emit-go` broken for most programs

## Milestones

### M1: Create Dictionary Generator Module (~150 LOC)

**File**: `internal/gen/golang/codegen_dictionaries.go` (NEW)

**Tasks**:
1. Create `generateDictionaries()` function that emits `dictionaries.go`
2. Implement built-in dictionaries:
   - `dict_Num_Int` (Add, Sub, Mul, Div, Neg, Abs)
   - `dict_Num_Float` (Add, Sub, Mul, Div, Neg, Abs)
   - `dict_Ord_Int` (Lt, Gt, Lte, Gte)
   - `dict_Ord_Float` (Lt, Gt, Lte, Gte)
   - `dict_Ord_String` (Lt, Gt, Lte, Gte)
   - `dict_Eq_Int` (Eq, Neq)
   - `dict_Eq_Float` (Eq, Neq)
   - `dict_Eq_String` (Eq, Neq)
   - `dict_Eq_Bool` (Eq, Neq)

**Acceptance Criteria**:
- [ ] `codegen_dictionaries.go` compiles
- [ ] `generateDictionaries()` outputs valid Go code
- [ ] All 9 dictionaries defined with correct method signatures

**Estimated LOC**: 150 (implementation)

---

### M2: Integrate with Main Codegen (~30 LOC)

**File**: `internal/gen/golang/codegen.go`

**Tasks**:
1. Call `generateDictionaries()` in the compilation output
2. Emit `dictionaries.go` to output directory alongside `runtime.go`

**Acceptance Criteria**:
- [ ] `ailang compile --emit-go` produces `dictionaries.go`
- [ ] Generated code is valid Go (compiles)

**Estimated LOC**: 30 (integration)

---

### M3: Fix Integration Tests (~20 LOC)

**File**: `internal/gen/golang/contracts_integration_test.go`

**Tasks**:
1. Verify `TestContractViolation_Integration` passes
2. Verify `TestContractViolation_NoVerify` passes
3. Add test for dictionary generation

**Acceptance Criteria**:
- [ ] `TestContractViolation_Integration` passes
- [ ] `go test ./internal/gen/golang/...` all pass

**Estimated LOC**: 20 (test fixes/additions)

---

### M4: Add Derived Eq Dictionary Generation (~80 LOC)

**Files**:
- `internal/gen/golang/codegen_dictionaries.go`
- `internal/gen/golang/codegen.go`

**Tasks**:
1. Collect ADT types with `deriving (Eq)` during compilation
2. Generate `dict_Eq_<TypeName>` for each derived type
3. Use structural comparison (tag + fields)

**Acceptance Criteria**:
- [ ] ADT with `deriving (Eq)` produces working Go code
- [ ] Generated equality uses structural comparison
- [ ] Test with examples/deriving_eq.ail compiled to Go

**Estimated LOC**: 80 (derived eq generation)

---

## Day-by-Day Plan

### Day 1 (4-5 hours total)

| Time | Task | Milestone |
|------|------|-----------|
| Hour 1 | Create `codegen_dictionaries.go` with `dict_Num_Int` | M1 |
| Hour 2 | Add remaining 8 dictionaries (Ord, Eq variants) | M1 |
| Hour 3 | Integrate with `codegen.go`, emit `dictionaries.go` | M2 |
| Hour 3.5 | Run and fix integration tests | M3 |
| Hour 4-5 | Add derived Eq dictionary generation | M4 |

## Success Metrics

1. **Tests**: `go test ./internal/gen/golang/...` passes (currently 1 failure)
2. **Integration**: `ailang compile --emit-go examples/runnable/contracts/basic.ail` produces compilable Go
3. **Feature**: Programs with operators (`+`, `-`, `*`, `/`, `<`, `>`, `==`) compile to Go

## Technical Notes

### Dictionary Structure Pattern

```go
var dict_<Class>_<Type> = struct {
    Method1 func(interface{}, interface{}) interface{}
    Method2 func(interface{}, interface{}) interface{}
}{
    Method1: func(x, y interface{}) interface{} { /* impl */ },
    Method2: func(x, y interface{}) interface{} { /* impl */ },
}
```

### Method Signatures by Class

| Class | Methods | Signature |
|-------|---------|-----------|
| Num | Add, Sub, Mul, Div | `(x, y) -> x` |
| Num | Neg, Abs | `(x) -> x` |
| Ord | Lt, Gt, Lte, Gte | `(x, y) -> bool` |
| Eq | Eq, Neq | `(x, y) -> bool` |

### Type Mappings

| AILANG Type | Go Type | Notes |
|-------------|---------|-------|
| int | int64 | Standard integer |
| float | float64 | IEEE 754 |
| string | string | UTF-8 |
| bool | bool | true/false |

## Dependencies

- None (self-contained in codegen package)

## Risks

1. **Low**: Method signature mismatch - mitigated by following existing pattern
2. **Low**: Missing dictionary type - will fail loudly at Go compile time

## Post-Sprint

After this sprint:
- `ailang compile --emit-go` works for real programs
- M-DX19 `deriving (Eq)` works in Go compilation
- Contract verification tests pass
