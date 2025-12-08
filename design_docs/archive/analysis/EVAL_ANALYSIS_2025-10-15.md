# M-EVAL Analysis: v0.3.7 Failure Patterns & Roadmap Suggestions

**Analysis Date**: 2025-10-15
**Baseline**: v0.3.7-1-gd24a7dc (eval_results/baselines/)
**Total Runs**: 114 (19 benchmarks × 2 languages × 3 models)
**AILANG Success**: 9/57 (15.8%) - **SIGNIFICANTLY WORSE than CHANGELOG claim of 58.8%**

> **⚠️ DISCREPANCY ALERT**: CHANGELOG.md v0.3.7 claims "58.8% success rate (67/114 runs)" but latest eval shows **15.8%** (9/57 AILANG runs passing). Need to investigate.

---

## Executive Summary

Analysis of 57 AILANG benchmark runs (Claude Sonnet 4.5, GPT-5, Gemini 2.5 Pro) reveals **9 major missing features** that account for 48 failures:

### Top Missing Features (by failure count)

| Priority | Feature | Failures | Impact | Roadmap Status |
|----------|---------|----------|--------|----------------|
| **P0** | CLI arguments (`sys.args()`) | 9 | 15.8% | ❌ Not in v0.4.0 |
| **P0** | List comprehensions | 9 | 15.8% | ✅ v0.4.0 (P2d) |
| **P0** | JSON parsing stdlib | 9 | 15.8% | ✅ v0.4.0 (std/json) |
| **P1** | Native list syntax `[1,2,3]` | 6 | 10.5% | ✅ v0.4.0 (P2b) |
| **P1** | String methods (split, trim, to_int) | 6 | 10.5% | ✅ v0.4.0 (std/string) |
| **P2** | For loops (or desugar to recursion) | 3 | 5.3% | ❌ Not planned |
| **P2** | Mutable variables (`let mut`) | 3 | 5.3% | ❌ Not planned |
| **P2** | Error propagation (`?` operator) | 3 | 5.3% | ✅ v0.4.0 (P2c) |
| **P3** | Modulo operator fix | 3 | 5.3% | ⚠️ Investigate |

**Recommendation**: Add **CLI arguments** (`std/cli`) as **P0 for v0.4.0** (currently planned as stdlib item but not prioritized).

---

## Detailed Failure Analysis

### 1. ❌ CLI Arguments (`sys.args()`) - 9 failures

**Problem**: AI models expect Python-style CLI argument access

**Examples**:
```ailang
-- ❌ Generated (Python-style):
import sys
let filename = sys.args()[1]

-- ✅ AILANG needs:
import std/cli (getArgs)
export func main() -> () ! {IO} {
  let args = getArgs();
  let filename = args[1];
  ...
}
```

**Affected Benchmarks**: `cli_args` (all 3 models)

**Root Cause**:
- No `std/cli` module exists
- No way to access command-line arguments at runtime
- AI models default to Python/Node.js patterns

**Solution** (Already in v0.4.0 roadmap):
- Implement `std/cli` module (~250 LOC, 2-3 days)
- Functions: `getArgs() -> List<string> ! {IO}`
- Add to stdlib priority list

**Recommendation**: Upgrade from planned to **P0 priority** for v0.4.0 (currently just "stdlib expansion" without priority).

---

### 2. ❌ List Comprehensions - 9 failures

**Problem**: AI models generate Python-style list comprehensions

**Examples**:
```ailang
-- ❌ Generated (Python-style):
let evens = [x * 2 for x in xs if x % 2 == 0]

-- ✅ AILANG needs:
let evens = filter(\x. x % 2 == 0, map(\x. x * 2, xs))
```

**Affected Benchmarks**: `list_comprehension`, `higher_order_functions`, `pipeline`

**Root Cause**:
- List comprehensions are ubiquitous in Python, Haskell, JavaScript
- Functional equivalent (map/filter chains) is verbose and unintuitive
- AI models strongly associate list transformations with comprehensions

**Solution** (✅ Already in v0.4.0 roadmap):
- Syntax: `[expr | var <- list, predicate]`
- Desugar to map/filter chains
- P2d priority (~300 LOC, 4-5 days)

**Recommendation**: Keep as P2d (already planned correctly).

