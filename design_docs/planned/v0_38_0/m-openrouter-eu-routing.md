# M-OPENROUTER-EU-ROUTING: EU In-Region Routing for All OpenRouter Eval Endpoints

**Status**: Planned
**Target**: v0.38.0
**Priority**: P1 (Medium) — compliance/data-residency option; not eval-blocking
**Estimated**: 1–2 days
**Dependencies**: None (OpenRouter EU in-region routing is now self-serve on the Business plan)

**Author**: Mark + coordinator agent
**Created**: 2026-09-10

---

## TL;DR

OpenRouter now offers self-serve EU in-region routing: point the client at
`https://eu.openrouter.ai/api/v1` and requests are decrypted in the EU and routed
**only** to provider endpoints in the EU — prompts and completions never leave the
region, and the endpoint **fails closed** (errors instead of leaving the region) when
no in-region provider can serve a model. Same API key, same model slugs, same billing.

This doc adds an EU option to every AILANG eval surface that constructs an OpenRouter
client, so a single flag/env var (`OPENROUTER_BASE_URL` plus a first-class
`openrouter-eu:` model prefix) routes eval traffic through the EU endpoint. Default
behaviour is unchanged (global endpoint).

---

## Problem Statement

AILANG evals route a large share of model traffic through OpenRouter
(`internal/ai/openrouter/client.go`, default base URL
`https://openrouter.ai/api/v1`). Today there is **no way** to keep that traffic inside
the EU:

**Current State:**
- `defaultBaseURL = "https://openrouter.ai/api/v1"` is hard-coded; `WithBaseURL`
  exists but is only used in tests (`internal/ai/openrouter/client.go:18,85-103`).
- The eval harness constructs the client with no base-URL override:
  `provider = openrouter.NewClient(apiKey)` (`internal/eval_harness/ai_provider.go:90`).
- Other OpenRouter call sites (mission quota, design-quorum/review, observatory
  broadcast traces) likewise assume the global endpoint.
- We already operate EU-adjacent routing for other providers (Lyceum EU-hosted
  endpoint honours `LYCEUM_BASE_URL`; z.ai honours `ZAI_BASE_URL` —
  `internal/ai/config.go`), so OpenRouter is the odd one out.

**Impact:**
- Eval prompts may contain proprietary benchmark content; EU data-residency
  guarantees (fail-closed in-region routing, DPA via Business plan ToS) are a
  requirement for some collaborators and future enterprise use.
- Ad-hoc workarounds (proxying, key juggling) would violate the repo's
  "no silent fallbacks / one source of truth" principles.

---

## Goals & Non-Goals

### Goals
1. **One base-URL change, everywhere.** A single resolution point
   (`ai.OpenRouterBaseURL()`, honouring `OPENROUTER_BASE_URL`) feeds every OpenRouter
   client construction, mirroring the existing `LyceumBaseURL()` / `ZAIBaseURL()`
   pattern.
2. **First-class model selection.** An `openrouter-eu:` model prefix (parallel to the
   existing `openrouter:`, `lyceum:`, `zai:` prefixes in `GuessProvider`) lets
   `models.yml` entries and CLI `--model` flags select EU routing per-model, per-run.
3. **Observability.** EU-routed calls are identifiable in traces/dashboards (endpoint
   host recorded on spans; `error_category` distinguishes the EU fail-closed error
   from generic `api_error`).
4. **Zero behaviour change by default.** No env var, no prefix → global endpoint,
   exactly as today.

### Non-Goals
- Changing billing/plan management (the OpenRouter Business upgrade is an account
  action, not code).
- Validating which models are EU-eligible at config time — the endpoint fails closed
  and OpenRouter publishes the eligible list; we surface the error, we don't
  replicate their catalogue.
- EU routing for non-OpenRouter providers (Lyceum/z.ai already have their own
  endpoints; Anthropic/Google direct APIs are out of scope).
- Migrating historical benchmark baselines; EU vs global latency differences are a
  documented caveat for cross-region result comparison.

---

## Proposed Design

### 1. Base URL resolution (`internal/ai/config.go`)

Add alongside `LyceumBaseURL()`:

```go
// OpenRouterEUBaseURL is the EU in-region endpoint (OpenRouter Business,
// fail-closed in-region routing).
const OpenRouterEUBaseURL = "https://eu.openrouter.ai/api/v1"

// OpenRouterBaseURL returns the OpenRouter endpoint, honouring
// OPENROUTER_BASE_URL for explicit overrides. Default: global endpoint.
func OpenRouterBaseURL() string {
    if u := os.Getenv("OPENROUTER_BASE_URL"); u != "" {
        return u
    }
    return defaultOpenRouterBaseURL // move constant here from client.go
}
```

No silent fallback subtlety: an unset env var yields the documented default, and an
explicit `openrouter-eu:` prefix always wins over the env var (see §3 resolution
order).

### 2. Provider type and detection (`internal/ai/config.go`)

- Do **not** add a new `ProviderType`; EU OpenRouter is the same provider, same auth,
  same wire format. Add a routing dimension instead:
  - `GuessProvider("openrouter-eu:anthropic/claude-sonnet-5")` → `ProviderOpenRouter`,
    with the prefix stripped and an EU flag carried to client construction.
- Mechanically: extend the prefix handling that already strips `openrouter:`
  (`internal/eval_harness/ai_provider.go:88-90`) to recognise `openrouter-eu:` and set
  a boolean, then:

