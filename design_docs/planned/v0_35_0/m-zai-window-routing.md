# M-ZAI-WINDOW-ROUTING: Time-Window-Aware Model Routing (z.ai GLM Coding Plan)

**Status**: Planned — Phase 0 (free discovery spike) UNGATED; Phases 1+ BLOCKED on D1
**Target**: v0.35.0
**Priority**: P1
**Estimated**: Phase 0 ~4h · Phases 1–3 ~3 days (only if Phase 0 clears)
**Dependencies**: M-LYCEUM-PROVIDER (provider template), M-OLLAMA-CLOUD-PROVIDER (subscription cost-provenance precedent)
**Planner-Lane**: opus-required (cost/KPI semantics + external-vendor premises)

---

## Problem Statement

z.ai sells GLM-5.3 / GLM-5.3-Flash through a **credit-based subscription** whose effective rate
depends on the **time of day**, and is currently running a **time-boxed campaign** that further
multiplies quota. Both windows land favourably for a CET-based fleet. We have no mechanism that can
see a clock, so today we cannot exploit either — and, worse, if we hard-coded the campaign we would
silently over-plan work after it lapses.

**Current State:**

- The fleet runs GLM-5.3-Flash on **four metered routes already** (`or-glm-5-3-flash`,
  `lyceum-glm-5-3-flash`, `oc-glm-5-3-flash`, `opencode-or-glm-5-3-flash`) — V24. None is
  subscription-backed, and none is time-aware.
- `internal/modelreg` has **no notion of time of day** (V18): `time.Now()` does not appear in any
  non-test file in the package. Pricing is a single static rate card per row.
- A **dated** pricing mechanism exists (`Pricing.Expires` + `Next *ScheduledPricing`, V17) but it is
  a one-hop future rate card enforced by a CI drift-checker — deliberately **not** chainable and not
  recurring. It cannot express "every day between 23:00 and 09:00".
- The mission loops that would benefit are `mission-control` (90 min), `mission-docs` (6 h) and
  `mission-motoko` (13 h) — all `StartInterval`, so they **drift against the wall clock and have no
  fixed hour to move** (V20).
- The designer rotation has **one usable authoring lane** (Fable) for structural reasons, so any doc
  that blocks at quorum exceeds the Fable diet by construction (V22). This is a documented,
  3-instance defect whose stated fix is to widen the rotation — and it needs a human.
- `m-eval-batch-api.md` already **defers** time-of-day pricing as a separate lever, citing DeepSeek's
  2× Beijing-peak pricing (V25). z.ai is the second instance of the same pattern.

**Impact:**

- **Cost/throughput**: the mission loop's authoring bottleneck is a *quota* constraint on a metered
  Anthropic bucket. A flat-rate lane relieves it directly.
- **Silent-fallback risk**: the campaign expires 2026-09-20. Any implementation that does not encode
  expiry will, on 2026-09-21, plan 2× the work against a 1× quota and surface as mystery
  `quota_exhausted` rows rather than as stale config. This is precisely the failure class CLAUDE.md §2
  exists to prevent, and the `Pricing.Expires` doc comment (V17) documents the identical lesson for
  Gemini 3.7 Flash's 2027 reversion.

---

## Verification Log

Every load-bearing claim, with its check. **External rows are vendor claims fetched on 2026-09-03 and
cannot be re-checked from the repo** — they are the reason quorum trigger #4 fires.

### External (z.ai) — fetched 2026-09-03

