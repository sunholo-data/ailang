# AILANG Execution Profiles — A Unified Architecture for Games, Agents, Services, and Tools

**Status**: Archived (2026-03-18)
**Original Target**: v0.6.0
**Priority**: P0 - Strategic
**Estimated**: 2 weeks
**Dependencies**: Go codegen (v0.5.x), Effect contexts (v0.5.x)
**Archived Reason**: Implementation plan is obsolete — assumed an IR pipeline that doesn't exist yet. The profile taxonomy (Sim/Service/CLI) has been extracted into [M-CODEGEN-IR-LAYERS](../planned/v0_10_0/m-codegen-ir-layers.md) Phase 5 as a downstream deliverable of the IR architecture.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Profiles define deterministic effect budgets per entry shape |
| A2: Replayability | +1 | Effect contexts enable trace replay and inspection |
| A3: Effect Legibility | +1 | Profiles make allowed effects explicit and machine-readable |
| A4: Explicit Authority | +1 | Effect budgets constrain what capabilities each profile allows |
| A5: Bounded Verification | +1 | Profile validation is local (entry shape + effects) |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Profiles are machine-decidable, CLI flag or auto-detected |
| A8: Minimal Syntax | 0 | No new syntax required |
| A9: Cost Visibility | +1 | Effect budgets show resource implications upfront |
| A10: Composability | +1 | Profiles compose with existing effect system |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | +1 | Profiles define explicit entry/exit boundaries |

**Net Score: +9** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Profiles enforce deterministic effect budgets
- [x] A3 (Effects): All effects remain explicit and typed per profile
- [x] A4 (Authority): Effect budgets constrain ambient access
- [x] A7 (Machines First): CLI auto-detection, not human-centric prose

---

## 1. Motivation

AILANG originally gained Go codegen support for real-time game simulations ("stapledon engine").
But the underlying semantics we designed—deterministic world-transition functions + explicit effect contexts—are not game-specific at all.

They form a **general computational model** suitable for:
- Multi-agent environments
- Intelligent agents (LLM-driven or RNN-driven)
- Workflow/state-machine engines
- Request/response microservices
- CLI tools
- Batch/ETL processing
- WASM/web simulations
- Offline scientific models
- Chatbot or tool-using agent cores

This document defines a formal notion of **"execution profiles"** that AILANG can target consistently across compilers, interpreters, build pipelines, and documentation.

### What Execution Profiles Enable

Profiles let AILANG:
- **Stay language-simple** (pure FP + effects + ADTs + arrays)
- **Stay compiler-simple** (same IR → multiple backends)
- **Gain broad applicability** without feature creep
- **Give partners/clients guarantees** about what they can rely on

---

## 2. Core Concept: AILANG is a Typed State-Machine DSL

At the semantic level, every serious AILANG program takes one of three shapes:

### A. Stateful Step Functions (Simulations, Agents)

```ailang
func step(world: World, input: Input) -> (World, Output) ! {Effects}
```

### B. Stateless Handler Functions (CLI, Microservices, Batch)

```ailang
func handle(request: Request) -> Response ! {Effects}
```

### C. Entrypoint Functions (CLI & Tools)

```ailang
func main(args: [string]) -> () ! {Effects}
```

Everything else—ADT pattern matching, recursion, lists, arrays, JSON encode/decode, Debug, AI—is orthogonal and profile-independent.

---

## 3. Execution Profiles

AILANG defines three profiles, each with a clear contract and effect budget.

These profiles define:
- **The shape of the entry function**
- **Which effects are allowed**
- **How the Go (or other) backend generates code**
- **What host responsibilities are** (e.g., providing contexts)

---

### Profile 1: SimProfile

**Primary use:** Games, multi-agent environments, workflow engines, agent worlds

**Entry Signatures:**
```ailang
func init(seed: int, params: InitParams) -> World ! {Debug}
func step(world: World, input: Input) -> (World, Output) ! {RNG, Debug, AI}
```

**Effect Budget:**
| Effect | Meaning | Notes |
|--------|---------|-------|
| RNG | Deterministic PRNG seeded by host | Required for simulations |
| Debug | Structured logs/assertions | Optional tracing |
| AI | Pluggable JSON-in/JSON-out effect | LLM integration |
| Time | Controlled virtual time | Future (optional) |

**Backend Support:**
- ✅ Interpreter
- ✅ Go codegen (high priority)
- 🔜 WASM codegen (future)

