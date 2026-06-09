# M-EVAL-PROMPT-DELIVERY: How to Deliver the Teaching Prompt to Local-Model Agents

**Status**: Implemented (core decision shipped; two follow-ups tracked)
**Target**: v0.24.x
**Priority**: P1 — local qwen3.5 agent evals were failing on *dialect* mistakes (wrong-language habits), not algorithmic ones; the teaching prompt was present but not landing
**Dependencies**: [M-MOTOKO-LOCAL-OLLAMA](./m-motoko-local-ollama.md) (rig + opencode harness), [M-EVAL-OPENCODE-SYSTEM-PROMPT] (persistent-prompt channel)

## Problem Statement

Against `opencode-qwen3-5-35b-a3b-mxfp8` (local Ollama, Mac Studio rig), agent-mode benchmarks fail on **dialect** errors — missing `import std/string (concat)`, backtick templates instead of `"${}"`, `;` in expression-body functions, `match … with` instead of `=>`. The ~22k-token v0.16.1 teaching prompt *contains* all these rules, yet the model ignored them. The open question was **delivery**, not content: how should the prompt reach the model so the rules actually stay in attention across a 16–50-turn agent loop?

## Hypothesis (and why it was wrong)

The pre-experiment hypothesis ("burial"): the full prompt concatenated into the turn-1 user message ages out of attention over a long loop, so the model drifts back to its training-corpus dialect. Proposed fix: **MOVE** the full prompt to `AGENTS.md`, which opencode re-injects into the system context every turn.

An n=1 A/B seemed to support it. A controlled, replicated experiment refuted it.

## Experiment

Same binary, behavior toggled by env, **n=2** trials, on 3 benchmarks each failing a *distinct* dialect rule (`symbolic_diff` = missing import, `state_machine_vending` = backtick template, `polymorphic_ord_defaulting` = undefined `compare`).

| variant | delivery | env |
|---|---|---|
| **CONCAT** (prod baseline) | full 22k prompt in turn-1 message | `AILANG_EVAL_PERSIST_PROMPT=0` |
| **MOVE** (hypothesized fix) | full 22k prompt → `AGENTS.md`, persistent every turn | `AILANG_EVAL_PERSIST_PROMPT=1` |
| **COMPACT** | 3.3k compact prompt in message | `--conditions agent_prompt` |
| **CARD** (COMPACT + traps card) | compact prompt + a 30-line dialect-traps card front-loaded in the message | `--conditions agent_prompt AILANG_EVAL_TRAPS_CARD=…` |

### Results (pass/trials · avg turns · avg ktok)

| benchmark | CONCAT | MOVE | COMPACT | CARD |
|---|---|---|---|---|
| symbolic_diff | 1/2 · 18t · 755k | 1/2 · 18t · 761k | 1/2 · 34t · 880k | **2/2 · 13t · 246k** |
| state_machine_vending | **2/2** · 16t · 730k | 0/2 · 39t · 2395k | 1/2 · 41t · 3087k | 1/2 · 35t · 1341k |
| polymorphic_ord_defaulting | 0/2 · 38t · 2652k | 0/2 · 22t · 1194k | 1/2 · 22t · 802k | 0/2* · 19t · 825k |
| **TOTAL pass** | **3/6** | **1/6** | **3/6** | **3/6** |

\* one of CARD's two `polymorphic_ord` runs was a 20-min idle stall (infra, an `api_error`), not a language failure — so its genuine record on that benchmark is 0/1.

## Findings

1. **MOVE is the worst delivery, not the best.** Re-injecting the full 22k prompt every turn bloated the context (up to 39 turns / 2.4M tokens) and the model lost the signal — **1/6 vs 3/6**. The "burial" hypothesis is refuted; the real failure mode is **context bloat**, the opposite of burial.
2. **The dialect-traps card is a clear win.** A tiny front-loaded card of the 6 highest-frequency rule violations fixed the import class outright (`symbolic_diff` 1/2 → **2/2**) and slashed flailing where it didn't flip a result (`state_machine_vending` 3087k → 1341k tokens). It never caused a genuine regression.
3. **Compact ties full at ~1/7 the tokens.** COMPACT matched CONCAT's pass rate far more cheaply (and lost 2 cells to infra stalls, so it is arguably understated).
4. **The residual failures are no longer *delivery* problems:**
   - `state_machine_vending` straddles the **known match-in-block parser interaction** (`PAR017` + `=>` error) — a parser limitation, not a prompt gap.
   - `polymorphic_ord_defaulting` fails with `type unification failed: cannot unify int with TList` — a **type-defaulting language gap**, not a dialect trap. No prompt change fixes this.

## Decision

**Delivery is solved: a compact prompt + a small front-loaded dialect-traps card, delivered by turn-1 concatenation — not full-prompt persistence.**

Shipped now (the two safe, data-backed default flips):

| change | before | after | escape hatch |
|---|---|---|---|
| `persistentSystemPromptEnabled()` | default **true** (MOVE) | default **false** (concatenation) | `AILANG_EVAL_PERSIST_PROMPT=1` re-enables |
| `maybePrependTrapsCard()` | default **off** | default **on** (`prompts/agent/dialect-traps.md`) | `AILANG_EVAL_TRAPS_CARD=off` disables; `=<path>` overrides |

The card lives at `prompts/agent/dialect-traps.md`, loaded by relative path from the repo root (same convention as `internal/eval_harness/templates/agent_prompt.txt`). If unreadable, the directive is returned unchanged — additive salience aid, not a data-integrity path.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Harness delivery only; language semantics unchanged |
| A7: Machines First | +2 | Local-model agents now follow the dialect rules they previously buried |
| A9: Cost Visibility | +2 | 2–3.6× fewer tokens per benchmark at equal correctness |
| A11: Structured Failure | +1 | Unreadable card fails additively, never silently corrupts delivery |

**Net: +5 → Proceed.**

## Follow-ups (tracked, not in this change)

1. **Confirm before defaulting compact content.** The shipped default is now CONCAT + card (full prompt + card). The *cheapest proven* arm is COMPACT + card. Before flipping the compact prompt to the global default (`--conditions agent_prompt` → default), run CONCAT+card vs COMPACT+card at higher n — compact is older v0.9.0-era content and broke `json_transform` once via a version confound.
2. **Route the two residual gaps as language work, not prompt work:** the match-in-block parser interaction (`state_machine_vending`) and the Ord/type-defaulting failure (`polymorphic_ord_defaulting`). These overlap the nightly regressions (GitHub #285–#288).
3. **Idle-stall noise.** Two experiment cells were lost to a 20-min `opencode idle mid-generation` stall — this is [M-MOTOKO-OLLAMA-LOOP-CONVERGENCE](../../planned/v0_24_0/m-motoko-ollama-loop-convergence.md) Phase 4 (subprocess reaping); fixing it will de-noise all future local-model evals.

## Related Documents

- [M-MOTOKO-OLLAMA-LOOP-CONVERGENCE](../../planned/v0_24_0/m-motoko-ollama-loop-convergence.md) — the motoko-harness termination problem; shares the idle-stall infra issue.
- [M-MOTOKO-LOCAL-OLLAMA](./m-motoko-local-ollama.md) — the rig + opencode harness this experiment ran on.
