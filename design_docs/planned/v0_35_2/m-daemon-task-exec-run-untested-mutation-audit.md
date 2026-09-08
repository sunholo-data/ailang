# Mutation audit — M-DAEMON-TASK-EXEC-RUN-UNTESTED (V1 mission iteration 352)

- **Sprint ID:** `v1-iter352-daemon-task-exec`
- **Design doc:** `design_docs/planned/v0_35_2/m-daemon-task-exec-run-untested.md` (rev 2)
- **Sprint plan:** `design_docs/planned/v0_35_2/m-daemon-task-exec-run-untested-sprint-plan.md`
- **Executor:** cross-provider executor sub-agent (this session)
- **Date:** 2026-09-08
- **Gate under test:** `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1`
  (the `-v` variant was always run to confirm the `--- PASS: TestExecuteTask_HandsDefaultBasedOptionsToExecutor` line — the B3 vacuous-green trap).

## Baseline shasums (before any mutation)

| File | sha256 |
|------|--------|
| `internal/coordinator/daemon_tasks_exec_run.go` | `efd85d4eba1f9673a6421c8a12302a531ec6966cbedd7b9edc32ed12cb094ed3` |
| `internal/coordinator/daemon.go` | `e13fd86996abaec2b318c3f329a882afe0b4fffc8ba7d803fb57bfc8ee810871` |
| `internal/coordinator/daemon_tasks_exec_run_test.go` | `6d6f247e3783e2c9516bddf397c2f743f97a9591c74c81430bfb9900c3d049d9` |

Every mutation was applied one at a time, the gate run, the result recorded, then reverted and
verified **byte-identical** to the baseline shasum above. A mutation that could not be reverted
byte-identically would be a failed audit; none was.

## Audit matrix

