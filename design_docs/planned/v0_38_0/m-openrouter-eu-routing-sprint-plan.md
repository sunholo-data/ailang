# Sprint Plan: M-OPENROUTER-EU-ROUTING

## Summary

Add explicit, fail-closed EU routing to every AILANG OpenRouter inference client while preserving the global endpoint as the default. The implementation provides one shared resolver, supports the `openrouter-eu:` model prefix for eval matrices, exposes the effective route in telemetry, and classifies EU-capacity failures without ever retrying globally.

**Duration:** 2 engineering days (about 12–14 hours)
**Dependencies:** Approved `m-openrouter-eu-routing.md`; an OpenRouter Business account is needed only for the optional live smoke
**Risk Level:** Medium — routing is mechanically small, but a silent global fallback would violate the feature's core guarantee

## Current Status Analysis

### Repository findings

- `openrouter.WithBaseURL` already exists and is well tested, so no transport rewrite is needed.
- Production constructors exist in `internal/eval_harness/ai_provider.go`, `cmd/ailang/ai_handlers.go`, and `cmd/ailang/exec.go`; all currently use the global client default.
- `internal/ai/config.go` already centralizes the analogous `LYCEUM_BASE_URL` and `ZAI_BASE_URL` policies.
- `internal/mission/openrouter_quota.go` calls `/api/v1/key`; this is control-plane quota data rather than eval prompt/completion traffic. M0 must verify whether the EU host supports it before changing it; inference residency must not depend on that answer.
- Recent-history velocity is not measurable in this grafted coordinator checkout (one visible commit and no usable LOC stat), so the estimate uses the design doc's 1–2 day range plus a conservative testing/observability allowance.

### Scope boundary

The sprint changes all in-repository OpenRouter inference construction paths. It does not add a new provider type, manage Business-plan enrollment, snapshot OpenRouter's changing EU model catalogue, or fall back from EU to global. Example `models.yml` entries are documentation fixtures, not a wholesale duplication of the model matrix.

## Proposed Milestones

### M0: Systemic routing inventory and shared contract

**Goal:** Establish one reusable representation of the effective OpenRouter route and prove the complete constructor inventory before behavior changes.
**Estimated:** 45 implementation LOC + 55 test LOC = 100 LOC
**Duration:** 2 hours

**Example files to update:**

- `internal/ai/config.go`
- `internal/ai/config_test.go`
- `internal/ai/openrouter/client.go`
- `internal/ai/openrouter/client_test.go`

**Tasks:**

- Add exported global and EU base URL constants and an `OpenRouterBaseURL()` resolver honoring a trimmed `OPENROUTER_BASE_URL`.
- Add a small shared model-route resolver that recognizes `openrouter-eu:` and `openrouter:`, strips exactly one routing prefix, and returns the endpoint decision without introducing a provider type.
- Lock resolution precedence in table tests: EU prefix > environment override > global default.
- Re-run `rg "openrouter\\.NewClient"` and document every production constructor in a test/table or implementation comment; keep `/key` quota routing explicitly classified as control plane.

**Acceptance Criteria:**

- [ ] Unset env plus ordinary OpenRouter model resolves to `https://openrouter.ai/api/v1`.
- [ ] `OPENROUTER_BASE_URL` overrides the ordinary route, with surrounding whitespace ignored.
- [ ] `openrouter-eu:` always resolves to `https://eu.openrouter.ai/api/v1`, even when the env var names another host.
- [ ] Both routing prefixes are stripped before the model slug reaches the API.
- [ ] No production OpenRouter inference constructor remains outside the inventory.

### M1: Wire all inference entry points

**Goal:** Make evals and direct/API execution use the same routing contract.
**Estimated:** 85 implementation LOC + 125 test LOC = 210 LOC
**Duration:** 4 hours
**Dependencies:** M0

**Example files to update:**

- `internal/eval_harness/ai_provider.go`
- `internal/eval_harness/ai_provider_test.go`
- `cmd/ailang/ai_handlers.go`
- `cmd/ailang/ai_handlers_test.go`
- `cmd/ailang/exec.go`
- relevant `cmd/ailang/*_test.go`

**Tasks:**

- Use the shared resolver when constructing OpenRouter clients in the eval harness, direct AI effect handler, and API-mode executor.
- Preserve explicit `provider: openrouter` handling from `models.yml` while allowing `api_name: openrouter-eu:vendor/model`.
- Add loopback HTTP tests for each entry point, asserting EU-prefix and env-override requests reach only the selected server.
- Add a negative test that makes the EU target fail and asserts there is zero request to a global fallback server.

**Acceptance Criteria:**

- [ ] All three production inference construction paths honor `OPENROUTER_BASE_URL`.
- [ ] Eval `openrouter-eu:vendor/model` traffic reaches the configured EU test endpoint with `vendor/model` on the wire.
- [ ] A failed EU request returns its error without a second/global request.
- [ ] Existing unprefixed behavior and existing `openrouter:` models remain byte-compatible apart from added telemetry.

