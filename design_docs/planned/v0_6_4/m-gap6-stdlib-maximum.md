# M-GAP6: Add `maximum` and `minimum` to std/list

## Status
- **Status:** Planned
- **Target:** v0.6.4
- **Priority:** P2 (Low)
- **Estimated:** 3 hours
- **Dependencies:** None

## Problem Statement

AILANG's standard library lacks `maximum` and `minimum` functions for finding extreme values in lists. These are fundamental operations needed for:
- Data analysis and aggregation
- Sorting bounds
- Game logic (scores, stats)

### Current Workaround
```ailang
-- Must implement manually with verbose foldl syntax
pure func maxInt(xs: [int]) -> int =
  foldl(func(acc: int, x: int) -> int { if x > acc then x else acc }, 0, xs)

-- Issues:
-- 1. Verbose due to GAP-2/GAP-3 (lambda syntax bug)
-- 2. Returns 0 for empty list (wrong!)
-- 3. Must reimplement for each type
```

### Desired API
```ailang
import std/list (maximum, minimum)

let maxScore = maximum([85, 92, 78, 95, 88])  -- Some(95)
let minTemp = minimum([32.5, 28.1, 35.0])     -- Some(28.1)
let empty = maximum([])                        -- None
```

## Goals

**Primary Goal:** Add `maximum` and `minimum` functions to std/list

**Success Metrics:**
- `maximum(xs)` returns `Option[a]` (None for empty list)
- `minimum(xs)` returns `Option[a]` (None for empty list)
- Works with any `Ord` type (int, float, string)

## Solution Design

### API Design

```ailang
-- std/list module additions

-- | Find the maximum element in a list.
-- | Returns None for empty lists.
-- | maximum([3, 1, 4, 1, 5]) = Some(5)
-- | maximum([]) = None
pure func maximum[a: Ord](xs: [a]) -> Option[a]

-- | Find the minimum element in a list.
-- | Returns None for empty lists.
-- | minimum([3, 1, 4, 1, 5]) = Some(1)
-- | minimum([]) = None
pure func minimum[a: Ord](xs: [a]) -> Option[a]
```

### Type Class Consideration

**Current AILANG Status:**
- Type classes exist but are hardcoded (Ord for int, float, string)
- User-defined instances not yet supported
- `maximum` can work with existing Ord types

**Implementation Approach:**
For v0.6.4, implement monomorphic versions:
- `maximumInt: [int] -> Option[int]`
- `maximumFloat: [float] -> Option[float]`
- `maximumString: [string] -> Option[string]`

Polymorphic version (`maximum[a: Ord]`) deferred to when type classes are fully supported.

### Implementation

```ailang
-- std/list.ail additions

-- Monomorphic versions (v0.6.4)
pure func maximumInt(xs: [int]) -> Option[int] =
  match xs with
  | [] -> None
  | [x] -> Some(x)
  | x :: rest -> match maximumInt(rest) with
      | None -> Some(x)
      | Some(m) -> if x > m then Some(x) else Some(m)

pure func minimumInt(xs: [int]) -> Option[int] =
  match xs with
  | [] -> None
  | [x] -> Some(x)
  | x :: rest -> match minimumInt(rest) with
      | None -> Some(x)
      | Some(m) -> if x < m then Some(x) else Some(m)

-- Float versions
pure func maximumFloat(xs: [float]) -> Option[float] = ...
pure func minimumFloat(xs: [float]) -> Option[float] = ...

-- String versions (lexicographic)
pure func maximumString(xs: [string]) -> Option[string] = ...
pure func minimumString(xs: [string]) -> Option[string] = ...
```

### Alternative: foldl-based Implementation

```ailang
-- More concise but requires GAP-2 fix for clean lambda syntax
pure func maximumInt(xs: [int]) -> Option[int] =
  match xs with
  | [] -> None
  | x :: rest -> Some(foldl(\acc y. if y > acc then y else acc, x, rest))
```

### Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `std/list.ail` | Add maximum/minimum functions | ~40 |
| `std/list.ail` | Export new functions | ~6 |

## Examples

```ailang
import std/list (maximumInt, minimumInt)

-- Basic usage
let scores = [85, 92, 78, 95, 88]
let topScore = maximumInt(scores)    -- Some(95)
let lowScore = minimumInt(scores)    -- Some(78)

-- Empty list handling
let noScores = maximumInt([])        -- None

-- With pattern matching
match maximumInt(scores) with
| Some(max) -> print("High score: " ++ intToString(max))
| None -> print("No scores yet")

-- Finding range
let temperatures = [72, 68, 75, 71, 69]
let range = match (minimumInt(temperatures), maximumInt(temperatures)) with
  | (Some(lo), Some(hi)) -> hi - lo
  | _ -> 0
-- range = 7
```

## Testing

### Test Cases
```ailang
-- test_list_extremes.ail
import std/list (maximumInt, minimumInt, maximumFloat, minimumFloat)

-- Basic cases
let _ = assert(maximumInt([3, 1, 4, 1, 5]) == Some(5))
let _ = assert(minimumInt([3, 1, 4, 1, 5]) == Some(1))

-- Single element
let _ = assert(maximumInt([42]) == Some(42))
let _ = assert(minimumInt([42]) == Some(42))

-- Empty list
let _ = assert(maximumInt([]) == None)
let _ = assert(minimumInt([]) == None)

-- Negative numbers
let _ = assert(maximumInt([-5, -2, -8]) == Some(-2))
let _ = assert(minimumInt([-5, -2, -8]) == Some(-8))

-- Duplicates
let _ = assert(maximumInt([5, 5, 5]) == Some(5))

-- Float versions
let _ = assert(maximumFloat([1.5, 2.5, 0.5]) == Some(2.5))
let _ = assert(minimumFloat([1.5, 2.5, 0.5]) == Some(0.5))
```

## Success Criteria

- [ ] `maximumInt`, `minimumInt` functions added
- [ ] `maximumFloat`, `minimumFloat` functions added
- [ ] `maximumString`, `minimumString` functions added
- [ ] All functions return `Option[a]` (None for empty)
- [ ] All test cases pass
- [ ] Functions exported from std/list

## Timeline

**Day 1:** Implement and test (3 hours)
- Add int/float/string versions
- Write comprehensive tests
- Update module exports

## Future Work

When type classes are fully supported (v0.7.0+):
```ailang
-- Polymorphic version
pure func maximum[a: Ord](xs: [a]) -> Option[a] = ...
pure func minimum[a: Ord](xs: [a]) -> Option[a] = ...
```

## Axiom Alignment

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A7: Machines First | +1 | Standard stdlib aids AI code generation |
| A11: Failure Is Data | +1 | Returns Option instead of crashing on empty |
| A8: Syntax Is Liability | 0 | Library function, no syntax change |

**Net Score:** +2 (Accept)

## Related Documents

- [std/list.ail](../../../std/list.ail) - List module
- [std/option.ail](../../../std/option.ail) - Option type
- Haskell: `maximum :: Ord a => [a] -> a` (partial, crashes on empty!)
- Python: `max(iterable, default=None)`
- Rust: `Iterator::max() -> Option<T>`
