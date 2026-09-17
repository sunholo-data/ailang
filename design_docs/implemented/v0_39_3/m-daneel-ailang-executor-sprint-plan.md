# Sprint Plan: M-DANEEL-AILANG-EXECUTOR

## Summary

Deliver Daneel's policy-bounded `ailang_only` executor with Ollama web search as a secret-lending `Net` capability. The plan replaces the superseded Gemini-grounding milestone: the executor model remains `z-ai/glm-5.3-flash` on OpenRouter, while `std/web` calls Ollama's web APIs without exposing `OLLAMA_API_KEY` to AILANG programs.

**Duration:** 4 engineering days across three repositories, plus release/deployment propagation time  
**Dependencies:** approved [M-DANEEL-AILANG-EXECUTOR design](m-daneel-ailang-executor.md), PR #1240; v0.39.1 lane tooling; access to `ailang-multivac` Terraform and the Daneel repository  
**Risk Level:** Medium — the core API is small, but release ordering and Cloud Run secret delivery are load-bearing

## Correction Incorporated

The original proposed “Gemini Google-Search grounding” milestone is removed. The measured 2026-09-16 control proved that `POST https://ollama.com/api/web_search` and `/api/web_fetch` work with the existing `OLLAMA_API_KEY`, and an AILANG program admitted under `[IO, Net, Env]` plus `net_allow = ["ollama.com"]` received HTTP 200 and three real results through `std/net.httpRequest` with no compiler change.

The production boundary is intentionally narrower than that probe:

- Search uses Ollama; Gemini grounding is out of scope.
- The executor remains `z-ai/glm-5.3-flash` through OpenRouter. Do not route pi through `ollama/...:cloud`; that route drops the system role and loses the teaching prompt.
- New Go-backed `std/web` operations read `OLLAMA_API_KEY` inside the handler. The AILANG program receives parsed results, never the key, and needs effect `{Net}` only.
- The final Daneel policy excludes `Env`. It retains the other ratified executor caps needed by non-search work, including narrowed read-only `Process`, and adds `ollama.com` to `net_allow`.
- Daneel's first package `ext/search` exposes an `Authority` row over `[Net]` and wraps `std/web`.

## Current Status and Capacity

- The approved design doc exists on PR #1240 but is not yet in this worktree's base commit.
- AILANG v0.39.1 is released at the current base (`0dfecf89`). The prior design's dependency on `ailang_cli`, `cli_allow`, Z3, and scratch exclusion is satisfied.
- The repository has only the release commit in its available seven-day history, so there is no defensible LOC/day measurement. Estimates below are seam-based with roughly 25% integration buffer.
- Existing reusable seams: `internal/policy.Policy.AIProvider`, `cmd/ailang/run_policy.go`, `internal/effects/net.go`, `internal/builtins/net.go`, `std/net.ail`, and generated stdlib interface/golden tests.

## Ordered Milestones

The dependency chain is strict:

`M1 ai_provider binding → M2 std/web → M3 release → M4 multivac agent/policy/template → M5 Daneel host dispatch + ext/search → M6 end-to-end closeout`

### M1: Bind `ai_provider` from policy ✅ (2026-09-16)

**Goal:** Make the policy's model pin enforceable before any fleet configuration depends on it.  
**Estimated:** 120 LOC implementation + 180 LOC tests/docs = 300 LOC  
**Duration:** 0.5 day

**Example files to update:**

- `cmd/ailang/run_policy.go`
- `cmd/ailang/main_run.go`
- `cmd/ailang/run_policy_test.go` or focused policy test files
- `docs/docs/guides/agent-tool-policy.md`

**Tasks and acceptance criteria:**

- [x] Resolve `pol.AIProvider` into the same handler setup used by `--ai`; preserve `ai_provider = "stub"` as an offline test route.
- [x] Refuse `--ai` with `--policy`, AI authority without `ai_provider`, and `ai_provider` without the `AI` cap, with stable named reasons and exit code 1.
- [x] Prove a policy-admitted AI program runs against the stub without a `--ai` flag.
- [x] Run focused Go tests, `make test-core`, `make lint`, and `make check-boundaries`.

**Risk:** Flag precedence could accidentally widen authority. **Mitigation:** all cross-product cases are table-driven and the policy remains the single source under `--policy`.

### M2: Add secret-safe `std/web` ✅ (2026-09-16)

**Goal:** Add the smallest typed web-search API whose only program-visible effect is `{Net}`.  
**Estimated:** 300 LOC implementation + 420 LOC tests/fixtures/examples = 720 LOC  
**Duration:** 1.25 days

