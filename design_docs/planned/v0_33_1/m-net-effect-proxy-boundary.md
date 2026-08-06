# m-net-effect-proxy-boundary — bring AILANG's Net effect inside the egress boundary (D5 Option B)

**Status**: Planned
**Target**: v0.33.1
**Priority**: P0 (High) — the default-deny test lane has a measured first-party HTTP(S) escape
**Estimated**: 3 days (2–4 day range; 4 milestones)
**Dependencies**: M-CI-FLAKE-SYSTEMIC-FIX (implemented); human D5 decision selecting Option B
**Planner-Lane**: opus-required — production networking and an SSRF control change together

---

## Problem Statement

The default test lane poisons `HTTP_PROXY` and `HTTPS_PROXY`, but seven production
`http.Transport` literals in four files currently omit a proxy function (V2–V3). Six are in the
runtime effects package and one is the Vertex Managed Agents client. Consequently these clients do
not participate in the process egress policy selected by an operator or by the poisoned CI lane.

That seven-site scope follows from Go's actual client defaults rather than from the literal search
alone. A bare `http.Client` with no explicit `Transport` uses `http.DefaultTransport`, whose Go
1.26.5 definition sets `Proxy: ProxyFromEnvironment` (V18). The 29 production inline client
literals found by the construction audit are therefore already proxy-aware unless they install a
custom transport. In the audited first-party scope, the escape hatch is precisely a hand-built
`http.Transport{}` whose `Proxy` remains nil: the seven sites measured by V2. The wider abstraction
audit found no shared HTTP factory, custom `RoundTripper`, transport clone, or alternate proxy
routing abstraction that changes that conclusion (V17).

This is not a mechanical “add one field” change. The three `Net` transports first resolve the
target, reject a forbidden IP, and install a `DialContext` that replaces the requested dial host
with the validated target IP (V4). When Go uses an HTTP proxy, the transport dials the proxy and
asks it to reach the target hostname. Leaving the existing closure unchanged would replace the
*proxy* address with the validated *target* address and then speak proxy protocol to the target;
that is functionally wrong. Removing the closure globally would make direct requests lose their
anti-DNS-rebinding pin.

The security controls have different scopes and must not be conflated:

- the `Net` capability check still controls whether an operation may begin (V5);
- protocol validation and the initial target-domain allowlist run before transport selection and
  require no DNS (V4);
- redirect validation still enforces redirect count and protocol policy before the next round trip;
  it no longer resolves the redirect target (V4, V19);
- target-IP resolution and validation occur exactly once per direct round trip, immediately before
  dialing, and the validated address is the address dialed; proxied round trips perform no local
  target DNS lookup;
- only the guarantee that the socket reaches the exact pre-validated **target IP** is traded away
  on a proxied request, because the proxy resolves/reaches the target by name;
- direct requests, including requests bypassed by `NO_PROXY`, must retain target-IP pinning.

**Impact:** AILANG programs using `Net`, programs using the HTTP forms of `Stream`, the eval harness
when generated programs call those effects, runnable network examples, and Managed Agents
executions will begin honoring an ambient operator proxy. A malformed, unavailable, filtering, or
buffering proxy can therefore change a previously direct success into a structured transport or
stream connection failure. Default CI should instead become stricter: an accidental non-loopback
request through these clients must hit the poison rather than escape.

## Goals

**Primary Goal:** Every first-party production HTTP transport identified by V2 honors the standard
Go proxy environment, while direct `Net` requests retain validated-IP dialing and all existing
capability, domain, protocol, redirect, budget, and size controls remain enforced.

**Success Metrics:**

- seven proxy-ignoring production constructors reduced to zero, guarded by a source audit with a
  known-positive fixture/control;
- the six effect transports and the Managed Agents client fail through a fake/poison proxy in
  tests that invoke their production constructors;
- direct and `NO_PROXY` `Net` requests demonstrably dial the pre-validated IP;
- the old “residual remains open” tripwire and matching implemented-doc/workflow-guide text are
  retired and replaced by positive boundary assertions;
- focused package tests, vet, and architecture-boundary checks pass outside the socket-restricted
  sandbox.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| **D-1:** request-aware split: direct `Net` requests keep pinned-IP dialing; proxied requests use normal proxy dialing and knowingly delegate target resolution to the operator-selected proxy | A single transport/dial closure cannot both substitute the target IP and dial a proxy correctly; this is the SSRF boundary | human via design approval | design | high |
| A configured proxy is an explicit operator trust decision; it supersedes only target-IP pinning, not the `Net` capability, protocol, target-domain, redirect, budget, or response-size checks | Precisely defines the security trade rather than silently weakening the whole Net policy | human via design approval | design | high |
| Use Go's `http.ProxyFromEnvironment` semantics, including `NO_PROXY`; do not add a new `ctx.Net` flag or version gate | Operators and CI already express routing policy in these standard variables; a second policy surface could disagree with them | human via design approval | design | high |
| **D-2:** include `internal/executor/managed_agents/client.go` | Excluding it would knowingly leave the only measured first-party production `http.Transport` residual outside `internal/effects` (V2) | human via design approval | design | med |
| Stream and Managed Agents transports use `ProxyFromEnvironment` directly; `Net` alone needs request-aware direct/proxy routing because it owns target-IP pinning | Avoids inventing security machinery where no target-IP pin exists today | agent constrained by this design | implementation | med |
| This is an immediate patch behavior change, documented in the v0.33.1 changelog and runtime docs; no compatibility flag or LIMITATIONS entry | Proxy variables conventionally express process HTTP routing, and `NO_PROXY` is the opt-out; a limitations entry would describe intentional behavior, not a language limitation | human via design approval | design | med |

### Design Freeze

- [x] Proxied `Net` requests knowingly trade target-IP pinning for operator-selected proxy routing.
- [x] Direct and `NO_PROXY` `Net` requests retain target-IP pinning.
- [x] All seven measured first-party production transport sites are in scope.
- [x] Standard proxy environment variables are authoritative; no capability flag or version gate.
- [x] The old open-residual tripwire and matching documentation are retired in this sprint.

## Solution Design

### Overview

Introduce a small, package-private proxy-aware round trip mechanism in `internal/effects`. For each
`Net` request it asks `http.ProxyFromEnvironment(req)` which route applies:

