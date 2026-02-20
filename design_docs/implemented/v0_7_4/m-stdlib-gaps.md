# Design Doc: M-STDLIB-GAPS - Standard Library Gap Analysis

**Status**: PLANNED
**Version**: v0.6.5
**Author**: Claude (Opus 4.5)
**Date**: 2026-01-05
**Priority**: High
**Estimated LOC**: ~180
**Reviewed**: 2026-01-05 (user feedback incorporated)

---

## Problem Statement

Analysis of v0.6.2 agent eval results reveals that AI agents consistently reimplement the same helper functions across multiple benchmarks. This indicates missing stdlib functions that, if added, would:

1. Reduce agent turns (less code to write)
2. Improve code quality (tested stdlib vs ad-hoc implementations)
3. Reduce compile errors (agents often make mistakes reimplementing these)

### Evidence from Eval Results

| Benchmark | Agent Reimplements | Should Use |
|-----------|-------------------|------------|
| json_parse | `getAt(xs, idx)` | `std/list.nth` |
| log_file_analyzer | `getLast(xs)` | `std/list.last` |
| csv_to_json_converter | `getAt`, `int2float` | `std/list.nth`, `std/math.intToFloat` |
| json_transform | 4-level nested match | Already fixed: `getString`, `getNumber` |
| config_file_parser | `[Json]` to `[string]` | `std/json.filterStrings` |

### Current Failure Patterns

1. **List Index Access (HIGH)**: Agents write `getAt(xs, idx)` in 4/10 failing benchmarks
2. **Last Element (MEDIUM)**: Agents write `getLast(xs)` in 2/10 failing benchmarks
3. **JSON Array Typing (HIGH)**: `getArray` returns `Option[[Json]]` but agents need `[string]`
4. **String Join (MEDIUM)**: No way to join `[string]` into single string with delimiter

---

## Design Decisions

### Stdlib vs Builtins

**Decision**: All functions in this design doc are **stdlib functions**, not builtins.

**Rationale**:
- All proposed functions are implementable in pure AILANG
- No host runtime interop required
- No asymptotic improvement needed (linked list is O(n) for index access regardless)
- No recursion depth concerns for typical agent-scale lists

**Future**: If performance becomes an issue (e.g., `join` being O(n²)), we can swap internals to a builtin without changing the surface API.

### List Type Notation

**Decision**: Use `[a]` consistently everywhere (not `List[a]`).

This affects JSON functions - use `[Json]` not `List[Json]`.

### Strict vs Permissive JSON Array Extraction

**Decision**: Provide **both** patterns for JSON array extraction.

| Pattern | Function | Behavior |
|---------|----------|----------|
| Permissive | `filterStrings(xs)` | Skip non-strings, collect valid ones |
| Strict | `allStrings(xs)` | Fail if any element is not a string |

**Rationale**: Permissive is convenient for flexible data, but can mask problems. Strict is safer for config parsing where bad data should fail loudly.

---

## Proposed Changes

### Priority 1: List Functions (High Impact)

#### 1.1 `nth[a](xs: [a], idx: int) -> Option[a]`

Get element at **0-based** index. Returns `None` if out of bounds or negative index.

```ailang
-- std/list.ail
export pure func nth[a](xs: [a], idx: int) -> Option[a] {
  if idx < 0 then None
  else match xs {
    [] => None,
    [x, ...rest] => if idx == 0 then Some(x) else nth(rest, idx - 1)
  }
}
```

**Complexity**: O(n) - acceptable for linked list structure.

**Impact**: Eliminates ~15 LOC of agent-written `getAt` functions per benchmark.

#### 1.2 `last[a](xs: [a]) -> Option[a]`

Get last element of list.

```ailang
export pure func last[a](xs: [a]) -> Option[a] {
  match xs {
    [] => None,
    [x] => Some(x),
    [_, ...rest] => last(rest)
  }
}
```

**Complexity**: O(n).

**Impact**: Eliminates `getLast` reimplementations in log parsing benchmarks.

