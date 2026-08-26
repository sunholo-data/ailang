# Sprint Plan: Remote Browser Session Providers

**Status:** Implemented; independent evaluation PASS 80/100. Production activation evidence remains pending as recorded in the design implementation report.

## Summary

Implement the provider-neutral browser-session layer targeted at v0.33.3 and
described in `m-remote-browser-session-providers.md`, with local Playwright/Chromium and
Browserbase backends, a Codex MCP vertical slice, eval result banking, and
operator documentation. The sprint keeps vendor credentials and connection
material out of prompts, serialized task payloads, logs, and traces.

**Sprint ID:** M-REMOTE-BROWSER-SESSIONS  
**Duration:** 7 engineering days  
**Dependencies:** Existing executor registry, eval harness, Cloud Run secret
pattern, Node.js 20+ for the local Playwright MCP command  
**Risk Level:** High

## Current Status Analysis

### Completed Recently

- ✅ Executor unification and stream event work provides a stable task/result
  boundary for optional MCP injection.
- ✅ Eval sessions already carry chain/stage identity and bank structured
  provider data.
- ✅ The design document freezes Browserbase as the first remote neutral
  provider and Playwright/Chromium as the local backend.
- ✅ The autonomous eval suite reported 6/6 passing immediately before sprint
  execution, providing a clean functional reference point.

### Velocity

- Recent repository history is dominated by mission-scale generated changes;
  the raw 14-day diff is not a useful single-sprint predictor.
- The changelog velocity analyzer found no reliable per-feature LOC samples.
- Planning basis: the design document's conservative 6–8 day estimate, reduced
  coupling through an `internal/browser` package, and incremental test-first
  milestones.
- Estimated capacity: approximately 2,450 implementation-and-test LOC across
  seven days.

### Remaining from Design Doc

- ⏳ Provider contract, controller, fake provider, manifest, and redaction:
  ~650 LOC.
- ⏳ Local Playwright MCP provider and isolation tests: ~500 LOC.
- ⏳ Codex MCP wiring and lifecycle integration: ~350 LOC.
- ⏳ Browserbase REST/CDP adapter and stub tests: ~500 LOC.
- ⏳ Eval banking, fixture, documentation, and deployment checks: ~450 LOC.

## Proposed Milestones

### Milestone 1: Provider Contract and Safety Boundary

**Goal:** Establish a deterministic provider/controller API whose public values
are safe to serialize and whose connection secrets remain process-local.

**Estimated:** 420 LOC implementation + 230 LOC tests = 650 LOC  
**Duration:** 1.5 days

**Tasks:**

- Day 1: Add browser session specs, stable error categories, public manifest,
  usage/cost fields, artifact references, and sensitive connection types.
- Day 1: Add provider registry/controller lifecycle with cleanup ordering and
  idempotent stop behavior.
- Day 2: Add a deterministic fake provider covering lifecycle and injected
  failure paths.
- Day 2: Add JSON, error, and diagnostic redaction tests, including recursive
  provider-data scrubbing.

**Acceptance Criteria:**

- [x] Provider interface supports create, connection, inspect, export, and
  stop without exposing raw connection material in public result types.
- [x] Stable failure categories cover unavailable, auth, quota, launch,
  connect, timeout, disconnect, export, and cleanup failures.
- [x] Fake-provider and adapter tests exercise success plus representative stable failure categories.
- [x] Controller cleanup is idempotent and preserves the primary failure while
  reporting cleanup failures separately.
- [x] Redaction tests prove URLs, headers, API keys, cookies, and profile secrets
  do not survive JSON serialization, ordinary errors, or structured diagnostics.
- [x] Package tests pass and linting is clean.

**Risks:**

- Go values can leak secrets through `String`, `%v`, or nested maps. Mitigation:
  use an opaque sensitive type, explicit safe projections, and recursive
  sanitizer tests.
- Provider-specific fields can contaminate the shared contract. Mitigation:
  keep vendor payloads private to adapters and expose stable public fields only.

### Milestone 2: Local Playwright/Chromium Provider

**Goal:** Produce isolated, pinned Playwright MCP session configurations suitable
for the current Mac Studio and generic Linux workers.

**Estimated:** 310 LOC implementation + 190 LOC tests = 500 LOC  
**Duration:** 1.5 days

**Dependencies:** Milestone 1

**Tasks:**

- Day 2: Implement the local provider with per-session state/artifact
  directories, headless-by-default policy, and pinned package/browser metadata.
