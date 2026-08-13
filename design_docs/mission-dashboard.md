# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-13 ~14:40 local (iteration 192)

## Now
- **v0.33.1** · `dev` @ `7bad0e609` — iteration 192's own merge. Gate 3b GREEN on the PR
  (SHA-addressed **21** checks, `pending=0`, zero failures, 4/4 REQUIRED contexts).
- **`m-batch-exit-panic` (`#607`) LANDED** — first code landing since the queue went human-gated.
  `exit()` in one batch item raised the `*eval.EvalExitCode` sentinel through `executeBatchItem`,
  which had no recover: rc=2, raw Go stack, every remaining input silently skipped (reported from
  a 2,500-file PDF batch). The recover existed on the single-file path; one call site lacked it.
  Now `exit(N≠0)` fails **that item** and the loop continues, `exit(0)` succeeds, real crashes
  still panic. Evaluator sonnet **PASS 96/100 r1, zero blocking**, every claim reproduced.
- **Batch mode had ZERO regression tests** — proven by the inverse arm (recover removed + new
  tests skipped → rest of `cmd/ailang` rc=0). Five now, all four recover branches mutation-pinned.

## New this iteration (both from the evaluator, both reproduced before filing)
- **`#691`** — `exit()` in *embedded* AILANG still panics the **host**: `internal/embed` has
  **zero** `recover()` (control: `run_helpers.go`=2). Same defect class one layer down; needs a
  one-word contract decision (typed error vs recover), so it was filed, not fixed inline.
- **`#692`** — batch mode never calls `flushDebugOutput`, so Debug output works per-file and
  silently vanishes per-batch.

## Parked on Mark (all on `#635`)
- `D-1`, `D-2`, `D-7`–`D-14` — no reply since the 2026-08-13T04:58Z watermark (0 of 43 comments).

## Next (if nothing unparks)
- **`#691`** is the likely pick — this iteration's own finding, and the only unblocked row that
  is not infra-gated. `m-mapE-queryall-retention` (`#610`) sits above it but its design is gated
  on repro infra this rig lacks (duckdb CLI + `sunholo/duckdb@0.1.1` + `-memprofile`).

## Loop health
- ⚠ **Driver `$MODEL` env UNSET again — instance 2 after iter-191.** Harmless this fire (session
  default *is* opus, the policy pin), but the protection is luck, not configuration. Driver fix owed.
- ⚠ **`gh issue close --comment` on an ALREADY-CLOSED issue silently drops the comment, rc=0.**
  A PR's `Fixes #N` auto-closes first, so this is the normal case. Use `gh issue comment --body-file`.
- Skill-drift `cmp` must target the **`readlink`** destination (the MAIN checkout), not the
  checkout you are standing in — the wrong copy reads green regardless.
- Designer rotation pointer unchanged (`codex:gpt-5.6-sol` last-used; no designer fired).
  `metered=$0.00` — no quorum, no cross-provider lane.
- Main checkout is **2 ahead / dirty** with a sibling's observatory work — Principle 0; this
  record was written from a worktree off `origin/dev`.
