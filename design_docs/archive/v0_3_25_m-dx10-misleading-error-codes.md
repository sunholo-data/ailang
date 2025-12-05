# M-DX10: Misleading WRONG_LANG error code for non-existent AILANG features

**Status**: Planned
**Priority**: Medium
**Milestone**: v0.3.25
**Effort**: 2 hours
**Category**: Developer Experience - Error Categorization

## Problem

**46 WRONG_LANG false positives in v0.3.24 eval caused by models using non-existent AILANG features**

Models write syntactically valid AILANG code using Haskell/ML-style features that don't exist in AILANG yet. This triggers parse errors (PAR_001) but gets miscategorized as WRONG_LANG ("wrote Python instead of AILANG").

### Evidence from v0.3.24 Eval

**Example: list_comprehension benchmark**

Generated code (valid AILANG syntax):
```ailang
module benchmark/solution

export func filter[a](pred: func(a) -> bool, xs: [a]) -> [a] {
  match xs {
    [] => [],
    Cons(x, rest) =>              # ← Cons doesn't exist in AILANG!
      if pred(x)
      then Cons(x, filter(pred, rest))
      else filter(pred, rest)
  }
}
```

Actual error:
```
PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:6:7: expected next token to be =>, got ( instead
```

Error categorization: **WRONG_LANG** ❌

Should be: **PAR_001** or new code like **FEATURE_NOT_IMPL** ✅

### Non-Existent Features Models Try to Use

**1. Cons/Nil list constructors** (67% of false positives)
```ailang
match xs {
  [] => ...
  Cons(x, rest) => ...  # AILANG doesn't have Cons/Nil pattern matching
}
```

**2. :: cons operator**
```ailang
match xs {
  [] => ...
  ::(x, rest) => ...  # AILANG doesn't have :: operator
}
```

**3. List[a] type syntax**
```ailang
export func map[a, b](f: func(a) -> b, xs: List[a]) -> List[b] { ... }
# AILANG uses [a], not List[a]
```

**4. Generic type parameters in function signatures** (complex patterns)
```ailang
export func filter[a](pred: func(a) -> bool, xs: [a]) -> [a] { ... }
# Type parameters work, but complex combinations confuse parser
```

### Why This Happens

**Root cause:** Error categorization logic (internal/eval_harness/errors.go) checks for Python/JS/C++ keywords BEFORE checking actual parse errors:

