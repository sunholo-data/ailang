# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by Gate 4 each iteration (history lives in the charter STATUS
> + [log](motoko-mission-log.md)). **Namespaced** — the bare `mission-dashboard.md` is not ours to
> write (V1 iter-216 rule); it holds motoko's stale iter-7 snapshot, left alone.

**Updated**: 2026-08-18 ~10:00 local (iteration 10)

## Where the mission is
- **Charter RATIFIED** (iter 0). 6 clauses; clause 6 (motoko graduates into the mission executor
  fleet) is the meta-goal. Epic = [DST migration](planned/m-motoko-dst-refactor-migration.md).
- **Iter 1–7**: 51 fork commits dispositioned ([ledger](planned/m-motoko-fork-disposition.md) — 14
  SUPERSEDED / 17 PORT / 14 DROP / 6 UNRESOLVED); Phase 0 fail-closed on **G5** (Arni's word);
  **R8 → PORT**; refusal epidemic traced to `session_start.sh` (**#721**); driver gate fixed (**#728**).
- **Iter 8**: the −74% we were about to re-prove is **one benchmark and one void pair** — on the four
  honest pairs it is **−5.7%**. Instrument designed, parked. **Iter 9**: dev red; motoko fixed it,
  found V1 four minutes ahead on the identical fix, **stood down**. Its record never landed.
- **Iter 10**: recovered that record. **Auto-merge is not a landing mechanism** — #760 sat `BLOCKED`
  12h on a base-inherited red that #759 fixed 26 min later; auto-merge never re-runs a PR's checks
  when the base moves. Silent loss: no timeout, no red, no failed command.

## Next
- **D-MOTOKO-FMT-1 is the only thing gating item 6** — one word from Mark (below).
- Row **6b CLOSED** as a measurement: motoko owns **0 of 15** orphaned issues (11 labelled
  AILANG-lane and already in V1's charter; `#687` is V1's declared next pick; 3 have closed).
- Untagged head is item **7** (profile restoration); item 5 stays bounded until **2026-08-27**.

## Blocked / parked
- **Item 6 PARKED** `needs-human-review` — [the instrument](planned/m-motoko-fmt-remeasurement-instrument.md)
  is written, twice-reviewed, **direction undisputed in both rounds**.
- **Blocks any fmt run regardless**: the Wednesday A/B lane has banked nothing since AC5 — both fires
  died at `internal/executor/motoko/healthcheck.go:64`, an unconditional `OPENROUTER_API_KEY` refusal,
  while both arms are local ollama.
- **Phase-0 gate REAL and unmet** (G1 `#154` OPEN, G2/G3 unmet): items 10/11/12 parked; 9/13/14 need
  a green tree.

## Open with Mark (issue #743) — one decision, one word
- **D-MOTOKO-FMT-1**: is tracing motoko's *resolved runtime provider* a **precondition** of D1, or
  does D1 need a **redesign** leaving the preflight alone? *(precondition / redesign)*

## Loop posture
- Cadence **12h**; issue **#743**. Metered iter 10 **$0.00** of $5; no GPU/`rig.lock`;
  `make quick-install` skipped deliberately (shared-write guardrail, V20).
- Controller opus only — **no designer/planner/executor/evaluator/quorum spawned**. Designer pointer
  untouched at `claude:claude-fable-5`; **codex lane quota-dry until Aug 20**.
