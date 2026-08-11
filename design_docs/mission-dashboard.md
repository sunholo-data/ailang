# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~18:15 local (iteration 177)

## Now
- **v0.33.0** · `origin/dev` `2cab77966` — **`#517` Lane B1 is COMPLETE** (PR
  [#655](https://github.com/sunholo-data/ailang/pull/655)). M1–M4 + M6 landed; **M5 descoped** per
  the plan's own §6 (derived shrinkers have no downstream observable, F-5) and filed as a follow-up.
- 📈 **The payoff, measured twice independently** — controller before routing, then the evaluator
  from a binary rebuilt at the pre-B1 base `22ba8626d`: **vacuous skips 111 → 24**, i.e. **87
  previously-never-executed contract properties across 15 examples now actually run**. 8 files flip
  rc 1 → 0; all 5 false-red guards hold rc 0.
- 🔴 **F-1 confirmed in the field: B1 does NOT fix the prompt-injection safety demos.** The surviving
  24 skips are imported + refined types (`inbox_injection_v2` 10, `inbox_v2_app` 10,
  `cross_module_types` 4) — B2 by design. **The design doc's original Success metric was wrong.**
- 💡 **Closing finding: the corpus's "Z3 catches this bug" demos had never run.** All 3 newly-failing
  properties are **(a) deliberate** (`record_verify`, `insurance`, `scoring` — all marked
  INTENTIONALLY BROKEN); zero were example bugs or B1 defects. They now do what they always claimed.
- 🟡 **Two vacuity defects caught inside the milestone about vacuity.** The executor shipped
  `ensures { result == result }` for the new example's tuple property — a contract that *cannot
  fail* (strengthened; both arms measured). And its header claimed a clean `ailang verify` when Z3
  **cannot encode tuple patterns at all** — reproduced with the tautological form too, so it
  predated the fix. Now stated honestly; the two checkers genuinely disagree for this shape.
- ⚠️ **The plan's §1.6 bounded-wait recipe is broken when piped** (watchdog inherits stdout → holds
  the pipe for the full limit). Cost 13 min and nearly produced a false "these files are slow now"
  finding — the direction I expected. Redirect to a file. Fixed in the plan.

## In flight / queued
- **`#618` rollout** (cp plists → `launchctl load` → *then* `unsetenv`) — human-sequenced, `D-8`.
- Batch remainder: `#619` → `#616` → `#617`. **#636** `[world-DEMAND]` · **#613** on `D-1` ·
  **#604**/`#614` on `D-2` · **#649** local-model capability gap · **#651** quorum zero-signal
  guard · **#654** `validate_manifest.go` prints a total it never asserts.

## Next
**Lane B1 is done — no successor milestone.** Iteration 178 picks the queue's next unblocked item
fresh. `#517` Lane B2 stays deferred (blocked on a deterministic evaluator fuel budget, quorum
2026-07-29).

## Loop + routing
Controller **opus** · designer/planner **not fired** (doc + plan quorum-cleared 2026-07-29; rotation
pointer unchanged, next `codex:gpt-5.6-sol`) · executor **codex:gpt-5.6-sol**, 2nd consecutive clean
fire · evaluator **sonnet** 94/100, zero blocking. Metered **$0.00** against the $5 ceiling.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. (A) as-written ·
  (B) narrow to literal-IPs · (C) rethink. — `#613` blocked on this.
- **`D-2`**: `#604`/`#614`.
- **`D-7`**: pi executor — codex is now **2/2**, so **(B) flip to codex** is the de-facto state.
- **`D-8`**: authorise the `#618` rig rollout (install flag-on plists, *then* clear the launchd
  global) — or hold the rig on the stopgap until attended.
