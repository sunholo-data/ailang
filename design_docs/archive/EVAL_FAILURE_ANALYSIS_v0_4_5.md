# Eval Failure Analysis - v0.4.5

**Analysis Date**: 2025-11-16
**Version**: v0.4.5
**Eval Mode**: 0-shot (Standard)
**Total Tests**: 284 (AILANG only)
**Success Rate**: 68.6% (195/284)
**Failure Rate**: 31.4% (89/284)

## Executive Summary

v0.4.5 achieved **68.6% success rate**, up **+8.1%** from v0.4.4 (60.5%). The two bug fixes (nullary constructors, concat operator) significantly improved performance. However, **compile errors still dominate failures (77%)**, primarily due to:

1. **Models generating Python/imperative code** (40% of failures)
2. **Module system misunderstanding** (models put multiple modules in one file)
3. **Missing operator knowledge** (modulo `%` doesn't exist, use `mod` function)

## Failure Categories

| Category | Count | % of Failures | Root Cause |
|----------|-------|---------------|------------|
| **Compile Error** | 69 | 77% | Wrong syntax, missing operators, module confusion |
| **Logic Error** | 13 | 14% | Correct syntax, wrong output (algorithm issues) |
| **Runtime Error** | 7 | 9% | Capability errors, type errors at runtime |

### Error Code Breakdown

| Error Code | Count | % | Description |
|------------|-------|---|-------------|
| **WRONG_LANG** | 36 | 40% | Python/imperative code instead of functional AILANG |
| **PAR_001** | 23 | 26% | Parser errors (syntax, multiple modules, missing operators) |
| **CAP_001** | 8 | 9% | Missing effect declarations (`! {IO}`, `! {FS}`) |
| **IMPERATIVE** | 8 | 9% | Imperative style (overlaps with WRONG_LANG) |
| **(other)** | 14 | 16% | Logic errors, runtime errors, etc. |

## Benchmarks with 0% Success Rate

These **10 benchmarks failed across all 6 models** (0% success):

1. **api_call_json** (6/6 failures) - JSON + HTTP effects
2. **cli_args** (6/6 failures) - Command-line arguments
3. **config_file_parser** (6/6 failures) - File I/O + parsing
4. **float_eq** (6/6 failures) - Floating-point comparison
5. **json_encode** (6/6 failures) - JSON encoding
6. **json_parse** (6/6 failures) - JSON parsing
7. **multi_module_imports** (6/6 failures) - Multi-module projects
8. **numeric_modulo** (6/6 failures) - Modulo operation
9. **pipeline** (6/6 failures) - Function composition
10. **record_update** (6/6 failures) - Record update syntax (logic errors)

## Pattern Analysis

### Pattern 1: WRONG_LANG - Python/Imperative Code (36 failures)

**Example** (json_parse):
```python
# ❌ Model generated Python-style code:
let json_str = "[{\"name\":\"Alice\",\"age\":30},...]";
let people = parse_json(json_str);

for person in people {         # ❌ for loop doesn't exist
    if person["age"] >= 30 {   # ❌ dict access doesn't work like this
        print(person["name"]); # ❌ No semicolons in AILANG
    }
}
```

**Root Cause**: Teaching prompt doesn't sufficiently emphasize:
- AILANG is functional (no for loops)
- Use recursion, `fold`, `map` instead of loops
- Records use `.field` syntax, not `["field"]`

**Fix**: Improve teaching prompt with:
- "AILANG has NO for loops or while loops"
- Show fold/map examples for iteration
- Clarify record field access syntax

### Pattern 2: Multiple Modules in One File (PAR_001 errors)

**Example** (multi_module_imports):
```ailang
# ❌ Model generated multiple modules in one file:
module benchmark/data
type User = { name: string, age: int }
...

module benchmark/storage        # ❌ Second module in same file!
import benchmark/data (User)
...

module benchmark/solution       # ❌ Third module in same file!
import benchmark/storage (...)
...
```

**Root Cause**: Teaching prompt doesn't explain module system clearly:
- One module per file
- File path determines module name
- Multi-file projects require directory structure

**Fix**: Add to teaching prompt:
- "ONE MODULE PER FILE - the file path IS the module name"
- Show multi-file project example with directory structure
- Explain how imports resolve to file paths

### Pattern 3: Missing Modulo Operator (numeric_modulo failures)

**Example**:
```ailang
# ❌ Model generated:
print(5 % 3)  # ❌ % operator doesn't exist!

# ✅ Should be:
import std/prelude (mod)
print(mod(5, 3))
```

**Root Cause**: Teaching prompt doesn't list available operators and functions.

**Fix**: Add operator reference table to prompt:
- Arithmetic: `+`, `-`, `*`, `/` (no `%`!)
- Comparison: `>`, `<`, `>=`, `<=`, `==`, `!=`
- Logical: `&&`, `||`, `!`
- String: `++` (concatenation)
- **For modulo: use `mod(a, b)` from std/prelude**

### Pattern 4: JSON Handling (json_parse, json_encode failures)

**Example**:
```ailang
# ❌ Models invent JSON functions:
let people = parse_json(json_str)  # ❌ parse_json doesn't exist
let json = to_json(data)           # ❌ to_json doesn't exist

# ✅ Should use std/json:
import std/json (decode, encode)
let people = decode(json_str)
let json = encode(data)
```

**Root Cause**: Teaching prompt doesn't show JSON standard library usage.

**Fix**: Add JSON example to prompt:
```ailang
-- JSON parsing and encoding:
import std/json (decode, encode)

let json_str = "{\"name\":\"Alice\",\"age\":30}"
let parsed = decode(json_str)     -- Returns dynamic value
let encoded = encode(parsed)      -- Returns string
```

### Pattern 5: Missing Effect Declarations (CAP_001 errors)

**Example** (print_missing_effect):
```ailang
# ❌ Model forgot effect declaration:
export func main() -> () {        # ❌ Missing ! {IO}
  println("Hello")                # println requires IO effect!
}

# ✅ Should be:
export func main() -> () ! {IO} {
  println("Hello")
}
```

**Root Cause**: Models forget effect declarations on main functions.

**Fix**: Emphasize in prompt:
- "**EVERY function that uses print/println/readFile/writeFile MUST declare effects**"
- Show examples: `-> () ! {IO}`, `-> string ! {FS}`, `-> int ! {IO, FS}`

## Model Performance Comparison

| Model | Success Rate | Strengths | Weaknesses |
|-------|--------------|-----------|------------|
| **claude-haiku-4-5** | 75% (45/60) | Best overall | Occasional module confusion |
| **claude-sonnet-4-5** | 75% (45/60) | Best overall | Occasional module confusion |
| **gemini-2-5-pro** | 68% (28/41) | Good functional style | JSON handling issues |
| **gpt5-mini** | 66% (27/41) | Fast, decent | Module system confusion |
| **gpt5** | 63% (26/41) | Good reasoning | Imperative fallback |
| **gemini-2-5-flash** | 59% (24/41) | Fast | Most WRONG_LANG errors |

**Observation**: Claude models (both haiku and sonnet) have best success rate (75%), likely due to better understanding of functional programming patterns.

## Recommended Improvements

### Priority 1: Teaching Prompt Improvements (M-DX-TEACHING-PROMPT-v0.4.6)

**Impact**: Could fix 40%+ of failures (WRONG_LANG + PAR_001)

**Changes needed**:
1. **Add "What AILANG Does NOT Have" section**:
   - ❌ No for/while loops → Use recursion, fold, map
   - ❌ No `%` operator → Use `mod(a, b)` function
   - ❌ No semicolons → Use block expressions `{ e1; e2 }`
   - ❌ No multiple modules per file → One module per file

2. **Add Multi-Module Project Example**:
   ```
   Project structure:
   myapp/
     data.ail       -- module myapp/data
     storage.ail    -- module myapp/storage
     main.ail       -- module myapp/main

   File: myapp/data.ail
   module myapp/data
   export type User = { name: string }

   File: myapp/storage.ail
   module myapp/storage
   import myapp/data (User)
   export func saveUser(u: User) -> () ! {FS} { ... }

   File: myapp/main.ail
   module myapp/main
   import myapp/storage (saveUser)
   export func main() -> () ! {IO, FS} { ... }
   ```

3. **Add JSON Example**:
   ```ailang
   import std/json (decode, encode)
   let parsed = decode("{\"x\":1}")  -- Parse JSON
   let json = encode({x: 1})         -- Encode to JSON
   ```

4. **Add Operator Reference Table**:
   - List all available operators
   - Explicitly state what's missing (`%`, `++` for lists)
   - Show function alternatives (`mod`, `append`)

### Priority 2: Better Error Messages (M-DX-ERROR-MESSAGES)

**Impact**: Helps models self-repair (repair success rate: 68.6% → ~75%?)

**Changes needed**:
- PAR_001 errors could suggest: "Did you mean to use `mod(a, b)` instead of `%`?"
- Multiple module errors could suggest: "Move each module to its own file"
- WRONG_LANG errors could suggest: "Use recursion instead of for loops"

### Priority 3: Standard Library Documentation

**Impact**: Reduces JSON/IO/FS confusion

**Changes needed**:
- Document std/json functions in prompt
- Document std/io functions in prompt
- Document std/fs functions in prompt
- Show effect requirements for each

## Success Stories

### What's Working Well

1. **Nullary constructors** (100% success on exhaustive_pattern_matching after v0.4.5 fix)
2. **Recursive string concatenation** (join function now works)
3. **Agent eval** (100% AILANG success with multi-turn repair)
4. **Basic functional patterns** (map, fold, recursion)

### Benchmarks with High Success (>80%)

- **simple_print**: 100%
- **fizzbuzz**: 83%
- **recursion_factorial**: 83%
- **records_person**: 83%
- **list_operations**: 83%

These show that models understand basic AILANG when the task is clear and doesn't require advanced features (JSON, multi-module, effects).

## Conclusion

**v0.4.5 is a significant improvement** (+8.1% from v0.4.4), but **teaching prompt gaps** remain the biggest blocker:

1. **40% of failures** are WRONG_LANG (Python/imperative style)
2. **26% of failures** are PAR_001 (module confusion, missing operators)
3. **Improving the teaching prompt could eliminate 60%+ of remaining failures**

**Next steps**:
1. Create design doc: `M-DX-TEACHING-PROMPT-v0.4.6.md`
2. Add multi-module example, JSON example, operator reference
3. Emphasize "What AILANG Does NOT Have"
4. Run eval baseline on v0.4.6 to measure improvement

**Target for v0.4.6**: 75%+ 0-shot success rate (from current 68.6%)
