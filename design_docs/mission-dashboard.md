# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log, NOT here). Any fresh session reads THIS + MEMORY.md
> and has full steering context. Humans steer via comments on the bookkeeping issue (directive
> channel) or attended charter stamps — never by needing a long-lived chat thread.

**Updated**: 2026-08-04 ~16:30 (attended bootstrap — first Gate-4 refresh pending)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 (#546 closed) + #498 Lane B M1; prod MCP serving 0.33.0
- **In flight / next pick**: recorded-stream S2 → evaluator-gemini lane → github-issue-triage batch (#573 soundness check first) → M4b (KPI baseline, $20 cap)
- **Loops**: v1 90min cadence · world 4h · both armed
- **Routing**: controller opus-5 · planner+executor codex (ChatGPT bucket) · evaluator sonnet · quorum gemini/gpt metered

## Parked on Mark
- (none blocking — release ask satisfied by v0.33.0)
- Standing low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op lang-items briefing offer

## Quota posture (week of 2026-08-03)
- Heaviest consumers last audit: controller 41% · attended sessions ~30% · opus sub-agents ~20-25% (moving to codex) · sonnet evaluator ~9% (gemini next)
- Watch: all-models bucket pace vs Thursday; Fable fallback OK'd but bounded by judgment

## Bookkeeping
- Issue: see `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `design_docs/v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
