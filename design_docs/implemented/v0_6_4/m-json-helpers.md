# M-JSON-HELPERS: Convenience Functions for JSON Access

**Status**: Planned
**Version**: v0.6.4
**Date**: 2026-01-05
**Priority**: P0
**Motivation**: Reduce agent turns from 47 to <20 for JSON benchmarks

---

## Problem Statement

Agent transcripts show excessive nesting when working with JSON:

```ailang
-- Current: 4 levels of nesting for one field
match get(obj, "name") {
  Some(nameJson) => {
    match asString(nameJson) {
      Some(name) => doSomething(name),
      None => handleError()
    }
  },
  None => handleError()
}
```

The `json_parse` benchmark took **47 turns** to complete with v0.6.3 prompt, mostly due to agents struggling with this pattern.

---

## Proposed Solution

Add convenience functions to `std/json` that combine `get` + `asX`:

```ailang
-- New helper functions in std/json
export func getString(obj: Json, key: string) -> Option[string]
export func getNumber(obj: Json, key: string) -> Option[float]
export func getInt(obj: Json, key: string) -> Option[int]  -- uses floatToInt internally
export func getBool(obj: Json, key: string) -> Option[bool]
export func getArray(obj: Json, key: string) -> Option[[Json]]
export func getObject(obj: Json, key: string) -> Option[Json]
```

### Usage After Change

```ailang
-- New: 2 levels of nesting
match getString(obj, "name") {
  Some(name) => doSomething(name),
  None => handleError()
}
```

---

## Implementation

### File Changes

**`std/json.ail`** - Add 6 helper functions:

```ailang
-- getString: Get string value from JSON object by key
export func getString(obj: Json, key: string) -> Option[string] =
  match get(obj, key) {
    Some(j) => asString(j),
    None => None
  }

-- getNumber: Get float value from JSON object by key
export func getNumber(obj: Json, key: string) -> Option[float] =
  match get(obj, key) {
    Some(j) => asNumber(j),
    None => None
  }

-- getInt: Get int value from JSON object by key (truncates float)
import std/math (floatToInt)
export func getInt(obj: Json, key: string) -> Option[int] =
  match get(obj, key) {
    Some(j) => match asNumber(j) {
      Some(f) => Some(floatToInt(f)),
      None => None
    },
    None => None
  }

-- getBool: Get bool value from JSON object by key
export func getBool(obj: Json, key: string) -> Option[bool] =
  match get(obj, key) {
    Some(j) => asBool(j),
    None => None
  }

-- getArray: Get array value from JSON object by key
export func getArray(obj: Json, key: string) -> Option[[Json]] =
  match get(obj, key) {
    Some(j) => asArray(j),
    None => None
  }

-- getObject: Get nested object from JSON object by key (returns Json)
export func getObject(obj: Json, key: string) -> Option[Json] =
  get(obj, key)
```

### Prompt Update

**`prompts/v0.6.4.md`** - Document the helpers:

```markdown
**JSON convenience functions** (std/json):
- `getString(obj, key) -> Option[string]` - Get string field
- `getNumber(obj, key) -> Option[float]` - Get numeric field
- `getInt(obj, key) -> Option[int]` - Get integer field (truncates)
- `getBool(obj, key) -> Option[bool]` - Get boolean field
- `getArray(obj, key) -> Option[[Json]]` - Get array field
- `getObject(obj, key) -> Option[Json]` - Get nested object
```

---

## Testing

### Unit Tests

```ailang
-- test_json_helpers.ail
import std/json (decode, getString, getNumber, getInt, getBool, getArray)

test "getString extracts string field" = {
  match decode("{\"name\":\"Alice\"}") {
    Ok(obj) => getString(obj, "name") == Some("Alice"),
    Err(_) => false
  }
}

test "getInt converts float to int" = {
  match decode("{\"age\":30}") {
    Ok(obj) => getInt(obj, "age") == Some(30),
    Err(_) => false
  }
}

test "getString returns None for missing key" = {
  match decode("{\"name\":\"Alice\"}") {
    Ok(obj) => getString(obj, "missing") == None,
    Err(_) => false
  }
}
```

### Benchmark Validation

Re-run `json_parse` benchmark with Haiku after changes:
- **Target**: <20 turns (down from 47)
- **Command**: `ailang eval-suite -agent -benchmarks json_parse -models claude-haiku-4-5`

---

## Success Metrics

| Metric | Before (v0.6.3) | Target (v0.6.4) |
|--------|-----------------|-----------------|
| `json_parse` turns | 47 | <20 |
| `config_file_parser` turns | 26 | <15 |
| JSON-related failures | 6 | 0-2 |

---

## Dependencies

- `std/math.floatToInt` (added in v0.6.3) - Required for `getInt`

## Risks

- **Import complexity**: `std/json` will need to import `std/math` for `getInt`
- **Circular imports**: Need to verify no circular dependency issues

---

## Alternatives Considered

1. **Monadic bind operator**: `obj >>= get "name" >>= asString` - Too complex for teaching
2. **Record coercion**: Auto-convert JSON to typed records - Requires type inference changes
3. **Exception-based API**: `getStringOrThrow` - Against AILANG's explicit error handling philosophy

The convenience function approach is simplest and matches agent expectations.
