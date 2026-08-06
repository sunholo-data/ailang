# Sprint Plan — M-NET-EFFECT-PROXY-BOUNDARY

**Design doc**: [m-net-effect-proxy-boundary.md](./m-net-effect-proxy-boundary.md)
**Sprint ID**: `M-NET-EFFECT-PROXY-BOUNDARY`
**Target**: v0.33.1
**Planned at**: HEAD `945f36727` (branch `dev`), 2026-08-06
**Baseline for all "base result" rows below**: `945f36727`, working tree clean except three
untracked-by-this-work files (`.claude/fmt_hook_events.jsonl`, `docs/static/benchmarks/os/*.json`)
which touch no Go package in scope.
**Duration**: 3 days, 4 milestones (M1–M4) — **unchanged from the design doc**
**Risk**: Medium-High (production networking + an SSRF control change in one sprint)
**Executor lane**: opus-required (design doc `Planner-Lane`)

---

## 0. Reader's summary — what a planner changed and why

This plan does **not** re-open the design. The design doc is the reviewed artifact and **wins on any
disagreement** (mission rule vii). What this plan adds is the thing a design doc cannot carry:
**every acceptance command executed on a pristine tree first**, so that no gate silently measures the
repository instead of the change (mission rule 3e).

Four things came out of that baselining that the executor must know before writing a line of code:

1. **`go build ./...` is rc=1 on unmodified `dev`.** Confirmed first-party (§2.1). It is *structurally*
   red — `cmd/wasm` is `//go:build js && wasm` and has no native `main`. It can never be an acceptance
   gate on this repo. A scoped substitute is baselined green and used instead.
2. **The M1 and M2 named-test gates are VACUOUSLY GREEN at base** — `rc=0`, `0` `=== RUN` lines,
   `[no tests to run]` (§2.2). A gate written as "`rc=0`" would pass after a *complete* revert that
   deletes the tests. Every named-test AC in this plan therefore carries a **`=== RUN` count
   assertion**, not just an exit code.
3. **The controller's briefing claim that this item's landing "must flip the AC10(d) tripwire RED" is
   REFUTED** (§5, M4). `testEffectsProxyResidual` builds its **own** `&http.Transport{}` inside
   `internal/testutil/egress_posture_test.go` and imports **nothing** from `internal/effects`. No
   production change can flip it. The doc's actual M4 AC (grep-to-zero + control) is correct and is
   what this plan implements; the "observe it go red" framing is not, and no AC depends on it.
4. **This workspace is NOT socket-restricted**, unlike the design doc's V13 (§2.3). All socket-bearing
   baselines ran informative and green here. That does **not** retire the sandbox rule — the executor
   may run somewhere stricter — so every socket-bearing gate keeps its
   **UNINFORMATIVE UNDER SANDBOX** escape hatch, and this plan records what those commands look like
   when they *are* informative so a denial is distinguishable from a failure.

---

## 1. Milestone → acceptance-criterion correspondence (rule vii)

The design doc's acceptance criteria are the contract. This table names exactly which the milestone
closes. **No milestone closes an AC belonging to another milestone**, and no AC is left unowned.

| Milestone | Design-doc ACs closed | Doc lines |
|---|---|---|
| **M1** — Proxy-aware Net transport + SSRF regression tests | M1 AC1, AC2, AC3, AC4 | 249–256 |
| **M2** — Stream + Managed Agents constructors | M2 AC1, AC2 | 266–267 |
| **M3** — Streaming behaviour + poisoned-lane integration | M3 AC1, AC2, AC3 | 277–279 |
| **M4** — Retire the Option-A residual, source audit, docs | M4 AC1, AC2, AC3 | 303–311 |

Coverage check: the doc states 4 + 2 + 3 + 3 = **12** acceptance criteria. All 12 are owned above.
The doc's top-level **Success Criteria** (lines 364–371) are the roll-up and are verified at the M4
boundary, not separately.

---

## 2. Pristine-tree baselines (rule 3e)

Every command below was executed at `945f36727` **before any implementation work**. A gate that is
already green at base cannot, by itself, demonstrate the change; a gate that is already red at base
measures the repo. Both cases are labelled.

### 2.1 Build and static gates

| Command | Base result at `945f36727` | Verdict |
|---|---|---|
| `go build ./...` | **rc=1** — `cmd/wasm`: `function main is undeclared in the main package`; `gen/main`: same | **ALREADY RED AT BASE — EXCLUDED as a gate.** Cause is structural: `cmd/wasm/*.go` carry `//go:build js && wasm`, so under native `GOOS` the package has no `main`. Not fixable by, and unrelated to, this sprint. |
| `go build ./internal/effects/... ./internal/executor/... ./internal/testutil/...` | **rc=0** | **GREEN at base — adopted as the scoped build gate** in place of `go build ./...`. Covers every package this sprint touches. |
| `go vet ./internal/effects ./internal/executor/managed_agents ./internal/testutil` | **rc=0** | GREEN at base. **NON-SUFFICIENT** — the doc says so itself (AC M4-3) and it is right: vet passes after a behavioural revert. Kept only as a regression tripwire. |
| `make check-boundaries` | **rc=0**, `OK: no architecture boundary violations.` | GREEN at base. **NON-SUFFICIENT**, same reason. Load-bearing only for the negative case in §4.3. |

