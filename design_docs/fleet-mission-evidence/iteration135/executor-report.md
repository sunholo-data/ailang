The scope guard now accepts rebased mission record pushes after verifying `origin/dev` against the push target. It still refuses branch-authored harness changes, including changes reverted by a later commit. Fast-forward pushes still check only the newly pushed delta.

**Files changed**

- [tools/launchd/githooks/pre-push](/Users/voightkampff/.ailang-driver-pin/.fleet-i135-wt/tools/launchd/githooks/pre-push)
- [tools/launchd/test_mission_scope_guard.sh](/Users/voightkampff/.ailang-driver-pin/.fleet-i135-wt/tools/launchd/test_mission_scope_guard.sh)
- [changelogs/unreleased/2026-11-24-scope-guard-rebased-push.md](/Users/voightkampff/.ailang-driver-pin/.fleet-i135-wt/changelogs/unreleased/2026-11-24-scope-guard-rebased-push.md)

**Gates**

| Command | Result |
|---|---|
| `/bin/bash tools/launchd/test_mission_scope_guard.sh` | `rc=0`. All five new arms and the existing assertions passed. |
| `shellcheck tools/launchd/githooks/pre-push tools/launchd/test_mission_scope_guard.sh` | `rc=0`. The baseline returned `rc=1` with the three reported warnings. The final run had no findings. |
| `make test-launchd-drivers` | `rc=2` on both runs. **UNINFORMATIVE UNDER SANDBOX** due to process-inspection and loopback-bind denials. The scope-guard suite passed within the final run. |

The same five driver suites returned nonzero on both runs: orphan reap, stall, kill tree, stage isolation, and Codex controller environment. Outside-sandbox verification remains outstanding. [Driver gate log](/tmp/fleet135-gates.GfsDNa/launchd.log)

**Mutation drills**

Each mutant ran alone through `/bin/bash tools/launchd/test_mission_scope_guard.sh` in a scratch copy. All four returned `rc=1`.

| Mutant | Killing arm | Diagnostic |
|---|---|---|
| M1 Restore the old base for existing refs | (a) | `MISSION_NAME=triage may not push design_docs/fleet-mission-log.md` |
| M2 Use the dev merge-base for fast-forwards | (c) | `MISSION_NAME=triage may not push tools/launchd/x.sh` |
| M3 Skip verification of the tracking SHA | (d) | Stale-dev push returned `rc=0` instead of `rc=1` and advanced `rec`. |
| M4 Skip the rewritten-commit scan | (e) | Push with the reverted harness change returned `rc=0` instead of `rc=1` and advanced `rec`. |

The drills left all three worktree files byte-for-byte unchanged. The focused suite passed again afterward.