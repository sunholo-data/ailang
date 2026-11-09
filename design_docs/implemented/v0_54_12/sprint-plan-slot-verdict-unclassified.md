# M-FLEET-SLOT-UNCLASSIFIED — reachable unknown-label verdict

Ticket: `driver:unclassified-state-unreachable`; fleet iteration 39, harness lane, P2 #15.
Verified HEAD: `1f5f0d3ac07dfb35d49c46bc9d66e4ec0e097b51` (2026-10-14). Initial worktree clean.
Scope: planning only. Controller owns approval, execution handoff, git writes and landing.
Spec: `design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md`, Q4 decision table.

## Premise checks at HEAD

Evidence below comes from `git show HEAD:<path>` with numbered lines; counts cover the entire HEAD blob.

| Claim | file:line at HEAD | Verified yes/no |
|---|---|---|
| Charter lists this ticket as P2 #15 | `design_docs/fleet-mission.md:245` | Yes |
| R9 was filed from the round-3 evaluation; iteration-315 provenance is recorded | `design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md:540`, `:561` | Yes |
| The design names UNCLASSIFIED exactly five times | `design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md:236`, `:253`, `:320`, `:321`, `:346`, `:540`, `:541`, `:543`, `:546` | **No: nine matching lines / nine occurrences now.** R9 quotes a historical five-count. |
| Q4 specifies UNCLASSIFIED for anything else; non-zero rc retains CRASHED/KILLED | `design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md:234`, `:235`, `:236` | Yes |
| Phase-2 notify list includes UNCLASSIFIED; Q6 uses it in the anti-vacuity argument | `design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md:253`, `:320` | Yes |
| Slot-verdict block ends with an unconditional `*:*` CRASHED arm | `tools/launchd/mission-control.sh:2512`, `:2525`, `:2531`, `:2546` | Yes: the scrutinee always contains a colon. |
| rc=0 with last=bogus is CRASHED; driver contains zero UNCLASSIFIED lines | `tools/launchd/mission-control.sh:2526`, `:2531` | Yes: sourced the HEAD classifier in isolation; both rc=0 and rc=1 produced `CRASHED at=bogus`. Whole-blob count = 0. |
| `0:gate-*` also accepts gate-bogus | `tools/launchd/mission-control.sh:2529` | Yes: wildcard is broader than the writer enum. |
| Stamp writer accepts only fired, gate-0/1/2/3/3b/4/5, complete, abort | `tools/launchd/mission-heartbeat.sh:11`, `:12`, `:13` | Yes: all other labels exit 2. |
| PAUSED-NO-CAPACITY overrides the classification; missing artifact is separate | `tools/launchd/mission-control.sh:2535`, `:2537` | Yes |
| History replaces verdict spaces with underscores | `tools/launchd/mission-control.sh:2542`, `:2543` | Yes |
| Shipped phase-2 list omits UNCLASSIFIED; existing episode gate and bounded delivery are reusable | `tools/launchd/mission-control.sh:2550`, `:2552`, `:2554`, `:2556`, `:2564` | Yes |
| Tests extract and source the real marked blocks, including the named CRASHED and DIED-PRE-GATE-0 controls | `tools/launchd/test_mission_heartbeat.sh:48`, `:59`, `:97`, `:130`, `:135`, `:141` | Yes |
| This suite already runs in test-launchd-drivers under /bin/bash | `make/test.mk:68`, `:96`; `tools/launchd/lib/suite-env.sh:18` | Yes: wrapper clears ambient mission variables. |
| CI runs this target on macOS with a bash-3 major-version guard | `.github/workflows/ci.yml:748`, `:756`, `:765`, `:769` | Yes: bash 3.2-compatible code required; no `declare -A` or `${v,,}`. |

## M1 — classify, notify and prove the unknown-label case

One milestone, `M1`, no dependencies. Estimate **100 added LOC**, approximately 60–90 minutes
for implementation, focused tests and mutation checks (0.25 day); live rollout gates are controller work.
One milestone suffices because the classifier and notify edits share one existing extraction suite.
Recent heartbeat changes (`e7dd6c127`, `063df9917`) reuse these same seams; this is a scope estimate,
not a measured LOC/day claim.

| Executor file | Prescriptive change | Estimated added LOC |
|---|---|---:|
| `tools/launchd/mission-control.sh` | Exact gate enum, explicit rc=0 unknown arm, phase-2 pattern | 8 |
| `tools/launchd/test_mission_heartbeat.sh` | Named runtime arms below; reuse extracted blocks and stub | 85 |
| `design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md` | Replace R9 residual paragraph with one fixed-status line | 1 |
| `changelogs/unreleased/2026-10-14-slot-verdict-unclassified.md` | Fragment opening `### Fixed — mission slot UNCLASSIFIED verdict` | 6 |

