# M-DAEMON-TASK-EXEC-RUN-UNTESTED — make `executeTask` executable under test

- **Queue item:** `m-daemon-task-exec-run-untested`
- **Scope:** sprint-sized (<= 3 days)
- **Commit under test:** `3ee5bb177ca1bf9fe498d190e37af1f6b8b10f07` (the tree this doc was written in)
- **Status:** planned
- **Date:** 2026-09-08
- **Revision:** 2 (post-quorum). Round 1 was BLOCKED 3/3; this revision answers the three
  evidence/hermeticity objections. See the **Quorum verification log** at the end.

## Problem

The coordinator daemon's task-execution path has **no unit test that executes it**.
`executeTask` — the 736-line function that runs a single task through the executor — is
exercised by nothing. The five `_test.go` files that mention its name either analyse the
source text statically or test the *different* function `executeTaskQueue`. The concrete
consequence, measured by the SonarCloud new-code coverage gate on PR #1111 (2026-09-08): of
33 new lines in that PR, 7 were uncovered and 6 sit in this file — the `ExecuteOptions`
construction at lines 238-241 and the `ExecuteWithRetry` call at line 335. They are
uncovered because **nothing exercises the function**, not because a test was skipped.

This item was deliberately NOT fixed inline by iteration 351, with this reason recorded in
the mission charter: *"covering a daemon path needs a fixture and a decision about how much
of the daemon to stand up, which is a design question, not a line to squeeze in behind a
coverage gate."* This doc makes that decision with evidence.

## Evidence

All facts below were re-measured first-party in this session at commit `3ee5bb177`; the
commands and outputs are in the Verification Log. Rows V26-V32 are new in this revision and
answer the three quorum objections directly.

1. `internal/coordinator/daemon_tasks_exec_run.go` is 736 lines and declares exactly two
   functions: `executeTask` (line 25) and `enrichResolutionEnvelope` (line 681).
2. `executeTask` has exactly **one** production caller: `daemon_tasks_exec.go:83`.
3. **No test executes `executeTask`.** Five `_test.go` files mention the name; every
   mention is either about the different function `executeTaskQueue`, or is static AST
   analysis of the source text rather than a call:
   - `completion_matrix_test.go:60` — `fn.Name.Name == "executeTask"` (go/ast walk)
   - `finalize_parity_test.go:59` — `callsInFunc(t, "daemon_tasks_exec_run.go", "executeTask")`
   - `completion_matrix_test.go:26` — comment: "executeTask, which runs a real agent, so
     they cannot be exercised directly"
   - `daemon_tasks_exec_shutdown_test.go:33` — calls `executeTaskQueue`, not `executeTask`
   - `task_finalize_matrix_test.go:17` — comment about effects living inside `executeTask`
   - `github_webhook_test.go:41` — comment about `executeTaskQueue`
   - **Positive control:** the package contains 104 `_test.go` files, and `ExecuteWithRetry`
     DOES appear in 2 of them (`integration_test.go`, `timer_injection_guard_test.go`), so
     the grep instrument can see a positive.
4. `d.executor` is declared at `daemon.go:88` as a **concrete `*TaskExecutor`**, not an
   interface. `ExecuteWithRetry` is a method on that concrete type
   (`task_executor.go:152`). There is currently **no injection seam** at the executor
   boundary.
5. The `ExecuteOptions` construction (lines 241-246) bases on `DefaultExecuteOptions()`
   and overrides five fields: `Timeout`, `IdleTimeout`, `Workspace`, `ObservatoryContext`,
   `AgentConfig`. The `ExecuteWithRetry` call is at line 335.
6. **Hermeticity is achievable with a shipped mechanism, not a new one.** `AILANG_CONFIG`
   is an existing environment override checked *before* the home directory (C1/V26), and a
   non-existent path makes `LoadBudgetsConfigFrom` return the compiled-in
   `DefaultBudgetsConfig()` (C2/V27). The test pins it to an absent path inside
   `t.TempDir()` (Design §4), so the budget configuration is a property of the binary, not
   of the developer's machine.
7. **The script-agent path is safe by construction, not by prose.** With `d.msgStore ==
   nil`, `targetAgent` is `"coordinator"` unconditionally (C5/V30), and the
   `worktreeMgr.CreateWorktree` deref sits inside the non-script branch, so a nil
   `worktreeMgr` is never dereferenced on the script path (C6/V31).

## Verification Log

Every row: exact command, observed output (quoted, trimmed), date. A zero result is a claim
unless a known-positive control fires in the same breath; the instrument is named next to
every number. **Exit codes are captured with `cmd; rc=$?`** (per C7), never inferred from a
command's stdout.

