# Fleet iteration 73 — draft records handed to the independent judge

Base: `origin/dev` = `490fb730b2f809f846c2c9b437b2aefcdfc12933` at Gate 1 (2026-10-30T18:53:58Z);
iteration 72's record PR #2028 was then merged by this fire (`dd183217458b5e897e8669262bf51dc1f82249d1`),
and every claim below was measured on `dd1832174`. Controller: `claude:claude-opus-5-5`. No code is
proposed. Both records are bookkeeping: two new decision rows.

## Record 1 — park `skill:design-doc-creator-grep-portability` as decision D-FLEET-36

Ticket (attended, filed 2026-10-03T07:14:46Z, message `inbox_1791011686923_b09ca1db`). Asks:
(a) replace `grep -E "^\d+\."` with `^[0-9]+\.` in `create_planned_doc.sh`; (b) make the
`| head -5` pipeline in `merge_results` not abort under `set -euo pipefail` on zero matches;
(c) land the fix in `ailang_bootstrap` too, so the daily sync-cloud-skills job does not revert it.

Claims:
- C1. Ask (a) is unbuilt at HEAD in both copies. `.claude/skills/design-doc-creator/scripts/create_planned_doc.sh`
  lines 128, 137, 154, 163 and `.agents/skills/design-doc-creator/scripts/create_planned_doc.sh`
  lines 125, 134, 153, 162 all read `grep -E "^\d+\."`. Each is followed by `|| echo ""`, so the
  failure mode is an empty duplicate-search result ("(none found)"), not an abort.
- C2. The defect is platform-dependent and does NOT reproduce on the rig. On this Mac,
  `/usr/bin/grep` is `grep (BSD grep, GNU compatible) 2.6.0-FreeBSD`, and
  `grep -E "^\d+\."` matched `1. design_docs/x.md (0.85)` (rc 0) with the `[0-9]` form as a control
  giving the same line. Under `/bin/bash`, `type grep` = `/usr/bin/grep`. No GNU grep is installed
  on the rig (no `ggrep`, no docker), so the GNU half was NOT measured first-party; per GNU grep's
  documented ERE, `\d` is not a digit class (it matches a literal `d`, and GNU grep ≥3.8 warns
  "stray \ before d"), so on Linux (cloud executors, the bootstrap-synced copies) the four searches
  return empty.
