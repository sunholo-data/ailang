# Sprint Plan: M-DEONTIC-PKG — sunholo/deontic extension package

**Sprint ID**: M-DEONTIC-PKG
**Design doc**: [m-contracts-as-code-vertical.md](m-contracts-as-code-vertical.md) (extension-routing variant — user-prioritized: ties into Aitana work)
**Repo**: `ailang-packages` (NOT the ailang core repo — routing decision: examples now, package on demand, core never; demand arrived, so package now)
**Duration**: 1 day (campaign velocity: wave-5 engine was written and cross-verified twice in ~2h)
**Risk**: low-medium (engine logic proven; risks are package-system conventions and Z3 coverage of generalized functions)

## Goal

Ship `sunholo/deontic` v0.1.0: a reusable, pure, Z3-contracted deontic reasoning
engine — obligations, notice-and-cure, waiver forgiveness, force-majeure
extension, amendment, termination cascade — generalized from the wave-5
`legal_obligation_engine` reference (byte-identical ground truth already
cross-validated against Python twice).

## Milestones

### M1 — Package scaffold + generalized core (est. 260 LOC)
- `packages/deontic/ailang.toml` — sunholo/deontic 0.1.0, `[effects] max = []`
  (the engine is PURE; consumers do their own IO), stability experimental,
  cascade budget default.
- `packages/deontic/core.ail` — generalization of the wave-5 engine:
  - policy knobs as a record (`Policy`: penalty per day/cap, pay-within, cure
    days, interest rate/period) instead of hardcoded constants
  - obligations as data (assoc lists keyed by obligation id) instead of
    hardcoded M1/M2/M3
  - `Event` ADT (Deliver/Pay/AmendPrice/ForceMajeure/Notice/Waive/Terminate),
    `Term` ADT, pure `applyEvent` + `runEvents` fold
  - pure report builders returning `[string]` (no IO in the package)
- `packages/deontic/AGENT.md` per repo convention.
- Acceptance: `ailang check` green on all modules; exports match manifest.

### M2 — Z3-contracted settlement module (est. 100 LOC)
- `packages/deontic/settle.ail` — `capAt`, `floorPeriods`, `interestFor`,
  `penaltyFor`, `netOf` with `requires`/`ensures` on each.
- Acceptance: `ailang verify` reports VERIFIED for every contracted function
  (transcript captured in AGENT.md/README).

### M3 — Ground-truth demo + tests (est. 140 LOC)
- `packages/deontic/tests/wave5_demo.ail` — reproduces the wave-5 timeline via
  the GENERALIZED engine; output must be byte-identical to the 16-line
  expected_stdout of `legal_obligation_engine` (shared provenance = the test).
- Investigate repo test conventions (testing-utils package / Makefile) and
  wire the demo into whatever gate exists; minimum: documented runnable check.
- Acceptance: demo output diffs clean against the known-good 16 lines.

### M4 — README + registry wiring (est. 160 LOC docs)
- `packages/deontic/README.md`: what/why, the four moats, usage snippet,
  verify transcript, Aitana-relevant extension points (custom policies,
  new event kinds).
- Registry: follow existing package conventions (whatever made
  `ailang install sunholo/config@…` work) — investigate and mirror.
- Acceptance: package importable as `pkg/sunholo/deontic/core` per repo docs.

### M5 — Routing note + cross-links (ailang repo, est. 30 LOC)
- Add Routing section to m-contracts-as-code-vertical.md recording:
  examples/docs = flagship showcase; `sunholo/deontic` = reusable engine;
  core = never (with rationale). Cross-link both directions. Changelog entry.

## Success Metrics
- All package modules `ailang check` green; `ailang verify` VERIFIED on settle.ail
- wave5 demo byte-identical to legal_obligation_engine expected output
- Committed to ailang-packages main per repo convention; routing recorded

## Risks
| Risk | Mitigation |
|---|---|
| Z3 can't prove generalized (parameterized) contracts | weaken ensures to provable bounds; keep unprovable invariants as runtime-checked docs |
| Package import path/registry conventions undocumented | mirror sunholo/config exactly; ask nothing, copy everything |
| Cross-repo sprint state | sprint JSON lives in ailang repo (state machinery); code lands in ailang-packages |
