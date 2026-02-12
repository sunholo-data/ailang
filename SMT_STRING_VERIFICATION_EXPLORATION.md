# SMT String Verification Exploration Report

**Date**: 2026-02-12  
**Purpose**: Comprehensive analysis of current string handling in AILANG and requirements for SMT string verification (Phase B of M-SMT-FRAGMENT-EXPANSION)

---

## Executive Summary

AILANG has **18 string builtins** with comprehensive UTF-8 support, but the SMT verification layer currently **rejects all string operations**. To enable Phase B (String Verification) of the SMT expansion roadmap, we need to:

1. **Type mapping**: `string` → SMT `String` sort
2. **Builtin encoding**: Map AILANG string operations to Z3 string theory (`str.*` functions)
3. **Encodable fragment check**: Remove string rejection from `hasUnencodableTypes()`
4. **Intrinsic handling**: Add `OpConcat` support to SMT codegen

**Estimated effort**: 10-12 hours of implementation  
**Risk**: Low (Z3 string theory is mature, similar to Phase C records which is already implemented)

---

## 1. String Builtins Inventory

### 1.1 Core String Operations (Already Implemented)

Located in `/Users/mark/dev/sunholo/ailang/internal/builtins/string.go`:

| Builtin Name | Type Signature | Implementation | SMT Equivalent | Priority |
|--------------|---|---|---|---|
| `_str_len` | `string → int` | UTF-8 rune count | `(str.len s)` | **HIGH** |
| `_str_compare` | `string → string → int` | Lexicographic ordering (-1/0/1) | `(str.< a b)` / `(str.> a b)` / `(= a b)` | **HIGH** |
| `_str_eq` | `string → string → bool` | String equality | `(= a b)` | **HIGH** |
| `concat_String` | `string → string → string` | String concatenation | `(str.++ a b)` | **HIGH** |
| `_str_find` | `string → string → int` | Find substring index | `(str.indexof haystack needle)` | **MEDIUM** |
| `_str_slice` | `string → int → int → string` | Extract substring by range | `(str.substr s start len)` | **MEDIUM** |
| `_str_trim` | `string → string` | Strip whitespace | **Cannot encode** (no Z3 op for general trim) | **LOW** |
| `_str_upper` | `string → string` | Uppercase conversion | **Cannot encode** (no Z3 builtin) | **LOW** |
| `_str_lower` | `string → string` | Lowercase conversion | **Cannot encode** (no Z3 builtin) | **LOW** |
| `_str_split` | `string → string → [string]` | Split by delimiter | **Cannot encode** (returns list) | **SKIP** |
| `_string_reverse` | `string → string` | Reverse string | **Cannot encode** (no Z3 builtin) | **LOW** |
| `_str_chars` | `string → [string]` | String to char list | **Cannot encode** (returns list) | **SKIP** |
| `_str_startsWith` | `string → string → bool` | Check prefix | `(str.prefixof prefix s)` | **MEDIUM** |
| `_str_endsWith` | `string → string → bool` | Check suffix | `(str.suffixof suffix s)` | **MEDIUM** |
| `_stringToInt` | `string → Option[int]` | Parse string to int | **Cannot encode** (returns Option) | **SKIP** |
| `_stringToFloat` | `string → Option[float]` | Parse string to float | **Cannot encode** (returns Option) | **SKIP** |

### 1.2 Grouping by Encodability in Phase B

**Phase B Scope (10-12h):**
- **HIGH priority** (5 ops): `_str_len`, `_str_compare`, `_str_eq`, `concat_String`, `_str_find`
- **MEDIUM priority** (4 ops): `_str_slice`, `_str_startsWith`, `_str_endsWith` 
- Total: 8 operations covering ~90% of string contract use cases

**Not in Scope (Phase D+ or unfeasible):**
- Operations returning lists (`_str_split`, `_str_chars`)
- Operations returning ADT types (`_stringToInt`, `_stringToFloat`)
- Case conversion (`_str_upper`, `_str_lower`) — no Z3 builtin
- General trimming (`_str_trim`) — no Z3 builtin for whitespace patterns
- Reversal (`_string_reverse`) — feasible but low priority

---

## 2. Current SMT String Rejection Mechanism

### 2.1 Rejection Points in Codebase