| # | Claim | Command | Observed output | Date |
|---|-------|---------|-----------------|------|
| V1 | HEAD is the tree under test | `git rev-parse HEAD` | `3ee5bb177ca1bf9fe498d190e37af1f6b8b10f07` | 2026-09-08 |
| V2 | `daemon_tasks_exec_run.go` is 736 lines | `wc -l internal/coordinator/daemon_tasks_exec_run.go` | `736 internal/coordinator/daemon_tasks_exec_run.go` | 2026-09-08 |
| V3 | exactly two funcs, at lines 25 and 681 | `grep -n '^func' internal/coordinator/daemon_tasks_exec_run.go` | `25:func (d *Daemon) executeTask(task *TaskRecord) error {` / `681:func (d *Daemon) enrichResolutionEnvelope(...)` | 2026-09-08 |
| V4 | one production caller | `grep -rn '\.executeTask(' internal/coordinator/*.go \| grep -v _test` | `internal/coordinator/daemon_tasks_exec.go:83: if err := d.executeTask(task); err != nil {` | 2026-09-08 |
| V5 | no test executes `executeTask`; mentions are static/other-func | `grep -n 'executeTask' internal/coordinator/*_test.go` | 7 lines: `completion_matrix_test.go:26/47/60/66/105`, `daemon_tasks_exec_shutdown_test.go:30/33`, `finalize_parity_test.go:59`, `github_webhook_test.go:41`, `task_finalize_matrix_test.go:17` — all static analysis or `executeTaskQueue` | 2026-09-08 |
| V6 | **positive control** for V5: 104 test files, `ExecuteWithRetry` in 2 | `ls internal/coordinator/*_test.go \| wc -l` → `104`; `grep -ln 'ExecuteWithRetry' internal/coordinator/*_test.go` → `integration_test.go`, `timer_injection_guard_test.go` | instrument sees a positive | 2026-09-08 |
| V7 | `d.executor` is concrete `*TaskExecutor` | `grep -n 'executor' internal/coordinator/daemon.go` | `88: executor *TaskExecutor` | 2026-09-08 |
| V8 | `ExecuteWithRetry` is a method on `*TaskExecutor` | `grep -n 'func.*ExecuteWithRetry' internal/coordinator/task_executor.go` | `152:func (te *TaskExecutor) ExecuteWithRetry(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions, maxRetries int) (*ExecuteResult, error) {` | 2026-09-08 |
| V9 | all production references to `d.executor` | `grep -rn 'd\.executor\|\.executor' internal/coordinator/*.go \| grep -v _test` (filtered to field refs) | `daemon.go:88` (decl), `daemon.go:516` (nil check), `daemon_tasks_exec.go:37` (nil check), `daemon_tasks_exec_run.go:335` (call), `daemon_tasks_init.go:274` (assign) | 2026-09-08 |
| V10 | five overrides + base at lines 241-246 | `sed -n '238,247p' internal/coordinator/daemon_tasks_exec_run.go` | `opts := DefaultExecuteOptions()` then `opts.Timeout/IdleTimeout/Workspace/ObservatoryContext/AgentConfig = ...` | 2026-09-08 |
| V11 | `ExecuteWithRetry` call at line 335 | `sed -n '335p' internal/coordinator/daemon_tasks_exec_run.go` | `result, err := d.executor.ExecuteWithRetry(taskCtx, analyzed, opts, 2)` | 2026-09-08 |
| V12 | coordinator package builds clean | `go build ./internal/coordinator/; rc=$?; echo $rc` | exit **rc=0**, no output | 2026-09-08 |
| V13 | main binary builds clean | `go build ./cmd/ailang/; rc=$?; echo $rc` | exit **rc=0**, no output | 2026-09-08 |
| V14 | `go build ./...` baseline (known wasm-only failure) | `go build ./...; rc=$?; echo $rc` | `# github.com/sunholo-data/ailang/cmd/wasm` / `runtime.main_main·f: function main is undeclared in the main package`; **rc=1** | 2026-09-08 |
| V15 | `MockStore` implements the `Store` interface | `grep -n 'func (m \*MockStore)' internal/coordinator/mock_store_test.go` | 50 methods incl. `MarkTaskRunning`, `MarkTaskPendingApproval`, `MarkTaskCompleted`, `MarkTaskFailed`, `CreateApprovalRequest`, `UpdateTaskMetrics`, `GetCostByProvider`, `UpdateApprovalEvaluationByTask` | 2026-09-08 |
| V16 | `MockStore.GetCostByProvider` returns empty map (budget check passes) | `sed -n '407,412p' internal/coordinator/mock_store_test.go` | `return make(map[string]float64), nil` | 2026-09-08 |
| V17 | `NewObservatorySync(nil, logger)` is a no-op backend but still builds `obsContext` | `sed -n '/func (s \*ObservatorySync) SyncTask/,/^}/p' internal/coordinator/observatory_sync.go` | `if s.backend == nil { return nil }`; `SyncAgentAssignment` → `return "", nil`; `GetWorkspaceID` → `return ""` | 2026-09-08 |
| V18 | `ResourceTracker` with pid=0 is a no-op poll | `sed -n '170,180p' internal/coordinator/resource_tracker.go` | `if rt.pid <= 0 { return }` | 2026-09-08 |
| V19 | `DefaultExecuteOptions` values | `sed -n '127,136p' internal/coordinator/provider.go` | `Timeout: 5*time.Minute, DryRun: false, RetryBaseDelay: time.Second, Wait: defaultWait` | 2026-09-08 |
| V20 | `GetEffectiveTimeout`/`GetEffectiveIdleTimeout` defaults | `sed -n '670,690p' internal/coordinator/agent_registry.go` | default `60*time.Minute` / `3*time.Minute`; honours `Timeout`/`IdleTimeout` strings | 2026-09-08 |
| V21 | `GetAgentByID` returns the same pointer (pointer-equality assertion valid) | `sed -n '340,345p' internal/coordinator/agent_registry.go` | `return r.agents[id]` | 2026-09-08 |
| V22 | `Register` requires non-empty `Inbox` | `sed -n '294,310p' internal/coordinator/agent_registry.go` | `if agent.Inbox == "" { return fmt.Errorf("agent inbox is required") }` | 2026-09-08 |
| V23 | `BuildDirectiveFromConfig` with a script agent returns `task.Content` (no panic) | `sed -n '82,120p' internal/coordinator/stage_execution.go` | `case "script": ... return task.Content` | 2026-09-08 |
| V24 | `ResolveModel` with empty `Model`+`Role` returns `("", nil)` (no error) | `sed -n '/func ResolveModelChain/,/^}/p' internal/coordinator/retry_chain.go` | `if agent.Model != "" {...}` / `if agent.Role == "" { return nil, nil }` | 2026-09-08 |
| V25 | `daemon_tasks_exec_shutdown_test.go` constructs `&Daemon{executor: &TaskExecutor{}}` (must still compile after seam) | `sed -n '20,22p' internal/coordinator/daemon_tasks_exec_shutdown_test.go` | `d := &Daemon{ ctx: context.Background(), executor: &TaskExecutor{}, }` | 2026-09-08 |
| V26 | **C1** — `AILANG_CONFIG` is checked before the home dir | `sed -n '14,22p' internal/coordinator/agent_config.go` | `if p := os.Getenv("AILANG_CONFIG"); p != "" { return p }` precedes `os.UserHomeDir()`; the env var short-circuits the home path entirely | 2026-09-08 |
| V27 | **C2** — a non-existent path returns `DefaultBudgetsConfig()` | `sed -n '214,240p' internal/coordinator/agent_config.go` | `if os.IsNotExist(err) { return DefaultBudgetsConfig(), nil }`; separately `if config.Budgets == nil { return DefaultBudgetsConfig(), nil }` | 2026-09-08 |
| V28 | **C3** — `LoadBudgetsConfig` is not cached | `sed -n '205,213p' internal/coordinator/agent_config.go` | `configPath := defaultConfigPath(); ... return LoadBudgetsConfigFrom(configPath)` — no `sync.Once`, no package-level cache var; `os.ReadFile` on every call, so `t.Setenv` takes effect per-call and is undone by its own cleanup | 2026-09-08 |
| V29 | **C4** — `DefaultBudgetsConfig` exact limits | `sed -n '180,203p' internal/coordinator/agent_config.go` | Global `WorkspaceBudget 100.0 / DailyBudget 50.0 / TaskMaxCost 25.0 / WarningThreshold 0.8`; Providers `claude {DailyBudget 30.0, TaskMaxCost 15.0, HardLimit true}`, `gemini {DailyBudget 20.0, TaskMaxCost 10.0, HardLimit false}` | 2026-09-08 |
| V30 | **C5** — agent resolution is deterministic on the fixture | `sed -n '60,76p' internal/coordinator/daemon_tasks_exec_run.go` | `targetAgent := "coordinator"` (line 60); override `if task.ThreadID != "" && d.msgStore != nil` (line 61) cannot be entered with `d.msgStore == nil` regardless of `ThreadID`; `agentConfig = d.agentRegistry.GetAgentByID(targetAgent)` (line 73); `stageToAgentID` fallback (line 76) only when the first lookup returns nil | 2026-09-08 |
| V31 | **C6** — the `worktreeMgr` deref is inside the non-script branch | `sed -n '129,156p' internal/coordinator/daemon_tasks_exec_run.go` | `isScriptAgent :=` (129); `worktreeMgr := d.worktreeManagers[targetAgent]` (132) + `d.worktreeMgr` fallback (134) evaluated *unconditionally*; `if isScriptAgent {` (142); `} else if worktreeMgr != nil {` (149); `worktreeMgr.CreateWorktree` (156) only inside the else-if. Indexing a nil map yields nil (no panic); the nil `worktreeMgr` is never dereferenced on the script path | 2026-09-08 |
| V32 | **M5** — blocking config verified against `checkBudgetBeforeExecution` | `sed -n '104,110p' internal/coordinator/daemon_tasks_exec_run.go` + `sed -n '60,90p' internal/coordinator/daemon_tasks_budget.go` | budget gate at exec_run.go:104-110 returns `nil` *before* the executor call (line 335); blocking requires `dailyLimit > 0 && currentSpend >= dailyLimit && hardLimit`; with `currentSpend=0` (empty MockStore map, V16) **no** config blocks, so M5 must also raise spend to `{"claude": 35.0}` against `daily_budget: 30.0, hard_limit: true` | 2026-09-08 |
| V33 | **oc-glm-5-2 round-2 fix, verbatim** — line 372 is guarded | `sed -n '365,378p' internal/coordinator/daemon_tasks_exec_run.go` | the deref sits inside `if worktree != nil && d.worktreeMgr != nil {` (line 368), i.e. it is guarded on `d.worktreeMgr` ITSELF, independently of `isScriptAgent`; `worktree` is additionally nil on the script path. Two independent guards, so the reviewer's "if either line is NOT guarded" branch does not fire and `d.worktreeMgr` STAYS in the stubbed table | 2026-09-08 |
| V34 | **oc-glm-5-2 round-2 fix, verbatim** — line 640 is guarded | `sed -n '630,648p' internal/coordinator/daemon_tasks_exec_run.go` | identical shape: `if worktree != nil && d.worktreeMgr != nil {` (line 640) in the failed-task cleanup branch. Same two independent guards | 2026-09-08 |
| V35 | **gemini-3-1-pro round-2 premise, MEASURED** — `executeTask` never uses `d.ctx` | `awk 'NR>=25 && NR<=680' internal/coordinator/daemon_tasks_exec_run.go \| grep -n 'd\.ctx'` | two hits, BOTH comments: `// CRITICAL: Use context.Background() instead of d.ctx to avoid trace contamination` and `// NOTE: Intentionally NOT mutating d.ctx`. `taskCtx, span := telemetry.StartSpan(context.Background(), ...)` at line 29. **POSITIVE CONTROL** for the grep: `d.ctx` returns `daemon.go:294,421,429`, so the instrument sees a positive. A nil `d.ctx` therefore cannot panic inside `executeTask` | 2026-09-08 |

