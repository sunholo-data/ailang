# M-EVAL-FMT-WEAKMODEL-AB: measure whether the fmt hook helps a weak model author correct AILANG

**Status**: PARKED — needs-human-review (quorum green-light) — created iter-71 (mission 2026-07-21).
Mark 2026-07-20 — "fmt should be a real help for weaker models creating AILANG… can we do a test
with a weak model to see if its making a difference?"; #422 directive: "test it's used by small
model such as haiku". Quorum (gemini-3-1-pro): round-1 reject (harness-integration surface conflated
with language-surface NONE) → designer revision resolved it; round-2 reject (missing verification
log) → controller added the Verification log below (all paths repo-confirmed). Automated 1-revision
budget exhausted → parked for a human green-light / re-quorum before routing to sprint-planner. Do
NOT execute (also GPU-rig-gated) until unparked.
**Target**: v0.31.x — eval infrastructure, no language surface
**Priority**: P2
**Estimated**: ~0.5d + eval rig time
**Dependencies**: `m-fmt-properties-printer-roundtrip` + `m-ailang-fmt-inline-interior` (BOTH
LANDED 2026-07-20); LANDED `format_ail.sh` opt-in PostToolUse hook from
`m-ailang-fmt-adoption`; agent-mode eval harness; GPU `rig.lock`
**Author**: codex:gpt-5.6-sol rotation designer, 2026-07-21

---

## Why (the data)

Mark's question is direct: "fmt should be a real help for weaker models creating AILANG… can we
do a test with a weak model to see if its making a difference?" His #422 directive narrows the
subject: "test it's used by small model such as haiku". The hook is already LANDED, but adoption
is not evidence: a weak-model agent may improve because canonical formatting removes syntax drift,
or the hook may refuse/no-op often enough to teach nothing. The mission's noisy-agentic-metrics
discipline forbids interpreting a single agent run; this needs matched N-run aggregates.

## Hypothesis

With **haiku (`claude-haiku-4-5`)** as the primary weak model, canonical formatting after every
edit reduces syntax drift, which reduces compile-stuck spirals and therefore increases pass rate
and speeds convergence to stable green.

**Null:** hook ON and hook OFF have no meaningful difference on matched benchmarks. The hypothesis
is refuted if the preregistered comparison shows no significant positive delta, convergence does
not improve, or `fmt` refuses on most edited files so the hook rarely produces canonical output.
Register the hypothesis, benchmarks, N, exclusions, and analysis thresholds BEFORE execution so a
null result is publishable rather than buried.

## Experiment design

1. **Primary model:** haiku (`claude-haiku-4-5`) only; subscription/cheap, with no metered
   frontier spend. An Ollama small model such as Qwen MAY be added as an **OPTIONAL replication
   arm**; it is not required for acceptance and requires the GPU rig.
2. **Matched A/B:** run the SAME benchmark set with the SAME **N ≥ 5 runs per arm per benchmark**:
   **ON** = LANDED `format_ail.sh` opt-in PostToolUse hook automatically runs `ailang fmt --write`
   after every edit; **OFF** = identical agent-mode setup with that hook disabled. Keep prompts,
   model settings, timeouts, tools, and benchmark versions fixed.
3. **Finished-tool gate:** execute only after `m-fmt-properties-printer-roundtrip` and
   `m-ailang-fmt-inline-interior` are confirmed LANDED (2026-07-20). Test the finished formatter,
   not an interim implementation.
4. **Execution gate:** the eval-execution milestone acquires `rig.lock` around the eval step only,
   per the mission GPU rule. This applies when running the agent-mode suite; any OPTIONAL local
   model arm additionally needs the GPU protected by the same mutex. Release the lock before
   analysis/write-up.
5. **Reporting:** aggregate all N runs, preserve per-run traces, and report variance/confidence
   intervals rather than point estimates or selected anecdotes.

## Harness integration surface

