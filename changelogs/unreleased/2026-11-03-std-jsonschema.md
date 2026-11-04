### Added

- `std/jsonschema`: pure Draft 2020-12 validation with structured JSON Pointer errors, strict unknown-keyword checking, local references, RE2 patterns, and explicit lenient and schema-only APIs. Supports evaluated-location tracking for `unevaluatedProperties` and `unevaluatedItems`. Format is annotation-only; numbers use float64.
- `std/list.all(pred, xs)`, including vacuous truth for empty lists and short circuit behavior on both interpreter and VM.
