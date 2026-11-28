All four mutants (`widen`, `narrow`, `nopr`, `nouniq`) **SURVIVED at `d1c8d568b`** before the edit. Each scratch-copy run returned rc 0, and each `sed` changed exactly one line of the helper.

```text
 tools/launchd/test_mission_idle_fingerprint.sh | 44 +++++++++++++++++++++++---
 1 file changed, 40 insertions(+), 4 deletions(-)
```

| Gate | RC | Result |
|---|---:|---|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 0 | Three PASS lines |
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations` | 0 | 27 KILLED |
| `shellcheck -s bash tools/launchd/test_mission_idle_fingerprint.sh` | 0 | No findings |

```text
MUTATION widen KILLED: FAIL nightly triage comment moved hash
MUTATION narrow KILLED: FAIL nightly golden failure comment did not move hash
MUTATION nopr KILLED: FAIL nightly pull request triggered a comments read
MUTATION nouniq KILLED: FAIL nightly duplicate comment id was accepted
```

No implementation deviations. An incidental `ps` diagnostic was **UNINFORMATIVE UNDER SANDBOX**.