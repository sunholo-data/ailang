# `verify` / `ai-check` have a per-function Z3 bound but no total-run budget (#513)

- **Date**: 2026-10-03
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `#513` (hits: mission logs and `design_docs/implemented/v0_31_0/m-z3-hard-timeout-sprint-plan.md`, which implemented the per-call hard bound for #510 and explicitly deferred this); "total budget", "run budget", "wall-clock budget", `--total-timeout`, `--budget` across design_docs/planned (no hits that cover verify/ai-check). `internal/proctree` now exists as the shared process-group helper (`internal/smt/solver.go:15,166`), so the issue's "local process-group helpers vs shared `internal/procgroup`" note is already resolved and drops out of scope.
- **Estimate**: omitted (design-doc)

Still true at origin/dev `790169359`. `smt.Solve` (`internal/smt/solver.go:114`) runs Z3 under
`exec.CommandContext` with a hard per-call bound and process-group kill (`proctree.Configure`, :165-166).
`smt.Verify` (`internal/smt/verify.go:77`) calls it once per contracted function inside
`for funcName, meta := range coreProg.Meta` (:211, Solve at :362), so a run over N functions can take
N × (`--timeout` + 2s grace). `--timeout` is documented per-function in both commands
(`cmd/ailang/verify.go:22,51`; `cmd/ailang/ai_check.go:60,77`), and both route through the same
`smt.Verify` with `VerifyOptions.Timeout` (`verify.go:41-43`), so one option struct carries the policy for
both legs.

A fact the issue did not have, and which shapes the answer to its question 2: the per-function loop iterates
a Go **map** (`core.Program.Meta map[string]*DeclMeta`, `internal/core/core.go:430`). Today that is harmless
because every function is visited. Under a total budget, *which* functions get verified before expiry would
be random from run to run — a nondeterministic verdict on identical source. Any budget design must first fix
the visit order (sorted by name, or source position).

Decisions the doc must make (the issue's four, answered where the code already constrains them):

1. **Flag shape.** New `--total-timeout` (recommended) rather than overloading `--timeout`, which keeps the
   documented per-function meaning and the eval harness's existing invocations unchanged. Default: off
   (unbounded) so no existing gate changes behaviour.
2. **On expiry.** Stop dispatching new solves; let the in-flight solve finish under its own per-call bound
   (it is already hard-bounded); results already computed stand. Visit order sorted, so the verified prefix
   is reproducible.
3. **Partial results.** Each unvisited function gets a result with `status: "skipped"` and a distinct reason
   code (e.g. `budget_exhausted`), plus a report-level `budget_exhausted: true` and counts; the JSON shape
   is otherwise unchanged, so existing consumers that iterate `results[]` keep working.
4. **`ai-check` consumer contract.** `ai-check` already exits non-zero on verifier errors (measured: rc=1 on
   a `status: error` result). An exhausted budget is *not* a pass: recommend a non-zero exit (distinct
   from the error code if the CLI has room) so the World `ailang-code` profile and the agent convergence
   loop cannot read a truncated run as green. `verify` should report it as skipped work, consistent with
   its honest-skip contract (#757).

Recommendation: write it up as a small M-VERIFY-RUN-BUDGET doc with the sorted-order change as M1 (useful on
its own: stable output ordering), the flag + partial-result shape as M2, and the `ai-check` exit code as M3,
which is the only part that needs the mission/World consumers to sign off. Issue:
https://github.com/sunholo-data/ailang/issues/513
