### Added — design doc: M-COORDINATOR-AUTO-APPROVAL-POLICY (Part A of M-TRIAGE-MISSION)

- `design_docs/planned/m-coordinator-auto-approval-policy.md` specifies how the coordinator would
  approve its own cards without a human click.
  - The decision is a typed AILANG module (`auto_approval_policy.ail`, per D-TRIAGE-8). It takes a
    record of facts gathered in Go and returns `Approve(approver) | Hold(Clause, reason)` over
    clauses P0–P8. A draft of the module type-checks and passes 25/25 inline tests.
  - Approvals dispatch only after the merge. A new card state, `approved_awaiting_merge`, comes
    first, then a REST squash merge pinned to the head SHA. The landed-card sweep fires the
    handoffs once the merge exists.
  - The authority check is enforced in warn-then-enforce stages, with the dashboard path left
    untouched.
  - Policy approvals carry an audit string, `policy:<v>+attestation:<judge>`.
  - The attestation is a signed PR comment, bound to the PR's head SHA (D-TRIAGE-9).
  - Rollout runs in shadow mode first. A kill switch is added, and P8 turns auto-approval off by
    itself, failing closed.
- The doc corrects several premises in the parent triage design:
  - the coordinator image has no `.ail` source and no `gh`;
  - the authority check refuses outside a repo checkout even when the attended grant is set
    (measured), so enforcing it as written would recreate the 2026-09 lockout;
  - authorship by the bot account cannot authenticate an attestation, because executors hold the
    same token;
  - a dashboard approval of a cloud task never merges its PR (found by reading the code).
- Design quorum: round 0 was blocked, and the P8 objection was accepted and fixed. Round 1
  proceeded. Only one off-Anthropic reviewer was reachable; the others failed on quota or credit.