**Exit-code re-check (C7):** V12 and V13 were re-run with `cmd; rc=$?` and are **rc=0**
(correct). V14 was re-run the same way and is **rc=1** — the log cell previously read
`(exit 0)` for a failing command; that cell is corrected above. Every other row's exit-code
claim was re-checked the same way; none other asserts a passing exit code for a failing
command.

## Design

### 1. The seam — a narrow interface on the Daemon field

`executeTask` calls exactly one method on `d.executor`: `ExecuteWithRetry`. The smallest
change that makes it injectable is to widen the field's type from the concrete
`*TaskExecutor` to a narrow interface holding just that method.

```go
// taskExecutor is the narrow seam executeTask needs. *TaskExecutor satisfies it.
type taskExecutor interface {
    ExecuteWithRetry(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions, maxRetries int) (*ExecuteResult, error)
}
```

Change `daemon.go:88` from `executor *TaskExecutor` to `executor taskExecutor`.

**Candidates considered and rejected:**

- **A. Narrow interface on the Daemon field (PICKED).** Cost at call sites: **one line** —
  the field type declaration at `daemon.go:88`. The assignment (`daemon_tasks_init.go:274`),
  both nil checks (`daemon.go:516`, `daemon_tasks_exec.go:37`), and the call
  (`daemon_tasks_exec_run.go:335`) all compile unchanged because `*TaskExecutor` satisfies
  the interface. The existing test `daemon_tasks_exec_shutdown_test.go:20`
  (`executor: &TaskExecutor{}`) also still compiles (V25). No production code path changes
  in effect.
