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
| 59 | 2026-10-23 | watchdog fix #1937 recovered after the post-merge capacity stop; independent MiniMax PASS94/100; ticket resolved [HARNESS] |
| 58 | 2026-10-22 | `routing-models-check:bold-colon-heading` LANDED through replacement #1935; independent MiniMax PASS 93; ticket resolved [HARNESS] |
| 57 | 2026-10-22 | `quorum-reader:zsh-nomatch` BUILT: #1932 head `6268fb0b8`; judge PASS 94 (sonnet); merge PARKED on dev's inherited required red (D-FLEET-17); ticke... |
| 56 | 2026-10-22 | `rotate-log:heading-parse-stapledon` RESOLVED as a ghost of #1912 (stale rig binary), judge CONFIRMED (sonnet); `quorum:claude-p-ration-usage-429` ... |
| 55 | 2026-10-22 | `stall:watchdog:foreground-agent-and-wrong-transcript` BUILT: #1929 head `9d131c76a`; judge PASS 91, round 2 PASS 91 (sonnet); merge PARKED on dev'... |
| 54 | 2026-10-21 | `routing-models-check:bold-colon-heading` BUILT: #1927 head `ec9e54a1e`; judge PASS 80 then PASS 88 (sonnet); merge PARKED on dev's inherited requi... |
| 53 | 2026-10-21 | `routing:retired-codex-models-still-spawnable-natively` LANDED: #1922 → `d75ef77f5`; judge PASS 85, round 2 PASS 85 (sonnet); ticket resolved; D-FL... |
| 52 | 2026-10-21 | `quota:opencode-reported-as-a-pool` LANDED: #1917 → `9aebccf9b`; judge PASS 93 (sonnet); ticket resolved; six `agent-tool` tickets parked on D-FLEE... |
| 51 | 2026-10-20 | `rotate-log:legacy-heading-format` LANDED: #1912 → `3b4d11759`; judge PASS 82 then PASS 90 (sonnet); ticket resolved [HARNESS] |
| 50 | 2026-10-20 | `probe:pi-duplicate-extension-discovery` LANDED: #1907 → `e5ab9f84f`; judge PASS 90 then PASS 91 (sonnet); ticket resolved [HARNESS] |
| 49 | 2026-10-20 | `crash:sprint-executor:sigbus-ram-tmp-full-make-test` ask 1 LANDED: #1905 → `3f9a0c092`; judge PASS 86 then PASS 87 (sonnet); ticket stays open, as... |
| 48 | 2026-10-19 | `skill:design-scaffold-hardcodes-owning-repo` + `sprint-skill:estimated-loc-zero-is-placeholder-sentinel` resolved as GHOSTS (#1542 `b51353bcd`, #1... |
| 47 | 2026-10-19 | `rotate-log:status-archive-structural-header` LANDED: orphaned iteration 46's #1877 verified and merged → `f048d6fa9`; judge PASS 93 (sonnet); tick... |
| 45 | 2026-10-17 | `mission:index-regenerate-needs-rotatable-archive` + `rotate-log:index-regen-drops-orphan-rows` LANDED: #1870 → `2061f445e`; judge PASS 90 (sonnet)... |
| 44 | 2026-10-16 | `skill:gate4-status-stamp-shape-unprescribed` LANDED: #1866 → `91d5d6390`; judge PASS 94 then PASS 97 (sonnet); ticket resolved [HARNESS] |
| 43 | 2026-10-16 | `skill:gate3b-ci-poll-exceeds-foreground-bash` LANDED: #1853 → `a8042fa3f`; judge PASS 74 (blocker) then PASS 90 (sonnet); D-FLEET-20 filed [HARNESS] |
| 42 | 2026-10-16 | `worktree:native-background-terminal-marker-missing` LANDED: #1843 → `51f9ecb29`; judge FAIL 66 then PASS 86 (sonnet); ticket resolved [HARNESS] |
| 41 | 2026-10-15 | The blocked stack LANDED: #1824 → `514ca3bc3`, #1839 (ex-#1825) → `10cb2fae0`, #1840 (ex-#1828) → `7c10ce00d`; three CI-only defects fixed on the w... |
| 40 | 2026-10-15 | stack recovery rechecked; required dev reds persist, no further builds; independent disposition review PASS 95 (fresh-context Sol) [ADMIN] |
| 39 | 2026-10-14 | P2 #15 slot verdict UNCLASSIFIED made reachable, judged PASS 91 (sonnet, round 2); stacked on #1825, BLOCKED on dev's required reds; Gate 1 skill f... |
| 38 | 2026-10-14 | P2 #14 quorum artifact dir designed, quorum'd, built and judged PASS 90 (sonnet); stacked PR on #1824, BLOCKED on dev's required reds [HARNESS] |
| 37 | 2026-10-14 | No landing possible: #12 (PR #1824) still BLOCKED on dev's required reds; #13 reproduction measured, live flakes outside fleet scope; judged PASS 8... |
| 36 | 2026-10-14 | P2 #11 ghost-resolved (`08ce76f30`); P2 #12 quorum salvage designed, quorum'd, built and judged PASS 88 (sonnet), PR #1824 BLOCKED on inherited dev... |
| 35 | 2026-10-13 | P2 #10 Gate 0 ledger provenance instrument, judged PASS 92 (sonnet, round 2), PR #1815 → `130486515` [HARNESS] |
| 34 | 2026-10-13 | Phase 3a skill-resolution spike measured (P1–P3), judged PASS 88 (sonnet); Phase 3b mechanism parked as D-FLEET-16 [HARNESS] |
| 33 | 2026-10-12 | P1 #9 exit-path notices bounded and store-scoped, judged PASS 90 (sonnet), PR #1791 → `e7dd6c127` [HARNESS] |
| 32 | 2026-10-12 | P1 #8 quorum zero-signal guard BUILT and judged PASS 95 (sonnet), PR #1767; orphan iterations 30/31 credited [HARNESS] |
| 31 | 2026-10-12 | ORPHAN (no record written): P1 #8 revisions replayed onto `0a2202ad6`, plan left untracked; slot PAUSED-NO-CAPACITY at gate-3 [HARNESS] |
| 30 | 2026-10-09 | ORPHAN (no record written): P1 #8 quorum rounds 1–2 BLOCKED, revisions 2–3 under the carve-out; slot PAUSED-NO-CAPACITY at gate-2 [HARNESS] |
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
