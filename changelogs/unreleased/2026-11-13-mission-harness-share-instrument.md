### Fixed — mission-control Gate 5 harness share counts log headlines, not the truncated index

Gate 5 step 2b's prescribed harness-share count was `grep -c '\[HARNESS\]'` over
`<mission>-mission-index.md`. That count covered the whole file, not the last 20 iterations, and
the regenerated index truncates long headlines to `...` before their trailing `[CLASS]` tag. On
fleet iterations 92–111 the last 20 index rows held 1 HARNESS tag where the log headlines held 6,
so the step's one-third alarm could not fire. The step now reads the last 20 `## N — ` headlines
across the mission log and its archive. It reports `harness/tagged`, and treats untagged rows as
unmeasured. The `.agents/skills` mirror is updated in step.
