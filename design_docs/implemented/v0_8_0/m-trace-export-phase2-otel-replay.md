# M-TRACE-EXPORT Phase 2-4: OTEL Integration, Replay, and Training Export

**Status**: Planned
**Target**: v0.8.0
**Priority**: P1 (Medium)
**Estimated**: 3-5 days
**Dependencies**: M-TRACE-EXPORT Phase 1 (implemented), observatory/chains infrastructure
**Author**: AILANG Core Team

---

## Context: Phase 1 Complete

Phase 1 (implemented in v0.7.4) delivered:
- `internal/trace/` package: schema, collector, JSONL serializer
- `--emit-trace jsonl` flag on `ailang run`
- Function enter/exit, effect, contract, budget delta, error events
- Call depth tracking with duration measurement
- 12 unit tests, validated JSONL output

**Key limitation of Phase 1:** Traces are standalone JSONL files. They don't flow into the existing OTEL/observatory/chains infrastructure. You can't see program-level traces in the dashboard alongside agent-level traces.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | OTEL spans for deterministic execution; replay verifies determinism |
| A2: Replayability | +2 | **Primary goal** - `ailang replay` verifies execution; OTEL spans enable time-travel debugging |
| A3: Effect Legibility | +1 | Effect invocations visible as OTEL spans in dashboard |
| A7: Machines First | +1 | Structured spans + JSONL training data for machine consumption |
| A9: Cost Visibility | +1 | Budget consumption visible in observatory alongside agent cost |

**Net Score: +6** → Proceed

---

## Problem Statement

**Two trace worlds exist but don't talk to each other:**

| System | What It Captures | Where It Lives |
|--------|-----------------|----------------|
| **Observatory/Chains** (Level 1) | Agent workflows, tool calls, cost, tokens | `observatory.db`, Cloud Trace, dashboard |
| **`--emit-trace` JSONL** (Level 2) | AILANG program execution | stdout, pipe to file |

When an agent runs `ailang run program.ail`, the observatory sees "Bash tool was called" but has no visibility into what the AILANG program actually did (which functions ran, which effects fired, whether contracts passed).

**Impact:**
- Can't correlate agent decisions with program behavior
- Can't see program traces in the dashboard
- Can't use `ailang chains diagnose` on program-level issues
- Replay requires separate tooling, not integrated with existing infrastructure

---

## Goals

### Phase 2: OTEL Span Emission (~12 hours)

Emit OTEL spans from `internal/trace/Collector` so program-level events flow into the existing observatory.

**What this enables:**
- Program traces visible in Cloud Trace alongside compiler spans
- `ailang chains view` shows program-level function calls
- Dashboard waterfall includes function/effect spans
- Unified query: "show me everything that happened when this agent ran this program"

### Phase 3: Trace Replay (~10 hours)

`ailang replay <trace.jsonl>` that re-executes a trace and verifies it produces the same results.

**What this enables:**
- Determinism verification: prove the program is deterministic
- Regression detection: detect when code changes alter execution
- Debugging: step through past executions

### Phase 4: Training Data Export (~6 hours)

Export high-quality traces for AI fine-tuning.

**What this enables:**
- AI self-improvement loop: successful AILANG programs become training data
- Quality scoring: filter traces by complexity, correctness, effect usage
- Export in standard fine-tuning formats

---

## Phase 2: OTEL Span Emission

### Architecture

The trace `Collector` already accumulates `[]TraceEvent` during execution. Phase 2 adds an OTEL emitter that converts these events to OTEL spans.

```
ailang run --emit-trace jsonl program.ail
    │
    ▼
EffContext.Trace = trace.Collector
    │
    ├── Phase 1: Events → JSONL (stdout)         ✅ Done
    │
    └── Phase 2: Events → OTEL spans (observatory)
                                │
                                ▼
                        observatory.db / Cloud Trace
                                │
                                ▼
                        ailang chains / dashboard
```

### New Files

**`internal/trace/otel_emitter.go`** (~200 LOC)
```go
// EmitOTELSpans converts collected trace events to OTEL spans.
// Called after program execution completes (batch emission).
func EmitOTELSpans(ctx context.Context, events []TraceEvent) error
```

Key decisions:
- **Batch emission** (not streaming): all events emitted after execution completes
- Function enter/exit pairs → single OTEL span with duration
- Effect events → child spans of the enclosing function
- Contract checks → span events (annotations) on the function span
- Budget deltas → span attributes on the effect span

### Span Mapping

| Trace Event | OTEL Representation |
|------------|---------------------|
| `function_enter` + `function_exit` | Span: `eval.function.<name>` |
| `effect` | Span: `eval.effect.<effectName>.<opName>` |
| `contract_check` | SpanEvent on parent function span |
| `budget_delta` | Attributes on effect span |
| `module_start` + `module_end` | Span: `eval.module.<name>` |
| `error` | SpanEvent with error status |

### Span Attributes

```go
// Function span attributes
"ailang.function.name": "factorial"
"ailang.function.args": "[5]"
"ailang.function.result": "120"
"ailang.function.depth": 3

// Effect span attributes
"ailang.effect.name": "IO"
"ailang.effect.op": "println"
"ailang.effect.budget.used": 3
"ailang.effect.budget.limit": 5
"ailang.effect.budget.remaining": 2
```

### Hierarchy Linking

When `ailang run` is invoked by the coordinator (as a subprocess), the parent trace context is available via `TRACEPARENT` environment variable (for script agents) or can be correlated via `AILANG_PARENT_TASK_ID`.

```
Coordinator span (exec.tool_use: Bash "ailang run ...")
  └── eval.module.examples/hello          ← NEW (Phase 2)
        ├── eval.function.main
        │     ├── eval.function.processFile
        │     │     ├── eval.effect.FS.readFile
        │     │     └── eval.effect.IO.println
        │     └── eval.function.transform
        └── eval.module.end
```

