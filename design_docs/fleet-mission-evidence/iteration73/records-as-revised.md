# Fleet iteration 73 — records as revised after the round-1 judgment

Round 1 (`evaluation.md`): record 1 PASS-WITH-EDITS 82, record 2 FAIL 66 as drafted. The controller
re-checked the judge's new finding first-party before applying it:
`internal/coordinator/daemon_tasks_exec_run.go:163-170` resets a task that hits
`ErrWorktreeLimitReached` to pending ("it will be retried when a worktree frees up") and returns the
error; `git log -S ErrWorktreeLimitReached -- internal/coordinator/` dates the code to `debf5b66e`
(2026-01-01). All five required edits are applied below. These are the ledger rows as they will be
committed.

## D-FLEET-36 (OPEN)

**`skill:design-doc-creator-grep-portability` (attended, 10-03): ask (a) is real on GNU grep only, and
the skill is outside fleet scope. Widen the fleet's scope to the design-doc-creator skill, or fix it
attended?** Measured at iteration 73 on `dd1832174`; an independent sonnet judge verified it
(PASS-WITH-EDITS 82, edits applied). All 8 `grep -E "^\d+\."` sites are still there (`.claude` copy
`create_planned_doc.sh:128/137/154/163`, `.agents` copy `:125/134/153/162`), each followed by
`|| echo ""`, so the failure is an empty duplicate search, not an abort. Measured on this Mac only:
BSD grep 2.6.0-FreeBSD treats `\d` as a digit class (control `^d+\.` matched only the literal-d
line), so the ticket's "BSD/GNU" claim is refuted for BSD. GNU grep was not available; GNU's
behaviour (`\d` = a literal `d`) is documented, unmeasured. Ask (b), the `head -5` abort, is built in
both copies by different code (`.claude`: #1542 `b51353bcd` moved the filter into awk; `.agents`:
`|| true` at the `merge_results` calls). Ask (c) is the `ailang_bootstrap` repo. The guard
(`tools/launchd/githooks/pre-push`) allows only `mission-*` and `sprint-*` skills.
**Recommendation A:** add `.claude/skills/design-doc-creator/*` and `.agents/skills/design-doc-creator/*`
to `_scope_is_fleet_support`, not `_scope_is_harness` (which would refuse the path for every product
loop). This widens the fleet to the whole design-doc-creator skill, not eight lines, and the Authority
section and the `pre-push` comments change together. The fleet then replaces the 8 sites with `[0-9]`
and resolves the ticket; ask (c) stays yours. **B:** no scope change; one attended commit makes the
8-line edit, and the fleet resolves the ticket against it. Default until answered: the ticket stays
open, Linux duplicate searches stay silently empty, and the bootstrap-synced cloud copies are
unaffected until ask (c) is done.

## D-FLEET-37 (OPEN)

**`sprint-evaluator:worktree-cap-and-exit1` (attended, 10-09): ask 3 is a ghost of #1688; asks 1 and 2
are in the coordinator, outside fleet scope. Who owns them?** Measured at iteration 73 on
`dd1832174`. An independent sonnet judge scored the draft FAIL 66 because it missed existing retry
code; its edits are applied. Ask 3: `face88a3f` (#1688, merged 2026-10-09T07:27Z, about 70 min after
the ticket) added "Commit after each milestone" (`sprint-executor/resources/unattended_runs.md:6`),
and #1905 `3f9a0c092` added push-every-milestone for cloud tasks. Prod executor behaviour since then
is not measured. Ask 1: the cap is `MaxWorktrees: 3` (`internal/coordinator/daemon.go:47,54`), and
`coordinator start --max-worktrees N` exists (`cmd/ailang/coordinator_lifecycle.go:43-49`); whether
prod passes it is assumed, not inspected. `internal/coordinator/daemon_tasks_exec_run.go:163-170`
already resets a task that hits `ErrWorktreeLimitReached` to pending for retry (since `debf5b66e`,
2026-01-01), so the open question is why the 10-08 evaluator handoffs surfaced as failed. Whether
queueing needs new coordinator code is unverified. Ask 2 (the exit-1 root cause) is not measured.
**Recommendation A:** V1/attended owns asks 1 and 2. Route ask 1 together with
`coordinator:cloud-lane-capped-by-local-worktree-limit`, which names the same cap but asks for a
different fix (exempt cloud tasks from the local worktree slot); start from why reset-to-pending did
not hold. The fleet then resolves this ticket. **B:** widen fleet scope to `internal/coordinator/**`.
Not recommended: the coordinator is the shared task plane for every agent, not the mission loop
harness. Default until answered: the ticket stays open.
