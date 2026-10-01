### Changed — motoko mission charter reset for motoko main (2026-09-30)

Mark ruled in an attended session to reset the motoko mission for motoko main (`~/dev/mk-main`,
`sunholo/main-dst`, extension ABI 8.0, in-repo extension packages) and keep it paused until he says
otherwise; it restarts on the shell driver `tools/launchd/mission-control.sh`. The goal is unchanged
(the best harness for writing AILANG, graduating to a mission executor); the headline KPI is
motoko's pass rate against pi and opencode on the GPU rotation, plus the cloud equivalent.

- `design_docs/motoko-mission.md`: new Repo Profile, CURRENT GOAL, Premise log (P1–P14), guardrails
  (Phase-0 gate dropped), clause 2 of the bar reworded to "extensions build from the pinned motoko main
  commit", and a new queue (rows 20–25: cloud executor on motoko main, upstream #200, fmt port and
  `context_limit`, the paired KPI instrument, `ailang_tools` on a set with headroom, executor-lane
  trial) plus the fork-era loop-health rows still open at HEAD. Decision ledger: `D-MOTOKO-RESET-1/2/3`
  resolved, `D-MOTOKO-RESTART-1` open.
- The fork-era queue, premise log and ITERATION 37 stamp moved verbatim to
  `design_docs/motoko-mission-status-archive.md`; log entries 18–39 moved to
  `design_docs/motoko-mission-log-archive.md`, so the live log starts empty at iteration 40.
- `m-motoko-dst-refactor-migration.md` and `m-motoko-fork-disposition.md` moved to
  `design_docs/implemented/v0_48_0/` — the migration is complete (see `MOTOKO.md`).
