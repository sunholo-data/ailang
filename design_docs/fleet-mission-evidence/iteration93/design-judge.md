# Design Judge Report — D-FLEET-25 fleet-wide rulings ledger (proposal review)

**Artifact**: `design_docs/planned/m-fleet-rulings-ledger.md` — SHA256 `bad5f6a2f40a1f6bdcca2495be6b4b92c0e31386d61822e4e2de7341375e1eac`, 698 lines (hash matches the frozen copy; worktree pin `a3bb8629106a4b8459df5aa33f867b4ed4dd84a5`).
**Effective model**: openrouter/minimax/minimax-m3.
**Method**: design + directly relevant primary source only. No quorum/sibling/judge/prior-verdict artifacts read; no code executed; no mutation runs (none exist).

## Verdict: NEEDS-HUMAN-REVIEW

Not a rejection, not a pass. The mechanism is specified well enough to build and the artifact ships inert (seed row OPEN, no claims active). But the artifact's central authority guarantee is stated more strongly than the code it consumes supports, and the human decision that closes that gap (Design Freeze item 2) has not been made. That is an unresolved human prerequisite, so I cannot certify the proposal "adequate as proposed design" — and I find no defect that would force a redesign.

## What the proposal does adequately

- **Mechanism**: two-verb claim DSL, exact byte canonicalization (`:281-290`), scope→corpus mapping, bounds, deterministic ordering, typed finding vocabulary (`:190-250`).
- **Fail-closed**: seven provenance/authorization verdict arms, each a typed finding, none enforcing (`:292-299`).
- **Approval boundaries**: five-item Design Freeze + Ratification Map separating "A = build the mechanism" from ratification/activation (`:109-135`); no auto-resolve, no cross-repo writes.
- **Validation**: mutation matrix M1–M21 with non-vacuity killers and snapshot-seeded fixtures.
- **Anchors (spot-checked, all confirmed)**: `doctor.go:281`, `mission_cmd.go:236`, `doctor_test.go:32`, `registry.go:90/251`, `internal/gitexec` `CommandContext`, `std/VERSION` v0.57.4, `missions/world.toml:3/:5` literals, `mission_decisions.sh` 4-field awk, provenance verdict vocabulary, `mission-lane-check.sh:49`, `test_mission_lane_check.sh:80`.

## Blockers (resolve before this proposal is called adequate; none forces a redesign)

**B1 — the "attended" authority root is satisfiable by the unattended loop, yet the doc says it is not.** The binding requires the helper's `attended` verdict (`:266-280`, table `:292-299`). `scripts/mission_ledger_provenance.sh:118` derives `attended` from the flip commit's author name not matching the fleet pattern — nothing more. `scripts/mission_answer.sh:107-111` stamps `ATT_NAME=Mark Edmondson` / Mark's noreply email as a **default for any caller**, refusing only identities that look like the fleet bot; its own header (`:44-48`) records this as "KNOWN WEAKNESS … The real control is the charter rule, not this code", and no invocation attestation exists (`:78-111`). So an unattended `mission_answer.sh --id D-FR-N --file design_docs/fleet-rulings.md --commit` that resolves a row and adds claims in the same commit satisfies requirements 1–4 and makes those claims enforceable fleet-wide. That contradicts `:81` ("no unattended edit can activate, alter or extend enforcement") and the A4 justification at `:35`. Byte-binding does correctly stop *post-hoc* mutation of an already-resolved row; the uncovered path is unattended *first* resolution. §Honest limits (`:313-318`) discloses only the generic "conventions, not authentication" caveat, not this one. **Human decision required**: accept convention-level provenance as the authority root (and strike the overclaim), or extend the binding to pin the flip author identity — a helper-touching change the doc itself routes to Mark.

**B2 — the cited provenance artifact does not exist.** The header (`:12-21`) cites `design_docs/fleet-mission-evidence/iteration93/pick-judge.md`, "MiniMax, SHA a3bb86291". At this pin `design_docs/fleet-mission-evidence/iteration93/` does not exist; `git log --all` shows no such path in any ref; the production worktree's `iteration93/` directory is empty. The quoted "SHA" is the branch-tip commit SHA, not a file hash. V14's corroboration ("the pick-judge's 8-draft-head inspection") is therefore unverifiable. Repair the citation or drop the BUILD-eligibility claim.

**B3 — no supersession semantics between fleet rulings.** No `supersede`/precedence rule exists (zero grep hits). Two RESOLVED rows whose claims conflict (e.g. forbid-text X vs require-text X) yield contradictory findings with no tie-break; "a fleet ruling outranks charter prose" (`:346-355`) does not order rulings against each other. Also `:305-312` says a revision "requires a NEW OPEN ID", but the helper attributes the *latest* flip, so a reopen/re-resolve of the same ID would bind a new claimset — that rule is advisory, not enforced. Define precedence before a second ruling exists.

**B4 — missing ledger is Drift, so foreign registries stay permanently red.** `:249-250` makes an absent ledger a `rulings-ledger-missing` Drift finding, and the CLI resolves the path unconditionally from `Mission.Root()` (`registry.go:251`); any registry without `design_docs/fleet-rulings.md` — which the design explicitly supports — reports Drift and exits 1 forever. Absent-ledger severity should be non-Drift while no claim is active.

## Non-blocking refinements

1. `review_packet.go:180` is a hardened `diff` invocation (`-c core.hooksPath=/dev/null …`), not the "exact pattern … `show`" claimed at `:274`; cite it only as precedent for gitexec-only git reads.
2. No interface contract with the planned `sprint-plan-gate0-ledger-provenance.md`, which may alter the very helper whose verdict vocabulary is consumed; add a compatibility condition.
3. The 5 in-repo charter link lines fall inside the fleet charter's own write scope (`design_docs/**`, `fleet-mission.md:79-84`) yet are framed as attended edits — clarify who may add them.
4. `except` is line-scoped only; no guidance for multi-line historical quotations.

## Approval boundaries (as read)

An "A" authorizes building only. Separately attended: the bounded-literal narrowing, the binding semantics, D-FR-1 resolution (activation), and charter links. Cross-repo charters are read-only with attended activation only. Nothing here authorizes resolution, activation, merging, or implementation.

## Strongest objection to my own outcome

The doc discloses convention-level authority and routes trust-tightening to Mark (`:313-318`, Freeze item 2); one could fairly say I penalized a *disclosed* limitation rather than a defect, and that the human decision is already reserved — so NEEDS-HUMAN-REVIEW is close to PASS. My counter: the reservation concerns *how strong* the convention is, whereas B1 is a factual claim about what the mechanism prevents (`:35`, `:81`); a reviewer deciding the freeze would be misled. If Mark accepts convention-level authority, B1 reduces to striking two sentences and the outcome flips to PASS-with-conditions.

## Limitations

No code executed; no tests or mutation runs (no implementation under review). Read the design plus directly relevant primary source: `doctor.go`, `mission_cmd.go`, `registry.go`, `internal/gitexec`, the three `scripts/mission_*.sh` helpers, `idle-fingerprint.sh`, `mission-lane-check.sh`, `missions/world.toml`, `fleet-mission.md` (D-FLEET-25 row, Authority, Guardrails). Did not read quorum/sibling/prior-verdict artifacts, credentials, inbox, or network; proposal and source left unmodified.
