# M-DX-MATCH-HOF: Fix `match` Inside Block-Body Lambdas in HOF Arguments

**Status**: Planned
**Version**: v0.8.2
**Priority**: P2 (Medium) - Ergonomics regression in validation and transformation code
**Reported by**: website-builder-demo (2026-03-03)
**GitHub**: N/A

## Summary

When a block-body lambda (`\x. { ... }`) containing a `match` expression is used
as an argument to a higher-order function (HOF), the parser fails with cascading
`PAR_UNEXPECTED_TOKEN` errors. The workaround is to extract the lambda body to a
named helper function.

## Problem Statement

### Failing pattern

```ailang
-- ❌ FAILS: match inside block-body lambda in HOF argument
let result = flatMap(\item. {
  let status = match item with
    | 0 -> Invalid("zero not allowed")
    | _ -> Valid;
  [status]
}, myList)
```

Error output:
```
PAR_UNEXPECTED_TOKEN at line 3: expected next token to be {, got with instead
PAR_UNEXPECTED_TOKEN at line 4: expected next token to be =>, got | instead
...
```

### Working patterns

```ailang
-- ✅ WORKS: Simple let bindings in block-body lambda (no match)
let result = flatMap(\page. {
  let a = page + 1;
  let b = page * 2;
  [a, b]
}, myList)

-- ✅ WORKS: match extracted to named helper
let processItem = \item. match item with
  | 0 -> Invalid("zero not allowed")
  | _ -> Valid

let result = flatMap(\item. [processItem(item)], myList)

-- ✅ WORKS: let...in chains (unwieldy for 3+ bindings)
let result = flatMap(\item.
  let status = match item with
    | 0 -> Invalid("zero not allowed")
    | _ -> Valid in
  [status]
, myList)
```

## Root Cause

The parser tracks nested delimiter depth to know when a HOF argument ends. When
it encounters `match item with`, the `with` keyword is not a standard delimiter
but the parser enters a "match arm" parsing mode that expects `{ ... }` as the
block delimiter.

Inside a block-body lambda `{ ... }`, the `}` is already the terminator for the
outer block. When the parser sees `match item with` inside this block, it attempts
to parse the match arms expecting `{ case | ... }` syntax, which conflicts with
the outer block's `}` delimiter.

The specific confusion: the parser sees `\item. {` and correctly enters block
mode. Inside the block, `let status = match item with` starts a match expression.
The parser then looks for `{` to open the match body (since `with { | ... }` is
the match syntax), but instead finds `\n    | 0 -> ...` — no opening brace —
causing the cascade of errors.

This is the "nested delimiter tracking" bug documented in CLAUDE.md and
LIMITATIONS.md. The limitation was previously noted for `match in blocks` in
general; this report confirms the exact failure mode in HOF argument position.

## Impact

- Real-world: validator code, data transformation pipelines, any multi-step
  per-element processing with conditional logic
- website-builder-demo hit this in `validator.ail` needing per-element match
  expressions inside `flatMap` calls
- Workaround available (extract to helper) but adds friction for inline logic

## Proposed Fix

### Option A: Allow `match` without braces inside block-body lambdas

Extend the block parser to detect `match ... with` followed by `| arm -> ...`
patterns and handle them without requiring the `{ | ... }` form. This requires
knowing we're inside a block-body lambda to correctly track when the outer `}`
terminates the block vs. the match.

**Complexity**: High — requires significant parser state changes.

### Option B: Improve error message and document workaround

Make the error message actionable: detect the `match ... with` pattern in a
block-body context and suggest extracting to a helper function.

**Complexity**: Low — error improvement only.

### Option C: Support `match ... with | arm -> ...` (whitespace-delimited) everywhere

Remove the requirement for `{ | ... }` braces in match bodies when used as
standalone expressions in let bindings. Only require braces when needed to
resolve ambiguity with containing contexts.

**Complexity**: Medium — changes match parsing globally, may introduce new
ambiguities.

### Recommended: Option A (v0.8.2) + Option B (immediate)

1. **Immediate**: Improve error message to detect this pattern and suggest the
   helper function workaround (Option B — low risk, high DX value)
2. **v0.8.2**: Fix parser to correctly handle `match` inside block-body lambdas
   in HOF argument position (Option A)

## Implementation Notes

Key files:
- `internal/parser/parser.go` — `parseBlockExpr()`, `parseMatchExpr()`
- `internal/parser/parser.go` — HOF argument parsing (lambda in function calls)
- `DEBUG_PARSER=1 ailang run test.ail` — trace token positions

The nested delimiter tracking is in the block expression parser. When we enter
a block (`{`), we track depth. The match parser needs to handle the case where
it's inside a block and `}` is already claimed by the outer block.

## Test Cases

```ailang
-- Must pass after fix:
module test/match_in_hof_block

import std/list (flatMap, map)

type Status = Ok | Err(string)

-- Basic case: match in flatMap lambda
let result1 = flatMap(\x. {
  let s = match x with
    | 0 -> Err("zero")
    | _ -> Ok;
  [s]
}, [0, 1, 2])

-- Multi-binding: let + match in map lambda
let result2 = map(\x. {
  let doubled = x * 2;
  let status = match doubled with
    | 0 -> Err("zero")
    | _ -> Ok;
  status
}, [0, 1, 2])
```

## Workaround (Current)

Extract the lambda body to a named helper function:

```ailang
-- Helper function (extracted from inline lambda)
let processItem = \item.
  match item with
  | 0 -> Err("zero not allowed")
  | _ -> Ok

-- Use helper in HOF
let result = flatMap(\item. [processItem(item)], myList)
```