- Day 3: Generate a safe stdio MCP connection spec for `@playwright/mcp` and
  validate required runtime executables without silent fallback.
- Day 3: Add simultaneous-session cookie/local-storage fixture coverage and
  cleanup tests using hermetic fake commands; add an opt-in real-browser smoke.
- Day 3: Add capacity-smoke tooling for sequential and stepped-concurrency runs,
  with explicit operator-selected limits instead of inferred RAM limits.

**Acceptance Criteria:**

- [x] Every local session receives unique state and artifact directories.
- [x] MCP package/version, Chromium identity, headless policy, and isolation mode
  appear in the public manifest.
- [x] Missing `npx` or an invalid configured package fails loudly as
  `provider_unavailable` or `launch_failed`.
- [x] Two concurrent test sessions cannot share their profile paths or fixture
  cookie/local-storage state.
- [x] Cancellation and timeout cleanup remove session-owned state and leave no
  controller-owned process running.
- [x] An opt-in 50-session/capacity command exists and records results without
  becoming a mandatory CI dependency.
- [x] Package tests pass and linting is clean.

**Risks:**

- Playwright MCP flags and package behavior can change. Mitigation: pin the npm
  package version and keep argument generation under exact tests.
- The MCP subprocess is owned by the executor, not the provider. Mitigation:
  express connection data as a first-class MCP spec and make the executor's
  context/process-group lifecycle authoritative.

### Milestone 3: Codex MCP Vertical Slice

**Goal:** Let one eval executor consume a browser session through standard MCP
tools while preserving non-browser Codex execution exactly.

**Estimated:** 220 LOC implementation + 130 LOC tests = 350 LOC  
**Duration:** 1 day

**Dependencies:** Milestones 1–2

**Tasks:**

- Day 4: Add generic per-task stdio MCP server configuration to the executor
  task contract, with environment-variable forwarding rather than secret values.
- Day 4: Generate ephemeral Codex `-c mcp_servers.*` overrides and pass remote
  endpoints only through the child environment.
- Day 4: Wrap browser-enabled executor runs in create/connect/export/stop with
  deferred cleanup on success, error, cancellation, and timeout.
- Day 4: Attach browser spans, artifacts, and manifest fields to the existing
  chain/stage/result identity without changing non-browser rows.

**Acceptance Criteria:**

- [x] Codex receives the same Playwright MCP tool surface for local and remote
  providers.
- [x] Non-browser Codex argument generation is byte-for-byte unchanged.
- [x] Secret endpoint/auth values are absent from argv, prompts, task metadata,
  banked provider data, errors, and trace attributes.
- [x] Timeout, cancellation, and executor failure still call export and stop,
  with bounded cleanup contexts.
- [x] Unit/integration tests cover local wiring, remote env forwarding, and all
  cleanup paths.
- [x] Package tests pass and linting is clean.

**Risks:**

- Codex configuration syntax could drift. Mitigation: isolate config generation,
  exact-test it, and document the minimum supported Codex CLI behavior.
- Passing secrets via command arguments exposes them in process listings.
  Mitigation: forward only environment variable names in Codex config; place
  values solely in the spawned process environment.

### Milestone 4: Browserbase Remote Provider

**Goal:** Implement the first hosted neutral backend using Browserbase session
lifecycle and CDP connection data, fully covered by stub-server tests.

**Estimated:** 310 LOC implementation + 190 LOC tests = 500 LOC  
**Duration:** 1.5 days

**Dependencies:** Milestones 1 and 3

**Tasks:**

- Day 5: Add Browserbase config validation and injectable HTTP client with
  bounded request timeouts and typed API errors.
- Day 5: Implement create, connection lookup, inspect, artifact projection, and
  idempotent stop/audit behavior against a stub server.
- Day 5: Convert CDP connection material to the common Playwright MCP spec via
  environment forwarding.
- Day 6: Add opt-in live smoke support for one session and a 20-session Cloud
  Run soak, gated by credentials and explicit enablement.

**Acceptance Criteria:**

- [x] Stub tests cover success, authentication failure, quota/rate limit,
  malformed response, timeout, disconnect, export failure, cleanup failure,
  idempotent stop, and leaked-session audit.
- [x] Browserbase credentials are sourced from environment/secret injection and
  never serialized into session specs or task payloads.
- [x] Remote CDP material is only held in opaque connection values and child
  process environment.
- [x] Manifest includes remote session ID, safe inspection identity,
  termination reason, artifact links, usage, and nullable separately sourced
  cost.
- [x] Live tests skip with an explicit reason when credentials are absent and
  never run implicitly in CI.