**Public contract:**

```text
webSearch(query: string, max: int)
  -> Result[list[{title: string, url: string, content: string}], NetError] ! {Net}

webFetch(url: string)
  -> Result[{title: string, content: string, links: list[string]}, NetError] ! {Net}
```

The exact AILANG type syntax must be taken from `ailang prompt` during execution; these record shapes and effect rows are the acceptance contract.

**Example files to create or update:**

- `std/web.ail` (preferred) or a clearly isolated `std/net` addition if module loading proves a blocker
- `internal/builtins/web.go` and registration metadata
- `internal/effects/web.go` (or a focused web section following the existing Net handler boundary)
- `internal/effects/testdata/ollama_web_search.json`
- `internal/effects/testdata/ollama_web_fetch.json`
- focused builtin, effect, type-golden, stdlib-interface, and fixture tests
- `examples/runnable/web_search.ail` plus its manifest/golden

**Tasks and acceptance criteria:**

- [x] Handler POSTs the documented JSON payloads to fixed HTTPS endpoints on `ollama.com`, sets authentication from `OLLAMA_API_KEY`, parses typed response records, and returns existing `NetError` variants on missing key, invalid input, non-2xx, transport, malformed JSON, and response-limit failures.
- [x] `OLLAMA_API_KEY` is read only in Go. It never appears in function arguments, AILANG values, traces, errors, fixture output, or example output; add a sentinel-secret non-leak assertion.
- [x] Both builtins declare effect `Net` only. A program with `{Net}` and no `{Env}` type-checks and runs against the recorded transport fixture; removing `Net` fails admission/type checking.
- [x] `max` has a documented positive upper bound and invalid values fail before the request. Response/body limits reuse or tighten existing Net limits.
- [x] Recorded-fixture tests cover search, fetch, malformed payload, non-2xx, missing key, and secret redaction deterministically.
- [x] A live positive-control test is opt-in and skipped unless `OLLAMA_API_KEY` and an explicit live-test flag are present; it asserts status-equivalent success, at least one result, and never blocks the normal suite.
- [x] Freeze stdlib interfaces/goldens, type-check and run the example, then run `make test`, `make lint`, and `make check-boundaries`.

**Risk:** A generic HTTP handler may tempt callers to smuggle arbitrary domains. **Mitigation:** endpoints are fixed in Go; `webFetch(url)` fetches through Ollama's `/api/web_fetch`, not directly from the supplied URL.

### M3: Release the binary contract ✅ (2026-09-16)

**Goal:** Publish the AILANG version containing M1 and M2 before any multivac config uses their fields or symbols.  
**Estimated:** 40 LOC release notes/metadata  
**Duration:** 0.25 day plus CI

**Example files to update:** `std/VERSION`, release changelog, generated/frozen stdlib artifacts as required by the release workflow.

**Acceptance criteria:**

- [x] Release notes name `ai_provider` enforcement, `std/web`, `{Net}`-only authority, and the secret non-exposure boundary.
- [x] Published binary reports the new version and `ailang iface std/web` exposes both exact signatures.
- [x] Container/image used by multivac resolves that version before M4 begins; unknown policy fields or missing modules are a hard stop.

### M4: Multivac agent, policy, template, and secret binding ✅ (2026-09-16)

**Goal:** Deploy `daneel-executor` with the corrected model route and key delivery. This is the M-registry milestone and is owned in the `ailang-multivac` repository.  
**Estimated:** 140 LOC config/template + 180 LOC Terraform/tests = 320 LOC  
**Duration:** 0.75 day

**Expected files (verify actual names in that repository):**

- agent registry/config YAML entry for `daneel-executor`
- the executor policy TOML
- Daneel task template
- Cloud Run Job Terraform/module variables and Secret Manager env bindings
- config/terraform validation fixtures

**Tasks and acceptance criteria:**

- [x] Register `daneel-executor` as `ailang_only`, `acknowledge_only: true`, with the Daneel workspace/template and released AILANG image.
- [x] Pin executor inference to `z-ai/glm-5.3-flash` on OpenRouter. Assert the rendered pi command/config does not use `ollama/...:cloud`.
- [x] Preserve the ratified policy caps needed by the executor, narrowed `process_allow = ["git:status", "git:diff", "git:log"]`, exclude `Env` and `Msg`, and add exactly `ollama.com` to `net_allow` for search.
- [x] Bind the existing Secret Manager value for `OLLAMA_API_KEY` into the pi Cloud Run Job environment. Verify Terraform plan/rendered Job contains the secret reference and env name, not secret material.
- [x] Run a container probe proving `std/web.webSearch` succeeds with `{Net}` and no `{Env}`, while raw `Env` use, an unlisted domain, and disallowed process commands are refused.
- [x] Assert banked `policy_digest` is non-empty and clean answer tasks complete rather than becoming `no_changes`.