1. **No proxy (unset or `NO_PROXY` match):** the RoundTripper's direct path calls
   `resolveAndValidateIP` exactly once for that request, then gives the returned IP to a direct
   transport whose dialer connects to that same IP without hostname re-resolution. This single
   resolve→validate→dial sequence preserves anti-DNS-rebinding pinning and closes the prior
   check/use gap.
2. **Proxy selected:** use a proxy transport with ordinary proxy dialing and skip local resolution
   and IP validation of the target entirely. This permits corporate proxy use on hosts without
   external DNS. The proxy address is never passed through the target-IP substitution closure. The
   request URL, Host/SNI semantics, and proxy CONNECT/absolute-URI behavior remain Go's
   responsibility; the proxy chooses the ultimate target IP.

The mechanism must make this choice per request, not once per process or initial URL, so redirects
and `NO_PROXY` are evaluated against the request actually being sent. A safe implementation is a
package-private `RoundTripper` that owns separate direct and proxy transport creation paths. It
must not mutate one shared transport between modes. Transport reuse/cache details may be chosen by
the implementer only if tests prove that mode selection and target pinning cannot bleed across
requests.

Remove the three initial/preflight `resolveAndValidateIP` calls and the redirect-validator call
measured by V19. Capability, URL parsing, protocol validation, and the initial domain allowlist stay
before `client.Do` in both modes because none requires DNS. `validateRedirect` keeps redirect-count
and protocol checks (plus the existing cross-origin Authorization stripping at its caller) but does
not resolve. Each accepted redirect is then independently classified by `ProxyFromEnvironment`:
a redirect from proxy to `NO_PROXY` enters the direct path and is resolved/validated/pinned once;
a redirect from direct to proxy skips target DNS; direct→direct resolves the new target once; and
proxy→proxy remains proxy-resolved. The domain allowlist remains an initial-request check, matching
current behavior; extending it to redirects is outside this patch.

Moving resolution from preflight to `RoundTrip` changes its timing: header parsing/request creation
can now precede a direct-route DNS/IP failure, and no socket is opened before that failure because
resolution occurs before the direct dial. It must not change public error categories. The localized
mechanism returns a typed internal target-validation error; legacy GET/POST callers recognize it
through `url.Error` wrapping and return the original `E_NET_DNS_FAILED`/`E_NET_IP_BLOCKED` error,
while structured `httpRequest`/`httpRequestBytes` continue returning `Err(Transport, message)`.
Tests must lock both mappings and prove that a proxy-selected request neither calls the injected
resolver nor produces a local DNS error.

The HTTP-based Stream constructors and Managed Agents client do not currently pin a resolved target
IP (V6). They should set `Proxy: http.ProxyFromEnvironment` while retaining their existing connect,
TLS-handshake, response-header, idle-connection, and overall-stream timeout choices. WebSocket
transport outside the measured HTTP literal set is not added to this sprint.

### D-1: SSRF interaction and recommendation

**Recommendation: preserve pinning on the direct route and knowingly replace it with trusted-proxy
routing only when `ProxyFromEnvironment` selects a proxy for that request.** This combines the
strongest property available in each mode:

- Option (a), merely adding `Proxy` beside the current closure, is rejected because the closure
  would rewrite the proxy dial address to the target IP; it is not merely a graceful degradation.
- Option (b), enabling proxy only where pinning is absent, is rejected because the principal `Net`
  operations would remain outside the egress boundary.
- Option (c) is adopted with a precise trust statement: proxy configuration is an operator decision
  that supersedes target-IP pinning for proxied requests. AILANG validates the proxy URL through
  Go's proxy selection/parsing path and surfaces selection/dial errors; it does not apply the
  target's private-IP or domain policy to the proxy endpoint itself. Corporate/local proxies often
  live on private networks, so doing so would make legitimate proxy deployment impossible.
- Option (d), adding a `ctx.Net` switch, is rejected for this sprint. `Net` authority still comes
  from the capability grant; routing is process deployment policy. `NO_PROXY` already supplies a
  standard per-target opt-out that falls back to the pinned direct path.

This does **not** claim equivalent SSRF resistance in proxy mode. The domain allowlist checks the
requested initial hostname, and redirect validation checks redirect protocol/count; neither can
prove which IP a remote proxy ultimately connects to. Deployments requiring the pinned-IP guarantee
must leave the destination out of proxy routing with `NO_PROXY`, or avoid setting a proxy for that
process.

### D-2: Managed Agents scope

`internal/executor/managed_agents/client.go` is in scope even though it is not an AILANG effect.
The user-facing title names the motivating `Net` defect, while the systemic definition of done is
“close the measured first-party proxy-ignoring HTTP residual.” V2 shows one residual constructor in
Managed Agents. Adding the standard proxy function there is localized, respects the executor/app
layer boundary (V12), and lets Vertex traffic follow the same corporate/CI egress policy. Excluding
it would require a follow-up whose entire content was one already-known constructor; that is not a
useful split.

### D-3: Behavior-change blast radius and compatibility

| Consumer | Behavior after this sprint | Compatibility action |
|---|---|---|
| AILANG user programs using `Net` | `HTTP_PROXY`/`HTTPS_PROXY` route eligible requests; `NO_PROXY` selects pinned direct routing | v0.33.1 changelog entry plus Net/effects documentation; no flag/version gate |
| AILANG programs using HTTP SSE/NDJSON Stream operations | eligible stream setup requests use the proxy; long-lived body reads remain governed by existing stream timers | document buffering/idle-timeout risk and test through a streaming proxy fixture |
| Eval harness | generated programs inherit the harness process proxy environment, so default poisoned lanes deny accidental egress and explicit eval environments must set proxy/`NO_PROXY` intentionally | focused eval-harness smoke using its local mock; no harness-specific bypass |
| Default CI/local `make test` | non-loopback effect/executor HTTP cannot silently escape; loopback `httptest` remains the hermetic path | production-constructor boundary tests plus out-of-sandbox poisoned package run |
| Runnable docs examples | network examples follow the invoking shell's proxy policy | update effects reference/run guidance; examples themselves need no syntax change |
| Managed Agents executor | Vertex SSE setup follows proxy environment | direct unit test of `defaultHTTPClient`; existing opt-in live probes remain opt-in |

There is no capability flag: granting `Net` or `Stream` still grants the effect, while the proxy
only chooses a route. There is no version gate: patch releases may correct a security-boundary
escape, and the standard opt-out is `NO_PROXY`. Add a changelog entry and remove the stale residual
paragraph from the development-workflow guide. Do not add a `docs/LIMITATIONS.md` entry; the
weaker proxy-mode target-IP guarantee belongs beside Net security/proxy documentation as an
operational security note.

