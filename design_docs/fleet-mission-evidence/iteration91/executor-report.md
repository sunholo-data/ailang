Implemented, with one blocked mirror copy: `.agents/skills/mission-control/resources/gate-5-retro.md` is read-only. Its exact replacement is included in [the snapshot manifest](/Users/voightkampff/.ailang-driver-pin/fleet-i91-idle/.snap/MANIFEST.txt) for the controller to apply before committing.

Changed: helper, driver, test suite, Make registration, retro instructions, changelog fragment, and [verification report](/Users/voightkampff/.ailang-driver-pin/fleet-i91-idle/verification/fleet-iter91-idle/report.md). Eight full-content snapshots include the pending mirror.

| Gate command | Result |
|---|---|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | PASS |
| Same suite with `--mutations` | PASS: 7 killed, 0 survived |
| `/bin/bash tools/launchd/test_mission_triage_idle.sh` | PASS |
| `make test-launchd-drivers` | UNINFORMATIVE UNDER SANDBOX; rc=2, findings below |
| `shellcheck -s bash tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh` | PASS: zero findings |
| `bash -n tools/launchd/mission-control.sh` | PASS |
| Driver ShellCheck finding count | PASS: 30 → 30 |
| `make check-context-docs check-changelog check-tmpfile-hygiene check-home-isolation check-skills` | PASS |
| `git diff --check` | PASS |
| `/bin/bash tools/launchd/sync-agents-skills.sh --check` | FAIL: pending read-only mirror |

The broad run had the same six nonzero suites as the untouched base, plus two mirror checks:

- Base fixture failures: `test_pin_root.sh`, `test_skill_sync.sh`, and `test_mission_pi_run_commits.sh` misclassify temporary “non-repo” directories nested inside this worktree.
- Sandbox-uninformative: `test_mission_kill_tree.sh`, `test_mission_stall.sh`, and `test_codex_controller_env.sh` depend on restricted process inspection or real Codex execution.
- New mirror failures: `test_agents_skills_sync.sh` and `test_quorum_artifacts.sh`. Apply the supplied mirror replacement.

Reader audit: driver attempt history only assigns a path; retry and final history append/trim rows, while retry decisions use heartbeat/current output. Notifications use the current verdict. The CLI stores verdict text opaquely. Existing tests read isolated fixtures. No dashboard or other runtime history readers were found. Documentation references introduce no parser. No reader changes were needed.

| Scratch-copy mutant | Killing arm |
|---|---|
| Remove Git exclusions | Record-only commit invariance |
| Read wrong record | Unchanged → IDLE |
| Retain changed record | Changed → removal |
| Fail closed | Error → normal fire |
| Remove daily expiry | 86401-second expiry |
| Remove list-limit guard | 1000-message refusal |
| Include pending checks | Pending-check invariance |

Live-service behavior remains unverified. One isolation mistake occurred: initial baseline static checks used their default temporary directory; subsequent runs and every new test used worktree-local temporary state. No repository Git writes or deployment were performed.