# Sprint Plan: M-TRACE-BRIDGE

## Summary
Bridge existing OTEL spans to TraceRegistry so `_trace_check` can verify them.

**Duration:** 0.5 days (~4 hours)
**Dependencies:** M-TRACE-TEST (completed)
**Risk Level:** Low

## Milestones

### M1: Create TracedTracer Wrapper (~1.5 hours)
**Goal:** Create wrapper that records span names to TraceRegistry

**Tasks:**
1. Create `internal/telemetry/traced_tracer.go`
2. Implement `TracedTracer` struct wrapping `trace.Tracer`
3. Add `AILANG_TRACE_RECORDING` environment toggle
4. Write unit tests

**Acceptance Criteria:**
- [ ] `NewTracedTracer(name)` returns a valid tracer
- [ ] Wrapper records span names to TraceRegistry when enabled
- [ ] Zero overhead when `AILANG_TRACE_RECORDING` not set
- [ ] Unit tests pass

### M2: Update Tracer Definitions (~1 hour)
**Goal:** Replace `otel.Tracer` with `telemetry.NewTracedTracer`

**Files to update:**
- `internal/pipeline/pipeline.go`
- `internal/repl/repl.go`
- `internal/ai/gemini/client.go`
- `internal/ai/anthropic/client.go`
- `internal/ai/ollama/client.go`
- `internal/ai/openai/client.go`
- `internal/executor/gemini/gemini.go`
- `internal/executor/claude/claude.go`
- `internal/messaging/store.go`
- `internal/coordinator/daemon_tasks.go`
- `internal/coordinator/approval_processor.go`
- `internal/coordinator/human_interaction.go`
- `internal/coordinator/daemon_approval.go`

**Acceptance Criteria:**
- [ ] All tracers use `telemetry.NewTracedTracer`
- [ ] Build passes
- [ ] Existing tests pass

### M3: Integration Test & Documentation (~1 hour)
**Goal:** Verify end-to-end and update docs

**Tasks:**
1. Create integration test that compiles a file and checks traces
2. Update CHANGELOG.md
3. Update example to demonstrate working traces

**Acceptance Criteria:**
- [ ] Integration test passes with `AILANG_TRACE_RECORDING=1`
- [ ] CHANGELOG updated
- [ ] All tests pass

## LOC Estimate
| Component | LOC |
|-----------|-----|
| traced_tracer.go | ~60 |
| traced_tracer_test.go | ~80 |
| Tracer updates (13 files) | ~13 (one line each) |
| Integration test | ~40 |
| **Total** | ~195 |
