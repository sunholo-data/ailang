# M-PROMPT: Module & Syntax Clarity Improvements

**Status**: Planned
**Target**: v0.4.3
**Priority**: P0 (Critical - addresses 21% of eval failures!)
**Estimated**: 0.5 days
**Based on**: v0.4.2 eval analysis (MOD_001: 64 failures, PAR_001: 36 failures)

## Problem Statement

**v0.4.2 Evaluation Results:**
- **MOD_001 errors**: 64 failures (21% of all AILANG failures!)
- **PAR_001 errors**: 36 failures (12% - mostly infix cons syntax)
- **Root cause**: Teaching prompt has gaps and inconsistencies

**Critical Issues Found:**

### 1. Comment Syntax Unclear
**Issue**: Models don't know AILANG supports `--` comments

**Evidence**:
```ailang
import std/io (println)  -- This comment WORKS but models don't know!
```

**Current prompt**: No mention of comment syntax at all!

### 2. Benchmark Hints Have Wrong Syntax
**Issue**: Benchmark files contain incorrect AILANG syntax in their hints

**Example** (config_file_parser.yml):
```ailang
# ❌ WRONG syntax in benchmark hint:
func loadConfig(filename: string) !: FS -> Config =

# ✅ CORRECT syntax:
export func loadConfig(filename: string) -> Config ! {FS} =
```

**Impact**: Models learn WRONG syntax from benchmark hints!

### 3. Effect Syntax Confusion
**Issue**: Multiple ways to write effects shown in examples

**Inconsistent examples found**:
- `! {IO, FS}` ✅ CORRECT
- `!: IO, FS` ❌ WRONG (shown in config_file_parser hint)
- `! IO` ❌ WRONG

### 4. List Cons Syntax Gap
**Issue**: Prompt warns against `::` but doesn't explain the CURRENT syntax

**Current prompt**:
```
PAR_NO_PREFIX_PARSE at benchmark/solution.ail:36:29: unexpected token in expression: ::
```

**Missing**: Clear explanation that `::` is NOT supported yet, use `::(h, t)` prefix form

### 5. `let` Binding Confusion
**Issue**: Models try to use bare assignments like Python/JavaScript

**Common error**:
```ailang
countHelper = \acc. \xs. ...  -- ❌ PAR015: bare assignment not supported
```

**Should be**:
```ailang
let countHelper = \acc. \xs. ... in  -- ✅ Requires 'let' and 'in'
```

## Solution: Prompt v0.4.3 Improvements

### 1. Add Comment Syntax Section

**New section to add after line 19**:

```markdown
## Comments

AILANG supports both Haskell-style and C-style comments:

```ailang
-- Single-line comment (Haskell style)
// Single-line comment (C style)

let x = 5 in  -- inline comment
  x * 2       // also inline
```

**Both styles work identically. Use whichever is more familiar.**
```

### 2. Clarify Effect Syntax

**Replace current effect examples with consistent syntax**:

**In Quick Reference (line 13):**
```markdown
- Effect: `! {IO, FS, Net}` after return type (SPACE before !)
- ❌ WRONG: `!: IO`, `! IO` (parse errors)
- ✅ CORRECT: `! {IO, FS}` (braces required, comma-separated)
```

**New complete example**:
```ailang
-- Pure function (no effects)
export func computeSum(a: int, b: int) -> int {
  a + b
}

-- IO effect (prints to console)
export func printSum(a: int, b: int) -> () ! {IO} {
  let sum = computeSum(a, b);
  print("Sum: " ++ show(sum))
}

-- Multiple effects (IO and FS)
export func logToFile(filename: string, message: string) -> () ! {IO, FS} {
  writeFile(filename, message);
  print("Logged to file")
}

-- Main with composed effects
export func main() -> () ! {IO, FS} {
  printSum(10, 20);
  logToFile("output.txt", "Sum was 30");
  print("Done")
}
```

### 3. Expand List Cons Section

**Add prominent warning in Quick Reference**:

```markdown
- List cons: `::(head, tail)` (PREFIX notation, space around ::)
- ❌ **NOT SUPPORTED YET**: `x :: xs` (infix cons coming in v0.5.0!)
- ✅ CORRECT: `::(x, xs)` or use list literal `[x]` then append
```

**In List Operations section (around line 328):**

