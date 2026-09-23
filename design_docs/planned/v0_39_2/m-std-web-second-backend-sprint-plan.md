# Sprint Plan: M-STD-WEB-SECOND-BACKEND

## Summary

Add operator-selected Gemini grounding behind the existing `std/web` backend seam while preserving the public AILANG contract, the `{Net}` authority boundary, and Ollama as the compatibility default. The sprint begins with a hard external-API probe and only proceeds to runtime wiring if Gemini Google Search grounding is available on the pinned model.

**Duration:** 3 working days, approximately 10 hours
**Dependencies:** Approved [M-STD-WEB-SECOND-BACKEND design](m-std-web-second-backend.md); `GEMINI_API_KEY` for the Phase 0 and opt-in live probes; existing M-DANEEL-AILANG-EXECUTOR M2 `std/web` implementation
**Risk Level:** Medium

## Current Status Analysis

### Completed Prerequisites

- `std/web` already exports `webSearch` and `webFetch` with typed `Result` values and `{Net}` effects.
- `internal/effects/web.go` already isolates provider behavior behind `webBackend`; Ollama remains the only implementation.
- `GEMINI_API_KEY` is already registered in `internal/config`.
- The operator has approved the selector contract: policy field `web_backend`, values `ollama` or `gemini`; environment selection only outside `--policy`.
- The design audits policy admission, environment resolution, allowlisting, auth headers, transcript metadata, response decoding, and secret non-leak carriers as one systemic change.

### Velocity and Capacity

The checkout is a v0.42.0 release snapshot with only the release commit visible in the last seven days, so recent LOC/day cannot be calculated responsibly. Planning therefore uses the design's measured work decomposition (8–10 hours), a three-day elapsed window for external probing and full gates, and approximately 670 changed lines including tests, fixtures, generated documentation, and comments. Progress is gated by observable acceptance criteria rather than an unsupported throughput number.

### Remaining Work

- Verify the current Gemini grounding and `url_context` contracts against the live API and bank sanitized fixtures.
- Add closed backend resolution to policy and non-policy admission paths.
- Implement and fixture-test the Gemini backend without changing `std/web` types or effects.
- Extend the existing secret non-leak test across both providers.
- Update operator documentation, generated environment-variable reference, and release notes; run repository-wide gates.

## Proposed Milestones

### M0: Verify External Gemini Premises

**Goal:** Resolve the design's only open freeze item before provider wiring.
**Estimated:** 20 LOC of sanitized JSON fixtures/implementation notes; 2 hours
**Example files:** `internal/effects/testdata/gemini_web_search.json`; optionally `internal/effects/testdata/gemini_web_fetch.json`
**Dependencies:** `GEMINI_API_KEY`; network access to `generativelanguage.googleapis.com`

**Tasks:**

- Probe a pinned Gemini model with `tools: [{"google_search": {}}]` and record the exact successful request/response shape.
- Probe `url_context` on the same model and choose the design-approved fetch implementation or typed unsupported-operation result.
- Sanitize and bank response fixtures; record the model and observed quota/pricing documentation in the implementation report or changelog notes.
- Stop and return to the operator if Google Search grounding itself is unavailable; lack of `url_context` alone does not block the sprint.

**Acceptance Criteria:**

- [ ] Search grounding succeeds on a named pinned model and the fixture contains real response structure with no credentials or sensitive query data.
- [ ] The `url_context` verdict is recorded and deterministically selects one D7 branch.
- [ ] No implementation milestone begins if the primary grounding premise fails.

**Risks:** API/model drift or exhausted quota. Mitigation: probe first, preserve the observed fixture, fail loudly instead of guessing a schema.

### M1: Add Closed Backend Selection and Admission

**Goal:** Resolve the selected backend once at admission while keeping policy files authoritative and backward compatible.
**Estimated:** 80 LOC implementation + 80 LOC tests = 160 LOC; 3 hours
**Example files:** update `internal/policy/policy.go`, `internal/config/compiler.go`, `internal/effects/web.go`, `cmd/ailang/run_policy.go`, `cmd/ailang/main_run.go`, and their existing test files
**Dependencies:** M0 confirms the Gemini hostname and pinned endpoint

