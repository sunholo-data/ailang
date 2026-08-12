# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-12 ~05:40 local (iteration 181)

## Now
- **v0.33.0** · `origin/dev` `eabab0611` — **`#617` design doc LANDED + quorum-resolved**
  (`planned/m-take-flatmap-peak-memory.md`, 878 lines, 8 ACs). Metered **$0.2200** of $5.
- 🔴 **The fix for `#617` shipped five months ago and has been unreachable ever since.** A fused
  `takeFlatMap` merged as `d41e43894` (v0.10.0), motivated by the **same** DocParse Moby Dick
  OOM — never exported (`IMP010`), and registered `TCon{"Int"}` vs the surface `int`, so a
  literal cap checks clean while `n: int` fails. The one shape a stdlib export needs never worked.
- 🔴 **A charter row was false for 11 iterations**: "stdlib has NO `flatMap`" came from grepping
  `stdlib/` — never existed here (it's `std/`) — with a control scoped to a *different* path, so it
  fired on the pattern, not the scope, pointing `#617` at docs/lint. **Rule 3a gained `(i-d)`.**
- 💡 **All four quorum objections were MEASURED, not forwarded — all four true.** `takeFlatMap`
  does *not* bound peak by `n`; and `take(n, map(f, xs))` amplifies identically once `f` allocates
  — **0.08 s/101 MB fused vs 18.78 s/559 MB unfused** — refuting the doc's own V7 and reversing a
  reviewer's own round-1 cut. ⚠ One arm was **discarded, not banked**: a 1.49 GB reading was a Go
  stack overflow in my list *builder*, not in `takeFlatMap` — a red in the predicted direction (3d).

## In flight / queued
- **`#617`** ready to route to **sprint-planner** — next iteration's pick.
- **`#616`** blocked on `D-10`; **`#619`** blocked on `D-9`; **`#618` rollout** human-sequenced (`D-8`).
- **#636** `[world-DEMAND]` · **#613** `D-1` · **#604**/`#614` `D-2` · **#649** · **#651** · **#654**.

## Loop + routing
Controller **opus** · designer **claude:claude-fable-5** (rotation → claude; probe rc=0, 3 bounded
runs 15/8/13 min, **no provider collision** this time) · planner/executor/evaluator **not fired**.
⚠ Local `dev` **1 ahead / 3 behind**; reconcile correctly REFUSED again → Gate-4 writes via a
worktree off `origin/dev`. Gate 5: **one skill edit** (rule 3a `(i-d)`, saved in the MAIN checkout
*and* committed byte-identically); World's `git diff`-omits-untracked finding corroborated
first-party but **not adopted** — next candidate.

## PARKED ON MARK — #635
- **`D-10`**: `#616`'s fix site is now `internal/types` row unification at the App constraint.
  **(A)** third revision, accept the widened surface · **(B)** hold `#616`.
- **`D-9`**: `#619` quorum reviews a 5-item umbrella while only **W8** routes. **(A)** split · **(B)** hold.
- **`D-1`** `#613` · **`D-2`** `#604`/`#614` · **`D-7`** codex **2/2** → (B) de facto · **`D-8`** `#618` rig.
- **No new decision item this iteration.**
