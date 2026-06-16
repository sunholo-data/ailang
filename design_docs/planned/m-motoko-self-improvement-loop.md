# M-MOTOKO-SELF-IMPROVEMENT: AILANG-Native Harness Self-Improvement Loop via Extension Packages

**Status**: Planned (initiative / living doc)
**Target**: rolling — multi-release initiative, not a single sprint
**Priority**: P1 — this is motoko's reason to exist (see [[motoko-strategic-goal]])
**Owner**: sunholo (we co-develop motoko with @arniwesth)
**Depends on**: motoko ollama integration working ([m-motoko-ollama-loop-convergence](implemented/../planned/v0_24_0/m-motoko-ollama-loop-convergence.md) — RESOLVED 2026-06-15, AILANG-side; upstream note: arniwesth/motoko_agent#44)

## Thesis

motoko is an **AILANG-native** AI coding harness — its agent loop is written in
`.ail` and it composes capabilities as **sunholo `motoko_ext_*` AILANG packages**.
The bet: a harness that *understands AILANG* can scaffold a model better on AILANG
tasks than a language-agnostic harness (pi, opencode) can. This doc defines the
**measure → hypothesize → build-as-package → A/B → promote** loop that turns that
bet into a falsifiable, repeatable program — and the flywheel where motoko
eventually helps improve the very AILANG packages that make motoko better.

**Win condition:** motoko > pi/opencode at **equal model, equal benchmarks, equal
($0) cost**. A passing motoko eval is necessary but not the win.

## Why now

- The ollama integration is fixed (AILANG-side), so for the first time all three
  harnesses run the **same local model** (`qwen3.6:35b-a3b-mxfp8`) at $0 on the
  M4 Max rig — a clean, harness-controlled comparison (`--parallel 1`,
  [[ailang-rig-parallel-one]]).
- The extension system (`motoko_ext_ailang_docs`, `motoko_ext_microrag`,
  `motoko_ext_compaction_ai`, …) is exactly the right surface to add
  AILANG-specific intelligence *as composable, versioned packages* we own.

## Two measurement axes

1. **Cross-harness (the KPI):** motoko vs pi vs opencode — same model, same
   benchmarks, $0. Isolates *harness* lift. Feeds the Agent Explorer
   cross-harness view.
2. **Intra-motoko (the tuning knob):** `motoko_profile` = which `motoko_ext_*`
   packages load. Lean `ollama` profile is the baseline; richer profiles
   (`ollama_docs` +ailang_docs, `ollama_microrag` +ailang_docs+microrag) test
   whether AILANG-aware tooling closes the gap. Isolates *capability* lift.

## The loop

```
   ┌─ 1. MEASURE ──────────────────────────────────────────────┐
   │  Harness-controlled A/B (motoko vs pi vs opencode) +       │
   │  profile matrix (lean vs +ext). Same model, $0, p=1.       │
   └───────────────┬───────────────────────────────────────────┘
                   ▼
   2. DIAGNOSE   Where does motoko lose? (wrong AILANG syntax, no
                 docs/RAG, weak verification, malformed tool calls…)
                   ▼
   3. BUILD      Add/refine the capability as a sunholo motoko_ext_*
                 AILANG package (or a core context_usage/verify change).
                   ▼
   4. A/B        Profile matrix: lean vs +new-ext, isolate the lift.
                 Promote only if it beats baseline at equal cost.
                   ▼
   5. PROMOTE    Fold winning ext into the default ollama profile;
                 publish the package; bump motoko ailang.lock.
                   ▼
   6. SCHEDULE   Once a config wins, add it to the weekly local A/B
                 (mirrors the microRAG weekly A/B cadence).
   └──────────────── feeds back into 1 ────────────────────────┘
```

### The flywheel ("self-improvement via AILANG packages")

The packages in steps 3–5 are AILANG code. motoko is an AILANG coding agent.
The end-state is **motoko improving its own `motoko_ext_*` packages** — using the
harness to write/refine the AILANG that makes the harness better. Gated by the
same A/B (no self-edit ships without beating baseline). This is the long-horizon
payoff and the reason the loop is package-shaped, not patch-shaped.

## Candidate capabilities (hypotheses to A/B)

| Capability | Package / layer | Hypothesis |
|---|---|---|
| AILANG docs in-context | `motoko_ext_ailang_docs` | model writes valid AILANG first-try (was the root of bare-directive failures) |
| μRAG over syntax/builtins/examples | `motoko_ext_microrag` | retrieval beats static docs on unfamiliar constructs |
| Malformed-tool-call tolerance | core loop / parser | recover the qwen `<function>/<parameter>` XML failures (1/5 loss today; motoko_agent#44) |
| **DP7 pre-finalize type-check gate** | core `agent_loop_v2.ail` `dp7_gate` (set `verification.{enabled,command}`) | **highest-priority arm.** Forces the model to pass `ailang check` before `done`; rejects + feeds errors back. AILANG-native advantage pi/opencode lack. Currently OFF in ALL profiles (`verification: {}`, opt-in since a78e4db) — enabling is likely a direct pass-rate lift |
| Semi-formal (Z3 contract) verification | `semi_formal_verifier_mode` (budget-split verifier, rpc.ail) | deeper than DP7; also OFF in all profiles. Carves 1/4 step budget for a verifier pass |
| Local compaction | core `context_usage.ail` + `compaction_ai.json` | long tasks stay $0 + coherent (register `ollama/` context limit) |

## Metrics

- **Primary:** per-harness pass rate at equal model/benchmarks/$0; motoko − max(pi, opencode).
- **Secondary:** turns-to-pass, tool-call validity rate, $ (must stay 0 for local).
- **Guardrail:** no profile that pulls in network/paid extensions (exa_search,
  omnigraph, openrouter compaction) counts as a "$0 local" result.

## Axiom compliance (brief)

- **A7 Machines First +2** — a self-improving AILANG-native harness is the
  sharpest "machines first" artifact we have.
- **A9 Cost Visibility +1** — entire loop runs at $0 local; cost is a hard guardrail.
- **A2 Replayability +1** — A/B is seed-pinned, p=1, deterministic harness.
- Net: Proceed.

## Open questions

- Does AILANG-doc/μRAG tooling actually lift a *local* model, or is the gap
  model-capability-bound (qwen3.6) regardless of scaffolding? (The current matrix
  run is the first data point.)
- Right cadence/cost ceiling for the eventual scheduled A/B.
- When is motoko good enough to safely edit its own packages (flywheel gate)?

## Status / first data (2026-06-16)

**Full core-tier matrix complete** — qwen3.6 × {lean, +docs, +microrag, +dp7} × 26
core benchmarks, single-trial, $0, 2h41m (chain 911b10b1). Full breakdown in
`eval_results/motoko_full_core_matrix/ANALYSIS.md`.

| profile | core pass | total turns |
|---|---|---|
| **lean** | **23/26 (88.5%)** | 221 |
| microrag | 22/26 | 222 |
| dp7 | 22/26 | 184 |
| docs | 21/26 (80.8%) | **172 (−22%)** |

Key findings (sobering but useful):
1. **No profile beats lean on pass rate** at single trial — spreads are within noise.
   Richer tooling did NOT lift pass rate on hard benchmarks.
2. **`ailang_docs` is a turn-efficiency win (~−22%), not a pass-rate win** — consistent
   across easy + hard sets. The model converges faster but doesn't pass more.
3. **DP7 effect negligible/unconfirmed** — fewer turns than lean (rarely rejecting;
   most qwen code passes `ailang check` first try). Needs a broken-solution probe.
4. **Single-trial is noise-dominated** (7/26 benchmarks flip between profiles). →
   **A/B framework MUST use N≥3 trials per cell.** This is the headline methodological
   input for the loop's "MEASURE" step.
5. `contract_roman_numeral` fails on all 4 → model-bound; the frontier subset
   (csv_to_json, json_transform, merge_sort, graph_bfs, config_file_parser,
   prompt_injection, ast_patch_roundtrip) is where qwen3.6 is unreliable — focus here.

**Baseline set:** motoko lean = **88.5% core** on qwen3.6 — the number for the 3-way
harness comparison (motoko vs pi vs opencode) to beat. That comparison is the next KPI.

## References

- Strategic goal memory: [[motoko-strategic-goal]]; integration memory: [[motoko-ollama-is-ailang-side]]
- Upstream note: https://github.com/arniwesth/motoko_agent/issues/44
- Resolved integration doc: design_docs/planned/v0_24_0/m-motoko-ollama-loop-convergence.md
