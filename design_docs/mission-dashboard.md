# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log, NOT here). Any fresh session reads THIS + MEMORY.md
> and has full steering context. Humans steer via comments on the bookkeeping issue (directive
> channel) or attended charter stamps — never by needing a long-lived chat thread.

**Updated**: 2026-08-05 ~23:35 local (iteration 147)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 + #498 Lane B M1/M2; prod MCP on 0.33.0
- **`#569` LANDED** (dependabot actions bump → `bc30912ea`, dev CI green): setup-go v6→v7,
  setup-uv v8.3.2→v9.0.0. Merged FIRST to clear the ci.yml/build.yml collision with M4.
- **CI-flake M4 IN FLIGHT** — PR **#599** (`6d8fb2474`), the sprint's **only CI-touching commit**.
  Poison wired across **6** legs + AC9 gatelint registration. First CI run went **RED on M4's own
  AC9 step** and was fixed forward; second run polling at hand-off. **If red: `git revert --no-edit
  <squash-sha>` is the staged remedy.**
- **THE FIND**: `go mod download all` writes to the **tracked go.sum**, and the binary-staleness
  detector compares binary mtime vs newest Go source — so prefetching AFTER `Build binaries` makes
  every ailang binary read STALE and silently skips every binary-gated test. Fix = prefetch BEFORE
  building (build.yml already did, which is why it passed). Reproduced locally both directions.
- **Next**: **M5** (docs/CHANGELOG/AC sweep + the doc↔plan reconciliation).
  Standing alternative: **#498 Lane B M3** (final Lane B milestone → then release ask to Mark).
- **Loops**: v1 90min cadence · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `claude:claude-fable-5` → next `codex:gpt-5.6-sol` (no designer fired)

## Parked on Mark
- **Nothing blocking.** D5 answered; iter-141 carve-out ratified.
- Standing low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator ·
  ?-op lang-items briefing offer

## Recently settled (don't re-ask)
- **D5 = Option A**; Option B queued as `m-net-effect-proxy-boundary`, gated on AC10(d), which now EXISTS
- **PR #532 closed as SUPERSEDED** · **Standing fast-forward** ratified (applied again this iteration)

## Known-deferred (measured, not forgotten)
- **`#598` NEW**: `TestSolve_HardTimeout_FakeSolverIgnoringT` fataled once on a pid-file race
  (1s solver timeout vs the fake solver's `echo $! >` write). Observed 1×, **0/12 on a targeted
  stress arm** — mechanism is code-reading, not proven.
- **C2 watch-item is WEAKER than recorded**: the 30s budget wraps only `probeServeAPIMCPTools`,
  whose subtests run **0.75s / 0.03s**; the 11.18s is setup OUTSIDE that context. Margin ~40×, not 2.9×.
- **M5 cleanup owed**: doc says "5 CI legs" in **6** places though its own V34 measured 6; 5
  pre-existing actionlint/shellcheck findings (3 SC2086, 1 SC2035, 1 SC2046) keep the plan's
  `actionlint → rc=0` gate unpassable. **`-shellcheck='-e SC2086'` DISABLES shellcheck entirely**
  (the flag takes an executable path) — do not reuse it as a filter.

## Quota posture (week of 2026-08-03)
- **Metered**: iter-147 spent **$0.00** of the $5/iteration ceiling (codex on subscription, no quorum)

## Bookkeeping
- Issue: see `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `design_docs/v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
