# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-14 ~13:20 local (iteration 199)

## Now
- **v0.33.1** · `dev` @ `29ad1c559` — squash of PR #711. Gate 3b GREEN (SHA-addressed **21** checks,
  `pending=0`, **0** not-green, **4/4** REQUIRED, `state=CLEAN`); count climbed 17→21 during the
  poll, so `pending=0` was required rather than inferred. Windows/ubuntu/macos legs all `success`.
- **`#692` LANDED** — batch mode dropped every `Debug` log. `flushDebugOutput` had one call site, in
  the single-file branch; `executeBatchItem` never called it. Measured before any work: single-file
  emits the line **1×**, `--batch` over two inputs **0×** while reporting `2/2 succeeded`.
- Two design calls the issue didn't ask for: the flush is **deferred** (so a *failing* item's logs
  survive too, incl. the `exit()` path from #607), and it carries its **own label** — the `[i/n]`
  header lives inside `if !quiet`, so under `--quiet` a bare flush is unattributable.
- Evaluator sonnet **99/100 PASS r1, zero blocking** — it reproduced all six named targets, added
  two mutants of its own, and byte-diffed the pre-fix binary's single-file stderr against the fixed
  one. One non-blocking find: `setupEnvContext`'s `os.Exit(1)` bypasses the defer (pre-existing).
- `metered=$0.00` — codex rode the OAuth bucket; no OpenRouter or quorum lane fired.

## Next
1. **`#706`** — no host special-cases `*embed.ExitError`, so `exit(0)` in a serve-api route returns
   HTTP 500. `D-17` settled the direction: hosts branch on `Code == 0`. Unowned.
2. **`#703`** — `govulncheck-filter` drops module-level findings; a vacuous green on a security gate.
3. `#709` / `#649` — nightly alarms, both triaged, correctly left open as capability work.
   `#610` infra-gated. `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin (`~/.ailang-driver-pin/v1`). Routing: controller **opus** ·
  executor **codex gpt-5.6-sol** · evaluator **sonnet** (generator≠judge holds). Designer, planner
  and quorum **not fired** — direct-fix lane, no design doc, same basis as `#691`/`#607`.
- **Skill drift: CLOSED.** Running skill == `origin/dev` (`cmp` silent), and `D-16` was applied
  again to ff-merge the main checkout on measured conditions (0 ahead; dirty∩incoming empty with a
  firing control; dirty files byte-identical after).
- ⚠ Owed from iter-198, still open: a judge doing mutation drills needs its **own** worktree. This
  iteration's evaluator shared the controller's again — no harm, but it is luck, not design.

## Parked on Mark (all on issue #635)
- **`D-1`–`D-14`** — unchanged, see charter. `D-15`/`D-16`/`D-17` remain discharged.
- **Nothing new is blocking.** The queue has two unowned, unblocked items.
