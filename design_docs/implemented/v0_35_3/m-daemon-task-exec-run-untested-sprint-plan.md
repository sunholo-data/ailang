# Sprint plan — M-DAEMON-TASK-EXEC-RUN-UNTESTED (V1 mission iteration 352)

- **Sprint ID:** `v1-iter352-daemon-task-exec`
- **Design doc:** `design_docs/planned/v0_35_2/m-daemon-task-exec-run-untested.md` (rev 2, post-quorum,
  closed under the narrow-refinement carve-out with all three reviewers' verbatim fixes applied)
- **Base commit:** `fc84c57d4` (the design doc's own commit; verified at base by `git rev-parse HEAD`)
- **Planner role:** cross-provider planner sub-agent; this plan writes no source code. The executor
  applies the milestones; the controller commits.
- **Duration:** ≤ 3 days (doc-scoped). Day 1 = M1, Day 2 = M2, Day 3 = M3 + buffer.

## Namespace disambiguation (read this first)

The doc uses **two namespaces that collide on the label "M1"–"M3"**:

- **Milestones M1, M2, M3** — commit boundaries (seam → test → audit). This plan's day-by-day units.
- **Mutations M1–M5** — deliberately broken variants of the *production* code (or, for M5, of the
  fixture's pin) that must turn the M2 test RED. They are audit artefacts, not milestones.

In this plan: milestones are always written **M1/M2/M3** and audit entries always
**mutation M1 … mutation M5**. Where a milestone section must reference a same-numbered mutation,
it says so explicitly (e.g. "mutation M1", not "M1"). Mutations M1–M4 are edits to
`internal/coordinator/daemon_tasks_exec_run.go`; mutation M5 is a **two-run experiment** on the
test file + process environment (see M3 section). Supplemental audit mutations added by this plan
are labelled **S1–S3** and never share the M1–M5 numbering.

## Baseline (pristine tree at `fc84c57d4`, measured by this planner session)

Every command run with `cmd; rc=$?` — exit codes observed, never inferred. All commands are
**local** (`go build`, `go test` over the warm module cache in this sandbox); none touches a
socket, so per the contract these are real in-sandbox evidence. Nothing network-touching is part
of any acceptance gate in this plan.

| # | Acceptance command (from the doc) | Observed | rc | Interpretation |
|---|-----------------------------------|----------|----|----------------|
| B1 | `go build ./internal/coordinator/` | no output | **0** | matches doc V12 |
| B2 | `go build ./cmd/ailang/` | no output | **0** | matches doc V13 |
| B3 | `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1` | `ok … 2.125s [no tests to run]` | **0** | **VACUOUS GREEN at base** — the test does not exist yet. This gate is load-bearing only from M2 on, and only if the executor proves the test actually ran (see `-v` check in M2 acceptance) |
| B4 | `go test ./internal/coordinator/ -run 'TestExecuteTaskQueueSurvivesClosedStore\|TestExecuteTask_HandsDefaultBasedOptionsToExecutor' -count=1` | `ok … 2.736s`; `-v` shows `--- PASS: TestExecuteTaskQueueSurvivesClosedStore (0.00s)` | **0** | real pass: the shutdown test runs at base (V25 anchor) |
| B5 | `go build ./...` | `# github.com/sunholo-data/ailang/cmd/wasm` / `runtime.main_main·f: function main is undeclared in the main package` | **1** | **confirmed first-party**: red at base on `cmd/wasm` only (doc V14). It measures the repo, not this change; no milestone gates on `./...` |

**Planner re-measurements of load-bearing doc claims (at `fc84c57d4`):**

- **`d.executor` reference count: 5 — agrees with doc V9.** `grep -rn 'd\.executor' internal/coordinator/*.go | grep -v _test` returns 4 uses; the declaration (`daemon.go:88: executor *TaskExecutor`, no `d.` prefix) is the 5th, verified by `sed`. Breakdown: declaration `daemon.go:88`, nil checks `daemon.go:516` and `daemon_tasks_exec.go:37`, call `daemon_tasks_exec_run.go:335`, assignment `daemon_tasks_init.go:274`. The only other `.executor` hits are `p.executorName` in `provider_executor.go` — a **different field** (string on the provider-executor struct), no conflict. No disagreement with the doc.
- **Line numbers at `fc84c57d4` match the doc's** (doc measured at parent `3ee5bb177`; tree is unchanged in this file): `opts := DefaultExecuteOptions()` at **241**; the five overrides at **242–246** (`Timeout`/`IdleTimeout`/`Workspace`/`ObservatoryContext`/`AgentConfig`); the `ExecuteWithRetry` call at **335**; budget gate `return nil` at **104–110**.
- **`taskExecutor` identifier collision measured:** the name appears only as a **local variable** in `daemon_tasks_init.go:269/274/275` (`taskExecutor, err := DefaultTaskExecutor()`). A package-level `type taskExecutor interface` is therefore legal Go — the local variable merely shadows the type name inside that one function, and the assignment at line 274 still type-checks because `*TaskExecutor` satisfies the interface. M1's build gate must prove this compiles.
- **`time` is imported** in `daemon_tasks_exec_run.go` (line 11) → mutation M2 (`5*time.Minute`) compiles.
- **`ExecuteResult` has `Success`/`Provider`/`SessionID`** (`provider.go:137`) → the capturing fake's literal compiles; `ExecuteOptions` has `RetryBaseDelay`/`Wait`/`DryRun` + the five overridden fields (`provider.go:53–70`).
- **`daemon_tasks_exec_run_test.go` does not exist at base** — M2 creates it.
- **Hermeticity mechanism spot-checked:** `agent_config.go:14-22` checks `AILANG_CONFIG` before home; `LoadBudgetsConfigFrom:214-229` returns `DefaultBudgetsConfig()` on `os.IsNotExist`; `DefaultBudgetsConfig:191-195` → claude `{DailyBudget 30, HardLimit true}`; provider default `"claude"` at `daemon_tasks_budget.go:22-26` when `agentConfig.Provider == ""`; `ConfigFile.Budgets` yaml key is `budgets` (`agent_config.go:134`), `ProviderLimit` keys `daily_budget`/`hard_limit` (`agent_config.go:173-175`). All mutation-M5 constants below key on these measured facts.

## Milestone M1 — introduce the seam (Day 1, ~half a day incl. verification)

**Change (doc §1, verbatim scope — a widening, nothing else):**

- File touched: **`internal/coordinator/daemon.go` only**.
  1. Add the narrow interface adjacent to the field, with the comment naming the single production
     implementor per the doc's Risk mitigation:

     ```go
     // taskExecutor is the narrow seam executeTask needs. The single production
     // implementor is *TaskExecutor (assigned in daemon_tasks_init.go).
     type taskExecutor interface {
         ExecuteWithRetry(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions, maxRetries int) (*ExecuteResult, error)
     }
     ```

  2. One-line widening at line 88: old `executor         *TaskExecutor` → new `executor         taskExecutor`.
- **Nothing else changes.** The 4 remaining references (nil checks `daemon.go:516`,
  `daemon_tasks_exec.go:37`; call `daemon_tasks_exec_run.go:335`; assignment
  `daemon_tasks_init.go:274`) compile unchanged because `*TaskExecutor` satisfies the interface.
  No production default or code path changes in effect — assert this at review time by diffing:
  the M1 commit must touch exactly one file and zero call sites.

**Acceptance (all rc captured, all local):**

| Command | Expected rc |
|---------|-------------|
| `go build ./internal/coordinator/; rc=$?; echo $rc` | 0 |
| `go build ./cmd/ailang/; rc=$?; echo $rc` | 0 |
| `go test ./internal/coordinator/ -run TestExecuteTaskQueueSurvivesClosedStore -count=1 -v` | 0, with `--- PASS` line (V25: `&Daemon{executor: &TaskExecutor{}}` at `daemon_tasks_exec_shutdown_test.go:20` still compiles and passes) |

**Test-plan table (M1):** M1's guard is a compile gate, not a unit test (the doc is explicit:
"(M1 lands before M2, so the compile gate is the guard)").

| gate | what it asserts | which mutation it kills (concrete edit) |
|------|-----------------|------------------------------------------|
| `go build ./internal/coordinator/` && `go build ./cmd/ailang/` rc=0 | `*TaskExecutor` satisfies `taskExecutor`; the 4 unchanged references still type-check | **G1** — `internal/coordinator/daemon.go`, interface block: old `maxRetries int) (*ExecuteResult, error)\n}` → new `maxRetries int) (*ExecuteResult, error)\n\tBogus()\n}` → build fails at `daemon_tasks_init.go:274` (`*TaskExecutor does not implement taskExecutor`). Proves the interface is load-bearing, not decorative |
| `TestExecuteTaskQueueSurvivesClosedStore` passes | existing concrete-constructing test compiles against the widened field | **G2** — revert the widening: `daemon.go:88` old `executor         taskExecutor` → new `executor         *TaskExecutor`. **Honest caveat:** at the M1 boundary this revert compiles cleanly (nothing yet depends on the interface); G2 only has teeth from M2 on, when `daemon_tasks_exec_run_test.go` sets `executor: cap` (`cap` is not a `*TaskExecutor`) and fails to compile. The executor demonstrates G2 as part of the M3 audit, not on Day 1 |

**Committed at the M1 boundary:** exactly one file — `internal/coordinator/daemon.go`
(interface decl + comment + one-line field widening). Tree state: both targeted builds rc=0, full
existing coordinator test suite compiles and passes. Bisectable: trivially revertible, provably
behaviour-neutral, and if bisect ever lands here the diff is ~6 lines in one file.

## Milestone M2 — the first executing test (Day 2)

**Change:** add exactly one new file, **`internal/coordinator/daemon_tasks_exec_run_test.go`**
(absent at base — verified), containing, per doc §3/§4 (all verbatim-mandated elements listed):

1. `capturingExecutor` fake implementing `taskExecutor`, capturing `opts` and `task`, returning
   `&ExecuteResult{Success: true, Provider: "script", SessionID: "sess-1"}`.
2. The fixture: agent registry with one script agent
   `&AgentConfig{ID: "coordinator", Inbox: "coordinator", Invoke: &InvokeConfig{Type: "script", Command: "true"}, Timeout: "15m", IdleTimeout: "90s", SkipApproval: true}`
   (all fields verified present at base: `agent_registry.go:101–200`, `InvokeConfig` at
   `agent_registry.go:19`); Daemon with `ctx: context.Background()` (gemini round-2 fix, applied
   even though its premise is REFUTED — see Risks R3, do **not** re-plan around the premise),
   `logger: log.New(io.Discard, "", 0)`, `taskStore: NewMockStore()`,
   `resourceRegistry: NewResourceTrackerRegistry()`, `agentRegistry: reg`,
   `observatorySync: NewObservatorySync(nil, logger)`, `executor: cap`; task
   `&TaskRecord{ID: "task-exec-1", Type: TaskTypeFeature, Stage: TaskStageImplementation, Title: "test task", Kind: "feature", Workspace: "/tmp/ailang-test-workspace", Iteration: 1}`
   (all fields verified present: `store.go:9–51`).
3. **REQUIRED first line of the test body** (the fix all three reviewers demanded; not optional,
   not a follow-up): `t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))`.
4. Task invocation `d.executeTask(task)` expecting nil error, then `cap.opts == nil` →
   `t.Fatalf("ExecuteWithRetry was never called (cap.opts == nil); check the budget gate")`,
   then the six assertions (base first: `RetryBaseDelay`, `Wait != nil`, `DryRun`; then
   `Timeout == 15m`, `IdleTimeout == 90s`, `Workspace == task.Workspace`,
   `ObservatoryContext.TaskID == task.ID`, `AgentConfig == scriptAgent` pointer equality).
5. **No production file changes.** Imports needed: `context`, `io`, `log`, `path/filepath`,
   `testing`, `time`. Package `coordinator` (same package — `NewMockStore` lives in
   `mock_store_test.go`).

**Acceptance:**

| Command | Expected rc | extra check |
|---------|-------------|-------------|
| `go build ./internal/coordinator/; rc=$?` | 0 | unchanged production |
| `go build ./cmd/ailang/; rc=$?` | 0 | unchanged production |
| `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1` | 0 | **plus** `-v` variant of the same command must print `--- PASS: TestExecuteTask_HandsDefaultBasedOptionsToExecutor` — kills the baseline vacuous-green trap (B3) |
| `go test ./internal/coordinator/ -run 'TestExecuteTaskQueueSurvivesClosedStore\|TestExecuteTask_HandsDefaultBasedOptionsToExecutor' -count=1` | 0 | `-v` shows BOTH tests ran (`=== RUN` ×2, `--- PASS` ×2) — the shutdown test compiles and passes with `executor: &TaskExecutor{}` against the widened field |

**Test-plan table (M2)** — every assertion row names its killing mutation as a mechanical edit.
Mutations M1–M4 are the doc's; M5 is the two-run experiment (full procedure in the M3 section).
**S1–S3 are planner-added supplements**: the doc's minimal set does not include one-edit kills for
assertions 3–5, and a row without a concrete mutation is a row not thought through — so the
concrete edits are supplied here. S1–S3 extend, never replace, the doc's M1–M5.

| test / assertion | what it asserts | which mutation it kills — concrete edit (file:line, old → new) |
|------------------|-----------------|---------------------------------------------------------------|
| `TestExecuteTask_HandsDefaultBasedOptionsToExecutor` (reach) | `executeTask` executes end-to-fixture; budget gate does not block; executor reached (`cap.opts != nil`) | **mutation M4** — `internal/coordinator/daemon_tasks_exec_run.go:335`: old `result, err := d.executor.ExecuteWithRetry(taskCtx, analyzed, opts, 2)` → new `result, err := &ExecuteResult{Success: true}, error(nil)` (compiles: `result`/`err` used below at 344+). RED with "ExecuteWithRetry was never called (cap.opts == nil); check the budget gate" |
| assertion 1 — base is `DefaultExecuteOptions()` | `cap.opts.RetryBaseDelay == 1s` and `cap.opts.Wait != nil` (**note for judge:** `DryRun == false` does NOT discriminate — false under both variants; the killers are `RetryBaseDelay`→0 and `Wait`→nil) | **mutation M1** — `daemon_tasks_exec_run.go:241`: old `opts := DefaultExecuteOptions()` → new `opts := &ExecuteOptions{}` |
| assertion 2 — `cap.opts.Timeout == 15*time.Minute` | the `GetEffectiveTimeout` override is dynamic, not hardcoded | **mutation M2** — `daemon_tasks_exec_run.go:242`: old `opts.Timeout = agentConfig.GetEffectiveTimeout()` → new `opts.Timeout = 5*time.Minute` (compiles — `time` imported at line 11). 5m ≠ 15m fixture pin makes it visible |
| assertion 3 — `cap.opts.IdleTimeout == 90*time.Second` | the `GetEffectiveIdleTimeout` override is dynamic | **supplement S1** — `daemon_tasks_exec_run.go:243`: old `opts.IdleTimeout = agentConfig.GetEffectiveIdleTimeout()` → new `opts.IdleTimeout = 3*time.Minute` |
| assertion 4 — `cap.opts.Workspace == task.Workspace` | script-agent path hands the task's workspace through unchanged | **supplement S2** — `daemon_tasks_exec_run.go:244`: old `opts.Workspace = workspacePath` → new `opts.Workspace = ""` |
| assertion 5 — `cap.opts.ObservatoryContext != nil && .TaskID == task.ID` | the obsContext override is built and tagged with the task ID | **supplement S3** — `daemon_tasks_exec_run.go:245`: old `opts.ObservatoryContext = obsContext` → new `opts.ObservatoryContext = nil` |
| assertion 6 — `cap.opts.AgentConfig == scriptAgent` (pointer equality, V21) | the agent config override is present and identical | **mutation M3** — `daemon_tasks_exec_run.go:246`: delete the whole line `opts.AgentConfig = agentConfig // For system prompt construction (v0.8.0+)` (old text → empty) |
| hermeticity guard — fixture first line | the budget config is a property of the binary, not of the machine; the pin is load-bearing | **mutation M5 — two-run experiment** (held-constants + run A/run B procedure in M3 below): run A expected GREEN, run B expected RED |

**Committed at the M2 boundary:** exactly one new file —
`internal/coordinator/daemon_tasks_exec_run_test.go`. Zero production diff (verify with
`git diff --stat` at commit time: the M2 commit shows only the test file). Bisectable: reverting
M2 removes the coverage but leaves the seam; reverting M1 after M2 breaks compilation loudly —
the intended ordering.

## Milestone M3 — mutation audit (Day 3; doc-optional hardening, budget-cut candidate #1)

**Change:** none to the tree. Apply each mutation one at a time in the working copy, observe RED,
revert, verify GREEN, record. The audit runs mutations in doc order (M1, M2, M3(mut), M4), then
the M5 two-run experiment, then supplements S1–S3 if budget remains.

**Acceptance:**

| Step | Command | Expected |
|------|---------|----------|
| per mutation M1–M4 | apply edit → `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1; rc=$?` | rc=1 (RED) with the failure mode named in the M2 table |
| revert check | `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1; rc=$?` | rc=0 after every revert |
| G2 demonstration (from M1) | revert `daemon.go:88` to `executor *TaskExecutor` → `go build ./internal/coordinator/` | rc=1 — test file no longer compiles (`cap` is not `*TaskExecutor`); then restore |
| M5 two-run experiment | procedure below | run A rc=0, run B rc=1 with the named failure message; BOTH outcomes recorded |
| record | PR/commit body | audit table (mutation → observed rc → failure line) written; nothing committed to the tree |

### Mutation M5 — the two-run controlled isolation experiment (judge-re-runnable procedure)

This is a **procedure**, not a single mutation. It varies exactly ONE variable — the fixture's pin —
across two runs, with spend and the hostile config byte-identical.

**Held constant in BOTH runs (applied to the working copy of the test file, then torn down):**

0. Provider-selection pre-check (assert, don't assume — doc §M5): the fixture agent sets no
   `Provider`, so `checkBudgetBeforeExecution` resolves `provider := "claude"` at
   `daemon_tasks_budget.go:22-26`. Verified at base by this planner. Because of this, the spend
   map and the hostile config both key on `"claude"`. If a future fixture change sets
   `agentConfig.Provider`, this audit's keys must be re-keyed — that is an audit-maintenance
   note, not a production concern.
1. Test-local store wrapper (NOT a change to shared `MockStore` — V16 untouched for every other
   test):

   ```go
   type fixedSpendStore struct{ *MockStore }
   func (fixedSpendStore) GetCostByProvider() (map[string]float64, error) {
       return map[string]float64{"claude": 5.0}, nil
   }
   ```

   and in the fixture: old `taskStore: NewMockStore(),` → new `taskStore: fixedSpendStore{NewMockStore()},`.
2. Hostile config at `$M5_DIR/hostile.yaml` (yaml keys verified at `agent_config.go:132-137,158-177`):

   ```yaml
   budgets:
     providers:
       claude:
         daily_budget: 1.0
         hard_limit: true
   ```

3. Outer environment: invoke `go test` with `AILANG_CONFIG=$M5_DIR/hostile.yaml` exported in the
   shell.

**Run A (fixture unchanged — the inner pin is present):**

```
AILANG_CONFIG=$M5_DIR/hostile.yaml go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1; rc=$?
```

Expected **GREEN (rc=0)**: the fixture's own `t.Setenv` overrides the outer value per-call
(uncached, C3/V28) → compiled-in `DefaultBudgetsConfig` (claude daily 30, hard) → spend 5 < 30 →
NOT blocked → executor reached → all six assertions pass.

**Run B (delete ONLY the fixture's pin — everything else byte-identical to run A):**

Edit the test file: delete the line `t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))`
and nothing else; re-run the identical command.

Expected **RED (rc=1)** with `ExecuteWithRetry was never called (cap.opts == nil); check the budget gate`:
the outer hostile config applies → claude `dailyLimit=1`, `currentSpend=5 >= 1`, `hardLimit=true` →
blocked → `executeTask` returns nil at `daemon_tasks_exec_run.go:110`, **before** line 335.

**Attribution:** spend (5.0) and the hostile file are byte-identical across A and B; the only
variable is the pin, so RED in B is attributable to the pin. The 30-vs-1 asymmetry is what makes
the two runs discriminate (astra's confounded-M5 fix). Record both `rc` values and run B's failure
line in the audit table. Teardown: restore the test file, `rm -rf $M5_DIR`, unset the env var.

**Test-plan table (M3 audit matrix):**

| audit entry | edit (file:line, old → new) | expected RED |
|-------------|------------------------------|--------------|
| mutation M1 | `daemon_tasks_exec_run.go:241`: `opts := DefaultExecuteOptions()` → `opts := &ExecuteOptions{}` | assertion 1 fails (RetryBaseDelay 0 ≠ 1s; Wait nil) |
| mutation M2 | `daemon_tasks_exec_run.go:242`: `opts.Timeout = agentConfig.GetEffectiveTimeout()` → `opts.Timeout = 5*time.Minute` | assertion 2 fails (5m ≠ 15m) |
| mutation M3 | `daemon_tasks_exec_run.go:246`: delete `opts.AgentConfig = agentConfig // For system prompt construction (v0.8.0+)` | assertion 6 fails (nil ≠ scriptAgent) |
| mutation M4 | `daemon_tasks_exec_run.go:335`: `result, err := d.executor.ExecuteWithRetry(taskCtx, analyzed, opts, 2)` → `result, err := &ExecuteResult{Success: true}, error(nil)` | Fatal: "ExecuteWithRetry was never called…" |
| mutation M5 | two-run experiment above | run B RED with the named Fatal; run A stays GREEN |
| guard G2 | `daemon.go:88`: `executor         taskExecutor` → `executor         *TaskExecutor` | `go build` rc=1 (test file no longer compiles) |
| supplement S1 | `daemon_tasks_exec_run.go:243`: `opts.IdleTimeout = agentConfig.GetEffectiveIdleTimeout()` → `opts.IdleTimeout = 3*time.Minute` | assertion 3 fails (3m ≠ 90s) |
| supplement S2 | `daemon_tasks_exec_run.go:244`: `opts.Workspace = workspacePath` → `opts.Workspace = ""` | assertion 4 fails ("" ≠ task.Workspace) |
| supplement S3 | `daemon_tasks_exec_run.go:245`: `opts.ObservatoryContext = obsContext` → `opts.ObservatoryContext = nil` | assertion 5 fails (nil context) |

**Committed at the M3 boundary:** nothing in the tree. The boundary artefact is the audit table in
the PR/commit description (see Open question OQ1 for the default when no PR exists yet). The tree
state at the end of M3 is byte-identical to the end of M2 — verify by running the M2 acceptance
commands once more after teardown (`rc=0` both targeted builds, both tests green, `-v` shows the
new test ran).

## Budget — the cut set (if the executor runs out of time)

Cut in this order; earlier cuts preserve later ones.

1. **Cut supplements S1–S3** (audit rows only). Partial state still satisfies: doc-complete M1–M5
   audit. Cost: assertions 3–5 lose their individually-named killing mutation in the record (they
   are still killed incidentally by M4's reach-mutation).
2. **Cut M3 (the whole audit milestone)** — the doc itself labels M3 "optional hardening". Within
   a partial audit, **mutation M4 and the M5 two-run are the last rows to be cut**: M4 proves the
   test is not vacuously passing (it would fail `cap.opts == nil` if the path were never
   reached), and M5 is the evidence the entire round-2 quorum hinged on. Cut order within the
   audit: S1–S3 → mutation M2 → mutation M3(mut) → mutation M1 → mutation M5 → mutation M4.
3. **Cut M2's landing** (worst case — Day 2 overruns): commit M1 alone. Partial state must still
   satisfy: both targeted builds rc=0; full pre-existing coordinator suite green; zero production
   behaviour change; no half-test. **Never land a de-scoped M2**: the `t.Setenv` first line, the
   `cap.opts == nil` Fatal naming the budget gate, and all six assertions are reviewer-mandated
   minima — an M2 without any of them does not ship in this sprint, it waits.

Non-negotiable regardless of budget: no production behaviour change anywhere (the only production
edit in the whole sprint is `daemon.go:88` + the interface block), and no gate may use
`go build ./...` (red at baseline on `cmd/wasm`, B5).

## Risks / open questions

- **R1 — Vacuous-green acceptance (measured at base, B3).** `go test -run <new test>` is rc=0 with
  `[no tests to run]` before M2 exists. Mitigation is built into every M2/M3 acceptance row: the
  `-v` variant must print `--- PASS: TestExecuteTask_HandsDefaultBasedOptionsToExecutor`.
- **R2 — `taskExecutor` name shadowing.** The doc's interface name collides with a local variable
  at `daemon_tasks_init.go:269`. Planner-verified legal (no package-level collision; the
  assignment at :274 type-checks because `*TaskExecutor` satisfies the interface). M1's build gate
  is the proof; executor keeps the doc's name verbatim. **Not an open question** — decided by the
  doc, verified here.
- **R3 — Refuted premise, applied anyway (do not re-plan around it).** gemini-3-1-pro's round-2
  premise ("nil `d.ctx` panics") is **REFUTED** by doc V35 — `executeTask` never reads `d.ctx`;
  `taskCtx` derives from `context.Background()` at line 29. The one-line fixture addition
  (`ctx: context.Background()`) was applied as belt-and-braces anyway. This plan therefore plans
  **no** nil-`d.ctx` panic handling, no assertion about `d.ctx`, and no production change in that
  area. The doc settled this; revisiting it is out of scope.
- **R4 — Line-number drift on rebase.** All mutation edits above are keyed to line numbers
  measured at `fc84c57d4` (which match the doc's). If the executor rebases, the mutations apply by
  **unique old-text match**, not by line number; every old text above was planner-verified unique
  in its file (`grep -n` the old text before applying; if two hits, stop and re-measure).
- **R5 — `cmd/wasm` red at base (B5, rc=1, planner-confirmed).** No acceptance gate uses
  `go build ./...`; a judge re-running gates must not "discover" this and attribute it to the
  change.
- **R6 — No-network evidence rule.** Every acceptance command in this plan is a local
  `go build`/`go test` over a warm module cache — real evidence. Nothing in this sprint touches a
  socket; if any executor step accidentally requires network (cold module cache), the command's
  verdict on that run is uninformative and must be re-run in a warm environment, not reported
  green/red from a sandbox artifact.
- **R7 — M5 outer-env leakage.** Run B's RED depends on the *outer* `AILANG_CONFIG` being set for
  the `go test` process and on the pin being the only deleted line. The procedure exports the env
  var in the invoking shell rather than via `os.Setenv` in code, so teardown is `unset`-clean and
  cannot leak into other tests or the dev's shell history files.
- **OQ1 — audit-record destination (noted, not decided).** The doc says the M3 audit is "recorded
  in the PR description", but the controller commits and a PR may not exist at M3 time.
  **Default if nobody decides: the audit table goes into the commit body of the M2 commit (or the
  controller's squash body), and nothing is added to the tree.** Coverage consequence of the
  default: **none** — this is record-keeping text, not code; unlike the cited prior incident,
  no production path or default is decided by this item, so deciding-by-default here cannot ship
  an untested behaviour change.
- **OQ2 — exact hunk placement of the interface block within `daemon.go` (noted, not decided).**
  The doc fixes content and name and says "next to the field declaration"; it leaves the precise
  line to the executor. **Default if nobody decides: immediately above the `Daemon` struct
  declaration.** Compile-gated either way; zero coverage consequence.

## Evidence summary for the judge

Baseline (this session, `fc84c57d4`): builds rc=0/0; new-test gate vacuous rc=0
(`[no tests to run]`); shutdown-test gate real rc=0 (`--- PASS: TestExecuteTaskQueueSurvivesClosedStore`);
`go build ./...` rc=1 on `cmd/wasm` only. `d.executor` refs: 5 (matches doc). Mutations: every row
keyed by planner-verified unique old text. M5 is a two-run procedure: run A GREEN, run B RED,
single variable (the pin).
