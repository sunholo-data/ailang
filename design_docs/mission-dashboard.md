# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-08 ~12:10 local (iteration 167)

## Now
- **Release** v0.33.0 · `origin/dev` `b67d415cd` · CI green. ⚠ changelog has a `v0.33.1` section
  with **no tag** — written, not released. ⚠ SonarCloud red on `dev` (standing, non-required, `#615`).
- ✅ **`D-5` root cause found and fixed.** `claude -p` reaps still-running **background** tasks 600 s
  after the controller's turn ends and exits **rc=0** — so the driver logs `iteration complete
  (rc=0)` and *neither* watchdog fires. Attribution exact: 2 hits in V1's log = iter-159 and
  iter-167 attempt 1; 2 in World's = its only 2 orphans in 67 iterations. Zero false positives.
  **Fix 1** driver `CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS=0` (live). **Fix 2** skill Standing rule 7,
  *every wait is ACTIVE*. Found by `mission-world` iter-65, corroborated first-party before adoption.
  PR `#634` → squash `746983299`, Gate 3b GREEN (20 checks, 0 failures, 4/4 REQUIRED, `CLEAN`).
  ⚠ Explains **2** dead slots, not all of `D-5` — iter-165's death had a different signature.
- ✅ **Iter-167 attempt 1's orphaned work recovered, not redone**: 6 verification rows on Lane B1's
  doc, adopted only after re-deriving the 3 load-bearing ones. **B1's scope SHRINKS** — `list[T]`
  already ships (`runner.go:615`, from `a81d66983`), so the doc as written would have re-derived it.
- ✅ Local `dev` **0 ahead / 0 behind** origin, first time since 08-03 (a concurrent session rebased
  onto `746983299` and pushed). Gate 4 may write the charter **in place** next iteration.
- **0** open `[nightly-eval]` alarms. ⚠ `#559` at **66** comments (<80) — rotation due next fire.

## In flight
- **`#624`** top-level `forall` never evaluates — does **not** block B1 (V36).
- **`#613`** proxy-boundary M1, DRAFT *DO-NOT-MERGE*, held on `D-1`. **`#604`** + `#614` held on `D-2`.
- **`#633`** (World SM.C, FYI): registry vendor-namespace auth deferred — unowned here, non-blocking.

## Next
**Lane B1** — scope-corrected, still the queue head, **not routed this iteration by design** (two
slots died the moment a background planner was spawned; the next fire is the first with both fixes
live, and must route it *without ending the turn*). Then `D-2` → `#604` · `D-1` → `#613` · `#624`.

## Loop + routing
Controller **opus** · designer **rotation** (pointer unchanged: next `claude:claude-fable-5`) ·
planner **opus** (codex bucket dry — probe hit its usage limit again) · executor
**`pi:deepseek-v4-flash-0731`** · evaluator **sonnet**. Iteration 167 metered **$0.00**.

## PARKED ON MARK — three live asks, all one word
- **`D-1`** (iter-150): proxy-boundary drops target-IP SSRF pinning on **proxied** routes.
  **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** ship top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-3`** (iter-162): uncommitted hook-timeout fixes live in the shared checkout, absent from
  origin. **(A)** commit · **(B)** leave · **(C)** revert.
- ~~`D-4`~~ **RESOLVED** 12:08 (pushed as `b67d415cd`, patch-id identical). ~~`D-5`~~ **fixed above**.

Full record: charter `## STATUS … ITERATION 167` + `v1-mission-log.md` entry 170.
