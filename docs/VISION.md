# AILANG Vision — The First AI-Native Language

**AILANG isn't designed for humans who code.**
**It's built for AIs that reason, refactor, and verify.**

## Why AILANG Exists

Human programming languages optimize for:
- **Comfort** — familiar syntax, IDE support, autocompletion
- **Ambiguity** — multiple ways to express the same thing
- **Speed** — fast typing, shortcuts, implicit behaviors

AILANG optimizes for:
- **Determinism** — predictable, reproducible execution
- **Reflectivity** — programs that can explain themselves
- **Compositional Reasoning** — machines that prove, transform, and reuse code

Every feature exists to make programs easier for machines to **analyze, verify, and improve**.

---

## Why Not Just Use Python?

Existing languages weren't designed for machine code generation. Here's what breaks:

### Problem 1: Implicit State Breaks AI Reasoning

**Python/JavaScript:**
```python
total = 0  # Global state - invisible to type checker

def add(x):
    global total
    total += x
    return total

add(5)  # Returns 5
add(5)  # Returns 10 (same input, different output!)
```

**AI struggles:** Can't trace effects. Can't prove correctness. Non-deterministic behavior.

**AILANG:**
```ailang
fn add(x: int, state: int) -> (int, int) =
  let newState = state + x;
  (newState, newState)

add(5, 0)  -- Returns (5, 5)
add(5, 5)  -- Returns (10, 10) - explicit state threading
```

**AI wins:** All dependencies visible in types. Referentially transparent. Provable.

---

### Problem 2: Multiple Correct Answers Create Non-Deterministic Generation

**Python - 5 ways to transform a list:**
```python
# Which will the AI choose? Inconsistent across runs!
result = [f(x) for x in items]          # Comprehension
result = list(map(f, items))            # map
result = []; [result.append(f(x)) for x in items]  # Loop
result = list(map(lambda x: f(x), items))  # Lambda
result = [*map(f, items)]               # Unpack
```

**AILANG - One canonical form:**
```ailang
result = map(f, items)
-- That's it. One way. Deterministic generation.
```

---

### Problem 3: Runtime Type Errors Are Silent Failures

**TypeScript - Types erased at runtime:**
```typescript
function process(data: any) {
    return data.field.value.toString();  // Crash if structure wrong!
}
```

**AILANG - Types enforced statically:**
```ailang
fn process(data: Record { field: Record { value: int } }) -> string =
    intToString(data.field.value)
-- Type checker PROVES this can't crash
```

---

## Current Capabilities — v0.3.x: The Deterministic Core

AILANG today provides:

- ✅ **Explicit Effect Tracking** — `!: IO,FS` declares all side effects in types
- ✅ **Total Functions** — no runtime crashes, exhaustive pattern matching
- ✅ **Pure Evaluation** — deterministic semantics for all expressions
- ✅ **Algebraic Data Types** — composable, type-safe data structures
- ✅ **JSON Integration** — structured data parsing and encoding
- ✅ **Comprehensive Testing** — >2,800 passing tests ensure correctness

These features guarantee **predictable execution** and **analyzable behavior** —
the foundation for AI-driven code reasoning.

---

## Proven Results — M-EVAL Benchmarks

**Real data from 264 benchmarks across GPT-5, Claude Sonnet 4.5, Gemini 2.5 Pro:**

| Metric | Python-Style Code | AILANG v0.3.14 | Improvement |
|--------|-------------------|----------------|-------------|
| **Compile Success** | ~45% | 67% | **+49%** |
| **Type Safety** | N/A (dynamic) | 89% | **∞** |
| **Effect Safety** | N/A (untracked) | 94% | **∞** |
| **Deterministic Output** | ~62% | 91% | **+47%** |

### What This Means

**Without AILANG:** AI generates code that compiles less than half the time. Runtime errors are unpredictable. Side effects are invisible.

**With AILANG:** AI success rate increased by 49%. Type checker catches errors before runtime. Effect system makes all I/O explicit and verifiable.

**Key Insight:** Deterministic semantics + explicit effects = measurably better AI code generation.

*See [static/benchmarks/latest.json](static/benchmarks/latest.json) for current benchmark results.*

---

## Next Milestone — v0.4 "Reflective Core"

**Goal:** A compact, self-describing language runtime that AIs can introspect and modify.

### What v0.4 Enables

| Capability | Benefit for AI Systems |
|------------|------------------------|
| **Deterministic Tooling** | Canonical IR, effect tracing, automated code repair (`normalize → suggest → apply`) |
| **Total Recursion** | Functional completeness without loops (`fold`, `unfold`, `iterateN`) |
| **Polymorphic Safety** | Reliable generics: `List[T]`, `Result[T,E]`, `Option[T]` |
| **Effect Sugar** | Clean syntax (`!: IO,FS`) with precise diagnostics |
| **Runtime Reflection** | `:type`, `reflectType`, `reflectEffect` for self-introspection |

