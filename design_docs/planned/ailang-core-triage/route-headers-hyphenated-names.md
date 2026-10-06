# @route response headers cannot express hyphenated header names (X-Frame-Options etc.)

- **Date**: 2026-09-15
- **Class**: feature (with a silent-failure bug attached)
- **Recommend**: design-doc
- **Searched**: `serve-api|apiserver` in design_docs/, `_headers` in design_docs/, `X-Frame-Options|Content-Security-Policy|security header`, grep of `design_docs/planned/` and `ailang-core-backlog.md` for issue 1597 — no existing planned doc covers this. Nearest prior art is `design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements.md`, which is the doc that *introduced* the `_headers` magic field (and chose the record form), so it can't rule on a change to its own format. #1597 itself has no design doc in the repo; only a partial fix exists as the `--static-cache` flag (`internal/apiserver/server.go:94`, applied by `staticCacheHandler` at `server.go:640`).

## Why

The mechanism is real, and it is worse than "missing feature" — it is two silently-dead forms. Verified in `internal/apiserver/routes_dispatch.go`:

1. **@nowrap map path** (~line 240): `goResult.(map[string]interface{})` → `headersVal.(map[string]interface{})` → `w.Header().Set(k, sv)` for string values only. A record field `_headers: {x_frame_options: "DENY"}` sends `X_frame_options` (underscores kept, Go canonicalises the first letter), which browsers ignore. A `Json` value for `_headers` matches neither `map[string]interface{}` nor anything else — every header is dropped without error.
2. **`writeRawResponse`** (~line 318, the `_body`/`_status`/`_headers` path): same shape — `rec.Fields["_headers"].(*eval.RecordValue)`, string fields only, name passed to `Header().Set` unmodified.

Record field labels in AILANG cannot contain `-`, so the record form can *never* express `X-Frame-Options`, `Content-Security-Policy`, `Referrer-Policy`, `X-Content-Type-Options`, or any security header. The only shipping alternative (`--static-cache`) covers one header name on the static handler only. The reporter's context is concrete: RFC 9700 §4.16 requires `X-Frame-Options` on OAuth consent pages, and docparse shipped a JS/CSS workaround (PR #218) because AILANG cannot send the header.

Note the report cites `routes_dispatch.go:240`; today the @nowrap header loop sits at ~line 240-253 and the second, equivalent site is `writeRawResponse` at ~line 318-330 — the fix must touch **both**, or the `_body`/binary path keeps the bug.

## Why design-doc, not direct-fix

- Row 3: the reporter offers (and the repo could support) at least three acceptable mechanisms — `'_'→'-'` mapping in record labels, accepting a `Json` object for `_headers`, accepting `[(string, string)]` — and they are not mutually exclusive. The `'_'→'-'` mapping changes what an existing (if broken) program does; a label like `x_frame_options` currently produces a header literally named that, so remapping is a public-surface change someone could disagree with.
- Row 4: changes a public surface (the `_headers` record format introduced by `m-serve-api-agent-enhancements.md`) and the "fail loudly" requirement (#3) defines new startup/type-error behaviour — exactly the class of decision design docs exist for.
- Row 5: spans at minimum both dispatch sites in `routes_dispatch.go`, the static handler for #1597, plus tests and docs.

The reporter's ASK is already a coherent design sketch: (1) map `_`→`-` in `_headers` labels, (2) also accept a `Json` object (or `[(string, string)]`) so arbitrary names work, (3) reject unusable `_headers` loudly (type error at check time or startup error in serve-api, not silent drop), (4) wire-level tests for `X-Frame-Options` + `Content-Security-Policy`, and it should be written as **one design covering both #1597 (security headers on `serve-api --static`) and this**, since the same header-naming problem and the same "no way to send a security header" outcome underlie both. The `--static-cache` flag shows the intended pattern; the design should generalize it or supersede it.