This experiment has **no language-surface change**, but it does integrate with existing evaluation
machinery. The implementation approach is to **extend the existing agent-mode harness with a
run-scoped config flag / hook-settings toggle**, not create a parallel runner. Both arms must execute
the byte-identical harness code path; the only config difference is whether the LANDED
`format_ail.sh` PostToolUse hook is present. A separate A/B script is explicitly rejected because
different orchestration, environment setup, result capture, or grading paths would introduce
confounds between arms.

1. **Agent-mode eval harness — EXTEND, do not fork:** reuse `internal/eval_harness/`, the selected
   benchmark definitions under `benchmarks/`, and the existing `claude-haiku-4-5` agent-mode model
   entry in `internal/eval_harness/models.yml` (`agent_cli: "claude"`, agent model `haiku`). Add the
   ON/OFF selection as run configuration in this existing path so prompts, workspace setup, model
   settings, timeouts, tools, grading, and result banking remain identical.
2. **PostToolUse hook toggle — EXTEND by configuration, do not fork:** the treatment is only the
   agent run's hook/settings configuration: ON includes the LANDED opt-in
   `scripts/hooks/format_ail.sh` PostToolUse hook; OFF omits that hook while retaining every other
   setting and hook. Record and review the generated/resolved config diff before scored runs.
3. **Result aggregation and banking — REUSE, do not fork:** consume the normal banked per-run eval
   results and the existing N-run aggregate / best-of-N tooling, including the native rotation
   summary path in `internal/eval_harness/` and `tools/eval_best_of_n.py`-class analysis. Extend the
   reported fields only if the preregistered convergence or hook-delivery metrics are not already
   emitted; do not create a separate aggregator, because one result pipeline is required for
   comparable pass-rate and convergence evidence.

## Metrics

1. **Pass-rate delta:** benchmark and overall pass rate, reported as **ON − OFF**, with N,
   variance, and confidence intervals.
2. **Convergence:** edits to first green and green-stability rate after first green, using the
   DOCX compile-stuck/green-stability work's definitions; also report compile-stuck incidence.
3. **Hook reality:** per-turn `fmt` invocation and exit code, separating exit 0 from refusal/error.
   Compare against the approximately **8% fail-closed refusal** observed in
   `m-ailang-fmt-phase2` / inline-interior work. A hook that no-ops or refuses teaches nothing.

## Milestones

1. **M1 — preregistration:** freeze benchmarks, N, seeds/settings, exclusions, metric definitions,
   confidence method, and refutation threshold before any scored run.
2. **M2 — matched execution:** acquire `rig.lock` for the eval step only; run haiku ON/OFF at the
   same N on every benchmark; optionally run the local replication arm under the same protocol.
3. **M3 — analysis:** publish aggregate deltas, variance/CIs, convergence traces, and per-turn
   formatter exit-code coverage, including the null result if observed.
4. **M4 — verdict:** state whether the hook helps, is neutral, harms, or is unevaluable because
   formatter refusal prevented the treatment from being delivered.

## Risks & guardrails

- **No single-run claims:** agentic pass/convergence outcomes are noisy; N-run aggregates are the
  unit of evidence, NEVER individual runs.
- **Treatment integrity:** record every PostToolUse invocation and exit code; do not classify the
  ON arm as treated when `fmt` did not run or did not exit 0.
- **Sequencing:** both fmt polish docs must be LANDED before execution; do not measure the interim
  formatter and generalize to the finished tool.
- **GPU mutex:** hold `rig.lock` around eval execution only; the OPTIONAL local arm requires it,
  and analysis/documentation must not retain it.
- **Cost:** subscription/cheap haiku plus rig time; no metered frontier spend.
- **Language surface: NONE:** no parser, types, codegen, runtime, syntax, or other language change.
- **Harness integration surface: YES:** this sprint extends the existing agent-mode eval harness,
  hook/settings configuration, result banking, and aggregate analysis path as specified above; the
  mission Conflict-Surface gate therefore applies to those integration points.

