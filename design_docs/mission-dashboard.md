# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log, NOT here). Any fresh session reads THIS + MEMORY.md
> and has full steering context. Humans steer via comments on the bookkeeping issue (directive
> channel) or attended charter stamps — never by needing a long-lived chat thread.

**Updated**: 2026-08-05 ~11:15 local (iteration 144)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 (#546 closed) + #498 Lane B M1; prod MCP serving 0.33.0
- **In flight**: **#498 Lane B** (embeddable exact tool surface, World's clause-6 blocker)
- **M2 LANDED** (iter-144, PR #592 → `6166adab8`): request-scoped MCP adapter, frozen -32603 envelopes,
  no ambient submit_feedback; evaluator sonnet **PASS 94/100 r1, zero blocking**
- **Next**: **Lane B M3** (FINAL milestone, ~10h: A2A projection + Mount + gates) → then Lane B COMPLETE
  → **release ask to Mark** (World consumes pinned releases only). CI-flake M2/M4/M3 still blocked on D5.
- **Loops**: v1 90min cadence · world 4h · both armed
- **Routing**: controller opus-5 (⚠ iter-144 ran on **fable** — `$MODEL` arrived empty; check driver export)
  · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `claude:claude-fable-5` → next is `codex:gpt-5.6-sol` (unchanged — no designer fired)

## Parked on Mark
- **D5 (iter-142, BLOCKING CI-flake M2/M4/M3)**: poisoned-proxy egress boundary does **not** cover AILANG's
  own `Net` effect (6 hand-built `http.Transport{}`, no `Proxy`). **(A)** leave outside + narrow AC3
  (controller rec) or **(B)** `ProxyFromEnvironment` = production change, own quorum. Highest-value unblock.
- **Carve-out disclosure (iter-141)**: CI-flake doc R3 fix applied under narrow-refinement carve-out,
  no re-quorum on that fix. Still veto-able before the remaining milestones run.
- Standing low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op lang-items briefing offer

## Recently settled (don't re-ask)
- **Curation runs through the loop**, not attended side-sessions (Mark 08-04; charter guardrail)
- **Standing fast-forward** ratified (0-ahead + known dirty files only) — applied iter-144, clean

## Quota posture (week of 2026-08-03)
- Heaviest consumers last audit: controller 41% · attended ~30% · opus sub-agents ~20-25% (moving to codex) · sonnet evaluator ~9%
- **Metered**: iter-144 spent **$0.00** of the $5/iteration ceiling (codex on subscription; no quorum round)

## Bookkeeping
- Issue: see `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `design_docs/v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
