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

### Problem 4: "Correct" Syntax Decays Across Versions

Pattern matching is the classic case. The `state_machine_elevator` benchmark asks for "an elevator state machine with ADTs and **exhaustive pattern matching**." That phrase has one obvious modern Python answer — and in a mixed-version ecosystem, picking it is a coin flip.

**Python — the idiomatic answer silently fails on older runtimes:**
```python
def transition(state, event):
    match (state, event):                       # Python 3.10+
        case (Idle(floor), Call(target)) if target == floor:
            return DoorOpen(floor, 3)
        case (Idle(floor), Call(target)):
            return Moving(floor, target)
        # ...
# SyntaxError on Python 3.9. 0/6 frontier models passed zero-shot.
# After repair, 5/6 fell back to isinstance() chains — ~2x more code,
# no compile-time exhaustiveness, reviewer must prove coverage by hand.
```

**AI struggles:** "Exhaustive pattern matching" maps to `match/case` for any model trained post-2022. The model can't see the grader's Python version; the compiler doesn't tell it what features are on the table. Failure mode is invisible until runtime.

**AILANG — one construct, compiler-enforced exhaustiveness:**
```ailang
func transition(state: State, event: Event) -> State =
  match (state, event) {
    (Idle(floor), Call(target)) if target == floor => DoorOpen(floor, 3),
    (Idle(floor), Call(target))                    => Moving(floor, target),
    (Moving(_, _), Tick)                           => state,
    (Moving(_, t), Arrive)                         => DoorOpen(t, 3),
    (DoorOpen(f, timer), Tick) if timer > 1        => DoorOpen(f, timer - 1),
    (DoorOpen(f, _), Tick)                         => Idle(f),
    _                                              => state
  }
```

**AI wins:** Pattern matching on ADTs is a single, stable construct. The compiler refuses to build code that misses a case, so "exhaustive" is machine-checked rather than prose in a spec. On the same task, 5/6 models passed zero-shot, 6/6 final, at ~570 output tokens — roughly half the Python variant.

*Data: v0.12.0 eval baseline, 6 frontier models. Raw logs: `eval_results/baselines/v0.12.0/standard/state_machine_elevator_*`. The eval harness now pins Python 3.12 and advertises the exact runtime to the model to avoid this class of unfairness going forward.*

---

## Current Capabilities — v0.12.x: Bytecode VM & Runtime

AILANG today provides:

- ✅ **Explicit Effect Tracking** — `! {IO, FS, Net, Clock, ...}` declares all side effects in types
- ✅ **Algebraic Data Types** — composable, type-safe data structures with exhaustive pattern matching
- ✅ **Hindley-Milner Inference** with row polymorphism for records and effects
- ✅ **Bytecode VM** — compact runtime suitable for embedding and introspection
- ✅ **Stdlib Coverage** — `std/json`, `std/http`, `std/fs`, `std/zip`, `std/tar`, `std/gzip`, `std/rand`, `std/crypto`, and more
- ✅ **Three-Tier OTEL Tracing** — `off | standard | deep` for structured execution traces
- ✅ **Agentic Eval Harness** — M-EVAL benchmark suite for measuring AI code generation quality

These features guarantee **predictable execution** and **analyzable behavior** —
the foundation for AI-driven code reasoning.

---

## Measured Results — M-EVAL Benchmarks (v0.12.0)

Results come from this repo's own eval harness, published at [static/benchmarks/latest.json](/benchmarks/latest.json) and regenerated every release.

**Latest run** (`v0.12.0`, 2026-04-17, 612 runs across 46 benchmarks, 6 frontier models):

| Metric | Result | Notes |
|--------|--------|-------|
| **Zero-shot compile+run success** | **75.3%** | One generation, no retry |
| **Final success (with repair)** | **81.2%** | After one automated repair pass |
| **Agent-mode success** | **68.9%** | Agentic CLI, avg 2.9 turns |
| **Repair lift** | **+5.9 pp** | Zero-shot → final delta |

**Per-model final-success rate** (46-benchmark suite, one run per benchmark):

| Model | Zero-shot | Final |
|---|---|---|
| claude-opus-4-7 | 82.4% | **84.3%** |
| gpt5-4 | 80.4% | **84.3%** |
| claude-sonnet-4-6 | 70.6% | 83.3% |
| gemini-3-1-pro | 71.6% | 80.4% |
| gpt5-2-codex | 75.5% | 78.4% |
| gemini-3-flash | 71.6% | 76.5% |

### What This Means

- **Frontier models now clear 80% on AILANG-native problems.** The eval suite covers type-safe record access, exhaustive pattern matching, effect tracking, JSON round-tripping, interpreter construction, and recursion.
- **Repair is meaningful but modest.** The +5.9 pp lift from a single automated repair pass shows AILANG's diagnostics carry enough structure for models to fix their own mistakes — the main gain is still zero-shot.
- **No Python-baseline claim is made here.** A like-for-like cross-language baseline would need matched problems, matched prompts, and matched graders; that work lives in [benchmarks/cross-language/](../benchmarks/cross-language/) and hasn't been measured yet.

*Run `ailang eval-suite` to reproduce; see [docs/docs/guides/evaluation/](docs/guides/evaluation) for methodology.*

---

## Next Horizons — v0.13 and v1.0

The v0.3–v0.4 roadmap items in earlier drafts of this doc (deterministic tooling, total recursion, polymorphic safety, effect sugar) shipped between v0.4 and v0.10. Current focus sits on two tracks:

