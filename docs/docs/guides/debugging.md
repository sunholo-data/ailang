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
  log("processing ${show(x)}");     -- ghost: invisible to callers
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
| `DEBUG_CODEGEN=1` | Record type fallback warnings | Codegen issues | Fallback warnings |
| `DEBUG_APPROVAL_WATCHER=1` | ApprovalWatcher polling | Coordinator approval flow | Poll tracing |
| `DEBUG_CONCURRENCY=1` | Per-request evaluator Fork/Call/Done tracing | Concurrency issues | Goroutine IDs |

### Ollama Streaming Timeouts (v0.34.0)

Local-model turns on the eval rig are long: a `qwen3.6:35b` thinking turn can run
well past the buffered path's 300s default cap. A single total-duration clock
cannot tell **"the model is thinking hard"** from **"the connection is wedged"**,
so it is either too short (a healthy turn is killed at 4m59.97s) or too long
(a wedged stream hangs for hours). The streaming `/v1` path replaces one coarse
clock with three sharp ones.

| Flag | Purpose | Use When | Default |
|------|---------|----------|---------|
| `AILANG_OLLAMA_V1_STREAM=1` | Opt into the streaming ollama `/v1` path. Exactly `"1"` opts in — anything else keeps today's buffered path with byte-identical requests | Long local-model turns timing out mid-generation | **off** |
| `AILANG_OLLAMA_IDLE_TIMEOUT_SEC` | Max silence **between** bytes. Trips a typed `idle-timeout` — this is the "wedged" detector | A hung stream should fail fast instead of burning the whole budget | `120` |
| `AILANG_OLLAMA_TTFT_TIMEOUT_SEC` | Max silence **before the first byte**. Long on purpose: a cold 35B load under GPU contention legitimately takes minutes | Cold-start false trips | `600` |
| `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` | Total budget for the whole call — **but its meaning changes with the flag**, see below | Bounding worst-case wall clock | `300` off / `3600` on |

:::warning `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` means two different things

- **Flag off (buffered `/v1`)** — an HTTP client timeout and whole-call cap,
  default **300s**, where `0` (or negative) means *no timeout at all*.
- **Flag on (streaming `/v1`)** — the **mandatory hard deadline** on the stream,
  default **3600s**, where `0`, a negative, or an unparseable value is
  **rejected at client construction** with a typed configuration error and **no
  HTTP request is sent**. An unbounded stream is precisely the hang the guard
  exists to prevent, so there is deliberately no "disabled" setting here.

The practical consequence: a value that was a safe raise on the buffered path
becomes a **ceiling** on the streaming path. Auditing a config, read the flag
first. This is why the rig's launchd plists dropped their `1800` stopgap pin in
the same edit that added the flag — 1800s sits below the worst-case legitimate
request and would have re-created the timeout it was added to fix.
:::

:::danger Two delivery sites — clearing one does not clear the other

On the eval rig this variable reaches a job by **two independent paths**, and
they must be audited separately:

1. **The plist**, via `EnvironmentVariables` in `tools/launchd/*.plist`.
2. **The launchd user-domain global**, set once by `launchctl setenv` and
   inherited by every job in the domain. **No plist edit touches it**, it
   survives reboots of the job, and it is invisible to any `grep` over the repo.

Editing the plists alone is therefore *not* enough to remove a pin — it silently
keeps applying from site 2. Check the live value before trusting a config:

```bash
launchctl getenv AILANG_OLLAMA_HTTP_TIMEOUT_SEC   # empty == not pinned
launchctl getenv AILANG_NOT_A_REAL_VAR            # control: also empty
```

Pair the check with a known-unset name as above, so an empty answer is a
measurement rather than a broken command.

**Clearing site 2 is ordered, and the wrong order is expensive.** Site 2 is
load-bearing while the streaming flag is off: with the flag unset the buffered
path runs, `ollamaV1Timeout()` falls back to its **300s** default, and the domain
global is the only thing raising it for invocations a plist does not cover. So
turn the flag on **first** — the plists in `tools/launchd/` are *source*, and the
installed copies under `~/Library/LaunchAgents/` are regular files updated by a
manual `cp` + `launchctl load`, so editing the repo changes nothing on the rig —
and only then:

