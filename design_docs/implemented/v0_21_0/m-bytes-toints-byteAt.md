# M-BYTES-TOINTS-BYTEAT: Add byte-to-int extraction primitives to std/bytes

**Status**: Planned
**Target**: v0.23.x (small stdlib addition)
**Priority**: P3 — Low (stdlib gap surfaced by benchmark; not blocking but a clean small addition)
**Estimated**: ~3–5 hours
**Dependencies**: existing `std/bytes` module + Go runtime builtin support

**Commissioning context**: While wiring AILANG into [VeraBench](https://github.com/aallan/vera-bench) as part of [M-VERA-BENCH-INTEGRATION](m-vera-bench-integration.md), the `VB_T2_013_get_char_code` benchmark (ASCII byte value at a string index) surfaced a real stdlib gap. AILANG's `std/bytes` module has all the bytes-construction operations but no way to extract a byte's integer value.

---

## Current State

`std/bytes` currently exports:

```
fromString(s: string) -> bytes
toString(b: bytes) -> string
toBase64(b: bytes) -> string
fromBase64(s: string) -> Option[bytes]
fromBase64URL(s: string) -> Option[bytes]
length(b: bytes) -> int
slice(b: bytes, start: int, len: int) -> Option[bytes]
concat(a: bytes, b: bytes) -> bytes
concatList(xs: [bytes]) -> bytes
fromInts(xs: [int]) -> bytes          ← INT → BYTES exists
```

**Conspicuous absence**: no inverse of `fromInts`. Given a `bytes` value, there is no way to extract any byte's integer value. `slice(b, i, 1)` returns `Option[bytes]` of length 1, and `length` returns the byte count, but the single byte's value is opaque.

## What VeraBench's `get_char_code` Needs

```
Signature: fn get_char_code(s: String, i: Int) -> Nat
Description: Return the ASCII character code at a given index in a string.
Examples:
  get_char_code("A", 0)   => 65
  get_char_code("hello", 0) => 104
  get_char_code("hello", 4) => 111
```

In Python/JavaScript/TypeScript this is trivial: `ord(s[i])`, `s.charCodeAt(i)`. In AILANG today there is no way to express this without external help. The benchmark currently ships a placeholder returning 0.

This is not specific to VeraBench: any AILANG program doing character-level analysis (parsing, hashing, checksum, binary protocol handling) hits the same wall.

## Proposal

Add **two** functions to `std/bytes`. Both are small wrappers around the same Go runtime primitive but cover different idiomatic patterns:

### 1. `toInts(b: bytes) -> [int]`

The natural inverse of `fromInts([int]) -> bytes`. Symmetric, easy to teach:

```ailang
import std/bytes (fromString, toInts)
import std/list (nth)

func get_char_code(s: string, i: int) -> int =
  match nth(toInts(fromString(s)), i) {
    Some(b) => b,
    None => 0   -- out-of-bounds
  }
```

### 2. `byteAt(b: bytes, i: int) -> Option[int]`

Direct indexed access without materialising the whole list. Faster for single-byte lookups; the natural counterpart to `std/list.nth`:

```ailang
import std/bytes (fromString, byteAt)

func get_char_code(s: string, i: int) -> int =
  match byteAt(fromString(s), i) {
    Some(b) => b,
    None => 0
  }
```

Both should ship — they're different shapes for different use cases (whole-list vs single-index). Both are trivial Go builtins (`byteAt` is `b[i]` with bounds check; `toInts` is `for-range` building a Go `[]int`).

## Implementation Sketch

### Go builtins (in `internal/builtins/` or wherever `std/bytes` lives)

```go
// _bytes_byteAt: bytes -> int -> Option[int]
// Returns Some(b[i]) if 0 <= i < len(b), None otherwise.
func bytesByteAt(b []byte, i int) eval.Value {
    if i < 0 || i >= len(b) {
        return mkOptionNone()
    }
    return mkOptionSome(eval.IntValue(int64(b[i])))
}

// _bytes_toInts: bytes -> [int]
// Returns the byte values as a list of non-negative ints (0..255).
func bytesToInts(b []byte) eval.Value {
    out := make([]eval.Value, len(b))
    for i, v := range b {
        out[i] = eval.IntValue(int64(v))
    }
    return mkListValue(out)
}
```

### AILANG stdlib (`std/bytes.ail`)

```ailang
-- ... existing exports ...

-- byteAt: Return the byte value at the given index, or None if out of bounds.
-- Example: byteAt(fromString("A"), 0) => Some(65)
pure func byteAt(b: bytes, i: int) -> Option[int] = _bytes_byteAt(b, i)

-- toInts: Return all byte values as a list of ints (0..255).
-- Inverse of fromInts.
-- Example: toInts(fromString("AB")) => [65, 66]
pure func toInts(b: bytes) -> [int] = _bytes_toInts(b)
```

### Tests

- Round-trip with `fromInts`: `toInts(fromInts(xs)) == xs` for any `[int]` where each value is in 0..255
- Bounds: `byteAt(empty, 0) == None`, `byteAt(b, -1) == None`, `byteAt(b, len(b)) == None`
- Single-char: `byteAt(fromString("A"), 0) == Some(65)`
- UTF-8 multi-byte: `byteAt(fromString("é"), 0)` should return the first UTF-8 byte (`Some(195)`), not the codepoint — document this clearly

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure deterministic primitives |
| A5: Bounded Verification | +1 | `byteAt` returns Option (no panics) — Z3-friendly |
| A7: Machines First | +2 | Fills a real gap; documents the UTF-8 byte-vs-codepoint distinction in the prompt |
| A8: Minimal Syntax | 0 | Pure stdlib addition, no syntax change |
| A10: Composability | +1 | `toInts` makes byte data inspectable via the standard list combinators |

**Net Score: +5** → Proceed.

## Out of Scope

- `charAt(s: string, i: int) -> Option[string]`: returns single-char string, not int. Different feature; punt until requested.
- UTF-8 codepoint extraction (decoding a multi-byte sequence into a Unicode scalar): different problem from byte extraction; punt.
- Mutable bytes / byte arrays: AILANG is immutable; `slice` + `concat` already cover the read-only manipulation cases.

## Open Questions

1. **Naming**: `byteAt` vs `at` vs `byteIndex`? `byteAt` is most explicit about the unit (vs `charAt` which would be Unicode-aware). Default: `byteAt`.
2. **Range check**: returning `Option[int]` is safest but slightly more verbose than panicking. AILANG's convention is Option for fallible operations — keep it.
3. **Should `nth` from std/list have a polymorphic specialisation for bytes**? Probably not — `bytes` isn't a `list[a]` even though it indexes; keeping them distinct mirrors the bytes-vs-list distinction in stdlib design.

## Success Criteria

- [ ] `byteAt` + `toInts` exported from `std/bytes`
- [ ] Go builtins implemented + unit-tested (`make test`)
- [ ] Round-trip property test: `toInts(fromInts(xs)) == xs` for valid `xs`
- [ ] Teaching prompt mentions byteAt for "ASCII / byte extraction" use case
- [ ] `VB_T2_013_get_char_code` placeholder updated to use `byteAt`; VeraBench AILANG tier 2 run_correct improves from 87.5% (7/8 testable) to 100%
