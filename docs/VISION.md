---
reviewed: 2026-11-26
reviewBy: 2027-05-26
---

# AILANG Vision — The First AI-Native Language

**AILANG isn't designed for humans who code.**
**It's built for AIs that reason, refactor, and verify.**

## Why AILANG Exists

Human programming languages optimize for familiarity, multiple idioms, and implicit shortcuts. AILANG optimizes for:

- **Determinism** — predictable, reproducible execution across engines
- **Reflectivity** — programs and interfaces that machines can inspect (`ailang inspect`, `ailang iface`)
- **Compositional Reasoning** — explicit effects, contracts, and exhaustive ADTs that machines can verify and repair

---

## Why Not Just Use Python or TypeScript?

### 1. Implicit State Breaks AI Reasoning

Global mutable state in Python/JS is invisible to the type checker, so identical calls return different values. In AILANG, state is threaded explicitly in pure functions:

```ailang
pure func add(x: int, state: int) -> (int, int) =
  let newState = state + x in
  (newState, newState)
```

### 2. Multiple Idioms Create Non-Deterministic Generation

Python offers five ways to transform a list (comprehensions, `map`, loops, lambdas, unpacking). AILANG provides one canonical form (`map(f, items)`), narrowing the search space for consistent synthesis.

### 3. Runtime Type Errors Become Compile-Time Proofs

Instead of erased `any` types crashing at runtime, AILANG enforces structural record types, nominal ADTs, and exhaustive pattern matching statically:

```ailang
pure func readValue(data: { field: { value: int } }) -> string =
  show(data.field.value)
```

### 4. Exhaustive Pattern Matching Without Version Drift

On the `state_machine_elevator` benchmark, Python's `match`/`case` syntax drifted across runtime versions while AILANG's single `match` construct enforces exhaustiveness at compile time (`TC_MATCH_002`).

---

## Current Capabilities (`v0.64.1`, Pre-1.0)

- ✅ **Explicit Effect Tracking** — `! {IO, FS, Net, Clock, AI, SharedMem, Declassify}` with `--caps` and `@limit=N` budgets
- ✅ **Algebraic Data Types & Row Polymorphism** — exhaustive pattern matching, nominal type identity, Hindley-Milner inference
- ✅ **Z3 Contract Verification** — `ailang verify` for `requires`/`ensures` with `--int-model=int64`, `Tuple0..8`, and AILANG counterexamples
- ✅ **IFC Taint Labels (Experimental)** — compile-time `T<label>`, `T{not LABEL}`, and `! {Declassify}`
- ✅ **Two Aligned Engines + WASM** — reference evaluator, ~40× faster `--bytecode` VM, and browser WebAssembly
- ✅ **Agent Tooling Loop** — `ailang prompt`, `ailang inspect --format=agent`, `ailang check --fix`, `ailang lsp`, hosted MCP, and JSONL/OTEL traces

---

## Next Horizons — `1.0` and `1.1+`

### `1.0.0` — A Promise We Can Keep

`1.0.0` answers *"what do we promise not to break?"*, not *"will people choose AILANG?"*. Its gates are: the [Stability Promise](docs/reference/stability.md) ratified against today's surface, zero open `priority:P0` issues, Stable package manifest/lockfile/registry semantics, Stable hosted executor MCP tools, and published benchmark/simplicity metrics. See [Why 1.0](docs/why-1-0.mdx).

### `1.1+` — AI-Native Platform

- **[M-AGENT](../design_docs/planned/v1_1_0/m-agent-orchestration.md)** — `std/agent` with `! {Agent}` effect and cost/turn budgets
- **[M-ENTROPY](../design_docs/planned/v1_1_0/m-entropy-budgets.md)** — first-class entropy budgets
- **[M-CSP / Session Types](../design_docs/planned/v1_1_0/m-csp-session-types.md)** — static effect-typed task graphs for structured concurrency
- **[M-EFFECT-REFINEMENT](../design_docs/planned/v1_1_0/m-effect-refinement.md)** — parameterised Clock/Net/FS capability scoping

---

## What AILANG Deliberately Excludes

- ❌ **Unstructured concurrency** — no ad-hoc threads, goroutines, or data races (structured task graphs arrive via `M-CSP`)
- ❌ **Implicit behaviors** — all effects, imports, and authority are explicit
- ❌ **Multiple syntaxes** — one canonical way to express each concept
- ❌ **Ambient authority** — every capability requires an explicit grant

*(Note: `ailang lsp --stdio` and `ailang check --fix` ARE supported so both editors and AI agents share one diagnostic and quick-fix engine.)*

---

## Roadmap & Resources

- **Live Roadmap**: [docs/roadmap/index.md](docs/roadmap/index.md) and [design_docs/planned/](../design_docs/planned/)
- **Live Benchmarks**: [Benchmark Dashboard](/docs/benchmarks/performance)
- **Web Vision Page**: [docs/vision.mdx](docs/vision.mdx)
- **Get Involved**: [Getting Started](/docs/guides/getting-started) · [CONTRIBUTING.md](CONTRIBUTING.md) · [GitHub](https://github.com/sunholo-data/ailang)

> **AILANG — When the coder is the model.**