### D-4: Streaming transports

SSE and NDJSON are long-lived response bodies. This design does **not** assume all proxies forward
them transparently. An HTTP proxy may buffer response chunks, impose an idle/request lifetime,
close a CONNECT tunnel, transform headers, or require authentication. The existing code has
connect/TLS/response-header bounds and stream idle/max-duration bounds (V6), but those do not prove
token-by-token delivery through an arbitrary proxy.

M3 therefore includes a local forward-proxy experiment that covers both HTTP absolute-form and
HTTPS CONNECT (using a test CA or an intentionally minimal tunnel), flushes at least two events with
a measured inter-event synchronization, and asserts the first event becomes observable before the
second is released. A proxy that buffers until EOF must fail. The experiment also holds a stream
open across a keep-alive interval and verifies cancellation/idle-timeout cleanup. Product behavior
for proxy authentication remains Go standard environment behavior; adding credential flags is a
non-goal.

### D-5: Tests under the poisoned lane

The current effect tests that actually send successful HTTP requests use local `httptest` servers
in the four transport-related test files, while the two public-httpbin subtests are already behind
`RequiresLiveNetwork` (V8–V9). Managed Agents production calls occur in opt-in live-test files;
ordinary tests use a stub client (V10). No existing live test should unset the poison inside a
process because Go caches environment proxy configuration (V11).

Handling policy:

- retain `RequiresLiveNetwork` for the two public Net subtests; the live lane must start without
  poison rather than unsetting it in-test;
- retain loopback `httptest` coverage and explicitly run it with poisoned proxy variables outside
  this sandbox; Go's `ProxyFromEnvironment` loopback bypass is already represented by the existing
  posture test, but the local run here was uninformative because socket binding was denied (V13);
- add fake-proxy tests that call the production Net/Stream/Managed Agents constructors, rather than
  tests of a standalone `http.Transport` literal;
- do not add `NO_PROXY=*` or explicitly unset proxy variables to make default tests green; those
  would evade the behavior being asserted.

### Implementation Plan and Milestones

#### M1 — Proxy-aware Net transport and SSRF regression tests (~1 day)

- [ ] Add the package-private request-aware direct/proxy round trip mechanism.
- [ ] Route all three `Net` constructors through it without changing public result/error categories.
- [ ] Add deterministic tests for direct pinning, proxy dialing, `NO_PROXY`, redirects, and
      capability/domain/protocol enforcement in both route modes.
- [ ] Remove preflight/redirect DNS resolution; preserve legacy and structured public error
      categories through typed internal validation-error unwrapping.

**Acceptance criteria:**

- [ ] `go test -count=1 ./internal/effects -run 'TestNetProxyBoundary|TestNetProxyDirectPin|TestNetProxyNoProxy|TestNetProxyRedirectControls' -v` passes outside the sandbox. Tests must use a target whose DNS answer/dial endpoint differs from the fake proxy and assert the observed dial/CONNECT destination. **Silent-revert check:** reverting production routing makes the fake proxy observe no request or makes the old pinning closure dial the target, so this gate fails.
- [ ] A direct/`NO_PROXY` subtest asserts the accepted connection reaches the injected validated IP rather than the request hostname's alternate endpoint. **Silent-revert check:** replacing both paths with ordinary proxy/default dialing fails the endpoint assertion.
- [ ] Capability denial, initial domain rejection, redirect protocol rejection, and redirect-count rejection run with a proxy selected; an injected resolver counter stays zero for proxy-selected initial and redirect requests. **Silent-revert check:** restoring preflight/redirect resolution increments the counter, while deleting proxy support makes the proxy-positive arm fail.
- [ ] Direct-route DNS/IP rejection tests assert exactly one resolver call, zero dial calls, and the
      existing legacy `E_NET_DNS_FAILED`/`E_NET_IP_BLOCKED` or structured `Err(Transport, ...)`
      category as applicable. **Silent-revert check:** double resolution fails the call count;
      ordinary dialing fails the zero-dial/security assertion; losing error unwrapping fails the
      exact category assertion.

#### M2 — Stream and Managed Agents constructors (~0.5 day)

- [ ] Add `ProxyFromEnvironment` to the HTTP SSE, SSE POST, NDJSON POST, and Managed Agents
      production transports without changing their timeout fields.
- [ ] Add constructor-level fake-proxy tests for all four paths.

**Acceptance criteria:**

- [ ] `go test -count=1 ./internal/effects -run 'TestStream(SSE|NDJSON).*ProxyBoundary' -v` passes outside the sandbox and the fake proxy observes GET, SSE POST, and NDJSON POST. **Silent-revert check:** a reverted constructor bypasses the fake proxy and fails its observation assertion.
- [ ] `go test -count=1 ./internal/executor/managed_agents -run TestDefaultHTTPClientProxyBoundary -v` passes and proves `defaultHTTPClient()` selects the fake proxy while retaining zero global request timeout and its existing header/idle timeouts. **Silent-revert check:** reverting the `Proxy` field makes selection/observation fail; a generic package pass is insufficient.

#### M3 — Streaming behavior and poisoned-lane integration (~0.75 day)

- [ ] Implement the buffering/CONNECT/keep-alive experiment described in D-4.
- [ ] Run all local-server transport suites under the poisoned environment outside the sandbox.
- [ ] Confirm both live public Net subtests remain skipped when `AILANG_LIVE_NET` is unset.

**Acceptance criteria:**

- [ ] The streaming fake-proxy test observes event 1 before permitting event 2, then cleanly closes after cancellation/idle timeout. **Silent-revert check:** direct routing produces zero proxy observations; a buffering proxy produces no early event; broken cleanup exceeds the test deadline.
- [ ] `HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 go test -count=1 ./internal/effects ./internal/executor/managed_agents` passes outside the sandbox. It must run the named proxy-boundary tests and local `httptest` tests, not merely report package success. **Silent-revert check:** the package command alone could pass after a revert, so acceptance additionally requires the named tests' `=== RUN`/`--- PASS` lines and the source audit in M4.
- [ ] `go test -count=1 -v ./internal/effects -run 'TestNetHttpPost/httpPost_to_httpbin.org|TestNetBodySizeLimit/small_response_under_limit'` with `AILANG_LIVE_NET` unset reports both named subtests as `SKIP`. **Silent-revert check:** this checks gating, not proxy support, and is therefore supporting evidence only; it cannot satisfy M1/M2 by itself.

