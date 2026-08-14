# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-14 ~17:15 local (iteration 200)

## Now
- **v0.33.1** · `dev` @ `1c7fa675b` — squash of PR #714. Gate 3b GREEN: **20** contexts,
  **15 pass / 5 skipped / zero failures**, `pending=0`, **4/4 REQUIRED**, `MERGEABLE CLEAN`,
  all three platform legs + `test-windows` green.
- **`#706` LANDED** — `exit(0)` in a route/A2A/MCP handler returned HTTP 500. Closes the
  `#607`→`#691`→`#706` guard-the-helper chain: one `isCleanExit` classifier plus a **separate
  branch at each of the three call sites**. Shipped per Mark's `D-17`; non-zero codes unchanged.
- Repro before routing: unit route **200**, `exit(0)` route **500**, host alive both. ⚠ The first
  attempt omitted `--caps IO` and **both** arms returned 500 for an unrelated reason — subject-only,
  that reads as a clean reproduction of a bug never exercised.
- Evaluator sonnet **90/100 PASS**, in its **own worktree** (iter-199's skill edit, first use).
  One BLOCKING find, reproduced first-party: `TestA2AExitNonzeroFails` asserted only
  `state == "failed"` — a state **every** failure reaches, incl. "exit() never ran". Neutering the
  IO grant: **5 of 6 arms died, that one passed**. Fixed `bd9984084`.
- It also refuted the controller's own commit message ("every arm would pass for the wrong reason" —
  only one did) and the positive control (`no_exit.ail` has no IO effect, so it never proved the grant).
- `metered=$0.00` — codex rode the OAuth bucket; no OpenRouter or quorum lane fired.

## Next
1. **`#703`** — `govulncheck-filter` drops module-level findings; a vacuous green on a security
   gate, P1 gate-integrity. Unowned, unblocked.
2. **`[email-parse-DEMAND]`** — `ailang install` on an already-declared dep writes a **duplicate
   TOML key** and breaks the manifest (`lock` rc=1). Reproduced first-party; two sibling helpers,
   four call sites. Half the report REFUTED: `lock` does *not* silently pass.
3. `#709` / `#649` nightly alarms triaged, correctly open. `#610` infra-gated. `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin (`~/.ailang-driver-pin/v1`). Routing: controller **opus** ·
  executor **codex gpt-5.6-sol** · evaluator **sonnet** (generator≠judge holds). Designer, planner
  and quorum **not fired** — direct-fix lane, no design doc, same basis as `#691`/`#692`/`#607`.
- **Skill drift: CLOSED.** Running skill == `origin/dev` (`cmp` silent); `D-16` applied to ff-merge
  the main checkout on measured conditions (0 ahead; dirty∩incoming empty, control firing).
- Gate 5 skill edit taken: **rule 3i**, at the ≥2-friction bar iteration 199 pre-registered.
- ⚠ My executor directive shipped an AC already red at base (`go build ./...`, `cmd/wasm`) — rule
  3e(a). codex caught it and reported rather than papering over.

## Parked on Mark (all on issue #635)
- **`D-1`–`D-14`** — unchanged, see charter. `D-15`/`D-16`/`D-17` remain discharged.
- **Nothing new is blocking.** The queue has two unowned, unblocked items.
