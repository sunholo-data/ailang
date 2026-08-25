# M-REMOTE-BROWSER-SESSION-PROVIDERS — provider-neutral browser sessions for local and cloud agents

**Status**: Implemented (production activation gates pending)
**Target**: v0.33.3
**Priority**: P1 (Medium-High — closes the browser vessel gap in agent-mode evals)
**Estimated**: 6–8 days including a local/Browserbase comparison run
**Dependencies**: Existing executor/eval harness and OTEL chain telemetry; pinned `@playwright/mcp`; Browserbase credentials only for the cloud activation milestone. M-NET-EFFECT-PROXY-BOUNDARY informs egress policy but does not block a loopback-only local spike.

## Summary

Add a provider-neutral browser-session plane to AILANG's agent eval harness. The same AI executor and the same pinned Playwright MCP tool surface can run against:

- **`local-playwright`** — the default local vessel, launching an isolated Playwright Chromium process per session on the worker; and
- **`browserbase`** — the first cloud vessel, provisioning a remote browser and connecting the same MCP server to its CDP endpoint.

The browser is infrastructure, not the agent. OpenAI computer use, Gemini Managed Agents/Antigravity, Anthropic computer use, Browser Use, Stagehand, and similar bundled loops remain separately named **managed-agent vessels**. Their results must not be pooled with provider-neutral rows because their prompting, observation, action, and recovery policies are part of the measured system.

This design deliberately starts with two providers and a narrow lifecycle contract. AWS AgentCore Browser, Steel, Browserless, Cloudflare Browser Run, and full-desktop sandboxes can be added after the contract is proven without changing executor-facing browser tools.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pins browser/MCP versions and records viewport, locale, browser build, provider, and initial profile hash; live websites remain explicitly external nondeterminism. |
| A2: Replayability | +1 | Banks Playwright trace, screenshots, action transcript, HAR/console artifacts where available, and provider recording references. |
| A3: Effect Legibility | +1 | Browser activity is a named tool/effect lane rather than hidden shell or ambient desktop access. |
| A4: Explicit Authority | +1 | Session specs declare origin policy, permissions, downloads, profile, duration, and human-takeover authority. |
| A5: Bounded Verification | +1 | Every session has hard creation, idle, action, and wall-clock bounds plus structured termination. |
| A6: Safe Concurrency | +1 | One browser process/remote allocation and one disposable profile per eval session; concurrency is explicitly capped. |
| A7: Machines First | +1 | A stable MCP surface and JSON session manifest are machine-readable across executors and providers. |
| A8: Minimal Syntax | 0 | No AILANG language syntax is added. |
| A9: Cost Visibility | +1 | Browser duration, provider usage, and cost are banked separately from model cost. |
| A10: Composability | +1 | One browser contract composes with local and cloud executors without embedding model-provider policy. |
| A11: Structured Failure | +1 | Provisioning, connection, policy, timeout, disconnect, and artifact failures use stable categories. |
| A12: System Boundary | +1 | Browser process, remote provider, credentials, profiles, and artifacts are explicit boundary objects. |

**Net Score: +11** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): browser/environment metadata and externally nondeterministic inputs are explicit
- [x] A3 (Effects): browser access is exposed only through the named MCP/tool lane
- [x] A4 (Authority): no operator Chrome profile, public CDP endpoint, or ambient credential access
- [x] A7 (Machines First): lifecycle, failures, artifacts, and usage are structured

## Problem Statement

AILANG can run standard and agent-mode evals locally and in Cloud Run, but it has no reusable browser-session vessel. The recent managed-Gemini design correctly distinguishes product-fidelity managed agents from neutral model ranking, yet a browser-capable eval currently has only vendor-bundled options or ad-hoc browser setup.

**Current State:**

- `internal/executor/` has a common agent executor contract and local/cloud deployment shape, but no browser-session provider or browser artifact contract (V1, V2).
- Managed Agents can run Google's Antigravity sandbox, but that measures model + Google scaffold and cannot serve as a provider-neutral browser baseline.
- The repository uses Playwright for browser CI/demo testing, not as a session service exposed consistently to agent executors.
- Local execution has no pinned per-session profile/process lifecycle; cloud execution has no remote-browser provisioning adapter.
- Browser costs, browser build, action transcript, recordings, and browser-specific termination cause are not banked as first-class eval metadata.
- M-NET-EFFECT-PROXY-BOUNDARY explicitly leaves browser/WebSocket egress out of scope, so a browser lane must not imply that the existing Net proxy policy already contains it (V7).

