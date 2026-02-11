# Sprint Plan: M-TRACE-EXPORT Phase 3 — Trace Replay

## Summary

Implement `ailang replay <trace.jsonl>` that re-executes a program and compares the new trace against a recorded baseline. Enables determinism verification, regression detection, and execution auditing.

**Duration:** 1 day (5-6 hours)
**Dependencies:** M-TRACE-EXPORT Phase 1 (complete), Phase 2 (complete)
**Risk Level:** Low — reads existing JSONL, re-runs existing pipeline, compares events

## Current Status Analysis

### Completed Recently
- M-TRACE-EXPORT Phase 1: trace package, collector, JSONL, `--emit-trace` flag (~500 LOC)
- M-TRACE-EXPORT Phase 2: OTEL emitter, multi-mode flag, auto-enable (~510 LOC impl+tests)
- All 23 trace package tests passing

### Velocity
- Phase 2 completed in ~2 hours (~510 LOC)
- Recent average: ~400-600 LOC/day
- This sprint: ~550 LOC estimated (well within capacity)

### What Exists (Don't Duplicate)
- `internal/trace/schema.go` — `TraceEvent` struct, 8 event types
- `internal/trace/collector.go` — Accumulates `[]TraceEvent` during execution
- `internal/trace/serializer.go` — `WriteJSONL()` (writer only, NO reader)
- `internal/trace/otel_emitter.go` — OTEL span conversion
- `cmd/ailang/main.go` — `runFile()` with full pipeline + trace collection
- Subcommand pattern: `switch command { case "run": ... case "repl": ... }`

## Key Design Decision: Comparison-Based Replay

**NOT effect-stubbing.** Instead:

1. Read the baseline JSONL trace
2. Extract the source file path from `module_start` event
3. Re-execute the program with trace collection enabled
4. Compare baseline events vs new events
5. Report matches and mismatches

**Why this approach:**
- Simpler (~300 LOC vs ~600+ for effect stubbing)
- More useful: answers "did my code change break anything?"
- No fragile mocking infrastructure
- Same infrastructure already works (`runFile` + `Collector`)
- Effect stubbing can be Phase 3.1 if needed

**What gets compared:**
| Field | Compared? | Notes |
|-------|-----------|-------|
| Event type | Yes | Must match exactly |
| Event order | Yes | Must match sequence |
| Function name | Yes | Must match |
| Function args | Yes | Must match (string comparison) |
| Function result | Yes | Must match |
| Effect name/op | Yes | Must match |
| Effect args/result | Yes | Must match |
| Contract kind/passed | Yes | Must match |
| Budget used/limit | Yes | Must match |
| Timestamp | **No** | Non-deterministic |
| Duration | **No** | Non-deterministic |
| Depth | Yes | Must match (structural) |

## Proposed Milestones

### Milestone 1: JSONL Reader + Trace Comparison Core
**Goal:** Create `ReadJSONL()` and `CompareTraces()` — the core logic.
**Estimated:** ~200 LOC implementation + ~200 LOC tests = ~400 LOC
**Duration:** ~3 hours

**Tasks:**

**Task 1: `internal/trace/reader.go` (~60 LOC)**

```go
package trace

// ReadJSONL reads trace events from JSONL format.
func ReadJSONL(r io.Reader) ([]TraceEvent, error)

// TraceMetadata extracts metadata from a trace event list.
// Returns module name, caps, and source file hint.
func TraceMetadata(events []TraceEvent) (moduleName string, caps []string)
```

- `bufio.Scanner` line-by-line reading
- `json.Unmarshal` each line into `TraceEvent`
- Skip empty lines
- Validate version field
- `TraceMetadata` scans for first `module_start` to extract module name + caps

**Task 2: `internal/trace/comparator.go` (~140 LOC)**

```go
package trace

// Mismatch describes a difference between two trace event streams.
type Mismatch struct {
    Index    int        // Event index where mismatch occurred
    Field    string     // Which field differs (e.g., "function.result")
    Expected string     // Value from baseline trace
    Actual   string     // Value from replay trace
    Event    TraceEvent // The baseline event for context
}

// CompareResult holds the result of comparing two traces.
type CompareResult struct {
    Match      bool
    Mismatches []Mismatch
    BaselineN  int // Number of events in baseline
    ReplayN    int // Number of events in replay
    Skipped    int // Non-deterministic fields skipped
}

// CompareTraces compares a baseline trace against a replay trace.
// Timestamps and durations are ignored (non-deterministic).
func CompareTraces(baseline, replay []TraceEvent) CompareResult
```

