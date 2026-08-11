# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-12 ~01:45 local (iteration 180)

## Now
- **v0.33.0** · `origin/dev` `817bb0274` — **`#616` doc REVISED (1,028 lines) and PARKED
  `needs-human-review` at quorum round 2** → new **`D-10`**. Metered **$0.1365** of $5.
- 🔴 **I ran R1's own probe (rule 3f) and it REFUTED the doc's architecture.** Occurrence type-info
  is **present, occurrence-shaped and UNINSTANTIATED**: `EffectRow RAW={labels=[] tail=ρ3}`.
- 💡 **The correct info lives in the ARGUMENT, not the result** — arm l's two call sites carry
  `param[0]` rows `{}` vs `{IO}` while **both** result rows are unsolved (`ρ8`/`ρ9`). **So `e` in the
  parameter and `e` in the return are NOT the same variable** (control: the same shape with a
  *concrete* row resolves fine, rc=0). Refutes iter-179's "the type layer is already correct".
- 🔴 **FIX SITE MOVED A 4th TIME, and R1's replacement is refuted too.** "One metavar per row-var
  name, reused everywhere" **already exists** (`types_v2.go:533-561` keys by NAME; `:383-401` applies
  it to Params+EffectRow+Return). Real gap: the App mints an **independent** `freshEffectRow()`
  (`inference.go:203-215`) and row unification never ties it. Both A2 and A3 aim at the wrong layer.
- ✅ **R2 confirmed at a line**: `UnionEffectRows` body has **0** `Tail` (control 3), returns
  `Tail: nil` — `effects.go:606-616`.
- ⚠ **`ailang check` CACHES by content and only hides the arms that PASS** (0 probe lines vs 10 on
  the failing control; one newline → 25) — and the soundness arms are the passing ones.
- ⚠ **Controller restored the no-regression gate**: the revision cut 14 ACs → 10, all mechanism pins,
  while over-rejection is this design's dominant risk. AC11/AC12 added → **12** ACs.

## In flight / queued
- **`#616`** blocked on `D-10`; **`#619`** blocked on `D-9` → next unblocked item is **`#617`**.
- **`#618` rollout** (cp plists → `launchctl load` → *then* `unsetenv`) — human-sequenced, `D-8`.
- **#636** `[world-DEMAND]` · **#613** `D-1` · **#604**/`#614` `D-2` · **#649** · **#651** · **#654**.

## Loop + routing
Controller **opus** · designer **codex:gpt-5.6-sol** (rotation → codex; ⚠ provider-collides with
reviewer R1, FLAGGED) · planner/executor/evaluator **NOT fired** (parked at the quorum gate).
⚠ Local `dev` **1 ahead / 2 behind**; reconcile correctly REFUSED (the ahead-commit is a genuine
unpushed sibling motoko doc) → Gate-4 writes via a worktree off `origin/dev`. ⚠ `gh run list
--limit 1` returned a **six-week-old** run once; the SHA-addressed read settled it (`checks=15`,
0 not-green). Gate 5: **no skill edit** (both frictions instance 1; World's iter-73 rule-3j
proposal is corroborated-pending).

## PARKED ON MARK — #635
- **`D-10`**: `#616`'s fix site is now `internal/types` row unification at the App constraint.
  **(A)** third revision, accept the widened surface · **(B)** hold `#616`.
- **`D-9`**: `#619` quorum reviews a 5-item umbrella while only **W8** routes. **(A)** split · **(B)** hold.
- **`D-1`** `#613` · **`D-2`** `#604`/`#614` · **`D-7`** codex **2/2** → (B) de facto · **`D-8`** `#618` rig.
