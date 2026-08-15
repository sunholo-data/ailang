# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-15 ~04:15 local (iteration 204)

## Now
- **v0.33.1** · `dev` @ `54fdcb32c` — squash of PR #723. **SonarCloud Quality Gate PASSED on it**,
  so the standing red is closed at source, not waived.
- **The SonarCloud triage is answered: the red is REAL, not an instrument artifact** — and
  iteration 203's own premise ("0.0% on a heavily-tested diff is implausible") is **REFUTED**.
  The diff *was* heavily tested; every one of #722's 358 new test lines targets
  `cmd/ailang/**`, which `sonar.coverage.exclusions` omits from the metric. What remained
  countable was 14 lines of `internal/pkg/manifest.go` — `pkg.ParseManifestFile` — and
  `make test-coverage` runs **without `-coverpkg`**, so `cmd/ailang`'s three call sites
  attribute nothing to `internal/pkg`. Numerator 0, denominator 7 → **0.0% is correct arithmetic
  on a real gap.**
- **The gap was worse than a metric.** `ParseManifestFile` — the load-bearing half of `#720`'s
  fix — had **zero tests in its own package** (0 hits across all 14 `internal/pkg/*_test.go`;
  same-scope control `LoadManifestFile` → 9). Measured: reverting the mechanism left the
  **entire `internal/pkg` package green**. Its only pin lived in `cmd/ailang`.
- Fixed with 4 arms: the discriminating one asserts `ParseManifestFile` ACCEPTS what
  `LoadManifestFile` REJECTS (both halves — the accept-half alone passes for a function returning
  nil unconditionally), both refusal branches pinned **by message** (`failed to read` /
  `failed to parse` — both return non-nil, so `err != nil` cannot tell them apart), a positive
  control, an anti-vacuity floor. `ParseManifestFile` **0.0% → 100.0%**.
- **Ruled out by measurement**: removing `cmd/ailang/**` from `sonar.coverage.exclusions` — the
  file's own comment says to, but `cmd/ailang` measures **9.3%** unit coverage, so the exclusion
  is still right and the comment's invariant is aspirational.
- Evaluator (sonnet, own worktree) **96/100 PASS, zero blocking**. It re-ran all 3 mutations
  independently and added a **precondition-neutering drill on all 4 arms — none survive**.

## Next
1. **`-coverpkg` is a real open question, not a bug**: cross-package coverage is unattributed
   repo-wide, so any helper added to a non-excluded package to serve `cmd/ailang` reads 0%.
   Changing it moves every number in the repo (badge, the 29% gate, Sonar's baseline) — wants a
   design doc, not a mechanical edit. Queued, not done.
2. `#717` (module-only allowlist entries skip expiry). `#709`/`#649` nightly alarms triaged,
   correctly open. `#610` infra-gated. `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin (`~/.ailang-driver-pin/v1`). Routing: controller **opus** ·
  executor **codex gpt-5.6-sol** · evaluator **sonnet in its own worktree** (gen≠judge holds).
  Designer/planner/quorum **not fired** — direct-fix lane. `metered=$0.00`.
- Skill drift **CLOSED** at pick time: running skill `cmp`-identical to `origin/dev`; local
  `dev` == `origin/dev`, 0 ahead.

## Parked on Mark (all on issue #635)
- **`D-1`–`D-14`** — unchanged, see charter. `D-15`/`D-16`/`D-17` remain discharged.
