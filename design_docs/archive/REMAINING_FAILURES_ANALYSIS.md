# Remaining Failures Analysis - v0.4.2.1

**Version:** v0.4.2.1-test
**Model:** claude-haiku-4-5
**Eval Mode:** standard
**Total Failures:** 23 benchmarks (out of 41)
**Success Rate:** 43.9% (18/41)

After fixing the stdlib path bug (MOD_001 errors eliminated), here's a detailed analysis of the remaining 23 failures.

---

## Category 1: WRONG_LANG - AI Generated Wrong Language (10 benchmarks) 🚫

### Benchmarks
1. api_call_json
2. cli_args
3. csv_to_json_converter
4. float_eq
5. json_encode
6. json_parse
7. list_comprehension
8. log_file_analyzer
9. numeric_modulo
10. pipeline

### Error Pattern
```
Error: module loading error: failed to load benchmark/solution.ail
parse errors in benchmark/solution.ail
```

### Root Cause
AI model generated Python code instead of AILANG. This is a **prompt clarity issue** - the model didn't understand it should generate AILANG.

### Expected Impact of v0.4.2 Prompt
The updated prompt with clearer language specification should reduce these failures significantly.

### Priority
**High** - These are prompt-fixable (no code changes needed)

---

## Category 2: Parser Errors - PAR_001 (2 benchmarks) 🔧

### 2a. canonical_normalization

**Error:**
```
PAR_NO_PREFIX_PARSE at benchmark/solution.ail:10:12:
unexpected token in expression: ::
```

**Root Cause:** Using `::` (cons) in pattern context where it's not supported. S-CONS sugar only works in expressions.

**Status:** Documented in prompt v0.4.2 (line 337)

**Fix:** Prompt update should help AI avoid this

**Priority:** Medium (prompt fix)

---

### 2b. multi_module_imports

**Error:**
```
PAR_NO_PREFIX_PARSE: unexpected token in expression: ILLEGAL (repeated multiple times)
PAR_NO_PREFIX_PARSE: unexpected token in expression: module
PAR_NO_PREFIX_PARSE: unexpected token in expression: import
```

