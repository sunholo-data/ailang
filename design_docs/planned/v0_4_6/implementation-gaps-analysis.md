# Implementation Gaps Analysis - v0.4.5

**Analysis Date**: 2025-11-16
**Version**: v0.4.5
**Focus**: Features models expect but aren't implemented (not prompt issues)

## Executive Summary

After analyzing the v0.4.5 eval failures, **most failures (66%) are due to teaching prompt gaps**, not missing features. However, there are **3 significant implementation gaps** that prevent certain benchmarks from succeeding:

1. **CLI arguments access** (cli_args benchmark - 6/6 failures)
2. **HTTP/Network capabilities** (api_call_json benchmark - 6/6 failures)
3. **JSON ergonomics** (json_parse/json_encode - indirect failures)

Additionally, **2 nice-to-have features** would improve success rates:
4. **Approximate float equality** (for numerical computing)
5. **Better file path handling** (config_file_parser failures)

## Implemented Features (Working)

### ✅ Record Update Syntax
**Status**: IMPLEMENTED and WORKING
**Syntax**: `{record | field: newValue}`

**Test**:
```ailang
let original = {name: "Alice", age: 30}
let updated = {original | age: 31}
-- Works! Output: 31
```

**Benchmark**: `record_update` (logic errors, not implementation issues)
- All 6 models generated syntactically correct code
- Failures were output mismatches (wrong formatting), not compile errors

### ✅ Modulo Function
**Status**: IMPLEMENTED
**Location**: Builtins `mod_Int`, `mod_Float`

**Test**:
```ailang
import std/prelude (mod)
let result = mod(5, 3)  -- Works! Output: 2
```

