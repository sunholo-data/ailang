# Correction: cp_* timeout rows — duration_ms=0 is pi/opencode-only, and the tier question is time-budget, not capability

- **Date**: 2026-11-03
- **Class**: bug (narrowed scope, superseding part of the original report `inbox_1793718624754_c0b782ef`)
- **Recommend**: design-doc
- **Searched**: `duration_ms` in design_docs/ (hits are archive eval docs, none covering the timeout-row banking fix); `mis-tier`, `long-frontier` (hits: `m-mission-role-elo-and-tier-order.md`, `m-harness-mission-loop.md`, `v1-mission.md` — tier-ordering policy docs, not this defect); `cp_60166|cp_60159|cp_60149` (hits: `planned/v0_55_0/m-eval-ailang-advantage-program.md`, `planned/v0_55_0/m-eval-hidden-tests.md`, `changelogs/unreleased/2026-11-03-os-filler-binary-pickup.md`); `hard timeout` / `ResolvedHardTimeoutSecs` in internal/eval_harness. No existing doc commits to banking real elapsed duration on executor-hard-timeout rows — none found.
- **Estimate**: omitted (design-doc)

## What the correction changes

Two refinements to the original rig report, both of which change what should happen:

1. **The instrument defect is narrower than reported.** Not "every timeout row banks
   `duration_ms=0"` — only **pi and opencode** rows that hit the executor's 1-hour hard
   timeout bank 0; **motoko** timeout rows bank the real ~3,600,000 ms. So the fix target is
   the pi/opencode result-banking path under a hard kill, not the row schema universally.
   The symptom and the current workaround are already documented
   (`changelogs/unreleased/2026-11-03-os-filler-binary-pickup.md`: "timeouts bank
   `duration_ms=0`, so they are listed separately"; `tools/rig-value-report.sh:77-79` computes
   banked wall-hours excluding them), but the workaround is reporting-side only — the rows
   themselves still lose the most expensive runs' hours.

2. **The mis-tiering question is answered: do not re-tier.** The timeouts are the 1-hour
   **agent wall clock**, not the benchmark's 30s `exec_timeout` on hidden tests (all three
   specs carry `timeout: 600` / `exec_timeout: 30` — `benchmarks/cp_60149.yml:13-14` and
   siblings). pi **solved** cp_60149 in 53 minutes (34/34 hidden tests) inside its hour, and
   motoko reached only 2–5 turns in the same budget. On qwen3.8:27b at local speed the CP
   pack is bound by the time budget, not by capability — consistent with what
   `planned/v0_55_0/m-eval-ailang-advantage-program.md` already measured for strong models
   ("Strong-model AILANG failures are mostly budget, not wrong code", and its note that the
   pack's `timeout: 600` overrides `--agent-timeout`). Zero-pass on these three should be
   read as budget exhaustion at local model speed, not as evidence they sit in the wrong
   tier — the park-and-probe policy in the 2026-11-03 filler changelog stands.

## Why design-doc and not direct-fix

The banking fix sounds small but the kill path is the question: when the executor's 1h hard
timeout fires, the pi/opencode child is killed before it can emit its result record, so
someone must bank wall-clock-since-start at kill time — and there is more than one
acceptable place and shape for that (the harness kill path in `internal/eval_harness`
(`agent_task.go`, `ResolvedHardTimeoutSecs`); the runner that assembles the row; or a
timestamp-pair carried in the result envelope). Per the rubric, multiple acceptable
remedies is design-doc territory, and I could not locate the exact banking site for a
confident line estimate within this triage. The design doc should also decide whether the
motoko path's behaviour (banking ~3,600,000 on a kill) is the contract the other two
harnesses should match, and whether `rig-value-report.sh`'s separate-listing workaround
retires once rows carry real durations.

Note: the rotation rows themselves (`eval_results/rotation/os-rolling/v0.54.18..v0.55.0`)
live on the rig, not in this checkout, so the 53-minute / 2–5-turn figures are taken from
the reporter and were not independently re-verified here.
