# Iteration 122 gates and done-gate

## Gate 3b
- PR #2419 head `ae343722d8d69a879588f64d89b6b204d5f469cb`: SHA-pinned check-runs poll (30 min cap), final read total=24 pending=0 notgreen=0. Direct re-read: checks=24, 0 not green; MERGEABLE CLEAN; 0 closing keywords.
- Merged with `--match-head-commit ae343722d…` → `d2f2ec238550dd02f9b3912fc46249d86b658726` (origin/dev == merge at the Gate-3b base read, 2026-11-18T19:05:09Z).
- Dev `d2f2ec238`: total=18 pending=0 notgreen=0; `CI` and `Build and Release` completed/success at d2f2ec238.

### PR poll log
```
18:54:21 total=22 pending=7 notgreen=0
18:54:52 total=22 pending=7 notgreen=0
18:55:22 total=22 pending=7 notgreen=0
18:55:53 total=22 pending=7 notgreen=0
18:56:24 total=22 pending=7 notgreen=0
18:56:55 total=22 pending=7 notgreen=0
18:57:26 total=22 pending=6 notgreen=0
18:57:57 total=23 pending=5 notgreen=0
18:58:28 total=23 pending=5 notgreen=0
18:58:59 total=23 pending=5 notgreen=0
18:59:29 total=23 pending=5 notgreen=0
19:00:00 total=23 pending=5 notgreen=0
19:00:31 total=23 pending=4 notgreen=0
19:01:02 total=23 pending=4 notgreen=0
19:01:33 total=23 pending=4 notgreen=0
19:02:04 total=23 pending=4 notgreen=0
19:02:35 total=23 pending=4 notgreen=0
19:03:06 total=23 pending=4 notgreen=0
19:03:37 total=23 pending=4 notgreen=0
19:04:08 total=23 pending=2 notgreen=0
19:04:39 total=24 pending=0 notgreen=0
```
### Dev poll log
```
19:05:09 total=11 pending=11 notgreen=0
19:05:40 total=16 pending=13 notgreen=0
19:06:11 total=16 pending=12 notgreen=0
19:06:41 total=16 pending=11 notgreen=0
19:07:12 total=16 pending=11 notgreen=0
19:07:43 total=16 pending=10 notgreen=0
19:08:13 total=16 pending=10 notgreen=0
19:08:44 total=16 pending=10 notgreen=0
19:09:15 total=16 pending=10 notgreen=0
19:09:45 total=16 pending=10 notgreen=0
19:10:16 total=16 pending=9 notgreen=0
19:10:47 total=16 pending=9 notgreen=0
19:11:18 total=16 pending=9 notgreen=0
19:11:48 total=16 pending=8 notgreen=0
19:12:19 total=16 pending=8 notgreen=0
19:12:50 total=16 pending=8 notgreen=0
19:13:20 total=16 pending=8 notgreen=0
19:13:51 total=16 pending=7 notgreen=0
19:14:22 total=16 pending=7 notgreen=0
19:14:53 total=16 pending=6 notgreen=0
19:15:23 total=16 pending=6 notgreen=0
19:15:54 total=16 pending=6 notgreen=0
19:16:25 total=16 pending=6 notgreen=0
19:16:55 total=16 pending=6 notgreen=0
19:17:26 total=16 pending=6 notgreen=0
19:17:57 total=17 pending=5 notgreen=0
19:18:28 total=17 pending=5 notgreen=0
19:18:58 total=17 pending=5 notgreen=0
19:19:29 total=17 pending=4 notgreen=0
19:20:00 total=17 pending=4 notgreen=0
19:20:31 total=17 pending=3 notgreen=0
19:21:02 total=17 pending=1 notgreen=0
19:21:32 total=17 pending=1 notgreen=0
19:22:03 total=18 pending=1 notgreen=0
19:22:34 total=18 pending=0 notgreen=0
```

## Scope guard (pinned copy at 18e848db6, before push)
- `MISSION_NAME=fleet`: rc 0, all 19 paths admitted. Control `MISSION_NAME=v1`: rc 1, refuses `tools/launchd/sync-agents-skills.sh` and `tools/launchd/test_agents_skills_sync.sh` (existing harness rule).

## Done-gate (at d2f2ec238, judge worktree moved to the merge after the judge finished)
- Surface: the skills, sync script and test that every mission runs from the pinned `origin/dev`.
- Reach: merged on origin/dev (`d2f2ec238`).
- Dry-run profile: stapledon was mid-fire (pid alive, fire started ~2 min earlier), world mid-fire, triage exits at its idle fingerprint, v1/docs/motoko disabled. So `MISSION_PROFILE=fleet` under a synthetic HOME (charter fallback), `AILANG_WORKTREE_SWEEP=0`, `AILANG_DRIVER_PINNED=d2f2ec238…`, `MISSION_DRY_RUN=1`, `env -i`, `/bin/bash`.
  - Healthy: rc 0, 87 s, `lanes=DEGRADED(2)` (natively: both codex lanes probe rc 1, see below), roles designer=claude:claude-opus-5-5 planner=opus executor=claude:claude-haiku-5-5 evaluator=sonnet.
  - Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i122`): rc 0, 93 s, `codex model 'bogus-model-i122' unusable: probe failed (rc=1)` → executor falls back to `claude:claude-haiku-5-5`.
  - Side effects: `~/.ailang-driver-pin/*/` 40 before and 40 after (diff empty); no pidfile in the synthetic state dir; judge worktree clean.
- `make test-launchd-drivers` at d2f2ec238: rc 0, 330 s, `run-suites: all 58 suites passed`, bash 3.2 syntax OK.

## Codex probe failure text (healthy dry-run, verbatim tail)
```
Error: Codex daemon turn failed: {"type":"error","status":400,"error":{"type":"invalid_request_error","message":"The 'gpt-6.2-sol' model is not supported when using Codex with a ChatGPT account.","code":"model_not_found"}}
```