#### M4 — Retire the Option-A residual and document compatibility (~0.75 day)

- [ ] Delete `effects_nil_proxy_remains_open` and its helper-only residual logic from
      `internal/testutil/egress_posture_test.go`; retain poison-sentinel, loopback, and raw-TCP
      posture coverage.
- [ ] Remove the matching Option-A residual/Non-Goals text from
      `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md` and replace it with a short
      historical note linking this implemented item after landing.
- [ ] Replace the development-workflow guide's seven-transport residual paragraph with the new
      proxy behavior, `NO_PROXY` direct-pin behavior, and proxy-mode pinning limitation.
- [ ] Add the v0.33.1 changelog entry and update the effects reference.

**Acceptance criteria:**

- [ ] A repository audit test/command reports zero production `http.Transport` literals with a nil
      `Proxy` in the audited first-party scope and proves the detector on a checked-in positive
      fixture. **Silent-revert check:** reverting any of the seven sites adds a finding; deleting or
      blinding the scanner makes the positive fixture fail.
- [ ] `grep -rn 'effects_nil_proxy_remains_open' internal/testutil docs design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md` returns zero, while `grep -rn 'loopback_bypasses_lane_poison' internal/testutil` returns a known-positive control. **Silent-revert check:** restoring the tripwire creates a hit; the control prevents an empty/mistyped search from passing silently.
- [ ] `go vet ./internal/effects ./internal/executor/managed_agents ./internal/testutil` and
      `make check-boundaries` pass. **Silent-revert check:** these quality gates would still pass
      after a behavioral revert, so they are explicitly non-sufficient and must be combined with
      the M1–M4 behavioral/source-audit gates.

### Files to Modify/Create

Exact test-helper filenames may be chosen by the implementer, but no production subsystem beyond
this list is authorized.

**Modified production files:**

- `internal/effects/net.go` — route three Net client construction paths through the request-aware transport
- `internal/effects/stream_ndjson.go` — environment proxy on NDJSON POST transport
- `internal/effects/stream_sse.go` — environment proxy on SSE GET and POST transports
- `internal/executor/managed_agents/client.go` — environment proxy on the Vertex SSE client

**Modified/new tests:**

- `internal/effects/*_test.go` — direct-pin, proxy, `NO_PROXY`, redirect, SSE, and NDJSON assertions
- `internal/executor/managed_agents/managed_agents_test.go` — production default-client proxy assertion
- `internal/testutil/egress_posture_test.go` — remove the self-retiring Option-A residual tripwire
- `internal/testutil/gatelint/*` — extend the existing first-party test lint only if it can audit nil-proxy production transports with a known-positive fixture; otherwise place the focused source audit beside the affected package

**Modified documentation:**

- `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md` — retire Option-A residual and Non-Goals text
- `docs/docs/guides/development-workflow.md` — replace residual guidance with closed-boundary behavior
- `docs/docs/reference/effects.md` — document proxy and direct-pin security semantics
- `changelogs/v0.18-current.md` — v0.33.1 behavior/security note

## Examples

### Corporate proxy route

```bash
HTTPS_PROXY=http://proxy.corp.example:8080 \
  ailang run --caps Net --entry main examples/runnable/http_simple.ail
```

The requested hostname is still checked against the AILANG Net policy. The proxy performs target
resolution, so the direct pinned-IP guarantee does not apply.

### Preserve direct pinned-IP routing for an allowlisted destination

```bash
HTTPS_PROXY=http://proxy.corp.example:8080 \
NO_PROXY=api.example.com \
  ailang run --caps Net --net-allow=api.example.com app.ail
```

`ProxyFromEnvironment` returns no proxy for the matching request; the direct route resolves,
validates, and dials the validated IP.

## Success Criteria

- [ ] M1–M4 acceptance criteria all pass outside the sandbox.
- [ ] All seven measured production HTTP transport sites are inside the proxy boundary.
- [ ] Direct and `NO_PROXY` Net routing retains target-IP pinning.
- [ ] Proxy-selected Net routing never feeds the proxy address into the target-IP substitution dialer.
- [ ] Capability grants, protocol/domain policy, redirect validation, budgets, timeouts, and size limits have regression coverage in the affected modes.
- [ ] Streaming proxy buffering/CONNECT/keep-alive behavior has an explicit experiment and recorded result.
- [ ] Option-A tripwire and residual documentation are retired and replaced by positive production-constructor assertions.
- [ ] Changelog and runtime effects documentation describe the compatibility/security change.

## Testing Strategy

**Unit tests:** inject resolver/dial behavior where needed; use a fake proxy and distinct target
endpoint; test proxy selection errors, direct pinning, `NO_PROXY`, timeout retention, and Managed
Agents client construction without public egress. Resolver-call and dial-call counters make the
single-resolution/no-proxy-resolution guarantees falsifiable.

**Integration tests:** invoke the actual Net, SSE, NDJSON, and Managed Agents constructor paths
through local origin/proxy servers under poisoned environment variables. Record named test output,
because a bare package PASS is vacuous if the new tests did not run.

**Regression tests:** retain existing loopback fixtures and security-policy tests; explicitly pair
proxy-selected paths with capability/domain/protocol/redirect rejection. Keep live public endpoint
tests opt-in.

**Sandbox rule:** any `bind: operation not permitted`, proxy `operation not permitted`, or outbound
network denial in this workspace is **UNINFORMATIVE UNDER SANDBOX**. The controller must rerun all
socket-bearing gates outside the sandbox. V13 records the baseline denial.

## Deferred Decisions

- Internal helper names and whether direct transports are cached per host — agent may choose, but
  cache keys must include every security-relevant route/target property and tests must rule out
  cross-request pin bleed.
- Exact local fake-proxy implementation and test CA arrangement — agent may choose.
- Whether the source audit extends `gatelint` or lives in an affected package — agent may choose;
  a known-positive fixture and all seven production sites are mandatory.
- Proxy observability fields/tracing — human at review may accept if localized; not required for DoD.

## Non-Goals

- Enforcing proxy use for raw TCP, SSH, WebSocket, browser/WASM fetch, or subprocesses that ignore
  proxy environment variables.
- Treating an operator-configured private proxy address as an SSRF target or applying the target
  domain allowlist to the proxy hostname.
- Preserving target-IP pinning *through* an HTTP proxy; ordinary HTTP proxy protocols do not provide
  that guarantee to the client.
