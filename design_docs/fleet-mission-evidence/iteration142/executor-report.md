Implemented fleet iteration 142. Record-only mission evaluation JSON commits no longer move the idle fingerprint. Ordinary sprint verdicts still move it.

| File | Lines |
|---|---:|
| [tools/launchd/lib/idle-fingerprint.sh](/Users/voightkampff/.ailang-driver-pin/fleet-i142-wt/tools/launchd/lib/idle-fingerprint.sh) | +5 / −3 |
| [tools/launchd/test_mission_idle_fingerprint.sh](/Users/voightkampff/.ailang-driver-pin/fleet-i142-wt/tools/launchd/test_mission_idle_fingerprint.sh) | +25 / −3 |
| [changelogs/unreleased/2026-11-26-idle-fingerprint-record-eval-json.md](/Users/voightkampff/.ailang-driver-pin/fleet-i142-wt/changelogs/unreleased/2026-11-26-idle-fingerprint-record-eval-json.md) | +7 / −0 |

| Gate | rc | Result |
|---|---:|---|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 0 | Three `PASS` lines |
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations` | 0 | 17/17 mutants killed |
| `IDLE_TEST_HELPER="$scratch/helper" /bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 1 | Expected `FAIL record-only evaluation JSON commit moved hash` |
| `shellcheck tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh` | 0 | No diagnostics |
| `make check-changelog` | 0 | `1 changelog fragment(s) valid` |

```text
PASS driver arms, slot layout, stdin EOF, state seam and placement
PASS fire-start snapshot: argv, stdin EOF, stale cleared, failures leave none, exclusions, placement
PASS helper sections, exclusions, failed reads, branch choice, atomic record and refusals
```

```text
MUTATION exclude-record-eval KILLED: FAIL record-only evaluation JSON commit moved hash
MUTATION exclude-evaluations-broad KILLED: FAIL ordinary evaluation JSON commit did not move hash
```

For the sole-killer check, I exported the helper from `HEAD` into a scratch file inside the worktree and ran the updated fixture through its existing `IDLE_TEST_HELPER` override. Only the helper used its base version, and the fixture stopped at the expected assertion.

The plan required no adaptations. The changelog ends with the requested plain-text ticket signature `idle-fingerprint:record-eval-json-moves-head`. Git operations on the worktree were read-only.