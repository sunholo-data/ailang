# M-SERVE-API-JSON-DECODE: Fix TaggedValue arg decoding in serve-api

## Status: Implemented
## Version: v0.9.4
## Priority: Bug (P1)
## Effort: Small (1 hour)

## Problem

Every function in `api_keys.ail` fails when called via serve-api with:
```
_json_decode: expected string, got *eval.TaggedValue
```

Error occurs at 0ms (function body never executes). Other modules work fine.

## Root Cause Analysis

The issue is that AILANG functions call `_json_decode()` on values returned by other
functions that return `Result[string, string]` or `Option[string]`. These return
`TaggedValue` wrappers (e.g., `Ok("...")`, `Some("...")`), which `_json_decode`
rejected because it only accepted `*eval.StringValue`.

The `FromGoPreserveFloats()` conversion from HTTP args is correct — it produces
StringValue for strings, RecordValue for objects, etc. The TaggedValue comes from
**within the AILANG function body** when calling other module functions.

### Affected Files

- `internal/builtins/json_decode.go` — New builtin JSON decode (effects system)
- `internal/eval/builtins_json.go` — Legacy JSON decode (same bug)

## Fix Applied

Added `unwrapToString` / `unwrapTaggedToString` helpers that extract a string from
common TaggedValue wrappers like `Ok(string)`, `Some(string)`, and nested wrappers
like `Ok(Some("..."))`.

Both implementations now:
1. Try direct `*StringValue` check (existing behavior)
2. If TaggedValue: try unwrapping single-field constructors to find a string
3. On failure: provide actionable error message naming the wrapper type

### Improved error message
Before: `_json_decode: expected string, got *eval.TaggedValue`
After: `_json_decode: expected string, got Result(Ok) — unwrap the Result before calling json_decode`

## Testing

- `go test ./internal/eval/` — all JSON tests pass
- `go build ./...` — clean build
- Existing _json_decode(string) behavior unchanged

## Origin

Agent message 80a588f1 from docparse (2026-03-19)
