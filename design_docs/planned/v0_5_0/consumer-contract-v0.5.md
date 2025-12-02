# AILANG Consumer Contract v0.5.x

**For: Stapledons Voyage (and other game/simulation consumers)**
**AILANG Version: v0.5.0 - v0.5.3**
**Status: Stable Preview (breaking changes allowed until v0.6.0)**

This document defines what AILANG guarantees to consumer projects that use AILANG as their simulation logic language with Go codegen.

---

## What We Assume

As a consumer of AILANG v0.5.x, we assume:

### 1. Go Codegen Available (v0.5.0+)

```bash
ailang compile --emit-go --package-name <name> --out <dir> <file.ail>
```

**Guarantees:**
- Generates valid, compilable Go code
- Output directory structure matches package name
- Generated code passes `go build` and `go vet`
- Deterministic output (same input → same output)

### 2. ADT → Discriminator Structs (v0.5.0+)

AILANG sum types generate discriminator-based Go structs, NOT interfaces.

**We define in AILANG:**
```ailang
type DrawCmd =
  | Sprite(x: int, y: int, id: int)
  | Rect(x: int, y: int, w: int, h: int, color: int)
  | Text(x: int, y: int, content: string)
```

**We receive in Go:**
```go
type DrawCmdKind int

const (
    DrawCmdKindSprite DrawCmdKind = iota
    DrawCmdKindRect
    DrawCmdKindText
)

type DrawCmd struct {
    Kind   DrawCmdKind
    Sprite *DrawCmdSprite
    Rect   *DrawCmdRect
    Text   *DrawCmdText
}

type DrawCmdSprite struct {
    X, Y, Id int64
}

type DrawCmdRect struct {
    X, Y, W, H, Color int64
}

type DrawCmdText struct {
    X, Y    int64
    Content string
}
```

**Why this matters:**
- No interface dispatch overhead in hot loops
- Cache-friendly contiguous memory layout
- Predictable switch-based pattern matching

### 3. Exported Functions Callable from Go (v0.5.0+)

**We define in AILANG:**
```ailang
export func init_world(seed: int) -> World { ... }

export func step(world: World, input: FrameInput) -> (World, FrameOutput) ! {RNG, Debug} { ... }
```

**We receive in Go:**
```go
func InitWorld(seed int64) World { ... }

func Step(world World, input FrameInput) (World, FrameOutput, error) { ... }
```

**Guarantees:**
- `export func` → public Go function (PascalCase)
- Non-exported → package-private (unexported)
- Effects propagate as `error` return value
- Pure functions have no error return

### 4. RNG Effect with Determinism (v0.5.1+)

**We use in AILANG:**
```ailang
func generate_map(seed: int) -> Map ! {RNG} {
    let width = RNG.rand_int(100)
    let height = RNG.rand_int(100)
    ...
}
```

**Guarantees:**
- `AILANG_SEED=N` produces identical sequences
- `RNG.rand_float()` returns `[0, 1)`
- `RNG.rand_int(max)` returns `[0, max)`
- Capability check enforced at runtime

### 5. Debug Effect with Structured Output (v0.5.1+)

**We use in AILANG:**
```ailang
func update_entity(e: Entity) -> Entity ! {Debug} {
    -- Locations auto-injected by compiler
    Debug.assert(e.health >= 0, "health must be non-negative")
    Debug.log("updating entity " ++ show(e.id))
    ...
}

func step(world: World, input: FrameInput) -> (World, FrameOutput) ! {RNG, Debug} {
    -- Debug writes accumulate during execution
    -- Host collects after step returns (not callable from AILANG)
    process_tick(world, input)
}
```

**Guarantees:**
- Assertions collected, not thrown
- Debug is **write-only** from AILANG (no `collect()` in language)
- Host calls `DebugContext.Collect()` after step returns
- Structured output for eval harness consumption
- `--release` mode **erases Debug from effect rows** (ghost effect)
- Location strings auto-injected by compiler

