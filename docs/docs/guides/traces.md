---
sidebar_position: 12
title: Execution Traces
description: Capture, replay, score, and export AILANG program execution traces
---

# Execution Traces

AILANG captures detailed execution traces when programs run — every function call, effect invocation, contract check, and budget change. These traces enable determinism verification, debugging, and AI training data export.

## Why Traces Matter

AILANG is designed as a **symbolic reasoning kernel** where AI agents write and execute programs. Traces provide the evidence chain:

- **Determinism proof**: Run a program twice, compare traces — identical traces mean identical behavior
- **Debugging**: See exactly which functions ran, in what order, with what arguments
- **Quality scoring**: Automatically score program executions for complexity, correctness, and resource efficiency
- **Training data**: Export high-quality traces as fine-tuning data for AI models

## Quick Start

```bash
# 1. Capture a trace
ailang run --emit-trace jsonl --caps IO --entry main program.ail > trace.jsonl

# 2. Inspect the trace
cat trace.jsonl | jq .

# 3. Verify determinism (re-runs and compares)
ailang replay trace.jsonl

# 4. Score the trace quality
ailang export-training --score trace.jsonl

# 5. Export as training data
ailang export-training --min-score 0.5 traces/
```

## Capturing Traces

Add `--emit-trace jsonl` to any `ailang run` command. When JSONL tracing is active, **all status messages and program output go to stderr**, leaving stdout as clean JSONL that pipes directly to `jq`:

```bash
# Capture trace, see program output on screen
ailang run --emit-trace jsonl --caps IO --entry main module.ail > trace.jsonl

# Pipe directly to jq (stdout is pure JSONL)
ailang run --emit-trace jsonl --caps IO --entry main module.ail 2>/dev/null | jq .

# Capture both trace and program output separately
ailang run --emit-trace jsonl --caps IO --entry main module.ail > trace.jsonl 2> output.txt

# Also emit OTEL spans (for Cloud Trace integration)
ailang run --emit-trace jsonl,otel --caps IO --entry main module.ail > trace.jsonl
```

### Trace Event Types

Each line in the JSONL file is one event:

| Event | When | Key Fields |
|-------|------|------------|
| `module_start` | Program begins | module name, granted capabilities |
| `function_enter` | Function called | function name, arguments, call depth |
| `function_exit` | Function returns | function name, result, duration |
| `effect` | Side effect invoked | effect name, operation, args |
| `contract_check` | Pre/postcondition tested | kind, passed/failed, message |
| `budget_delta` | Resource consumed | used, limit, remaining |
| `error` | Runtime error | message, position |
| `module_end` | Program ends | module name, error count |

### Example Trace

Running a simple "Hello, AILANG!" program produces:

```json
{"version":"1.0","event":"module_start","timestamp_ns":750,"module":{"name":"examples/runnable/hello","caps":["IO"]}}
{"version":"1.0","event":"function_enter","timestamp_ns":24375,"depth":1,"function":{"name":"std/io.print","args":["Hello, AILANG!"]}}
{"version":"1.0","event":"effect","timestamp_ns":30000,"depth":1,"effect":{"effect_name":"IO","op_name":"print","args":["Hello, AILANG!"],"result":"()"}}
{"version":"1.0","event":"function_exit","timestamp_ns":39167,"depth":1,"function":{"name":"std/io.print","result":"()","duration_ns":15000}}
{"version":"1.0","event":"module_end","timestamp_ns":39375,"module":{"name":"examples/runnable/hello","duration_ns":38625}}
```

You can see: the module started with `IO` capability, called `std/io.print` with the greeting, the IO effect was invoked (`effect` event), the function returned `()` in 15μs, and the module ended with total duration recorded.

Non-deterministic effects (like `Net.httpGet`, `Clock.now`, `IO.readLine`) are automatically flagged with `"deterministic": false` in their `effect` event. The replay comparator skips argument/result comparison for these events, allowing traces with network calls or time-dependent operations to replay successfully.

## Replaying Traces (Determinism Verification)

The replay command re-executes the source program and compares the new trace against a baseline:

```bash
# Basic replay — auto-resolves source file and capabilities from the trace
ailang replay trace.jsonl

# JSON output for programmatic comparison
ailang replay --json trace.jsonl

# Override the source file (e.g., to test a modified version)
ailang replay --file modified.ail trace.jsonl
```

### How Auto-Resolution Works

Replay reads the `module_start` event from the baseline trace to determine:
- **Source file**: Module name `examples/runnable/hello` → resolves to `examples/runnable/hello.ail`
- **Capabilities**: `"caps":["IO"]` → passes `--caps IO` to the re-execution

This means you typically don't need `--file` or `--caps` overrides — just point at the trace file.

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Traces match — deterministic |
| 1 | Traces differ — non-deterministic behavior detected |
| 2 | Error (file not found, parse error, etc.) |

### What Gets Compared

Replay compares:
- Event types and order
- Function names and arguments
- Effect operations and parameters
- Contract check results

