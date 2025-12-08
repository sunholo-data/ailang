# M-JSON: JSON Convenience Builders

**Status**: Planned
**Target**: v0.4.0
**Priority**: P2 (Low - Nice-to-have ergonomics improvement)
**Estimated**: 3-4 days (~20 hours)
**Dependencies**: std/json.decode and std/json.encode exist

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Reduces verbose JSONValue construction |
| Preserve Semantic Clarity | 0 | 0 | Neutral - same semantics, different syntax |
| Increase Determinism | 0 | 0 | No change - deterministic either way |
| Lower Token Cost | + | +1 | Fewer tokens needed for JSON construction |
| **Net Score** | | **+2** | **Decision: Move forward (if time permits)** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- `std/json.decode` and `std/json.encode` work well
- Constructing JSONValue ADTs manually is verbose
- API examples require explicit JSONObject/JSONArray construction

**Impact:**
- **DX**: Verbose JSON construction in API code
- **Example clarity**: `ai_call.ail` example has repetitive boilerplate
- **AI code gen**: Models generate verbose JSON construction code

**Current pain:**
```ailang
-- Manual JSONValue construction (verbose)
let body = encode(
  JSONObject([
    ("model", JSONString("gpt-4")),
    ("messages", JSONArray([
      JSONObject([
        ("role", JSONString("user")),
        ("content", JSONString(prompt))
      ])
    ])),
    ("max_tokens", JSONNumber(100.0))
  ])
)
```

## Goals

**Primary Goal:** Add convenience functions for building JSONValue ADTs to reduce boilerplate

**Success Metrics:**
- 50% reduction in JSON construction LOC
- `ai_call.ail` example becomes more readable
- API code is clearer and more maintainable

## Solution Design

### Overview

Add helper functions to `std/json` module for building JSONValue ADTs:

- `jo(fields)` - JSON Object constructor
- `ja(items)` - JSON Array constructor
- `js(str)` - JSON String wrapper
- `jnum(n)` - JSON Number wrapper
- `jbool(b)` - JSON Bool wrapper
- `jnull()` - JSON Null constant
- `kv(key, value)` - Key-value pair helper

### Architecture

**Components:**

1. **stdlib Extensions** (`stdlib/std/json.ail`)
   - Add convenience builder functions
   - Wrap existing JSONValue ADT

2. **No compiler changes needed!**
   - Pure library functions
   - No new syntax
   - No type system changes

### Implementation

**All functions in stdlib/std/json.ail:**

```ailang
module std/json

-- Existing JSONValue ADT (from M-LANG-JSON-DECODE)
type JSONValue =
  | JSONString(string)
  | JSONNumber(float)
  | JSONBool(bool)
  | JSONNull
  | JSONArray([JSONValue])
  | JSONObject([(string, JSONValue)])

-- NEW: Convenience builders

-- JSON Object builder
export func jo(fields: [(string, JSONValue)]) -> JSONValue {
  JSONObject(fields)
}

-- JSON Array builder
export func ja(items: [JSONValue]) -> JSONValue {
  JSONArray(items)
}

-- JSON String wrapper
export func js(str: string) -> JSONValue {
  JSONString(str)
}

-- JSON Number wrapper
export func jnum(n: float) -> JSONValue {
  JSONNumber(n)
}

-- JSON Bool wrapper
export func jbool(b: bool) -> JSONValue {
  JSONBool(b)
}

-- JSON Null constant
export func jnull() -> JSONValue {
  JSONNull
}

-- Key-value pair helper (for clarity)
export func kv(key: string, value: JSONValue) -> (string, JSONValue) {
  (key, value)
}
```

**That's it!** ~30 LOC of pure AILANG code. No compiler changes needed.

### Implementation Plan

**Phase 1: Add Functions** (~2 hours)
- [ ] Add builder functions to stdlib/std/json.ail
- [ ] Update module exports
- [ ] Basic sanity tests

**Phase 2: Update Examples** (~8 hours)
- [ ] Refactor ai_call.ail to use builders
- [ ] Update demo_openai_api.ail if needed
- [ ] Add JSON builder examples to docs
- [ ] Compare before/after LOC

**Phase 3: Tests & Documentation** (~10 hours)
- [ ] Unit tests for each builder function
- [ ] Integration tests (build + encode roundtrip)
- [ ] Update teaching prompt with builders
- [ ] CHANGELOG entry

### Files to Modify/Create

**Modified files:**
- `stdlib/std/json.ail` - Add convenience functions (~30 LOC)

**New test files:**
- `stdlib/std/json_builders_test.ail` - Tests (~100 LOC)

**Total new code: ~130 LOC**

## Examples

### Example 1: OpenAI API Request

**Before (verbose):**
```ailang
import std/json (encode, JSONObject, JSONString, JSONArray, JSONNumber)

let body = encode(
  JSONObject([
    ("model", JSONString("gpt-4o-mini")),
    ("messages", JSONArray([
      JSONObject([
        ("role", JSONString("user")),
        ("content", JSONString(prompt))
      ])
    ])),
    ("max_tokens", JSONNumber(100.0))
  ])
)
```