- Adding proxy URL/authentication CLI flags, PAC support, certificate installation, or a second
  `ctx.Net` routing policy.
- Guaranteeing that every third-party proxy streams SSE/NDJSON without buffering; this sprint
  measures and documents the risk with a controlled proxy.
- Changing AILANG syntax, effect rows, capability names/grants, network budgets, public operation
  signatures, or the eval harness's capability flags.
- Refactoring unrelated HTTP clients not identified by the V2 production-literal audit.

## Timeline

| Milestone | Estimate | Cumulative |
|---|---:|---:|
| M1 — request-aware Net route + security tests | 1 day | 1 day |
| M2 — Stream + Managed Agents transports/tests | 0.5 day | 1.5 days |
| M3 — streaming experiment + poisoned integration | 0.75 day | 2.25 days |
| M4 — tripwire retirement, audit, docs, final gates | 0.75 day | 3 days |

**Total:** 3 days, with a 2–4 day execution range depending on fake CONNECT-proxy test complexity.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Target pin closure rewrites the proxy address | High: proxy mode is broken and may connect to the wrong endpoint | Separate request-aware direct/proxy paths; assert the dial endpoint |
| Proxy resolves an allowed hostname to a private/metadata IP | High: proxy mode weakens the direct target-IP SSRF guarantee | Explicit operator-trust decision, clear docs, preserve `NO_PROXY` direct pin; never claim equivalence |
| Redirect switches proxy applicability | High: direct redirect could become unpinned or proxy policy could be ignored | Evaluate `ProxyFromEnvironment` per request/round trip and test redirects across `NO_PROXY` boundaries |
| Moving IP validation into `RoundTrip` changes timing or error wrapping | High: callers could lose stable `E_NET_*`/`NetError.Transport` categories | Typed internal validation error, unwrap through `url.Error`, and exact-category tests for legacy and structured operations |
| SSE/NDJSON buffering or tunnel timeout | Medium: delayed tokens or failed long runs | M3 forward-proxy flush/keep-alive/cancellation experiment; retain existing stream bounds |
| Go proxy environment caching makes tests order-dependent | Medium: false green/red | subprocess tests where environment identity matters; never mutate/unset poison after proxy use |
| Loopback tests fail in the managed sandbox | Low to code, high to verification | label denial uninformative and rerun outside sandbox; do not rewrite tests around sandbox policy |
| Managed Agents proxy authentication/corporate TLS differs by deployment | Medium | standard Go behavior, documented operationally; no hidden fallback |

## Conflict Surface

### Existing HTTP construction machinery and reuse decision

The production audit in V17 establishes the complete relevant construction surface:

- **Bare/inline clients:** 29 `http.Client` literals exist. Clients without an explicit transport
  already inherit proxy-aware `http.DefaultTransport` (V18), so they need no new factory and are
  outside the seven-site residual.
- **Custom transports:** the seven literals in V2 are the only measured nil-Proxy production
  transports. The three Net literals contain validated-IP substitution and are replaced by the
  localized request-aware mechanism; the three Stream literals and one Managed Agents literal
  reuse Go's `http.ProxyFromEnvironment` directly.
- **Dial helpers:** the only custom production `DialContext` logic is the six Net closure lines
  implementing validated-IP substitution plus ordinary timeout dialers in SSE/NDJSON (V17). Reuse
  the ordinary `net.Dialer` behavior in both new route transports, but retain substitution only in
  the direct path after its one validation.
- **RoundTrippers, shared factories, clones, and proxy helpers:** production has zero custom
  `RoundTripper`/`RoundTrip`, zero `http.DefaultTransport` references, zero `Transport.Clone`, zero
  `ProxyFromEnvironment`, zero `httpproxy`, and no shared HTTP client/transport factory (V17). There
  is therefore no repository abstraction to extend or import.

No existing mechanism provides both per-request proxy selection and validated-IP direct dialing.
The new package-private `internal/effects` RoundTripper is consequently localized new machinery,
not a duplicate; Stream and Managed Agents remain simple standard-library configuration.

### Runtime and subsystem collisions

- `internal/effects/net.go`: concurrent changes to Net validation, redirects, HTTP response shaping,
  resolver injection, or the three client constructors can collide. Invariants: `Net` capability,
  protocol validation, initial domain allowlist, redirect count/protocol validation, response size,
  timeout, stable public validation-error categories, exactly-once direct resolution, no proxy-route
  target resolution, and direct target-IP pinning. Intentional timing change: direct DNS/IP failures
  move from preflight into `client.Do`, before any dial.
- `internal/effects/stream_sse.go`, `internal/effects/stream_ndjson.go`, and
  `internal/effects/stream_context.go`: concurrent Stream security, timeout, connection-budget,
  event-order, or read-loop work can collide. Invariants: `Stream` capability/budget grants, URL
  validation, allowlist/private/local checks, connection limits, event ordering, idle timeout, and
  max duration.
- `internal/executor/managed_agents/client.go`: concurrent Vertex endpoint, ADC, SSE, timeout, or
  executor work can collide. Preserve request context cancellation, authorization headers,
  response-header timeout, idle-connection timeout, and absence of a global streaming timeout.
- `internal/testutil/egress_posture_test.go` and `internal/testutil/gatelint/`: concurrent egress
  boundary/lint changes can collide. Preserve poison sentinel, loopback bypass, raw-TCP residual,
  and known-positive anti-vacuity controls while replacing only the Option-A nil-proxy assertion.
- `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md`,
  `docs/docs/guides/development-workflow.md`, `docs/docs/reference/effects.md`, and
  `changelogs/v0.18-current.md`: concurrent release/documentation edits can conflict textually and
  must not leave contradictory “residual open” guidance.

### Architecture/import directions

`internal/effects` is core/runtime; it must not import dashboard/app packages.
`internal/executor` is an agent-executor integration package and must not be imported by core.
The shared mechanism therefore stays inside `internal/effects`; Managed Agents receives its own
localized standard proxy field rather than importing an effects helper. `make check-boundaries`
enforces the repository's core↔apps restrictions and is green at baseline (V12–V13).

### Programs and fixtures that must still work

These existing fixtures were verified to exist (V15) and exercise Net-facing workflows:

- `examples/runnable/http_simple.ail`
- `examples/runnable/demo_ai_api.ail`
- `examples/runnable/ai_call.ail`
- `examples/runnable/claude_haiku_call.ail`
- `examples/tests/micro_net_fetch.ail`

