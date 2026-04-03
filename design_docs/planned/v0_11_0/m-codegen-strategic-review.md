# M-CODEGEN-STRATEGIC-REVIEW: Honest Assessment of Compile-to-Go and the Path Forward

**Status**: Planned (Strategic Review)
**Target**: v0.11.0+
**Priority**: P0 (Strategic — determines compilation roadmap)
**Created**: 2026-04-03
**Reporter**: Architecture review (Mark + multi-AI review)
**Supersedes**: Contextualizes M-CODEGEN-IR-STRATEGY, M-CODEGEN-SUSTAINABILITY

---

## TL;DR

**Problem**: The Go codegen has consumed 30+ design docs, 17,000+ LOC, and months of work, yet cannot reliably compile most AILANG programs. The issue is not that codegen is incomplete — it is that the mapping from AILANG → Go is **non-convergent under feature growth**. Each feature introduces cross-product interactions, not linear additions. Continuing is provably the wrong strategy.

**Root cause**: We are attempting to translate a polymorphic, effect-typed functional language into Go source code — a language whose type system **cannot express AILANG's invariants**. If the target language cannot represent the source language's semantics, compilation cannot preserve meaning. This forces `interface{}` everywhere, negating every supposed benefit of native Go.

**Key insight**: We have the right diagnosis (M-CODEGEN-IR-STRATEGY, Dec 2025). We haven't executed the treatment. The evaluator is not a fallback — it is the **reference implementation** of the language. WASM works because it runs the evaluator, not because it generates code.

**Recommendation**: Stop treating Go codegen as a primary execution path. AILANG already has a correct runtime (the evaluator). Build a Statement IR to preserve semantics, then **project** into targets selectively. Codegen is a projection, not a translation — projection accepts controlled loss of structure; translation implies equivalence we cannot deliver.

**Enabler**: There is exactly one consumer of Go codegen (Stapledon's Voyage). No backward compatibility required. We can **delete** 17,000+ LOC and build fresh from Statement IR. The acceptance gate is: Stapledon's Voyage compiles and runs.

---

## 1. Where We Are: An Honest Assessment

### 1.1 Timeline of Go Codegen Effort

| Version | Period | Design Docs | What Happened |
|---------|--------|-------------|---------------|
| v0.5.0 | Oct 2025 | M-GAME-A foundation | Initial codegen for Stapledon's Voyage. Early success on simple programs |
| v0.5.7-v0.5.10 | Nov-Dec 2025 | 20 docs | Case-by-case fixes: typed slices, bool slices, blank identifiers, pointer returns, flat if-else, option helpers, list flatten, tuple patterns, cross-module, ADT type asserts, record typename preservation, nested records, stdlib math |
| v0.5.9 | Dec 2025 | M-CODEGEN-V2 | **Bright spot**: Block IR reduced IIFEs by 58% with just 121 LOC. Proved small IRs work |
| v0.6.0-v0.6.2 | Dec-Jan | 6 docs | More fixes: unified slices, ADT double-paren, value types, bool assertions, dictionaries |
| v0.7.0 | Feb 2026 | 1 doc | List type definitions |
| v0.9.2 | Mar 2026 | 1 doc | Registry-only builtins |
| Mar 2026 | — | 8 emergency commits | DocParse (22 modules) broke catastrophically. Full day of whack-a-mole debugging |

**Total investment**: 30+ codegen-specific design docs. 49 files / 17,067 LOC. Months of engineering time.

### 1.2 What Works

- **Simple programs compile**: Basic types, pure functions, pattern matching, ADTs
- **Stapledon's Voyage**: 22 modules compiled to Go (March 2025), but generated code is stale and not trusted to compile after subsequent changes
- **sim_stub example**: Small, controlled example works end-to-end
- **Unit tests pass**: 200+ codegen tests verify individual expression generation

### 1.3 What Doesn't Work

- **Multi-module projects break unpredictably**: DocParse (22 modules) required 8 emergency commits
- **Each new AILANG feature requires manual codegen work**: New builtins, type features, and patterns all need hand-coded Go emission
- **Generated code is not idiomatic Go**: `interface{}` everywhere, runtime type assertions, `_impl` + typed wrapper pairs
- **No CI integration test**: Unit tests pass but real projects break
- **Regression detection is zero**: Changes to the evaluator or type system can silently break codegen
- **Stdlib mapping is manual and linear**: 80+ entries, growing to 150+ projected

### 1.4 The Numbers That Matter

| Metric | Value | Assessment |
|--------|-------|------------|
| Design docs for codegen | 30+ | Unsustainable — each adds special cases |
| LOC in codegen module | 17,067 | Larger than many production compilers |
| Switch cases in match codegen | 41 | Combinatorial complexity |
| Switch cases in ops codegen | 47 | Same pattern |
| Stdlib builtin mappings | 80+ | Superlinear growth (interactions, not just additions) |
| Multi-module CI tests | 0 | Flying blind |
| Strategic architectural changes implemented | 0 of 2 planned | M-CODEGEN-IR-STRATEGY and M-CODEGEN-SUSTAINABILITY still in planned/ |

### 1.5 Why This Cannot Converge

The failure pattern is not "incomplete coverage" — it is **non-convergent under feature growth**.

Each new AILANG feature (a new type form, a new pattern, a new effect) creates interactions with every existing codegen path. The bug surface is not `O(features)` but `O(features × features)`:

- A new ADT variant interacts with pattern matching, type analysis, record access, list operations, and cross-module imports
- A new stdlib function interacts with type mapping, runtime helpers, dictionary dispatch, and every call-site pattern
- A new type system feature interacts with monomorphization, type projection, and every expression generator

**30+ design docs over 6 months with continued breakage is not bad luck. It is the expected behavior of a non-convergent system.** The rate of new bugs introduced by feature growth will always exceed the rate of fixes. This is not a quality problem — it is an architectural impossibility.

---

## 2. Why WASM Works But Go Codegen Doesn't

### 2.1 The Critical Insight

What we call "WASM compilation" is **not code generation**. It is the Go evaluator compiled to WebAssembly:

```
"WASM compilation":
    AILANG source → Parse → Elaborate → Core IR → EVALUATE (same as REPL)
    The Go evaluator binary is compiled with GOOS=js GOARCH=wasm
    = Interpreter running in browser

Go codegen:
    AILANG source → Parse → Elaborate → Core IR → GENERATE GO SOURCE → go build
    = Actual compilation to a different language
```

WASM "works well" because **it doesn't try to bridge a type system gap**. The evaluator already handles every AILANG feature correctly. Compiling the evaluator to WASM is a solved problem (Go's toolchain does it).