**Tasks:**

- Add `web_backend` to the policy and `AILANG_WEB_BACKEND` to the config registry.
- Extend `webBackend` with provider-specific key and auth-header behavior; add a closed `SetWebBackend` resolver.
- Under `--policy`, resolve only the policy value, default absent to Ollama, validate capability/host consistency, and ignore the environment.
- Outside `--policy`, resolve the environment value, default absent to Ollama, and reject unknown values.
- Add `web_backend` to the JSON admission line beside `ai_provider`.

**Acceptance Criteria:**

- [ ] Resolution matrix covers absent, explicit Ollama, explicit Gemini, unknown value, environment-only selection, and environment ignored under `--policy`.
- [ ] Unknown selectors name the invalid value and valid closed set; no fallback occurs.
- [ ] A configured backend without `Net`, or without its required host in `net_allow`, is refused before execution.
- [ ] Existing policy fixtures lacking `web_backend` retain Ollama behavior.
- [ ] Admission JSON records the resolved backend.

**Risks:** mutable package selection leaking between tests. Mitigation: restore backend globals with test cleanup and avoid parallel tests that mutate them.

### M2: Implement Gemini Grounding Backend

**Goal:** Serve the unchanged `std/web` contract from Gemini through the existing secure request and allowlist path.
**Estimated:** 170 LOC implementation + 80 LOC tests = 250 LOC; 4 hours
**Example files:** create `internal/effects/web_gemini.go`; update `internal/effects/web_test.go`; use `internal/effects/testdata/gemini_web_search.json` and the conditional fetch fixture
**Dependencies:** M0 and M1

**Tasks:**

- Implement search request construction with the pinned model, Google Search tool, and `x-goog-api-key` header.
- Decode grounding chunks/supports into `{title,url,content}`, deduplicate by URI, preserve order, and enforce `max` locally.
- Implement fetch using the M0-confirmed `url_context` contract or return the design-approved typed unsupported error without issuing a request.
- Add fixture-server tests for URL, payload, auth-header form, response mapping, malformed responses, missing metadata, deduplication, and truncation.
- Prove the fixed Gemini hostname still passes through normal `net_allow` enforcement.

**Acceptance Criteria:**

- [ ] Fixture-backed Gemini search returns the same program-visible record shape as Ollama without changes to `std/web.ail` signatures or effects.
- [ ] Gemini credentials are sent only in `x-goog-api-key`, never in a URL.
- [ ] Malformed or incomplete provider responses become typed `Transport` errors naming Gemini, not panics or partial silent results.
- [ ] Missing Gemini host permission produces typed `DisallowedHost`.
- [ ] Fetch follows the recorded D7 branch and never falls back across providers.

**Risks:** grounding support indices or metadata vary across responses. Mitigation: decode conservatively from a live fixture and fail with a typed provider-named error when required structure is absent.

### M3: Generalize the Lending-Boundary Security Tests

**Goal:** Demonstrate that neither backend can expose its API key through any existing program-visible carrier.
**Estimated:** 20 LOC implementation adjustments + 160 LOC tests = 180 LOC; 2 hours
**Example files:** update the single `TestWeb_SecretNeverLeaks` table in `internal/effects/web_test.go`; add `TestWeb_LiveGemini` beside the existing live Ollama control
**Dependencies:** M2

**Tasks:**

- Parameterize the existing non-leak test over Ollama and Gemini rather than copying it.
- Give each backend an independent sentinel, fixture server, base URL, expected key variable, and expected auth-header shape.
- Run the complete existing carrier sweep and positive control for each provider; retain at least two key-echoing mutation cases per provider.
- Add an opt-in Gemini live test guarded by `AILANG_WEB_LIVE_TEST=1` and `GEMINI_API_KEY`.

**Acceptance Criteria:**

