# M-GAP5: Add `repeat` to std/string

## Status
- **Status:** Planned
- **Target:** v0.6.4
- **Priority:** P2 (Low)
- **Estimated:** 2 hours
- **Dependencies:** None

## Problem Statement

AILANG's standard library lacks a `repeat` function for string repetition. This is a common operation needed for:
- Box-drawing UI (borders, separators)
- Formatting and alignment
- Generating test data

### Current Workaround
```ailang
-- Must implement locally every time
pure func repeat(s: string, n: int) -> string =
  if n <= 0 then "" else s ++ repeat(s, n - 1)

let border = repeat("-", 40)  -- "----------------------------------------"
```

### Desired API
```ailang
import std/string (repeat)

let border = repeat("-", 40)
let indent = repeat("  ", 3)  -- "      "
```

## Goals

**Primary Goal:** Add `repeat` function to std/string module

**Success Metrics:**
- `repeat(s, n)` available from std/string
- O(n) concatenation (acceptable for typical use)
- Works with any string including multi-character patterns

## Solution Design

### API Design

```ailang
-- std/string module addition
-- Repeat a string n times
-- repeat("ab", 3) = "ababab"
-- repeat("x", 0) = ""
-- repeat("", 5) = ""
pure func repeat(s: string, n: int) -> string
```

### Implementation Options

**Option A: Pure AILANG (Recommended)**
```ailang
pure func repeat(s: string, n: int) -> string =
  if n <= 0 then "" else s ++ repeat(s, n - 1)
```
- Simple, correct, self-documenting
- O(n) string allocations
- Sufficient for typical use cases (n < 1000)

**Option B: Builtin with Go implementation**
```go
// internal/builtins/string.go
func builtinRepeat(args []Value) (Value, error) {
    s := args[0].(StringValue).Value
    n := args[1].(IntValue).Value
    return StringValue{Value: strings.Repeat(s, int(n))}, nil
}
```
- O(1) allocation via `strings.Builder`
- Better for large n
- More complexity

**Recommendation:** Start with Option A. Add builtin later if performance becomes an issue.

### Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `std/string.ail` | Add `repeat` function | ~5 |
| `std/string.ail` | Export in module | ~1 |

### Implementation

```ailang
-- Add to std/string.ail

-- | Repeat a string n times.
-- | repeat("ab", 3) = "ababab"
-- | repeat("x", 0) = ""
pure func repeat(s: string, n: int) -> string =
  if n <= 0
  then ""
  else s ++ repeat(s, n - 1)
```

## Examples

```ailang
import std/string (repeat)

-- Box drawing
let width = 40
let border = "+" ++ repeat("-", width - 2) ++ "+"
-- "+--------------------------------------+"

-- Indentation
let indent = repeat("  ", depth)
let line = indent ++ "item"

-- Separators
let separator = repeat("=", 60)
print(separator)
print("Section Title")
print(separator)

-- Pattern generation
let pattern = repeat("*-", 10)  -- "*-*-*-*-*-*-*-*-*-*-"
```

## Testing

### Test Cases
```ailang
-- test_string_repeat.ail
import std/string (repeat)

-- Basic repetition
let _ = assert(repeat("x", 3) == "xxx")
let _ = assert(repeat("ab", 2) == "abab")

-- Edge cases
let _ = assert(repeat("x", 0) == "")
let _ = assert(repeat("x", 1) == "x")
let _ = assert(repeat("", 5) == "")

-- Negative n
let _ = assert(repeat("x", -1) == "")

-- Unicode
let _ = assert(repeat("🔥", 3) == "🔥🔥🔥")
```

## Success Criteria

- [ ] `repeat` function added to std/string
- [ ] Function exported and importable
- [ ] All test cases pass
- [ ] Used in at least one example file

## Timeline

**Day 1:** Implement and test (2 hours)
- Add function to std/string.ail
- Write test cases
- Verify with examples

## Axiom Alignment

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A7: Machines First | +1 | Common stdlib function aids AI code generation |
| A8: Syntax Is Liability | 0 | No new syntax, just library function |

**Net Score:** +1 (Accept)

## Related Documents

- [std/string.ail](../../../std/string.ail) - String module
- [internal/builtins/](../../../internal/builtins/) - If builtin needed later
- Python: `"x" * 3`, Haskell: `replicate 3 'x'`, Go: `strings.Repeat("x", 3)`
