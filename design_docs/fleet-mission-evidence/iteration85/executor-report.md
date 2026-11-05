Implemented and left uncommitted for the controller. All seven executor gates passed on Bash 3.2.57. Tests used synthetic homes and fake secrets only.

| File | Added lines |
|---|---:|
| [transcript-secret-scan.sh](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter85/tools/launchd/transcript-secret-scan.sh) | 174 |
| [test_transcript_secret_scan.sh](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter85/tools/launchd/test_transcript_secret_scan.sh) | 372 |
| [mission-control.sh](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter85/tools/launchd/mission-control.sh) | 68 |
| [make/test.mk](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter85/make/test.mk) | 1 |
| [Changelog fragment](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter85/changelogs/unreleased/2026-11-04-transcript-secret-scan.md) | 11 |

Both new scripts are executable (`0755`). Existing iteration-85 evidence remains untouched.

| Gate command | Result |
|---|---|
| `/bin/bash -n tools/launchd/transcript-secret-scan.sh` | rc 0 |
| `/bin/bash -n tools/launchd/test_transcript_secret_scan.sh` | rc 0 |
| `/bin/bash -n tools/launchd/mission-control.sh` | rc 0 |
| `shellcheck --shell=bash tools/launchd/transcript-secret-scan.sh tools/launchd/test_transcript_secret_scan.sh` | rc 0 |
| `/bin/bash tools/launchd/test_transcript_secret_scan.sh` | rc 0; 30 arms passed |
| `/bin/bash tools/launchd/lib/run-suites.sh 1 tools/launchd/test_transcript_secret_scan.sh` | rc 0 |
| `/bin/bash scripts/check_changelog.sh` | rc 0 |
| `git diff --check` | rc 0 |
| `make test-launchd-drivers` | **UNINFORMATIVE UNDER SANDBOX** — not run; designated controller integration gate |

Every required mutation ran against a disposable copy:

| Arm | Mutation result |
|---|---|
| Clean | Killed |
| Fresh leak | Killed |
| Literal metacharacters | Killed |
| Older leak | Killed |
| Marker boundary | Killed |
| Default minimum | Killed |
| Explicit minimum | Killed |
| Assignment forms | Killed |
| Export prefix | Killed |
| No execution | Killed |
| Missing secrets | Killed |
| Missing marker | Killed |
| Zero eligible keys | Killed |
| Missing root | Killed |
| Repeated roots | Killed |
| Default roots | Killed |
| JSON escaping | Killed |
| Escaped slash | Killed |
| Deduplication | Killed |
| Enumeration failure | Killed |
| Matcher failure | Killed |
| Secret-free argv | Killed |
| Termination cleanup | Killed |
| Driver marker | Killed |
| Driver leak | Killed |
| Driver clean | Killed |
| Driver failure | Killed |
| Driver timeout | Killed |
| Driver dry-run | Killed |

No mutations survived or were skipped; disposable trees were removed.

No scope deviations or plan defects found. One additional contract arm covers duplicate assignments, invalid minima, pathname controls, and malformed summaries. Binary `ailang mission iterate` coverage remains excluded as planned.