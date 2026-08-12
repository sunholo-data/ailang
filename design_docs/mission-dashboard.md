# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-12 ~08:20 local (iteration 182)

## Now
- **v0.33.0** · `origin/dev` `aec905da2` — **`#617` sprint plan + M1 LANDED** (PR #661 squashed,
  4 commits). Metered **$0.00**: plan/eval on quota buckets, executor on the codex bucket.
- 🟢 **M1: the fused bounded builtins now accept an annotated `int`.** `_list_takeFlatMap` /
  `_list_takeMap` registered `TCon{"Int"}` against the surface `int` since v0.10.0 — bare literals
  are polymorphic so the shipped tests always passed, while `n: int` (the one shape a stdlib export
  needs) never worked. Both sites → `T.Int()`, golden regenerated (exactly lines 138–139).
- 🔴 **The doc's teaching target was a FROZEN prompt** (`v0.16.2`; active is `v0.16.5` in *both*
  trees, content-hashed, eval-pinned, **no sync automation**). AC-5 as written would have left the
  shipped prompt at zero mentions — **#617's own shipped-but-unreachable failure, one layer up.**
- 💡 **AC-3a was rewritten because it was already green at base.** The instrument a quorum reviewer
  prescribed already exists twice and passes; existing tests call `takeFlatMapImpl` **directly**
  (22 sites), bypassing the export layer this sprint adds. Reviewer's instrument right, layer wrong.

## In flight / queued
- **`#617` M2** (`std/list` exports, example, parity) next; then M3 teach, M4 note (**cut line**).
- **`#616`** blocked on `D-10`; **`#619`** blocked on `D-9`; **`#618` rollout** human-sequenced (`D-8`).
- **#636** `[world-DEMAND]` · **#613** `D-1` · **#604**/`#614` `D-2` · **#649** · **#651** · **#654**.

## Loop + routing
Controller **opus** · planner **opus** (lane `opus fail-closed:planner-lane-field-missing` —
no `Planner-Lane` field in the doc) · executor **codex `gpt-5.6-sol`** (probe rc=0, ~4 min,
uncommitted delivery as designed) · evaluator **sonnet PASS 97/100 r1, zero blocking** —
generator≠judge held (OpenAI vs Anthropic). ⚠ Local `dev` **1 ahead / 4 behind**; reconcile
REFUSED (obligations 1 *and* 2 both fail) → all writes via worktrees off `origin/dev`.
SonarCloud red on the **PR** (new-code duplication 4.8% → 3.9% vs ≤3%; cut by a verified-non-vacuous
helper extraction, stopped there — the residual encodes the `b` vs `list[b]` distinction under test).
It did **not** reproduce on dev: Sonar is `success` on the merge, so nothing red was carried over.

## PARKED ON MARK — #635
- **`D-10`**: `#616` fix site → `internal/types` row unification. **(A)** revise · **(B)** hold.
- **`D-9`**: `#619` umbrella of 5, only **W8** routes. **(A)** split · **(B)** hold.
- **`D-1`** `#613` · **`D-2`** `#604`/`#614` · **`D-7`** codex now **3/3** → (B) de facto · **`D-8`** `#618`.
- **No new decision item this iteration.**
