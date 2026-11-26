Implemented fleet iteration 143 verbatim. The two new test cases verify that MISSION-ITER round verdicts and ordinary sprint record verdicts still wake the mission. The production helper is byte-for-byte identical to HEAD.

| Changed file | Insertions | Deletions |
|---|---:|---:|
| [tools/launchd/test_mission_idle_fingerprint.sh](/Users/voightkampff/.ailang-driver-pin/fleet-i142-wt/tools/launchd/test_mission_idle_fingerprint.sh) | 38 | 8 |
| [changelogs/unreleased/2026-11-26-idle-fingerprint-record-eval-json.md](/Users/voightkampff/.ailang-driver-pin/fleet-i142-wt/changelogs/unreleased/2026-11-26-idle-fingerprint-record-eval-json.md) | 1 | 0 |

### Gates

| Command | rc | Result |
|---|---:|---|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 0 | Exactly 3 `PASS` lines |
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations` | 0 | Exactly 21 `KILLED` lines |
| `IDLE_TEST_HELPER=<scratch>/helper /bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 1 each | All four scratch copies produced exactly one `FAIL` line, matching the expected text below |
| `shellcheck tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh` | 0 | No diagnostics |
| `make check-changelog` | 0 | `✓ 2 changelog fragment(s) valid` and `✓ CHANGELOG.md is index-only and links changelogs/v0.32-current.md` |
| `git diff --stat` | 0 | `2 files changed, 39 insertions(+), 8 deletions(-)` |
| `git diff --quiet -- tools/launchd/lib/idle-fingerprint.sh` | 0 | `helper-unchanged-rc=0` |

```text
PASS driver arms, slot layout, stdin EOF, state seam and placement
PASS fire-start snapshot: argv, stdin EOF, stale cleared, failures leave none, exclusions, placement
PASS helper sections, exclusions, failed reads, branch choice, atomic record and refusals
```

```text
MUTATION exclude-record-eval-no-suffix KILLED: FAIL MISSION-ITER round evaluation JSON commit did not move hash
MUTATION exclude-record-eval-any-mission KILLED: FAIL non-MISSION-ITER record evaluation JSON commit did not move hash
MUTATION exclude-record-eval-any-verdict KILLED: FAIL MISSION-ITER round evaluation JSON commit did not move hash
MUTATION exclude-record-any-dir KILLED: FAIL non-MISSION-ITER record evaluation JSON commit did not move hash
```

### Scratch copies

Each `sed` exited 0 and each `cmp` exited 1, confirming the scratch helper differed from the pristine helper.

| Scratch helper | Test rc | First `FAIL` line |
|---|---:|---|
| `exclude-record-eval-no-suffix` | 1 | `FAIL MISSION-ITER round evaluation JSON commit did not move hash` |
| `exclude-record-eval-any-mission` | 1 | `FAIL non-MISSION-ITER record evaluation JSON commit did not move hash` |
| `exclude-record-eval-any-verdict` | 1 | `FAIL MISSION-ITER round evaluation JSON commit did not move hash` |
| `exclude-record-any-dir` | 1 | `FAIL non-MISSION-ITER record evaluation JSON commit did not move hash` |

No plan deviations or sandbox-blocked gates. Scratch files and disposable fixtures were removed.