#### 1.3 `exists[a](p: (a) -> bool, xs: [a]) -> bool`

Check if any element satisfies predicate. Short-circuits on first match.

```ailang
export pure func exists[a](p: (a) -> bool, xs: [a]) -> bool {
  match xs {
    [] => false,
    [x, ...rest] => if p(x) then true else exists(p, rest)
  }
}
```

**Impact**: Enables declarative membership tests.

**Note**: Takes higher-order function parameter. This is fine for stdlib (pure, no effects) but excluded from SMT verification fragment.

#### 1.4 `findIndex[a](p: (a) -> bool, xs: [a]) -> Option[int]`

Find index of first element matching predicate.

```ailang
export pure func findIndex[a](p: (a) -> bool, xs: [a]) -> Option[int] {
  findIndexHelper(p, xs, 0)
}

pure func findIndexHelper[a](p: (a) -> bool, xs: [a], idx: int) -> Option[int] {
  match xs {
    [] => None,
    [x, ...rest] => if p(x) then Some(idx) else findIndexHelper(p, rest, idx + 1)
  }
}
```

### Priority 2: String Functions (Medium Impact)

#### 2.1 `join(delimiter: string, xs: [string]) -> string`

Join list of strings with delimiter.

```ailang
-- std/string.ail
export pure func join(delimiter: string, xs: [string]) -> string {
  match xs {
    [] => "",
    [x] => x,
    [x, ...rest] => x ++ delimiter ++ join(delimiter, rest)
  }
}
```

**Complexity**: O(n²) due to repeated string concatenation. Acceptable for typical agent-scale lists (< 100 elements). For larger lists, a future `StringBuilder` effect or `concatAll` builtin could provide O(n).

**Impact**: Enables CSV generation, formatted output without manual concatenation.

### Priority 3: JSON Typed Array Helpers (High Impact)

**Note**: All functions use `[Json]` syntax (not `List[Json]`) for consistency.

#### 3.1 Permissive Extraction: `filterStrings`, `filterNumbers`

Extract typed values from JSON array, **skipping** non-matching elements.

```ailang
-- std/json.ail

-- filterStrings: Extract strings, skip non-strings
export func filterStrings(xs: [Json]) -> [string] {
  match xs {
    [] => [],
    [j, ...rest] => match asString(j) {
      Some(s) => [s] ++ filterStrings(rest),
      None => filterStrings(rest)  -- skip non-string
    }
  }
}

-- filterNumbers: Extract numbers, skip non-numbers
export func filterNumbers(xs: [Json]) -> [float] {
  match xs {
    [] => [],
    [j, ...rest] => match asNumber(j) {
      Some(n) => [n] ++ filterNumbers(rest),
      None => filterNumbers(rest)  -- skip non-number
    }
  }
}
```

**Use case**: Flexible data where some elements may be null or wrong type.

#### 3.2 Strict Extraction: `allStrings`, `allNumbers`

Extract typed values from JSON array, **failing** if any element doesn't match.

```ailang
-- allStrings: Extract all as strings, fail if any non-string
export func allStrings(xs: [Json]) -> Option[[string]] {
  match xs {
    [] => Some([]),
    [j, ...rest] => match asString(j) {
      Some(s) => match allStrings(rest) {
        Some(ss) => Some([s] ++ ss),
        None => None
      },
      None => None  -- fail on non-string
    }
  }
}

-- allNumbers: Extract all as numbers, fail if any non-number
export func allNumbers(xs: [Json]) -> Option[[float]] {
  match xs {
    [] => Some([]),
    [j, ...rest] => match asNumber(j) {
      Some(n) => match allNumbers(rest) {
        Some(ns) => Some([n] ++ ns),
        None => None
      },
      None => None  -- fail on non-number
    }
  }
}
```

**Use case**: Config parsing where data integrity matters - bad data should fail loudly.

#### 3.3 Convenience Wrappers

