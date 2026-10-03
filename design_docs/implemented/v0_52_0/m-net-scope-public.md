# M-NET-SCOPE-PUBLIC: redirect-hop containment + `Net[scope=public]` (#1522, S0)

**Status**: Implemented (2026-10-02; self-evaluated 95/100 in-session — not an independent evaluator)
**Target**: v0.52.0
**Priority**: P0 (security, defence in depth; blocks deploy of `sunholo/mcp_oauth`)
**Estimated**: 0.5–1 day
**Dependencies**: none
**Parent**: [m-mcp-oauth-package](../../planned/v0_52_0/m-mcp-oauth-package.md) — S0, T5, D5, D6 (**Design Freeze ratified by Mark 2026-10-02**)
**Issue**: #1522 (duplicate #1526). Triage: [serve-api-net-ssrf-surface-1522](../../planned/ailang-core-triage/serve-api-net-ssrf-surface-1522.md)

## Quorum

**Skipped.** The parent was quorum-reviewed twice (round 0 and round 1; the round-1 objection on T5 is
what produced S0 part (ii)) and its Design Freeze (D5: two parts; D6: `Net[scope=public]`) was ratified by
Mark. This child doc carries no new design-freeze item that a human must ratify: it records the
implementation decisions the parent left to the agent (D-A..D-D below), all verifiable in-repo.

## Problem

`cmd/ailang/serve_api.go` (the `effCtx.HasCap("Net")` block) sets `AllowHTTP`, `AllowLocalhost` and
`AllowMetadata` on the one Net context of the process, because `sunholo/gcp_auth` fetches tokens from
`http://metadata.google.internal/...` (V2). The same policy then applies to:

1. **redirect hops** of any fetch — `destinationPolicy.checkRedirect` and `netProxyRoundTripper.RoundTrip`
   use the hop-0 policy unchanged (V3), so an https URL that 302s to `http://169.254.169.254/...` is dialled;
2. **user-chosen URLs** whose hostname *resolves* to loopback/metadata. URL-string checks cannot see this;
   only the dial-time check (`resolvePinned`, which validates every resolved IP, V4) can.

## Solution

### Part (i) — redirect hops are always public-only

A redirect hop is evaluated under `destinationPolicy.forRedirectHop()`: a copy of the policy with
`allowLocalhost=false`, `allowMetadata=false`, `blockPrivate=true`. It is applied at both gates:

- `checkRedirect` (before any dial, lexical + literal-IP) — so `http://localhost/` and `http://127.0.0.1/`
  targets are refused by name;
- `RoundTrip` when `req.Response != nil` (Go's client sets `Request.Response` only on redirect requests,
  V5), so `resolvePinned` refuses a redirect *hostname* that resolves to loopback/link-local/private.

Hop 0 is unchanged: a direct `gcp_auth` metadata call keeps working (V2, Conflict Surface §4).

### Part (ii) — `Net[scope=public]`

```ailang
import std/net (httpGet)
export func fetchClient(u: string) -> string ! {Net[scope=public]} = httpGet(u)
```

- **Types.** `effectSchema["Net"] = {"scope": {"public"}}`. No default entry for Net, so bare `Net` stays
  bare (no row printing changes; V6). Any other key/value is the existing `EFF_UNKNOWN_PARAM_KEY` /
  `EFF_UNKNOWN_MODE` error.
- **Runtime.** Mirrors `Rand[mode=crypto]` (V7): `buildClosure` extracts the declared scope into
  `FunctionValue.EffectNetScope`; `applyFunctionValue` pushes it at frame entry and pops it on exit (a
  frame with a scope is never tail-replaced — `replaceable`). `netPolicy(ctx)` reads
  `ctx.NetScopePublic()`; when true, `allowLocalhost` and `allowMetadata` are forced off for hop 0 *and*
  every hop. Enforcement is at dial time after DNS via `resolvePinned`.
- **Per-execution state.** Shared across `WithBudget` scopes, reset in `Clone` (exactly the `randMode`
  lifecycle, V8). Because `public` is the only value, the state is a depth counter, not a stack.

### Decisions (agent-resolved; recorded here)

