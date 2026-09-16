# M-STD-WEB-SECOND-BACKEND: a second web-search provider behind the std/web seam

**Status**: Planned
**Target**: v0.39.2
**Priority**: P1 (Medium) — removes a single-provider dependency on the `ailang_only` lane
**Estimated**: 3 days (implementation 1d, tests/fixtures 1d, docs + policy plumbing 0.5d, buffer 0.5d)
**Dependencies**: M-DANEEL-AILANG-EXECUTOR M2 (std/web, shipped on dev 6377683dc) — this doc is the follow-up the seam was built for
**Author**: design-doc-creator, 2026-09-16, on coordinator task (Mark, 2026-09-16: "ollama may dry up")

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | External API non-determinism unchanged; backend selection is a deterministic function of the policy/env, refused loudly when unknown |
| A2: Replayability | +1 | The backend name is banked on the `policy:` admission line, so a search that fails because "ollama dried up" is attributable from the banked run, not a mystery |
| A3: Effect Legibility | 0 | std/web still exposes exactly `{Net}`; no effect-row change |
| A4: Explicit Authority | +1 | The provider choice becomes an operator-pinned policy field (`web_backend`) rather than an ambient compile-time constant; unknown names are refused; no silent fallback anywhere (unsupported fetch is a typed Err) |
| A5: Bounded Verification | 0 | No local verification surface changes |
| A6: Safe Concurrency | 0 | No concurrency changes (backend selection resolved once per process, before any run) |
| A7: Machines First | +1 | Provider failover becomes a one-line policy diff an agent can propose and a human can review; the unsupported-fetch refusal is a decidable typed case, not prose |
| A8: Minimal Syntax | 0 | No language syntax; one policy TOML field, one env var |
| A9: Cost Visibility | +1 | Each backend has a different quota/cost posture (Ollama subscription quota vs Gemini token-billed interactions); the admission line records which one a run used |
| A10: Composability | +1 | The second provider proves the `webBackend` seam composes: builtins, std/web and the effect row are untouched |
| A11: Structured Failure | +1 | Unknown backend name = loud refusal (exit 1 naming valid values); fetch on a fetch-less backend = `Err(Transport(...))` naming the backend — never a silent fallback |
| A12: System Boundary | +1 | The lending boundary is preserved and extended: the program still cannot name a key, a host or a model — now the OPERATOR, not the binary, picks the provider |

**Net Score: +7** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (backend choice is pinned before execution)
- [x] A3 (Effects): no hidden side effects (`{Net}` only, unchanged)
- [x] A4 (Authority): no ambient access granted — selection flows operator → policy/env → runtime; the program never sees it as authority
- [x] A7 (Machines First): typed refusals and a machine-readable admission line, not human-only prose

## Problem Statement

**Current State (measured, on dev 6377683dc — M-DANEEL-AILANG-EXECUTOR M2):**

- `std/web` exists with a fixed contract: `webSearch(query, max) -> Result[list[{title,url,content}], NetError] ! {Net}` and `webFetch(url) -> Result[{title,content,links}, NetError] ! {Net}` (std/web.ail; sprint plan M2 recorded complete).
- `internal/effects/web.go` already has the seam: interface `webBackend { name; searchRequest; decodeSearch; fetchRequest; decodeFetch; apiKey }` with exactly ONE implementation (`ollamaWebBackend`, fixed base `https://ollama.com`, key from `OLLAMA_API_KEY` via `internal/config`).
- Every request goes through `buildSecureRequest`, so the policy's `net_allow` must list the backend host.
- The non-leak invariant is tested with a sentinel key across every carrier (`TestWeb_SecretNeverLeaks` in internal/effects/web_test.go: values, errors, 401/500/malformed/refused, plus a positive control); two key-echoing mutants are caught by that test.

**The gap:** the `ailang_only` lane (daneel-executor) depends on a single search provider — Ollama Cloud under Mark's `OLLAMA_API_KEY`. Mark's own words, 2026-09-16: **"ollama may dry up."** When that quota dries up, the lane loses web search entirely; there is no second provider to fail over to, and switching would require a code change rather than an operator decision. The single point of failure sits in the one place the design deliberately made operator-owned: the lending boundary.

**Impact:** the daneel-executor lane (P1 mission, M-DANEEL-AILANG-EXECUTOR) and any future policy-bounded agent that lends `Net` for search. Blocker-level for lane continuity if the Ollama quota is exhausted; an annoyance today only because the quota has not yet dried up (see [m-ollama-quota-observation.md](../m-ollama-quota-observation.md) — legacy session/weekly plans with observed usage, no plan upgrade authorized).

## Goals

**Primary Goal:** Remove the single-provider dependency: a second web-search backend (Gemini, via `GEMINI_API_KEY` — already in internal/config) behind the existing `webBackend` seam, selectable by the OPERATOR per lane via a policy field, with zero change to std/web's contract or the builtins.

