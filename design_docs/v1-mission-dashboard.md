# V1 Mission Dashboard

*Snapshot, overwritten each iteration — history lives in the charter STATUS block and the log.*
*Last written: 2026-09-03, iteration 325.*

## Where we are
- **Latest release**: v0.34.0. **Goal distance: N = 12 design docs remaining before v1.0.0** — unmoved this iteration (HARNESS). Ledger 54 rows, **0 OPEN**.
- **dev CI**: GREEN on required contexts at `5e860afeb` (checks=20). Standing non-required red: `SonarCloud`, INHERITED on every walked-back commit (queue row `sonarcloud-new-code-gate-red`).

## This iteration (325)
- **`m-spawn-pin-enforcement` is COMPLETE, 4/4 milestones.** M3 (`11aff5819`) exports `MISSION_CONTROL_ACTIVE=1` + `MISSION_<ROLE>_RESOLVED/_PATH` **after** the driver's lane degradation, so the published plan is the verified one; M4 (`e21c3f1bd`) points Gate 3 at `resolve-role-spawn.sh` and requires `MISSION-ROLE:` on every role prompt. Routing suite 45 → 51 arms. Judge sonnet **PASS 94/100, zero blocking**.
- **THE HOOK IS NOW ARMED.** From the next fire on, a mission-session Agent/Task spawn with no `MISSION-ROLE:` line is DENIED at the tool boundary; `subagent_type: Explore` is the one read-only exception. A denial naming the fix is the design working, not a regression.
- **Finding**: the plan's A3.2 — its only criterion aimed at M3's production code running — is vacuous *unconditionally* (dry-run `exit 0` at line 858, Layer-3 block at 1008), proven by running it against the base script for byte-identical output. Arm D2 is the real coverage. The plan's A4.4/S2 grep could never match either (line-wrapped literal); corrected before routing.

## Next picks
1. **First ARMED fire** — bookkeeping-light; watch whether a stale-skill controller gets denied with a reason that names the fix.
2. `m-ci-serial-gate-masking` — one early red hid 45 gates for a day (iter-323 finding).
3. `m1b-nolint-suppression-owed` — debt with a named owner.
4. `m-messages-send-type-misfiled` **[WORLD-DEMAND]** — `messages send --type` binds to `Category`, not `MessageType`; confirmed first-party. Approvals still reach Discord (routes on `ToInbox`), so this is aggregation/routing, not a lost human channel.

## Loop cadence + routing
- Controller **opus** (session). Executor **`codex:gpt-5.6-sol`** — lane UP this fire (probe rc=0, against iteration 324's 404). Evaluator **sonnet**, own worktree. **No designer, no planner**: both artifacts already existed, so spawning either would have re-authored a quorum'd document.
- **Metered this iteration: $0.00** of the $5 ceiling — every lane a quota bucket, no quorum ran.
- Friction logged: a `2>/dev/null` on my own `git add` swallowed a fatal pathspec error and produced a record commit containing no record (1st instance; caught by `git show --stat`).

## Parked on Mark
- none.
