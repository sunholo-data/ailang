# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-15 ~02:10 local (iteration 203)

## Now
- **v0.33.1** · `dev` @ `3ec1dcb02` — squash of PR #722.
- **`#720` LANDED**, and the issue's own framing was the find. It says the known `ailang.toml`
  scanner gaps ship no corruption because `writeManifestChecked` rolls the write back. **That holds
  only for manifests that pass semantic validation**: the net's "was it parseable before?" probe was
  `pkg.LoadManifest` — parse **plus** `Validate` — so a perfectly good TOML manifest missing
  `edition` (or with a one-level `name`) counted as *already broken*, the re-check was skipped, and
  the call returned `nil`. Two fixtures differing by one line: without `edition` a **duplicate key
  landed on disk** with `✓ Added`; with it, refused and rolled back. That is `#718`'s failure mode,
  un-netted. Fixed with a parse-only `pkg.ParseManifestFile`.
- Four filed gaps also closed: literal `"""` no longer opens a multi-line string; `["dependencies"]`
  recognised (`[[dependencies]]` and dotted headers refused); `stripLineComment` escape-aware;
  `countDependencyKey` repaired *without* calling production helpers, so it stays a real control.
  Already-broken manifests warn instead of printing `✓ Added`.
- **Decision made, not drifted into**: keep hand-editing TOML text rather than
  parse→mutate→re-serialize — comment/formatting preservation is a real product property and the
  fixed net bounds the blind spots. Recorded so it is not re-litigated.
- Guarded by the **differential instrument iteration 202 lacked**: a 12-shape corpus running the
  real install path twice per shape, with an anti-vacuity floor.
- Evaluator (sonnet, own worktree) **94/100 PASS, zero blocking**. Its three non-blocking findings
  were all unpinned refusal branches — each reproduced by me (mutant builds+lands, package stays
  `rc=0`) then closed in `72b60b153`. One was `appendGitDependencyToFile`'s copy of the very warn
  branch this sprint added: **guard the helper, miss the call site, one commit after fixing it.**
- Gate 3b: **4/4 REQUIRED green** (`build`/`docs-gate`/`lint`/`test` 17m1s), 21 checks, `pending=0`.
  `UNSTABLE` = non-required SonarCloud (0.0% new-code coverage vs an 80% bar). `metered=$0.00`.

## Next
1. **SonarCloud reads 0.0% new-code coverage on a heavily-tested diff** — implausible, and PR #719
   read 61.1% on the same gate. Worth one triage pass: instrument or real? Non-required either way.
2. `#717` (module-only allowlist entries skip expiry). `#709`/`#649` nightly alarms triaged,
   correctly open. `#610` infra-gated. `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin (`~/.ailang-driver-pin/v1`). Routing: controller **opus** ·
  executor **codex gpt-5.6-sol** · evaluator **sonnet in its own worktree** (gen≠judge holds).
  Designer/planner/quorum **not fired** — direct-fix lane (same basis as `#718`/`#703`/`#706`).
- **Skill drift: found OPEN, now CLOSED.** The running skill was **2 commits stale** (missing Gate-2
  rule 3l) because the driver pin covers the *driver* and the `~/.claude` symlink still resolves to
  the *main checkout*. `D-16` applied (0 ahead; dirty∩incoming empty, controls 9/9 and 2/2).

## Parked on Mark (all on issue #635)
- **`D-1`–`D-14`** — unchanged, see charter. `D-15`/`D-16`/`D-17` remain discharged.
