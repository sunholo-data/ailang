# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-14 ~23:15 local (iteration 202)

## Now
- **v0.33.1** · `dev` @ `afe06487e` — squash of PR #719.
- **`#718` LANDED** — `ailang install` on an already-declared dep wrote a **duplicate TOML key**
  (manifest unparseable, `lock` rc=1), and the resulting error blamed a *missing* `ailang.toml`.
  Both halves fixed: one shared idempotent `upsertDependencyLine` in the helpers (so all 5 call
  sites inherit it), and `tryLoadPackageResolver` now returns `(resolver, error)` — absent manifest
  still `(nil, nil)`, existing-but-broken names its path + the TOML error. Lock load split the same
  way (missing optional, malformed loud).
- **Round 1 FAILED 66/100 on four TOML-legal blind spots — one a regression the PR introduced**:
  `[dependencies] # comment` failed the new exact-string compare, so a SECOND table was appended,
  *reproducing `#718`'s own error on input the pre-fix substring match handled*. Lesson worth
  keeping: replacing a permissive mechanism with a precise one silently withdraws the permissive
  version's accidental coverage, and **no test written for the new code can see that**. The missing
  instrument is a *differential* one (old vs new over a corpus), and nothing asked for it.
- Answered structurally, not by four patches: `writeManifestChecked` re-parses after every write and
  **rolls back** anything that would leave a manifest less parseable than it found it. Round 2
  **PASS 95/100, zero blocking**; every new gap the judge then found was caught by that net → `#720`.
- Gate 3b GREEN: 21 checks, `pending=0`, 4/4 REQUIRED. `UNSTABLE` = non-required SonarCloud only,
  and that red is **not inherited** (parents green): new-code coverage 55.6%→61.1% vs an 80% bar.
  `metered=$0.00` (codex OAuth bucket; sonnet quota; no OpenRouter/quorum lane fired).

## Next
1. **`#720`** — residual `ailang.toml` upsert scanner gaps (literal string containing `"""`; quoted
   header `["dependencies"]`; escape-unaware comment stripping; the test helper's own version of the
   same gap). None ships corruption — the rollback net refuses the write. Carries the real decision:
   keep hand-editing TOML text, or parse → mutate → re-serialize and lose comment preservation.
2. `#717` (module-only allowlist entries skip expiry). `#709`/`#649` nightly alarms triaged,
   correctly open. `#610` infra-gated. `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin (`~/.ailang-driver-pin/v1`). Routing: controller **opus** ·
  executor **codex gpt-5.6-sol** · evaluator **sonnet in its own worktree** (gen≠judge holds).
  Designer/planner/quorum **not fired** — direct-fix lane (same basis as `#703`/`#706`/`#692`).
- **Skill drift: CLOSED.** Running skill == `origin/dev` (`cmp` silent); `D-16` applied to ff-merge
  the main checkout (0 ahead; dirty∩incoming empty against a firing 6-file control).

## Parked on Mark (all on issue #635)
- **`D-1`–`D-14`** — unchanged, see charter. `D-15`/`D-16`/`D-17` remain discharged.