**Host Responsibilities:**
- Supply contexts: `RNGContext`, `AIContext`, `DebugContext`
- Advance logical tick counters
- Maintain world instance between steps

**Why SimProfile is Powerful:**
Exactly the same shape as:
- RL Gym environments (`reset()` + `step()`)
- Game loops (init + update)
- Agent-swarm simulators
- State-machine workflow engines
- Multi-agent modelling (economic, ecological simulations)

---

### Profile 2: ServiceProfile

**Primary use:** Microservices, agent tools, HTTP/gRPC handlers, request classifiers

**Entry Signature:**
```ailang
pure func handle(req: Request) -> Response ! {AI, Debug}
```

**Effect Budget:**
| Effect | Meaning | Notes |
|--------|---------|-------|
| AI | Call out to LLM or policy model | Core capability |
| Debug | Structured logs/assertions | Optional |
| FS | File access | Optional, sandboxed |
| Env | Environment variables | Optional |
| Clock | Time queries | Optional |

**Backend Support:**
- ✅ Interpreter for quick dev
- ✅ Go codegen (adapter emits HTTP/gRPC server)

**Host Responsibilities:**
- Map HTTP/gRPC requests → `Request`
- Provide `AIContext` (OpenAI, VertexAI, Local models)
- Encapsulate all network, DB, FS effects (not done in AILANG)

**Key Insight:**
The AI effect + JSON encoding makes AILANG handlers ideal for "cognitive microservices" or tools for LLM agents.

---

### Profile 3: CliProfile

**Primary use:** Command-line tools, scripting, utilities, config transformers

**Entry Signature:**
```ailang
func main(args: [string]) -> () ! {IO, FS, Env, Debug}
```

**Effect Budget:**
| Effect | Meaning | Notes |
|--------|---------|-------|
| IO | stdout printing | Core |
| FS | File reading/writing | Core |
| Env | Environment variables, CLI args | Core |
| Debug | Assertions/logging | Optional |

**Backend Support:**
- ✅ Interpreter
- ✅ Go codegen → standalone binary

**Host Responsibilities:**
- No special API: AILANG program IS the tool
- Provide FS/IO/Env contexts

---

## 4. Cross-Cutting Runtime Components

### 4.1 Effect Contexts (Reusable Across Profiles)

| Effect | Meaning | Provided by Host? | SimProfile | ServiceProfile | CliProfile |
|--------|---------|-------------------|------------|----------------|------------|
| RNG | Seeded deterministic random | ✅ | ✅ | ❌ | ❌ |
| Debug | Structured logs/asserts | ✅ | ✅ | ✅ | ✅ |
| AI | JSON-based model calls | ✅ | ✅ | ✅ | optional |
| IO | Text printing | ✅ | optional | optional | ✅ |
| FS | File IO | ✅ | ❌ | optional | ✅ |
| Env | CLI args, environment | ✅ | ❌ | optional | ✅ |

**Key Principle:** All effects produce pure trace data (not side effects) which the host interprets.

### 4.2 Go Runtime Package (Shared)

The Go backend produces a package:

```
<module>/
    world.go       -- types + core structs
    funcs.go       -- compiled AILANG functions
    effects.go     -- effect interface stubs
    debug.go       -- Debug effect impl
    ai.go          -- AI effect impl
    rng.go         -- RNG context
    step.go        -- step/init wrappers if SimProfile
    main.go (?)    -- only for CliProfile
```

These runtime components are shared across all profiles.

---

## 5. How Profiles Share the Same IR

All profiles compile through:

```
Surface → Core → ANF → Effect-Lowered IR → Go
```

The only difference is **what wrapper we generate**:
- **SimProfile:** `Init()` + `Step()` wrappers
- **ServiceProfile:** `Handle()` wrapper
- **CliProfile:** `Main()` wrapper

Everything else (pattern match, ADTs, arrays, functions) is **identical**.

**Key Point:** Profiles do not fragment the compiler. They only define entry semantics + effect budgets.

---

## 6. CLI Integration

### Profile Flag

```bash
# Compile with explicit profile
ailang compile mymod.ail --profile sim --emit-go
ailang compile tool.ail --profile cli --emit-go
ailang compile service.ail --profile service --emit-go

# Profile auto-detection based on entry function shape
ailang compile mymod.ail --emit-go  # Detects from main/step/handle
```

### Profile Validation

The compiler validates that:
1. Entry function matches profile shape
2. Only allowed effects are used
3. Required effects have contexts

