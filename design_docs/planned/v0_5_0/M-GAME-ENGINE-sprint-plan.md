# Sprint Plan: M-GAME-ENGINE - Game Support Enablement

## Summary

Enable AILANG to function as a deterministic simulation and AI logic language for external game projects, with stable Go interop, exported function ABI, effects (RNG/AI/Debug), and extern function support.

**Duration:** 8-10 weeks (across v0.5.0 - v0.5.3 releases)
**Dependencies:** Core language stability (v0.4.x), ADT implementation complete
**Risk Level:** HIGH - This is a new backend (Go codegen), not just language features

## Executive Assessment

### This is a Major Architecture Change

**What exists today:**
- AILANG compiles to Core AST → interpreted by Go evaluator
- Effects exist in type system (IO, FS, Net, Clock, Rand) with runtime capability grants
- ADTs parse and evaluate correctly, with pattern matching and exhaustiveness checking
- No Go code generation whatsoever

**What this design requires:**
- New compilation target: AILANG → Go source code
- Stable ABI for exported functions callable from Go
- Effect handlers that return errors to Go callers
- Extern function stubs for Go-implemented performance kernels

**Gap analysis: ~4,500-5,500 LOC of new code**

## Current Status Analysis

### What Already Exists (✅)

| Component | Status | Notes |
|-----------|--------|-------|
| ADT syntax parsing | ✅ Complete | `type Tree = \| Leaf(int) \| Node(Tree, int, Tree)` |
| Pattern matching | ✅ Complete | Exhaustiveness checking, guards |
| Effect type system | ✅ Complete | Row polymorphism, capability tracking |
| Effect runtime | ✅ Partial | IO, FS, Net, Clock implemented |
| AILANG_SEED | ✅ Complete | Deterministic randomness infrastructure |
| Core AST lowering | ✅ Complete | Surface → Core elaboration |
| Interpreter | ✅ Complete | Core evaluator with effects |

### What Needs to Be Built (❌)

| Component | Status | Estimated LOC |
|-----------|--------|---------------|
| Go codegen module | ❌ Not started | ~1,500 |
| ADT → Go struct codegen | ❌ Not started | ~400 |
| Function → Go func codegen | ❌ Not started | ~600 |
| `export func` syntax | ❌ Not started | ~200 |
| `extern func` syntax | ❌ Not started | ~200 |
| RNG effect implementation | ❌ Not started | ~150 |
| Debug effect (assert/log) | ❌ Not started | ~150 |
| AI effect stub | ❌ Not started | ~100 |
| Compiler flags (--emit-go, etc.) | ❌ Not started | ~200 |
| Sim example + CI | ❌ Not started | ~400 |
| Tests | ❌ Not started | ~1,500 |

### Velocity Analysis

**Recent releases:**
- v0.4.7: ~400 LOC (M-TESTING-DEPS)
- v0.4.5: ~260 LOC (M-BUG-CONCAT-INFERENCE)
- v0.4.0: ~800 LOC (Monomorphization Phase 1)

**Calculated velocity:** 100-150 LOC/day of production code (including tests)

**Estimated timeline:** 4,500 LOC ÷ 125 LOC/day = **36 working days** (~7-8 weeks)

## Proposed Phased Approach

Given the scope, this should be split across **4 releases**:

### Phase 1: v0.5.0 - Go Codegen Foundation (M-GAME-A)
### Phase 2: v0.5.1 - Effects for Games (M-GAME-B)
### Phase 3: v0.5.2 - Compiler UX & Extern (M-GAME-C)
### Phase 4: v0.5.3 - Sim Example & Integration (M-GAME-D)

---

## Phase 1: v0.5.0 - Go Codegen Foundation (M-GAME-A)

**Goal:** Establish the `ailang/gen/go` module and generate valid Go code for ADTs and pure functions.

**Estimated:** 800 LOC implementation + 400 LOC tests = **1,200 LOC**
**Duration:** 2 weeks

### Milestone A1: Go Codegen Infrastructure
**Day 1-3:** Create `internal/gen/go/` package structure

**Tasks:**
- Create `internal/gen/go/codegen.go` - main entry point
- Create `internal/gen/go/types.go` - type lowering
- Create `internal/gen/go/naming.go` - stable naming scheme
- Create `internal/gen/go/package.go` - Go package generation

