# Codex Provider/Billing Lane Resolution

**Status**: Planned
**Target**: v0.39.0
**Type**: Provider-routing / billing ruling (semantics of the `codex:` model prefix in the AI effect)
**Priority**: P1 (High) — silent billing exposure
**Estimated**: 2 days
**Dependencies**: M-MODEL-REGISTRY-SINGLE-SOURCE (v0.35.0, implemented lane semantics)
**Source**: GitHub issue #903 (sunholo-data/ailang, triaged from public-feedback inbox; verified sharper than reported at v0.34.0, still present at v0.38.5)

## Problem Statement

Two coupled defects share one root cause — no billing-lane concept in the AI effect's provider resolution:

1. **`codex:` prefix silently routes to the metered OpenAI lane.** `GuessProvider` ([internal/ai/config.go:83](../../../internal/ai/config.go)) maps any `codex*` prefix to `ProviderOpenAI`, so an AI-effect call with a `codex:` model constructs the metered OpenAI API client (`internal/eval_harness/ai_provider.go` → `openai.NewClient(apiKey)`) — silently spending API credits.
2. **The AI effect cannot bill the ChatGPT subscription at all.** The fleet's billing guard ([tools/launchd/mission-control.sh:122-127](../../../tools/launchd/mission-control.sh)) unsets `OPENAI_API_KEY` so codex authenticates *only* via the ChatGPT-subscription OAuth in `~/.codex/auth.json` (auth_mode `chatgpt`) — subscription-or-nothing by construction. The codex EXECUTOR lane honors this (auth materialized via `EnsureAPIKeyAuth` in `auth_bootstrap.go`, commit 7b444096, with the explicit rule that construction must NOT materialize credentials). The AI-effect lane has no equivalent: given a `codex:` model it wants an API key that the fleet deliberately strips, and given no prefix knowledge it has no way to reach the subscription at all.

Net effect: on the fleet, an AI-effect `codex:` call either fails on missing `OPENAI_API_KEY` or — on any machine where a metered key happens to be in the environment — silently bills API credits in violation of the driver's billing rule. Both outcomes are wrong, and neither is loud.

**Impact:** billing integrity (CLAUDE.md Critical Principle 2 — no silent fallbacks on pricing paths); users of `.ail` programs using `codex:` models.

## Ruling

**Fail loud. A `codex:` prefix in the AI-effect lane resolves to NO provider and returns a typed error.**

Reasons, in order of weight:

