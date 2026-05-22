# M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP — Wire chain_id/stage_id into eval-suite OTEL resource attrs

**Status**: Planned
**Target**: v0.22.0
**Priority**: P2 — Polish for M-EVAL-LOCAL-OBSERVABILITY
**Estimated**: 30 minutes, ~15 LOC
**Dependencies**: [M-EVAL-LOCAL-OBSERVABILITY](./m-eval-local-observability.md) (already shipped)

## TL;DR

After M-EVAL-LOCAL-OBSERVABILITY shipped, opencode spans land in observatory.db but lack `ailang.chain_id` and `ailang.stage_id` resource attributes. The `ailang chains live <id>` per-stage span join falls back to "(no spans yet)" because spans can't be linked to their stage. This follow-up plumbs the two IDs through the eval-suite OTLP-resource layer in 15 LOC.

## Problem

The plumbing is already mostly in place:

1. `internal/executor/environment.go::BuildResourceAttributes` reads `task.Metadata["chain_id"]` / `task.Metadata["stage_id"]` and writes them as OTEL_RESOURCE_ATTRIBUTES — works ✅
2. `internal/observatory/otlp_receiver.go` extracts `ailang.chain_id` from resource attrs and stores on spans.chain_id — works ✅
3. The coordinator path (`internal/coordinator/provider_executor.go:109-113`) populates these fields — works ✅

**The gap:** the eval-suite path (`internal/eval_harness/agent_runner_multi.go::RunAgentBenchmarkWithExecutor`) builds `executor.Task{}` without populating `Metadata`. So even though the chain/stage IDs are known at the call site in `cmd/ailang/eval_benchmark.go`, they don't reach the OTEL resource.

## Solution

Thread two strings through:

1. **`MultiExecutorConfig`** ([`internal/eval_harness/agent_runner_multi.go:25`](../../../internal/eval_harness/agent_runner_multi.go#L25)): add `ChainID string` + `StageID string` fields.
2. **`cmd/ailang/eval_benchmark.go:116-121`**: populate them from `evalChain.ChainID` + `stageID` before calling `RunAgentBenchmarkWithExecutor`.
3. **`agent_runner_multi.go:155`**: when building the `executor.Task`, populate `Metadata["chain_id"]` + `Metadata["stage_id"]` from the config.

After: opencode subprocess inherits OTEL_RESOURCE_ATTRIBUTES with `ailang.chain_id=...,ailang.stage_id=...`, OTLP receiver extracts them, spans.chain_id and spans.stage_id columns populate, `ailang chains live` per-stage join works precisely.

## Files to modify

| File | LOC delta | Why |
|---|---|---|
| `internal/eval_harness/agent_runner_multi.go` | +5 (fields + Metadata populate) | Thread the IDs through |
| `cmd/ailang/eval_benchmark.go` | +3 (set config fields) | Pass IDs into config |
| `internal/eval_harness/agent_runner_multi_test.go` or `cmd/ailang/chains_live_test.go` | +30 (verify end-to-end via in-memory backend) | Regression test |

**Total**: ~40 LOC (15 impl + ~30 test). Estimate generous; could be done in 15 minutes.

## Success Criteria

- [ ] `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957 make eval-smoke ...` produces spans with `chain_id IS NOT NULL` in observatory.db
- [ ] `ailang chains live <id>` shows real "X s ago" last-span ages instead of "(no spans yet)"
- [ ] No regression on existing tests
- [ ] CHANGELOG entry under M-EVAL-LOCAL-OBSERVABILITY-FOLLOWUP

## Axiom Compliance

Same +7 net score as M-EVAL-LOCAL-OBSERVABILITY parent (this is a completeness patch).
