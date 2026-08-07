# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-07 ~21:15 local (iteration 162)

## Now
- **Latest release**: **v0.33.0** · `origin/dev` `bd5f74362`. ⚠ the changelog still carries a
  `## [v0.33.1] - 2026-08-06` section with **no `v0.33.1` tag** — written, not released.
- **`#535`'s mechanism is dead.** M2B landed (PR `#620`): all three property-RNG sites derive their
  seed, `newRNG`'s wall clock is gone. Measured with a negative control on
  `list_recursive_verify.ail` — **5 distinct output hashes / 5 runs before, 1 after**.
- ⚠ **A test that was supposed to guard the sweep could not.** §5.6's S11 observed a value stamped
  *alongside* the RNG rather than *by* it, so a constant-seed mutant at two of the three sites left
  the suite **green**. Closed by S11b; the forall site still has **no** stream observable (M3 owes
  it a CLI arm). This is now skill rule **3i**.
- `dev` CI green on `bd5f74362`. ⚠ **SonarCloud red on `dev`** (standing, 4+ consecutive analysed
  commits, non-required, `#615`) — but **`success` on `#620`**, so this diff moved coverage up.
- Nightly `[nightly-eval]` open alarms **0** (control: 52 in `--state all`).
- ⚠ **A concurrent interactive session is live in the shared checkout** — it committed `de50f203a`
  to *local* `dev` (unpushed) mid-iteration. Nothing is committed in the main tree by the loop.

## In flight
- **`m-property-seed-determinism`**: M1 ✅ · M2A ✅ · **M2B ✅** (`bd5f74362`, evaluator 91/100).
  **M2C is the resume point** — §5.5: `--seed`/`--random-seed`, both `cmd/ailang/test.go`
  aggregates, JSON+human reporting, S15–S16, plus one inherited task (replace the swallowed
  `filepath.Abs` fallback, CLAUDE.md §2). Then M3 closes `#535` and unblocks Lane B1.
- **`#613`** proxy-boundary M1 — DRAFT *DO-NOT-MERGE*, implemented and held. Blocked on **`D-1`**.
- **`#604`** named-test vacuous pass — design written, **PARKED** on **`D-2`**. `#614` open.

## Next
M2C → M3 (closes `#535`) → Lane B1 · `D-2` → `#604` · `D-1` → `#613` · swept-issue batch · `#615`.

## Loop + routing
Controller **opus** · designer **rotation** (pointer unchanged: next `claude:claude-fable-5`)
· planner **opus** (env pin) · executor **`pi:deepseek-v4-flash-0731`** (codex bucket dry)
· evaluator **sonnet**. Iteration 162 metered **$0.0373** of the $5 ceiling.

## PARKED ON MARK — three asks, all one word
- **`D-1`** (iter-150): proxy-boundary drops target-IP SSRF pinning on **proxied** routes.
  **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** ship top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-3`** (iter-162, NEW): uncommitted hook-timeout fixes sit in the shared checkout
  (`.claude/settings.json` + `scripts/hooks/session_start.sh`, bounding `ailang cache search` at
  3 s so a busy GPU cannot stall SessionStart). Verified working; **live on the rig, absent from
  origin**. Yours or a dead session's? **(A)** commit them · **(B)** leave them · **(C)** revert.

Full record: charter `## STATUS … ITERATION 162` + `v1-mission-log.md` entry 166.
