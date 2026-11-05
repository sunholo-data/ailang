# Agent-mode evals bank "never wrote the solution file" as runtime_error

- **Date**: 2026-11-05
- **Class**: bug (measurement taxonomy — a tool-protocol miss is banked as if AILANG broke)
- **Recommend**: design-doc
- **Searched**: `no_solution_written`, `template-unchanged`, `exports no functions`, `tool protocol`, `solution template`, `error_category`, `eval-summary`, `eval-elo`; browsed `design_docs/planned/` (incl. m-eval-failure-attribution, m-eval-validity-discipline). No doc covers this class: **none found** for agent-mode "solution file left as placeholder" detection.
- **Related (not duplicate)**: [m-eval-failure-attribution](../m-eval-failure-attribution.md) is the same family — failure rows that misrepresent what broke — but it scopes to **standard-mode** harness/AILANG/benchmark attribution and explicitly freezes `error_category` values as unchanged. This report asks for a **new agent-mode `error_category` value**, which that doc's freeze decision does not authorize; it needs its own (small) decision.

## Why

The mechanism is verified in code. `agent_runner_multi.go` (placeholder seed, `// TODO: Add your solution code below`) writes `benchmark/solution.ail` as a placeholder before the agent runs. `validateOnce` (internal/eval_harness/agent_validation.go) then grades whatever is in that file — and if the agent delivered its final code in chat and never wrote the file, the graded content is the untouched template (~102–112 bytes). An empty AILANG module compiles clean, exports nothing, and the run dies with "module benchmark/solution exports no functions" (internal/runner/entry_unexported.go), which `CategorizeError` banks as `runtime_error` — the category every downstream reader treats as "the model wrote code that crashed."

Measured consequence on the rig (2026-11-05, pi-gemma4-12b, 24 agent benchmarks): 6 of 13 failures banked `runtime_error` with the template string as the graded code field; the report asserts 12 of 13 failures were protocol misses and only 1 a real compile error — so the current banking makes a protocol-failure model look like it broke AILANG 6 times. I could not re-count the JSONs here: `eval_results/finetune-phase0/agent/pi-gemma4-12b/` does not exist in this checkout (rig-side evidence). The code mechanism above is verified regardless; the 12/13 split should be re-confirmed at design time from the rig.

Note the attribution polarity, which the design must get right: unlike the harness-caused classes in m-eval-failure-attribution, a never-written solution file **is the model's failure** (it had write access and didn't use it). The ask is not to excuse it but to give it its own name, so a protocol miss stops inflating "AILANG is broken" and the two failure populations stay separable in eval-summary / eval-elo error mixes.

## Why design-doc, not direct-fix

- **There is more than one acceptable way** (row 3): detect pre-compile by comparing the graded content against the seeded placeholder (needs the placeholder hash/string plumbed from `agent_runner_multi.go` to `validateSolution`), or post-hoc as a signature ("exports no functions" **and** content == template) in the error-categorizer table; and separately, whether such rows should also carry an m-eval-failure-attribution `attribution` value (`model`) so validity/filtering composes.
- **It spans more than one file** (row 5): new `ErrorCategory*` constant (metrics.go), detection at the agent grading path (agent_validation.go and/or agent_runner_multi.go, incl. the python lane's `# TODO: Write your solution here` placeholder), plus surfacing in the eval-summary and eval-elo error mixes (cmd/ailang).
- **It touches a gate's data contract** (row 4): a new value in the `error_category` namespace is read by dashboards, ELO fitting and curation; the m-eval-failure-attribution freeze decision ("error_category keeps its exact current values") means adding a value needs an explicit, written decision rather than a drive-by.

## Suggested scope for the design doc

1. New `ErrorCategoryNoSolution = "no_solution_written"`; detected when graded solution content equals the seeded placeholder (exact-match or hash, per language).
2. Detection at the grading path, before compile/run, at both AILANG and non-AILANG placeholder sites.
3. Attribution: `model` (a protocol miss is a genuine capability signal — but a *different* one than writing-bad-code); validity stays valid=true so it still FAILs.
4. Surface the category in eval-summary and eval-elo error mixes.
5. Decide re-classification of existing agent rows (annotate-in-place per the v0.30.0 precedent, as in m-eval-failure-attribution).
