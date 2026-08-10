# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-10 ~13:50 local (iteration 169)

## Now
- **v0.33.0** · `origin/dev` `632024121` · CI green. ⚠ SonarCloud red is standing, non-required,
  inherited (`#615`) — negative control: `failure` on 4 pre-dating commits, absent on the dependabot ones.
- ✅ **Lane B1 M2 landed** (`#517`) — PR **#638** → `632024121`. The splicer now **refuses** instead
  of fabricating a `()` value, so a property can no longer pass on an input it never tested.
  Evaluator **96/100 PASS**, zero blocking. **B1-2 (the debt M1 owed) is paid.**
- ⚠ **M1 had been RECORDED as landed while PR #637 sat OPEN.** Iteration 168 wrote the charter row,
  the log entry and the STATUS stamp, then never merged. New died-mid-flight shape: the traces whose
  *absence* the rule looks for were all **present and pointing the wrong way**. Merged here as
  `59b74e06d`. Cheap guard: check `gh pr list --author sunholo-voight-kampff` against the charter's
  own "LANDED" claims, not just against `[NEXT]` rows.
- ⚠ **codex lane still DOWN and still NOT quota** — `401 … refresh token was revoked` since 08-09.
  Needs a human `codex login`; the loop cannot self-fix (`D-6`).

## In flight
- **#636** `[world-DEMAND]`: `publish --dry-run` truncates digests to 68 bits. Normal queue ordering.
- **#613** proxy M1 DRAFT on `D-1`. **#604**/`#614` on `D-2`. **#624** forall — does not block B1.

## Next
**Lane B1 M3** — structural derivation for records (named/anonymous/nested), tuples, unit, aliases.
Carries the **mandatory** `RecordGenerator` map-order fix: B1-4 is REFUTED at base, so a fixed seed
does *not* reproduce a counterexample today. Then M4 → M6.

## Loop + routing
Controller **opus** · designer **rotation** (next `claude:claude-fable-5`, not fired) · planner
**opus** (codex down) · executor **`pi:deepseek-v4-flash-0731`** · evaluator **sonnet**.
Metered **$0.086**. pi lane datapoint **4** — prescriptive directives suit it; the explicit
"if stuck, write it in FINDINGS.md and move on" clause produced the slot's best finding.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. **(A)** as-written ·
  **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.
- **`D-6`** (iter-168): codex OAuth refresh token revoked — re-auth, or drop codex from routing?

Full record: charter `## STATUS … ITERATION 169` + `v1-mission-log.md` entry 172.