**Root Cause:** AI likely generated markdown code fences (```) or multiple module declarations in one file

**Fix:** Clarify in prompt that:
- One module per file
- No markdown syntax in code output
- Module declaration must be first line

**Priority:** Medium (prompt fix + benchmark spec clarification)

---

## Category 3: Missing Builtins/Features (5 benchmarks) ❌

### 3a. config_file_parser

**Error:**
```
Error: IMP010: symbol 'readFile' not exported by 'std/io'
```

**Root Cause:** AI tried to import `readFile` from `std/io` but it's in `std/fs`

**Status:** ✅ FIXED in prompt v0.4.2 with stdlib module reference table

**Expected Impact:** Should be fixed in next eval

---

### 3b. effect_composition

**Error:**
```
Error: type error: undefined variable: _str_to_int at benchmark/solution.ail:13:3
```

**Root Cause:** String parsing builtin doesn't exist

**Fix:** Implement `stringToInt : string -> Option[int]` (M-DX10)

**Priority:** **High** (3 benchmarks need this)

---

### 3c. error_handling

**Error:**
```
Error: type error: undefined variable: _str_is_int at benchmark/solution.ail:8:6
```

**Root Cause:** String validation builtin doesn't exist

**Fix:** Could be covered by `stringToInt` (check for Some vs None), or implement `isIntString : string -> bool`

**Priority:** **High** (same as 3b)

---

### 3d. deterministic_list_transform

**Error:**
```
Error: type error: undefined variable: Cons at benchmark/solution.ail:8:22
```

**Root Cause:** Prompt documentation bug - line 333 mentioned `Cons(x, rest)` as valid syntax

**Status:** ✅ FIXED in prompt v0.4.2

**Expected Impact:** Should be fixed in next eval

---

### 3e. tree_transformation_pipeline

**Error:**
```
Error: type error: undefined variable: Cons at benchmark/solution.ail:41:22
```

**Root Cause:** Same as 3d

**Status:** ✅ FIXED in prompt v0.4.2

**Expected Impact:** Should be fixed in next eval

---

## Category 4: Logic Errors - Output Mismatch (4 benchmarks) 🐛

These benchmarks compile and run successfully, but produce wrong output.

### 4a. effect_tracking_io_fs

**Expected:** `"Sum: 30\nDone\n"`
**Actual:** `"Sum: 30\nLogged to file\nDone\n"`

**Root Cause:** AI added extra `println("Logged to file")` not requested in spec

**Fix:** Benchmark spec could be more explicit about exact output requirements

**Priority:** Low (logic error, not language issue)

---

### 4b. exhaustive_pattern_matching

**Expected:** `"Waiting to start\nCurrently working\nAll done\n"`
**Actual:** `"Waiting to start\nWaiting to start\nWaiting to start\n"`

**Root Cause:** Pattern matching logic error - not handling all cases correctly

**Fix:** Benchmark spec or AI logic error

**Priority:** Low (logic error)

---

### 4c. record_update

**Expected:** `"Alice, 30, NYC\nAlice, 31, NYC\nAlice, 31, SF\n"`
**Actual:** `"Alice, 30, NYC\nAlice, 31, NYC\nAlice, 30, SF\n"`

**Root Cause:** Record update not chaining correctly - second update uses original record instead of updated one

**Example bug:**
```ailang
let p2 = {p1 | age: 31};  -- p2.age = 31
let p3 = {p1 | city: "SF"}; -- ❌ WRONG: should be p2, not p1
-- Result: p3.age = 30 (from p1), not 31 (from p2)
```

**Fix:** This is an AI logic error - it should use p2 not p1

**Priority:** Low (logic error)

---

### 4d. state_machine_traffic_light

**Expected & Actual:** Both start with `"GREEN (19sec)\nGREEN (18sec)..."`

**Root Cause:** Output looks correct in the first 150 chars, need to check full output

**Fix:** May not be an actual failure - need full output comparison

**Priority:** Very Low (may be false positive)

---

### 4e. print_missing_effect

**Expected:** `null` (expects compilation failure?)
**Actual:** `"Hello!\n"` (compiles and runs)

**Root Cause:** Benchmark expects AILANG to catch missing IO effect declaration, but it doesn't

**Example:**
```ailang
-- This should fail because print needs IO effect
export func main() -> () {  -- Missing ! {IO}
  print("Hello!")
}
```

**Fix:** This is a **language design issue** - AILANG currently allows effect violations in certain contexts

**Priority:** **High** - This is a soundness issue!

---

## Category 5: Runtime Errors (1 benchmark) 💥

### 5a. no_runtime_crashes_option

**Error:**
```
Error: execution failed: no pattern matched in match expression
```

**Expected:** `"Found: 4\nNot found\n"`
**Actual:** `null` (crashed)

**Root Cause:** Non-exhaustive pattern match - AI didn't handle all Option cases

**Example bug:**
```ailang
match findValue(xs) {
  Some(v) => "Found: " ++ show(v)
  -- Missing None case!
}
```

**Fix:** This is an AI logic error - exhaustiveness checking should catch this at compile time

**Priority:** **High** - Exhaustiveness checker may not be working correctly!

---

## Summary by Priority

### 🔥 Critical (Language/Compiler Issues)

1. **print_missing_effect** - Effect system not catching violations
2. **no_runtime_crashes_option** - Exhaustiveness checking not working

### 🚀 High Priority (Prompt Fixes + 1 Feature)

1. **10 WRONG_LANG failures** - Prompt clarity (v0.4.2 should fix)
2. **3 string parsing failures** - Need `stringToInt` builtin (M-DX10)
3. **2 Cons failures** - Prompt doc bug (v0.4.2 fixes)
4. **1 readFile failure** - Wrong module (v0.4.2 fixes)

### 📝 Medium Priority (Prompt Improvements)

1. **canonical_normalization** - S-CONS in pattern (v0.4.2 documents)
2. **multi_module_imports** - Markdown/multi-module confusion

### 🐛 Low Priority (Logic Errors)

1. **effect_tracking_io_fs** - Extra output
2. **exhaustive_pattern_matching** - Pattern logic error
3. **record_update** - Update chaining error
4. **state_machine_traffic_light** - Possibly false positive

---

## Expected Impact of Fixes

### After v0.4.2 Prompt (No Code Changes)

| Fix | Affected Benchmarks | Expected Success Increase |
|-----|-------------------|---------------------------|
| Language clarity | 10 WRONG_LANG | +10 (24.4%) |
| Cons removal | 2 Cons failures | +2 (4.9%) |
| readFile module | 1 config_file | +1 (2.4%) |
| S-CONS clarification | 1 canonical_norm | +1 (2.4%) |
| **Total** | **14 benchmarks** | **+14 (34.1%)** |

**Projected success rate after v0.4.2 prompt:** 43.9% + 34.1% = **78.0%**

### After String Parsing Builtins (M-DX10)

| Fix | Affected Benchmarks | Expected Success Increase |
|-----|-------------------|---------------------------|
| stringToInt | 2-3 failures | +2-3 (4.9-7.3%) |

**Projected success rate after builtins:** 78.0% + 6.0% = **84.0%**

### After Effect System/Exhaustiveness Fixes

| Fix | Affected Benchmarks | Expected Success Increase |
|-----|-------------------|---------------------------|
| Effect checking | 1 print_missing | +1 (2.4%) |
| Exhaustiveness | 1 no_runtime | +1 (2.4%) |

**Projected success rate after compiler fixes:** 84.0% + 4.8% = **88.8%**

---

## Action Plan

### Phase 1: Test v0.4.2 Prompt (No Code Changes)

1. ✅ Create v0.4.2 prompt with fixes
2. ⏳ Run validation eval with v0.4.2 prompt
3. ⏳ Verify 14 benchmarks now pass

### Phase 2: Implement String Parsing (M-DX10)

1. ⏳ Implement `stringToInt : string -> Option[int]`
2. ⏳ Implement `stringToFloat : string -> Option[float]`
3. ⏳ Add to std/string module
4. ⏳ Run eval to verify 2-3 more benchmarks pass

### Phase 3: Investigate Compiler Issues

1. ⏳ Debug why `print_missing_effect` compiles (should fail)
2. ⏳ Debug why `no_runtime_crashes_option` crashes (non-exhaustive match)
3. ⏳ Fix effect system soundness
4. ⏳ Fix exhaustiveness checker

### Phase 4: Benchmark Spec Improvements (Optional)

1. ⏳ Clarify exact output requirements
2. ⏳ Add examples to difficult benchmarks
3. ⏳ Document expected behavior more explicitly

---

## Next Steps

1. **Immediate:** Run validation eval with v0.4.2 prompt to test hypothesis
2. **Next:** Implement M-DX10 (string parsing builtins)
3. **Then:** Investigate effect system and exhaustiveness checker issues
