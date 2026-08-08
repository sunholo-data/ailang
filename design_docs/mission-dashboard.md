# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-08 ~11:30 local (iteration 167)

## Now
- **Latest release**: **v0.33.0** · `origin/dev` `306633d83`. ⚠ the changelog still carries a
  `## [v0.33.1] - 2026-08-06` section with **no `v0.33.1` tag** — written, not released.
- ✅ **`D-5` HAS A ROOT CAUSE, MEASURED, AND A TWO-PART FIX.** The harness reaps still-running
  **background** tasks 600 s after the controller's turn ends and exits **rc=0**, so the driver
  logs `iteration complete (rc=0)` and **neither watchdog fires**. Attribution is exact:
  `grep -c 'Background tasks still running after 600s'` = **2** in V1's driver log = the
  2026-08-07 12:26 fire (**iter-159**) and the 2026-08-08 09:09 fire (**iter-167 attempt 1**);
  **2** in World's log = its only 2 orphans in 67 iterations. Zero misses, zero false positives.
  Found by `mission-world` iter-65; corroborated first-party in V1 before adoption.
  **Fix 1** driver `export CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS=0` (`mission-control.sh:470`,
  live in the main checkout now). **Fix 2** skill Standing rule **7 — every wait is ACTIVE**.
- ✅ **Iteration 167 attempt 1's orphaned work was recovered, not redone**: 6 verification rows
  (V31–V36) on the queue head's design doc, adopted only after re-deriving the 3 load-bearing
  ones. **B1's scope SHRINKS** — `list[T]` already ships (`runner.go:615`, from `a81d66983`), so
  it is not B1 work; routing the doc as written would have re-derived an existing generator.
- `dev` CI green (`checks=6`, zero NOT-GREEN on `306633d83`). ⚠ **SonarCloud red on `dev`**
  (standing, non-required, `#615`; *absent* — not failing — on the dependabot commits).
- **0** open `[nightly-eval]` alarms (control 52 in `--state all`).
- ⚠ Local `dev` **1 ahead / 11 behind** origin — the sibling's unpushed `de50f203a`. See `D-4`.
- ⚠ Bookkeeping issue **`#559` at 65 comments** (<80) — rotation due next fire or Monday 07:00.

## In flight
- **`#624`**: top-level `forall` properties never evaluate. Does **not** block B1 (V36).
- **`#613`** proxy-boundary M1 — DRAFT *DO-NOT-MERGE*, held on **`D-1`**.
- **`#604`** named-test vacuous pass — design written, **PARKED** on **`D-2`**. `#614` open.
- **`#633`** (World SM.C, FYI): registry vendor-namespace auth deferred; one shared key can
  permanently consume any vendor's name. Non-blocking for World; needs an owner here.

## Next
**Lane B1** — scope-corrected, still the queue head, **not routed this iteration by design** (two
consecutive slots died at the moment a background planner was spawned; the next fire is the first
to run with both fixes live). Then: `D-2` → `#604` · `D-1` → `#613` · `#624` · swept-issue batch.

## Loop + routing
Controller **opus** · designer **rotation** (pointer unchanged: next `claude:claude-fable-5`)
· planner **opus** (codex bucket dry — probe hit its usage limit again this fire)
· executor **`pi:deepseek-v4-flash-0731`** · evaluator **sonnet**. Iteration 167 metered **$0.00**
(no heavy role spawned).

## PARKED ON MARK — five asks, all one word
- **`D-1`** (iter-150): proxy-boundary drops target-IP SSRF pinning on **proxied** routes.
  **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** ship top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-3`** (iter-162): uncommitted hook-timeout fixes live in the shared checkout, absent from
  origin. **(A)** commit · **(B)** leave · **(C)** revert.
- **`D-4`** (iter-164): `de50f203a` — a real ollama GPU-cap fix — has sat **unpushed** on local
  `dev` for four iterations. **(A)** loop may publish it · **(B)** it is yours, leave it.
- **`D-5`** (iter-164, **RESOLVED-PENDING-PROOF** iter-167): root cause found and both fixes
  landed. Nothing to decide — the next two fires prove or refute it on their own. No ask.

Full record: charter `## STATUS … ITERATION 167` + `v1-mission-log.md` entry 170.
