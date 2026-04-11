---
sidebar_position: 11
title: Debugging Guide
description: Complete guide to debugging AILANG with environment variables and tools
---

# AILANG Debugging Guide

Complete guide to debugging AILANG with environment variables and tools.

## Debug Ghost Effect (In-Program Logging)

The `Debug` effect is a **ghost effect** — use it instead of `IO` (println) for logging. It's fully invisible: no `! {Debug}` in signatures, no `--caps Debug` needed, no `[effects].max` config.

```ailang
import std/debug (log, check)

-- Or use sunholo/logging for structured JSON:
-- import pkg/sunholo/logging/logger (info, warn, infoWith)

func processData(x: int) -> int {
  log("processing " ++ show(x));     -- ghost: invisible to callers
  check(x > 0, "x must be positive"); -- recorded, doesn't throw
  x * 2
}
```

Debug output is collected by the host and printed to **stderr** after execution. Control it with:

```bash
# See all debug output (default)
ailang run --caps IO --entry main app.ail

# Filter by severity (works with sunholo/logging JSON output)
ailang run --caps IO --log-level warn --entry main app.ail

# Suppress all debug output
ailang run --caps IO --log-level none --entry main app.ail

# Erase debug calls entirely (zero cost, production)
ailang run --release --caps IO --entry main app.ail
```

The `sunholo/logging` package provides structured JSON logging (`info`, `warn`, `err`, `trace`, `infoWith`, etc.) that integrates with `--log-level` filtering and Cloud Run log aggregation:
```bash
ailang install sunholo/logging@0.4.0
```

## Debug Flags

AILANG provides environment variables for verbose debugging and strict error checking.

### `DEBUG_STRICT=1` - Catch Silent Failures Early

**What it does**: Makes incomplete switch statements and unhandled cases **fail loudly** with panic instead of silently returning unchanged values.

**When to use**:
- During development of new compiler passes
- When debugging AST traversal code
- To catch missing cases in switch statements
- In CI to enforce completeness

**Example**:
```bash
# Normal mode - unhandled cases return unchanged (silent failure)
$ ailang run test.ail
# ✓ May complete successfully even with bugs!

# Strict mode - unhandled cases panic immediately
$ DEBUG_STRICT=1 ailang run test.ail
panic: cloneExpr: unhandled node type *core.Record (NodeID 42).
    Add a case for this type or explicitly mark as unsupported.
# ✓ Bug caught immediately!
```

**Affected functions** (as of v0.4.1):
- `internal/pipeline/specialize.go`:
  - `cloneExpr()` - Cloning during monomorphization
  - `specializeExpr()` - Specializing expressions

### `DEBUG_MONO_VERBOSE=1` - Monomorphization Tracing

**What it does**: Logs detailed information about monomorphization (polymorphic function specialization).

**When to use**:
- Debugging type substitution issues
- Understanding which functions are specialized
- Tracking down operator re-linking problems

**Example**:
```bash
$ DEBUG_MONO_VERBOSE=1 ailang run --entry main --debug-compile test.ail
[DEBUG_MONO_VERBOSE] Found lambda, type=α2 -> α2 -> α2, isPoly=true
[DEBUG_MONO_VERBOSE] lambda type from CoreTI: α2 -> (α2 -> α2)
[DEBUG_MONO_VERBOSE] extracted paramTVars: [α2]
[DEBUG_MONO_VERBOSE] typeSubst built: map[α2:float]
[DEBUG_MONO_VERBOSE] Cloning DictApp: method=gt
[DEBUG_MONO_VERBOSE]   Original DictRef: class=Ord, type=Int, NodeID=15
[DEBUG_MONO_VERBOSE]   Cloned DictRef: class=Ord, type=float, NodeID=42
```

### `DEBUG_OPERATOR_LOWERING=1` - Operator Resolution Tracing

**What it does**: Logs operator lowering decisions (BinOp/DictApp → Intrinsic).

**When to use**:
- Debugging operator dispatch issues
- Understanding which builtin is selected
- Tracking type-guided operator selection

### `DEBUG_PARSER=1` - Parser Token Tracing

**What it does**: Shows ENTER/EXIT for parser functions with current/peek tokens.

