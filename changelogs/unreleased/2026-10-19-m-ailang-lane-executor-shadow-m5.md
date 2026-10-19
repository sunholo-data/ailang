### Added — executor shadowing, M5: the mechanical grade job (2026-10-19)

M-AILANG-LANE-EXECUTOR-SHADOW M5. Once both sides of a `ShadowComparison` are in, the cloud coordinator's tick
(`dispatchShadowGrades`, `internal/coordinator/shadow_grade.go`) claims it and dispatches a grade job: a mode of
the existing agent executor job (`AILANG_JOB_MODE=shadow_grade`, `AILANG_SHADOW_PARENT`, new registry vars), so
no new Cloud Run job or infrastructure. The job (`cmd/ailang/coordinator_shadow_grade_job.go`) runs no agent and
pushes nothing. For each side it fetches exactly the run's base commit, applies the run's diff, and runs only
AILANG gates through the policy tool under the agent's restricted policy: `check` and `test` (inline tests) on
every changed module, plus `test --package .` for a package that ships `*_test.ail` files. Each gate's sandbox is
its module's package directory, the agent's own view; a module inside a package does not resolve from the clone
root. Gates are killed at 120 s each and the job stops at 900 s; a late grade never overwrites one the
coordinator already settled. Results: per-gate output, a per-side pass, and a verdict (`primary_better`,
`shadow_better`, `equivalent`, `both_failed`, or `ungraded` with a reason such as `apply_failed`, `timeout`,
`gate_refused`, `no_ail_changes`, `grade_timeout`). A side that produced no diff counts as failing; a setup
problem never does. Grading code lives only in the job; the coordinator package cannot run it. Verified end to
end with a real ailang binary and the package-lane policy shape (opt-in `TestShadowGradeLive`); the Firestore
query and claim were verified live on the dev project.