- C3. Ask (b) is built in both copies, by different code. `.claude` copy: `b51353bcd` (#1542,
  2026-10-03T07:16Z, two minutes after the ticket) moved the filter into awk
  (`awk '/^[0-9]+\./ {…}' | head -5`, line 120), so the pipeline no longer has a failing `grep`.
  `.agents` copy (still the pre-#1542 version): line 116 keeps `grep -E "^[0-9]+\." | awk … | head -5`,
  and the `IMPLEMENTED=$(merge_results …)` call carries a `|| true` (comment at line 141).
- C4. The fix is outside fleet authority. `tools/launchd/githooks/pre-push` `_scope_is_harness`
  allows only `.claude/skills/mission-*|sprint-*` and `.agents/skills/mission-*|sprint-*` among
  skills; `_scope_is_fleet_support` has no skill path. So `.claude/skills/design-doc-creator/**`
  is refused for `MISSION_NAME=fleet`. Ask (c) is a different repo (`ailang_bootstrap`).
  Queue row 25 already records `.agents/skills/design-doc-creator` as outside scope (residue owned by
  `skills:agents-copies-stale-non-mission`).
- C5. Widening scope is a ruling, not a mechanical fix: every earlier scope widening was a Mark
  decision (D-FLEET-5, D-FLEET-6). Note the mechanism: adding the path to `_scope_is_harness`
  would also REFUSE it for every product loop (`_scope_verdict` line 60); adding it to
  `_scope_is_fleet_support` widens the fleet only.
- Proposed D-FLEET-36 (OPEN): "`skill:design-doc-creator-grep-portability` (attended, 10-03): ask (a)
  is real on GNU grep only, and the skill is outside fleet scope. Widen scope, or fix attended?"
  Recommendation **A**: add `.claude/skills/design-doc-creator/*` and `.agents/skills/design-doc-creator/*`
  to `_scope_is_fleet_support` (not `_scope_is_harness`, so product loops are unaffected); the fleet
  then replaces the four `\d` sites in each copy with `[0-9]` and resolves the ticket. Ask (c),
  `ailang_bootstrap`, stays with Mark. Alternative **B**: no scope change; an attended session makes the
  eight-line edit and the fleet resolves the ticket against that SHA. Default until answered: ticket
  stays open; nothing changes; impact is a silently empty duplicate search on Linux only.

## Record 2 — park `sprint-evaluator:worktree-cap-and-exit1` as decision D-FLEET-37

Ticket (attended, filed 2026-10-09T06:17:36Z, message `inbox_1791526656110_18821c77`). Prod
coordinator, 2026-10-08: six sprint-evaluator handoffs failed `max worktrees limit reached (3)`,
three failed `exit status 1`, two executors timed out at 3 h with no PR. Asks: (1) the evaluator
must not share a 3-worktree cap with concurrent executors, or must queue instead of failing;
(2) root-cause the exit-1 evaluator failures; (3) executors commit per milestone so a timeout keeps
partial work.

Claims:
- C6. Ask 3 is built: a ghost of #1688. `face88a3f` (#1688, merged 2026-10-09T07:27:27Z, about
  70 min after the ticket) added `.claude/skills/sprint-executor/resources/unattended_runs.md`, whose
  first rule is "**Commit after each milestone** on the work branch. Uncommitted work is lost work".
  `3f9a0c092` (#1905, fleet iteration 49, 2026-10-20) extended it to cloud tasks: "Cloud tasks: push
  every milestone" via `cloud_branch_sync.sh`. Both are ancestors of `origin/dev`.
- C7. Ask 1 is unbuilt and outside fleet scope. The cap is `MaxWorktrees: 3` in
  `internal/coordinator/daemon.go:47,54` (`DefaultConfig`), enforced via `ErrWorktreeLimitReached`
  (`internal/coordinator/worktree.go:17`). A `--max-worktrees N` flag already exists on
  `coordinator start` (`cmd/ailang/coordinator_lifecycle.go:43-49`), so raising the prod cap is a
  deploy-config change (multivac) and making evaluators queue is a coordinator code change; both are
  outside the fleet allowlist (`internal/coordinator/**`, multivac repo).
- C8. Ask 1 overlaps the open ticket `coordinator:cloud-lane-capped-by-local-worktree-limit`, which
  cites the same `daemon.go:47` cap and Mark's 2026-10-12 remark that cloud execution should be
  bounded by explicit config, not a local disk cap. Neither ticket has a queue row or decision row in
  the charter.
- C9. Ask 2 is an investigation of prod coordinator task failures (`internal/coordinator`, Cloud Run
  logs); nothing in fleet scope produces or consumes those exit codes. Not measured this iteration.
- Proposed D-FLEET-37 (OPEN): "`sprint-evaluator:worktree-cap-and-exit1` (attended, 10-09): ask 3 is a
  ghost of #1688; asks 1–2 are coordinator code and prod config, outside fleet scope. Who owns them?"
  Recommendation **A**: V1/attended owns both; fold ask 1 into
  `coordinator:cloud-lane-capped-by-local-worktree-limit` (one fix for one cap: a configured cloud
  limit, with evaluator handoffs queuing rather than failing), and route ask 2 with it. The fleet then
  resolves this ticket as a duplicate + ghost. Alternative **B**: widen fleet scope to
  `internal/coordinator/**`. Not recommended: the coordinator is the shared task plane for every
  agent, not the mission loop harness. Default until answered: ticket stays open.