Preserve the marked block boundaries. The driver already exceeds 800 lines; make only the local
case edits, with no refactor, companion module or new helper. No `.ail` code or new example is needed;
the heartbeat suite supplies the executable examples. Registry reuse for M1: **none** (no package-like
capability, so registry search is inapplicable); reuse the existing shell extraction and `_mc_bounded`
stub, rather than introduce a package or notification mechanism.

Implementation order for the single session:

1. Add the tests first. Hand-write TSV fixtures for invalid labels: the real writer correctly rejects
   them, so bypass it only in corruption fixtures. Use the writer for valid gate controls. Record the
   pre-fix reds for bogus/gate-bogus and notification; unchanged controls must stay green.
2. Replace `0:gate-*` with `0:gate-0|0:gate-1|0:gate-2|0:gate-3|0:gate-3b|0:gate-4|0:gate-5`.
   The decision table's gate-N means real mission gates: the writer's closed enum is authoritative.
   `gate-bogus` must therefore report instrument confusion, rather than a reap at a nonexistent gate.
3. Immediately after the known rc=0 arms and **before** `143:*|137:*` and `*:*`, add
   `0:*) _mc_slot_verdict="UNCLASSIFIED at=$_mc_slot_last" ;;`.
   Keep 143/137 KILLED and all other non-zero CRASHED arms byte-for-byte; keep the pause override,
   missing-artifact path, RC, retry flow and history serialization unchanged.
4. Add `UNCLASSIFIED*` to the existing phase-2 notify case. Use the same episode file, verdict-string
   comparison, `_mc_bounded 30` deliveries and COMPLETED reset. Keep the existing title and channels.
5. Replace R9 with: `- **R9 — FIXED (M-FLEET-SLOT-UNCLASSIFIED):** unknown rc=0 labels emit UNCLASSIFIED and use the existing episode-gated phase-2 notice.`
   Leave R10 and other residuals alone. Add the named changelog fragment, describing exact gate labels
   and unchanged non-zero/pause behavior. Run mutation checks, restore extracted copies, and validate.

This is a classification/observability correction within the existing spec. No threshold, routing,
lane order, quota, billing guard, schedule or other driver behavior changes (fleet policy-class boundary).

## Exact test arms to add

Use these strings as the suite's `check` names; assert complete verdict prefixes through the next
`rc=` field rather than accepting any occurrence of UNCLASSIFIED. Reuse `run_verdict` and the real
awk-extracted blocks; keep extraction-empty FATAL checks. If needed, give `run_verdict` an optional
sixth pause argument defaulting to 0, preserving its first five arguments.

| Exact arm name | Fixture and assertion |
|---|---|
| `rc=0 last=bogus => UNCLASSIFIED at=bogus` | Invalid-label TSV, rc=0; log prefix `slot-verdict: UNCLASSIFIED at=bogus rc=0`. |
| `rc=1 last=bogus => CRASHED at=bogus` | Same fixture, rc=1; exact CRASHED prefix, unchanged. |
| `rc=0 last=gate-3 => REAPED at=gate-3` | Real helper stamp gate-3, rc=0; exact REAPED prefix (positive control). |
| `rc=0 last=gate-bogus => UNCLASSIFIED at=gate-bogus` | Invalid gate-like TSV, rc=0; exact UNCLASSIFIED prefix. |
| `rc=0 last=gate-3b => REAPED at=gate-3b` | Real helper stamp gate-3b; protects the nonnumeric valid gate. |
| `rc=143 and rc=137 last=bogus => KILLED at=bogus` | Loop over both kill codes, require both exact KILLED prefixes. |
| `paused rc=0 last=bogus => PAUSED-NO-CAPACITY at=bogus` | Source verdict with MC_PAUSED=1; override remains authoritative. |
| `phase-2 notifies UNCLASSIFIED once through _mc_bounded` | Fresh state/calls, verdict `UNCLASSIFIED at=bogus`, GH issue unset; source once, require exactly one `30 env ... ailang messages send` call containing verdict and exact episode contents. Stub records only; no network. |
| `phase-2 dedupes repeated UNCLASSIFIED verdict` | Source again with same verdict/state; count remains exactly one and episode unchanged. |
| `phase-2 COMPLETED resets UNCLASSIFIED episode` | COMPLETED removes marker; next same UNCLASSIFIED produces exactly a second bounded message. |
| `UNCLASSIFIED history verdict stays one whitespace-free token` | TSV label `bogus label`; log retains raw `UNCLASSIFIED at=bogus label`; newest history row has exactly seven whitespace-separated fields, `$2 == "verdict=UNCLASSIFIED_at=bogus_label"`, `$3 == "rc=0"`, and no detached `at=` field. |

