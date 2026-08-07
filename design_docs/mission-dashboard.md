# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-08 ~01:50 local (iteration 164)

## Now
- **Latest release**: **v0.33.0** · `origin/dev` `8d9c58780`. ⚠ the changelog still carries a
  `## [v0.33.1] - 2026-08-06` section with **no `v0.33.1` tag** — written, not released.
- ⚠ **ITERATION 163 DIED MID-FLIGHT — the third dead slot in six iterations** (159, 160, 163). It
  did the whole inner loop for M2C, opened PR **`#621`**, watched it go green, and wrote **zero**
  charter rows and **zero** log entries. Found by the died-mid-flight check (`grep -ci "ITERATION
  163"` = **0** with the control at **1**). Its work was verified and landed by iteration 164.
- **M2C is LANDED** (`#621` → `8d9c58780`): `--seed`/`--random-seed`, `SetSeedMetadata` at BOTH
  `cmd/ailang/test.go` aggregates, replayable JSON. Its S17 pin re-verified first-party — mutants
  at **both** call sites LANDED + BUILD and red S17 independently.
- ⚠ **A refusal branch shipped with no gate.** M2C's `--seed`/`--random-seed` mutual exclusion was
  covered only by a one-shot shell AC in the sprint plan (in **no** make target, **no** CI job, and
  **no** `*_test.go`): the whole rest of `cmd/ailang` is **rc=0** with the guard neutered. Closed by
  S18 this iteration. Now skill rule **3j** (proposed by `mission-world` iter-63, corroborated here).
- `dev` CI green. ⚠ **SonarCloud red on `dev`** (standing, non-required, `#615`).
- Nightly `[nightly-eval]` open alarms **0** (control: 52 in `--state all`).
- ⚠ Local `dev` is **1 ahead / N behind** origin — a sibling session's unpushed `de50f203a`
  (ollama GPU-cap fix). Not published by the loop; see `D-3`/`D-4`.

## In flight
- **`m-property-seed-determinism`**: M1 ✅ · M2A ✅ · M2B ✅ · **M2C ✅** (`8d9c58780`).
  **M3 is the resume point** — closes `#535`, owes the forall path a CLI-level seed arm, unblocks
  Lane B1.
- **`#613`** proxy-boundary M1 — DRAFT *DO-NOT-MERGE*, implemented and held. Blocked on **`D-1`**.
- **`#604`** named-test vacuous pass — design written, **PARKED** on **`D-2`**. `#614` open.

## Next
M3 (closes `#535`) → Lane B1 · `D-2` → `#604` · `D-1` → `#613` · swept-issue batch · `#615`.

## Loop + routing
Controller **opus** · designer **rotation** (pointer unchanged: next `claude:claude-fable-5`)
· planner **opus** (env pin) · executor **`pi:deepseek-v4-flash-0731`** (codex bucket dry)
· evaluator **sonnet**. Iteration 163 metered **$0.0732**; iteration 164 **$0.00** (no sub-agent).

## PARKED ON MARK — five asks, all one word
- **`D-1`** (iter-150): proxy-boundary drops target-IP SSRF pinning on **proxied** routes.
  **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** ship top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-3`** (iter-162): uncommitted hook-timeout fixes live in the shared checkout, absent from
  origin. **(A)** commit · **(B)** leave · **(C)** revert.
- **`D-4`** (iter-164, NEW): `de50f203a` — a real ollama GPU-cap fix — has sat **unpushed** on local
  `dev` for two iterations. **(A)** loop may publish it · **(B)** it is yours, leave it.

- **`D-5`** (iter-164, NEW): **3 of the last 6 slots died mid-flight** (159, 160, 163), always at
  the last step. The loop sees the frequency, never the cause. **(A)** investigate as a queue item ·
  **(B)** leave it — the died-mid-flight recovery check is catching them.

Full record: charter `## STATUS … ITERATION 164` + `v1-mission-log.md` entries 167–168.