```bash
launchctl unsetenv AILANG_OLLAMA_HTTP_TIMEOUT_SEC
```

Doing it the other way round drops those calls back to 300s and re-creates the
timeout the pin was added to fix.

This is not hypothetical. Both plists were cleaned up and every grep read green
while the domain global still held `1800`, so streamed requests kept logging
`hard_deadline_sec = 1800` / `effective_deadline_sec = 1800` instead of the 3600s
default. The `effective_deadline_sec` read-back is what exposed it — which is
exactly why that field is read back from the context rather than reported from
the configured value.
:::

### Ollama Context Window (`AILANG_OLLAMA_NUM_CTX`)

Pins ollama's `num_ctx`. **Unset (the default) sends no `num_ctx` at all**, so
ollama sizes the context from the model (measured 2026-08-13: 262144 for
`qwen3.6:35b-a3b-mxfp8`) — matching the `/v1` lanes pi and opencode already use.
Both option maps in `internal/ai/ollama` previously hardcoded **8192**, below the
28k–44k-token prompts the eval harness sends these models.

Affects the non-tool paths only (`Generate`, tool-less chat, and the legacy
native tool path incl. motoko's `compaction_ai`); motoko's tool-calling turns
route via `/v1`, where `num_ctx` is not expressible. Raise or lower only for
VRAM: the KV cache scales with it — see **Ollama Memory Budget on the Rig**
below for the machine-level bound and the panic that established it.

When request logging is enabled — `AILANG_OLLAMA_LOG_REQUESTS=<path>`, or a path
written to the `~/.ailang/state/ollama-log-requests` **sentinel file** whose
contents are the dump path (it exists because harnesses like motoko's
`bun`→`ailang` process chain drop our custom env, but `HOME` always propagates)
— each streaming request appends a `"kind":"stream_metrics"` JSONL record with
`ttft_ms`, `max_gap_ms`, `total_ms`, `bytes`, the delta counts, and both
`hard_deadline_sec` (configured) and `effective_deadline_sec` (**read back** from
the request context). Those numbers are the evidence for whether the three
defaults above are right. If the two deadline fields disagree, something upstream
is capping the stream — a run killed at ~300s with the flag on means the deadline
never reached the wire, which is a harness bug, not a model one.

:::caution The sentinel makes the log a shared global sink

Because the sentinel is keyed on `HOME`, **every** `ailang` process on the machine
writes to the same file — including `go test ./internal/ai/...`, whose `httptest`
fake servers emit real `stream_metrics` records. A unit-test run during a field
capture therefore pollutes the capture. This happened on 2026-08-11: 10
test-generated records landed in the middle of a live rig measurement.

They were separable only by their **window fields**: tests use non-production
values (`idle_window_sec: 1`, `ttft_window_sec: 2` or `3`) against a real run's
`120` / `600`. Filter on those before analysing a capture:

```bash
jq -c 'select(.kind=="stream_metrics" and .idle_window_sec==120)' "$LOG"
```

Prefer an explicit `AILANG_OLLAMA_LOG_REQUESTS=<path>` per capture where the
harness propagates env; reach for the sentinel only when it does not, and remove
it when the capture ends.
:::

### Ollama Memory Budget on the Rig (`OLLAMA_GPU_OVERHEAD`, `OLLAMA_CONTEXT_LENGTH`)

The section above tunes `num_ctx` per request for **quality** — don't truncate the
28k–44k-token prompts the harness sends. These two **server-side** variables bound
what ollama may consume on the *machine*, and they are what stands between a local
eval and a kernel panic.

Measured 2026-09-03, after the rig panicked at 02:23 (incident `561F0912`):

- ollama claims **84% of unified memory** as VRAM — `total="107.5 GiB"` of 128 GiB —
  and by default reserves nothing for anything else: `overhead="0 B"`.
- Because that budget looks large, it auto-selects the model's full native context:
  `msg="vram-based default context" total_vram="107.5 GiB" default_num_ctx=262144`.
  This is not ollama over-reaching — 262144 *is* `qwen3.8:27b`'s trained maximum.
- At 256k that runner peaked at **90.39 GiB** (max of 2,322 `peak memory` samples),
  leaving ~38 GB for the desktop, the agent fleet and the eval harness. Memory ran
  out, the pager stopped making progress (20 pages reclaimed of 3,088 wanted), and
  the hardware watchdog panicked the machine.

**`OLLAMA_GPU_OVERHEAD` is admission control, not a runtime limit.** It is a
bookkeeping subtraction inside the scheduler, not an allocation — nothing is held,
and every other process still sees the full machine:

```
available="45.3 GiB"   free="77.8 GiB"   overhead="32.0 GiB"
```

`free` is real free VRAM; `available = free − overhead` is only what ollama will
*consider* when deciding whether a model fits. A model admitted under that budget
can still grow past it as the KV cache fills — which is precisely what the 90 GiB
peaks were.

**`OLLAMA_CONTEXT_LENGTH` is the runtime bound**, because KV size is a direct
function of context length. Both are required: the reservation stops ollama loading
something too big, the context length stops what it *did* load from growing into
the reservation.

Current rig values (`~/Library/LaunchAgents/dev.ollama.serve.plist`):

| Variable | Value | Why |
|----------|-------|-----|
| `OLLAMA_GPU_OVERHEAD` | `34359738368` (32 GiB) | headroom for desktop + agent fleet + harness |
| `OLLAMA_CONTEXT_LENGTH` | `131072` | halves KV against the 256k native max; largest prompt ever observed was 108,738 tokens |
| `OLLAMA_MAX_LOADED_MODELS` | `2` | keeps the embedder resident — see "Embedder evicts the eval LLM" |

Sizing a new model is one inequality: `weights + KV(context) < available`. Raising
context is safe only while that holds. A 1M-context model's KV alone would exceed
this machine no matter how these are set — at that point it is hardware talking,
not configuration.

:::caution No swap on the rig — there is no warning phase

`/private/var/vm` is empty and ollama logs `free_swap="0 B"` on every sample. A
machine with swap thrashes audibly before it dies and someone notices; this one
goes straight from healthy to wedged. The kernel's own `memoryPressure` flag also
read **false** throughout the panic, so never key a guard to it — use free pages
and reclaim rate instead.
:::

:::caution `OLLAMA_CONTEXT_LENGTH` is applied but UNPROVEN (2026-09-03)

After the change ollama still logged `default_num_ctx=262144` against the reduced
75.5 GiB budget, and small-prompt probes cannot discriminate — a 33-token prompt
populates almost no KV, so load peak barely moves either way. The discriminating
test is a real large-context eval: a peak near **59 GiB** means the cap is working,
near **90 GiB** means it is not. Do not record this as fixed until that
measurement exists.
:::

### CLI Flags

| Flag | Purpose | Use When |
|------|---------|----------|
| `--debug-compile` | Show compilation phases/timing (see also [Telemetry](/docs/guides/telemetry)) | Performance issues |
| `--debug-types` | Type inference debug output | Type mismatch errors |
| `--debug-types --node N` | Filter to specific node ID | Investigating specific node |
| `--trace` | Type-defaulting tracing | Type inference debugging |
| `--trace-tier` | Tracing tier: `off`, `standard`, `deep` (v0.12.0+) | Observability control |
| `-cpuprofile FILE` | Write Go CPU profile | Performance profiling |
| `-memprofile FILE` | Write Go memory allocation profile | Allocation profiling |

:::tip Performance
Function-level tracing (tier `deep`) adds ~2x overhead on average and up to ~6×
on data-intensive list pipelines. As of v0.12.0 the **default is `standard`**,
which skips per-call spans. For benchmark measurements, disable tracing entirely
with `AILANG_NO_TRACE=1` (legacy) or `--trace-tier off`. Opt into per-call spans
explicitly with `--trace-tier deep` or `AILANG_TRACE=deep` when you need them.
See [Telemetry: Tracing tiers](/docs/guides/telemetry#tracing-tiers).
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

## Sandbox Debugging (`AILANG_FS_SANDBOX`)

When `AILANG_FS_SANDBOX` is set, all FS operations are restricted to a root directory. `exists`, `isDir`, and `isFile` silently return `false` for out-of-sandbox paths (correct public contract — they don't throw). This can cause programs with fallback logic to silently degrade to defaults with no error or warning.

### Symptom

A program reads config, gets empty values (`extensions.order = []`, `cost_rates = {}`), and no error is reported. The real cause is that the config file path escapes the sandbox and `fileExists(path)` returned `false`.

### 3-step diagnosis

**Step 1 — identify which paths are being rejected:**
```bash
AILANG_FS_SANDBOX=/tmp/task AILANG_FS_SANDBOX_DEBUG=1 ailang run your_program.ail
# stderr output:
# [ailang/sandbox] REJECT exists("/home/mark/.motoko/config/default/config.json") → escapes sandbox "/tmp/task" (returns false)
```

**Step 2 — confirm the resolution interactively:**
```bash
AILANG_FS_SANDBOX=/tmp/task ailang sandbox-check /home/mark/.motoko/config/default/config.json
# sandbox:  /tmp/task
# path:     /home/mark/.motoko/config/default/config.json (absolute)
# result:   REJECT — escapes sandbox "/tmp/task"
#           exists/isDir/isFile → false
#           readFile/writeFile/etc → error
```

**Step 3 — fix** by ensuring the config path is within the sandbox, or by setting an env var that directs the program to a fallback within the sandbox:
```bash
# Example: motoko uses MOTOKO_REPO to find config when workdir is a scratch dir
MOTOKO_REPO=/tmp/task AILANG_FS_SANDBOX=/tmp/task ailang run your_program.ail
```

### `ailang sandbox-check` reference

```bash
ailang sandbox-check <path>   # ALLOW/REJECT + resolved path, exits 0/1
```

Exit 0 = ALLOW (path is within sandbox or sandbox not configured).  
Exit 1 = REJECT (path escapes sandbox).

No `AILANG_FS_SANDBOX` set → prints "no sandbox configured", exits 0.

### Trace-based diagnosis

Under `AILANG_TRACE=deep`, sandbox rejections are recorded as `FS.<op>.sandbox.reject` events in the OTEL trace collector and visible in `ailang trace list`:

```bash
AILANG_TRACE=deep ailang run your_program.ail --emit-trace auto
ailang trace list --hours 1   # look for FS.exists.sandbox.reject events
```

## Debugging an LLM call from the PROVIDER's side (OpenRouter Broadcast)

For any OpenRouter call the rig makes, there is a second, independent record of what
happened — pushed by OpenRouter itself into the prod observatory
(`M-OPENROUTER-BROADCAST-INGEST`, v0.33.1). Reach for it **before** theorising from our
own logs: it is the only instrument that shows the request we actually sent, as opposed
to the one we believe we configured.

```bash
curl -s "https://dashboard.ailang.sunholo.com/api/observatory/spans?limit=1000&start_after=2026-08-18T00:00:00Z&start_before=2026-08-19T00:00:00Z" > /tmp/spans.json

# One row per generation: model, provider host, finish_reason, token split.
jq -r '.[] | select(.name=="LLM Generation") | [
    .start_time, (.duration_ms|tostring)+"ms",
    (.attributes["gen_ai.request.model"]//"-"),
    (.attributes["trace.metadata.openrouter.provider_name"]//"-"),
    (.attributes["gen_ai.response.finish_reason"]//"NONE(cancelled)"),
    "out="+((.attributes["gen_ai.usage.output_tokens"]//0)|tostring),
    "reasoning="+((.attributes["gen_ai.usage.output_tokens.reasoning"]//0)|tostring)
  ] | @tsv' /tmp/spans.json | sort
```

Attributes live on the **`LLM Generation`** span (~92 of them). The sibling `generation`
and `provider attempt N: <host>` spans carry only 4 and 10 — querying those and concluding
"the traces have no model data" is a mistake already made once.

The two highest-value fields:

```bash
# 1. The REQUEST WE ACTUALLY SENT — budget, reasoning config, tool count.
#    This is how the pi lane was found still sending max_completion_tokens=32000
#    while its config declared 65536.
jq -rc '.[] | select(.name=="LLM Generation") | (.attributes["gen_ai.completion"]|tostring|fromjson? // {}) | .rawRequest | select(.!=null)' /tmp/spans.json | head -1

# 2. The COMPLETION, split into content and reasoning. `{"completion":"","reasoning":"..."}`
#    with output_tokens == reasoning_tokens is the reasoning-stall signature
#    (error_category=reasoning_stall) — the model engaged and never answered.
jq -rc '.[] | select(.name=="LLM Generation") | (.attributes["gen_ai.completion"]|tostring|fromjson? // {}) | select(.reasoning!=null) | {content: .completion, reasoning_head: (.reasoning|tostring|.[0:160])}' /tmp/spans.json | head -5
```

**Finding the failures fast.** A cancelled generation has no `finish_reason` and
`cancelled: true`, and OpenRouter does not bill it — so a dead run costs `$0` and leaves
no trace in our own spend ledger:

```bash
jq -r '.[] | select(.name=="LLM Generation")
       | select((.attributes["gen_ai.response.finish_reason"]//null)==null)
       | [.start_time, (.attributes["gen_ai.request.model"]//"-"),
          (.attributes["span.metadata.openrouter_generation.app.title"]//"-"),
          "reasoning="+((.attributes["gen_ai.usage.output_tokens.reasoning"]//0)|tostring)] | @tsv' /tmp/spans.json
```

### Asserting what pi actually sends

`scripts/check_pi_wire_budget.sh` (or `make check-pi-wire-budget`) makes one real pi call
and reads the request back from the Broadcast trace, comparing the budget pi's registry
declares against what the wire carried. It is deliberately NOT a CI gate: it costs a
fraction of a cent, needs `OPENROUTER_API_KEY`, and depends on ingest.

It exists because **every other guard on that number compares one config file to another,
and none of them is the wire.** pi-ai's `buildBaseOptions` clamps to
`Math.min(model.maxTokens, 32000)` whenever the caller passes no explicit `maxTokens` —
which pi-coding-agent never does for main agent turns — so a 2x understatement sat green
in CI for weeks while `TestPiModelsConfigMatchesRegistry` happily compared 65536 to 65536.
Measured: declaring 20000 sends 20000; declaring 65536 sends 32000.

### Gotchas, each one measured

- **An empty window is not a broken pipe.** Ingest showed zero spans for 08-23..08-26 and
  the key reported `usage_weekly: 0` — there was simply no traffic. Prove liveness with a
  positive control (make one call, re-query) before reporting an outage.
- **`/api/v1/activity` returns 403.** The account key is an inference key
  (`is_management_key: false`). `/api/v1/key`, `/api/v1/credits` and
  `/api/v1/generation?id=<gen-id>` all work; account-wide lookup by time does not. Use the
  observatory, which is time-indexed anyway.
- **`finish_reason` proves nothing about whether work happened.** It read `length` before
  2026-08-13, a clean `stop` at 625 tokens after, and was absent on the cancelled runs. It
  fired on 0 of 4 real pi-lane failures. Assert on output, never on how the turn ended.
- **pi's NDJSON size is not a proxy for tokens.** `message_update` replays the whole
  accumulated message per delta (pi 0.73.1, `dist/core/agent-session.js:421-427`), so bytes
  are *quadratic*: 7,130 reasoning tokens produced 330 MB. Use
  `scripts/mission_pi_run.sh`, which filters the updates out and gives a typed verdict.
- **pi hangs forever on an open stdin.** Redirect from a file or `/dev/null`; a run that
  produces zero bytes of stdout *and* stderr is this, not a provider problem.

## See Also

- [Telemetry & Tracing](/docs/guides/telemetry) - Distributed tracing for performance analysis and debugging
- [Evaluation Framework](/docs/guides/evaluation) - Debugging failed AI benchmarks
- [Development Guide](/docs/guides/development) - Full development workflow
- [Known Limitations](/docs/reference/limitations) - Current limitations and workarounds
