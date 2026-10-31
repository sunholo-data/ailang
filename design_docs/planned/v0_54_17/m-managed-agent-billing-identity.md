# M-MANAGED-AGENT-BILLING-IDENTITY — Effective billing identity and pricing gates for pinned managed agents

**Status**: Planned
**Target**: v0.54.17
**Priority**: P0 (High) — `origin/dev` HEAD d25d9b8d is red on `TestModels_PricingIsSlugConsistent` in three CI jobs (test, Windows, macOS-release)
**Estimated**: ~2 days (4h modelreg helper + 4h gates + 3h models.yml evidence markers + 3h tests + 2h docs)
**Dependencies**: none new; builds on M-EVAL-PRICING-DRIFT (offline pricing gates, 2026-08-13), M-MODEL-REGISTRY-SINGLE-SOURCE (registry owns name grammar), and the managed-agent pin support shipped in d25d9b8d

## Problem Statement

`TestModels_PricingIsSlugConsistent` (internal/eval_harness/models_pricing_consistency_test.go) keys billing identity by `provider + "|" + api_name`. Commit d25d9b8d added `gemini-agents-argon`, a second `google` row with `api_name: "antigravity-preview-09-2026"`, and the gate went red deterministically:

- `gemini-agents-antigravity` — `.00075/.00375/.000075` per-1K in/out/cache. **Measured**: the GCP billing export (2026-10-14) shows the unpinned agent bills as SKU "gemini 3.8 flash text".
- `gemini-agents-argon` — `.002/.01/.0001`. **Listed**: Gemini 4 Argon list rates; the row's own notes say "Argon billing SKU verification is pending".

Failing jobs: CI 39606168604 test job 119242793153 (step 13) and Windows 119242954630 (step 11); Build/Release 39606168603 macOS 119242793748 (step 7). Parent 267cf2fd (CI 39589515608) was green — the only commit between the two states is d25d9b8d. Reproduced on current dev (see Verification Log V1): the `google|antigravity-preview-09-2026` pair is the *only* collision in the registry.

**The gate is right that something is wrong — it names the wrong thing.** The two rows do not claim to bill the same model: one is a version-free agent lane (Google runs whatever it runs behind `antigravity-preview-09-2026`), the other explicitly *requests* `gemini-4-argon` via `agent_config` (the `agent/model` pin form in `agent_model_name`). Their billing identities differ; their *wire* identity (provider + api_name) does not. The test conflates the two.

**The two easy "fixes" are both wrong, and both are explicitly rejected:**

1. *Equalize the rates* — set both rows to one number. Wrong because the two rows request different SKUs: the unpinned row measurably bills 3.8-Flash rates, the pinned row records Argon list rates. Whichever number "wins", the other row's cost accounting becomes fiction. Worse, the pinned row's *true* billing SKU is **UNMEASURED** — the pin is accepted and echoed (and unknown models now 404 since 2026-10-21, TestLiveArgonPin), but "only the billing SKU proves it served the run" (devapi.go `splitAgentModel` comment). No number we could write today is evidence-based for that row.
2. *Silence/exempt the collision* — add an exemption list entry. Wrong because it would mute the real defect class the gate exists for: rows that truly bill the same model must agree, or the same tokens cost different amounts depending on which row invoked them (the 2026-08-13 drift audit found exactly this four times over, worst case three different prices for one slug).

**Impact:** dev is red for every PR until resolved; the release macOS leg is red (the release has three cancelled matrix legs and is otherwise unmeasured; CI currently masks 58 downstream test steps, 3 Windows steps, 1 govuln step — the masking means this gate is one of the few honest signals left). Any agent-mode cost comparison involving the antigravity lanes is blocked or untrustworthy meanwhile.

## Goals

**Primary Goal:** Make the offline pricing gate key on **effective billing identity** (what a row actually asks to be billed as) instead of wire identity, so the CI gate goes green *because the registry is now correct*, not because the gate was muted — while retaining full drift detection for rows that genuinely bill the same model.