**File**: `/Users/mark/dev/sunholo/ailang/internal/smt/encodable.go`

#### Point 1: Type-Level Rejection (Line 406)
```go
case *core.Lit:
    return e.Kind == core.StringLit  // ← String literals rejected
```
**Action**: Remove this check or make it conditional based on fragment mode.

#### Point 2: Intrinsic Rejection (Line 473-474)
```go
case *core.Intrinsic:
    if e.Op == core.OpConcat {
        return true  // ← String concatenation (OpConcat) rejected
    }
```
**Action**: Change `OpConcat` from rejection to conditional encoding check.

#### Point 3: Builtin Function Rejection (Line 434-435)
```go
case *core.VarGlobal:
    if e.Ref.Module == "$builtin" {
        return isStringOrListBuiltin(e.Ref.Name)  // ← Rejects all string builtins
    }
```

**Current Implementation** (Lines 489-499):
```go
func isStringOrListBuiltin(name string) bool {
    for _, suffix := range []string{"_String", "_List"} {
        if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
            return true
        }
    }
    return false
}
```

**Problem**: This checks for `_String` suffix, but AILANG string builtins use `_str_*` prefix or `concat_String` pattern.

**Actual rejection pattern**: Catches:
- `concat_String` (contains `_String`)
- `eq_String` (contains `_String`)
- `lt_String` (contains `_String`)
- `ne_String` (contains `_String`)

But MISSES:
- `_str_len`, `_str_compare`, `_str_find`, etc. (use `_str_` prefix)

### 2.2 Current Behavior Example

Test case scenario:
```ailang
export func validatePrefix(s: string, prefix: string) -> bool ! {}
requires { _str_len(prefix) > 0 }
ensures { result == _str_startsWith(s, prefix) }
{
  _str_startsWith(s, prefix)
}
```

**Current outcome**: 
- `_str_len`, `_str_startsWith` would pass (not caught by suffix check)
- But then encoding fails in `codegen.go:323` when encountering `StringLit` case
- Overall: **Function rejected as "UNENCODABLE_TYPE"**

---

## 3. Core AST String Representation

### 3.1 String Literal in Core

**File**: `/Users/mark/dev/sunholo/ailang/internal/core/core.go` (Line 72-80)

```go
type LitKind int

const (
    IntLit LitKind = iota
    FloatLit
    StringLit        // ← String literals represented as Lit{Kind: StringLit, Value: string}
    BoolLit
    UnitLit
)
```

**Usage**: `&core.Lit{Kind: core.StringLit, Value: "hello"}`

**Current encoding** (codegen.go:323-324):
```go
case core.StringLit:
    return "", fmt.Errorf("string literals cannot be encoded in SMT-LIB")
```

### 3.2 String Concatenation in Core

**Pre-lowering** (using BinOp or Intrinsic):
- Surface: `"hello" ++ " world"`
- Elaborated to: `&core.Intrinsic{Op: core.OpConcat, Args: [lhs, rhs]}`

**Post-lowering** (after op_lowering):
- Converted to: `&core.App{Func: concat_String_builtin, Args: [lhs, rhs]}`

**In SMT codegen** (Line 473-474):
```go
case *core.Intrinsic:
    if e.Op == core.OpConcat {
        return true  // ← Rejected pre-lowering
    }
```

---

## 4. SMT Type Mapping Required

### 4.1 Current Type Mapping

**File**: `/Users/mark/dev/sunholo/ailang/internal/smt/types.go` (Line 35-80)

```go
func MapType(t types.Type) (string, error) {
    switch ty := t.(type) {
    case *types.TCon:
        return mapTCon(ty.Name)
    case *types.TVar:
        return "", fmt.Errorf("type variable %q cannot be encoded...", ty.Name)
    case *types.TFunc:
        return "", fmt.Errorf("function types cannot be encoded...")
    case *types.TList:
        return "", fmt.Errorf("list types cannot be encoded...")
    case *types.TRecord:
        return MapRecordSortName(ty), nil  // ← Records ARE supported (Phase C done)
    ...
    }
}

func mapTCon(name string) (string, error) {
    switch name {
    case "int":
        return "Int", nil
    case "float":
        return "Real", nil
    case "bool":
        return "Bool", nil
    case "string":
        return "", fmt.Errorf("string type cannot be encoded in SMT-LIB")  // ← Current rejection
    case "unit":
        return "", fmt.Errorf("unit type cannot be encoded in SMT-LIB")
    ...
    }
}
```

