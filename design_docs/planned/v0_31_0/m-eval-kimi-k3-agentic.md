# M-EVAL-KIMI-K3-AGENTIC: onboard Kimi K3 as an agentic suite model (OpenRouter × motoko/pi harnesses)

**Status**: Planned (Mark 2026-07-19 — "Kimi K3 did very well on the evals — lets look into using
it within the suite via open router and Pi or motoko harness")
**Target**: v0.31.x — eval infrastructure, no language surface
**Priority**: P2, **HARD-SEQUENCED AFTER [m-eval-reasoning-model-fairness](../m-eval-reasoning-model-fairness.md) (P1)**
— K3 is an always-reasoning model; agentic numbers taken BEFORE the reasoning-budget/finish_reason
fixes land would inherit exactly the artifact class that doc exists to kill (truncated answers,
negative token counts, empty code capture). Do not spend rig/API time measuring with a broken ruler.
**Estimated**: ~0.5–1d (config + tiered runs + comparison write-up)
**Dependencies**: OpenRouter key (present); motoko harness (precedent: `motoko-or-kimi-k2-6`,
v0.19.0 sweet-spot probe); pi harness (same executor contract); the fairness fixes above
**Author**: interactive session with Mark, 2026-07-19

---

## Why (the data)

v0.30.0 standard baseline, AILANG: **or-kimi-k3 = 97/109 (89.0%)** — the strongest OpenRouter
model on the board, ahead of or-glm-5.2 (88/109), or-kimi-k2-7-code (88/109), or-glm-5.1
(85/109). If that strength holds AGENTICALLY (multi-turn, tool-use, per-edit `ailang check`
feedback), K3 becomes a serious cheap-executor candidate for the suite and potentially the
mission fleet's evidence-based assignment table (Phase E) — a mid-cost lane between the local-GPU
free tier and frontier metered lanes.

## Investigation plan

1. **M1 — roster entries (mechanical):** add `motoko-or-kimi-k3` (mirror the `motoko-or-kimi-k2-6`
   block: `agent_cli: motoko`, `agent_model_name: openrouter/moonshotai/kimi-k3`, per-model
   `max_cost_usd` budget) and `pi-or-kimi-k3` (same, pi harness). Pricing from OpenRouter's listed
   K3 rates — verify at add time, NO fallback guesses (Critical Principle 2).
2. **M2 — tiered agentic runs:** smoke tier first (cheap gate), then core; both harnesses.
   Reasoning-model discipline from the fairness doc applies: reasoning-aware budgets, per-turn
   `finish_reason` capture, fail-loud truncation. Compare four ways: K3-agentic vs K3-standard
   (does agency help or hurt it), vs K2.6-agentic (family delta), vs GLM-5.x-agentic (the
   OpenRouter rivalry), vs the harness pair motoko-vs-pi (harness effect at fixed model).
3. **M3 — verdict + routing evidence:** write the comparison into the eval analysis + a routing
   evidence row `(openrouter, motoko|pi, kimi-k3, task-class, score, $)`; if K3-agentic clears
   the bar the sweet-spot criteria define, propose it to the fleet's Phase-E assignment table
   (proposal only — fleet admission stays a Mark/routing-policy decision).

## Harness-selection rule (Mark 2026-07-19: "Pi or Motoko depending on what is best for our
codebase — Pi since it's Go, Motoko if it's AILANG")

**The harness follows the task's language.** motoko is the AILANG-specialized harness (per-edit
`ailang check` feedback, AILANG teaching); pi is the deliberately-minimal general multi-provider
harness (`@mariozechner/pi-coding-agent`) — the right neutral fit for Go/general work. Applied:
- **Eval-suite benchmarks are AILANG tasks → motoko is K3's PRIMARY suite harness**; pi runs as
  the cross-harness CONTROL (the suite's existing pairing discipline) — M2's motoko-vs-pi
  comparison is the empirical test of this very rule at fixed model.
- **Fleet executor lanes** (if M3 proposes admission): work on THIS repo is Go → the **pi** lane
  is the candidate; work on Ailang World is AILANG → the **motoko** lane. The Phase-E assignment
  table gets `(provider, harness, model) × task-LANGUAGE` as an explicit dimension — decided by
  the M2 evidence, not assumed.

## Guardrails
- Metered OpenRouter $ — per-model `max_cost_usd` caps + the mission's per-iteration metered
  ceiling apply; smoke-before-core keeps the failure case cheap.
- Local GPU NOT involved (OpenRouter-served) — no `rig.lock`; but the os-rolling rotation shares
  the OpenRouter key's rate limits — run outside the nightly window or accept queuing.
- K2.6 history note (from its models.yml entry): Kimi needed long multi-turn iteration under
  opencode (~90–180s on csv_to_json) — expect K3 similar; set harness timeouts accordingly
  before concluding anything about capability (harness-first diagnosis, always).

## Verification log
| Claim | Method | Result |
|---|---|---|
| K3 standard = 97/109, top OpenRouter model | v0.30.0 baseline sweep | Confirmed 2026-07-19 |
| Agentic Kimi precedent exists | models.yml `motoko-or-kimi-k2-6` + `opencode-or-kimi-k2-6` (v0.19.0) | Confirmed |
| K3 is a reasoning model → fairness dependency | m-eval-reasoning-model-fairness names the class; K3 added to roster 2d83426d4 | Confirmed |

---
**Document created**: 2026-07-19 (interactive; expect quorum-at-pick)
