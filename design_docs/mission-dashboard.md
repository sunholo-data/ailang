# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-08 ~07:00 local (iteration 166)

## Now
- **Latest release**: **v0.33.0** · `origin/dev` `2ab7b3d31`. ⚠ the changelog still carries a
  `## [v0.33.1] - 2026-08-06` section with **no `v0.33.1` tag** — written, not released.
- ✅ **`#535` IS CLOSED. The `m-property-seed-determinism` sprint is COMPLETE** (M1·M2A·M2B·M2C·M3).
  PR **`#625`** → squash `2ab7b3d31`, Gate 3b green with completeness asserted (`total=20`,
  0 pending, 0 failed, all four REQUIRED contexts, `CLEAN`). Evaluator **sonnet PASS 95/100 r1**.
- ⚠ **The feature's headline deliverable had never worked.** Every failing property printed
  `replay: ailang test --seed 0 All Tests` — the aggregate *display label*, not a path. Green the
  whole time because `AC6-M2` **rebuilt** the command from `.seed` instead of **executing** the one
  the tool emits. Now skill rule **3k**.
- ⚠ **ITERATION 165 DIED — 4 of the last 7 slots** (159, 160, 163, 165). New signature: both opus
  probes timed out, the driver fell back to **Fable**, and the slot exited **rc=0 after 10.5 min**.
  It looked like zero output; it had in fact filed **`#624`** — invisible to all three traces the
  died-mid-flight rule names, because the artifact is an **issue**.
- `dev` CI green. ⚠ **SonarCloud red on `dev`** (standing, non-required, 8 consecutive commits, `#615`).
- Nightly `13/24`, **0** regressions, **0** sustained failures, **0** open `[nightly-eval]` alarms.
- ⚠ Local `dev` is **1 ahead / N behind** origin — a sibling's unpushed `de50f203a`. See `D-4`.
- ⚠ Three stale sprint worktrees (`.wt-iter117`, `.wt-iter121`, `.wt-iter159`) each carry **unmerged
  commits**; left alone rather than swept blind, but they are noise in the died-mid-flight check.

## In flight
- **`#624`** (NEW, from iter-165): top-level `forall` properties never evaluate — `empty program` on
  the simplest property, parse error when the body calls a function. The forall seed site is pinned
  by *stamp* only until this is fixed; M3's CLI arm was retracted as unachievable because of it.
- **`#613`** proxy-boundary M1 — DRAFT *DO-NOT-MERGE*, implemented and held. Blocked on **`D-1`**.
- **`#604`** named-test vacuous pass — design written, **PARKED** on **`D-2`**. `#614` open.

## Next
**Lane B1** (unblocked by `#535`) · `D-2` → `#604` · `D-1` → `#613` · `#624` · swept-issue batch · `#615`.

## Loop + routing
Controller **opus** · designer **rotation** (pointer unchanged: next `claude:claude-fable-5`)
· planner **opus** (env pin) · executor **`pi:deepseek-v4-flash-0731`** (codex bucket dry)
· evaluator **sonnet**. Iteration 166 metered **$0.2530**. pi lane **datapoint 2** — it refuted
three of the controller's own directive claims, all three adjudicated correct by command.

## PARKED ON MARK — five asks, all one word
- **`D-1`** (iter-150): proxy-boundary drops target-IP SSRF pinning on **proxied** routes.
  **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** ship top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-3`** (iter-162): uncommitted hook-timeout fixes live in the shared checkout, absent from
  origin. **(A)** commit · **(B)** leave · **(C)** revert.
- **`D-4`** (iter-164): `de50f203a` — a real ollama GPU-cap fix — has sat **unpushed** on local
  `dev` for three iterations. **(A)** loop may publish it · **(B)** it is yours, leave it.
- **`D-5`** (iter-164, UPDATED iter-166): **4 of the last 7 slots died** (159, 160, 163, 165), and
  165's death correlates with both opus probes timing out and the driver falling back to Fable.
  **(A)** investigate as a queue item · **(B)** leave it — recovery keeps catching them.

Full record: charter `## STATUS … ITERATION 166` + `v1-mission-log.md` entry 169.