**Acceptance Criteria:**
- [ ] Package structure compiles
- [ ] Can generate empty Go package with correct structure
- [ ] Unit tests for naming scheme (CamelCase conversion)

### Milestone A2: ADT → Go Struct Generation (Discriminator-Based)
**Day 4-7:** Generate Go types from AILANG ADTs

**Tasks:**
- Implement sum type codegen (discriminator struct pattern)
- Implement product type codegen (plain structs)
- Implement recursive type handling
- Implement list/array/map type mapping

**Example transformation:**
```ailang
type Tree = | Leaf(int) | Node(Tree, int, Tree)
```
→
```go
// Discriminator-based sum type (NOT interfaces - see Design Decisions)
type TreeKind int

const (
    TreeKindLeaf TreeKind = iota
    TreeKindNode
)

type Tree struct {
    Kind TreeKind
    Leaf *TreeLeaf  // populated when Kind == TreeKindLeaf
    Node *TreeNode  // populated when Kind == TreeKindNode
}

type TreeLeaf struct {
    Value0 int64
}

type TreeNode struct {
    Left  *Tree
    Value int64
    Right *Tree
}
```

**Acceptance Criteria:**
- [ ] Sum types generate discriminator structs (not interfaces)
- [ ] Product types generate plain structs
- [ ] Recursive types compile correctly
- [ ] Generated Go code compiles with `go build`
- [ ] Branchy-but-contiguous layout suitable for hot loops

### Milestone A3: Pure Function → Go Func Generation
**Day 8-12:** Generate Go functions from pure AILANG functions

**Tasks:**
- Implement expression codegen (literals, binops, if/then/else)
- Implement pattern matching codegen (match → switch)
- Implement lambda/closure codegen
- Implement recursion handling

**Example transformation:**
```ailang
func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}
```
→
```go
func Factorial(n int64) int64 {
    if n <= 1 {
        return 1
    }
    return n * Factorial(n - 1)
}
```

**Acceptance Criteria:**
- [ ] Pure functions generate correct Go code
- [ ] Recursion works
- [ ] Pattern matching generates switch statements
- [ ] Generated code passes `go vet`

### Milestone A4: `export func` Syntax
**Day 13-14:** Add export keyword to parser

**Tasks:**
- Add EXPORT token to lexer
- Add `export func` parsing
- Add export flag to AST function node
- Wire through to codegen

**Acceptance Criteria:**
- [ ] `export func foo() -> int { 42 }` parses
- [ ] Non-exported functions are package-private in Go
- [ ] Exported functions are public (PascalCase)

**Risks:**
- Pattern matching on discriminator structs may generate verbose switch statements
- Mitigation: Keep generated code readable; optimize later if profiling shows issues

---

## Phase 2: v0.5.1 - Effects for Games (M-GAME-B)

**Goal:** Implement RNG, Debug, and AI stub effects with Go runtime handlers.

**Estimated:** 500 LOC implementation + 300 LOC tests = **800 LOC**
**Duration:** 1.5 weeks

### Milestone B1: RNG Effect
**Day 1-3:** Implement seeded random number generation

**Tasks:**
- Add `RNG` effect capability check (surface name: `RNG`, not `Rand`)
- Implement `RNG.rand_float() -> float` builtin
- Implement `RNG.rand_int(max: int) -> int` builtin
- Wire to Go's `math/rand` with AILANG_SEED

**AILANG surface:**
```ailang
-- Effect name is RNG (consistent across all docs)
effect RNG {
    rand_float() -> float
    rand_int(max: int) -> int
}
```

**Go runtime:**
```go
type RNGContext struct {
    rng *rand.Rand  // Internal name can differ from surface
}

func (ctx *RNGContext) RandFloat() float64 {
    return ctx.rng.Float64()
}

func (ctx *RNGContext) RandInt(max int64) int64 {
    return ctx.rng.Int63n(max)
}
```

**Acceptance Criteria:**
- [ ] `RNG.rand_float()` returns value in [0, 1)
- [ ] `RNG.rand_int(10)` returns value in [0, 10)
- [ ] Same seed produces same sequence
- [ ] `RNG` capability check enforced

### Milestone B2: Debug Effect (Structured Output for Eval Harness)
**Day 4-6:** Implement assert and log with structured output