**Impact:**

- AILANG cannot fairly compare OpenAI, Gemini, Anthropic, OpenRouter, or local models on browser tasks while holding the browser harness constant.
- Local browser evals and Cloud Run browser evals would produce different tool surfaces unless one contract is established first.
- Vendor-managed agent results risk being misread as pure model results.
- Debugging browser failures lacks the replay artifacts and session linkage already expected from AILANG chains.

## Goals

**Primary Goal:** Run the same browser task through the same pinned MCP action surface on a local isolated Chromium session and a Browserbase remote session, with complete lifecycle, provenance, artifacts, and cost attribution.

**Success Metrics:**

- One fixture task passes through `local-playwright` and `browserbase` without changing the benchmark prompt or browser tool names.
- 100% of browser runs bank provider, browser version, MCP version, viewport, locale/timezone, session duration, termination category, and artifact manifest.
- Local sessions use separate browser processes and disposable user-data directories; a two-session isolation test proves cookies/local storage do not cross.
- Browser model cost and browser infrastructure cost are reported as separate fields; unknown provider pricing remains unknown, never zero-filled.
- A 50-session local smoke has no orphaned browser processes or retained disposable profiles; a 20-session cloud smoke has no leaked active sessions.
- Managed-agent/browser-scaffold rows are labeled separately and cannot be pooled with neutral-browser rows by default.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Browser infrastructure is separate from the AI executor/model | Prevents vendor scaffolding from becoming an untracked eval confound | human via design approval | design | high |
| One narrow `SessionProvider` lifecycle underlies all browser backends | Provider APIs differ; leaking them upward would create per-executor integrations | human via design approval | design | high |
| Pinned Playwright MCP is the executor-facing tool plane | MCP is supported across the relevant agent harnesses and can target local or remote browsers; changing the tool schema later invalidates comparisons | human via design approval | design | high |
| `local-playwright` uses a fresh browser process and profile per session | Browser contexts alone are not the intended isolation boundary for eval sessions | human via design approval | design | high |
| Browserbase is the first cloud adapter; it is not hard-coded as the permanent provider | Fastest GCP/Cloud Run fit while retaining a Steel/AgentCore/Browserless path | human via design approval | design | med |
| Raw Playwright/MCP is the neutral baseline; Stagehand/Browser Use/computer-use loops are named vessels | Keeps agent-policy improvements measurable instead of silently changing the harness | human via design approval | design | high |
| CDP and live-view endpoints are credentials and never enter prompts, normal logs, or public telemetry | Leakage grants session control | human via design approval | compile | high |

### Design Freeze

- [x] Separate browser provider from model/executor and managed-agent vessel identity.
- [x] Freeze the v1 provider lifecycle to create, connect, inspect, export, and stop.
- [x] Use a pinned Playwright MCP tool schema for the first neutral baseline.
- [x] Use fresh process + disposable profile isolation for local sessions.
- [x] Implement local Playwright first and Browserbase second.
- [x] Treat endpoint URLs/headers and profile secrets as sensitive values with redaction tests.
- [x] Do not claim the existing Net-effect proxy boundary contains browser/CDP/WebSocket traffic.

## Solution Design

### Overview

Introduce a browser-session package below the eval harness and beside, not inside, individual model executors. It owns browser allocation, provenance, secret endpoint handling, artifact export, usage, and teardown. The executor receives only a generated MCP configuration whose process connects to that session.

For local runs, the pinned Playwright MCP process launches pinned Chromium in isolated mode with a session-owned user-data directory. For cloud runs, the Browserbase adapter creates a session and supplies its sensitive CDP endpoint to the same MCP process. The benchmark prompt and exposed MCP tool names remain unchanged.

The controller owns teardown. Agent-requested `browser_close`, executor cancellation, timeout, panic, and process exit all converge on idempotent `Stop`. Cleanup failure is banked and retried; it is never silently treated as success.

### Architecture

```text
eval/coordinator run
  │
  ├─ model executor: codex | claude | pi | opencode | motoko | ...
  │    └─ generated MCP config (no endpoint in prompt)
  │          └─ pinned @playwright/mcp
  │                └─ local Playwright browser OR remote CDP session
  │
  └─ browser session controller
       ├─ SessionProvider: local-playwright | browserbase
       ├─ policy + timeout + concurrency admission
       ├─ OTEL spans and structured action/session events
       ├─ artifact manifest: trace/video/HAR/screenshots/console/recording ref
       └─ idempotent stop + leak audit
```

