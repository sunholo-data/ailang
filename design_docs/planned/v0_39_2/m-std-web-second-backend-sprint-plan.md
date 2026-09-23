# Sprint Plan — M-STD-WEB-SECOND-BACKEND

**Design doc:** [m-std-web-second-backend.md](m-std-web-second-backend.md)  
**Sprint ID:** `M-STD-WEB-SECOND-BACKEND`  
**Task:** `task-18d0125a` (amendment to `inbox_1789582783331_40c6a750`)  
**Planned at:** v0.42.0, HEAD `cb6fc0f0`, 2026-09-23  
**Duration:** 3 working days (8–12 focused hours)  
**Risk:** Medium overall; Phase 0 has a high-impact external API premise  
**Dependencies:** Approved design doc; `GEMINI_API_KEY` for opt-in probes; landed std/web seam

## Summary

Add Gemini grounding as an operator-selected second implementation behind the existing `std/web`
contract. The operator selects `web_backend = "ollama" | "gemini"` per policy lane; plain runs
use `AILANG_WEB_BACKEND`; unknown values are refused before execution; and the resolved backend is
recorded beside `ai_provider` on the admission line. No AILANG API, builtin signature, or effect
row changes.

The repository has only one commit in the current 7/14-day velocity windows (the v0.42.0 release),
so commit-derived LOC/day is not a reliable estimator. This plan instead uses the approved design's
file-level estimates, adds fixture and regression-test effort, and preserves its conservative
three-day schedule.

## Scope and invariants

- Under `--policy`, the policy file is the sole authority. An absent field resolves to `ollama`;
  `AILANG_WEB_BACKEND` must not be read.
- Outside `--policy`, unset `AILANG_WEB_BACKEND` resolves to `ollama`.
- The only accepted values are `ollama` and `gemini`; unknown values fail loudly without fallback.
- Gemini uses `x-goog-api-key`, never a query parameter, and remains subject to `net_allow`.
- The admission JSON gains additive field `web_backend` beside `ai_provider`.
- Existing `std/web` types, effect rows, examples, and Ollama-default behavior remain unchanged.
- The existing secret-carrier sweep is parameterized for both backends rather than duplicated.

## Milestones

### M1: Verify Gemini contract and bank fixtures (~80 LOC)

**Goal:** Resolve the design's only open external premise before production wiring.  
**Duration:** 2 hours  
**Dependencies:** None

**Files:**

- `internal/effects/testdata/gemini_web_search.json` (new)
- `internal/effects/testdata/gemini_web_fetch.json` (new only if `url_context` is supported)
- implementation report or milestone notes in the sprint JSON

**Tasks:**

- Run an opt-in `generateContent` probe against the pinned model with Google Search grounding.
- Record and sanitize the exact `groundingMetadata` response as a fixture.
- Probe `url_context`; record its fixture if supported, otherwise record the typed unsupported-fetch
  decision required by D7.
- Record the observed grounding quota/pricing source and chosen pinned model in milestone notes.
- Stop and return to the operator if Google Search grounding itself is unavailable.

**Acceptance criteria:**

- [ ] A sanitized real response fixture proves the request and response field shapes used by search.
- [ ] The pinned model and `url_context` verdict are recorded before M3 begins.
- [ ] Fixtures contain no API key, authorization header, user-specific data, or transient request ID.
- [ ] If grounding is unavailable, execution stops without selector or fallback code being shipped.

### M2: Implement selector, policy admission, and environment resolution (~260 LOC)

**Goal:** Establish one validated backend selection path with policy authority and observable
admission behavior before adding Gemini decoding.  
**Duration:** 3 hours  
**Dependencies:** M1 contract verdict (fixture decoding may wait for M3)

**Files:**

- `internal/policy/policy.go`, `internal/policy/policy_test.go`
- `internal/config/compiler.go` and its tests
- `internal/effects/web.go`, `internal/effects/web_test.go`
- `cmd/ailang/run_policy.go`, `cmd/ailang/main_run.go`
- `cmd/ailang/run_policy_test.go` and/or `cmd/ailang/run_policy_hardening_test.go`
- generated `docs/docs/reference/env-vars.md`

**Tasks:**

- Add `Policy.WebBackend` with TOML name `web_backend`.
- Declare `AILANG_WEB_BACKEND` in the central config registry and regenerate environment docs.
- Generalize the backend seam for per-backend key names and authorization headers; add a closed
  `SetWebBackend` resolver.
- In the policy path, resolve absent to Ollama without consulting the environment; reject unknown
  names, missing `Net`, and a missing selected-backend host in `net_allow`.
- In the non-policy path, resolve the environment value once before runtime startup.
- Add `web_backend` to the admission JSON beside `ai_provider`.

**Acceptance criteria:**

- [ ] Policy matrix covers absent, explicit Ollama, Gemini, unknown, missing `Net`, and missing host.
- [ ] A negative test proves `AILANG_WEB_BACKEND=gemini` is ignored when a policy omits the field.
- [ ] Non-policy tests prove unset→Ollama, both valid values, and unknown→named refusal.
- [ ] Admission output records the resolved backend and preserves existing fields.
- [ ] Existing policies without `web_backend` continue to load and resolve to Ollama.
- [ ] `make docs-env` leaves generated environment documentation current.

### M3: Implement Gemini backend and security regressions (~420 LOC)