- **B. Function-value injection** (`d.executeWithRetry func(...)`). Cost: touches the call
  site at line 335 (rewrite the call to go through the field), requires a production
  initialiser to set the field, and adds a second way to express the same dependency. More
  invasive than A for no benefit.
- **C. Extract option-construction into a smaller testable unit.** Cost: refactors the
  736-line function (a stated non-goal) and still does not exercise the actual
  `ExecuteWithRetry` call wiring — the very lines the coverage gate flagged.

**Call-site measurement (V9):** exactly 5 production references to `d.executor` exist
(declaration, 2 nil checks, 1 call, 1 assignment). Only the declaration changes. This is
the measured, not guessed, cost.

**No production behaviour change:** the interface is satisfied by the identical
`*TaskExecutor`; every production default and every production code path is byte-identical
in effect. The seam is purely a widening of a field's static type.

### 2. How much daemon — stood up vs stubbed

By reading `executeTask` (lines 25-673), every collaborator it touches, with whether a nil
is already tolerated (checked, not assumed):

**Stood up (must be non-nil — the code dereferences them unconditionally):**

| Collaborator | Where used | Why it must be stood up | Fixture |
|---|---|---|---|
| `d.logger` | line 40 `d.logger.Printf` | not nil-guarded | `log.New(io.Discard, "", 0)` |
| `d.taskStore` | lines 48, 359, 501/521, 604, 645 | not nil-guarded (`MarkTaskRunning` at line 48 is the first deref) | `NewMockStore()` (V15) |
| `d.resourceRegistry` | lines 319, 340 `Register`/`Unregister` | not nil-guarded | `NewResourceTrackerRegistry()` |
| `d.executor` | line 335 `ExecuteWithRetry` | the seam under test | the capturing fake |
| `d.agentRegistry` | lines 68, 548 | nil-guarded, but **required** to reach the script-agent path (see below) | `NewAgentRegistry()` + one script agent |
| `d.ctx` | **not referenced by `executeTask`** — see V35 | Supplied anyway, per `gemini-3-1-pro`'s round-2 fix and the V25 precedent. The reviewer's stated failure mode (a nil parent context panicking `context.WithTimeout`) is REFUTED by V35: `taskCtx` derives from `context.Background()` at line 29, and the only two occurrences of `d.ctx` in the whole function are comments saying it is deliberately NOT used. Setting it costs one line and removes the question. | `ctx: context.Background()` |

**Stubbed (nil is already tolerated by the code — checked, not assumed):**

| Collaborator | Where used | Nil tolerance (verified) |
|---|---|---|
| `d.msgStore` | lines 60, 78, 430, 493, 620 | every use is behind `d.msgStore != nil` (or `task.ThreadID == ""` short-circuits `postTaskStatus`/`postTaskResult`) |
| `d.worktreeManagers` | line 132 | nil map → `nil` → falls back to `d.worktreeMgr`; script-agent path never reaches it (C6/V31) |
| `d.worktreeMgr` | lines 156, 372, 640 | only reached for non-script agents; script-agent path (`isScriptAgent`) skips worktree creation entirely (C6/V31) |
| `d.observatorySync` | lines 200-215 | nil-guarded (`d.observatorySync != nil`); we supply `NewObservatorySync(nil, logger)` so `obsContext` is still built (V17) |
| `d.coordConfig` | line 207 | nil-guarded (`d.coordConfig != nil`) |
| `d.eventBroadcaster` | lines 300-315 | nil-guarded (`d.eventBroadcaster != nil`) |
| `d.obsBackend` | via chain helpers (lines 52-53, 505-530, 656-662) | every helper returns early on `d.obsBackend == nil` |
| `d.pubsubPublisher` | line 625 | `publishInboxNotification` returns on `d.pubsubPublisher == nil` |
| `d.taskChain` | line 534 | `ProcessStageCompletion` returns on `d.taskChain == nil` |
| `d.approvalCheckpoint`, `d.githubPoster`, `d.approvalWatcher`, `d.taskChain` | — | not referenced by `executeTask` at all |

