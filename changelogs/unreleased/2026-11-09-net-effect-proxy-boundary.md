### Fixed — HTTP Stream and the Managed Agents client no longer bypass `HTTP_PROXY`/`HTTPS_PROXY` (security, 2026-11-09)

**Security.** The default-deny test lane poisons `HTTP_PROXY`/`HTTPS_PROXY`, but three
first-party HTTP paths still dialed straight past it, and past any operator proxy: the HTTP
`Stream` operations (`sseConnect`, `ssePost`, NDJSON POST), whose policy hard-coded "no proxy",
and the Managed Agents Vertex client, whose `http.Transport` had a nil `Proxy`. `Net` already went
through the request-aware route (M1). All of them now follow `http.ProxyFromEnvironment`, chosen
per request:

- direct, or the host matches `NO_PROXY`: the target is resolved once, validated, and the dial is
  pinned to that IP, as before;
- proxied literal-IP target: still validated against the IP policy, with no DNS;
- proxied hostname: resolved by the proxy, so the pinned-IP guarantee does not hold (decision D-1).
  Put a destination in `NO_PROXY` if it needs pinning.

**Behaviour change:** a process that has a proxy set now sends SSE/NDJSON setup requests and
Vertex Managed Agents traffic through it. A proxy that filters, needs authentication or buffers
can turn a previously direct success into `ConnectionFailed`. Restricted policy mode
(`ailang run --policy`) now refuses a configured proxy for Stream with `E_STREAM_PROXY_REFUSED`,
as it already did for Net. WebSocket `connect` is unchanged and never uses a proxy.

A new source audit, `TestProxyAudit_Repo` (`internal/testutil/gatelint`), fails on any production
`http.Transport` literal with a nil `Proxy`. It caught one more: the never-taken fallback in
`internal/ai/ollama`'s stream transport, now fixed. The Option-A tripwire
`effects_nil_proxy_remains_open` is retired. Design:
`design_docs/implemented/v0_57_5/m-net-effect-proxy-boundary.md`; durable type-aware analyzer: #612.