**Tasks:**
- Add `Debug` effect capability
- Implement `assert(cond: bool, msg: string) -> unit`
- Implement `log(msg: string) -> unit`
- Collect logs/asserts in DebugOutput (part of FrameOutput)
- Support `--debug` vs `--release` build modes

**Go runtime:**
```go
// DebugOutput is embedded in FrameOutput, accessible to eval harness
type DebugOutput struct {
    Logs       []LogEntry
    Assertions []AssertionResult
}

type LogEntry struct {
    Message  string
    Location string  // "file.ail:42:10"
}

type AssertionResult struct {
    Passed   bool
    Message  string
    Location string  // "file.ail:42:10"
}
```

**Build modes:**
- `--debug`: Assertions collected in FrameOutput.Debug, available for report.json
- `--release`: Assertions compiled out (no-op, zero overhead)

**Acceptance Criteria:**
- [ ] `assert(true, "ok")` passes silently
- [ ] `assert(false, "fail")` records failure with location
- [ ] `log("message")` records message with location
- [ ] Debug data accessible in FrameOutput (not just error channel)
- [ ] Eval harness can persist all assertions to report.json
- [ ] Release mode compiles out debug overhead

### Milestone B3: AI Effect Stub (Generic JSON Interface)
**Day 7-10:** Create generic AI effect - domain types live in consumer repos

**Tasks:**
- Define generic AI effect in AILANG core: `AI.decide(input: string) -> string`
- Implement stub handler returning deterministic JSON
- Document typed wrapper pattern for consumer repos
- Wire to Go runtime with extension points

**AILANG core provides:**
```ailang
-- Generic JSON-in/JSON-out interface
effect AI {
    decide(input: string) -> string
}
```

**Consumer repo (PlanetWorld) wraps with typed interface:**
```ailang
type NPCContext = { position: Vec2, health: int, nearbyEntities: [Entity] }
type NPCAction = { kind: string, target: Vec2 }

func choose_action(ctx: NPCContext) -> NPCAction ! {AI} {
    let input = std/json.encode(ctx)
    let output = AI.decide(input)
    std/json.decode[NPCAction](output)
}
```

**Go runtime:**
```go
type AIContext struct {
    Handler func(input string) (string, error)  // Pluggable
}

// Default stub returns deterministic placeholder
func DefaultAIHandler(input string) (string, error) {
    return `{"kind": "idle", "target": {"x": 0, "y": 0}}`, nil
}
```

**Acceptance Criteria:**
- [ ] AI effect type-checks with generic string signature
- [ ] Stub returns deterministic JSON action
- [ ] Consumer repos can define their own typed wrappers
- [ ] Handler is pluggable (local model, remote API, etc.)
- [ ] ABI stable - can swap implementation without recompiling AILANG

**Why generic?** AILANG core shouldn't know what an NPC is. Domain types belong in consumer repos.

---

## Phase 3: v0.5.2 - Compiler UX & Extern (M-GAME-C)

**Goal:** Add compiler flags for Go output and extern function support.

**Estimated:** 500 LOC implementation + 300 LOC tests = **800 LOC**
**Duration:** 1.5 weeks

### Milestone C1: Compiler Flags
**Day 1-3:** Add --emit-go, --out, --package-name

**Tasks:**
- Add `--emit-go` flag to `ailang run`
- Add `--out <dir>` for output directory
- Add `--package-name <name>` for Go package
- Integrate with existing CLI structure

**Usage:**
```bash
ailang compile --emit-go --package-name game --out gen/ world.ail
```

**Acceptance Criteria:**
- [ ] `--emit-go` generates Go files instead of running
- [ ] `--out` controls output directory
- [ ] `--package-name` sets Go package name
- [ ] Help text documents new flags

### Milestone C2: Extern Function Support (Monomorphic Only)
**Day 4-7:** Allow Go-implemented performance helpers

**Tasks:**
- Add EXTERN token to lexer
- Parse `extern func find_path(...) -> Path`
- Generate Go function stubs
- Type-check extern signatures (monomorphic types only)

**Supported types (v0.5.x):**
- Primitives: `int`, `float`, `bool`, `string`
- Structs/records already generated by AILANG
- Lists/arrays that map to `[]T`