```go
// WRONG_LANG pattern (line 55)
regexp.MustCompile(`(?i)(def |class |import json|import sys|function |var |const |#include|using namespace|public static|interface |enum class)`)
```

This regex **doesn't match** Haskell/ML syntax like `Cons` or `::`, so code falls through to PAR_001 detection. BUT, the categorization function `CategorizeErrorWithCode()` is called with **both code and stderr**, and something is triggering WRONG_LANG when it shouldn't.

**Hypothesis:** There may be additional heuristics or the error flow is different than expected. Need to investigate why PAR_001 errors become WRONG_LANG.

## Impact

**Misleading metrics:**
- v0.3.24: 69 WRONG_LANG errors
  - 4 (6%) correctly detected Python/mixed syntax ✅
  - 19 (28%) stdlib bugs (see m-bug-stdlib-reserved-keyword.md) ❌
  - 46 (67%) non-existent features (this issue) ❌

**False impression:** Users see "models are writing Python" when actually models are writing AILANG with features that don't exist yet.

**Repair guidance mismatch:** Self-repair tells models "write AILANG not Python" when the real issue is "don't use Cons/Nil patterns".

## Solution Options

### Option 1: Add new error code FEATURE_NOT_IMPL (Recommended)

**New error pattern** (add to internal/eval_harness/errors.go):
```go
{
    FEATURE_NOT_IMPL,
    regexp.MustCompile(`(?i)(Cons\(|::\(|List\[|Nil\b)`),
    RepairHint{
        Title: "AILANG feature not implemented",
        Why:   "Used Haskell/ML-style list syntax (Cons/Nil/::) that AILANG doesn't support yet.",
        How:   "AILANG lists use different syntax: 1) Pattern match with [x, ...rest] NOT Cons(x, rest), 2) Use [a] NOT List[a] for types, 3) Use list literals [1,2,3] NOT Cons/Nil constructors. See AILANG syntax reference in prompt.",
    },
}
```

**Add constant:**
```go
const (
    // ... existing codes ...
    FEATURE_NOT_IMPL ErrCode = "FEATURE_NOT_IMPL" // Used non-existent Haskell/ML features
)
```

**Update categorization order:**
```go
// CRITICAL: Check order must be:
// 1. WRONG_LANG (Python/JS/C++ keywords)
// 2. FEATURE_NOT_IMPL (Haskell/ML features not in AILANG)
// 3. IMPERATIVE (loops/mutation)
// 4. PAR_001 (generic parse errors)
```

### Option 2: Improve PAR_001 repair hint

Keep PAR_001 but add specific guidance for common mistakes:
```go
RepairHint{
    Title: "Parse error",
    Why:   "AILANG syntax error - common issues: 1) Missing semicolons in blocks, 2) Wrong let/lambda/record syntax, 3) Using Haskell features like Cons/Nil (not supported)",
    How:   "Check: 1) Use `{ e1; e2; e3 }` for blocks (semicolons between exprs), 2) Use `let x = expr in body` or `let x = expr; rest`, 3) Lambda: `\\x -> body` or `func(x) { body }`, 4) Lists: use [x, ...rest] NOT Cons(x, rest), 5) No `=` in function params.",
}
```

### Option 3: Update AILANG prompt to clarify what's NOT supported

**Add to prompts/v0.3.*.md:**
```markdown
## List Pattern Matching (IMPORTANT!)

AILANG does NOT support Haskell-style Cons/Nil constructors:

❌ WRONG (Haskell/ML style - NOT supported):
```ailang
match xs {
  [] => ...
  Cons(x, rest) => ...
}
```

✅ CORRECT (AILANG syntax):
```ailang
match xs {
  [] => ...
  [x, ...rest] => ...  # Use [head, ...tail] pattern
}
```

NOTE: `::` cons operator and `List[a]` type syntax are also NOT supported.
Use `[a]` for list types and list literals for construction.
```

## Recommended Approach

**Implement all three options:**

1. **Short term (v0.3.25)**: Add FEATURE_NOT_IMPL error code (Option 1) - 1 hour
2. **Short term (v0.3.25)**: Update prompt to clarify unsupported features (Option 3) - 30 minutes
3. **Medium term (v0.4.0)**: Implement [x, ...rest] pattern matching or Cons/Nil properly - 8 hours

## Implementation Plan

### Step 1: Add FEATURE_NOT_IMPL error code (1 hour)

**Files to modify:**
1. `internal/eval_harness/errors.go`
   - Add FEATURE_NOT_IMPL constant
   - Add pattern matching rule
   - Insert before PAR_001 in Rules array

2. `internal/eval_harness/errors_test.go`
   - Add test cases for Cons/Nil/:: detection

**Test cases:**
```go
func TestFeatureNotImpl(t *testing.T) {
    code := `match xs { Cons(x, rest) => ... }`
    errCode, hint := CategorizeErrorWithCode(code, "")
    assert.Equal(t, FEATURE_NOT_IMPL, errCode)
    assert.Contains(t, hint.How, "[x, ...rest]")
}
```

### Step 2: Update prompt (30 minutes)

**Files to modify:**
1. `prompts/v0.3.25.md` (create from v0.3.23.md)
   - Add "List Pattern Matching" section
   - Add examples of wrong vs correct syntax
   - Clarify [a] vs List[a]

2. `prompts/versions.json`
   - Add v0.3.25 entry
   - Mark as active

### Step 3: Verify fix (30 minutes)

**Test affected benchmarks:**
```bash
# Should now get FEATURE_NOT_IMPL instead of WRONG_LANG
ailang eval-suite --models gpt5 --benchmarks list_comprehension,higher_order_functions --output /tmp/verify_fix

jq -r 'select(.id == "list_comprehension") | .err_code' /tmp/verify_fix/standard/*.json
# Expected: FEATURE_NOT_IMPL (not WRONG_LANG)
```

**Verify repair hint is helpful:**
```bash
# Check that repair guidance mentions [x, ...rest] syntax
jq -r 'select(.id == "list_comprehension") | .stderr' /tmp/verify_fix/standard/*.json | grep -i "rest"
```

## Success Criteria

1. ✅ WRONG_LANG only fires for actual Python/JS/C++ code (not Haskell/ML)
2. ✅ FEATURE_NOT_IMPL fires for Cons/Nil/:: syntax
3. ✅ Repair hints guide models toward correct AILANG syntax
4. ✅ Prompt clearly documents what's NOT supported
5. ✅ 46 false positives eliminated from future eval runs

## Related Issues

- See `m-bug-stdlib-reserved-keyword.md` for other cause of WRONG_LANG false positives
- This affects ~20% of AILANG eval failures (46/226)

## Future Work (v0.4.0+)

**Implement Cons/Nil pattern matching properly:**

Either add Haskell-style syntax:
```ailang
match xs {
  Nil => ...
  Cons(x, rest) => ...
}
```

Or add spread pattern matching:
```ailang
match xs {
  [] => ...
  [x, ...rest] => ...
}
```

This would eliminate the need for FEATURE_NOT_IMPL on these patterns.

## Timeline

- Error code implementation: 1 hour
- Prompt updates: 30 minutes
- Testing & verification: 30 minutes
- **Total: 2 hours**

## Notes

This issue reveals that models have good intuition for functional programming patterns (Cons/Nil is standard ML/Haskell). The error is AILANG not supporting these features yet, not models "writing wrong language".

Consider prioritizing spread pattern matching ([x, ...rest]) in v0.4.0 to reduce this friction.