| # | Decision | Why |
|---|---|---|
| D-A | **Subsumption: `scope` is a narrowing param, compatible with absence in BOTH directions** at the validation subsumption check (`DiffEffectRows`). `f ! {Net[scope=public]}` may call `httpGet ! {Net}` (the crypto-satisfies-os direction), and a bare `! {Net}` handler may call `fetchClient ! {Net[scope=public]}`. | The scope is a restriction a frame places on its own dynamic extent, not a capability the caller must hold. Forcing callers to declare it would force a serve-api handler's other Net calls (e.g. `gcp_auth`) to be public too and break them. Function-*value* effect rows stay invariant in `unifyRows` (unchanged, same as Rand modes). |
| D-B | **Dynamic extent.** Everything called (transitively) from a public frame is public. Bare `Net` pushes nothing, so callees cannot widen it. | A public function calling a helper that calls `gcp_auth` *should* be refused — that is the point. |
| D-C | **Bytecode VM fails closed.** The VM (`--bytecode`, opt-in) has no moded-frame hook (V9). A top-level function declaring `Net[scope=public]` is lowered as an EvalOnly stub, so the call routes to the evaluator where the scope is enforced. | Silently dropping a security mode on one execution path is a silent fallback. |
| D-D | **Stream redirects get part (i) too.** The hop rule lives in the shared `destinationPolicy`, so SSE/NDJSON redirects are also public-only. | One authorizer (M-EXECUTOR-POLICY-HARDENING D2); no reason a stream redirect may reach metadata. |
| D-E | **Concurrency caveat.** Tasks sharing one EffContext share the counter: while a public frame is active, a concurrent bare-Net task on the *same* context is also public. Over-restrictive, never under-restrictive. serve-api requests use `Clone`, so requests do not interfere. | Fail closed. |

## Conflict Surface

1. **Positions extended**
   - effect-param schema `internal/types/effects.go` (`effectSchema`): Net gains a `scope` key;
   - validation subsumption `internal/types/effect_subsumption.go` (`DiffEffectRows`): a narrowing-param table;
   - evaluator frame entry `internal/eval/eval_apply.go` (`applyFunctionValue`, `replaceable`) and
     `internal/eval/eval_expressions.go` (`buildClosure`);
   - `internal/effects/net_authorize.go` (`netPolicy`, `checkRedirect`) and `net_proxy.go` (`RoundTrip`);
   - lowering `internal/gen/lower/program.go` (`bindingToFuncDecl`) for D-C.
2. **Constructs already there**
   - `Rand[mode=…]` and `AI[mode=…, scope=byok]` params, defaults (`defaultEffectModes`) and the mode
     subsumption edges (crypto/seeded ⊑ os). Untouched; the narrowing table is keyed by (effect, key) so
     it cannot match `mode`.
   - `Net[mode=live]` appears only in a parser test (V10); the parser accepts any `[k=v]`, the schema
     decides legality, so no parser change.
   - `m-effect-scope-params` (planned v1.1.0) frames `scope=` as a typed narrowing of a grant (V11). This
     doc is consistent with that reading and registers only the one value it needs.
3. **Disambiguation:** none needed at parse time; legality is the closed schema. Hop 0 vs redirect is
   `req.Response != nil` at RoundTrip and the `CheckRedirect` call site.
4. **Programs that must still work**
   - `sunholo/gcp_auth` `getMetadataToken` — direct (hop-0) GET of `http://metadata.google.internal/…` with bare `Net` (V2);
   - `internal/effects/net_authorize_test.go`, `net_test.go`, `net_redirect_containment_test.go`, `net_proxy_test.go`;
   - `examples/modal_rand.ail` (Rand mode push/pop unchanged);
   - `internal/types` effect-param tests (`EFF_PARAMS_NOT_SUPPORTED` for Clock/FS still fires).
