# Fleet mission — iteration 42, 2026-10-16

- Latest released binary: v0.54.11. dev: `51f9ecb29` (this iteration's landing).
- LANDED this fire: #1843 → `51f9ecb29`, `worktree:native-background-terminal-marker-missing` (world 268).
  `mission-worktree.sh` no longer reports `pending` forever after its writer dies. It detaches the writer and returns `failed:writer_died` or `failed:owner_never_published`.
- Judge (sonnet, Agent tool, own worktree): FAIL 66 (a caller-death race), then PASS 86. 19 of 20 mutants were killed on the host. Linux bash 5 is unmeasured.
- Planner and executor: codex `gpt-6.1-sol` (recipe). Controller: opus. Metered $0.
- Dev CI on `51f9ecb29`: required checks green. The non-required macOS Build leg failed on an `internal/effects` timing test (not fleet code).
- Next (banked): #16 `quorum:tier2-salvage-fold` (not live), then the `slots=1` tickets oldest first. The agent-tool `slots>1` tickets stay PARKED-ON-LANE (codex-controller native spawn).
- Parked on Mark:
  - D-FLEET-16: skill source (A recommended / C default).
  - D-FLEET-18: `tools/eval` scope for #13 and the new `ci:motoko-probe-…` ticket (A recommended / C default).
  - D-FLEET-19 (new): Pub/Sub notify fails because the rig's ADC quota project is `sunholo-gemini-enterprise` (A: `set-quota-project ailang-multivac` / C default).
  - D-FLEET-17 is moot; Mark may close it.
- Cadence: fleet 6h, ticket-driven. Thread #1768; no new human directive. Nothing reloaded.
- Clauses: 1 unmeasured, 2 unmet, 3 met (48 open), 4 prior evidence, 5 preserved.