```ailang
-- ✅ CORRECT - Pattern matching with prefix cons
export func sum(xs: List[int]) -> int =
  match xs {
    [] => 0,
    ::(x, rest) => x + sum(rest)  -- Note: ::(x, rest) is prefix!
  }

-- ❌ WRONG - Infix cons not supported yet
export func sum(xs: List[int]) -> int =
  match xs {
    [] => 0,
    x :: rest => x + sum(rest)  -- Parse error! Use ::(x, rest) instead
  }

-- ✅ CORRECT - Building lists with prefix cons
let list1 = ::(1, ::(2, ::(3, []))) in  -- Explicit prefix
let list2 = [1, 2, 3] in                -- Sugar (recommended!)
  list1 == list2  -- true
```

### 4. Emphasize `let` Bindings

**Add prominent section after "Critical Limitations" (line 154)**:

```markdown
## Critical: `let` Bindings Are Required

**AILANG has NO bare assignments like Python/JavaScript:**

```ailang
-- ❌ WRONG - Bare assignment (PAR015 error)
x = 10;
y = x + 1;
print(show(y))

-- ❌ WRONG - Function definition without let
countHelper = \acc. \xs. match xs { ... }

-- ✅ CORRECT - Use let...in for all bindings
let x = 10 in
let y = x + 1 in
  print(show(y))

-- ✅ CORRECT - Multiple bindings with semicolons
export func main() -> () ! {IO} {
  let x = 10;
  let y = x + 1;
  print(show(y))
}

-- ✅ CORRECT - Lambda bindings also need let
export func countHelper() -> (int -> List[int] -> int) {
  let helper = \acc. \xs.
    match xs {
      [] => acc,
      ::(h, t) => helper(acc + h)(t)
    } in
  helper
}
```

**Key rules:**
1. EVERY binding needs `let` keyword
2. In expressions: use `let x = val in body`
3. In blocks: use `let x = val;` then continue
4. NO bare assignments - not Python!
```

### 5. Add Module Declaration Checklist

**New section after "MANDATORY Structure" (line 93)**:

```markdown
## Module Declaration Checklist

**Before writing AILANG code, verify:**

✅ First line: `module benchmark/solution` (NO quotes!)
✅ Imports use NO quotes: `import std/io (println)`
✅ Effect syntax: `! {IO, FS}` (braces required, AFTER return type)
✅ Comments: Both `--` and `//` work
✅ Zero-arg calls: `f ()` (space required) NOT `f()`
✅ List cons: `::(h, t)` (prefix) NOT `h :: t` (infix not supported yet)
✅ Bindings: `let x = val in` or `let x = val;` NEVER bare `x = val`

**Common mistakes that cause MOD_001 or PAR_001 errors:**
- ❌ `module "benchmark/solution"` (quotes not allowed)
- ❌ `import "std/io"` (quotes not allowed)
- ❌ `func f(x: int) !: IO -> ()` (wrong effect syntax)
- ❌ `x = 10` (missing `let` keyword)
- ❌ `x :: xs` (infix cons not supported yet)
- ❌ `f()` (missing space, use `f ()`)
```

### 6. Fix Benchmark Hints

**Action items**:
1. Audit ALL benchmark .yml files for syntax errors
2. Update config_file_parser.yml effect syntax: `!: FS` → `! {FS}`
3. Add AILANG-specific hints to effect_tracking_io_fs.yml
4. Ensure all hints use CURRENT syntax (not future features)

**Example fix for effect_tracking_io_fs.yml**:

```yaml
task_prompt: |
  ...

  <LANG=AILANG>
  Write to: benchmark/solution.ail

  **AILANG-specific syntax:**
  ```ailang
  module benchmark/solution

  -- Pure function (no effect annotation)
  export func computeSum(a: int, b: int) -> int {
    a + b
  }

  -- IO effect
  export func printSum(a: int, b: int) -> () ! {IO} {
    let sum = computeSum(a, b);
    print("Sum: " ++ show(sum))
  }

  -- IO and FS effects (comma-separated in braces)
  export func logToFile(filename: string, message: string) -> () ! {IO, FS} {
    writeFile(filename, message);
    ()
  }

  -- Main function with composed effects
  export func main() -> () ! {IO, FS} {
    let result = computeSum(10, 20);
    printSum(10, 20);
    logToFile("output.txt", "Sum was 30");
    print("Done")
  }
  ```

  **Key syntax notes:**
  - Effects come AFTER return type: `-> () ! {IO, FS}`
  - Multiple effects in braces: `! {IO, FS}` not `! IO, FS`
  - Use `writeFile` from stdlib (no import needed in entry modules)
  - Remember semicolons between statements in blocks
