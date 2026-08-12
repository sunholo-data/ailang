# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-12 ~11:40 local (iteration 183)

## Now
- **v0.33.0** · merge `6a67bb7a7` — **`#617` M2 LANDED** (PR #668). Gate 3b GREEN: SHA-addressed
  `checks=16`, **zero NOT-GREEN**. Evaluator **sonnet PASS 95/100 r1, zero blocking**, every drill
  reproduced independently. Metered **$0.00** (quota + codex buckets).
- 🟢 **`import std/list (takeFlatMap)` / `(takeMap)` now work.** Doc comments carry the corrected
  cost model and say plainly neither bounds peak by `n` — what they bound is how many inputs `f`
  is invoked on (verified empirically: 84.8 MB peak at `n=2`).
- 🔴 **The milestone's own pins had no gate.** `grep -rn "ailang test" make/ Makefile` → **0**
  (control `.ail` in `make/` = 35): **nothing** ran **any** `.ail` suite, so AC-3a/AC-4 would have
  shipped as one-shot commands against a tree that no longer exists. Added `make test-stdlib-ail`
  + a `ci.yml` step, anti-vacuity floors on both loops.

## Two new bugs, found by drills rather than by the sprint
- **`#669`** — `ailang test` reports **FALSE FAILURES**: a stdlib export delegating to another
  same-module AILANG export cannot be `match`-destructured; same code under `ailang run` is fine.
- **`#670`** — `expected.stdout` in `examples/manifest.json` is **display-only** for all 194
  examples: corrupted one to a wrong value → `make verify-examples` still **rc=0**.

## In flight / queued
- **`#617` M3** (limitations, prompt **v0.16.6**, footgun row, changelog) next; then M4 (**cut line**).
  **`#616`** `D-10` · **`#619`** `D-9` · **`#618`** `D-8` · **#636** `[world-DEMAND]` · **#613**
  `D-1` · **#604**/`#614` `D-2` · **#649** · **#651** · **#654**.

## Loop + routing
Controller **opus** · executor **codex `gpt-5.6-sol`** (probe rc=0; no planner run — plan existed) ·
evaluator **sonnet** — generator≠judge held. ⚠ Local `dev` **1 ahead / 19 behind**; reconcile REFUSED
— obligation 1 now PASSES (patch-id dup of `99486ad02`), **obligation 2 fails** (dirty ∩ incoming =
`SKILL.md`, `mission-control.sh`; control 24) → writes via worktrees off `origin/dev`. ⚠ `origin/dev`
moved **twice** mid-iteration (motoko merging) — the per-workflow `--limit 1` read watched a
sibling's SHA; only SHA-addressed reads are trustworthy.

## PARKED ON MARK — #635
- **`D-10`** `#616` · **`D-9`** `#619` · **`D-1`** `#613` · **`D-2`** `#604`/`#614` · **`D-7`**
  codex 3/3 → (B) de facto · **`D-8`** `#618`. **No new item** — `#669`/`#670` are work, not asks.