- [x] Package tests pass and linting is clean.

**Risks:**

- Vendor API response and artifact availability vary by plan/version.
  Mitigation: tolerant private decoding, stable public projection, stub
  fixtures, and an opt-in contract smoke.
- Remote sessions can leak on controller failure. Mitigation: bounded deferred
  stop plus a provider audit method that reports active sessions.

### Milestone 5: Eval Integration, Fixture, Docs, and Release Gates

**Goal:** Make browser providers selectable in eval/session configuration, bank
neutral comparison fields, and document local and cloud operation.

**Estimated:** 270 LOC implementation + 180 LOC tests/docs = 450 LOC  
**Duration:** 1.5 days

**Dependencies:** Milestones 1–4

**Tasks:**

- Day 6: Add explicit browser provider selection to multi-executor eval config
  without modifying the user-edited model registry.
- Day 6: Add a hermetic local browser fixture and exact grader; bank browser
  identity, termination, usage, artifact completeness, nullable cost, and
  managed-vessel label.
- Day 7: Add setup/operations documentation for local capacity tuning,
  Browserbase secrets, failure recovery, live-smoke activation, and
  neutral-versus-managed interpretation.
- Day 7: Run focused tests, race-sensitive lifecycle tests, repository test,
  lint, formatting, boundary, and file-size gates; record any credential-gated
  live checks as pending deployment evidence.

**Acceptance Criteria:**

- [x] Eval configuration selects `local-playwright` or `browserbase` explicitly;
  ordinary evals remain unchanged when browser config is absent.
- [x] Browser-enabled results bank provider, session identity, termination,
  duration, usage, artifact inventory/completeness, nullable cost and cost
  source, plus neutral/managed-vessel classification.
- [x] The same fixture prompt/tool contract can run against both neutral
  providers.
- [x] Managed-vessel rows are explicitly marked non-comparable by default.
- [x] Operator guide includes runnable local smoke, Browserbase setup, Cloud Run
  secret pattern, cleanup/audit recovery, and capacity commands.
- [x] Existing unrelated `models.yml`, benchmark history, and eval helper-script
  changes remain untouched.
- [x] `make test`, `make lint`, `make fmt-check`, `make check-boundaries`, and
  `make check-file-sizes` pass.

**Risks:**

- Full live parity depends on external Browserbase credentials and billing.
  Mitigation: make the contract test hermetic, keep live checks explicit, and
  record deployment evidence separately.
- Eval configuration changes can disturb existing sessions. Mitigation: use an
  additive optional field and zero-value-preserving behavior with regression
  tests.

## Success Metrics

- Provider lifecycle paths: 100% of stable failure categories represented in
  deterministic tests.
- Secret safety: zero known connection/auth/profile values in serialization,
  argv, logs, errors, or trace projection tests.
- Compatibility: all existing non-browser executor and eval tests pass without
  configuration changes.
- Local isolation: concurrent fixture sessions use distinct profiles and do not
  exchange cookie/local-storage values.
- Remote contract: Browserbase adapter behavior covered by a hermetic HTTP stub;
  live smoke is opt-in and evidence-bearing.
- Documentation: design doc, sprint plan, operator guide, and implementation
  report/changelog are synchronized.
- Repository gates: test, lint, formatting, boundaries, and file-size checks all
  pass.

## Dependencies

- Go standard HTTP/process primitives and existing executor/eval abstractions.
- Node.js 20+ and `npx` for local live execution.
- Pinned `@playwright/mcp` package and compatible Chromium for live execution.
- Browserbase project ID/API key only for opt-in live checks.
- Codex CLI with per-invocation MCP configuration support for the first executor
  vertical slice.

## Open Questions

- No blocking product decisions. The design freezes Browserbase first, local
  Playwright first, Codex as the reference executor, and AWS AgentCore/Steel as
  future adapters.
- A production concurrency default remains an operational measurement, not a
  compiled assumption; start conservatively and promote only after the capacity
  smoke on the actual worker image.

## Notes

- Milestones execute sequentially because each extends the same lifecycle and
  safety boundary; this avoids conflicting edits in executor/eval integration.
- Live Browserbase and high-concurrency local smokes are deployment gates. Their
  absence must be reported explicitly and cannot be represented as passing.
- The sprint will not edit `internal/eval_harness/models.yml`; browser selection
  is additive session configuration so existing user model work is preserved.
- Managed OpenAI/Anthropic/Gemini computer-use products remain distinct eval
  vessels and are not pooled with neutral Browserbase/local rows.