**After (concise):**
```ailang
import std/json (encode, jo, ja, js, jnum, kv)

let body = encode(
  jo([
    kv("model", js("gpt-4o-mini")),
    kv("messages", ja([
      jo([
        kv("role", js("user")),
        kv("content", js(prompt))
      ])
    ])),
    kv("max_tokens", jnum(100.0))
  ])
)
```

**Improvement:** 40% fewer characters, much more readable

### Example 2: Simple Response

**Before:**
```ailang
let response = encode(
  JSONObject([
    ("status", JSONString("ok")),
    ("count", JSONNumber(float(len(items)))),
    ("items", JSONArray(map(\x. JSONString(x.name), items)))
  ])
)
```

**After:**
```ailang
let response = encode(
  jo([
    kv("status", js("ok")),
    kv("count", jnum(float(len(items)))),
    kv("items", ja(map(\x. js(x.name), items)))
  ])
)
```

### Example 3: Nested Objects

**Before:**
```ailang
JSONObject([
  ("user", JSONObject([
    ("id", JSONNumber(float(user.id))),
    ("name", JSONString(user.name)),
    ("email", JSONString(user.email))
  ])),
  ("metadata", JSONObject([
    ("created", JSONString(show(now()))),
    ("version", JSONNumber(1.0))
  ]))
])
```

**After:**
```ailang
jo([
  kv("user", jo([
    kv("id", jnum(float(user.id))),
    kv("name", js(user.name)),
    kv("email", js(user.email))
  ])),
  kv("metadata", jo([
    kv("created", js(show(now()))),
    kv("version", jnum(1.0))
  ]))
])
```

## Success Criteria

- [ ] All builder functions work correctly
- [ ] `ai_call.ail` example updated and works
- [ ] 40-50% reduction in JSON construction LOC
- [ ] Roundtrip tests pass (build → encode → decode → verify)
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Teaching prompt includes builder examples

## Testing Strategy

**Unit tests:**
- Each builder function produces correct JSONValue
- `kv` creates proper tuple
- Type checking works correctly

**Integration tests:**
- Build complex nested JSON
- Encode + decode roundtrip
- Compare with manual JSONValue construction (should be equivalent)

**Example tests:**
- `ai_call.ail` example runs correctly
- Generated JSON matches expected format

## Non-Goals

**Not in this feature:**
- **JSON quasiquotes** - Use `json{}` syntax (see M-QUASI design doc)
- **JSON schema validation** - Out of scope
- **JSON pretty-printing** - Use external tools
- **JSON merging/patching** - Future feature
- **Type inference from JSON** - Would require compile-time evaluation

## Timeline

**Day 1** (8 hours):
- Phase 1: Add Functions (2 hours)
- Phase 2: Update Examples (6 hours)

**Day 2** (12 hours):
- Phase 3: Tests & Documentation (12 hours)

**Total: ~20 hours across 2-3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Name conflicts** | Low | Choose short, unlikely-to-conflict names (jo, ja, js) |
| **Type errors** | Low | Simple wrappers - hard to get wrong |
| **Adoption** | Low | Show clear before/after examples |

## References

- **JSON convenience APIs**:
  - Clojure: `(json/generate-string {:key "value"})`
  - JavaScript: `JSON.stringify({key: "value"})`
  - Elm: json package with `object`, `string`, `int` helpers

- **Example files requiring this feature**:
  - `examples/experimental/ai_call.ail`
  - `examples/experimental/demo_openai_api.ail`

- **Related design docs**:
  - [M-LANG-JSON-DECODE](../../implemented/M-LANG-JSON-DECODE.md) - JSON decoding (existing)
  - [v0_4_0_net_enhancements.md](v0_4_0_net_enhancements.md) - HTTP/Net features
  - [m-quasi-typed-quasiquotes.md](../v0_4_2/m-quasi-typed-quasiquotes.md) - JSON quasiquotes (future)

## Future Work

**v0.4.2+:**
- JSON quasiquotes (`json{ "key": ${value} }`) - see M-QUASI design doc
- JSON schema validation
- JSON diff/patch operations
- Typed JSON builders (infer ADT from JSON structure)

## Decision: Implement or Defer?

**Recommendation**: **Implement in v0.4.0** - Low cost, high value.

**Rationale:**
- **Low complexity**: ~130 LOC, pure library code, no compiler changes
- **High value**: Significantly improves API code readability
- **Net score: +2** (passes AI-first criteria)
- **Quick win**: Can be done in 2-3 days
- **Unblocks**: Makes API examples much cleaner

**Implementation strategy:**
- Implement after Net enhancements (depends on them for context)
- Can be done by any contributor (pure AILANG code)
- Good first issue for learning AILANG stdlib

---

**Document created**: 2025-10-26
**Last updated**: 2025-10-26
