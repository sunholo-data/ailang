# M-PROVIDER-FAILOVER: same-weights route failover for transient infrastructure failures

**Status**: Planned
**Target**: v0.34.1 (harness-only, no release-blocking surface)
**Priority**: P1 — directly targets the two measured incidents (OR qwen 429 invalidation 2026-08-31; Lyceum frontier 504s 2026-09-03)
**Estimated**: 0.75 day (~120 impl + ~120 test LOC)
**Dependencies**: M-LYCEUM-PROVIDER (merged to dev: provider, 3 rows, M3 decision note). Latency telemetry (llm_wall_ms/ttft_ms, merged).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Failover chain is derived deterministically from registry `model_family` + explicit run config; same inputs → same route order |
| A2: Replayability | +1 | Every failover event is banked (`failover_from`/`failover_to` + first error in stderr); a replay sees exactly what happened |
| A3: Effect Legibility | 0 | Harness-internal; no AILANG effect surface changes |
| A4: Explicit Authority | +1 | Failover is opt-in per run (`--provider-failover`); EU-residency ordering is explicit, never ambient |
| A5: Bounded Verification | 0 | No language semantics touched |
| A6: Safe Concurrency | +1 | Chain state is per-job (single-threaded per benchmark trial, same as today's lastRoute) |
| A7: Machines First | +1 | Turns wasted banked failures (0-token api_error rows) into measurements; fewer re-runs |
| A8: Minimal Syntax | +1 | No new syntax; one flag + two result fields |
| A9: Cost Visibility | +1 | llm_wall_ms already accumulates across attempts; failover attempts are metered rows; budget caps apply cumulatively |
| A10: Composability | +1 | Reuses isRetryableError classification and the existing M3 latency plumbing; no new failure paths |
| A11: Structured Failure | +1 | api_error rows gain the failed-call latency + failover provenance instead of prose-only stderr |
| A12: System Boundary | +1 | Cross-provider retry is an explicit, recorded boundary crossing (jurisdiction noted) |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): chain order derived from registry + flags, not environment
- [x] A3 (Effects): no hidden side effects; all attempts banked
- [x] A4 (Authority): opt-in flag; no ambient provider switching
- [x] A7 (Machines First): reduces wasted measurement, not human convenience

### Decision Thresholds

Net +9, no hard violations → proceed to implementation.

## Problem Statement

Transient provider-infrastructure failures currently bank as permanent 0-token
`api_error` rows:

- 2026-08-31: OR qwen3-8-flash smoke invalidated **19/23** by upstream Alibaba-pool 429s.
- 2026-09-03: Lyceum frontier A/B — 6/8 calls died at their gateway (504 upstream), 4/6
  reproduced on manual retry; every one banked as a 0-token failed row.
- M-EVAL-STANDARD-CONFIDENCE-GATING and rotation cohorts then read these as model failures
  unless someone manually re-runs and deletes artifacts.

The M-LYCEUM-PROVIDER M3 A/B proved the fix surface: the SAME open-weight release exists on
2–3 metered routes per family (`model_family` already groups them), and per-call latency is
now banked, so failover is observable. Retrying the same weights on a different route converts
"infrastructure died" rows into measurements — without changing the model, the prompt, or the
seed.

## Goals

1. On a retryable INFRASTRUCTURE failure (429 non-quota, 502/503/504, timeout, connection),
   retry the same benchmark attempt on the next route carrying the same weights.
2. Bank everything: which route failed, which served, first error text, per-call latency.
3. Keep it bounded: max ONE failover per generation attempt (no loops), remaining-budget aware.
4. Never fail over on: quota exhaustion (existing carve-out), 4xx client/auth errors,
   reasoning stalls / finish=length (same weights ⇒ same behavior — failover would just pay
   twice for model behavior), or anything post-generation (compile/execute).

## High-Impact Decisions

