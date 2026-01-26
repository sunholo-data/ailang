# M-EVAL-GAP-FIXES: Address Language Gaps from v0.7.0 Eval Analysis

**Status**: Planned
**Priority**: High
**Effort**: Small (1-2 hours)
**Version**: v0.7.1

## Summary

Analysis of v0.7.0 eval baseline revealed several language gaps causing agent failures. This document addresses the most impactful issues to improve AILANG teachability.

## Findings from v0.7.0 Eval

### Overall Results
- **Standard Mode**: 80.6% success rate (448 runs)
- **Agent Mode**: 72.8% AILANG success vs 76.0% Python
- **AILANG costs 78% more** than Python in agent mode (higher token usage)

### Error Distribution
- **60 WRONG_LANG errors** - Models writing Python instead of AILANG
- **16 PAR_001 errors** - Parse errors from syntax confusion
- **23 compile_error** - Compilation failures
- **14 logic_error** - Wrong output

## Identified Gaps

### 1. Boolean NOT Operator (`!` vs `not`)

**Problem**: Agents try `!condition` which causes parse errors.

```ailang
-- Agent writes:
if !isEmpty(list) then ...  -- FAILS: unknown unary operator: !

-- Should write:
if not isEmpty(list) then ...  -- WORKS
```

**Impact**: 5+ failures in balanced_parens and other benchmarks

**Fix Options**:
- A) Add `!` as alias for `not` in lexer/parser
- B) Add error message: "Did you mean `not`?" when `!` is encountered
- C) Document prominently in prompt (immediate fix)

**Recommendation**: Option C (immediate) + Option B (v0.7.2)

### 2. `std/env.args` Not Exported

**Problem**: Agents can't access CLI arguments - `args` not exported from `std/env`.

```
Error: IMP010: symbol 'args' not exported by 'std/env'
```

**Impact**: 5 failures in cli_args benchmark

**Fix**: Export `args` function from std/env.ail

**Blocker**: Nullary function call syntax issue (M-DX10)

### 3. Hallucinated Functions

Agents expect these functions but they don't exist:

| Function | Agent Expectation | Current Solution |
|----------|-------------------|------------------|
| `toJsonList` | Convert to JSON array | Use `json.encode()` |
| `asArray` | JSON value to list | Pattern match on Json type |
| `concat` | Concatenate strings/lists | Use `++` operator |
| `filterValid` | Filter helper | Write manual filter |
| `floatToStr` | Float to string | Use `show` or `_float_to_string` builtin |

**Fix**: Add stdlib wrappers OR document alternatives in prompt

### 4. Effect Declaration Confusion

**Problem**: Agents forget to declare effects in function signatures.

```
Error: Effect checking failed for function 'main'
  Function uses effects not declared in signature
  Missing effects: Env
```

**Impact**: 3+ failures

**Fix**: Add more examples in prompt showing effect declarations

## Implementation Plan

### Phase 1: Prompt Improvements (Immediate)

Update v0.7.0 prompt to v0.7.1 with:

1. **Add `not` operator section** prominently
   ```
   ## Boolean Operations
   - Use `not condition` (NOT `!condition` - that's Python!)
   - Use `&&` for AND, `||` for OR
   ```

2. **Add effect declaration examples**
   ```
   -- Function with IO effect:
   func greet: string -> ! IO unit

   -- Main with multiple effects:
   func main: ! {IO, FS} unit
   ```

3. **Document string/list concatenation**
   ```
   -- Use ++ for concatenation (NOT concat())
   "hello" ++ " " ++ "world"  -- "hello world"
   [1, 2] ++ [3, 4]           -- [1, 2, 3, 4]
   ```

4. **Add JSON manipulation examples**
   ```
   -- Converting to JSON
   let jsonStr = json.encode(myRecord)

   -- Parsing JSON
   let parsed = json.decode(jsonStr)
   ```

### Phase 2: Stdlib Additions (v0.7.2)

1. Add to `std/string.ail`:
   - `concat: string -> string -> string` (alias for `++`)

2. Add to `std/json.ail`:
   - `toArray: Json -> [Json]` (extract array from Json value)
   - `toObject: Json -> {string: Json}` (extract object)

3. Add to `std/prelude.ail`:
   - `floatToString: float -> string` (wrapper for `_float_to_string`)

### Phase 3: Error Messages (v0.7.2)

Add helpful error for common mistakes:
- `unknown unary operator: !` -> "Did you mean `not`? AILANG uses `not` for boolean negation."

## Success Criteria

- WRONG_LANG errors reduced by 50%
- Agent mode AILANG success rate ≥ 75%
- Token usage gap between Python and AILANG reduced

## Related Work

- M-DX10: Nullary function call syntax (blocks std/env.args)
- M-PROMPT: Prompt version management
- M-EVAL-LOOP: Evaluation infrastructure

## Testing

After prompt update:
```bash
# Run quick eval to validate
ailang eval-suite --models gpt5-mini --benchmarks balanced_parens,cli_args

# Check for WRONG_LANG reduction
jq -r '.err_code' eval_results/*.json | sort | uniq -c
```