### 2.2 Named-test gates — the vacuous-green finding

| Command | Base result at `945f36727` | Verdict |
|---|---|---|
| `go test -count=1 ./internal/effects -run 'TestNetProxyBoundary\|TestNetProxyDirectPin\|TestNetProxyNoProxy\|TestNetProxyRedirectControls' -v` | **rc=0**, `=== RUN` count = **0**, `testing: warning: no tests to run`, `ok ... [no tests to run]` | **VACUOUSLY GREEN.** Exit code alone is worthless here. AC must assert `=== RUN` count ≥ 4 and one `--- PASS` per named test. |
| `go test -count=1 ./internal/effects -run 'TestStream(SSE\|NDJSON).*ProxyBoundary' -v` | **rc=0**, `=== RUN` count = **0**, `[no tests to run]` | **VACUOUSLY GREEN.** Same remedy. |
| `go test -count=1 ./internal/executor/managed_agents -run TestDefaultHTTPClientProxyBoundary -v` | **rc=0**, `[no tests to run]` | **VACUOUSLY GREEN.** Same remedy. |

### 2.3 Socket-bearing / integration gates

**This workspace permitted socket binding.** All of the following were informative here — which
*contradicts* the design doc's V13 row (recorded under a socket-denied sandbox). The rule stands
anyway; see §6.

| Command | Base result at `945f36727` | Verdict |
|---|---|---|
| `go test -count=1 ./internal/effects ./internal/executor/managed_agents` (unpoisoned) | **rc=0** — `ok effects 11.866s`, `ok managed_agents 0.484s` | GREEN at base. Regression gate only. |
| `HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 go test -count=1 ./internal/effects ./internal/executor/managed_agents` | **rc=0** — `ok effects 13.590s`, `ok managed_agents 0.250s` | **ALREADY GREEN AT BASE.** The bare package command proves nothing. The design doc already anticipates this (AC M3-2: "must run the named proxy-boundary tests … not merely report package success"). This plan enforces that with a `=== RUN`/`--- PASS` line check. |
| `go test -count=1 ./internal/testutil` | **rc=0** — `ok 2.342s` | GREEN at base. |
| `HTTP_PROXY=… HTTPS_PROXY=… go test -count=1 -v ./internal/testutil -run TestEgressPosture` | **rc=0**; 4 subtests run: `poison_sentinel_denies_HTTP_egress`, `loopback_bypasses_lane_poison`, `raw_TCP_remains_open`, `effects_nil_proxy_remains_open` | GREEN at base. Note the residual subtest **skips** unless `AILANG_LIVE_NET=1` (`requireLiveEgressPosture`). |
| `env -u AILANG_LIVE_NET go test -count=1 -v ./internal/effects -run 'TestNetHttpPost/httpPost_to_httpbin.org\|TestNetBodySizeLimit/small_response_under_limit'` | **rc=0**; both named subtests report **`--- SKIP`** | **ALREADY GREEN AT BASE.** The doc itself labels this "supporting evidence only … cannot satisfy M1/M2 by itself" (AC M3-3). Retained unchanged as a *non-regression* check. |

### 2.4 Source-audit gates

| Command | Base result at `945f36727` | Verdict |
|---|---|---|
| `grep -rn 'effects_nil_proxy_remains_open' internal/testutil docs design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md` | **4 hits** — `internal/testutil/egress_posture_test.go:20,67`; `docs/docs/guides/development-workflow.md:330`; `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md:394` | **CORRECTLY RED AT BASE** (must reach 0). This is a properly falsifiable gate. |
| `grep -rn 'loopback_bypasses_lane_poison' internal/testutil` (anti-vacuity control) | **2 hits** — `egress_posture_test.go:18,38` | Control **FIRES**. The empty search will be a measurement, not a broken instrument. |
| Production `http.Transport{` literal audit (V2 re-derivation) | **7 literals / 4 files** — `internal/effects/net.go:96,212,587`, `stream_ndjson.go:80`, `stream_sse.go:70,329`, `internal/executor/managed_agents/client.go:141` | Reproduces V2 exactly. Target after sprint: **0** nil-`Proxy`. |
| Production `ProxyFromEnvironment` count | **0**; test-inclusive control **3** (all in `internal/testutil/egress_posture_test.go`) | Reproduces V3 exactly. Control fires. |

---

## 3. Velocity basis

Measured from the immediate predecessor in the same subsystem (`M-CI-FLAKE-SYSTEMIC-FIX`, which this
item depends on and which landed 2026-08-05/06), plus the two largest recent feature landings:

| Commit | Milestone | Lines changed |
|---|---|---:|
| `c440a1628` | CI-flake M1 (testutil gate + bounded helpers) | 361 |
| `368f940cf` | CI-flake M2 (call-site migration + httptest coverage) | 201 |
| `13c570063` | CI-flake M3 (gatelint + AC10 egress probe) | 359 |
| `4b47f8b0a` | CI-flake M4 (poisoned proxy across 6 CI legs) | 87 |
| `c9e1a4f98` | CI-flake M5 (docs/changelog reconciliation) | 242 |
| `34951811d` | test-block execution fix | 1348 |
| `b8c038647` | #498 Lane B M3 (A2A projection) | 878 |

