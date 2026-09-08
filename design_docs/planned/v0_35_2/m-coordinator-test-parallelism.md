# M-COORDINATOR-TEST-PARALLELISM: Attack the Four Timer-Bound Coordinator Tests by Injecting Their Timers

**Status**: Planned — mission iteration 350, design-doc-creator (REVISION pass, round 3, final). **Target**: v0.35.2 ·
**Priority**: P1 (standing latency floor, not a CI timeout fix) · **Estimated**: 3–3.5 hours
(3 injection points + 4 test updates + regression guard + mutation + evaluation)
**Dependencies**: none (self-contained in `internal/coordinator`); base commit `f3783c976` (origin/dev).
**Created**: 2026-09-08 · **Revision**: iter-350-r3 (final; round-2 quorum BLOCK 3/3, narrow-refinement carve-out applying reviewers' verbatim fixes).

## Problem Statement

`internal/coordinator` is the slowest package in the test matrix by wall-clock, and it is
**effectively serial**: `go test ./internal/coordinator/ -count=1` takes **13.12 s** wall while the
sum of the per-test durations is **12.77 s** across **739 `func Test` declarations** — wall ≈ sum,
so there is no intra-package concurrency to hide the cost. Four timer-bound tests account for
**8.14 s of that 12.77 s (63.7%)**:

| Test | Duration | Source of the time |
|---|---|---|
| `TestIntegration_TaskExecutorWithRetry` | 3.01 s | hardcoded `baseDelay := time.Second` exponential backoff in `ExecuteWithRetry` (1 s + 2 s) |
| `TestStoreBackedApprovalCheckpoint_Rejection` | 2.02 s | hardcoded `pollInterval: 2 * time.Second` in `NewStoreBackedApprovalCheckpoint` |
| `TestStoreBackedApprovalCheckpoint` | 2.01 s | same hardcoded `pollInterval` |
| `TestCoordinatorEventHandler_RateLimitReset` | 1.10 s | hardcoded `time.Second` rate-limit window in `checkRateLimit()` + `time.Sleep(1100ms)` |

Every other test is ≤ 0.17 s. These 8.14 s are **real sleep/poll time, not CPU** — the tests wait on
hardcoded production timers that the test does not control. That makes them the one part of the
package that is both the largest share of the wall time and safe to attack without touching shared
state.

This item is **NOT** a fix for the Windows CI timeout. Iteration 348 measured that the package that
blows next on Windows is `cmd/ailang`, not `internal/coordinator`, and that speeding the coordinator
up would not have prevented the failing run. This design is justified on its own merits: the
wall-clock cost of the slowest package in the matrix, and a serial 739-test package being a standing
latency floor that every `go test ./...` pays.

## Verification Log

All rows measured first-party in this iter-350 session at base `f3783c976` (arm64 Darwin, `go test`
`-count=1`). The Windows readings are historical controller measurements at the named commits, cited
as prior evidence, not re-measured here.

| ID | Claim / command | Observed |
|---|---|---|
| V1 | `git rev-parse HEAD` | `f3783c9765d1fcdca51533c401bea6a7e53d19b8` (matches base `f3783c976`). |
| V2 | `grep -rho 't\.Parallel()' internal/coordinator/*_test.go \| wc -l` | **0** — no `t.Parallel()` in the package. |
| V3 | positive control: `grep -rho 't\.Parallel()' internal/ --include='*_test.go' \| wc -l` | **17** — the same pattern is visible repo-wide under `internal/`, so the zero in V2 is a fact, not a broken instrument. |
| V4 | `go test ./internal/coordinator/ -count=1 -json` | exit 0; wall **13.12 s** (start 03:16:47.200 → end 03:17:00.320). |
| V5 | sum of per-test `Elapsed` from the same `-json` run | **12.77 s** across 129 top-level pass events; **739 `func Test` declarations** in source (`grep -rhoE 'func Test[A-Za-z0-9_]+' internal/coordinator/*_test.go`). Wall ≈ sum ⇒ serial. |
| V6 | top-4 durations from the `-json` run | 3.01 s `TestIntegration_TaskExecutorWithRetry`; 2.02 s `TestStoreBackedApprovalCheckpoint_Rejection`; 2.01 s `TestStoreBackedApprovalCheckpoint`; 1.10 s `TestCoordinatorEventHandler_RateLimitReset`. Combined **8.14 s = 63.7 %** of the 12.77 s sum. Every other test ≤ 0.17 s. |
| V7 | shared-state surface: `grep -rho 't\.TempDir()' internal/coordinator/*_test.go` | **68** `t.TempDir()`. |
| V8 | `grep -rho 'os\.MkdirTemp' internal/coordinator/*_test.go` | **5** `os.MkdirTemp`. |
| V9 | `grep -rho 'exec\.Command' internal/coordinator/*_test.go` | **13** `exec.Command`. |
| V10 | read `TestIntegration_TaskExecutorWithRetry` (integration_test.go:186) | mock provider fails twice then succeeds; `ExecuteWithRetry(ctx, task, opts, 3)`; `opts.Workspace = t.TempDir()`. The 3.01 s is the backoff, not CPU. |
| V11 | read `ExecuteWithRetry` (task_executor.go:152-158) | `baseDelay := time.Second` is a **local**; `delay := baseDelay * time.Duration(1<<(attempt-1))` → 1 s then 2 s. Hardcoded, not injectable. |
| V12 | read `TestStoreBackedApprovalCheckpoint` / `_Rejection` (approval_checkpoint_edge_test.go:12,68) | each opens a real SQLite store in `t.TempDir()`, calls `NewStoreBackedApprovalCheckpoint(store, 1*time.Hour)`, resolves via the store, then `wg.Wait()` for the poll to detect the change. |
| V13 | read `NewStoreBackedApprovalCheckpoint` (approval_checkpoint.go:334-338) | `pollInterval: 2 * time.Second` hardcoded in the constructor; `pollTicker := time.NewTicker(sac.pollInterval)` at :387. The 2 s is the poll interval. |
| V14 | read `TestCoordinatorEventHandler_RateLimitReset` (event_handler_test.go:123) | sends 15 messages (burst), `time.Sleep(1100 * time.Millisecond)`, sends one more, asserts it is broadcast. The 1.10 s is the sleep. |
| V15 | read `checkRateLimit` (event_handler.go:275-282) | `if now.Sub(h.lastEventTime) >= time.Second` — the reset window is hardcoded `time.Second`; `maxEventsPerSec: 10` is set in the constructor (event_handler.go:66). |
| V16 | **production construction sites** of the three types (`grep -rn '<ctor>' --include='*.go' . \| grep -v '_test.go'`) | `NewStoreBackedApprovalCheckpoint`: **0** production callers (the only 2 callers are both in `approval_checkpoint_edge_test.go`). `NewCoordinatorEventHandler`: **1** production caller (`daemon_tasks_exec_run.go:292`). `ExecuteWithRetry`: **1** production caller (`daemon_tasks_exec_run.go:333`), which builds `opts := &ExecuteOptions{...}` at :238. Injection is low-blast-radius; the checkpoint constructor has zero production blast radius. |
| V17 | **conflict-surface, clock — NEGATIVE with control + disambiguation**: `grep -rnoE 'Clock\b' internal/coordinator/` | **16 hits, EVERY ONE** the `CapabilityClock` capability-type constant in `capability_detector.go` — a capability **enum**, not an injectable clock. **Control**: `grep -rnoE 'Clock\b' internal/ --include='*.go' \| wc -l` = **357** (the grep is not broken). **Disambiguation**: `internal/cognition/clock.go` defines a `Clock` type but is a different subsystem (cognition, not coordinator). **Conclusion**: there is **no injectable clock abstraction in `internal/coordinator`**; the rate-limit test's clock injection (M2/M3) is a **new** injection point, not a reuse. |
| V18 | **conflict-surface, timer/retry machinery**: `grep -rnoE 'time\.(NewTicker|NewTimer|After|Sleep|Tick)\b' internal/coordinator/*.go` (non-test) | **18 hits across 14 files** (apikey_cache, approval_checkpoint, approval_watcher, backstop_sweep, daemon, daemon_github, daemon_lifecycle, heartbeat, resource_tracker, stale_task_detector, task_executor, triage_router, watcher). These are production daemon loops / watchers / heartbeats; **none is an injectable retry/poll/rate-limit knob**. The three target timers (backoff at task_executor.go:165, poll at approval_checkpoint.go:387, rate-limit window in `checkRateLimit`) are the only ones under test here. |
| V19 | **conflict-surface, test-synchronization machinery**: existing test-sync surface | `t.TempDir()` (68, V7), `os.MkdirTemp` (5, V8), `exec.Command` (13, V9). **No clock/timer-injection machinery exists in the test files** — there is no existing test-sync mechanism to reuse for the four timer-bound tests. |
| V20 | **FIX 2 verification row — `IsThrottled()` exists**: `grep -n 'func (h \*CoordinatorEventHandler) IsThrottled' internal/coordinator/event_handler.go` | **POSITIVE** — `func (h *CoordinatorEventHandler) IsThrottled() bool` at **event_handler.go:306**. So nothing is added to M1's production-change list for it; the row records the positive. |
| V21 | **FIX 2 — `DefaultExecuteOptions()` current shape**: read provider.go:99-104 | returns only `{Timeout: 5*time.Minute, DryRun: false}`. It **must gain `RetryBaseDelay: time.Second`** (FIX 2). |
| V22 | **FIX 2 — production caller rebase surface**: read daemon_tasks_exec_run.go:238,333 | builds `opts := &ExecuteOptions{...}` setting Timeout, IdleTimeout, Workspace, ObservatoryContext, AgentConfig; call `d.executor.ExecuteWithRetry(taskCtx, analyzed, opts, 2)` at :333. Rebasing on `DefaultExecuteOptions()` and overriding those five fields preserves behavior exactly (each override wins over the default). |
| V23 | **FIX 1 — wait/ticker seam search, NEGATIVE with control, VERIFIED BY THE CONTROLLER**: `grep -rnoE 'Clock\b' internal/coordinator/` | **16 hits, every one** the `CapabilityClock` capability-type enum in `capability_detector.go`, not an injectable clock; repo-wide control `grep -rnoE 'Clock\b' internal/ --include='*.go' \| wc -l` = **357** (grep works). **No existing wait/ticker seam in this package** — the retry/poll wait/ticker seams (M1) are new injection points, not a reuse. (This is the search astra's fix requires; run by the controller.) |
| W1 | (historical, controller) Windows CI steady-state for this package | 88.7 s @ `81abc956d`, 100.8 s @ `8e3927950`, 99.4 s @ `98730db02`, 124.4 s @ `e5a325a20`; once hit the whole-package timeout at >300 s. Cited as prior evidence; not re-measured on this Mac. |
| V24 | (post-M2, executor iter-351) four-test combined duration | `go test ./internal/coordinator/ -run 'TestIntegration_TaskExecutorWithRetry\|TestStoreBackedApprovalCheckpoint\|TestStoreBackedApprovalCheckpoint_Rejection\|TestCoordinatorEventHandler_RateLimitReset' -count=1 -json` → sum of `Elapsed` over the four top-level pass events = **0.23 s** (base 8.14 s). |
| V25 | (post-M2, executor iter-351) full-package wall | `time go test ./internal/coordinator/ -count=1` → **6.6 s** real (base 13.12 s; controller re-measured 16 s on a loaded box). |

## Quorum Verification Log (round 1)

Round-1 design-quorum on this doc: **3 of 3 external reviewers rejected → BLOCKED**. This revision
answers every objection. The design **direction** (injectable timers, four timer-bound tests only,
no `t.Parallel()`) was not disputed by any reviewer; this is a completeness/mechanism revision.

| Reviewer | Verdict | Objection(s) | How answered in this revision |
|---|---|---|---|
| gemini-3-1-pro | REJECT (blocking) | **O1 — Conflict Surface**: retry delay should live on the existing `ExecuteOptions` surface (which already carries `Timeout`/`IdleTimeout` duration knobs), not a new receiver field + test-only setter. | Retry delay moved to `ExecuteOptions.RetryBaseDelay` (per-call config on the existing surface). The other two injection points re-examined against their existing config surfaces (constructor param / functional option). **All setters removed** — no shared mutable state. |
| gpt6-astra | REJECT (blocking) | **O2 — the catch**: M3 guards assert field assignment, not production consumption; the mutation table hides the body-only reversion behind compile failures. | M3 guards rewritten to **observe behavior** (elapsed-time bound / call count / clock-driven reset), not the field. Mutation table rewritten so **every mutant compiles** and the body-only reversion is the key mutant, each row naming the single killing assertion. |
| gpt6-astra | REJECT (blocking) | **O3 — determinism**: the rate-limit test can go **vacuous** under a shortened window; A1/A6 and the "no more flaky than today" claim are unsupported. | Rate-limit test rewritten with **clock injection** (no wall-clock racing) and a mandatory **"throttling engaged"** assertion so it cannot pass vacuously. A1/A6 and Goal 4 corrected to what the revised design actually supports. |
| gpt6-astra | REJECT (blocking) | **O4 — conflict-surface search**: no search for existing clock/timer/retry/test-sync machinery; V16 greps test files, not production construction. | Conflict-surface search run and published as a **negative with control + disambiguation** (V17–V19). V16 replaced with **production construction sites**. |
| oc-glm-5-2 | REJECT (blocking) | Same surface as gemini (conflict surface). **Recorded ABSENT(invalid) by the tool** because its JSON was malformed; its raw text begins `"verdict": "reject"`. Counted as a reject. | Same answer as O1. |

**Verdict: 3/3 reject → BLOCKED.** All objections addressed in this revision.

**Tool defect noted:** `oc-glm-5-2`'s JSON was malformed, so the quorum tool recorded it
`ABSENT(invalid)` rather than as a verdict. Its raw text begins `"verdict": "reject"` on the same
conflict-surface as gemini-3-1-pro, so it is counted as a reject. The tool should surface
malformed-JSON verdicts as rejects instead of silently dropping them.

## Quorum Verification Log (round 2)

Round-2 design-quorum on this doc: **3 of 3 external reviewers rejected → BLOCKED**, all present
(no absences at the raised cap). The rejections land on **two surfaces** — the elapsed-time guard
bounds and the silent zero-value fallback — and none disputes the design **direction**. Every
reviewer supplied a concrete fix, so the mission protocol's **narrow-refinement carve-out** applies:
round 3 applies the reviewers' fixes **as written**; there is **no third quorum round**.

| Reviewer | Verdict | Objection(s) | Fix (applied verbatim in round 3) |
|---|---|---|---|
| gemini-3-1-pro | REJECT (blocking) | **O5 — elapsed-time guard bounds**: the M3 retry/poll guards assert `elapsed time < 1 s`; a wall-clock upper bound on a loaded CI runner can fail a correct implementation, a NEW flake source contradicting Goal 4 and the A1/A6 scores. | Apply the clock-injection pattern (already designed for the rate-limit test) to the retry and poll mechanisms: inject a deterministic test clock/ticker interface so M3 guards advance virtual time and verify retries/polls trigger without real `time.Sleep` or non-deterministic wall-clock bounds. |
| gpt6-astra | REJECT (blocking) | **O6 — same surface, more detail**: replace the retry/poll elapsed-time guards with behavior tests using an injected, test-controlled wait/ticker seam (after verifying whether suitable machinery already exists); record and assert the requested retry delays and poll interval, explicitly release waits/ticks, assert resulting attempts and approval status; give async operations cancellation and a bounded failure/cleanup path rather than an unconditional `wg.Wait()`; keep wall-time targets as local measurements outside CI; describe watchdog deadlines as liveness safeguards; make these behavioral guards mandatory with M1/M2; update Goals, Risks, A1/A6 to remove unsupported determinism and load-immunity claims. | Applied in full (FIX 1). |
| oc-glm-5-2 | REJECT (blocking) | **O7 — silent zero-value fallback** (CLAUDE.md Principle 2): remove the `if baseDelay == 0 { baseDelay = time.Second }` fallback from `ExecuteWithRetry`; update the single production caller at daemon_tasks_exec_run.go:238 to use `DefaultExecuteOptions()` as the base and override specific fields, so `RetryBaseDelay` is set explicitly to `time.Second` by the default constructor rather than via silent zero-value coercion; add a verification row confirming whether `IsThrottled()` exists on `CoordinatorEventHandler`. | Applied exactly (FIX 2). `IsThrottled()` **exists** (event_handler.go:306) — verification row V20 records the positive; nothing added to M1's production-change list for it. |

**Verdict: 3/3 reject → BLOCKED.** Round 3 is a **narrow-refinement carve-out** applying the
reviewers' verbatim fixes; no third quorum round.

## Goals

1. Remove the 8.14 s of real sleep/poll time from the four timer-bound tests by making their
   production timers **injectable**, with defaults preserved so production behavior is unchanged.
2. Drop the four tests' combined duration from **8.14 s to < 0.5 s** (target ~0.25 s), and the full
   package wall time from **13.12 s to < 6 s**, measured with a named instrument under stated
   conditions.
3. Add a **behavior-observing regression guard** (white-box assertions that observe the injected
   value's **effect on production behavior** through a test-controlled wait/ticker seam — not the
   field, and not elapsed wall time) so a future reversion to hardcoded timers fails loudly instead
   of silently re-serialising the package.
4. Leave the package **no more flaky than it is today** — no new `t.Parallel()`, no new shared-state
   exposure. All three guards assert **behaviour through a test-controlled seam** (a clock for the
   rate-limit test, a wait/ticker seam for retry and poll), so none depends on real wall-clock timing
   and none can pass vacuously. The design does **not** claim load-immunity: the wall-time targets
   are recorded once as local measurements (see Local Measurement), not asserted in CI.

## Non-Goals

1. **Not a Windows-timeout fix.** Iteration 348 measured `cmd/ailang` as the next package to blow on
   Windows; speeding the coordinator up would not have prevented that run. This design is justified
   by the coordinator's own wall-clock cost and serial floor, not by the Windows timeout.
2. **No blanket `t.Parallel()` across the 739 tests.** With 68 `t.TempDir()`, 5 `os.MkdirTemp`, 13
   `exec.Command` and real SQLite store opens, a blanket change is a new flake source. This repo has
   already paid for exactly that class twice (`m-message-watcher-windows-wallclock-flake`,
   `m-debugcacheforms-flaky-on-macos-ci`). A fix for slowness that introduces intermittency is the
   same defect with a longer mean time to discovery.
3. **No parallelising the other 125 tests** (each ≤ 0.17 s). Their combined cost is ~4.6 s and they
   touch the shared-state surface; that is a separate, riskier row (see Out-of-scope).
4. **No change to production timing semantics.** Defaults stay identical; only the ability to override
   them for tests is added.

## Solution Design

Three production injection points, each **per-call config** (no setters, no shared mutable state),
then four test updates. All four tests are fixed with **option (a) — inject/shorten the timer or
poll interval**, not `t.Parallel()`. Rationale per test:

- **`TestIntegration_TaskExecutorWithRetry`** — the 3.01 s is `ExecuteWithRetry`'s hardcoded
  `baseDelay := time.Second` (1 s + 2 s backoff). **Fix (a)**: add `RetryBaseDelay` to the existing
  `ExecuteOptions` struct (per-call config; `DefaultExecuteOptions()` sets `RetryBaseDelay:
  time.Second` explicitly — **no `0 → time.Second` fallback**, per FIX 2 — and the production caller
  at daemon_tasks_exec_run.go:238 rebases on `DefaultExecuteOptions()` and overrides its five fields,
  so behavior is preserved exactly); the test sets `opts.RetryBaseDelay = 10*time.Millisecond` and
  injects a `Wait` seam. `t.Parallel()` would overlap the sleeps but not remove them, and would add
  nothing here (the test is already isolated via `t.TempDir()` and a local mock) — so (a) is strictly
  better: it removes the time entirely and is deterministic in outcome.
- **`TestStoreBackedApprovalCheckpoint` / `_Rejection`** — the 2 s each is the hardcoded
  `pollInterval: 2 * time.Second`. **Fix (a)**: add `pollInterval` as a third constructor parameter
  and a `tick func(time.Duration) <-chan time.Time` seam as a fourth to
  `NewStoreBackedApprovalCheckpoint(store, defaultTimeout, pollInterval, tick)` (per-call config; the
  constructor is the existing config surface and has **zero production callers** — V16 — so the
  signature change is trivially safe); the tests pass `10*time.Millisecond` and a test-controlled
  tick. `t.Parallel()` would overlap the two 2 s polls but each still costs 2 s and both open real
  SQLite stores — (a) removes the time and avoids the store-open concurrency question entirely.
- **`TestCoordinatorEventHandler_RateLimitReset`** — the 1.10 s is the hardcoded `time.Second`
  rate-limit window plus the test's `time.Sleep(1100ms)`. **Fix (a)**: inject a **clock** — add a
  `now func() time.Time` field (default `time.Now`) to `CoordinatorEventHandler`, injected via a
  functional option `WithClock(now func() time.Time)` on the constructor (per-call config; the 1
  production caller at daemon_tasks_exec_run.go:292 is unchanged). `checkRateLimit` uses `h.now()`
  instead of `time.Now()`. The test injects a controllable clock, asserts throttling **engaged**,
  advances the clock past the window, and asserts the reset — **no `time.Sleep`, no wall-clock
  racing, cannot go vacuous** (see M2/M3). This is a **different mechanism** than the other three
  (clock injection rather than shortening a real timer) because the rate-limit test's assertion is
  about a time-window reset, which is inherently wall-clock-bound; shortening the window makes it
  scheduling-sensitive and vacuous-prone (O3), so the clock is the correct injection point.

Because every fix is (a), no test becomes parallel, so the "isolation argument for a parallel test"
requirement does not apply. The regression guard (M3) is the tripwire that fails if an injection is
reverted.

### Conflict-surface findings (O1, applied to all three injection points)

- **Retry delay** → the existing config surface is `ExecuteOptions`, which **already carries**
  `Timeout` and `IdleTimeout` duration knobs (provider.go:55-56). `RetryBaseDelay` is a per-call
  field on that same struct, set explicitly to `time.Second` by `DefaultExecuteOptions()` (no silent
  `0 → time.Second` fallback — FIX 2). The retry **wait seam** (`Wait func(time.Duration)`, default
  `time.Sleep`) is a second per-call field on the same struct. No receiver field, no setter.
- **Poll interval** → the existing config surface is the `NewStoreBackedApprovalCheckpoint`
  constructor, which already takes `defaultTimeout` as a per-call param. `pollInterval` becomes a
  third constructor param. **Zero production callers** (V16), so no production blast radius.
- **Rate-limit window** → the existing config surface is the `NewCoordinatorEventHandler`
  constructor, which already sets `maxEventsPerSec: 10` and `maxBufferSize: 100` as hardcoded
  fields. The clock is injected via a functional option `WithClock`; the 1 production caller is
  unchanged. No setter.

No `SetX` setter is introduced anywhere. A setter on a shared object is global mutable state and is
itself a parallelism hazard — awkward in a doc about parallelism — so all injection is per-call
config at construction.

### Wait/ticker seam (FIX 1)

The retry and poll guards assert **behaviour through a test-controlled seam**, not elapsed wall
time. The controller verified there is **no existing wait/ticker seam** in this package (V23 —
NEGATIVE with a firing control), so the seam is designed here as new per-call config:

- **Retry** — `ExecuteOptions.Wait func(time.Duration)`, default `time.Sleep`. `ExecuteWithRetry`
  calls `opts.Wait(delay)` instead of `time.After(delay)`. The guard injects a `Wait` that records
  each requested delay and returns immediately; it asserts the recorded delays match the expected
  backoff (`[10ms, 20ms]`) and that `attemptCount == 3`. No wall-clock bound.
- **Poll** — `NewStoreBackedApprovalCheckpoint(store, defaultTimeout, pollInterval, tick)` where
  `tick func(time.Duration) <-chan time.Time`, default `func(i time.Duration) <-chan time.Time {
  return time.NewTicker(i).C }`. The poll loop selects on `sac.tick(sac.pollInterval)` instead of
  `time.NewTicker(sac.pollInterval).C`. The guard injects a `tick` that records the poll interval and
  returns a channel the test controls; the test **explicitly releases ticks** and asserts the status
  resolves to approved. No wall-clock bound.

Both seams are per-call config at construction (no setters, no shared mutable state), consistent
with the rate-limit clock injection.

### Milestone 1 (M1) — Production injection points (defaults preserved)

- `provider.go`: add `RetryBaseDelay time.Duration` and `Wait func(time.Duration)` to
  `ExecuteOptions`; `DefaultExecuteOptions()` sets `RetryBaseDelay: time.Second` and
  `Wait: time.Sleep`. `task_executor.go`: `ExecuteWithRetry` uses `baseDelay := opts.RetryBaseDelay`
  (no `0 → time.Second` fallback — FIX 2) and calls `opts.Wait(delay)` instead of `time.After(delay)`
  (:165). The default is set explicitly by `DefaultExecuteOptions()`, and the production caller at
  daemon_tasks_exec_run.go:238 **rebases on `DefaultExecuteOptions()`** and overrides its five fields
  (Timeout, IdleTimeout, Workspace, ObservatoryContext, AgentConfig) — each override wins over the
  default, so behavior is preserved exactly.
- `approval_checkpoint.go`: change the constructor to
  `NewStoreBackedApprovalCheckpoint(store ApprovalStore, defaultTimeout, pollInterval time.Duration,
  tick func(time.Duration) <-chan time.Time)` and use the params at :387 (`sac.tick(sac.pollInterval)`
  instead of `time.NewTicker(sac.pollInterval)`). Default `2 * time.Second` and the default ticker are
  preserved by the callers (the two test callers pass `2*time.Second` and the default tick until M2;
  there are no production callers).
- `event_handler.go`: add `now func() time.Time` field (default `time.Now` in the constructor) and a
  functional option `WithClock(now func() time.Time)`; `checkRateLimit` uses `h.now()` instead of
  `time.Now()` (:278). The rate-limit window stays `time.Second` (default preserved).

**Acceptance criteria (each a command with base and after readings):**

| Criterion | Command | Base `f3783c976` | After M1 |
|---|---|---|---|
| Builds | `go build ./internal/coordinator/` | passes | passes |
| Injection point exists (retry) | `grep -n 'RetryBaseDelay\|Wait func' internal/coordinator/provider.go internal/coordinator/task_executor.go` | no match (0) | field + default + `Wait` seam present; **no `baseDelay == 0` fallback** |
| Injection point exists (poll) | `grep -n 'pollInterval time.Duration\|tick func' internal/coordinator/approval_checkpoint.go` | no match (0) | constructor params present |
| Injection point exists (handler) | `grep -n 'now func() time.Time\|WithClock' internal/coordinator/event_handler.go` | no match (0) | field + option present |
| Defaults preserved | `grep -n 'RetryBaseDelay:.*time.Second\|Wait:.*time.Sleep\|pollInterval.*2 \* time.Second\|time.Second' internal/coordinator/provider.go internal/coordinator/task_executor.go internal/coordinator/approval_checkpoint.go internal/coordinator/event_handler.go` | n/a (no fields) | retry default `time.Second` set by `DefaultExecuteOptions()` (no fallback); poll default `2 * time.Second`; rate-limit window `time.Second` all present |
| **Rebase preserves behavior (FIX 2)** | read daemon_tasks_exec_run.go:238 | `opts := &ExecuteOptions{...}` literal | `opts := DefaultExecuteOptions()` then override Timeout, IdleTimeout, Workspace, ObservatoryContext, AgentConfig — each override wins over the default, so the five fields are identical to today |
| No behavior change | `go test ./internal/coordinator/ -run 'TestIntegration_TaskExecutorWithRetry\|TestStoreBackedApprovalCheckpoint\|TestStoreBackedApprovalCheckpoint_Rejection\|TestCoordinatorEventHandler_RateLimitReset' -count=1` | ~8.1 s, all pass | still ~8.1 s, all pass (tests not yet updated; defaults preserved) |

### Milestone 2 (M2) — Update the four tests to inject short intervals + mandatory behavioral guards

- `TestIntegration_TaskExecutorWithRetry`: add `RetryBaseDelay: 10*time.Millisecond` to the `opts`
  literal (integration_test.go:203) and inject a `Wait` seam. Backoff becomes 10 ms + 20 ms.
- `TestStoreBackedApprovalCheckpoint` / `_Rejection`: pass `10*time.Millisecond` as the third
  constructor arg and a test-controlled `tick` as the fourth (approval_checkpoint_edge_test.go:21,76).
  Poll becomes ~10 ms.
- `TestCoordinatorEventHandler_RateLimitReset`: rewrite with a controllable clock — construct with
  `WithClock(clock.Now)`, send 15 events at a frozen time, **assert throttling engaged** (see M3),
  advance the clock past the window, send one more, assert it is broadcast. **No `time.Sleep`.**

This is the milestone that actually removes the wall time. The three behavioral guards (M3) are
**mandatory with M1/M2** — they are not deferrable (FIX 1).

**Acceptance criteria:**

| Criterion | Command | Base `f3783c976` | After M2 |
|---|---|---|---|
| Four tests pass | `go test ./internal/coordinator/ -run 'TestIntegration_TaskExecutorWithRetry\|TestStoreBackedApprovalCheckpoint\|TestStoreBackedApprovalCheckpoint_Rejection\|TestCoordinatorEventHandler_RateLimitReset' -count=1` | ~8.1 s, all pass | all pass |
| Full package passes | `go test ./internal/coordinator/ -count=1` | 13.12 s, exit 0 | exit 0 |
| Behavioral guards pass (mandatory) | `go test ./internal/coordinator/ -run 'TestRetryBaseDelayInjected\|TestStoreBackedApprovalCheckpoint_PollIntervalInjected\|TestCoordinatorEventHandler_RateLimitWindowInjected' -count=1` | no such tests (0 run) | 3 pass |

The wall-time targets are **local measurements**, not CI assertions — see **Local Measurement**
below.

### Local Measurement (evidence of the win — NOT a CI assertion)

The wall-time numbers below are recorded **once by the executor on this machine** as evidence of the
win. They are **not** acceptance criteria that CI could ever run: a wall-clock upper bound on a
loaded CI runner can fail a correct implementation (FIX 1). The CI-enforced tripwire is the three
behavioral guards (M3), which assert behavior through test-controlled seams and never assert elapsed
wall time.

| Measurement | Command | Base `f3783c976` | After M2 (record once) |
|---|---|---|---|
| Four-test combined duration | `go test ./internal/coordinator/ -run '<the four>' -count=1 -json` → sum of `Elapsed` over the four top-level pass events. Instrument: `go test -json` `Elapsed`; conditions: single run, `-count=1`, on this arm64 Darwin machine, no other load. | **8.14 s** | **< 0.5 s** (target ~0.25 s) |
| Full-package wall | `time go test ./internal/coordinator/ -count=1`. Instrument: `time` wall clock; conditions: single run, `-count=1`, this machine, no other load. | **13.12 s** | **< 6 s** |

### Milestone 3 (M3) — Behavioral regression guards (MANDATORY with M1/M2)

Three white-box tests (same package `coordinator`, so they can read the unexported fields) that
**observe the injected value's effect on production behavior** through a test-controlled seam — not
the field, and **not elapsed wall time** (FIX 1). Each guard asserts that production **behaviour
changes** when the injected value changes. These are **mandatory with M1/M2**, not deferrable.

- `TestRetryBaseDelayInjected`: mock provider fails twice then succeeds; `opts.RetryBaseDelay =
  5*time.Millisecond`; inject a `Wait` that records each requested delay and returns immediately; call
  `ExecuteWithRetry(ctx, task, opts, 3)`. Assert `attemptCount == 3` (retries happened) **and** the
  recorded delays are `[5ms, 10ms]` (the injected backoff, not the reverted 1 s + 2 s). No wall-clock
  bound.
- `TestStoreBackedApprovalCheckpoint_PollIntervalInjected`: construct with `pollInterval =
  5*time.Millisecond` and a test-controlled `tick` that records the interval and returns a channel the
  test releases explicitly; start `RequestApproval`, resolve via the store, **explicitly release ticks**,
  and wait on a **cancellable, bounded failure/cleanup path** (a context with a deadline as a LIVENESS
  safeguard — not a timing-correctness assertion — that fails with a clear message if the poll never
  detects the change). Assert the status resolves to approved **and** the recorded poll interval is
  `5ms`. No wall-clock bound; no unconditional `wg.Wait()`.
- `TestCoordinatorEventHandler_RateLimitWindowInjected`: construct with `WithClock(clock.Now)`;
  send 15 events at a frozen time; **assert throttling engaged** (`IsThrottled()` true and exactly 10
  broadcast); advance the clock past the window; send one more; assert it is broadcast (11 total).
  Fully deterministic (clock injection), and **cannot go vacuous** — it fails if throttling never
  engaged (O3).

All three assert **behaviour through a test-controlled seam** (clock for the rate-limit guard,
wait/ticker for retry and poll), so none depends on real wall-clock timing and none can pass
vacuously. The post-M2 wall time and four-test combined duration are recorded once in the
**Local Measurement** section, not asserted here.

**Acceptance criteria:**

| Criterion | Command | Base `f3783c976` | After M3 |
|---|---|---|---|
| Guard tests exist and pass | `go test ./internal/coordinator/ -run 'TestRetryBaseDelayInjected\|TestStoreBackedApprovalCheckpoint_PollIntervalInjected\|TestCoordinatorEventHandler_RateLimitWindowInjected' -count=1` | no such tests (0 run) | 3 pass |
| Guard kills a reversion | apply Mutant 1 (below) in a temp copy → the guard test fails | n/a | guard test fails on the mutant |
| Full package still green | `go test ./internal/coordinator/ -count=1` | 13.12 s | exit 0 |
| Verification log updated | read `## Verification Log` of this doc | base rows only | post-M2/M3 rows present |

**Cut set if the executor runs out of time:** the three behavioral guards are **mandatory with
M1/M2** (FIX 1), so they are not the cut. The new cut set is the **Local Measurement** recording
(evidence of the win) and the **mutation table** — both can be deferred to a follow-up without losing
the latency win or the reversion tripwire. If only M1 lands, M2 is the next cut (M1 alone changes
nothing observable).

## Test Plan (kills which mutation)

Run each mutation in a bounded temp copy; keep the production tree untouched; prove each landed by
diff/hash; re-run the clean suite afterward. **Every mutant below COMPILES** (O2b) — each keeps the
injection surface but reverts the body or ignores the injected value, so a compile failure is never
what catches it. The **body-only reversion** is the mutant that matters. Each row names the **single
assertion** that kills it.

| Mutant (all compile) | Single assertion that kills it |
|---|---|
| Restore `baseDelay := time.Second` in the `ExecuteWithRetry` body (keep `opts.RetryBaseDelay` field) | `TestRetryBaseDelayInjected` recorded-delay assertion (`[5ms, 10ms]`) — reverted backoff records `[1s, 2s]` |
| Restore `pollInterval: 2 * time.Second` in the checkpoint constructor (ignore the injected param) | `TestStoreBackedApprovalCheckpoint_PollIntervalInjected` recorded-interval assertion (`5ms`) — reverted poll records `2s` |
| Restore `time.Now()` in `checkRateLimit` (ignore the injected clock) | `TestCoordinatorEventHandler_RateLimitWindowInjected` step-2 assertion (11 broadcast) — the controlled clock no longer drives the reset, so the post-advance event is still throttled |
| Make `ExecuteWithRetry` ignore `opts.RetryBaseDelay` (always `time.Second`) | `TestRetryBaseDelayInjected` recorded-delay assertion (`[5ms, 10ms]`) |
| Make the checkpoint ignore the injected `pollInterval` (always `2*time.Second`) | `TestStoreBackedApprovalCheckpoint_PollIntervalInjected` recorded-interval assertion (`5ms`) |
| Make `checkRateLimit` ignore the injected clock (always `time.Now()`) | `TestCoordinatorEventHandler_RateLimitWindowInjected` step-2 assertion (11 broadcast) |
| Change a default (e.g., `RetryBaseDelay` default to `2*time.Second`) | "Defaults preserved" M1 criterion (grep) — catches an accidental production behavior change |

**What each guard fails to catch (O2c):**

- `TestRetryBaseDelayInjected` fails to catch a mutant that changes the backoff to a different
  **but still short** value (e.g., 100 ms) — that is not a reversion to the original slow value, and
  the guard is a reversion tripwire, not a full behavioral spec. It also does not bound the retry
  count (that is `attemptCount == 3`).
- `TestStoreBackedApprovalCheckpoint_PollIntervalInjected` fails to catch a mutant that changes the
  poll interval to a different **but still short** value (e.g., 100 ms) — same reasoning.
- `TestCoordinatorEventHandler_RateLimitWindowInjected` fails to catch a mutant that changes the
  window to a **shorter** value that the advanced clock still crosses (e.g., 500 ms) — that is not a
  reversion to the original `time.Second`; a default change is caught by the "Defaults preserved" M1
  criterion. It also relies on the injected clock being honored; if a mutant makes the clock a no-op,
  the step-2 assertion fails (the controlled clock no longer drives the reset).

Record exact outcomes, not predictions; re-run the clean suite; verify the tree is unchanged.

## Risks

- **Injection changes production timing by accident.** Guarded by the "Defaults preserved" M1
  criterion (grep) and the M1 "no behavior change" run (four tests still ~8.1 s before M2). The
  retry default is set explicitly by `DefaultExecuteOptions()` (`RetryBaseDelay: time.Second`), and
  the production caller at daemon_tasks_exec_run.go:238 rebases on that constructor and overrides its
  five fields, so behavior is preserved exactly (FIX 2). There is **no silent `0 → time.Second`
  fallback**.
- **A reversion silently re-serialises the package.** Guarded by the three behavior-observing guards
  in M3 (mandatory with M1/M2), which fail on any of the Mutant 1–6 reversions.
- **Wall-time criteria flake on a loaded rig.** The wall-time targets are **local measurements**
  recorded once by the executor (see Local Measurement), **not** asserted in CI — a wall-clock upper
  bound on a loaded CI runner can fail a correct implementation, so none is a CI gate. The
  CI-enforced tripwire is the three M3 guards, which assert **behaviour through test-controlled
  seams** (clock for the rate-limit guard, wait/ticker for retry and poll) and never assert elapsed
  wall time, so a loaded rig cannot red them. The design does **not** claim load-immunity.
- **Blast radius.** Each production function is exercised by exactly one test file (V16), and the
  checkpoint constructor has zero production callers (V16), so the injection points are low-risk.

## Out-of-scope (evidence, queued as separate rows)

- **Windows CI timeout.** Iteration 348 measured `cmd/ailang` as the next package to blow; this
  design does not claim to fix it. Separate row, already tracked by iteration 348's `-timeout 416s`
  mitigation.
- **Blanket `t.Parallel()` across the 739 tests.** With 68 `t.TempDir()`, 5 `os.MkdirTemp`, 13
  `exec.Command` and real store opens (V7–V9), a blanket change is a new flake source; the repo has
  already paid for that class twice. Separate row.
- **Parallelising the other 125 tests** (each ≤ 0.17 s, combined ~4.6 s). They touch the shared-state
  surface; parallelising them safely needs per-test isolation proofs and is a distinct, riskier row.
- **The 13 `exec.Command` / 5 `os.MkdirTemp` tests.** Parallelising those is a separate row with its
  own isolation argument and mutation requirement.

## Axiom Compliance

Harness-scoped scoring; no language-support claim.

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | The rate-limit test is fully deterministic via clock injection (no wall-clock dependence). The retry and poll guards are deterministic **in outcome** — they assert behavior through a test-controlled wait/ticker seam (recorded delays/interval, `attemptCount == 3`, status resolved), so they cannot pass vacuously and do not depend on real wall-clock timing. |
| A2 Replayability | 0 | Existing replay contract preserved |
| A3 Effect Legibility | 0 | No new effects; timers are internal |
| A4 Explicit Authority | 0 | No external channels touched |
| A5 Bounded Verification | +1 | Four tests drop from 8.14 s to < 0.5 s; package wall from 13.12 s to < 6 s |
| A6 Safe Concurrency | +1 | No new `t.Parallel()`; no new shared-state exposure — all injection is per-call config (`ExecuteOptions` fields, constructor params, functional option), no setters, no shared mutable state. The rate-limit test's flake surface is **reduced** (clock injection removes wall-clock racing). The retry/poll guards assert behavior through a test-controlled wait/ticker seam, so they do not depend on real wall-clock timing and cannot pass vacuously. The design does **not** claim load-immunity: wall-time targets are local measurements, not CI assertions. |
| A7 Machines First | +1 | A reversion to hardcoded timers is caught by the behavior-observing guard tests, not silently re-serialised |
| A8 Minimal Syntax | 0 | No language change |
| A9 Cost Visibility | 0 | No billing change |
| A10 Composability | 0 | Existing test target retained |
| A11 Structured Failure | 0 | Failure contract preserved |
| A12 System Boundary | 0 | Production timing semantics unchanged (defaults preserved) |

**Net +4** (A5, A6, A7). Hard gates A1/A3/A4/A7 have no negative score.
