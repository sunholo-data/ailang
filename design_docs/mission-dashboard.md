# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-15 ~16:10 local (iteration 207)

## Now
- **v0.33.1** · `dev` @ `7c23f6a3d`; CI and Build-and-Release green; 16 SHA-addressed checks green/neutral.
- **`#709` diagnosed**: `config_file_parser` is a stable local-model capability gap, not a fresh
  AILANG regression. Five-night banked sample: 2/10 pass, 4/10 token-budget thrash, 4/10 independent
  agent mistakes against explicit task requirements or existing APIs.
- No code or design sprint was justified. The issue remains open as tracked capability evidence;
  the evidence verdict was posted and delivery verified (comment count 1 → 2).

## Next
1. **`D-COV-1` parked on Mark — one word.** Does coverage mean **LOCALITY** or **EXECUTION**?
   Recommendation LOCALITY. No sprint runs on the doc until answered.
2. `#649` correctly open as a diagnosed capability gap · `#610` infra-gated · `#613` blocked on `D-1`.

## Loop
- Analysis-only iteration; controller used the eval-analyzer workflow directly. No designer,
  planner, executor, evaluator, quorum, GPU, or metered provider lane fired.
- `metered=$0.00`; running skill matches origin; billing CLEAN; five eval-suite notices triaged as
  informational; no allowlisted human directive.

## Parked on Mark (issue #635)
`D-1`, `D-2`, `D-8`, `D-9`, `D-10`, `D-11`, `D-12`, `D-13`, `D-14`, and `D-COV-1` are OPEN in
the authoritative decision ledger.
