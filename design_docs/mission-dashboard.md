# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-06 ~04:30 local (iteration 149)

## Now
- **Latest release**: v0.33.0 (2026-08-04) — recorded-stream S1 + #498 Lane B M1/M2
- **M-CI-FLAKE-SYSTEMIC-FIX IS COMPLETE** — M5 landed (PR #600 → `c9e1a4f98`); all 5 milestones in,
  doc + plan moved to `design_docs/implemented/v0_33_1/`. Closed #583/#494/#509/#587/#561.
- **M5 was done by an UNRECORDED iteration 148** that died between opening PR #600 and merging it —
  green, mergeable, and invisible to the charter. Found by the Gate-2 already-landed PR search.
- **Next**: **`m-net-effect-proxy-boundary`** (D5 Option B) — now unblocked, NEW-DOC + quorum.
  Standing alternative: **#498 Lane B M3** (final Lane B milestone → then the release ask to Mark).
- **Loops**: v1 90min · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `claude:claude-fable-5` → next `codex:gpt-5.6-sol`

## Parked on Mark
- **Nothing blocking.** D5 answered; iter-141 carve-out ratified.
- Low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op briefing

## Recently settled (don't re-ask)
- **D5 = Option A**; Option B queued as `m-net-effect-proxy-boundary`, gated on AC10(d), which exists
- **#532 closed as SUPERSEDED** · **standing fast-forward** ratified (held a 13th iteration)

## Known-deferred (measured, not forgotten)
- **Option B's scope is 7 transports across 4 files, not 6 across 3** — the 7th is
  `internal/executor/managed_agents/client.go:141`, OUTSIDE `internal/effects`. Re-measured iter-149.
- **`go mod download all` writes to the TRACKED go.sum** — prefetch must precede binary building or
  the staleness detector silently skips every binary-gated test. Cost iter-147 a red CI run.
- **`-shellcheck='-e SC2086'` DISABLES shellcheck** (flag takes an executable path); 5 pre-existing
  findings keep `actionlint → rc=0` unpassable at base. **`#598`** pid-file race: 0/12 on a stress arm.

## Quota posture (week of 2026-08-03) · Bookkeeping
- **Metered**: iter-149 spent **$0.00** of the $5/iteration ceiling (controller-only; no spawns)
- Issue: `~/.ailang/state/mission-gh-issue` (rotates Mondays) — Mark's comments there = directives
- Full state: `v1-mission.md` (charter) · `v1-mission-log.md` (history) · this file = snapshot only
