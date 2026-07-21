# PREREGISTRATION — M-EVAL-FMT-WEAKMODEL-AB

**Status**: FROZEN 2026-07-21 (M1, before any scored run).
**Design doc**: [`m-eval-fmt-weakmodel-ab.md`](m-eval-fmt-weakmodel-ab.md)
**Sprint plan**: [`m-eval-fmt-weakmodel-ab-sprint-plan.md`](m-eval-fmt-weakmodel-ab-sprint-plan.md)
**Frozen against repo SHA**: `2bb1820d65b4e8aee3077ece6b46c5535ef2dcee` (worktree base at M1 time).

This document freezes the experiment BEFORE any scored A/B run so a null result is
publishable rather than buried (design doc "Hypothesis" section). Nothing below may be
changed after the first scored run banks. If a change is unavoidable, it must be recorded
as a dated amendment at the bottom, and the prior-frozen runs re-labelled.

---

## 1. Hypothesis (restated, frozen)

**H1 (directional):** With `claude-haiku-4-5` as the weak model, wiring the LANDED
`scripts/hooks/format_ail.sh` PostToolUse fmt hook (ON) — which runs `ailang fmt --write`
after every Edit/Write on a `.ail` file — reduces syntax drift, which reduces compile-stuck
spirals, which raises pass rate and speeds convergence versus the identical harness with the
hook absent (OFF).

**H0 (null):** ON and OFF have no meaningful difference on the matched benchmark set.

**Refutation of H1** (any one suffices):
- The pass-rate delta (ON − OFF) is not a positive delta beyond the threshold in §6, OR
- convergence (edits-to-first-green / green-stability) does not improve, OR
- the hook did not actually run / did not exit 0 on most edited `.ail` files (treatment
  never delivered → **unevaluable**, distinct from a true null; see §5.3 and the M4 verdict).

---

## 2. Frozen matched benchmark set

Both arms run the SAME set, same versions, pinned to the repo SHA above (benchmark YAMLs
carry no independent `version` field; the repo SHA IS the benchmark version). All six are
`ailang`-supporting, agent-mode benchmarks where the agent authors/edits a `.ail` file in
its workspace — so the PostToolUse Edit|Write hook on `*.ail` can actually fire. The set
deliberately spans easy→hard so a weak model has room to spiral (the regime where canonical
formatting could plausibly help), while staying cheap enough for N≥5 × 2 arms on haiku.

| Benchmark ID              | File                                  | difficulty | tier  | Why chosen |
|---------------------------|---------------------------------------|------------|-------|------------|
| `fizzbuzz`                | `benchmarks/fizzbuzz.yml`             | easy       | smoke | Floor/control: near-ceiling for most models; ON should not REGRESS an easy task (guards against the hook harming). |
| `gcd_lcm`                 | `benchmarks/gcd_lcm.yml`              | medium     | smoke | Recursion + multiple small functions; realistic multi-edit surface for a weak model. |
| `adt_option`             | `benchmarks/adt_option.yml`           | medium     | smoke | ADTs + pattern matching — a known weak-model syntax-drift hotspot (`=>`/`->` confusion per docx dialect notes); canonical fmt most plausibly helps here. |
| `higher_order_functions` | `benchmarks/higher_order_functions.yml` | medium     | core  | Lambdas (`\x. ...`) — another documented drift point; multi-turn editing likely. |
| `json_parse`             | `benchmarks/json_parse.yml`           | medium     | smoke | Larger program, more edits, more chances for the hook to fire per run. |
| `cli_args`               | `benchmarks/cli_args.yml`             | hard       | core  | Hard enough to induce compile-stuck spirals in a weak model — the exact regime H1 targets; seeds input files, exercises effect rows. |