**Issue**: Models try to use `%` operator (doesn't exist)
**Solution**: Teaching prompt needs to show `mod(a, b)` function

### ✅ JSON Encoding/Decoding
**Status**: IMPLEMENTED (with caveats)
**Location**: `std/json.ail`

**Available**:
- `encode(obj: Json) -> string` ✅
- `decode(s: string) -> Result[Json, string]` ✅
- JSON ADT: `JNull`, `JBool`, `JNumber`, `JString`, `JArray`, `JObject`

**Issue**: Models expect Python-style JSON:
```python
# ❌ What models generate:
let people = parse_json(json_str)  # parse_json doesn't exist
let name = people[0]["name"]        # Dict-style access doesn't work

# ✅ What AILANG requires:
import std/json (decode, Json, JArray, JObject)
let result = decode(json_str)
match result {
  Ok(JArray(items)) => ...,  # Must pattern match on ADT
  Err(msg) => ...
}
```

**Ergonomics gap**: Models struggle with:
1. `decode` returns `Result`, not plain value
2. Must pattern match on JSON ADT (no direct field access)
3. Many accessor functions are commented out (constructor scope issue)

**Recommendation**: Low priority - this is mostly a prompt issue. Show JSON ADT pattern matching in teaching prompt.

### ✅ Float Equality
**Status**: IMPLEMENTED (exact equality)
**Location**: Builtin `eq_Float`

**Test**:
```ailang
let result = 0.0 == 0.0  -- Works! (exact equality)
```

**Issue**: Models generate ternary operator:
```python
# ❌ Model generated:
print((0.0 == 0.0) ? "true" : "false")  # ? : doesn't exist

# ✅ Correct AILANG:
import std/io (println)
let result = if 0.0 == 0.0 then "true" else "false"
println(result)
```

**Missing**: Approximate equality with epsilon
```ailang
-- ❌ Doesn't exist:
approxEqual(0.1 + 0.2, 0.3, epsilon: 0.0001)

-- Workaround: Manual implementation
func approxEq(a: float, b: float, eps: float) -> bool =
  abs(a - b) < eps
```

**Recommendation**: Low priority for v0.4.6 (teaching prompt can show if-then-else)

## Missing Features (Implementation Gaps)

### ❌ Gap 1: CLI Arguments Access (HIGH PRIORITY)

**Status**: NOT IMPLEMENTED
**Benchmark**: `cli_args` (6/6 failures)
**Impact**: 2.1% of failures (6/284)

**What models expect**:
```python
# Python-style:
import sys
args = sys.argv[1:]

# Or functional style:
args = getArgs()
```

**What AILANG needs**:
```ailang
-- Option 1: Builtin function
import std/env (getArgs)
export func main() -> () ! {IO} {
  let args = getArgs();  -- Returns [string]
  ...
}

-- Option 2: Main function signature
export func main(args: [string]) -> () ! {IO} {
  -- args passed automatically
  ...
}
```

**Design considerations**:
1. **Effect**: Should `getArgs` require `! {IO}` or be pure?
   - Pro pure: Args don't change during execution
   - Pro IO: External input (like environment variables)
   - **Recommendation**: `! {IO}` for consistency with other OS interaction

2. **Return type**: `[string]` vs `{argc: int, argv: [string]}`
   - **Recommendation**: Just `[string]` (simpler, functional style)

3. **Module**: `std/env` (with other environment functions)

**Implementation estimate**: 3-4 hours
- Add `getArgs` builtin (~1 hour)
- Add to `std/env.ail` (~30 min)
- Tests (~1 hour)
- Update teaching prompt (~30 min)
- Eval testing (~1 hour)

**Design doc**: Create `M-LANG-CLI-ARGS.md`

### ❌ Gap 2: HTTP/Network Capabilities (MEDIUM PRIORITY)

**Status**: NOT IMPLEMENTED
**Benchmark**: `api_call_json` (6/6 failures)
**Impact**: 2.1% of failures (6/284)

**What models expect**:
```python
import requests
response = requests.get("https://api.example.com/users")
data = response.json()
```

**What AILANG needs**:
```ailang
import std/http (get, HttpResponse)

export func main() -> () ! {IO, Net} {
  let response = get("https://api.example.com/users");
  match response {
    Ok(resp) => println(resp.body),
    Err(msg) => println("Error: " ++ msg)
  }
}
```

**Design considerations**:
1. **New effect**: `! {Net}` for network operations
   - Separate from `IO` (different capability)
   - Allows fine-grained permission control

2. **Functions needed**:
   - `get(url: string) -> Result[HttpResponse, string]`
   - `post(url: string, body: string) -> Result[HttpResponse, string]`
   - `HttpResponse` type: `{status: int, body: string, headers: {key: string, value: string}}`

3. **Complexity**: Higher than CLI args
   - HTTP client implementation
   - Error handling (timeouts, DNS, SSL)
   - Response parsing
   - Effect system extension (new capability)

**Implementation estimate**: 12-16 hours
- Effect system extension (~2 hours)
- HTTP client builtin (~4 hours)
- std/http module (~2 hours)
- Tests (~3 hours)
- Documentation (~1 hour)
- Eval testing (~2 hours)

**Recommendation**: DEFER to v0.5.0 or later
- Higher complexity
- Only 2.1% of failures
- Can work around with file I/O for now
- Requires careful security design

**Design doc**: Create `M-LANG-HTTP-NET-EFFECT.md` (for future)

### ❌ Gap 3: Module System - Multiple Modules Per Project (PROMPT ISSUE, NOT IMPLEMENTATION GAP)

**Status**: IMPLEMENTED but misunderstood
**Benchmark**: `multi_module_imports` (6/6 failures)
**Impact**: 2.1% of failures (6/284)

**Root cause**: Models don't understand "one module per file"

**Test**: Multi-module imports DO work when structured correctly:
```
myapp/
  data.ail       -- module myapp/data
  storage.ail    -- module myapp/storage
  main.ail       -- module myapp/main
```

**Issue**: Models generate all modules in one file:
```ailang
# ❌ WRONG (models do this):
module benchmark/data
...
module benchmark/storage  # Can't have 2nd module!
...
```

**Solution**: Teaching prompt needs better multi-module example (already in teaching-prompt-improvements.md)

**Recommendation**: Fix in teaching prompt, not implementation

## Nice-to-Have Features (Low Priority)

### 🟡 Feature 1: Approximate Float Equality

**Status**: NOT IMPLEMENTED
**Workaround**: Manual implementation

**Use case**: Scientific computing, numerical algorithms
```ailang
-- Desired:
import std/math (approxEq)
let result = approxEq(0.1 + 0.2, 0.3, epsilon: 0.0001)

-- Current workaround:
func approxEq(a: float, b: float, eps: float) -> bool =
  let diff = if a > b then a - b else b - a in
  diff < eps
```

**Implementation estimate**: 2 hours
- Add `approxEq` to std/math (~30 min)
- Tests (~30 min)
- Documentation (~30 min)
- Default epsilon value (~30 min)

**Recommendation**: DEFER - not needed for v0.4.6 eval improvements

### 🟡 Feature 2: File Path Utilities

**Status**: PARTIALLY IMPLEMENTED
**Benchmark**: `config_file_parser` (mixed results)

**Available**:
- `readFile`, `writeFile`, `fileExists` ✅

**Missing**:
- Path joining: `joinPath("/home/user", "config.txt")` → `"/home/user/config.txt"`
- Path parsing: `dirname("/home/user/config.txt")` → `"/home/user"`
- Path checking: `isAbsolute`, `isRelative`

**Recommendation**: DEFER - nice to have but not blocking

## Summary and Recommendations

### High Priority (v0.4.6)
1. **✅ Teaching Prompt Improvements** (teaching-prompt-improvements.md)
   - Fixes 66% of failures (59/89)
   - Effort: 5-7 hours
   - Impact: 68.6% → 75%+ success rate

2. **🟡 CLI Arguments Access** (NEW: M-LANG-CLI-ARGS.md)
   - Fixes 2.1% of failures (6/284)
   - Effort: 3-4 hours
   - Impact: 75% → 77% success rate
   - **Rationale**: Simple to implement, common use case

**v0.4.6 Target**: 77%+ 0-shot success rate

### Medium Priority (v0.5.0)
3. **🟡 HTTP/Network Effect** (M-LANG-HTTP-NET-EFFECT.md)
   - Fixes 2.1% of failures (6/284)
   - Effort: 12-16 hours
   - Complex: New effect, HTTP client, error handling
   - **Rationale**: Defer until effect system is more mature

### Low Priority (Future)
4. **Approximate Float Equality** - Nice to have for scientific computing
5. **File Path Utilities** - Convenience functions, not blocking

## Non-Gaps (Working Features)

These features work but models misuse them (prompt issues):
- ✅ Record update syntax (`{r | field: val}`)
- ✅ Modulo function (`mod(a, b)`)
- ✅ JSON encode/decode (with ADT pattern matching)
- ✅ Float equality (exact: `==`)
- ✅ Multi-module imports (one module per file)

## Conclusion

**v0.4.5 has a solid implementation**. Only 1 significant gap for v0.4.6:
- **CLI arguments** (quick win, 3-4 hours)

Most failures (66%) are **teaching prompt issues**, not implementation gaps. Focus on:
1. Teaching prompt improvements (Priority 1)
2. CLI args (Priority 2, quick win)

This approach achieves **77%+ success rate** with **~10 hours total work**.