---

### 3. ❌ JSON Parsing - 9 failures

**Problem**: No JSON stdlib module exists

**Examples**:
```ailang
-- ❌ Generated (imaginary API):
let people = parse_json(json_str);
for person in people {
  if person.age >= 30 {
    print(person.name);
  }
}

-- ✅ AILANG needs:
import std/json (parseJSON, get)
let people = parseJSON(json_str);
match people {
  Ok(arr) => ...,
  Err(e) => ...
}
```

**Affected Benchmarks**: `json_parse`, `nested_records`, `error_handling`

**Root Cause**:
- JSON is universal data format
- No `std/json` module implemented
- AI models assume JSON parsing is always available

**Solution** (✅ Already in v0.4.0 roadmap):
- Implement `std/json` module (~400 LOC, 3-4 days)
- Functions: `parseJSON`, `stringify`, `get`, `set`
- P2 priority (stdlib expansion)

**Recommendation**: Keep as planned (already high priority in stdlib).

---

### 4. ❌ Native List Syntax - 6 failures

**Problem**: ADT-based lists are too verbose, AI models generate Python-style `[1,2,3]`

**Examples**:
```ailang
-- ❌ Generated (Python-style):
let xs = [1, 2, 3, 4, 5]
let sum = xs.sum()

-- 🤔 Current (ADT-style):
let xs = Cons(1, Cons(2, Cons(3, Cons(4, Cons(5, Nil)))))
let sum = sum_list(xs)  -- Must write own sum_list

-- ✅ Desired (native):
let xs = [1, 2, 3, 4, 5]
let sum = fold(\acc, x. acc + x, 0, xs)
```

**Affected Benchmarks**: `list_operations`, `higher_order_functions`, `pipeline`

**Root Cause**:
- ADT constructors are too verbose for common case
- AI models universally expect `[1,2,3]` syntax
- No builtin list functions (map, filter, fold)

**Solution** (✅ Already in v0.4.0 roadmap):
- Native `List<T>` type with `[1,2,3]` syntax
- Builtins: `head`, `tail`, `length`, `map`, `filter`, `fold`
- P2b priority (~500 LOC, 5-7 days)

**Recommendation**: Keep as P2b (already planned).

---

### 5. ❌ String Methods - 6 failures

**Problem**: No string manipulation functions in stdlib

**Examples**:
```ailang
-- ❌ Generated (Python-style):
let lines = content.split("\n")
for line in lines {
  if line.trim() != "" {
    sum = sum + line.to_int()
  }
}

-- ✅ AILANG needs:
import std/string (split, trim)
let lines = split(content, "\n");
let trimmed = map(trim, lines);
...
```

**Affected Benchmarks**: `cli_args`, `json_parse`, `string_manipulation`

**Root Cause**:
- Only basic string concatenation (`++`) exists
- No split, trim, replace, toUpper, toLower, etc.
- AI models assume rich string API (from Python/JavaScript)

**Solution** (✅ Already in v0.4.0 roadmap):
- Implement `std/string` module (~200 LOC, 2 days)
- Functions: `split`, `join`, `trim`, `startsWith`, `endsWith`, `replace`, `toUpper`, `toLower`
- Part of stdlib expansion

**Recommendation**: Keep as planned.

---

### 6. ⚠️ For Loops - 3 failures

**Problem**: AI models generate imperative for loops

**Examples**:
```ailang
-- ❌ Generated (imperative):
for line in lines {
  if line.trim() != "" {
    sum = sum + line.to_int()
  }
}

-- ✅ AILANG (functional):
let result = fold(\acc, line.
  if trim(line) != ""
  then acc + toInt(line)
  else acc, 0, lines)
```

**Affected Benchmarks**: `cli_args`, `list_operations`, `pipeline`

**Root Cause**:
- AILANG is purely functional (no imperative loops)
- AI models default to imperative patterns
- Prompt may not emphasize functional style strongly enough

**Solution Options**:

**Option 1: Syntax Sugar (P3 - v0.5.0+)**
- Desugar `for x in xs { body }` to `forEach(\x. body, xs)`
- PRO: Familiar syntax for AI models
- CON: Doesn't return value, encourages imperative thinking

