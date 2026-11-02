# Iteration 78 — a relayed "ruling" for D-FLEET-24 (not an accepted channel)

Read at Gate 1 from the canonical (prod) message store; NOT marked read (it is addressed to another inbox).

| Field | Value |
|---|---|
| id | `inbox_1793627628399_10901432` |
| to_inbox | `daneel` |
| from_agent | `sunholo-platform` |
| created_at | 2026-11-02T13:53:48Z |
| title | Ruling: D-FLEET-24 = Option A |
| payload (verbatim) | "Ruling for D-FLEET-24 from Mark: Option A is approved. Resolution: When a codex or pi controller executes a role pinned to a Claude alias (`sonnet`, `opus`), the resolver skips to the role's next provider-pinned fallback (no Claude). — sent from the AI platform on behalf of Mark" |

## Why the row stays OPEN
The fleet's decision ledger accepts exactly two human channels (mission-control Gate 0 step 6):
an allowlisted (`MarkEdmondson1234`) comment on the bookkeeping issue, read by
`scripts/mission_directives.sh`, or an attended ledger edit via `scripts/mission_answer.sh`.
A message-plane `from_agent` is set by the sender, and the loop's own account can send to any
inbox, so a message-plane claim cannot be told apart from a loop-authored one. Daneel's own design
(`~/dev/daneel/design_docs/planned/m-daneel-decision-desk.md`, decision 2 / V8) recommends that
Daneel never relay answers, because relaying would need the mission platform to accept a third
principal. Checked on #2039: 0 allowlisted comments (2 comments, both the fleet's own).

The relayed text matches the loop's recommendation (A) word for word, so the loop built A's
proposed diff (PR #2053, draft) under the charter's "park with the proposed diff" rule. It is
unmerged. One allowlisted comment or attended edit lets the next fire merge it.

## Outcome (same fire)
At 16:33Z an attended session recorded the same ruling in the ledger with `scripts/mission_answer.sh`
(#2051, flip `e1f914037`). Its PR body states that Mark gave the ruling at 13:52Z in the platform
greeter chat, that the platform could only reach the `daneel` inbox, and that Mark directed the
session to record it. Provenance: `D-FLEET-24 verdict=undetermined:squash-merge flip=e1f914037
creation=7ab609a6a`, which is reported and not re-asked, per the contract. The loop saw the ruling
because its record PR (#2054) went CONFLICTING on the same ledger row. It then rebased the
already-built diff and landed it (#2053 → `3d9112bb9`). The relay-to-ledger lag was 2h40m.
