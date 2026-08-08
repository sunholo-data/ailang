# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-08 ~12:15 local (iteration 167)

## Now
- **v0.33.0** · `origin/dev` `b6ef01708` · CI green. ⚠ changelog has a `v0.33.1` section, **no tag**.
  ⚠ SonarCloud red on `dev` — standing, non-required, inherited (`#615`).
- ✅ **`D-5` root cause found + fixed.** `claude -p` reaps still-running **background** tasks 600 s
  after the controller's turn ends and exits **rc=0**, so the driver logs `iteration complete
  (rc=0)` and *neither* watchdog fires. Attribution exact: 2 hits in V1's log = iter-159 and
  iter-167 attempt 1; 2 in World's = its only 2 orphans in 67 iterations. Zero false positives.
  **Fix 1** driver `CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS=0` (live). **Fix 2** skill Standing rule 7,
  *every wait is ACTIVE*. From `mission-world` iter-65, corroborated first-party before adoption.
  PR `#634` → `746983299`, Gate 3b GREEN (20 checks, 0 fail, 4/4 REQUIRED, `CLEAN`).
  ⚠ Explains **2** dead slots, not all of `D-5` — iter-165's death had a different signature.
- ✅ **Iter-167 attempt 1's orphaned work recovered, not redone**: 6 verification rows on Lane B1's
  doc, adopted only after re-deriving the 3 load-bearing ones. **B1's scope SHRINKS** — `list[T]`
  already ships (`runner.go:615`, `a81d66983`), so the doc as written would have re-derived it.
- ✅ Local `dev` **0 ahead / 0 behind** origin, first time since 08-03. Gate 4 may write in place.
- **0** open `[nightly-eval]` alarms. ⚠ `#559` at **66** comments — rotation due next fire.

## In flight
- **`#624`** top-level `forall` never evaluates — does **not** block B1 (V36).
- **`#613`** proxy M1, DRAFT *DO-NOT-MERGE*, on `D-1`. **`#604`**/`#614` on `D-2`.
- **`#633`** (World SM.C, FYI): registry vendor-namespace auth deferred — unowned, non-blocking.

## Next
**Lane B1** — scope-corrected, still queue head, **not routed this iteration by design**: two slots
died the moment a background planner was spawned. Next fire is the first with both fixes live and
must route it *without ending the turn*. Then `D-2` → `#604` · `D-1` → `#613` · `#624`.

## Loop + routing
Controller **opus** · designer **rotation** (next `claude:claude-fable-5`) · planner **opus** (codex
bucket dry) · executor **`pi:deepseek-v4-flash-0731`** · evaluator **sonnet**. Metered **$0.00**.

## PARKED ON MARK — three live asks, all one word
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes.
  **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-3`** (iter-162): uncommitted hook-timeout fixes in the shared checkout, absent from origin.
  **(A)** commit · **(B)** leave · **(C)** revert.
- ~~`D-4`~~ **RESOLVED** (pushed as `b67d415cd`). ~~`D-5`~~ **fixed above**.

Full record: charter `## STATUS … ITERATION 167` + `v1-mission-log.md` entry 170.
