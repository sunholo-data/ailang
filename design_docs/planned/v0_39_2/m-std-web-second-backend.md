# M-STD-WEB-SECOND-BACKEND: Gemini Grounding as an Operator-Selected Second Backend Behind std/web

**Status**: Planned
**Target**: v0.39.2
**Priority**: P1 (Medium) — single-vendor dependency on a lane Mark has flagged ("ollama may dry up", 2026-09-16); no lane is red today
**Estimated**: ~2 days (8–10 hours across 5 phases)
**Dependencies**: M-DANEEL-AILANG-EXECUTOR M2 (std/web, landed v0.39.x — dev 6377683dc). GEMINI_API_KEY already declared in `internal/config` (V4). No other code dependencies.

**Task brief**: coordinator inbox `inbox_1789582783331_40c6a750` + amendment `inbox_1789582800669_18d0125a`
(the amendment restored the selector text lost to shell quoting — the selector is a policy TOML field named
`web_backend`, values `"ollama"` | `"gemini"`; see D1–D4 below).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is runtime/effect-infrastructure, not a language change: std/web's public contract, types and
effect row are untouched. Most axioms are genuinely neutral; scoring them 0 is the honest answer, not
an evasion (same posture as M-LYCEUM-PROVIDER).

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | The Net effect is already explicitly nondeterministic; which provider serves it is operator-pinned authority, the exact shape of `ai_provider`. No new class of nondeterminism reaches the program. |
| A2: Replayability | 0 | Trace shape unchanged. The backend identity is banked once on the admission line (V9), not per-operation. |
| A3: Effect Legibility | 0 | std/web stays `{Net}`-only; no effect row, builtin signature or std type changes. |
| A4: Explicit Authority | +1 | The backend stops being an ambient single-vendor constant and becomes explicit operator authority: a policy field the operator picks per lane, refused at admission when unknown (D4), with the environment deliberately NOT consulted under `--policy` (D2) — the same "authority comes from the file and nowhere else" doctrine the run path already states in code. |
| A5: Bounded Verification | 0 | No verification-surface change. |
| A6: Safe Concurrency | 0 | One HTTP request per webSearch/webFetch call, unchanged. |
| A7: Machines First | 0 | Refusal errors name the field, the bad value and the valid set — machine-decidable, same shape as the existing `--policy` refusals. |
| A8: Minimal Syntax | 0 | No AILANG syntax change. One TOML field + one env var (a new env-var route: it must be declared in `internal/config` and land in the generated `env-vars.md`, V13-class surface — noted for `make simplicity-audit`). |
| A9: Cost Visibility | +1 | The two backends have different quota postures (Ollama subscription quota vs Gemini grounding quota/pricing — exact terms UNMEASURED, V18). Making the choice explicit and recording it on the admission line is what makes per-backend spend attributable at all. |
| A10: Composability | +1 | The second provider lands entirely behind the existing `webBackend` seam (V1): zero changes to builtins, std/web.ail's contract, or the effect row. |
| A11: Structured Failure | +1 | An operation the selected backend cannot serve is a typed `Err` naming the backend (D7) — never a silent cross-backend fallback; an unknown selector name is a loud admission-time refusal (D4). |
| A12: System Boundary | +1 | The Gemini host is explicit in `net_allow` (enforced by the existing allowlist path, V3), the key is read in Go only, and Gemini auth uses a header, not a `?key=` query — the query form would leak the key into program-visible `Err` text through `*url.Error` (V11, D6). |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced — provider selection is operator-pinned, program semantics unchanged
- [x] A3 (Effects): No hidden side effects — still `{Net}` only, key never program-visible
- [x] A4 (Authority): No ambient access granted — under `--policy` the env var is deliberately not read (D2); outside it the env var is the documented default, not a silent one
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Quorum trigger check

**Trigger 4 fires** (external-system premise): the Gemini grounding response shape, `url_context`
availability on the pinned model, and the grounding quota/pricing posture are claims about an
external API contract that cannot be verified in this checkout (V16–V18). Triggers 1–3 do not fire
(no design-freeze item needs a human ruling — D1–D4 were ruled by the operator in the task brief
and amendment; nothing overrides shared machinery — the seam, the security path and the non-leak
test are reused, not forked; no cost/KPI/banked-schema surface beyond one additive JSON field).

**Recommended:** run `ailang design-quorum` (or at minimum one `ailang design-review`) on this doc
before sprint planning — reviewers `gpt5-6-sol`, `gemini-3-1-pro`, `oc-glm-5-2` as the parent doc
used. The external premises themselves are sprint Phase 0 probes (below), recorded in the
implementation report before any wiring.

## Problem Statement