**NOT supported (deferred to v0.6.0+):**
- ❌ Polymorphic externs: `extern func map[a,b](f: a -> b, xs: [a]) -> [b]`
- ❌ Higher-kinded types
- ❌ Function parameters

**Example:**
```ailang
extern func find_path(world: World, from: Coord, to: Coord) -> Path
extern func compute_influence_map(world: World, source: Coord) -> [[float]]
```
→
```go
// Generated stub - implement in path_impl.go:
// func FindPath(world World, from Coord, to Coord) Path { ... }

// Generated stub - implement in influence_impl.go:
// func ComputeInfluenceMap(world World, source Coord) [][]float64 { ... }
```

**Acceptance Criteria:**
- [ ] `extern func` parses correctly
- [ ] Stubs generated with correct signature
- [ ] Monomorphic type compatibility checked
- [ ] Clear error if extern not implemented
- [ ] Clear error if polymorphic extern attempted (with "not supported in v0.5.x" message)

### Milestone C3: Error Messages for Interop
**Day 8-10:** Improve errors for Go interop issues

**Tasks:**
- Add clear errors for type mismatches in codegen
- Add warnings for non-serializable types
- Add suggestions for extern function implementation
- Add diagnostic for missing exports

**Acceptance Criteria:**
- [ ] Error messages mention Go types explicitly
- [ ] Suggestions for fixing interop issues
- [ ] Documentation links in error messages

**Risks:**
- Extern function type checking complex
- Mitigation: Start with simple types, defer generics to v0.5.4

---

## Phase 4: v0.5.3 - Sim Example & Integration (M-GAME-D)

**Goal:** Create working example and CI integration proving the system works.

**Estimated:** 400 LOC implementation + 400 LOC tests = **800 LOC**
**Duration:** 1.5 weeks

### Milestone D1: Sim Stub Example
**Day 1-4:** Create `examples/sim_stub/`

**Tasks:**
- Create `world.ail` with World, FrameInput, FrameOutput types
- Implement trivial `init_world(seed: int) -> World`
- Implement trivial `step(world, input) -> (World, FrameOutput)`
- Create `main.go` that calls AILANG functions

**Example structure:**
```
examples/sim_stub/
├── world.ail          # AILANG game types and logic
├── main.go            # Go driver that calls AILANG
├── gen/               # Generated Go code (gitignored)
│   └── game/
│       └── world.go
└── README.md          # Usage instructions
```

**Acceptance Criteria:**
- [ ] `ailang compile --emit-go` generates valid Go
- [ ] `go run main.go` executes 10 ticks
- [ ] Output is deterministic with same seed
- [ ] README documents full workflow

### Milestone D2: CI Integration
**Day 5-7:** Add integration test to CI

**Tasks:**
- Add `make test-sim-stub` target
- Add GitHub Actions job for sim integration
- Test: compile AILANG → Go → build Go → run
- Verify output matches expected

**Acceptance Criteria:**
- [ ] CI job runs on every PR
- [ ] Fails if Go codegen broken
- [ ] Fails if generated Go doesn't compile
- [ ] Fails if output differs from expected

### Milestone D3: Documentation & ABI Freeze
**Day 8-10:** Document stable interfaces

**Tasks:**
- Document ADT → Go mapping rules
- Document exported function ABI
- Document effect handler interface
- Mark interfaces as stable (no breaking changes)

**Acceptance Criteria:**
- [ ] `docs/guides/go-interop.md` complete
- [ ] API stability promise in README
- [ ] CHANGELOG announces ABI freeze
- [ ] Migration guide for v0.4.x → v0.5.x

**Risks:**
- ABI freeze may be premature
- Mitigation: Mark as "stable preview", allow breaking changes until v0.6.0

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Test coverage (new code) | >80% |
| Generated Go compiles | 100% |
| Sim example runs | 10 ticks deterministically |
| Lint clean | All new code |
| Documentation | go-interop.md complete |
| Examples | sim_stub working end-to-end |

## Dependencies

1. **v0.4.x stability** - Core language must be stable before adding backend
2. **ADT implementation** - Already complete, no blockers
3. **Effect system** - Already designed, runtime needs work
4. **Go 1.21+** - For generics in generated code (optional)

## Design Decisions (Locked for v0.5.x)

