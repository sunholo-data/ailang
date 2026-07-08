# M-MOTOKO-AGENT-SYSTEM-PROMPT — give motoko a lean agentic system prompt

**Status**: Tested — NULL at scale (do not productionize as-is). See analysis log 2026-06-18.
**Target**: v0.25.0
**Priority**: P0 (mission-critical — the motoko↔pi gap on local AILANG)

> **RESULT (2026-06-18):** Proper A/B (core tier 26, n=2, both arms fresh) = empty **75%** vs
> lean-agentic-system **77%** (+1/52, noise). The 6-flaky smoke (+14pp) did NOT generalize — it
> was a biased hard-case subset. Disengagement IS the dominant failure mode (8/10 failures are
> 0-tool-call), but the lean prompt only partially/noisily fixes it (3 improved, 2 regressions on
> previously-passing benchmarks). Success criteria NOT met → keep the knob, do NOT productionize.
> Next: tool-result/error feedback diff; measure pi-on-core for the true gap.
**Estimated**: 1 day (A/B validate) + a motoko_agent PR to productionize
**Dependencies**: None (delivery knob reuses the existing `writeMotokoSystemPrompt`/`SYSTEM_MD` path)

## Problem Statement

On the local-AILANG **agent** eval (qwen3.6:35b-a3b-mxfp8), motoko trails pi badly:
**~63% vs ~96%** (rotation aggregate, agent mode, AILANG). The dominant motoko failure is
**"1 turn, 0 tool calls"** — the model answers once in prose and never calls a tool, so no
solution is written ("Mode A" disengagement).

**Root cause (source-grounded, 2026-06-18 — NOT inferred):** a 3-way request capture
(motoko/pi/opencode on the same benchmark) + reading pi's loop source show that **all three
receive the SAME AILANG teaching prompt from the eval harness and place it in the USER
message.** The teaching is a *controlled variable* — by construction it cannot explain a
harness difference. The ONLY structural difference is the **system role**:

| harness | system role | AILANG teaching |
|---|---|---|
| pi | lean ~2.5 KB agentic coding prompt ("use tools, be concise") | user message |
| opencode | ~10 KB agentic system prompt | user message |
| **motoko** | **EMPTY** | user message |

motoko enters every benchmark with **no agentic framing**, so qwen3.6 defaults to "answer the
question" (prose, 0 tool calls) instead of "act as a tool-using agent."

**Why motoko's system role is empty:** `internal/executor/motoko/motoko.go` folds
`task.SystemPrompt` (the teaching) into the user directive and sets no system prompt. motoko's
own `SYSTEM.md` is (a) a *motoko-self-development* prompt (mandatory `<thinking>` tags, "you are
Motoko the runtime"), not an agentic coding prompt, and (b) rejected in eval anyway —
`systemPromptForWorkspace` (motoko_agent `index.ts`) rejects any `SYSTEM_MD` path outside the
per-task temp workspace.

**Impact:** this is the single largest, cheapest-to-fix lever on the mission's north-star gap.

### Ruled out (record — do NOT repeat these assumption-driven mis-steps)
- **M-MOTOKO-SYSTEM-ROLE** (commit `7a0caf7a`) put the **89 KB TEACHING** into the system role
  (wrong content for the slot). A/B = **−2/18**. It tested "teaching in user vs teaching in
  system", NOT "empty vs lean agentic prompt". Different question; ruled out.
- **M-MOTOKO-PERSIST-NUDGE** (PR arniwesth/motoko_agent#47) force-continued the loop. But pi has
  **NO persistence** — `pi-agent-core/agent-loop.js` stops on empty tool calls exactly like
  motoko. Divergent band-aid (+3/18, kept default-off); NOT how pi wins.

## Goals

**Primary Goal:** give motoko a lean, pi-faithful **agentic system prompt** in the system role
(teaching stays in the user message), eliminating the Mode-A prose disengagement.

**Success Metrics:**
- Proper A/B (core tier, n≥3): motoko **with** lean agentic system prompt materially beats the
  empty-system baseline, **no net regression**.
- Tool-call engagement: median tool calls per run rises from the disengagement floor (~0–2) into
  healthy iteration (≥5) on the Mode-A benchmarks.
- Narrows the gap to pi (96%) on the same set.

## Signal already collected (justifies this doc)
Exploratory smoke (6 flaky ×2, motoko + lean agentic system prompt vs empty baseline 11/18):
**9/12 (75%) vs 61%**, and — the causal tell — **avg tool calls jumped to 6–17** on most
benchmarks. `graph_bfs` (the worst 0-turn disengager) went **1/3 → 2/2 @ 6.5 tool calls**. One
holdout: `config_file_parser` (still 1.5 tool calls / 0 pass).

## Solution Design

### Overview
Deliver a lean agentic coding system prompt (adapted to motoko's tools:
ReadFile/WriteFile/EditFile/BashExec/RunTests/Search; "act through tools, don't answer in prose,
keep iterating until it compiles/runs, be concise") in motoko's **system role**. The AILANG
teaching remains in the user message (unchanged). One variable changes vs baseline.

### Architecture
- **Experiment delivery (AILANG side, this repo):** `AILANG_MOTOKO_AGENT_SYSTEM_FILE=<path>` in
  `motoko.go` — read the file, write its content as the system-role prompt via the existing
  `writeMotokoSystemPrompt` (workspace file + `SYSTEM_MD`, which stock `index.ts` accepts because
  it is inside the workspace). Keep the teaching in the directive. Already implemented + built.
- **Production delivery (motoko_agent PR, if validated):** motoko ships a built-in DEFAULT
  agentic system prompt (like pi does) used whenever no `SYSTEM_MD`/`SYSTEM.md` is provided, so
  motoko is never run with an empty system role. This is the durable fix; the AILANG knob is just
  the A/B harness.

### Implementation Plan
- **Phase 1 — A/B harness (DONE):** `AILANG_MOTOKO_AGENT_SYSTEM_FILE` knob in `motoko.go`;
  `/tmp/motoko-agent-system.md` lean prompt; smoke = signal.
- **Phase 2 — proper A/B:** core tier (26), agent, AILANG, motoko-local-qwen3.6, n=3, arm A =
  empty (baseline), arm B = lean agentic system prompt. Lock-respecting, `--parallel 1`. Compare
  to pi as the bar. Record in the analysis log.
- **Phase 3 — productionize (if B wins):** motoko_agent PR adding a built-in default agentic
  system prompt; promote the prompt text into the repo (not `/tmp`).

### Files to Modify/Create
**Modified (AILANG):** `internal/executor/motoko/motoko.go` (knob — done).
**New:** the lean prompt text (promote from `/tmp` into `tools/motoko/` if validated);
motoko_agent built-in default prompt (separate PR).

## Success Criteria
- [ ] Proper A/B run (core tier, n≥3) recorded with per-benchmark pass + tool-call deltas.
- [ ] Arm B > arm A in aggregate with no net regression → proceed to productionize.
- [ ] If arm B does NOT beat A → record the negative honestly, keep the knob, move to the next
      diff (tool-result/error feedback format).
- [ ] Analysis log + mission backlog updated.

## Non-Goals
- NOT moving the AILANG teaching into the system role (tested −2, ruled out).
- NOT a loop-persistence/force-continue mechanism (pi has none).
- NOT tuning sampling/temperature in this doc.

## Deferred Decisions (agent latitude)
- Exact wording of the lean prompt (within the pi-faithful spirit: tools, no prose, iterate, be
  concise).
- Whether to also test a `config_file_parser`-specific follow-up (the one smoke holdout).