**Exclusions (frozen):**
- No Python/JS/Go arms — the hook only touches `.ail` files; other languages would dilute.
- No frontier/`stretch` benchmarks (e.g. `float_eq`) — saturated for stronger models and off-scope; this experiment is weak-model-specific.
- No `docx_reimplement` / multi-file reimplement benchmarks — too expensive for N≥5 × 2 arms on the cheap-model budget and dominated by non-fmt failure modes.
- Any benchmark that does NOT get the agent to write a `.ail` file (pure-Python-only specs) is excluded by construction.

---

## 3. Frozen model, N, and settings

| Parameter | Frozen value | Source |
|---|---|---|
| Model | `claude-haiku-4-5` (`agent_model_name: haiku`, `agent_cli: claude`) | `internal/eval_harness/models.yml` |
| API model id | `claude-haiku-4-5-20251001` | `models.yml` |
| N (runs per arm per benchmark) | **N = 5** (≥5 required) | design doc §Experiment design |
| Arms | 2: **ON** (`-fmt-hook on`) and **OFF** (`-fmt-hook off`) | this sprint (M2a) |
| Total scored runs | 6 benchmarks × 2 arms × 5 = **60** | — |
| Cost budget | `max_cost_usd: 0.30` per benchmark run | `models.yml` haiku `budgets` |
| Hard timeout | `hard_timeout_secs: 600` (10 min) per run | `models.yml` haiku `budgets` |
| Allowed tools | `Bash, Read, Write, Edit, Grep` (harness default `DefaultAgentConfig`) | `internal/eval_harness/agent_runner.go` |
| Permission mode | `bypassPermissions` (streaming runner default) | `agent_runner_streaming.go` |
| System prompt | The versioned AILANG teaching prompt selected by `GenerateAgentPromptsWithSystemPrompt` at the frozen SHA (record `prompt_version` from each banked result; MUST be identical across both arms) | harness |
| Trials flag | `--trials 5` (native N-trial banking) | `cmd/ailang/eval_suite.go` |
| Seed | fixed `--seed` value recorded at execution; identical for both arms | harness |

**The ONLY difference between ON and OFF is the `-fmt-hook` flag.** All other flags,
prompts, model settings, timeouts, tools, budgets, seed, and benchmark versions are held
byte-identical. This is verified at execution time by diffing the two arms' resolved config
(the banked `fmt_hook` field + logged resolved `.claude/settings.json`), per acceptance
criterion "config diff between arms shows ONLY the hook".

---

## 4. Treatment definition (frozen)

- **ON arm:** the harness writes a `.claude/settings.json` into each agent workspace that
  registers `scripts/hooks/format_ail.sh` as a `PostToolUse` hook matching `Edit|Write`, and
  passes `--settings <workspace>/.claude/settings.json` to the `claude` CLI. After every
  Edit/Write on a `*.ail` file, `ailang fmt --write` canonically formats it.
- **OFF arm:** no `.claude/settings.json` is written and no `--settings` flag is passed —
  byte-identical to today's harness path (the hook is *absent*, exactly as it is in every
  agent run today).
- Banked per run: `fmt_hook: "on" | "off"` so the arm is unambiguous in analysis.

---

## 5. Metric definitions (frozen)

### 5.1 Pass-rate delta (primary)