Their syntax and capability requirements must not change. Network outcome may intentionally change
when the invoking environment selects a proxy.

### Intentional incompatibilities

- A process with `HTTP_PROXY`/`HTTPS_PROXY` set may now route, filter, authenticate, buffer, or fail
  requests that previously bypassed that proxy.
- A proxied Net request no longer guarantees a socket to AILANG's pre-validated target IP; direct
  and `NO_PROXY` requests still do.
- The Option-A `effects_nil_proxy_remains_open` test is intentionally deleted because success would
  now assert the bug.

No AILANG grammar, types, effect rows, public operation signatures, or capability grant semantics
change.

## Verification Log

All commands ran in this worktree at the unmodified design baseline. Socket-bearing failures marked
uninformative are not code defects and are not accepted as implementation results.

| ID | Codebase claim | Exact command | Observed output |
|---|---|---|---|
| V1 | The version baseline is v0.33.0 and the active changelog file is identified | `cat std/VERSION; ls changelogs \| grep current` | `v0.33.0`; `v0.18-current.md` |
| V2 | Seven production `http.Transport` literals exist across four files in first-party scope | `find internal cmd runtime std tests -name '*.go' ! -name '*_test.go' -type f -print0 \| xargs -0 grep -nH 'http\.Transport{' \| sort`; same pipeline ending `grep -h ... \| wc -l`; same ending `grep -l ... \| sort \| wc -l` | Listed `internal/effects/net.go` (3), `stream_ndjson.go` (1), `stream_sse.go` (2), `internal/executor/managed_agents/client.go` (1); counts `7` literals and `4` files |
| V3 | Production scope has no `ProxyFromEnvironment`; the search instrument sees the test controls | `find internal cmd runtime std tests -name '*.go' ! -name '*_test.go' -type f -print0 \| xargs -0 grep -nH 'ProxyFromEnvironment'`; control: same without `! -name '*_test.go'` | production: no output; control: three hits in `internal/testutil/egress_posture_test.go` (one comment, two code uses) |
| V4 | Net preflight and redirect controls, plus three target-IP-substituting transports, exist in the described order | `nl -ba internal/effects/net.go \| sed -n '53,125p;164,240p;260,375p;535,623p'` | Reads show capability → parse → protocol → initial domain allowlist → resolve/validate IP → dial validated IP for GET/POST/shared request; redirect validator enforces count, protocol, and `resolveAndValidateIP` |
| V5 | Net operations and HTTP Stream operations check their respective capability/budget before dialing | `rg -n 'HasCap\("Net"\)|RequireCapWithBudget\("Stream"' internal/effects/net.go internal/effects/stream_sse.go internal/effects/stream_ndjson.go` | Net `HasCap` hits in legacy and structured operations; Stream budget hits in SSE connect/post and NDJSON post |
| V6 | HTTP Stream and Managed Agents transports have streaming-oriented timeouts but no target-IP substitution closure | `nl -ba internal/effects/stream_ndjson.go \| sed -n '80,98p'; nl -ba internal/effects/stream_sse.go \| sed -n '67,91p;328,349p'; nl -ba internal/executor/managed_agents/client.go \| sed -n '136,147p'` | Stream literals use ordinary `net.Dialer` plus connect/TLS/header bounds; Managed Agents uses header/idle bounds and no global timeout; none substitutes a validated target IP |
| V7 | Existing Net/Stream contexts have security configuration but no proxy-routing field; negative search has positive field controls | `nl -ba internal/effects/context.go \| sed -n '181,223p'; nl -ba internal/effects/stream_context.go \| sed -n '12,72p'; rg -n 'Proxy' internal/effects/context.go internal/effects/stream_context.go` | Context reads show timeout, allowlist, localhost/private-IP, budget-related settings; `rg Proxy` returns no output while the displayed structs contain many other fields |
| V8 | The self-retiring Option-A tripwire exists and explicitly instructs retirement when Option B lands | `nl -ba internal/testutil/egress_posture_test.go \| sed -n '14,117p'` | `TestEgressPosture` includes `effects_nil_proxy_remains_open`; its comment says Option B should trip red and the tripwire/Non-Goals text must be retired; loopback and proxy controls are adjacent |
| V9 | Transport-path effect tests use local servers, while the two public httpbin execution subtests use `RequiresLiveNetwork` | `for f in internal/effects/net_test.go internal/effects/net_bytes_test.go internal/effects/stream_sse_test.go internal/effects/stream_ndjson_test.go; do printf '%s ' "$f"; rg -c 'httptest\.New(Server\|TLSServer\|UnstartedServer)' "$f"; done`; `rg -n 'httpbin|RequiresLiveNetwork' internal/effects/net_test.go` | Local-server counts: 1, 4, 4, 3; httpbin execution subtests call `RequiresLiveNetwork` before requests (other httpbin hit is argument-validation setup) |
| V10 | Ordinary Managed Agents tests use an HTTP stub; live entrypoints carry explicit opt-in gates or call the shared gated helper | `sed -n '1,80p' internal/executor/managed_agents/managed_agents_test.go`; `rg -n 'os\.Getenv|t\.Skip|func Test|requireFeatureLive' internal/executor/managed_agents/*_live_test.go`; `sed -n '480,505p' internal/executor/managed_agents/managed_agents_features_live_test.go` | `stubHTTP` implements `Do`; mount/reuse/raw live tests check named env vars, and feature tests call `requireFeatureLive`, whose body skips unless `AILANG_LIVE_MA_FEATURES=1` and ADC is available |
| V11 | The shared live-network gate forbids a poisoned live lane and warns not to unset proxy variables after use | `nl -ba internal/testutil/gate.go \| sed -n '37,84p'` | Four proxy env names are checked; poison yields `LiveNetworkFatal`; comment records process-wide proxy caching and `t.Fatalf` behavior |
| V12 | Architecture classifies effects as core/runtime and enforces core↔apps import restrictions | `sed -n '45,125p' ARCHITECTURE.md; sed -n '1,90p' .claude/rules/architecture.md` | `internal/effects` is back end/runtime and in core; rules forbid core→dashboard and dashboard→compiler imports; executor/provider distinction is documented |
| V13 | Baseline quality gates are green, but socket-bearing package/loopback tests are uninformative in this sandbox | `go vet ./internal/effects ./internal/executor/managed_agents ./internal/testutil`; `make check-boundaries`; unpoisoned and poisoned `go test -count=1 ./internal/effects ./internal/executor/managed_agents ./internal/testutil`; poisoned `go test -count=1 -v ./internal/testutil -run 'TestEgressPosture/loopback_bypasses_lane_poison'` | vet rc=0; boundaries `OK` rc=0. Test runs rc=1 with `httptest: failed to listen ... bind: operation not permitted` and poison dial `operation not permitted`: **UNINFORMATIVE UNDER SANDBOX** |
| V14 | Runtime documentation currently names the seven-site residual as open | `sed -n '300,345p' docs/docs/guides/development-workflow.md` | Guide says poisoned lanes do not govern nil-Proxy transports, lists seven across four files, and names the Option-B tripwire |
| V15 | Five named Net-facing regression fixtures exist | `for f in examples/runnable/http_simple.ail examples/runnable/demo_ai_api.ail examples/runnable/ai_call.ail examples/runnable/claude_haiku_call.ail examples/tests/micro_net_fetch.ail; do test -f "$f" && echo "EXISTS $f"; done` | Five `EXISTS <path>` lines, one for every named fixture |
| V16 | Duplicate/coverage search did not identify a related proxy-boundary design; the instrument returned known-positive unrelated matches | `ailang docs search --limit 10 "net effect proxy boundary"`; `ailang docs search --neural --limit 10 "net effect proxy boundary"` | Both returned a populated ten-result SimHash list whose entries concern debug events, record inference, parser/array/type bugs, etc.; no proxy/egress design appeared. Neural mode fell back to the same populated result set in this environment, so no similarity threshold is claimed |
| V17 | Existing production HTTP construction/routing abstractions were audited; negative classes use the 29 inline-client hits from the same matcher as a known-positive control | `find internal cmd runtime std -name '*.go' ! -name '*_test.go' -type f -print0 \| xargs -0 grep -nH -E 'RoundTripper\|RoundTrip\(\|DefaultTransport\|http\.Client\{\|Transport\.Clone\|DialContext\|ProxyFromEnvironment\|httpproxy' \| grep -v '/\.claude/' \| sort`; per-pattern counts using the same `find ... \| xargs ... grep -nH -E "$p" \| wc -l` for `RoundTripper\|RoundTrip\(`, `DefaultTransport`, `Transport\.Clone`, `http\.Client\{`, `DialContext`, `ProxyFromEnvironment`, and `httpproxy`; `find internal cmd runtime std -name '*.go' ! -name '*_test.go' -type f -print0 \| xargs -0 grep -nH -E 'Transport:[[:space:]]' \| sort` | 29 `http.Client{` hits: `cmd/ailang/{chains_live,coordinator_browse,coordinator_lifecycle,dashboard,eval_events,messages_send,pkg_info,pkg_publish,pkg_unpublish,server}.go`; `internal/ai/configdriven/provider.go`, `internal/ai/gemini/client.go`, `internal/ai/ollama/step.go`, `internal/auth/gcp/adc.go`, `internal/coordinator/http_broadcaster.go`, `internal/effects/net.go` (3), `internal/effects/stream_ndjson.go`, `internal/effects/stream_sse.go` (2), `internal/eval_harness/telemetry_reporter.go`, `internal/executor/managed_agents/client.go`, `internal/mcp_client/client.go`, `internal/messaging/{embedder_gemini,embedder_openai}.go`, `internal/pipeline/metrics.go`, `internal/pkg/registry.go`, and `internal/secrets/cloud_approver.go`. Counts: custom `RoundTripper`/`RoundTrip(` 0; `DefaultTransport` 0; `Transport.Clone` 0; `http.Client{` 29 (known-positive control); `DialContext` 13: six functional validated-IP references plus one descriptive comment in `net.go`, two ordinary dialer references in `stream_ndjson.go`, and four ordinary dialer references in `stream_sse.go`; `ProxyFromEnvironment` 0; `httpproxy` 0. The `Transport:` audit finds the same seven client assignments in `net.go` (3), `stream_ndjson.go` (1), `stream_sse.go` (2), and Managed Agents (1), plus one unrelated comment in `cmd/ailang-microrag-mcp/main.go`. No shared factory appears: all client constructions are inline. |
| V18 | A client with nil `Transport` inherits a proxy-aware default transport | `go version; goroot=$(go env GOROOT); printf 'GOROOT=%s\n' "$goroot"; nl -ba "$goroot/src/net/http/client.go" \| sed -n '204,209p'; nl -ba "$goroot/src/net/http/transport.go" \| sed -n '40,55p'` | `go version go1.26.5 darwin/arm64`; GOROOT `/Users/voightkampff/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.5.darwin-arm64`; `client.go:204-208` returns `c.Transport` when non-nil and otherwise `DefaultTransport`; `transport.go:46-48` defines `var DefaultTransport RoundTripper = &Transport{` followed by `Proxy: ProxyFromEnvironment` and the default dialer. |
| V19 | Today `resolveAndValidateIP` is called in three initial preflight paths and once in redirect validation | `rg -n 'resolveAndValidateIP' internal/effects/net.go internal/effects --glob '*.go'` | Calls at `internal/effects/net.go:85` (legacy GET preflight), `:201` (legacy POST preflight), `:317` (`validateRedirect`), and `:565` (shared structured-request preflight); definition/comments at `internal/effects/net_security.go:62,78`. |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|---|---:|---|
| A1: Determinism | +1 | CI and deployed routing now obey explicit process proxy policy instead of constructor accident |
| A2: Replayability | 0 | External network responses remain externally determined |
| A3: Effect Legibility | +1 | HTTP egress consistently crosses the documented process boundary |
| A4: Explicit Authority | +1 | Effect capability remains mandatory; operator proxy/`NO_PROXY` policy becomes effective |
| A5: Bounded Verification | +1 | Fake-proxy, pin, and positive-control tests make each route locally falsifiable |
| A6: Safe Concurrency | 0 | No language concurrency semantics change; transport mode must not mutate shared state |
| A7: Machines First | +1 | Poisoned CI mechanically detects unintended effect egress |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | Proxy operational cost is external; no new language cost model |
| A10: Composability | +1 | Standard Go proxy environment composes with CI and corporate deployment |
| A11: Structured Failure | 0 | Existing Net/Stream transport error forms are retained |
| A12: System Boundary | +1 | Closes the measured first-party HTTP(S) boundary residual |