- [ ] The same non-leak test executes at least 12 carriers, a positive control, and two mutations for each backend.
- [ ] Missing-key errors identify the selected backend's own environment variable.
- [ ] Both Bearer and `x-goog-api-key` header paths are asserted by fixture servers.
- [ ] Live controls remain opt-in and CI does not require external credentials.

**Risks:** a false-green parameterized test could fail to exercise one provider. Mitigation: named subtests, per-provider request counters, and positive controls prove each row ran.

### M4: Documentation, Example Regression, and Full Gates

**Goal:** Make the operator surface discoverable and verify no repository-wide regressions.
**Estimated:** 40 LOC documentation/comments + 20 LOC generated/reference changes = 60 LOC; 1–2 hours
**Example files:** update `std/web.ail`, `docs/docs/guides/agent-tool-policy.md`, generated `docs/docs/reference/env-vars.md`, and the current changelog; verify `examples/runnable/web_search.ail`
**Dependencies:** M1–M3

**Tasks:**

- Document `web_backend`, the policy-vs-environment precedence rule, backend host requirements, and admission metadata.
- Generalize the `std/web.ail` header comment without changing exported declarations.
- Regenerate the environment-variable reference using the repository target.
- Type-check/run the existing web search example where credentials permit and confirm its source needs no provider-specific change.
- Run focused tests, `make test`, `make lint`, `make check-boundaries`, example verification, and `make simplicity-audit`; explain the intentional new env-var/policy surface.

**Acceptance Criteria:**

- [ ] Operator docs describe both valid values, Ollama compatibility default, environment behavior, and required host/key.
- [ ] Generated env-var documentation matches the config registry.
- [ ] `examples/runnable/web_search.ail` remains provider-neutral and passes non-live validation.
- [ ] Focused and full tests pass; lint and architecture boundaries are clean.
- [ ] Simplicity audit contains no unexplained regression.
- [ ] Changelog links the design and records the new operator surface.

**Risks:** the new environment route trips a simplicity threshold. Mitigation: measure explicitly and document the approved, bounded surface rather than weakening the gate.

## Day-by-Day Execution

### Day 1 (approximately 5 hours)

- Complete M0 and record the fetch verdict.
- Implement M1 test-first, including the full resolution/refusal matrix.
- Gate: focused policy/config/effects and CLI tests are green before provider decoding starts.

### Day 2 (approximately 4 hours)

- Complete M2 from the recorded fixtures.
- Complete M3's parameterized non-leak sweep and opt-in live control.
- Gate: all `internal/effects` tests pass with no live credentials required.

### Day 3 (approximately 1–2 hours)

- Complete M4 documentation, generation, example regression, and full repository gates.
- Record any API premise deviation in the implementation report and reconcile it with the design before completion.

## Success Metrics

- Gemini `webSearch` succeeds end-to-end in the opt-in live control and fixture-backed tests.
- Zero public signature, builtin, or effect-row changes in `std/web`.
- Every unknown selector or unsupported operation fails loudly and names the selected backend; no cross-provider fallback exists.
- Both API keys pass the shared secret-carrier sweep with positive controls.
- Existing policies default to Ollama; policy authority cannot be overridden by `AILANG_WEB_BACKEND`.
- The selected backend is observable on the admission line.
- Existing provider-neutral example remains valid.
- Full tests, lint, boundary checks, docs generation, and simplicity audit are green or have an explicitly accepted pre-existing failure documented by the executor.

## Dependencies and Stop Conditions

- Primary grounding unavailable on the pinned model: stop after M0 and return to the operator.
- `url_context` unavailable: continue with the typed unsupported-fetch branch already approved in D7.
- Credentials unavailable locally: fixture-based work may proceed only if a trustworthy sanitized live fixture already exists; otherwise M0 remains blocked.
- No implementation may introduce a query-parameter API key, policy-to-environment fallback, dynamic backend URL/model policy, or cross-backend operation fallback.

## Handoff

After human approval, invoke `sprint-executor` with this plan and `.ailang/state/sprints/sprint_M-STD-WEB-SECOND-BACKEND.json`. Execution must begin at M0 and update only the progress fields in the sprint JSON as milestones complete.
