# Iteration 135 gates (controller, outside the sandbox)

## Ticket repro, pre-fix hook (f7d54a9c7)
```
ARM rebased-record force-push rc=1 (mission scope guard: MISSION_NAME=triage may not push design_docs/fleet-mission-log.md (loop-harness path; file it with: ailang mission ticket file --signature <key> --evidence <...>) mission scope guard: push refused (M-HARNESS-MISSION-LOOP). Tickets go to the mission-fleet inbox. error: failed to push some refs to '<tmp>/o.git' )
three-dot own paths: design_docs/triage-mission-log.md 
ARM own-harness-path rc=1 (To <tmp>/o.git  ! [rejected]        rec -> rec (non-fast-forward) error: failed to push some refs to '<tmp>/o.git' hint: Updates were rejected because the tip of your current branch is behind hint: its remote counterpart. If you want to integrate the remote changes, hint: use 'git pull' before pushing again. hint: See the 'Note about fast-forwards' in 'git push --help' for details. )
```

## Ticket repro, fixed hook (18acb95b7)
```
ARM rebased-record force-push rc=0 ()
three-dot own paths: design_docs/triage-mission-log.md 
ARM own-harness-path rc=1 (mission scope guard: MISSION_NAME=triage may not push tools/launchd/x.sh (loop-harness path; file it with: ailang mission ticket file --signature <key> --evidence <...>) mission scope guard: push refused (M-HARNESS-MISSION-LOOP). Tickets go to the mission-fleet inbox. error: failed to push some refs to '<tmp>/o.git' )
```

Note: the second arm is a fast-forward after the first push, so the old hook's own-harness arm reads a non-fast-forward rejection instead; the fixed hook's arm is the guard refusal.

## Suites at dd9c8222a (tree-identical to merge 18acb95b7)
- `/bin/bash tools/launchd/test_mission_scope_guard.sh`: rc 0, 8 PASS
- `shellcheck` on hook and test: rc 0 (base f7d54a9c7: rc 1, 3 warnings)
- `make test-launchd-drivers`: rc 0, all 60 suites passed
- temp files left (`mission-scope.*` in /tmp and $TMPDIR): 0
- `git ls-remote` against the real GitHub remote returned refs/heads/dev equal to local origin/dev (f7d54a9c7)

## Done-gate dry-runs (stapledon profile, idle and armed; AILANG_DRIVER_PINNED=18acb95b7)
- healthy: rc 0, 44 s
- degraded (MISSION_EXECUTOR_MODEL=codex:bogus-model-i135): rc 0, 41 s; probe rc 1, executor fell back to claude:claude-haiku-5-5
- no stapledon pidfile left

## Base make test-launchd-drivers
Started on the executor worktree before the executor launched and overlapped its edits; read 1 of 60 suites FAILED (the scope-guard suite, mid-edit). Void as a base reading.