**Observed shape**: ~200–360 changed lines per milestone in this subsystem; single-milestone feature
landings up to ~1350. This sprint's estimate (**1,250 lines across 4 milestones — M1 580, M2 210,
M3 280, M4 180 — ~313/milestone**) sits inside the observed band. M1 is deliberately the outlier: it
carries the entire new mechanism plus the four security-regression test suites.

**Decision on the doc's 3-day / 4-milestone shape: KEEP UNCHANGED.** There is no measured reason to
change it. The one thing that could have justified growth — D-6's `go/packages` AST analyzer — was
**ruled out of scope by human ruling** (Mark, attended 2026-08-06) and filed as
[#612](https://github.com/sunholo-data/ailang/issues/612). Planning it in would reverse a human
decision and is explicitly forbidden.

---

## 4. Milestones

### M1 — Proxy-aware Net transport and SSRF regression tests

**Closes**: design-doc M1 AC1–AC4 (doc lines 249–256)
**Estimate**: 1 day · ~250 production LOC + ~330 test LOC
**Files**: `internal/effects/net.go`, new package-private mechanism file in `internal/effects/`,
`internal/effects/*_test.go`

#### Tasks

1. Add a package-private request-aware `RoundTripper` in `internal/effects` owning **separate**
   direct and proxy transport creation paths. It must not mutate one shared transport between modes.
   Per request it calls `http.ProxyFromEnvironment(req)`:
   - **no proxy** → call `resolveAndValidateIP` exactly once, hand the returned IP to a direct
     transport whose dialer connects to that IP with no hostname re-resolution;
   - **proxy selected** → ordinary proxy dialing, **zero** local target resolution/IP validation. The
     proxy address must never enter the target-IP substitution closure.
2. Route all three Net client constructors (`net.go:96`, `:212`, `:587`) through it.
3. Remove the three preflight `resolveAndValidateIP` calls (`net.go:85`, `:201`, `:565`) and the
   redirect-validator call (`:317`). `validateRedirect` keeps redirect-count and protocol checks and
   the caller's cross-origin `Authorization` stripping; it no longer resolves.
4. Preserve public error categories via a **typed internal target-validation error** unwrapped through
   `*url.Error`. **The two `makeResultErr("Transport", …)` sites this must update are `net.go:567`**
   (the preflight path being moved) **and `net.go:631`** (post-`client.Do`, where a proxy-route failure
   now arrives `url.Error`-wrapped). Both re-verified first-party at `945f36727`.
5. Write the four named tests. Each must use a target whose DNS answer / dial endpoint **differs from
   the fake proxy**, with injected resolver-call and dial-call counters.

#### Acceptance criteria

**AC-M1.1** (doc M1 AC1) — proxy boundary observed by production constructors
```bash
go test -count=1 ./internal/effects \
  -run 'TestNetProxyBoundary|TestNetProxyDirectPin|TestNetProxyNoProxy|TestNetProxyRedirectControls' -v
```
- **Base result at `945f36727`: rc=0, `=== RUN` count = 0, `[no tests to run]` — VACUOUSLY GREEN.**
- **Pass requires all three**: (a) rc=0; (b) `grep -c '^=== RUN' ` ≥ **4**; (c) a `--- PASS` line for
  each of the four named top-level tests. Exit code alone is **not** acceptance.
- Tests assert the **observed dial/CONNECT destination**, not merely that a request succeeded.
- **Silent-revert check**: reverting production routing makes the fake proxy observe no request, or
  makes the old pinning closure dial the target instead of the proxy — either fails. Deleting the
  tests drops the `=== RUN` count to 0 and fails (b).
- **UNINFORMATIVE UNDER SANDBOX** if the output contains `bind: operation not permitted`.

**AC-M1.2** (doc M1 AC2) — direct/`NO_PROXY` pinning survives
- A direct/`NO_PROXY` subtest asserts the accepted connection reaches **the injected validated IP**,
  not the request hostname's alternate endpoint.
- **Base**: no such test exists — nothing to baseline; the gate is new by construction.
- **Silent-revert check**: replacing both paths with ordinary proxy/default dialing fails the endpoint
  assertion. Removing `NO_PROXY` handling routes the request to the fake proxy and fails.

**AC-M1.3** (doc M1 AC3) — policy controls hold in **both** route modes, and proxy routes do no DNS
- Capability denial, initial-domain rejection, redirect-protocol rejection, and redirect-count
  rejection each run **with a proxy selected**.
- The injected **resolver counter stays exactly 0** for proxy-selected initial *and* redirect requests.
- **Silent-revert check**: restoring preflight/redirect resolution increments the counter → red;
  deleting proxy support makes the proxy-positive arm fail → red.

**AC-M1.4** (doc M1 AC4) — direct-route failure shape is byte-stable
- Direct-route DNS/IP rejection tests assert **exactly 1 resolver call**, **exactly 0 dial calls**, and
  the existing category: legacy `E_NET_DNS_FAILED` / `E_NET_IP_BLOCKED`, or structured
  `Err(Transport, …)` as applicable.
- Source of truth for the categories (V20, re-verified at `945f36727`): `E_NET_IP_BLOCKED` at
  `internal/effects/net_security.go:27,34,46,51,56`; `E_NET_DNS_FAILED` at `:90,94`.
- **Silent-revert check**: double resolution fails the call count; ordinary dialing fails the zero-dial
  assertion; losing the `url.Error` unwrapping fails the exact-category assertion.

#### Milestone boundary (bisectability)

```bash
go build ./internal/effects/... ./internal/executor/... ./internal/testutil/...   # base rc=0
go test -count=1 ./internal/effects                                              # base rc=0, 11.9s
go vet ./internal/effects                                                        # base rc=0
```
All three were green at base and **must remain green**. Commit: `feat(effects): M1 — request-aware
proxy/direct Net transport with target-IP pinning on the direct route`.

#### Risks

| Risk | Mitigation |
|---|---|
| Pin closure rewrites the **proxy** address (doc's top risk) | Separate transports per mode; AC-M1.1 asserts the observed dial endpoint |
| Transport caching bleeds a pin across requests | Deferred decision: cache keys **must** include every security-relevant route/target property; add an explicit cross-request no-bleed test |
| Error-category drift | AC-M1.4 pins both mappings at the two enumerated call sites |

---

### M2 — Stream and Managed Agents constructors

**Closes**: design-doc M2 AC1–AC2 (doc lines 266–267)
**Estimate**: 0.5 day · ~30 production LOC + ~180 test LOC
**Files**: `internal/effects/stream_sse.go`, `internal/effects/stream_ndjson.go`,
`internal/executor/managed_agents/client.go`, plus tests in each package

#### Tasks

1. Add `Proxy: http.ProxyFromEnvironment` to the four remaining production transports:
   `stream_sse.go:70` (SSE GET), `stream_sse.go:329` (SSE POST), `stream_ndjson.go:80` (NDJSON POST),
   `managed_agents/client.go:141` (Vertex SSE).
2. **Change no timeout field.** Preserve connect, TLS-handshake, response-header, idle-connection, and
   overall-stream bounds exactly (doc V6). Managed Agents in particular keeps **zero global request
   timeout**.
3. Constructor-level fake-proxy tests for all four paths — tests must invoke the **production
   constructors**, never a standalone `http.Transport` literal (doc D-5).

#### Acceptance criteria

**AC-M2.1** (doc M2 AC1) — stream constructors observed at the fake proxy
```bash
go test -count=1 ./internal/effects -run 'TestStream(SSE|NDJSON).*ProxyBoundary' -v
```
- **Base result at `945f36727`: rc=0, `=== RUN` count = 0, `[no tests to run]` — VACUOUSLY GREEN.**
- **Pass requires**: rc=0 **and** `=== RUN` count ≥ 3 **and** the fake proxy observes all three of
  **GET**, **SSE POST**, and **NDJSON POST**.
- **Silent-revert check**: a reverted constructor bypasses the fake proxy and fails its observation
  assertion; deleting the tests fails the `=== RUN` count.
- **UNINFORMATIVE UNDER SANDBOX** on `bind: operation not permitted`.

**AC-M2.2** (doc M2 AC2) — Managed Agents production default client
```bash
go test -count=1 ./internal/executor/managed_agents -run TestDefaultHTTPClientProxyBoundary -v
```
- **Base result at `945f36727`: rc=0, `[no tests to run]` — VACUOUSLY GREEN.**
- **Pass requires**: rc=0 **and** ≥1 `=== RUN TestDefaultHTTPClientProxyBoundary` **and** a `--- PASS`
  for it. The test must prove `defaultHTTPClient()` selects the fake proxy **while retaining zero
  global request timeout** and its existing header/idle timeouts.
- The doc states explicitly: **"a generic package pass is insufficient."** Honoured by the `=== RUN`
  requirement.
- **Silent-revert check**: reverting the `Proxy` field makes selection/observation fail; adding a
  global timeout fails the retention assertion.

#### Milestone boundary

```bash
go build ./internal/effects/... ./internal/executor/...        # base rc=0
go test -count=1 ./internal/effects ./internal/executor/managed_agents   # base rc=0
```
Commit: `feat(effects,managed_agents): M2 — ProxyFromEnvironment on the four remaining production
transports`.

#### Risks

| Risk | Mitigation |
|---|---|
| A timeout field is silently altered while adding `Proxy` | AC-M2.2 asserts timeout retention; reviewer diff should be a one-line field add per site |
| Managed Agents test reaches for public egress | Fake proxy + stub only; existing live probes stay opt-in (doc V10) |

---

### M3 — Streaming behaviour and poisoned-lane integration

**Closes**: design-doc M3 AC1–AC3 (doc lines 277–279)
**Estimate**: 0.75 day · ~280 test LOC (no production LOC expected)

#### Tasks

1. Implement the D-4 local forward-proxy experiment: cover **both** HTTP absolute-form **and** HTTPS
   `CONNECT` (test CA or an intentionally minimal tunnel); flush ≥2 events with a measured
   inter-event synchronisation; assert event 1 is observable **before** event 2 is released. A proxy
   that buffers until EOF **must** fail. Hold a stream open across a keep-alive interval and verify
   cancellation / idle-timeout cleanup.
2. Run all local-server transport suites under the poisoned environment, outside the sandbox.
3. Confirm both live public Net subtests remain skipped with `AILANG_LIVE_NET` unset.

#### Acceptance criteria

**AC-M3.1** (doc M3 AC1) — streaming is not buffered, and cleans up
- The streaming fake-proxy test observes **event 1 before permitting event 2**, then closes cleanly
  after cancellation / idle timeout.
- **Base**: no such test exists — new by construction.
- **Silent-revert check**: direct routing produces **zero** proxy observations; a buffering proxy
  produces no early event; broken cleanup exceeds the test deadline.
- **UNINFORMATIVE UNDER SANDBOX** on `bind: operation not permitted`.

**AC-M3.2** (doc M3 AC2) — the poisoned lane, with the vacuity hole closed
```bash
HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 \
  go test -count=1 ./internal/effects ./internal/executor/managed_agents
```
- **Base result at `945f36727`: rc=0 (`ok effects 13.590s`, `ok managed_agents 0.250s`) —
  ALREADY GREEN AT BASE.** The bare package command therefore proves **nothing** on its own.
- **The doc says so and this plan enforces it.** Pass requires additionally running with `-v` and
  showing `=== RUN` / `--- PASS` lines for **the named M1 and M2 proxy-boundary tests** and for the
  local `httptest` tests, plus the M4 source audit.
- **`AILANG_LIVE_NET` must be UNSET for this run.** `internal/testutil/gate.go:47–58`
  (`LiveNetworkStatus`) returns `LiveNetworkFatal` when a proxy variable points at `127.0.0.1:9` in
  the live lane, and `RequiresLiveNetwork` then `t.Fatalf`s. Running poisoned **and** live is a
  configuration error, not a code failure.
- **Silent-revert check**: explicitly acknowledged as insufficient alone — a revert can leave the bare
  package command green, which is exactly what the base measurement shows. The named-test lines and
  AC-M4.1 carry the falsifiability.

**AC-M3.3** (doc M3 AC3) — live gating unchanged (supporting evidence only)
```bash
env -u AILANG_LIVE_NET go test -count=1 -v ./internal/effects \
  -run 'TestNetHttpPost/httpPost_to_httpbin.org|TestNetBodySizeLimit/small_response_under_limit'
```
- **Base result at `945f36727`: rc=0, both named subtests report `--- SKIP` — ALREADY GREEN AT BASE.**
- This is a **non-regression** check: it must still report both as `SKIP`. The doc is explicit that it
  "checks gating, not proxy support, and is therefore supporting evidence only; it cannot satisfy
  M1/M2 by itself." Recorded as such.
- **Do not** add `NO_PROXY=*` or unset proxy variables to make default tests green (doc D-5) — that
  would evade the behaviour under assertion.

#### Milestone boundary

```bash
go test -count=1 ./internal/effects ./internal/executor/managed_agents             # base rc=0
HTTP_PROXY=… HTTPS_PROXY=… go test -count=1 ./internal/effects ./internal/executor/managed_agents  # base rc=0
```
Commit: `test(effects): M3 — forward-proxy streaming experiment and poisoned-lane integration`.

#### Risks

| Risk | Mitigation |
|---|---|
| CONNECT-proxy fixture complexity is the doc's stated 2–4 day swing driver | Minimal tunnel accepted as an alternative to a test CA (doc D-4 permits either); timebox and escalate rather than expanding scope |
| Go caches proxy configuration process-wide | Never mutate/unset poison after a proxy use in-process (doc V11 comment); use subprocess tests where environment identity matters |

---

### M4 — Retire the Option-A residual, source audit, and documentation

**Closes**: design-doc M4 AC1–AC3 (doc lines 303–311)
**Estimate**: 0.75 day · ~120 audit LOC + ~60 deletions + docs

#### Tasks

1. Delete `effects_nil_proxy_remains_open` (`internal/testutil/egress_posture_test.go:20`) and
   `testEffectsProxyResidual` (`:66`–`:85`) and any helper that becomes residual-only. **Retain**
   `poison_sentinel_denies_HTTP_egress`, `loopback_bypasses_lane_poison`, `raw_TCP_remains_open`,
   `requireLiveEgressPosture`, `laneIsPoisoned`, and `assertPoisonProxyError`.
2. Build the source audit with a **checked-in positive fixture**. `internal/testutil/gatelint/`
   already has the shape: `Scan(root string) []Violation` plus
   `testdata/fixtures/{cmd,internal,runtime,std,tests}/…`, including a production-scope
   `runtime/production.go.fixture`. Extending gatelint or placing the audit beside the affected
   package are both permitted (doc Deferred Decisions); **the fixture and all seven sites are
   mandatory** either way.
3. Remove the Option-A residual / Non-Goals text from
   `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md:394` and replace with a short historical
   note linking this item.
4. Replace the seven-transport residual paragraph in
   `docs/docs/guides/development-workflow.md:322–332` with: the new proxy behaviour, `NO_PROXY`
   direct-pin behaviour, and the **proxy-mode pinning limitation**.
5. Add the v0.33.1 changelog entry to `changelogs/v0.18-current.md` (the `## [v0.33.1] - 2026-08-06`
   section already exists) and update `docs/docs/reference/effects.md` with the proxy and direct-pin
   security semantics.
6. **Link `#612` from the M4 gate's source comment** so the next reader of the textual scanner finds
   the durable replacement rather than re-deriving the objection. (#612 is already filed — that filing
   is part of D-6 Option A's definition of done and is **already complete**.)
7. **Supplementary, non-blocking** (see §5, correction 4): annotate
   `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix-sprint-plan.md:446,466` — those two
   references survive the doc's AC-M4.2 grep scope by design (they are the historical record of a
   completed sprint), but line 460's AC10(d) falsification-drill row describes a drill that
   **cannot execute**. One superseded-by note is enough. **Do not** widen AC-M4.2's grep to cover them;
   the doc's scope wins.

#### Acceptance criteria

**AC-M4.1** (doc M4 AC1) — completeness audit with a proven detector
- A repository audit test/command reports **zero** production `http.Transport` literals with a nil
  `Proxy` in the audited first-party scope, **and** proves the detector on a checked-in positive
  fixture.
- **Base result at `945f36727`: 7 nil-`Proxy` literals across 4 files** — `net.go:96,212,587`,
  `stream_ndjson.go:80`, `stream_sse.go:70,329`, `managed_agents/client.go:141`. Production
  `ProxyFromEnvironment` count = **0**; test-inclusive control = **3**. Both re-derived first-party at
  this HEAD, matching V2/V3 exactly.
- **Silent-revert check**: reverting **any one** of the seven sites adds a finding → red. Deleting or
  blinding the scanner makes the **positive fixture** fail → red. This is the gate that carries M3's
  falsifiability, per AC-M3.2.
- **Scope note for the executor**: the audit is a **textual/source** gate by human ruling
  **D-6 = Option A** (Mark, attended 2026-08-06). The `go/packages` AST/type analyzer is
  [#612](https://github.com/sunholo-data/ailang/issues/612) and is **explicitly out of this sprint**.
  Implementing it here would reverse a human decision.

**AC-M4.2** (doc M4 AC2) — tripwire retired, with an anti-vacuity control
```bash
grep -rn 'effects_nil_proxy_remains_open' internal/testutil docs \
  design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md   # must yield 0 hits
grep -rn 'loopback_bypasses_lane_poison' internal/testutil      # must yield >0 hits (control)
```
- **Base result at `945f36727`**: first command **4 hits** (`egress_posture_test.go:20,67`;
  `development-workflow.md:330`; `m-ci-flake-systemic-fix.md:394`) — **correctly RED at base**.
  Second command **2 hits** (`egress_posture_test.go:18,38`) — **control FIRES at base**.
- **Silent-revert check**: restoring the tripwire creates a hit → red. The control prevents an
  empty/mistyped search from passing silently.

**AC-M4.3** (doc M4 AC3) — quality gates, explicitly NON-SUFFICIENT
```bash
go vet ./internal/effects ./internal/executor/managed_agents ./internal/testutil   # base rc=0
make check-boundaries                                                             # base rc=0, "OK"
```
- **Base result at `945f36727`: both rc=0 — ALREADY GREEN AT BASE.**
- The doc states these "would still pass after a behavioural revert, so they are explicitly
  non-sufficient and must be combined with the M1–M4 behavioural/source-audit gates." **Agreed and
  recorded.** They are regression tripwires only.
- `make check-boundaries` **is** load-bearing for one thing: it must stay green while the new
  mechanism lives inside `internal/effects` and Managed Agents gets its own local field. See §4.3.

#### Milestone boundary

```bash
go build ./internal/effects/... ./internal/executor/... ./internal/testutil/...   # base rc=0
go test -count=1 ./internal/effects ./internal/executor/managed_agents ./internal/testutil  # base rc=0
go vet ./internal/effects ./internal/executor/managed_agents ./internal/testutil  # base rc=0
make check-boundaries                                                            # base rc=0
```
Commit: `refactor(testutil,docs): M4 — retire the Option-A residual tripwire, add the nil-Proxy source
audit, document the proxy boundary`.

---

### 4.3 Hard architectural guard for M4 (measured, not assumed)

`internal/effects/net_test.go` is `package effects` and **imports
`github.com/sunholo-data/ailang/internal/testutil`** (`net_test.go:11`). Therefore
**`internal/testutil` must NOT import `internal/effects`** — that is an import cycle in the effects
test binary, and it will not compile.

**Consequence**: the "positive boundary assertions" replacing the retired tripwire must live in
`internal/effects/*_test.go` and `internal/executor/managed_agents/*_test.go` — which is exactly where
the design doc places them (doc lines 327–329). Do **not** attempt to move them into
`internal/testutil`. This also matches `ARCHITECTURE.md`: `internal/effects` is core/runtime,
`internal/executor` is an executor-integration package, and the doc's rule that the new mechanism
stays inside `internal/effects` while Managed Agents receives its own localized standard proxy field.

---

## 5. Corrections found in the design doc and in the controller's briefing

**This mission treats a planner refuting the controller as the loop WORKING.** Stating these plainly.

### Correction 1 — REFUTES THE CONTROLLER. The AC10(d) tripwire cannot flip RED from this change.

The briefing states: *"This item's landing must flip that tripwire RED and retire it."*

**REFUTED, first-party at `945f36727`.** `testEffectsProxyResidual`
(`internal/testutil/egress_posture_test.go:66–85`) constructs its **own** bare transport in the test
file:

```go
effectsTransport := &http.Transport{}
response, err := (&http.Client{Transport: effectsTransport, Timeout: 10 * time.Second}).Get("https://example.com")
```

The file's imports are `net`, `net/http`, `net/http/httptest`, `net/url`, `os`, `strings`, `testing`,
`time` — **nothing from `internal/effects`**. `grep -n 'effects'` on the file returns only the subtest
*name* (`:20`, `:67`), a *comment* (`:72`), and the local variable name (`:74`). Control: `net/http`
appears 2× in the same file, so the search instrument is live.

**No change to `internal/effects/net.go` can alter this test's outcome.** It is a hand-rolled
*simulation* of the effects transport, not an observation of it.

**Impact on the plan**: none of the ACs depend on observing the tripwire go red. M4 is a **deliberate
deletion** justified by the doc's own reasoning ("success would now assert the bug", doc line 519),
gated by AC-M4.2's grep-to-zero plus a firing control. **The design doc is right and the briefing's
framing is wrong** — and notably the doc already hedges correctly, calling it
"**helper-only** residual logic" (line 283). The doc wins, as rule vii requires.

### Correction 2 — inherited defect in an already-implemented doc

`design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix-sprint-plan.md:460` lists the AC10(d)
falsification drill as: *"set the effects transport proxy to `http.ProxyFromEnvironment` | `git diff`
on the effects client constructor; the residual half turns red."* **That drill is not executable**,
for the reason in Correction 1 — the residual half never reads the effects client constructor. This
is a defect in a *landed* artifact, not in this design doc. Handled as M4 task 7 (one superseded-by
note), **not** by widening any AC.

### Correction 3 — CONFIRMS the controller. `go build ./...` is red at base.

The briefing handed this over **unverified** and asked for verification. **VERIFIED: rc=1 at
`945f36727`.** Both failures are `function main is undeclared in the main package` — `cmd/wasm`
(build-tagged `//go:build js && wasm`, so no native `main`) and `gen/main`. This is structural and
permanent under native `GOOS`. **`go build ./...` is excluded from every gate in this plan** and
replaced by the scoped, base-green
`go build ./internal/effects/... ./internal/executor/... ./internal/testutil/...`.

### Correction 4 — the AC-M4.2 grep scope has a known, accepted hole

Repo-wide, `effects_nil_proxy_remains_open` appears **13** times. The doc's AC-M4.2 grep covers
`internal/testutil`, `docs`, and `m-ci-flake-systemic-fix.md` — **4** of those hits. Two more live in
`m-ci-flake-systemic-fix-sprint-plan.md:446,466`, which the grep does **not** cover; the rest are in
this design doc, the archived status log, and the mission log.

**Assessment: this is correct as written, not a bug.** The implemented *sprint plan* and the mission
logs are historical records of what was true at the time; rewriting them would falsify the record. The
forward-looking guidance — the `m-ci-flake-systemic-fix.md` Non-Goals text and the
development-workflow guide — is what must not survive, and that is exactly what the doc's scope
covers. **The doc's scope is adopted unchanged.** The hole is documented here so a future reader does
not mistake it for an oversight.

### Correction 5 — this workspace contradicts V13's sandbox denial

Doc row **V13** records socket-bearing tests as **UNINFORMATIVE UNDER SANDBOX**
(`httptest: failed to listen … bind: operation not permitted`). **In this planning workspace at
`945f36727`, they were fully informative and green** — `./internal/effects` `ok 11.866s` unpoisoned,
`ok 13.590s` poisoned, `./internal/testutil` `ok 2.342s`, and `TestEgressPosture` ran all four
subtests.

**This does not retire the sandbox rule.** The executor may run under a stricter policy. It does mean:
a denial is now *distinguishable from a failure*, because §2.3 records what the informative output
looks like. **Any gate emitting `bind: operation not permitted` or proxy `operation not permitted` is
UNINFORMATIVE UNDER SANDBOX and must be re-run by the controller outside the sandbox — never reported
as pass or fail.**

### Non-correction: the design doc is internally consistent on the points checked

Explicitly checked and found **consistent** — no disagreement to escalate under rule vii:
- 7 sites = 6 effect transports + 1 Managed Agents; M2's "four paths" = 3 stream + 1 MA. Adds up.
- Timeline 1 + 0.5 + 0.75 + 0.75 = 3.0 days, matching the header.
- 12 acceptance criteria, all owned by exactly one milestone.
- The doc's own AC-M3.2 already warns that a bare package PASS is vacuous — which the base measurement
  independently confirms. The doc anticipated the exact failure mode rule 3e is designed to catch.

---

## 6. Sandbox policy for this sprint

Per mission constraint 6 and the doc's Testing Strategy:

- Any `bind: operation not permitted`, proxy `operation not permitted`, or outbound network denial is
  **UNINFORMATIVE UNDER SANDBOX**.
- Such a result is **never** reported as pass or fail. It is reported as *uninformative*, and the
  controller re-runs the gate outside the sandbox.
- Socket-bearing gates in this plan: **AC-M1.1, AC-M1.2, AC-M2.1, AC-M2.2, AC-M3.1, AC-M3.2**.
- Non-socket gates, always informative: **AC-M4.1, AC-M4.2, AC-M4.3**, and every `go build` / `go vet`
  / `make check-boundaries` milestone boundary.
- **Do not rewrite tests to work around sandbox policy** (doc Risks table). Label and escalate.

---

## 7. Security-regression posture (constraint 5)

This sprint knowingly changes an SSRF control (**D-1**). The guard against silently widening the
**direct**-route boundary is that every one of these is asserted:

| Invariant | Where asserted | How it reds |
|---|---|---|
| Direct route resolves **exactly once**, then dials **that IP** | AC-M1.2, AC-M1.4 | double resolution fails the counter; hostname dialing fails the endpoint assertion |
| `NO_PROXY` route behaves as a direct route (pinned) | AC-M1.2 | request appearing at the fake proxy fails |
| Proxy route performs **zero** local target DNS | AC-M1.3 | resolver counter > 0 fails |
| Proxy address never enters the target-IP substitution closure | AC-M1.1 | proxy observes no request / wrong dial endpoint fails |
| `Net` capability still gates the operation, in **both** modes | AC-M1.3 | capability-denial arm with proxy selected fails |
| Domain allowlist + protocol + redirect count/protocol hold in **both** modes | AC-M1.3 | each rejection arm fails |
| Public error categories unchanged | AC-M1.4 | exact-category assertion fails |
| No new nil-`Proxy` production transport can be added unnoticed | AC-M4.1 | scanner finds it; fixture proves the scanner is not blind |

**What is knowingly traded, and only this**: on a **proxy-selected** request, the guarantee that the
socket reaches AILANG's pre-validated **target IP**. Direct and `NO_PROXY` requests keep it. The doc
never claims equivalence, and neither does this plan. Per the doc's Quorum Verification Log, **D-1
remains an open ratification ask to the human** — it is carried on the bookkeeping issue and is
explicitly **non-blocking** for this sprint.

---

## 8. Out of scope — do not implement

- **The `go/packages` AST/type analyzer.** Human ruling **D-6 = Option A** (Mark, attended
  2026-08-06). Filed as [#612](https://github.com/sunholo-data/ailang/issues/612). Building it here
  reverses a human decision.
- WebSocket, raw TCP, SSH, browser/WASM fetch, or subprocess proxy enforcement.
- Applying the target domain allowlist to the proxy hostname.
- Preserving target-IP pinning *through* a proxy.
- Proxy URL/auth CLI flags, PAC support, certificate installation, or a `ctx.Net` routing flag.
- Any `docs/LIMITATIONS.md` entry (doc D-3: this is intentional behaviour, not a language limitation).
- Refactoring HTTP clients outside the V2 seven-site set — in particular the **29** inline
  `http.Client{}` sites, which already inherit proxy-aware `http.DefaultTransport` (doc V18).

---

## 9. Success metrics

- All **12** design-doc acceptance criteria pass outside the sandbox, each with its `=== RUN` /
  hit-count evidence recorded — not just an exit code.
- **7 → 0** nil-`Proxy` production transports, proven by a detector that itself passes a positive
  fixture.
- Base-green gates (`go vet`, `make check-boundaries`, unpoisoned and poisoned package tests, the
  live-skip check) all **still** green — no regression.
- Four bisectable commits, one per milestone, each with the relevant package test passing at the
  boundary.
- Changelog + `docs/docs/reference/effects.md` + `docs/docs/guides/development-workflow.md` describe
  the compatibility and security change; **no** contradictory "residual open" text survives in
  forward-looking guidance.

---

## 10. Open questions for the executor

1. **Transport caching** (doc Deferred Decision): if direct transports are cached per host, cache keys
   must include every security-relevant route/target property, and a cross-request pin-bleed test is
   mandatory. Simplest safe answer: do not cache in M1; revisit only if a measured cost appears.
2. **Fake CONNECT proxy shape** (doc Deferred Decision): test CA vs. minimal tunnel. This is the
   doc's stated 2–4 day swing driver. Prefer the minimal tunnel; escalate rather than expand scope.
3. **Audit placement** (doc Deferred Decision): extend `internal/testutil/gatelint` (which already has
   `Scan(root)` and a `testdata/fixtures/runtime/production.go.fixture` production-scope pattern) or
   place the audit beside the affected package. Either is permitted; the **fixture and all seven
   sites are mandatory**.
