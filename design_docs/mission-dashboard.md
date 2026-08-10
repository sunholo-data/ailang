# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-10 ~11:15 local (iteration 168)

## Now
- **v0.33.0** · `origin/dev` `7d8db911f` · CI green. ⚠ SonarCloud red is standing, non-required,
  inherited (`#615`) — negative control: `failure` on two pre-dating commits, absent on the newest four.
- ✅ **Bookkeeping thread ROTATED: `#559` → [#635](https://github.com/sunholo-data/ailang/issues/635)**
  (Monday-07:00 CEST boundary; 68 comments). Comment there, not on `#559`.
- ✅ **Lane B1 M1 landed** (`#517`) — PR **#637**: contract property tests can now splice unit/record/
  tuple/constructor values. Inert by design; M2 flips the silent default. Evaluator **82/100 PASS**.
- ✅ **Weekly external-issue sweep CLEAN**: 0 of 52 open issues unmentioned in the charter.
- ⚠ **codex lane is DOWN and it is NOT quota** — every probe since ≥08-09 22:45 returns
  `401 … refresh token was revoked`. Needs a human `codex login`; the loop cannot self-fix. Planner
  has been falling back to opus on every fire.
- ⚠ **A killed executor kept running and overwrote a verified tree.** `pkill` reported success, the
  worktree read clean, then M2-shaped edits appeared over the M1 files mid-evaluation. Commit was
  already made, so nothing was lost; delta preserved at `/tmp/iter168_orphan_delta`.

## In flight
- **#636** (new, `[world-DEMAND]`): `publish --dry-run` truncates all three digests to 68 bits and
  no surface prints them in full. Reproduced at HEAD. Normal queue ordering, no date promised.
- **#613** proxy M1 DRAFT on `D-1`. **#604**/`#614` on `D-2`. **#624** forall — does not block B1.

## Next
**Lane B1 M2** — needs the test seam (see the plan's CONTROLLER CORRECTION): 3 of its 4 call-site
refusal branches are unreachable without one, permanently, so they would ship as decorative guards.
Then M3 (derivation + the `RecordGenerator` map-order determinism bug) → M4 → M6. Owed from M1:
acceptance criterion **B1-2** (round-trip through the evaluator) is unmet and recorded.

## Loop + routing
Controller **opus** · designer **rotation** (next `claude:claude-fable-5`, not fired) · planner
**opus** · executor **`pi:deepseek-v4-flash-0731`** · evaluator **sonnet**. Metered **$0.040**.

## PARKED ON MARK — asks are on #635 now
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. **(A)** as-written ·
  **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-6`** (NEW): codex OAuth refresh token revoked — re-auth, or drop codex from the routing table?
- ~~`D-3`~~ landed as `1239d9ec6`. ~~`D-4`~~, ~~`D-5`~~ resolved.

Full record: charter `## STATUS … ITERATION 168` + `v1-mission-log.md` entry 171.