### 4.2 Required Changes for Phase B

**Change 1**: String type mapping
```go
case "string":
    return "String", nil  // ← Z3 has a built-in String sort
```

**Change 2**: List type already correctly rejects (Phase D will handle)
```go
case *types.TList:
    return "", fmt.Errorf("list types cannot be encoded...")
```

---

## 5. Builtin-to-SMT Operation Mapping

### 5.1 Current Mapping

**File**: `/Users/mark/dev/sunholo/ailang/internal/smt/types.go` (Line 236-276)

```go
var BuiltinToSMTOp = map[string]string{
    // Arithmetic
    "add_Int":   "+",
    "add_Float": "+",
    ...
    // Comparison
    "eq_Int":    "=",
    "eq_Float":  "=",
    "eq_Bool":   "=",
    ...
    // Boolean
    "and_Bool":  "and",
    "or_Bool":   "or",
    "not_Bool":  "not",
    ...
}
```

**Note**: No string operations in current map!

### 5.2 Phase B Additions Needed

```go
var BuiltinToSMTOp = map[string]string{
    // ... existing ...
    
    // String operations (NEW)
    "concat_String":   "str.++",      // Binary concat
    "length_String":   "str.len",     // Unary length
    "eq_String":       "=",           // Use standard equality for strings
    "lt_String":       "str.<",       // Lexicographic less-than
    "le_String":       "str.<=",      // Lexicographic less-than-or-equal (if available)
    "gt_String":       "str.>",       // Lexicographic greater-than
    "ge_String":       "str.>=",      // Lexicographic greater-than-or-equal
    "ne_String":       "distinct",    // String inequality
    "indexof_String":  "str.indexof", // Find substring (3 args: haystack, needle, returns int or -1)
    "prefixof_String": "str.prefixof", // Check prefix (2 args: prefix, string, returns bool)
    "suffixof_String": "str.suffixof", // Check suffix (2 args: suffix, string, returns bool)
    "substr_String":   "str.substr",  // Substring (3 args: string, start, length)
}
```

**Note on arity**: Some Z3 operations have different arities:
- `str.indexof` in SMT-LIB is ternary: `(str.indexof haystack needle [start-pos])`
- AILANG `_str_find` is binary: `_str_find(haystack, needle)` returns index or -1

---

## 6. Expression Encoding in SMT Codegen

### 6.1 Current Literal Encoding

**File**: `/Users/mark/dev/sunholo/ailang/internal/smt/codegen.go` (Line 292-328)

```go
func encodeLit(lit *core.Lit) (string, error) {
    switch lit.Kind {
    case core.IntLit:
        v, ok := lit.Value.(int64)
        if !ok {
            return "", fmt.Errorf("IntLit with non-int64 value: %T", lit.Value)
        }
        if v < 0 {
            return fmt.Sprintf("(- %d)", -v), nil
        }
        return fmt.Sprintf("%d", v), nil
    case core.FloatLit:
        // ... similar ...
    case core.BoolLit:
        // ... similar ...
    case core.UnitLit:
        return "", fmt.Errorf("unit literals cannot be encoded in SMT-LIB")
    case core.StringLit:
        return "", fmt.Errorf("string literals cannot be encoded in SMT-LIB")  // ← CHANGE THIS
    ...
    }
}
```

### 6.2 Phase B Changes to encodeLit

```go
case core.StringLit:
    v, ok := lit.Value.(string)
    if !ok {
        return "", fmt.Errorf("StringLit with non-string value: %T", lit.Value)
    }
    // Z3 string literals: "hello" becomes "hello" (same format)
    // But need to escape internal quotes
    escaped := strings.ReplaceAll(v, "\"", "\\\"")
    return fmt.Sprintf("\"%s\"", escaped), nil
```

### 6.3 Intrinsic Encoding for OpConcat

**Current** (Line 244-246):
```go
case *core.Intrinsic:
    // Pre-lowered intrinsic (shouldn't appear after op_lowering, but handle gracefully)
    return encodeIntrinsic(e)
```