- **Per-benchmark pass rate** for an arm = (# of the 5 runs that fully pass) / 5, where a
  run "passes" iff `compile_ok && runtime_ok && stdout_ok` (the harness `Success`
  definition, already banked in `RunMetrics`).
- **Overall pass rate** for an arm = pooled passes / pooled runs across all 6 benchmarks
  (30 runs/arm).
- **Delta = pass_rate(ON) − pass_rate(OFF)**, reported per-benchmark AND overall, each with
  N, variance, and a confidence interval (§6).

### 5.2 Convergence (secondary)

Using the DOCX compile-stuck / green-stability definitions already in the codebase and docs
(`project_docx_compile_stuck_green_stability` memory; `analyze_stuck` per-edit `typecheck`
field):
- **Edits-to-first-green:** number of `.ail` Edit/Write operations before the first
  compile-clean (type-checking) state of the solution file. Lower is better.
- **Green-stability rate:** fraction of post-first-green edits that PRESERVE the green
  (compile-clean) state (i.e. do not re-break it). Higher is better. This is the
  green-stability metric from the DOCX convergence work (compile-preserving incremental
  edits converge; big-bang rewrites spiral).
- **Compile-stuck incidence:** fraction of runs that never reach first-green (stuck the
  whole run). Lower is better.
- Reported as ON vs OFF with variance/CIs; source is the per-turn edit + typecheck stream
  banked in M2a's hook-reality capture plus existing transcript/`typecheck` data.

### 5.3 Hook reality / treatment integrity (gate)

Per turn, for the ON arm, record whether `fmt` ran on the edited `.ail` file and its exit
status, classified as:
- **exit 0** — canonical formatting applied (`✓ Formatted` on hook stderr), OR
- **exit 3** — file did not parse yet (expected mid-edit; the hook defers silently), OR
- **refusal/error** — any other exit (surfaced via `hookSpecificOutput.additionalContext`),
  including the ~8% fail-closed refusal baseline observed in `m-ailang-fmt-phase2` /
  inline-interior work.
- **Treatment-delivery rate** = fraction of ON-arm `.ail` edits where `fmt` exited 0.
- If the treatment-delivery rate is low (hook mostly refused/no-op'd), the ON arm was NOT
  actually treated → the result is **unevaluable**, not a null. The M4 verdict MUST
  distinguish "formatter is neutral/harmful" from "treatment never delivered". Compare the
  observed refusal rate against the ~8% baseline.

---

## 6. Confidence method and refutation threshold (frozen)

- **Unit of evidence:** the N-run aggregate, NEVER a single run (mission noisy-agentic-metrics
  discipline). All deltas are aggregate over the 5 runs/arm/benchmark.
- **Confidence method:** for each pass rate (a binomial proportion over the pooled runs),
  report the **Wilson score 95% confidence interval**. For the ON−OFF pass-rate delta,
  report the **95% CI of the difference of two independent binomial proportions**
  (Newcombe/Wilson-based method). Convergence metrics (continuous/count) report
  mean ± 95% CI (normal/bootstrap over the run-level values). This reuses the existing
  N-run aggregate tooling (`tools/eval_best_of_n.py`-class); no separate aggregator.
- **Refutation / "no meaningful difference" threshold (frozen):**
  H1 is considered **supported** only if the overall ON−OFF pass-rate delta is **≥ +0.10
  (10 percentage points)** AND the 95% CI of that delta **excludes 0** (lower bound > 0).
  - If the delta's 95% CI **includes 0**, or the point delta is **< +0.10**, the result is
    declared **NULL** (no meaningful positive difference) — and this null is published.
  - If the point delta is **≤ −0.10** with a 95% CI excluding 0, the hook is declared to
    **HARM** weak-model authoring.
  - Convergence acts as corroboration, not an independent pass: even a positive pass-rate
    delta is reported as "weak/unconfirmed" if edits-to-first-green does not also improve.
  - Any headline verdict is **void** (→ **unevaluable**) if the ON-arm treatment-delivery
    rate (§5.3) is low enough that fmt did not actually run on most edited files.

---

## 7. Reporting commitments (frozen)

- Aggregate all 60 runs; preserve every per-run trace/bank; report variance and CIs, never
  point estimates or selected anecdotes.
- Publish the sign of the result — positive, negative, or NULL — and explicitly separate a
  true formatter effect from a treatment-delivery failure.
- Record `rig.lock` acquire/release around the M2b eval step only; any optional local
  (Ollama) replication arm is reported separately and is NOT part of this frozen primary
  result.

---

**Frozen by**: sprint-executor (M1), 2026-07-21. No scored runs have executed as of freezing.

## Amendments

(none)