Algorithm:
- Walk both event lists in parallel
- Compare event types, then payload fields per type
- Skip `TimestampNS` and `DurationNS` fields
- Length mismatch → report as final mismatch
- Collect all mismatches (don't stop at first)

**Task 3: Tests (~200 LOC)**

`internal/trace/reader_test.go`:
- Read valid JSONL (round-trip with WriteJSONL)
- Read empty input → empty slice
- Read malformed line → error
- Read with blank lines → skip
- TraceMetadata extracts module name and caps

`internal/trace/comparator_test.go`:
- Identical traces → match
- Different function result → mismatch with field="function.result"
- Different event count → mismatch
- Different event order → mismatch
- Timestamps differ but content same → match (skipped)
- Durations differ but content same → match (skipped)
- Empty traces → match
- Contract pass/fail difference → mismatch

### Milestone 2: CLI Command + Integration
**Goal:** `ailang replay <trace.jsonl>` command with `--verify`, `--diff`, `--json` flags.
**Estimated:** ~120 LOC implementation + ~30 LOC tests = ~150 LOC
**Duration:** ~2 hours

**Tasks:**

**Task 1: `cmd/ailang/replay.go` (~100 LOC)**

```go
func replayCommand() {
    fs := flag.NewFlagSet("replay", flag.ExitOnError)
    verify := fs.Bool("verify", true, "Verify replay matches baseline")
    jsonOutput := fs.Bool("json", false, "Output comparison result as JSON")
    caps := fs.String("caps", "", "Override capabilities (default: from trace)")
    entry := fs.String("entry", "", "Override entry function (default: from trace)")
    quiet := fs.Bool("quiet", false, "Suppress progress messages")
    _ = fs.Parse(os.Args[2:])
    // ...
}
```

Flow:
1. Read trace file (`trace.ReadJSONL`)
2. Extract metadata (`trace.TraceMetadata` → module name, caps)
3. Find source file from module name (convention: module path → file path)
4. Re-run with `--emit-trace` equivalent (reuse `runFile` infrastructure or call pipeline directly)
5. Compare traces (`trace.CompareTraces`)
6. Print result (human-readable or JSON)

Output formats:

**Human-readable (default):**
```
Replaying: examples/runnable/hello.ail
Module: examples/runnable/hello
Capabilities: IO
Events: 2 baseline, 2 replay

✓ REPLAY MATCHES (2/2 events identical)
```

**Mismatch output:**
```
Replaying: examples/runnable/factorial.ail

✗ REPLAY MISMATCH (3 differences in 12 events)

  Event #5: function.result differs
    Expected: "120"
    Actual:   "119"
    Context:  function_exit "factorial" at depth 1

  Event #8: effect.result differs
    Expected: "()"
    Actual:   "error: IO budget exceeded"
    Context:  effect "IO.println" at depth 1

  Event count: expected 12, got 10
    Missing events from index 10
```

**JSON output (`--json`):**
```json
{
  "match": false,
  "baseline_events": 12,
  "replay_events": 10,
  "mismatches": [
    {"index": 5, "field": "function.result", "expected": "120", "actual": "119"}
  ]
}
```

Exit codes:
- 0: replay matches baseline
- 1: replay has mismatches
- 2: error (file not found, parse error, etc.)

**Task 2: Register in `cmd/ailang/main.go` (~5 LOC)**

Add to the command switch:
```go
case "replay":
    replayCommand()
```

Add to help text.

**Task 3: Integration test (~30 LOC)**

- Capture trace: `ailang run --emit-trace jsonl --quiet --caps IO --entry main hello.ail > trace.jsonl`
- Replay: `ailang replay trace.jsonl`
- Verify exit code 0

**Acceptance Criteria:**
- [ ] `ailang replay trace.jsonl` re-runs the program and compares events
- [ ] `ailang replay trace.jsonl --json` outputs JSON comparison result
- [ ] Exit code 0 for matching traces, 1 for mismatches, 2 for errors
- [ ] Timestamps and durations are ignored during comparison
- [ ] Module name and caps extracted from baseline trace
- [ ] `ReadJSONL` round-trips with `WriteJSONL`
- [ ] Human-readable mismatch output shows event index, field, expected/actual
- [ ] `go test ./internal/trace/` passes (all existing + new tests)
- [ ] `go build ./...` clean

## Source File Resolution

The trace records `module_start` with the module name (e.g., `examples/runnable/hello`). To replay, we need the `.ail` source file. Resolution:

1. Check if module name + `.ail` exists relative to CWD
2. Check if module name + `.ail` exists relative to trace file location
3. Allow `--file` flag override: `ailang replay trace.jsonl --file path/to/hello.ail`

This is simple and covers the common case where you replay from the project root.

## What This Does NOT Include
- No effect stubbing (run real program, compare output)
- No step-through mode (`--step` flag) — Phase 3.1
- No interactive debugging — future
- No non-module file replay — modules only (same limitation as `--emit-trace`)
- No `--diff` colorized output — can add later

## Open Questions
- **Should `--verify` be default?** Yes — it's the whole point of replay. Flag exists to disable if needed.
- **Should we re-capture OTEL spans during replay?** No — replay is about comparison, not observability. User can run `--emit-trace otel` separately.

## Success Metrics
- All existing tests passing: `go test ./...`
- New tests: 10+ test cases across reader + comparator
- `ailang replay` works on hello.ail trace round-trip
- Deterministic programs produce matching replays
- Non-deterministic differences (timestamps) are correctly skipped