**DebugOutput schema (shared across backends):**
```ailang
type DebugOutput = {
    logs: [LogEntry],
    assertions: [AssertionResult]
}

type LogEntry = { message: string, location: string, timestamp: int }
type AssertionResult = { passed: bool, message: string, location: string }
```

**Host integration (Go example):**
```go
debugCtx := game.NewDebugContext()
debugCtx.SetTimestamp(int64(tick))
world, output, err := game.Step(world, input, debugCtx, rngCtx)
debugData := debugCtx.Collect()  // Host-only operation
debugCtx.Reset()  // Clear for next tick
```

### 6. AI Effect with Pluggable Handler (v0.5.1+)

**AILANG core provides:**
```ailang
effect AI {
    call(input: string) -> string  -- Opaque string→string, JSON by convention
}
```

**We wrap with our own typed interface:**
```ailang
type NPCContext = { position: Vec2, health: int, nearby: [Entity] }
type NPCAction =
  | Move(Vec2)
  | Attack(int)
  | Wait

func choose_action(ctx: NPCContext) -> NPCAction ! {AI} {
    let input = std/json.encode(ctx)
    let output = AI.call(input)
    match std/json.decode[NPCAction](output) {
        Ok(action) => action,
        Err(_)     => Wait  -- Safe fallback
    }
}
```

**Guarantees:**
- Generic string→string interface (JSON by convention, not enforced)
- Operation named `call` (neutral), not `decide` (game-flavored)
- Handler pluggable at Go runtime level
- Nil handler = **error** (no silent stub fallback)
- Explicit stub handler for tests: `game.NewStubAIHandler()`
- Can swap to real AI without recompiling AILANG

**Host integration (Go example):**
```go
// EXPLICIT handler choice - no silent fallback
var aiHandler game.AIHandler
switch os.Getenv("GAME_AI_MODE") {
case "openai":
    aiHandler = NewOpenAIHandler(os.Getenv("OPENAI_API_KEY"))
case "stub":
    aiHandler = game.NewStubAIHandler()
default:
    log.Fatal("GAME_AI_MODE must be set")
}
aiCtx := game.NewAIContext(aiHandler)
```

### 7. Extern Functions for Performance Kernels (v0.5.2+)

**We declare in AILANG:**
```ailang
extern func find_path(world: World, from: Coord, to: Coord) -> Path
extern func compute_influence(world: World, source: Coord) -> [[float]]
```

**We implement in Go:**
```go
// In path_impl.go (our code, not generated)
func FindPath(world World, from Coord, to Coord) Path {
    // A* implementation here
}

func ComputeInfluence(world World, source Coord) [][]float64 {
    // Influence map implementation here
}
```

**Guarantees:**
- AILANG generates stub signatures
- Type compatibility checked at compile time
- Clear error if extern not implemented
- Monomorphic types only (no generics in v0.5.x)

**Supported extern types:**
- `int`, `float`, `bool`, `string`
- Structs/records generated by AILANG
- `[T]` → `[]T` in Go

**NOT supported (v0.5.x):**
- Polymorphic externs
- Function parameters
- Higher-kinded types

---

## What We Provide

As a consumer, we commit to:

### 1. Protocol Definition

We define our own `World`, `FrameInput`, `FrameOutput` types:

```ailang
-- sim/protocol.ail
module sim/protocol

type Vec2 = { x: float, y: float }
type Coord = { x: int, y: int }

type FrameInput = {
    tick: int,
    dt: float,
    keys_pressed: [int],
    mouse_pos: Vec2
}

type FrameOutput = {
    draw_cmds: [DrawCmd],
    sounds: [SoundCmd],
    debug: DebugOutput
}

type DrawCmd =
  | Sprite(x: int, y: int, sprite_id: int)
  | Rect(x: int, y: int, w: int, h: int, color: int)
  | Text(x: int, y: int, content: string)
```

