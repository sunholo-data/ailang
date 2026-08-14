# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-14 ~06:40 local (iteration 197)

## Now
- **v0.33.1** · `dev` @ `20d538a43` — merge of PR #705. Gate 3b GREEN (SHA-addressed **21** checks,
  `pending=0`, **4/4** REQUIRED, `state=CLEAN`); platform legs **named** (`test-windows` + 3 Builds).
- **`#691` LANDED**: `exit()` in an embedded module no longer panics the **host** — `#607`'s defect
  one layer down. Evaluator sonnet **96/100**, zero blocking. `metered=$0.00`.
- **Contract decided by the controller, flagged for Mark** (`D-17` below): `exit(N)` → typed
  `*embed.ExitError{Code: N}`; **`exit(0)` is an error too, not nil** — the CLI batch path diverges
  deliberately because it owns a process and embed does not. `runtime.CallEntrypoint` stays
  panic-based **on purpose** (the CLI needs the sentinel for `os.Exit`); 4 sites repo-wide.
- **Inverse arm**: both guards removed + the new tests deleted → `internal/embed` rc=0, **60 PASS**.
  The defect had shipped entirely undetected.

## Next
1. **`#692`** — batch mode silently drops `Debug` output (`flushDebugOutput` never runs per item).
2. **`#706`** *(new, from the iteration-197 judge)* — no host special-cases `*embed.ExitError`, so
   `exit(0)` in a serve-api route returns HTTP 500. **Not a regression** (it used to crash the
   process); needs a one-word contract call.
3. **`#703`** — `govulncheck-filter` drops module-level findings; a vacuous green on a security gate.
4. `#610` infra-gated. `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin (`~/.ailang-driver-pin/v1`, == `origin/dev`). Routing:
  controller **opus** · evaluator **sonnet** (generator≠judge); other lanes idle on direct-fix picks.
- ⚠ **Skill drift widening**: the running skill resolves to the MAIN checkout, now **9 behind**
  origin (7 last iteration). Every Gate-5 edit lands via worktree and never reaches the executing
  copy. Blocked on `D-16`.

## Parked on Mark (all on issue #635)
- **`D-17` (new)** — ratify or overturn the `#691` contract: should `exit(0)` from an embedded call
  be an `ExitError` (as shipped) or a nil error? Answering also settles `#706`.
- **`D-16`** — may I `git merge --ff-only` the main checkout when 0-ahead and the dirty files
  provably don't collide? Owed 3 iterations; the drift grows monotonically. **yes/no**
- **`D-15`** — `#698` part 1: should `--remote` reach `view` or `eval`? (recommend `view`)
- **`D-1`–`D-14`** — unchanged, see charter.
