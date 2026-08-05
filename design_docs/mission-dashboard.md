# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-06 ~00:15 local (iteration 147)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 + #498 Lane B M1/M2
- **CI-flake M4 LANDED** — PR #599 → `4b47f8b0a`, the sprint's **only CI-touching commit**. Poison
  across **6** legs + AC9 gatelint registration (AC9/AC11/AC12). Evaluator sonnet 88/100 r1, zero
  blocking. **Gate 3b GREEN on the dev merge: 15/15, 0 non-success**, incl. all 4 build legs and
  `test-windows` — closing the PowerShell guard that was unverifiable locally.
- **`#569` landed first** (`bc30912ea`, actions bump) to clear the ci.yml/build.yml collision.
- **Next**: **M5** — docs/CHANGELOG/AC sweep. Standing alternative: **#498 Lane B M3** (final Lane B
  milestone → then the release ask to Mark).
- **Loops**: v1 90min · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `claude:claude-fable-5` → next `codex:gpt-5.6-sol`

## Parked on Mark
- **Nothing blocking.** D5 answered; iter-141 carve-out ratified.
- Low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op briefing

## Recently settled (don't re-ask)
- **D5 = Option A**; Option B queued as `m-net-effect-proxy-boundary`, gated on AC10(d), which exists
- **#532 closed as SUPERSEDED** · **standing fast-forward** ratified (applied again this iteration)

## Known-deferred (measured, not forgotten)
- **`go mod download all` writes to the TRACKED go.sum** — prefetch must precede binary building or
  the staleness detector silently skips every binary-gated test. Cost iter-147 a red CI run.
- **`-shellcheck='-e SC2086'` DISABLES shellcheck entirely** (the flag takes an executable path).
  Never reuse it as a filter. 5 pre-existing findings keep `actionlint → rc=0` unpassable.
- **`#598`**: `TestSolve_HardTimeout_FakeSolverIgnoringT` pid-file race — 1 observation, **0/12** on
  a stress arm, so the mechanism is code-reading, not proven.
- **C2 watch-item is WEAKER than recorded**: the 30s budget wraps only `probeServeAPIMCPTools`
  (subtests 0.75s/0.03s), so the margin is ~40×, not 2.9×.
- **M5 owes**: "5 CI legs"→6 in 6 places; AC10 a/b/c→(d) reconciliation; the actionlint findings above.

## Quota posture (week of 2026-08-03)
- **Metered**: iter-147 spent **$0.00** of the $5/iteration ceiling (codex on subscription, no quorum)

## Bookkeeping
- Issue: `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
