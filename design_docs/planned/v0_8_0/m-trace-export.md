# M-TRACE-EXPORT: Execution Trace Export for AI Training

**Status**: Planned
**Target**: v0.7.0
**Priority**: P1 (Medium) - Enables Axiom A2 compliance
**Estimated**: 3-4 days
**Dependencies**: M-TOOLING-DETERMINISTIC (v0.7.0)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Traces capture deterministic execution flow |
| A2: Replayability | +1 | **Primary goal** - enables full trace replay |
| A3: Effect Legibility | +1 | Effects are recorded in trace output |
| A4: Explicit Authority | 0 | No change to authority model |
| A5: Bounded Verification | +1 | Traces enable verification of execution |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | JSON/JSONL format for machine consumption |
| A8: Minimal Syntax | 0 | No new syntax (CLI flag only) |
| A9: Cost Visibility | +1 | Traces include cost/resource metadata |
| A10: Composability | 0 | No compositional changes |
| A11: Structured Failure | +1 | Errors captured in structured trace format |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** -> **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Traces are deterministic artifacts
- [x] A3 (Effects): Effects explicitly recorded in traces
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): JSONL format optimized for machine processing

## Problem Statement

AILANG's Axiom A2 (Replayability) states: "Execution is a first-class artifact. Traces are inspectable, serializable, replayable, and suitable for verification."

**Current State:**
- Basic `--trace` flag exists for runtime tracing
- No standardized export format
- No ability to serialize traces for replay
- No training data export for AI self-improvement
- **Axiom A2 score: 1/2 (partial)**

**Impact:**
- Cannot verify execution by replaying traces
- Cannot use AILANG programs for AI fine-tuning
- Limited debugging capabilities for autonomous agents

## Goals

**Primary Goal:** Export structured execution traces in JSONL format for replay and AI training.

**Success Metrics:**
- `--emit-trace jsonl` flag available on `ailang run`
- Trace files replayable with `ailang replay <trace.jsonl>`
- Traces include: expressions, types, values, effects, timing
- AI training data exportable from high-quality traces
- **Axiom A2 score improved to 2/2 (strong)**

## Solution Design

### Overview

Add `--emit-trace <format>` flag that captures execution events and outputs structured trace data suitable for replay and analysis.

### Architecture

**Components:**
1. **Trace Collector**: Captures execution events during evaluation
2. **Trace Serializer**: Converts events to JSONL format
3. **Trace Replayer**: Reconstructs execution from trace file
4. **Training Exporter**: Filters high-quality traces for AI fine-tuning

### Trace Schema

```json
{
  "version": "1.0",
  "event": "eval",
  "timestamp_ns": 1703001234567890123,
  "node_id": 42,
  "expression": "add(1, 2)",
  "input_type": "(int, int)",
  "output_type": "int",
  "result": "3",
  "effects": [],
  "duration_ns": 1234,
  "depth": 3
}
```

**Event Types:**
- `module_start` / `module_end`
- `function_enter` / `function_exit`
- `eval` - Expression evaluation
- `effect` - Effect execution (IO, FS, etc.)
- `error` - Error occurrence

### Implementation Plan

**Phase 1: Trace Collection** (~8 hours)
- [ ] Create `internal/trace/collector.go` with event capture
- [ ] Add trace hooks to `internal/eval/eval.go`
- [ ] Capture expression, type, value, timing for each eval step
- [ ] Unit tests for collector

**Phase 2: Serialization** (~6 hours)
- [ ] Create `internal/trace/serializer.go` for JSONL output
- [ ] Add `--emit-trace jsonl` flag to CLI
- [ ] Stream traces to file during execution
- [ ] Add metadata (version, module, capabilities)

**Phase 3: Replay** (~8 hours)
- [ ] Create `internal/trace/replayer.go`
- [ ] Add `ailang replay <trace.jsonl>` command
- [ ] Verify trace matches actual execution
- [ ] Support partial replay (step-by-step)