Replay ignores:
- Timestamps (execution speed varies)
- Durations (performance isn't determinism)

### Use Cases

**Regression testing**: Capture a baseline trace, make code changes, replay to verify behavior is preserved:

```bash
# Before changes
ailang run --emit-trace jsonl --caps IO --entry main module.ail > baseline.jsonl

# Make your changes...

# After changes — exit 0 means behavior unchanged
ailang replay baseline.jsonl
```

**CI/CD integration**: Add replay checks to your CI pipeline:

```bash
# Exits non-zero on mismatch — fails the build
ailang replay tests/traces/critical_path.jsonl
```

## Scoring Traces

Every trace can be scored for quality on a 0.0-1.0 scale:

```bash
# Human-readable score report
ailang export-training --score trace.jsonl

# Machine-readable (JSON)
ailang export-training --score --json trace.jsonl

# Score all traces in a directory
ailang export-training --score traces/
```

### Scoring Components

| Component | Weight | What It Measures |
|-----------|--------|-----------------|
| **Completion** | 30% | 1.0 if clean `module_end`, 0.0 if errors |
| **Complexity** | 25% | Function count, call depth, total calls (log scale) |
| **Contracts** | 20% | Pass rate of pre/postconditions (0.5 neutral if none) |
| **Budget efficiency** | 15% | 1.0 if 20-80% of budget used; penalizes waste or exhaustion |
| **Effect diversity** | 10% | More effect types = higher score |

### Interpreting Scores

- **0.0-0.3**: Trivial or broken — program crashed or did very little
- **0.3-0.5**: Simple — basic execution, few interesting behaviors
- **0.5-0.7**: Good — non-trivial logic with some verification
- **0.7-1.0**: Excellent — complex logic, contracts passing, efficient resource use

### Why Score?

Scoring enables **automated quality gating** for AI training pipelines. Instead of training on every program execution, you can filter for high-quality examples:

```bash
# Only export traces scoring above 0.7
ailang export-training --min-score 0.7 traces/ > training.jsonl
```

## Exporting Training Data

Convert scored traces into AI fine-tuning data:

```bash
# Export all traces as training JSONL
ailang export-training traces/

# Filter by quality
ailang export-training --min-score 0.5 traces/

# Write to file instead of stdout
ailang export-training --output training.jsonl traces/

# Include source code resolution
ailang export-training --source-dir src/ traces/
```

### Output Format

Each line is a complete training example:

```json
{
  "source": "module examples/runnable/hello\nimport std/io...",
  "trace": "{\"version\":\"1.0\",\"event\":\"module_start\"...}\n{...}\n...",
  "score": 0.85,
  "metadata": {
    "module": "examples/runnable/hello",
    "caps": ["IO"],
    "event_count": 4,
    "function_count": 1,
    "max_depth": 1,
    "has_errors": false
  }
}
```

The `source` field contains the original AILANG source code (resolved from the module name), and the `trace` field contains the full JSONL trace. Together they form a (program, execution) pair suitable for training.

### Source Resolution

The exporter tries to find source files in this order:
1. `--source-dir` flag (if provided)
2. Same directory as the trace file
3. Current working directory

It resolves the module name from the `module_start` event (e.g., `examples/runnable/hello` → `examples/runnable/hello.ail`).

## Full Pipeline Example

Here's a complete workflow for building an AI training dataset from AILANG program executions:

```bash
# Step 1: Run multiple programs, capturing traces
for f in examples/runnable/*.ail; do
  name=$(basename "$f" .ail)
  ailang run --emit-trace jsonl --caps IO --entry main "$f" > "traces/${name}.jsonl" 2>/dev/null
done

# Step 2: Verify all traces are deterministic
for t in traces/*.jsonl; do
  if ! ailang replay "$t" > /dev/null 2>&1; then
    echo "WARNING: Non-deterministic trace: $t"
  fi
done

# Step 3: Score and review
ailang export-training --score traces/

# Step 4: Export high-quality examples
ailang export-training --min-score 0.5 --source-dir examples/runnable/ --output training.jsonl traces/

# Result: training.jsonl contains scored (source, trace) pairs
echo "Training examples: $(wc -l < training.jsonl)"
```

## Two Levels of Traces

AILANG has traces at two complementary levels:

| Aspect | Program Traces (`--emit-trace`) | Agent Traces (`ailang chains`) |
|--------|-------------------------------|-------------------------------|
| **What** | AILANG program execution | AI agent workflows |
| **Granularity** | Functions, effects, contracts | Sessions, turns, tool calls |
| **When** | `ailang run program.ail` | Coordinator task execution |
| **Storage** | JSONL files (standalone) | observatory.db (SQLite) |
| **Example** | "println called 3 times, budget 3/5 used" | "Agent ran 12 turns, called Bash 5 times" |

Program traces (this guide) capture what happens **inside** AILANG code. Agent traces capture what happens **around** it — the AI agent's reasoning, tool usage, and multi-step workflows.

For agent-level tracing, see the [Telemetry & Tracing](telemetry.md) and [Coordinator](coordinator.md) guides.

## Known Limitations

- **Non-module files**: Traces currently require module files (`module` declaration + `--entry main`). Single-expression files are not yet supported
- **Step-through mode**: No interactive step-through replay yet (planned for future)
