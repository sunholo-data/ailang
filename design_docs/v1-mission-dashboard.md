# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.*

**Last iteration:** 288 · 2026-08-27 · controller `opus`

## Release
- **v0.34.0** shipped 2026-08-26. Next planned bucket: `v0_35_0`.

## In flight / next
1. **`m-fmt-measurement-att-isolation-unpinned` + `m-fmt-measurementerr-propagation-no-killer` —
   LANDED (PR #926), evaluator PASS 92/100 at round 2 after a round-1 FAIL.** Three unpinned hunks in
   `internal/format` now pinned or explicitly declared. Production diff is **comments only**;
   corpus-neutral (405 identical / 0 divergent / 45 fail-closed of 450, control fires at 21).
2. **NEXT: `m-fmt-printer-line-width-limit` M2 (continuation layout) then M3 (corpus reformat).**
   M3 first would bank lines M2 would have wrapped. Residual ~116 corpus lines >120 runes.
   Both new tests carry re-check notes telling M2 to re-run the differential.
3. **NEW `m-fmt-att-isolation-test-diagnostic-narrowing`** (round-1 judge, non-blocking N1) — the
   isolation test loops a map with `t.Fatalf`, so a red run reports only the first fixture visited
   (measured 19/20 vs 1/20 across 20 runs). Both fixtures independently kill the mutant; the gap is
   diagnostic, not correctness. `t.Run` + anchored grep fixes it.
4. **`m-feedback-dispatch-workspace-path`** — feedback still does not route (`chdir /workspace/...`).

## Loop cadence + routing
- controller `opus` · designer rotation `claude-fable-5` -> `pi:ollama/kimi-k3:cloud` · planner lane
  derived VERBATIM (`opus fail-closed:planner-lane-field-missing`) · executor `codex:gpt-5.6-sol`
  · evaluator `sonnet`. generator≠judge held (OpenAI wrote, Anthropic judged).
- **Fable diet UNSPENT** this iteration (no new design doc was needed). metered **$0.00** of $5.

## Parked on Mark
- **`D-41`** (only OPEN ledger row) — may an ACTIVE prompt version be edited in place, or must a
  content change bump the version? Bears on eval-baseline reproducibility.
- **`D-42` (NEW)** — standing authorization to reconcile this checkout to `origin/dev` unattended?
  Now has a **measured** cost, not a predicted one: motoko's driver-pin fix `ff0da7445` is not an
  ancestor of local HEAD, so V1's driver kept running unpinned this fire.

## Quota posture
- Anthropic available (`MISSION_ANTHROPIC_AVAILABLE=1`); codex probe rc=0. SonarCloud standing red
  (no token on this rig), non-required, inherited across ≥6 commits — named, not this iteration's pick.
