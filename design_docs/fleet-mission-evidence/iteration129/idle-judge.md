# Fleet iteration 129 — independent idle-screen judge

- **Role**: `evaluator`, read-only at `429eaf806` (origin/dev). Agent tool, `model=sonnet`, `MISSION-ROLE: evaluator`, foreground (62,507 tok, 6 tool calls, 40 s). Inputs only; no controller conclusion was carried in. The iteration 128 report was given as input to re-verify, not to inherit.
- Files it read: `design_docs/fleet-mission-evidence/iteration129/`, `design_docs/fleet-mission-evidence/iteration128/idle-judge.md`, `~/dev/sunholo-data/ailang-world/design_docs/world-mission-log.md`, `/tmp/ailang-mission-world.log`.

---

VERDICT: NONE (confidence 78%)

## Q1. Ticket dispositions (`ailang mission ticket open --json`: 16 tickets)

| Ticket | Slots lost | Disposition |
|---|---|---|
| `reporting:daneel-reports-not-declared-triage` | 20 | Out of scope: prod-config rollout (row 24, D-FLEET-22), no repo commit. |
| `skill:idle-record-barred-on-degraded-no-pick-fire` | 9 | Parked on OPEN D-FLEET-45. Gate-5 skill resource is in scope, but the design failed quorum twice. |
| `quorum:gpt6-1-sol-openai-api-credits-exhausted` | 3 | Awaiting D-FLEET-46 (#2347). |
| `messaging:pubsub-notify-wrong-project` | 2 | Out of scope: rig change, no commit to cite. |
| `pi:openrouter:uncapped-max-tokens-402` | 1 | Awaiting D-FLEET-46 (#2349). |
| `executor:3h-timeout-from-queue-and-unsliced-sprints` | 1 | Awaiting D-FLEET-46 (#2343). |
| `executor:timeout-loses-work-and-may-never-start` | 1 | Awaiting D-FLEET-46 (#2343). |
| `executor:codex-credential-lease-collision-fails-concurrent-dispatch` | 1 | Awaiting D-FLEET-46 (#2348). |
| `executor:codex-credential-collision-drops-automatic-planner-handoffs` | 1 | Awaiting D-FLEET-46 (#2348). |
| `coordinator:job-retry-success-dropped-after-failed-completion` | 1 | Awaiting D-FLEET-46 (#2348). |
| `quorum:claude-p-ration-usage-429` | 1 | Awaiting D-FLEET-46 (addendum, #2344). |
| `eval:motoko-connection-probe-orphaned` | 1 | Other: folded into D-FLEET-18 (ruled, landed). Resolving it is bookkeeping that D-FLEET-46 does not cover. |
| `crash:sprint-executor:sigbus-ram-tmp-full-make-test` | 1 | Out of scope: the remainder is attended or terraform work. |
| `coordinator:cloud-lane-capped-by-local-worktree-limit` | 1 | Out of scope: coordinator code. |
| `ci:distinct-build-test-failures` | 1 | Out of scope: failing tests in `cmd/ailang`, `internal/daemon`, `internal/runner`. |
| `tool-hang:triage-judge:home-metadata-search` | 1 | Parked on OPEN D-FLEET-44. |

8 await D-FLEET-46, 5 are out of scope, 1 is other (no row covers it), 2 are parked on OPEN rows. (Controller correction: the judge's summary line said "3 out of scope"; its own table lists 5 — daneel, pubsub, sigbus, cloud-lane, ci. 8+5+1+2 = 16.) No ticket is a mechanical, in-scope, unparked fix. Only the idle-record ticket changed (7 → 9 slots, `last_seen` 2026-11-20T17:40Z); every other `last_seen` is October or early November.

## Q2. OPEN ledger rows

- **D-FLEET-44**: recommended A is a request for a corrected design and evidence — a document request, not an in-scope code step.
- **D-FLEET-45**: recommendation B (no change), so no code step. Option A begins with one more bounded revision of a design doc that already failed two quorums, plus a retro-analysis; nothing in it ships as code.
- **D-FLEET-46**: A = the fleet resolves each ticket as bookkeeping; no diff, and it is the permission the row asks for.

None of the three recommended options starts with an in-scope code step the loop may ship without a ruling.

## Q3. World 419 and 420

`grep "idle snapshot taken" /tmp/ailang-mission-world.log`:

- **419** (11-20 14:39:42 local): `dae8092b025c9639d6637ead60b69a51ac094f0756708da2df0ce39ba0782fde head=301131e ledger=107 open=2 directives=0 inbox=0 red=0`
- **420** (11-20 18:39:02 local): same full hash, same fields. **Byte-identical: confirmed.**

Would a recorded 419 have let 420 exit at the pre-check? Yes on the checkable inputs: 4 h apart, inside the 24 h expiry; 419's own record commits (`0a8a5191`, `97248e09`, merge `f25e8558`) did not move `head` (420 still reads `301131e`, so the `ff08818ae` `design_docs/verification/` exclusion works). Caveat: whether 419 met every other Gate-5 record condition is UNMEASURED (no lane park or clock wait found).

418 → 419: PR #619 (`48725b7a`, merged 11:27 local) added product commit `301131e7` after 418's 10:39 snapshot (`head=0e9e8d3`, `0355e1e2`). 419 was not skippable at all (its own log: it would have run in full even undegraded), and was itself degraded (codex ration).

Counterfactual over 12 snapshots, 11-19 02:39 → 11-20 18:39 local: `0355e1e2`, `15d7c16b`, `5067a493`, `56c2e062`, `f8eacc0f`, `4d83abf8`, `01aae166`, `0355e1e2`, `0355e1e2`, `0355e1e2`, `dae8092b`, `dae8092b`. Exactly 2 of 12 would have skipped had every eligible no-pick fire recorded: 418 (needs 417's record) and 420 (needs 419's). Entries 1 and 8 share a hash but are ~27 h apart (24 h expiry forces a full fire). Over the 8 fires after 11-20 02:39: still 2 of 8.

## Q4. Does the new evidence change D-FLEET-45?

No. The second identical pair shows two consecutive identical fires both ran in full because of the degraded bar (ticket now 9 slots). It does not touch the two round-2 defects that blocked the design (the proof block ignoring the ledger parser's exit status; no qualifying World fire), nor the unmeasured "only unblock event is a dev-side ledger flip" test — World 419's own log lists rows (150, 151, 263) that unblock through attended work. The saving is ~2 of 12 fires. World measured the codex ration gap at 0.7 points 4 h after 418's "closes in ~3.5 h" estimate (used 22.0% vs allowed 21.6% at 18:38), so degraded fires will likely continue and the bar keeps binding. Recommendation stays B.

## Strongest argument against NONE

The degraded bar was a fleet-controller choice (`33190af48`), not a ruling; the Gate-5 file is in Authority scope; two identical pairs, 9 slots, every World fire degraded. A docs-only draft of the revised design with the retro-analysis would cost little and touch no skill file.

Why NONE holds: the row recommends B and says the ticket stays open until Mark rules; doing the revision acts on A without a ruling; the charter forbids self-sourcing; the design was blocked twice by independent review; expected benefit ~2 skips in 12 fires. The residual 22% is the literal reading in which A's design revision counts as an in-scope step needing no ruling.

## UNMEASURED

- Whether World 419 met every Gate-5 record condition beyond the degraded bar.
- Whether the narrowed rule's "only unblock event is a ledger flip" passes for World 410–420.
- Whether the codex ration gap closes before the next World fire.
- Whether the merged fixes still cover their tickets' repros at HEAD (D-FLEET-46 A's own step).
- World 420's log entry (not yet written; only its snapshot line was used).