**The script-agent path is the key to "how much daemon".** `executeTask` creates a git
worktree for AI agents (`worktreeMgr.CreateWorktree`, line 156) — a real filesystem/git
operation a unit test must not perform. The code already has a branch that avoids it:
`isScriptAgent := agentConfig != nil && agentConfig.Invoke != nil && agentConfig.Invoke.Type == "script"`
(line 129). A script agent uses `task.Workspace` directly (line 144) and never touches
`worktreeMgr`. So the fixture registers one script agent with ID `"coordinator"` (the
default `targetAgent` when `ThreadID == ""` and `msgStore == nil` — C5/V30), which:
- avoids worktree creation (no `worktreeMgr` needed),
- keeps `worktreePath == task.Workspace` (non-empty, but `enrichResolutionEnvelope` is
  skipped because `d.msgStore == nil`),
- keeps `opts.Workspace` assertable to `task.Workspace`.

**Why the script path is safe by construction, not by prose (O3).** The safety does not
come from the `worktreeMgr` assignment being skipped — it is evaluated *unconditionally* at
lines 132-134, before the `if isScriptAgent` branch at line 142. The safety comes from the
**deref** being inside the branch: `worktreeMgr.CreateWorktree` (line 156) sits inside
`} else if worktreeMgr != nil {` (line 149), which is only reached when `isScriptAgent` is
false. Indexing a nil map in Go returns the zero value (nil) rather than panicking, so on
the script path `worktreeMgr` is simply nil and is never dereferenced (C6/V31). This is the
distinction O3 said the previous revision failed to prove; it is now a verification-log row.

**Budget check is not a blocker — and is hermetic by construction (O1/O2).**
`checkBudgetBeforeExecution` (line 104) calls `LoadBudgetsConfig()`. The test pins
`AILANG_CONFIG` to a non-existent path inside `t.TempDir()` (Design §4), so
`LoadBudgetsConfigFrom` returns `DefaultBudgetsConfig()` (C2/V27) — provider "claude":
DailyBudget=30, HardLimit=true (C4/V29) — with **no dependence on the developer's home**.
Then `d.taskStore.GetCostByProvider()` returns an empty map (V16), so `currentSpend=0 < 30`
and the task proceeds. No budget approval is created. The test **asserts** the executor was
reached (`cap.opts != nil`), so a future change to the compiled-in defaults fails loudly
and legibly instead of silently skipping the call.

**Resource tracker is safe.** `NewResourceTracker(task.ID, task.ThreadID, 0)` uses pid=0;
`pollMetrics` returns immediately on `pid <= 0` (V18). `Start`/`Stop` are clean.

**The line between stood up and stubbed:** stood up = the 5 collaborators above; stubbed =
everything else (nil-guarded or unreachable on the script-agent path). This is the
justified minimum — no real executor, no real store, no real worktree, no real observatory.

### 3. What the first test asserts

The minimum bar: a test that **executes** `executeTask` and observes the `ExecuteOptions`
actually handed to `ExecuteWithRetry`, pinning the five overridden fields **and** that the
base is `DefaultExecuteOptions()`.

The capturing fake:

```go
type capturingExecutor struct {
    opts *ExecuteOptions
    task *AnalyzedTask
}

func (c *capturingExecutor) ExecuteWithRetry(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions, maxRetries int) (*ExecuteResult, error) {
    c.opts = opts
    c.task = task
    return &ExecuteResult{Success: true, Provider: "script", SessionID: "sess-1"}, nil
}
```

The fixture (script agent, distinctive timeouts so a hardcoded default would differ):

```go
func TestExecuteTask_HandsDefaultBasedOptionsToExecutor(t *testing.T) {
    // REQUIRED first line — hermeticity by construction (Design §4, C1/C2).
    // A non-existent path inside t.TempDir() pins the budget configuration to the
    // compiled-in DefaultBudgetsConfig() with no file on disk and no dependence on the
    // developer's ~/.ailang/config.yaml. t.Setenv's own cleanup undoes it (C3/V28).
    t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))

    reg := NewAgentRegistry()
    scriptAgent := &AgentConfig{
        ID: "coordinator", Inbox: "coordinator",
        Invoke: &InvokeConfig{Type: "script", Command: "true"},
        Timeout: "15m", IdleTimeout: "90s", SkipApproval: true,
    }
    reg.Register(scriptAgent) // V22: Inbox required

    logger := log.New(io.Discard, "", 0)
    cap := &capturingExecutor{}
    d := &Daemon{
        ctx:    context.Background(), // gemini-3-1-pro round-2 fix, verbatim; V25 precedent
        logger: logger, taskStore: NewMockStore(),
        resourceRegistry: NewResourceTrackerRegistry(),
        agentRegistry: reg, observatorySync: NewObservatorySync(nil, logger),
        executor: cap,
    }
    task := &TaskRecord{ID: "task-exec-1", Type: TaskTypeFeature, Stage: TaskStageImplementation,
        Title: "test task", Kind: "feature", Workspace: "/tmp/ailang-test-workspace", Iteration: 1}

    if err := d.executeTask(task); err != nil {
        t.Fatalf("executeTask returned error: %v", err)
    }
    if cap.opts == nil {
        // Naming the budget gate as the likely cause: if a future change to the
        // compiled-in defaults (or a removed AILANG_CONFIG pin) blocks the task before
        // the executor call, this is where it fails — loudly and legibly.
        t.Fatalf("ExecuteWithRetry was never called (cap.opts == nil); check the budget gate")
    }
    // ... six assertions below ...
}
```

Assertions (base first, then the five overrides):

1. **Base is `DefaultExecuteOptions()`** — the three fields NOT overridden:
   `cap.opts.RetryBaseDelay == DefaultExecuteOptions().RetryBaseDelay` (1s),
   `cap.opts.Wait != nil`, `cap.opts.DryRun == DefaultExecuteOptions().DryRun` (false).
