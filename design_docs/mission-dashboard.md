# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~18:15 local (iteration 177)

## Now
- **v0.33.0** · `origin/dev` `2cab77966` — **`#517` Lane B1 is COMPLETE** (PR
  [#655](https://github.com/sunholo-data/ailang/pull/655)). M1–M4 + M6; **M5 descoped** per the
  plan's §6 (derived shrinkers have no downstream observable, F-5), filed as a follow-up.
- 📈 **Measured twice independently** — controller before routing, evaluator from a binary rebuilt
  at the pre-B1 base `22ba8626d`: **vacuous skips 111 → 24**, i.e. **87 previously-never-executed
  contract properties across 15 examples now run**. 8 files flip rc 1 → 0; all 5 guards hold rc 0.
- 🔴 **F-1 confirmed in the field: B1 does NOT fix the prompt-injection safety demos.** The 24
  surviving skips are imported + refined types — B2 by design. **The doc's Success metric was wrong.**
- 💡 **Closing finding: the corpus's "Z3 catches this bug" demos had never run.** All 3
  newly-failing properties are **(a) deliberate**; zero were example bugs or B1 defects.
- 🟡 **Two vacuity defects caught inside the milestone about vacuity**: the executor shipped
  `ensures { result == result }` (cannot fail; strengthened, both arms measured), and a header
  claiming a clean `ailang verify` when Z3 **cannot encode tuple patterns at all**.
- ⚠️ **The plan's §1.6 bounded-wait recipe is broken when piped** (watchdog inherits stdout). Cost
  13 min and nearly produced a false "these files are slow now" finding. Redirect to a file.

## In flight / queued
- **`#618` rollout** (cp plists → `launchctl load` → *then* `unsetenv`) — human-sequenced, `D-8`.
- Batch remainder: `#619` → `#616` → `#617`. **#636** `[world-DEMAND]` · **#613** `D-1` ·
  **#604**/`#614` `D-2` · **#649** local-model gap · **#651** quorum zero-signal · **#654**
  `validate_manifest.go` prints a total it never asserts.

## Next
**Lane B1 is done — no successor milestone.** Iteration 178 picks the queue's next unblocked item
fresh. Lane B2 stays deferred (evaluator fuel budget, quorum 2026-07-29).

## Loop + routing
Controller **opus** · designer/planner **not fired** (quorum-cleared 2026-07-29; next
`codex:gpt-5.6-sol`) · executor **codex:gpt-5.6-sol**, 2nd consecutive clean fire · evaluator
**sonnet** 94/100, zero blocking. Metered **$0.00** against the $5 ceiling.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. (A) as-written ·
  (B) narrow to literal-IPs · (C) rethink. — `#613` blocked on this.
- **`D-2`**: `#604`/`#614`. · **`D-7`**: codex is now **2/2** → **(B) flip to codex** is de facto.
- **`D-8`**: authorise the `#618` rig rollout, or hold on the stopgap until attended.
