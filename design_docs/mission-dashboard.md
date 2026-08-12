# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-12 ~16:35 local (iteration 185)

## Now
- **v0.33.0** · merge `ebbc5a749` — **`#617` M3 LANDED** (PR #675). Gate 3b GREEN: SHA-addressed
  `checks=21`, `pending=0`, **zero NOT-GREEN**, 4/4 REQUIRED from real `pull_request` events.
  Evaluator **sonnet PASS 91/100 r1, zero blocking**. Metered **$0.00** (quota + codex buckets).
- 🟢 **The trap is now taught where users meet it**: prompt **v0.16.6** active in *both* trees,
  LIMITATIONS ×2, footgun row 22, changelog. Verified **behaviourally** — the built binary's
  `prompt --source=embedded` really serves it (control: v0.16.5's `toInts` intact). That check is
  the point: `#617` exists *because* a v0.10.0 fix shipped unreachable for 5 months.
- 🟢 **This milestone's guard is a real gate** (unlike M2's pins): corrupting the recorded prompt
  hash reds `TestAILANGPromptLoading` + `TestPromptDisambiguation` for the right *mechanism*.

## 🔴 The slot before mine died silently — 4th instance
Iteration **184** spawned the codex executor, announced a wait, and logged `iteration complete
(rc=0)` **6 minutes later**: 333 KB executor log, clean worktree, **zero** records (`ITERATION 184`
greps 0 in charter *and* log; control `183`=1/1). Standing rule 7, instances **159·167·176·184**.
It ran with `bg-wait-ceiling=0ms`, so the rule's own grep-tell is blind — as iter-176 predicted.
**Structural, not a wording gap**: the codex lane *mandates* a background spawn (30-min cap > the
10-min foreground limit), so every executor iteration is one lapse from this. → **`D-11`**.

## In flight / queued
- **`#617` M4** (the `LIST_TAKE_AFTER_FLATMAP` note — **cut line**; site resolved to
  `result.Warnings`) next. **`#616`** `D-10` · **`#619`** `D-9` · **`#618`** `D-8` · **#636**
  `[world-DEMAND]` · **#613** `D-1` · **#604**/`#614` `D-2` · **#649** · **#651** · **#654**.

## Loop + routing
Controller **opus** · executor **codex `gpt-5.6-sol`** (probe rc=0; no designer/planner — doc
quorum-resolved, plan existed) · evaluator **sonnet** — generator≠judge held. ⚠ **The RUNNING skill
was NOT origin's**: `~/.claude/skills/…` symlinks to the *main checkout* (188,002 B vs origin's
190,937 B), missing `e96cf210d`; delta read, origin's rules followed. Writes via worktrees.

## PARKED ON MARK — #635
- **`D-11`** *(new)* slot-death guard: should a driver exit `rc=0` when elapsed ≪ claimed work?
- **`D-10`** `#616` · **`D-9`** `#619` · **`D-1`** `#613` · **`D-2`** `#604`/`#614` · **`D-7`**
  codex 3/3 → (B) de facto · **`D-8`** `#618`.
