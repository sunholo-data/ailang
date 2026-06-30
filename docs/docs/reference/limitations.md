---
sidebar_position: 5
title: Known Limitations
description: Known limitations, workarounds, and design constraints in AILANG
---

# AILANG Known Limitations

This document tracks known limitations, workarounds, and design constraints in AILANG.

For features that have been implemented, see [Design Documents](/docs/design-docs).

Last verified against AILANG **v0.14.2**.

---

## Type System Limitations

### Y-Combinator and Recursive Lambdas (By Design)

**Status**: Design constraint, not a bug

The Y-combinator and similar recursive lambda expressions fail with "occurs check" errors:

```ailang
-- This fails:
let Y = \f. (\x. f(x(x)))(\x. f(x(x))) in
-- Error: occurs check failed: type variable α occurs in (α → β)
```

**Root Cause**: Hindley-Milner type inference prevents infinite types to ensure decidability. The Y-combinator requires a recursive type `α = α → β`, which would create an infinite type.

**Why This Exists**:
1. **Type Inference Decidability** — Allowing infinite types makes type inference undecidable
2. **AI-Friendly Design** — AILANG prioritizes deterministic, verifiable type checking
3. **Semantic Clarity** — Named recursion is more explicit than anonymous recursion

**Workaround**: Use named recursive functions:

```ailang
func factorial(n: int) -> int =
  if n <= 1 then 1 else n * factorial(n - 1)
```

Named `func` recursion is also supported by `ailang verify` via bounded unrolling
(`--verify-recursive-depth N`, v0.8.0+).

---

### WASM Type-Checker Depth Limit (By Design, WASM-only)

**Status**: WASM-specific constraint with structured error
**Since**: v0.22.x
**Affects**: AILANG modules compiled to WebAssembly (browser demos using `wasm/ailang.wasm`). **CLI is unaffected.**

The WASM-compiled type-checker is bound by the host JavaScript engine's call-stack limit (~10–15K frames in Node, Chromium, Firefox). Modules with deeply-recursive type structure exceed it. On native Go the CLI handles them fine because goroutine stacks grow dynamically up to 1 GiB; on WASM we hit the cliff.

**Example failure**:

```
loadModule cognitive_commons/services/citizen failed:
WASM type-checker depth budget exceeded (5000 frames) in module
"cognitive_commons/services/citizen".

This module's type structure recurses too deeply for the WASM host stack.
The same source likely works on the AILANG CLI — this limit is specific
to browser execution.
```

**Common triggers**:
1. Triple-nested `match` patterns (match inside match inside match)
2. Multiple back-to-back matches on the same tagged-union value with field access:
   ```ailang
   -- Each line is a separate match on the same Result — the type-checker
   -- repeats tagged-union analysis on `s` three times.
   let x = match r { Ok(s) => s.x, Err(_) => 0.0 };
   let y = match r { Ok(s) => s.y, Err(_) => 0.0 };
   let z = match r { Ok(s) => s.z, Err(_) => 0.0 };
   ```
3. Long chains of intra-package imports with many destructured constructors

**Workarounds** (in order of simplicity):

1. **Flatten nested matches** into sequential `let`-bindings:
   ```ailang
   -- Before: triple-nested
   match decode(raw) {
     Err(e) => Err(e),
     Ok(j) => match getString(j, "error") {
       Some(p) => Err(p),
       None => match getString(j, "text") { ... }
     }
   }

   -- After: flat
   match decode(raw) {
     Err(e) => Err(e),
     Ok(j) => {
       let err = optional_string(j, "error");
       if length(err) > 0 then Err(err)
       else match getString(j, "text") { ... }
     }
   }
   ```

2. **Extract a helper** that does one match and returns a record:
   ```ailang
   pure func unpack(r: Result[Score, string]) -> { has: bool, x: float, y: float } =
     match r {
       Ok(s)  => { has: true,  x: s.x, y: s.y },
       Err(_) => { has: false, x: 0.0, y: 0.0 }
     }
   -- Now `unpack(r)` returns everything at once; one match instead of three.
   ```

3. **Split the function** into smaller top-level functions — each gets its own type-check pass.