### 2. World State Definition

We define game-specific world state:

```ailang
-- sim/world.ail
module sim/world

import sim/protocol

type Entity = {
    id: int,
    pos: Vec2,
    health: int,
    kind: EntityKind
}

type EntityKind = | Player | NPC | Item

type World = {
    tick: int,
    entities: [Entity],
    map_data: MapData
}
```

### 3. Step Function Implementation

We implement the core game loop:

```ailang
-- sim/step.ail
module sim/step

import sim/protocol
import sim/world

export func init_world(seed: int) -> World ! {RNG} {
    -- Initialize world with RNG
}

export func step(world: World, input: FrameInput) -> (World, FrameOutput) ! {RNG, Debug} {
    -- Update world state
    -- Collect draw commands
    -- Return new world + output
}
```

### 4. Go Driver

We implement the Go side that calls AILANG:

```go
// cmd/game/main.go
package main

import (
    "github.com/our-org/stapledons-voyage/gen/sim"
    "github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
    world sim.World
}

func (g *Game) Update() error {
    input := captureInput()
    newWorld, output, err := sim.Step(g.world, input)
    if err != nil {
        return err
    }
    g.world = newWorld
    g.renderOutput(output)
    return nil
}
```

### 5. Extern Implementations

We implement performance-critical Go code:

```go
// internal/pathfinding/astar.go
package pathfinding

import "github.com/our-org/stapledons-voyage/gen/sim"

func FindPath(world sim.World, from, to sim.Coord) sim.Path {
    // Our A* implementation
}
```

---

## Integration Test Contract

To verify AILANG fulfills this contract, we expect:

### AILANG Provides (in `examples/sim_stub/`)

1. Minimal `world.ail` with types matching this pattern
2. Minimal `main.go` driver that runs 10 ticks
3. CI job that compiles → builds → runs → validates output
4. **Debug effect demonstration** with DebugContext lifecycle ✅ (v0.4.10+)
5. **Build-tagged release mode** with no-op Debug implementation ✅ (v0.4.10+)

### We Verify

1. Our types compile with AILANG codegen
2. Generated Go code builds without errors
3. Step function executes deterministically with same seed
4. Debug output accessible via DebugContext.Collect() ✅ (host-controlled)
5. Extern stubs generate with correct signatures
6. Release mode builds successfully with `-tags release` ✅

---

## Version Compatibility

| Feature | Minimum AILANG Version | Status |
|---------|------------------------|--------|
| Go codegen | v0.5.0 | Planned |
| ADT → discriminator structs | v0.5.0 | Planned |
| `export func` | v0.5.0 | Planned |
| RNG effect | v0.5.1 | Planned |
| **Debug effect (Go codegen)** | **v0.4.10** | **✅ Implemented** |
| **Debug release mode (build tags)** | **v0.4.10** | **✅ Implemented** |
| **AI effect** | **v0.4.10** | **✅ Implemented** |
| `extern func` | v0.5.2 | Planned |
| CLI flags (`--out`, `--package-name`) | v0.5.2 | Planned |
| ABI "stable preview" | v0.5.3 | Planned |

---

## Breaking Change Policy

**v0.5.x (Stable Preview):**
- Breaking changes allowed with notice
- We track AILANG releases and update accordingly
- Report issues via AILANG agent messaging system

**v0.6.0+ (Stable):**
- No breaking changes to codegen output
- ADT representation locked
- Effect signatures locked
- Extern ABI locked

---

## Contact

- **AILANG Issues**: https://github.com/sunholo-data/ailang/issues
- **Agent Messaging**: `ailang agent send ailang-core '{"type": "feedback", ...}'`
- **Sprint Plan**: See `design_docs/planned/v0_4_8/M-GAME-ENGINE-sprint-plan.md`

---

*This contract was drafted alongside the M-GAME-ENGINE sprint plan. It represents the integration agreement between AILANG and Stapledons Voyage.*
