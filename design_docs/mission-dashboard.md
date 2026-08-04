# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log, NOT here). Any fresh session reads THIS + MEMORY.md
> and has full steering context. Humans steer via comments on the bookkeeping issue (directive
> channel) or attended charter stamps — never by needing a long-lived chat thread.

**Updated**: 2026-08-04 ~21:05 local / 19:05Z (iteration 142)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 (#546 closed) + #498 Lane B M1; prod MCP serving 0.33.0
- **In flight**: **m-ci-flake-systemic-fix** — doc `dec17dab1`, **sprint plan LANDED `7cb798d98`** (5 milestones / 26h / ~4.5d, revised up from 3–4d)
- **⚠ M2 + M4 BLOCKED on Mark (D5 below); M1 unblocked** — next pick is M1 (5h, zero blast radius) unless D5 lands first
- **Then**: `#498` Lane B **M2** → recorded-stream S2
- **Loops**: v1 90min cadence · world 4h · both armed
- **Routing**: controller opus-5 · planner opus (lane fail-closed) · executor codex · evaluator sonnet · quorum gemini/gpt metered
- **Designer rotation**: last-used `claude:claude-fable-5` → next is `codex:gpt-5.6-sol` (unchanged — no designer fired)

## Parked on Mark
- **D5 (NEW, iter-142, BLOCKING M2/M4)**: the poisoned-proxy egress boundary does **not** cover AILANG's own
  `Net` effect — 6 hand-built `http.Transport{}` in `internal/effects` set no `Proxy`, so AC3 passed
  pre-sprint *by reaching the live internet*. **(A)** leave it outside + narrow AC3 (controller rec), or
  **(B)** set `ProxyFromEnvironment` — a production change touching the pinned-IP SSRF guard, needs its own quorum.
- **Carve-out disclosure (iter-141)**: the CI-flake doc's R3 fix was applied under the narrow-refinement
  carve-out with **no re-quorum on that fix**. Still veto-able before the sprint runs.
- Standing low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op lang-items briefing offer

## Recently settled (don't re-ask)
- **Curation now runs through the loop**, not attended side-sessions (Mark 08-04; charter guardrail) — evidence: iter-140's 2h red dev
- **CI-flake sprint authorized** (Mark 08-04, "Yes sprint a CI flake fix")

## Quota posture (week of 2026-08-03)
- Heaviest consumers last audit: controller 41% · attended sessions ~30% · opus sub-agents ~20-25% (moving to codex) · sonnet evaluator ~9%
- Watch: all-models bucket pace vs Thursday; Fable fallback OK'd but bounded by judgment
- **Metered**: iter-142 spent **$0.00** of the $5/iteration ceiling (no quorum round, no metered lane)

## Bookkeeping
- Issue: see `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `design_docs/v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
