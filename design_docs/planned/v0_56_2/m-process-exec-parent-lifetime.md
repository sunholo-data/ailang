# M-PROCESS-EXEC-PARENT-LIFETIME: host shutdown reaches the synchronous `Process.exec` child

**Status**: Planned (awaiting human design approval — no implementation, sprint planning, release or deployment is authorized by this document)
**Target**: v0.56.2
**Priority**: P1 (issue #2034, labeled `priority:P1 area:effects`)
**Estimated**: ~2–3 days (≈120 LOC + tests in `internal/effects`, one probe in `internal/proctree`)
**Dependencies**: None (runlife and WorkerOwner machinery already shipped in v0.54.5 / earlier)
**Author**: design-doc-creator (triage iteration25, issue #2034), 2026-11-03

---

## Problem Statement

`std/process.exec` — the ordinary synchronous builtin, `processExec` in
`internal/effects/process.go` — is the **only** process-spawning path in the
runtime that is invisible to every host shutdown mechanism:

1. **Background-only deadline.** `internal/effects/process.go:159` builds the
   child's context as `context.WithTimeout(context.Background(), pc.Timeout)`.
   Nothing the host can do — signal, inherited owner stop, policy admission —
   can reach that context.
2. **No worker registration.** The managed path (`process_managed.go:73`) and
   the async/stream path (`stream_process.go:67`) register an `OwnedWorker` in
   the execution `WorkerOwner` (via `ctx.BeginWorker()`, `worker_owner.go:123`).
   `processExec` registers nothing, so `CloseWorkers()` cannot see it.
3. **No runtime lifetime.** The managed/stream paths call
   `runlife.Prepare(cmd, nil)` so a nested same-runtime `ailang run` child gets
   the inherited stop-request/cleanup-receipt channel. `processExec` never
   calls `Prepare`.

The signal path already exists and already fires: `ailang run` installs
`effCtx.InstallWorkerSignalHandler(...)` (`internal/runner/run.go:645`), and on
SIGINT/SIGTERM it runs `runlife.Complete(effCtx.CloseWorkers())` and exits with
130/143. But `CloseWorkers()` only stops *registered* workers — so the ordinary
exec child is orphaned with no deadline.

**Measured impact** (issue #2034 repro, `ailang v0.54.14`/`v0.55.0`, Linux
arm64; runtime exits *while* `exec("bash", ["-c","sleep 3; touch
late_marker"])` is in flight, one signal to the `ailang` pid only):

| Signal | ailang exit | Child group alive after | `late_marker` written |
|---|---|---|---|
| SIGTERM | 143 | yes | yes |
| SIGINT | 130 | yes | yes |
| SIGKILL | 137 | yes | yes |
| (control, no signal) | — | no (`Err(Timeout(1010))`) | no |

Who is affected: **motoko_agent**. Its TUI cancels a running tool call by
sending SIGTERM to the AILANG runtime (`src/tui/src/runtime-process.ts`), so a
cancelled `BashExec` keeps changing files with no limit. This is the exact
production path that produced the 2026-10-16 ~18 GB runaway (`cd /tmp/qtest &&
python3 gen.py && …` timed out, `python3` reparented to launchd) — that incident
fixed the *timeout* reach (group kill, `ConfigureContained`), but the
*parent-death* reach is still missing for the ordinary synchronous path.

Related but **separate** concerns (see Non-Goals): #2037 (setsid/daemon
descendants escaping the group), #2035 (per-call timeout override), #2036
(configured-timeout query), and SIGKILL-of-the-runtime containment.

## Goals

**Primary Goal:** When the AILANG runtime shuts down on SIGTERM/SIGINT (host
signal) or an inherited owner stop request, the in-flight ordinary
`std/process.exec` child is terminated with the runtime's existing ownership
mechanisms — no new kill path, no new signal subscription, policy-owned
process groups preserved.

**Success Metrics:**
1. Issue #2034 repro: SIGTERM or SIGINT to the runtime during an in-flight
   `exec` leaves no `late_marker` and no surviving child-group members (POSIX).
2. Runtime exit codes stay exactly 143 (SIGTERM) / 130 (SIGINT) — motoko's
   cancellation contract is unchanged.
3. `--policy` owned runs: the exec child still stays in the owner's group and
   the supervisor's group kill still contains it; the runtime never
   group-signals the owner's group from inside.
4. A nested `ailang run` child spawned via ordinary `exec` now participates in
   the inherited lifetime protocol (cooperative stop + terminal cleanup
   receipt), matching the managed/stream paths.
5. All existing process tests, runtime-lifetime tests and policy tests pass
   unchanged (no Result-semantics change).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Reach the exec child through the **existing** `WorkerOwner` admission + `OwnedWorker` registration, so `CloseWorkers()` (signal handler and inherited `runlife.Bind` cleanup) stops it — no second signal path | One ownership registry is the security invariant; a parallel "signal-time kill list" would be a second authority surface with its own races | agent (design), human ratifies | design | med |
| D2: Derive the exec timeout context from `admission.Context()` instead of `context.Background()` | Makes host cancellation and the per-call timeout share the one `exec.CommandContext` cancel path already wired by `ConfigureContained` | agent | design | low |
| D3: Call `runlife.Prepare(cmd, nil)` in `processExec` (same-binary `ailang run` gate unchanged); cooperative stop via `lifetime.RequestStop` **only on host shutdown** — the per-call timeout stays a hard group kill | Cooperative timeout would change `Err(Timeout)` latency semantics; host shutdown is where receipts matter | agent (design), human ratifies | design | med |
| D4: Shutdown force-kill is scope-selected: process-group kill in standalone mode, **single-process kill under `StayInOwnerGroup`** — add a one-line probe `proctree.InOwnerGroup()` | Wrong choice here group-signals the policy supervisor's group (regression of the 7ae77f35 containment fix) | agent | design (freeze) | med |
| D5: Late admission (owner already closing when exec starts) and failed/missing lifetime receipts surface through the **existing** `WorkerCleanupFailure` path — no new `ProcessError` constructor, no change to `Result` semantics | Keeps `Ok`-for-nonzero / `Err`-for-infra intact; host cleanup failures are host telemetry, not language values | agent | design | low |
| D6: Explicitly do **not** promise SIGKILL-of-runtime cleanup; document it as out of scope next to #2037 | Kernel parent-death mechanisms (Pdeathsig/cgroup/subreaper) are a separate, portability-sensitive policy decision (D-TRIAGE-39 lane) | human | design (freeze) | low |

### Design Freeze

Before implementation begins, these must be resolved (checked = frozen):

- [ ] D1 ratified: exec registers as an `OwnedWorker` in the execution owner (no parallel kill list).
- [ ] D3 ratified: `runlife.Prepare` added to `processExec`; timeout remains a hard group kill; cooperative stop applies only to host shutdown.
- [ ] D4 frozen: force-kill scope selection (`InOwnerGroup` → `KillProcess`, else `KillGroup`); policy-owned groups never group-signaled from inside the runtime.
- [ ] D6 frozen: SIGKILL-of-runtime / crash / power-loss cleanup is documented as NOT guaranteed for ordinary runs (only policy-owned groups and registered nested runtimes have containment).

## Solution Design

### Overview

Make the synchronous exec visible to the two shutdown authorities that
already own process cleanup, by reusing exactly the machinery the managed and
stream paths use:

- **Host signal (SIGTERM/SIGINT)** → `InstallWorkerSignalHandler` →
  `HandleWorkerSignal`/`CloseWorkers` → `WorkerOwner.close()` → registered
  workers' `RequestStop`/`Join`. Today this list cannot contain the exec child;
  after the change it does.
- **Inherited owner stop** (policy supervisor / nested runtime stop request) →
  `runlife.Session.stop` → `Bind(ShutdownWorkersContext)` →
  `CloseWorkersContext` (same registry) and, for same-runtime children, the
  private `runlife` stop-request/receipt channel.

No new signal subscription, no new kill primitive, no new registry. The child's
process-group placement (`ConfigureContained`) is untouched.

### Architecture

**New helper (file: `internal/effects/process_exec_worker.go`, ~80 LOC): a
`syncExecWorker` implementing `OwnedWorker`:**

```
RequestStop()        once: if lifetime != nil → lifetime.RequestStop(bounded, force)
                            else → cancel execCtx (ConfigureContained's cmd.Cancel does the kill)
Join(ctx)           wait for runDone (closed after Wait + receipt join), bounded by ctx
Done()              runDone
StdinClosing()      false
```

`force` is D4's scope-selected kill, `sync.Once`-wrapped like the managed
path's `kill`: `proctree.KillGroup(pid)` in standalone mode,
`proctree.KillProcess(pid)` under `proctree.InOwnerGroup()`. When `lifetime !=
nil`, plain context cancellation must NOT also fire (it would trigger
`cmd.Cancel`'s hard kill and defeat the cooperative stop) — mirror
`managedProcess.RequestStopContext`'s either/or structure exactly.

**Changes to `processExec` (`internal/effects/process.go`), in order:**

1. After the Step-1/1b authorization and confinement, before building the
   command: `admission, err := ctx.BeginWorker()`. On error (owner shutting
   down / parent ctx already cancelled) return
   `makeProcessResultErr("SpawnFailed", detail)` — the existing infra-failure
   constructor; no new error code (D5, no MOD/EFF namespace allocation
   needed).
2. `execCtx, cancel := context.WithTimeout(admission.Context(), pc.Timeout)`
   (replaces `:159`'s `Background()`). `defer cancel()` unchanged.
3. `proctree.ConfigureContained(cmd)` — **unchanged** (owns group placement and
   the ctx-cancel kill semantics).
4. `lifetime, err := runlife.Prepare(cmd, nil)` before Start (D3). On error →
   `admission.Abort()`, return `Err(SpawnFailed, ...)`. For arbitrary commands
   (`bash`, `python3`) `Prepare` returns `(nil, nil)` by the same-binary
   `os.SameFile` + `Args[1] == "run"` gate — behavior for non-runtime children
   is unchanged.
5. Replace `cmd.Run()` with `cmd.Start()` + registration + `cmd.Wait()`:
   - `Start` fails → `lifetime.Abort()`, `admission.Abort()`, existing
     NotFound/PermissionDenied/SpawnFailed mapping unchanged.
   - `lifetime.AfterStart()` when `lifetime != nil`.
   - `admission.Complete(&worker)` registers the child with the owner. If the
     owner is already closing, `Complete` itself stop+joins the worker with a
     `WorkerShutdownTimeout` bound and returns an error → the builtin returns
     `Err(SpawnFailed, "execution owner is shutting down")` (late-admission
     path is existing `WorkerAdmission.Complete` behavior).
   - `err := cmd.Wait()` — the rest of the result mapping (output-limit,
     Timeout, AbnormalExit, Ok-for-nonzero) is byte-identical.
   - `lifetime.AfterWait()`; receipt join bounded by `WorkerShutdownTimeout`
     (or the admission deadline when set); join failure is recorded into the
     worker's join error → existing `WorkerCleanupFailure{"natural
     completion"|"join"}` surfaces it in `CloseWorkers()` output (D5 —
     stdout-visible, not a language value).
   - `close(runDone)`; `admission.ReleaseWorker(worker)` (no-op fast path: Done
     already closed).

**Why this is safe for the signal path's "never wait for the evaluator"
contract:** `HandleWorkerSignal` does not wait for the evaluator goroutine; it
waits for *workers*. `syncExecWorker.Join` waits only for `cmd.Wait()` to
return, which is bounded by the already-installed `WaitDelay = 5s` pipe
deadline after the kill lands. The 2s `WorkerShutdownTimeout` bounds the host's
wait; if a pipe-holding descendant outlives it, the failure is a recorded
`WorkerCleanupFailure` and the kill has already been delivered — identical
outcome to today's timeout path.

**Policy-owned group preservation:** under `--policy`, `run_lifetime.go` calls
`proctree.StayInOwnerGroup()`; `ConfigureContained` keeps the exec child in the
owner's group and its `cmd.Cancel` kills only the child. The design adds no
`Setpgid`, no group kill in that mode (D4), and no change to the supervisor's
outer group kill — the containment fixed in 7ae77f35 is byte-identical.

**Exit codes:** `InstallWorkerSignalHandler` already computes 130/143 and
`TerminalSignalExit` (os.Exit) preserves them. Untouched.

### Implementation Plan

**Phase 1: worker + lifetime wiring** (~1 day)
- [ ] Add `proctree.InOwnerGroup() bool` probe (one line beside `StayInOwnerGroup`).
- [ ] Add `internal/effects/process_exec_worker.go`: `syncExecWorker` with
      D4-scoped force, either/or lifetime-vs-cancel RequestStop, bounded Join.
- [ ] Rewire `processExec` steps 1–5 as above; keep all result constructors
      and the `DEBUG_PROCESS` trace lines identical.

**Phase 2: tests** (~1 day)
- [ ] Unit: `exec` during owner-close → late-admission `Err(SpawnFailed)`; no
      leaked child (reuse the harness in `process_runtime_lifetime_test.go`).
- [ ] Unit: signal-style `CloseWorkersContext` while `sleep 10` exec in flight
      → child group gone within `WorkerShutdownTimeout`.
- [ ] Unit: nested `ailang run` via ordinary `exec` → stop request reaches it,
      terminal receipt required (mirror `TestRuntimeNestedStopUsesRemainingDeadline`,
      `process_runtime_lifetime_test.go:257`).
- [ ] E2E (issue #2034 repro script, POSIX, CI-gated): SIGTERM and SIGINT to
      the runtime at 0.5s → no `late_marker`, exit 143/130.
- [ ] Regression: policy-mode exec child stays in the owner's group;
      runtime-side stop kills only the child, never the owner's group
      (extend the 7ae77f35-era fixtures).

**Phase 3: documentation** (~0.5 day)
- [ ] Update `docs/docs/reference/` process-effect page and `docs/LIMITATIONS.md`:
      SIGTERM/SIGINT cleanup guaranteed; SIGKILL/crash/power-loss **not**
      guaranteed for ordinary runs (D6) — only policy-owned groups (owner
      group kill) and registered same-runtime children (lifetime receipts)
      contain those; setsid/daemon descendants remain #2037.

### Pipeline Pass Coverage

Not applicable — no parser, typechecker, elaboration, codegen, VM or
effect-row changes. The `std/process.ail` signature
`exec(cmd: String, args: List[String]) -> Result[ProcessOutput, ProcessError] ! {Process}`
is untouched (`std/process.ail:64–68` verified in triage).

### Files to Modify/Create

**New files:**
- `internal/effects/process_exec_worker.go` — `syncExecWorker` (~80 LOC)

**Modified files:**
- `internal/effects/process.go` — admission, Prepare, Start/Wait split, release (~40 LOC delta)
- `internal/proctree/proctree.go` — `InOwnerGroup()` probe (~4 LOC)
- `internal/effects/process_runtime_lifetime_test.go` or new
  `process_exec_lifetime_test.go` — tests (~120 LOC)
- `docs/LIMITATIONS.md`, process reference page — containment guarantees

## Examples

### Example 1: cancelled BashExec (the #2034 repro)

**Before:** SIGTERM to the runtime → exit 143, `bash`/`sleep` survive with no
deadline, `late_marker` written at t+3s.

**After:** SIGTERM → signal handler → `CloseWorkers()` → `syncExecWorker.RequestStop()`
→ (non-runtime child) `execCtx` cancelled → `ConfigureContained`'s
`cmd.Cancel` SIGKILLs the child's whole group → `Wait` returns → exit 143.
No marker.

### Example 2: nested runtime under a policy owner

**Before:** `exec(resolvedAilang, ["run", "child.ail"])` inside an owned run —
the policy supervisor's stop request reaches the runtime's session, which stops
registered workers, but the ordinary exec child (itself an `ailang run`)
never gets a stop request or sends a cleanup receipt.

**After:** `runlife.Prepare` engages (same-binary `run` child) →
`lifetime.RequestStop(deadline, force)` delivers the remaining-duration stop
request → the child runtime runs its bound cleanup, writes the terminal
receipt, and only a valid receipt clears the join — exactly the
`TestRuntimeOwnedWorkersRequireReceipt` contract the managed path already
meets. In owner-group mode `force` is `KillProcess(child)` — the supervisor's
group is never signaled from inside.

## Success Criteria

- [ ] Issue #2034 repro: no `late_marker` and no surviving group member after SIGTERM or SIGINT; exit codes 143/130 (acceptance test, POSIX e2e)
- [ ] `TestProcessExec_Timeout` (`process_test.go:243`) still returns `Err(Timeout)`; `TestProcessExec_TimeoutKillsGrandchildren` (`process_timeout_group_test.go:23`) still passes — timeout semantics unchanged
- [ ] `TestProcessExec_NonZeroExit_IsOk` still Ok; `TestProcessExec_EchoSuccess`, `_NotFound`, `_Allowlist_*`, confined tests unchanged
- [ ] `TestRuntimeNestedStopUsesRemainingDeadline`, `TestRuntimeOwnedWorkersRequireReceipt`, `TestRuntimeLateStopJoinsBackpressuredInheritedOutput` pass, and an ordinary-exec nested runtime now also requires a receipt
- [ ] Policy-mode: runtime-side shutdown kills only the child process; supervisor group untouched (fixture)
- [ ] Windows: leader-only termination retained (`WorkerCancellationSupported()` gate unchanged); no build break
- [ ] All tests passing (`make test-core` + effects suite); documentation updated (LIMITATIONS + process reference)
- [ ] Engine parity: evaluator path only — no VM/codegen involvement

## Testing Strategy

**Unit tests:** admission failure and late-admission paths; either/or
lifetime-vs-cancel in `RequestStop`; force-kill scope under `InOwnerGroup()`
(true and false); receipt join failure recorded but not mutating the Result.

**Integration tests:** end-to-end signal repro (spawn the built `ailang`
binary, signal it, assert marker absence + exit code); nested-runtime stop
with remaining-deadline propagation through ordinary exec.

**Manual testing:** the exact issue #2034 reproduction table, all four rows,
plus the policy-mode counterpart (`ailang run` under a `--policy` supervisor,
kill the runtime, confirm the supervisor survives and the child does not
outlive the owner's group kill).

## Deferred Decisions

- Exact receipt-join bound for the synchronous path (`WorkerShutdownTimeout` vs
  admission deadline) — agent may choose, must stay ≤ existing managed-path
  bounds.
- Whether the e2e signal test lives in `internal/effects` or the POSIX CI lane
  — agent may choose (needs a built binary; follow `mission_activation_unix.go`
  precedent if the CI lane is easier).
- Whether to emit a `DEBUG_PROCESS` trace line on host-shutdown stop — agent
  may choose (recommended: yes, one line, same format).

## Non-Goals

**Not attempted in this feature:**
- **SIGKILL-of-runtime / crash / power-loss cleanup for ordinary runs** — cannot
  be handled in-process; kernel parent-death mechanisms (Linux `Pdeathsig`,
  cgroups, subreapers) are a separate portability-and-ownership decision. We
  document the limitation instead of promising it (D6). Policy-owned groups and
  registered nested runtimes keep their existing containment.
- **#2037 (setsid/daemon descendants escaping the group)** — separate
  stronger-containment policy (triage D-TRIAGE-39); deliberately not bundled
  here.
- **Per-call timeout override (#2035)** and **configured-timeout query (#2036)** —
  orthogonal API-surface asks, own decisions.
- **No change to `Result` semantics** — `Ok` for all completed processes,
  `Err` only for infra/spawn failures; host cleanup failures are not language
  values (D5).
- **No new signal subscription** — `ailang run` keeps the single
  `InstallWorkerSignalHandler` channel.

## Timeline

**Week 1** (~2.5 days): Phase 1 wiring (1d) + Phase 2 tests (1d) + Phase 3 docs (0.5d).
Human design approval gates the start; sprint planning is a separate step.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Signal-path wait exceeds `WorkerShutdownTimeout` when a pipe-holding descendant lingers | Low (kill already delivered; failure recorded) | Existing `WaitDelay`/`WorkerCleanupFailure` semantics; documented in design |
| `Prepare` changes env/ExtraFiles ordering for confined runs | Med | `confine` runs before command build; `Prepare` appends after — same ordering as managed path; confined tests are regression fixtures |
| A second concurrent exec per evaluation registers two workers | Low | `WorkerOwner.workers` is already a map keyed by worker; admission barrier serializes with close |
| Policy regression: runtime group-signals the owner's group | High | D4 freeze + dedicated fixture; `InOwnerGroup` probe is the only new authority decision and is tested both ways |
| Nested-runtime stop changes `Err(Timeout)` timing | Med | Timeout path never uses `lifetime.RequestStop` (D3); `TestProcessExec_Timeout` + `TestProcessExec_TimeoutKillsGrandchildren` pin it |

## Related Documents

**Implemented (mechanisms reused):**
- [m-worker-policy-and-lazy-ai](../../implemented/v0_54_5/m-worker-policy-and-lazy-ai.md) — the runlife protocol, its same-binary `run` gate, receipts, and the explicit note (V-row, line ~133) that the ordinary exec spawn path had no inherited lifecycle transport: this design closes that gap
- [m-executor-policy-hardening](../../implemented/v0_41_0/m-executor-policy-hardening.md) — policy worker isolation and the admission fd
- [m-process-exec](../../implemented/v0_8_1/m-process-exec.md) — original builtin contract (timeout, output limits, completion semantics)
- [m-process-pgid-cleanup](../../archive/m-process-pgid-cleanup.md) — history of group-kill-on-timeout; this doc is the parent-death counterpart
- [m-process-result-semantics](../ailang-core-triage/m-process-result-semantics.md) — Result payload decisions this design deliberately does not touch

**Planned (check for overlap):**
- None found. Duplicate/coverage gate: SimHash + neural search on "process exec
  parent lifetime" returned zero matches in both `implemented/` and `planned/`;
  the iteration25 ownership lookup (verifier-a/issue2034-ownership.md) found no
  existing design, issue comment, open PR, or recent task covering ordinary
  synchronous exec parent lifetime (0/0/0 of 100).

## Conflict Surface

This design touches `internal/effects/` (a builtin's host-cleanup wiring, not
the effect-row algebra), so the section is included.

### Syntactic positions touched

None. No parser, lexer, AST, type, effect-row or signature changes.
`std/process.ail` `exec(cmd, args)` and its result types are untouched.

### What else lives here

The positions extended are **runtime host-cleanup positions**, not grammar
positions:

| Position | Existing occupant | This design |
|----------|-------------------|-------------|
| `WorkerOwner.workers` registry | managed processes (`process_spawn.go:77`), async/stream sources (`stream_async_process.go:75`) | adds `syncExecWorker` for the synchronous builtin |
| `runlife.Prepare` call sites | `process_managed.go:73`, `stream_process.go:67`, `run_policy_lifetime.go:36` | adds `process.go` |
| `proctree.ConfigureContained(cmd)` | `process.go:174` only | unchanged — still the single group-placement decision |
| Signal subscription | `InstallWorkerSignalHandler` (one per `ailang run`) | unchanged — no second subscription |

### Disambiguation strategy

The either/or in `RequestStop` (lifetime vs plain cancel) mirrors
`managedProcess.RequestStopContext` structurally, so no new policy decision is
introduced at shutdown time. `Prepare`'s same-binary gate (`child.go:37–50`)
already disambiguates runtime children from arbitrary commands.

### Programs that MUST still work

1. Issue #2034 repro module (verified: `ailang check` passes — see
   Verification Log) — timeout control row must be unchanged.
2. `examples/` process examples and every fixture behind
   `TestProcessExec_*` (`process_test.go`), the subcommand allowlist suite
   (`process_subcmd_test.go`), the confined-mode suite
   (`process_confined_test.go`, `process_confined_ceiling_test.go`).
3. Policy-mode containment fixtures from the 7ae77f35 fix and
   `process_runtime_lifetime_test.go` (receipt, remaining-deadline,
   natural-completion race tests).
4. motoko_agent's BashExec cancellation contract: SIGTERM → exit 143, tool
   call ends, no surviving child.
5. Windows: `process` builtins still build and run with leader-only
   termination (`WorkerCancellationSupported()` false path).

### What deliberately changes

- A `std/process.exec` child can no longer outlive a SIGTERM/SIGINT'd runtime
  (standalone POSIX runs). This is the requested fix; no valid program relied
  on the orphan.
- An `exec` started *while* the host is already shutting down now fails fast
  with `Err(SpawnFailed)` instead of starting an unkillable-after-exit child.
  No other Result change.

## Verification Log

| Claim | Check | Result |
|---|---|---|
| `processExec` uses a background timeout and no `runlife.Prepare`/worker registration | Read `internal/effects/process.go:159,174,210`; grep `BeginWorker()` callers: only `process_spawn.go:77`, `stream_async_process.go:75` | Confirmed |
| Managed/async paths already use `runlife.Prepare` | `process_managed.go:73`, `stream_process.go:67` | Confirmed |
| `ailang run` already installs a SIGINT/SIGTERM handler that closes workers and exits 130/143 | `internal/runner/run.go:645–651`, `effects/terminal_lifecycle.go:63–92` | Confirmed — the handler fires today; the exec child is merely invisible to it |
| `runlife.Prepare` engages only for same-binary `run` children (arbitrary commands unaffected) | `internal/runlife/child.go:33–50` (`os.SameFile` + `cmd.Args[1] == "run"`) | Confirmed (documented in m-worker-policy-and-lazy-ai.md line 49) |
| No kernel parent-death / subreaper / cgroup mechanism exists in the runtime | grep `Pdeathsig\|PDEATHSIG\|subreaper` in `internal/`, `cmd/`: only `prctl(PR_SET_DUMPABLE)` in `proctree/dumpable_linux.go` (unrelated) | Confirmed absent (negative existence) |
| No existing design/PR/task covers ordinary-exec parent lifetime | Script search (SimHash + neural): 0 matches; iteration25 ownership lookup: 0 issue comments, 0 open PRs, 0/100 recent tasks | Confirmed (duplicate gate passed) |
| The #2034 repro program is valid AILANG today | `ailang check` on the issue's exact module: `✓ No errors found!` (rc=0, 2026-11-03) | Confirmed |
| `ConfigureContained` preserves policy-owned groups (child stays in owner group; cancel kills only the child) | `internal/proctree/proctree.go:63–79`; `run_lifetime.go:13–17` | Confirmed |
| No new error-code allocation needed | D5 reuses `SpawnFailed`/`AbnormalExit`/`Timeout` constructors (`makeProcessResultErr*`); no `MODxxx`/`EFF_*` strings introduced | Confirmed |
| Cited tests exist and assert what is claimed | `process_test.go:243` asserts `Err` ctor == `Timeout`; `process_timeout_group_test.go:23` kills grandchildren; `process_runtime_lifetime_test.go:180,257` require receipts / remaining-deadline | Confirmed (read bodies) |

## References

- Issue #2034 — [motoko_agent] std/process.exec: a killed runtime leaves the command running with no deadline (P1, `area:effects`)
- Triage iteration25 independent verifier/judge evidence: #2069 (comment 6599231990) — KEEP, CONCUR/CONCUR, design-doc-creator-only route
- Issue #2037 — setsid/daemon stronger containment (separate; D-TRIAGE-39)
- Issues #2035 / #2036 — per-call timeout override / configured-timeout query (separate)
- [Design Axioms](/docs/references/axioms) — scoring below

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Program-visible results byte-identical; cleanup is host-side only |
| A2: Replayability | 0 | No trace/replay surface change |
| A3: Effect Legibility | +1 | No hidden side effects; makes an existing declared `Process` effect's cleanup actually bounded |
| A4: Explicit Authority | +1 | No new ambient authority; shutdown reaches the child only through the capability-granted effect that spawned it |
| A5: Bounded Verification | 0 | No verification-surface change |
| A6: Safe Concurrency | +1 | Reuses the admission barrier; no new lock ordering, one registry |
| A7: Machines First | +1 | Removes an unbounded-escape failure mode that AI-generated pipelines hit deterministically |
| A8: Minimal Syntax | +1 | Zero syntax/API change |
| A9: Cost Visibility | 0 | Host cleanup cost already recorded via `WorkerCleanupFailure` receipts |
| A10: Composability | +1 | Ordinary exec now composes with policy/nested ownership like its managed siblings |
| A11: Structured Failure | +1 | Late admission and receipt failures surface as typed, bounded host failures |
| A12: System Boundary | +1 | Explicit documented boundary: what dies with the runtime and what does not (D6) |

**Net Score: +9** → **Decision: Move forward (pending human design approval per D1/D3/D4/D6 freeze).**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): removes a machine-hostile failure mode; no human-convenience tradeoff

## Future Work

- #2037's stronger containment decision (cgroup/subreaper/Pdeathsig on Linux)
  would extend the SIGKILL-of-runtime row of the #2034 table — deliberately a
  separate, human-owned policy decision.
- #2035/#2036 per-call timeout surface could later share the admission context
  introduced here.

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03