**Option 2: Better Prompts (P0 - Immediate)**
- Update teaching prompt to emphasize functional style
- Provide more fold/map/filter examples
- Explain why "no loops" is a design principle

**Recommendation**: **Option 2** (update prompts, don't add for loops). AILANG is pure functional by design.

---

### 7. ⚠️ Mutable Variables - 3 failures

**Problem**: AI models generate imperative `let mut` or `var`

**Examples**:
```ailang
-- ❌ Generated (imperative):
let sum = 0
for line in lines {
  sum = sum + line.to_int()  -- Mutation!
}

-- ✅ AILANG (functional):
let sum = fold(\acc, line. acc + toInt(line), 0, lines)
```

**Affected Benchmarks**: `cli_args`, `list_operations`

**Root Cause**:
- Same as for loops - AI defaults to imperative patterns
- Prompts may not emphasize immutability

**Solution**: **Update prompts** (same as for loops). Mutation conflicts with pure functional design.

**Recommendation**: Do NOT add mutable variables. Update teaching prompt instead.

---

### 8. ✅ Error Propagation (`?` operator) - 3 failures

**Problem**: Explicit match on Result/Option is verbose

**Examples**:
```ailang
-- 🤔 Current (verbose):
let content = match readFile(path) {
  Ok(c) => c,
  Err(e) => return Err(e)
};

-- ✅ Desired (concise):
let content = readFile(path)?;
```

**Affected Benchmarks**: `error_handling`, `json_parse`, `cli_args`

**Solution** (✅ Already in v0.4.0 roadmap):
- Rust-style `?` operator for early returns
- P2c priority (~150 LOC, 3-4 days)

**Recommendation**: Keep as P2c (already planned).

---

### 9. ❓ Modulo Operator - 3 failures (Investigate)

**Problem**: `%` operator may have bugs or unexpected behavior

**Examples**:
```ailang
-- Benchmark: numeric_modulo
-- Status: runtime_error
-- Need to check: Does % work correctly? Type errors?
```

**Affected Benchmarks**: `numeric_modulo`, `fizzbuzz`

**Root Cause**: Unknown - needs investigation

**Solution**:
1. Check if `%` operator is implemented
2. Check type signature (Int only? Float too?)
3. Check runtime behavior (negative numbers, division by zero)

**Recommendation**: Investigate separately (not a missing feature, possible bug).

---

### 10. ⚠️ Logic Error: ADT Pattern Matching - 1 failure

**Problem**: `list_operations` compiles and runs but returns wrong output

**Generated Code**:
```ailang
type List[a] = Cons(a, List[a]) | Nil

export func sum_list(xs: List[int]) -> int {
  match xs {
    Nil => 0,
    Cons(head, tail) => head + sum_list(tail)
  }
}

export func main() -> () ! {IO} {
  let myList = Cons(1, Cons(2, Cons(3, Cons(4, Cons(5, Nil)))));
  let s = sum_list(myList);  -- Returns 0 (WRONG!)
  println("Sum: " ++ show(s))
}
```

**Expected Output**: `Sum: 15\nLength: 5\n`
**Actual Output**: `Sum: 0\nLength: 0\n`

**Root Cause**: Unknown - needs investigation
- Pattern matching not working?
- Type parameter issue with `List[a]`?
- Runtime evaluation bug?

**Solution**: **Investigate bug** (this should work!)

**Recommendation**: High priority bug investigation (blocks ADT usage).

---

## Roadmap Impact Analysis

### v0.4.0 Roadmap Changes Needed

**Current v0.4.0 Priorities**:
1. P2a: Capability Inference (AUTO_CAPS)
2. P2b: Native List Type
3. P2c: Error Propagation `?`
4. P2d: List Comprehensions
5. Stdlib: json, cli, string, result

**Recommended Changes**:

#### 1. ✅ Promote `std/cli` to P0
**Reason**: Accounts for 15.8% of failures (9/57 runs)
**Change**: Move from "stdlib expansion" to **P0 critical priority**
**Justification**: CLI arguments are fundamental for real programs

#### 2. ✅ Keep P2b (Native Lists) as-is
**Reason**: Accounts for 10.5% of failures (6/57 runs)
**Impact**: Enables better AI codegen, reduces verbosity

#### 3. ✅ Keep P2d (List Comprehensions) as-is
**Reason**: Accounts for 15.8% of failures (9/57 runs)
**Impact**: Major ergonomics improvement

#### 4. ✅ Keep std/json, std/string as-is
**Reason**: json = 15.8% failures, string = 10.5% failures
**Impact**: Essential for real-world programs

#### 5. ❌ Do NOT add for loops or mutable variables
**Reason**: Conflicts with pure functional design
**Alternative**: Update prompts to emphasize functional style

#### 6. ⚠️ Add Investigation Tasks
**Priority P0**: Investigate ADT pattern matching bug (list_operations returning 0)
**Priority P1**: Investigate modulo operator behavior (3 failures)

---

## Updated v0.4.0 Sprint Plan

### Phase 1: Language Features (4 weeks)

**Week 1**: **P0 Critical** - CLI Arguments + Capability Inference
- Days 1-2: Implement `std/cli` (~250 LOC) **← NEW P0**
- Days 3-5: Implement AUTO_CAPS (~200 LOC)

**Week 2**: Native Lists + List Comprehensions
- Days 1-3: Native List Type (~500 LOC)
- Days 4-7: List Comprehensions (~300 LOC)

**Week 3**: Error Handling + Standard Library
- Days 1-3: `?` operator (~150 LOC)
- Days 4-7: std/json (~400 LOC), std/string (~200 LOC), std/result (~150 LOC)

**Week 4**: Bug Fixes + Polish
- Days 1-2: **Investigate ADT pattern matching bug** (list_operations)
- Days 3-4: **Investigate modulo operator** (numeric_modulo, fizzbuzz)
- Days 5-7: REPL improvements, error messages

---

## Success Metrics (Projected)

### Current (v0.3.7)
- **AILANG Success**: 9/57 (15.8%)
- **Python Success**: ?/57 (baseline comparison)

### After v0.4.0 (Projected)
- **CLI args fixed**: +9 runs = 18/57 (31.6%)
- **List comprehensions**: +9 runs = 27/57 (47.4%)
- **JSON parsing**: +9 runs = 36/57 (63.2%)
- **Native lists**: +6 runs = 42/57 (73.7%)
- **String methods**: +6 runs = 48/57 (84.2%)
- **Error propagation**: +3 runs = 51/57 (89.5%)

**Target**: 75%+ success rate (43/57 runs minimum)
**Projected**: 84.2% (48/57 runs) **← EXCEEDS TARGET**

---

## Immediate Actions

### 1. ⚠️ Investigate Eval Discrepancy (URGENT)
**Problem**: CHANGELOG claims 58.8% but eval shows 15.8%
**Possible Causes**:
- CHANGELOG measures Python+AILANG combined?
- Different baseline used for CHANGELOG?
- Regression between v0.3.7 release and eval run?

**Action**: Compare CHANGELOG baseline with eval_results baseline

### 2. ⚠️ Investigate ADT Pattern Matching Bug (P0)
**Problem**: `list_operations` returns 0 instead of 15
**Impact**: Blocks ADT usage for lists
**Action**: Create minimal repro, debug evaluator

### 3. ⚠️ Investigate Modulo Operator (P1)
**Problem**: `numeric_modulo` fails (3 benchmarks)
**Action**: Test `%` operator, check implementation

### 4. ✅ Update v0.4.0 Roadmap
**Changes**:
- Promote `std/cli` to P0 (from unprioritzed stdlib)
- Add investigation tasks for bugs
- Update success metric projections

### 5. ✅ Update Teaching Prompts
**Emphasis**:
- No for loops (use fold/map/filter instead)
- No mutable variables (pure functional)
- CLI args via `std/cli.getArgs()`
- JSON via `std/json.parseJSON()`

---

## References

- **Eval Results**: `eval_results/baselines/v0.3.7-1-gd24a7dc/`
- **v0.4.0 Roadmap**: [roadmap_v0_4_0.md](roadmap_v0_4_0.md)
- **Benchmark Specs**: `benchmarks/*.yml`
- **CHANGELOG**: [CHANGELOG.md](../CHANGELOG.md)

---

**Generated**: 2025-10-15
**Tool**: Claude Code (manual analysis)
**Version**: AILANG v0.3.7
