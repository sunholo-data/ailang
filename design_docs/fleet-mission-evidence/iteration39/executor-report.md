Implemented M1. All 36 heartbeat arms pass; all ten mutants were killed. No git write operations were performed.

| File | LOC +/− |
|---|---:|
| [mission-control.sh](/Users/voightkampff/.ailang-driver-pin/fleet-i39-unclassified/tools/launchd/mission-control.sh) | +3/−2 |
| [test_mission_heartbeat.sh](/Users/voightkampff/.ailang-driver-pin/fleet-i39-unclassified/tools/launchd/test_mission_heartbeat.sh) | +51/−1 |
| [Heartbeat design doc](/Users/voightkampff/.ailang-driver-pin/fleet-i39-unclassified/design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md) | +1/−14 |
| [Changelog fragment](/Users/voightkampff/.ailang-driver-pin/fleet-i39-unclassified/changelogs/unreleased/2026-10-14-slot-verdict-unclassified.md) | +6/−0 |

All four files have byte-identical snapshots under `.snap/M1/<same relative path>`. Driver and R9 edits match the prescribed changes exactly.

**Before-fix reds:** untouched baseline passed 25/25; adding tests produced rc=1 with these six failures:

- `rc=0 last=bogus => UNCLASSIFIED at=bogus`
- `rc=0 last=gate-bogus => UNCLASSIFIED at=gate-bogus`
- `UNCLASSIFIED history verdict stays one whitespace-free token`
- `phase-2 notifies UNCLASSIFIED once through _mc_bounded`
- `phase-2 dedupes repeated UNCLASSIFIED verdict`
- `phase-2 COMPLETED resets UNCLASSIFIED episode`

All other controls stayed green.

**Mutation results:** each mutant changed only temporary extracted-block copies.

| Mutant | Required arm observed red | rc |
|---|---|---:|
| Delete `0:*` | rc=0 bogus → UNCLASSIFIED | 1 |
| Move `0:*` after catch-all | rc=0 bogus → UNCLASSIFIED | 1 |
| Broaden unknown arm to `*:*` | rc=1 bogus → CRASHED; both kill-code controls | 1 |
| Restore `0:gate-*` | gate-bogus → UNCLASSIFIED | 1 |
| Omit gate-3b | gate-3b → REAPED | 1 |
| Remove UNCLASSIFIED notification | phase-2 notifies UNCLASSIFIED once | 1 |
| Send unconditionally | phase-2 dedupes repeated UNCLASSIFIED | 1 |
| Remove COMPLETED cleanup | phase-2 COMPLETED resets episode | 1 |
| Remove history substitution | history stays one whitespace-free token | 1 |
| Remove pause override | paused bogus → PAUSED-NO-CAPACITY | 1 |

Pristine extracted-block control: **36/36, rc=0**. No survivors. [Full mutation evidence](/tmp/slot-unclassified-M1.cPPlsP/mutation-results.json).

| Gate | Result |
|---|---|
| `/bin/bash tools/launchd/test_mission_heartbeat.sh` | rc=0, 36/36; Bash 3.2 |
| `bash -n tools/launchd/mission-control.sh` | rc=0 |
| `/bin/bash -n tools/launchd/test_mission_heartbeat.sh` | rc=0 |
| ShellCheck on both files | rc=1 before and after |
| `make check-changelog` | rc=0 |
| `git diff --check` | rc=0 |

ShellCheck has **69 inherited findings**: 35 driver, 34 suite. **New findings: zero.** [Detailed findings](/tmp/slot-unclassified-M1.cPPlsP/shellcheck-final.json).

The generic sprint validator returned rc=1 solely because it requires two milestones; the approved plan explicitly defines one.

Controller-owned work not run: `make test-launchd-drivers`, independent evaluation, landing/reach checks, and live dry-runs. Sprint JSON remains unchanged within the four-file scope. Tests used temporary state and recording-only delivery stubs; no network or socket checks ran.