| # | Claim | Evidence | Status |
|---|---|---|---|
| V1 | Peak = **Mon–Fri 14:00–18:00 SGT (UTC+8)**; off-peak bills at **50%** of standard credit rate | [docs.z.ai/devpack/overview](https://docs.z.ai/devpack/overview) | Confirmed |
| V2 | Credits: Lite 2,000/5h · 10,000/wk; Pro 12,000 · 60,000; Max 28,000 · 140,000 | same | Confirmed |
| V3 | Campaign **2026-09-03 → 2026-09-20 SGT**, daily **23:00–09:00 SGT**, **GLM-5.3-Flash only**. ZCode = "zero quota consumption for unlimited usage"; other supported agents = "available quota is **doubled**". All paid plans, no activation | [notice/event-glm-5.3-flash](https://docs.z.ai/devpack/notice/event-glm-5.3-flash) | Confirmed |
| V4 | That window in our clock = **17:00–03:00 CEST** (23:00 SGT = 15:00 UTC = 17:00 CEST); same calendar day either side; campaign ends before the Oct DST change, so CET never applies to it | computed, `TZ=` cross-check 2026-09-03 | Confirmed |
| V5 | Plan usable **only in officially supported tools**; violations → "rate limiting, account freezing"; **>3 violations may be banned** | [devpack/usage-policy](https://docs.z.ai/devpack/usage-policy) | Confirmed |
| V6 | PAYG: GLM-5.3 **$1.40/$4.40** per 1M; Flash **$0.075/$0.25**; Flash 50% promo ends **2026-09-09 24:00 SGT** | [guides/overview/pricing](https://docs.z.ai/guides/overview/pricing) | Confirmed |
| V7 | Endpoints live and auth-gated: `/api/anthropic/v1/messages` → **401**; `/api/coding/paas/v4/models` and `/api/paas/v4/models` → **`{"error":{"code":"1001"…}}`** | live curl, 2026-09-03 | Confirmed |
| V8 | ZCode is a **macOS GUI IDE**; no CLI, headless mode, or automation entry point documented | [zcode.z.ai/en/docs/configuration](https://zcode.z.ai/en/docs/configuration); `/Applications/ZCode.app` absent locally | **Doc-only — P0.1 must confirm by installing** |
| V9 | Credit model effective **2026-07-30** (replaced prompt-count limits); weekends off-peak all day | [notice/usage-revision](https://docs.z.ai/devpack/notice/usage-revision) | Confirmed |
| V10 | Credit multipliers ≈ **6.9 input / 1.7 cached / 24 output per 10k tokens** | secondary blog only; **absent from z.ai docs** | ⚠️ **UNVERIFIED — do not size a plan on this alone** |

### In-repo

| # | Claim | Evidence | Status |
|---|---|---|---|
| V11 | `ProviderLyceum` is a complete template for an OpenAI-compatible provider | `internal/ai/config.go:18,103-110,128,158,184`; `internal/eval_harness/ai_provider.go:91` | Confirmed |
| V12 | `AuthLaneSubscription` → `CostListPriceEquivalent` already exists | `internal/executor/cost.go:132,148` | Confirmed (reuse) |
| V13 | **Negative**: `AuthLaneForModel` recognises *only* Ollama Cloud as a subscription route; every other model returns `AuthLaneBilled` | `internal/executor/cost.go:217-222` | Confirmed — must be extended |
| V14 | `quota_exhausted` error category exists | `internal/eval_harness/metrics.go:201` | Confirmed (reuse) |
| V15 | `isRetryableError` **already** checks quota exhaustion *before* the `"429"` substring rule | `internal/eval_harness/ai_agent.go`, `isQuotaExhaustion` | Confirmed — M-OLLAMA-CLOUD AC8 landed; **no change needed**, but z.ai's 5h/weekly reset must match `isQuotaExhaustion` |
| V16 | **Negative**: no `offpeak` / `off-peak` identifier anywhere in `cmd/` or `internal/` Go | grep, empty | Confirmed |
| V17 | `Pricing.Expires` + `Next *ScheduledPricing` exist; `ScheduledPricing` is **deliberately not** a `*Pricing` to forbid chains | `internal/modelreg/models.go:53-83` | Confirmed |
| V18 | **Negative**: no time-of-day resolver — `time.Now()` absent from non-test `internal/modelreg/*.go` | grep, empty | Confirmed |
| V19 | `Expires` is enforced by `tools/verify-model-pricing` (`TestCheckSchedules`, `TestCheckSchedules_IdenticalSuccessorIsNotFlagged`, `main_test.go:54,156`) — **the test name cited in `models.go:65`, `TestModels_PricingScheduleIsHonoured`, DOES NOT EXIST** | grep both | Confirmed — **stale citation, file separately** |
| V20 | launchd split: `nightly-eval` (03:00, local ollama) and `os-rotation-filler` (2700 s, local ollama) are **GPU-bound, not price-bound**; only `mission-control` (5400 s), `mission-docs` (21600 s), `mission-motoko` (46800 s) route to cloud | `tools/launchd/*.plist` | Confirmed |
| V21 | Gate 3 resolves each heavy role's model from env (`$MISSION_DESIGNER_MODEL`, `$MISSION_EXECUTOR_MODEL`, …), spawned as model-pinned sub-agents | `.claude/skills/mission-control/SKILL.md` §Gate 3 | Confirmed |
| V22 | Designer rotation has one usable authoring lane; widening it is a routing-policy change requiring a human | same | Confirmed |
| V23 | Quorum reviewers are `gpt5-6-sol` (OpenAI) and `gemini-3-1-pro` (Google); `oc-glm-5-2` (**Z-AI**) is named as a reviewer elsewhere in the rotation rationale | `internal/mission/quorum/call.go:152`; SKILL.md | Confirmed — **vendor-collision risk for a GLM designer (D4)** |
| V24 | Existing GLM-5.3-Flash rows across 4 routes; `opencode-or-glm-5-3-flash` carries **AGENT SMOKE GATE: PENDING** | `internal/modelreg/models.yml:2179+,4386,4700` | Confirmed |
| V25 | Time-of-day pricing already deferred once as a distinct lever | `design_docs/planned/m-eval-batch-api.md:234,288` | Confirmed |

---

## Goals

**Primary Goal:** Let the fleet route work by **clock and calendar**, so that a provider's
time-varying rate or quota is exploited deliberately — and expires safely — rather than being
invisible to the harness.

**Success Metrics:**

1. A single declarative window source; **zero** hard-coded campaign dates in Go or in skills.
2. An expired window is **inert by construction** — proven by a test that runs the clock past
   2026-09-20 and asserts the multiplier returns to 1.0.
3. `ailang offpeak status --json` answers "what is the multiplier right now, for which models" in
   one call, consumable by Gate 3 without shelling into Go internals.
4. A subscription-lane GLM row banks `cost_provenance: list-price-equivalent`, **never** `metered`
   and **never** `free-local`.
5. Phase 0 answers the capability question (**is GLM good enough to author/execute?**) for **< $0.20**
   on routes we already pay for, before any subscription is purchased.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| **D1**: Buy a plan at all, and which tier | Everything after Phase 0 depends on it. V10 (the credit exchange rate) is unverified, so tier sizing rests on a blog number — Phase 0 must not pretend otherwise | **human (Mark)** | design | high |
| **D2**: Does the subscription lane enter **banked eval rotations**, or stay mission-loop-only | A resetting 5h/weekly credit bucket can starve a rotation mid-run and bank a cohort with a hole. M-OLLAMA-CLOUD D2 ruled *no* on the identical question | **human (Mark)** | design | high |
| **D3**: Imputed prices for subscription GLM rows | `0/0` maps to the positively-false `free-local` (V12/V13). Mechanism already exists; only the numbers are open | **human (Mark)** | design | med |
| **D4**: Is a GLM designer acceptable given `oc-glm-5-2` (Z-AI) sits in the reviewer set (V23) | The rotation rationale explicitly chose kimi for being independent of *all three* reviewers. A GLM designer re-introduces a vendor-level collision — not the model-level one the rule forbids, but the same hazard | **human (Mark)** | design | med |
| **D5**: Where windows are declared — provider-level block in `models.yml` vs a sibling file | `ModelConfig` is per-model and has no provider-level section today; windows are a provider property, not a model property | agent | implementation | low |
| **D6**: New mission vs. enhancing existing loops | Scope. A new mission adds a charter, heartbeat, recovery job; enhancing Gate 3 does not | **human (Mark)** | design | med |

### Design Freeze

- [ ] **D1** — plan purchased (tier named), or Phase 0 declared sufficient and Phases 1+ dropped
- [ ] **D2** — subscription rows in banked rotations: yes/no
- [ ] **D3** — imputed rate card for subscription GLM rows
- [ ] **D4** — GLM-designer vendor-collision ruling
- [ ] **D6** — new mission vs. Gate 3 enhancement

**Phase 0 is deliberately outside the freeze** — it is free, reversible, and exists to inform D1/D4.

---

## Solution Design

### Overview

Three separable pieces. Only the first is unconditional.

1. **A window engine** (`internal/pricing`): declarative, dated, recurring rate/quota windows with a
   pure `Resolve(provider, model, t)`. No network, no vendor names in code.
2. **A metered `zai` provider**: a ~2-hour clone of M-LYCEUM-PROVIDER against `/api/paas/v4`, giving
   a **route control** that pins z.ai first-party as the upstream instead of OpenRouter's 16-way
   fan-out. Independent of any subscription decision, and independent of the window engine.
3. **Window-aware role routing** in mission Gate 3: consult the engine, prefer the GLM lane when a
   window is open, fall back to the current pin otherwise.

### Why not reuse `Pricing.Expires` (V17)

`Expires`/`Next` is a **one-hop, whole-day** rate card whose enforcement is a CI drift-check — its
own comment forbids chaining, because "a schedule that could itself carry a schedule invites a chain
of future prices nobody has verified". Our windows are **recurring and intra-day** (daily 23:00–09:00;
Mon–Fri 14:00–18:00), which that type cannot express and should not be bent to.

They compose rather than compete, and the split is clean:

- **`Expires`/`Next`** = "the rate card changed" → the **GLM-5.3-Flash PAYG promo lapsing 2026-09-09**
  (V6) is exactly this, and should be filed as an `expires:` row on the new `zai-glm-5-3-flash` model.
- **Window engine** = "the same rate card is applied at a different multiplier depending on the clock".

### Architecture

**Components:**

1. **`Window`** — `{tz, days, start, end, multiplier, applies_to, valid_from, valid_until, source}`.
   `valid_until` is what makes the campaign inert on 2026-09-21 without a code change.
2. **`Resolve(provider, model, t) → (rateMult, quotaMult, []activeWindow)`** — pure, total, and
   `t`-injected so tests drive the clock. Overlapping windows compose multiplicatively; each
   contributing window is *named* in the result so a decision is explainable rather than a bare float.
3. **`ailang offpeak status [--model X] [--at RFC3339] [--json]`** — the seam Gate 3 uses. A CLI query,
   not Go internals, so a skill can consult it without importing the harness.
4. **`AuthLaneForModel` extension** — V13's Ollama-Cloud-only check gains a z.ai-subscription case,
   keeping one definition of "subscription lane" rather than two.

### Conflict Surface

Not strictly required (no parser/typechecker/codegen files), but the M-OLLAMA-CLOUD precedent wrote
one anyway and it caught two real defects. Doing the same.

| Existing machinery | What it does today | Our need | Decision | Rejected alternative |
|---|---|---|---|---|
| `Pricing.Expires`/`Next` (V17) | One-hop whole-day rate card, CI-enforced | Recurring intra-day windows | **Extend, don't bend** — new type, and file the Flash PAYG promo as a genuine `expires:` row | Encoding a daily window as a chain of `Next` hops: rejected, the type explicitly forbids chains |
| `AuthLaneForModel` (V13) | Ollama Cloud → Subscription; everything else → Billed | z.ai plan rows are a subscription lane | **Extend** with a z.ai case | A second parallel notion of "subscription": rejected, forks the concept |
| `ResolveCostProvenance` (V12) | Subscription → `list-price-equivalent`; `0/0` → `free-local` | Plan rows must not read as free or as spend | **Reuse unchanged**; impute non-zero prices (D3) | New provenance constant: rejected, the existing one describes this verbatim |
| `isRetryableError` (V15) | Already excludes quota exhaustion before the `"429"` rule | z.ai 5h/weekly exhaustion must not be retried into | **Reuse** — but *verify* z.ai's exhaustion string matches `isQuotaExhaustion`; if it does not, the guard silently does not apply | Assuming it matches: rejected, that is the V15 trap wearing a new vendor's clothes |
| `quota_exhausted` category (V14) | Distinct from `rate_limit` and `api_error` | Same need | **Reuse** | New category: rejected, duplicates a concept |
| Gate 3 role env (V21) | Per-role model pins from env | Time-varying preference | **Extend** the resolution step, not the pins | A launchd gate: rejected — the jobs are `StartInterval` and drift (V20), and it lacks role granularity |

**What deliberately changes:** `AuthLaneForModel` stops being Ollama-Cloud-specific. Everything else
is additive.

### Implementation Plan

**Phase 0 — free discovery spike (~4 h, NO money, gates everything)**

- [ ] **P0.1** Install ZCode; inspect the bundle for any CLI/headless entry point. **This single check
      decides whether "zero quota, unlimited" is reachable by automation at all (V8).** If it is
      GUI-only, the unlimited lane is attended-only and must be described as such everywhere.
- [ ] **P0.2** Transport probe: point `opencode` and `claude` at the z.ai base URLs with a deliberately
      invalid key; confirm a clean **401** rather than a transport/shape error. V7 already shows all
      three endpoints live and auth-gated, so this is confirming *our client*, not their server.
- [ ] **P0.3** **Capability, on routes we already pay for**: run the pending agent smoke gate for
      `opencode-or-glm-5-3-flash` (V24). Standard-mode precedent for this model was **$0.055 for 23
      benchmarks**, so this is cents. It answers "can GLM author/execute?" with **no subscription**,
      and it is a gate we already owe.
- [ ] **P0.4** Build the window engine + tests entirely offline (pure time math).
- [ ] **P0.5** Verify z.ai's quota-exhaustion error string against `isQuotaExhaustion` (V15) — from the
      docs if a sample exists, else mark PENDING for Phase 1.

**Phase 1 — metered `zai` provider (~2 h, needs a PAYG key only)**

- [ ] `ProviderZAI` + `ZAIBaseURL()` (+`ZAI_BASE_URL` override), mirroring V11 exactly
- [ ] One `case` in `newProviderAdapter`; `ZAI_API_KEY` in both provider switches
- [ ] Register `zai-glm-5-3` and `zai-glm-5-3-flash`, opt-in, **with the Flash promo as an
      `expires: "2026-09-09"` + `next:` row** (V6) — the honest use of V17
- [ ] Smoke gate vs the `or-` twin; this is a **route control**, identical list price, not a cost play

**Phase 2 — subscription lane (~1 day, BLOCKED on D1)**

- [ ] Extend `AuthLaneForModel`; assert no plan row can bank `metered` or `free-local`
- [ ] Imputed pricing per D3
- [ ] Confirm exhaustion → `quota_exhausted`, non-retryable (inherit V15)

**Phase 3 — window-aware Gate 3 (~1 day, BLOCKED on D4/D6)**

- [ ] `ailang offpeak status --json`
- [ ] Gate 3 consults it when resolving heavy-role models; records the chosen lane **and the window
      that justified it** in the evidence row

### Files to Modify/Create

**New files:**
- `internal/pricing/window.go` (~180 LOC) — `Window`, `Resolve`, multiplicative composition
- `internal/pricing/window_test.go` (~220 LOC) — clock-injected; DST, expiry, overlap, weekend
- `cmd/ailang/offpeak.go` (~120 LOC) — `ailang offpeak status`

**Modified files:**
- `internal/ai/config.go` (+30 LOC) — `ProviderZAI`, `ZAIBaseURL`, two switch arms
- `internal/eval_harness/ai_provider.go` (+5 LOC) — one `case`
- `internal/executor/cost.go` (+15 LOC) — `AuthLaneForModel` z.ai case
- `internal/modelreg/models.yml` (+80 LOC) — `zai-*` rows; provider-level `windows:` block (D5)
- `internal/modelreg/models.go` (+40 LOC) — window declaration schema; **fix the stale test citation at
  `:65`** (V19)
- `.claude/skills/mission-control/SKILL.md` — Gate 3 window step (Phase 3 only)

---

## Examples

### Example 1: The campaign expires without anyone touching code

```yaml
providers:
  zai:
    windows:
      - name: "standard-off-peak"
        tz: "Asia/Singapore"
        days: [Mon, Tue, Wed, Thu, Fri]
        outside: "14:00-18:00"      # peak is the exception; off-peak is everything else
        rate_multiplier: 0.5
        source: "https://docs.z.ai/devpack/overview (2026-09-03)"
      - name: "glm-5.3-flash-campaign"
        tz: "Asia/Singapore"
        window: "23:00-09:00"
        applies_to: ["glm-5.3-flash"]
        quota_multiplier: 2.0
        valid_from: "2026-09-03"
        valid_until: "2026-09-20"   # ← inert on the 21st, by construction
        source: "https://docs.z.ai/devpack/notice/event-glm-5.3-flash (2026-09-03)"
```

```console
$ ailang offpeak status --model zai-glm-5-3-flash --json
{"at":"2026-09-03T19:35:00+02:00","rate_multiplier":0.5,"quota_multiplier":2.0,
 "active":["standard-off-peak","glm-5.3-flash-campaign"],"next_change":"2026-09-04T03:00:00+02:00"}

$ ailang offpeak status --model zai-glm-5-3-flash --at 2026-09-21T20:00:00+02:00 --json
{"rate_multiplier":0.5,"quota_multiplier":1.0,"active":["standard-off-peak"],
 "expired":["glm-5.3-flash-campaign"]}
```

### Example 2: Gate 3 picks a lane, and says why

```
Gate 3 · designer
  window check: zai-glm-5-3-flash → quota×2.0, rate×0.5 (standard-off-peak, glm-5.3-flash-campaign)
  lane: opencode/zai-coding-plan/glm-5.3-flash   [window-preferred]
  evidence row: designer=zai-glm-5-3-flash, window=glm-5.3-flash-campaign, expires=2026-09-20
```

Outside the window the same step resolves to the existing pin, and records `window=none`.

---

## Success Criteria

- [ ] **AC1** `Resolve` is pure and clock-injected; no `time.Now()` in the engine's decision path
- [ ] **AC2** A test at `2026-09-21T20:00+02:00` asserts `quota_multiplier == 1.0` and names the
      campaign as expired — the anti-silent-fallback gate
- [ ] **AC3** DST: the *standard* window is asserted at both CET and CEST dates and lands on
      07:00–11:00 and 08:00–12:00 local respectively
- [ ] **AC4** No z.ai row can resolve `0/0` pricing (mirrors M-OLLAMA-CLOUD AC5)
- [ ] **AC5** A subscription-lane trial banks `list-price-equivalent`, never `metered`/`free-local`
- [ ] **AC6** Every window carries a `source:` URL and a fetch date
- [ ] **AC7** Phase 0 concludes with a written yes/no on ZCode automatability (V8)
- [ ] **AC8** `make test` and `make lint` green; `models.go:65` stale citation fixed (V19)

## Testing Strategy

Pure-unit, offline. The engine takes `t`, so every case is deterministic: inside/outside window,
weekend, both DST sides, campaign before/during/after, overlapping composition, and an empty
declaration (multiplier 1.0, never 0 — a 0 would silently make everything look free).

**Not tested here**: whether z.ai *honours* its own published windows. That is unobservable from our
side without an account and a credit gauge, and is listed as a PENDING premise rather than an
assumption.

---

## Deferred Decisions

- Window YAML key naming (`window:` vs `start:`/`end:`) — agent
- `offpeak status` human-readable formatting — agent
- Whether `Resolve` returns a struct or multiple values — agent
- Whether other vendors' windows (DeepSeek's 2× Beijing peak, V25) are filed in the same sprint — human at review
- Credit-gauge polling, if z.ai exposes one — agent, Phase 2

## Non-Goals

- **Not** putting subscription rows in banked eval rotations (that is D2, and the precedent says no)
- **Not** driving the plan from `ailang eval-suite`'s own HTTP client — V5 makes that a policy
  violation with account-level consequences
- **Not** a general scheduler; this only *answers* what the multiplier is, it does not move jobs
- **Not** modelling per-request credit cost — V10 is unverified, so any such arithmetic would be
  fiction presented as a number

## Timeline

| Phase | Effort | Gate |
|---|---|---|
| 0 — discovery | ~4 h | none — start now |
| 1 — metered provider | ~2 h | PAYG key; ideally before the 2026-09-09 Flash promo lapses |
| 2 — subscription lane | ~1 day | **D1** |
| 3 — window routing | ~1 day | **D4**, **D6** |

## Risks & Mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| **Campaign expiry silently over-plans work** | **High** | AC2 — the expiry is a tested behaviour, not a comment |
| V10 (credit exchange rate) is a blog number, and tier sizing rests on it | **High** | Phase 0 buys nothing; D1 is explicitly a human decision made *after* the free evidence |
| ToS violation → rate limiting or ban (V5) | **High** | Subscription confined to supported tools; evals on a PAYG key. Lane separation is a design constraint, not a convention |
| z.ai exhaustion string doesn't match `isQuotaExhaustion` → retry into a spent bucket | Med | P0.5 checks it; PENDING if undeterminable offline |
| GLM designer collides with the Z-AI quorum reviewer (V23) | Med | D4 — ruled before Phase 3, not discovered at Gate 2 |
| Campaign ends mid-sprint, making measurements incomparable | Med | Record the active window in every evidence row (Example 2) |
| ZCode turns out to have a headless mode we dismissed | Low (upside) | P0.1 checks it first, precisely because it would change the plan |

## Axiom Compliance

### Axiom Scoring

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | `Resolve` is pure and clock-injected; the alternative (ambient `time.Now()`) is exactly the nondeterminism this avoids |
| A2: Replayability | +1 | The active window is recorded on each evidence row, so a past routing decision is reconstructible |
| A3: Effect Legibility | 0 | No new effects |
| A4: Explicit Authority | 0 | No new ambient authority; keys stay in existing env vars |
| A5: Bounded Verification | +1 | Window logic is total and locally checkable |
| A6: Safe Concurrency | 0 | Stateless |
| A7: Machines First | +1 | `--json` output designed for Gate 3, not for a human reader |
| A8: Minimal Syntax | 0 | No language syntax |
| A9: Cost Visibility | +1 | Makes a time-varying rate legible where it is currently invisible; forces the imputed-price question into the open (D3) rather than letting a plan lane bank as `0/0` |
| A10: Composability | +1 | Reuses `AuthLane`/provenance/`quota_exhausted` unchanged; composes with `Expires`/`Next` rather than replacing it |
| A11: Structured Failure | +1 | Exhaustion stays `quota_exhausted`, never `api_error` |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +7** ✅

### Hard Violation Check

- [x] A1: no implicit nondeterminism — clock is injected
- [x] A3: no hidden side effects
- [x] A4: no ambient authority granted
- [x] A7: machine-first output

### Decision Thresholds

Net ≥ +2 → proceed to draft. No −1 on A1/A3/A4/A7.

## Quorum

**Required.** Triggers **1** (design-freeze items), **3** (cost/KPI semantics — subscription cost
provenance) and **4** (load-bearing premises about an external vendor) all fire.

⚠️ Per the design-doc-creator guidance: several premises (V8, V10, and z.ai's exhaustion contract)
**cannot be closed inside a session** — they need an install, a purchase, or a live measurement. They
are therefore filed as explicitly PENDING rows and the phases gated behind Phase 0, rather than
spending re-quorum rounds on objections nobody in-session can answer.

## Related Documents

- [M-LYCEUM-PROVIDER](../m-lyceum-provider.md) — the provider template (V11)
- [M-OLLAMA-CLOUD-PROVIDER](../../implemented/v0_34_0/m-ollama-cloud-provider.md) — subscription cost
  provenance, quota exhaustion, and the D2 precedent this doc follows
- [m-eval-batch-api](../m-eval-batch-api.md) — deferred time-of-day pricing (V25); this doc is the
  general mechanism it anticipated
- [m-mission-elo-routing](../m-mission-elo-routing.md) — adjacent (routing by *strength*; this is
  routing by *clock*). Distinct: neural similarity 0.32
- [M-PROVIDER-FAILOVER](../m-provider-failover.md) — **composes directly**. That doc selects among
  same-weights routes *reactively* (on 429/504); this one selects *proactively* (on the clock). They
  meet at `model_family`: a `zai-glm-5-3-flash` row registered by Phase 1 joins the `glm-5` failover
  chain automatically, giving the campaign lane a fallback when it is exhausted or out of window.
  **Sequencing note:** land M-PROVIDER-FAILOVER first if both are scheduled — its chain is the
  natural place for "window closed → fall back to the metered twin", and building that logic
  separately here would duplicate it

## References

- z.ai: [overview](https://docs.z.ai/devpack/overview) · [campaign](https://docs.z.ai/devpack/notice/event-glm-5.3-flash) · [usage policy](https://docs.z.ai/devpack/usage-policy) · [pricing](https://docs.z.ai/guides/overview/pricing) · [plan revision](https://docs.z.ai/devpack/notice/usage-revision)
- [Design Axioms](/docs/references/axioms)

## Future Work

- DeepSeek's 2× Beijing-peak window as a second declaration (V25) — mechanism already built
- A credit gauge, if z.ai exposes one (Ollama Cloud's `/api/usage` had consumption but **no
  denominator** — expect the same limit)
- Window-aware `eval-suite` scheduling, only if D2 ever flips
