# serve-api Net reaches localhost + cloud metadata on every call, redirect hops included (#1522)

- **Date**: 2026-10-02
- **Class**: bug
- **Recommend**: design-doc (child of `design_docs/planned/v0_52_0/m-mcp-oauth-package.md` S0) → `design_docs/implemented/v0_52_0/m-net-scope-public.md`
- **Searched**: `scope=public`, `Net[scope`, `AllowMetadata`, `redirect hop`, `resolvePinned`, `m-effect-scope-params`, `m-effect-clock-net-fs-modes`

Report verified against HEAD: `cmd/ailang/serve_api.go` (the `effCtx.HasCap("Net")` block) sets
`AllowHTTP`/`AllowLocalhost`/`AllowMetadata` on the one shared Net context, and
`internal/effects/net_authorize.go` (`destinationPolicy.checkRedirect`, `netProxyRoundTripper.RoundTrip`)
applies the same policy to redirect hops as to hop 0, so a redirect to `169.254.169.254` is dialled.
The ratified parent `m-mcp-oauth-package.md` rules on **what** (S0 = D5 (i) redirect hops never reach
loopback/link-local/private, (ii) a per-call public-only mode; D6 = surface `Net[scope=public]`).
It does not rule on the implementation decisions a fix forces: whether `Net[scope=public]` subsumes bare
`Net` in both call directions, how the mode reaches the dial site (Rand-style per-context stack), the
bytecode-VM path, and whether Stream hops get the same rule. Rubric rows 3/4/5 (changes the effect-param
schema, a public surface, spans `internal/types` + `internal/effects` + `internal/eval`). An earlier
coordinator triage (task-500799f0, never merged) said `duplicate-of` the parent; that was correct for
"is this already decided" but the attended fix needs the child doc to pin the decisions above.
Related, not conflicting: `design_docs/planned/v1_0_0/m-effect-scope-params.md` (frames `scope=` as a
typed narrowing — the same reading this uses).

TRIAGE_FILE: design_docs/planned/ailang-core-triage/serve-api-net-ssrf-surface-1522.md
RECOMMEND: design-doc
