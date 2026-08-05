# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log, NOT here). Any fresh session reads THIS + MEMORY.md
> and has full steering context. Humans steer via comments on the bookkeeping issue (directive
> channel) or attended charter stamps — never by needing a long-lived chat thread.

**Updated**: 2026-08-05 ~15:10 local (iteration 145)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 + #498 Lane B M1/M2; prod MCP on 0.33.0
- **D5 ANSWERED** (Mark, Option A): AILANG's `Net` effect stays OUTSIDE the egress boundary.
  AC3 → AC3′(a/b/c), new **AC10(d)** asserts the residual openly. **CI-flake M2/M3/M4 unblocked.**
- **CI-flake M2 LANDED** (iter-145, PR #593): live-network tests no longer run by default —
  `httpPost_to_httpbin.org` FAIL(503) → **SKIP**, plus 2 deterministic `httptest` subtests.
  Evaluator sonnet **PASS 91/100 r1, zero blocking**.
- **Next**: **CI-flake M3** (gatelint + egress posture probe incl. the new AC10(d); pure Go, zero
  workflow edits) → then **M4**, the ONLY CI-touching commit → M5 docs.
  Standing alternative: **#498 Lane B M3** (final Lane B milestone → then release ask to Mark).
- **Loops**: v1 90min cadence · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `claude:claude-fable-5` → next `codex:gpt-5.6-sol` (no designer fired)

## Parked on Mark
- **Nothing blocking.** D5 answered; iter-141 carve-out ratified (veto window closed).
- Standing low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator ·
  ?-op lang-items briefing offer

## Recently settled (don't re-ask)
- **D5 = Option A**; **Option B queued separately** as `m-net-effect-proxy-boundary` (needs own quorum;
  gated on AC10(d) existing first — AC10(d) is its landing tripwire)
- **PR #532 closed as SUPERSEDED** — its `sync.Once` fix was already on dev via #564 (`3c28cc322`).
  The "#532 blocks M2" collision the plan carried was already moot.
- **Curation runs through the loop**, not attended side-sessions · **Standing fast-forward** ratified

## Known-deferred (measured, not forgotten)
- **C2 generator survives**: 31 absolute `context.WithTimeout(Background(), N)` in tests vs 2
  deadline-derived; gatelint R1–R3 don't cover it. `serve_api_mcp_surface_test.go:60` (30s budget,
  10.34s actual, woken by M2) — **watch on the first dev run after M4**.
- **C3 generator survives**: 62 unbounded `exec.Command` in tests (§6.3, deferred by plan)

## Quota posture (week of 2026-08-03)
- **Metered**: iter-145 spent **$0.00** of the $5/iteration ceiling (codex on subscription, no quorum round)

## Bookkeeping
- Issue: see `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `design_docs/v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