```ailang
-- getStringArray: Get typed string array from object (permissive)
export func getStringArray(obj: Json, key: string) -> Option[[string]] {
  match getArray(obj, key) {
    Some(arr) => Some(filterStrings(arr)),
    None => None
  }
}

-- getStringArrayStrict: Get typed string array (strict - fails on non-strings)
export func getStringArrayStrict(obj: Json, key: string) -> Option[[string]] {
  match getArray(obj, key) {
    Some(arr) => allStrings(arr),
    None => None
  }
}

-- getStringArrayOrEmpty: Get typed string array with empty default
export func getStringArrayOrEmpty(obj: Json, key: string) -> [string] {
  match getStringArray(obj, key) {
    Some(arr) => arr,
    None => []
  }
}

-- Same pattern for numbers
export func getNumberArray(obj: Json, key: string) -> Option[[float]] {
  match getArray(obj, key) {
    Some(arr) => Some(filterNumbers(arr)),
    None => None
  }
}

export func getNumberArrayStrict(obj: Json, key: string) -> Option[[float]] {
  match getArray(obj, key) {
    Some(arr) => allNumbers(arr),
    None => None
  }
}

export func getNumberArrayOrEmpty(obj: Json, key: string) -> [float] {
  match getNumberArray(obj, key) {
    Some(arr) => arr,
    None => []
  }
}
```

**Impact**: Eliminates the `config_file_parser` pattern where agents struggle with `[Json]` to `[string]` conversion.

---

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `std/list.ail` | Add `nth`, `last`, `exists`, `findIndex` | ~50 |
| `std/string.ail` | Add `join` | ~15 |
| `std/json.ail` | Add `filterStrings`, `filterNumbers`, `allStrings`, `allNumbers`, convenience wrappers | ~70 |
| `prompts/v0.6.5.md` | Document new functions | ~40 |
| `prompts/versions.json` | Register v0.6.5 | ~5 |
| **Total** | | ~180 |

---

## Prompt Updates Required

Add to v0.6.5 prompt:

```markdown
**List functions (v0.6.5):**
- `nth(xs, idx)` -> `Option[a]` - Get element at index (**0-based**, returns `None` for negative/out-of-bounds)
- `last(xs)` -> `Option[a]` - Get last element
- `exists(pred, xs)` -> `bool` - Check if any element matches predicate
- `findIndex(pred, xs)` -> `Option[int]` - Find index of first matching element

**String functions (v0.6.5):**
- `join(delimiter, xs)` -> `string` - Join strings with delimiter

**JSON typed array extraction (v0.6.5):**

*Permissive (skip non-matching elements):*
- `filterStrings(jsonArray)` -> `[string]` - Extract strings, skip others
- `filterNumbers(jsonArray)` -> `[float]` - Extract numbers, skip others

*Strict (fail on non-matching elements):*
- `allStrings(jsonArray)` -> `Option[[string]]` - All must be strings or returns `None`
- `allNumbers(jsonArray)` -> `Option[[float]]` - All must be numbers or returns `None`

*Convenience wrappers:*
- `getStringArray(obj, key)` -> `Option[[string]]` - Get typed string array (permissive)
- `getStringArrayStrict(obj, key)` -> `Option[[string]]` - Get typed string array (strict)
- `getStringArrayOrEmpty(obj, key)` -> `[string]` - Get typed string array with empty default
- `getNumberArray(obj, key)` -> `Option[[float]]` - Get typed float array (permissive)
- `getNumberArrayStrict(obj, key)` -> `Option[[float]]` - Get typed float array (strict)
- `getNumberArrayOrEmpty(obj, key)` -> `[float]` - Get typed float array with empty default
```

---

## Acceptance Criteria

### Functional Tests

1. All new functions compile and pass manual tests
2. Example files demonstrate usage:
   - `examples/runnable/list_helpers.ail`
   - `examples/runnable/json_array_extraction.ail`
3. Prompt v0.6.5 documents all new functions

### Property Tests

