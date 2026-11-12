# Fleet mission dashboard — iteration 109

Updated 2026-11-12; scheduled unattended. [Bookkeeping #2172](https://github.com/sunholo-data/ailang/issues/2172) (predecessor #2039, closed).

- **Outcome: LANDED / HARNESS.** `orphan:pi-runner:own-pgid-children-survive-role-exit` (stapledon 89) merged as #2267 → `47b123139`. Independent sonnet judge PASS 86, then PASS 91; 0 blocking. Ticket resolved (stapledon notified).
  - `scripts/mission_pi_run.sh` now reaps what a pi role leaves behind, on every outcome including `ok`.
  - It uses an ancestry ledger matched by pid plus birth, never by command line, pgid or session.
  - The verdict gains `reaped_count` and `reaped` (the executable name only).
- **Inputs:** 28 tickets became 27 after the resolution. Ledger: 44 rows, 28 OPEN (D-FLEET-16..23, 25..29, 31..45). 0 directives on #2172 or #2039.
- **Routing:**
  - Planner: opus (Agent tool).
  - Executor: `claude:claude-haiku-5-5` via claude-sub. Round 1 hit its 1800 s alarm after committing all three milestones; round 2 finished rc 0.
  - Judge: sonnet (Agent tool), two rounds.
  - Designer: not spawned.
  - The driver moved planner and executor off codex `gpt-6.1-sol` (probe rc 75).
- **Done-gate:**
  - `make test-launchd-drivers` 58/58 at `0d0d73a4a`, and 18/18 for the new suite (the full run hit its alarm at load ~200; only the unrelated driver-notify timing arms failed) on the merge commit.
  - Healthy and degraded dry-runs passed under stapledon.
  - Dev CI on `47b123139`: `launchd drivers (bash 3.2)` success, 0 red; macOS build legs queued behind a repo-wide runner backlog.
- **Goal:** clause 2 moved; this ticket was resolved about 4.2 h (filed 12:41:53Z, resolved 16:54Z) after it was filed. Clause 1 UNMEASURED, 3 MET, 4 met by its letter, 5 preserved. Harness share 8/20 (iterations 90–109).
- **Cost:** metered $0. Everything ran on subscription: planner 174k, judge 103k + 125k, and executor haiku (round 2: 65k output tokens).
- **Records 95–109** sit on stacked draft PRs (#2169 → … → #2250 → iteration 109). origin/dev's log still ends at 94.
- **Gate 0 self-notice:** the watermark form returned rc 2 for the thirteenth fire in a row. The `--since` form found 0 crash notices.
- **Observation, not ticketed:** a `claude -p` executor killed by its alarm leaves its running tool shell orphaned (ppid 1). This is the same class as this ticket, one layer up.
- **Next:** any ruling on D-FLEET-16..45, or a new ticket.
- **Evidence:** [log](fleet-mission-log.md), [iteration 109](fleet-mission-evidence/iteration109/).
