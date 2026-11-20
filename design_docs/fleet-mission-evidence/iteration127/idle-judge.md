# Fleet iteration 127 — independent idle-screen judge

- **Role**: `evaluator` (`MISSION-ROLE: evaluator`), Agent tool `model=sonnet` (resolver `tools/launchd/resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`), foreground, 192 s, 28 tool uses, 134,365 tok. Read-only at `226a01056`.
- **Brief**: inputs only. The brief listed [tickets.json](tickets.json), [open-decisions.tsv](open-decisions.tsv), the charter, log entries 114 and 126, the D-FLEET-45 design and its iteration-97 record, the fleet sections of `gate-2-pick.md` and `gate-5-retro.md`, `idle-fingerprint.sh`, World's `world-iter416/gate1-2-readings.txt`, and the World driver log ([excerpt](world-idle-snapshots.txt), [slot verdicts](world-slot-verdicts.txt)). It asked the judge to argue against idling and to test World 416's ticket sentence first-party. It carried no controller conclusion.
- **Controller re-checks (first-party)**:
  - `ailang mission ticket open --json` still lists 16 tickets. `skill:idle-record-barred-on-degraded-no-pick-fire` shows `slots_lost=6`, `last_seen=2026-11-20T05:41:42Z`, latest = World iteration 417. That occurrence arrived during this fire.
  - `0222baeb9` is "fix(quorum): Anthropic seat runs on an unreadable quota (D-FLEET-28 A) (#2344)" and is an ancestor of `origin/dev`.
  - `quorum:claude-p-ration-usage-429` appears once in the charter, in D-FLEET-28's row. It has no queue row and is not one of D-FLEET-46's seven.
  - No `mission-*-idle-fingerprint` record file exists under `~/.ailang/state/`. Only the `world` and `stapledon` `.pending` snapshots are there.

---

VERDICT: NONE (confidence 75%)

## Q1. Ticket to gate (16 of 16)