## Acceptance criteria

- Preregistration exists before scored runs and fixes the matched benchmark set, SAME N with
  **N ≥ 5 per arm per benchmark**, metrics, exclusions, and statistical reporting method.
- Haiku ON/OFF runs complete under identical conditions except for the LANDED PostToolUse hook.
- Both arms share the same harness code path and differ ONLY in the PostToolUse hook toggle, verified
  by config diff; aggregation reuses the existing eval banking and N-run aggregate tooling.
- Results report pass-rate delta, compile-stuck/green-stability convergence, and per-turn `fmt`
  exit codes with variance/confidence intervals.
- The write-up publishes positive, negative, or null evidence and distinguishes formatter benefit
  from a treatment-delivery failure caused by refusals/no-ops.
- `rig.lock` acquisition/release is recorded for the eval-execution step; any OPTIONAL local
  replication is clearly separated from the primary haiku result.

## Out of scope

- Formatter implementation or polish beyond the already-LANDED dependency pair.
- Parser, type-system, codegen, runtime, syntax, or other language-surface changes.
- A separate A/B runner or result-aggregation script; the existing agent-mode harness and aggregate
  eval tooling are extended/reused to avoid cross-arm confounds.
- Frontier-model comparisons, metered API spend, or general model-ranking claims.
- Making the hook mandatory or changing adoption policy based on a single benchmark or run.

## Verification log

Referenced paths/configs confirmed present in the tree at design time (controller check, 2026-07-21,
HEAD `553a0032b`):

| Claim | Method | Result |
|---|---|---|
| `scripts/hooks/format_ail.sh` (the LANDED PostToolUse fmt hook) exists | `ls scripts/hooks/format_ail.sh` | Confirmed |
| `tools/eval_best_of_n.py`-class N-run aggregate tooling exists | `ls tools/eval_best_of_n.py` | Confirmed |
| `internal/eval_harness/` agent-mode harness exists | `ls -d internal/eval_harness` | Confirmed |
| A `haiku` / `claude-haiku-4-5` agent-mode model entry exists | `grep haiku internal/eval_harness/models.yml` | Confirmed |
| fmt polish dependency pair LANDED | mission log iters 69 (`942931816`/#424 properties-roundtrip) + 70 (`3c1cec57d`/#434 inline-interior) | Confirmed |

Note: the ~8% fail-closed `fmt` refusal figure (m-ailang-fmt-phase2 / inline-interior) is the treatment-integrity
risk the "Hook reality" metric measures; M1 preregistration re-verifies benchmark set + config diff before scored runs.

---
**Document created**: 2026-07-21 (rotation design; codex:gpt-5.6-sol author, controller-hardened; preregister before execution)

## Lock-scope correction (Mark 2026-07-21: "its blocked 'due to GPU' but its not going to run on GPU?")
The M2B plan over-applied rig.lock to the PRIMARY haiku arms. The charter's GPU rule is a
question, not a pattern: **"does this step touch the GPU?"** Haiku is cloud-billed — it does not.
Corrected: primary arms take NO rig.lock (they must never queue behind local rotations); ONLY the
optional local-Ollama replication arm locks, around its own eval step. Sprint JSON amended.

## Process lesson (retro-grade, from Mark's "why is it so hard to just get a demo")
This experiment went design→quorum→park→greenlight→plan→integrity-fix across 3 iterations before
any model ever ran with the hook on. The rigor is right for the HEADLINE numbers — but the miss
was not running a 20-minute SMOKE DEMO first (tiny N, no freezing, no lock) to show the effect
exists and de-risk the wiring. Rule of thumb going forward: **experiment-class items ship a cheap
smoke demo in the SAME iteration the flag lands; full-rigor A/B follows.** (An interactive smoke
demo was run 2026-07-21 evening; results recorded below when banked.)