**Goal:** Add fixture-driven Gemini search (and fetch per M1's verdict) behind the seam while
preserving the lending boundary and typed failure behavior.  
**Duration:** 4 hours  
**Dependencies:** M1, M2

**Files:**

- `internal/effects/web_gemini.go` (new)
- `internal/effects/web.go`
- `internal/effects/web_test.go`
- fixtures created by M1

**Tasks:**

- Implement the pinned Gemini `generateContent` request with `google_search` and
  `x-goog-api-key` authentication.
- Decode grounded chunks into the existing `{title,url,content}` contract, preserving response
  order, deduplicating by URI, and enforcing `max` locally.
- Implement `webFetch` with `url_context` if M1 verifies it; otherwise return the designed typed
  unsupported-backend transport error without making a request.
- Cover malformed and incomplete responses with backend-named typed errors.
- Parameterize `TestWeb_SecretNeverLeaks` across Ollama and Gemini, retaining per-backend sentinel,
  carrier sweep, positive control, and header-shape assertions.
- Add an opt-in `TestWeb_LiveGemini` guarded by `AILANG_WEB_LIVE_TEST=1` and `GEMINI_API_KEY`.

**Acceptance criteria:**

- [ ] Fixture tests verify request URL/body/header and exact search-result mapping.
- [ ] Dedupe, response order, `max`, malformed JSON, and absent metadata are covered.
- [ ] Missing Gemini key names `GEMINI_API_KEY` in a typed error without leaking its value.
- [ ] The same secret non-leak test executes its full carrier sweep and positive control for both backends.
- [ ] Gemini requests are rejected by the existing allowlist when its fixed host is not granted.
- [ ] No code path silently crosses from Gemini to Ollama.

### M4: Documentation, examples, and repository gates (~90 LOC)

**Goal:** Document the operator control, verify unchanged public behavior, and close repository
quality gates.  
**Duration:** 2–3 hours  
**Dependencies:** M2, M3

**Files:**

- `docs/docs/guides/agent-tool-policy.md`
- `docs/docs/reference/env-vars.md` (generated)
- `std/web.ail` (comment only)
- current unreleased changelog
- `examples/runnable/web_search.ail` (verification only; change only if documentation is stale)

**Tasks:**

- Document `web_backend` beside `ai_provider`, including policy-only authority, valid values,
  backend hosts, keys, and admission recording.
- Clarify the backend-dependent key in the `std/web.ail` header without changing its API.
- Add the changelog entry and verify the existing runnable example under the default backend.
- Run focused tests, full tests, boundaries, formatting/lint, and simplicity audit.

**Acceptance criteria:**

- [ ] Policy guide contains a valid Ollama and Gemini example and states env isolation under `--policy`.
- [ ] `ailang check examples/runnable/web_search.ail` passes and the file's public contract is unchanged.
- [ ] Focused `go test ./internal/effects ./internal/policy ./internal/config ./cmd/ailang` passes.
- [ ] `make test`, `make lint`, and `make check-boundaries` pass.
- [ ] `make simplicity-audit` has no unexplained regression; the intentional env-var route is documented.
- [ ] `git diff --check` is clean and generated docs are up to date.

## Day-by-day execution

| Day | Work | Exit condition |
|---|---|---|
| 1 | M1 external probes; begin M2 selector/config tests and plumbing | Contract fixture/verdict banked; policy and env resolution tests green |
| 2 | Finish M2; implement M3 request/decoder and typed failures | Both backends pass fixture, allowlist, and malformed-response tests |
| 3 | Parameterized non-leak sweep, opt-in live control, M4 docs and full gates | All acceptance criteria green; no API/effect-row change |

## Estimates and capacity

| Milestone | Estimated LOC | Focused hours |
|---|---:|---:|
| M1 — probes and fixtures | 80 | 2 |
| M2 — selector and admission | 260 | 3 |
| M3 — Gemini backend and security tests | 420 | 4 |
| M4 — docs and gates | 90 | 2–3 |
| **Total** | **850** | **11–12** |

The LOC figure includes tests, JSON fixtures, and generated/documentation changes. Target planning
velocity is approximately 285 LOC/day across the three-day sprint; hours, not raw LOC, are the
more reliable capacity measure for this integration-heavy change.

## Risks and controls

| Risk | Control |
|---|---|
| Grounding unavailable on pinned model | M1 is a hard gate; return to operator rather than shipping a dead selector |
| Gemini response shape drift | Bank real fixtures, fail typed and loud on missing structural fields |
| API key exposed through URL/error | Header-only auth plus parameterized carrier sweep and positive controls |
| Policy/env authority accidentally mixed | Negative test sets a conflicting env value under `--policy` |
| Admission check over-restricts generic Net users | Apply backend-host consistency only when `web_backend` is explicitly set; preserve absent-field compatibility as specified by the design |
| Global backend state leaks between tests/runs | Restore test globals with cleanup and resolve once before execution; run race-sensitive focused tests if practical |
| New config route trips simplicity gate | Generate env docs and record the intentional route against this approved design |

## Handoff and approval

This plan creates execution infrastructure only. Per repository routing, implementation begins only
after the user explicitly says **“execute sprint”**, at which point `sprint-executor` owns TDD,
progress updates, and the final `sprint-evaluator` handoff.