### Modified Files

- `cmd/ailang/main.go` (~15 LOC): After trace collection, call `EmitOTELSpans` if OTEL is configured
- `internal/trace/otel_emitter.go` (NEW ~200 LOC): Batch span emission
- `internal/trace/otel_emitter_test.go` (NEW ~150 LOC): Tests with mock exporter

### CLI Integration

```bash
# JSONL only (Phase 1 behavior)
ailang run --emit-trace jsonl --caps IO --entry main program.ail

# OTEL only (new in Phase 2)
ailang run --emit-trace otel --caps IO --entry main program.ail

# Both JSONL + OTEL (new in Phase 2)
ailang run --emit-trace jsonl,otel --caps IO --entry main program.ail

# OTEL happens automatically when GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT is set
# (like existing compiler spans)
ailang run --emit-trace auto --caps IO --entry main program.ail
```

---

## Phase 3: Trace Replay

### Architecture

```bash
# Capture trace
ailang run --emit-trace jsonl --caps IO --entry main program.ail > trace.jsonl

# Replay and verify
ailang replay trace.jsonl --verify

# Step through
ailang replay trace.jsonl --step
```

### New Files

**`internal/trace/replayer.go`** (~300 LOC)
- Parse JSONL trace file
- Re-execute the same program with trace verification
- Compare actual events against expected events
- Report mismatches with context

**`cmd/ailang/replay.go`** (~120 LOC)
- `ailang replay <trace.jsonl>` command
- `--verify` flag: compare replay trace against original
- `--step` flag: pause after each function enter/exit
- `--diff` flag: show only differences between original and replay

### Verification Rules

| Check | Pass Condition |
|-------|----------------|
| Function call order | Same functions called in same order |
| Function arguments | Same argument values (string comparison) |
| Function results | Same return values |
| Effect invocations | Same effects called with same arguments |
| Contract results | Same pass/fail for all contract checks |
| Budget consumption | Same usage counts |

### Mismatch Report

```
REPLAY MISMATCH at event #47:
  Expected: function_exit "factorial" result="120" depth=1
  Actual:   function_exit "factorial" result="119" depth=1
  Context:  After factorial(5) → ... → factorial(2) → factorial(1)

Possible causes:
  - Non-deterministic code (random, time, external state)
  - Code changed between capture and replay
  - Different environment (arguments, capabilities)
```

### Determinism Guarantee

Replay will only match for deterministic programs. Programs using:
- `Clock.now` → non-deterministic unless `--virtual-time` is set
- `IO.readLine` → non-deterministic (depends on stdin)
- `Net.httpGet` → non-deterministic (depends on network)

The replay report flags these as expected mismatches.

---

## Phase 4: Training Data Export

### Architecture

```bash
# Export training data from trace directory
ailang export-training traces/ --format jsonl --min-score 0.7

# Score a single trace
ailang score-trace trace.jsonl
```

### Quality Scoring

Each trace gets a quality score based on:

| Factor | Weight | Description |
|--------|--------|-------------|
| **Completion** | 30% | Program completed without errors |
| **Complexity** | 25% | Number of distinct functions, effects, depth |
| **Contract coverage** | 20% | Percentage of contracts that passed |
| **Budget efficiency** | 15% | Used budget vs declared budget (not wasteful) |
| **Effect diversity** | 10% | Uses multiple effect types |

### Export Format

```jsonl
{"input": "<AILANG source>", "output": "<trace events>", "score": 0.85, "metadata": {"effects": ["IO", "FS"], "functions": 12, "max_depth": 5}}
```

### New Files

- `internal/trace/scorer.go` (~150 LOC): Quality scoring
- `internal/trace/exporter.go` (~100 LOC): Training data formatting
- `cmd/ailang/export_training.go` (~80 LOC): CLI command

---

## Implementation Order

1. **Phase 2** (OTEL integration) — highest value, connects to existing infrastructure
2. **Phase 3** (Replay) — enables determinism verification
3. **Phase 4** (Training export) — enables AI self-improvement

Phases are independent and can be released separately.

---

## Success Criteria

### Phase 2
- [ ] Program-level spans appear in Cloud Trace when `GOOGLE_CLOUD_PROJECT` is set
- [ ] `ailang chains view` shows function/effect spans for `ailang run` executions
- [ ] Dashboard waterfall includes program-level spans
- [ ] No performance regression when OTEL is not configured
- [ ] Tests with mock exporter

### Phase 3
- [ ] `ailang replay trace.jsonl --verify` passes for deterministic programs
- [ ] Mismatches reported with context and location
- [ ] Non-deterministic effects flagged as expected mismatches
- [ ] Step-through mode works interactively

### Phase 4
- [ ] Quality scoring produces meaningful scores (0.0 - 1.0)
- [ ] Export filters low-quality traces
- [ ] Exported data is valid JSONL
- [ ] Documentation with examples

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| OTEL overhead during execution | Medium | Batch emission (post-execution), not streaming |
| Large programs produce many spans | Medium | Depth limit, sampling for deep recursion |
| Replay false positives | Low | Clear docs on determinism requirements |
| Training data quality | Low | Scoring with configurable thresholds |

---

## Related Documents

- [M-TRACE-EXPORT Phase 1](m-trace-export.md) — Original design doc (Phase 1 implemented)
- [M-SEM-KERNEL-VISION](m-sem-kernel-vision.md) — Pillar 4: Traces as Semantic Artifacts
- [M-TASK-GRAPH-SPANS-UNIFICATION](m-task-graph-spans-unification.md) — Dashboard span integration

---

**Document created**: 2026-02-10
**Last updated**: 2026-02-10
