Iteration 145 now includes open nightly-rig issues and their failure reports in the idle fingerprint. The tests verified that a new failure moves the hash, triage comments leave it stable, and closing the issue restores the legacy hash.

**Files changed**

| File | Added | Deleted |
|---|---:|---:|
| [tools/launchd/lib/idle-fingerprint.sh](/Users/voightkampff/.ailang-driver-pin/.fleet-i145-build/tools/launchd/lib/idle-fingerprint.sh) | 34 | 1 |
| [tools/launchd/test_mission_idle_fingerprint.sh](/Users/voightkampff/.ailang-driver-pin/.fleet-i145-build/tools/launchd/test_mission_idle_fingerprint.sh) | 58 | 1 |
| [.claude/skills/mission-control/resources/gate-5-retro.md](/Users/voightkampff/.ailang-driver-pin/.fleet-i145-build/.claude/skills/mission-control/resources/gate-5-retro.md:28) | 1 | 1 |
| [changelogs/unreleased/2026-11-27-idle-fingerprint-nightly-rig.md](/Users/voightkampff/.ailang-driver-pin/.fleet-i145-build/changelogs/unreleased/2026-11-27-idle-fingerprint-nightly-rig.md) | 8 | 0 |

**Plan deviation.** The sandbox rejected the one-line edit to [.agents/skills/mission-control/resources/gate-5-retro.md](/Users/voightkampff/.ailang-driver-pin/.fleet-i145-build/.agents/skills/mission-control/resources/gate-5-retro.md:28). I verified that the requested substitution would make it byte-identical to the updated `.claude` copy. The helper, stub argument order, and mutation expressions matched the plan verbatim.

**Gate results**

| Gate command | RC | Key output |
|---|---:|---|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 0 | Three `PASS` lines covering the driver, fire-start snapshot, and helper |
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations` | 0 | 23 `KILLED` |
| `IDLE_TEST_HELPER=<scratch-copy> /bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 1 each, expected | Prefix filter removed → `FAIL nightly triage comment moved hash`. Hash input removed → `FAIL nightly failure comment did not move hash`. |
| `shellcheck -s bash tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh` | 0 | Zero findings |
| `make check-changelog` | 0 | Four valid fragments. `CHANGELOG.md` is index-only and links the active archive. |

```text
MUTATION nightly-omit KILLED: FAIL nightly failure comment did not move hash
MUTATION nightly-all-comments KILLED: FAIL nightly triage comment moved hash
```

For the sole-killer checks, I made two scratch copies and used `diff -u` to verify that each contained only its intended removal. The completed mutation run used `/usr/bin/git` after I interrupted the slower Homebrew Git attempt (rc 130).