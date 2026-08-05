# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log, NOT here). Any fresh session reads THIS + MEMORY.md
> and has full steering context. Humans steer via comments on the bookkeeping issue (directive
> channel) or attended charter stamps — never by needing a long-lived chat thread.

**Updated**: 2026-08-05 ~20:45 local (iteration 146)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 + #498 Lane B M1/M2; prod MCP on 0.33.0
- **CI-flake M3 LANDED** (iter-146, PR #597 → squash `13c570063`): `internal/testutil/gatelint`
  (R1/R2/R3 legibility lint, 941 files scanned, 0 violations) + `egress_posture_test.go` closing
  **AC8 + AC10(a/b/c/d)**. Evaluator sonnet **79/100 r1 FAIL → fixed → green**.
- **Next**: **CI-flake M4** — the **ONLY CI-touching commit** (poison wiring across **6** legs:
  ci.yml `test` + `test-windows`, build.yml's 4-entry matrix, make/test.mk). Land with
  `git revert --no-edit <M4-sha>` staged; watch the first `dev` run. **Re-check `#569` first** —
  it touches ci.yml + build.yml = M4's exact surface. Then **M5** (docs/CHANGELOG/AC sweep).
  Standing alternative: **#498 Lane B M3** (final Lane B milestone → then release ask to Mark).
- **Loops**: v1 90min cadence · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `claude:claude-fable-5` → next `codex:gpt-5.6-sol` (no designer fired)

## Parked on Mark
- **Nothing blocking.** D5 answered; iter-141 carve-out ratified.
- Standing low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator ·
  ?-op lang-items briefing offer

## Recently settled (don't re-ask)
- **D5 = Option A**; **Option B queued separately** as `m-net-effect-proxy-boundary`, gated on
  AC10(d) existing first — **AC10(d) now EXISTS and is its landing tripwire** (it reds when B lands)
- **PR #532 closed as SUPERSEDED** (by #564) · **Standing fast-forward** ratified · curation runs
  through the loop

## Known-deferred (measured, not forgotten)
- **C2 generator survives**: 31 absolute `context.WithTimeout(Background(), N)` in tests vs 2
  deadline-derived; gatelint R1–R3 cover none of it. `serve_api_mcp_surface_test.go:60` (30s
  budget, 10.34s actual) — **watch on the first dev run after M4**.
- **C3 generator survives**: 62 unbounded `exec.Command` in tests (§6.3, deferred by plan)
- **M5 cleanup owed**: doc says "5 CI legs" in **6** places though its own V34 measured **6**;
  doc's Files-to-Create + plan's M3 still say AC10 "a/b/c" though (d) shipped. SonarCloud
  new_coverage gap from M1 (`String()` at 0.0%; `RequiresLiveNetwork` 0.0% is a re-exec artifact —
  do NOT "fix" by weakening the subprocess pattern).

## Quota posture (week of 2026-08-03)
- **Metered**: iter-146 spent **$0.00** of the $5/iteration ceiling (codex on subscription, no quorum)

## Bookkeeping
- Issue: see `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `design_docs/v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
