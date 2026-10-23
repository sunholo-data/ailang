# fleet-mission-status-archive — ITERATION INDEX

One line per iteration, newest first, covering the live log AND the archive.

**GREP THIS BEFORE PICKING WORK.** It is the cheapest way to find out whether
something has already been tried, and it is small enough to read in full — which the
log (2.8 MB, ~715k tokens) has not been for a long time.

Regenerated wholesale by `ailang mission rotate-log`, never appended to: an
append-only index drifts the moment an entry is edited, and an index that answers
confidently and wrongly is worse than none.

| # | date | what happened |
|---|---|---|
| 56 | 2026-10-22 | `rotate-log:heading-parse-stapledon` RESOLVED as a ghost of #1912 (the rig binary is stale), judged CONFIRMED by sonnet; `quorum:claude-p-ration-us... |
| 55 | 2026-10-22 | `stall:watchdog:foreground-agent-and-wrong-transcript` BUILT, merge PARKED on dev's inherited required red (D-FLEET-17) — #1929 head `9d131c76a`, j... |
| 54 | 2026-10-21 | `routing-models-check:bold-colon-heading` BUILT, merge PARKED on dev's inherited required red (D-FLEET-17) — #1927 head `ec9e54a1e`, judged PASS 80... |
| 53 | 2026-10-21 | `routing:retired-codex-models-still-spawnable-natively` LANDED — #1922 → `d75ef77f5`, judged PASS 85 twice by sonnet; ticket resolved; D-FLEET-25, ... |
| 52 | 2026-10-21 | `quota:opencode-reported-as-a-pool` LANDED — #1917 → `9aebccf9b`, judged PASS 93 by sonnet; ticket resolved; the six `agent-tool` tickets parked on... |
| 51 | 2026-10-20 | `rotate-log:legacy-heading-format` LANDED — #1912 → `3b4d11759`, judged PASS 82 then PASS 90 by sonnet; ticket resolved [HARNESS] |
| 50 | 2026-10-20 | `probe:pi-duplicate-extension-discovery` LANDED — #1907 → `e5ab9f84f`, judged PASS 90 then PASS 91 by sonnet; ticket resolved [HARNESS] |
| 49 | 2026-10-20 | `crash:sprint-executor:sigbus-ram-tmp-full-make-test` ask 1 LANDED — #1905 → `3f9a0c092`, judged PASS 86 then PASS 87 by sonnet; ticket stays open,... |
| 48 | 2026-10-19 | `skill:design-scaffold-hardcodes-owning-repo` + `sprint-skill:estimated-loc-zero-is-placeholder-sentinel` resolved as GHOSTS — already fixed by #15... |
| 47 | 2026-10-19 | `rotate-log:status-archive-structural-header` LANDED — orphaned iteration 46's #1877 verified and merged → `f048d6fa9`, judged PASS 93 by sonnet; t... |
| 45 | 2026-10-17 | `mission:index-regenerate-needs-rotatable-archive` + `rotate-log:index-regen-drops-orphan-rows` LANDED — #1870 → `2061f445e`, judged PASS 90 by son... |
| 44 | 2026-10-16 | `skill:gate4-status-stamp-shape-unprescribed` LANDED — #1866 → `91d5d6390`, judged PASS 94 then PASS 97 by sonnet; ticket resolved; D-FLEET-21 file... |
| 43 | 2026-10-16 | `skill:gate3b-ci-poll-exceeds-foreground-bash` LANDED — #1853 → `a8042fa3f`, judged PASS 90 by sonnet (round 2, after a PASS 74 with a blocker); ti... |
| 42 | 2026-10-16 | `worktree:native-background-terminal-marker-missing` LANDED — #1843 → `51f9ecb29`, judged PASS 86 by sonnet (round 2, after FAIL 66); ticket resolv... |
| 41 | 2026-10-15 | the blocked stack LANDED — #1824 → `514ca3bc3`, #1839 (ex-#1825) → `10cb2fae0`, #1840 (ex-#1828) → `7c10ce00d`; 4 tickets resolved [HARNESS] |
| 40 | 2026-10-15 | stack recovery rechecked; PARKED on D-FLEET-17, independent fresh-context Sol judge PASS 95 for disposition only [ADMIN] |
| 39 | 2026-10-14 | P2 #15 slot verdict UNCLASSIFIED made reachable — built and judged PASS 91 by sonnet (round 2); stacked on #1825, BLOCKED on dev's required reds; G... |
| 38 | 2026-10-14 | P2 #14 quorum artifact dir designed, quorum'd, built and judged PASS 90 by sonnet — stacked PR on #1824, BLOCKED on the same dev required reds [HAR... |
| 37 | 2026-10-14 | no landing possible — #12 still BLOCKED on dev's required reds (owner V1 disabled); #13 reproduction measured (B gone, A + an untracked timer flake... |
| 36 | 2026-10-14 | P2 #11 ghost-resolved (`08ce76f30`); P2 #12 quorum salvage built and judged PASS 88 by sonnet — PR #1824 BLOCKED on an inherited required red on de... |
| 35 | 2026-10-13 | P2 #10 Gate 0 ledger provenance LANDED — #1815 `130486515`, judged PASS 92 by sonnet (round 2); ticket resolved [HARNESS] |
| 34 | 2026-10-13 | Phase 3a skill-resolution spike MEASURED — P1–P3 answered with controlled marker skills, judged PASS 88 by sonnet; Phase 3b mechanism parked for Ma... |
| 33 | 2026-10-12 | P1 #9 exit-path notices bounded and store-scoped LANDED — #1791 `e7dd6c127`, judged PASS 90 by sonnet; ticket resolved; P1 tier exhausted [HARNESS] |
| 32 | 2026-10-12 | P1 #8 quorum zero-signal guard LANDED — #1767 `a5f9af982`, judged PASS 95 by sonnet; ticket resolved; orphans 30/31 credited [HARNESS] |
| 29 | 2026-10-09 | native evaluator unavailable → PARKED-ON-LANE (external native capability); documented independent fallback, no runtime fix [HARNESS] |
| 28 | 2026-10-08 | P1 #8 candidate designed, PARKED-ON-LANE — native Sonnet/Opus unsupported; declared independent judge lanes refused; no implementation [HARNESS] |
| 27 | 2026-10-08 | P1 #6 main-checkout ff-only auto-sync LANDED — #1647 `15d47b5dc` (D-FLEET-15 = A), judged PASS 91 by sonnet; ticket resolved [HARNESS] |
| 26 | 2026-10-08 | P1 #7 gate0 self-notice read LANDED — #1604 `59c3e6a55`, re-judged PASS 96 on the merged head (minimax); ticket resolved; record #1636 landed [HARN... |
| 25 | 2026-10-07 | `blocking=all` codex-controller role-env ticket LANDED — #1635 `0ceb1db01`, judged PASS 93; ticket resolved [HARNESS] |
| 24 | 2026-10-07 | native role admission failed → PARKED-ON-LANE; no independent evaluator, nothing landed [ADMIN] |
| 23 | 2026-10-06 | P1 #8 quorum guard verified live at HEAD but ALL role lanes over ration (ollama hard-capped mid-fire by the controller's own session) → PARKED-ON-L... |
| 22 | 2026-10-06 | Gate-0 self-notice read BUILT + judged PASS 87→91→93 as PR #1604; merge blocked by an inherited dev red (07e1a89bc, V1's lane) → PARKED-ON-CLOCK; P... |
| 21 | 2026-10-06 | pi-runner pre-dirty fix LANDED — #1593 `c2bf04af3` after the Actions incident cleared, re-judged PASS 100 (minimax); ticket resolved (39 → 38) |
| 20 | 2026-10-05 | pi-runner pre-dirty fix BUILT + judged PASS 86→85 as PR #1593; landing blocked by a GitHub Actions major outage → PARKED-ON-CLOCK |
| 19 | 2026-10-05 | both P0 heads LANDED — heartbeat #1578 `c55ca4398` and rotate-log pair #1580 `e7628b05e` (after a Windows-hang fix); 3 tickets resolved (41 → 38) |
| 18 | 2026-10-04 | rotate-log pair BUILT + judged PASS 97, PR #1580 PARKED-ON-CLOCK on the same weekend red as #1578; heartbeat #1578 re-probed still red |
| 17 | 2026-10-03 | heartbeat fix PUSHED as #1578, re-judged PASS 95; merge BLOCKED by a weekend-only red in `TestOllamaQuota*` (outside the diff) → PARKED-ON-CLOCK; t... |
| 16 | 2026-10-03 | heartbeat fix BUILT + judged PASS 96, push-blocked by the pinned scope guard; guard arm LANDED #1575 `adab9b7d9`; ticket stays open, pushes next fire |
| 15 | 2026-10-03 | controller HTTP 402 classified as capacity — LANDED #1549 `27dab5bb4`, evaluator PASS 95; ticket resolved |
| 14 | 2026-10-02 | controller capacity ticket verified; narrow 402 proposal PARKED on D-FLEET-11, no implementation |
| 13 | 2026-10-02 | iteration12 record recovered with separate Agent judge; heartbeat remains parked on D-FLEET-10 |
| 12 | 2026-10-02 | heartbeat design advanced to Revision 4 across three quorum rounds, then PARKED needs-human-review on a reviewer-vs-measurement deadlock (D-FLEET-1... |
| 11 | 2026-10-01 | heartbeat ticket PARKED-ON-LANE; independent evaluator unavailable in the requested Agent tool |
| 10 | 2026-10-01 | `pi-runner:quota-429-reported-as-empty-worktree` LANDED in [#1424](https://github.com/sunholo-data/ailang/pull/1424) (`94524a6fc`), independent eva... |
| 9 | 2026-09-30 | `mission-base:hardcoded-origin-dev` LANDED in [#1418](https://github.com/sunholo-data/ailang/pull/1418) (`cb7c51c8e`), independent evaluator PASS 1... |
| 8 | 2026-09-30 | paired rotate-log design PARKED · D-FLEET-9; quorum BLOCKED twice, independent Sonnet technical checkpoint PASS, implementation score UNMEASURED |
| 7 | 2026-09-29 | D-FLEET-7 stall-watchdog M2 merged in [#1399](https://github.com/sunholo-data/ailang/pull/1399) (`ce1c0639f`), independent evaluator PASS 88; merge... |
| 6 | 2026-09-28 | iteration 5's two fixes **re-judged (PASS 92, PASS 96) and LANDED in [#1377](https://github.com/sunholo-data/ailang/pull/1377) (`eec86ca4f`)**; bot... |
| 4 | 2026-09-27 | P0 #2 M1 **measured**. On the rig, `ps -S` cannot see reaped-child CPU; `proc_pid_rusage` separates the drill from both wedges. Threshold returns t... |
| 3 | 2026-09-27 | P0 #4 **PARKED (D-FLEET-6: fix needs `tools/pi-extensions/**`)**, plan committed; P1 #0 driver-env scrub **built, evaluated PASS 97, merge blocked ... |
| 2 | 2026-09-27 | P0 #2 **PARKED (policy, D-FLEET-4)**; P0 #3 commit-blind half **built, evaluated PASS 97, merge blocked on the inherited dev red** ([#1329](https:/... |
| 1 | 2026-09-26 | **P0 #1 LANDED**, `driver:slot-kill-leaves-orphan-descendants` ([#1325](https://github.com/sunholo-data/ailang/pull/1325), `e3dadcd07`) |
| 0 | 2026-09-26 | **charter RATIFIED as written** (Mark, attended) |
