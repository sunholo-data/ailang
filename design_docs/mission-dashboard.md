# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-07 ~18:20 local (iteration 161)

## Now
- **Latest release**: **v0.33.0** · HEAD `ac306084b`. ⚠ the changelog still carries a
  `## [v0.33.1] - 2026-08-06` section with **no `v0.33.1` tag** — written, not released.
- ⚠ **ITERATIONS 159 AND 160 BOTH DIED MID-FLIGHT** — each landed commits and wrote zero charter
  rows, log entries or dashboard updates. Reconstructed and credited by iteration 161 (log entries
  163/164). `#609` is recorded as *unattributed between them* rather than guessed. Two iterations
  in a row is a pattern, not an accident; cause unknown (both `pi` runs ended `stopReason: toolUse`
  mid-work, ~2h apart, so not the 6h watchdog).
- `dev` CI: **CI + Build-and-Release GREEN** on `7be6a2b8a`'s Windows fix `ac306084b` — verdict in
  the charter STATUS addendum. Nightly `[nightly-eval]` open alarms **0** (control: 52 in `all`).
- `dev == origin/dev`; running `SKILL.md` back in sync with origin (iteration 160's uncommitted
  Gate-5 edit landed as `f2b2cf468` — it had been LIVE on the rig and absent from origin).
- ⚠ **SonarCloud still red** (was 8 consecutive analysed commits at iter-158). Non-required.
  Tracked in `#615`.

## In flight
- **`m-property-seed-determinism` M2A LANDED** (`7be6a2b8a` + `ac306084b`) · evaluator 97/100.
  **M2B is the immediate resume point** — §5.4, three RNG sites + delete the wall clock. Directive
  already written at `/tmp/pi_directive_iter160_m2b.txt` (its "M2A is already committed" premise
  was false when written and is true now). Worktree `.wt-iter159`.
- **`#613`** proxy-boundary M1 — DRAFT *DO-NOT-MERGE*, implemented and held. Blocked on **`D-1`**.
- **`#604`** named-test vacuous pass — design written, **PARKED** on **`D-2`**. `#614` open.

## Next
M2B → M2C → M3 (closes `#535`, the P0 prerequisite for Lane B1) · `D-2` → `#604` · `D-1` → `#613`
· then the swept-issue batch · `#615`.

## Loop + routing
Controller **opus** · designer **rotation** (pointer unchanged: next `claude:claude-fable-5`)
· planner **opus** (env pin) · executor **`pi:deepseek-v4-flash-0731`** (codex bucket dry)
· evaluator **sonnet**. Metered **$0.032** — $0.000 spent by iteration 161; that figure is
iterations 159 (`$0.020`) and 160 (`$0.012`) **ledgered retroactively**, since neither recorded
its own spend before dying.

## PARKED ON MARK — two asks, both one word (unchanged, no reply yet)
- **`D-1`** (open since iter-150): proxy-boundary drops target-IP SSRF pinning on **proxied**
  routes. **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604`'s fix closes the top-level vacuous pass but leaves the nested one
  (`#614`). Bound measured — **27** expr node types, so an exhaustive switch with a loud
  `default:` makes "a walk silently misses a node" impossible. **(A)** ship top-level-only ·
  **(B)** widen to close nested · **(C)** make multi-expression test-body blocks an error.

Full record: charter `## STATUS … ITERATION 161` + `v1-mission-log.md` entries 163/164/165.
