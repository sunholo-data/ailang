# M-CODEX-SUBSCRIPTION-LANE: `codex*` fails loud now, a bounded `chatgpt/` client, and a `codex exec --json` AI-effect provider to replace the direct backend

Refs #903 (see also #1259)

**Status**: Planned
**Target**: v0.52.6 (Phase 1) · v0.53.0 (Phase 2)
**Priority**: P1 (High) — silent billing exposure on a model-name prefix
**Estimated**: Phase 1: 1 day · Phase 2: 4 days (2× the naive 2.5-day estimate, house convention)
**Dependencies**: None blocking. Phase 1's client deadline is aligned with open #1259 (the umbrella for AI-effect HTTP deadlines); #1259's `ctx.AI.Timeout`/`--ai-timeout` will absorb the constant when it lands. Supersedes `design_docs/planned/m-codex-billing-lane-resolution.md` (planned v0.39.0, never implemented — see Verification Log V2).
**Issues**: [#903](https://github.com/sunholo-data/ailang/issues/903) (P1, area:ai). Ruling: maintainer comment 2026-10-08, **option 3, both** — (1) now: `codex*` stops silently resolving to the metered OpenAI lane, fails loudly pointing at `chatgpt/`, and the `chatgpt/` client gets an HTTP deadline (see #1259); (2) long term: a `codex:` AI-effect provider driving `codex exec --json`, replacing the direct-backend `chatgpt/` client (commit `07492bb2c`), which calls `chatgpt.com/backend-api/codex` with the codex login token — a route the issue's research marks **NOT SANCTIONED**. All premises below were verified by code reads at `c3bfb1c0` (shallow checkout; the `07492bb2c` citation is the ruling's, not re-derived here — see Verification Log).

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | The AI effect is already the language's explicit non-determinism surface; no determinism guarantee is weakened or strengthened. |
| A2: Replayability | 0 | No trace/replay surface change; provider selection is recorded as before. |
| A3: Effect Legibility | +1 | The AI effect's mechanism becomes explicit per lane: HTTP providers speak HTTP; the codex lane declares itself a subprocess route behind the same effect signature — and the *unsanctioned* hidden route (mimicking the Codex CLI wire shape) is removed (V6). |
| A4: Explicit Authority | +1 | Billing authority becomes explicit: no model string can silently spend metered API credits (V1); the subscription lane refuses an api-key auth.json by construction (V17). |
| A5: Bounded Verification | 0 | No change to check cost or scope. |
| A6: Safe Concurrency | +1 | The one-auth.json serialization constraint (OpenAI's documented condition) gets an in-process lock and a fleet doctrine where today nothing enforces it (V18). |
| A7: Machine First | +1 | The fail-loud errors name the ruling, the sanctioned alternative, and the migration target — machine-actionable, single-sourced in the factory. |
| A8: Minimal Syntax | +1 | No new syntax; Phase 1 *removes* a guess arm and adds one sentinel type. |
| A9: Cost Visibility | +1 | The core of the doc: a subscription lane reports `CostListPriceEquivalent` (never metered spend), real token counts, and quota attribution via the existing instruments (V15, V16). |
| A10: Composability | 0 | The `ai.Provider` interface is unchanged; the new provider composes behind it. |
| A11: Structured Failure | +1 | Every new refusal is a typed, named error (existing `CodeToolsNotSupported`/`CodeAuthFailed`/`CodeCapabilityNotSupported` — V21), never a zero/empty completion. |
| A12: System Boundary | +1 | The sanctioned boundary (drive the codex CLI; never reimplement its OAuth against the backend) becomes the implemented architecture instead of a comment. |

**Net Score: +6** → **Decision: Move forward.** No −1 on A1/A3/A4/A7.

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced (the AI effect is the declared non-determinism surface; the subprocess is its mechanism)
- [x] A3: no hidden side effect added — one removed (the direct-backend mimic route), one made explicit (the CLI subprocess the effect's docs will name)
- [x] A4: ambient billing authority (silent metered-lane resolution) removed, not granted
- [x] A7: no human-convenience optimization over machine analysis

## Problem Statement

Three coupled defects, one root cause: **the AI effect has no honest ChatGPT-subscription lane.**

1. **`codex*` silently resolves to the metered OpenAI lane.** `GuessProvider` maps any `codex` prefix to `ProviderOpenAI` (`internal/ai/config.go:86–93`, V1), so an AI-effect call with a `codex:` model constructs the metered OpenAI API client — silently spending API credits, or failing on a missing `OPENAI_API_KEY` with nothing naming the mismatch. This is a silent fallback on a pricing path (CLAUDE.md Critical Principle 2). Verified still live at HEAD (V1); a test even pins the defect (`internal/ai/provider_test.go:125` `{"codex-max", ProviderOpenAI}`, V3).
2. **The interim subscription lane has no deadline.** `chatgpt.NewClient` uses a bare `&http.Client{}` (`internal/ai/chatgpt/client.go:52`, V4) — `Timeout 0`, the exact defect class of open #1259 (std/ai HTTP calls unbounded; V5). Every other oracle in the runtime is bounded.
3. **The interim subscription lane is the unsanctioned route.** The `chatgpt/` provider (commit `07492bb2c`) reads `~/.codex/auth.json` and calls `https://chatgpt.com/backend-api/codex` directly (V6), mimicking the Codex CLI request shape. The issue's research (verified 2026-08-16 against OpenAI's docs) marks third-party direct-backend use **NOT SANCTIONED**: no third-party client-ID allocation exists for ChatGPT-plan access, and it breaks whenever the backend's auth check changes. The sanctioned route for ChatGPT-plan auth is **driving the Codex CLI** (`codex exec --json`), which the executor lane already does.

Plus a bookkeeping defect: `design_docs/planned/m-codex-billing-lane-resolution.md` (planned v0.39.0) ruled "fail loud, forever — never build an AI-effect codex provider" (its Non-Goals explicitly reject the `codex exec` provider). The 2026-10-08 maintainer ruling supersedes it: fail loud **now**, and build the sanctioned provider **next**. That doc was never implemented (V2), so nothing on disk follows it.

**Impact:** anyone writing `codex:gpt-6.1-sol` in a model field today (billing integrity); motoko agents and `.ail` programs that need the subscription lane (currently routed through the unsanctioned backend); the fleet's billing guard, which cannot audit the direct-backend route it never sanctioned.

## Ruling (settled by the maintainer, 2026-10-08)

**Option 3, both:**

1. **Now (Phase 1, v0.52.6):** `codex*` model names stop silently resolving to the metered OpenAI lane. Fail loudly, pointing at `chatgpt/`. Give the `chatgpt/` client an HTTP deadline consistent with #1259.
2. **Long term (Phase 2, v0.53.0):** a `codex:` AI-effect provider that runs `codex exec --json` and adapts the NDJSON stream — the route OpenAI sanctions for ChatGPT-plan auth — replacing the `chatgpt/` direct-backend client. The design doc settles the issue's three open questions: tool-calling shape, serialized use of one auth.json, and quota-vs-dollar cost accounting. Supersedes the stale doc.

Phase 1's error message points at `chatgpt/` because that is the only subscription lane that exists **today**; Phase 2 replaces `chatgpt/` with `codex:` and flips the message. Two loud flips, each documented in its release's changelog — never a silent reroute.

## Goals

**Primary Goal:** no AI-effect call can silently bill the metered OpenAI lane via a `codex*` model name; the subscription lane is reached only through the sanctioned Codex-CLI route, bounded, serialized, and priced as a subscription.

**Success Metrics:**
- `GuessProvider("codex:gpt-6.1-sol")` and `{"codex-max"}` no longer yield `ProviderOpenAI`; AI-effect construction with a `codex*` model returns the ruling error naming `chatgpt/` (Phase 1, acceptance A1/A2).
- The `chatgpt/` client has a non-zero HTTP deadline by default; a hung backend connection is cut, not parked forever (A3, #1259 class).
- Phase 2: `codex:gpt-6.1-sol` through the AI effect runs `codex exec --json`, succeeds on the subscription lane with `OPENAI_API_KEY` stripped (live probe, the mission-control billing-guard shape), and its cost is recorded as `CostListPriceEquivalent`, never `CostMetered` (A6/A7).
- No in-repo caller of the mission executor's `codex:gpt-*` pins breaks (audit + regression, A4).
- The stale doc is marked superseded and no code follows it (A8).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: `codex*` resolves to a new sentinel `ProviderCodex` (not `""`), and `factory.New` refuses it loudly with the ruling message | Sentinel keeps telemetry distinguishable and gives the error a single source (the factory) instead of per-caller string checks; resolves the stale doc's OQ1 the opposite way | human (ruling: fail loud + point at `chatgpt/`) | design | low |
| D2: `chatgpt` client deadline = a package constant (default 10m) + `WithTimeout` option; no new env/flag surface in Phase 1 | #1259 owns the configurable routing (`ctx.AI.Timeout`/`--ai-timeout`); inventing a parallel env var now would split the mechanism | agent (number), human (mechanism, per ruling "see #1259") | design | low |
| D3: Phase 2 tool-calling shape — single-turn `Generate`; the codex agent loop (its own tools) is **one** AI-effect turn; `Step` with `req.Tools` fails `CodeToolsNotSupported` | The harness tool loop (`std/ai.ail` M-AI-TOOL-LOOP) cannot be honored by a CLI that executes its own tools in its own sandbox; injecting harness tool schemas would require the Responses wire shape — the retired route | human (ruling asks the doc to settle it; ratified with this doc) | design | high |
| D4: Phase 2 provider lives at the platform layer (`internal/executor/codex/aieffect`), injected into `factory.New` by host binaries — it must NOT live under `internal/ai/` | `internal/ai` is core and the language-closure gate (`TestLanguageCoreIsALeaf`) forbids core reaching `internal/executor` (platform); `internal/executor` imports `internal/ai`, so `internal/ai` importing it back is also a cycle | agent (mechanical boundary, V10/V11) | design | med |
| D5: One auth.json = one serialized stream: in-process mutex keyed by the auth path; fleet doctrine documented; Cloud Run `("codex","oauth")` template serialization verified in the ops repo | OpenAI's CI/CD condition is explicit ("only one machine or serialized job stream will use a given auth.json copy"); the cloud template installs a shared Secret Manager credential with write-back | human (fleet/ops part) | design | med |
| D6: Cost accounting — the lane is `AuthLaneSubscription` → `CostListPriceEquivalent`; real tokens, notional list-price dollars, quota via the existing provider-observation instrument; never `CostMetered` | Per Critical Principle 2, metered dollars on a subscription lane would be fabricated spend landing in the observatory | human (ruling asks the doc to settle it; ratified with this doc) | design | med |
| D7: `chatgpt/` is removed in the same release that ships the `codex:` provider — no dual-lane window; a stale `chatgpt/` model then fails loudly naming `codex:` | Keeping the unsanctioned route alive beside the sanctioned one leaves the defect the ruling closed | human (ruling: "replaces") | design | med |
| D8: `motoko-chatgpt-*` migration depends on measured usage: tool-loop callers cannot ride `codex:` (D3); they either move to a sanctioned metered lane or restructure to delegate the loop to codex | The motoko agent loop uses the harness tool loop (`std/ai.ail`), which D3 does not serve — silently migrating would break it | human (what the comparison row is *for*) | design | med |

### Design Freeze

Before implementation begins, these must be resolved (ratified when this doc is approved):

- [x] D3 ratified: single-turn `Generate`, `Step(tools)` refused — including the accepted consequence that harness tool loops cannot ride this lane — **Ruled 2026-10-08 by Mark: yes.**
- [x] D6 ratified: `CostListPriceEquivalent` + quota observation, never metered dollars — **Ruled 2026-10-08 by Mark: yes.**
- [x] D7 ratified: same-release replacement, no dual-lane window — **Ruled 2026-10-08 by Mark: yes.**
- [x] D8 ratified: the `motoko-chatgpt-gpt-6-1-sol` row's fate (delegate-to-codex restructure vs. metered-lane move) — needs one measurement of what that row's runs actually do with tools — **Ruled 2026-10-08 by Mark: deferred. Take the measurement first (P2.5) and bring the result back for a ruling; the executor must stop there, not choose.**

## Solution Design

### Overview

Two phases, exactly as ruled:

- **Phase 1 (v0.52.6, small, ships first):** stop the silent metered resolution (`ProviderCodex` sentinel + factory refusal), bound the `chatgpt/` client, extend the motoko preflight and auth-lane classification to the sentinel, and reconcile the executor README's metered-first auth advice with the fleet's subscription-first guard.
- **Phase 2 (v0.53.0):** a `codex:` AI-effect provider that drives `codex exec --json` through the **existing codex executor** (reuse, not reimplementation — the quorum's `agentic_provider.go` pattern), settling tool shape (D3), serialization (D5), and cost (D6); then migrate `motoko-chatgpt-*` and delete the direct-backend client (D7).

### Phase 1 — fail loud + deadline (v0.52.6)

**P1.1 `GuessProvider` stops claiming `codex*` → `ProviderOpenAI`.** The `strings.HasPrefix(lower, "codex")` arm moves out of the `gpt/o1/o3` case into its own early check returning the new sentinel `ProviderCodex` (V1). Same match shape as today, different target — the prefix-namespace conflict surface is unchanged (see Conflict Surface). `ProviderFromString("codex")` → `ProviderCodex`. `EnvVarForProvider(ProviderCodex)` returns `""` (the subscription lane reads no env key; codex owns `~/.codex/auth.json`).

**P1.2 The factory refuses the sentinel, loudly, in one place.** `factory.New`'s `ProviderCodex` case (phase 1) returns an error pinned in a test:

```
model %q: the codex: prefix is the ChatGPT-subscription lane, not a metered
OpenAI API model — the AI effect will not silently bill the metered lane
(issue #903). Until the codex exec provider lands, use a chatgpt/ model
(e.g. chatgpt/gpt-6.1-sol) for the subscription lane, the coordinator /
provider_executor lane for codex agent loops, or an explicit provider: row
in models.yml for a deliberately metered OpenAI model.
```

Both AI-effect construction paths route through `factory.New` (`cmd/ailang/ai_handlers.go:235→262` direct path, `internal/eval_harness/ai_provider.go:63→71` registry-fallback path — V7), so one error site covers both. Registry rows with an explicit `provider:` bypass guessing entirely (V8), so deliberately metered codex-family API models (e.g. `gpt5-2-codex`, `provider: "openai"`, V9) are unaffected.

**P1.3 The `chatgpt` client gets a deadline.** `chatgpt.NewClient` constructs `&http.Client{Timeout: defaultDeadline}` with `defaultDeadline = 10 * time.Minute` and a `WithTimeout` option (tests). Rationale: `http.Client.Timeout` bounds the whole exchange *including the streamed SSE body*, which is the point; 10m covers a maximal single-turn generation on this lane (65,536-token declared output ceiling at ≥110 tok/s ≈ 10m, V12) where #1259's proposed 30s default is aimed at single-shot `DoJSON`-class calls. Context deadlines (already threaded via `http.NewRequestWithContext`) still apply; the constant is the backstop. No new env/flag in Phase 1 — #1259 lands the configurable routing (`ctx.AI.Timeout` / `--ai-timeout`) and this client then reads it (Dependency note, D2).

**P1.4 Motoko preflight + auth lane learn the sentinel.** `internal/executor/motoko/provider_preflight.go` treats `ProviderCodex` exactly like `ProviderChatGPT` today (V13): check `chatgpt.LoadCredential()` (same `auth.json`), so a `codex:` motoko model fails at run setup with the login guidance instead of deep in handler construction. `motokoAuthLane` maps `ProviderCodex` → `executor.AuthLaneSubscription` (today a `codex:` motoko model is classified **billed** — the classification side of the same defect, V13).

**P1.5 Executor README auth advice reconciled.** `internal/executor/codex/README.md:93–100` recommends `OPENAI_API_KEY` in the coordinator environment as "the most reliable approach" (V14) — the metered lane, contradicting both the driver's billing guard (which unsets the key) and the cloud lane (Secret Manager subscription credential, V18). Rewrite to: subscription is the default lane (local `codex login --device-auth`; cloud `AILANG_CODEX_AUTH_SECRET`); `OPENAI_API_KEY`/`AILANG_AUTH_MODE=apikey` is the *explicitly metered* alternative.

**P1.6 Supersede the stale doc.** Banner + status change on `m-codex-billing-lane-resolution.md` (nothing on disk follows it — V2 — so this is documentation-only).

### Phase 2 — the `codex:` AI-effect provider (v0.53.0)

**P2.1 Architecture: reuse the executor, inject it at the platform layer.** The provider is a thin adapter over the **existing codex executor** (`internal/executor/codex`) — the same reuse the mission quorum already proved (`internal/mission/quorum/agentic_provider.go` wraps `coordinator.NewExecutorProvider("codex")` with bounded turns, read-only tool mode, cancellation, and observed cost — V19). It cannot live under `internal/ai/`: `internal/ai` is core, the language-closure gate forbids core reaching `internal/executor` (platform), and `internal/executor` imports `internal/ai` back (cycle) — V10/V11. So:

- New package `internal/executor/codex/aieffect` (platform layer): `New(opts...) (ai.Provider, error)` building a `CodexExecutor` via `codex.New(&executor.Config{...})` and shaping tasks the way `coordinator/provider_executor.go` shapes `Kind=="question"` (read-only `AllowedTools`, per-task model, timeout/idle clocks — replicated, not imported, because `provider_executor.go` lives in `internal/coordinator` (apps) and importing it from here would be a tools→apps dependency; the read-only set is five tool names, V19/V20).
- `factory.New` grows an injected-constructor option in the mould of `WithConfigDriven` (the M-AI-PROVIDER-CONFIG precedent, V22): the host binaries that host the AI effect (`cmd/ailang/ai_handlers.go` — covers `ailang run`, the REPL, and the motoko executor child, which runs this binary — plus `internal/eval_harness/ai_provider.go`, plus the apiserver if it hosts the effect; exact site list is a sprint-verification grep of `factory.New` callers, V7) pass the constructor. A caller that does not wire it gets a loud `codex: lane not registered in this build` error — never a fallback.

**P2.2 The provider contract (settles the issue's Q1 — D3).**

- `Generate(ctx, req)`: one `codex exec --json` run. The prompt is `req.SystemPrompt` + messages/user prompt; the model is the `codex:` pin stripped (e.g. `codex:gpt-6.1-sol` → `--model gpt-6.1-sol`); a bare `codex*` string without the pin passes through unchanged and fails loudly at the CLI. Result mapping: `Result.Output` → `Response.Text`, `InputTokens`/`OutputTokens`/`CacheReadInputTokens` → the response fields, `Success=false` → a typed `AIError` (never an empty-text success). The codex agent loop — codex's **own** tools under the read-only question set — is the turn.
- `Step(ctx, req)`: `len(req.Tools) > 0` → `ai.NewAIError(CodeToolsNotSupported, ...)` naming the alternatives (coordinator/provider_executor lane for tool-using agents; `Generate` for single-turn work). Without tools, `Step` = `Generate`.
- Images: `req.InputImages` → `CodeCapabilityNotSupported` (same as the `chatgpt` client today, V6).
- Streaming: not implemented initially — `ai.StreamingProvider` stays unimplemented (it is optional; the `chatgpt` client implements it, `Provider` alone suffices). Mapping the executor's turn events into `StreamChunk`s is Future Work.
- Wall-clock: timeout from the `ctx` deadline (as `agentic_provider.go` does) plus an idle-timeout default; the executor's existing process-tree kill (`procgroup`) handles hung codex trees — the reason to reuse it rather than spawn `exec.Command` ourselves (V19).

**P2.3 One auth.json, one stream (settles the issue's Q2 — D5).**

- **In process:** a package-level mutex keyed by the resolved auth path (`$CODEX_HOME/auth.json`, else `~/.codex/auth.json`) serializes every `Generate` — two concurrent AI-effect calls on one rig's login queue instead of racing on a credential file codex rewrites on refresh.
- **Fleet doctrine (documented, enforced where we own the knob):** one `CODEX_HOME` = one serialized job stream. The mission rig's rotation already treats it that way (one codex role fires at a time per mission; cross-mission serialization is the driver's state domain, noted as a Risk, not re-designed here).
- **Cloud Run (`("codex","oauth")` → `agent-executor-codex`, V23):** this template *does* mount subscription credentials — `installCodexCredential` installs a ChatGPT `auth.json` from Secret Manager and writes refreshed tokens back (`NewerRefresh`, newest-wins; V18) — so the issue's "may be naming-only today" suspicion is outdated, and the constraint is live: concurrent executions of that job template share one `auth.json` copy, violating OpenAI's stated condition. The terraform ground truth is in the ops repo (no `.tf` in this repo, V24), so Phase 2 ships with an **ops-repo task**: pin that job template to serial executions (or split per-stream secrets) and record the finding. Human decision point, tracked in Design Freeze adjacent — the in-repo half (mutex + doctrine + loud classification) does not wait on it.

**P2.4 Cost: quota lane, not dollars (settles the issue's Q3 — D6).**

- The lane is `AuthLaneSubscription` by construction: the auth check refuses an `auth_mode: "apikey"` auth.json (V17), and executor-lane probe knowledge (2026-07-30, codex-cli 0.145.0) holds that an env key does not override `auth_mode: chatgpt` — still, the provider refuses to construct when `OPENAI_API_KEY` is set **and** no chatgpt-mode auth.json exists, so the only silent path to metered billing is closed from both ends.
- `executor.AuthLaneForModel` grows `codex:`/`chatgpt/` arms → `AuthLaneSubscription` (today only Ollama-Cloud routes map there, V15), consolidating `motokoAuthLane`'s special case into the shared classifier. `ResolveCostProvenance` then yields `CostListPriceEquivalent` — "went through a subscription lane and was never billed" — the existing, defined provenance, not a new concept.
- Tokens are real (from the NDJSON `token_count` events, as the executor already parses them — V19). Dollars are list-price-**equivalent** arithmetic for comparability (the registry rows already declare them that way, V16); nobody is billed. Quota consumption is observed by the existing instrument — a real codex run writes the rollout records and `codex app-server account/rateLimits/read` that `mission quota`'s provider-observation lane consumes (M-CODEX-QUOTA-OBSERVATION, V25). No new instrument.
- Unpriced models keep `CostProvenanceUnknown` (existing `ResolveCostModel` behavior — a fabricated free run is the other silent fallback, V15).

**P2.5 Migration (D7/D8).**

1. `models.yml` `motoko-chatgpt-gpt-6-1-sol` (V16): `provider: "chatgpt"` → `"codex"`, `agent_model_name: "chatgpt/gpt-6.1-sol"` → `"codex:gpt-6.1-sol"`, notes updated. The registry **key** stays (it is an identifier pinned by configs and state files, not a routing prefix; future rows use `motoko-codex-*`). Whether to also rename the key with an alias window is Deferred.
2. The row's runs must be measured first (D8): its purpose is "motoko vs codex on the same model". If those runs use the harness tool loop (`std/ai` `Step` with tools — the motoko parser surfaces `tool_calls`, V26), they cannot ride `codex:` as-is; the row either (a) restructures to delegate the loop to codex (one `Generate`), or (b) moves to a sanctioned metered lane for the tool-loop arm and keeps `codex:` for single-turn arms. **This is the one migration decision that needs a human ruling on the row's purpose**, recorded before the sprint executes the flip.
3. Delete `internal/ai/chatgpt/client.go` (the direct-backend route). The read-only auth.json loader (`auth.go`: refuses apikey mode, refuses expired tokens — V17) moves into the new package; preflight's import follows.
4. `ProviderChatGPT` stays for exactly one release as a **deprecated marker**: `GuessProvider`'s `chatgpt/` arm still matches, but `factory.New`'s case returns the loud migration error ("`chatgpt/` was the direct-backend route, which OpenAI does not sanction; use `codex:gpt-6.1-sol`"). No silent reroute (D7). The constant and arm are deleted in v0.54.
5. `CanonicalQuotaBucket`'s `chatgpt` folding is **kept** — it canonicalizes stored agent_ids at read time, and history must still roll up (V27).
6. Changelog migration note, both releases.

### Conflict Surface (who else touches these code paths)

Not a parser/typechecker change. The shared machinery touched is **`GuessProvider`'s prefix namespace** and **`factory.New`'s switch**. Full consumer set of the `codex` arm, with the effect of the change on each:

| Caller | Site | Today (`codex*` → openai) | After Phase 1 | Correct? |
|---|---|---|---|---|
| AI-effect direct path | `cmd/ailang/ai_handlers.go:235` | builds metered OpenAI client | sentinel → factory ruling error | **the fix** (A1) |
| AI-effect registry fallback | `internal/eval_harness/ai_provider.go:65` | ditto | ditto | **the fix** (A2) |
| Motoko preflight | `internal/executor/motoko/provider_preflight.go:30` | demands `OPENAI_API_KEY` (metered!) | `LoadCredential` check, `AuthLaneSubscription` (P1.4) | **the fix** (classification side) |
| Executor env policy | `internal/executor/envpolicy.go:268` (`ProviderCredentialVars`) | would inherit `OPENAI_API_KEY` for a `codex:` `ai_provider` policy value | `EnvVarForProvider(ProviderCodex)` = `""` → inherits nothing | correct (subscription lane reads no env key; no repo policy uses `codex:` today — V28) |
| Mission executor `codex:gpt-*` pins | modelreg `agent_cli: codex` rows → `GetExecutorForModel` → `coordinator.NewExecutorProvider("codex")` → `internal/executor/codex` | **never reaches `GuessProvider`** (spawn-recipe pins, V29) | unchanged | unaffected (A4) — the audit the ruling asked for |
| Registry rows, explicit provider | e.g. `gpt5-2-codex` `provider: "openai"` (V9) | guessing bypassed | unchanged | unaffected (deliberate metered use stays possible) |
| Dashboard test data | `internal/dashboard_transforms/approval_authority.ail:76` string `"codex:gpt-5.6-sol"` | pure data, no AI call (V30) | unchanged | unaffected |

**Prefix disambiguation (unchanged match order):** `ollama:`/`ollama/` → ollama; `chatgpt/` → chatgpt; `openrouter:` → openrouter; `vendor/model` (known vendors) → openrouter; then the bare-prefix switch, where `codex*` now yields the sentinel instead of openai. `"codex/..."` (slash, no known vendor) still falls through to the generic unknown-provider error — loud, unchanged. `gpt-5.2-codex`-shaped names still match the `gpt` arm → openai (correct: they are metered API model names). `"codellama"` never matched the `codex` arm (`code` ≠ `codex`, V31).

**Must still work (fixtures):** `ollama:qwen3.5:35b` and `ollama/qwen3.5` routing; `chatgpt/gpt-6.1-sol` (until P2.5 flips it to the deprecation error — then *that* error is the pinned behavior); `anthropic/claude-sonnet-4.5` → openrouter; `gpt-5.1`, `o3-mini`, `claude-…`, `gemini-…` guesses; `TestGuessProvider`'s remaining rows; the `gpt5-2-codex` registry row path; `approval_authority.ail` tests.

**Deliberately changes:** `{"codex-max", ProviderOpenAI}` and every `codex*` AI-effect guess (silent metered → loud sentinel error); `codex:` motoko preflight (metered-key demand → subscription-credential check); `chatgpt/` in v0.53.0 (live provider → loud deprecation error naming `codex:`).

### Implementation Plan

**Phase 1: fail loud + deadline** (~1 day)
- [ ] `internal/ai/config.go`: `ProviderCodex` sentinel; move the `codex` arm; `ProviderFromString("codex")`; `EnvVarForProvider` comment row
- [ ] `internal/ai/factory/factory.go`: `ProviderCodex` case → the P1.2 error (message pinned in a test)
- [ ] `internal/ai/chatgpt/client.go`: `defaultDeadline` + `WithTimeout`
- [ ] `internal/executor/motoko/provider_preflight.go`: `ProviderCodex` like `ProviderChatGPT`; `motokoAuthLane` arm
- [ ] `internal/ai/provider_test.go`: `{"codex-max", ProviderOpenAI}` → `{"codex-max", ProviderCodex}`, `{"codex:gpt-6.1-sol", ProviderCodex}` (V3)
- [ ] `internal/executor/codex/README.md`: subscription-first auth section (P1.5)
- [ ] Stale-doc supersession banner (P1.6); changelog entry
- [ ] Acceptance A1–A5; `make test`, `make fmt`, `make lint`, `make check-boundaries`, `make check-architecture-closure` (the sentinel adds no import — the closure must stay clean)

**Phase 2: the provider + migration** (~4 days)
- [ ] `internal/executor/codex/aieffect`: provider (P2.2), read-only question tool set, auth-loader moved from `internal/ai/chatgpt/auth.go`, process-wide auth-path mutex (P2.3)
- [ ] `internal/ai/factory`: injected-constructor option (P2.1); wire at `cmd/ailang/ai_handlers.go`, `internal/eval_harness/ai_provider.go`, apiserver if applicable; unwired → loud error
- [ ] `executor.AuthLaneForModel` codex/chatgpt arms; consolidate `motokoAuthLane` (P2.4)
- [ ] D8 measurement: what `motoko-chatgpt-gpt-6-1-sol` runs actually do with tools → human ruling → `models.yml` migration (P2.5)
- [ ] Delete `internal/ai/chatgpt/client.go`; `ProviderChatGPT` deprecation error; keep `CanonicalQuotaBucket` folding
- [ ] Ops-repo task filed: serialize `agent-executor-codex` (oauth) executions or split secrets (P2.3) — linked from the sprint report
- [ ] Live probe: `codex:gpt-6.1-sol` via the AI effect with `OPENAI_API_KEY` stripped (the mission-control guard shape) — rc=0, subscription auth, `CostListPriceEquivalent` in the observatory (A6/A7)
- [ ] Docs: `docs/docs/guides/ai-routing.md`, `docs/docs/guides/mission-model-fleet.md` (the `chatgpt/`→`codex:` flip); changelog migration note

### Files to Modify/Create

**New files:**
- `internal/executor/codex/aieffect/provider.go` — the AI-effect adapter over the codex executor, ~180 LOC
- `internal/executor/codex/aieffect/auth.go` — the read-only auth.json loader (moved from `internal/ai/chatgpt/auth.go`), ~90 LOC
- `internal/executor/codex/aieffect/provider_test.go` — fake-executor seam (the `codexAppServerCall` var-seam pattern), ~200 LOC

**Modified files:**
- `internal/ai/config.go` — sentinel + arm move, ~10 LOC
- `internal/ai/factory/factory.go` — refusal case, then injected constructor, ~30 LOC
- `internal/ai/chatgpt/client.go` — deadline (Phase 1); deleted in Phase 2
- `internal/executor/motoko/provider_preflight.go` — sentinel handling, ~10 LOC
- `internal/executor/cost.go` — `AuthLaneForModel` arms, ~8 LOC
- `internal/ai/provider_test.go`, `internal/ai/factory/factory_test.go`, `internal/executor/motoko/provider_preflight_test.go` — fixtures, ~120 LOC
- `internal/modelreg/models.yml` — the motoko row migration, ~6 LOC
- `cmd/ailang/ai_handlers.go`, `internal/eval_harness/ai_provider.go` — constructor wiring, ~10 LOC
- `internal/executor/codex/README.md`, `docs/docs/guides/ai-routing.md`, `docs/docs/guides/mission-model-fleet.md`, `changelogs/` — docs, ~60 LOC
- `design_docs/planned/m-codex-billing-lane-resolution.md` — supersession banner (Phase 1)

## Examples

### Example 1: the Phase 1 flip (A1/A2)

**Before** (v0.52.5):
```
$ ailang run -e 'ai_generate("codex:gpt-6.1-sol", "hi")'   # OPENAI_API_KEY present in env
→ 200 OK from api.openai.com …   # silently billed METERED API credits
```

**After** (v0.52.6):
```
$ ailang run -e '…("codex:gpt-6.1-sol", "hi")…'
Error: model "codex:gpt-6.1-sol": the codex: prefix is the ChatGPT-subscription lane,
not a metered OpenAI API model — the AI effect will not silently bill the metered lane
(issue #903). Until the codex exec provider lands, use a chatgpt/ model …
```

### Example 2: Phase 2, the sanctioned lane under the billing guard

```
$ unset OPENAI_API_KEY   # mission-control guard shape: subscription-or-nothing
$ ailang run agent.ail   # agent.model = "codex:gpt-6.1-sol"
→ codex exec --json --skip-git-repo-check --model gpt-6.1-sol …
→ NDJSON adapted: Text, InputTokens/OutputTokens real, FinishReason stop
→ observatory row: auth_lane=subscription, cost_provenance=list_price_equivalent
$ ailang mission quota
→ codex bucket refreshed from the rollout the run just wrote (no new instrument)
```

### Example 3: the tool-loop refusal is loud, with the exit named (D3)

```
$ motoko run --model codex:gpt-6.1-sol   # a harness tool loop calls Step with tools
Error [ToolsNotSupported]: the codex: lane runs codex's own agent loop as one turn;
harness-declared tools cannot be injected into codex exec. Use the coordinator /
provider_executor lane for tool-using agents, or Generate for single-turn work.
```

## Success Criteria

- [ ] **A1**: `GuessProvider("codex:gpt-6.1-sol")`, `GuessProvider("codex-max")` → `ProviderCodex` (unit, updated `TestGuessProvider`)
- [ ] **A2**: AI-effect construction with a `codex*` model returns the pinned ruling error on both the direct and registry-fallback paths (factory test + handler-level test)
- [ ] **A3**: `chatgpt.NewClient()` has `httpClient.Timeout > 0` by default; `WithTimeout` overrides; a server that streams forever is cut at the deadline (httptest)
- [ ] **A4**: mission-executor `codex:gpt-*` pins unaffected — coordinator executor path regression (the spawn-recipe fixtures) plus the audit table above re-verified by the executor
- [ ] **A5**: `gpt5-2-codex`-row (explicit `provider: openai`) metered use still constructs the OpenAI client
- [ ] **A6**: Phase 2 live probe with `OPENAI_API_KEY` stripped: `codex:` model succeeds via `codex exec`, subscription auth logged
- [ ] **A7**: the probe's cost row is `CostListPriceEquivalent` / `auth_lane=subscription`, tokens real, never `CostMetered`
- [ ] **A8**: `Step` with tools → `CodeToolsNotSupported` with the exit named; `chatgpt/` after Phase 2 → the deprecation error naming `codex:`
- [ ] All tests passing (`make test`, `make test-core`), `make fmt`/`make lint`/`make check-boundaries`/`make check-architecture-closure` clean
- [ ] Documentation updated (README auth section, ai-routing, mission-model-fleet, changelog migration notes both releases)

## Testing Strategy

**Unit tests:**
- `TestGuessProvider` rows (A1); factory refusal message pinned (A2); `EnvVarForProvider(ProviderCodex) == ""`
- chatgpt deadline: default > 0, option override, hung-stream cut (A3) — a `httptest` server that writes SSE deltas forever
- `AuthLaneForModel("codex:…")`, `("chatgpt/…")` → `AuthLaneSubscription`; unpriced → `CostProvenanceUnknown`
- aieffect provider against a fake executor (var seam): success mapping, `Success=false` → typed error, tools refusal, images refusal, bare-model passthrough, mutex serialization (two concurrent Generates observe lock ordering)

**Integration tests:**
- motoko preflight with a temp `CODEX_HOME`: chatgpt-mode auth.json admits; apikey-mode refuses; missing refuses with the login command
- registry-row precedence: explicit `provider: openai` bypasses the sentinel (A5)
- handler-level: a `codex:` model through `setupAIHandlerDirect` surfaces the factory error verbatim

**Manual testing:**
- the A6 live probe (subscription lane, key stripped), including `mission quota` refresh afterwards
- one motoko run on the migrated row post-D8-ruling, whichever arm the ruling picks

## Deferred Decisions

- The exact deadline constant (10m proposed; 5–15m defensible) — agent may tune with a live probe before ship; the property that matters (bounded by default) is pinned by test.
- The injected-constructor option's exact shape (`WithCodexLane(ctor)` vs a general `WithExecutorLane(name, ctor)`) — agent may choose; generalizing is justified only if a second executor-backed lane is imminent.
- Whether to rename the `motoko-chatgpt-gpt-6-1-sol` registry key with an alias window — agent may decide after grepping external references; values migrate regardless.
- Streaming (`StreamingProvider`) for the codex lane — deferred until a caller needs chunk callbacks; the executor's event handler is the seam.
- Whether the aieffect provider ever exposes codex's *write* tools (a workspace + full tool set) — deliberately not in scope; the question-set default is the safe shape and widening is its own attended ruling.

## Non-Goals

- **Not reimplementing codex OAuth against the backend** — the retired `chatgpt/` route; this doc deletes it (the ruling's core).
- **Not supporting harness tool loops on the codex lane** (D3) — the AI-effect tool contract (`std/ai.ail` M-AI-TOOL-LOOP: host dispatches tool calls between steps) cannot be honored by a CLI that runs its own tools; an MCP-bridge pattern (a `.ail` program serving its tools to codex via `task.MCPServers`) is Future Work, not Phase 2.
- **Not changing the executor lane** — `internal/executor/codex`, its auth bootstrap, or the driver's billing guard are reused, not modified (beyond the README correction, which is docs).
- **Not implementing #1259's general `ctx.AI.Timeout`/`--ai-timeout`** — that issue owns the mechanism; Phase 1 only bounds the one client the ruling named, in a way #1259 can absorb.
- **Not metering subscription runs in dollars** — list-price-equivalent is the ceiling of what this lane reports (D6).

## Timeline

**Week 1** (1 day): Phase 1 — sentinel, factory refusal, deadline, preflight, README, supersession banner; A1–A5 green; ship v0.52.6.

**Weeks 2–3** (4 days): Phase 2 — aieffect provider + wiring (2d), auth-lane consolidation + migration + D8 ruling (1d), deletion of the direct-backend client + docs + live probe + ops task (1d); ship v0.53.0.

**Total: ~5 days across ~3 weeks.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| A host binary forgets to wire the injected constructor; `codex:` fails there | Med | Fail-loud by design (named build error); wiring is 2–3 named sites (V7); the error names the fix |
| Codex CLI NDJSON schema drift breaks the adapter | Med | The adapter rides the *executor's* parser, which the mission fleet exercises continuously — drift surfaces there first, not only in the AI effect |
| Concurrent Cloud Run oauth-codex executions share one Secret Manager auth.json (OpenAI condition) | Med | Ops-repo serialization task shipped with Phase 2 (P2.3); in-repo mutex + doctrine now; `NewerRefresh` newest-wins write-back already tolerates racing refreshes |
| Motoko tool-loop runs silently break after the row migration | High | D8 is a freeze-gated human ruling with a measurement *before* the flip; the tool refusal is loud (A8), so breakage is visible, not quiet |
| Someone relied on bare `codex-max`-style guessing for a real metered model | Low | Such a model needs an explicit registry row (D3 of the superseded doc, carried forward); the changelog names the migration |
| The 10m deadline cuts a legitimate maximal generation | Low | Option override + #1259's future config; the ceiling math is recorded (V12); tuned by probe before ship |

## Related Documents

**Implemented (may inform design):**
- `design_docs/implemented/v0_15_0/m-exec-expand-codex-opencode.md` — the codex executor lane and auth-bootstrap pattern this design reuses
- M-MODEL-REGISTRY-SINGLE-SOURCE (implemented v0.35.0) — explicit `provider:` rows bypass guessing; carried forward here

**Planned (check for overlap):**
- `design_docs/planned/m-codex-billing-lane-resolution.md` — **superseded by this doc** (banner added in Phase 1); its fail-loud analysis (D1–D3, D5) is carried forward, its "never build the provider" ruling (D4) is reversed by the maintainer ruling of 2026-10-08
- `design_docs/planned/m-codex-quota-observation.md` — the quota instrument this design feeds (a codex run refreshes the provider observation); no overlap, complementary
- `design_docs/planned/ailang-core-triage/test-executor-module-env-miswiring.md` — cites #1259; the deadline mechanism this doc defers to

## References

- [Design Axioms](/docs/references/axioms)
- Issue [#903](https://github.com/sunholo-data/ailang/issues/903) — source report, research (SANCTIONED / DISCOURAGED / NOT SANCTIONED classification, verified 2026-08-16), and the 2026-10-08 maintainer ruling (option 3)
- Issue [#1259](https://github.com/sunholo-data/ailang/issues/1259) — the AI-effect HTTP deadline umbrella
- OpenAI sources (via #903): Codex authentication (developers.openai.com/codex/auth), CI/CD account auth (the "one serialized job stream" condition), Using Codex with your ChatGPT plan (help.openai.com)
- `internal/mission/quorum/agentic_provider.go` — the executor-as-provider reuse pattern

## Verification Log

All rows verified by code reads / greps in this checkout (`c3bfb1c0`, shallow depth 1) on 2026-10-08.

| # | Claim | Evidence |
|---|-------|----------|
| V1 | `GuessProvider` maps the `codex` prefix to `ProviderOpenAI` | Read `internal/ai/config.go:86–93`: the `gpt`/`o1`/`o3`/`codex` case returns `ProviderOpenAI` |
| V2 | The stale doc is planned, never implemented | `design_docs/planned/m-codex-billing-lane-resolution.md` status "Planned", target v0.39.0; `grep -rn "m-codex-billing-lane-resolution" changelogs/ design_docs/README.md` → no hits (nothing shipped or moved); its D1 (remove the arm) is not at HEAD (V1) |
| V3 | A test pins the defect | Read `internal/ai/provider_test.go:113–125`: `TestGuessProvider` row `{"codex-max", ProviderOpenAI}` |
| V4 | The chatgpt client has no timeout | Read `internal/ai/chatgpt/client.go` `NewClient`: `httpClient: &http.Client{}` (no `Timeout`) |
| V5 | #1259 is open and its proposal is unimplemented for std/ai | GitHub API: state `open`; body cites `internal/ai/httpjson.go:71` and `internal/ai/openrouter/client.go:124` falling back to `http.DefaultClient`; `grep -rn "ai-timeout\|AI.Timeout\|AITimeout" internal/ cmd/` → only the messaging embedder's OpenAI timeout, nothing in `internal/ai` |
| V6 | The chatgpt/ route is the direct backend | Read `internal/ai/chatgpt/client.go`: `DefaultBaseURL = "https://chatgpt.com/backend-api/codex"`; `originator: "ailang"` header (the CLI-mimic shape); added by commit `07492bb2c` per the ruling text (history not re-derived — this checkout is shallow, depth 1) |
| V7 | Both AI-effect construction paths go through `factory.New` | Read `cmd/ailang/ai_handlers.go:235` (`GuessProvider`) and `:262` (`factory.New`); `internal/eval_harness/ai_provider.go:65` (guess) and `:71` (`factory.New`); the factory doc names its caller set (exec.go, ai_handlers ×2, eval_harness, mission/quorum/call.go, coordinator_lifecycle.go) |
| V8 | Explicit registry `provider:` bypasses guessing | Read `internal/eval_harness/ai_provider.go:57–65`: `explicitProvider` non-empty skips `GuessProvider` |
| V9 | `gpt5-2-codex` is an explicit openai row | Read `internal/modelreg/models.yml:745–751`: `api_name: "gpt-5.2-codex"`, `provider: "openai"`, `env_var: "OPENAI_API_KEY"` |
| V10 | `internal/ai` is core; the closure gate forbids core reaching `internal/executor` | Read `tools/simplicity_metrics.sh:47–57`: `LANGUAGE_ROOTS` includes `internal/effects`; `PLATFORM_PKGS` includes `internal/executor`; `internal/diag/closure_test.go` `TestLanguageCoreIsALeaf` runs `go list -deps` over the roots and fails on any platform package; `ARCHITECTURE.md` layer table: "(`internal/ai` is core: the AI effect is a language feature)" |
| V11 | `internal/executor` imports `internal/ai` (so `internal/ai` cannot import it back) | `grep internal/executor/envpolicy.go:10` imports `internal/ai`; `internal/coordinator/task_executor.go:10` also imports `internal/ai` (rules out the coordinator too) |
| V12 | The lane's declared output ceiling is 65,536 tokens | Read `internal/modelreg/models.yml` `motoko-chatgpt-gpt-6-1-sol` row: `max_output_tokens: 65536` |
| V13 | Motoko preflight special-cases chatgpt today; `codex:` today classifies as billed | Read `internal/executor/motoko/provider_preflight.go:52–53` (`ProviderChatGPT` → `chatgpt.LoadCredential`), `:75–76` (`motokoAuthLane`: only chatgpt maps to `AuthLaneSubscription`; a `codex:` model guesses openai → `AuthLaneForModel` → `AuthLaneBilled`) |
| V14 | The executor README recommends the metered key as most reliable | Read `internal/executor/codex/README.md:93–100`: "OPENAI_API_KEY in the process environment is the most reliable approach" |
| V15 | The subscription provenance taxonomy exists; `AuthLaneForModel` knows only ollama-cloud | Read `internal/executor/cost.go:186–218` (`AuthLaneSubscription` → `CostListPriceEquivalent`; zero-rate → `CostFreeLocal`; unpriced → `CostProvenanceUnknown` via `ResolveCostModel`/`Unpriced`), `:266–288` (`AuthLaneForModel`: ollama-cloud only) |
| V16 | The motoko-chatgpt row exists and is declared list-price-equivalent | Read `internal/modelreg/models.yml:2356–`: `motoko-chatgpt-gpt-6-1-sol`, `provider: "chatgpt"`, `agent_model_name: "chatgpt/gpt-6.1-sol"`, pricing comment "list-price-EQUIVALENT on this lane (subscription)" |
| V17 | The auth loader refuses apikey-mode and expired tokens | Read `internal/ai/chatgpt/auth.go` `LoadCredential`: `auth_mode != "chatgpt"` → error; JWT expiry check; `AuthPath` honors `CODEX_HOME` |
| V18 | Cloud codex installs a shared subscription credential with write-back | Read `internal/executor/codex/subscription_auth.go` (header: Secret Manager, dedicated token family, write-back or stale) + `NewerRefresh`; `cmd/ailang/coordinator_cloud_codexauth.go:33–56` `installCodexCredential` (installs from `AILANG_CODEX_AUTH_SECRET`, fails loudly without it unless `AILANG_AUTH_MODE=apikey`) |
| V19 | The quorum already wraps the codex executor as a generation provider with the properties this design reuses | Read `internal/mission/quorum/agentic_provider.go`: `coordinator.NewExecutorProvider("codex")`, `Kind: "question"` (read-only tools), ctx-deadline timeout + idle timeout, cancellation, `res.Cost`, token mapping |
| V20 | The read-only question tool set is five tools, defined in the coordinator (apps layer) | Read `internal/coordinator/provider_executor.go:141–151` (`questionTools` per executor) — replicated in the new package because importing `internal/coordinator` from an executor-adjacent package would be a tools→apps dependency |
| V21 | The error codes used are existing and allocated | `grep internal/ai/errors.go:36` `CodeToolsNotSupported = "ToolsNotSupported"`; the chatgpt client already uses `CodeCapabilityNotSupported`/`CodeAuthFailed`/`CodeProtocolError` — no new code is allocated by this doc |
| V22 | The injected-constructor precedent is `WithConfigDriven` | Read `internal/ai/factory/factory.go` `WithConfigDriven` option + `New`'s config-driven fallback branch |
| V23 | `("codex","oauth")` selects the `agent-executor-codex` job | Read `internal/dispatch/cloudrun/dispatcher.go:170–186` `jobSuffixForVariant` |
| V24 | No terraform in this repo (ops ground truth) | `grep -rn --include=*.tf .` → no hits; the dispatcher comments point at "the Terraform-defined job name in cloud_run_jobs.tf" (ops repo) |
| V25 | Quota observation consumes rollout records / app-server, refreshed by real codex runs | Read `internal/mission/codex_app_server_quota.go` (header: app-server `account/rateLimits/read`, session scan fallback) and `design_docs/planned/m-codex-quota-observation.md` (contract: a fresh ordinary codex call refreshes records) |
| V26 | Motoko's agent loop uses the harness tool loop | Read `std/ai.ail:54–91` (M-AI-TOOL-LOOP: "The host dispatches it… then feeds the result back"); `internal/executor/motoko/parser.go:49–88` (session events carry `tool_calls`) — the D8 consequence is real, not hypothetical |
| V27 | Quota-bucket folding must keep chatgpt for history | Read `internal/observatory/mission_rollup.go:215–233`: `CanonicalQuotaBucket` folds `chatgpt`/`codex`/`gpt-*` → `"codex"` at READ time over stored agent_ids |
| V28 | No repo policy/config uses `codex:` as an `ai_provider` value | `grep -rn "codex:" --include=*.yml --include=*.yaml --include=*.toml --include=*.ail internal/ std/ examples/ cmd/` → only `approval_authority.ail` test data (V30) and the `gpt5-2-codex` row name (a registry key, not a policy value) |
| V29 | Mission `codex:gpt-*` pins never reach `GuessProvider` | Read `internal/modelreg/identity.go:66` (`case "codex"` in the executor-vendor switch) and `models.yml` `agent_cli: "codex"` rows (e.g. `gpt6-1-sol`, line 577); `.agents/skills/mission-control/resources/gate-3-route.md:98`: "`provider:model` values signal cross-provider spawn recipes"; `agentic_provider.go` constructs via `coordinator.NewExecutorProvider("codex")` |
| V30 | The `approval_authority.ail` codex string is pure data | Read `internal/dashboard_transforms/approval_authority.ail:70–82`: `rungIsTrusted` is a pure string-contains test; no AI call |
| V31 | `"codellama"` never matched the codex arm | Prefix is `strings.HasPrefix(lower, "codex")`; `"codellama"` starts `"code"` — no match; verified against the arm at `config.go:86–93` |
| V32 | `ProviderFromString` maps `"chatgpt"` today | Read `internal/ai/config.go:175`; the `chatgpt/` GuessProvider arm at `:66–68` |
| V33 | No language-surface or new-diagnostic claims | The doc makes no "AILANG does/does not support X" statement; all changes are Go provider/factory/CLI behaviour; no MOD/PAR/TC/EFF code is allocated (V21) |

## Future Work

- `StreamingProvider` for the codex lane (map executor turn/token events onto `StreamChunk`s) — when a caller needs chunk callbacks.
- MCP-bridge pattern: a `.ail` program serving its own tools to codex via `task.MCPServers`, giving harness tools a sanctioned path into the codex loop without provider-level tool injection.
- Write-tool exposure (workspace + full codex tool set) behind its own attended ruling.
- `ProviderChatGPT` constant and `chatgpt/` arm deletion (v0.54, after the deprecation release).
- #1259's general `ctx.AI.Timeout` / `--ai-timeout` — absorbs the Phase 1 deadline constant.

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08

## Operational addendum: three OAuth profiles on one account (2026-10-09)

**Status:** Operational addendum implemented and validated in v0.53.1 on 2026-10-09; independent evaluation passed (97/100). The parent provider migration remains planned.
**Rollout evidence:** [completed operational sprint](m-codex-oauth-profiles-sprint-plan.md#execution-evidence-2026-10-09).
**Priority:** P1. **Estimated:** 2–4 days including cloud rollout and fault tests.
**Scope:** Credential ownership for interactive use, local missions, and cloud
executions. This closes the fleet-auth follow-up explicitly deferred by Phase 1;
it does not implement the AI-effect provider or choose the deferred D8 migration.

### User requirement and diagnosis

Mark requests one ChatGPT OAuth account for all three operational profiles.
Retain subscription authentication and the existing shared quota accounting.
Separate OAuth authorizations and refresh state; creating additional accounts
or switching automation to metered API billing is outside this request.

Observed on 2026-10-09:

| Evidence | Finding and limit |
| --- | --- |
| Local Codex auth logs | First failure at 15:28 Copenhagen; server code `refresh_token_expired`. No observed `refresh_token_reused` diagnostic. This does not prove a refresh collision. |
| Mission profiles, launchd plists, and secrets.env | No CODEX_HOME override found. Mission probes/controllers use the default local home. Fleet ran Codex controller/author roles at 07:43. |
| PR #1731, merged 09:35:32 UTC | Phase 1 routing/timeout changes; fleet serialization remains follow-up work. |
| Cloud Run codex and codex-go jobs | Both reference `ailang-codex-auth-json`; taskCount=1 per execution does not serialize separate executions. |
| Secret Manager version metadata | Only version 1, created September 30. No refresh write-back version observed; this alone does not establish current validity or task activity. |
| install/persist code | Restore happens before preflight; persist compares timestamps then adds a version. These checks do not hold a distributed lease through execution. |
| Codex 0.162.0 CLI help | Shared daemon and `app-server proxy` are available. Availability does not prove that every exec/quota caller shares one auth manager. |

### Profile contract

| Profile | Credential location | Ownership |
| --- | --- | --- |
| Interactive | Existing `~/.codex` | Existing interactive login and runtime; do not migrate active sessions. |
| Local missions | Proposed `~/.codex-missions` | Fresh login to the same account; all mission profiles and nested executors inherit this home. |
| Cloud | Existing secret `ailang-codex-auth-json`, restored to explicit container CODEX_HOME | Fresh cloud-only login to the same account, with exclusive execution ownership across job variants. |

The homes are distinct directories, not symlinks. Run `codex login` independently
for the mission and cloud profiles. Never seed either from the interactive
auth.json. Codex TOML model profiles do not substitute for separate auth homes.
Use file-backed credential storage for the automation roots, permissions 0700
for directories and 0600 for credential files. Provision audited configuration,
skills/plugins, MCP definitions and trust settings separately from credentials;
moving CODEX_HOME without those resources could change mission behavior.

### Refresh ownership and concurrency

**Local:** Prefer one managed daemon for the mission home, preserving concurrent
mission threads while one auth manager owns refresh. Route quota RPC through
`codex app-server proxy` to that same daemon, rather than launching an independent
app-server against its auth.json. Verify exec, controller, probes, nested roles,
and quota RPC actually converge on that owner before enabling concurrent use.
Existing `internal/ai/chatgpt` remains a read-only credential consumer.

This requires a measured implementation spike with an isolated temporary home,
fake rotating-token endpoint and concurrent requests. Do not assert that merely
starting a daemon solves the problem. If installed CLI exec cannot use the same
owner, use an explicit app-server execution adapter or bring a serialized-mode
tradeoff back for approval. Do not add a non-reentrant whole-mission lock that
deadlocks when a controller invokes a nested role.

**Cloud:** Acquire a distributed exclusive credential lease keyed by canonical
project/secret identity BEFORE fetching auth. Hold it across installation,
health/preflight, execution and refreshed-secret persistence. Codex and codex-go,
retries, and every subscription execution entry point share this key. A
process-local mutex and taskCount=1 are insufficient.

Use the existing Firestore storage boundary for transactional lease ownership.
Heartbeat and cancellation must stop the Codex process tree on lease loss. Lease
expiry alone is not fencing: a stale live worker must not continue refreshing.
Before takeover, verify termination of the previous Cloud Run execution; if that
cannot be verified, block rather than issue the same rotating credential twice.
Use owner/generation checks for write-back and release. Persist the refreshed
file before releasing ownership, including failed tasks that refreshed first.
Write-back failure blocks subsequent use of the old secret and raises an
operational recovery requirement. Do not log credentials or silently resume
from an obsolete seed. No change to the OAuth wire flow: Codex owns refresh.

### Implementation and rollout

1. Add one attended provisioning/check command under `tools/attended/` for the
   three roots. Validate subscription mode, equal account/workspace identity,
   and distinct refresh credentials locally; output only pass/fail and profile
   labels. Do not expose tokens, emails, account IDs or token digests in reports.
2. Add a tested shared mission-home selection to the launchd driver and installer;
   preserve explicit operator overrides. Quota and executor helpers resolve the
   same home. Audit pinned driver copies and local coordinator/eval entry points.
3. Complete the daemon ownership spike, then implement tested quota RPC routing
   and any execution changes it demonstrates are necessary.
4. Implement the cloud ownership lease at the execute-job boundary, covering all
   restore/run/persist paths; update IAM and infrastructure in their owning repo.
5. At an idle boundary, obtain two user-completed OAuth logins to the same account.
   Validate identity/isolation before publishing the cloud secret. Activate only
   after code, daemon, infrastructure and provisioning checks pass.
6. Smoke-test local missions plus an interactive turn, then cloud codex and
   codex-go executions. Rollback must keep the isolated homes or pause automation;
   restoring shared interactive credentials would reintroduce the defect.

### Acceptance criteria and validation commands

- [x] `bash tools/attended/check_codex_oauth_profiles.sh` reports same account and
  workspace, distinct credentials, subscription mode, and file permissions.
  Implemented and passed before cloud handoff; the staging seed was intentionally
  retired after durable publication and rotation.
- [x] New `tools/launchd/test_codex_auth_profiles.sh` exercises all mission profiles,
  probes, nested roles and quota reads, including missing-auth refusal.
- [x] A focused executor/mission test proves concurrent local requests use one
  refresh owner and a simulated rotation yields one refresh and valid later calls.
- [x] Cloud credential lease tests exercise codex versus codex-go overlap,
  preflight refresh, cancellation, process termination, lease loss, crash recovery,
  failed persistence, stale writer rejection and release after durable write-back.
- [x] `go test ./internal/executor/... ./internal/mission/... ./cmd/ailang/...`
  and `make check-boundaries` pass. Add lease/storage tests in the owning package.
- [x] Deployment smoke evidence records the profile labels and success outcomes
  without token values; cloud version metadata advances after a forced test refresh.
- [x] Documentation explains one account/shared subscription limits and three
  independent authorizations. Authentication failures remain explicit.

### Axiom compliance for this operational extension

| Axiom | Score | Reason |
| --- | --- | --- |
| A1 Determinism | 0 | Operational scheduling only; language evaluation unchanged. |
| A2 Replayability | +1 | Ownership/generation and profile labels provide auditable lifecycle evidence. |
| A3 Explicit effects | 0 | Auth/network writes remain explicit host operations. |
| A4 Explicit authority | +1 | Runtime-specific credentials replace shared interactive authority. |
| A5 Bounded verification | +1 | Fake rotating-token tests and bounded lease failure tests. |
| A6 Safe concurrency | +1 | One refresh owner per OAuth authorization. |
| A7 Machines first | +1 | Check command gives explicit machine-readable outcomes. |
| A8 Minimal syntax | 0 | No language syntax change. |
| A9 Cost visibility | 0 | Existing subscription provenance and account quota remain. |
| A10 Composability | +1 | Reuses Codex auth and existing storage/executor boundaries. |
| A11 Structured failure | +1 | Lease loss and failed persistence block credential reuse. |
| A12 System boundary | 0 | Host integrations remain outside the language core. |

Net +7; no hard-gate negative. Scores are proposed design judgments, not
implementation evidence.

### References and remaining uncertainty

- [Official Codex CI/CD auth guidance](https://learn.chatgpt.com/docs/auth/ci-cd-auth):
  per-runner/serialized credential streams, refreshed-file persistence, and refresh
  failure recovery. This is operational guidance, not a blanket account-policy ruling.
- [Official accounts/sessions guidance](https://developers.openai.com/siwc/token-sharing-open-source/profiles-and-sessions):
  serialize refreshes for the same renewable session. Its custom-client flow is
  not a replacement for Codex CLI's login implementation.
- Local code: `tools/launchd/mission-control.sh`, `tools/launchd/lib/lane-probe.sh`,
  `internal/mission/codex_app_server_quota.go`,
  `cmd/ailang/coordinator_cloud_codexauth.go`,
  `internal/executor/codex/subscription_auth.go`,
  `internal/executor/envpolicy.go` (CODEX_ inheritance),
  `internal/ai/chatgpt/auth.go` (read-only consumer).
- The previous cloud-login comment says a dedicated token family is intended;
  the historical seed identity was not established and that seed has been replaced.
  Fresh independent cloud and mission authorizations passed the same-account and
  workspace provisioning checker before handoff.
- Explicit account-policy authorization remains Mark's reported OpenAI guidance;
  this proposal does not claim three isolated logins grant extra quota or guarantee
  permanent sessions. User participation is required for the new browser/device
  authorizations.