**Phase 4: Training Export** (~4 hours)
- [ ] Add quality scoring for traces
- [ ] Filter high-quality traces (successful, non-trivial)
- [ ] Export in AI fine-tuning format
- [ ] Documentation and examples

### Files to Modify/Create

**New files:**
- `internal/trace/collector.go` - Event capture (~200 LOC)
- `internal/trace/serializer.go` - JSONL output (~150 LOC)
- `internal/trace/replayer.go` - Trace replay (~250 LOC)
- `internal/trace/schema.go` - Event types (~100 LOC)
- `cmd/ailang/replay.go` - CLI command (~100 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Add `--emit-trace` flag (~20 LOC)
- `internal/eval/eval.go` - Add trace hooks (~50 LOC)
- `internal/runtime/runtime.go` - Wire up collector (~30 LOC)

## Examples

### Example 1: Basic Trace Export

**Command:**
```bash
ailang run --emit-trace jsonl --caps IO examples/factorial.ail > trace.jsonl
```

**Output (trace.jsonl):**
```jsonl
{"version":"1.0","event":"module_start","module":"examples/factorial","timestamp_ns":1703001234567890123}
{"event":"function_enter","name":"factorial","args":[5],"depth":1,"timestamp_ns":1703001234567890200}
{"event":"eval","node_id":12,"expression":"n <= 1","result":"false","depth":2}
{"event":"eval","node_id":15,"expression":"n * factorial(n - 1)","result":"120","depth":2}
{"event":"function_exit","name":"factorial","result":"120","duration_ns":45000,"depth":1}
{"event":"module_end","module":"examples/factorial","duration_ns":50000}
```

### Example 2: Replay Verification

```bash
# Original run
ailang run --emit-trace jsonl examples/hello.ail > original.jsonl

# Replay and verify
ailang replay original.jsonl --verify
# Output: Trace replay successful: 42 events, 0 mismatches
```

### Example 3: Training Data Export

```bash
# Export high-quality traces for AI training
ailang export-training --score-threshold 0.8 traces/ > training_data.jsonl
```

## Success Criteria

- [ ] `--emit-trace jsonl` produces valid JSONL output
- [ ] Traces are deterministic (same input = same trace)
- [ ] `ailang replay` can verify trace execution
- [ ] Training data export filters low-quality traces
- [ ] All tests passing
- [ ] Documentation updated with trace format spec
- [ ] Examples added to docs/guides/

## Testing Strategy

**Unit tests:**
- Collector captures all event types
- Serializer produces valid JSONL
- Replayer detects mismatches

**Integration tests:**
- End-to-end trace capture for example programs
- Replay verification passes for deterministic programs
- Effect traces captured correctly

**Manual testing:**
- Verify trace files are human-readable
- Test with large programs (performance)
- Test training export workflow

## Non-Goals

**Not in this feature:**
- Interactive trace debugger - Deferred to v0.8.0+
- Distributed tracing - Out of scope
- Binary trace format - JSONL sufficient for now
- Trace visualization UI - Separate tooling

## Timeline

**Week 1** (14 hours):
- Phase 1: Trace collection
- Phase 2: Serialization

**Week 2** (12 hours):
- Phase 3: Replay
- Phase 4: Training export
- Documentation

**Total: ~26 hours across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Performance overhead | Medium | Use buffered writes, optional tracing |
| Large trace files | Low | Compress traces, streaming output |
| Schema evolution | Medium | Version field, migration tools |

## Related Documents

**Related to M-TOOLING-DETERMINISTIC:**
- [design_docs/planned/v0_7_0/M-TOOLING-DETERMINISTIC.md](M-TOOLING-DETERMINISTIC.md) - Deterministic CLI tools

**Axiom References:**
- [Design Axioms](/docs/references/axioms) - A2: Replayability
- [Axiom Scorecard](docs/static/benchmarks/axiom_scorecard.json) - KPI tracking

## Future Work

- Interactive trace debugger with step-through
- Trace diff tool for comparing executions
- Binary trace format for high-performance use cases
- Trace-based coverage analysis

---

**Document created**: 2025-12-19
**Last updated**: 2025-12-19
