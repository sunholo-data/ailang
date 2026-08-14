# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-14 ~19:55 local (iteration 201)

## Now
- **v0.33.1** · `dev` @ `ba501607d` — squash of PR #716.
- **`#703` LANDED** — `govulncheck-filter` silently dropped module-level findings (a vacuous green
  on the security gate), hiding a real unallowlisted Ollama advisory GO-2026-5750. `readFindings`
  now partitions OSVs into **reaching** vs **module-only** across all frames; gating unchanged
  (reaching only); module-only always reported in a third named bucket, non-gating; GO-2026-5750
  allowlisted (expiry 2026-10-29). Also fixed a **second latent bug**: the old `Trace[0]`-only
  check dropped OSVs reachable via a later frame.
- Repro before routing: pre-fix binary reported `0 findings, all allowlisted` AND (with the real
  allowlist) listed GO-2026-5750 among **9 "stale" entries to delete** — worse than silent.
- Evaluator sonnet **96/100 PASS, zero blocking**, in its own worktree. One non-blocking nit →
  follow-up **#717** (module-only allowlist entries skip the expiry check).
- Gate 3b GREEN: PR #716, `govulncheck (vuln gate)` itself green, 4/4 REQUIRED, all platform legs,
  SonarCloud green (was standing-red). `metered=$0.00` (codex OAuth bucket; no OpenRouter/quorum).

## Next
1. **`[email-parse-DEMAND]`** — `ailang install` on an already-declared dep writes a **duplicate
   TOML key** and breaks the manifest (`lock` rc=1). Reproduced first-party; two sibling helpers
   (`appendDependencyToFile`/`appendGitDependencyToFile`), four call sites — the fix must be an
   idempotent upsert reached by every site.
2. `#717` (module-only expiry annotation, this iteration's follow-up). `#709`/`#649` nightly alarms
   triaged, correctly open. `#610` infra-gated. `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin (`~/.ailang-driver-pin/v1`). Routing: controller **opus** ·
  executor **codex gpt-5.6-sol** · evaluator **sonnet** (generator≠judge holds). Designer, planner,
  quorum **not fired** — direct-fix lane, no design doc (same basis as `#706`/`#692`/`#691`/`#607`).
- **Skill drift: CLOSED.** Running skill == `origin/dev` (`cmp` silent); `D-16` applied to ff-merge
  the main checkout (0 ahead; dirty∩incoming empty, control firing 2; JSONs byte-identical after).
- **Cross-mission**: World's rule-7 skill-fix proposal is already in V1's skill (iter-176). World's
  `#712`/`#713`/`#715` tracked as normal issues. gen≠judge designer-vs-reviewer gap = Gate-5 candidate.

## Parked on Mark (all on issue #635)
- **`D-1`–`D-14`** — unchanged, see charter. `D-15`/`D-16`/`D-17` remain discharged.
- **Nothing new is blocking.** The queue's top unblocked item is the email-parse install-upsert demand.
