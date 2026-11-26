# Iteration 141 — landing evidence

## Rebase (stack onto origin/dev 63b605cbc)
- range-diff 2da0c4592..05e81cc48 vs origin/dev..2c9273918: 17 lines, 17 '=' (17 commits, all patch-identical)
- stack files (59) ∩ dev files changed since 2da0c4592 (205): 0
- after #2463 squash (25cc5bc42): diff 86daac1f2..dev = 0 lines; rebase --onto replayed i138+i140; tree diff vs judged heads: i138 0, i140 0
- after #2466 squash (fcc3d3481): diff a055ea325..dev = 0 lines; rebase --onto replayed i140; tree diff vs judged head 2c9273918: 0
- after #2471 squash (92bc214ba): diff 2c9273918..dev = 0 lines (dev tree == judged tree)

## PR CI (SHA-pinned pollers, full check set)
- PR #2463 GREEN for 86daac1f2ef52c3dd0e3fa037d4b9cbd3164c7a9 (total=23): test=completed/success lint=completed/success build=completed/success docs-gate=completed/success checks=completed/success
- PR #2466 GREEN for d4ae955cd1c0fe9b4cbb899ded97712557e070d2 (total=23): test=completed/success lint=completed/success build=completed/success docs-gate=completed/success checks=completed/success
- PR #2471 GREEN for 2c9273918127ede2ccbbf20cde7070e0221ab8e7 (total=23): test=completed/success lint=completed/success build=completed/success docs-gate=completed/success checks=completed/success
- PR #2466 GREEN for a055ea32509c7fdae1b439879b763ec9e033f03f (total=23): test=completed/success lint=completed/success build=completed/success docs-gate=completed/success checks=completed/success
- pr2466 ALL COMPLETE for a055ea32509c7fdae1b439879b763ec9e033f03f total=24 not-green=[]
- pr2471c ALL COMPLETE for fab01494371534da1f0b8a0d6fc256dcf2bb444e total=24 not-green=[Build macos-latest=failure]

## Dev CI
- dev25cc-b ALL COMPLETE for 25cc5bc4249f875caff23c02d9d4b3392d9b671a total=18 not-green=[]
- devfcc ALL COMPLETE for fcc3d3481d276a5cefc15c4abaedb04dc313eb38 total=17 not-green=[checks=failure, test=failure]  (attempt 1)
  - checks: step 11 Install z3, curl 403 from the z3 release download; same step green on fab014943 (00:25:47Z), 25cc5bc42 and a055ea325
  - test: internal/pipeline package timeout 6m56s (running TestTyvarHygiene_StdExportSchemesAreClosed; gatedReader in pipeline_stderr_capture_test.go); same class as iteration 139's df96971d4
  - stack files touching internal/pipeline|internal/riglock|install_z3: 0 (control pattern cmd/ailang|tools/launchd: 5)
- PR #2471 Build macos-latest (non-required): internal/riglock TestAcquireAs_QueuedWaiterCannotTakeTheGap (reclaim_test.go:131), timing test; stack does not touch internal/riglock

## Done-gate dry-runs (stapledon profile, idle; AILANG_DRIVER_PINNED=92bc214ba; env -i)
- healthy: [2026-11-26 01:42:54] driver pin: running committed origin/dev @ 92bc214badb719081b9aca61c6510bc950e67bfb with mission work repository /Users/voightkampff/dev/sunholo-data/stapledons-godot (source clo
- healthy: [2026-11-26 01:44:36] DRY RUN ok: mission=stapledon repo-slug=sunholo-data/stapledons-godot doc=design_docs/stapledon-mission.md workdir=/Users/voightkampff/dev/sunholo-data/stapledons-godot pidfile=/
- healthy rc=0 (102 s)
- degraded: [2026-11-26 01:45:11] codex model 'bogus-model-i141' unusable: probe failed (rc=1)
- degraded: [2026-11-26 01:45:18] codex executor lane -> falling back to 'claude:claude-haiku-5-5' for this fire (model 'bogus-model-i141')
- degraded: [2026-11-26 01:46:27] DRY RUN ok: mission=stapledon repo-slug=sunholo-data/stapledons-godot doc=design_docs/stapledon-mission.md workdir=/Users/voightkampff/dev/sunholo-data/stapledons-godot pidfile=/
- degraded (MISSION_EXECUTOR_MODEL=codex:bogus-model-i141) rc=0 (104 s); no pidfile left after either run