**The result:** Programs that can explain, verify, and improve themselves.

---

## Future Horizons — Multi-Agent Coordination

Once reflection and normalization are stable, AILANG will enable **multi-agent cooperation**:

### Example: Two AIs Refactor a Module

**Agent A (Refactorer) proposes a change:**
```ailang
-- Original
fn loadConfig() !: IO,FS -> Result[Config, Error] =
    do! readFile("config.json")
    |> parseJSON
    |> validateConfig

-- Refactored (adds caching)
fn loadConfig() !: IO,FS -> Result[Config, Error] =
    do! readFile("config.json")
    |> parseJSON
    |> validateConfig
    |> cacheConfig !: FS  -- Added caching with FS write
```

**Agent B (Verifier) checks the refactoring:**
```ailang
fn verifyRefactor(original, refactored) =
    let originalEffects = reflectEffect(original);
    let refactoredEffects = reflectEffect(refactored);

    if originalEffects == refactoredEffects then
        checkBehavioralEquivalence(normalize(original), normalize(refactored))
    else
        Err("Effects changed: " ++ show(refactoredEffects))
```

**Result:** Agent B rejects the refactoring (FS write effect added). Agent A must either:
1. Update signature to declare new effect
2. Remove caching to preserve original behavior

**Proof-synchronized:** Both agents agree on equivalence via deterministic hashing and effect tracking.

---

### Key Capabilities for Multi-Agent Systems

- **Shared Effect Algebra** — AIs coordinate through explicit effect declarations
- **Canonical Normalization** — `normalize()` ensures AIs compare semantically equivalent forms
- **Cryptographic Proofs** — Hash-based verification prevents adversarial code changes
- **Equivalence Checking** — Automated verification that refactorings preserve behavior

**The vision:** A language where AIs can reason about, improve, and trust each other's code.

---

## Measurable Success Criteria

| Metric | Target | Why It Matters |
|---------|--------|----------------|
| **Determinism Rate** | ≥95% | Identical outputs across runs enable caching and verification |
| **Type Correctness** | ≥98% | Type safety prevents entire classes of bugs |
| **Model Pass Rate** | ≥70% | AIs can solve real problems without human intervention |
| **Normalizer Lift** | +20pp | Automated repairs reduce AI failure rates by 20+ percentage points |
| **Eval Runtime** | <2 min | Fast feedback loops enable iterative AI development |

---

## Timeline

| Phase | Focus | Target |
|--------|--------|--------|
| **v0.3.14** | JSON decode + stable core | Week 1 |
| **v0.3.15** | Deterministic tooling (`normalize`, `suggest`, `apply`) | Week 3 |
| **v0.3.16** | Total recursion combinators + import ergonomics | Week 5 |
| **v0.4.0** | Public launch with deterministic tooling | Week 7 |
| **v0.4.1** | Runtime reflection (monomorphic types) | Week 9 |
| **v0.4.2** | Training data export + polymorphic reflection | Week 11 |

**Note:** Reflection (Tier 3) deferred to v0.4.1+ to derisk public launch. v0.4.0 ships with `normalize → suggest → apply` pipeline, which is sufficient to demonstrate AI-native design.

---

## What AILANG Doesn't Do

AILANG deliberately excludes features designed for human convenience:

- ❌ **LSP/IDE servers** — AIs use CLI/API, not text editors
- ❌ **CSP/concurrency** — replaced by static effect-typed task graphs
- ❌ **Implicit behaviors** — all effects, imports, and types are explicit
- ❌ **Multiple syntaxes** — one canonical way to express each concept

These aren't limitations — they're **design choices** that prioritize machine reasoning over human ergonomics.

---

## Who AILANG Is For

### Primary Users
- **AI Code Generators** (Claude, GPT, Gemini) — deterministic output, verifiable correctness
- **Autonomous Agents** — self-repairing code through reflection and normalization
- **Multi-Agent Systems** — coordinated reasoning through shared effect algebra

### Secondary Users
- **Researchers** — studying AI code generation, program synthesis, and verification
- **Tool Builders** — creating AI-powered development tools with provable guarantees
- **Curious Humans** — exploring what languages look like when designed for machines first

---

## Get Involved

AILANG is open source and evolving rapidly:

- **Try it:** See [Getting Started](/docs/guides/getting-started) for installation
- **Benchmark it:** Run `ailang eval-suite` to test AI code generation
- **Contribute:** See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow
- **Discuss:** Join discussions at [github.com/sunholo-data/ailang](https://github.com/sunholo-data/ailang)

---

> **AILANG — When the coder is the model.**
