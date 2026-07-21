# Sprint Plan: M-EVAL-FMT-WEAKMODEL-AB

**Design doc**: [`design_docs/planned/v0_31_0/m-eval-fmt-weakmodel-ab.md`](m-eval-fmt-weakmodel-ab.md)
**Sprint ID**: M-EVAL-FMT-WEAKMODEL-AB
**Created**: 2026-07-21 (mission iter-72, sprint-planner)
**Status**: PLAN READY (human green-lit; NOT re-quorumed)
**Risk level**: medium (integration point does not yet exist; GPU-gated execution)
**Estimated effort**: ~1.5d engineering (M1 + hook-toggle build in M2) + eval rig time

---

## Summary

Green-lit A/B experiment: does the LANDED `format_ail.sh` PostToolUse fmt hook help a **weak
model** (`claude-haiku-4-5`) author correct AILANG? Pure eval-infrastructure sprint — **NO
language-surface change**. Both arms run the byte-identical agent-mode harness; the only
difference is whether the `ailang fmt --write` PostToolUse hook is wired in.

### KEY INTEGRATION FINDING (grounding the plan)

**A hook ON/OFF toggle does NOT exist in the harness today — it must be built.**

Verified against the tree (HEAD dev, 2026-07-21):

- The active agent-mode path is `internal/eval_harness/agent_runner_streaming.go`
  (`runHeadlessSessionStreaming`). It builds the `claude` command with `cmd.Dir = workspace`
  but passes **no** `--settings` flag and writes **no** `.claude/settings.json` into the
  workspace. Therefore `scripts/hooks/format_ail.sh` is **never wired into an agent-mode run
  today** — not "off by default", but *absent entirely*.
- `AgentBenchmarkConfig` (`agent_runner.go:19`) has no hook/settings field. It is constructed
  in one place for the suite: `cmd/ailang/eval_suite.go:664`.
- **The exact precedent to copy is `MicroragMode`** (`internal/eval_harness/microrag_mode.go`):
  a `type FooMode string` with `Parse*`, `ApplyToEnv`/apply, and `ResolvedState()` methods,
  plumbed as one `AgentBenchmarkConfig` field, set from one CLI flag, applied in BOTH runner
  paths. The fmt-hook toggle should follow this structure exactly (an `on/off` `FmtHookMode`),
  applied by writing/omitting a workspace `.claude/settings.json` that registers the LANDED
  hook and passing `--settings <path>` to the claude command.
- **Treatment-integrity ("hook reality") metric** needs per-turn `fmt` exit codes. The
  stream-json parser already inspects `tool_use` blocks (`agent_runner_streaming.go:170`) but
  does not currently surface PostToolUse hook `additionalContext` / exit codes. The hook script
  emits its status via stderr (`✓ Formatted`) and `hookSpecificOutput.additionalContext` on
  failure — capturing those per turn is a real (small) parsing task, not free.

Conclusion: this is **~1 day of real harness engineering** (a new `FmtHookMode` toggle + workspace
settings emission + hook-reality capture), plus M1 preregistration (cheap), plus GPU-gated
execution and no-GPU analysis/verdict.

---

## GPU-touch map (per milestone)

| Milestone | GPU / rig.lock? | Headless-schedulable when? |
|---|---|---|
| **M1 preregistration** | **NO** | Immediately — cheap, no eval, no GPU. Safe in any headless iteration. |
| **M2 matched execution** | **YES** — acquires `tools/launchd/rig-lock.sh` (`rig_lock_acquire wait`) around the eval step ONLY; releases before M3. Optional local/Ollama replication arm also GPU. | GPU-available iteration only. |
| **M3 analysis** | **NO** — consumes banked results; lock released. | Any headless iteration after M2 banks results. |
| **M4 verdict** | **NO** — write-up only. | Any headless iteration after M3. |

The **hook-toggle build itself (M2a below)** is pure Go/shell engineering and touches **NO
GPU** — it can and should be built + unit-tested in a no-GPU iteration, so that when a GPU slot
opens, M2b execution is a config-only run.