### M2: Diagnosability and route telemetry

**Goal:** Make EU routing and fail-closed capacity errors visible in banked eval results and spans.
**Estimated:** 65 implementation LOC + 95 test LOC = 160 LOC
**Duration:** 4 hours
**Dependencies:** M1

**Example files to update:**

- `internal/eval_harness/metrics.go`
- `internal/eval_harness/error_categorizer.go`
- `internal/eval_harness/error_categorizer_test.go`
- OpenRouter telemetry files/tests identified during M0

**Tasks:**

- Add `eu_region_unavailable` to the stable error taxonomy using a narrowly evidenced response/status predicate; unknown EU errors remain `api_error` rather than being guessed.
- Attach the normalized effective endpoint host (never credentials or URL query data) to OpenRouter request spans.
- Ensure Generate, Step, and StreamStep share the attribute behavior.
- If the exact Business/eligibility response shape cannot be captured, land a fixture-backed conservative classifier and record the live refinement as pending rather than broad string matching.

**Acceptance Criteria:**

- [ ] A representative EU in-region-unavailable fixture maps to `eu_region_unavailable`.
- [ ] Unrelated 4xx/5xx fixtures do not map to the EU category.
- [ ] Global and EU test calls emit distinguishable sanitized endpoint-host attributes across all request modes.
- [ ] No telemetry field contains API keys or arbitrary query parameters.

### M3: CLI documentation, config example, and verification

**Goal:** Make the option discoverable and close the sprint with automated and optional live evidence.
**Estimated:** 35 implementation/docs LOC + 45 test/docs-validation LOC = 80 LOC
**Duration:** 2–4 hours
**Dependencies:** M2

**Example files to update:**

- CLI help source identified by the `cli-doc-maintainer` workflow
- `docs/docs/guides/evaluation/README.md` or the current evaluation guide
- `models.yml` (one disabled/example EU variant only if consistent with current schema conventions)
- `CHANGELOG.md`

**Tasks:**

- Document `OPENROUTER_BASE_URL`, `openrouter-eu:`, precedence, Business-plan dependency, and strict no-fallback behavior in CLI help and evaluation docs.
- Provide one validated model configuration example without claiming a permanent EU-eligibility list.
- Run focused tests, `make fmt`, `make lint`, `make test`, and `make check-boundaries`.
- Optional metered smoke: one eligible model succeeds through the EU host; one known-ineligible model fails closed; verify span host and category. Do not block the code sprint when Business credentials are unavailable—record `NEEDS-LIVE-SMOKE` explicitly.

**Acceptance Criteria:**

- [ ] CLI help and docs describe both selection mechanisms and their precedence.
- [ ] Documentation explicitly says EU routing requires OpenRouter Business and never falls back globally.
- [ ] Any shipped `models.yml` example passes config loading and preserves the provider/model slug.
- [ ] Focused and repository gates pass; any environment-only live check is recorded separately and is not represented as green without evidence.

## Day-by-Day Plan

### Day 1

1. Complete M0 inventory/resolver with table tests.
2. Complete M1 wiring using loopback endpoints and the no-global-fallback negative control.
3. Run focused `internal/ai`, `internal/eval_harness`, and `cmd/ailang` tests.

### Day 2

1. Complete M2 error taxonomy and telemetry across Generate/Step/StreamStep.
2. Complete M3 help/docs/example work using the `cli-doc-maintainer` skill during execution.
3. Run formatting, lint, full tests, architecture boundaries, and the optional live smoke if authorized credentials support it.

## Success Metrics

- 100% of the three inventoried OpenRouter inference constructors use the shared route decision.
- Automated tests prove prefix/env/default precedence and zero global fallback after an EU failure.
- Global and EU requests are distinguishable by sanitized endpoint host in telemetry.
- `eu_region_unavailable` is specific and fixture-backed; ambiguous failures remain `api_error`.
- One configuration example loads successfully; no AILANG `.ail` example is required because this is host-side eval routing.
- `make fmt`, `make lint`, `make test`, and `make check-boundaries` pass.

## Dependencies and Risks

- **Exact OpenRouter error shape:** capture from a Business-plan smoke if possible; mitigate with conservative matching and fixture provenance.
- **Constructor drift:** central helper plus an inventory test/search gate reduces future bypasses.
- **False compliance claim:** docs state that endpoint choice supplies routing, while account plan and OpenRouter service guarantees remain external prerequisites.
- **Mixed-result comparability:** EU/global latency and provider selection may differ; keep them labeled and avoid combining them blindly in Elo pools.
- **Quota endpoint ambiguity:** do not route `/api/v1/key` through the EU host unless documented or tested; quota control-plane locality is not equivalent to inference data residency.

## Executor Handoff Notes

- Start with M0 and revise the affected-file list from evidence before editing behavior.
- Use TDD for precedence and the no-fallback invariant; that negative control is release-critical.
- Invoke `cli-doc-maintainer` for CLI help/environment-variable changes.
- A user must say **execute sprint** before implementation begins.