```

## Implementation Plan

### Phase 1: Update Prompt (0.5 days)

1. ✅ Add comment syntax section
2. ✅ Clarify effect syntax consistently
3. ✅ Expand list cons warnings
4. ✅ Add `let` binding emphasis
5. ✅ Add module declaration checklist
6. ✅ Update all examples to use consistent syntax

**Output**: prompts/v0.4.3.md

### Phase 2: Audit & Fix Benchmarks (0.5 days)

1. ✅ Scan all benchmark .yml files for syntax errors
2. ✅ Fix config_file_parser.yml effect syntax
3. ✅ Add AILANG hints to effect_tracking_io_fs.yml
4. ✅ Add hints to simple_print.yml (21 turns is too many!)
5. ✅ Validate hints compile correctly

**Output**: Updated benchmark/*.yml files

### Phase 3: Validation (0.25 days)

1. ✅ Run eval on 3 dev models with new prompt
2. ✅ Verify MOD_001 reduction
3. ✅ Verify PAR_001 reduction
4. ✅ Compare v0.4.2 vs v0.4.3 metrics

## Expected Impact

**Current (v0.4.2)**:
- MOD_001: 64 failures (21% of failures)
- PAR_001: 36 failures (12% of failures)
- Total: 100 failures from confusion (33% of all failures!)

**Projected (v0.4.3)**:
- MOD_001: ~10-15 failures (eliminate most confusion)
- PAR_001: ~36 failures (will remain until surface-sugar-pack)
- Total: ~45-50 failures from these categories

**Improvement**: -50-55 failures (~17-20pp success rate gain!)

**Combined with surface-sugar-pack (v0.4.3+)**:
- MOD_001: ~10-15
- PAR_001: ~0 (infix cons will work!)
- **Total success rate: ~68-75%** (up from 54.2%)

## Testing Strategy

**Before merging**:
```bash
# 1. Validate prompt syntax examples compile
ailang check prompts/v0.4.3_examples/*.ail

# 2. Run eval on dev models (quick validation)
make eval-baseline EVAL_VERSION=v0.4.3-rc1 MODELS=gpt5-mini,claude-haiku-4-5,gemini-2-5-flash

# 3. Compare metrics
ailang eval-compare eval_results/baselines/v0.4.2 eval_results/baselines/v0.4.3-rc1

# 4. Check specific error reductions
jq -s 'map(select(.err_code == "MOD_001")) | length' eval_results/baselines/v0.4.3-rc1/summary.jsonl
jq -s 'map(select(.err_code == "PAR_001")) | length' eval_results/baselines/v0.4.3-rc1/summary.jsonl
```

**Success criteria**:
- MOD_001 count < 20 (down from 64)
- Overall AILANG success rate > 60% (up from 54.2%)
- No regressions in working benchmarks

## Related Work

**Complementary improvements**:
- [surface-sugar-pack.md](v0_4_2/surface-sugar-pack.md) - Eliminates PAR_001 errors (infix cons)
- [AILANG_EASE_OF_USE_ASSESSMENT.md](../v0_3_18/AILANG_EASE_OF_USE_ASSESSMENT.md) - DX improvements

**Why prompt-first approach**:
1. **Fast iteration**: Prompt changes deploy immediately (no code changes)
2. **Zero risk**: Can't break existing functionality
3. **High ROI**: 20pp improvement for 0.5 days effort
4. **Validates syntax choices**: See which patterns models prefer

After prompt improvements prove effective, implement syntactic sugar for remaining gaps.

## References

- v0.4.2 eval analysis: 64 MOD_001, 36 PAR_001 errors
- Agent KPI analysis: simple_print takes 21 turns (should be trivial!)
- Benchmark audit: config_file_parser.yml has wrong effect syntax
- Lexer source: `internal/lexer/lexer.go:104-107` - `--` comments supported!

---

**Next steps**:
1. Create prompts/v0.4.3.md with improvements
2. Audit and fix benchmark .yml files
3. Run validation eval on dev models
4. Measure MOD_001/PAR_001 reduction
5. If successful (>60% AILANG success), merge to dev