1. **The subscription lane is not reachable from the AI effect, and should not be made reachable by prefix-mapping.** The ChatGPT subscription is an *OAuth session of a subprocess CLI* (`codex login --device-auth`, `~/.codex/auth.json`). It is not an HTTP API an `ai.Provider` client can call with the subscription credential: the subscription auth exists precisely to serve codex CLI sessions, and the executor lane already spent its probe-verified knowledge (2026-07-30, codex-cli 0.145.0) establishing that env-key auth does not override `auth_mode: chatgpt`. An `ai.Provider` implementation that shells out to `codex exec` would not be a provider at all — it would be a second, worse executor inside the effect handler, duplicating internal/executor/codex (NDJSON parsing, token-split semantics, timeout/idle clocks) at a layer whose contract is a `Provider` interface.
2. **Registry single-source already forbids prefix-based provider claims for this case.** M-MODEL-REGISTRY-SINGLE-SOURCE made models.yml the source of truth and removed defaults; `GuessProvider` is explicitly the fallback "when models.yml is not available". Under that doctrine, `codex:` should never be *guessable* — it's not an OpenAI API model name, it's an executor-lane pin (mission-control's cross-provider spawn recipe uses `provider:model` syntax where `codex` means "route via provider_executor", never the Agent tool). Mapping it to `ProviderOpenAI` is a stale guess that re-introduces exactly the silent-default failure the registry mission closed.
3. **Fail-loud beats a new lane here (CLAUDE.md §2).** A fallback that picks the metered lane affects user money; the correct zero-value is an error naming the two real options.

**Consequence of the ruling:** a `codex:` model is valid ONLY in the EXECUTOR lane (coordinator / provider_executor), where the subscription-or-API-key choice is made deliberately at the job entry point (`EnsureAPIKeyAuth` writes an api-key auth.json *only* when no subscription auth.json exists and a key is present — the inverse of the driver guard, which strips the key so only subscription remains). The AI effect must reject it loudly.

## Design

### D1. `GuessProvider` stops claiming `codex:` → `ProviderOpenAI`

Remove the `strings.HasPrefix(lower, "codex")` arm from the prefix switch in [internal/ai/config.go](../../../internal/ai/config.go). Result: `GuessProvider("codex:gpt-5-codex")` and `GuessProvider("gpt-5-codex")`-style *executor* model names no longer resolve to a metered API client by guessing.

Two sub-cases:

- **`codex:` prefixed** (explicit executor-lane pin used in an AI-effect context): return `""` (cannot determine provider). This is already the fail-loud path — the empty provider type propagates as "cannot determine provider for model" at handler construction. Verify that propagation is a typed, actionable error (see D2), not a panic or a silent nil.
- **Bare `gpt-5-codex` / future codex-named API models**: these ARE real OpenAI Responses-API model names when called with a metered key deliberately. If such a model appears in models.yml with `provider: openai`, the registry row governs and guessing is bypassed entirely. Bare-prefix guessing for genuinely metered OpenAI models (the `gpt` arm) is unchanged. `codex` as a bare prefix is removed because the name is executor-lane vocabulary, and any legitimate API exposure of such a model arrives via the registry, not via a guess.

### D2. Typed error, not empty-string silence

Wherever the empty `ProviderType` from D1 is consumed (AI-effect handler construction in `internal/effects/ai.go` path, `internal/eval_harness/ai_provider.go`), the error must name the ruling:

```
model "codex:gpt-5-codex": the codex: prefix is an executor-lane pin, not an
AI-effect model. The AI effect cannot bill the ChatGPT subscription (it is
OAuth of the codex CLI, not an API credential) and will not silently bill
the metered OpenAI lane. Use the coordinator/provider_executor lane for
codex, or register an explicit provider: row in models.yml.
```

Concretely: a `ProviderError`/typed diagnostic at handler construction, effect-visible as a loud eval error — never a zero/empty-completion fallback (which would be a silent fallback on a billing path).

### D3. Registry row takes precedence — unchanged, but restate it

models.yml remains the only way a codex-family model can reach the AI effect at all: if and when OpenAI exposes a codex-family model on a metered API we intend to use, the row is written there with `provider: openai` and an explicit api_name, and the AI effect follows the registry. No change to registry code; this doc just records that D1 must not break that path (explicit provider bypasses `GuessProvider` entirely, per `newProviderAdapter`).

### D4. The executor lane is the pattern, not the fix site

The billing split is *intentional and correct* in the executor lane and needs no change:

| Lane | Credential choice | Enforced by |
|---|---|---|
| Driver (mission-control.sh) | subscription-or-nothing | `unset OPENAI_API_KEY` (billing guard) |
| Codex executor (job entry) | subscription if present, else explicit api-key auth.json | `EnsureAPIKeyAuth` (only writes when absent + key set) |
| AI effect (this doc) | **no codex lane — reject** | D1 + D2 |

Mirroring the driver-side guard means: the AI effect is what the guard *protects*, so its only compliant codex behavior is to refuse. Do not "fix" the AI effect by reading `~/.codex/auth.json` and shelling out to codex CLI inside an `ai.Provider` — that duplicates the executor (D1 reason 1) and would create a second code path around the subscription that the driver guard cannot audit.

### D5. Migration for existing configs

Search the repo, eval harness configs, and mission-control role env values for AI-effect (non-executor) usages of `codex:` models:

1. **Mission-control role env values** (`MISSION_*_MODEL`): values matching `^codex:(.+)$` are *executor* pins by the M1b spawn recipe — these are correct and unaffected (they never reach GuessProvider). No migration.
2. **models.yml rows or `.ail` programs / std-ai calls** naming a `codex:` or bare codex model as an AI-effect model: these were silently metering. They break loudly after D1/D2 — by design. Each either (a) moves to an executor task, or (b) is re-pointed at the intended AI model with an explicit registry row.
3. **Changelog note** at release: "AI-effect `codex:` models now fail loud instead of silently billing the metered OpenAI lane; use the executor lane or an explicit models.yml row."
4. Because this is a behavior change on a billing path, ship it in a minor release with the changelog entry as the migration document; no compat shim (a shim that maps codex→metered OpenAI is the defect).

## Goals

**Primary Goal:** No AI-effect call can silently bill the metered OpenAI lane via a `codex:` prefix; the prefix fails loud with an actionable error, and the executor lane remains the only codex billing path.

**Success Metrics:**
- `GuessProvider("codex:...")` returns empty / typed "cannot determine" in unit test.
- AI-effect construction with a `codex:` model returns the D2 error string (unit test on the effect handler path).
- Registry-row path (explicit `provider: openai`) still works unchanged (regression test exists in `config_test.go` style).
- No changes required in internal/executor/codex or mission-control.sh.

## Non-Goals

- Building an AI-effect provider that shells out to `codex exec` (rejected, D4).
- Reading or refreshing `~/.codex/auth.json` from the AI effect (rejected — deployment action, mirrors the executor's own construction-time rule).
- Changes to the executor lane's auth bootstrap or the driver billing guard.
- Subscription billing for arbitrary models: out of scope; the subscription is codex CLI's, not ours to broker.

## Related Documents

- [m-model-registry-single-source.md](v0_35_0/m-model-registry-single-source.md) — models come from the registry, defaults removed; this ruling removes the last executor-vocabulary prefix from the guesser.
- [m-exec-expand-codex-opencode.md](../implemented/v0_15_0/m-exec-expand-codex-opencode.md) — codex executor lane; `auth_bootstrap.go` / commit 7b444096 is the auth-materialization pattern.
- CLAUDE.md §2 (no silent fallbacks) — the principle this ruling applies.
- Issue #903 — source report.

## Open Questions

1. Should `GuessProvider` return `""` or a new sentinel `ProviderCodex` that consumers must explicitly reject? Ruling leans `""` (no new provider type implies no new lane exists to reach); revisit only if the D2 error path needs a distinguishable code for telemetry.
2. Are there external user configs (public-feedback reports beyond #903) relying on AI-effect `codex:` for metered use deliberately? If such a use exists, it migrates via D5(2b) — an explicit registry row — not via restoring the guess.