# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by Gate 4 (history: charter STATUS + [log](motoko-mission-log.md)).
> **Namespaced** — the bare `mission-dashboard.md` is not ours (V1 iter-216); it holds motoko's stale
> iter-7 snapshot, left alone.

**As of**: 2026-08-20 · iteration **14** · release `v0.33.1` · loop `dev.ailang.mission-motoko`, 12h

## In flight / next
- **Item 6 — fmt re-measurement instrument**: **M1 (D1) LANDED** — PR [#794](https://github.com/sunholo-data/ailang/pull/794) → `bc0b5a8d4`.
  The unconditional `OPENROUTER_API_KEY` preflight that killed both Wednesday fmt slots is gone;
  the refusal is now per-task, on the resolved provider, at `ExecuteStreaming`.
  **Resume point: M2–M5 of the same 5-milestone plan** — M2 `AC-D1-live` (needs the rig),
  M3 D1b counterbalanced block, M4 D2 censored-pair analyzer, M5 smoke-bank wiring.
  Plan: `design_docs/planned/m-motoko-fmt-remeasurement-instrument-sprint-plan.md`.
- Then item **7** (profile restoration design), item **8** (repin stale OpenRouter motoko models).

## Queue posture
- Rows 1–5, 5a, 5b, 6b: LANDED/CLOSED. Row 6: M1 landed, M2–M5 open.
- Rows 10/11/12 **Phase-0 gated** — predicates re-run 2026-08-20 as commands, all still FALSE:
  G1 `#154` `state=OPEN`/`mergedAt=-` (control `#161`/`#162` MERGED) · G2 rc=**128** with the
  mandatory `README.md` control rc=**0** · G3 registry `latest=2.2.0`, no 5.x · G4 unrunnable · G5 unchanged.
- Rows 9/13 need a green tree; row 14 follows 13.

## Loop cadence + routing
- 12h `StartInterval`, staggered against V1 (90m) and World (4h).
- Controller `claude:claude-opus-5` · planner **opus** (`derive-planner-lane.sh` → `opus fail-closed:env-pin`)
  · executor `pi:openrouter/deepseek/deepseek-v4-flash-0731` · evaluator **sonnet** (generator≠judge holds).
- Designer rotation pointer untouched at `claude:claude-fable-5` (no designer ran).

## Parked on Mark
- **None.** Decision ledger: 3 rows, **0 OPEN** (`scripts/mission_decisions.sh --check`).

## Quota / cost posture
- Metered **$0.2326** of $5 this iteration (pi executor; probe $0.0001). Quota buckets: opus, sonnet.
- Codex reported exhausted until 2026-08-20 05:34 by both sibling missions; not needed here.
