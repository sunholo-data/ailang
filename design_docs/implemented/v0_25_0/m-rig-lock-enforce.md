# M-RIG-LOCK-ENFORCE: Enforce the Rig Lock in `ailang eval-suite`

**Status**: Implemented (v0.25.x, 2026-06-17)
**Target**: v0.25.x
**Priority**: P0 incident follow-up (rig hang, 2026-06-17)
**Estimated**: ~120 LOC + tests; <0.5 day

## Problem

The rig is a single GPU / bandwidth-bound box. Two eval jobs hitting ollama at
once thrash, and an ollama model reload mid-request silently kills a stream —
which, on the `/v1` tool-calling path, hung a live motoko run for **~1h54m** and
wedged the whole rotation (incident 2026-06-17).

A `rig-lock.sh` (atomic `mkdir` lock at `~/.ailang/state/rig.lock.d`) already
exists, but it was **opt-in and shell-only**: only the launchd jobs
(nightly-eval, nightly-lang-eval, os-rotation-filler) sourced it. An ad-hoc
`ailang eval-suite` from a shell took **no** lock, so it freely collided with the
rotation. The safety depended on a human (or agent) remembering to source a
script — which is exactly how the incident happened.

## Approach

Make the command we already run enforce the lock itself, and fail fast.

1. **`internal/riglock`** — Go port of `rig-lock.sh` with identical semantics:
   same dir (`RIG_LOCK_DIR` / `~/.ailang/state/rig.lock.d`), same staleness steal
   (`RIG_LOCK_STALE_MIN`, default 360 min), holder file `"PID timestamp"`.
   `Acquire(NoWait|Wait)` → `(acquired, release, err)`; `Holder()` for the
   busy message.
2. **`ailang eval-suite`** — after flag parse, `riglock.Acquire(NoWait)`. If held
   by another job, print the holder and `exit 1` with guidance; else `defer
   release()`. `--dry-run` (no GPU) and `--no-rig-lock` (isolated box) bypass.
3. **No double-lock with the wrappers.** `rig-lock.sh` now `export`s
   `AILANG_RIG_LOCK_HELD=1` after acquiring; `riglock.Acquire` treats that as
   "an ancestor owns it" and becomes a no-op. So a launchd-driven eval-suite
   skips its own acquire instead of deadlocking against its parent's lock. The
   sentinel is exported, so it propagates to nested helper scripts too.

## Acceptance criteria
- [x] `ailang eval-suite` fails fast with the holder identity when the rig lock
      is held (verified: `PID 99999 since … holds the rig lock`, exit 1).
- [x] `--dry-run` bypasses the lock (verified: prints plan with lock held).
- [x] `AILANG_RIG_LOCK_HELD=1` makes acquire a no-op (unit:
      `TestAcquire_AncestorHeldIsNoop`) → launchd wrappers don't break.
- [x] Stale locks (>window) are stolen (unit: `TestAcquire_StealsStaleLock`).
- [x] All 4 scheduled rig scripts source `rig-lock.sh` (sentinel exported);
      ad-hoc helper scripts correctly fail-fast when the rig is busy (intended).
- [x] `go build ./...`, `go vet`, `go test ./internal/riglock/` green.

## Out of scope
- A `Wait`-mode flag for eval-suite (block-until-free). NoWait fail-fast is the
  right default for ad-hoc runs; scheduled jobs already wait via the shell.
- Locking other GPU commands (`ailang run --ai ollama`) — eval-suite is the
  contention source that mattered; revisit if a non-eval path collides.

## References
- Incident + root cause: [motoko-harness-analysis-log.md](../../motoko-harness-analysis-log.md)
- Pairs with the `/v1` HTTP-timeout fix (commit 63fc63e0) — defense in depth:
  the timeout bounds a hang if contention slips through; the lock prevents the
  contention in the first place.
- Shell lock: `tools/launchd/rig-lock.sh`.
