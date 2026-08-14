# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-14 ~10:45 local (iteration 198)

## Now
- **v0.33.1** · `dev` @ `4942362f4` — squash of PR #710. Gate 3b GREEN (SHA-addressed **21** checks,
  `pending=0`, **4/4** REQUIRED, `state=CLEAN`); all five platform legs named and `success`.
- **`#698` part 1 LANDED** at the `view` scope you ratified as `D-15` this morning. `ailang chains`
  can now read a REMOTE observatory (`--remote` / `AILANG_CHAINS_READ`); **local stays the default**.
  The surfaces that physically cannot go remote **error** instead of silently reading local, and all
  13 `eval-*` commands refuse by name — so wanting remote eval read now leaves a dated signal.
- Evaluator sonnet **88/100 PASS**; it found a real bypass (`-remote`, one dash, lost the `D-15`
  text — Go's flag pkg treats `-x` and `--x` alike), fixed + pinned with an over-match control.
- **`D-16` used for the first time**: the main checkout was ff-merged twice on measured conditions,
  so the **skill drift that ran 7–9 commits deep for three iterations is CLOSED** and this
  iteration's Gate-5 edit is live for every mission immediately.
- `metered=$0.00` — codex rode the OAuth bucket; no OpenRouter or quorum lane fired.

## Next
1. **`#692`** — batch mode silently drops `Debug` output (`flushDebugOutput` never runs per item).
2. **`#706`** — no host special-cases `*embed.ExitError`, so `exit(0)` in a serve-api route returns
   HTTP 500. `D-17` settled the direction: hosts branch on `Code == 0`. Unowned.
3. **`#703`** — `govulncheck-filter` drops module-level findings; a vacuous green on a security gate.
4. `#709` (new) — `config_file_parser` sustained-fail on the local model; triaged NOT a regression,
   queued for capability work. `#610` infra-gated. `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin (`~/.ailang-driver-pin/v1`). Routing: controller **opus** ·
  planner **opus** (lane derived: `fail-closed:planner-lane-field-missing`) · executor
  **codex gpt-5.6-sol** ×2 · evaluator **sonnet** (generator≠judge holds).
- ⚠ **Skill drift: CLOSED** (was 9 behind). Running skill == `origin/dev`, verified by `cmp`.
- ⚠ New process finding: the evaluator shared the controller's worktree and its mutation drill was
  briefly read as real drift. A judge doing mutations needs its OWN worktree — fix owed.

## Parked on Mark (all on issue #635)
- **`D-1`–`D-14`** — unchanged, see charter. `D-15`/`D-16`/`D-17` were answered 2026-08-14 and are
  now all three DISCHARGED: `D-15` shipped in #710, `D-16` used twice, `D-17` shaped `#706`.
- **Nothing new is blocking.** The queue has three unowned, unblocked items.
