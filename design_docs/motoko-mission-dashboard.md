# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by mission-control Gate 4 each iteration (history lives in
> the charter STATUS + [log](motoko-mission-log.md)). Fresh session = THIS + MEMORY.md. Humans
> steer via the bookkeeping issue. **Namespaced** — `mission-dashboard.md` is V1's.

**Updated**: 2026-08-12 ~12:15 local (iteration 2)

## Where the mission is
- **Charter RATIFIED** (iter 0). Bar = 6 clauses; clause 6 (motoko graduates into the mission
  executor fleet) is the meta-goal. Epic = [DST refactor migration](planned/m-motoko-dst-refactor-migration.md).
- **Iter 1** landed item 3: 51 fork commits dispositioned
  ([ledger](planned/m-motoko-fork-disposition.md)) · 14 SUPERSEDED / 16 PORT / 14 DROP /
  **7 UNRESOLVED**. Gate 3b GREEN.
- **Iter 2** (`1d0e2e511`) made **Phase 0 a bounded fail-closed gate** — 4 conjunctive predicates
  with commands + observed values, a 28-fire (~14d) timebox, a structured BLOCKED expiry, a
  declared human residual — and answered R1's other objection with rows V21–V24.

## Next
- **Queue head is item 5** (output-headroom upstream issue) — ungated; the disposition's `R8`
  UNRESOLVED row now gives it a concrete instrument to cite.
- If Mark answers **D1**, item 4 unparks and outranks item 5 (one edit, no further paid round).

## Blocked / parked
- **Phase-0 gate REAL and unmet, measured today**: G1 `#154` **OPEN** (control: `#150–152` MERGED),
  G2 `packages/` absent from motoko_agent's `origin/main`, G3 registry has only `1.0.0–2.2.0`, no
  `5.x` (control: 4-entry list). Items 10/11/12 parked. An idle iteration here is correct.
- **Item 4 parked `needs-human-review`** on D1 — the one re-quorum is spent. Nothing is blocked:
  Phase 0 is closed, so no sprint existed to hold. Items 9, 13, 14 parked on a green tree.

## Open with Mark (see bookkeeping issue #663)
1. **D1 (new, blocking item 4)** — Is Arni's explicit ABI-settled acknowledgement a **gate
   predicate (G5)** or an **accepted risk**? The ratified charter guardrail says the port waits for
   it; the doc's *Declared residual* starts Phase 1 on G1–G4 alone. The two disagree. (Branch A also
   drags in item 11, itself Phase-0 gated — a dependency loop worth your eyes.)
2. **Carried from iter 1, unanswered**: (a) routing item 3's *analysis* while its *design* was
   quorum-blocked — too loose? (b) keep the namespaced `motoko-mission-dashboard.md`?

## Loop posture
- Cadence **12h** (`43200s`) this week while quotas are watched.
- Routing: controller opus · designer `claude:claude-fable-5` (rotation start; probe rc=0) ·
  planner/executor `codex:gpt-5.6-sol` · evaluator sonnet. **No lane degradation this fire.**
- **Metered iter 2: $0.096** of $5 (quorum R2 only). Both reviewers present both rounds.
