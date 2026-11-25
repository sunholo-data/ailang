# Judge report — fleet iteration 138 (candidate 8fc5c0e6b)

## Verdict: PASS, 90/100 (pass mark 70). No blocking findings.

| Rubric area | Score | Note |
|---|---|---|
| Correctness / ticket failure prevented | 38/40 | Incident reproduced and refused (see below). |
| Tests and non-vacuity | 22/25 | 37/37 arms pass. Every mutation I ran was killed except two minor survivors. |
| Constraints (bash 3.2, allowlist, no new env, generated mirror, fail loud) | 18/20 | All met. Small inconsistency in snippet fallbacks. |
| Docs / changelog / hygiene | 12/15 | Gates green. Some prose and edge-case notes below. |

## 1. Incident reproduction (own measurement)
- Fixture: repo with origin `https://github.com/sunholo-data/ailang.git`, `MISSION_NAME=world`, `MISSION_REPO=sunholo-data/ailang-world`, fresh state dir, `record gate1`.
  - OLD helper (3f5143370): rc=0, wrote an ailang SHA into `mission-world-base` (the ticket's failure, reproduced).
  - NEW helper: rc=3, stderr `mission-base: CWD origin is sunholo-data/ailang, expected sunholo-data/ailang-world (MISSION_REPO)`, no stdout, and no base file created.

## 2. Regression risk for the seven missions
- Ran the new `snap` with the correct `MISSION_REPO` against all five ailang pin worktrees (docs, fleet, motoko, triage, v1): rc=0 each.
- Ran it against the real clones: ailang, ailang-docs, ailang-motoko (`sunholo-data/ailang`), ailang-world, stapledons-godot: rc=0 each.
- Origin URLs of every one of these are plain `https://github.com/<slug>.git`.
- A linked worktree resolves the common config, so the check works there too (docs/fleet/triage/v1 pins are linked worktrees).
- Call sites:
  - gate-1: `|| exit $?` propagates 3.
  - gate-3 `snap`: `|| exit 2`, which collapses 3 to 2 but stderr names the cause. `drift`: `exit "$drift_rc"` for anything other than 1. Both loud.
  - gate-3b `record`: `|| exit 2`. `drift`: `*)` arm exits with the rc.
  - gate-4: `|| exit 2`.
  - Nothing swallows rc 3 into a "steady" or "no record" outcome.
- `last` is unguarded, so gate-3's `last gate1` works from any CWD.
- Unset or empty `MISSION_REPO` leaves attended and scratch use unchanged. The driver defaults `MISSION_REPO` to sunholo-data/ailang and exports it (mission-control.sh:74-76).

## 3. Mutation table (copy of helper, `MISSION_BASE_HELPER=<copy> /bin/bash tools/launchd/test_mission_base.sh`)
| Mutation | rc | `not ok` count / first lines that fired |
|---|---|---|
| guard removed (verify_repo returns 0) | 1 | 11: wrong repo absent/preserved, missing origin, outside Git... |
| guard moved after the append in `record` | 1 | 7: wrong repo ref-override absent/preserved, missing origin, outside Git... |
| case-sensitive compare | 1 | 7: mixed-case SCP/ssh origins, plus mixed-case MISSION_REPO |
| `last` guarded | 1 | 1: "last returns a recorded SHA with MISSION_REPO set and no origin URL" |
| snap unguarded | 1 | 2: both wrong-repo snap arms |
| drift unguarded | 1 | 2: both wrong-repo drift arms |
| record unguarded | 1 | 7 |
| exit 1 instead of 3 | 1 | 11 |
| no `.git` strip | 1 | 9 |
| no trailing-slash strip | 1 | 3 (all three `.git/` variants) |
| https branch dropped | 1 | 10 |
| scp branch dropped | 1 | 3 |
| ssh:// branch dropped | 1 | 3 |
| arbitrary host accepted | 1 | 1 (unsupported host arm) |
| slug regex loosened to `.+` | 1 | 2 (extra path component, blank-entry arm) |
| missing origin passes | 1 | 1 |
| outside Git passes | 1 | 1 |
| guard activated by presence of MISSION_REPO, not nonempty | 1 | 1 (empty MISSION_REPO arm) |
| `--get-all` -> `--get` (first URL only) | 1 | 1 (multi-URL arm) |
| blank-entry terminator stripping removed | 1 | 1 |
| guard moved before arg validation in `record` | 0 | SURVIVES (behaviourally near-equivalent; see N3) |
| drop `--local` from the config read | 0 | SURVIVES (see N2) |

Every non-survivor was killed and the killing arm is the one the plan assigned. The first "last guarded" attempt was a broken mutant (36 failures from a syntax slip). I redid it correctly, and the result above is the clean one.

## 4. Gates
- `/bin/bash tools/launchd/test_mission_base.sh`: rc 0, "PASS: 37 mission-base arms ran", 37 ok, 0 not ok.
- `shellcheck -s bash` on `mission-base.sh` and `test_mission_base.sh`: rc 0, clean.
- `/bin/bash tools/launchd/test_agents_skills_sync.sh`: rc 0, 7 passed, 0 failed.
- The 5 `.agents` resource mirrors are byte-identical (`cmp`) to the `.claude` sources.
- `/bin/bash scripts/check_context_docs.sh`: rc 0.
- `make check-changelog`: rc 0, 29 fragments valid.
- Not run, per instruction: `make test-launchd-drivers`.

## Blocking findings
None.

## Non-blocking findings
- N1. URL parser is strict. `https://user@github.com/...` (token-embedded), uppercase host, `ssh://git@github.com:22/...` and `git://` all refuse with rc 3 ("unsupported or ambiguous origin"). This fails loudly, per principle 2. All 7 real missions use the plain form, so there is no current stall. A future clone with a credentialed URL would stall its loop until the URL is fixed, and the message does not print the raw URL (by design), so it is harder to diagnose.
- N2. Dropping `--local` survives the suite. No arm sets a global `remote.origin.url`. With `--local` dropped, a global origin would be accepted. Low impact (a global origin is unusual), but the choice is untested. A mutation-proof arm would use `GIT_CONFIG_GLOBAL`.
- N3. Order of the guard versus arg validation in `record` is not tested. If `record` is called with no label and the wrong repo, the user sees exit 2 (usage) rather than exit 3. Harmless.
- N4. Snippet inconsistency. gate-3 builds the worktree helper as `${MISSION_DRIVER_ROOT:-.}/tools/launchd/mission-worktree.sh`, while the base snippets use bare `$MISSION_DRIVER_ROOT`. If it is unset (attended session), `bash "/tools/launchd/mission-base.sh"` fails rc 127, which is loud and safe. The driver always exports it for controllers (mission-control.sh:2437).
- N5. gate-3's `snap ... || exit 2` collapses the new rc 3 to 2. The stderr message still names the cause, so this is only an observability nit.
- N6. The gate-4 prose rewrite ("The pick note may quote the mission base helper's `snap` result") drops the explicit `mission-base.sh` name and the "this sprint writes no `base-gate2` row" clause. It is slightly vaguer, but not wrong.
- N7. Remaining relative `mission-base.sh` references live only in historical docs/retros and design logs, not in any controller-read skill resource. `tools/launchd/mission-lane-check.sh:63` is only a comment.

## Prose check
- ref-drift.md says "Run from the mission's own repository, since the helper refuses a mismatched origin when `MISSION_REPO` is set". This is accurate.
- The non-vacuity recipe now uses `$MISSION_DRIVER_ROOT`. Accurate.

EVALUATION_SCORE: 90/100
EVALUATION_VERDICT: PASS score=90