**Components:**

1. **`SessionProvider`**: Provider-neutral lifecycle. It returns opaque handles and sensitive connection material separately from safe metadata.
2. **`SessionController`**: Applies defaults and policy, owns timeout/cancellation, starts MCP, records spans, exports artifacts, and guarantees teardown.
3. **`LocalPlaywrightProvider`**: Allocates session directories and drives pinned Playwright MCP isolated mode. It never uses the operator's real Chrome profile.
4. **`BrowserbaseProvider`**: Creates/stops remote sessions and returns CDP/live-inspection data through redacted wrapper types. API interaction is stub-testable.
5. **MCP config injector**: Supplies executor-specific temporary MCP configuration without adding browser endpoints to prompts or durable generic config.
6. **Browser result manifest**: Banks identity, versions, timings, action counts, termination, usage/cost, and content-addressed artifact references alongside the agent result.

### Provider Contract

The exact Go names may vary, but the contract semantics are frozen:

```go
type SessionProvider interface {
    Create(ctx context.Context, spec SessionSpec) (Session, error)
    Connection(ctx context.Context, session Session) (SensitiveConnection, error)
    Inspect(ctx context.Context, session Session) (InspectionRef, error)
    Export(ctx context.Context, session Session, dst string) (ArtifactManifest, error)
    Stop(ctx context.Context, session Session) (Usage, error)
}
```

`SessionSpec` includes:

- provider and pinned browser/MCP versions;
- viewport, locale, timezone, color scheme, and user agent policy;
- maximum duration, idle timeout, per-action timeout, and concurrency class;
- origin allow/block policy and download/upload policy;
- permission grants (deny by default);
- fresh or named profile reference, never raw profile secrets;
- headless/headful mode and recording policy;
- requested region/proxy class for providers that support them.

`BrowserRunManifest` includes safe fields only:

- run/chain/stage/session correlation IDs;
- provider, provider session ID hash or safe ID, browser name/version, MCP version;
- normalized session spec and profile content hash;
- start/end/duration, action counts, disconnect/reconnect counts;
- structured termination and artifact-export status;
- measured provider usage and nullable cost with currency/source;
- artifact references and a separately access-controlled inspection/recording reference.

### Stable Failure Categories

```text
browser_policy_denied
browser_capacity_exhausted
browser_provision_failed
browser_connect_failed
browser_action_timeout
browser_session_timeout
browser_remote_disconnected
browser_artifact_export_failed
browser_cleanup_failed
browser_cost_unknown
```

Provider messages remain diagnostic detail. Eval classification and dashboards use the stable category.

### Isolation and Authority Rules

- Local: one browser process and disposable profile directory per eval session.
- Remote: one provider session per eval session unless a future explicitly named warm-profile experiment says otherwise.
- Named authenticated profiles are copied/materialized into a disposable session; they are not edited in place.
- CDP/MCP endpoints bind to loopback locally. Remote endpoints and auth headers use sensitive wrapper values and are redacted.
- The origin allowlist is an application policy, not a complete security boundary: redirects, DNS rebinding, WebSockets, service workers, and browser subprocess traffic require separate containment tests.
- Arbitrary model-authored Playwright/JavaScript is outside the neutral baseline. The model receives the pinned MCP tools, not shell authority over Chrome.
- Downloads land only inside the run workspace and are included in the artifact manifest.
- Human takeover is disabled for unattended evals; when enabled for debugging it emits explicit start/end events and marks the run non-comparable.

### Result Identity and Pooling

At minimum, comparable result identity becomes:

```text
(benchmark, language, model, executor, browser_tool_surface,
 browser_provider, browser_build, browser_policy_version)
```

Managed vessels additionally carry `agent_scaffold`. Reports must not aggregate a row with non-empty `agent_scaffold` into a neutral-browser model row unless the user explicitly requests a cross-vessel view.

### Implementation Plan

#### M1 — Contract, manifest, and fake provider (~1.5 days)

- [x] Add provider, controller, sensitive value, session spec, usage, artifact manifest, and error-category types.
- [x] Add a deterministic fake provider covering lifecycle success, connection, export, and cleanup failures; stable timeout/disconnect categories are adapter-testable.
- [x] Add redaction tests proving connection URLs, headers, API keys, and profile secrets do not appear in serialized results or ordinary errors; trace attributes use safe projections only.
- [x] Define browser result identity and banking fields without changing existing non-browser row semantics.

#### M2 — Local Playwright provider (~2 days)