**Helper** `encodeIntrinsic()` (need to find current implementation):
- Likely rejects all Intrinsics before lowering

**Phase B Addition**:
```go
func encodeIntrinsic(e *core.Intrinsic) (string, error) {
    if e.Op == core.OpConcat {
        if len(e.Args) != 2 {
            return "", fmt.Errorf("OpConcat expects 2 args, got %d", len(e.Args))
        }
        left, err := EncodeExpr(e.Args[0])
        if err != nil {
            return "", err
        }
        right, err := EncodeExpr(e.Args[1])
        if err != nil {
            return "", err
        }
        return fmt.Sprintf("(str.++ %s %s)", left, right), nil
    }
    // ... other intrinsics ...
    return "", fmt.Errorf("unsupported intrinsic: %v", e.Op)
}
```

---

## 7. Implementation Checklist for Phase B

### 7.1 Type Mapping (30 minutes)

**File**: `internal/smt/types.go`

- [ ] Line 72-75: Change `mapTCon("string")` case to return `"String"` instead of error
- [ ] Add test: `TestMapType_StringSort()` in `types_test.go`
- [ ] Add test: `TestMapType_StringSortError()` (verify other string types still fail if any)
- [ ] Verify existing record mapping tests still pass

**LOC**: ~5 lines of code + ~30 lines of tests

### 7.2 Builtin Operation Mapping (1 hour)

**File**: `internal/smt/types.go`

- [ ] Line 236-276: Add 8 new entries to `BuiltinToSMTOp` map:
  - `concat_String: "str.++"`
  - `length_String: "str.len"` (need special handling for unary)
  - `eq_String: "="`
  - `lt_String: "str.<"`
  - `le_String: "str.<="`
  - `gt_String: "str.>"`
  - `ge_String: "str.>="`
  - `ne_String: "distinct"`
  - `indexof_String: "str.indexof"` (needs arity adjustment)
  - `prefixof_String: "str.prefixof"` (boolean result)
  - `suffixof_String: "str.suffixof"` (boolean result)
  - `substr_String: "str.substr"` (3-arg version)

**Note**: Some operations like `length_String` may need special handling in `encodeBuiltinOp()` because they're unary while appearing as method calls in resolved callees.

**LOC**: ~15 lines for map + ~60 lines of tests

### 7.3 Encodable Fragment Check (2 hours)

**File**: `internal/smt/encodable.go`

- [ ] Line 406: Remove or modify string literal rejection
  ```go
  case *core.Lit:
      // Return false (unencodable) only for StringLit in specific contexts
      // For now: StringLit IS encodable → don't reject
      return false  // Was: return e.Kind == core.StringLit
  ```
  
- [ ] Line 433-435: Add string builtin allowlist
  ```go
  case *core.VarGlobal:
      if e.Ref.Module == "$builtin" {
          // Allow string operations in Phase B
          if isEncodableStringBuiltin(e.Ref.Name) {
              return false  // Not unencodable
          }
          // Still reject list builtins, etc.
          return isStringOrListBuiltin(e.Ref.Name)
      }
  ```

- [ ] Line 489-499: Update `isStringOrListBuiltin()` to only check List operations
  ```go
  func isStringOrListBuiltin(name string) bool {
      // Only reject list builtins now; string builtins have separate handling
      for _, suffix := range []string{"_List"} {
          if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
              return true
          }
      }
      return false
  }
  
  func isEncodableStringBuiltin(name string) bool {
      // Phase B: These 8 builtins are now encodable
      encodable := map[string]bool{
          "_str_len": true,
          "_str_compare": true,
          "_str_eq": true,
          "concat_String": true,
          "_str_find": true,
          "_str_slice": true,
          "_str_startsWith": true,
          "_str_endsWith": true,
      }
      return encodable[name]
  }
  ```

- [ ] Line 473-474: Modify Intrinsic handling
  ```go
  case *core.Intrinsic:
      if e.Op == core.OpConcat {
          // OpConcat (string concatenation) IS now encodable
          return false  // Don't reject
      }
      // All other Intrinsics still rejected
      ...
  ```

**LOC**: ~40 lines of code + ~80 lines of tests

### 7.4 Expression Encoding in Codegen (3 hours)

**File**: `internal/smt/codegen.go`

