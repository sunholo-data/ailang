# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by Gate 4 each iteration (history lives in the charter STATUS
> + [log](motoko-mission-log.md)). **Namespaced** — the bare `mission-dashboard.md` is not ours to
> write (V1 iter-216 rule); it holds motoko's stale iter-7 snapshot, left alone.

**Updated**: 2026-08-18 ~15:00 local (iteration 11)

## Where the mission is
- **Charter RATIFIED** (iter 0). 6 clauses; clause 6 (motoko graduates into the mission executor
  fleet) is the meta-goal. Epic = [DST migration](planned/m-motoko-dst-refactor-migration.md).
- **Iter 1–10**: 51 fork commits dispositioned ([ledger](planned/m-motoko-fork-disposition.md) — 14
  SUPERSEDED / 17 PORT / 14 DROP / 6 UNRESOLVED); Phase 0 fail-closed on **G5**; **R8 → PORT**;
  **#721**/**#728** fixed the loop's own health; the −74% is really **−5.7%** on the honest pairs;
  auto-merge is not a landing mechanism.
- **Iter 11**: **item 5 LANDED — Arni answered #97 five days ago and the charter never noticed.** He
  agreed on all four points, asked us to close #97 as superseded, and **explicitly invited** the
  output-headroom issue. Filed as upstream
  [**#165**](https://github.com/arniwesth/motoko_agent/issues/165); #97 closed with the verdict.

## Next
- **D-MOTOKO-FMT-1 is the only thing gating item 6** — one word from Mark (below).
- Untagged queue head is now item **7** (profile restoration design), then item **8** (repin the
  stale OpenRouter motoko models). Both ungated, both ~1 iteration.
- Upstream **#165** is Arni's to triage; no local work waits on it. Item 5 is CLOSED, not waiting.

## Blocked / parked
- **Item 6 PARKED** `needs-human-review` — [the instrument](planned/m-motoko-fmt-remeasurement-instrument.md)
  is written, twice-reviewed, **direction undisputed in both rounds**.
- **Blocks any fmt run regardless**: the Wednesday A/B lane has banked nothing since AC5 — both fires
  died at `internal/executor/motoko/healthcheck.go:64`, an unconditional `OPENROUTER_API_KEY` refusal,
  while both arms are local ollama.
- **Phase-0 gate REAL and unmet** (G1 `#154` OPEN, G2/G3 unmet): 10/11/12 parked; 9/13/14 need a green tree.

## Open with Mark (issue #743) — one decision, one word
- **D-MOTOKO-FMT-1**: is tracing motoko's *resolved runtime provider* a **precondition** of D1, or
  does D1 need a **redesign** leaving the preflight alone? *(precondition / redesign)*

## Loop posture
- Cadence **12h**; issue **#743**. Metered iter 11 **$0.00** of $5; no GPU/`rig.lock`;
  `make quick-install` skipped deliberately (shared-write guardrail, V20).
- Controller opus only — **no designer/planner/executor/evaluator/quorum spawned**. Designer pointer
  untouched at `claude:claude-fable-5`; **codex lane quota-dry until Aug 20**.
