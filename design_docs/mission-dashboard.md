# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-15 ~13:25 local (iteration 206 recovery)

## Now
- **v0.33.1** · `dev` @ `de0e41099`; CI and Build-and-Release green on this HEAD.
- **`#717` LANDED** via PR #726 / merge `640bab054`: module-only govulncheck allowlist entries now
  surface expiry honestly, malformed expiries exit 2, and expired module-only findings remain
  deliberately non-gating per `#703`.
- Iteration 206's original controller hit its weekly quota and exited before records. The sprint
  and evaluator path continued to a merged PR; this recovery found the residue, independently
  re-ran the landed gates, and completed the missing bookkeeping.
- PR #726: 21 checks green including Windows, Ubuntu, macOS, CodeQL and SonarCloud. Recovery gates:
  both binaries rebuilt/version-matched; package build/test/vet/gofmt; `go test ./tools/...`; and
  the derived make checks — all green on darwin/arm64.

## Next
1. **`D-COV-1` parked on Mark — one word.** Does coverage mean **LOCALITY** or **EXECUTION**?
   Recommendation LOCALITY. No sprint runs on the doc until answered.
2. `#709`/`#649` correctly open · `#610` infra-gated · `#613` blocked on `D-1`.

## Loop
- Scheduled controller recovered a died-mid-flight iteration; no second backlog item was taken.
- Routing reconstructed from artifacts: original controller opus, executor codex `gpt-5.6-sol`,
  evaluator sonnet in its own worktree; generator≠judge held. Evaluator PASS, score unavailable.
- `metered=$0.00`; running skill matches origin; billing CLEAN; inbox and human directives empty.

## Parked on Mark (issue #635)
`D-1`, `D-2`, `D-8`–`D-14`, and `D-COV-1` are OPEN in the authoritative decision ledger.
