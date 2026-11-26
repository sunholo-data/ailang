---
reviewed: 2026-11-26
reviewBy: 2027-05-26
---

# Option vs Result: which to use when

`Option[T]` and `Result[T, E]` are AILANG's two main "this might not have a value" types — but they answer different questions and use **different constructors**. Mixing them up is one of the most common AILANG mistakes (a single typo crashed a production package in 2026 — see [M-MATCH-ADT-XCHECK](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/v0_18_10/m-match-adt-xcheck.md)). Since v0.18.10, the AILANG typechecker rejects pattern matches that mix the two ADTs.

## Quick reference

| Type | Constructors | Means |
|---|---|---|
| `Option[T]` | `Some(T)`, `None` | "There may or may not be a value" — but if there isn't, no explanation needed |
| `Result[T, E]` | `Ok(T)`, `Err(E)` | "This operation either succeeded with a value, or failed with a typed reason" |

**Rule of thumb**: If the absence of a value carries information (an error code, a message, a reason), use `Result`. If absence just means "not present", use `Option`.

## Which functions return which?

This is the part that bites people. The same operation in two different stdlib modules may return different shapes:

### Option-returning (use `Some`/`None`)

| Function | Module | Why Option? |
|---|---|---|
| `getString(json, key)` | `std/json` | Field may be absent — caller decides if that's an error |
| `getInt(json, key)` | `std/json` | Same |
| `getBool(json, key)` | `std/json` | Same |
| `getArray(json, key)` | `std/json` | Same |
| `getObject(json, key)` | `std/json` | Same |
| `head(xs)` | `std/list` | Empty list has no head — not an error, just absence |
| `last(xs)` | `std/list` | Same |
| `find(p, xs)` | `std/list` | Predicate may match nothing |
| `lookup(k, kvs)` | `std/list` | Key may not be in the list |
| `stringToInt(s)` | `std/string` | "abc" isn't a number — not an error worth explaining |
| `getEnv(name)` | `std/env` | Env var may be unset |

### Result-returning (use `Ok`/`Err`)

| Function | Module | Why Result? |
|---|---|---|
| `decode(s)` | `std/json` | JSON parse can fail with a SPECIFIC syntax error |
| `readFileResult(path)` | `std/fs` | FS errors carry meaningful info (permissions, not-found, etc.) |
| `httpGet(url)` | `std/net` | HTTP failures have status codes + bodies |
| `step(model, msgs, tools)` | `std/ai` | AI errors have typed kinds (rate-limit, auth, etc.) |
| `callResult(prompt)` | `std/ai` | Same |

## Common mistake

```ailang parse
-- ❌ WRONG: matches Option with Result constructors
import std/json (Json, getString)

func wrongModel(json: Json) -> string =
  match getString(json, "model") {
    Err(_) => "fallback",         -- Err is from Result, not Option
    Ok(m)  => m
  }
```

```ailang
-- ✓ CORRECT: matches Option with its own constructors
import std/json (Json, getString)

func getModel(json: Json) -> string =
  match getString(json, "model") {
    None => "fallback",
    Some(m) => m
  }
```

```ailang parse
-- ❌ WRONG: matches Result with Option constructors
import std/json (decode, encode)

func wrongDecode(input: string) -> string =
  match decode(input) {
    Some(json) => encode(json),   -- Some is from Option, not Result
    None => "error"
  }
```

```ailang
-- ✓ CORRECT
import std/json (decode, encode)

func parseAndEncode(input: string) -> string =
  match decode(input) {
    Ok(json) => encode(json),
    Err(msg) => "parse failed: ${msg}"
  }
```

## Converting between them

When you need to bridge Option and Result:

```ailang
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)

-- Option → Result with a default error
pure func opt_to_result[T, E](opt: Option[T], err: E) -> Result[T, E] {
  match opt {
    Some(x) => Ok(x),
    None => Err(err)
  }
}

-- Result → Option (discards the error)
pure func result_to_opt[T, E](res: Result[T, E]) -> Option[T] {
  match res {
    Ok(x) => Some(x),
    Err(_) => None
  }
}
```

## Spine propagation with `let x = expr?`

Instead of nesting `match` expressions, bind `let x = expr?` along the return spine of a function returning `Result[T, E]` or `Option[T]`. If `expr` evaluates to `Err(e)` or `None`, the enclosing function returns that failure immediately; otherwise `x` receives the unwrapped `Ok` / `Some` payload:

```ailang
module docs/option_vs_result_try

import std/json (decode, getInt, getString)
import std/result (Result, Ok, Err)

pure func optToResult[T, E](opt: Option[T], err: E) -> Result[T, E] =
  match opt {
    Some(x) => Ok(x),
    None => Err(err)
  }

export pure func parseUserAge(raw: string) -> Result[int, string]
  tests [
    ("{\"age\": 42}", Ok(42)),
    ("{}", Err("missing age"))
  ]
{
  let j = decode(raw)?;
  let age = optToResult(getInt(j, "age"), "missing age")?;
  Ok(age)
}

export pure func parseOptionalCity(raw: string) -> Option[string] {
  let j = match decode(raw) { Ok(v) => Some(v), Err(_) => None }?;
  let city = getString(j, "city")?;
  Some(city)
}
```

> **Constraint (`PAR_TRY_LET_ONLY`):** `?` is restricted to `let x = expr?` bindings on the return spine (not arbitrary sub-expressions like `f(g()?)`). See [Known Limitations](/docs/reference/limitations).

## Why the typechecker enforces this (v0.18.10+)

Before v0.18.10, AILANG's typechecker accepted constructor patterns from any ADT (the patterns were checked in isolation against their own ADT, not cross-checked against the scrutinee's type). Mismatched patterns would silently fall through and the match would panic at runtime with `no pattern matched in match expression`.

Since v0.18.10 ([M-MATCH-ADT-XCHECK](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/v0_18_10/m-match-adt-xcheck.md)), the typechecker rejects this at compile time with a structured error message that names both ADTs and lists their constructors. AI coding agents (and humans) get a precise, actionable error pointing at the wrong constructor instead of a confusing runtime crash.

## See also

- [Standard library reference](./language-syntax) — full type signatures for `std/option`, `std/result`, `std/json`, etc.
- [M-MATCH-ADT-XCHECK design doc](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/v0_18_10/m-match-adt-xcheck.md) — implementation rationale and history
