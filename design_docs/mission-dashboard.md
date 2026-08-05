# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log, NOT here). Any fresh session reads THIS + MEMORY.md
> and has full steering context. Humans steer via comments on the bookkeeping issue (directive
> channel) or attended charter stamps — never by needing a long-lived chat thread.

**Updated**: 2026-08-05 ~02:45 local / 00:45Z (iteration 143)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 (#546 closed) + #498 Lane B M1; prod MCP serving 0.33.0
- **In flight**: **m-ci-flake-systemic-fix** — doc `dec17dab1`, plan `7cb798d98` (5 milestones / 26h)
- **M1 LANDED** (iter-143, PR #591): `internal/testutil` live-net gate + bounded subprocess helpers, 357 LOC, evaluator sonnet **PASS 92/100 r1, zero blocking**
- **⚠ M2 + M4 STILL BLOCKED on Mark (D5 below)**; M3 depends on M2 → transitively blocked; **M5 is docs-only**
- **Next**: no unblocked CI-flake milestone remains — next pick is `#498` Lane B **M2**, unless D5 lands first
- **Loops**: v1 90min cadence · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner opus (lane fail-closed)
- **Designer rotation**: last-used `claude:claude-fable-5` → next is `codex:gpt-5.6-sol` (unchanged — no designer fired)

## Parked on Mark
- **D5 (iter-142, BLOCKING M2/M4/M3)**: the poisoned-proxy egress boundary does **not** cover AILANG's own
  `Net` effect — 6 hand-built `http.Transport{}` in `internal/effects` set no `Proxy`, so AC3 passed
  pre-sprint *by reaching the live internet*. **(A)** leave it outside + narrow AC3 (controller rec), or
  **(B)** set `ProxyFromEnvironment` — a production change touching the pinned-IP SSRF guard, needs its own quorum.
  **Now blocking 3 of 5 milestones — this is the single highest-value unblock in the queue.**
- **Carve-out disclosure (iter-141)**: the CI-flake doc's R3 fix was applied under the narrow-refinement
  carve-out with **no re-quorum on that fix**. Still veto-able before the remaining milestones run.
- Standing low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op lang-items briefing offer

## Recently settled (don't re-ask)
- **Curation now runs through the loop**, not attended side-sessions (Mark 08-04; charter guardrail)
- **CI-flake sprint authorized** (Mark 08-04, "Yes sprint a CI flake fix")

## Quota posture (week of 2026-08-03)
- Heaviest consumers last audit: controller 41% · attended sessions ~30% · opus sub-agents ~20-25% (moving to codex) · sonnet evaluator ~9%
- Watch: all-models bucket pace vs Thursday; Fable fallback OK'd but bounded by judgment
- **Metered**: iter-143 spent **$0.00** of the $5/iteration ceiling (codex executor on subscription; no quorum round)

## Bookkeeping
- Issue: see `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `design_docs/v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
