# M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP — Sprint Plan

**Sprint ID**: M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP
**Design Doc**: [m-eval-local-observability-followup.md](./m-eval-local-observability-followup.md)
**Target Version**: v0.22.0
**Estimated Duration**: 30 minutes
**Total LOC**: ~40 (15 impl + 25 test)
**Risk Level**: Very low

## Goal

Make `ailang chains live <id>` per-stage span join work precisely by threading `chain_id` and `stage_id` through the eval-suite OTEL resource attribute pipeline. The infrastructure (BuildResourceAttributes, OTLP receiver extraction, ailang chains live join) is already in place — eval-suite just needs to populate `executor.Task.Metadata["chain_id"]` and `["stage_id"]` like the coordinator path does.

## Single Milestone

### M1 — Plumb chain_id/stage_id from eval_benchmark.go through to executor.Task.Metadata

**Estimate**: 30 minutes, ~40 LOC

**Files**:
- `internal/eval_harness/agent_runner_multi.go` (+5 LOC): add `ChainID string`, `StageID string` fields to `MultiExecutorConfig`; when building `executor.Task` at line 155, initialize `Metadata: map[string]string{"chain_id": config.ChainID, "stage_id": config.StageID}` (only populate keys when values non-empty).
- `cmd/ailang/eval_benchmark.go` (+3 LOC): before calling `RunAgentBenchmarkWithExecutor`, set `multiConfig.ChainID = evalChain.ChainID` and `multiConfig.StageID = stageID` (guarded by `evalChain != nil && stageID != ""`).
- `internal/eval_harness/agent_runner_multi_test.go` (+25 LOC): new regression test `TestRunAgentBenchmark_PopulatesChainStageMetadata` using a stub executor that captures the Task.Metadata; asserts `chain_id` and `stage_id` flow through.

**Acceptance criteria**:
- Build clean
- New regression test passes (assertion: stub executor sees `task.Metadata["chain_id"] == "<chain>"` and `task.Metadata["stage_id"] == "<stage>"`)
- Existing observatory + cmd/ailang tests pass (no regression)
- Manual verification: after rebuild, `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957 ailang eval-suite ... -agent ...` produces spans with non-NULL `chain_id` and `stage_id` columns. Query: `SELECT COUNT(*) FROM spans WHERE chain_id IS NOT NULL` should be > 0.
- `ailang chains live <id>` now shows real "X s ago" last-span ages instead of "(no spans yet)" for active chains.

**Dependencies**: None (extends already-shipped M-EVAL-LOCAL-OBSERVABILITY)

**Risk**: Very low. The two metadata keys are pure additions; existing code paths that don't populate them remain unchanged.

## Day-by-Day Plan

| Block | Hours | Work |
|---|---|---|
| 1 | 0.5 | M1: implement + test + commit + CHANGELOG entry |

That's the whole sprint.

## Success Metrics

- Build + tests green
- `SELECT COUNT(*) FROM spans WHERE chain_id IS NOT NULL` returns > 0 after a fresh smoke run
- `ailang chains live <id>` shows live per-stage span ages (no more "(no spans yet)" for active chains)

## Handoff

After approval, sprint-executor takes over with the JSON progress file at `.ailang/state/sprints/sprint_M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP.json`.
