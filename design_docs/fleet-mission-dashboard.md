# Fleet Mission Dashboard

Snapshot: iteration 9 — 2026-09-30. History: [charter](fleet-mission.md), [log](fleet-mission-log.md).

- **Latest release**: v0.49.0; Fleet does not release.
- **This fire**: `mission-base:hardcoded-origin-dev` LANDED in [#1418](https://github.com/sunholo-data/ailang/pull/1418) (`cb7c51c8e`), independent evaluator PASS 100, ticket resolved (stapledon notified). Iteration 8's record [#1416](https://github.com/sunholo-data/ailang/pull/1416) also landed (`60af11ee0`).
- **Next READY**: P1 #4 quota-429 verdict in `mission_pi_run.sh`, then #5 pre-dirty verdict (Mark's 2026-09-29 order).
- **Parked for Mark**: D-FLEET-8 (heartbeat mirror scope), D-FLEET-9 (rotate-log flag compatibility + design scope).
- **New finding for Mark**: pi roles cannot start in any ailang-repo worktree. The ailang-managed `~/.pi/agent/extensions` tools collide with the repo's `.pi/extensions`, and the runner reports it as `sandbox_not_ready`. The fleet cannot file tickets, so it needs a filer.
- **Open tickets**: 15. Clause 2 improved (one ticket, ~72h); clauses 3–5 met; clause 1 unmeasured.
- **CI**: dev green on required checks; SonarCloud new-code coverage red (74.5% < 80) on every recent dev commit — inherited, not fleet's.
- **Loop**: every 6h, dev.ailang.mission-fleet, origin/dev pin; zero-ticket exit before spend.
- **Actual roles**: controller claude-opus-5-5; planner opus (Agent); executor claude-sonnet-5-5 (claude-sub); evaluator pi minimax-m3 dead → claude-sonnet-4-6 (claude-sub, FLAGGED fallback).
- **Quota**: Anthropic subscription only (billing tripwire CLEAN); no metered spend beyond two 1-token pi probes (<$0.001).
