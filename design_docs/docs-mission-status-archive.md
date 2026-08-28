# Docs Mission — STATUS archive

Append-only. Newest entry at the TOP (Gate 4 moves the 4th-newest charter stamp here, per
`docs-mission.md`'s rotation rule). The charter itself always carries the newest 3.

## STATUS 2026-08-28 — ITERATION 0 PENDING: charter written, **not yet ratified**, loop armed-but-silent

Charter drafted attended with Mark 2026-08-28. Bar clauses 1-7 are Mark's own selection and must
still be **ratified through the design quorum at iteration 0** before any sprint routes.

**ARMED-BUT-SILENT as of 2026-08-28 07:59.** `dev.ailang.mission-docs` is installed and
bootstrapped; the kill switch `~/.ailang/state/mission-docs.disabled` is set, so every fire exits at
Gate 0 until it is removed deliberately. The whole chain is proven live at **zero token cost**:
launchd fired the job, the plist's `MISSION_PROFILE=docs` resolved, the source clone was correctly
`ailang-docs` (so `WorkingDirectory` took effect), the driver pin advanced to latest `origin/dev` at
0 behind, the kill switch caught it, exit 0. **Iteration 0 is charter ratification, run ATTENDED** —
it is not a sprint.

**Two infrastructure defects were found and fixed *before* the loop ever ran**, both in
`derive-planner-lane.sh`, and both of the same shape: a cheap pin that reads as configured in the
driver log while an expensive model actually runs.

1. **Fail-closed to opus on every docs item.** The path allowlist was infra-only, so every docs
   design doc emitted `opus fail-closed:path-not-in-codex-allowlist` — routing the planner to the
   most expensive model in the fleet, on the mission built to avoid it. Fixed as per-mission data
   (`MISSION_PLANNER_ALLOWLIST`; default unchanged, so v1/world/motoko are byte-for-byte
   unaffected).
2. **The emitted lane dropped its model.** Step 5 emitted a hardcoded bare `codex` for any `codex:*`
   pin. Invisible on V1 by coincidence — its pin is `gpt-5.6-sol` and the consumer default is also
   sol, so the dropped value equalled the fallback. Not invisible here: this mission pins
   `gpt-5.6-luna` ($0.20/$1.20 per M) and would have planned on `gpt-5.6-sol` ($2/$10) every
   iteration. Found end-to-end, by routing a real docs doc through the *pinned* driver with the live
   mission env — not by reading the script.

Both measured with discriminating controls. `tools/launchd/test_mission_routing.sh` is at **34/34**,
with 9 new assertions covering both directions of each risk (widening must work and must not become
a hole; the lane must keep its model, asserted with two different codex models so a re-hardcoded
literal cannot satisfy both).
