# M-EVAL-FMT-WEAKMODEL-AB — M5 (hard-set re-run) preregistration

**Status**: PREREGISTERED — frozen before any M5 run. Follow-up to the CLOSED M4 verdict
([`…-M4-verdict.md`](../../implemented/v0_31_0/m-eval-fmt-weakmodel-ab-M4-verdict.md)).
**Date frozen**: 2026-07-23.
**Author**: Claude Opus 4.8 (with Mark).

## Why re-run

M4 returned **NEUTRAL / true-null**, but the mechanistic reading identified the cause: the frozen
easy→medium set left `claude-haiku-4-5` **near-ceiling (58/60)**, so the fmt hook — whose mechanism is
"canonical formatting removes syntax drift → fewer compile-stuck spirals" — had **no drift to fix**.
"A hook that fixes drift cannot show a benefit where there is no drift to fix." (M4 verdict.)

M5 tests the SAME treatment in the regime where the mechanism has headroom: benchmarks where haiku
**produces code but spirals sometimes** (the drift zone), measured from banked agent runs.

This does NOT test the on-device (qwen) thesis — the fmt hook is Claude-Code-only (writes
`.claude/settings.json` + `--settings`; opencode/pi/motoko never load it). A local-model A/B needs
per-harness fmt delivery built first (opencode has a `postToolUse` plugin path via the microRAG
precedent; motoko needs a loop step). M5 is the cheap validity check that de-risks that build: if fmt
doesn't help even a Claude model WHEN DRIFT IS PRESENT, local delivery isn't worth building.

## Frozen benchmark set (the drift zone: haiku 17–73% agent pass, not 0, not 100)

Selected from banked `claude-haiku-4-5` agent-mode pass rates:

| benchmark | prior haiku agent pass |
|---|---|
| csv_to_json_converter | 17% |
| log_file_analyzer | 50% |
| json_transform | 50% |
| graph_bfs | 60% |
| symbolic_diff | 60% |
| run_length_encode | 60% |
| api_call_json | 60% |
| deterministic_list_transform | 62% |
| cli_args | 71% |
| exhaustive_pattern_matching | 73% |

Excluded: haiku-0% benchmarks (config_file_parser, quine, pipeline, commonmark_emphasis,
multi_module_imports) — no code produced for fmt to act on; and haiku-100% benchmarks — no drift.

## Design

- **Model**: `claude-haiku-4-5` (`agent_cli: claude`, subscription OAuth, `ANTHROPIC_API_KEY` unset). Cloud — NO rig contention with the local rotation.
- **Arms**: 2 — `--fmt-hook on` and `--fmt-hook off`. **The ONLY difference is that flag.**
- **Trials**: 5 per (benchmark, arm) → 10 × 5 × 2 = 100 runs.
- **Banking**: `--output eval_results/fmt_ab_haiku_hard/{on,off}`, isolated dirs, run sequentially.

## Primary metric + decision rule (frozen)

- **Primary**: pass-rate delta `ON − OFF` over the 10-benchmark set, Newcombe 95% CI.
- **Secondary**: compile-stuck rate (`compile_ok == false`) delta, and mean `.ail` edits/run.
- **Treatment-integrity gate (void clause)**: the ON arm MUST show `fmt_hook_events > 0` (hook fired).
  If the hook does not deliver, the run is **void/unevaluable**, not a null — same discipline as M4.
- **Verdict**:
  - **helps** — delta CI excludes 0 AND point ≥ **+0.10**
  - **harms** — delta CI excludes 0 AND point ≤ **−0.10**, or any control regression
  - **neutral (NULL)** — CI includes 0 or |point| < 0.10, with treatment integrity proven
