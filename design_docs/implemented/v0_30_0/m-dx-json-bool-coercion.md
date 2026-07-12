# M-DX-JSON-BOOL: JSON Boolean Coercion and Firestore Encoding Consistency

**Status**: Implemented (in-repo half — v0.30.0, mission iteration 16); Phase 1 PARKED (out-of-repo)
**Target**: v0.9.5 → landed v0.30.0
**Priority**: P3 (DX — package-level bug with std/json implications)
**Estimated**: 0.5 days
**Dependencies**: M-DX-XPKG-RESOLVE (the asString cross-package bug blocks testing this fix)
**Milestone ID**: M-DX-JSON-BOOL
**Created**: 2026-03-25
**Source**: DocParse agent message `eafa7e06` (boolVal encoding inconsistency)

---

## Outcome (mission iteration 16, 2026-07-12)

Reality-check against the repo split this doc into an in-repo half and an out-of-repo half:

- **`std/json` already had the correct primitives** — `jb(b)` (JBool) and `asBool` (JBool → Option)
  exist and round-trip correctly. `asBool(jb(true)) == Some(true)` was verified live. So the core
  encoder/decoder are not broken; the bug is *use of the wrong constructor* at the package layer.
- **Phase 2 LANDED**: added **`asBoolLoose(j) -> Option[bool]`** (accepts `JBool` OR
  `JString "true"/"false"`, else `None` — structured failure, never a silent default). This is the
  "system boundary" resilience helper (A12). Shipped with runnable example
  `examples/runnable/json_bool_encoding.ail` and Go regression test
  `internal/repl/json_asboolloose_test.go` (7 cases incl. the Firestore `booleanValue` round-trip;
  non-vacuity proven — the same test fails on stringified booleans if `asBool` is substituted).
- **Phase 3 (teaching prompt)** folded into the example's header comment (`jb` vs `js` footgun)
  rather than editing the embedded prompt — prompt-budget is GATED in the v1 mission (prompt-diet).
- **Phase 1 (the actual data-integrity bug) PARKED**: `sunholo/firestore/fields.ail` lives in the
  **separate `ailang-packages` repo**, absent from this checkout, and the doc's own dependency
  M-DX-XPKG-RESOLVE gates testing it. The one-line encoder fix (`js("true")` → `jb(b)`) plus the
  decoder swap to `asBoolLoose` is a coordinator/human task in that repo — see the mission log
  iteration 16 and GH issue #329. `asBoolLoose` now exists so that decoder fix is a drop-in.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Encoder and decoder should be inverse operations — currently they're not |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Type-correct encoding means local verification catches mismatches |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | AI agents building Firestore integrations hit this silently |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Encode/decode should compose: `decode(encode(x)) == x` |
| A11: Structured Failure | 0 | No change |
| A12: System Boundary | +1 | Firestore REST API is a system boundary — encoding must match its expectations |

**Net Score: +5** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): This is a determinism FIX — encode/decode roundtrip currently loses data
- [x] A10 (Composability): Encoder/decoder pair must be compositionally correct

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Partially. The core issue is that `sunholo/firestore/fields` uses `js("true")` (JSON string) to encode booleans, but the Firestore REST API normalizes `booleanValue` fields to actual JSON booleans. This creates an encode/decode mismatch.

**Pattern**: Any external API that normalizes JSON types on write will break AILANG packages that use the wrong JSON constructor. This is a **package-level** bug, not a core language bug. But `std/json` could provide better helpers to prevent it.

**Audit of related functions in std/json:**

| Constructor | Purpose | Correct Use |
|-------------|---------|-------------|
| `js(s)` | JSON string: `"hello"` | String values |
| `jn(n)` | JSON number: `42` | Numeric values |
| `jb(b)` | JSON boolean: `true`/`false` | **Boolean values** |
| `jNull` | JSON null | Null values |

The package mistakenly uses `js("true")` instead of `jb(true)` for boolean fields.

---

## Problem Statement

### The Bug

In `sunholo/firestore/fields`, the `boolVal` encoder and `asBoolField` decoder are inconsistent:

```ailang
-- ENCODER: wraps boolean as JSON STRING (wrong)
func boolVal(b: bool) -> Json {
  jo([kv("booleanValue", js(if b then "true" else "false"))])
}

-- DECODER: checks for string "true" (matches encoder but not Firestore)
func asBoolField(fields: Json, fieldName: string) -> bool {
  match asStr(fields, fieldName) {
    "" => false,
    s  => s == "true"
  }
}
```

**What Firestore does**: The REST API accepts `{"booleanValue": "true"}` but normalizes it to `{"booleanValue": true}` (actual JSON boolean) on storage. When reading back:
- Encoder sent: `{"booleanValue": "true"}` (string)
- Firestore returns: `{"booleanValue": true}` (boolean)
- Decoder expects: string `"true"` via `asString`
- Decoder gets: `JBool(true)` via JSON parsing -> `asString` returns `None` -> decoded as `false`

### Impact

Every boolean field in Firestore packages silently reads back as `false` after the first write, regardless of the actual value. This is a data integrity bug.

---

## Proposed Fix

### Phase 1: Fix the Package (sunholo/firestore/fields)

**Encoder fix** — use `jb()` instead of `js()`:
```ailang
func boolVal(b: bool) -> Json {
  jo([kv("booleanValue", jb(b))])
}
```

**Decoder fix** — use `asBool` from std/json instead of string comparison:
```ailang
func asBoolField(fields: Json, fieldName: string) -> bool {
  match get(fields, fieldName) {
    None => false,
    Some(field) => match get(field, "booleanValue") {
      None => false,
      Some(bv) => match asBool(bv) {
        Some(b) => b,
        None => false  -- fallback for string "true"/"false"
      }
    }
  }
}
```

**Location**: `ailang-packages/packages/firestore/fields.ail`

### Phase 2: Add `asBoolLoose` to std/json (optional)

For resilience against APIs that return booleans as strings, add a helper:

```ailang
-- Accepts both JBool(true) and JString("true")
export func asBoolLoose(j: Json) -> Option[bool] {
  match j {
    JBool(b) => Some(b),
    JString("true") => Some(true),
    JString("false") => Some(false),
    _ => None
  }
}
```

This is the "system boundary" pattern — at the boundary with external APIs, be liberal in what you accept.

**Location**: `std/json.ail`

### Phase 3: Teaching Prompt Update

Add a note about JSON boolean encoding:

```
-- Boolean encoding: use jb(), NOT js("true")
jb(true)          -- correct: JBool(true) -> JSON true
js("true")        -- WRONG for booleans: JString("true") -> JSON "true"
```

---

## Test Plan

1. **Roundtrip test**: `asBool(jb(true))` == `Some(true)`
2. **Loose coercion test**: `asBoolLoose(js("true"))` == `Some(true)`
3. **Firestore integration**: Write a boolean, read it back, verify correct value
4. **Package test**: Update `sunholo/firestore/fields` tests to cover boolean roundtrip

---

## Key Files

| File | Purpose |
|------|---------|
| `ailang-packages/packages/firestore/fields.ail` | Package with the bug |
| `std/json.ail` | Core JSON helpers (asBool, potential asBoolLoose) |
| `prompts/v0.9.0.md` | Teaching prompt to update |