2. `cap.opts.Timeout == 15*time.Minute` (the `GetEffectiveTimeout` override).
3. `cap.opts.IdleTimeout == 90*time.Second` (the `GetEffectiveIdleTimeout` override).
4. `cap.opts.Workspace == task.Workspace` (script agent uses workspace directly).
5. `cap.opts.ObservatoryContext != nil && cap.opts.ObservatoryContext.TaskID == task.ID`
   (the `obsContext` override, built via `NewObservatorySync(nil, logger)`).
6. `cap.opts.AgentConfig == scriptAgent` (pointer equality — V21).

### 4. Hermeticity by construction (the design change this revision makes)

The previous revision deferred budget-config isolation to a "follow-up": the test did not
set `AILANG_CONFIG`, so it read the developer's real `~/.ailang/config.yaml`, and a hard
budget already exceeded on that machine could block the task and fail the test for
environmental reasons. All three reviewers rejected that framing. This revision removes it
entirely and makes the test hermetic **by construction**:

- **The mechanism already exists — no new helper is invented.** `AILANG_CONFIG` is a
  shipped environment override checked *before* the home directory (C1/V26). astra's
  "reuse an existing configuration-isolation helper if one exists" is satisfied by this
  env var; there is no need for a second mechanism.
- **A non-existent path is the strongest hermetic form.** `LoadBudgetsConfigFrom` returns
  `DefaultBudgetsConfig(), nil` on `os.IsNotExist(err)`, and separately returns the same
  when the file parses but `config.Budgets == nil` (C2/V27). So
  `t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))` pins the budget
  configuration to the compiled-in defaults with **no file on disk** and **no dependence on
  the developer's home**.
- **Why a non-existent path beats a written YAML fixture.** A written YAML fixture is a
  moving part that can drift with the schema and must be kept in sync with
  `BudgetsConfig`/`ProviderLimit` yaml tags; an absent path has zero moving parts and
  cannot drift. It also exercises the exact `os.IsNotExist` branch that a developer with no
  config file hits in production, so the test's budget behaviour matches the common real
  deployment.
- **No cross-test leakage.** `LoadBudgetsConfig` is not cached (C3/V28): it calls
  `defaultConfigPath()` and `os.ReadFile` on every call, so `t.Setenv` takes effect for
  that call and is undone by `t.Setenv`'s own cleanup.
- **The defaults are asserted, not assumed.** The test asserts `cap.opts != nil` with a
  failure message naming the budget gate as the likely cause (Design §3). If a future
  change to `DefaultBudgetsConfig` (C4/V29) makes the gate block, the test fails loudly and
  legibly instead of silently skipping the executor call.

**Mutations that must turn it RED** (a test that cannot be killed by a mutation is not
coverage):

- **M1 — drop the base:** change `opts := DefaultExecuteOptions()` to `opts := &ExecuteOptions{}`.
  Then `RetryBaseDelay` becomes 0 and `Wait` becomes nil → assertions 1 fail.
- **M2 — hardcode an override:** change `opts.Timeout = agentConfig.GetEffectiveTimeout()`
  to `opts.Timeout = 5*time.Minute`. Then assertion 2 fails (5m != 15m). The distinctive
  `15m` fixture is what makes this mutation visible.
- **M3 — drop an override:** delete `opts.AgentConfig = agentConfig`. Then assertion 6 fails
  (`nil != scriptAgent`).
- **M4 — stop calling the executor:** replace the `ExecuteWithRetry` call with a no-op.
  Then `cap.opts == nil` and the test fails with "ExecuteWithRetry was never called".
