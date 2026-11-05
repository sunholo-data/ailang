### Fixed — `eval-suite --budget-usd` now actually stops a parallel run (2026-11-05)

The budget cap printed "Budget exceeded … stopping new trials" and then kept
starting them. `runBenchmarksParallel`'s dispatch loop never blocks, so every
job passed the pre-semaphore abort check at once and queued on the semaphore;
after acquiring a slot only the `--max-wall-clock` deadline was re-checked.
Measured on a 918-job `--parallel 6` OpenRouter run: the $8 cap tripped at run
609 and 309 more runs started (about $12 traced). The same gap let the
"first 50 results all failed" abort run every queued job too.

All stop conditions (budget, failure-streak abort, deadline) are now re-checked
after the semaphore, so at most the in-flight `--parallel` trials finish after a
trip. Regression test `TestRunBenchmarksParallel_BudgetStopsQueuedJobs` fails
10/10 trials started without the fix.

### Added — fine-tune Phase 0 size-ladder lanes (2026-11-05)

`internal/modelreg/models.yml` gains attended-only lanes for the
`m-eval-finetuning-data-pipeline` Phase 0 baseline: Gemma 4 E4B / 12B (MXFP8)
and Qwen 3.5 9B / Qwen 3.8 27B in standard (`ollama-*`) and pi agent (`pi-*`)
modes with identical budgets, plus open-weight OpenRouter rungs
(`or-qwen3-5-9b`, `or-gemma-4-31b`, `or-ministral-8b`, `or-ministral-14b`).
Not part of any rotation.

### Fixed — agent mode honours `--prompt-version` (2026-11-05)

`eval-suite --agent` passed a hard-coded `""` as the teaching-prompt version, so
`--prompt-version` silently had no effect in agent mode: every agent run got the
active prompt. The version now flows through `AgentBenchmarkConfig.PromptVersion`
(regression tests: `TestBuildAgentSuiteConfig_CarriesPromptVersion`,
`TestGenerateAgentPrompts_UsesRequestedPromptVersion`).

### Added — experimental fine-tune prompt versions `ft-slim-v1`, `ft-none` (2026-11-05)

Two frozen, never-active prompt versions for the fine-tune A/B: `ft-slim-v1` is a
snapshot of the 2.4 KB dialect-traps card, `ft-none` is one line. Select with
`--prompt-version`; the active prompt stays `v0.16.11`.
