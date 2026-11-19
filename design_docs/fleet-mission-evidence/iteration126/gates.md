# Iteration 126 — gates, merge and done-gate

## PR #2431 checks (head `ad2d4196edf3dee46ef8a479891c1ab966a1c51c`)
```
PR CI for ad2d4196edf3dee46ef8a479891c1ab966a1c51c: total=24 pending=0 notgreen=[]
```
- Merged `gh pr merge 2431 --merge --match-head-commit ad2d4196e…` → `ff08818ae344971ddd9ca6de5d69a2a3fc7d8e92`.

## Dev CI at the merge
```
dev CI for ff08818ae344971ddd9ca6de5d69a2a3fc7d8e92: total=18 pending=0 CIworkflow=[completed success] notgreen=[]
Build and Release: ff08818ae completed success
```

## Controller mutation re-run at `ad2d4196e` (rc=0)
```
MUTATION excludes KILLED: FAIL record-only design doc commit moved hash
MUTATION exclude-root-verification KILLED: FAIL record-only verification commit moved hash
MUTATION exclude-dd-verification KILLED: FAIL record-only design_docs/verification commit moved hash
MUTATION exclude-dd-broad KILLED: FAIL non-iter design_docs/verification commit did not move hash
MUTATION wrong-record KILLED: FAIL unchanged: FELLTHROUGH CALLS=0 kept=yes rows=0 
MUTATION keep-changed KILLED: FAIL changed: FELLTHROUGH CALLS=1 kept=yes rows=0 |world: inputs changed since the last no-pick fire (00000000
MUTATION fail-closed KILLED: FAIL failed read error: EXIT 0 CALLS=1 kept=no rows=0 
MUTATION age KILLED: FAIL expired (1795041965): EXIT 0 CALLS=1 kept=yes rows=1 |world: nothing changed since the last no-pick fire (00000000
MUTATION limit KILLED: FAIL failed read accepted rc=0: efcc15e7b05724c1a310d7f8b614e7cf0047e6b2a0c46edf106a4c35e1f72034 head=c14f023 ledger=
MUTATION pending KILLED: FAIL green/pending null moved hash
MUTATION store-pin KILLED: FAIL good helper: idle-fingerprint: inbox read failed: store not pinned: local
MUTATION dry-run KILLED: FAIL dry run touched the idle record: EXIT 0 CALLS=1 kept=yes rows=1 |world: nothing changed since the last no-pick
MUTATION fire-check KILLED: FAIL another fire recorded this snapshot
MUTATION record-as-snapshot KILLED: FAIL record without a snapshot
MUTATION snapshot-keeps-stale KILLED: FAIL stale pending not cleared before the snapshot
```

## Done-gate at `ff08818ae`
- Sibling states: v1/docs/motoko disabled; world armed **mid-fire** (pid 82390); stapledon armed, no pidfile → `MISSION_PROFILE=stapledon`.
```
healthy rc=0 secs=64
degraded rc=0 secs=66   (MISSION_EXECUTOR_MODEL=codex:bogus-model-i126)
worktrees before=140 after=140
no stapledon pidfile
test-launchd-drivers rc=0 secs=328 (all 59 suites passed; bash 3.2 syntax OK)
```

### Dry-run tails
**healthy**
```
[2026-11-20 00:00:02] codex model 'gpt-6.2-sol' unusable: quota-limited
[2026-11-20 00:00:02] codex planner lane -> falling back to 'pi:ollama/kimi-k3:cloud' for this fire (model 'gpt-6.2-sol')
[2026-11-20 00:00:13] codex executor lane -> falling back to 'claude:claude-haiku-5-5' for this fire (model 'gpt-6.2-sol')
[2026-11-20 00:00:36] DRY RUN ok: mission=stapledon repo-slug=sunholo-data/stapledons-godot doc=design_docs/stapledon-mission.md workdir=/Users/voightkampff/.ailang-driver-pin/fleet-i126-judge pidfile
```
**degraded**
```
[2026-11-20 00:01:08] codex model 'gpt-6.2-sol' unusable: quota-limited
[2026-11-20 00:01:08] codex planner lane -> falling back to 'pi:ollama/kimi-k3:cloud' for this fire (model 'gpt-6.2-sol')
[2026-11-20 00:01:08] codex model 'bogus-model-i126' unusable: quota-limited
[2026-11-20 00:01:19] codex executor lane -> falling back to 'claude:claude-haiku-5-5' for this fire (model 'bogus-model-i126')
[2026-11-20 00:01:42] DRY RUN ok: mission=stapledon repo-slug=sunholo-data/stapledons-godot doc=design_docs/stapledon-mission.md workdir=/Users/voightkampff/.ailang-driver-pin/fleet-i126-judge pidfile
```
