# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~23:30 local (iteration 179)

## Now
- **v0.33.0** · `origin/dev` `0a84f5377` — **`#616` RECLASSIFIED P0, design doc LANDED** (PR `#657`,
  docs-only, 845 lines). Quorum r1 **BLOCKED**. Metered **$0.1103** of $5.
- 🔴 **Not a DX bug — a STATIC EFFECT-SOUNDNESS HOLE.** A function with **no** effect annotation
  calling a row-poly function **TWICE** passes `check` rc=0 and **executes IO**; a **wrong** row
  (`! {FS}` over `{IO}`) also passes and runs.
- ✅ **Severity bounded by measurement**: with no `--caps IO` the run is refused — the capability
  layer **backstops**. This defeats capability **planning from signatures** (the reporter's
  MCP-embedder case), not enforcement.
- 💡 **Framing wrong twice.** `e` is a row **TAIL**, not a phantom concrete effect: `Required`=[]
  **and** `Declared`=[] while the check fails — same fact explains the blank `Missing effects:`.
- ✅ **"Reject lowercase (fastest)" REFUTED** — **13** row-var signatures ship in `std`. Fix site is
  the **effect checker only**. **No new D-item — direction settled by data.**
- ⚠ Both blocking objections **measured, not forwarded** (rule 3f), **both CONFIRMED**: gemini's
  laundering reproduces *inside* the row-poly fn (Phase 2 can't reach it); gpt5 is right that the
  App branch prefers `declaredEffects` and never consults `typeInfo`.

## In flight / queued
- **`#616` revision** + **ONE re-quorum** — **no human input needed**.
- **`#618` rollout** (cp plists → `launchctl load` → *then* `unsetenv`) — human-sequenced, `D-8`.
- `D-9` gates **`#619`**, then **`#617`**. **#636** `[world-DEMAND]` · **#613** `D-1` · **#604**/
  `#614` `D-2` · **#649** local-model gap · **#651** quorum zero-signal · **#654**.
- **Next**: Iteration 180 revises `#616`'s doc and re-quorums — both objection measurements already banked.

## Loop + routing
Controller **opus** · designer **claude-fable-5** (rotation → claude) · planner/executor/evaluator
**NOT fired** (parked at quorum). Gate 3b GREEN: 4/4 required, `checks=20`, 0 not-green — required
set grew **2→3→4** (late-registration trap, 4th time). ⚠ Local `dev` was **1 ahead** (sibling
motoko doc, unpushed) → ff-authorization did NOT apply; Gate-4 writes went via a worktree off
`origin/dev`. Gate 5: **no skill edit** (both frictions instance 1, already covered).

## PARKED ON MARK — #635
- **`D-9`**: `#619`'s quorum reviews a 5-item umbrella while only **W8** routes. **(A)** split · **(B)** hold.
- **`D-1`** `#613` · **`D-2`** `#604`/`#614` · **`D-7`** codex **2/2** → (B) de facto · **`D-8`** `#618` rig rollout.
