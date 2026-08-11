# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-11 ~14:50 local (iteration 176)

## Now
- **v0.33.0** · `origin/dev` `03ab3e7de` — **`#517` Lane B1 M4 LANDED** (PR
  [#653](https://github.com/sunholo-data/ailang/pull/653), 4 commits). ADTs derive; recursion bounded
  by depth *and* size; `TypeApp` substitution. Corpus: `park.ail` 6 vacuous → **0** (rc 1 → 0),
  `record_adt_cycle_verify.ail` 1 → **0**; `hof_verify`/`list_verify` unchanged (B1-13 holds).
  Package **265 → 274** `--- PASS`. Gate 3b green, `checks=20` **zero** not-green.
- ✅ **Executor lane is healthy again** — `codex:gpt-5.6-sol` (the `#611` fallback chain) delivered
  the whole milestone in **23 min**, first real fire since pi's 3/3 failures. **D-7 is answered by
  events**; leaving it listed only for Mark to confirm the pin.
- 🔴 **Three defects the sprint's own gates could not see, all found after the executor's green.**
  (1) The milestone's central invariant — an underivable field in *any* constructor must refuse the
  *whole* ADT — had **zero** coverage: mutating it left the package green at 272/272. (2) The only
  end-to-end pin on `ailang test` exiting 1 for a vacuous suite built its fixture from an ADT M4 now
  derives; **CI caught it, my local sweep did not**. (3) `typeDefReferencesDecl` (indirect recursion
  through a named type) sat at **0.0%** coverage — deletable with the suite still green.
- 🟡 **`validate_manifest.go` prints a total it never asserts** — "186 modules checked" feeds no
  comparison; the only `exit(1)` is `driftCount > 0`, so a zero-enumeration passes. NEW, unfiled.

## In flight / queued
- **`#618` rollout** (cp plists → `launchctl load` → *then* `unsetenv`) — human-sequenced, `D-8`.
- **Lane B1 M5** (droppable, unit-observable only — plan §6 recommends cutting) → **M6** closeout.
- Batch remainder: `#619` → `#616` → `#617`. **#636** `[world-DEMAND]` · **#613** on `D-1` ·
  **#604**/`#614` on `D-2` · **#649** local-model capability gap (triaged, not a regression) ·
  **#651** quorum zero-signal guard.

## Next
**Lane B1 M6** (corpus triage + `shapes_verify.ail` + closeout), skipping **M5** per the plan's own
descope recommendation — no downstream observable exists for derived shrinkers (F-5).

## Loop + routing
Controller **opus** · designer/planner **not fired** (doc + plan quorum-cleared 2026-07-29; rotation
pointer unchanged, next `codex:gpt-5.6-sol`) · executor **codex:gpt-5.6-sol** · evaluator **sonnet**
— generator≠judge holds (OpenAI ≠ Anthropic). Metered **$0.00** against the $5 ceiling.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. (A) as-written ·
  (B) narrow to literal-IPs · (C) rethink. — `#613` blocked on this.
- **`D-2`**: `#604`/`#614`.
- **`D-7`**: pi executor — codex now works, so **(B) flip to codex** is the de-facto state. Confirm?
- **`D-8`**: authorise the `#618` rig rollout (install flag-on plists, *then* clear the launchd
  global) — or hold the rig on the stopgap until attended.
