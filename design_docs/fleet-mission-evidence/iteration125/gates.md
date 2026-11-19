# Iteration 125 — gates, merge and done-gate

## PR #2411 checks (controller-pushed head `05a2b065e1f1b848450b1c67e865834e93c4b686`)
- SHA-pinned check-runs poll, 60 min cap, under /bin/bash. Final: `PR checks COMPLETE for 05a2b065e1f1b848450b1c67e865834e93c4b686: total=24 bad=0`.
- Merged by the attended session at 2026-11-19T15:42:35Z (`gh` events: `merged sunholo-voight-kampff`), about 10 s after the last check completed, at the same head. Merge commit `224f1257f95f48dd3271f1c3f1db7dc3adde82fd`.

### PR poll log
```
15:14:00 total=13 pending=10 bad=0
15:14:31 total=19 pending=14 bad=0
15:15:02 total=19 pending=13 bad=0
15:15:33 total=19 pending=12 bad=0
15:16:04 total=19 pending=12 bad=0
15:16:35 total=19 pending=11 bad=0
15:17:05 total=19 pending=11 bad=0
15:17:36 total=19 pending=11 bad=0
15:18:07 total=19 pending=10 bad=0
15:18:38 total=19 pending=10 bad=0
15:19:09 total=19 pending=10 bad=0
15:19:39 total=20 pending=9 bad=0
15:20:10 total=20 pending=9 bad=0
15:20:41 total=20 pending=9 bad=0
15:21:12 total=20 pending=9 bad=0
15:21:43 total=20 pending=9 bad=0
15:22:14 total=20 pending=9 bad=0
15:22:44 total=20 pending=9 bad=0
15:23:15 total=20 pending=9 bad=0
15:23:46 total=20 pending=7 bad=0
15:24:17 total=20 pending=7 bad=0
15:24:48 total=20 pending=7 bad=0
15:25:19 total=20 pending=6 bad=0
15:25:49 total=20 pending=6 bad=0
15:26:20 total=20 pending=6 bad=0
15:26:51 total=20 pending=6 bad=0
15:27:22 total=20 pending=6 bad=0
15:27:53 total=20 pending=5 bad=1
15:28:24 total=20 pending=4 bad=1
15:28:55 total=20 pending=4 bad=1
15:29:25 total=20 pending=4 bad=1
15:29:56 total=20 pending=3 bad=1
15:30:27 total=20 pending=3 bad=1
15:30:58 total=9 pending=2 bad=0
15:31:29 total=22 pending=2 bad=0
15:31:59 total=22 pending=2 bad=0
15:32:30 total=22 pending=2 bad=0
15:33:01 total=22 pending=2 bad=0
15:33:32 total=22 pending=2 bad=0
15:34:03 total=22 pending=2 bad=0
15:34:33 total=22 pending=2 bad=0
15:35:04 total=22 pending=2 bad=0
15:35:35 total=22 pending=2 bad=0
15:36:06 total=22 pending=2 bad=0
15:36:37 total=22 pending=2 bad=0
15:37:08 total=22 pending=2 bad=0
15:37:39 total=22 pending=2 bad=0
15:38:09 total=22 pending=2 bad=0
15:38:40 total=23 pending=1 bad=0
15:39:11 total=23 pending=1 bad=0
15:39:42 total=23 pending=1 bad=0
15:40:13 total=23 pending=1 bad=0
15:40:44 total=23 pending=1 bad=0
15:41:14 total=23 pending=1 bad=0
15:41:45 total=23 pending=1 bad=0
15:42:16 total=24 pending=1 bad=0
15:42:47 total=24 pending=0 bad=0
```