### 1. Sum Type Representation: Discriminator Structs ✅

**Decision:** Use discriminator-based representation for all exported ADTs.

**Rationale:** Game engines iterate over thousands of DrawCmds per frame. Interface dispatch creates GC pressure and cache misses. Discriminator structs give branchy-but-contiguous layouts.

```go
// Generated sum type (discriminator-based)
type DrawCmdKind int

const (
    DrawCmdKindSprite DrawCmdKind = iota
    DrawCmdKindRect
    DrawCmdKindText
)

type DrawCmd struct {
    Kind   DrawCmdKind
    Sprite *DrawSprite  // populated when Kind == DrawCmdKindSprite
    Rect   *DrawRect    // populated when Kind == DrawCmdKindRect
    Text   *DrawText    // populated when Kind == DrawCmdKindText
}
```

Interfaces may be used for internal-only types, but exported ADTs use discriminators.

### 2. AI Effect: Generic JSON Interface ✅

**Decision:** AILANG core defines a generic AI effect. Game-specific types (NPCContext, NPCAction) live in consumer repos.

```ailang
-- AILANG core provides:
effect AI {
    decide(input: string) -> string  -- JSON-in/JSON-out
}

-- Game repo wraps with typed interface:
type NPCContext = { ... }
type NPCAction = { ... }

func choose_action(ctx: NPCContext) -> NPCAction ! {AI} {
    let input = std/json.encode(ctx)
    let output = AI.decide(input)
    std/json.decode[NPCAction](output)
}
```

**Rationale:** Keeps AILANG stdlib small and game-agnostic. Consumer repos define domain types.

### 3. Extern Functions: Monomorphic Only ✅

**Decision:** v0.5.x extern supports only monomorphic functions.

**Supported types:**
- Primitives: `int`, `float`, `bool`, `string`
- Structs/records already generated by AILANG
- Lists/arrays that map to `[]T`

**NOT supported (deferred to v0.6.0+):**
- Polymorphic externs (`extern func map[a,b](f: a -> b, xs: [a]) -> [b]`)
- Higher-kinded types
- Function parameters

**Rationale:** Keeps implementation manageable. Game only needs pathfinding, influence maps, etc.

### 4. Debug Effect: Structured Output for Eval Harness ✅

**Decision:** Debug assertions available in structured form, not hidden in Go error channels.

**AILANG core provides:**
- `Debug.assert(cond, msg)` and `Debug.log(msg)` effect operations
- Runtime collects assertions/logs and exposes them via `Debug.collect() -> DebugOutput`

**Consumer repos (PlanetWorld) wire it into their own protocol:**
```ailang
type FrameOutput = {
    draw_cmds: [DrawCmd],
    debug: DebugOutput,  -- Consumer defines this field
    ...
}

func step(world: World, input: FrameInput) -> (World, FrameOutput) ! {RNG, Debug} {
    -- Game logic with Debug.assert and Debug.log calls
    let result = update_world(world, input)

    -- Pull debug data into FrameOutput at end of tick
    let debug_data = Debug.collect()
    (result.world, { draw_cmds = result.cmds, debug = debug_data })
}
```

**Go types (generated by consumer, not AILANG core):**
```go
// DebugOutput - generated from consumer's AILANG type
type DebugOutput struct {
    Logs       []LogEntry
    Assertions []AssertionResult
}

type AssertionResult struct {
    Passed   bool
    Message  string
    Location string  // "file.ail:42:10"
}
```

**Build modes:**
- `--debug`: Debug effect active, collect() returns real data
- `--release`: Debug effect compiled out, collect() returns empty

**Rationale:** AILANG core stays generic; consumer repos define their own protocol. Eval harness can inspect all assertions, not die on first failure.

### 5. Effect Propagation: Error Return ✅

**Decision:** Effects propagate via explicit error return, not panic.

```go
func Step(world World, input FrameInput) (World, FrameOutput, error) {
    // Effects that fail → return error (recoverable)
    // Compiler/runtime bugs → panic (unrecoverable)
}
```

**Important distinction:**
- **Game/sim invariants** → Use `Debug.assert()`, collected in FrameOutput, never panic
- **Compiler/runtime bugs** (e.g., codegen produced invalid Go) → panic is acceptable

