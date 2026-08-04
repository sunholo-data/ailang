# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log, NOT here). Any fresh session reads THIS + MEMORY.md
> and has full steering context. Humans steer via comments on the bookkeeping issue (directive
> channel) or attended charter stamps — never by needing a long-lived chat thread.

**Updated**: 2026-08-04 ~17:15 local / 15:15Z (iteration 141)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 (#546 closed) + #498 Lane B M1; prod MCP serving 0.33.0
- **In flight / next pick**: **m-ci-flake-systemic-fix SPRINT** (doc landed `dec17dab1`, quorum-cleared via carve-out — plan it next) → `#498` Lane B **M2** → recorded-stream S2
- **Loops**: v1 90min cadence · world 4h · both armed
- **Routing**: controller opus-5 · planner+executor codex (ChatGPT bucket) · evaluator sonnet · quorum gemini/gpt metered
- **Designer rotation**: last-used `claude:claude-fable-5` → next is `codex:gpt-5.6-sol`

## Parked on Mark
- **Carve-out disclosure (iter-141)**: the CI-flake doc's R3 fix was applied by the controller under
  the narrow-refinement carve-out with **no re-quorum on that fix**. Veto-able before the sprint runs.
- Standing low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op lang-items briefing offer

## Recently settled (don't re-ask)
- **Curation now runs through the loop**, not attended side-sessions (Mark 08-04; charter guardrail) — evidence: iter-140's 2h red dev
- **CI-flake sprint authorized** (Mark 08-04, "Yes sprint a CI flake fix")

## Quota posture (week of 2026-08-03)
- Heaviest consumers last audit: controller 41% · attended sessions ~30% · opus sub-agents ~20-25% (moving to codex) · sonnet evaluator ~9% (gemini next)
- Watch: all-models bucket pace vs Thursday; Fable fallback OK'd but bounded by judgment
- **Metered**: iter-141 spent **$0.26** of the $5/iteration ceiling (3 quorum rounds)

## Bookkeeping
- Issue: see `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `design_docs/v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