**Success Metrics:**
1. `web_backend = "gemini"` in a lane policy makes `std/web.webSearch` reach `generativelanguage.googleapis.com` (which must then be in `net_allow`) with no recompile and no std/web change.
2. The shared non-leak test runs parameterised over BOTH backends (each with its own sentinel key and positive control) and still catches key-echoing mutants on each.
3. Unknown backend name (`web_backend = "brave"`) refuses at admission with exit 1 naming the valid values — no fallback, no run.
4. `webFetch` under the Gemini backend returns a typed `Err(Transport("webFetch: not supported by the gemini backend ..."))` having made ZERO requests — the doc-level verification that no silent fallback exists.
5. Every existing std/web test and fixture (Ollama path) passes unchanged except for the deliberate parameterisation.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1 — the selector is a policy field `web_backend`, not a flag/env under `--policy`** | Under `--policy` the authority must come from the file and nowhere else (the run_policy.go doctrine, established for `ai_provider`); an env var under `--policy` would be ambient authority an agent's environment could set | human | design | med |
| **D2 — `webFetch` stays Ollama-only; Gemini returns a typed Err** | Gemini's URL-context tool is model-mediated and returns neither raw page content nor any links list (verified, V6) — faking `{title,content,links}` would change what the contract means silently | human | design | high |
| **D3 — reuse `Err(Transport(...))` for the unsupported-fetch Err, no new NetError variant** | A new `NetError` ctor changes a public stdlib ADT (std/net.ail) and risks breaking exhaustive matches in user programs; Transport with an explicit message keeps the surface frozen | human | design | med |
| **D4 — Gemini key on the wire goes in the `x-goog-api-key` HEADER, never a `?key=` query param** | `transportMessage` (net_proxy.go:168) returns `err.Error()` of a `*url.Error`, which embeds the full request URL — a `?key=` URL would leak the key into the error carrier on any connection failure (the sentinel test would catch it; the existing internal/ai/gemini client's `?key=` pattern must NOT be copied) | agent | design | low |
| **D5 — the Gemini backend model is a pinned Go constant (`gemini-3.8-flash`), not an operator knob** | The program must not be able to influence the model (lending boundary); the operator picks the BACKEND, whose shape — endpoints, model — is fixed in Go exactly like Ollama's endpoints are | agent | design | low |
| **D6 — the Gemini client lives in `internal/effects/web_gemini.go` with its own minimal wire types, no import of `internal/ai/gemini`** | `internal/ai/gemini` is generateContent-based with `functionDeclarations`-only tool blocks (V9) — the Interactions API shape does not exist there; the seam's wire types are backend-private by design (same as Ollama's) | agent | design | low |
| **D7 — `web_backend` empty/unset defaults to `"ollama"` everywhere** | Refusing on empty would break every existing Net policy (Net does not imply std/web); typo-safety is complete anyway: a misspelled FIELD is refused by policy.Load's undecoded check, a misspelled VALUE by the admission check | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — selector is a policy field under `--policy` (matches the ratified `ai_provider` lending-boundary pattern; no human ruling pending beyond this doc's approval)
- [x] D2 — webFetch stays Ollama-only (verified against the Gemini API docs, V6; a fetch-capable second provider is future work behind the same seam)
- [x] D3 — no new NetError variant (stdlib surface frozen)
- [ ] **Mark ratifies the Gemini backend as the second provider** (the choice of provider — and paying for its quota — is a human decision, same class as the D2 model ruling in m-daneel-ailang-executor.md)

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact JSON struct field names/tags inside `web_gemini.go` (must decode the verified response shape; naming is agent's choice) — agent may choose
- Fixture capture method for `gemini_web_search.json` (live capture with a real key if available at implementation time, else hand-shaped from the documented example with provenance in the test comment) — agent may choose
- Whether `decodeSearch` returns `Ok([])` or a Transport Err when the model answers with zero citations — **decided here: `Ok([])`** (a grounded "found nothing" is a valid answer, mirroring Ollama's empty `results` array); only structural absence of `model_output`/`steps` is an Err — resolved, listed only so implementers do not re-open it
- Env-var registry Area constant for `AILANG_WEB_BACKEND` in internal/config (must be registered and documented; which Area bucket is agent's choice) — agent may choose
- Whether serveapi ever grows backend selection (today it keeps the default; a lane must ask first) — human at review, if ever

## Solution Design

### Overview

Add a second implementation of the existing `webBackend` seam and make the ACTIVE backend a runtime selection resolved from operator authority (policy field under `--policy`, env var otherwise), refused loudly when unknown. Gemini is the second backend: Google Search grounding via the Gemini **Interactions API** (`POST /v1beta/interactions`, `tools: [{"type": "google_search"}]`), whose citation-bearing response is mapped onto the SAME `{title,url,content}` records. `webFetch` remains Ollama-only because Gemini's URL-context tool cannot faithfully produce `{title, content, links}` — under Gemini it is a typed Err, not a fallback.

The program-visible contract of std/web does not change at all: same types, same effects, same NetError carriers. What changes is who decides which provider serves the request.

### Architecture

**Component 1 — the backend selector (policy + env, the lending boundary).**

- `internal/policy.Policy` gains `WebBackend string \`toml:"web_backend"\`` (policy.go, beside `AIProvider`). Empty = `"ollama"` (D7).
- Under `--policy` (cmd/ailang/run_policy.go, `applyRunPolicy`):
  - After loading, validate: `pol.WebBackend != ""` and not in the known set → `refuse("web_backend names unknown backend %q (known: ollama, gemini)")` — the same refusal pattern as unknown `allowed_caps` entries. The known set is exported by the effects package (`effects.KnownWebBackends() []string`) so the seam stays the single source of truth and a third backend registers itself in one place.
  - `runPolicyResolved` gains `webBackend string`; the admission line JSON (run_policy.go:157 map) gains `"web_backend": <resolved name>`.
  - **The `AILANG_WEB_BACKEND` env var is deliberately IGNORED under `--policy`** — "the authority comes from the file and nowhere else" (run_policy.go header doctrine). Unlike `fs_sandbox`, no env-var override happens at all.
- Outside `--policy` (plain `ailang run`): `config.WebBackend()` reads `AILANG_WEB_BACKEND` (default `"ollama"` when unset/empty; registered in the internal/config env registry and docs/docs/reference/env-vars.md). Unknown value → the run fails loudly BEFORE execution (exit 1, message naming valid values) — no silent fallback to ollama (A4/no-silent-fallbacks).
- Application point (both paths converge): in `cmd/ailang/main_run.go`, after policy resolution (or env read on the non-policy path), call `effects.SetActiveWebBackend(name) error` once before `runFile`; on error, print and exit 1. The setter mutates the package-level `activeWebBackend` var (web.go:95) — the same package-var discipline the tests already use (`pointWebAt`), resolved once per process before any run, so no mid-run switching and no race.
- `internal/builtins/web.go` and std/web.ail are NOT touched: the builtins call the effect handlers, which read `activeWebBackend` at call time.

**Component 2 — the Gemini backend (`internal/effects/web_gemini.go`, package effects).**

Verified wire contract (V3–V5, fetched from ai.google.dev 2026-09-16):

- Request: `POST {geminiWebBaseURL}/v1beta/interactions` with header `x-goog-api-key: <GEMINI_API_KEY>` (D4) and body
  `{"model": "gemini-3.8-flash", "input": <query>, "tools": [{"type": "google_search"}]}`.
- Base URL: fixed package var `geminiWebBaseURL = "https://generativelanguage.googleapis.com"` — test-overridable exactly like `ollamaWebBaseURL`; deliberately NO env var (same discipline, web.go:35).
- Model: pinned constant (D5): `geminiWebModel = "gemini-3.8-flash"` — doc-verified supported for google_search grounding (V5).
- `max` does NOT go on the wire: the google_search tool has no max-results parameter (V7). It truncates the decoded citation list client-side, after the full interaction is billed — recorded in the cost posture below.

Response shape (verified example, V4) — `steps` containing `thought` steps, `google_search_call {arguments:{queries}}`, `google_search_result {call_id, result:[{search_suggestions}]}`, and `model_output {content: [{type:"text", text, annotations: [{type:"url_citation", url, title, start_index, end_index}]}]}`.

`decodeSearch` mapping onto the SAME `{title,url,content}`:
1. Walk `steps`, collect `url_citation` annotations from `model_output` text content blocks, in response order.
2. Dedupe by URL (first occurrence wins; all segments cited to that URL are joined with `"\n"` in index order).
3. Each unique URL → `{title: annotation.title, url: annotation.url, content: joined cited segments}`.
4. Truncate to `max`; return. Zero citations → `Ok([])` (Deferred Decisions). Structural absence of `steps`/`model_output` or malformed JSON → Transport Err ("malformed search response: ...").

**Documented semantic difference (must land in std/web.ail's comment and this doc):** Gemini's `content` is the MODEL'S grounded answer segment citing that source, not the provider's own page extract (Ollama returns the page's own snippet). A consumer that needs the source's own text must `webFetch` it — which under Gemini is the typed Err, so the honest per-backend capability matrix is:

| op | ollama | gemini |
|---|---|---|
| `webSearch` | ✅ provider snippets | ✅ model-cited answer segments (deduped citations, client-side `max`) |
| `webFetch` | ✅ | ❌ `Err(Transport("webFetch: not supported by the gemini backend — set web_backend = \"ollama\" for fetch"))`, ZERO requests |

**Component 3 — seam refinement (three interface additions, no signature breaks).**

`webBackend` (web.go:31) gains:

- `supportsFetch() bool` — ollama: true; gemini: false. `WebFetch` checks it FIRST and returns the D2/D3 Err before any request (mirrors the E_WEB_INVALID_INPUT discipline: fail before the wire).
- `keyEnv() string` — the env-var NAME for the missing-key message. **This fixes a latent per-backend bug:** `webPost` (web.go:187-189) hardcodes `config.EnvOllamaAPIKey` in the "not set for the backend" message — correct today with one backend, wrong the moment a second lands. After the change the message names `GEMINI_API_KEY` under gemini.
- `authHeaders(key string) []eval.Value` — ollama returns the existing `Authorization: Bearer <key>`; gemini returns `x-goog-api-key: <key>` (D4). `webPost` stops hardcoding the Authorization header and uses this; the Ollama wire format must remain byte-identical (regression fixture below).

**Component 4 — cost/quota posture and what the transcript banks.**

- Ollama: one `OLLAMA_API_KEY` on a subscription quota (m-ollama-quota-observation.md: legacy session/weekly plans, observed usage 0.356/week at 2026-09-07, no plan upgrade authorized). Dry-up risk: total loss of lane search — the reason for this doc.
- Gemini: `GEMINI_API_KEY` (already read by internal/config, providers.go:39-40). Each `webSearch` is a full Interactions call billed as input+output+thought tokens, subject to the Gemini API's own rate limits (see the API rate-limits page — do NOT bake quota numbers into code; the admission policy does not enforce web quotas today and this doc does not add that).
- **The transcript banks: `"web_backend": "<name>"` on the `policy:` admission line** (stderr, the line `.pi/extensions/ailang-exec.ts` already parses — additive JSON key, tolerated by the parser, V19). **Nothing on the wire values:** `SearchResult`/`FetchResult` carry only `{title,url,content}`/`{title,content,links}` — no backend name, no model, no key material; the program cannot name the provider it is talking to. Backend names DO appear in NetError text ("gemini backend returned HTTP 429", missing-key names `GEMINI_API_KEY`) — that is the existing pattern (web.go uses `b.name()` in messages) and carries no secret.
- Switching backends for a dry-up event is a one-line policy edit (`web_backend = "gemini"`, `net_allow` gains `generativelanguage.googleapis.com`) — an operator decision, reviewable as a diff, no recompile.

### Implementation Plan

**Phase 1 — seam refinement (backend-agnostic, keeps Ollama byte-identical)** (~3 hours)
- [ ] Add `supportsFetch()`, `keyEnv()`, `authHeaders()` to the `webBackend` interface; implement on `ollamaWebBackend`; rewire `webPost`/`WebFetch` (web.go). Ollama's wire behavior unchanged (fixture test proves it).
- [ ] Add `effects.KnownWebBackends()` and `effects.SetActiveWebBackend(name) error` (registry map name→instance; unknown → error naming valid values).

**Phase 2 — the Gemini backend** (~5 hours)
- [ ] `internal/effects/web_gemini.go`: wire types (request/response/steps/annotations), `geminiWebBackend` implementing the seam (search only, `supportsFetch() = false`), pinned base URL + model constants, the D4 header.
- [ ] `decodeSearch` per the mapping above (dedupe, join, truncate, `Ok([])` on zero citations, Transport on malformed).
- [ ] `WebFetch` unsupported path (typed Err, zero requests).

**Phase 3 — selector plumbing** (~4 hours)
- [ ] `internal/policy.Policy.WebBackend` field (+ policy.Load doc comment update).
- [ ] `applyRunPolicy`: unknown-name refusal, `runPolicyResolved.webBackend`, admission-line key; `main_run.go` resolve-and-apply for both paths.
- [ ] `internal/config`: `WebBackend()` reading `AILANG_WEB_BACKEND`, default `"ollama"`, loud refusal of unknown values at the call site; register the env var.

**Phase 4 — tests and fixtures** (~6 hours)
- [ ] `internal/effects/testdata/gemini_web_search.json` (recorded/hand-shaped per Deferred Decisions; must contain a thought step, a google_search_call, a google_search_result, ≥2 url_citations and one URL cited twice).
- [ ] Per-backend fixture tests (wire + decode + max truncation + dedupe).
- [ ] Parameterise `TestWeb_SecretNeverLeaks` over backends (per-backend sentinel, carriers, positive control, per-backend carrier-count floor).
- [ ] Unsupported-fetch test; unknown-name refusal tests (policy path and env path); admission-line test asserting `"web_backend"`.
- [ ] `TestWeb_LiveGemini` opt-in live control (`AILANG_WEB_LIVE_TEST=1` + `GEMINI_API_KEY`), mirroring `TestWeb_LiveOllama`.

**Phase 5 — docs** (~2 hours)
- [ ] std/web.ail doc comment: backend-agnostic wording + the per-backend capability matrix + the content-semantics difference.
- [ ] docs/docs/guides/agent-tool-policy.md + examples/safety/agent-policy.example.toml: `web_backend` field beside `ai_provider`; admission-line mention.
- [ ] docs/docs/reference/env-vars.md: `AILANG_WEB_BACKEND`.

### Files to Modify/Create

**New files:**
- `internal/effects/web_gemini.go` (~140 LOC) — Gemini Interactions wire types + backend (search-only)
- `internal/effects/web_gemini_test.go` (~200 LOC) — Gemini fixture tests + parameterised non-leak participation
- `internal/effects/testdata/gemini_web_search.json` (~60 LOC) — recorded fixture

**Modified files:**
- `internal/effects/web.go` (+45/−15 LOC) — three interface methods, `webPost`/`WebFetch` rewire, backend registry + setter
- `internal/effects/web_test.go` (+80/−40 LOC) — parameterise the non-leak test; unsupported-fetch and unknown-name tests
- `internal/policy/policy.go` (+4 LOC) — `WebBackend` field
- `cmd/ailang/run_policy.go` (+25 LOC) — validation, resolution, admission-line key
- `cmd/ailang/main_run.go` (+10 LOC) — resolve-and-apply on both paths
- `internal/config/providers.go` (or sibling, +10 LOC) — `WebBackend()`, `AILANG_WEB_BACKEND`, env registry entry
- `std/web.ail` (doc comment only), `docs/docs/guides/agent-tool-policy.md`, `docs/docs/reference/env-vars.md`, `examples/safety/agent-policy.example.toml` (doc edits)

## Examples

### A lane policy pinning Gemini (the operator's failover)

```toml
# agent-policy.toml (daneel-executor lane, failover posture)
allowed_caps = ["IO", "Net"]          # search needs only Net
net_allow = ["generativelanguage.googleapis.com"]  # the BACKEND host, not search targets
web_backend = "gemini"                # NEW — the operator picks the provider
entry = "main"
```

The AILANG program is UNCHANGED — this is the point:

```ailang
module examples/runnable/web_search
import std/web (webSearch)
import std/result (Result)
-- webSearch("AILANG programming language", 3) returns the same
-- Result[list[{title,url,content}], NetError] ! {Net} whichever backend
-- the operator pinned. The program cannot name a key, a host, or a model.
```

### The typed unsupported-fetch refusal (no silent fallback)

```ailang
-- under web_backend = "gemini":
match webFetch("https://ailang.sunholo.com/") {
  Ok(page) => ...   -- not reached
  Err(Transport(msg)) => ... -- msg: webFetch: not supported by the gemini
                             -- backend — set web_backend = "ollama" for fetch
}
```

### An unknown backend refuses at admission

```
$ ailang run --policy lane.toml prog.ail
Error: --policy: web_backend names unknown backend "brave" (known: ollama, gemini)
(exit 1; nothing executes)
```

## Success Criteria

- [ ] `web_backend = "gemini"` policy + `net_allow = ["generativelanguage.googleapis.com"]` yields real search results from a live `GEMINI_API_KEY` (opt-in live test), and the recorded-fixture test passes offline.
- [ ] Ollama path byte-identical: existing `ollama_web_search.json`/`ollama_web_fetch.json` fixture tests pass with zero fixture edits.
- [ ] Parameterised `TestWeb_SecretNeverLeaks` exercises BOTH backends with distinct sentinels, per-backend carrier floors, per-backend positive controls; re-inserting either known key-echoing mutant (or a `?key=` URL mutant for gemini) fails it.
- [ ] Unknown backend name refuses (exit 1, valid values named) on both the policy path and the env path.
- [ ] `webFetch` under gemini: typed Err, ZERO requests observed at the fixture server.
- [ ] Admission line contains `"web_backend"`.
- [ ] `make check-boundaries` still passes (no new cross-layer imports — web_gemini.go imports only stdlib + internal/config + internal/eval, same as web.go).
- [ ] All tests passing; docs updated (std/web.ail comment, agent-tool-policy guide, env-vars reference, example policy).
- [ ] Mark has ratified the Gemini backend as the second provider (Design Freeze item).

## Conflict Surface

Touches `internal/effects/` and `cmd/ailang/` — required section.

### Positions touched

1. **`activeWebBackend` read sites** (web.go `WebSearch`/`WebFetch`): today a fixed var; becomes a registry-selected var. Nothing else reads it.
2. **`webBackend` interface**: three methods added — every implementation must grow them. Enumerate ALL implementations: exactly one exists today (`ollamaWebBackend`, web.go:60); grep for other `webBackend` implementors is empty (V17-class claim, verified: the interface is package-private and has one impl).
3. **`webPost` header construction**: `Authorization: Bearer` hardcoded (web.go:191-194) — becomes backend-supplied. Conflict: the Ollama wire format must not change (fixture test pins it).
4. **`webPost` missing-key message**: hardcodes `config.EnvOllamaAPIKey` (web.go:188) — becomes backend-supplied. Conflict: the existing test `TestWeb_MissingKeyIsATypedErrThatNamesTheVariable` asserts the OLLAMA variable name; it must keep passing (it will — ollama's `keyEnv()` returns the same constant).
5. **Policy struct / admission line**: a new TOML field changes what `policy.Load` ACCEPTS — a policy file containing `web_backend` that previously failed the unknown-field check now loads (intentional, additive). The admission-line JSON gains a key; consumer `.pi/extensions/ailang-exec.ts` `parsePolicyLine` reads named fields from parsed JSON (admission schema additive — verified V19).
6. **std/net NetError**: NOT touched (D3) — all existing `match`/`show` on NetError unchanged.

### Disambiguation strategy

No parser/typechecker ambiguity is introduced (no syntax change). The two real disambiguation points are authority flows, resolved by construction: (a) under `--policy` the env var is not consulted at all, so there is no precedence question; (b) the unknown-name check runs at admission, before any effect handler can observe a half-selected backend.

### Programs that MUST still work (regression fixtures)

1. `examples/runnable/web_search.ail` — the std/web consumer (exists, verified by ls).
2. `std/web.ail` — the module itself (compiles unchanged; only its comment changes).
3. `internal/effects/web_test.go` Ollama fixture tests — pin the byte-identical Ollama wire (payload, Authorization header, endpoints).
4. `docs/docs/guides/agent-tool-policy.md` policies — every existing example policy still loads (empty `web_backend` = ollama default).
5. The `TestWeb_LiveOllama` live control — unchanged gate, unchanged wire.

### What deliberately changes

- A policy/setting that NAMES an unknown backend now refuses where before the concept did not exist (no old behavior to break — additive).
- Gemini-backend `webSearch` results carry model-cited answer segments, not provider snippets — a semantic difference documented at the contract level, not a regression (ollama behavior unchanged).
- `AILANG_WEB_BACKEND` unset outside `--policy` keeps ollama — no existing run changes behavior.

## Testing Strategy

**Unit / fixture tests (hermetic):**
- Gemini recorded fixture: wire assertions (POST `/v1beta/interactions`, `x-goog-api-key` header — never `?key=` in the URL, `tools:[{"type":"google_search"}]`, pinned model, `input` = query, NO max on the wire) and decode assertions (dedupe by URL, segment join, `max` truncation, `Ok([])` on zero citations, Transport on malformed).
- Unsupported fetch: fixture server records ZERO requests; typed Err names the backend.
- Selector: `SetActiveWebBackend("nope")` errors naming valid values; policy with `web_backend="brave"` refuses; env `AILANG_WEB_BACKEND=bing` fails the run loudly before execution.

**Non-leak (the shared invariant, parameterised — not copied):**
- Refactor `TestWeb_SecretNeverLeaks` into a table over backends: per-backend sentinel (`sk-ollama-SENTINEL...` / `sk-gemini-SENTINEL...`), per-backend env var, per-backend base-URL override, the full carrier sweep (values, errors, 401/500/malformed/refused) and a per-backend positive control. A per-backend carrier-count floor keeps the assertion non-vacuous for each backend. The two known key-echoing mutants plus a new `?key=`-in-URL mutant for gemini must each still trip it (the D4 rationale is exactly this class).

**Live controls (opt-in, never block CI):**
- `TestWeb_LiveGemini` (mirror of `TestWeb_LiveOllama`): gated on `AILANG_WEB_LIVE_TEST=1` + `GEMINI_API_KEY`; also verify the allowlist check fires for a net_allow that omits `generativelanguage.googleapis.com`.

**Regression-surface tests:** the Ollama fixture tests double as the "still work" fixtures (Conflict Surface list).

**Manual:** the Examples section's policy file against the live lane; `ailang run --policy` prints the admission line with `"web_backend"`.

## Non-Goals

**Not in this feature:**
- A pi harness-level `web_search` tool — that is the deferred "extensions:" decision in m-daneel-ailang-executor, a different layer (the lane reads search THROUGH std/web in AILANG programs).
- Brave/Bing — a third backend is one more impl behind the seam, later; the registry makes it a single-file addition.
- Gemini `url_context` as a webFetch implementation — verified unable to satisfy the contract (no links, model-mediated content; V6). Revisit only if a distinct std/web function for model-mediated page reads is ever designed.
- New NetError variants (D3) or any stdlib type change.
- Quota enforcement / admission-time quota gauges for web backends (the policy does not enforce web quotas today; posture is documented, not enforced).
- Serveapi/backend-per-request selection (process-level selection only).

## Timeline

**Day 1** (6h): Phase 1 seam refinement + Phase 2 Gemini backend.
**Day 2** (6h): Phase 3 selector plumbing + Phase 4 tests/fixtures (bulk).
**Day 3** (4h): remaining tests, docs (Phase 5), `make check-boundaries`, full `make test`, buffer.

**Total: ~16 hours across 3 days** (2x the naive estimate, per house rules).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Gemini Interactions API shape drift (the doc page carries a May 2026 breaking-changes note; curl examples pin `v1beta`) | Medium | Fixed endpoint revision in the constant; decode fails loudly as Transport ("malformed"), never silently; fixture test pins the shape we rely on |
| Gemini free-tier rate limits throttle the lane under load | Medium | Loud HTTP-429 → `Err(Transport(...))` naming the backend (no retry loop); operator can flip back to ollama with a policy edit — that agility is the point of the doc |
| The model returns zero citations for a real query (grounded "found nothing") | Low | `Ok([])` is a valid answer (decided); consumers already handle empty lists |
| Thought tokens inflate cost per search | Low | Cost posture documented; no unverified thinking-config request field is asserted (implementer may add one ONLY if verified against the API reference at implementation time) |
| Parameterised non-leak test hides a backend (vacuity) | Medium | Per-backend carrier floor + per-backend positive control; mutant re-insertion check |

## Quorum Triggers

**Trigger 4 (external systems / external API contract) fires**: the design pins a wire contract with a third-party API (endpoint, request shape, response shape, model support — V3–V7), and the provider choice spends a real quota. This is exactly the class the quorum exists for. **Run `ailang design-quorum` on this doc before sprint planning** (per the design-doc-creator skill's optional pre-sprint step; reviewers as the parent doc used: `gpt5-6-sol`, `gemini-3-1-pro`, `oc-glm-5-2`). The Design Freeze item (Mark ratifying the provider) is the human gate; quorum may precede or accompany it but the sprint may not start on an unchecked freeze item.

## Verification Log

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | The `webBackend` seam exists with exactly one implementation; `activeWebBackend` is a package var; endpoints fixed; `WebSearchMaxResults = 10` | Read internal/effects/web.go:23-95 (interface at :31, ollama impl :60-89, var :95) | Confirmed |
| V2 | The non-leak test exists with sentinel, multi-carrier sweep, positive control, ≥12-carrier floor; missing-key test asserts the env-var NAME | Read internal/effects/web_test.go (`TestWeb_SecretNeverLeaks`, `TestWeb_MissingKeyIsATypedErrThatNamesTheVariable`) | Confirmed |
| V3 | Gemini search goes through the Interactions API: `POST https://generativelanguage.googleapis.com/v1beta/interactions`, header `x-goog-api-key: $GEMINI_API_KEY`, body `{"model":..., "input":..., "tools":[{"type":"google_search"}]}` | Fetched https://ai.google.dev/gemini-api/docs/google-search (curl example, REST section) on 2026-09-16; also the url-context page's REST example | Confirmed |
| V4 | Response shape: `steps` with `thought`, `google_search_call {arguments:{queries}}`, `google_search_result {call_id, result:[{search_suggestions}]}`, `model_output.content[].annotations[] = {type:"url_citation", url, title, start_index, end_index}` | Same fetched doc, "Understanding the grounding response" example JSON | Confirmed |
| V5 | `google_search` grounding supported on `gemini-3.8-flash` (and 3.7/3.6/3.5*/3.1/3/2.5*/2.0); older models use `google_search_retrieval` | Same fetched doc, "Supported models" table | Confirmed |
| V6 | `url_context` exists BUT: response is model-synthesized text + `url_citation` annotations + a `url_context_result` metadata step (status, retrieved URL); **no nested-link retrieval** ("The model will only retrieve content from the URLs you provide, not any content from nested links"); no raw page text; no links list | Fetched https://ai.google.dev/gemini-api/docs/url-context (How it works / Understanding the response / Best Practices / Limitations) on 2026-09-16 | Confirmed — webFetch NOT faithfully implementable on Gemini (D2) |
| V7 | The google_search tool has NO max-results parameter (negative-existence on the API surface) | The fetched google-search doc's request examples carry only model/input/tools; no result-count field appears anywhere on the page; absence claim scoped to what the page documents | Confirmed within scope — `max` truncates client-side |
| V8 | `transportMessage` returns the raw `err.Error()` of a `*url.Error`, which embeds the full request URL → a `?key=` URL leaks the key into the error carrier on connection failure | Read internal/effects/net_proxy.go:168-176; the existing `internal/ai/gemini` client uses `?key=` (generate.go:239-241) and must NOT be copied for the web backend (D4) | Confirmed |
| V9 | `internal/ai/gemini`'s `toolBlock` is `functionDeclarations`-only and the client is generateContent-based | Read internal/ai/gemini/types.go:18-19, generate.go:234-241 | Confirmed — reuse would not fit; separate minimal client (D6) |
| V10 | `make check-boundaries` does not police effects→internal/ai (CORE vs DASHBOARD only), and internal/effects already imports internal/ai for the AI handler — but this design adds NO internal/ai import anyway | Read scripts/check_boundaries.sh (CORE_PKGS/DASHBOARD_PKGS); internal/effects/ai.go:10 | Confirmed |
| V11 | `policy.Load` refuses unknown fields (undecoded check) — a misspelled `web_backend` field fails loudly today, and the new field is additive | Read internal/policy/policy.go:79-86 | Confirmed |
| V12 | `ai_provider` precedent: policy field, widening-flag refusal, resolution into run settings, admission-line key; doctrine "authority comes from the file and nowhere else" | Read cmd/ailang/run_policy.go (header, :116-119, :136-139, :158), main_run.go:215-242 | Confirmed — D1 copies this pattern |
| V13 | `NetError` has exactly 4 ctors: Transport, DisallowedHost, InvalidHeader, BodyTooLarge — adding a variant would change the public ADT | Read std/net.ail:15-19 | Confirmed (D3 rationale) |
| V14 | `examples/runnable/web_search.ail` exists; std/web contract is as stated | `ls` + read std/web.ail. NOTE: a live `ailang check` could NOT be run on the authoring box (no go/make toolchain; the installed binary is stale — predates the std/web builtins, errors on `_web_search`). The contract claims therefore rest on (b)-class citations: std/web.ail source + m-daneel-ailang-executor-sprint-plan.md M2 record. The sprint MUST re-run `ailang check examples/runnable/web_search.ail` on a rebuilt binary as its first act | Confirmed by citation; live check deferred to sprint start (recorded, not skipped silently) |
| V15 | `GEMINI_API_KEY` is already in internal/config with getter `GeminiAPIKey()` | Read internal/config/providers.go:16, 39-40 | Confirmed |
| V16 | `webPost`'s missing-key message hardcodes `config.EnvOllamaAPIKey` (latent per-backend bug) | Read internal/effects/web.go:187-189 | Confirmed — fixed via `keyEnv()` |
| V17 | Negative-existence: NO existing policy field, env var, or CLI flag selects a web backend (`web_backend`/`WebBackend`/`AILANG_WEB_BACKEND` appear nowhere) | `grep -rn "web_backend\|WebBackend\|AILANG_WEB_BACKEND"` across internal/, cmd/, docs/, examples/, std/ → only unrelated `webBackend` seam hits in web.go | Confirmed |
| V18 | Negative-existence: no Brave/Bing (or any third) web backend exists in the codebase | `grep -rin "brave\|bing"` internal/effects/ internal/policy/ → no backend hits | Confirmed |
| V19 | `.pi/extensions/ailang-exec.ts` `parsePolicyLine` tolerates additive admission-line keys (it extracts named fields from the parsed JSON) | Read .pi/extensions/ailang-exec.ts:198-235 | Confirmed — `"web_backend"` is additive-safe |
| V20 | The sprint plan's non-goal "Gemini Google Search grounding or new Gemini provider tool blocks" (m-daneel-ailang-executor-sprint-plan.md:194) refers to grounding for the EXECUTOR MODEL's inference lane — this doc adds grounding for the SEARCH backend behind std/web, a different consumer of the same API family; the relation is stated here so reviewers see no contradiction | Read design_docs/planned/v0_39_1/m-daneel-ailang-executor-sprint-plan.md:13-20, 194-195 | Confirmed — distinct scopes, cross-referenced |

## References

- Parent workstream: [m-daneel-ailang-executor.md](m-daneel-ailang-executor.md) · [m-daneel-ailang-executor-sprint-plan.md](m-daneel-ailang-executor-sprint-plan.md) (the M2 correction that shipped std/web on the Ollama backend)
- Quota context: [m-ollama-quota-observation.md](../m-ollama-quota-observation.md) — the "may dry up" substrate
- API references (fetched 2026-09-16): https://ai.google.dev/gemini-api/docs/google-search · https://ai.google.dev/gemini-api/docs/url-context
- Policy precedent: cmd/ailang/run_policy.go (`ai_provider`), internal/policy/policy.go
- Axioms: [docs/references/axioms](../../docs/references/axioms.mdx)
- Deferred: [m-gemini-interactions-api.md](../deferred/m-gemini-interactions-api.md) — the AI-effect client's own Interactions migration (separate lane; D6 keeps this doc independent of it)

## Future Work

- A third backend (Brave/Bing) — one file + one registry entry behind the seam; the refusal message and KnownWebBackends update themselves from the registry.
- A model-mediated page-read primitive (Gemini `url_context`) as a NEW std/web function with an honest contract — only if a lane asks; not a webFetch replacement (V6).
- Quota-aware backend admission (observe Gemini RPD like m-ollama-quota-observation does for Ollama) — a policy/quota-plane change, not a std/web change.
- serveapi backend selection if a serving lane ever needs it (Deferred).