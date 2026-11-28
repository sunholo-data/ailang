Updated only [test_mission_idle_fingerprint.sh](/Users/voightkampff/.ailang-driver-pin/fleet-i146-wt/tools/launchd/test_mission_idle_fingerprint.sh). The pull-request arm now captures `fp`’s exit status directly before extracting the hash.

| Added test arm | FAIL message |
|---|---|
| Reject issue number `392.5` | `FAIL nightly fractional issue number was accepted` |
| Reject comment count `-1` | `FAIL nightly negative comment count was accepted` |
| Reject comment count `1.5` | `FAIL nightly fractional comment count was accepted` |
| Change one report ID while keeping the issue and both counts fixed | `FAIL nightly replaced report did not move hash` |

Added `nowhole`, which replaces `whole` with `true` and is killed by the fractional issue number arm, and `idonly`, which removes the comment ID from the fingerprint row and is killed by the replacement arm. Both matched their exact expected FAIL lines. The negative count also exercises the separate `>= 0` check.

| Gate command | Exit code |
|---|---:|
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` | 0 |
| `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations` | 0 |
| `shellcheck tools/launchd/test_mission_idle_fingerprint.sh` | 0 |

The mutation suite emitted **29 KILLED lines**, including `MUTATION nowhole KILLED` and `MUTATION idonly KILLED`.