- [x] Pin Node package and Chromium versions; fail loudly when `npx` is unavailable.
- [x] Create per-session directories and generated MCP configuration; launch isolated headless Chromium by default.
- [x] Capture MCP session output and Playwright artifacts supported by the chosen launch mode.
- [x] Make cancellation and timeout kill the complete MCP/browser process group.
- [ ] Prove cookie/local-storage isolation with two simultaneous sessions.
- [ ] Run a 50-session sequential leak smoke and stepped concurrency smoke at 8/16/24 sessions; record the safe default rather than assuming it from RAM.

#### M3 — Executor and eval-harness vertical slice (~1.5 days)

- [x] Wire Codex with first-class per-task MCP support as the reference vertical slice; preserve its ordinary non-browser mode.
- [x] Add benchmark/session configuration and provider selection with explicit defaults.
- [x] Attach browser spans/artifacts to the existing chain/stage IDs.
- [x] Bank browser identity, termination, usage, nullable cost, and neutral/managed comparability fields.
- [x] Add one hermetic browser fixture task and exact success grader.

#### M4 — Browserbase cloud adapter (~1.5 days)

- [x] Implement create/connect/inspect/export/stop using an injectable HTTP client and stub server tests.
- [x] Bind credentials through environment/Cloud Run secret injection; never serialize them into task payloads.
- [x] Generate Playwright MCP CDP configuration from sensitive connection material.
- [x] Add idempotent stop and leaked-session audit.
- [ ] Add opt-in live tests and a 20-session Cloud Run smoke.

#### M5 — Comparison, operations, and documentation (~1.5 days)

- [ ] Run the same fixed task/model/executor against local and Browserbase lanes.
- [ ] Compare success, cold start, wall time, disconnects, artifact completeness, and cost per success.
- [ ] Add dashboard links for safe local artifacts and access-controlled remote inspection.
- [x] Document provider setup, auth, local capacity tuning, failure recovery, and the neutral-vs-managed interpretation rule.
- [x] Run boundary, unit, integration, lint, and targeted process-lifecycle tests.

### Files to Modify/Create

Exact executor-specific wiring file is selected in M3 after confirming the reference executor's current MCP injection path.

**New files:**

- `internal/browser/provider.go` — lifecycle and provider registry (~180 LOC)
- `internal/browser/types.go` — specs, manifests, usage, stable failures, sensitive values (~260 LOC)
- `internal/browser/controller.go` — admission, MCP lifecycle, timeout, export, teardown (~300 LOC)
- `internal/browser/controller_test.go` — fake-provider lifecycle and failure tests (~350 LOC)
- `internal/browser/local/playwright.go` — local session/process/profile implementation (~300 LOC)
- `internal/browser/local/playwright_test.go` — isolation, cancellation, version, leak tests (~300 LOC)
- `internal/browser/browserbase/client.go` — Browserbase REST adapter (~260 LOC)
- `internal/browser/browserbase/client_test.go` — stub HTTP contract tests (~300 LOC)
- `internal/browser/testdata/` — hermetic browser fixture page and expected artifacts
- `docs/docs/guides/evaluation/browser-sessions.md` — setup, security, interpretation, operations (~500 lines)

**Modified files:**

- `internal/executor/factory.go` — browser/MCP session inputs without secrets in durable config (~50 LOC)
- reference executor package selected in M3 — generated MCP injection and cleanup (~100 LOC)
- `internal/eval_harness/agent_runner_multi.go` — browser session orchestration and result banking (~120 LOC)
- `internal/eval_harness/models.go` / `models.yml` — browser capability/vessel identity, not provider credentials (~50 LOC)
- eval result/schema and dashboard readers selected by code audit — optional browser metadata and artifact links (~150 LOC)
- Cloud Run job/secret configuration in the multivac infrastructure repository — Browserbase key and pinned MCP image/runtime

## Examples

### Example 1: Same neutral task locally and in cloud

```bash
# Local Mac Studio lane: pinned Chromium, disposable profile.
ailang eval-suite --agent \
  --models pi-gpt5-4 \
  --benchmarks browser-form-fixture \
  --browser-provider local-playwright

# Same prompt, model, executor, and MCP tools; only browser allocation changes.
ailang eval-suite --agent \
  --models pi-gpt5-4 \
  --benchmarks browser-form-fixture \
  --browser-provider browserbase
```

### Example 2: Structured result identity

