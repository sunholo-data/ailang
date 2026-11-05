
### Fixed — parking window survives release churn

- `tools/eval-park-set.sh` windows each benchmark over the last N banks that actually RAN it, not
  the last N banks overall. On 2026-11-04 two quick releases (v0.57.0, v0.57.1) that had not yet
  reached `cp_60166` pushed its old rows out of the global window (3 < min-rows), un-parking it
  with no new evidence and queueing ~9 h of full trials it has timed out on every time.

### Changed — the rig runs signal benchmarks only, and routes its failures

- os-rotation filler `OS_FILLER_SIGNAL_ONLY=1` (default): the AILANG pass drops **locally
  saturated** benchmarks (`tools/eval-park-set.sh --mode saturated`: every local harness passed every
  trial, >= 3 rows each, in the benchmark's last 3 banks) and runs the rest at full trials; the global
  ELO "Trivial" band no longer thins trials (it blends cloud models). On 2026-11-05 data: 30 of 56
  kept. `=0` restores full coverage. PROGRAM.md §7 records the trade-off (local ELO is no longer a
  full-coverage on-device-vs-cloud measure).
- `tools/rig-friction-digest.sh`: failures grouped for PROGRAM.md §4 — AILANG error kinds across
  benchmarks, motoko-gap / harness-gap / budget per benchmark, plus non_agentic harness rows.
  `--route` (weekly, filler step 0b, `OS_FILLER_FRICTION=0` disables) sends NEW items only:
  error kinds seen >= 2 times → `ailang-core`; harness gaps → the motoko mission, or the user inbox
  while it is paused. Fixture test `tools/launchd/test_rig_friction_digest.sh` (6 mutants caught).
