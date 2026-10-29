# Sprint report: exit-path notices bounded (fleet iteration 33)

**Plan:** `design_docs/planned/sprint-plan-exit-path-notices-bounded.md` (commit `468604ba4`)
**Ticket:** `driver:exit-path-notices-unbounded`
**Branch:** `fleet/i33-exit-notices-bounded` (base `06d524c59`)
**Fix commit:** `1fe1e3911` (driver, four test files, changelog fragment)

## Files changed

| File | Change |
|------|--------|
| `tools/launchd/mission-control.sh` | `_mc_exit_notice` helper after `_mc_notify`. Refusal, late-kill and rc-fail sites now call it. SLOT NOTIFY send gets `env` store scoping. Four comment-only markers. |
| `tools/launchd/test_driver_notify.sh` | `MC_NOTIFY_DRIVER` override. Marker-based extraction. Arms E0–E9 and guards G1–G3b. |
| `tools/launchd/test_controller_capacity.sh` | Extracts `_mc_exit_notice` into its fixture, plus a guard that names a missing extraction. |
| `tools/launchd/test_mission_heartbeat.sh` | SLOT NOTIFY regex now matches the scoped send. |
| `changelogs/unreleased/2026-10-12-exit-path-notices-bounded.md` | `### Fixed` fragment. |
| `.ailang/state/sprints/exit-path-notices-bounded-{red,green-notify}.txt` | RED and GREEN evidence logs. |

Titles, bodies and exit statements are unchanged. The three moved message strings match HEAD
character for character. What changed is the wrapper: the `--title … 2>/dev/null` tail, the
`[ -n … ] && gh` prefix, and the `--body "` opener. Exit codes are untouched (`exit 1` after the
refusal, `exit "$RC"` at the end). The F8 literal appears once, inside the rc-fail call's GH-body
argument.

## Pre-fix RED evidence

Markers and arms were added first, and the suite ran against the marker-only driver (the unfixed
sites). Log: `/tmp/fleet_i33_prefix_red.txt`, copied to `.ailang/state/sprints/exit-path-notices-bounded-red.txt`.
Result: **rc=1, 104 passed, 11 failed.**

| Arm | Pre-fix result | Why |
|-----|----------------|-----|
| E1, E2, E3 | FAIL | The bare send hangs. The outer watchdog kills it at 30 s, so the arm reports "send is UNBOUNDED". |
| E7a, E7b, E7c | FAIL | The `2>/dev/null` hides both the failure reason and the store scope, so no WARNING is logged. |
| E9 | FAIL | A duplicate title is never logged as "already in the inbox". |
| G1 | FAIL | Unbounded lines 2038, 2041, 2529, 2532, 2550, 2553 (pre-marker numbers; the markers shift them). |
| G2 | FAIL | Unscoped lines at the three exit sites and the SLOT NOTIFY send. |
| G3b | FAIL | The F8 literal is not yet inside an `_mc_exit_notice` call. |
| E0 | FAIL | `_mc_exit_notice` is not defined yet. |

Green by design (preservation): E4, E5, E6, E8a, E8b and G3a. They pass pre-fix, which shows the
arms hold the behaviour the fix must keep.

## Mutation drills (scratch copies, `MC_NOTIFY_DRIVER`)

Every drill returns rc=1 and fails the arm it targets.

| Drill | Mutation | Caught by |
|-------|----------|-----------|
| d1 | Drop `env` store scope from helper | E7a–c, G2 |
| d2 | Change the late-kill title | E5 (after the title-anchor fix below) |
| d3 | Drop `⚠️ ` from the rc-fail F8 body | E6, G3a |
| d4 | Disable the refusal episode marker | E8a |
| d5 | Return before the GH call in the helper | E1–E6 (E4 is the named one) |
| d6 | Unbounded `gh issue comment` in the helper | G1, E1–E3 |
| d7 | Unbounded `ailang messages send` in the helper | E1–E3, E7a–c, E9, G1 |
| d8 | Disable duplicate-title classification | E9 |
| d9 | Drop `env` scope from SLOT NOTIFY | G2 |

## GREEN results

Run on `/bin/bash` 3.2.57, with each rc captured unpiped.

| Suite | rc | Result |
|-------|----|--------|
| `tools/launchd/test_driver_notify.sh` (via suite-env) | 0 | 115 passed, 0 failed. E1–E3 return inside the 14 s ceiling. |
| `tools/launchd/test_controller_capacity.sh` | 0 | 15 cases, fail=0 |
| `tools/launchd/test_mission_heartbeat.sh` | 0 | 25 arms ran |
| `scripts/test_mission_gate0_self_notices.sh` | 0 | 108 passed, 0 failed (the F8 literal still extracts) |
| `make test-launchd-drivers` (full, unpiped) | 0 | Every suite green. Log: `/tmp/fleet_i33_launchd_suite.log` |
| `make check-changelog` | 0 | Fragment accepted |

`bash -n` passes on all four edited shell files.

## Deviations from the plan, and why

1. **E0 is an arm failure, not a suite abort.** The plan says the helper extraction should be
   FATAL when empty. That would stop the pre-fix run before any arm ran, and the RED evidence would
   then be a single FATAL line. The marker blocks are still FATAL when empty. The helper's absence
   is now assertion E0.
2. **G3 is split into G3a and G3b.** The plan's table calls G3 a green/green tripwire, but its text
   also requires the literal to be inside an `_mc_exit_notice` call. G3a (exactly one literal) stays
   green pre-fix as a tripwire. G3b (inside the helper call) is red pre-fix and green post-fix.
3. **Title anchoring added to E4, E5 and E6.** Mutation drill d2 (`work landed` → `work landed!`)
   survived the first version of E5. The case pattern ended at `work landed` and so matched the
   longer title. Each title assertion now includes the following ` --from`, which anchors the title.
   After that change the suite is green, and d2 is caught by E5.
4. **M3 is not split out.** The plan allowed splitting the SLOT NOTIFY change off as its own commit.
   It landed in the fix commit instead, because G2 needed the exemption to be gone.
5. **The routing and wiring arms fail when the driver runs from a scratch path.** In the drill
   copies, `lane sets the digest routing note`, `no routing note when healthy` and
   `wiring: pi controller loads controller-bash-cap.ts` fail. An unmutated control copy fails the
   same three arms, so this is the location, not any mutation. I did not find the exact cause.
   Those three arms pass in the real worktree, which is what the full make run covers.
6. **Hang arms print "Killed: 9" noise.** That is the outer watchdog's process-tree kill, which
   the RED run triggers on purpose. It is expected output, not a failure.

## Not verified here

- The rig claim (no launchd path exports `AILANG_STORAGE_MESSAGING` or `AILANG_MESSAGES_PROJECT`)
  is from the plan. I did not re-check it on the rig, and I did not run any real send or comment.
- The `driver:exit-notice-titles-undated` follow-up is still open. Until it lands, a repeat of the
  same title is refused by the inbox and logged as "already in the inbox", not re-sent. The plan
  records this as a known consequence.
- Nothing was pushed, no launchd job was reloaded, and no real `ailang messages send` or
  `gh issue comment` was run. All tests used PATH stubs.