```json
{
  "model": "pi-gpt5-4",
  "executor": "pi",
  "browser": {
    "tool_surface": "playwright-mcp@PINNED",
    "provider": "local-playwright",
    "browser_name": "chromium",
    "browser_version": "PINNED",
    "policy_version": "browser-policy-v1",
    "duration_ms": 18422,
    "termination": "completed",
    "cost_usd": null,
    "cost_source": "local-resource-unpriced",
    "artifacts": ["trace", "screenshots", "console"]
  },
  "agent_scaffold": null
}
```

The local infrastructure price remains null with the explicit `local-resource-unpriced` source; energy/amortization is not silently claimed to be free. A cloud adapter with unavailable pricing also records `cost_usd: null` and `browser_cost_unknown`.

### Example 3: Managed vessel remains distinct

```json
{
  "model": "gemini-3.6-flash",
  "executor": "managed_agents",
  "browser": {"provider": "google-antigravity-managed"},
  "agent_scaffold": "antigravity-preview-05-2026"
}
```

This row can appear in a cross-vessel report but is not pooled with the neutral Playwright-MCP baseline.

## Success Criteria

- [ ] Local and Browserbase runs complete the same browser fixture through identical MCP tool names and prompt.
- [x] Provider lifecycle is covered by fake/stub tests for success and representative stable failure categories.
- [ ] Local session isolation test proves cookies and local storage do not cross sessions.
- [x] Redaction tests prove endpoint/auth/profile secrets are absent from prompts, argv, JSON results, ordinary errors, and OTEL attributes.
- [x] Cancellation, timeout, and executor failure use bounded cleanup; the Codex process-group test proves MCP descendants are terminated.
- [x] Every completed browser lifecycle has a manifest with versions, policy, duration, termination, artifacts, and separately sourced nullable cost.
- [x] Managed-vessel and human-takeover rows are explicitly marked non-comparable.
- [ ] Local 50-session and Browserbase 20-session smoke results are recorded in the implementation report.
- [x] `make test`, `make lint`, and `make check-boundaries` pass.
- [x] Browser-session guide and runnable fixture example are added.

## Testing Strategy

**Unit tests:**

- Provider registry and configuration validation.
- Sensitive-value redaction across formatting, JSON, errors, spans, and logs.
- Controller state machine: create → connect → export → stop, including every failure edge.
- Idempotent stop and cleanup retry.
- Cost source/nullable-cost rules and neutral/managed pooling identity.
- Browserbase request/response/error mapping against a stub server.

**Integration tests:**

- Local pinned browser launches, navigates to a loopback fixture, interacts, exports artifacts, and exits.
- Two concurrent local sessions cannot observe each other's cookie or local storage.
- Executor cancellation terminates its MCP server and complete browser process group.
- Same fixture and prompt run through local and cloud providers.
- Cloud tests are opt-in and always stop sessions in cleanup handlers.

**Operational tests:**

- Sequential leak test plus stepped 8/16/24 local concurrency measurement.
- Browserbase 20-session leak/disconnect smoke through the Cloud Run job.
- Expired/invalid credential, capacity exhaustion, remote disconnect, and artifact-export fault injection.
- Confirm live-view/human takeover marks runs non-comparable.

**Security tests:**

- Origin policy direct navigation and redirect cases.
- Download path escape rejection.
- No public listener for local CDP/MCP; loopback binding assertion.
- Secret scan over banked result, transcript, spans, logs, and artifacts manifest.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Reference executor for the M3 vertical slice — agent may choose after auditing current first-class MCP injection; prefer the smallest reversible integration.
- Internal package helper names and provider registry mechanics — agent may choose while preserving the contract semantics.
- Exact pinned Playwright MCP/Chromium versions — agent selects current mutually compatible versions and records lockfile/digests.
- Safe local default concurrency — determined from the mandated 8/16/24 measurement, not hard-coded from hardware size.
- Artifact storage backend and retention defaults — human at implementation review; local filesystem is sufficient for the first hermetic test.
- Dashboard presentation — agent may choose the minimal existing artifact-link pattern.

## Non-Goals

**Not attempted in this feature:**

- Adding an AILANG `Browser` language effect or new `.ail` syntax — this is eval/executor infrastructure.
- Treating Playwright origin allow/block lists as a complete network security boundary.
- Executing arbitrary model-authored Playwright or JavaScript in the neutral baseline.
- Reusing the operator's normal Chrome profile or silently importing local credentials.
- Solving CAPTCHA, bot evasion, residential proxying, or automated purchasing.
- Making Browserbase the only permanent backend.
- Implementing AWS AgentCore, Steel, Browserless, Cloudflare Browser Run, Hyperbrowser, Anchor, Browser Use Cloud, or E2B adapters in v1.
- Comparing every agent/model/provider combination; the activation compare is deliberately small.
- Normalizing managed-agent scaffolds into pure model scores.
- Providing strong hostile-code isolation on native macOS; that requires a Linux VM/container or remote microVM lane.

