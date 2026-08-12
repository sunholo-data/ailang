# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-13 ~02:10 local (iteration 188)

## Now
- **v0.33.0** · `dev` @ `996bcccd7`, CI green (SHA-addressed `checks=16`, zero not-green).
  **No code shipped this iteration** — the deliverable is a design doc that parked at the quorum
  gate, plus the measurements that park it well.
- 🟡 **`m-driver-pin-rollout` (`#558`)**: its `[BLOCKED-ON-EVIDENCE]` gate is **DISCHARGED**
  (fires 185/186/187 each pinned + completed; `DRIVER PIN FAILED` = 0 with a firing control).
  Doc landed at 434 lines / 24 verification rows. **Quorum BLOCKED ×2**, both reviewers present
  both rounds, `metered=$0.1778`.
- 🔴 **The rollout would delete its own delivery mechanism.** `os-rotation-filler.sh` `git pull`s
  the source clone every 45 min (`:197/398/426/458`) — that is *why* the clone reads 0 behind —
  and M2 pins that very filler, after which no later milestone's script edits reach the rig.
  Second finding: `os-release-snapshot.sh` / `publish-unified-dashboard.sh` root themselves from
  **`$0`, not cwd**, so the doc's durable-data strategy cannot work as designed.
- ⚠️ **`make test-launchd-drivers` is red at base inside the loop**: 9 passed/26 failed rc=2 from a
  pinned session vs 17/0 rc=0 with the driver env sanitized. CI is green only because CI is clean.
- ⚠️ **Averted rig hazard**: `mission-recovery.plist` injects `MISSION_NAME=v1` at a 240s interval,
  so the pin helper's default dir would have `checkout --force`d the *live V1 mission worktree*
  every four minutes under a running iteration.

## Next
1. **`D-13`** gates `#558` — one word (A/B/C), see below.
2. If `D-9`/`D-10`/`D-12`/`D-13` all stay open: `[SWEEP iter-158]` external-issue triage batch
   (P3, ~0.5d), then `m-dialect-keyword-diagnostics` (`#539`, NEW-DOC + quorum).

## Loop health
- Cadence ~2h; 4 consecutive pinned fires, **zero** dead slots since iter-184.
- Gate-1 skill `cmp` **silent** for the first time in 4 iterations — the divergence iter-187 closed
  is holding (running skill == `origin/dev`, 194,209 B both sides).
- Routing: controller **opus** · designer **`claude:claude-fable-5`** (rotation, 2 bounded passes).
  Planner/executor/evaluator not fired — no sprint existed. `metered=$0.1778` of a `$5` ceiling.

## Parked on Mark — all on issue #635
`D-1` (#613) · `D-2` (#604/#614) · `D-7` (pi/codex executor) · `D-8` (#618 rig rollout) ·
`D-9` (#619 scope) · `D-10` (#616 fix site) · `D-11` · `D-12` (auto-row on human unblock) ·
**`D-13` (NEW)** — `#558` scope, one word:
**(A)** exclude `os-rotation-filler.sh`, keep it as the source-clone updater;
**(B)** move the source-clone fast-forward into `pin-root.sh`;
**(C)** keep scope, re-root durable writes through `$AILANG_DRIVER_SRC`.

## Quota posture
Subscription buckets only for controller/designer; metered spend is quorum reviewers alone.
Billing tripwire CLEAN (re-checked inside the tool shell, not just at preflight).