5. **Deliberate changes**
   - a redirect to loopback / link-local / private now fails `E_NET_IP_BLOCKED` / `E_NET_DNS_REBINDING`
     even when `AllowLocalhost`/`AllowMetadata` are set (also for Stream: `E_STREAM_DISALLOWED_HOST`).
     This includes a **same-host loopback redirect** (`http://127.0.0.1:P/a` → `/b`): found by
     `TestRunPolicyE2E_RedirectToNonAllowlistedHostDenied`, whose positive control was exactly that and
     now expects `DENIED`. A same-literal-host exemption would add no reach an attacker lacks at hop 0,
     but the ratified rule is "never"; left as an open follow-up if local-dev redirects need it;
   - `Net[scope=x]` for `x ≠ public`, and any other Net key, change from `EFF_PARAMS_NOT_SUPPORTED` to
     `EFF_UNKNOWN_MODE` / `EFF_UNKNOWN_PARAM_KEY`.

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | serve-api forces all three Net flags when `--caps` has Net | `sed -n 134,138p cmd/ailang/serve_api.go` | Confirmed (lines still hold) |
| V2 | `gcp_auth` reaches metadata by a direct hop-0 GET with bare `Net` | `grep -n 'metadata.google\|func getMetadataToken' ailang-packages/packages/gcp-auth/token.ail` → l.78 `! {Net}`, l.79 URL | Confirmed |
| V3 | redirect hops use the hop-0 policy | read `net_authorize.go` `checkRedirect` (calls `p.authorizeURL`) and `net_proxy.go` `RoundTrip` (same `rt.pol`) | Confirmed |
| V4 | `resolvePinned` validates every resolved IP; `lookupIP` injectable; private refused unconditionally | read `net_authorize.go` `resolvePinned`, `validateIP`, `lookupIP` field | Confirmed |
| V5 | Go sets `Request.Response` only on client redirects | `$(go env GOROOT)/src/net/http/request.go:320-323` doc; `client.go:667 Response: resp` | Confirmed |
| V6 | `Net[scope=public]` is rejected today | `ailang check` (repo build) → `EFF_PARAMS_NOT_SUPPORTED: effect 'Net' does not support parameters (found: scope)` | Confirmed |
| V6b | a declared-only `scope` key fails subsumption today | `ailang check` of `func f(u) ! {AI[scope=byok]} = call(u)` → `Effect scope mismatch: AI requires scope=; declaration provides scope=byok` | Confirmed (so D-A needs code, not just a schema row) |
| V6c | `Rand[mode=crypto]` calling `rand_int ! {Rand}` checks | `ailang check` → `No errors found` | Confirmed |
| V7 | Rand mode reaches dispatch via a per-context stack pushed in `applyFunctionValue` | `grep -rn PushRandMode` → `eval_evaluator.go:364`, `eval_apply.go:136`; `rand_mode.go` | Confirmed |
| V8 | `randMode` shared by `WithBudget`, reset by `Clone` | `internal/effects/context.go:496`, `:703` | Confirmed |
| V9 | the bytecode VM has no moded-frame hook | `grep -rn 'RandMode\|PushBudgetFrame' internal/vm internal/bytecode` → empty | Confirmed (negative) |
| V10 | no shipped `.ail`/Go uses `Net[...]` outside a parser test | `grep -rn 'Net\[' --include=*.go --include=*.ail .` → only `internal/parser/parser_effect_params_test.go` | Confirmed (negative) |
| V11 | `m-effect-scope-params` reads `scope=` as a narrowing | `design_docs/planned/v1_0_0/m-effect-scope-params.md` l.15–16 | Confirmed |

## Tests (named in the sprint plan)

Injected `lookupIP`: hostname→127.0.0.1 refused under scope=public; →169.254.169.254 refused under
scope=public; a public httptest server redirecting to a loopback / link-local target refused even with
`AllowLocalhost`/`AllowMetadata`; a direct metadata call with bare Net + `AllowMetadata` still allowed;
`ailang check` accepts the example above. Each guard mutation-tested.

## Axiom Compliance

| Axiom | Score | Note |
|---|---|---|
| A4 Explicit Authority | +1 | the narrowing is in the signature, greppable |
| A3 Explicit effects | +1 | a mode on an existing effect, not a hidden option |
| A11 Structured failure | +1 | typed `E_NET_*` refusals |
| A1 Determinism | 0 | no change |
| Net | +3 | no hard violations |
