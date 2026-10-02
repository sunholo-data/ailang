### Fixed — Net redirect hops could reach loopback and the cloud metadata server (#1522, 2026-10-02)

`ailang serve-api --caps Net` allows `http://`, loopback and `169.254.169.254` for the whole
process (for `sunholo/gcp_auth`), and redirect hops used the same policy as the first request, so an
https URL that redirected to `http://169.254.169.254/...` was followed. A redirect hop now runs under
a public-only policy: loopback, link-local and private addresses are refused whatever
`AllowLocalhost`/`AllowMetadata` say, both before the hop (`CheckRedirect`) and at dial time after
DNS (`resolvePinned` on the redirect round trip). Applies to Net and Stream. Direct requests are
unchanged, so `gcp_auth`'s metadata call still works. The refusal names the rule instead of
suggesting a flag.

### Added — `! {Net[scope=public]}` effect mode (#1522, 2026-10-02)

A function declaring `! {Net[scope=public]}` runs every Net call in its dynamic extent (hop 0 and
redirects) with loopback and the metadata server forced off, checked at connect time against every
resolved address, so a hostname resolving to `127.0.0.1` or `169.254.169.254` is refused. Declare
it on functions that fetch user-chosen URLs. Mirrors `Rand[mode=crypto]`: the evaluator pushes the
scope at frame entry (per-execution state, reset per serve-api request). `scope` is a narrowing
parameter: a public function may call bare-`Net` helpers and a bare-`Net` function may call a public
one. Bare `Net` is unchanged; `Net[scope=x]` for other values is `EFF_UNKNOWN_MODE`. Under
`--bytecode` such functions run on the evaluator. Example: `examples/runnable/net_scope_public.ail`.
Design: `design_docs/planned/v0_52_0/m-net-scope-public.md` (S0 of `m-mcp-oauth-package`).