| audit entry | exact edit (old → new) | command | observed rc | observed `--- PASS`/`--- FAIL` line | before sha256 | after sha256 (reverted) |
|-------------|------------------------|---------|-------------|--------------------------------------|---------------|--------------------------|
| mutation M1 | `daemon_tasks_exec_run.go:241` `opts := DefaultExecuteOptions()` → `opts := &ExecuteOptions{}` | `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1 -v` | **1 (RED)** | `--- FAIL: TestExecuteTask_HandsDefaultBasedOptionsToExecutor` / `daemon_tasks_exec_run_test.go:67: RetryBaseDelay = 0s, want 1s (base DefaultExecuteOptions)` | `efd85d4e…` | `efd85d4e…` (identical) |
| mutation M2 | `daemon_tasks_exec_run.go:242` `opts.Timeout = agentConfig.GetEffectiveTimeout()` → `opts.Timeout = 5 * time.Minute` | same | **1 (RED)** | `--- FAIL` / `daemon_tasks_exec_run_test.go:78: Timeout = 5m0s, want 15m` | `efd85d4e…` | `efd85d4e…` (identical) |
| mutation M3 | `daemon_tasks_exec_run.go:246` delete `opts.AgentConfig = agentConfig // For system prompt construction (v0.8.0+)` | same | **1 (RED)** | `--- FAIL` / `daemon_tasks_exec_run_test.go:98: AgentConfig = 0x0, want 0x18f0dcf74908 (scriptAgent)` | `efd85d4e…` | `efd85d4e…` (identical) |
| mutation M4 (plan's edit) | `daemon_tasks_exec_run.go:335` `result, err := d.executor.ExecuteWithRetry(taskCtx, analyzed, opts, 2)` → `result, err := &ExecuteResult{Success: true}, error(nil)` | same | **1 (BUILD FAILURE, not test-failure)** | `# …/coordinator [build failed]` / `daemon_tasks_exec_run.go:113:2: declared and not used: analyzed` | `efd85d4e…` | `efd85d4e…` (identical) |
| mutation M4 (corrected variant, supplementary) | `daemon_tasks_exec_run.go:335` → `result, err := &ExecuteResult{Success: true}, error(nil)` + `_ = analyzed` | same | **1 (RED)** | `--- FAIL` / `daemon_tasks_exec_run_test.go:62: ExecuteWithRetry was never called (cap.opts == nil); check the budget gate` | `efd85d4e…` | `efd85d4e…` (identical) |
| guard G2 | `daemon.go:88` `executor taskExecutor` → `executor *TaskExecutor` | `go build ./internal/coordinator/` | **0 (NOT RED — see note)** | `go build` does not compile `_test.go` files | `e13fd869…` | `e13fd869…` (identical) |
| guard G2 (correct gate) | same | `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1` | **1 (RED)** | `# …/coordinator [build failed]` / `daemon_tasks_exec_run_test.go:50:13: cannot use cap (variable of type *capturingExecutor) as *TaskExecutor value in struct literal` | `e13fd869…` | `e13fd869…` (identical) |
| supplement S1 | `daemon_tasks_exec_run.go:243` `opts.IdleTimeout = agentConfig.GetEffectiveIdleTimeout()` → `opts.IdleTimeout = 3 * time.Minute` | same | **1 (RED)** | `--- FAIL` / `daemon_tasks_exec_run_test.go:83: IdleTimeout = 3m0s, want 90s` | `efd85d4e…` | `efd85d4e…` (identical) |
| supplement S2 | `daemon_tasks_exec_run.go:244` `opts.Workspace = workspacePath` → `opts.Workspace = ""` | same | **1 (RED)** | `--- FAIL` / `daemon_tasks_exec_run_test.go:88: Workspace = "", want "/tmp/ailang-test-workspace"` | `efd85d4e…` | `efd85d4e…` (identical) |
| supplement S3 | `daemon_tasks_exec_run.go:245` `opts.ObservatoryContext = obsContext` → `opts.ObservatoryContext = nil` | same | **1 (RED)** | `--- FAIL` / `daemon_tasks_exec_run_test.go:93: ObservatoryContext = <nil>, want non-nil with TaskID "task-exec-1"` | `efd85d4e…` | `efd85d4e…` (identical) |

Every mutation that was expected to turn the gate RED did so (via test assertion failure), except
the two plan-defect rows (M4 plan edit, G2 plan gate) which are documented below. Every revert
restored the file byte-identically (after-sha256 == before-sha256 == baseline).

## Mutation M5 — the two-run controlled isolation experiment

**Held constant in BOTH runs** (applied to the working copy, then torn down):
- Test-local store wrapper `fixedSpendStore{*MockStore}` returning `GetCostByProvider() → {"claude": 5.0}` (a wrapper, NOT a change to shared `MockStore`).
- Hostile config at `$M5_DIR/hostile.yaml`:
  ```yaml
  budgets:
    providers:
      claude:
        daily_budget: 1.0
        hard_limit: true
  ```
- Outer env `AILANG_CONFIG=$M5_DIR/hostile.yaml` exported in the invoking shell.

**Provider-selection pre-check (asserted, not assumed):** the fixture agent sets no `Provider`,
so `checkBudgetBeforeExecution` resolves `provider := "claude"` (the default at
`daemon_tasks_budget.go:22-26`). Spend and hostile file both key on `"claude"`.

| run | variable changed | command | observed rc | observed `--- PASS`/`--- FAIL` line |
|-----|------------------|---------|-------------|--------------------------------------|
| Run A | none (fixture's inner `t.Setenv` pin present) | `AILANG_CONFIG=$M5_DIR/hostile.yaml go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1 -v` | **0 (GREEN)** | `--- PASS: TestExecuteTask_HandsDefaultBasedOptionsToExecutor` |
| Run B | delete ONLY the fixture's `t.Setenv("AILANG_CONFIG", …)` pin (blank-imported `path/filepath` to keep the file compiling — mechanical, not a second variable) | identical command | **1 (RED)** | `--- FAIL` / `daemon_tasks_exec_run_test.go:69: ExecuteWithRetry was never called (cap.opts == nil); check the budget gate` |

**Attribution:** spend (5.0) and the hostile file were byte-identical across A and B; the only
variable was the pin. Run A GREEN (inner pin overrides outer hostile config → compiled-in
`DefaultBudgetsConfig` claude daily 30, hard → spend 5 < 30 → NOT blocked → executor reached).
Run B RED (outer hostile config applies → claude dailyLimit=1, currentSpend=5 ≥ 1, hardLimit=true
→ blocked → `executeTask` returns nil at `daemon_tasks_exec_run.go:110` before the executor call
at line 335 → `cap.opts == nil`). The 30-vs-1 asymmetry is what makes the two runs discriminate.
**The pin is load-bearing.**

**Teardown:** test file restored byte-identical to the M2 snapshot
(`6d6f247e3783e2c9516bddf397c2f743f97a9591c74c81430bfb9900c3d049d9`), `$M5_DIR` removed, env
unset.

## Deviations from the plan (stated plainly)

1. **Mutation M4 (plan's exact edit) does NOT compile.** The plan claimed
   "compiles: `result`/`err` used below at 344+", but `analyzed` is also a declared variable used
   only in the original call. Replacing the call with `&ExecuteResult{Success: true}, error(nil)`
   leaves `analyzed` unused → build failure (`declared and not used: analyzed` at line 113), rc=1.
   This is a **plan defect**, not a test defect. Per the directive I recorded the survivor verbatim
   and did not adjust it. I additionally ran a **corrected variant** (adding `_ = analyzed`) to
   prove the test's reach-assertion actually works: it turned RED with the intended message
   "ExecuteWithRetry was never called (cap.opts == nil); check the budget gate". The corrected
   variant is supplementary evidence, clearly labelled, and was reverted byte-identically.

2. **Guard G2's plan gate (`go build ./internal/coordinator/`) does NOT catch the test-file
   compile failure.** `go build` compiles only non-test files; the `_test.go` file is compiled by
   `go test`/`go vet`. So the plan's G2 command returned rc=0 even with the field reverted to
   `*TaskExecutor`. The correct gate is `go test`, which returned rc=1 with
   `cannot use cap (variable of type *capturingExecutor) as *TaskExecutor value in struct literal`
   — proving the seam is load-bearing. This is a **plan defect** in the gate command, not a
   production issue.

3. **M5 Run B required a mechanical import fix.** Deleting the fixture's `t.Setenv` line makes the
   `path/filepath` import unused (it is used only in that line), which would cause a build failure
   rather than the intended test-failure RED. I blank-imported `path/filepath` (`_ "path/filepath"`)
   to keep the file compiling. This is a mechanical necessity, not a second experimental variable;
   the pin (the `t.Setenv` call) was the only behavioural change between Run A and Run B.

4. **Parallel tool-call ordering artifacts.** Early in the audit I ran an edit and its test in the
   same parallel block; the test occasionally observed the post-revert file (PASS) instead of the
   mutated file. I detected this (shasum mismatch / unexpected PASS), re-ran each affected
   mutation cleanly (apply → test alone → revert → verify), and confirmed the correct RED. The
   final recorded results above are all from clean sequential runs.

## Final tree state (end of M3 == end of M2, byte-identical)

| File | sha256 | matches |
|------|--------|---------|
| `internal/coordinator/daemon.go` | `e13fd86996abaec2b318c3f329a882afe0b4fffc8ba7d803fb57bfc8ee810871` | M2 snapshot |
| `internal/coordinator/daemon_tasks_exec_run.go` | `efd85d4eba1f9673a6421c8a12302a531ec6966cbedd7b9edc32ed12cb094ed3` | baseline (unmodified production) |
| `internal/coordinator/daemon_tasks_exec_run_test.go` | `6d6f247e3783e2c9516bddf397c2f743f97a9591c74c81430bfb9900c3d049d9` | M2 snapshot |

**M2 acceptance re-verified after teardown:** `go build ./internal/coordinator/` rc=0,
`go build ./cmd/ailang/` rc=0, and
`go test ./internal/coordinator/ -run 'TestExecuteTaskQueueSurvivesClosedStore|TestExecuteTask_HandsDefaultBasedOptionsToExecutor' -count=1 -v`
rc=0 with `--- PASS: TestExecuteTask_HandsDefaultBasedOptionsToExecutor` and
`--- PASS: TestExecuteTaskQueueSurvivesClosedStore` (both ran).

## Conclusion

The M2 test is **not vacuously passing** and every assertion is individually load-bearing:
- M1/M2/M3/S1/S2/S3 each turn the gate RED via their specific assertion.
- M4 (corrected) proves the reach-assertion fires when the executor is never reached.
- M5 proves the hermeticity pin is load-bearing (Run A GREEN, Run B RED, single variable).
- G2 (correct gate) proves the seam is load-bearing (reverting the widening breaks the test file).

Two plan defects were found and are recorded verbatim above (M4's non-compiling edit and G2's
wrong gate command); neither affects production behaviour or the validity of the delivered test.
