# M-JSON-ACCESSORS: Enable JSON Accessor Functions

**Status:** IMPLEMENTED
**Version:** v0.5.11
**Date:** 2025-12-15

## Summary

Enabled previously commented-out JSON accessor functions in `std/json.ail` and documented them in the teaching prompt. These functions were blocked by a constructor scope issue (fixed in v0.4.10) but never uncommented.

## Problem

JSON benchmarks were failing across nearly all models:

| Benchmark | Pass Rate (Before) |
|-----------|-------------------|
| json_parse | 2/9 models |
| json_transform | 1/9 models |
| json_encode | 0/9 models |

**Root cause:**
1. JSON accessor functions were commented out in `std/json.ail` since v0.3.14
2. The teaching prompt only documented `encode`/`decode`, not how to extract values
3. AIs had no way to know these functions existed

The functions were originally blocked by a "constructor scope in module system" issue where `Some(...)` and `None` constructors couldn't be resolved when called from imported helper functions. This was fixed in commit `f79615bd` (v0.4.10) but the functions were never re-enabled.

## Solution

### 1. Enabled Accessor Functions in std/json.ail

**Uncommented functions:**
- `get(obj, key) -> Option[Json]` - Get value by key
- `has(obj, key) -> bool` - Check if key exists
- `getOr(obj, key, default) -> Json` - Get with fallback
- `asString(j) -> Option[string]` - Extract string
- `asNumber(j) -> Option[float]` - Extract number
- `asBool(j) -> Option[bool]` - Extract boolean
- `asArray(j) -> Option[List[Json]]` - Extract array
- `asObject(j) -> Option[List[{key: string, value: Json}]]` - Extract object

**New functions added:**
- `keys(obj) -> List[string]` - Get all keys from object
- `values(obj) -> List[Json]` - Get all values from object

Note: `keys()` and `values()` use helper functions instead of inline lambdas due to type inference limitations with field access in lambdas.

### 2. Updated Teaching Prompt (v0.5.10)

Added comprehensive JSON accessor documentation including:
- Example showing how to decode and extract values
- Function reference table with signatures
- Proper import statements

### 3. Caveat for keys/values

The `keys()` and `values()` functions require callers to also `import std/list (map)` due to a module runtime resolution issue. This is documented in the source with NOTE comments.

## Files Changed

- `std/json.ail` - Uncommented accessor functions, added keys/values with helpers
- `cmd/ailang/prompts/v0.5.10.md` - Added JSON accessor documentation
- `prompts/v0.5.10.md` - Added JSON accessor documentation (canonical copy)
- `internal/pipeline/testdata/builtin_types.golden` - Updated (unrelated: new conversion builtins)

## Testing

All functions verified working:
1. Type checking passes
2. Runtime execution works correctly
3. Full test suite passes

## Expected Impact

JSON benchmark pass rates should significantly improve in the next eval baseline since AIs now:
1. Know the accessor functions exist (documented in prompt)
2. Can use them to extract values from parsed JSON
3. Have working examples to follow

## Related

- Original blocker: Constructor scope in module system (v0.3.14)
- Fix commit: `f79615bd` - "Fix Blocker 1: $adt cross-module constructor resolution" (v0.4.10)
- CHANGELOG entry in v0.3.14: "Module Constructor Scope: ADT constructors work for pattern matching but not construction"
