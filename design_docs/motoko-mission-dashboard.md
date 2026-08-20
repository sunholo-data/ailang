# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by Gate 4 (history: charter STATUS + [log](motoko-mission-log.md)).
> **Namespaced** — the bare `mission-dashboard.md` is not ours (V1 iter-216); it holds motoko's stale
> iter-7 snapshot, left alone.

**As of**: 2026-08-20 · iteration **15** · release `v0.33.1` · loop `dev.ailang.mission-motoko`, 12h

## In flight / next
- **Item 6 — fmt re-measurement instrument**: **M1 + M4 of 5 LANDED.**
  M4 (D2 censored-pair analyzer) → PR [#806](https://github.com/sunholo-data/ailang/pull/806) → `d5bcfa0c8`.
  `ailang eval-censored-pairs` ships as a **sibling** of `eval-paired` (whose JSON contract is byte-identical).
  **Resume point: M3 → M5 → M2.** M3 (D1b counterbalanced Wednesday block) is **now unblocked** —
  its AC-M3-4 closes by calling M4's order-integrity checker. M5 depends on M3. **M2 (`AC-D1-live`)
  is the only one needing the rig** (ollama + a metered OpenRouter control leg, ~15 min).
  Plan: `design_docs/planned/m-motoko-fmt-remeasurement-instrument-sprint-plan.md`.
- Then item **7** (profile restoration design), item **8** (repin stale OpenRouter motoko models).

## Queue posture
- Rows 1–5, 5a, 5b, 6b: LANDED/CLOSED. Row 6: M1+M4 landed, M2/M3/M5 open.
- Rows 10/11/12 **Phase-0 gated** — predicates re-run 2026-08-20 as commands, all still FALSE:
  G1 `#154` `state=OPEN`/`mergedAt=-` (control `#161`/`#162` MERGED) · G2 rc=**128** with the
  mandatory `README.md` control rc=**0** · G3 registry `latest=2.2.0`, no 5.x · G4 unrunnable · G5 unchanged.
- Rows 9/13 need a green tree; row 14 follows 13.
- **Upstream**: `arniwesth/motoko_agent#165` (row 5's artifact) is OPEN, labelled by the maintainer
  today, and reported **implemented in their PR #166** (MERGEABLE, base `main_dst`, not yet merged).

## Loop cadence + routing
- 12h `StartInterval`, staggered against V1 (90m) and World (4h).
- Controller `claude:claude-opus-5` · executor **`codex:gpt-5.6-sol`** (the ratified primary; probe
  rc=0 — iteration 14 had fallen to the pi lane, this one did not) · evaluator **sonnet**.
- generator≠judge holds against the executor. **FLAGGED**: the controller-authored repair was
  Anthropic-authored and judged by an Anthropic evaluator — same provider, different model.
- No planner, designer or quorum ran (the plan and the doc's spent quorum already cover M4).
- Designer rotation pointer untouched at `claude:claude-fable-5`.

## Parked on Mark
- **None.** Decision ledger: 3 rows, **0 OPEN** (`scripts/mission_decisions.sh --check`).

## Quota / cost posture
- Metered **$0.00** of $5 this iteration — codex and sonnet are both quota lanes.
- Quota buckets in use: codex (`gpt-5.6-sol`), opus, sonnet.
