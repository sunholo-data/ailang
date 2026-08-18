# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by Gate 4 (history: charter STATUS + [log](motoko-mission-log.md)).
> **Namespaced** — the bare `mission-dashboard.md` is not ours (V1 iter-216); it holds motoko's stale
> iter-7 snapshot, left alone.

**Updated**: 2026-08-18 ~21:00 local (iteration 12)

## Where the mission is
- **Charter RATIFIED** (iter 0). 6 clauses; clause 6 (motoko graduates into the mission executor
  fleet) is the meta-goal. Epic = [DST migration](planned/m-motoko-dst-refactor-migration.md).
- **Iter 1–10**: 51 fork commits dispositioned ([ledger](planned/m-motoko-fork-disposition.md) — 14
  SUPERSEDED/17 PORT/14 DROP/6 UNRESOLVED); Phase 0 fail-closed on **G5**; **R8 → PORT**; `#721`/`#728`
  fixed the loop's own health; the −74% is really **−5.7%**; auto-merge is not a landing mechanism.
- **Iter 11**: item 5 LANDED — Arni had answered `#97` five days earlier and the charter never noticed.
  Output-headroom case filed as [**#165**](https://github.com/arniwesth/motoko_agent/issues/165); `#97` closed as superseded.
- **Iter 12**: **`#165` re-anchored — `main_dst` moved 59 commits within six hours of filing.** Defect
  intact (window sha256-identical), its three `session.ail` citations shift **+54**; corrected
  upstream. Arni has cut `arniwesth/mot-100-fix-output-headroom`.

## Next
- Untagged head is item **7** (profile restoration design) — ungated, ~1 iteration, the pick for
  iteration 13 unless a predicate flips; then item **8** (repin the stale OpenRouter pins).
  **D-MOTOKO-FMT-1 is the only thing gating item 6.** `#165` is Arni's to triage, nothing waits on it.

## Blocked / parked
- **Item 6 PARKED** `needs-human-review` — [the instrument](planned/m-motoko-fmt-remeasurement-instrument.md)
  is written, twice-reviewed, **direction undisputed**. Any fmt run is blocked anyway: both Wednesday
  A/B fires died at `healthcheck.go:64`, an unconditional `OPENROUTER_API_KEY` refusal, while both
  arms are local ollama.
- **Phase-0 gate REAL and unmet, re-measured iter 12**: G1 `#154` OPEN (control `#161`/`#162` MERGED),
  G2 rc=128 / control rc=0, G3 registry `latest=2.2.0` (no 5.x), G4 unrunnable, G5 unchanged →
  10/11/12 parked; 9/13/14 need a green tree. `#154` is moving (`updatedAt` 08-18 16:48Z).

## Open with Mark (issue #743) — one decision, one word
- **D-MOTOKO-FMT-1**: is tracing motoko's *resolved runtime provider* a **precondition** of D1, or does
  D1 need a **redesign** leaving the preflight alone? *(precondition / redesign)*

## Loop posture
- Cadence **12h**; issue **#743**. Metered iter 12 **$0.00** of $5; no GPU/`rig.lock`; controller opus
  only — no roles spawned. Designer pointer at `claude:claude-fable-5`; codex quota-dry until Aug 20.
