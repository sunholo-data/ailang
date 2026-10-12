# fleet-mission — ITERATION INDEX

One line per iteration, newest first, covering the live log AND the archive.

**GREP THIS BEFORE PICKING WORK.** It is the cheapest way to find out whether
something has already been tried, and it is small enough to read in full — which the
log (2.8 MB, ~715k tokens) has not been for a long time.

Regenerated wholesale by `ailang mission rotate-log`, never appended to: an
append-only index drifts the moment an entry is edited, and an index that answers
confidently and wrongly is worse than none.

| # | date | what happened |
|---|---|---|
| 29 | 2026-10-09 | native evaluator capability confirmed absent → PARKED-ON-LANE (external native capability); no runtime fix [HARNESS] |
| 28 | 2026-10-08 | P1 #8 candidate designed; independent evaluator unavailable → PARKED-ON-LANE, no implementation [HARNESS] |
| 27 | 2026-10-08 | main-checkout ff-only auto-sync LANDED: #1647 `15d47b5dc` (D-FLEET-15 = A), judged PASS 91; ticket resolved [HARNESS] |
| 26 | 2026-10-08 | gate0 self-notice read LANDED: #1604 `59c3e6a55`, re-judged PASS 96 on the merged head; ticket resolved; record #1636 landed [HARNESS] |
| 25 | 2026-10-07 | `blocking=all` codex-controller role-env ticket LANDED: #1635 `0ceb1db01`, judged PASS 93; ticket resolved [HARNESS] |
| 24 | 2026-10-07 | native role pins absent; planner/evaluator Agent pins rejected → PARKED-ON-LANE, no acceptance [ADMIN] |
| 23 | 2026-10-06 | P1 #8 verified live at HEAD, but every role lane is over ration (ollama hard-capped mid-fire by this controller's own session) → PARKED-ON-LANE; ze... |
| 22 | 2026-10-06 | Gate-0 self-notice read built + judged PASS 87→91→93 as PR #1604; merge blocked by an inherited dev red (07e1a89bc) → PARKED-ON-CLOCK; P1 #6 parked... |
| 21 | 2026-10-06 | pi-runner pre-dirty fix LANDED: #1593 `c2bf04af3`, re-judged PASS 100 after the Actions incident cleared; ticket resolved [HARNESS] |
| 20 | 2026-10-05 | pi-runner pre-dirty fix built + judged PASS 86→85 as PR #1593; landing blocked by a GitHub Actions major outage → PARKED-ON-CLOCK [HARNESS] |
| 19 | 2026-10-05 | both P0 heads LANDED: heartbeat #1578 and rotate-log pair #1580 (after a Windows walk-up hang fix); 3 tickets resolved [HARNESS] |
| 18 | 2026-10-04 | rotate-log pair built + judged PASS 97 as PR #1580; parked on the same weekend TestOllamaQuota* red; #1578 re-probed still red [HARNESS] |
| 17 | 2026-10-03 | heartbeat fix pushed as #1578 and re-judged PASS 95; merge blocked by a weekend-only red in TestOllamaQuota* → PARKED-ON-CLOCK [HARNESS] |
| 16 | 2026-10-03 | heartbeat driver-root fix built + judged PASS 96; push blocked by the pinned scope guard, guard arm LANDED #1575 [HARNESS] |
| 15 | 2026-10-03 | controller HTTP 402 classified as capacity; pause no longer reads as a crash — LANDED #1549 [HARNESS] |
| 14 | 2026-10-02 | controller HTTP 402 proposal parked for D-FLEET-11; admission-bypass premise refuted [HARNESS] |
| 13 | 2026-10-02 | interrupted iteration 12 record recovered; heartbeat remains parked for D-FLEET-10 [ADMIN] |
| 12 | 2026-10-02 | heartbeat design Revision 4 after three quorum rounds; PARKED needs-human-review on reviewer-vs-measurement deadlock (D-FLEET-10) [HARNESS] |
| 11 | 2026-10-01 | heartbeat ticket parked on unavailable independent Agent evaluator lanes [HARNESS] |
| 10 | 2026-10-01 | pi runner types a provider quota refusal as `provider_quota` rc 19; stale-pin re-file of mission-base resolved [HARNESS] |
| 9 | 2026-09-30 | mission-base derives its base ref from origin/HEAD (stapledon's `main`); ticket resolved; iteration 8's record landed [HARNESS] |
| 8 | 2026-09-30 | paired rotate-log design parked after two quorum blocks; independent Sonnet review completed, compatibility decision D-FLEET-9 [HARNESS] |
| 7 | 2026-09-29 | approved stall-watchdog CPU arm merged; heartbeat design parked after two quorum blocks [HARNESS] |
| 6 | 2026-09-28 | iteration 5's kicker and sandbox fixes re-judged (PASS 92 / PASS 96) and combined in #1377; merge blocked on the inherited dev `lint` red [HARNESS] |
| 5 | 2026-09-28 | slot crashed at Gate 3b (API DNS error) with both fixes built and evaluated; no record written, recovered by iteration 6 [HARNESS] |
| 4 | 2026-09-27 | stall-watchdog CPU measured: `ps -S` blind on the rig, rusage separates drill from wedges; threshold to Mark (D-FLEET-7) [HARNESS] |
| 3 | 2026-09-27 | pi sandbox wiring parks for Mark (extension fails open); launchd suites run under an allowlisted env (built, PASS 97, merge blocked) [HARNESS] |
| 2 | 2026-09-27 | stall-watchdog descendant arm parks as policy; pi runner counts commits (built, PASS 97, merge blocked) [HARNESS] |
| 1 | 2026-09-26 | slot kill now reaps the controller's whole process tree (P0 #1) [HARNESS] |
| 0 | 2026-09-26 | charter ratified as written (attended, Mark) |