| # | Signature | Gate |
|---|---|---|
| 1 | `ci:distinct-build-test-failures` | Row 44. The failing tests are in `cmd/ailang`, `internal/daemon` and `internal/runner`, outside the Authority allowlist. |
| 2 | `coordinator:cloud-lane-capped-by-local-worktree-limit` | Row 52. Coordinator code, out of scope. |
| 3 | `coordinator:job-retry-success-dropped-after-failed-completion` | Row 55, RULED A 11-16, attended PR #2348. Open only for bookkeeping, so it falls under D-FLEET-46. |
| 4 | `crash:sprint-executor:sigbus-ram-tmp-full-make-test` | Row 27 and D-FLEET-23, RULED. The rest is attended or V1 (`cmd/ailang/coordinator_cloud_task.go`) plus terraform, out of scope. |
| 5 | `eval:motoko-connection-probe-orphaned` | Row 58, folded into D-FLEET-18 (RULED A; the allowlist arm #2382 and the flake fixes landed). Resolving it is a bookkeeping act with no fleet diff, the same class as D-FLEET-46. |
| 6 | `executor:3h-timeout-from-queue-and-unsliced-sprints` | Row 56, RULED A, #2343. D-FLEET-46. |
| 7 | `executor:codex-credential-collision-drops-automatic-planner-handoffs` | Row 54, RULED A, #2348. D-FLEET-46. |
| 8 | `executor:codex-credential-lease-collision-fails-concurrent-dispatch` | Row 54. D-FLEET-46. |
| 9 | `executor:timeout-loses-work-and-may-never-start` | Row 56, RULED A, #2343. D-FLEET-46. |
| 10 | `messaging:pubsub-notify-wrong-project` | Row 18, D-FLEET-19 RULED A. An attended rig change with no repo commit, so `ticket resolve --sha` cannot be satisfied. D-FLEET-46 excludes it. |
| 11 | `pi:openrouter:uncapped-max-tokens-402` | Row 50, RULED A, #2349. D-FLEET-46. |
| 12 | `quorum:claude-p-ration-usage-429` | D-FLEET-28 RULED A, merged as #2344 (`0222baeb9`). No queue row, and not on D-FLEET-46's list of seven. Same class, so it needs the same ruling. An observation, not a pick. |
| 13 | `quorum:gpt6-1-sol-openai-api-credits-exhausted` | Row 21, RULED A, #2347 (`41c628b94`). D-FLEET-46. |
| 14 | `reporting:daneel-reports-not-declared-triage` | Row 24, D-FLEET-22 RULED A. A prod-config rollout with no PR, out of scope, excluded from D-FLEET-46. |
| 15 | `skill:idle-record-barred-on-degraded-no-pick-fire` | Row 62, D-FLEET-45 OPEN. 6 slots lost; the latest message is World 417. |
| 16 | `tool-hang:triage-judge:home-metadata-search` | Row 61, D-FLEET-44 OPEN. The design is quorum-blocked. |

**Screen applied to the three OPEN rows**
- **D-FLEET-44**: A is "request a corrected design and evidence", a design revision, not a code step; the design already failed two reviews. B is the default.
- **D-FLEET-45**: the recommendation is B (no change). A is the only non-default option, so a literal reading of the "step shared by all non-default options" prong matches it, but A begins with a design revision, not an in-scope code step.
- **D-FLEET-46**: A is a bookkeeping resolve of tickets whose fixes landed attended. There is no fleet diff to build, and resolving is what the row asks permission for.

## Strongest argument against NONE

D-FLEET-45 option A, read against the screen's second prong: the loop could build the revised design plus a 10-fire retro-analysis as a draft PR. The ticket has a 6th occurrence and B's premise has aged (Q2). NONE still holds: the row recommends B and a build would pre-empt it; the screen's intent is code arms that must ship alone first; and the design is already quorum-blocked twice. Second strongest: resolve row 58 as "owned" under D-FLEET-18 A. Rejected because it infers ownership from landed code, which is the act D-FLEET-46 asks permission for.

## Q2. World 416's claim

World driver log timestamps are local (UTC+1). World 416 started 01:39Z and World 417 05:39Z.

**(a) A recorded 416 snapshot would NOT have let 417 idle.** 416: `01aae166… head=0e9e8d3 ledger=107 open=2 directives=0 inbox=1 red=0`. 417: `0355e1e2… head=0e9e8d3 ledger=107 open=2 directives=0 inbox=0 red=0`. The driver compares the fresh fingerprint with the record (`tools/launchd/mission-control.sh:1789`), so this is "inputs changed". The only difference is the inbox, 1 → 0: the fleet's `[harness-resolved]` reply, which 416 acked itself. World 417's own ticket text says the same.

**(b) Byte-identical fingerprint pairs** over 81 snapshots: `40d76c3d` (11 fires, 11-08 → 11-09), `145eb5c6` (18 fires, 11-13 → 11-16, red=1, crashed/paused fires, not no-pick), `cba752ed` (2 fires, 11-07), `0355e1e2` (11-19 02:39 and 11-20 06:39 local, 27 h apart, past the 24 h expiry at `mission-control.sh:1773`). What separates the fires that differ: head moved with record commits (the old exclusion), ledger count 104 → 105 → 107, inbox 0 ↔ 1. Recomputed from the World repo: with the new pathspec every fire since 11-19 01:39Z has head `0e9e8d35`; with the old one it moved through 8 heads. That supports the row-264 part of the claim.

**(c) What else would still stop an idle skip**: the self-changed inbox (a); the 24 h expiry (b); the IDLE RECORD gate conditions (lane park, clock wait, pick), whose state in 416 is UNMEASURED. Counterfactual, assuming every no-pick fire could record: of the 8 fires from 11-19 01:39Z to 11-20 05:39Z (inbox 0,1,0,0,0,0,1,0, head/ledger/open constant), at most 3 would have skipped. No "nothing changed since the last no-pick fire" line has ever appeared in the World log (0 matches).

**(d) Bearing on D-FLEET-45 B**: B's premise was measured on World 370 (external predicates, a weekly sweep, a branch-only D-WORLD-113). World 410–417 are all no-pick fires whose stated gate is the dev-side OPEN rows D-WORLD-119/120, the shape the narrowed rule targets, so the premise may no longer hold. Beyond the headlines that is UNMEASURED (rows 150/151 and the weekly sweep may have non-ledger unblock events). The evidence does not support World 416's sentence that the degraded bar is "the only thing" keeping identical fires from idling, since the inbox ack alone woke 417. It does not show that A would fix World either, because the 24 h expiry and inbox churn also bound the savings. Not a recommendation either way.

**UNMEASURED**: whether the narrowed rule's "only unblock event is a ledger flip" test passes for World 410–417's candidates; whether World's Gate-0 crash-notice condition bars a record; whether the merged fixes cover their tickets' repros at HEAD (D-FLEET-46's A step).
