# Fleet Mission Dashboard

Snapshot: iteration 10 — 2026-10-01. History: [charter](fleet-mission.md), [log](fleet-mission-log.md).

- **Latest release**: v0.49.0; Fleet does not release.
- **This fire**: `pi-runner:quota-429-reported-as-empty-worktree` LANDED in [#1424](https://github.com/sunholo-data/ailang/pull/1424) (`94524a6fc`): verdict `provider_quota` rc 19. Independent evaluator PASS 97; ticket resolved (world notified). Stapledon's stale-pin re-file of `mission-base:hardcoded-origin-dev` was resolved against `cb7c51c8e`.
- **Next READY**: P1 #5, the pre-dirty half of `pi-runner:verdict-blind-to-commits-and-predirty`.
- **Parked for Mark**: D-FLEET-8 (heartbeat + `.agents` mirror scope; now also holds the gate-3-route.md rc-19 line), D-FLEET-9 (rotate-log flag compatibility + design scope).
- **Needs a filer (2nd fire)**: pi roles cannot start in any ailang-repo worktree. The `~/.pi/agent/extensions` tools collide with the repo's `.pi/extensions` (`Tool "ailang_run" conflicts …`), and the runner reports it as `sandbox_not_ready` rc 17. The pi evaluator lane is dead every fire.
- **Open tickets**: 14. Clause 2 at risk (this ticket ~4.5 days); clauses 3–5 met; clause 1 unmeasured.
- **CI**: dev green on required checks; SonarCloud new-code coverage red on every recent dev commit (inherited, not fleet's).
- **Loop**: every 6h, dev.ailang.mission-fleet, origin/dev pin; zero-ticket exit before spend.
- **Actual roles**: controller claude-opus-5-5; planner opus (Agent); executor claude-sonnet-5-5 (claude-sub); evaluator pi minimax-m3 dead (rc 17) → claude-sonnet-4-6 (claude-sub, FLAGGED fallback).
- **Quota**: Anthropic subscription only (billing tripwire CLEAN); metered $0 (one pi launch with no model call).