Go codegen is fundamentally harder because it must:
1. Map AILANG's Hindley-Milner types → Go's structural types
2. Map algebraic data types → discriminator structs
3. Map polymorphism → monomorphization + `interface{}` fallbacks
4. Map row polymorphism → concrete struct types (losing polymorphism)
5. Map effects → handler interfaces
6. Map pattern matching → switch/if-else chains
7. Map every stdlib function → a Go runtime helper

**Each mapping is a source of bugs**, and the mappings interact combinatorially.

### 2.2 The Type System Mismatch

This is the fundamental problem. AILANG and Go have incompatible type systems:

| AILANG Feature | Go Equivalent | Impedance |
|---------------|---------------|-----------|
| Parametric polymorphism (`a -> a`) | `interface{}` or Go generics (limited) | High — loses type safety |
| Algebraic data types (`type Option = Some(a) \| None`) | Discriminator struct + kind enum | Medium — works but verbose |
| Row polymorphism (`{name: string, ...r}`) | No equivalent | **Critical** — must erase to concrete type |
| Effect types (`-> T ! {IO, Net}`) | No equivalent | **Critical** — must use handler interfaces |
| Type classes (`Eq a => ...`) | No equivalent | High — must use dictionary passing |
| Pattern matching (nested, exhaustive) | switch + type assertions | Medium — verbose but expressible |
| Curried functions (`a -> b -> c`) | `func(a) func(b) c` | Medium — creates closure chains |

**The conclusion is not that Go is awkward as a target. The conclusion is stronger:**

> **Go is not a valid compilation target for AILANG semantics.** If the target language cannot express the source language's invariants, compilation cannot preserve meaning. What remains is not compilation — it is lossy transcription with runtime patches (`interface{}` assertions) to recover what was lost.

The generated Go code is full of `interface{}`, type assertions, and wrapper functions. It is not idiomatic Go. It is not fast Go. It doesn't leverage Go's type system. The supposed benefits of "compile to Go" — type safety, performance, ecosystem access — are not just "hard to achieve." They are **structurally unavailable** given the type system mismatch.

### 2.3 The Maintenance Multiplier

Every AILANG feature must be maintained in **two independent systems**:

```
AILANG feature
    ├── Evaluator implementation (internal/eval/)     ← always works
    └── Go codegen implementation (internal/gen/golang/) ← guaranteed to diverge without duplication
```

The evaluator is the source of truth. The codegen is a **shadow implementation** that must be kept in sync manually. With 57 builtins and 283 stdlib exports, this is a **superlinear** maintenance burden (interaction-based, not additive) that grows with every feature.

---

## 3. Why We Got Here: The Case-by-Case Trap

### 3.1 The Pattern

The development pattern has been:
1. Try to compile a real program (Stapledon's Voyage, DocParse)
2. Hit a codegen bug
3. Write a design doc for the specific bug
4. Fix the specific case
5. Discover the fix reveals another bug
6. Repeat from step 3

This is **reactive engineering** — fixing symptoms, not causes. Each fix is locally correct but globally adds complexity. After 30+ iterations, the codegen is a patchwork of special cases.

### 3.2 What Should Have Happened

The **M-CODEGEN-IR-STRATEGY** doc (written Dec 2025) correctly diagnosed the problem and proposed the right solution: multi-layer IR with Statement IR as the single emitter-facing representation. The **M-CODEGEN-V2 Block IR** proved the approach works (58% IIFE reduction, 121 LOC).

But the IR strategy was never implemented. Instead, we continued adding special cases. The sustainability doc (M-CODEGEN-SUSTAINABILITY, Mar 2026) diagnosed the same symptoms again from a different angle.

**We have the right diagnosis. We haven't executed the treatment.**

### 3.3 The Sunk Cost Problem

17,000+ LOC and 30+ design docs create psychological pressure to "keep going." But the question is not "how much have we invested?" — it's "what's the fastest path to reliable compilation?"

---

## 4. Strategic Options

### Evaluation Axis: Semantic Authority

Every option must be evaluated against a hidden axis the original doc missed:

> **Does this preserve AILANG as the semantic authority?**

The evaluator is the reference implementation. Any approach that creates a shadow implementation (a second system that must independently re-express AILANG semantics) will diverge. This is not a risk — it is a certainty, already demonstrated by 6 months of evidence.

| Option | Semantic Authority | Assessment |
|--------|-------------------|------------|
| A (current) | Split — shadow implementation | Guaranteed divergence |
| B (Statement IR) | Centralized — emitters are projections | Strong |
| C (embedded evaluator) | Single source — evaluator IS the runtime | Strongest |
| D (bytecode) | Single source — bytecode from one IR | Strong |
| E (LLVM) | Centralized (if IR-first) | Strong |
| F (hybrid) | Partially split | Requires discipline |
| G (evaluator-first, no source codegen) | Single source | Strongest |

### Option A: Continue Current Approach (Case-by-Case Fixes)

Keep fixing bugs as they appear. Eventually stabilize through exhaustive testing.

| Pros | Cons |
|------|------|
| No architectural change needed | 30+ docs hasn't stabilized it yet |
| Preserves existing code | Each fix reveals more fixes |
| Incremental progress | Unbounded completion time |
| | Not portable to Rust or other targets |
| | Maintenance burden grows linearly |

**Verdict**: This is what we've been doing. It hasn't worked.

### Option B: Implement the IR Strategy (M-CODEGEN-IR-STRATEGY)

Build the Statement IR as planned. Refactor Go codegen to be a thin emitter over Statement IR.

| Pros | Cons |
|------|------|
| Proven by Block IR (121 LOC, 58% improvement) | 2-3 weeks to implement |
| Enables future backends (Rust, C) | Still has `interface{}` problem in Go |
| Reduces switch cases by 70-80% | Generated Go is still not idiomatic |
| Centralizes pattern handling | Doesn't solve stdlib mapping problem |

**Verdict**: Right architecture, but doesn't address the fundamental type system mismatch. Should be done regardless of compilation target.

### Option C: Embedded Evaluator Library ("WASM Strategy for Native")

Instead of generating Go source code, package the AILANG evaluator as a Go library. Generated "Go code" is thin wrapping around evaluator calls.

```go
// Instead of generating 17,000 LOC of Go source...
import ailang "github.com/sunholo/ailang/runtime"

func main() {
    vm := ailang.New()
    vm.LoadModule("sim/world.ail")
    result := vm.Call("world", "step", frameInput)
    // result is already a Go value via reflection
}
```

| Pros | Cons |
|------|------|
| Works for ANY AILANG feature (evaluator handles it) | No ahead-of-time optimization |
| Zero maintenance for new features | Runtime overhead (interpretation) |
| This is exactly why WASM works | Not "compiled" in the traditional sense |
| Thin API, easy to maintain | May not satisfy game engine perf needs |
| Portable: same approach works for Rust FFI | Requires reflection-based value conversion |

**Verdict**: Honest about what "WASM" actually does. Eliminates the 17,000 LOC codegen entirely. But doesn't provide native performance.

### Option D: Bytecode Compiler + VM

Design an AILANG bytecode format. Write a portable VM. Compile AILANG to bytecode instead of source code.

```
AILANG source → Core IR → Statement IR → AILANG Bytecode → VM execution
                                              ↓
                                    Portable to any host language
```

| Pros | Cons |
|------|------|
| No type system mismatch (bytecode is our own) | Significant design + implementation effort |
| Fast compilation (no `go build` step) | Custom debugging tools needed |
| Portable across all host languages | Performance depends on VM quality |
| Proven approach (Lua, Erlang BEAM, JVM) | 4-8 weeks for initial implementation |
| Natural path to JIT compilation later | |

**Verdict**: The right long-term answer for a language that takes compilation seriously. Requires significant investment. The Statement IR from Option B is a stepping stone.

### Option E: Target LLVM or Cranelift

Use an established compiler backend instead of generating source code.

| Pros | Cons |
|------|------|
| Mature optimization pipelines | Massive dependency (LLVM) |
| Multiple hardware targets | Complex FFI for Go integration |
| Well-suited for functional languages | Cranelift is Rust-only |
| Used by GHC, Rust, Swift | Compilation times can be slow |
| No type system mismatch | Steep learning curve |

**Verdict**: The right answer if AILANG needs C/Rust-level performance. Overkill for current use cases. Consider for v1.0+.

### Option F: Hybrid — Statement IR + Selective Compilation

Build the Statement IR (Option B). Use it for **selective compilation** of hot paths, while the evaluator handles everything else.

```
AILANG program
    ├── Hot path functions (pure, simple types) → Statement IR → Go/Rust emitter → native
    └── Everything else → Evaluator (fast enough)
```

| Pros | Cons |
|------|------|
| Pragmatic: compile what you can, interpret the rest | Two execution modes to maintain |
| Statement IR is useful regardless | Boundary between compiled/interpreted is complex |
| Evaluator handles edge cases | Users must understand which functions compile |
| Incremental: start small, expand | |

**Verdict**: Pragmatic middle ground. Acknowledges that not everything needs to compile.

### Option G: Evaluator-First Architecture (No Source Codegen)

Accept that the evaluator is the canonical runtime. Build Statement IR for analysis and optimization, but emit only to:
- Bytecode (for performance)
- Embedded evaluator (for deployment/integration)
- LLVM IR (future, for native performance)

**No Go/Rust/C source emission as a primary path.** Source emission becomes an optional debug/inspection tool, not a production compilation strategy.

```
AILANG source → Core IR → Statement IR
                                ↓
                    ┌───────────┼───────────────┐
                    ↓           ↓               ↓
              Bytecode VM   Evaluator embed   [LLVM, future]
              (performance) (deployment)      (native perf)
```

| Pros | Cons |
|------|------|
| Strongest semantic authority (evaluator is canonical) | "No compiled output" may feel like a step backward |
| Eliminates 17,000 LOC codegen and its maintenance burden | Bytecode VM requires investment |
| Statement IR still enables analysis, optimization, linting | Must prove evaluator performance is sufficient |
| Aligns with AILANG axioms (A7: machines first) | |
| Most honest about what we actually have | |

**Verdict**: Most aligned with AILANG's philosophy and the evidence. The evaluator already works. Statement IR is the right abstraction. Source codegen adds complexity without unique value.

---

## 5. The Core Strategic Insight

The document up to this point circles around a claim that should be stated directly:

> **AILANG already has a correct execution model. The problem is not execution — it is representation.**

More precisely:
- **Evaluator** = correct semantics (proven by WASM, REPL, all tests)
- **Go codegen** = attempt to re-express those semantics in a foreign type system
- **Statement IR** = bridge that preserves semantics before projection into constrained targets

The evaluator is not a fallback. It is the reference implementation of the language. Removing the implicit bias toward "real compilation = native source code" is the key conceptual move that unlocks the right strategy.

**Codegen is a projection, not a translation:**
- *Translation* implies equivalence — the output means the same thing as the input
- *Projection* accepts controlled loss of structure under explicit rules
- The current Go codegen pretends to be translation but is actually lossy projection without explicit rules for what is lost

The sustainable mental model is: **AILANG semantics → IR → projection into host constraints**, where each projection has a documented "projection contract" specifying what is preserved and what degrades.

---

## 6. Recommendation: Three-Phase Strategy

### Phase 0: Stop the Bleeding (1 week)

**Actions:**
1. **Freeze case-by-case codegen patches.** No more design docs for individual codegen bugs unless they fix the architecture.
2. **Reject new language features that require codegen support until Statement IR exists.** Otherwise pressure will reintroduce the same failure mode. New features target the evaluator only.
3. **Add the multi-module CI test** (from M-CODEGEN-SUSTAINABILITY). This takes 1 day and prevents future regression.
4. **Document the current compilation boundary**: explicitly list what AILANG features compile to Go and what doesn't. Users and AI agents need this.
5. **Acknowledge the codegen as experimental** in documentation and CLI output.

**Deliverables:**
- CI workflow: `test-codegen-multimodule.yml`
- Doc: `docs/guides/go-compilation-status.md` (supported/unsupported features)
- CLI: `ailang compile --emit-go` prints "experimental" warning
- Policy: no new codegen feature work without IR-first architecture

### Phase 1: Build the Statement IR (2-3 weeks)

This is the **right work regardless of compilation target**. Implement M-CODEGEN-IR-STRATEGY:

1. **Statement IR types** (`internal/gen/stmt/`) — the ONLY representation emitters see
2. **Match lowering** (`internal/gen/lower/`) — Core patterns → decision trees → Statement IR
3. **Type projection** (`internal/gen/typeres/`) — pure mapping, no inference, no fallbacks
4. **Block lowering** — already done (121 LOC), integrate into Statement IR pipeline

```
Core AST + CoreTypeInfo
    → Match Lowering (patterns → if/switch)
    → Type Projection (AILANG types → ResolvedType)
    → Block Lowering (let chains → flat statements)
    → Statement IR
        → Go Emitter (~500 LOC, replaces 15,000+ LOC)
        → [Future] Rust Emitter
        → [Future] C Emitter
        → [Future] Bytecode Emitter
```

**Key principle**: Statement IR is **target-agnostic**. It knows about if/switch/variable declarations/function calls — concepts that exist in EVERY target language. It does NOT know about Go's `interface{}`, Rust's `Box<dyn Any>`, or C's `void*`.

**The `interface{}` problem becomes the emitter's problem**, contained in ~500 LOC instead of spread across 17,000 LOC.

**Hard constraint**: **No emitter may inspect Core AST directly.** All emitters must consume only Statement IR. This prevents regression into the current architecture. If an emitter needs information not present in Statement IR, the answer is to extend Statement IR — not to reach around it.

**Architecture (current vs proposed):**

```
CURRENT (non-convergent):
    Core AST ──────────────────────────────► Go Source
                 17,000 LOC, 41+47 switch cases,
                 30+ special-case design docs

PROPOSED (layered projection):
    Core AST + CoreTypeInfo
        → Match Lowering (patterns → decision trees → if/switch)
        → Type Projection (AILANG types → ResolvedType, pure function)
        → Block Lowering (let chains → flat statements)
        → Statement IR (the ONLY thing emitters see)
            → Go Emitter (~500 LOC)
            → Bytecode Emitter (~500 LOC)
            → [Future] Rust/LLVM/C Emitter
```

**Deliverables:**
- `internal/gen/stmt/` — Statement IR types + invariants
- `internal/gen/lower/` — Match/block/type lowering passes
- `internal/gen/golang/emitter.go` — Thin Go emitter over Statement IR
- Golden tests: 20+ AILANG → Go golden files
- Metamorphic tests: old codegen vs new codegen produce equivalent Go

### Phase 2: Choose the Right Compilation Target (Strategic Decision)

With Statement IR in place, we can make an informed choice. But the framing matters — AILANG's philosophy (A7: machines first, A9: cost is meaning) implies requirements should be stated in terms of **what the language needs**, not what humans conventionally expect:

**Requirements axis (AILANG-native framing):**

| Requirement | What It Means | Best Target |
|-------------|---------------|-------------|
| **Semantic fidelity** | Output preserves AILANG's type/effect/determinism guarantees | Evaluator (canonical) or bytecode (faithful) |
| **Packaging simplicity** | Single deployable artifact, no runtime dependency | Embedded evaluator or bytecode VM |
| **Bounded, predictable cost** | Execution time/memory proportional to declared budgets | Bytecode VM (instrumentable) or evaluator with profiling |
| **Host-language interop** | AILANG functions callable from Go/Rust/Python | Embedded evaluator (FFI) or thin emitter (API surface) |
| **Analysis & tooling** | Linting, optimization, dead code detection | Statement IR (machine-readable, analyzable) |

**Mapping requirements to targets:**

| Concrete Use Case | Primary Requirement | Best Target | Why |
|-------------------|-------------------|-------------|-----|
| Browser (WASM) | Packaging | Evaluator-as-WASM (current) | Already works. Don't change it |
| Game engine (60 FPS) | Bounded cost | Bytecode VM or evaluator + profiling | Need to measure before choosing |
| API server | Packaging + interop | Embedded evaluator library | Simplest, most maintainable |
| CLI tools | Semantic fidelity | Evaluator (current `ailang run`) | Already works |
| Library for Go consumers | Host interop | Statement IR → thin Go emitter | Emitter generates API surface, not full reimplementation |
| Library for Rust consumers | Host interop | Statement IR → thin Rust emitter | Same architecture, different emitter |
| Maximum performance | Bounded cost | Bytecode VM → JIT (long-term) | The Lua/LuaJIT path |
| AI reasoning/training | Semantic fidelity + analysis | Statement IR + evaluator | IR is inspectable; evaluator is correct |

**Critical pre-requisite for Phase 2:** Benchmark the evaluator's actual performance envelope. If the evaluator handles 60 FPS game loops, bytecode VM is unnecessary. **Do not build what you do not need.**

---

## 7. What This Means for Existing Code

### No Backward Compatibility Required

There is exactly **one consumer** of Go codegen: Stapledon's Voyage. No public API. No published Go module. No external users depending on generated code structure, function signatures, or runtime helpers.

This means we can **delete** rather than refactor. The constraint is not "preserve the existing codegen" — it is "Stapledon's Voyage compiles and runs through the new pipeline." Everything else is expendable.

This dramatically simplifies the transition:

### Delete (17,000+ LOC)
- `internal/gen/golang/codegen*.go` — all 40+ files. The entire current codegen. Gone.
- `internal/gen/golang/adt.go`, `contracts.go`, `debug.go`, `ai.go`, `effects.go` — superseded by Statement IR + thin emitter
- `internal/gen/golang/codegen_runtime*.go` — 5 files of reimplemented stdlib. Replaced by runtime library or evaluator embedding

### Extract Before Deleting
- `internal/gen/golang/*_test.go` — test *cases* are valuable (input/output pairs). Extract as golden test corpus before deleting the test infrastructure
- `internal/gen/block/` — Block IR (proven, 121 LOC). Conceptually sound, integrate pattern into Statement IR lowering
- Registry pattern from `codegen_registry.go` — the idea of querying builtin metadata is correct, even if the current implementation is deleted

### Keep Unchanged
- Core pipeline (parse → elaborate → typecheck → monomorphize → lower) — this is correct and shared
- `internal/eval/` — the reference implementation. Untouched.
- `cmd/ailang/compile.go` — CLI entry point, will be rewired to new pipeline

### Build Fresh
- `internal/gen/stmt/` — Statement IR types (new)
- `internal/gen/lower/` — Lowering passes: match, block, type projection (new)
- `internal/gen/golang/emitter.go` — ~500 LOC thin Go emitter over Statement IR (new)
- `tests/golden/codegen/` — golden test corpus extracted from old tests (new)

### Preserve as Archive
- All 30+ codegen design docs in `design_docs/` — valuable history, no code impact
- Old `internal/gen/golang/` can live in git history — no need to keep dead code in tree

---

## 8. Addressing the "No Benefits" Concern

The user observation that Go codegen provides "no benefits" is essentially correct for the current implementation:

| Expected Benefit | Reality |
|-----------------|---------|
| **Type safety** | Lost — `interface{}` everywhere |
| **Performance** | Minimal — type assertions at runtime |
| **Go ecosystem access** | Limited — generated code is hard to integrate |
| **Static analysis** | Defeated — Go tools can't analyze `interface{}` code |
| **Deployment simplicity** | True — single binary is a real benefit |
| **Compile-time error detection** | Partially true — `go build` catches some issues |

**The single binary deployment is the only surviving benefit.** But embedding the evaluator as a Go library (Option C) achieves the same thing without the 17,000 LOC codegen.

> **If the only surviving benefit can be achieved via embedding the evaluator, then the Go codegen has zero unique value.** This is the definitive argument against continuing Option A.

**What WOULD provide real benefits (via different approaches):**
- **Bytecode compilation** — real performance gains, portable, no type mismatch
- **Statement IR** — thin, maintainable backends for any target
- **Embedded evaluator** — zero-maintenance "compilation" for deployment

---

## 9. Comparison: How Other Functional Languages Solve This

| Language | Compilation Strategy | What We Can Learn |
|----------|---------------------|-------------------|
| **Haskell (GHC)** | Surface → Core → STG → Cmm → LLVM/NCG | Multiple IR layers, each simpler than the last. Codegen sees only Cmm (C-minus-minus), a very simple IR |
| **OCaml** | Surface → Lambda → Flambda → Cmm → assembly | Same pattern: IR layers. Backend sees flat C-like IR |
| **Elm** | Surface → optimized AST → JavaScript | Direct JS emission works because JS is dynamically typed (no type system mismatch) |
| **PureScript** | Surface → CoreFn → JavaScript/Erlang | CoreFn is the backend-facing IR. Backends are thin (~1000 LOC each) |
| **Erlang** | Surface → Core Erlang → BEAM bytecode | Bytecode VM approach. BEAM is the universal target |
| **Lua** | Surface → bytecode → Lua VM | Bytecode + VM. LuaJIT adds JIT for performance |

**The common pattern**: Every successful functional language compiler has **multiple IR layers** between the surface language and the backend. **No successful compiler goes directly from a rich AST to target source code.** Our M-CODEGEN-IR-STRATEGY diagnosed this correctly but was never implemented.

**The additional insight**: Languages that target other source languages (Elm → JS, PureScript → JS) succeed because **JavaScript is dynamically typed** — there's no type system mismatch. Targeting Go (statically typed, no ADTs, limited generics) creates a fundamental mismatch that JavaScript doesn't have.

---

## 10. The Rust Question

The user asks about extending to Rust. Here's the honest analysis:

**Rust has the same type system mismatch as Go**, but different:

| AILANG Feature | Go Problem | Rust Problem |
|---------------|------------|--------------|
| Polymorphism | `interface{}` | `Box<dyn Any>` |
| ADTs | Discriminator structs | `enum` (natural fit!) |
| Pattern matching | switch chains | `match` (natural fit!) |
| Effects | Handler interfaces | Traits (decent fit) |
| Closures | `func` values | `Fn`/`FnMut`/`FnOnce` (complex) |
| Memory management | GC (fine) | Ownership (very hard to generate) |

Rust is **better for ADTs and pattern matching** but **much worse for memory management**. Generating correct Rust ownership/lifetime annotations from a GC'd language is an unsolved problem in compiler engineering.

**With Statement IR**: A Rust emitter would be ~500 LOC, handling only the emission. The hard problems (match lowering, type resolution) are solved once in the IR passes. But the ownership problem would require a separate design effort.

**Practical recommendation**: Don't target Rust source code. If you want Rust-level performance, target LLVM IR or Cranelift (which Rust itself uses). The Statement IR can lower to LLVM IR through a thin emitter.

---

## 11. Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Statement IR transformations are deterministic. Bytecode execution is deterministic. Current codegen has non-deterministic `interface{}` behavior at runtime |
| A2: Replayable Execution | +1 | Statement IR preserves trace information. Bytecode VM can be instrumented for replay |
| A3: Effects Are Legible | +1 | Statement IR can represent effects explicitly. Current codegen erases effect information |
| A5: Verification Local & Bounded | +2 | Statement IR enables local verification of lowering passes. Golden tests verify transformations |
| A7: Machines First | +2 | Statement IR is machine-readable, analyzable. Multiple backends from one IR serves machine consumers |
| A8: Syntax Is a Liability | +1 | Reduces special cases (from 41+47 switch cases to ~20 total). Each IR layer has minimal syntax |
| A9: Cost Is Part of Meaning | +1 | Bytecode VM can track cost natively. Statement IR can annotate cost |
| A10: Composability | +2 | IR layers compose cleanly. Emitters compose with any Statement IR producer. Current codegen has composability failures (30+ special case docs) |
| A11: Failure Must Be Representable | +1 | Type projection errors loudly on unresolved types (no silent fallbacks). Current codegen has silent `interface{}` defaults |
| A12: Language Is a System Boundary | +1 | Statement IR is an explicit boundary between language semantics and target emission |
| **Net Score** | **+14** | Strong alignment with AILANG axioms |

---

## 12. Success Criteria

### Phase 0 (Stop the Bleeding)
- [ ] CI test compiles multi-module project and runs `go build`
- [ ] Go compilation documented as experimental with feature matrix
- [ ] No new case-by-case codegen design docs without architectural justification

### Phase 1 (Statement IR)
- [ ] Statement IR types defined and documented
- [ ] Match lowering passes all golden tests (extracted from old test corpus)
- [ ] Type projection is pure function, errors on unresolved types
- [ ] Go emitter over Statement IR is <600 LOC
- [ ] Old `internal/gen/golang/` deleted (17,000+ LOC removed from tree)
- [ ] Golden tests: 20+ AILANG → Go files
- [ ] **Stapledon's Voyage compiles and runs through new pipeline** (the single acceptance gate)

### Phase 2 (Compilation Target Decision)
- [ ] Performance benchmark: evaluator vs Go codegen vs bytecode (if built)
- [ ] Decision documented: which use cases get which compilation target
- [ ] At least one non-Go emitter stub (Rust or C) validates portability

---

## 13. Related Documents

### Existing (contextualizes, does not supersede)
- [M-CODEGEN-IR-STRATEGY](m-codegen-ir-strategy.md) — Correct diagnosis, unimplemented. Phase 1 implements this
- [M-CODEGEN-SUSTAINABILITY](m-codegen-sustainability.md) — Registry + CI. Phase 0 implements the CI portion
- [M-CODEGEN-V2](../../implemented/v0_5_9/m-codegen-v2-flat-output.md) — Block IR success story. Validates the IR approach
- [M-GAME-A](../../implemented/v0_5_0/M-GAME-A-go-codegen-foundation.md) — Original motivation (game engines)

### History (30+ codegen design docs)
- v0.5.7: typed-slices
- v0.5.8: list-concat, bool-slice, blank-identifier, zero-arg, type-assertions, typed-params
- v0.5.9: pointer-return, option-helpers, flat-if-else, getopt-slices, v2-flat-output, stdlib-math
- v0.5.10: list-flatten, tuple-pattern, cross-module, adt-type-assert, record-typename, nested-record
- v0.6.0: unified-slice-converters
- v0.6.1: adt-double-paren
- v0.6.2: value-types, bool-assertions, dictionaries
- v0.7.0: list-type-definition
- v0.9.2: registry-only, multimodule-bugs

Each of these is a symptom of the same root cause: direct Core → Go source emission without intermediate abstraction layers.

---

## 14. Follow-Up Directions

These are the natural next steps if this document is accepted:

### 14.1 Statement IR Formal Spec (Keystone)

Define Statement IR as types + invariants only, no implementation. Answer:
- What invariants does Statement IR guarantee? (e.g., "all variables declared before use", "no nested expressions deeper than 1")
- Is it effect-aware? (Should effects be represented in the IR, or resolved before it?)
- What information from CoreTypeInfo survives into Statement IR?

This is the **keystone** — everything else depends on getting this right.

### 14.2 Projection Contracts

For each emitter target, define a "projection contract":
- What must an emitter **preserve**? (determinism, type safety at boundaries, effect tracking)
- What is **allowed to degrade**? (row polymorphism → concrete types, currying → multi-arg functions)
- What **must error** rather than silently degrade? (unresolved type variables, unsupported effects)

### 14.3 Evaluator Performance Envelope

Before building bytecode VM or any new compilation target, **measure what we have**:
- Can the evaluator sustain 60 FPS for Stapledon's Voyage game loops?
- Where does the evaluator actually bottleneck? (dispatch overhead? allocation? pattern matching?)
- What is the evaluator's cost per function call vs. compiled Go?

If the evaluator is fast enough for all current use cases, bytecode VM is unnecessary. **Do not build what you do not need.**

### 14.4 AI-Facing Implications

Statement IR is a machine-readable representation of AILANG program semantics. This has product implications:
- Should Statement IR be part of the public surface? (AI agents analyzing AILANG code)
- Can Statement IR serve as the basis for AILANG → AILANG optimization passes?
- Does exposing IR enable new forms of AI-assisted verification?

---

## Changelog

| Date | Change |
|------|--------|
| 2026-04-03 | Initial strategic review document |
| 2026-04-03 | Incorporated multi-AI review feedback: strengthened non-convergence argument, added semantic authority axis, added Option G (evaluator-first), elevated projection-vs-translation framing, hardened phase plan constraints, added follow-up directions |