## Timeline

**Week 1** (~24–28 hours):

- M1 contract, fake provider, manifest, redaction
- M2 local Playwright provider and isolation/leak tests
- Begin M3 reference executor vertical slice

**Week 2** (~20–28 hours):

- Complete M3 harness/banking integration
- M4 Browserbase adapter and Cloud Run smoke
- M5 comparison, operational hardening, documentation
- Buffer for browser process cleanup and cross-executor MCP differences

**Total: ~44–56 hours across 6–8 working days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| MCP/tool schema drift changes benchmark difficulty | High | Pin exact package/browser versions and include them in result identity; upgrades are named experiments. |
| CDP connection has lower fidelity than native Playwright protocol | Medium | Keep the tool surface common, record transport, test required artifacts/actions on each provider, and fail capability preflight rather than degrade silently. |
| Browser endpoint or profile credential leaks | High | Sensitive wrapper types, no generic serialization, redaction tests across every output plane, access-controlled inspection refs. |
| Browser/WebSocket traffic bypasses intended egress policy | High | State the boundary honestly; loopback fixture by default; add a dedicated browser-egress follow-up before untrusted destinations. |
| Local Chrome processes/profiles leak after cancellation | High | Controller-owned process groups, idempotent stop, orphan audit, sequential and concurrent leak tests. |
| Remote sessions leak and continue billing | High | Deferred cleanup on all paths, provider timeout, idempotent stop, post-run active-session audit. |
| Provider outage or API drift invalidates cloud evals | Medium | Typed provider failure, fake/stub contract tests, local lane remains available, provider is part of result identity. |
| Native macOS isolation is mistaken for a hostile-code sandbox | High | Constrained MCP tools only; explicit non-goal; require VM/microVM provider for arbitrary code or multi-tenant workloads. |
| Local and cloud browser fingerprints change task behavior | Medium | Record complete environment identity; compare by vessel and do not assume interchangeable results. |
| Pricing changes or cannot be derived reliably | Medium | Provider usage and price source are versioned; unknown is null/error, never guessed or silently zero. |

