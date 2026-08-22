# Mission Dashboard — Motoko

**Last iteration**: 18 · 2026-08-22 · pick = queue row 6 milestone **M2 (`AC-D1-live`)**

## Where the mission is

- **Release**: v0.33.1 (`v0.33.1-201-gb59255831` at iteration start)
- **Current epic**: `m-motoko-dst-refactor-migration` — **Phase-0 gated, still CLOSED**
  (re-measured: G1 `#154` OPEN, G2 rc=128 w/ control rc=0, G3 latest=2.2.0 no 5.x, G4 unrunnable, G5 outstanding)
- **In flight**: [#829](https://github.com/sunholo-data/ailang/pull/829) — M2 instrument
- **Queue row 6** (`m-motoko-fmt-remeasurement-instrument`): M1 + M3 + M4 + M5 LANDED.
  **M2 still the named resume point** — the instrument now exists, its live verdict is VOID.

## This iteration

The M2 connection probe lands and its own live sweep came back **VOID**: both lanes
`driver_rc=1` with empty peer sets, so `AC-M2-control` did not fire and the doc's own rule
voids the treatment verdict. The instrument refused to certify rather than reporting a green.

## Next

1. **Isolate the V38 defect** — the probe as shipped breaks the runs it observes
   (rc=1 / 8m15s), while a faithful replication of its own `run_lane` is rc=0 / 1m1s with
   `127.0.0.1:11434` present. Keep the driver logs (the `trap` deletes them) — that is the
   first fix and it is what made this need a re-run.
2. **Evaluator finding B2** — ~15 refusal branches in the probe's live path have zero
   self-test coverage; a neutered darwin/arm64 gate survives with an identical PASS.
3. Then row 7 (profile restoration design) / row 8 (repin stale OpenRouter motoko models).

## Loop + routing

- Cadence: launchd `dev.ailang.mission-motoko`, 12h
- Controller `claude:claude-opus-5` · executor `codex:gpt-5.6-sol` · evaluator **sonnet**
  (distinct provider → generator≠judge holds) · no designer, no planner, no quorum
- Designer rotation pointer untouched at `claude:claude-fable-5`
- Metered **$0.00** of $5 this iteration. GPU: `rig.lock` taken and released (3 bounded runs)

## Parked on Mark

**None.** Decision ledger valid at 3 rows, **0 OPEN**.
