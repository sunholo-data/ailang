# Fleet iteration 71 — draft records for independent review

Reviewed tree: /Users/voightkampff/.ailang-driver-pin/fleet (detached at origin/dev d45cf4e6b135070199a976d3f6bbe7cc2a2ed5b0). Do NOT edit it.

## Record 1 — resolve TWO tickets as ghosts of #981 / #1962

Tickets (open in `ailang mission ticket open --json`; JSON snapshot /tmp/fleet_i71_tickets.json):
- `gate0:watermark-advanced-by-dying-fire` (attended, 2026-10-03)
- `gate0:watermark-advanced-before-processing` (attended, 2026-10-08)
Both say: scripts/mission_directives.sh uses an exclusive `createdAt > $since`; the watermark is written
before the directive is actioned (gate-0-preflight.md); design m-gate0-watermark-crash-safety is unbuilt.

Proposed resolution text (both tickets, `--sha 5ad75cad97c66fb279618b03c092198e2c516703`):
"Ghost: built by #1962 (merge 5ad75cad9, 2026-10-29; GitHub #981 closed 2026-10-29T07:43Z).
`scripts/mission_directives.sh --watermark-file` reads an inclusive window (`createdAt >= $since`) and the
watermark moves only via `--mark <id>` after routing; Gate 0 step 8 in both skill copies
(.claude and .agents) is PROCESS-THEN-MARK. `scripts/test_mission_directives.sh` 33/0 at d45cf4e6b,
run by `make test-launchd-drivers`. The deprecated `--since` mode stays exclusive by design (24c77c15b)."

Controller's evidence: `git log origin/dev --grep=981`; `scripts/mission_directives.sh:192-243`;
`/bin/bash scripts/test_mission_directives.sh` rc 0 "33 passed, 0 failed"; make/test.mk:93;
gate-0-preflight.md:279 in both skill copies.

## Record 2 — new decision row D-FLEET-34 (ticket stays open)

Ticket `stall:watchdog:slow-on-hung-tool` (stapledon, 2026-10-07). Asks: (1) check the controller
bash cap applies to pi controllers' find/ailang calls; (2) shorten the no-progress threshold for pi controllers.

Draft row:
"**`stall:watchdog:slow-on-hung-tool` (stapledon, 10-07): ask 1 is a ghost of #1391. Should ask 2, a shorter
no-progress threshold for pi controllers, still be built?** Measured at iteration 71 on `d45cf4e6b`.
Ask 1: yes, the cap reaches them. #1391 (`7eaf9bece`, 2026-09-29 07:24Z) loads
`tools/pi-extensions/controller-bash-cap.ts` with `-e` on every pi controller spawn
(`mission-control.sh:2480`). It clamps every pi bash call to 540 s (`MISSION_PI_BASH_CAP_S`), inside the
600-s window, and `test_driver_notify.sh:1003` pins the wiring. The stapledon 09-28 `find ~ -maxdepth 6`
hang predates it. Ask 2: the thresholds (`STALL_GRACE` 2400 s; `STALL_SAMPLES` 5 × `STALL_INTERVAL`
120 s; `mission-control.sh:1121-1127`) are the values D-FLEET-4 froze and D-FLEET-7 kept ("sample counts
and the 600-s budget unchanged"), so a change is yours. Since #1391 the slot-verdict ledgers of all 7
missions hold 133 pi-controller slots. 30 got past fire-time capacity admission, and exactly one was
KILLED: the ticket's own fleet 10-04 event (`KILLED_at=gate-5`, elapsed 5217 s), a model-side stall after
a tool result that no bash cap can bound. It came at gate 5, far past the 40-min grace, so a shorter grace
would not have changed it; fewer samples would have saved at most a few of its 10 minutes.
**A:** the fleet designs pi-only thresholds (a shorter grace and/or fewer samples when
`CONTROLLER_PROVIDER=pi`). **B (recommended):** waive ask 2 and resolve the ticket citing `7eaf9bece`. The
cap removes the hang shape the ticket describes, the one post-cap kill would not have been shortened by a
smaller grace, and a wrong kill costs a whole iteration. Default until answered: thresholds unchanged."

Controller's evidence: ~/.ailang/state/mission-*-slot-verdicts.log filtered to `$1 >= 2026-09-29T07:24`
and `controller=pi:`; mission-control.sh:1110-1135, :2455-2482; tools/pi-extensions/controller-bash-cap.ts;
retained driver logs in /tmp/ailang-mission-*.log start 2026-10-27, so STALL lines for 10-04 are rotated out.
