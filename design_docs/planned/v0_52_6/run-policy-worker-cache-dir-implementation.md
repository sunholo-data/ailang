# Restricted worker compile-cache isolation: implementation

- Sprint: M-RUN-POLICY-WORKER-CACHE
- Issue: #1547
- Branch: `coordinator/task-00bafb12`
- Implementation status: complete; both milestones passed.
- Local milestone commits: M1 `f14e86c4`; M2 `3c6a2517`.
- No push, merge, or PR creation performed.

## Result

Restricted workers receive a non-empty operator `AILANG_CACHE_DIR` unchanged;
empty and unset values receive exactly one private per-run cache value. The
supervisor allocates after pipe setup, immediately before starting the worker,
checks the generated directory with `entryInsideSandbox`, and explicitly removes
it before unsafe-placement or start-failure refusal. Normal worker completion,
worker errors, admission denial, timeout and output-cap termination remove it
only after the worker lifecycle finishes. Operator directories are never removed.

Operator paths inside the sandbox warn once through the existing fresh-line
`supervisorLine` helper, beginning with `warning:` and naming poisoning and
`fs_deny_write`. Classification resolves missing leaves below symlinks. An absent
sandbox is not treated as the current directory. `trusted_host` keeps its full
environment and normal cache defaults.

The generated environment reference was regenerated with `make docs-env` from
`internal/config/paths.go`; only the intended cache description changed. The
policy guide and unreleased changelog fragment document ownership and cold
compilation, warnings, cleanup and unsafe-TMPDIR refusal.

## Validation

| Check | Result |
|---|---|
| `go test ./cmd/ailang/... -run 'Policy\|Supervise\|Worker\|Cache' -v` | PASS: 76 top-level tests, including 9 new cache tests; 145.760s |
| `make test-core` | PASS |
| `make lint` | PASS: zero issues |
| `make check-file-sizes` | PASS |
| `make check-home-isolation` | PASS |
| Sprint JSON validation | PASS before execution and after milestone updates |
| `make docs-env` | PASS: generated 245-variable reference; intended row only |

No full `make test` was run. Go, make, jq and a rootless C compiler were made
available in the environment. Final focused tests used `CGO_ENABLED=1`, `CC=cc`,
`GOMEMLIMIT=384MiB`, `GOGC=20`, `GOMAXPROCS=1`, and `GOFLAGS=-p=1`. A temporary
Python `PR_SET_CHILD_SUBREAPER` wrapper reaped killed orphan descendants because
this container's PID 1 leaves zombies that the existing descendant test treats
as alive. It changed no repository code or test assertions. Earlier attempts
failed from CGO-disabled SQLite, unreaped zombies or concurrent-build resource
limits; corrected-environment final gates passed. Lint ran separately with
`GOMEMLIMIT=768MiB`, `GOGC=30`, `GOMAXPROCS=2`, and serial Go package builds.

Observed small-program compile/run latency: unset default 131.7ms, explicitly
empty default 119.3ms, external override first run 142.4ms, external override
second run 103.3ms. These single-run timings are informational.

## Independent evaluation

PASS: 98/100, all 11 acceptance criteria met, no hard failures or correctness
findings. The only deduction is the rubric's 50-line limit on a table-driven
integration test; no functional revision was requested. Evaluation is recorded
in `.ailang/state/evaluations/eval_M-RUN-POLICY-WORKER-CACHE_round_1.json`.

## Artifacts

Files created:

- `cmd/ailang/run_policy_cache.go`
- `cmd/ailang/run_policy_cache_test.go`
- `cmd/ailang/run_policy_cache_integration_test.go`
- `changelogs/unreleased/2026-10-09-run-policy-worker-cache-dir.md`
- `design_docs/planned/v0_52_6/run-policy-worker-cache-dir-implementation.md`
- `.ailang/state/evaluations/eval_M-RUN-POLICY-WORKER-CACHE_round_1.json` (independent evaluation)

Files modified:

- `cmd/ailang/run_policy_supervise.go`
- `cmd/ailang/run_policy_hardening_test.go`
- `internal/config/paths.go` (environment documentation metadata)
- `docs/docs/reference/env-vars.md`
- `docs/docs/guides/agent-tool-policy.md`
- `design_docs/planned/ailang-core-triage/run-policy-worker-cache-dir.md`
- `design_docs/planned/v0_52_6/run-policy-worker-cache-dir-sprint-plan.md`
- `.ailang/state/sprints/sprint_M-RUN-POLICY-WORKER-CACHE.json`

## PR body

Restricted `run --policy` now honours operator `AILANG_CACHE_DIR` values and
otherwise uses a private per-run compile cache outside `fs_sandbox`. Generated
caches are removed after worker termination and before start or placement
refusals. Explicit in-sandbox overrides proceed with one fresh-line `warning:`
naming the poisoning risk and `fs_deny_write`; trusted-host behaviour is preserved.

Adds environment precedence, symlink placement, private ownership and lifecycle
cleanup regressions, and updates the generated environment reference, policy
guide and unreleased changelog.

Validation: focused Policy/Supervise/Worker/Cache suite, `make test-core`,
`make lint`, `make check-file-sizes`, and `make check-home-isolation` passed.
The focused suite used CGO and a temporary child-reaping wrapper for the container.
No full `make test` was run.

Closes #1547
