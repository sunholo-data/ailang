# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by mission-control Gate 4 every iteration (history lives in
> the charter STATUS + [log](motoko-mission-log.md)). Fresh session = THIS + MEMORY.md.
> Humans steer via the bookkeeping issue. **Namespaced**: `mission-dashboard.md` is V1's — three
> missions share this repo, so one file per mission or each overwrites the others.

**Updated**: 2026-08-12 ~11:10 local (iteration 1 — the first iteration that actually ran)

## Where the mission is
- **Charter RATIFIED** (iteration 0). Bar = 6 clauses; clause 6 (motoko graduates into the mission
  executor fleet) is the meta-goal. Epic = [DST refactor migration](planned/m-motoko-dst-refactor-migration.md).
- **Iteration 1 landed queue item 3**: all 51 non-merge fork commits dispositioned —
  [m-motoko-fork-disposition.md](planned/m-motoko-fork-disposition.md) · 14 SUPERSEDED / 16 PORT /
  14 DROP / **7 UNRESOLVED** (each names the measurement that settles it).
- **Iteration 1 = the mission's first real fire.** Iteration 0 was bootstrap; the 09:11 fire died
  on the trust-dialog defect (charter V22).

## Next
- **Queue head is item 4** (output-headroom upstream issue) — but see DECISIONS: item 5 (fmt
  re-measurement) may outrank it now that R8 is a named residual.
- **Owed**: a designer pass on the migration doc for the two live quorum objections (below).

## Blocked / parked
- **Phase-0 gate is REAL and unmet**: `arniwesth/motoko_agent#154` is still **OPEN** (base `main`).
  Queue items 9/10/11 stay parked. An idle iteration here is a correct outcome, not a failure.
- Items 8, 12, 13 parked on a green tree.

## Open with Mark (see bookkeeping issue)
1. Quorum **BLOCKED** the migration doc (both reviewers present, $0.064). Objection 1 (Phase 0 is
   an unbounded wait) is upheld **and its obvious remedy is vacuous** — the ABI's
   `[stability] level = "stable"` reads identical on the 2.2.0 we call unstable. Needs a real
   bounded condition; that is a design call.
2. Controller routed item 3's *analysis* while the doc's *design* is blocked, on the reading that
   evidence-gathering is what the objection asks for. Confirm or correct.

## Loop posture
- Cadence **12h** (`43200s`) this week while quotas are watched — charter text still says 6h.
- Routing: controller opus · designer `claude:claude-fable-5` (rotation) · planner/executor
  `codex:gpt-5.6-sol` · evaluator sonnet. Lanes healthy this fire; codex probe rc=0.
- **Metered spend iteration 1: $0.064** of the $5 ceiling (quorum only; codex is a quota bucket).