**Risk:** Terraform may bind secrets only to coordinator services, not ephemeral pi Jobs. **Mitigation:** inspect the rendered Job spec and execute one real Job positive control; configuration inspection alone is insufficient.

### M5: Daneel host dispatch and `ext/search` ✅ (2026-09-16)

**Goal:** Give Daneel its first policy-bounded search package and consume executor completions safely.  
**Estimated:** 220 LOC implementation + 260 LOC tests/smokes = 480 LOC  
**Duration:** 0.75 day

**Expected files in the Daneel repository:**

- `ext/search/ailang.toml`
- `ext/search/register.ail`
- `ext/search/_smoke.ail`
- host dispatch/completion-consumer modules
- capability template/teaching docs and tests

**Tasks and acceptance criteria:**

- [x] Create `ext/search` as a path-dependent package wrapping `std/web`; its exported authority row is exactly `[Net]`, never `[Env]` or `[AI]`.
- [x] Type-check and run recorded smokes for search and fetch, including structured error propagation.
- [x] Host validates the typed Request before dispatch, targets only `daneel-executor`, correlates the completion, consumes the bounded `summary`/`ANSWER:` contract, and maps malformed requests or executor `BLOCKED:` results to structured blocked outcomes.
- [x] Search package cannot select a model, provide a key, widen domains, or invoke raw host dispatch directly.

### M6: End-to-end boundary proof and closeout ✅ (2026-09-16)

**Goal:** Prove the entire lending path in production and record falsifiable evidence.  
**Estimated:** 110 LOC tests/docs/report  
**Duration:** 0.5 day

**Acceptance criteria:**

- [x] One live Daneel request dispatches to `daneel-executor`, produces at least one real Ollama search result, and returns a correlated structured answer to Daneel.
- [x] Banked evidence records OpenRouter `z-ai/glm-5.3-flash` as the executor model, a non-empty policy digest, completed status, and no secret in transcript/summary/artifacts.
- [x] Negative controls prove `{Env}`, an unlisted network host, and a forbidden process subcommand fail under named policy categories.
- [x] The live `std/web` positive control passes when explicitly enabled with the key; recorded fixtures remain the deterministic CI gate.
- [x] Update AILANG, multivac, and Daneel docs/changelogs; record exact release/image, Terraform plan/apply evidence, Job execution, task/correlation IDs, and the three search result URLs in an implementation report without recording credentials.

## Day-by-Day Plan

| Day | Work |
|---|---|
| 1 | M1 red/green policy tests and implementation; begin M2 types, handler boundary, and fixtures |
| 2 | Finish M2 parsing, secret non-leak tests, example, goldens, live gated control; full core validation |
| 3 | M3 release and image verification; M4 multivac registry/policy/template/Terraform secret binding and Job probe |
| 4 | M5 Daneel host + `ext/search`; M6 live round trip, negative controls, evidence and documentation |

## Success Metrics

- Six populated milestones complete in the specified dependency order.
- `webSearch` and `webFetch` expose `{Net}` only and run without program-visible `Env`.
- Recorded fixtures gate CI; the opt-in live control returns at least one result with a configured key.
- No `OLLAMA_API_KEY` value appears in program arguments, output, errors, traces, summaries, fixtures, or committed files.
- Multivac's rendered pi Job has a Secret Manager-backed `OLLAMA_API_KEY` env binding and the policy allows `ollama.com` while excluding `Env`.
- Executor model evidence is GLM on OpenRouter, never the Ollama cloud pi route.
- `ext/search` declares an Authority row exactly over `[Net]`.
- Full tests, lint, boundary checks, package smokes, Terraform validation, and the production round trip pass.

## Explicit Non-Goals

- Gemini Google Search grounding or new Gemini provider tool blocks.
- Changing the executor model to an Ollama cloud route.
- Exposing `OLLAMA_API_KEY` through `std/env`, function parameters, records, or traces.
- A generic arbitrary-provider search abstraction in v1.
- Granting Daneel's search package `AI`, `Env`, `Process`, or arbitrary domains.

## Execution Gate

This plan creates no implementation authorization. Per repository policy, execution begins only after human approval of this revised plan and an explicit “execute sprint”; use the `sprint-executor` workflow at that point.

