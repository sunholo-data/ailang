---
title: Strict Fallbacks (STRICT_FALLBACK_001)
sidebar_label: Strict Fallbacks
---

# Strict Fallbacks — `STRICT_FALLBACK_001`

AILANG statically detects the **"`Ok` contains a default/empty value"** anti-pattern
in `Result`-returning functions. This is the static enforcement of the
[NO SILENT FALLBACKS](https://github.com/sunholo-data/ailang/blob/main/CLAUDE.md)
principle: a function that promises `Result[T, E]` must not return `Ok(<empty>)`
where it means "failure".

## The bug it catches

A `Result`-returning function whose `Ok` branch carries an empty or zero value
can't be distinguished from a populated success by the caller:

```ailang
export func getDoc(o: Json) -> Result[Json, string] =
  match get(o, "fields") {
    Some(fields) => Ok(fields),
    None         => Ok(jo([]))   -- ← STRICT_FALLBACK_001: empty object
  }
```

Downstream, the caller takes the `Ok` arm with empty data instead of falling
through to its error handler:

```ailang
match getDoc(doc) {
  Ok(fields) => useFields(fields),  -- silently sees {} — no error signal
  Err(e)     => handleError(e)      -- never reached
}
```

This exact shape shipped in a production package (`firestore/client@0.7.1`) and
returned all-zero data to customers with no error signal until complaints
surfaced it.

## What flags

In a function whose declared return type is `Result[_, _]`, an `Ok(...)` whose
argument resolves to any of:

| Form | Example |
|------|---------|
| Empty string | `Ok("")` |
| Empty list | `Ok([])` |
| Empty record | `Ok({})` |
| All-zero record | `Ok({name: "", age: 0, active: false})` |
| Zero-valued constructor | `Ok(MyCtor("", 0))` |
| Known-empty builder | `Ok(jo([]))` (from `std/json`) |

The detector runs **after name resolution**, so it keys on the *resolved
identity* of a builder — `std/json.jo`, module-qualified — not a bare name. A
user-defined local `jo` is a different symbol and is **never** flagged.

## What does NOT flag

- `Ok(realValue)` where the value is a variable, a function-call result, or a
  non-empty literal.
- A bare `Ok(0)` / `Ok(false)` — a legitimate scalar success value. (Empty
  *string* is treated as empty; numeric/bool zeroes are not, to avoid
  false-flagging a real `Ok(0)`.)
- Mixed records: `{name: realName, age: 0}` has a real field, so it is not
  all-zero.
- The same patterns in a function that does **not** return `Result`.

## The two channels

| Command | Behaviour |
|---------|-----------|
| `ailang check file.ail` | **Warning**, exits **0** — advisory during development. |
| `ailang check --package dir/` | **Hard error**, exits **1** — the publish boundary fails loudly. |

Development stays unblocked; publishing a package with an unannotated violation
fails.

## Opting out: `@allow_empty_ok`

If an empty-`Ok` is genuinely correct (e.g. an empty collection is a valid
result), annotate the function with a **required rationale string**:

```ailang
@allow_empty_ok("missing 'documents' key means an empty collection")
export func listDocs(o: Json) -> Result[Json, string] =
  match get(o, "documents") {
    Some(docs) => Ok(docs),
    None       => Ok(jo([]))   -- suppressed: rationale recorded
  }
```

The rationale is **mandatory** — `@allow_empty_ok()` (no argument) or
`@allow_empty_ok("")` is a parse error. The rationale is the audit record for
why the empty-`Ok` is intentional.

## Fixing a violation

Prefer returning an error with a descriptive message:

```ailang
None => Err("response has no 'fields' key")
```

Only reach for `@allow_empty_ok` when the empty result is semantically a success.

## See also

- Runnable example: [`examples/strict_fallbacks_demo.ail`](https://github.com/sunholo-data/ailang/blob/main/examples/strict_fallbacks_demo.ail)
- Design axiom [A11 — Structured Failure](/docs/references/axioms#a11-structured-failure)