| Decision | Choice | Decider | Change cost |
|----------|--------|---------|-------------|
| D-F1: Default state | **OFF by default**; enabled per run via `--provider-failover on` (or `AILANG_PROVIDER_FAILOVER=on`). Retry-storm history (m-coordinator-child-env-opencode-retry-storm) says land bounded + observed first. | Mark | Low (flip a default later with data) |
| D-F2: Chain order | `eu` = [primary, lyceum, …rest]; `auto` = [primary, then family siblings in registry order]. Default `auto`. | Mark (flag semantics) | Low |
| D-F3: Failover budget | 1 failover per generation attempt; the failover call inherits the REMAINING row budget (max_cost_usd cumulative across attempts; hard_timeout minus elapsed wall, floor 60s). | Agent latitude (bounded by acceptance criteria) | — |
| D-F4: EU residency | Documented per row: OR→Lyceum failover moves data US→EU (zero data retention on both). EU-resident runs should set `--provider-failover eu,lyceum-glm-5-3-flash` style ordering or pin the model row outright. | Mark (ratified with M3 decision note) | — |

### Design Freeze

- [x] D-F1 default OFF — frozen (bounded-first landing, per the retry-storm precedent)
- [x] D-F2 family-sibling chains only — frozen (cross-family = model selection, a Non-Goal)
- [x] D-F3 single failover, remaining-budget — frozen
- [ ] D-F4 residency note wording — wording latitude granted to executor; substance frozen

## Solution Design

### Overview

A `RouteChain` (new, ~40 LOC) resolves the failover order for a primary model row from the
registry: all rows sharing `model_family` (same weights) minus the primary, ordered by D-F2.
The AIAgent's generation funnel (`adapter.generate`) consults the chain only when
`--provider-failover` is on, the failing error is failover-class (shared classifier with the
error categorizer — one definition, two consumers, as the quota carve-out already does), and
budget/attempt gates pass. Each attempt goes through the existing provider dispatch (M1/M3
plumbing: every route is already a first-class row with auth, pricing, budgets).

### Architecture

```
RepairRunner.runSingleAttempt
  └─ AIAgent.GenerateCodeSplit          (unchanged signature)
       └─ adapter.generate              (unchanged signature)
            └─ NEW: failoverLoop (eval_harness/route_chain.go)
                 attempt 1: primary row (today's exact path)
                 on retryable-infra error AND failover enabled AND attempts<2:
                   attempt 2: next family sibling row (same weights, other provider)
                 bank: primaryRoute, failoverFrom, failoverTo, firstError, llm_wall_ms (accumulates)
```

- Registry: no schema change — `model_family` already declares the grouping. The oc row
  (provider `ollama`) joins chains by the same key, giving or→oc→lyceum depth for free.
- Step/agent-mode paths are untouched (Phase B; see Non-Goals). The retry-storm incident
  happened in coordinator child dispatch — this design does NOT go near that layer.

### Implementation Plan

1. `internal/eval_harness/route_chain.go` (~60 LOC): `RouteChain` — derivation from
   `modelreg` by `model_family`; `Next(err)` gate = shared retryable classifier + budget checks.
2. `internal/eval_harness/ai_agent.go` (~25 LOC): consult chain in the generation funnel when
   enabled; return the serving route + failover metadata on `GenerateResult`.
3. `internal/eval_harness/metrics.go` (~8 LOC): `FailoverFrom`, `FailoverTo`,
   `RouteOrder` fields (`omitempty`, provenance semantics same as `cost_provenance`).
4. `cmd/ailang/eval_suite.go` (~15 LOC): `--provider-failover` flag (`off|auto|eu`, default off)
   + per-model resolution + loud log line on every failover event.
5. `internal/modelreg/models.yml` (~15 LOC): one-line role note per family row pointing at the
   chain (glm-5-3-flash: or→oc→lyceum; qwen3-8-flash: or→lyceum; kimi: or→lyceum).

### Files to Modify/Create

- `internal/eval_harness/route_chain.go` (new) + `route_chain_test.go`
- `internal/eval_harness/ai_agent.go` (failover funnel)
- `internal/eval_harness/metrics.go` (provenance fields)
- `cmd/ailang/eval_suite.go` (flag wiring)
- `internal/modelreg/models.yml` (chain annotations)

## Examples

### Example 1: OR Alibaba-429 storm (the 2026-08-31 case)