- **M5 — the controlled isolation test (replaces the round-2 M5; `gpt6-astra`'s fix applied
  verbatim).** The previous M5 was **confounded** and astra is right about why: it removed the
  fixture's `AILANG_CONFIG` pin *and* raised MockStore spend in the same step, and its hostile
  YAML re-used `DefaultBudgetsConfig`'s own `claude` limits (daily 30, hard) — so RED could not
  distinguish "the pin was removed" from "the spend mutation alone was sufficient". The
  replacement varies exactly ONE thing, the fixture's pin:
  - **Held constant in both runs:** a test-local store wrapper returning a fixed `claude` spend
    of **5.0** (a wrapper, NOT a mutation of the shared `MockStore` — nothing about V16's
    behaviour changes for any other test), and an **outer** `AILANG_CONFIG` pointing at a
    temporary YAML with `budgets.providers.claude.daily_budget: 1.0` and `hard_limit: true`.
  - **Run A (unchanged fixture):** the fixture's own
    `t.Setenv("AILANG_CONFIG", <absent path in t.TempDir()>)` overrides the outer value, so the
    compiled-in `DefaultBudgetsConfig` applies: `dailyLimit=30 > currentSpend=5` → NOT blocked →
    the executor IS reached (`cap.opts != nil`).
  - **Run B (delete ONLY the fixture's pin):** the outer hostile config applies:
    `dailyLimit=1`, `currentSpend=5 >= 1`, `hardLimit=true` → blocked → `executeTask` returns
    `nil` at line 110, **before** the executor call at line 335 → `cap.opts == nil` → RED with
    "ExecuteWithRetry was never called".
  - Because spend and the hostile file are byte-identical across A and B, the only variable is
    the pin, so RED in B is attributable to it. **Record BOTH outcomes**, and before claiming the
    guard is load-bearing, verify that `checkBudgetBeforeExecution` actually selected the
    `claude` provider on this fixture (the agent carries no `Provider`, so the default at
    `daemon_tasks_budget.go:22` applies — assert it rather than assume it).
  - The 30-vs-1 asymmetry is what makes the two runs discriminate; a hostile config re-using the
    compiled-in limits cannot, which is exactly the defect being fixed.

## Milestones

Each milestone is independently committable; each has a runnable acceptance command and the
mutation that must kill its test.

**M1 — introduce the seam (interface + field type).**
- Change `daemon.go:88` to `executor taskExecutor`; add the `taskExecutor` interface.
- Acceptance: `go build ./internal/coordinator/ && go build ./cmd/ailang/` (both exit 0).
- Mutation that must kill it: revert the field type to `*TaskExecutor` → the new test file
  (M2) would not compile because the fake does not satisfy `*TaskExecutor`. (M1 lands before
  M2, so the compile gate is the guard.)

**M2 — the first executing test.**
- Add `internal/coordinator/daemon_tasks_exec_run_test.go` with the fixture, the capturing
  fake, the six assertions above, and the **required first line**
  `t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))` (Design §3/§4).
- Acceptance: `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1`
  passes.
- Mutations that must kill it: M1-M5 above (each independently turns it RED).

**M3 — mutation audit (optional hardening).**
- Run the M1-M5 mutations one at a time and confirm each turns the test RED, then revert.
- Acceptance: `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1`
  still passes after each revert; the mutation run is recorded in the PR description.
- Mutation that must kill it: any of M1-M5 left un-reverted.

## Acceptance

Baseline on the **pristine tree** (recorded in the Verification Log): `go build
./internal/coordinator/` exit 0 (V12), `go build ./cmd/ailang/` exit 0 (V13), `go build
./...` fails only on the known `cmd/wasm` main-package issue (V14, rc=1). A gate already red
at base measures the repo, not this change.

1. `go build ./internal/coordinator/` → exit 0.
2. `go build ./cmd/ailang/` → exit 0.
3. `go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1` → PASS.
4. `go test ./internal/coordinator/ -run 'TestExecuteTaskQueueSurvivesClosedStore|TestExecuteTask_HandsDefaultBasedOptionsToExecutor' -count=1` → PASS (the existing shutdown test still compiles and passes after the seam — V25).
5. Mutation audit (M3): each of M1-M5 turns the new test RED.

## Conflict Surface

Measured, not speculated:

- **`d.executor` field** — 5 production references (V9): declaration (`daemon.go:88`), two
  nil checks (`daemon.go:516`, `daemon_tasks_exec.go:37`), one call
  (`daemon_tasks_exec_run.go:335`), one assignment (`daemon_tasks_init.go:274`). Only the
  declaration changes. A rebase that adds a new method call on `d.executor` would need that
  method added to the interface — the interface is deliberately narrow, so any such addition
  is a compile error that surfaces immediately, not a silent break.
- **`ExecuteWithRetry`** — production callers: only `daemon_tasks_exec_run.go:335` (the
  other non-test hits are comments and the definition). Test callers:
  `timer_injection_guard_test.go:56,236`, `integration_test.go:246` — these call it on a
  `*TaskExecutor` directly and are unaffected by the interface.
- **`DefaultExecuteOptions`** — production callers: `daemon_tasks_exec_run.go:241`,
  `provider_executor.go:67`, `task_executor.go:82,154`, `provider_gemini.go:61`. This change
  touches none of them. Test callers: `timer_injection_guard_test.go:51,224,273,276` — the
  new test reuses the same symbol, so a change to `DefaultExecuteOptions` would be caught by
  both the new test and the existing `timer_injection_guard_test.go:273-277`.
- **`MockStore`** — shared test helper (`mock_store_test.go`). The new test only *reads*
  it; no method is added or changed. No conflict. (M5 mutates `GetCostByProvider` only as a
  throwaway mutation-audit step, reverted immediately.)
- **`daemon_tasks_exec_shutdown_test.go:20`** — constructs `&Daemon{executor: &TaskExecutor{}}`.
  `*TaskExecutor` satisfies the interface, so this compiles unchanged (V25). A rebase that
  changes `TaskExecutor`'s `ExecuteWithRetry` signature would break both this doc's test and
  the existing `timer_injection_guard_test.go` — a compile-time, not silent, failure.

## Non-Goals

- **Not** testing the whole result-handling/finalize path (the `result.Success` branches,
  approval creation, handoffs, `enrichResolutionEnvelope`). The first test stops at the
  `ExecuteWithRetry` call and asserts the options; the post-call branches run only to prove
  they do not panic on the fixture.
- **Not** changing production behaviour. The seam is a field-type widening; every production
  default and code path is byte-identical in effect.
- **Not** restructuring the 736-line function. No extraction, no reordering, no signature
  change to `executeTask`.
- **Not** adding a real executor, real store, real worktree, or real observatory to the
  fixture. The script-agent path and nil-guards keep the fixture in-memory.
- **Not** covering the cloud-dispatch path (`dispatchTasksCloud`) or `executeTaskQueue`
  itself (already covered by `daemon_tasks_exec_shutdown_test.go`).
- **Not** introducing a new configuration-isolation mechanism. The test reuses the shipped
  `AILANG_CONFIG` env override (C1/V26); no new helper is added.

## Risks

- **Interface drift:** if a future change adds a second method call on `d.executor`, the
  narrow interface forces a compile error until the method is added — a feature, not a bug,
  but it means the interface must be kept in sync deliberately. Mitigation: the interface
  lives next to the field declaration with a comment naming its single production
  implementor.
- **Fixture brittleness:** the test depends on the script-agent path and on
  `NewObservatorySync(nil, logger)` producing a non-nil `obsContext`. If `executeTask` is
  refactored to require a worktree even for script agents, the fixture must grow a
  `worktreeMgr`. Mitigation: the test's failure message names the assumption
  ("ExecuteWithRetry was never called") so a refactor surfaces loudly.
- **Compiled-in default drift:** if a future change to `DefaultBudgetsConfig` (C4/V29)
  makes the budget gate block under the pinned defaults, the test would fail. Mitigation:
  the test asserts `cap.opts != nil` with a failure message naming the budget gate as the
  likely cause, so a default change fails loudly and legibly instead of silently skipping
  the call. This is the intended behaviour of the hermeticity guard, not a residual
  environmental risk.
- **`go build ./...` is red at base** on `cmd/wasm` (V14, rc=1). Acceptance uses the two
  targeted builds, not `./...`, so this known-red gate does not mask the change.

## Quorum verification log

Round 1 verdict: **BLOCKED** — 3/3 reviewers rejected (reject-by-default). None disputed the
design direction (the narrow `taskExecutor` seam and the script-agent fixture); all three
objections were about **evidence** and **hermeticity**. This revision answers each with a
verification-log row and a design change.

| Reviewer | Objection (one line) | This revision's response |
|---|---|---|
| `gemini-3-1-pro` | The unit test relies on the absence of a local `~/.ailang/config.yaml` to pass; environmental isolation must be built in, not deferred to a follow-up. | Design §4 makes the test hermetic **by construction**: `t.Setenv("AILANG_CONFIG", <absent path in t.TempDir()>)` is a required first line (Design §3/§4 and M2), backed by C1/V26 and C2/V27. All "follow-up" isolation framing is deleted. |
| `gpt6-astra` | `executeTask` reads the real budget config; an empty MockStore spend map does not prove execution under arbitrary limits; verify `LoadBudgetsConfig` path resolution, env override, caching, and disable semantics; reuse an existing isolation helper. | C1-C4/V26-V29 verify `AILANG_CONFIG` is a real shipped mechanism (checked before home dir), a non-existent path → `DefaultBudgetsConfig`, no caching, and the exact default limits. §4 asserts the executor was reached (`cap.opts != nil`) so a default change fails loudly. M5 proves the guard is load-bearing. The existing `AILANG_CONFIG` env var is the reused helper — no new mechanism. |
| `oc-glm-5-2` | The script-agent path's safety (no worktree deref) and the agent-resolution path are asserted in prose, not verified; if resolution differs, the fixture hits a real git operation. | C5/V30 and C6/V31 add verification-log rows with commands and line numbers proving `targetAgent` is `"coordinator"` unconditionally when `msgStore==nil` (lines 60-76), and that the `worktreeMgr.CreateWorktree` deref sits inside the non-script branch (lines 129-156), so a nil `worktreeMgr` is never dereferenced on the script path. |

### Round 2 — BLOCKED again (3/3 reject, **zero absentees**), closed under the narrow-refinement carve-out

Round 2 verdict: **BLOCKED**, `gpt6-astra` / `gemini-3-1-pro` / `oc-glm-5-2` all reject,
`absent_reviewers` = `[]` (cross-checked against `[.reviewers[] | select(.present==false)]`,
also empty). Again **none disputes the design direction**; all three objections are
completeness/determinism defects and all three carry a concrete, reviewer-authored
`proposed_fix`. The controller therefore closed the doc under the mission protocol's
narrow-refinement carve-out: a bounded second revision applying the reviewers' **verbatim**
fixes, with no controller-invented resolution and no objection overridden.

**Round-1 note:** `gpt6-astra` was ABSENT on budget in round 1 (cap $0.10). It was re-run
alone at a raised cap ($0.35) rather than accepting the N-1 reading, and it rejected — on the
same hermeticity surface as `gemini-3-1-pro`, which is why revision 1 treated that as the
dominant objection.

| Reviewer | Round-2 objection | Fix applied, verbatim | Controller's own measurement |
|---|---|---|---|
| `gpt6-astra` | M5's hostile config re-used `DefaultBudgetsConfig`'s own claude limits, so raising spend blocks with the pin retained — RED cannot distinguish pin-removal from the spend mutation. | M5 replaced with the reviewer's controlled isolation test: a test-local store wrapper at fixed claude spend 5.0 and an **outer** `AILANG_CONFIG` at `daily_budget: 1.0, hard_limit: true`, both held constant; only the fixture's own pin is deleted between run A and run B. | CONFIRMED — the round-2 M5 was genuinely two-variable. This is the one round-2 objection that required a design change rather than evidence. |
| `gemini-3-1-pro` | The fixture omits `ctx`; if `taskCtx` derives from `d.ctx`, a nil parent panics. | `d.ctx` added to the Stood-up table and `ctx: context.Background()` added to the fixture (V25 precedent), exactly as proposed. | **The premise is REFUTED** (V35): `executeTask` never uses `d.ctx` — `taskCtx` comes from `context.Background()` at line 29 and the only two `d.ctx` occurrences in the function are comments saying it is deliberately unused (positive control: `d.ctx` at `daemon.go:294,421,429`). The fix is applied anyway because it costs one line and removes the question; the doc records the measurement rather than laundering the premise. |
| `oc-glm-5-2` | Only line 156 was traced; lines 372 and 640 could deref a nil `worktreeMgr` outside the `isScriptAgent` guard. | V33 and V34 added with the reviewer's exact `sed` ranges. | Both lines sit inside `if worktree != nil && d.worktreeMgr != nil {` — guarded on `d.worktreeMgr` ITSELF, independently of `isScriptAgent`, with `worktree` nil on the script path as a second guard. The reviewer's "if either is NOT guarded" branch does not fire, so `d.worktreeMgr` stays stubbed. The doc's safety claim is now **stronger** than it was: the guards, not the branch, are what make a nil manager safe. |

**Carve-out conditions checked before use:** (a) every remaining blocking objection carries a
concrete reviewer-authored `proposed_fix` — verified by reading
`.reviewers[].result.proposed_fix` in the round-2 artifact, all three non-empty; (b) none
disputes the design direction; (c) zero absentees, so no verdict is missing. This is the
second revision and the carve-out's ceiling: any further block parks for a human.