- [ ] Line 323-324: Fix string literal encoding
  ```go
  case core.StringLit:
      v, ok := lit.Value.(string)
      if !ok {
          return "", fmt.Errorf("StringLit with non-string value: %T", lit.Value)
      }
      escaped := escapeStringForSMT(v)
      return fmt.Sprintf("\"%s\"", escaped), nil
  ```

- [ ] Add helper function: `escapeStringForSMT(s string) string`
  - Escape double quotes: `"` → `\"`
  - Escape backslashes: `\` → `\\`
  - Handle other special chars as needed

- [ ] Line 244-246: Ensure `encodeIntrinsic()` handles `OpConcat`
  - Add case for `core.OpConcat` with 2 args
  - Encode as `(str.++ left right)`

- [ ] Line 382-405: `encodeBuiltinOp()` handling for variable-arity operations
  - `str.len` is unary: `(str.len x)`
  - `str.++` is binary: `(str.++ x y)`
  - `str.substr` is ternary: `(str.substr s start len)`
  - Verify the current binary/unary dispatch handles these correctly

**LOC**: ~60 lines of code + ~100 lines of tests

### 7.5 Update Rejection Diagnostics (1 hour)

**File**: `internal/smt/encodable.go`

- [ ] Update `SMTRejectionReason` hint text in `IsSMTEncodable()` (Line 95-96)
  - Old: `"Use int, float, bool, or enum ADTs"`
  - New: `"Use int, float, bool, enum ADTs, or simple string operations"`

- [ ] Create detailed documentation of what IS/ISN'T encodable for strings:
  - ✅ String literals `"hello"`
  - ✅ `_str_len(s)`, `_str_compare(a,b)`, `_str_eq(a,b)`, `concat_String(a,b)`
  - ✅ `_str_find()`, `_str_slice()`, `_str_startsWith()`, `_str_endsWith()`
  - ❌ `_str_split()` (returns list)
  - ❌ `_str_upper()`, `_str_lower()` (no Z3 builtin)
  - ❌ `_stringToInt()`, `_stringToFloat()` (returns Option ADT)

**LOC**: ~15 lines code + ~30 lines docs

### 7.6 Testing (4-5 hours)

**Files to create/modify**:

1. **Type mapping tests** (`types_test.go`):
   - `TestMapType_StringSort()` — verify `string` → `String`
   - `TestMapRecordFields_WithStringFields()` — records containing strings
   - `TestMapRecordFields_StringFieldsRejected()` — ensure rejection only in appropriate contexts

2. **Builtin operation tests** (`types_test.go`):
   - For each of 8 string builtins, verify mapping exists
   - `TestBuiltinToSMTOp_StringOps()` — all 8 operations map correctly

3. **Encodable fragment tests** (`encodable_test.go`):
   - `TestHasUnencodableTypes_StringLiteralNowEncodable()`
   - `TestHasUnencodableTypes_StringBuiltinsEncodable()`
   - `TestHasUnencodableTypes_StringBuiltinsNotInList()`
   - `TestHasUnencodableTypes_OpConcatNowEncodable()`
   - `TestHasUnencodableTypes_StringSplitStillRejected()` (returns list)
   - `TestHasUnencodableTypes_StringToIntStillRejected()` (returns Option)

4. **Codegen tests** (`codegen_test.go`):
   - `TestEncodeLit_String()` — encode string literal to SMT format
   - `TestEncodeLit_StringWithQuotes()` — escape internal quotes
   - `TestEncodeLit_StringWithBackslashes()` — escape backslashes
   - `TestEncodeExpr_StringConcat()` — encode `OpConcat` as `str.++`
   - `TestEncodeExpr_StringLen()` — encode `_str_len` call
   - `TestEncodeExpr_StringEq()` — encode `_str_eq` call
   - `TestEncodeExpr_StringComparison()` — encode `_str_compare`
   - `TestEncodeExpr_StringFind()` — encode `_str_find`
   - `TestEncodeExpr_StringSlice()` — encode `_str_slice`
   - `TestEncodeExpr_StringPrefixSuffix()` — encode `_str_startsWith`, `_str_endsWith`

5. **Integration tests** (new file: `string_verify_test.go`):
   ```go
   func TestVerifyStringContracts(t *testing.T) {
       // Test function with string contracts
       funcName := "formatCode"
       body := &core.App{...}  // concat_String("prefix", intToString(42))
       meta := &core.DeclMeta{
           IsPure: true,
           Contracts: []*core.Contract{
               {
                   Kind: core.RequiresKind,
                   Expr: &core.App{...}, // contract expr
               },
           },
       }
       
       // Should encode without errors
       result, err := EncodeFunction(funcName, params, body, "String", meta, nil)
       if err != nil {
           t.Fatalf("failed to encode: %v", err)
       }
       // Verify SMT-LIB contains string operations
       if !strings.Contains(result.SMTLib, "str.++") {
           t.Error("expected str.++ in encoded SMT")
       }
   }
   ```

**Total test LOC**: ~150-200 lines

---

## 8. Gap Analysis: Current vs. Phase B

### 8.1 What's Already Working (from Phase A/C investigation)

✅ **Phase A (Cross-Function Calls)**: Already implemented!
- `internal/smt/callee_resolver.go` (443 LOC) ✅
- Can resolve and inline user-defined function calls
- Detects circular dependencies

✅ **Phase C (Records)**: Already implemented!
- Record types map to `declare-datatype` with accessors
- Record construction and field access encoded
- Record updates encoded as constructor calls

### 8.2 What Needs Phase B Implementation

❌ **String Literals**: Currently rejected at line 324 of `codegen.go`
❌ **String Builtins**: Currently checked by incorrect suffix pattern (line 434)
❌ **OpConcat**: Currently rejected at line 473 of `encodable.go`
❌ **String Type Mapping**: Currently returns error at line 72-75 of `types.go`
❌ **String Operation Mapping**: Only has arithmetic/comparison ops, no string ops (line 236-276 of `types.go`)

### 8.3 Phases D & E Status

⏳ **Phase D (Lists)**: Not started
⏳ **Phase E (Bounded Recursion)**: Not started

---

## 9. Risk Assessment

### 9.1 Technical Risks

| Risk | Likelihood | Severity | Mitigation |
|------|-----------|----------|-----------|
| Z3 string theory performance | Medium | Medium | Add timeouts; skip slow functions |
| Incorrect escape handling | Medium | Low | Comprehensive string literal tests |
| `str.indexof` return type mismatch | Low | Medium | Use `(ite (< idx 0) -1 idx)` wrapper if needed |
| String operations in nested contexts | Low | Low | Recursive encoding handles naturally |
| Interaction with Phase D list encoding | Low | Low | List rejection stays in place; no conflict |

### 9.2 Scope Creep Risks

**Tempting additions to avoid**:
- ❌ Regex matching (`str.in_re`) — adds complexity, low priority
- ❌ Case conversion via SMT — no builtin, would need multiple helper operations
- ❌ Encoding list-returning operations — defer to Phase D
- ❌ Parsing operations (`_stringToInt`) — returns Option, needs ADT integration

**Mitigation**: Stick strictly to the 8 operations listed in Section 7.2.

### 9.3 Testing Risk

**If testing is insufficient**: Silent failures where correct contracts are marked as "unencodable" or where malformed SMT is sent to Z3.

**Mitigation**: 
- Comprehensive unit tests for each builtin
- Integration tests with actual Z3 solver (Z3-gated tests)
- Regression tests ensure Phase A/C still work

---

## 10. Acceptance Criteria for Sprint

### 10.1 Functional Criteria

- [ ] String literals encode to SMT-LIB string format
- [ ] 8 string builtins map correctly to Z3 string operations
- [ ] `OpConcat` pre-lowering encodes as `str.++`
- [ ] String type maps to SMT `String` sort
- [ ] Fragment checker accepts simple string contracts
- [ ] No existing `ailang verify` tests regress
- [ ] Example contract with string operations verifies with Z3

### 10.2 Code Quality Criteria

- [ ] All new functions have complete docstrings
- [ ] Test coverage ≥ 90% for new code
- [ ] `golangci-lint` passes with no errors
- [ ] Inline TODOs only for Phase D+ work

### 10.3 Documentation Criteria

- [ ] Update `design_docs/planned/v0_8_0/m-smt-fragment-expansion.md` mark Phase B complete
- [ ] Add section to `docs/docs/guides/contracts.mdx` showing string contract examples
- [ ] Create example file `examples/runnable/contracts/string_verify.ail` with working string contracts
- [ ] Update rejection messages to reflect new capabilities

---

## 11. Sprint Dependencies & Scheduling

### 11.1 Hard Dependencies

✅ Already satisfied:
- M-SMT-BACKEND (v0.7.4) — Z3 integration ✅
- M-CONTRACTS-OPLOWERING (v0.7.4) — contracts work post-lowering ✅
- Phase A (Cross-Function Calls) — resolved callees work ✅
- Phase C (Records) — similar encoding patterns established ✅

### 11.2 Soft Dependencies

- Phase B must complete BEFORE Phase D starts
- Phase B changes to `encodable.go` may need adjustment when Phase D adds list rejection logic
- Recommend completing Phase B and merging to `dev` before starting Phase D

### 11.3 Estimated Timeline

- **Setup & type mapping**: 2 hours
- **Builtin operation mapping**: 2 hours
- **Encodable fragment updates**: 2 hours
- **Codegen literal + intrinsic**: 2 hours
- **Testing & integration**: 4-5 hours
- **Documentation & examples**: 1 hour
- **Buffer & debugging**: 1-2 hours

**Total**: 14-16 hours (slightly more than initial 10-12h estimate due to testing depth)

---

## 12. Key Implementation Files Summary

| File | Current LOC | Phase B Changes | Impact |
|------|---|---|---|
| `internal/smt/types.go` | 276 | +120 | Type mapping, builtin-to-SMT ops |
| `internal/smt/codegen.go` | 862 | +150 | String literal encoding, intrinsic handling |
| `internal/smt/encodable.go` | 499 | +80 | Remove string rejection, add allowlist |
| `internal/smt/codegen_test.go` | 1042 | +200 | Comprehensive string encoding tests |
| `internal/smt/encodable_test.go` | 518 | +100 | String fragment check tests |
| `internal/smt/types_test.go` | 400 | +80 | Type mapping tests |
| New: `internal/smt/string_verify_test.go` | 0 | +200 | Integration tests with Z3 |

**Total new/modified**: ~930 LOC across 8 files

---

## 13. Recommendations

### 13.1 Implementation Order

1. **Start with Phase B types (30 min)** — low risk, quick win
2. **Add builtin mapping (1 hr)** — prerequisite for testing
3. **Update encodable checks (2 hrs)** — core logic change
4. **Implement codegen changes (2 hrs)** — directly enable encoding
5. **Comprehensive testing (4-5 hrs)** — catch issues early
6. **Integration & Z3 testing (2 hrs)** — validate end-to-end

### 13.2 Before Starting Phase B

- ✅ Verify Phase A & C tests pass with current `dev` branch
- ✅ Review `callee_resolver.go` to understand existing cross-function call pattern
- ✅ Review `codegen.go` record encoding (Phase C) to match patterns
- ✅ Set up Z3-gated integration test infrastructure if not already present

### 13.3 After Completing Phase B

- Schedule Phase D (Lists) planning for ~8-10 hours after Phase B ships
- Consider parallel work on documentation/examples during Phase D sprint planning
- Monitor Z3 performance with string operations in real codebases

---

## 14. References

**Codebase Locations**:
- String builtins: `/Users/mark/dev/sunholo/ailang/internal/builtins/string.go` (970 LOC)
- SMT encoding: `/Users/mark/dev/sunholo/ailang/internal/smt/` (4,994 LOC total)
- Current design doc: `/Users/mark/dev/sunholo/ailang/design_docs/planned/v0_8_0/m-smt-fragment-expansion.md`
- Phase A (cross-function): `/Users/mark/dev/sunholo/ailang/internal/smt/callee_resolver.go`
- Phase C (records): `/Users/mark/dev/sunholo/ailang/internal/smt/types.go` (MapRecordSortName, etc.)

**External References**:
- [Z3 String Theory Docs](https://z3prover.github.io/api/html/group__z3__api.html)
- SMT-LIB string theory: `(str.++ "hello" "world")`, `(str.len "hello")`, etc.

---

**Report Generated**: 2026-02-12  
**Explorer**: Claude Code Exploration Agent  
**Status**: ✅ Ready for Phase B Sprint Planning
