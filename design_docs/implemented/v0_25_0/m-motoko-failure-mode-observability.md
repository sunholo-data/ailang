# M-MOTOKO-FAILURE-MODE-OBSERVABILITY — segment agent failures by mode (disengage vs grind-wrong)

**Status**: Implemented (this run) — observability tool + gap-location finding.
**Target**: v0.25.0
**Priority**: P1 (unblocks the mission — locates the motoko↔pi gap precisely, non-GPU)
**Dependencies**: None (reads existing `eval_results/**/agent/*.json`; `agent_tool_calls` already logged)

## Problem Statement

The motoko mission has repeatedly stalled on "we don't know WHERE motoko loses to pi and by what
MODE." Existing eval tooling (`ailang eval-matrix --by-harness`, `analyze_failures.sh`) groups by
`error_category` only — it does NOT distinguish the two qualitatively different agent failure
modes the mission cares about:
- **DISENGAGE** — the run fails having made ≤2 tool calls (the model answered in prose / inspected
  once and stopped; no real solution attempt). This is motoko's signature failure.
- **GRIND-WRONG** — the run fails after ≥3 tool calls (the model engaged, iterated, but produced
  an incorrect solution). A correctness problem, not an engagement problem.

Without this split, fixes were aimed at the wrong thing (e.g. the M-MOTOKO-AGENT-SYSTEM-PROMPT A/B
was run on the core tier where motoko is already 75%, and the M-MOTOKO-PERSIST-NUDGE was a
loop-level band-aid). We need to see, at rotation scale, how much of the gap is each mode.

**Impact:** every future fix should target the dominant mode; this is the cheap, non-GPU
prerequisite the analysis log called for (2026-06-18).

## Goals
**Primary:** a reusable, tested classifier + aggregator that reports pass / disengage / grind-wrong
per harness (and per benchmark) from existing result JSONs, so the gap is located before any GPU.

**Success metrics:**
- Tool reproduces the segmentation on the live rotation, with a passing `--self-test`.
- Produces a recorded analysis-log entry locating the gap by mode.

## Finding (this run, fresh `eval_results/rotation/os-rolling`, qwen3.6, AILANG agent)

| harness | N | pass | **disengage** | grind-wrong |
|---|---|---|---|---|
| **motoko** | 117 | 69% | **29%** | 1% |
| **pi** | 113 | 95% | **3%** | 0% |
| opencode | 114 | 80% | 18% | 1% |

**The motoko↔pi gap is almost entirely DISENGAGEMENT** (motoko 29% vs pi 3% ≈ 26pp). Correctness
(grind-wrong) is ~1% for both — NOT the gap. This confirms the disengagement diagnosis at full
rotation scale (not the biased 6-flaky subset) and tells every future cycle what to attack.

## Solution Design
**Tool:** `tools/eval_failure_modes.py` — reads a results dir (default
`eval_results/rotation/os-rolling/agent`), classifies each agent run via a pure
`classify(stdout_ok, agent_tool_calls)` function (pass | disengage | grind_wrong), and prints a
per-harness (and optional `--by-benchmark`) table. Filters: `--lang`, `--model-substr`,
`--disengage-threshold` (default 2). `--self-test` runs assertions on synthetic inputs and exits.

**Classifier (the unit under test):**
- `stdout_ok` → `pass`
- else `agent_tool_calls <= threshold` → `disengage`
- else → `grind_wrong`

## Files to Modify/Create
**New:** `tools/eval_failure_modes.py` (+ embedded `--self-test`).

## Success Criteria
- [x] `--self-test` passes (classifier correctness).
- [x] Reproduces the rotation segmentation (motoko 29% disengage vs pi 3%).
- [x] Analysis-log entry recorded; mission backlog updated to "gap = disengagement".

## Non-Goals
- NOT fixing disengagement (that's the next, GPU-bound cycle — now precisely targeted).
- NOT a Go subcommand (a tools/ script is the conservative, idiomatic home alongside
  `fair_comparison.py`); promote to `ailang eval-*` later if it earns its keep.

## Deferred Decisions
- Exact disengage threshold (default 2; exposed as a flag for sensitivity checks).