4. `nth(xs, i)` returns `Some(x)` implies `0 <= i < length(xs)`
5. `last(xs)` equals `nth(xs, length(xs) - 1)` for non-empty lists
6. `filterStrings(xs)` has length `<= length(xs)`
7. `allStrings(xs) == Some(ss)` implies `length(ss) == length(xs)`

### Agent Eval Validation

8. Re-run `json_parse`, `config_file_parser` benchmarks - expect:
   - `json_parse`: < 20 turns (currently 22 with v0.6.4)
   - `config_file_parser`: passes (currently fails)

### Agent-Style Example

9. Include a complete agent-style example showing JSON → typed array → join → output:

```ailang
-- Parse JSON config, extract features array, format output
module examples/runnable/json_config_features

import std/json (decode, getString, getStringArrayOrEmpty)
import std/string (join)
import std/list (length)

export func main() -> () ! {IO} {
  let config = "{\"app\":\"MyApp\",\"features\":[\"logging\",\"auth\",\"api\"]}";
  match decode(config) {
    Ok(json) => {
      let app = match getString(json, "app") { Some(s) => s, None => "unknown" };
      let features = getStringArrayOrEmpty(json, "features");
      let count = length(features);
      let list = join(", ", features);
      print(app ++ " has " ++ show(count) ++ " features: " ++ list)
    },
    Err(e) => print("Parse error: " ++ e)
  }
}
-- Output: MyApp has 3 features: logging, auth, api
```

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Type inference issues with polymorphic functions | Medium | Medium | Test with multiple types in examples |
| Import confusion (agents forget imports) | High | Low | Already addressed in v0.6.4 prompt with import reminder |
| Function name collision | Low | Low | Use descriptive names, check existing stdlib |
| `join` O(n²) performance | Medium | Low | Document complexity; acceptable for agent-scale lists |
| Permissive JSON extraction masks problems | Medium | Medium | Provide strict variants; document trade-offs |

---

## Future Considerations

### Deferred to v0.6.6+

1. **List comprehension syntax sugar**: `[f(x) for x in xs if p(x)]`
2. **Optional chaining**: `obj?.field?.subfield` for nested Option access
3. **Typed JSON schema validation**: Compile-time JSON structure verification
4. **StringBuilder effect**: O(n) string joining for large lists
5. **`concatAll` builtin**: Host-backed efficient string concatenation

### Performance Optimization Path

If `join` or list functions become performance bottlenecks:
1. Keep surface API unchanged
2. Swap internal implementation to host-backed builtin
3. No breaking changes for users

### Open Questions

1. **Should `nth` accept `uint` instead of `int`?** Current design accepts `int` and returns `None` for negative indices. Alternative: use `uint` to make negative indices a type error. **Decision**: Keep `int` - more ergonomic for agents who may compute indices.

2. **Tail-call optimization**: Does the Go backend optimize tail recursion? If not, deep lists could stack overflow. **Mitigation**: Document maximum recommended list size; consider iterative builtin if needed.

---

## Review Notes (2026-01-05)

Incorporated feedback:

1. **Stdlib vs builtins**: Clarified all functions are stdlib, not builtins
2. **List type notation**: Normalized to `[a]` everywhere (not `List[a]`)
3. **`join` complexity**: Documented O(n²) with rationale
4. **Naming**: Changed `mapStrings`→`filterStrings` (reflects skip semantics)
5. **Strict vs permissive**: Added `allStrings`/`allNumbers` strict variants
6. **Convenience variants**: Added `getStringArrayOrEmpty` pattern
7. **Property tests**: Added to acceptance criteria
8. **0-based indexing**: Made explicit in `nth` documentation
9. **Agent example**: Added complete JSON → array → join workflow

---

## References

- [v0.6.2 Eval Results](eval_results/baselines/v0.6.2/)
- [v0.6.4 JSON Helpers Sprint](design_docs/planned/v0_6_4/m-json-helpers.md)
- [std/list.ail](std/list.ail)
- [std/json.ail](std/json.ail)