**When to use**:
- Debugging parser token position issues
- Understanding parser flow
- Tracking token consumption

**Example**:
```bash
$ DEBUG_PARSER=1 ailang run test.ail
[ENTER parseType] cur=IDENT(int) peek=,
[EXIT parseType] cur=IDENT(int) peek=,
[ENTER parseExpression] cur=IDENT(x) peek=+
[EXIT parseExpression] cur=IDENT(x) peek=+
```

## Combining Debug Flags

**Recommended combinations**:

```bash
# Development mode - catch bugs early + verbose output
$ DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 ailang run test.ail

# CI mode - strict checking only (no verbose output)
$ DEBUG_STRICT=1 make test

# Deep debugging - all flags
$ DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 DEBUG_OPERATOR_LOWERING=1 ailang run --debug-compile test.ail

# Parser debugging
$ DEBUG_PARSER=1 ailang run test.ail
```

## Quick Reference Table

### Environment Variables

| Flag | Purpose | Use When | Output |
|------|---------|----------|--------|
| `DEBUG_STRICT=1` | Fail loudly on unhandled cases | Development, CI | Panics with diagnostic |
| `DEBUG_MONO_VERBOSE=1` | Monomorphization tracing | Type issues | Specialization details |
| `DEBUG_OPERATOR_LOWERING=1` | Operator resolution | Dispatch issues | Builtin selection |
| `DEBUG_PARSER=1` | Token position tracing | Parser bugs | Token flow |

### CLI Flags

| Flag | Purpose | Use When |
|------|---------|----------|
| `--debug-compile` | Show compilation phases/timing (see also [Telemetry](/docs/guides/telemetry)) | Performance issues |
| `--debug-types` | Type inference debug output | Type mismatch errors |
| `--debug-types --node N` | Filter to specific node ID | Investigating specific node |
| `--trace` | Execution tracing | Runtime debugging |
| `-cpuprofile FILE` | Write Go CPU profile | Performance profiling |
| `-memprofile FILE` | Write Go memory allocation profile | Allocation profiling |