**Net Score: +7** → Proceed to sprint planning.

### Hard Violation Check

- [x] A1: no new hidden nondeterminism; routing follows declared process environment.
- [x] A3: no new hidden effect; existing Net/Stream effects own the requests.
- [x] A4: no capability is granted implicitly; proxy routing does not grant Net/Stream.
- [x] A7: production-constructor and anti-vacuity tests are machine-checkable.

## Related Documents

- [M-CI-FLAKE-SYSTEMIC-FIX](../../implemented/v0_33_1/m-ci-flake-systemic-fix.md) — implemented
  Option A, created the poisoned lane and self-retiring residual tripwire; this document is its
  deliberately separate production-runtime Option B.
- [Development workflow](../../../docs/docs/guides/development-workflow.md) — current operational
  boundary and residual guidance to update when this lands.

The repository search is recorded in V16. It found no duplicate proxy-boundary design; the listed
high SimHash matches were unrelated topics and no neural similarity score is asserted.

## References

- Go standard library: `net/http.ProxyFromEnvironment` and `Transport.Proxy` behavior.
- `ARCHITECTURE.md` and `.claude/rules/architecture.md` — enforced import directions.
- `internal/testutil/egress_posture_test.go` — poison, loopback, raw-TCP, and retiring residual probes.
- [Design Axioms](/docs/references/axioms)

