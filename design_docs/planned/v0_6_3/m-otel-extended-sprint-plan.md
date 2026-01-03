# M-OTEL-EXTENDED Sprint Plan

**Sprint ID:** M-OTEL-EXTENDED
**Design Doc:** [m-otel-extended-instrumentation.md](m-otel-extended-instrumentation.md)
**Target:** v0.6.3
**Duration:** 1 day (~6 hours)
**Risk Level:** Low

## Goal

Add OpenTelemetry instrumentation to Compiler Pipeline, Eval Harness, and Message System with zero overhead when disabled.

## Milestones

### M1: Compiler Pipeline Instrumentation (~2 hours, ~120 LOC)

**Description:** Add spans for compilation phases (lex, parse, elaborate, typecheck, lower).

**Files to modify:**
- `internal/pipeline/pipeline.go` (+80 LOC)
- `internal/pipeline/module_pipeline.go` (+40 LOC)

**Tasks:**
- [ ] Add tracer import and package-level tracer
- [ ] Add `compile.pipeline` parent span with file attributes
- [ ] Add `compile.lex` span with token count
- [ ] Add `compile.parse` span with AST node count
- [ ] Add `compile.elaborate` span with core node count
- [ ] Add `compile.typecheck` span with type/constraint counts
- [ ] Add `compile.lower` span with IR size
- [ ] Propagate context through all phases

**Acceptance Criteria:**
- [ ] Compilation produces spans visible in GCP/Jaeger
- [ ] Spans show parent-child hierarchy
- [ ] Zero overhead when OTEL disabled (no-op tracer)
- [ ] Existing tests pass

### M2: Eval Harness Instrumentation (~2 hours, ~100 LOC)

**Description:** Add spans for benchmark suite execution and individual benchmarks.

**Files to modify:**
- `internal/eval_harness/runner.go` (+60 LOC)
- `internal/eval_harness/model_runner.go` (+40 LOC)

**Tasks:**
- [ ] Add tracer import and package-level tracer
- [ ] Add `eval.suite` span with benchmark/model counts
- [ ] Add `eval.benchmark` span with ID and difficulty
- [ ] Add `eval.model_call` span with model/tokens
- [ ] Add `eval.validation` span with pass/fail status
- [ ] Record token counts and costs on spans

**Acceptance Criteria:**
- [ ] Eval runs produce spans visible in GCP/Jaeger
- [ ] Each benchmark is a child span of suite
- [ ] Token/cost data visible in span attributes
- [ ] Existing eval tests pass

### M3: Message System Instrumentation (~2 hours, ~110 LOC)

**Description:** Add spans for message CRUD operations and search.

**Files to modify:**
- `internal/messaging/store.go` (+50 LOC)
- `internal/messaging/search.go` (+30 LOC)
- `internal/coordinator/daemon_github.go` (+30 LOC)

**Tasks:**
- [ ] Add tracer import and package-level tracer
- [ ] Add `messages.send` span with inbox/type attributes
- [ ] Add `messages.read` span with message ID
- [ ] Add `messages.list` span with query/count attributes
- [ ] Add `messages.search` span with query/neural flag
- [ ] Add `messages.ack` span with message ID
- [ ] Add `messages.github_sync` span with repo/count

**Acceptance Criteria:**
- [ ] Message operations produce spans
- [ ] Search operations show query and result count
- [ ] GitHub sync shows imported issue count
- [ ] Existing messaging tests pass

### M4: Documentation Update (~30 min, ~50 LOC)

**Description:** Update telemetry docs with new instrumented components.

**Files to modify:**
- `docs/docs/guides/telemetry.md` (+50 LOC)

**Tasks:**
- [ ] Add Compiler section with span names/attributes
- [ ] Add Eval Harness section
- [ ] Add Message System section
- [ ] Update architecture diagram

**Acceptance Criteria:**
- [ ] Docs build without errors
- [ ] All new spans documented

## Success Metrics

- [ ] All 4 milestones complete
- [ ] Tests pass: `make test`
- [ ] Lint clean: `make lint`
- [ ] ~380 LOC added total
- [ ] Zero performance impact when telemetry disabled

## Dependencies

- Telemetry infrastructure (completed in v0.6.1) ✅
- Global TracerProvider pattern established ✅

## Notes

- Uses same pattern as existing AI provider instrumentation
- No-op overhead measured at <5ns per span when disabled
- All changes are additive (no behavioral changes)