:::tip Performance
Tracing adds ~2x overhead on average and up to ~6× on data-intensive list
pipelines. For production workloads — and **always** for benchmark
measurements — disable with `AILANG_NO_TRACE=1`.
See [Telemetry: Performance Trade-off](/docs/guides/telemetry#performance-vs-observability-trade-off).
:::

### Latency Budget Workloads

`benchmarks/workloads/` holds six self-contained `.ail` programs that act as
release-gating latency probes — cold start, warm evaluator, type checking,
IO effect dispatch, and small/large `std/list` pipelines. They are the
canary suite for "is this commit slower than the last release on a realistic
program?" and the only suite whose p95 is treated as an SLO.

```bash
# 5 runs each, write benchmarks/latency_budgets.json
make bench-workloads

# 3 runs, dry-run, dump JSON to stdout
make bench-workloads-quick

# Single workload, verbose
tools/bench_workloads.sh --workload list_large --runs 10 --verbose
```

The harness always sets `AILANG_NO_TRACE=1`, runs from the project root with
relative paths (so canonical module IDs match), and discards the first run
as a warm-up when N≥3. Targets and the dev-pool balance live in
[`benchmarks/budget_ledger.md`](https://github.com/sunholo/ailang/tree/dev/benchmarks/budget_ledger.md);
the on-disk JSON is regenerated by `make bench-workloads` and is **not**
hand-edited. Runs from one machine class are not comparable to another —
the JSON records `cpu`, `os`, `arch`, and `go` so a different machine's
measurements never overwrite a baseline silently.

## CLI Debug Flags

In addition to environment variables, AILANG CLI provides debug flags:

```bash
# Show compilation phases
ailang run --debug-compile file.ail

# Enable execution tracing
ailang run --trace file.ail

# Type-check only (no execution)
ailang check file.ail

# Show module interface
ailang iface mymodule

# Type inference debugging (v0.5.11+)
ailang run --debug-types file.ail
ailang run --debug-types --node 42 file.ail  # Filter to specific node
```

### `--debug-types` - Type Inference Debugging (v0.5.11+)

**What it does**: Shows detailed type inference information including:
- Substitution map (type variable → resolved type)
- Constraints (type class constraints and their resolution status)
- CoreTI entries (type information for each Core AST node)
- Origins/provenance (where each type came from)

**When to use**:
- Understanding why a type was inferred
- Debugging type mismatch errors
- Investigating constraint resolution
- Verifying type annotations are applied correctly
- Answering "why does this have type X?"

**Demo file**: See [`examples/runnable/debug_types_demo.ail`](https://github.com/sunholo-data/ailang/blob/main/examples/runnable/debug_types_demo.ail) for a complete example demonstrating all debugging sections.

**Example**:
```bash
$ ailang run --debug-types --caps IO --entry main examples/runnable/debug_types_demo.ail
=== Type Inference Debug ===

[Substitution Map]
  α1 → α2
  α5 → α7 → α11 (CHAIN)
  α6 → α10
  α8 → α6 -> α7 (direct)

[Constraints]
  Added:
    Num α1 at node 9
    Num α3 at node 14
    Fractional α41 at node 60
  Resolved:
    Num Int →  at node 9
    Fractional Float →  at node 60

[CoreTI Entries]
  NodeID 1: string -> () ! {IO}
  NodeID 9: int
    Constraint: Num → add
  NodeID 60: float
    Constraint: Fractional (resolved)
  ...
```

**Filtering by node**:
```bash
# Show type info only for node ID 9
$ ailang run --debug-types --node 9 --caps IO examples/debug_types_demo.ail

[Constraints]
  Added:
    Num α1 at node 9
  Resolved:
    Num Int →  at node 9

[CoreTI Entries]
  NodeID 9: int
    Constraint: Num → add
```

**Output sections**:
- **Substitution Map**: Shows type variable substitutions (α → β → int means α resolved to β which resolved to int)
- **Constraints**: Type class constraints (Num, Eq, Ord) and whether they're resolved
- **CoreTI Entries**: Every Core AST node's inferred type, constraints, and origins
- **Origins**: Where the type came from (annotation, literal, inferred, defaulted, etc.)

### Understanding Origins (Provenance)

The `Origins:` section answers "why does this expression have this type?" Each origin shows:
- **Kind**: How the type was determined (annotation, literal, inferred, defaulted, from_use, from_pattern)
- **Note**: Human-readable explanation
- **Location**: Source file:line:column when available

**Origin kinds**:
| Kind | Meaning | Example |
|------|---------|---------|
| `annotation` | Explicit type annotation | `let x: int = 42` |
| `literal` | Inferred from literal value | `3.14` → float |
| `inferred` | Created during type inference | Fresh type variable α |
| `defaulted` | Type variable defaulted | Num α defaulted to int |
| `from_use` | Inferred from call site | Function applied to int |
| `from_pattern` | Inferred from pattern match | `Some(x)` binds x to inner type |

**Example with multiple origins**:
```
NodeID 42: int
  Raw: α1
  Resolved: int
  Origins:
    - inferred: fresh type variable
    - defaulted: defaulted to int (Num constraint)
```
This shows that node 42 started as a type variable α1, then was defaulted to `int` because of a Num constraint.

## Troubleshooting Workflows

### Scenario 1: "Why is my float becoming int?"

**Symptom**: You expected `float` but got `int` arithmetic.

```ailang
-- Problem: add(3.14)(2.71) gives unexpected result
let add = \x. \y. x + y
let result = add(3.14)(2.71)  -- Expected: 5.85, got: 5?
```

**Debug workflow**:
```bash
$ ailang run --debug-types myfile.ail
```

**What to look for**:
```
[CoreTI Entries]
  NodeID 5: int -> int -> int    -- The add function got type int!
    Constraint: Num → add
    Origins:
      - inferred: fresh type variable
      - defaulted: defaulted to int (Num constraint)  -- HERE'S THE PROBLEM
```

**Root cause**: The Num constraint defaulted to `int` before the float literals were seen.

**Fix**: Add type annotations:
```ailang
let add: float -> float -> float = \x. \y. x + y
```

### Scenario 2: "Type mismatch at line X"

**Symptom**: Error says types don't match but you're not sure why.

```
Error: type mismatch at line 15: expected int, got α42
```

**Debug workflow**:
```bash
$ ailang run --debug-types --node 42 myfile.ail
```

**What to look for**:
```
NodeID 42: α42
  Origins:
    - inferred: fresh type variable
```

**Root cause**: Node 42 is still a type variable (α42) - it was never unified with a concrete type.

**Fix**: Check that the expression at node 42 is actually used with concrete types, or add an annotation.

### Scenario 3: "Which operator is being called?"

**Symptom**: `x + y` behaves unexpectedly for your types.

**Debug workflow**:
```bash
$ ailang run --debug-types myfile.ail | grep -A2 "Constraint: Num"
```

**What to look for**:
```
  NodeID 9: int
    Constraint: Num → add     -- Shows which method resolved
  NodeID 14: int
    Constraint: Num → mul     -- mul was selected for *
```

The `→ add` shows the constraint resolved to the `add` method. If it says `(resolved)` without a method, the constraint was satisfied but method selection may differ.

### Scenario 4: "Understanding polymorphic function types"

**Symptom**: Function type shows type variables (α, β) instead of concrete types.

```bash
$ ailang run --debug-types myfile.ail
```

**What to look for**:
```
NodeID 31: α22
  Origins:
    - inferred: fresh type variable
NodeID 35: α22
  Origins:
    - inferred: fresh type variable
```

**Interpretation**: Multiple nodes share the same type variable (α22), meaning they must have the same type. This is polymorphism working correctly - the type will be specialized at each call site.

### Problem: Type Inference Issues

```bash
# 1. Use --debug-types to see all type information (v0.5.11+)
ailang run --debug-types problematic.ail

# 2. Filter to specific node if you know the ID
ailang run --debug-types --node 42 problematic.ail

# 3. Check types at each phase
ailang check problematic.ail

# 4. Enable monomorphization debugging
DEBUG_MONO_VERBOSE=1 ailang run --debug-compile problematic.ail

# 5. Check operator resolution
DEBUG_OPERATOR_LOWERING=1 ailang run problematic.ail
```

### Problem: Parser Not Recognizing Syntax

```bash
# 1. Trace token flow
DEBUG_PARSER=1 ailang run problematic.ail

# 2. Check for lexer issues
# (Lexer never generates NEWLINE tokens!)

# 3. Use parser-developer skill for conventions
```

### Problem: Silent Failures in Compiler Pass

```bash
# 1. Enable strict mode
DEBUG_STRICT=1 ailang run problematic.ail

# 2. Will panic on unhandled cases with diagnostic
# 3. Add missing cases to switch statement
```

### Problem: Unexpected Operator Behavior

```bash
# 1. Check type defaulting (Num typeclass defaults to int)
ailang check problematic.ail

# 2. Add type annotations
# let add: float -> float -> float = \x. \y. x + y

# 3. Enable operator tracing
DEBUG_OPERATOR_LOWERING=1 ailang run problematic.ail
```

## Keeping `ailang` Up to Date

**After making code changes to the ailang binary:**
```bash
make quick-install  # Fast reinstall (recommended for development)
# OR
make install        # Full reinstall with version info
```

**Important**: The `ailang` command in your PATH points to the system install, NOT the local build. Always run `make install` or `make quick-install` after building to update the system binary.

**For local testing without install:**
```bash
./bin/ailang <command>  # Use local build directly
```

## Development Tools

```bash
# Code quality
make lint                 # Run golangci-lint
make fmt                  # Format all Go code
make vet                  # Run go vet
make test-coverage        # Run tests with coverage

# File organization
make check-file-sizes     # Fails CI if any file >800 lines
make report-file-sizes    # Show files >500 lines

# Documentation
make doc PKG=<package>    # Show package documentation
```

## See Also

- [Telemetry & Tracing](/docs/guides/telemetry) - Distributed tracing for performance analysis and debugging
- [Evaluation Framework](/docs/guides/evaluation) - Debugging failed AI benchmarks
- [Development Guide](/docs/guides/development) - Full development workflow
- [Known Limitations](/docs/reference/limitations) - Current limitations and workarounds