std/web gives the ailang_only lane web search and page fetch behind a lending boundary: the
program holds only `{Net}`, never the key, never a free host choice (V1–V3). But the seam has
exactly ONE implementation — Ollama Cloud (`https://ollama.com`, `OLLAMA_API_KEY`). Every research
answer the lane produces is a function of one vendor's availability and quota. Mark, 2026-09-16:
"ollama may dry up". If it does, the lane's search capability dies with it and there is no
operator lever to move to another provider short of a code change.

**Current State:**

- `internal/effects/web.go` defines the `webBackend` seam (name / searchRequest / decodeSearch /
  fetchRequest / decodeFetch / apiKey, V1) with one implementation, `ollamaWebBackend`, selected by
  a package variable `activeWebBackend` — no runtime selection path exists.
- The backend identity is invisible to the operator and to the banked transcript: the admission
  line records `ai_provider` but nothing about the web backend (V9).
- The non-leak invariant (key never in a value, error or stringified result) is tested for the ONE
  backend with a sentinel key, ≥12 carriers and a positive control (V12). A second backend added
  outside that test would be an untested lending boundary.

**Impact:**

- **Who**: the ailang_only lane (Daneel executor) and any future operator running std/web programs
  under a policy.
- **How significant**: a single-vendor availability failure would take out the lane's research
  capability entirely; today the only remediation is a code change, which is exactly the wrong
  granularity for an operational choice. The fix is a selector, not a rewrite — the seam was built
  for this (web.go: "The backend is a seam (webBackend) so a second provider can sit behind the
  same typed contract without touching the builtins or std/web").

## Goals

**Primary Goal:** Let the OPERATOR pick the std/web backend per lane — a policy TOML field
`web_backend` with values `"ollama"` or `"gemini"` — so the lane survives an Ollama availability
failure with a policy edit, not a code change, and add Gemini grounding as the second
implementation behind the existing seam.

**Success Metrics:**

- A lane policy with `web_backend = "gemini"` completes a webSearch end-to-end against
  `generativelanguage.googleapis.com` with **zero** changes to `std/web.ail`'s exported types, the
  builtins, or the effect row.
- The admission line records `web_backend` beside `ai_provider` (V9's JSON line gains one field).
- An unknown `web_backend` value is refused at admission with a loud named error — no silent
  fallback to ollama (Critical Principle 2).
- `TestWeb_SecretNeverLeaks` covers BOTH backends in the same test (per-backend sentinel, ≥12
  carriers each, positive control each) — not a copy.
- `net_allow` without the selected backend's host still yields a typed `DisallowedHost` Err — the
  operator's allowlist stays the authority (V3).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1: Selector = policy TOML string field `web_backend`, values `"ollama"` \| `"gemini"`** | The operator amendment fixes the selector shape: a closed string enum in the policy, picked per lane — the lending boundary is "which provider may this program's {Net} reach", an operator decision, not a program one. A free-form value (model names, URLs) would re-open host/key choice to the policy author and break the closed-set refusal. | **human (ruled in the amendment)** | design | med |
| **D2: Under `--policy`, the policy is the sole authority — absent `web_backend` means `"ollama"`, and `AILANG_WEB_BACKEND` is deliberately NOT read** | The run path's existing doctrine, stated in code: "with --policy the authority comes from the file and nowhere else". Letting the env var leak into a policy run would let the machine's environment silently override the operator's lane config. Defaulting absent→ollama keeps every existing policy file working unchanged. | agent (per documented doctrine) | design | low |
| **D3: Outside `--policy`, the default comes from `AILANG_WEB_BACKEND` (declared in `internal/config`), unset → `"ollama"`; an unknown value is a loud error, never a fallback** | Plain `ailang run` has no policy; the env var is the documented machine-level default. Unknown value → fail loudly naming the valid set (Critical Principle 2), since a typo'd backend name that silently meant "ollama" would misattribute every search. | agent | design | low |
| **D4: Unknown value refused at admission; admission line records `web_backend` beside `ai_provider`** | Mirrors the `ai_provider` validation block exactly (V10): refusals by name with the valid set, exit 1, before any program executes; the JSON admission line (read by ailang-exec.ts, additive-field tolerant, V9) gains `web_backend` so the transcript banks the choice. Consistency checks mirror the existing pattern: `web_backend` set without `Net` in `allowed_caps` → refuse; backend host missing from `net_allow` when `Net` is allowed → refuse naming the host. | agent (per existing pattern) | design | med |
| **D5: Gemini client is minimal and lives in `internal/effects/web_gemini.go`, NOT a reuse of `internal/ai/gemini`** | `internal/ai/gemini/types.go`'s `toolBlock` carries `functionDeclarations` only (V7) — grounding needs `tools:[{google_search:{}}]`, a shape that package does not model; and that package is shaped around the AI effect (models, streaming, tool dispatch), not raw grounding. Importing internal/ai from effects is boundary-legal (V8) but buys nothing here; a ~150-LOC self-contained backend behind the seam is smaller than widening the AI-effect client. | agent | design | med |
| **D6: Gemini auth via `x-goog-api-key` header — never a `?key=` query parameter** | `client.Do` failures surface as `*url.Error`, whose `Error()` includes the full URL, and `transportMessage` passes it straight through into a program-visible `Err` (V11). The existing in-repo Gemini client uses `?key=` (V15) — fine for the AI effect's error path, disqualifying here. Header auth keeps the key out of every URL-shaped carrier. | agent | design | low |
| **D7: `webFetch` on Gemini: implement via `url_context` if Phase 0 verifies it on the pinned model; otherwise a typed `Err(Transport("webFetch: not supported by the gemini backend"))` — never a silent fallback to Ollama** | The amendment and task brief are explicit: no silent cross-backend fallback. A gemini-backend lane whose fetch quietly went to ollama.com would (a) leak the lane's queries to a vendor the operator deselected and (b) still require OLLAMA_API_KEY, failing confusingly when it is unset. Refusal is typed, named, and program-checkable. | agent (Phase 0 decides) | design | med |
| **D8: Transcript banks the backend NAME on the admission line; nothing backend-shaped enters program-visible values** | Cost/quota attribution needs per-backend rows; the values contract `{title,url,content}` must stay identical across backends or programs become provider-sensitive. The admission line already exists and is already consumed (V9) — one field, no new banked schema. | agent (per task brief) | design | low |
| **D9: The Gemini grounding MODEL is pinned in the effects package (constant + test-overridable var, mirroring `ollamaWebBaseURL`), not a policy/env field** | The amendment's selector is exactly one field with a closed set — adding `web_backend_model` would widen the policy surface for a choice that only matters while grounding exists and that the operator did not ask for. `ollamaWebBaseURL` set the precedent: fixed endpoint, test flips the var, deliberately no env var (V1). Phase 0 verifies grounding availability on the pinned model. | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — ruled by the operator in the amendment (inbox_1789582800669_18d0125a): policy field
  `web_backend`, values `"ollama"` | `"gemini"`; env var default outside `--policy`; unknown value
  refused at admission; admission line records `web_backend` beside `ai_provider`.
- [ ] D7's external premise — `url_context` availability on the pinned model: resolved by the
  Phase 0 probe, before any webFetch wiring (agent-resolvable; BLOCK and return to the operator
  only if Google Search grounding itself is unavailable, which would void the whole feature).

No other freeze items: D2–D6, D8, D9 follow documented in-repo patterns and are agent-resolvable.

## Solution Design

### Overview

Add a second `webBackend` implementation (Gemini Google Search grounding) behind the existing seam
in `internal/effects`, and make the active backend a RESOLVED setting instead of a compile-time
constant: the policy's `web_backend` field under `--policy`, the `AILANG_WEB_BACKEND` env var
outside it, default `"ollama"`. Resolution happens once, in `cmd/ailang`, before the runtime
starts — the same place `ai_provider` is resolved. Unknown names are refused loudly at admission;
the admission line records the resolved backend.

### Architecture

**Components:**

1. **The seam extension** (`internal/effects/web.go`, ~25 LOC): `webBackend` gains two methods —
   `keyEnv() string` (for the missing-key error: today's message hardcodes
   `config.EnvOllamaAPIKey`, V14) and `authHeader(key) (name, value)` (ollama: `Authorization` /
   `Bearer <key>`; gemini: `x-goog-api-key` / `<key>`, D6). `webPost` uses them instead of the
   hardcoded Bearer construction. A `SetWebBackend(name string) error` resolver sets
   `activeWebBackend`; unknown name → error (no fallback). `activeWebBackend` stays a package var
   so tests keep pointing it at fixture servers.

2. **The selector plumbing** (~35 LOC):
   - `internal/policy/policy.go`: `WebBackend string \`toml:"web_backend"\`` + field-semantics
     comment. `policy.Load` already refuses unknown TOML fields (V13), so a typo'd field name
     fails loudly today; the VALUE check is admission-time (D4).
   - `internal/config` (Compiler-and-runtime area, beside `EnvFSSandbox`): `EnvWebBackend =
     "AILANG_WEB_BACKEND"` + `Var` row + `WebBackend()` getter returning "" when unset.
   - `cmd/ailang/run_policy.go`: validation block additions mirroring the `ai_provider` rows
     (V10) — unknown value (valid set named in the error); set without `Net` in `allowed_caps`;
     `Net` allowed but the selected backend's host absent from `net_allow` (error names the host).
     Resolution: `resolved.webBackend = pol.WebBackend` (absent → `"ollama"`, D2 — the env var is
     not read on this path). The admission-line JSON gains `"web_backend"`.
   - `cmd/ailang/main_run.go`: on the NON-policy path only, resolve
     `config.WebBackend()` → `effects.SetWebBackend` (unset → ollama; unknown → loud error, D3).

3. **The Gemini backend** (`internal/effects/web_gemini.go`, ~150–200 LOC, new): `geminiWebBackend`
   implementing the seam. Fixed base `https://generativelanguage.googleapis.com/v1beta` (matches
   the in-repo Gemini client, V15), fixed pinned model (D9, test-overridable var).
   - `searchRequest`: POST `models/{model}:generateContent` with the query as the prompt and
     `tools: [{"google_search": {}}]` (external premise V16 — Phase 0 verifies the shape).
   - `decodeSearch`: map `candidates[0].groundingMetadata.groundingChunks[].web` (`uri`, `title`)
     onto `{title, url, content}`; `content` from the first `groundingSupports[]` entry citing
     that chunk (`segment_indices`), else `""`. Dedupe by `uri`, preserve response order,
     truncate to `max` (Gemini has no max parameter — the backend enforces the contract's
     1..10 bound, which stays input validation unchanged).
   - `fetchRequest`/`decodeFetch`: via the `url_context` tool if Phase 0 confirms it (D7);
     `links` may be `[]` if the response carries no link set — contract-safe, `FetchResult.links`
     is `list[string]` and the ollama decoder already normalises nil→empty (V1). If Phase 0
     refutes it: `webFetch` returns `Err(Transport("webFetch: not supported by the gemini backend"))`
     before any request — typed, named, no fallback.
   - Auth: `x-goog-api-key` header (D6). Key from `config.GeminiAPIKey()` (declared, V4).

4. **The tests** (`internal/effects/web_test.go`, ~120 LOC + fixtures): see Testing Strategy.

### Implementation Plan

**Phase 0: External premise probes** (~2 hours) — before any wiring
- [ ] Live probe: `generateContent` with `tools:[{"google_search":{}}]` against the pinned model
      with GEMINI_API_KEY; record the exact `groundingMetadata` shape (chunk/support field names)
      as a recorded fixture `testdata/gemini_web_search.json`.
- [ ] Live probe: `url_context` tool on the same model; record the response shape or refute.
- [ ] Read the Gemini API reference + pricing page for the grounding quota/pricing posture (V18);
      record the numbers in the implementation report.
- [ ] If grounding is unavailable on the pinned model: BLOCK — return to the operator (this
      voids the feature, not just D7).

**Phase 1: Selector plumbing** (~3 hours)
- [ ] `policy.WebBackend` field; `config.EnvWebBackend` + getter + `Var` row; `make docs-env`
      regenerates `env-vars.md`.
- [ ] `effects.SetWebBackend` + seam extension (`keyEnv`, `authHeader`); generalise the
      missing-key message (V14) and header construction.
- [ ] `run_policy.go`: the three validation rows (D4) + `web_backend` on the admission line;
      `main_run.go`: the non-policy env default.
- [ ] Unit tests: unknown value refused (both paths); absent→ollama; env var not read under
      `--policy` (negative test).

**Phase 2: Gemini backend** (~4 hours)
- [ ] `web_gemini.go`: search via grounding (fixture-driven), fetch per D7's Phase 0 verdict.
- [ ] Recorded-fixture tests per backend: shape mapping, dedupe, truncation to `max`,
      malformed-response → `Err(Transport(...))` naming the backend.
- [ ] Allowlist negative test: `net_allow` without `generativelanguage.googleapis.com` →
      `DisallowedHost` (the fixed endpoint is subject to the operator's allowlist, V3).

**Phase 3: Non-leak parameterisation** (~2 hours)
- [ ] `TestWeb_SecretNeverLeaks` covers both backends in the SAME test: loop over the backend
      set, per-backend sentinel (OLLAMA_API_KEY / GEMINI_API_KEY), per-backend base-URL var,
      fixture servers requiring the correct auth header shape (Bearer vs x-goog-api-key — the
      header difference is itself under test), ≥12 carriers each, positive control each (V12
      discipline).
- [ ] Opt-in live controls mirroring `TestWeb_LiveOllama`: `TestWeb_LiveGemini`
      (`AILANG_WEB_LIVE_TEST=1` + `GEMINI_API_KEY`).

**Phase 4: Docs + examples** (~1 hour)
- [ ] `std/web.ail` header comment: key/backend are backend-dependent (contract unchanged);
      `docs/docs/guides/agent-tool-policy.md`: `web_backend` row beside `ai_provider` (the guide
      already documents the admission-line recording, V9-docs).
- [ ] `examples/runnable/web_search.ail` unchanged — assert it still runs (V19); CHANGELOG entry.

### Files to Modify/Create

**New files:**
- `internal/effects/web_gemini.go` — the Gemini backend behind the seam, ~150–200 LOC
- `internal/effects/testdata/gemini_web_search.json` (+ `gemini_web_fetch.json` if D7 lands fetch) — recorded fixtures from the Phase 0 probes

**Modified files:**
- `internal/policy/policy.go` — `WebBackend` field + comment, ~4 LOC
- `internal/config/compiler.go` — `EnvWebBackend`, `Var` row, getter, ~5 LOC
- `internal/effects/web.go` — seam methods, `SetWebBackend`, header/message generalisation, ~25 LOC
- `cmd/ailang/run_policy.go` — validation rows + admission-line field + resolution, ~25 LOC
- `cmd/ailang/main_run.go` — non-policy env default, ~6 LOC
- `internal/effects/web_test.go` — parameterised non-leak + fixture/allowlist/live tests, ~120 LOC
- `std/web.ail` (comment only), `docs/docs/guides/agent-tool-policy.md`, `docs/docs/reference/env-vars.md` (regenerated)

**Verified no change needed:**
- `cmd/ailang/pi_assets/ailang-exec.ts` — parses the `policy: {...}` line as JSON via regex; an
  added field passes through (V9).
- `internal/builtins/web.go`, `std/web.ail` types/signatures, `examples/runnable/web_search.ail`.

## Conflict Surface

This change touches `internal/effects/` (and the policy/admission surface), so per the
design-doc-creator skill the conflict surface is enumerated:

**1. Positions this change extends:**
- The `webBackend` seam's implementation set (one package var `activeWebBackend` and its methods).
- The policy struct's closed field set (beside `ai_provider`, `net_allow`, `process_allow`).
- The admission-line JSON field set (beside `ai_provider`).
- The `internal/config` env-var registry (a new `AILANG_*` route — simplicity-gate surface).
- The `net_allow` hostname set a policy may meaningfully list (`ollama.com` today;
  `generativelanguage.googleapis.com` added).

**2. Other valid constructs already living in those positions:**
- `ai_provider` validation rows and their exact refusal wording/format (the pattern being
  mirrored, V10); `AIProvider == "stub"` special-casing in resolution.
- The `runPolicyWidening` flag-refusal block — no new widening flag is introduced (the backend is
  never a CLI flag; D2/D3 keep it policy-or-env).
- `net_allow` entries for other Net users (std/net, AI providers) — the gemini host joins a list
  that already legitimately names generativelanguage.googleapis.com for the AI effect (the daneel
  lane policy already grants it, per M-DANEEL-AILANG-EXECUTOR D2).
- `webPost`'s header list — extended per-backend, must keep Content-Type for both.
- The `Transport` NetError constructor's carrier set (V11) — the unsupported-op message and the
  key-leak carriers ride existing paths; no new constructor is added to std/net.

**3. How the system disambiguates:**
- No parser-level ambiguity: the policy field is a closed string enum validated at admission, not
  syntax. The env var is read exactly once, on the non-policy path (a path-level disambiguation:
  `--policy` present vs absent — the same fork that already governs `--caps`/`--ai` refusal).
- Backend name vs model name: `web_backend` values are backend names, never model names; the
  model is pinned in code (D9), so there is no name-space collision with `ai_provider` values
  (e.g. a policy may legitimately say `ai_provider = "gemini-2.5-flash"` AND
  `web_backend = "ollama"`).

**4. Existing programs/tests that MUST still work (fixtures):**
- `examples/runnable/web_search.ail` (V19) — the public contract is unchanged.
- `TestWeb_MissingKeyIsATypedErrThatNamesTheVariable`, `TestWeb_Non2xxAndMalformedAreTransportErrs`,
  `TestWeb_InvalidInputFailsBeforeAnyRequest`, `TestWeb_RequiresNetCapAndHonoursTheAllowlist` —
  all keep passing for ollama (default resolution unchanged, absent field → ollama).
- `internal/policy/policy_test.go`'s policy fixture (`ai_provider = "stub"`) — loads unchanged.
- Every existing lane policy file on disk: absent `web_backend` → `"ollama"`, byte-identical
  behaviour; the admission line gains a field, which ailang-exec.ts tolerates (V9).

**5. Deliberate changes (intentional, small incompatibilities):**
- The missing-key error text for the ollama backend changes shape only if wording is unified
  (the env var name it names stays `OLLAMA_API_KEY` via `keyEnv()`); the admission line is a
  superset (additive JSON field).
- A policy that sets `web_backend` without granting `Net`, or grants `Net` without listing the
  selected backend's host, now REFUSES where it previously ran (it never could have used std/web
  successfully — the refusal surfaces a latent misconfiguration loudly; this is the point).

## Examples

### Example 1: The lane policy (operator side)

**Before:**
```toml
allowed_caps = ["IO", "FS", "Clock", "Net", "AI"]
net_allow    = ["ollama.com", "generativelanguage.googleapis.com"]
ai_provider  = "gemini-2.5-flash"   # std/web silently always used ollama
```

**After (Gemini selected for search):**
```toml
allowed_caps = ["IO", "FS", "Clock", "Net", "AI"]
net_allow    = ["generativelanguage.googleapis.com"]
ai_provider  = "gemini-2.5-flash"
web_backend  = "gemini"              # NEW: the operator picks, per lane
```

Admission (stderr, read by the pi tool):
```
policy: {"ok":true,"policy":"agent-policy.toml","policy_digest":"…","caps":["IO","FS","Clock","Net","AI"],"fs_sandbox":"…","net_allow":["generativelanguage.googleapis.com"],"process_allow":null,"ai_provider":"gemini-2.5-flash","web_backend":"gemini","decision":{"ok":true}}
```

### Example 2: Refusal shapes (machine-checkable)

```
Error: --policy: unknown web_backend "gemmini" (valid: ollama, gemini)
Error: --policy: web_backend is set but Net is not in allowed_caps
Error: --policy: web_backend "gemini" but its host generativelanguage.googleapis.com is not in net_allow
```

### Example 3: webFetch on a backend without fetch support (if D7 refutes url_context)

```ail
match webFetch("https://example.com/") {
  Ok(page)  => ...   -- unreachable on this backend
  Err(e)    => ...   -- Err(Transport("webFetch: not supported by the gemini backend"))
}
```

## Success Criteria

- [ ] A gemini-backend lane policy completes webSearch end-to-end (Phase 0 live probe banked as
      fixture; `TestWeb_LiveGemini` opt-in control passes).
- [ ] `web_backend = "gemini"` appears on the admission line; `absent → "ollama"` for every
      existing policy (regression: policy_test fixtures load unchanged).
- [ ] Unknown `web_backend` value refused with the valid set named, on both `--policy` and env
      paths; no silent fallback anywhere (negative tests).
- [ ] `AILANG_WEB_BACKEND` is NOT consulted under `--policy` (negative test).
- [ ] `TestWeb_SecretNeverLeaks` runs its full carrier sweep per backend with per-backend
      sentinels and positive controls; two key-echoing mutants per backend are caught.
- [ ] Allowlist enforcement: gemini host absent from `net_allow` → `Err(DisallowedHost)`.
- [ ] All tests passing (`make test`); `make check-boundaries` clean; `env-vars.md` regenerated;
      simplicity-audit diff explains the +1 env-var route and +1 policy field.
- [ ] Documentation updated (agent-tool-policy.md, std/web.ail comment, CHANGELOG).

## Testing Strategy

**Unit tests:**
- Policy/env resolution matrix: absent→ollama, explicit ollama, gemini, unknown (refused), env var
  on non-policy path, env var ignored under `--policy`.
- decodeSearch mapping: full grounding fixture, dedupe-by-uri, truncation to `max`, malformed JSON,
  missing `groundingMetadata` (→ typed Transport Err naming the backend, not a panic).
- Missing-key per backend: error names the backend's OWN env var (OLLAMA_API_KEY /
  GEMINI_API_KEY) — V14 generalised.

**Integration tests:**
- Recorded-fixture server per backend asserting the request SHAPE (URL, tools payload, auth
  header form: `Authorization: Bearer` vs `x-goog-api-key`).
- The parameterised non-leak sweep (Phase 3) — the amendment's "added to that same test, not a
  copy".
- Allowlist negative tests per backend host.

**Manual testing:**
- `AILANG_WEB_LIVE_TEST=1` live controls against real Ollama and real Gemini (opt-in, never in CI).

## Deferred Decisions

- The pinned grounding model name (D9 names the mechanism: constant + test var; Phase 0 picks the
  concrete value and verifies grounding availability on it) — agent may resolve.
- `groundingSupports` → `content` mapping detail when supports cite multiple chunks — agent may
  resolve within the contract (content is a plain string; first-citing-support is the default).
- Whether a THIRD backend (Brave/Bing) later reuses `keyEnv`/`authHeader` as-is — out of scope
  (Non-Goals), but the seam extension must not hardcode two names.

## Non-Goals

**Not attempted in this feature:**
- A pi harness-level `web_search` tool — that is the deferred `extensions:` decision in
  M-DANEEL-AILANG-EXECUTOR, a different layer.
- Brave/Bing/other third backends — one more impl behind the seam, later; nothing here should
  assume exactly two.
- Model selection per policy (D9 rules this out; the selector is exactly `web_backend`).
- Changing std/web's public contract, types, effect row, or `WebSearchMaxResults` (stays 10).
- Cross-backend fallback of any kind (D7; A11).

## Timeline

**Days 1** (4 hours): Phase 0 probes (2h) + Phase 1 selector plumbing (3h, overlaps by intent —
plumbing is premise-independent).

**Day 2** (5 hours): Phase 2 Gemini backend (4h) + Phase 3 non-leak parameterisation (2h).

**Day 3** (2 hours): Phase 4 docs/examples + CHANGELOG + full gates (`make test`,
`check-boundaries`, simplicity-audit diff note).

**Total: ~10 hours across 3 days** (2x the naive estimate, per skill guidance).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Google Search grounding unavailable/refused on the pinned model (V16 wrong) | High — feature voids | Phase 0 probe FIRST; BLOCK and return to the operator rather than shipping a dead selector |
| `groundingMetadata` shape differs from the fixture (supports/segment fields vary by model) | Med | Fixture recorded from the live probe, not from memory; decodeSearch fails loudly (typed Transport Err) on unknown shapes — no partial silent results |
| Gemini quota exhausted mid-lane (the "may dry up" scenario mirrored) | Med | The selector is the mitigation: operator flips the policy field; refusal shapes name the backend so attribution is unambiguous |
| Key leak via a carrier the sweep misses (x-goog-api-key is a new header form) | High — lending boundary | The parameterised sweep's carriers include the header-form difference itself; positive control per backend; mutants per backend |
| Policy host check too strict (refusing a policy that sets `web_backend` but never uses std/web) | Low | Setting `web_backend` signals intent to use std/web; the refusal message names exactly what to add. Documented under Conflict Surface §5 as deliberate |
| Simplicity-gate regression (+1 env-var route, +1 policy field) | Low | Both are documented surfaces (env-vars.md, agent-tool-policy.md); the audit diff cites this doc |

## Related Documents

**Implemented (may inform design):**
- [M-AGENT-AILANG-ONLY-EXECUTION](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) — the
  lane, the gate, the `--policy` authority doctrine this doc mirrors
- [M-AGENT-SAFE-RUNNER](../../planned/v1_1_0/m-agent-safe-runner.md) — the policy admission contract
  (`policyCheckOutput` stability: this doc adds NO field to that shape — only to the stderr
  admission line, which has no schema-version pin, V9)

**Planned (check for overlap):**
- [M-DANEEL-AILANG-EXECUTOR](../v0_39_1/m-daneel-ailang-executor.md) — the parent: std/web landed
  under its M2; its D2 policy grants and its deferred `extensions:` decision bound this doc's
  Non-Goals; its lane policy is the first consumer of `web_backend`

No duplicate/coverage conflict: `ailang docs search` over planned/ and implemented/ found no
std/web-backend doc (the neural search backend was unreachable in this session; SimHash scan of
252 planned docs found no topical match — the only std/web references are the M-DANEEL-AILANG-
EXECUTOR artifacts).

## Verification Log

All rows verified against this checkout (2026-09-16) unless marked EXTERNAL/INHERITED.

| # | Claim | Evidence | Status |
|---|---|---|---|
| V1 | The `webBackend` seam exists with exactly ONE implementation (`ollamaWebBackend`) selected by package var `activeWebBackend`; fixed base `https://ollama.com`, "deliberately no env var"; `WebSearchMaxResults = 10`; ollama decoder normalises nil links → empty | read `internal/effects/web.go:44–100` | **Confirmed** |
| V2 | std/web contract: `webSearch(query, max) -> Result[list[{title,url,content}], NetError] ! {Net}`; `webFetch(url) -> Result[{title,content,links}, NetError] ! {Net}` | read `std/web.ail`, `internal/builtins/web.go` | **Confirmed** |
| V3 | Every request goes through `buildSecureRequest`; `net_allow` enforced via `isAllowedDomain(u.Hostname(), ctx.Net.AllowedDomains)` | read `internal/effects/net.go:506,531` | **Confirmed** |
| V4 | `GEMINI_API_KEY` already declared with getter `GeminiAPIKey()` | read `internal/config/providers.go:16,39,54` | **Confirmed** |
| V5 | NEGATIVE: no `web_backend` field and no `AILANG_WEB_BACKEND` exist anywhere today | `grep -rn "web_backend\|AILANG_WEB_BACKEND" internal/ cmd/` → 0 hits | **Confirmed (negative)** |
| V6 | NEGATIVE: no `google_search`/`grounding`/`url_context` code exists in `internal/` | grep → only the web.go:100 seam comment ("a second (e.g. Gemini grounding) is a follow-up") | **Confirmed (negative)** |
| V7 | `internal/ai/gemini/types.go` `toolBlock` models `functionDeclarations` only | read `types.go:15–19` | **Confirmed** |
| V8 | `internal/effects` already imports `internal/ai`; `internal/ai` is inside the measured language closure | `internal/effects/ai.go:10`; ARCHITECTURE.md generated closure (2026-09-15) lists `internal/ai` | **Confirmed** |
| V9 | The admission line is a JSON object on stderr (`"policy: "…`) whose field set includes `ai_provider`; `ailang-exec.ts` extracts it by regex and JSON-parses (additive-field tolerant) | read `cmd/ailang/run_policy.go:152–165`, `cmd/ailang/pi_assets/ailang-exec.ts:198–200,233` | **Confirmed** |
| V10 | `ai_provider` consistency validation lives ONLY in `applyRunPolicy` (refuse → exit 1), not in `policy-check`/`admitProgram` | read `run_policy.go:116–120` vs `policy_check.go` (no `ai_provider` reference) | **Confirmed** |
| V11 | `transportMessage` returns `err.Error()`; `client.Do` failures are `*url.Error` carrying the full URL — a `?key=` query would leak the key into a program-visible Err | read `internal/effects/net_proxy.go:164–171` + the url.Error comment at `net.go:17`; `webPost` uses `transportMessage` | **Confirmed** |
| V12 | `TestWeb_SecretNeverLeaks` sweeps ≥12 carriers + positive control over ONE backend via `pointWebAt` (flips the single base-URL var); sentinel `webSentinelKey` set via `config.EnvOllamaAPIKey` | read `internal/effects/web_test.go:21,69–74,258–316` | **Confirmed** |
| V13 | `policy.Load` refuses unknown TOML fields (typo'd field names fail loudly); env vars are declared as `Var` rows in `internal/config` and documented in GENERATED `docs/docs/reference/env-vars.md` (`make docs-env`) | read `internal/policy/policy.go:87–98`; `env-vars.md` header; `internal/config/doc.go:26` | **Confirmed** |
| V14 | `webPost`'s missing-key message hardcodes `config.EnvOllamaAPIKey`; the Authorization-Bearer header is hardcoded in `webPost` | read `internal/effects/web.go:186–194` | **Confirmed** |
| V15 | Gemini AI Studio base `https://generativelanguage.googleapis.com/v1beta`; request shape `models/{model}:generateContent?key=…` (query-param auth, in the AI-effect client) | read `internal/ai/gemini/client.go:23`, `client_test.go:309` | **Confirmed** |
| V16 | EXTERNAL: `generateContent` accepts `tools:[{"google_search":{}}]`; response carries `groundingMetadata.groundingChunks[].web{uri,title}` + `groundingSupports[].text/segment_indices` | Gemini API reference — not verifiable in this checkout | **UNVERIFIED — Phase 0 live probe (hard gate: fixture is recorded from the probe, not from this claim)** |
| V17 | EXTERNAL: the `url_context` tool is available on the pinned model and returns usable page content/links | same | **UNVERIFIED — Phase 0 decides D7** |
| V18 | EXTERNAL: Gemini grounding quota/pricing posture (free tier? per-1k-query price) | Google pricing page — not verifiable in this checkout | **UNVERIFIED — Phase 0 records the numbers for the implementation report** |
| V19 | `examples/runnable/web_search.ail` exists as the public-contract regression fixture | `ls examples/runnable/web_search.ail` | **Confirmed** |
| V20 | The parent doc's lane policy already grants `generativelanguage.googleapis.com` for the AI effect, and defers a pi-level web_search tool ("extensions:") | read `design_docs/planned/v0_39_1/m-daneel-ailang-executor.md` (D2, deferred-extensions note) | **Confirmed (inherited context)** |

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Gemini API reference — Grounding with Google Search (`tools: google_search`) and the
  `url_context` tool (the two external premises; Phase 0 verifies against the live API, V16–V17)
- [M-DANEEL-AILANG-EXECUTOR](../v0_39_1/m-daneel-ailang-executor.md) — parent doc; std/web's
  reason to exist; the first `web_backend` consumer
- [M-AGENT-SAFE-RUNNER](../../planned/v1_1_0/m-agent-safe-runner.md) — the policy admission contract
- [agent-tool-policy guide](/docs/guides/agent-tool-policy) — operator-facing policy field docs
  (where `web_backend` will be documented beside `ai_provider`)

## Future Work

- A third backend (Brave/Bing) — one more `webBackend` impl; the seam must stay two-agnostic.
- `web_backend`-conditional policy linting in `ailang policy-check` (today field-VALUE validation
  mirrors `ai_provider` on the run path only, V10 — if lane tooling wants policy-check to agree,
  that is a separate, symmetric change to both fields).
- Per-lane backend telemetry beyond the admission line (e.g. a per-backend quota observer, mirroring
  the ollama/gemini quota observers) — only if a lane measurably needs it.

---

**Document created**: 2026-09-16
**Last updated**: 2026-09-16