**Why This Exists**:
1. **Host-stack limit** — Go compiled to WebAssembly shares the host JavaScript engine's call stack, which has a fixed limit (~10K frames in Node/Chromium). Native Go has dynamically-growable per-goroutine stacks; WASM does not.
2. **Silent failure mode otherwise** — Before this limit, the browser would freeze for 80–120 seconds before the JS engine threw `Maximum call stack size exceeded`. The structured error fires immediately with actionable workarounds instead.
3. **Workarounds are idiomatic** — Helper extraction and flat matches are the AILANG style anyway; the cliff catches patterns that are also harder to read.

**History**: First hit 2026-05-20 by `cognitive_commons/services/citizen.ail` in the demos repo. The half-day diagnostic trail (silent freeze, no console error, DevTools locked) led to the limit being added so future demo authors get a clear error in seconds instead of an opaque hang. Full postmortem: [demos/debug-notes/wasm-citizen-stack-overflow.md](https://github.com/sunholo-data/ailang-demos/blob/main/debug-notes/wasm-citizen-stack-overflow.md).

**Prevention guide**: the five patterns that trigger this cliff are actually general AILANG style — pure core, single-effect leaves, multi-effect orchestrator, destructure once, flat matches. Run `ailang prompt | grep -B 1 -A 200 "Idiomatic AILANG: Pure Core"` for the full guide with bad/good examples. (The WASM budget is just the toolchain-side enforcement mechanism; the patterns themselves are the architecture, so they live in the syntax prompt. The devtools prompt has a short toolchain-side pointer at `ailang devtools-prompt | grep -A 20 "WASM Type-Checker Budget"`.) The patterns produce more readable, more testable code regardless of target — the WASM cliff is the universe nudging you toward the style.

**Headless smoke test**: [demos/scripts/wasm-loadmodule-harness.js](https://github.com/sunholo-data/ailang-demos/blob/main/scripts/wasm-loadmodule-harness.js) runs the actual WASM in Node and exits with code 4 when this limit fires (vs 0 for clean modules). CI-friendly.

**Future improvement**: Converting the type-checker to iterative work-stack passes would remove the limit entirely. Tracked in [M-WASM-TYPECHECK-ITERATIVE](https://github.com/sunholo-data/ailang/blob/dev/design_docs/deferred/m-wasm-typecheck-iterative.md) (deferred — high-risk refactor for a low-frequency cliff).

---

### Duplicate Record Types with Identical Fields

**Status**: Known limitation with workarounds
**Since**: v0.5.10
**Affects**: Go code generation when multiple record types share identical field structures

When multiple record types declare the same field names and types, the Go
codegen may pick the wrong struct because `GetRecordTypeByFields` returns the
first match:

```ailang
-- starmap.ail
export type Vec3 = { x: float, y: float, z: float }

-- celestial.ail
export type SystemPos = { x: float, y: float, z: float }

func initSystem() -> StarSystem {
    { position: { x: 0.0, y: 0.0, z: 0.0 }, ... }
    -- May generate &Vec3{} instead of &SystemPos{}
}
```

**Workarounds**:

1. **Merge duplicates** if semantically equivalent — keep one type and import it everywhere.
2. **Rename to be unique** — `GalacticCoord` vs `SystemPos` instead of two `Vec3`-shaped types.
3. **Add a discriminator field** — e.g. `_tag: string` — to break the structural tie.

**See Also**: [implemented/v0_5_10/m-codegen-nested-record-type.md](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_5_10/m-codegen-nested-record-type.md)

---

## Parser Limitations

### If-Else Branches Require Explicit Braces

**Status**: Design constraint (improved error message in v0.5.9)
**Affects**: Multi-statement branches in if-else expressions

AILANG is not layout-sensitive, so multi-statement branches must be wrapped in
explicit braces. Without them, only the first `let` is parsed as the branch:

```ailang
-- Fails:
if x > maxX then [] else
    let v = x * 2;
    let rest = buildList(x + 1, maxX);
    v :: rest
-- Error: if-else branches require explicit braces when using let bindings
```

**Fix**:

```ailang
if x > maxX then [] else {
    let v = x * 2;
    let rest = buildList(x + 1, maxX);
    v :: rest
}
```

Single-expression branches don't need braces.

---

### `match` Inside Block-Body Lambdas in HOF Arguments

**Status**: Known parser bug — [Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_13_0/m-dx-match-in-hof-block-lambda.md)
**Affects**: Inline `\x. { ... match ... with ... }` lambdas passed directly to higher-order functions

The parser's nested-delimiter tracking gets confused when a `match ... with`
expression appears inside a brace-block lambda passed to a HOF:

```ailang
-- Fails: PAR_UNEXPECTED_TOKEN at 'with'
let result = flatMap(\item. {
  let status = match item with
    | 0 -> Err("zero")
    | _ -> Ok;
  [status]
}, myList)
```

**Workaround**: Extract the lambda body into a named helper:

```ailang
let processItem = \item.
  match item with
  | 0 -> Err("zero")
  | _ -> Ok

let result = flatMap(\item. [processItem(item)], myList)
```

Simple `let` bindings inside block-body lambdas (without `match`) are fine.

---

## Language Feature Gaps

### Error Propagation Operator (`?`)

**Status**: Planned — [Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_13_0/m-error-propagation.md)

The `?` operator for early return on `Result` errors is designed but not yet
implemented. For now, use explicit `match` on `Result`:

```ailang
match readFile(path) {
  Ok(contents) => process(contents),
  Err(e) => Err(e)
}
```

---

### Typed Quasiquotes

**Status**: Planned — [Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-quasi-typed-quasiquotes.md)

Typed quasiquotes for deterministic AST templates and secure string templating
are designed but not yet implemented. Use string interpolation (`"${expr}"`,
v0.12.1+) or `concat([..])` for now.

---

### CSP Concurrency

**Status**: Deferred — [Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-csp-session-types.md)

CSP-style concurrency with channels and session types is deferred to v1.0.0+.
AILANG currently focuses on deterministic, single-threaded execution. The
runtime does support effect-level concurrency primitives in some contexts (see
the `Concurrency` effect), but full CSP with session types remains future work.

---

### Interactive stdin / Keyboard Input

**Status**: Line input works; two narrower gaps remain (see the three rows below).

`std/io` provides `readLine()`, which reads one line from stdin and blocks until the user
presses Enter — covering prompt-and-read and REPL-style input. `std/stream.asyncReadStdinLines`
additionally provides a **concurrent, non-blocking line source** for `selectEvents` dispatch.
So "no stdin" is a myth: line-oriented input is fully supported today.

Two distinct things are *not* yet supported — keep them separate, they route differently:

| Need | Status | Tracking |
|---|---|---|
| **Line input** (`readLine`, `asyncReadStdinLines`) | ✅ Supported | — |
| **Abort a long-running `std/ai.step()`** mid-call via stdin/signal | ❌ In progress | [#231](https://github.com/sunholo-data/ailang/issues/231) → `m-agent-step-cancellation` (extension + small fix) |
| **Raw-mode single keypress** (no Enter/echo) for human-controlled games | ❌ By design — not in core | non-deterministic, unavailable in WASM, no agentic use case; would be a host/extension capability if ever needed |

For a keyboard-controlled game today, design around it with a self-driving loop or
line-at-a-time (`readLine`) input.

For real-time *output* (games, progress bars, dashboards), see `flush()` and the C-style
string escapes (`\x1b`, `\u{…}`) added in v0.27.0 — `examples/progress_bar.ail`.

---

## Recently Resolved

These limitations existed in earlier versions and are now fully resolved.
Listed for users following older docs:

- **String interpolation** — Implemented in v0.12.1 (`"Hello, ${name}!"`).
  Phase 2 of [M-CONCAT-DISAMBIG](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_13_0/m-concat-disambiguation.md)
  in v0.13.0 made `++` list-only.
- **Pattern guards** — Implemented in v0.6.2
  ([design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_6_2/m-pattern-guards.md)).
  `match x { x if x > 100 => ..., x if x > 0 => ... }` now evaluates guards
  correctly.
- **Polymorphic arithmetic in lambdas** — Fixed in v0.7.0
  ([design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_7_0/m-poly-arithmetic-fix.md)).
  `let add = \x. \y. x + y in add(3.14)(2.71)` now returns `5.85`.

---

## Notes on Parser Error Messages

Parser errors include:
- **Error codes** (e.g., `PAR_UNEXPECTED_TOKEN`)
- **Precise positions** (file, line, column)
- **Suggestions** for fixing the error

Example:
```
PAR_UNEXPECTED_TOKEN at file.ail:2:14: expected ), got INT
Suggestion: Add ')' to close grouped expression
```

---

## Reporting New Limitations

Found a limitation not listed here? Please file an issue at:
https://github.com/sunholo-data/ailang/issues

Include:
- AILANG version (`ailang --version`)
- Minimal reproduction code
- Expected vs actual behavior
- Whether it's a bug or design limitation