```go
case ai.ProviderOpenRouter:
    model = strings.TrimPrefix(model, "openrouter:")
    if euRequested {
        provider = openrouter.NewClient(apiKey,
            openrouter.WithBaseURL(ai.OpenRouterEUBaseURL))
    } else {
        provider = openrouter.NewClient(apiKey,
            openrouter.WithBaseURL(ai.OpenRouterBaseURL()))
    }
```

### 3. Resolution order (explicit beats ambient)

1. `openrouter-eu:` model prefix → EU endpoint (per-model, per-run; wins always).
2. `OPENROUTER_BASE_URL` env var → operator override (any compatible endpoint).
3. Default → `https://openrouter.ai/api/v1` (unchanged).

This matches the repo's existing env-override convention (`LYCEUM_BASE_URL`,
`ZAI_BASE_URL`) and keeps eval configs declarative.

### 4. models.yml integration

Entries can opt in per model:

```yaml
- name: claude-sonnet-5-eu
  api_name: "openrouter-eu:anthropic/claude-sonnet-5"
  provider: openrouter
  env_var: OPENROUTER_API_KEY
```

The eval-gap/benchmark tooling consumes `api_name` unchanged; only provider
construction changes.

### 5. Fail-closed error surfacing

When no EU provider can serve a model, OpenRouter returns an error instead of
routing globally. In `internal/eval_harness/error_categorizer*`, map the EU
in-region unavailability response (provider error from `eu.openrouter.ai` host) to a
distinct category (e.g. `eu_region_unavailable`) so a misconfigured `openrouter-eu:`
model is diagnosable in one look, not lumped into the `api_error` catch-all.
Guardrail compliance: **fail loudly, no retry-onto-global fallback** — silently
falling back to the global endpoint would defeat the entire data-residency purpose.

### 6. Observability

- The OpenRouter client already emits spans; record the effective endpoint host
  (`openrouter.ai` vs `eu.openrouter.ai`) as a span attribute so the observatory and
  Broadcast traces can filter EU vs global traffic.
- `ailang chains chat` / dashboard run metadata: include resolved base URL in the
  run's provider config dump.

### 7. CLI surface

- `ailang eval-*`, `ailang design-quorum`, and any other OpenRouter-consuming
  command inherit the env var automatically via §1. Document in CLI help
  (cli-doc-maintainer skill): `OPENROUTER_BASE_URL` and the `openrouter-eu:` prefix.

---

## Alternatives Considered

| Alternative | Rejected because |
|---|---|
| Env var only, no model prefix | Can't mix EU and global models in one eval matrix; benchmark suites need per-model control. |
| New `ProviderType` (`openrouter-eu`) | Duplicates provider plumbing (auth, retry, streaming) for what is one base-URL change; the prefix+flag approach reuses the existing OpenRouter client wholesale. |
| HTTP proxy / egress firewall | Out-of-band, invisible to traces, and can't guarantee fail-closed semantics at the model level. |
| Wait for per-request `provider.region` routing in the API body | OpenRouter's shipped self-serve mechanism is the EU base URL; designing for an unannounced API is speculation. |

---

## Affected Files (implementation map)

| File | Change |
|---|---|
| `internal/ai/config.go` | Add `OpenRouterEUBaseURL`, `OpenRouterBaseURL()`, prefix detection for `openrouter-eu:` |
| `internal/ai/openrouter/client.go` | Default base URL now sourced from `ai.OpenRouterBaseURL()` (or keep constant, passed in) |
| `internal/eval_harness/ai_provider.go` | Strip `openrouter-eu:`, pass `WithBaseURL` |
| `internal/eval_harness/error_categorizer*.go` | New `eu_region_unavailable` category |
| `internal/mission/openrouter_quota.go` | Route quota checks through resolved base URL |
| `models.yml` (eval configs) | Optional `-eu` model entries |
| CLI help / docs | `OPENROUTER_BASE_URL`, `openrouter-eu:` prefix |

---

## Testing Strategy

1. **Unit**: prefix detection (`openrouter-eu:`, `openrouter:`, bare) in
   `GuessProvider`; env-var resolution table tests (unset/set/prefix-wins).
2. **Client**: existing `WithBaseURL(httptest.Server)` pattern proves endpoint
   override works; add a test asserting `openrouter-eu:` constructs a client pointed
   at `eu.openrouter.ai`.
3. **Categorizer**: fixture response → `eu_region_unavailable`.
4. **Smoke (manual, requires Business plan)**: one benchmark trial with
   `openrouter-eu:anthropic/claude-sonnet-5`; verify span host attribute and that a
   knowingly non-EU-eligible model fails closed with the new category.

---

## Risks & Open Questions

- **Plan dependency**: EU routing requires the OpenRouter Business plan (8% credit
  fee vs 5.5% Standard). Code must behave sanely on Standard: expect a clear API
  error, surfaced via the new error category. *Open question*: exact error shape on
  Standard-plan EU requests — capture during smoke test.
- **Latency/parity**: EU-routed evals may show different latency and (transiently)
  different provider mixes; do not mix EU and global results in the same Elo pool
  without a paired-comparison check (`ailang eval-paired`).
- **Model catalogue drift**: EU-eligible list will grow; our docs should link
  OpenRouter's list rather than snapshot it.

---

## Related Documents

- [M-EU-COMPLIANCE-EFFECTS](../v1_1_0/m-eu-compliance-effects.md) — language-level
  EU compliance effects; this doc is the infra-side complement (where eval traffic
  physically goes).
- [M-LYCEUM-PROVIDER / M-ZAI-WINDOW-ROUTING] — precedent for env-overridable
  provider base URLs (`LYCEUM_BASE_URL`, `ZAI_BASE_URL`).
- Source email: OpenRouter, "EU in-region routing is now self-serve", 2026-09-09.
