# V1 Mission Dashboard — snapshot, OVERWRITTEN each iteration (history lives in the charter + log)

**As of:** 2026-08-27 · **iteration 289** · controller `opus`

## Latest
- **Release:** v0.34.0 · `dev` HEAD `3ee848bb0`
- **Just landed:** **M2 — continuation layout for long non-chain equation bodies** (PR #928).
  Evaluator PASS 90/100, zero blocking. All 4 required contexts green before merge.

## In flight / next
- **NEXT: M3 — the second corpus reformat**, the LAST milestone of
  `design_docs/planned/v0_35_0/m-fmt-printer-line-width-limit.md`.
  Needs iteration 282's evidence discipline in full: poisoned comment-count control that FIRES,
  per-file `ailang check` rc joins, two-armed gates, the residual width table.
  When it lands, the doc moves to `implemented/` **with its sprint plan**.
- Ordering was load-bearing and is now discharged: M2 before M3, so M3 cannot bank lines M2 wraps.

## Loop cadence + routing
- controller `opus` · designer ROTATION (`claude:claude-fable-5` → `pi:ollama/kimi-k3:cloud`)
  · planner `codex:gpt-5.6-sol` · executor `codex:gpt-5.6-sol` · evaluator `sonnet`
- generator≠judge held this iteration (OpenAI wrote, Anthropic judged).
- **Fable diet UNSPENT** for the 4th consecutive iteration — doc and plan both already existed.

## Parked on Mark
- **`D-41` (the only OPEN row of 41):** may an ACTIVE prompt version be edited in place, or must a
  fix bump the version? Bears on eval-baseline reproducibility. Options (a) active-is-mutable,
  (b) immutable-always, (c) mutable-until-first-banked-use.

## Known reds / posture
- **SonarCloud red, INHERITED ≥6 consecutive commits**, non-required (UNSTABLE ≠ BLOCKED). Named
  each iteration, never the pick. No Sonar token on this rig.
- **Driver ran UNPINNED for the 5th consecutive fire** (`MISSION_WORKDIR`/`AILANG_DRIVER_SRC` unset).
- Local `dev` clone runs **6 behind origin**; reconcile obligations all measured as satisfiable,
  but standing authorisation is a human call, so the loop routes around origin instead.
- metered spend this iteration **$0.00** of $5.
