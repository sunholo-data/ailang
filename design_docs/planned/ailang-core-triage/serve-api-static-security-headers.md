# serve-api --static sends no security headers — OAuth consent pages are framable (clickjacking)

- **Date**: 2026-10-06
- **Class**: bug (security, medium)
- **Recommend**: design-doc
- **Searched**: `X-Frame-Options`, `frame-ancestors`, `clickjacking`, `security header`, `static-cache`, `oauth consent` across `design_docs/` and `docs/`; no existing doc rules on response headers for `--static` (the closest, `design_docs/implemented/v0_46_0/m-serveapi-operator-surface.md`, ruled only on `--static-cache` Cache-Control, D6)
- **Estimate**: ~60–90 lines in `internal/apiserver/` (new `static_headers.go` beside `static_cache.go`, same wrapper pattern) + flag/wiring in `cmd/ailang/serve_api.go` and `internal/apiserver/server.go`, plus tests and the `ailang mcp check` addition — exceeds the direct-fix thresholds in every dimension

Verified against HEAD: `internal/apiserver/server.go` (`mux.Handle("/", ...)`, :640) serves `--static` via
`staticCacheHandler` → `http.FileServer` and sets only `Cache-Control` when `--static-cache` is given —
no `X-Frame-Options`, `Content-Security-Policy`, `Referrer-Policy`, or `X-Content-Type-Options` anywhere
in the static path. `cmd/ailang/serve_api.go` offers no header flag. The report's prod repro
(docparse.ailang.sunholo.com serving `oauth-login.html` bare) matches this code, and RFC 9700 §4.16
makes an framable consent page a hard MUST-violation for adopters steered to this hosting (per
`design_docs/planned/v0_52_0/m-mcp-oauth-package.md`).

Why design-doc, not direct-fix (rubric rows 3/4/5):

1. **A semantics decision the fix forces**: are the four headers opt-out defaults or opt-in via
   `--static-header`? Defaults are the safe answer and the reporter's ask, but they are a behavior
   change for existing deployments (any app that legitimately frames its own static HTML breaks,
   silently, on upgrade) — that deserves a documented default + escape hatch decision, not an
   incidental implementation choice.
2. **Two acceptable configurability shapes** (row 3): repeatable `--static-header 'Name: value'` vs a
   per-path override map. The report says "either". A doc should pick one and say why.
3. **Spans ≥4 files plus a second surface** (row 5): headers/wrapper + CLI flag + config struct + tests,
   and the separate `cmd/ailang/mcp_check.go` enhancement (fetch the authorization endpoint's login
   page, fail when framable — needs its own spec: what URL, what headers fail the check, how it
   composes with `--target`).
4. **Public-surface change** (row 4): a new CLI flag and new default response headers.

Scope note for the designer: the mechanical part is small — `internal/apiserver/static_cache.go`
(`ParseStaticCache` + `staticCacheHandler` + `cacheWriter`) is a ready-made pattern to clone for
`--static-header`, and the header-set defaults are quoted verbatim in the report. The decisions worth
a doc are (a) defaults on/off, (b) override shape, (c) the `mcp check` framing probe, and (d) whether
the workaround in the report (serve consent HTML from an `@raw` route with `_headers`) is documented
as the sanctioned interim. Related, not conflicting:
`design_docs/planned/v0_52_0/m-mcp-oauth-package.md` (owns the OAuth surface; this is a transport/
hosting fix that belongs to the serve-api lane).