Game logic should never cause panics. If `Debug.assert(false, "...")` is called, it's recorded and the tick continues. The eval harness inspects all assertions at the end.

## Notes

- This is a **new compilation target**, not just language features
- External game repos depend on this ABI - stability is critical
- Start with interpreter path working, then add Go codegen
- Consider [TinyGo](https://tinygo.org/) compatibility for WASM games
- Keep generated code readable - developers will debug it

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Go codegen complexity | High | High | Start simple, iterate |
| ABI instability | Medium | High | Freeze early, version |
| Performance overhead | Medium | Medium | Benchmark, optimize hot paths |
| Extern type mismatches | Medium | Medium | Clear error messages |
| AI effect evolution | High | Low | Generic interface |

## Recommended Next Steps

1. **Approve phased approach** - Confirm v0.5.0-v0.5.3 breakdown ✅
2. **Design decisions locked** - See "Design Decisions (Locked for v0.5.x)" section ✅
3. **Start M-GAME-A (Phase 1)** - Go codegen foundation (2 weeks)
4. **Parallel: Bootstrap game repo** - Can start after v0.5.0 with pure codegen

---

**Total Estimated LOC:** 4,500+ (implementation + tests)
**Total Estimated Duration:** 8-10 weeks across 4 releases
**Risk Level:** HIGH - New backend requires careful design

---

## Consumer Timeline (PlanetWorld Game Repo)

The game repo doesn't need to wait until v0.5.3. Here's when to start consuming each capability:

### After v0.5.0 (Go Codegen Foundation) ← Game Repo Can Start Here!

**What's available:**
- ADT → Go codegen (discriminator structs)
- Pure function → Go function codegen
- `export func` syntax
- CLI: `ailang compile --emit-go`

**What game repo can do:**
- Define `World`, `FrameInput`, `FrameOutput`, `DrawCmd` in AILANG
- Use Go codegen for pure game logic (no effects yet)
- Wire Ebiten to draw from AILANG's `FrameOutput`
- Validate ADT ↔ Go mapping and build toolchain

**Example workflow:**
```bash
# In game repo
ailang compile --emit-go --package-name game --out gen/ world.ail
go build -o sim ./cmd/sim
./sim  # Draws colored rectangles from AILANG logic
```

### After v0.5.1 (Effects for Games)

**What's available:**
- RNG effect with AILANG_SEED determinism
- Debug effect (assert/log) with structured output
- AI effect stub (JSON-in/JSON-out)

**What game repo can do:**
- Add RNG-based procedural map generation
- Add assertions for game invariants
- Design AI wrappers over generic AI effect
- Start minimal eval harness (1-2 scenarios, 1 benchmark)

### After v0.5.2 (Compiler UX & Extern)

**What's available:**
- `extern func` for Go-implemented kernels
- Full CLI flags (`--out`, `--package-name`)
- Improved error messages for interop

**What game repo can do:**
- Implement extern pathfinding / influence maps
- Harden benchmark suite
- Loop AI-driven iteration on performance (report.json)

### After v0.5.3 (Sim Example & ABI Freeze)

**What's available:**
- Validated sim_stub example in AILANG repo
- CI integration tests proving ABI stability
- Documentation for Go interop

**Status:** ABI considered "stable preview" - breaking changes allowed until v0.6.0

---

## Alignment Check: Game Requirements vs Sprint Coverage

| Game Requirement | Sprint Coverage | Phase |
|------------------|-----------------|-------|
| ADT → Go codegen | ✅ A2 (discriminator structs) | v0.5.0 |
| Pure function codegen | ✅ A3 | v0.5.0 |
| `export func` | ✅ A4 | v0.5.0 |
| RNG effect | ✅ B1 | v0.5.1 |
| Debug/Assert effect | ✅ B2 (structured output) | v0.5.1 |
| AI effect | ✅ B3 (generic JSON) | v0.5.1 |
| CLI flags | ✅ C1 | v0.5.2 |
| Extern functions | ✅ C2 (monomorphic) | v0.5.2 |
| Sim example | ✅ D1 | v0.5.3 |
| CI integration | ✅ D2 | v0.5.3 |
| ABI documentation | ✅ D3 | v0.5.3 |

**Everything the game needs is covered. No overreach into renderer/UI territory.**
