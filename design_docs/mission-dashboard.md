# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-10 ~18:00 local (iteration 170)

## Now
- **v0.33.0** · `origin/dev` `48cf25cff` · PR #645 merged, all required contexts green. Standing
  SonarCloud red (`#615`) unchanged, non-required, inherited.
- ✅ **Lane B1 M3 landed** (`#517`) — records, tuples, unit, aliases all derive generators now;
  seeded replay actually replays (F-3 map-order fix). Evaluator **95/100, zero blocking**.
- ⚠ Sharpest find: the plan's "preserve the list arms untouched" preserved a **stack overflow** —
  `type Tree = { val: int, kids: [Tree] }` crashed the deriver once named types became derivable.
  Fixed + crash-pinned in the same PR. "Preserve verbatim" is a claim about the OLD reachability graph.
- ✅ **`D-6` resolved** — Mark re-authed codex 09:43; planner lane restored (not needed this slot).
- ⚠ The weekly sweep's 08-10 CLEAN was a **false negative** — 4 orphaned issues (`#616`–`#619`)
  found by attended re-measure, all triaged + queued this iteration. Skill now demands a
  per-issue count table, never a summary verdict (`2d4a8118a`).

## In flight / queued
- **Orphaned-issue batch** (iter-170): `#618` (ollama 300s cap — live eval-noise cause, doc exists)
  → `#619` (publisher counts harness errors as capability fails, W8 P0) → `#616` (effect-row vars
  never unify, repro'd) → `#617` (strict-eval flatMap OOM class, design-first).
- **#636** `[world-DEMAND]` publish digest truncation · **#613** proxy M1 DRAFT on `D-1` ·
  **#604**/`#614` on `D-2` · **#624** forall — none block B1.

## Next
**Lane B1 M4** — ADTs, recursion/size budgets, `TypeApp` substitution. Now also owns the
evaluator's NB: the depth-3 budget makes record-via-list types unconditionally underivable even
when legitimately inhabited (`{val:1, kids:[]}`). Then M5/M6.

## Loop + routing
Controller **fable this slot** (driver-selected; table default opus — FLAGGED) · designer rotation
(next `claude:claude-fable-5`, not fired) · planner codex restored (not fired) · executor
**`pi:deepseek-v4-flash-0731`** (5 good prescriptive datapoints, $0.02–$0.16/run) · evaluator
**sonnet**. Metered **$0.160** this iteration.

## PARKED ON MARK — asks are on #635
- **`D-1`** (iter-150): proxy drops target-IP SSRF pinning on **proxied** routes. **(A)** as-written ·
  **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604` closes the top-level vacuous pass, leaves the nested one (`#614`).
  **(A)** top-level-only · **(B)** widen · **(C)** reject multi-expression test bodies.

Full record: charter `## STATUS … ITERATION 170` + `v1-mission-log.md` entry 173.
