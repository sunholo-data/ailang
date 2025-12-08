# M-DX10: String Parsing Builtins

**Status:** Planned
**Version:** v0.4.3
**Priority:** High (fixes 3 eval failures)
**Estimated Effort:** 4-6 hours

## Problem Statement

AI models frequently need to parse strings to numeric types, but AILANG lacks string parsing builtins. This causes 3 eval failures in v0.4.2.1:

**Affected benchmarks:**
- `effect_composition` - needs `_str_to_int`
- `error_handling` - needs `_str_is_int`
- `tree_transformation_pipeline` - needs `Cons` (separate issue)

**Current situation:**
- ✅ Numeric conversion exists: `floatToInt`, `intToFloat` (pure)
- ❌ String parsing: NO builtins for `string -> int` or `string -> float`
- ❌ Validation: NO way to check if string is parseable

**Why this matters:**
- Common use case: CLI args, config files, user input, API responses
- AI models expect these functions to exist (standard in most languages)
- Without them, benchmarks fail with "undefined variable" errors

## Design Goals

1. **Safe parsing** - Return `Option[T]` to handle invalid input
2. **Standard naming** - Follow AILANG conventions (`stringToInt`, not `_str_to_int`)
3. **Pure functions** - No effects needed for parsing
4. **Complete coverage** - Support both int and float parsing

## Proposed API

### Core Parsing Functions

```ailang
-- Parse string to integer
-- Returns Some(n) if valid, None if invalid
-- Example: stringToInt("42") => Some(42)
--          stringToInt("abc") => None
stringToInt : string -> Option[int]

-- Parse string to float
-- Returns Some(f) if valid, None if invalid
-- Example: stringToFloat("3.14") => Some(3.14)
--          stringToFloat("not-a-number") => None
stringToFloat : string -> Option[float]
```

### Optional Helper Functions (Lower Priority)

```ailang
-- Check if string is valid integer (no allocation)
-- Example: isIntString("42") => true
--          isIntString("3.14") => false
isIntString : string -> bool

-- Check if string is valid float (no allocation)
-- Example: isFloatString("3.14") => true
--          isFloatString("abc") => false
isFloatString : string -> bool
```

## Implementation Plan

### Phase 1: Core Parsing (Required)

**1. Add Go implementations** (`internal/builtins/`)
- `builtinStringToInt` - Use `strconv.ParseInt(s, 10, 64)`
- `builtinStringToFloat` - Use `strconv.ParseFloat(s, 64)`
- Return `Option` ADT (Some/None)

**2. Register in builtin spec** (`internal/builtins/spec.go`)
```go
{
    Name:   "stringToInt",
    GoFunc: builtinStringToInt,
    Type:   types.FuncType(types.StringType, types.OptionType(types.IntType)),
    Module: "std/string",
    Pure:   true,
},
{
    Name:   "stringToFloat",
    GoFunc: builtinStringToFloat,
    Type:   types.FuncType(types.StringType, types.OptionType(types.FloatType)),
    Module: "std/string",
    Pure:   true,
},
```

**3. Export from std/string.ail**
```ailang
module std/string

-- (existing string functions...)

-- Parse string to integer
export func stringToInt(s: string) -> Option[int] = _stringToInt(s)

-- Parse string to float
export func stringToFloat(s: string) -> Option[float] = _stringToFloat(s)
```

**4. Add tests** (`internal/builtins/string_test.go`)
- Valid integers: "42", "-123", "0"
- Invalid integers: "abc", "3.14", ""
- Valid floats: "3.14", "-2.5", "0.0", "1e-10"
- Invalid floats: "abc", "", "not-a-number"

### Phase 2: Validation Helpers (Optional)

Add `isIntString` and `isFloatString` if benchmarks show need for them.

## Error Handling

**Parse failures return `None`:**
- Empty strings
- Non-numeric characters
- Overflow (numbers too large)
- Invalid syntax (e.g., "1.2.3")

**No exceptions/panics** - all errors handled via Option type.

## Usage Examples

```ailang
module example/parsing

import std/string (stringToInt, stringToFloat)
import std/io (println)

export func parseAge(input: string) -> string {
  match stringToInt(input) {
    None => "Invalid age",
    Some(age) =>
      if age < 0 then "Age must be positive"
      else if age > 150 then "Age seems unrealistic"
      else "Valid age: " ++ show(age)
  }
}

export func parsePrice(input: string) -> Option[float] {
  match stringToFloat(input) {
    None => None,
    Some(price) => if price >= 0.0 then Some(price) else None
  }
}
```

## Migration Notes

**For AI models:**
- **Old (doesn't work):** `_str_to_int(s)` - undefined
- **New (works):** `stringToInt(s)` - returns `Option[int]`
- **Import:** `import std/string (stringToInt, stringToFloat)`

**For existing code:**
- No breaking changes (new functions)
- Consider using these instead of custom parsing logic

## Testing Strategy

1. **Unit tests** - Test all parsing cases (valid/invalid)
2. **Integration tests** - Test in actual AILANG programs
3. **Eval validation** - Re-run affected benchmarks:
   - `effect_composition`
   - `error_handling`
   - Other benchmarks using string parsing

## Success Metrics

- ✅ All unit tests pass
- ✅ Functions registered in builtin spec
- ✅ Exported from std/string
- ✅ 2-3 eval failures fixed (effect_composition, error_handling)
- ✅ Prompt updated to document new functions

## Related Issues

**Prompt documentation bug (line 333):**
Current: "use `::(x, rest)` or `Cons(x, rest)` constructor"
Problem: `Cons` doesn't exist, should be removed

## Future Enhancements (Out of Scope)

- Parse with radix: `stringToIntRadix(s, 16)` for hex
- Parse with error messages: `Result[int, string]` instead of `Option[int]`
- Parse other types: `stringToBool`, `stringToChar`
- Formatted number parsing: `parseFormattedNumber("1,234.56")`

## Timeline

- **Design:** 30 minutes (done)
- **Implementation:** 2-3 hours
- **Testing:** 1-2 hours
- **Documentation:** 1 hour
- **Total:** 4-6 hours

## Dependencies

- Option ADT must be available (already exists in std/prelude)
- Go strconv package (already in stdlib)
- Builtin registry system (already implemented in M-DX1)

## Risks

**Low risk:**
- Pure functions, no side effects
- Standard Go strconv implementation
- No breaking changes

**Mitigations:**
- Comprehensive test coverage
- Follow existing builtin patterns
- Validate with eval benchmarks