## Gate 3b — dev at the merge `224f1257f`
- `CI for 224f1257f95f48dd3271f1c3f1db7dc3adde82fd (run 44557862889): completed success`; Build and Release completed/success at `224f1257f`.
- check-runs at the merge: 21 total; one not green, `ailang-core-dev (ailang-multivac-deploy)` failure — see below.
```
rid=44557862889
15:43:19 in_progress 
15:43:51 in_progress 
15:44:21 in_progress 
15:44:53 in_progress 
15:45:23 in_progress 
15:45:54 in_progress 
15:46:25 in_progress 
15:46:56 in_progress 
15:47:27 in_progress 
15:47:58 in_progress 
15:48:29 in_progress 
15:49:00 in_progress 
15:49:31 in_progress 
15:50:02 in_progress 
15:50:33 in_progress 
15:51:04 in_progress 
15:51:35 in_progress 
15:52:06 in_progress 
15:52:38 in_progress 
15:53:09 in_progress 
15:53:40 in_progress 
15:54:11 in_progress 
15:54:42 in_progress 
15:55:13 in_progress 
15:55:44 in_progress 
15:56:15 in_progress 
15:56:46 in_progress 
15:57:17 in_progress 
15:57:48 in_progress 
15:58:19 in_progress 
15:58:50 in_progress 
15:59:21 in_progress 
15:59:52 in_progress 
16:00:23 completed success
```

## Deploy red at the merge — concurrent dev deploys raced (not code)
- Cloud Build `d377c0fd-f8a6-4bcd-b177-bcdbd026fb5d` (europe-west3, project ailang-multivac-deploy): every build and test step SUCCESS (incl. `test-agent-pi`, `test-resident-pi`); only step 22 `deploy-services` failed.
- Error, five jobs (`ailang-dev-agent-executor`, `-codex`, `-codex-go`, `-opencode`, `-pi`): `(gcloud.run.jobs.update) ABORTED: Conflict for resource … version '…' was specified but current version is '…'`.
- Windows: f902e4abd 15:20:45Z→15:31:59Z success; **224f1257f 15:42:39Z→15:53:58Z failure; c5fd312a0 (#2141) 15:43:11Z→15:53:59Z failure** — two builds 32 s apart updated the same Cloud Run jobs. Negative control: the parent f902e4abd and 7d483e8f5, 8c9b83a73, 43fba158b are all success. Outside fleet scope (deploy pipeline); not ticketed; reported.

## Done-gate at `224f1257f` (judge worktree moved to the merge)
- Sibling states: v1/docs/motoko disabled; world armed, no pidfile (idle) → `MISSION_PROFILE=world` per the charter.
```
healthy rc=0 secs=85
degraded rc=0 secs=97
pins before=      48 after=      48 diffrc=0
no world pidfile
test-launchd-drivers rc=0 secs=324
```
- Degraded arm (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i125`): `codex model 'bogus-model-i125' unusable: probe failed (rc=1)` → `codex executor lane -> falling back to 'claude:claude-haiku-5-5'`; both arms end `DRY RUN ok: mission=world`.
- `make test-launchd-drivers`: `run-suites: all 58 suites passed`; `launchd drivers: tests + bash 3.2 syntax OK`.

## Controller checks on the judge
- `node --test .unowned-dirty.test.ts` at `d2ee7dc03`: rc 0, 8/8.
- Core-fix mutation (`unownedDirty(all, ownFiles)`): rc 1, 6 pass / 2 fail — matches the judge's M1. Restored: rc 0; worktree clean.
- `make verify-pi-assets`: rc 0; `cmp` of the extension against `cmd/ailang/pi_assets/unowned-dirty.ts` at `d2ee7dc03`: identical (4,631 B each); dev's asset differed before (control).

## D-FLEET-47 smoke reading (ruled A attended 2026-11-19)
- `~/.codex-missions/packages/app-server-daemon/current` → `releases/0.169.1-aarch64-apple-darwin` (relinked 14:19Z); `dev.ailang.codex-mission-owner` restarted 14:19:12Z.
- This fire's driver probe of `codex:gpt-6.2-sol` passed: no `unusable` line, and the fire banner kept `planner=codex:gpt-6.2-sol executor=codex:gpt-6.2-sol` (`MISSION_ROUTING_NOTE` empty). Control: the 08:38Z fire logged `codex model 'gpt-6.2-sol' unusable: probe failed (rc=1)`.
- Scope: one fleet probe via the driver's own probe function. Quorum seat and sibling missions not re-measured.