---

## Milestones

### M1 — Preregistration (NO GPU, do first, headless-safe)

Freeze the experiment before any scored run so a null result is publishable.

**Tasks**
- Select and freeze the matched benchmark set (draw from the curated weak-model set; prefer
  benchmarks known to exercise `.ail` edits so the hook actually fires). Record exact IDs +
  versions.
- Fix N ≥ 5 runs per arm per benchmark; fix model = `claude-haiku-4-5`; fix timeouts, budgets
  (haiku `max_cost_usd: 0.30`, `hard_timeout_secs: 600` from models.yml), allowed tools, and
  prompt/system-prompt version.
- Define metrics precisely: pass-rate delta (ON − OFF) with N/variance/CI; convergence (edits
  to first green, green-stability rate) using the DOCX compile-stuck/green-stability
  definitions; hook-reality (per-turn `fmt` exit-0 vs refusal/error; compare to the ~8%
  fail-closed refusal baseline).
- Fix the confidence method and the refutation threshold (what delta counts as "no meaningful
  difference").
- Write the preregistration into the design doc (or a sibling `prereg` section) and commit.

**Acceptance**: preregistration committed, fixing benchmark set, N≥5/arm/bench, metrics,
exclusions, statistical method, and refutation threshold BEFORE any scored run.
**Effort**: ~0.25d. **GPU**: no.

### M2 — Matched execution

Two sub-parts, deliberately separated so the build is GPU-free and only execution is gated.

#### M2a — Build the fmt-hook toggle (NO GPU, headless-safe)

**Tasks**
- Add `FmtHookMode` (`on`/`off`) modeled on `internal/eval_harness/microrag_mode.go`
  (type + `ParseFmtHookMode` + apply + `ResolvedState()`), with a unit test mirroring
  `microrag_mode_test.go`.
- Add a `FmtHook FmtHookMode` field to `AgentBenchmarkConfig`; wire it in
  `cmd/ailang/eval_suite.go:664` from a new CLI flag (e.g. `-fmt-hook on|off`, default `off`
  to preserve current behaviour). Follow the `-microrag` flag precedent.
- In workspace prep, when `FmtHook == on`, emit a `.claude/settings.json` into the agent
  workspace registering the LANDED `scripts/hooks/format_ail.sh` as a PostToolUse hook for
  Edit/Write, and pass `--settings <workspace>/.claude/settings.json` to the claude command in
  BOTH `agent_runner_streaming.go` (active) and `agent_runner.go` (legacy). When `off`, emit
  nothing / pass no settings. This is the ONLY per-arm difference.
- Record the resolved config into the banked result (a `fmt_hook: on|off` field) and log the
  resolved settings so the required **config diff** can be reviewed.
- Capture per-turn hook reality: extend the stream-json handling to record `fmt` invocation +
  exit status (from hook stderr / `additionalContext`) per turn, banked for M3's hook-reality
  metric.

**Acceptance**: `-fmt-hook on` writes the workspace settings + `--settings` flag and the hook
fires on `.ail` edits; `-fmt-hook off` is byte-identical to today's path; config diff between
arms shows ONLY the hook; unit tests green; `make test` + `make lint` + `make check-boundaries`
pass.
**Effort**: ~0.75d–1d. **GPU**: no.

#### M2b — Run the matched A/B (GPU — rig.lock gated)

**Tasks**
- Acquire `tools/launchd/rig-lock.sh` (`rig_lock_acquire wait`) around the eval step ONLY.
- Run `claude-haiku-4-5` ON and OFF at the frozen N on every frozen benchmark, via the existing
  `ailang eval-suite` agent path (`--agent`, `-fmt-hook on` / `-fmt-hook off`), banking per
  run. Keep everything else fixed.
- OPTIONAL replication arm (local Ollama small model, e.g. Qwen) under the SAME protocol — also
  GPU, also under the same lock; explicitly separated from the primary haiku result. Not
  required for acceptance.
- Release the lock BEFORE analysis.

**Acceptance**: ON/OFF haiku runs complete under identical conditions except the hook; N≥5/arm
per bench banked; `rig.lock` acquire/release recorded for the eval step; any local arm clearly
separated.
**Effort**: rig time (~hours, model/bench-count dependent), minimal engineering. **GPU**: YES.

### M3 — Analysis (NO GPU, headless-safe after M2 banks)

**Tasks**
- Aggregate all N runs via the existing `tools/eval_best_of_n.py`-class N-run tooling and the
  native rotation summary path — REUSE, do not fork.
- Report pass-rate delta (ON − OFF) with variance/CIs; convergence (edits-to-first-green,
  green-stability, compile-stuck incidence); per-turn `fmt` exit-code coverage (exit-0 vs
  refusal/error), compared to the ~8% baseline.
- Include the null result if observed.

**Acceptance**: aggregate deltas + variance/CIs + convergence traces + per-turn formatter
exit-code coverage published; reuses existing banking + aggregate tooling (no separate
aggregator).
**Effort**: ~0.25d. **GPU**: no.

### M4 — Verdict (NO GPU, headless-safe)

**Tasks**
- State whether the hook **helps / is neutral / harms / is unevaluable** (the last if formatter
  refusal/no-op prevented treatment delivery — distinguish benefit from delivery failure).
- Publish positive, negative, or null evidence. Feed the verdict back to the mission bookkeeping
  issue and route follow-up (do NOT change adoption policy off a single benchmark/run).

**Acceptance**: verdict published distinguishing formatter benefit from treatment-delivery
failure; evidence (whatever sign) recorded.
**Effort**: ~0.25d. **GPU**: no.

---

## Execution sequencing for headless mission iterations

1. **Iteration A (any slot, no GPU)**: execute **M1** (preregistration) + **M2a** (build the
   `FmtHookMode` toggle, settings emission, hook-reality capture; unit-tested, lint/boundaries
   green). Both are cheap and GPU-free.
2. **Iteration B (GPU-available slot)**: execute **M2b** under `rig.lock` — config-only run once
   M2a is merged.
3. **Iteration C (any slot, no GPU)**: execute **M3** analysis + **M4** verdict on banked
   results.

This keeps every GPU-touching step (M2b, optional local arm) isolated to a single
GPU-available iteration; all engineering, prereg, analysis, and verdict are safely headless.

---

## Success metrics

- Preregistration committed before any scored run.
- `-fmt-hook on|off` toggle exists, unit-tested; config diff between arms shows ONLY the hook.
- Both arms share the identical harness code path (verified by config diff).
- Results report pass-rate delta, compile-stuck/green-stability convergence, and per-turn `fmt`
  exit codes with variance/CIs.
- Verdict distinguishes formatter benefit from treatment-delivery failure.
- `rig.lock` acquire/release recorded for M2b; optional local arm separated from primary result.

## Risks & guardrails

- **Integration surface is NEW** (biggest risk): the hook toggle does not exist; M2a is real
  engineering, not config. Estimated honestly at ~0.75–1d.
- **No single-run claims**: N-run aggregates are the unit of evidence.
- **Treatment integrity**: do not classify the ON arm as treated when `fmt` did not run / did
  not exit 0 — the hook-reality capture in M2a is what enforces this.
- **GPU mutex**: hold `rig.lock` around M2b execution only; release before M3.
- **Boundaries**: changes live in `internal/eval_harness/` + `cmd/ailang/` (dashboard/apps
  layer for the CLI) — run `make check-boundaries`.
- **Language surface: NONE.**

## Out of scope

- Formatter implementation/polish (dependency pair already LANDED).
- Any parser/type/codegen/runtime/syntax change.
- A separate A/B runner or aggregator (extend/reuse only).
- Frontier-model comparison or metered spend; making the hook mandatory off one run.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_31_0/m-eval-fmt-weakmodel-ab-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-EVAL-FMT-WEAKMODEL-AB.json`
