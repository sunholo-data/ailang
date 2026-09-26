# M-RIG-LOCK-YIELD: Cooperative Yield Requests for `rig-lock.sh`

**Status**: Implemented (2026-09-11, v0.38.x) — this document records the as-built design after the fact; the code landed without a design doc, and issue #1136 (daneel) asked for the design to be written down. No new implementation accompanies this doc.
**Target**: v0.38.x
**Priority**: P1 — rig fairness, not correctness (the lock itself is P0, M-RIG-LOCK-ENFORCE)
**Estimated**: 0 days remaining (implemented); doc-only retro
**Dependencies**: M-RIG-LOCK-ENFORCE (`design_docs/implemented/v0_25_0/m-rig-lock-enforce.md`)
**Scope**: Tooling protocol change — shell + Go, no language/compiler/stdlib surface.

## Problem Statement

`rig-lock.sh` offered exactly two modes — `wait` (block until free) and `nowait` (give up) —
with no priority. A multi-hour eval job and a 40-second interactive job were equals, and the
batch always won because it arrived first and never let go. Concrete case (issue #1136): a
~40-second local-model summarization (Daneel's mail intake) shares the single Studio GPU with
a multi-hour nightly eval. Waiting out the lock means deferring for hours; bypassing the lock
means concurrent runs thrash and an ollama model reload mid-run silently kills a stream
(the 2026-06-17 incident that motivated the lock in the first place).

**Measured 2026-09-11**: the nightly held the lock 03:00→~13:00 while Daneel's mail intake
deferred on 83% of its runs (34/41) and the os-rotation-filler got one acquisition in a day.
Nothing was broken; the protocol simply had no way to *ask* a holder to step aside.

## Approach

A cooperative yield protocol layered on the existing `mkdir` lock:

1. A short-job requester writes a **handoff marker** (a single file naming itself, its pid, and
   an expiry) — it does *not* acquire the lock.
2. The current holder **polls the marker at safe points** (see below). If one is pending, it
   releases the lock, waits for the requester to take and finish with it, then re-acquires.
3. The requester races for the lock normally, passing its own name so a guard admits it (and
   only it) through the gap the holder opened.

Both halves exist and must stay wire-compatible: the shell half in
`tools/launchd/rig-lock.sh` (`rig_yield_pending`, `rig_lock_request_yield`,
`rig_lock_clear_yield`, plus the handoff guard inside `rig_lock_acquire`) and the Go half in
`internal/riglock/yield.go` (`RequestYield`, `ClearYield`, `PendingYield`, `Checkpoint`).
A shell requester can yield to a Go holder and vice versa.

### What a safe point IS

The holder only checks for a pending yield where **the GPU is idle and nothing of its own is
in flight**. For `ailang eval-suite` that is **between benchmarks** — after one benchmark's
trials fully complete and before the next dispatches — never mid-stream.

- The live call site is the between-benchmarks hook in `cmd/ailang/eval_parallel.go`, and it
  is armed **only at `--parallel 1`**. The lock means "nobody else is driving the GPU"; with
  sibling trials still streaming, releasing it would hand out a promise that is already false.
  Every rig job runs `--parallel 1`, so the checkpoint is live exactly where it matters.
- The fast path is a single `stat` of a file that usually does not exist — cheap enough to
  call at every benchmark boundary with zero overhead when nobody is asking.

## Holder obligations and timeouts

The design principle: **an unresponsive holder must not strand the requester, and an
unresponsive requester must not strand the holder.** Both directions are bounded.

- **Holder never blocks waiting to be asked.** `Checkpoint` returns immediately when nothing
  is pending. A holder that never calls `Checkpoint` simply never yields — the requester
  falls back to ordinary contention semantics (wait it out or give up). The protocol cannot
  preempt; it can only be honored.
- **The handoff carries an expiry (`until`, RFC3339 UTC; default window 180 s, sized for a
  ~40 s GPU job plus cold model-load headroom).** After expiry the marker is void.
- **The handoff carries the requester's pid.** If the requester dies between asking and
  acquiring, the holder's poll loop (`PendingYield`, 2 s interval) sees the dead pid and
  returns to work instead of waiting out the window.
- **Expired, unparseable, or orphaned markers are removed by the next reader and reported
  absent** — a stuck marker would refuse every `nowait` acquirer until a human noticed, which
  is worse than the contention it prevents. No unbounded grants: no requester field or no
  deadline means the marker is garbage, not a guessable grant.
- **The holder re-acquires with `wait`, not `nowait`**, so the requester can finish a run
  slightly past its own clear; the whole point was to let it complete.
- **A handoff is owned, not inherited.** A second short job arriving while a handoff is in
  force gets refused (`rig_lock_request_yield` → 1; `RequestYield` → error) rather than
  silently inheriting the first one's grant — it falls back to ordinary contention.
- **The requester is obliged to clear the marker (`rig_lock_clear_yield` / `ClearYield`) as
  soon as it is done** — the holder is blocked polling for exactly that, and skipping it costs
  the holder the whole remaining window.

## State layout

```
~/.ailang/state/
  rig.lock.d/          # the lock itself: atomic mkdir, holder file "PID timestamp"
  rig.handoff          # the yield marker: OUTSIDE the lock dir, deliberately
```

**Why the marker lives outside `$RIG_LOCK_DIR`**: a holder that yields *removes* the lock
directory, so anything stored inside it would vanish at the exact moment it is needed, and the
gap would read as an ordinary free lock — which the 45-minute background filler would win.
Marker format: one line, `requester=<name> pid=<pid> until=<RFC3339 UTC>`, parsed identically
by the shell and Go halves (the shell parses the stamp as UTC in both BSD and GNU `date`
forms — a local-zone parse is a known CEST-class bug, and treating "cannot parse" as
"expired" is a known Linux-CI trap, both documented in the code).

Env overrides mirror each other: `RIG_LOCK_DIR`, `RIG_HANDOFF_FILE` (shell) /
`EnvHandoffFile` (Go).

## Fail-loud rules

- **A refused yield is reported, never retried silently**: `rig_lock_request_yield` returns
  non-zero (handoff owned by someone else) and `RequestYield` returns a named error. Callers
  surface this — the requester then queues or defers like any ordinary contended job, exactly
  as it did before this protocol existed.
- **No silent double-yield**: the Go half re-checks the marker under a mutex after deciding to
  yield; a sibling goroutine that already served the same handoff does not open a second gap
  for the filler to race into.
- **A failed re-acquire is announced**: if the holder cannot re-take the lock after yielding,
  it prints a warning instead of carrying on as if it still held a lock it does not — a silent
  false hold is how two jobs end up on one GPU.
- **Non-holder checkpoint is a no-op**: `Checkpoint` verifies `AILANG_RIG_LOCK_HELD` first, so
  a cloud-only or `--no-rig-lock` run never releases a lock it never took.
- **Unparseable marker = absent, not guessed**: no requester/deadline means removed-and-refuse,
  per the no-silent-fallback principle.

## Callers

- **Eval jobs** (`nightly-eval.sh`, `nightly-lang-eval.sh`, and `eval-suite` behind them): the
  long-holding side. They acquire `wait`, hold for the whole run, and honor yields only via
  the between-benchmarks checkpoint above. The nightly additionally carries a wall-clock
  ceiling (`NIGHT_MAX_WALL_CLOCK_HOURS`, same 2026-09-11 incident) so a multi-hour holder is
  at least a *bounded* multi-hour holder.
- **`ab_ast_autoread.sh`** and the **os-rotation-filler**: shorter holders that acquire `wait`
  / `nowait` respectively; both now also *see* the handoff guard in `rig_lock_acquire`, which
  refuses `nowait` acquirers that are not the named requester for as long as a handoff is in
  force — the gap belongs to the requester, not to whoever polls fastest.

## Acceptance criteria (as verified by the existing tests)

- Shell/Go cross-visibility: a marker written by either half is parsed and expired
  identically by the other (`yield_shell_test.go`, run on macOS and Linux CI — the platform
  divergence above is itself regression-tested).
- Dead requester pid and expired `until` both release the holder promptly.
- A second requester cannot overwrite a live handoff; the filler cannot win a gap opened for
  a named requester.
- `Checkpoint` at `--parallel 1` yields between benchmarks and never mid-stream; at any other
  parallelism it is inert.

## Known limitations

- The protocol is cooperative: a holder with no safe points (e.g. a single 6-hour benchmark)
  never yields. The wall-clock ceiling and per-bench token caps bound that case instead.
- One handoff at a time; queued short jobs are not FIFO-served, they re-race after refusal.