Retain every existing arm, especially COMPLETED, ABORTED, DIED-PRE-GATE-0, HEARTBEAT-MISSING,
rc=1 gate-2, episode changes, history bound, retry history and ordering around PIDFILE release.

## Mutation table

During execution, mutate temporary copies of the **extracted production blocks**, one change at a
time, and run the same named assertion against each. The controller records failing arm/output/rc
and the pristine green control. No mutation may touch the live driver or require git restoration.

| Mutant | Arm that must go red |
|---|---|
| Delete the new `0:*` arm | `rc=0 last=bogus => UNCLASSIFIED at=bogus` |
| Move `0:*` after `*:*` | `rc=0 last=bogus => UNCLASSIFIED at=bogus` |
| Broaden unknown-label arm to `*:*` before non-zero arms | `rc=1 last=bogus => CRASHED at=bogus`; kill-code control also fails |
| Revert exact gate enum to `0:gate-*` | `rc=0 last=gate-bogus => UNCLASSIFIED at=gate-bogus` |
| Omit gate-3b from exact enum | `rc=0 last=gate-3b => REAPED at=gate-3b` |
| Remove UNCLASSIFIED from phase-2 pattern | `phase-2 notifies UNCLASSIFIED once through _mc_bounded` |
| Remove episode comparison / send unconditionally | `phase-2 dedupes repeated UNCLASSIFIED verdict` |
| Remove COMPLETED episode cleanup | `phase-2 COMPLETED resets UNCLASSIFIED episode` |
| Remove verdict space-to-underscore substitution | `UNCLASSIFIED history verdict stays one whitespace-free token` |
| Remove pause override | `paused rc=0 last=bogus => PAUSED-NO-CAPACITY at=bogus` |

## Controller validation and done-gate

Run these gates **outside the sandbox**, before approval/landing. Record commands, actual rc and
relevant output; all existing and new arms must pass, with each mutant killed. Socket-touching checks
inside this sandbox are **UNINFORMATIVE**, not pass/fail evidence. No host suite or live dry-run was
performed by the planner.

```bash
bash tools/launchd/test_mission_heartbeat.sh
env -i HOME="$HOME" PATH="$PATH" make test-launchd-drivers
bash -n tools/launchd/mission-control.sh
/bin/bash -n tools/launchd/test_mission_heartbeat.sh
if command -v shellcheck >/dev/null 2>&1; then
  shellcheck -s bash tools/launchd/mission-control.sh tools/launchd/test_mission_heartbeat.sh
fi
make check-changelog
```

Use macOS `/bin/bash` 3.2 for the aggregate gate. Record shellcheck as unavailable if absent; surface
any inherited findings separately. Independent sprint-evaluator must review M1 against the table,
test outputs and mutation evidence; no self-approval.

Done-gate (`design_docs/fleet-mission.md:134`): confirm the edited driver is the actual running surface
and its committed SHA has reached the pinned origin/dev. Check an **idle, armed sibling** profile's
kill switch and pidfile; record that profile. Do not dry-run fleet during its own active iteration.
Using controller-selected `slot_fixed_sha` and `slot_idle_profile`, run:

```bash
env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR:-/tmp}" \
  AILANG_DRIVER_PINNED="$slot_fixed_sha" MISSION_PROFILE="$slot_idle_profile" \
  MISSION_DRY_RUN=1 /bin/bash tools/launchd/mission-control.sh
env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR:-/tmp}" \
  AILANG_DRIVER_PINNED="$slot_fixed_sha" MISSION_PROFILE="$slot_idle_profile" \
  MISSION_DRY_RUN=1 MISSION_EXECUTOR_MODEL=codex:bogus MISSION_PROBE_TIMEOUT=10 \
  /bin/bash tools/launchd/mission-control.sh
```

Require both rc=0, healthy `lanes=ok`, degraded `lanes=DEGRADED(...)`, and neither a disabled/overlap
early yield nor a pidfile write. This existing degradation seam is documented at
`tools/launchd/mission-control.sh:1973`. If no sibling is idle and armed, use the charter's synthetic-HOME
fallback (`design_docs/fleet-mission.md:144`) and record it. These dry-runs validate reach/pre-flight;
the extracted tests validate exit classification. Nothing is reloaded mid-iteration. Keep the ticket
open until independent evaluation, host gates, landing/reach and both dry-runs are evidenced.