### Near-term — v0.13 (Eval Expansion)

**Goal:** Broaden the evidence base for the claims above.

- **[M-EVAL-EXPAND](../design_docs/planned/v0_13_0/m-eval-expand-harnesses-languages.md)** — Add cross-language harnesses (Python, TypeScript), open-source model coverage (Ollama, local Llama/Qwen variants), and a proper like-for-like language baseline so the "no Python baseline" gap in the benchmark section above can be closed.
- **[M-WASM-TRACE](../design_docs/planned/v0_13_0/m-wasm-trace.md)** — Structured traces available in browser / WASM targets.

### Longer-term — v1.0 (AI-Native Platform)

**Goal:** Move from "good language for AI to write" to "platform AI agents run on."

- **[M-AGENT](../design_docs/planned/v1_0_0/m-agent-orchestration.md)** — `std/agent` with an explicit `! {Agent}` effect, capability-bounded AI invocation, cost/turn budgets.
- **[M-ENTROPY](../design_docs/planned/v1_0_0/m-entropy-budgets.md)** — Entropy budgets as a first-class concept: permitted ambiguity × designated resolver × collapse deadline.
- **[M-CSP / Session Types](../design_docs/planned/v1_0_0/m-csp-session-types.md)** — Static effect-typed task graphs replacing ad-hoc concurrency.
- **[M-EFFECT-REFINEMENT](../design_docs/planned/v1_0_0/m-effect-refinement.md)** — Tighter effect rows (e.g. separating reproducible PRNG from security-grade `CryptoRand` — see [rand-determinism-sitrep](../design_docs/planned/rand-determinism-sitrep.md)).
- **[M-TYPE-V2](../design_docs/planned/v1_0_0/m-type-v2-migration.md)** — Type system refresh.
- **[Global Collaboration Hub](../design_docs/planned/v1_0_0/global-collaboration-hub.md)** — Cross-machine agent collaboration with IAM-scoped message buses.

**The result:** Programs that don't just explain themselves — they coordinate, budget themselves, and make their uncertainty negotiable.

---

## Future Horizons — Multi-Agent Coordination

> **Aspirational.** The capabilities used in the sketch below (`reflectEffect`, `normalize`, `checkBehavioralEquivalence`) are not yet shipped. They depend on [M-AGENT](../design_docs/planned/v1_0_0/m-agent-orchestration.md) and the type/effect work in [M-TYPE-V2](../design_docs/planned/v1_0_0/m-type-v2-migration.md) / [M-EFFECT-REFINEMENT](../design_docs/planned/v1_0_0/m-effect-refinement.md). Treat this section as design intent, not a feature list.

### Example: Two AIs Refactor a Module

**Agent A (Refactorer) proposes a change:**
```ailang
-- Original
func loadConfig() -> Result[Config, Error] ! {IO, FS} =
    readFile("config.json")
    |> parseJSON
    |> validateConfig

-- Refactored (adds caching)
func loadConfig() -> Result[Config, Error] ! {IO, FS} =
    readFile("config.json")
    |> parseJSON
    |> validateConfig
    |> cacheConfig  -- Added caching; still in {IO, FS}, but new write path
```

**Agent B (Verifier) checks the refactoring** (sketch — APIs below are aspirational):
```ailang
func verifyRefactor(original, refactored) =
  let originalEffects = reflectEffect(original) in
  let refactoredEffects = reflectEffect(refactored) in
  if originalEffects == refactoredEffects
  then checkBehavioralEquivalence(normalize(original), normalize(refactored))
  else Err("Effects changed: " ++ show(refactoredEffects))
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

Targets used to gate public milestones. Current status pulled from [latest.json](/benchmarks/latest.json) (v0.12.0).

| Metric | Target | v0.12.0 |
|---------|--------|---------|
| **Model Pass Rate** (frontier, zero-shot) | ≥70% | **75.3%** ✅ |
| **Model Pass Rate** (frontier, with repair) | ≥80% | **81.2%** ✅ |
| **Best-model Final Success** | ≥80% | **84.3%** (opus-4-7, gpt5-4) ✅ |
| **Repair Lift** | +10 pp | +5.9 pp ⚠️ |
| **Benchmark Count** | ≥50 | 46 ⚠️ (expanding in v0.13) |
| **Cross-language Baseline** | published | not yet (planned in M-EVAL-EXPAND) |

The ⚠️ rows are active work — tracked by [M-EVAL-EXPAND](../design_docs/planned/v0_13_0/m-eval-expand-harnesses-languages.md).

---

## Roadmap

Roadmap lives in [design_docs/planned/](../design_docs/planned/), not in this file. The earlier versions of this doc hardcoded a v0.3→v0.4 week-by-week schedule that has long since shipped. For current planned work:

- **Near-term sprints**: [design_docs/planned/v0_13_0/](../design_docs/planned/v0_13_0/)
- **v1.0 track**: [design_docs/planned/v1_0_0/](../design_docs/planned/v1_0_0/)
- **Recent releases**: see [CHANGELOG.md](../CHANGELOG.md)

---

## What AILANG Doesn't Do

AILANG deliberately excludes features designed for human convenience:

- ❌ **LSP/IDE servers** — AIs use CLI/API, not text editors
- ❌ **Unstructured concurrency** — no threads / goroutines / async-await. Concurrency arrives via static effect-typed task graphs and session types ([M-CSP](../design_docs/planned/v1_0_0/m-csp-session-types.md)).
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