`--tier smoke --models or-qwen3-8-flash --provider-failover auto` — or-qwen3-8-flash hits
Alibaba-pool 429s; chain = [or, lyceum] (family qwen3-8-flash). Attempt 1 fails 429 →
failover attempt on `lyceum-qwen3-8-flash-next` → banked row carries
`failover_from=or-qwen3-8-flash`, `failover_to=lyceum-qwen3-8-flash-next`, the 429 text, and
cumulative llm_wall_ms. Suite completes with measurements instead of 19 invalidated rows.

### Example 2: Lyceum frontier 504 (the 2026-09-03 case)

`--tier frontier --models lyceum-glm-5-3-flash --provider-failover eu` — chain = [lyceum, or]
(EU first). Gateway 504 after ~5min → failover to `or-glm-5-3-flash` with the remaining budget
→ row completes on OR, banked with `failover_to=or-glm-5-3-flash` and both calls' latency.
The M3 caveat (Lyceum long-horizon instability) stops being a data-loss event.

## Success Criteria

- [ ] SC1: A seeded transient failure (mock provider) fails over once, banks provenance, and
      never loops (unit).
- [ ] AC-A: 2026-09-03 frontier 504 cells, re-run with `--provider-failover eu`, bank as
      completed rows (live validation on 2+ cells).
- [ ] AC-B: non-retryable classes (400, quota exhaustion, reasoning stall) do NOT fail over (unit).
- [ ] AC-C: cumulative `max_cost_usd` and remaining-time budget respected across the chain (unit).
- [ ] AC-D: aggregates can exclude/flag failover rows via the banked provenance fields.

## Testing Strategy

- Unit: chain derivation from models.yml fixtures (family with 2 and 3 rows; missing family);
  classifier gating (each error class in/out); budget propagation; provenance fields; idempotent
  ordering (`eu` vs `auto`).
- Integration: existing eval_harness suite green; a two-model dry-run plans the chain and prints
  it (`--dry-run` shows the route order per model — planned-runs visibility).
- Live: AC-A cells on the studio (keys already verified there).

## Deferred Decisions

- Agent-mode/coordinator failover (Handler-level chain) — Phase B, own design doc; the
  retry-storm incident's authority surface lives there and must not be touched in passing.
- Whether rotation/nightly lanes enable failover by default — revisit after one release cycle
  of banked failover events.
- Cross-family failover (different weights as backup) — that is model selection, explicitly out.

## Non-Goals

- No Handler/agent-mode/coordinator changes (Phase B; retry-storm authority surface).
- No cross-family failover, no model-quality fallbacks (only same-weights routes).
- No failover on quota exhaustion, auth errors, client errors, or model-behavior failures.
- No new AILANG syntax, stdlib, or effect-surface changes.

## Timeline

| Phase | Work | Est |
|-------|------|-----|
| M1 | RouteChain + classification + provenance + tests | ~0.4d |
| M2 | eval-suite flag wiring + dry-run visibility + row notes | ~0.2d |
| M3 | Live validation (AC-A cells) + decision-note update | ~0.15d |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Retry storms (fleet history: coordinator incident) | Max 1 failover per attempt; per-row budget caps enforced cumulatively; eval-only scope; OFF by default |
| Silent route switches contaminate A/B cohorts | D-F1/D-F2 provenance fields are mandatory on every failover row; eval-matrix/elo filters read them |
| Jurisdiction drift on EU-resident work | D-F4 row notes + `eu` ordering; never ambient |
| Failover masks a route's real reliability (the M3 question) | Failover events are counted; route-reliability KPIs read the primary route's rows only |
| Wasted spend double-billing a dying route | 504-class failures carry llm_wall_ms (M3); budget caps cut the chain off mid-flight |

## Related Documents

- design_docs/planned/m-lyceum-provider.md + sprint plan (M3 record: route verdicts)
- design_docs/planned/v0_35_0/m-coordinator-route-authority-recovery.md (retry-storm authority surface — the layer this design deliberately avoids)
- internal/eval_harness/ai_agent.go `isRetryableError` (classification to reuse; quota carve-out shared with the categorizer)

## References

- M3 A/B record: m-lyceum-provider-sprint-plan.md (2026-09-03, three-route frontier + smoke)
- OR qwen 429 invalidation: or-qwen3-8-flash row notes (2026-08-31, 19/23 invalidated)