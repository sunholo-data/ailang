# M-DX-MATCH-HOF: Fix `match` Inside Block-Body Lambdas in HOF Arguments

**Status**: Archived — Not Applicable
**Version**: v0.8.2 (planned), archived v0.18.7
**Priority**: P2 (Medium) — no longer actionable
**Reported by**: website-builder-demo (2026-03-03)
**Archived by**: Investigation 2026-05-09

## Why Archived

**Two separate issues were conflated in this design doc:**

### Issue 1: `match x with { | arm -> ... }` syntax — RESOLVED BY SYNTAX CHANGE

The original failing pattern used the OLD match syntax:
```ailang
-- ❌ OLD syntax (no longer valid)
let status = match item with
  | 0 -> Invalid("zero not allowed")
  | _ -> Valid
```

Error: `expected next token to be {, got with instead`

This syntax was replaced. The **current syntax** `match x { Pattern => body }` does NOT have
this issue — match inside block-body lambdas works correctly:
```ailang
-- ✅ Works in current AILANG
let on_chunk = \chunk. match chunk {
  ContentDelta(text) => emit_event(...),
  Usage(_) => ()
};
```
Verified 2026-05-09 — all test patterns from the design doc pass with current syntax.

### Issue 2: Wrong lambda arrow syntax — MISLEADING ERROR MESSAGE

What developers actually hit (motoko_agent, 2026-05-09) was writing `\chunk ->` instead of `\chunk.`:
```ailang
-- ❌ Wrong: -> is type-arrow syntax, not lambda body separator
let render = \chunk ->
  match chunk { ... }

-- ✅ Correct: use . (dot) as the body separator
let on_chunk = \chunk. match chunk { ... }
```

The parser error for wrong arrow syntax cascades into confusing messages like
`expected '}' to close function body` far from the actual mistake. This was
mistaken for a "match in lambda" parser bug.

**The error message improvement** (better diagnostic for `\x ->` → suggest `\x.`) was
filed as a separate small fix in place of this design doc.

## Original Summary (preserved for reference)

When a block-body lambda (`\x. { ... }`) containing a `match` expression is used
as an argument to a higher-order function (HOF), the parser fails with cascading
`PAR_UNEXPECTED_TOKEN` errors.

The original test case used `with` keyword match syntax that is no longer part of AILANG.

## Workaround (still valid pattern, not required)

Extract match to a named top-level function:
```ailang
func handle_chunk(session_id: string, chunk: StreamChunk) -> () ! {IO} {
  match chunk {
    ContentDelta(text) => emit_event(session_id, "delta", [js(text)]),
    Usage(_) => ()
  }
}
let on_chunk = \chunk. handle_chunk(session_id, chunk);
```
This is documented in `std/ai.ail`'s `stepWithStream` usage example as the
recommended pattern for clarity, but it is not required.