## Verification Log

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | No first-party browser session provider currently exists under executor/eval infrastructure | `rg -n -i 'browserbase|SessionProvider|browser session provider|connectOverCDP' internal/executor internal/eval_harness cmd` | No provider contract/adapter found; current managed agent and executor code is model/agent oriented. |
| V2 | AILANG has a common local/cloud executor shape and Managed Agents is HTTP/SSE rather than CLI subprocess | Read `docs/internal/EXECUTOR_SHAPE.md:1-29,127-233` and `internal/executor/executor.go` | Confirmed common executor interface, local registration, and separate Cloud Run deployment pillar; Managed Agents is explicitly the non-CLI exception. |
| V3 | Official Playwright MCP can launch isolated profiles and connect to CDP or a remote Playwright endpoint | [Microsoft Playwright MCP](https://github.com/microsoft/playwright-mcp) configuration documents `--isolated`, `--cdp-endpoint`, sensitive CDP headers, and `remoteEndpoint` | Confirmed; pinning is required because this external surface evolves. |
| V4 | Playwright warns CDP is Chromium-only and lower fidelity than native Playwright protocol | [Playwright `connectOverCDP`](https://github.com/microsoft/playwright/blob/main/docs/src/api/class-browsertype.md) | Confirmed; provider capability preflight and transport identity are required. |
| V5 | Browserbase supplies remote Playwright/CDP sessions plus live inspection/recording surfaces | [Browserbase session documentation](https://docs.browserbase.com/platform/browser/getting-started/using-browser-session) | Confirmed create/connect, live view, console/network/performance inspection, and recording/replay. |
| V6 | The current local server can support a native Playwright lane without a container runtime | `system_profiler SPHardwareDataType`; `/Applications` audit; `command -v docker podman` | Mac Studio M4 Max, 16 cores, 128 GB RAM, Google Chrome installed; Docker/Podman not found. Capacity is still measured rather than inferred. |
| V7 | The planned Net proxy boundary does not cover browser/WebSocket egress | Read `design_docs/planned/v0_33_1/m-net-effect-proxy-boundary.md`, especially Non-Goals | Confirmed: raw TCP, WebSocket, browser/WASM fetch, and subprocess bypasses are out of scope. |
| V8 | Managed Gemini Agents is intentionally a product-fidelity vessel, not a neutral model-ranking harness | Read `design_docs/planned/v0_31_0/m-managed-agents-model-eval.md`, Summary and Non-Goals | Confirmed; this design is distinct and supplies the neutral browser vessel it lacks. |
| V9 | Local/cloud eval banking already shares baseline/reporting paths | Read `design_docs/implemented/v0_30_0/m-eval-local-cloud-unify.md` | Confirmed; browser identity extends shared results rather than creating a separate database. |
| V10 | A self-hosted Steel follow-up is viable on this Apple Silicon server if a service boundary is later desired | [Steel repository](https://github.com/steel-dev/steel-browser) documents macOS native Node+Chrome operation; [container package](https://github.com/steel-dev/steel-browser/pkgs/container/steel-browser) lists `linux/arm64` | Confirmed as future option; not required for v1 local native Playwright. |
| V11 | The pinned local tool surface is installable on this server | `npx -y @playwright/mcp@0.0.79 --help`; `npm view @playwright/mcp@0.0.79 dependencies --json` | Passed; package exposes the required isolated/output/CDP options and pins compatible Playwright packages. |
| V12 | Browser contract, local provider, Browserbase adapter, Codex MCP injection, and eval orchestration pass focused tests | `go test ./internal/browser/... ./internal/eval_harness ./internal/executor/codex ./cmd/ailang` | Passed. Browserbase HTTP tests used a loopback stub. |
| V13 | MCP/Chromium descendants are killed on cancellation without data races | `go test -race ./internal/browser/... ./internal/executor/codex` | Passed outside the workspace sandbox because `httptest` requires a loopback listener. |
| V14 | Repository regression suite remains green | `make test` | Passed after implementation; subsequent manifest-only changes passed all directly affected package tests. |
| V15 | Static and architecture gates remain green | `make fmt-check`; `make lint`; `make check-boundaries`; `make check-file-sizes`; `git diff --check` | All passed; linter reported 0 issues. |
| V16 | Operator documentation builds in the production site | `cd docs && npx docusaurus build` | Passed. Three pre-existing broken-anchor warnings remain outside this feature. |
| V17 | Browser flags are discoverable in the built CLI | `make build`; `./bin/ailang eval-suite --help` | Passed; provider, artifact, region, and MCP-version flags are present. |
| V18 | Browserbase live lifecycle and 20-session Cloud Run soak | Credential check plus `AILANG_BROWSERBASE_LIVE=1 go test ...` gate | Not run: `BROWSERBASE_API_KEY` is absent. Stub coverage passes and the live test skips unless explicitly enabled. |
| V19 | Local 50-session and 8/16/24 capacity smoke | Commands documented in the operator guide | Not run in this implementation session; deployment/runtime capacity evidence remains required before choosing a production parallelism default. |
| V20 | Independent sprint evaluation | `sprint-evaluator` rubric against commit `19c8c38af`; report `.ailang/state/evaluations/eval_M-REMOTE-BROWSER-SESSIONS_round_1.json` | PASS, 80/100, 17/20 acceptance criteria verified, no hard failures. |
| V21 | Slow artifact export cannot consume the remote-session release budget | `TestControllerReservesIndependentStopBudgetAfterExportTimeout`; focused browser/eval tests | Passed after giving export and stop separate bounded cleanup contexts. Effective viewport and explicit unpinned locale/timezone defaults are also banked and tested. |

No AILANG language support/unsupported claims are made by this design, so no `ailang check` language probe is required.

## Related Documents

**Implemented:**

- [M-EVAL-LOCAL-CLOUD-UNIFY](../../implemented/v0_30_0/m-eval-local-cloud-unify.md) — shared banking/reporting path; this design adds browser vessel identity and artifacts.
- [M-COG-RUNTIME-BROWSER](../../implemented/v0_21_0/m-cog-runtime-browser.md) — Playwright browser-matrix and deterministic browser-runtime precedent; it tests AILANG WASM rather than providing agent sessions.
- [M-EXECUTOR-VARIANTS](../../implemented/v0_15_0/m-executor-variants.md) — executor configuration precedent.

**Planned:**

- [M-MANAGED-AGENTS-MODEL-EVAL](../../planned/v0_31_0/m-managed-agents-model-eval.md) — product-fidelity Gemini managed-agent vessel; explicitly distinct from neutral ranking.
- [M-NET-EFFECT-PROXY-BOUNDARY](../../planned/v0_33_1/m-net-effect-proxy-boundary.md) — adjacent HTTP egress boundary whose browser/WebSocket exclusions this design preserves.
- [M-AGENT-SAFE-RUNNER](../../planned/v1_1_0/m-agent-safe-runner.md) — operator-pinned execution policy precedent; does not supply browser infrastructure.
- [M-EVAL-EXPERIMENT-REGISTRY](../../planned/v0_31_0/m-eval-experiment-registry.md) — future location for named browser-provider/tool-version comparisons.

**Duplicate-gate result:** the skill's automatic neural/SimHash search returned unrelated high-score matches (recursion depth, flat if/else, monomorphization, and nightly measurement docs). Manual semantic/code search found no planned or implemented provider-neutral browser-session design. The related documents above overlap only in executor deployment, result banking, managed-vessel interpretation, or egress policy.

## References

- [Microsoft Playwright MCP](https://github.com/microsoft/playwright-mcp)
- [Playwright `connectOverCDP`](https://github.com/microsoft/playwright/blob/main/docs/src/api/class-browsertype.md)
- [Browserbase browser sessions](https://docs.browserbase.com/platform/browser/getting-started/using-browser-session)
- [AWS AgentCore Browser fundamentals](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/browser-resource-session-management.html) — future high-isolation adapter prior art
- [Steel self-hosted browser](https://github.com/steel-dev/steel-browser) — future portable service adapter prior art
- [Design Axioms](/docs/references/axioms)

## Implementation Report

**Completed**: 2026-08-23
**Target version**: v0.33.3
**Primary implementation commit**: `19c8c38af`
**Independent evaluation**: PASS, 80/100

### What Was Built

- Provider-neutral browser contract, secret-safe connection values, stable failures, manifests, controller, and lifecycle tests.
- Pinned local Playwright MCP provider with isolated sessions, content-addressed artifacts, and Codex process-group ownership.
- Browserbase create/connect/inspect/export/stop/audit adapter with injectable HTTP client, stub coverage, and an explicit credential-gated live test.
- Per-task Codex MCP configuration and executor capability validation without changing non-browser argument generation.
- Eval/CLI selection, safe OTEL/result banking, nullable browser cost, neutral/managed comparability, fixture, operator guide, and Cloud Run secret pattern.
- Post-evaluation hardening: independent stop cleanup budget after slow export, normalized effective viewport/policy provenance, and explicit unpinned locale/timezone provenance.

### Verification Outcome

Repository tests, lint, formatting, architecture boundaries, file-size checks, focused race tests, pinned MCP package smoke, CLI build/help, and Docusaurus production build passed. The evaluator verified 17 of 20 acceptance criteria and found no rubric hard fail.

### Production Activation Gates

- Real two-browser cookie/local-storage isolation evidence remains required; current hermetic tests prove unique process/session storage paths only.
- The fixture prompt is runnable but its grader does not yet prove a Playwright-specific tool call. Result tool-histogram enforcement is a follow-up.
- Browserbase live lifecycle/20-session Cloud Run soak and local 50-session plus 8/16/24 capacity measurements remain unexecuted because credentials/deployment capacity were not supplied.
- Safe dashboard rendering/access control for artifact and inspection links remains a UI follow-up.

These items block production qualification or full comparison claims, not the provider-neutral API and reference vertical slice delivered here.

## Future Work

- Add AWS AgentCore Browser when microVM isolation, session replay to owned storage, OS-level actions, or enterprise IAM become requirements.
- Add Steel as the preferred open/self-hosted service provider and Browserless as a mature compatibility provider.
- Add Cloudflare Browser Run as a cost/edge lane after long-session observability is validated.
- Add a dedicated browser/WebSocket egress boundary with redirect, DNS, service-worker, and proxy enforcement tests.
- Add a full-desktop provider such as E2B under a separate `desktop` capability and result identity.
- Add named Stagehand, Browser Use, computer-use, and managed-agent scaffold comparisons without changing the neutral baseline.
- Implement [M-BROWSER-AUTH-PROFILES](../../planned/v0_33_4/m-browser-auth-profiles.md) before using persistent authenticated identities; its P0 egress and artifact-data-policy follow-ups block authenticated production use.
- Consider an AILANG `Browser` effect only after the executor-side protocol and authority model have demonstrated stable semantics.

---

**Document created**: 2026-08-23
**Last updated**: 2026-08-23
