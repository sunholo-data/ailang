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
