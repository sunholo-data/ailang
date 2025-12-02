# Sim Stub Example

A minimal simulation example demonstrating the AILANG -> Go code generation workflow, including the **Debug effect** for structured tracing and the **AI effect** for pluggable AI oracle.

## Overview

This example shows:
1. **Type definitions** in AILANG (`world.ail`)
2. **Extern function declarations** for Go-implemented functions
3. **Generated Go types** from AILANG ADTs
4. **Deterministic simulation** with fixed seed
5. **Debug effect** for logging and assertions (ghost effect - erasable in release mode)
6. **AI effect** for pluggable AI calls (string→string, JSON by convention)

## Files

```
sim_stub/
├── world.ail              # AILANG types and extern function declarations
├── impl.go                # Go implementation with Debug effect usage
├── main.go                # Go driver with Debug context lifecycle
├── go.mod                 # Go module file
├── Makefile               # Build automation
├── expected_output.txt    # Golden file for CI testing
├── README.md              # This file
└── gen/                   # Generated code (gitignored)
    └── game/
        ├── types.go              # Generated type definitions
        ├── debug_types_debug.go  # Debug context (full implementation)
        ├── debug_types_release.go # Debug context (no-op for release)
        ├── ai_types.go           # AI effect types and context
        ├── handlers.go           # Effect handler interfaces
        └── extern_stubs.go       # Generated function stubs
```

## Usage

### Prerequisites

- Go 1.21 or later
- AILANG compiler (`ailang` in PATH)

### Build and Run

```bash
# Generate Go code, build, and run
make run

# Or step by step:
make generate  # Generate Go from AILANG
make build     # Build the Go binary
./sim_stub     # Run the simulation
```

### Test Determinism

```bash
make test
```

This runs the simulation and compares output to `expected_output.txt`.

## Workflow

1. **Define types in AILANG** (`world.ail`):
   ```ailang
   type World = { tick: int, value: int, seed: int }
   type FrameInput = { dummy: int }
   type FrameOutput = { message: string, tick: int }

   extern func init_world(seed: int) -> World
   extern func step(world: World, input: FrameInput) -> FrameOutput
   ```

2. **Generate Go code**:
   ```bash
   ailang compile --emit-go --package-name game world.ail
   ```

3. **Implement extern functions** (`impl.go`):
   - Uses generated types from `gen/game/types.go`
   - Implements `InitWorld()` and `Step()` functions

4. **Write the driver** (`main.go`):
   - Imports generated types
   - Calls the implemented functions

5. **Run and verify**:
   ```bash
   go run .
   ```

## Determinism

The simulation is fully deterministic:
- Same seed (42) always produces identical output
- RNG state is derived from seed + tick number
- No external dependencies that could vary

This makes it suitable for:
- Reproducible testing
- Replay systems
- Lockstep multiplayer

## Type Mapping

| AILANG Type | Go Type |
|-------------|---------|
| `int` | `int64` |
| `float` | `float64` |
| `string` | `string` |
| `bool` | `bool` |
| `{ field: T }` | `*StructName` |
| `[T]` | `[]T` |

## Debug Effect

The Debug effect provides structured tracing for debugging and testing. It's a **ghost effect** - erasable in release mode for zero-cost production builds.

### Host Integration Pattern

```go
// 1. Create Debug context (host-controlled)
debugCtx := game.NewDebugContext()

// 2. In your game loop
for tick := 0; tick < maxTicks; tick++ {
    debugCtx.Reset()                    // Clear for this step
    debugCtx.SetTimestamp(int64(tick))  // Set logical time

    // 3. Call game logic (passes debug context)
    world, output = Step(world, input, debugCtx)

    // 4. Check assertions
    if debugCtx.HasFailedAssertions() {
        for _, a := range debugCtx.FailedAssertions() {
            log.Printf("FAIL at %s: %s", a.Location, a.Message)
        }
    }
}

// 5. Collect all debug data
data := debugCtx.Collect()
```

### Building Debug vs Release

```bash
# Debug mode (default) - Debug.log/assert collect data
go build .

# Release mode - Debug operations are zero-cost no-ops
go build -tags release .
```

### Design Principles

1. **Write-only from game code**: Can Log/Assert, cannot read traces
2. **Host-controlled lifecycle**: Only host calls Collect()/Reset()
3. **Ghost effect**: Erased at build time in release mode
4. **Auto-injected locations**: Source positions added by compiler

See [docs/guides/go-interop.md](../../docs/docs/guides/go-interop.md) for complete documentation.

## AI Effect

The AI effect provides a pluggable AI oracle for calling external AI/ML systems. It uses a simple string→string interface (JSON by convention).

### Host Integration Pattern

```go
// 1. Create AI handler (StubAIHandler for testing)
aiHandler := game.NewStubAIHandler()
aiHandler.SetDefaultResponse(`{"action":"wait"}`)

// 2. Create AI context
aiCtx := game.NewAIContext(aiHandler)

// 3. Call AI from your game logic
func ChooseAction(ctx game.NPCContext, ai *game.AIContext) (game.Action, error) {
    input, _ := json.Marshal(ctx)
    output, err := ai.Call(string(input))
    if err != nil {
        return game.Wait{}, err  // ErrNoAIHandler if nil handler
    }
    var action game.Action
    json.Unmarshal([]byte(output), &action)
    return action, nil
}
```

### Production vs Testing

```go
// Testing: Use stub handler with canned responses
handler := game.NewStubAIHandler()
handler.SetResponse(`{"health":50}`, `{"action":"heal"}`)
aiCtx := game.NewAIContext(handler)

// Production: Implement your own AIHandler
type OpenAIHandler struct { client *openai.Client }
func (h *OpenAIHandler) Call(input string) (string, error) {
    // Call OpenAI API...
}
aiCtx := game.NewAIContext(&OpenAIHandler{client: myClient})
```

### Design Principles

1. **String→string interface**: JSON by convention, not enforced
2. **Neutral naming**: `AI.call` not `AI.decide` (not game-flavored)
3. **No silent fallback**: Nil handler returns `ErrNoAIHandler`
4. **Pluggable handlers**: Swap between stub/OpenAI/Ollama without recompiling