---

## 7. Future Profiles (Proposed)

### Profile 4: BatchProfile (v0.7+)

```ailang
func run(config: Config, data: [Record]) -> [Result] ! {FS, Debug}
```

Good for ETL/ML preprocessing jobs.

### Profile 5: WasmProfile (v0.7+)

```ailang
func step(world: World, input: Input) -> (World, Output)
```

Same semantics as SimProfile but compiled to WASM for browser simulations.

### Profile 6: AgentProfile (v0.7+)

AILANG as the deterministic "core brain" of tool-using LLM agents.

```ailang
func decide(state: AgentState, observation: Obs) -> (AgentState, Action) ! {AI, Tools, Debug}
```

This is the direct evolution of SimProfile into a **full agent framework**.

---

## 8. Strategic Justification

### 8.1 Go is the Ideal Host Language
- Predictable runtime
- Static binary linking
- Excellent embedding story
- Can call anything (LLMs, FS, network)

### 8.2 Effects-as-Contexts Make AILANG Embedding Safe
All side effects flow into host-owned contexts, never into AILANG code.
This gives us:
- Reproducibility
- Determinism
- Safety constraints
- Policy-enforceable boundaries

### 8.3 Profiles Turn AILANG into a Multi-Domain DSL
You now serve:
- Simulators
- Game engines
- Cognitive microservices
- CLI tooling
- Agent frameworks

All from **one IR and one compiler**, not a fragmented ecosystem.

---

## 9. Implementation Plan

### Phase 1: Profile Infrastructure (~3 days)
- [ ] Add `--profile` flag to `ailang compile`
- [ ] Create `internal/profile/` package for profile definitions
- [ ] Define `SimProfile`, `ServiceProfile`, `CliProfile` structs
- [ ] Add profile validation to compilation pipeline

### Phase 2: Entry Function Detection (~2 days)
- [ ] Auto-detect profile from entry function shape
- [ ] Validate effect budget against profile
- [ ] Generate appropriate wrappers

### Phase 3: Go Runtime Components (~3 days)
- [ ] Factor out shared runtime (effects.go, debug.go, ai.go)
- [ ] Create profile-specific entry wrappers
- [ ] Add context injection for effects

### Phase 4: Documentation & Examples (~2 days)
- [ ] Update CLI help with profile documentation
- [ ] Create example for each profile
- [ ] Update website with profiles architecture

---

## 10. Success Criteria

- [ ] `--profile sim|service|cli` flag works
- [ ] Profile auto-detection from entry function
- [ ] Effect budget validation per profile
- [ ] Go codegen produces correct wrappers for each profile
- [ ] At least one working example per profile
- [ ] Documentation updated

---

## 11. Files to Modify/Create

**New files:**
- `internal/profile/profile.go` (~200 LOC) - Profile definitions
- `internal/profile/detect.go` (~100 LOC) - Auto-detection
- `internal/profile/validate.go` (~150 LOC) - Effect budget validation
- `runtime/sim/` - SimProfile Go runtime
- `runtime/service/` - ServiceProfile Go runtime
- `examples/profiles/` - Example per profile

**Modified files:**
- `cmd/ailang/compile.go` - Add `--profile` flag
- `internal/gen/golang/codegen.go` - Profile-aware generation
- `internal/pipeline/pipeline.go` - Profile validation step

---

## 12. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Profile proliferation | Medium | Limit to 3 core profiles, mark others as experimental |
| Effect budget confusion | Low | Clear documentation, helpful error messages |
| Breaking existing users | Low | Default to CliProfile for backward compatibility |

---

## 13. References

- Go codegen implementation (v0.5.x)
- Effect system design (v0.3.x)
- [stapledon game project](https://github.com/sunholo-data/stapledons_voyage) - SimProfile user
- CUDA shared memory analogy for effect contexts

---

## 14. Conclusion

The Go codegen work wasn't "just for games".
It uncovered AILANG's true architecture:

> **A deterministic FP DSL with effect contexts, compiling into pluggable state machines embedded in Go.**

By formalizing this into execution profiles, we now have a roadmap that:
- Serves multiple product verticals
- Avoids fragmentation
- Reuses 95% of compiler infrastructure
- Establishes AILANG as a serious language for simulations, agents, and services
- Maximizes long-term leverage of every feature we add

---

**Document created**: 2025-12-03
**Last updated**: 2025-12-03
**Author**: Mark Edmondson & Multivac R&D