## Future Work

- If deployments need both a remote proxy and cryptographic proof of the final target endpoint,
  design an authenticated proxy/attestation protocol; ordinary HTTP CONNECT cannot provide it.
- Audit non-HTTP first-party clients (WebSocket, raw TCP, SSH, subprocesses) as separate boundary
  items using mechanisms appropriate to each protocol.
- Consider structured trace fields for selected proxy mode without logging proxy credentials.

---

## Quorum Verification Log (mission iteration 150, 2026-08-06)

**Status: PARKED `needs-human-review`.** Two quorum rounds, both BLOCKED. Round 2's remaining
objection needs a scope decision that is not the controller's to make (see D-6 below).

### Round 1 — BLOCKED (`m-net-effect-proxy-boundary-2026-08-06T04-38-48Z.json`)
- **`gpt5-6-sol` reject** — the conflict surface never audited existing HTTP construction/routing
  abstractions, so a new RoundTripper might duplicate machinery. **Controller ran the audit itself**
  rather than forwarding the objection: production has **0** custom `RoundTripper`s, **0**
  `DefaultTransport` uses, **0** `Transport.Clone`, and no shared factory (**29** inline
  `http.Client{}` sites as the known-positive control). Handed to the designer as a measurement →
  rows **V17**/**V18**.
- **`gemini-3-1-pro` reject** — a genuine design defect: the doc specified target-IP resolution in
  *two* places (preflight `resolveAndValidateIP` **and** the new RoundTripper), a TOCTOU
  DNS-rebinding race, plus a broken-proxied-request risk where local DNS is unavailable. Routed
  without a controller-invented resolution. Designer made the direct RoundTripper the sole
  resolve-validate-dial site and skips local target DNS entirely on proxy routes → row **V19**.

### Round 2 — BLOCKED (`m-net-effect-proxy-boundary-2026-08-06T04-47-14Z.json`)
- **`gemini-3-1-pro` reject** — unverified premise: the doc claims the moved validation will still
  surface `E_NET_DNS_FAILED`/`E_NET_IP_BLOCKED`/`Err(Transport)`, but never verified where those are
  produced today. **Carve-out-eligible** (concrete fix, no direction dispute). Controller measured it
  in advance so the resume is cheap — see V20-PENDING below.
- **`gpt5-6-sol` reject** — the seven-site completeness claim rests on textual matching, which cannot
  see aliased `net/http` imports, `new(http.Transport)`, post-construction `Client.Transport =`,
  transport-returning factories, or custom `RoundTripper`s. Asks that V2/V17 **and the M4 source
  gate** be replaced by a checked-in `go/packages` AST/type analyzer with positive fixtures for each
  shape. **NOT carve-out-eligible** — it materially expands scope and requires a judgment call.

### Controller measurements banked for the resume (all first-party, controls fired)

**Are the shapes the AST analyzer would catch actually present at HEAD? No — all five are zero.**

| Shape | Production hits | Control (proves the matcher sees positives) |
|---|---:|---|
| aliased `net/http` import | **0** | 1505 alias-shaped hits incl. a real `_ "embed"` |
| `new(http.Transport)` | **0** | 4 `new(` hits (e.g. `new(Executor)`) |
| post-construction `.Transport =` | **0** | 8 `Transport:` struct-literal fields |
| func returning `*http.Transport`/`RoundTripper` | **0** | 2 funcs returning `*http.Client` |
| type implementing `RoundTrip(` | **0** | (matches V17's independent zero) |

So the seven-site claim is **empirically complete today**. The reviewer's surviving point is about
**durability of the M4 gate**, not about present correctness: a grep scanner cannot *prevent* a
future escape via one of those shapes.

**V20-PENDING** (gemini's requested row, measured by the controller, not yet written into the table):
`E_NET_IP_BLOCKED` is produced at `internal/effects/net_security.go:27,34,46,51,56` and
`E_NET_DNS_FAILED` at `:90,94`. These surface through `makeResultErr("Transport", …)` at
`internal/effects/net.go:551,556,567,605,631,639` (control: 11 `makeResultErr("` sites overall).
`net.go:567` is the preflight `resolveAndValidateIP` path that this design moves, and `net.go:631`
is the post-`client.Do` path where a `url.Error` would arrive — so those two are exactly the sites
the error-mapping change must update.

### D-6 — THE PARKED DECISION (for the human)

**What should the M4 completeness gate be?**
- **Option A — grep/source scanner now** (as designed), with the AST analyzer filed as a separate
  follow-up item. Justified by the measurement above: zero instances of every alternative shape
  exist today, so the cheap gate is *sufficient for present correctness*. Keeps the sprint at 3 days.
- **Option B — `go/packages` AST/type analyzer in-sprint**, with positive fixtures for all five
  shapes. Durable against future escapes and satisfies the reviewer outright, but adds roughly a day
  (3d → 4d) and makes a security sprint also a static-analysis sprint.

The controller's reservation, recorded separately and **not** blocking: **D-1 knowingly trades
target-IP SSRF pinning on proxied requests.** The doc is explicit about this, preserves pinning on
direct/`NO_PROXY` routes, and never claims equivalence — but it is a real security-boundary change
and deserves human ratification alongside D-6.
