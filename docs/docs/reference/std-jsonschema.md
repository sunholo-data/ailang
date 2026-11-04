---
title: JSON Schema validation
---

# std/jsonschema

Pure, Go-backed JSON Schema Draft 2020-12 validation over the `std/json.Json`
ADT. No network, filesystem, capabilities, or new dependencies are needed.

```ailang
import std/jsonschema (ValidationError, validate, validateLenient, validateSchema)
```

| API | Result |
| --- | --- |
| `validate(schema: Json, doc: Json)` | `[ValidationError]`, strict by default |
| `validateLenient(schema: Json, doc: Json)` | Same, ignoring unknown keywords only |
| `validateSchema(schema: Json)` | Strict schema audit without an instance |

`ValidationError` is `{ path: string, keyword: string, message: string }`.
An empty list means success. Errors use deterministic sorted traversal of object
keys and preserve array order. All schema compilation errors return before any
instance validation; malformed supported keywords fail even in lenient mode.
The whole schema is audited, including unused `$defs` and unselected branches.
Annotation values such as `default` and `examples` are data, not subschemas.

`path` is an RFC 6901 JSON Pointer: the root is `""`, `/a~1b` addresses key
`a/b`, and `/a~0b` addresses `a~b`. Schema errors point into the **schema** and
start their message with `schema:`. Instance errors point into the **document**.
Branch failures inside a successful `anyOf`, or an `if` predicate, are not
reported as instance failures. Error messages are explanatory; route on `keyword`.

## Vocabulary

Core: boolean schemas, `$schema`, relative `$id`, `$ref`, `$defs`, `$anchor`,
`$dynamicAnchor`, `$dynamicRef`, `$vocabulary`, `$comment`.

Assertions: `type` (including integer-valued numbers), `enum`, `const`,
`minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`, `multipleOf`,
`minLength`, `maxLength`, `pattern`, `minItems`, `maxItems`, `uniqueItems`,
`minContains`, `maxContains`, `minProperties`, `maxProperties`, `required`,
`dependentRequired`.

Applicators: `properties`, `patternProperties`, `additionalProperties`,
`propertyNames`, `prefixItems`, `items`, `contains`, `allOf`, `anyOf`, `oneOf`,
`not`, `if`, `then`, `else`, `dependentSchemas`, `unevaluatedProperties`,
`unevaluatedItems`. Successful branches propagate evaluated locations;
`unevaluated*` runs last. `items` applies after `prefixItems` in this dialect.

Metadata: `title`, `description`, `default`, `deprecated`, `readOnly`,
`writeOnly`, `examples`, and `format`. **Format is annotation-only**: values like
`email` do not add assertions. Unknown keywords are errors by default; use
`validateLenient` only when intentionally accepting that loss of validation.

Only the Draft 2020-12 dialect is supported. Older array-form `items` must be
migrated to `prefixItems`. References must stay inside the in-memory schema:
`#`, `#/$defs/name`, escaped pointers, and named anchors. Remote references are
schema errors and are never fetched. Cycles at the same instance location are
bounded at 64; recursive schemas may descend into child instance locations.

## Regular expressions and numbers

Patterns compile once per call using Go `regexp` (RE2). They are unanchored by
default. Use `^` and `$` for whole-string matching. Length bounds count Unicode
code points, not bytes or grapheme clusters.

| ECMA-262 feature | Behavior here |
| --- | --- |
| Lookahead and lookbehind | Schema compilation error |
| Backreferences such as `\1` | Schema compilation error |
| Unicode property classes | Go/RE2 class names and escape syntax |
| Unicode sets / JavaScript flags | No JavaScript `u` or `v` flag interpretation |
| Code point escapes | Go/RE2 syntax; differs from ECMA-262 |

RE2 also accepts extensions such as inline flags that ECMA-262 may reject.
Accepted patterns can differ at Unicode and anchor edge cases; this API does
not claim full ECMA-262 compatibility. Unsupported syntax is an error naming
the pattern and keyword, including inside unused subschemas.

`JNumber` uses float64. Numeric comparisons and `multipleOf` use float64
arithmetic, so decimal rounding and integers beyond exact float64 precision
can differ from arbitrary-precision validators. Non-finite numbers are not JSON.

See [the runnable self-check](https://github.com/sunholo-data/ailang/blob/dev/examples/runnable/jsonschema_validate.ail).
The census and benchmark fixtures in this repository are synthetic; they are
not the external client envelope or evidence reproducing its reported failures.
