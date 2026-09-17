# Nightly-eval holds the rig lock all night — widen the yield ask from os-rotation-filler to nightly-eval

- **Date**: 2026-09-15
- **Class**: already-covered
- **Recommend**: duplicate-of design_docs/implemented/v0_38_0/m-rig-lock-yield.md
- **Searched**: `rig_lock` (design_docs/, tools/), `auto-merge|autoMerge` (design_docs/), plus direct reads of `tools/launchd/nightly-eval.sh`, `tools/launchd/rig-lock.sh`, `internal/riglock/yield.go`, `cmd/ailang/eval_parallel.go`
- **Existing coverage**: `design_docs/implemented/v0_38_0/m-rig-lock-yield.md` (Implemented 2026-09-11, v0.38.x)

<The report's overnight measurement (10–11 Sept, nightly-eval holding 01:00→05:07Z with 52
deferred Daneel intake runs) describes exactly the incident M-RIG-LOCK-YIELD was built for
— its Problem Statement cites the same day ("the nightly held the lock 03:00→~13:00 …
Daneel's mail intake deferred on 83% of its runs"). The ask "release the lock between
benchmarks and re-acquire" is already implemented and documented as-built: the cooperative
yield protocol lives in `tools/launchd/rig-lock.sh` (`rig_yield_pending`,
`rig_lock_request_yield`, `rig_lock_clear_yield`) and `internal/riglock/yield.go`
(`RequestYield`/`ClearYield`/`Checkpoint`), and the holder-side checkpoint is wired into the
eval path the nightly actually runs — `cmd/ailang/eval_parallel.go` (`riglock.Checkpoint`)
yields between benchmarks at `--parallel 1`, which is how every rig job runs. The design
doc's Callers section names `nightly-eval.sh` explicitly as a long-holding side that honors
yields via that checkpoint, and the nightly additionally carries `NIGHT_MAX_WALL_CLOCK_HOURS`
as a bound. So no new work is warranted on the ask as stated.

One residual worth a glance, not a sprint: the incident predates the implementation
landed 2026-09-11, so it does not falsify the fix — but whether the deployed nightly
binary/wrapper actually contains the checkpoint path has not been verified since. A cheap
confirmation (one overnight log showing a `yielded the rig for …` line) would close the
loop; if that line never appears, the gap is wiring/deployment, not design.>