**Success Metrics:**
- `TestModels_PricingIsSlugConsistent` passes on dev HEAD with zero exemption-list entries for `antigravity-preview-09-2026` (no test-text `zeroIsCorrect`-style carve-outs grow).
- Existing drift detection retained: a synthetic registry fixture with two rows sharing provider+api_name and *no* pin still fails the gate (regression test).
- New directed gate: a pinned managed-agent row whose rates or schedule drift from the pinned model's canonical row fails CI (regression tests for rate drift and schedule drift).
- Every managed-agent row's epistemic pricing status is machine-readable (`pricing.evidence`), and `ailang eval-models` surfaces it — an UNMEASURED rate can no longer sit in a cost table indistinguishable from a measured one.
- No banked result is recomputed (M-EVAL-PRICING-DRIFT measurement contract: banked costs stay as-is, only annotated).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Billing identity includes the pin **only for managed-agent CLIs** (`agent_cli: managed_agents`/`gemini_agents`); never parsed for other rows | `agent_model_name` uses `provider/model` form on ~90 non-managed rows (`openai/gpt-5.5`, `anthropic/claude-haiku-4-5`, …) — an unscoped `<agent>/<model>` parse would mis-split every one of them | human | design | high |
| Pinned rows keep **explicit copied rates + a parity gate** against the referenced model's row (Option A), not rate-by-reference inheritance (Option B) | Option B (a `pricing.ref:` field dereferenced at load) removes transcription entirely but touches LoadModelsConfig, provenance/publishing validation, and every cost helper; Option A gets the same detection with a test. B is future work, not now | human | design | med |
| `pricing.evidence: measured \| listed \| proxy` required on managed-agent rows, optional elsewhere | Creates a new epistemic column in the registry ( precedent: `default_thinking`'s explicit "unknown"); scoping keeps the YAML edit to 4 rows now, with a documented path to widen | human | design | med |
| The pinned row's rates are **NOT equalized** with the unpinned row's, and which SKU actually bills stays recorded as UNMEASURED until independent billing evidence exists | This is the directive constraint: guessing a number would corrupt cost accounting silently; the honest state is "listed, pending verification", made visible | human | design | high |
| The `agent/model` split grammar moves to `modelreg` as the single owner; the executor's `splitAgentModel` delegates to it | Two copies of a name grammar is exactly the split that M-MODEL-REGISTRY-SINGLE-SOURCE M1 removed for `IsOllamaCloudRoute`; the registry owns name grammar | agent | compile | low |
| When billing evidence later shows the pin does NOT route billing to Argon, the fix is to correct the row's rates *and* its evidence in the same change, with the affected baselines annotated (not recomputed) | Decides the promotion path now, so the eventual measurement has a procedure instead of an argument | human | design | low |

### Design Freeze

- [x] Pin parsing is scoped by `agent_cli` (managed-agent CLIs only) — see Verification Log V5 for why unscoped parsing is unsafe.
- [x] Option A (explicit rates + parity gate) over Option B (rate-by-reference) for this milestone.
- [x] No rate equalization, no exemption-list entries; the `antigravity-preview-09-2026` collision is resolved by identity, not by numbers.
- [x] `pricing.evidence` is required on managed-agent rows only in this milestone.
- [ ] Exact enum values and YAML key name (`pricing.evidence`) — confirm at design review.

## Solution Design

### Overview

Introduce **effective billing identity (EBI)** in `modelreg` — one helper that computes, for any row, the identity its rates must be consistent within: `provider | api_name` for ordinary rows (unchanged), and `provider | agent | pinned-model` for managed-agent rows, where the agent and optional pin come from the `agent_model_name` grammar — parsed **only** when the row's `agent_cli` is a managed-agent CLI. Rekey the existing slug-consistency gate on EBI (same-EBI rows must agree on rates — drift detection fully retained), and add a directed **pin-parity gate**: a pinned row must byte-match the registry row whose `api_name` equals the pin, including the scheduled-price fields (`expires`/`next`). Add a machine-readable `pricing.evidence` marker for managed-agent rows so the UNMEASURED status of the pinned row's Argon rates is visible in every listing instead of living in prose.

### Architecture

**Components:**

1. **`modelreg.SplitAgentModelName(s string) (agent, pin string)`** (new, in `identity.go` next to `OriginVendor`): the canonical `<agent>/<model>` split, moved from `internal/executor/managed_agents/devapi.go`. The executor's `splitAgentModel` becomes a one-line delegate. The registry owns name grammar (M-MODEL-REGISTRY-SINGLE-SOURCE M1 precedent: `IsOllamaCloudRoute`).
2. **`(*ModelConfig).EffectiveBillingKey() string`**: returns `provider|api_name` normally; for rows with `agent_cli` in {`managed_agents`, `gemini_agents`}, returns `provider|agent|pin` (pin empty when unpinned — the version-free lane). This is the one function every pricing gate uses; the eval_harness test stops hand-rolling the key.
3. **Rekeyed gate** — `TestModels_PricingIsSlugConsistent` builds `bySlug` on `EffectiveBillingKey()`. Everything else in the test is unchanged: billed providers, exact float equality, the failure message (fix text updated to point at EBI, not `make verify-model-pricing`, when the collision crosses a pin).
4. **New gate** — `TestModels_PinnedAgentRatesMatchReferencedModel` (same file, same offline/no-network stance): for every managed-agent row with a non-empty pin:
   - the pin must equal the `api_name` of an existing registry row **of the same provider** (a pin naming an unknown model is a typo with a pricing budget attached);
   - the pinned row's `input/output/cache_read` rates must exactly equal that row's (same exact-float rule as the existing gate);
   - the pinned row's `expires`/`next` schedule must exactly equal that row's — a repricing updated on the canonical row but not the pinned twin is precisely the half-updated-schedule defect this catches.
5. **`pricing.evidence`** (`Pricing.Evidence string`, yaml `evidence`): `measured` (rate observed from a billing export or live vendor API for this row's billed identity), `listed` (vendor list rate for the requested model; run→SKU mapping not yet confirmed by billing evidence), `proxy` (best-available rate for an identity whose true billed model is unknowable). Required on rows whose `agent_cli` is a managed-agent CLI; enforced by a new `TestModels_ManagedAgentPricingEvidenceIsExplicit`. The citation for a `measured`/`listed` value stays in `notes:` per the existing PRICING header convention.
6. **Surfacing** — `ailang eval-models` prints the evidence marker in its model table, so "listed (pending SKU verification)" is visible next to any cost figure. No cost math changes anywhere: `Task.Pricing` already comes from the row (agent_runner_multi.go:349), so banked costs follow the row's rates and evidence automatically.

### Implementation Plan

**Phase 1: modelreg grammar + EBI** (~4 hours)
- [ ] Add `SplitAgentModelName` to `internal/modelreg/identity.go` (moved verbatim from devapi.go, including the "request, not verified fact" comment); executor `splitAgentModel` delegates; `devapi_test.go` pin-split cases move to `identity_test.go` alongside new ones.
- [ ] Add `EffectiveBillingKey()` on `ModelConfig` in `identity.go`; unit tests covering: ordinary row, unpinned managed-agent row, pinned managed-agent row, and the guard that a `provider/model`-style `agent_model_name` on a NON-managed row never splits.

**Phase 2: gates** (~7 hours)
- [ ] Rekey `TestModels_PricingIsSlugConsistent` on `EffectiveBillingKey()`; update its doc comment and failure message; add a fixture-based regression test (in-memory `ModelsConfig`) proving two same-provider+api_name unpinned rows still fail.
- [ ] Add `TestModels_PinnedAgentRatesMatchReferencedModel` with fixture regressions: unknown pin, rate drift, schedule drift.
- [ ] Add `Pricing.Evidence` field; add `TestModels_ManagedAgentPricingEvidenceIsExplicit` (fails on missing/unknown enum values for managed-agent rows); wire evidence into `ailang eval-models` output.

**Phase 3: registry truth + docs** (~5 hours)
- [ ] models.yml: set `evidence` on the four managed-agent rows (`managed-agents-antigravity: proxy`; `gemini-agents-antigravity: measured`; `gemini-agents-argon: listed`; `gemini-3-5-flash: listed`) with a one-line comment each.
- [ ] Update the models.yml PRICING header block: billing identity is EBI, pins are requests, evidence is required on managed-agent rows; document the promotion procedure (billing-export evidence → correct rates + flip `evidence` → annotate affected baselines, never recompute).
- [ ] Run the full modelreg + eval_harness gate suite; confirm the three CI jobs' failing test goes green for the right reason.

### Files to Modify/Create

**New files:** none (all changes land in existing files — bounded by design).

**Modified files:**
- `internal/modelreg/identity.go` (~+45 LOC) — `SplitAgentModelName`, `EffectiveBillingKey`
- `internal/modelreg/identity_test.go` (~+90 LOC) — grammar + EBI tests, moved pin-split cases
- `internal/modelreg/models.go` (~+8 LOC) — `Pricing.Evidence` field with doc comment
- `internal/executor/managed_agents/devapi.go` (~−15/+5 LOC) — `splitAgentModel` delegates to modelreg
- `internal/eval_harness/models_pricing_consistency_test.go` (~+120 LOC) — EBI rekey, pin-parity gate, evidence gate, fixture regressions
- `internal/modelreg/models.yml` (~+12 LOC + header block edit) — evidence markers, PRICING header update
- `ailang eval-models` surface (internal/eval_harness listing path; ~+10 LOC) — evidence column
- This doc: move to implemented/ on completion

## Examples

### Example 1: the current failure, resolved by identity

**Before (red, on dev HEAD d25d9b8d):**
```
"google|antigravity-preview-09-2026" has 2 different recorded prices across 2 entries
    in=0.00075 out=0.00375 cache=7.5e-05  <- [gemini-agents-antigravity]
    in=0.002 out=0.01 cache=0.0001         <- [gemini-agents-argon]
```

**After (green, and for the right reason):**
```
EffectiveBillingKey(gemini-agents-antigravity) = google|antigravity-preview-09-2026|      (unpinned lane)
EffectiveBillingKey(gemini-agents-argon)      = google|antigravity-preview-09-2026|gemini-4-argon  (pinned)
→ different keys → no slug-collision check between them.
Instead: pin-parity gate checks gemini-agents-argon against the gemini-4-argon row
(in/out/cache = .002/.01/.0001, expires 2026-11-30, next .004/.02/.0002 — byte-identical → green),
and evidence gate requires both antigravity rows to state measured/listed/proxy.
```

### Example 2: drift detection retained (the gate must still bite)

If someone later adds `motoko-agents-argon` — same executor, same agent, same pin — with a mistyped rate:
```yaml
motoko-agents-argon:
  api_name: "antigravity-preview-09-2026"
  agent_cli: "gemini_agents"
  agent_model_name: "antigravity-preview-09-2026/gemini-4-argon"
  pricing: { input_per_1k: 0.002, output_per_1k: 0.05, ... }   # output mistyped
```
EBI is identical to `gemini-agents-argon` (`google|antigravity-preview-09-2026|gemini-4-argon`) with different rates → the rekeyed slug gate fails, exactly as the 2026-08-13 drift class demands. The pin does not mute same-model drift; it only stops the gate from asserting that an unpinned lane and a pinned lane bill the same thing.

### Example 3: the UNMEASURED status stays honest and visible

`ailang eval-models` today prints rates with no epistemic status; after this change the managed-agent rows print `measured`, `proxy`, or `listed (SKU verification pending)`. When the first billing export for a pinned run lands, the procedure is mechanical: if the export shows Argon SKUs → flip `gemini-agents-argon` to `evidence: measured` (rates already correct); if it shows 3.8-Flash SKUs → the pin does not route billing, correct the rates to the measured ones, flip evidence, and annotate the affected baselines (never recompute — M-EVAL-PRICING-DRIFT contract).

## Success Criteria

- [ ] `TestModels_PricingIsSlugConsistent` green on dev; zero new exemption-list entries anywhere in the gate file
- [ ] Regression fixtures prove: unpinned same-slug drift still fails; pinned-vs-canonical rate drift fails; pinned-vs-canonical schedule drift fails; unknown pin fails; `provider/model` agent_model_name on non-managed rows is never parsed as a pin
- [ ] All four managed-agent rows declare `pricing.evidence`; `ailang eval-models` shows it
- [ ] `make test` (modelreg + eval_harness) green; executor managed_agents tests green after the delegate change
- [ ] models.yml PRICING header documents EBI, pins-as-requests, evidence, and the promotion procedure
- [ ] No banked result recomputed; no cost-formula change (`Pricing.Cost` untouched)
- [ ] Documentation updated (this doc → implemented/ at completion)

## Testing Strategy

**Unit tests:**
- modelreg: split grammar (moved cases + provider-slug negative cases), EBI key per row class, evidence field parsing
- eval_harness gates: fixture registries (in-memory `ModelsConfig`, no file I/O) for every failure mode listed above, plus the existing prod-registry run

**Integration tests:**
- Full gate suite against the real `models.yml` (embedded + disk load, as the existing tests do)
- Executor pin behavior unchanged: Vertex-refuses-pin and Developer-API pin paths (existing devapi tests, now exercising the delegated split)

**Manual testing:**
- `ailang eval-models` output shows the evidence column
- Re-run the reproduction from Verification Log V1 — expected: no collisions

## Deferred Decisions

- Widening `pricing.evidence` to every billed-provider row (a ~200-row mechanical YAML edit with a script + review) — human at review; this milestone requires it only where the billed model is not directly observable (managed-agent rows)
- Rate-by-reference (`pricing.ref:` inheritance) replacing copied rates on pinned rows — agent may propose in a follow-up doc once Option A has run a repricing cycle (the 2026-11-30 Argon expiry will be the first live test)
- The `managed_agents` executor's hardcoded `CostModel()` fallback (managed_agents.go:116 — gemini-3.5-flash rates labelled "as of 2026-05-20", now known stale relative to the measured 09-2026 lane) — it is unreachable from the eval harness (agent_runner_multi.go:349 always sets `Task.Pricing`, which wins in `ResolveCostModel`), so it only affects non-registry callers; decide whether to update it or mark it `Unpriced` in a separate change
- Whether `proxy` rows should be allowed to carry `expires`/`next` schedules at all (a proxy rate with a schedule is a guess with a calendar) — human at review

## Non-Goals

- **Equalizing the two antigravity rows' rates** — rejected outright: the rows bill (or request) different SKUs, and the pinned row's true SKU is UNMEASURED
- **Silencing or exempting the collision** — rejected outright: no `zeroIsCorrect`-style carve-out is added; the gate keeps its full drift surface
- **Guessing the pinned row's billing** — the design records "listed, pending verification" and makes it visible; it does not resolve the measurement
- Networked vendor drift checks (`make verify-model-pricing` semantics) — unchanged, still out of CI
- Recomputing any banked cost — forbidden by the M-EVAL-PRICING-DRIFT measurement contract
- Changes to `Pricing.Cost`, budgets, or any cost arithmetic — the row rates are already the single source for enforcement and banking; only the gate keying and visibility change

## Timeline

**Week 1** (2 days):
- Phase 1: grammar + EBI helper with tests (4h)
- Phase 2: three gates + fixtures (7h)
- Phase 3: models.yml markers, header docs, eval-models column, full gate suite green (5h)

**Total: ~16 hours across 2 days** (estimates already 2x'd)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Unscoped pin parsing mis-splits `provider/model` agent_model_name on ~90 rows | High | Parse only when `agent_cli` is a managed-agent CLI; negative fixture tests pin this (V5) |
| EBI rekey weakens drift detection somewhere unseen | Medium | The rekey is mechanical (same key shape, one extra component for managed rows); fixture regressions prove unpinned collisions still fail; the 2026-08-13 audit cases (all openrouter/anthropic, no pins) keep identical keys |
| Pin-parity gate false-positives if a pinned model's canonical row legitimately reprices on a different schedule than the pin lane | Low | The gate demands schedule equality — if the lanes genuinely diverge, that fact must be recorded in the registry (different expires), which is a visible edit, not a silent drift |
| `evidence` values rot (a `measured` marker outliving its measurement) | Medium | Same decay class as any verified-on date; the marker states epistemic status, not freshness — the models.yml header already forbids reading notes as current rates, and the promotion procedure covers updates |
| Reviewer rejects "another epistemic column" as scope creep | Low | Scoped to 4 rows; precedent is `default_thinking`, whose "unknown" value is the exact pattern being reused |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The gate becomes deterministic in the strong sense: the same registry state implies the same verdict, now under a key that matches what is actually billed |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | +1 | A pin is an explicit request for a billed identity; the design refuses to treat an accepted-but-unverified pin as granted authority, and records the difference |
| A5: Bounded Verification | +1 | All new checks are decidable from the file offline, same as the existing gates |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | `pricing.evidence` is a machine-readable epistemic column replacing prose-only status; EBI is a function, not a convention |
| A8: Minimal Syntax | +1 | No language syntax; one optional YAML field and one helper, both removing a hand-rolled key in a test |
| A9: Cost Visibility | +1 | UNMEASURED cost figures become labelled as such in every listing — the direct purpose of the design |
| A10: Composability | +1 | EBI composes with the existing gates, Resolve, and CostForName without changing any of them |
| A11: Structured Failure | +1 | Gate failure messages name the EBI key and the exact rows/rates; the promotion procedure is a documented path, not an argument |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted (the pin's authority is explicitly *not* assumed)
- [x] A7 (Machines First): evidence and EBI are machine-first, not human conveniences

## Conflict Surface

This change touches no parser/lexer/type/codegen surface. The analogous surface here is **name-grammar positions**, enumerated because the same failure mode (an unexamined position already occupied) applies:

### Positions touched

1. `agent_model_name` strings containing `/` — the pin grammar claims this position **only** for `agent_cli: managed_agents|gemini_agents` rows.

### What else lives there

| Row class | `agent_model_name` form | Example (models.yml) |
|-----------|------------------------|----------------------|
| opencode rows | `provider/model` | `openai/gpt-5.5` (line 239), `anthropic/claude-haiku-4-5` (line 2254) |
| pi rows | `provider/model` | `anthropic/claude-sonnet-4-6` (line 2375) |
| Vertex managed-agent rows | bare agent | `antigravity-preview-05-2026` |
| Developer-API managed-agent rows | bare agent, or `agent/model` pin | `antigravity-preview-09-2026`, `antigravity-preview-09-2026/gemini-4-argon` |

~90 of 96 `agent_model_name` values contain `/` as a *provider* separator; only managed-agent rows use it as an *agent/pin* separator.

### Disambiguation strategy

`agent_cli` is the discriminator, and it is sound: the pin form is defined by the managed-agents wire protocol (`agent_config: {type, model}`), which only those two executors speak; no opencode/pi row can carry a managed-agent pin because its harness would not send one. The executor's own `splitAgentModel` is only ever called on strings routed to managed-agent executors (managed_agents.go:193), so moving the grammar to modelreg must carry the same scoping: `EffectiveBillingKey` splits only when `agent_cli` is in the managed set — a condition the executor path guarantees structurally and the gate path enforces explicitly. Negative fixtures lock this.

### Existing behavior that MUST still work

- `Resolve()`'s `agent_model_name` tier (resolve.go) — exact-string match, untouched: the pinned wire name resolves to exactly one row; the unpinned name still resolves to the unpinned row; the `sameRates` ambiguity guard is unchanged
- `HarnessCostModel("gemini_agents", "antigravity-preview-09-2026/gemini-4-argon")` — full-string `Resolve` hits the pinned row first (cost.go:262)
- Vertex pin refusal and Developer-API pin echo in the executor — behavior unchanged, only the split function moves
- All 2026-08-13 drift-audit guarantees: same-slug same-price across harnesses for every non-managed row (keys unchanged)

### What deliberately changes

- The slug-consistency gate stops asserting that a pinned managed-agent row and an unpinned row sharing `api_name` bill the same model. That assertion was false; the pin-parity gate replaces it with a directed, stronger assertion (parity with the referenced model's row, including schedule).

## Verification Log

Every load-bearing claim in this doc, checked against the code at dev HEAD d25d9b8d:

| # | Claim | How verified |
|---|-------|--------------|
| V1 | The gate keys by `provider + "\|" + api_name` and the two 09-2026 rows are the **only** collision in the registry | Read models_pricing_consistency_test.go (key line in `TestModels_PricingIsSlugConsistent`); reproduced the test's exact logic (billed providers, exact float keys) over internal/modelreg/models.yml with an offline script — output: one FAIL, `google|antigravity-preview-09-2026`, rates `.00075/.00375/.000075` (gemini-agents-antigravity) vs `.002/.01/.0001` (gemini-agents-argon). Go toolchain absent in the authoring sandbox; the test is pure YAML-parse + compare, so the simulation is faithful. CI evidence: runs 39606168604 (test step 13, Windows step 11), 39606168603 (macOS step 7) red; parent 267cf2fd / run 39589515608 green |
| V2 | The unpinned row's rates are billing-export-measured (3.8-Flash SKU); the pinned row's rates are Argon list rates with SKU verification pending | models.yml: billing-export comment block in `gemini-agents-antigravity` (2026-10-14, SKU "gemini 3.8 flash text"); `gemini-agents-argon` pricing comment "Argon list rates … To confirm: billing export SKU" |
| V3 | The pin travels as `agent_config {type, model}`, is a REQUEST not a verified fact, and unknown models 404 on the Developer API since 2026-10-21 while Vertex ignores pins (executor refuses) | devapi.go:70-83 (`splitAgentModel` doc), managed_agents.go:193-200, 236-237, 403-405; d25d9b8d commit message (TestLiveArgonPin) |
| V4 | modelreg has **no** existing helper that splits `agent/model` or computes a billing identity — the grammar exists only in the executor | `grep -rn "splitAgentModel" internal/` → only internal/executor/managed_agents/{devapi.go,devapi_test.go,managed_agents.go}; identity.go holds only OriginVendor/DispatchOriginVendor; resolve.go's tiers treat agent_model_name as an opaque exact string |
| V5 | Unscoped `/`-splitting is unsafe: 23 of 96 agent_model_name values contain `/` as a provider separator on non-managed rows | models.yml lines 239, 281, 688, 2254, 2294, 2315, 2343, 2375 … (`openai/gpt-5.5`, `anthropic/claude-haiku-4-5`, `google-vertex/gemini-3-flash-preview`, …) |
| V6 | Exactly four rows use managed-agent CLIs; only one carries a pin | models.yml: `gemini-3-5-flash`, `managed-agents-antigravity` (both `managed_agents`), `gemini-agents-antigravity`, `gemini-agents-argon` (both `gemini_agents`); only `gemini-agents-argon` has `antigravity-preview-09-2026/gemini-4-argon` |
| V7 | The pinned row's rates and schedule currently byte-match the `gemini-4-argon` canonical row, so the pin-parity gate is green today | models.yml gemini-4-argon (in .002/out .01/cache .0001; expires 2026-11-30; next .004/.02/.0002) vs gemini-agents-argon (identical values) |
| V8 | Eval-harness runs price at the registry row's rates (no executor fallback on that path), so evidence + row rates flow to budget enforcement and banked cost with no cost-code change | agent_runner_multi.go:349 (`task.Pricing = executor.CostModelFromPricing(...)` unconditional), cost.go:227-233 (`ResolveCostModel`: Task.Pricing wins), managed_agents.go:339,351 |
| V9 | Scheduled-price semantics are per-row `expires`/`next`, enforced offline by `TestModels_PricingScheduleIsHonoured`; the pin row carries a schedule mirroring the canonical row | models.go Pricing.Expires/Next docs; models_pricing_consistency_test.go third test; models.yml gemini-agents-argon pricing block |
| V10 | Resolve's ambiguity guard for shared wire names compares rates only (`sameRates`), and neither antigravity wire name is ambiguous (exact-string tier, one row each) | resolve.go `Resolve`/`matchAgentModelName`/`sameRates`; agent_model_name census in V5/V6 |
| V11 | No existing design doc covers this topic (duplicate gate) | `ailang docs search` (SimHash + neural) on three phrasings: no match ≥ 0.45 in implemented/ or planned/; create-script search returned "(none found)" in both directories. Nearest relatives cited in Related Documents |
| V12 | Banked results are never recomputed on price correction — annotation only | models.yml PRICING header block ("Banked results are NEVER recomputed…", citing m-eval-measurement-contract); M-EVAL-PRICING-DRIFT framework |

Negative-existence claims: V4 and V11 each carry their grep/search evidence above, per the skill's rule.

## Related Documents

**Implemented (inform this design):**
- [m-eval-measurement-contract.md](../implemented/v0_31_0/m-eval-measurement-contract.md) — banked data stays as-is, only annotated; the promotion procedure inherits this
- [M-V1-SIMPLIFY-S3 / M-MODEL-REGISTRY-SINGLE-SOURCE](../implemented/) — registry owns name grammar; `IsOllamaCloudRoute` move-down precedent for relocating `splitAgentModel`
- M-EVAL-PRICING-DRIFT framework (models_pricing_consistency_test.go header + models.yml PRICING header, 2026-08-13) — the gate being repaired; no standalone doc exists for it

**Planned (checked for overlap — distinct):**
- [m-managed-agents-model-eval.md](../planned/v0_31_0/m-managed-agents-model-eval.md) — running managed agents on a *chosen* model via the Developer API; that doc established the agent_config mechanism this design's gates now price correctly. No overlap: it does not touch billing identity or pricing gates
- [m-mission-runtime-contract.md](../planned/m-mission-runtime-contract.md) — references the slug-consistency test; no conflict, this design preserves its guarantees

## References

- CI evidence: https://github.com/sunholo-data/ailang/actions/runs/39606168604 (red: test 119242793153 step 13, Windows 119242954630 step 11), https://github.com/sunholo-data/ailang/actions/runs/39606168603 (macOS 119242793748 step 7), https://github.com/sunholo-data/ailang/actions/runs/39589515608 (parent, green)
- d25d9b8d commit message — TestLiveArgonPin findings; "Billing SKU confirmation pending"
- `internal/executor/managed_agents/devapi.go` — pin grammar and its "request, not verified fact" contract
- `internal/modelreg/resolve.go` — resolution tiers and the ambiguity guard this design must not disturb
- [Design Axioms](/docs/references/axioms)

## Future Work

- Rate-by-reference (`pricing.ref:`) so pinned rows stop carrying copied numbers at all — propose after the 2026-11-30 Argon repricing exercises the parity gate once
- Widening `pricing.evidence` to all billed rows
- A billing-export reconciliation tool: diff a GCP billing export's SKUs against `pricing.evidence` claims and auto-draft the promotion commit — turns the manual procedure in Example 3 into a checked one
- Decide the fate of the managed_agents executor's stale hardcoded `CostModel()` fallback (see Deferred Decisions)

---

**Document created**: 2026-10-21
**Last updated**: 2026-10-21
